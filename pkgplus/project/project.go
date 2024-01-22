package project

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/samber/lo"
	"github.com/spf13/afero"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	"github.com/nitrictech/cli/pkgplus/cloud"
	"github.com/nitrictech/cli/pkgplus/collector"
	"github.com/nitrictech/cli/pkgplus/docker"
	"github.com/nitrictech/cli/pkgplus/netx"
	"github.com/nitrictech/cli/pkgplus/project/runtime"
	apispb "github.com/nitrictech/nitric/core/pkg/proto/apis/v1"
	resourcespb "github.com/nitrictech/nitric/core/pkg/proto/resources/v1"
	schedulespb "github.com/nitrictech/nitric/core/pkg/proto/schedules/v1"
	storagepb "github.com/nitrictech/nitric/core/pkg/proto/storage/v1"
	topicspb "github.com/nitrictech/nitric/core/pkg/proto/topics/v1"
	websocketspb "github.com/nitrictech/nitric/core/pkg/proto/websockets/v1"
)

type Service struct {
	Name string
	Type string

	file         string
	buildContext runtime.RuntimeBuildContext

	start string
}

const tempBuildDir = "./.nitric/build"

// BuildImage - Builds the docker image for the service
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
	commandParts := strings.Split(fmt.Sprintf(command, s.file), " ")
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

type Project struct {
	Name      string
	Directory string

	Services []Service
}

type ServiceBuildStatus string

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

// BuildServices - Builds all the services in the project
func (p *Project) BuildServices(fs afero.Fs) (chan ServiceBuildUpdate, error) {
	updatesChan := make(chan ServiceBuildUpdate)

	if len(p.Services) == 0 {
		return nil, fmt.Errorf("no services found in project, nothing to build")
	}

	waitGroup := sync.WaitGroup{}

	for _, service := range p.Services {
		waitGroup.Add(1)
		// Create writer
		serviceBuildUpdateWriter := &serviceBuildUpdateWriter{
			buildUpdateChan: updatesChan,
			serviceName:     service.Name,
		}

		go func(svc Service, writer io.Writer) {
			// Start goroutine
			if err := svc.BuildImage(fs, writer); err != nil {
				updatesChan <- ServiceBuildUpdate{
					ServiceName: svc.Name,
					Err:         err,
					Status:      ServiceBuildStatus_Error,
				}
			} else {
				updatesChan <- ServiceBuildUpdate{
					ServiceName: svc.Name,
					Message:     "Build Complete",
					Status:      ServiceBuildStatus_Complete,
				}
			}

			waitGroup.Done()
		}(service, serviceBuildUpdateWriter)
	}

	go func() {
		waitGroup.Wait()
		close(updatesChan)
	}()

	return updatesChan, nil
}

func (p *Project) collectServiceRequirements(service Service) (*collector.ServiceRequirements, error) {
	serviceRequirements := collector.NewServiceRequirements(service.Name, service.file, service.Type)

	// start a grpc service with this registered
	grpcServer := grpc.NewServer()

	resourcespb.RegisterResourcesServer(grpcServer, serviceRequirements)
	apispb.RegisterApiServer(grpcServer, serviceRequirements)
	schedulespb.RegisterSchedulesServer(grpcServer, serviceRequirements)
	topicspb.RegisterTopicsServer(grpcServer, serviceRequirements)
	websocketspb.RegisterWebsocketHandlerServer(grpcServer, serviceRequirements)
	storagepb.RegisterStorageListenerServer(grpcServer, serviceRequirements)

	listener, err := net.Listen("tcp", ":")
	if err != nil {
		return nil, err
	}

	// register non-blocking
	go func() {
		grpcServer.Serve(listener)
	}()
	defer grpcServer.Stop()

	// run the service we want to collect for targeting the grpc server
	// TODO: load and run .env files, etc.
	stopChannel := make(chan bool)
	updatesChannel := make(chan ServiceRunUpdate)
	go func() {
		for range updatesChannel {
			// TODO: Provide some updates - bubbletea nice output
			// fmt.Println("container update:", update)
			continue
		}
	}()

	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		return nil, fmt.Errorf("unable to split host and port for local Nitric collection server: %v", err)
	}

	err = service.RunContainer(stopChannel, updatesChannel, WithNitricPort(port))
	if err != nil {
		return nil, err
	}

	return serviceRequirements, nil
}

func (p *Project) CollectServicesRequirements() ([]*collector.ServiceRequirements, error) {
	allServiceRequirements := []*collector.ServiceRequirements{}
	serviceErrors := []error{}

	reqLock := sync.Mutex{}
	errorLock := sync.Mutex{}
	wg := sync.WaitGroup{}

	for _, service := range p.Services {
		svc := service
		wg.Add(1)
		go func(s Service) {
			defer wg.Done()
			serviceRequirements, err := p.collectServiceRequirements(s)
			if err != nil {
				errorLock.Lock()
				defer errorLock.Unlock()
				serviceErrors = append(serviceErrors, err)
				return
			}

			reqLock.Lock()
			defer reqLock.Unlock()
			allServiceRequirements = append(allServiceRequirements, serviceRequirements)
		}(svc)
	}

	wg.Wait()

	if len(serviceErrors) > 0 {
		return nil, errors.Join(serviceErrors...)
	}

	return allServiceRequirements, nil
}

