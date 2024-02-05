package project

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/samber/lo"
	"github.com/spf13/afero"

	"github.com/nitrictech/cli/pkgplus/docker"
	"github.com/nitrictech/cli/pkgplus/netx"
	"github.com/nitrictech/cli/pkgplus/project/runtime"
)

type ServiceBuildStatus string

type Service struct {
	Name string
	Type string

	// filepath relative to the project root directory
	filepath     string
	buildContext runtime.RuntimeBuildContext

	start string
}

func (s *Service) GetFilePath() string {
	return s.filepath
}

func (s *Service) GetAbsoluteFilePath() (string, error) {
	return filepath.Abs(s.filepath)
}

const (
	ServiceBuildStatus_InProgress ServiceBuildStatus = "In Progress"
	ServiceBuildStatus_Complete   ServiceBuildStatus = "Complete"
	ServiceBuildStatus_Error      ServiceBuildStatus = "Error"
)

type ServiceBuildUpdate struct {
	ServiceName string
	Message     string
	Status      ServiceBuildStatus
	Err         error
}

type ServiceRunStatus string

const (
	ServiceRunStatus_Running ServiceRunStatus = "Running"
	ServiceRunStatus_Done    ServiceRunStatus = "Done"
	ServiceRunStatus_Error   ServiceRunStatus = "Error"
)

type ServiceRunUpdate struct {
	ServiceName string
	Message     string
	Status      ServiceRunStatus
	Err         error
}

type ServiceRunUpdateWriter struct {
	updates     chan<- ServiceRunUpdate
	serviceName string
	status      ServiceRunStatus
}

func (s *ServiceRunUpdateWriter) Write(data []byte) (int, error) {
	msg := string(data)

	s.updates <- ServiceRunUpdate{
		ServiceName: s.serviceName,
		Message:     msg,
		Status:      s.status,
	}

	return len(data), nil
}

type serviceBuildUpdateWriter struct {
	serviceName     string
	buildUpdateChan chan ServiceBuildUpdate
}

func (b *serviceBuildUpdateWriter) Write(data []byte) (int, error) {
	b.buildUpdateChan <- ServiceBuildUpdate{
		ServiceName: b.serviceName,
		Message:     string(data),
		Status:      ServiceBuildStatus_InProgress,
	}

	return len(data), nil
}

func (s *Service) BuildImage(fs afero.Fs, logs io.Writer) error {
	dockerClient, err := docker.New()
	if err != nil {
		return err
	}

	err = fs.MkdirAll(tempBuildDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to create temporary build directory %s: %w", tempBuildDir, err)
	}

	tmpDockerFile, err := afero.TempFile(fs, tempBuildDir, fmt.Sprintf("%s-*.dockerfile", s.Name))
	if err != nil {
		return fmt.Errorf("unable to create temporary dockerfile for service %s: %w", s.Name, err)
	}

	if err := afero.WriteFile(fs, tmpDockerFile.Name(), []byte(s.buildContext.DockerfileContents), os.ModePerm); err != nil {
		return fmt.Errorf("unable to write temporary dockerfile for service %s: %w", s.Name, err)
	}
	defer func() {
		tmpDockerFile.Close()
		fs.Remove(tmpDockerFile.Name())
	}()

	// build the docker image
	err = dockerClient.Build(
		tmpDockerFile.Name(),
		s.buildContext.BaseDirectory,
		s.Name,
		s.buildContext.BuildArguments,
		strings.Split(s.buildContext.IgnoreFileContents, "\n"),
		logs,
	)
	if err != nil {
		return err
	}

	return nil
}

type runContainerOptions struct {
	nitricHost        string
	nitricPort        string
	nitricEnvironment string
	envVars           map[string]string
}

type RunContainerOption func(*runContainerOptions)

var defaultRunContainerOptions = runContainerOptions{
	nitricHost:        "host.docker.internal",
	nitricPort:        "50051",
	nitricEnvironment: "run",
	envVars:           map[string]string{},
}

func WithNitricHost(host string) RunContainerOption {
	return func(o *runContainerOptions) {
		o.nitricHost = host
	}
}

func WithNitricPort(port string) RunContainerOption {
	return func(o *runContainerOptions) {
		o.nitricPort = port
	}
}

func WithNitricEnvironment(environment string) RunContainerOption {
	return func(o *runContainerOptions) {
		o.nitricEnvironment = environment
	}
}

func WithEnvVars(envVars map[string]string) RunContainerOption {
	return func(o *runContainerOptions) {
		o.envVars = envVars
	}
}

type writerFunc func(p []byte) (n int, err error)

func (wf writerFunc) Write(p []byte) (n int, err error) {
	return wf(p)
}

// Run - runs the service using the provided command, typically not in a container.
func (s *Service) Run(stop <-chan bool, updates chan<- ServiceRunUpdate, command string, env map[string]string) error {
	commandParts := strings.Split(fmt.Sprintf(command, s.filepath), " ")
	cmd := exec.Command(
		commandParts[0],
		commandParts[1:]...,
	)

	cmd.Env = append(cmd.Env, os.Environ()...)
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	cmd.Stdout = &ServiceRunUpdateWriter{
		updates:     updates,
		serviceName: s.Name,
		status:      ServiceRunStatus_Running,
	}

	cmd.Stderr = &ServiceRunUpdateWriter{
		updates:     updates,
		serviceName: s.Name,
		status:      ServiceRunStatus_Error,
	}

	errChan := make(chan error)

	go func() {
		err := cmd.Start()
		if err != nil {
			errChan <- err
		}

		err = cmd.Wait()
		errChan <- err
	}()

	go func(cmd *exec.Cmd) {
		<-stop
		_ = cmd.Process.Kill()
	}(cmd)

	err := <-errChan
	updates <- ServiceRunUpdate{
		ServiceName: s.Name,
		Status:      ServiceRunStatus_Error,
		Err:         err,
	}
	return err
}