// RunServices - Runs all the services locally using a startup command
// use the stop channel to stop all running services
func (p *Project) RunServicesWithCommand(localCloud *cloud.LocalCloud, stop <-chan bool, updates chan<- ServiceRunUpdate) error {
	stopChannels := lo.FanOut[bool](len(p.Services), 1, stop)

	group, _ := errgroup.WithContext(context.TODO())

	for i, service := range p.Services {
		idx := i
		svc := service

		// start the service with the given file reference from its projects CWD
		group.Go(func() error {
			port, err := localCloud.AddService(svc.Name)
			if err != nil {
				return err
			}

			return svc.Run(stopChannels[idx], updates, svc.start, map[string]string{
				"NITRIC_ENVIRONMENT": "run",
				"SERVICE_ADDRESS":    "localhost:" + strconv.Itoa(port),
				// TODO: add .env variables.
			})
		})
	}

	return group.Wait()
}

// RunServices - Runs all the services as containers
// use the stop channel to stop all running services
func (p *Project) RunServices(localCloud *cloud.LocalCloud, stop <-chan bool, updates chan<- ServiceRunUpdate) error {
	stopChannels := lo.FanOut[bool](len(p.Services), 1, stop)

	group, _ := errgroup.WithContext(context.TODO())

	for i, service := range p.Services {
		idx := i
		svc := service

		group.Go(func() error {
			port, err := localCloud.AddService(svc.Name)
			if err != nil {
				return err
			}

			return svc.RunContainer(stopChannels[idx], updates, WithNitricPort(strconv.Itoa(port)))
		})
	}

	return group.Wait()
}

func (pc *ProjectConfiguration) pathToNormalizedServiceName(servicePath string) string {
	// Add the project name as a prefix to group service images
	servicePath = fmt.Sprintf("%s_%s", pc.Name, servicePath)
	// replace path separators with dashes
	servicePath = strings.ReplaceAll(servicePath, string(os.PathSeparator), "-")
	// remove the file extension
	servicePath = strings.ReplaceAll(servicePath, filepath.Ext(servicePath), "")
	// replace dots with dashes
	servicePath = strings.ReplaceAll(servicePath, ".", "-")
	// replace all non-word characters
	servicePath = strings.ReplaceAll(servicePath, "[^\\w]", "-")

	return strings.ToLower(servicePath)
}

// fromProjectConfiguration creates a new Instance of a nitric Project from a configuration files contents
func fromProjectConfiguration(projectConfig *ProjectConfiguration, fs afero.Fs) (*Project, error) {
	services := []Service{}

	for _, service := range projectConfig.Services {
		files, err := filepath.Glob(service.Match)
		if err != nil {
			return nil, fmt.Errorf("unable to match service files for pattern %s: %v", service.Match, err)
		}

		for _, f := range files {
			relativeServiceEntrypointPath, err := filepath.Rel(projectConfig.Directory, f)

			serviceName := projectConfig.pathToNormalizedServiceName(relativeServiceEntrypointPath)

			var buildContext *runtime.RuntimeBuildContext = nil
			if service.Runtime != "" {
				// We have a custom runtime
				customRuntime, ok := projectConfig.Runtimes[service.Runtime]
				if !ok {
					return nil, fmt.Errorf("unable to find runtime %s", service.Runtime)
				}

				buildContext, err = runtime.NewBuildContext(
					f,
					customRuntime.Dockerfile,
					customRuntime.Args,
					// TODO: Get other entrypoint files as ignores
					[]string{},
					fs,
				)
				if err != nil {
					return nil, fmt.Errorf("unable to create build context for custom service file %s: %v", f, err)
				}
			} else {
				buildContext, err = runtime.NewBuildContext(
					f,
					"",
					map[string]string{},
					// TODO: Get other entrypoint files as ignores
					[]string{},
					fs,
				)
				if err != nil {
					return nil, fmt.Errorf("unable to create build context for service file %s: %v", f, err)
				}
			}

			newService := Service{
				Name:         serviceName,
				file:         f,
				buildContext: *buildContext,
				Type:         service.Type,
				start:        service.Start,
			}

			if service.Type == "" {
				service.Type = "default"
			}

			services = append(services, newService)
		}
	}

	return &Project{
		Name:      projectConfig.Name,
		Directory: projectConfig.Directory,
		Services:  services,
	}, nil
}

// FromFile - Loads a nitric project from a nitric.yaml file
// If no filepath is provided, the default location './nitric.yaml' is used
func FromFile(fs afero.Fs, filepath string) (*Project, error) {
	projectConfig, err := ConfigurationFromFile(fs, filepath)
	if err != nil {
		return nil, fmt.Errorf("unable to load nitric.yaml, are you currently in a nitric project?: %v", err)
	}

	return fromProjectConfiguration(projectConfig, fs)
}