// RunContainer - Runs a container for the service, blocking until the container exits
func (s *Service) RunContainer(stop <-chan bool, updates chan<- ServiceRunUpdate, opts ...RunContainerOption) error {
	runtimeOptions := lo.ToPtr(defaultRunContainerOptions)

	for _, opt := range opts {
		opt(runtimeOptions)
	}

	dockerClient, err := docker.New()
	if err != nil {
		return err
	}

	hostConfig := &container.HostConfig{
		// TODO: make this configurable through an cmd param
		AutoRemove: true,
		// LogConfig:  *f.ce.Logger(f.runCtx).Config(),
		LogConfig: container.LogConfig{
			Type: "json-file",
			Config: map[string]string{
				"max-size": "10m",
				"max-file": "3",
			},
		},
	}

	if goruntime.GOOS == "linux" {
		// setup host.docker.internal to route to host gateway
		// to access rpc server hosted by local CLI run
		hostConfig.ExtraHosts = []string{"host.docker.internal:172.17.0.1"}
	}

	randomPort, _ := netx.TakePort(1)
	hostProxyPort := fmt.Sprint(randomPort[0])
	env := []string{
		"NITRIC_ENVIRONMENT=run",
		// FIXME: Ensure environment variable consistency in all SDKs, then remove duplicates here.
		fmt.Sprintf("SERVICE_ADDRESS=%s", fmt.Sprintf("%s:%s", runtimeOptions.nitricHost, runtimeOptions.nitricPort)),
		fmt.Sprintf("NITRIC_SERVICE_PORT=%s", runtimeOptions.nitricPort),
		fmt.Sprintf("NITRIC_SERVICE_HOST=%s", runtimeOptions.nitricHost),
		fmt.Sprintf("NITRIC_HTTP_PROXY_PORT=%d", randomPort[0]),
	}

	for k, v := range runtimeOptions.envVars {
		env = append(env, k+"="+v)
	}

	hostConfig.PortBindings = nat.PortMap{
		nat.Port(hostProxyPort): []nat.PortBinding{
			{
				HostPort: hostProxyPort,
			},
		},
	}

	containerConfig := &container.Config{
		Image: s.Name, // Select an image to use based on the handler
		Env:   env,
		ExposedPorts: nat.PortSet{
			nat.Port(hostProxyPort): struct{}{},
		},
	}

	// Create the container
	containerId, err := dockerClient.ContainerCreate(
		containerConfig,
		hostConfig,
		nil,
		s.Name,
	)
	if err != nil {
		updates <- ServiceRunUpdate{
			ServiceName: s.Name,
			Status:      ServiceRunStatus_Error,
			Err:         err,
		}
		return nil
	}

	err = dockerClient.ContainerStart(context.TODO(), containerId, types.ContainerStartOptions{})
	if err != nil {
		updates <- ServiceRunUpdate{
			ServiceName: s.Name,
			Status:      ServiceRunStatus_Error,
			Err:         err,
		}
		return nil
	}

	updates <- ServiceRunUpdate{
		ServiceName: s.Name,
		Message:     fmt.Sprintf("Service %s started", s.Name),
		Status:      ServiceRunStatus_Running,
	}

	// Attach to the container to get stdout and stderr
	attachOptions := types.ContainerAttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
	}
	attachResponse, err := dockerClient.ContainerAttach(context.TODO(), containerId, attachOptions)
	if err != nil {
		// ... handle error
	}

	// Use a separate goroutine to handle the container's output
	go func() {
		defer attachResponse.Close()
		// Using io.Copy to send the output to a writer
		_, err := io.Copy(writerFunc(func(p []byte) (int, error) {
			updates <- ServiceRunUpdate{
				ServiceName: s.Name,
				Message:     string(p),
				Status:      ServiceRunStatus_Running,
			}
			return len(p), nil
		}), attachResponse.Reader)
		if err != nil {
			updates <- ServiceRunUpdate{
				ServiceName: s.Name,
				Status:      ServiceRunStatus_Error,
				Err:         err,
			}
		}
	}()

	okChan, errChan := dockerClient.ContainerWait(context.TODO(), containerId, container.WaitConditionNotRunning)

	for {
		select {
		case err := <-errChan:
			updates <- ServiceRunUpdate{
				ServiceName: s.Name,
				Err:         err,
				Status:      ServiceRunStatus_Error,
			}
			return err
		case <-okChan:
			updates <- ServiceRunUpdate{
				ServiceName: s.Name,
				Message:     "Service successfully exited",
				Status:      ServiceRunStatus_Done,
			}
			return nil
		case <-stop:
			if err := dockerClient.ContainerStop(context.Background(), containerId, nil); err != nil {
				updates <- ServiceRunUpdate{
					ServiceName: s.Name,
					Status:      ServiceRunStatus_Error,
					Err:         err,
				}
				return nil
			}
		}
	}
}
