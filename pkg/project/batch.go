// Copyright Nitric Pty Ltd.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package project

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"syscall"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"github.com/samber/lo"
	"github.com/spf13/afero"

	"github.com/nitrictech/cli/pkg/docker"
	"github.com/nitrictech/cli/pkg/netx"
	"github.com/nitrictech/cli/pkg/project/runtime"
	"github.com/nitrictech/nitric/core/pkg/env"
	"github.com/nitrictech/nitric/core/pkg/logger"
)

type Batch struct {
	Name string

	// filepath relative to the project root directory
	basedir string
	// filepath relative to the basedir
	filepath     string
	buildContext runtime.RuntimeBuildContext

	runCmd string
}

func (s *Batch) GetFilePath() string {
	return filepath.Join(s.basedir, s.filepath)
}

func (s *Batch) GetAbsoluteFilePath() (string, error) {
	return filepath.Abs(s.GetFilePath())
}

// Run - runs the service using the provided command, typically not in a container.
func (s *Batch) Run(stop <-chan bool, updates chan<- ServiceRunUpdate, env map[string]string) error {
	if s.runCmd == "" {
		return fmt.Errorf("no start command provided for service %s", s.filepath)
	}

	// this could be improve with real env var substitution.
	startCmd := strings.ReplaceAll(s.runCmd, "$SERVICE_PATH", s.filepath)
	startCmd = strings.ReplaceAll(startCmd, "${SERVICE_PATH}", s.filepath)

	if !strings.Contains(startCmd, s.filepath) {
		logger.Warnf("Start cmd for service %s does not contain $SERVICE_PATH, check the service start configuration in nitric.yaml", s.filepath)
	}

	commandParts := strings.Split(startCmd, " ")
	cmd := exec.Command(
		commandParts[0],
		commandParts[1:]...,
	)

	cmd.Env = append([]string{}, os.Environ()...)
	cmd.Dir = s.basedir

	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	cmd.Stdout = &ServiceRunUpdateWriter{
		updates:     updates,
		serviceName: s.Name,
		label:       s.GetFilePath(),
		status:      ServiceRunStatus_Running,
	}

	cmd.Stderr = &ServiceRunUpdateWriter{
		updates:     updates,
		serviceName: s.Name,
		label:       s.GetFilePath(),
		status:      ServiceRunStatus_Error,
	}

	errChan := make(chan error)

	go func() {
		err := cmd.Start()
		if err != nil {
			errChan <- fmt.Errorf("error starting service %s: %w", s.Name, err)
		} else {
			updates <- ServiceRunUpdate{
				ServiceName: s.Name,
				Label:       "nitric",
				Status:      ServiceRunStatus_Running,
				Message:     fmt.Sprintf("started service %s", s.GetFilePath()),
			}
		}

		err = cmd.Wait()
		if err != nil {
			// provide runtime errors as a run update rather than as a fatal error
			updates <- ServiceRunUpdate{
				ServiceName: s.Name,
				Label:       "nitric",
				Status:      ServiceRunStatus_Error,
				Err:         err,
			}
		}

		errChan <- nil
	}()

	go func(cmd *exec.Cmd) {
		<-stop

		err := cmd.Process.Signal(syscall.SIGTERM)
		if err != nil {
			_ = cmd.Process.Kill()
		}
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
func (s *Batch) RunContainer(stop <-chan bool, updates chan<- ServiceRunUpdate, opts ...RunContainerOption) error {
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
		dockerHost := env.GetEnv("NITRIC_DOCKER_HOST", "172.17.0.1")

		// FIXME: Skip dns resolution for now to allow nitric apps using deno to work on linux
		// https://github.com/denoland/deno/issues/25927
		runtimeOptions.nitricHost = dockerHost.String()
		// setup host.docker.internal to route to host gateway
		// to access rpc server hosted by local CLI run
		hostConfig.ExtraHosts = []string{"host.docker.internal:" + dockerHost.String()}
	}

	randomPort, _ := netx.TakePort(1)
	hostProxyPort := fmt.Sprint(randomPort[0])
	env := []string{
		fmt.Sprintf("NITRIC_ENVIRONMENT=%s", runtimeOptions.nitricEnvironment),
		// FIXME: Ensure environment variable consistency in all SDKs, then remove duplicates here.
		fmt.Sprintf("SERVICE_ADDRESS=%s", fmt.Sprintf("%s:%s", runtimeOptions.nitricHost, runtimeOptions.nitricPort)),
		fmt.Sprintf("NITRIC_SERVICE_PORT=%s", runtimeOptions.nitricPort),
		fmt.Sprintf("NITRIC_SERVICE_HOST=%s", runtimeOptions.nitricHost),
		fmt.Sprintf("NITRIC_HTTP_PROXY_PORT=%d", randomPort[0]),
	}

	for k, v := range runtimeOptions.envVars {
		// detect blacklisted env vars set by nitric in env vars
		_, isBlacklisted := lo.Find(env, func(e string) bool {
			return strings.HasPrefix(e, k)
		})

		if isBlacklisted {
			updates <- ServiceRunUpdate{
				ServiceName: s.Name,
				Label:       s.GetFilePath(),
				Message:     fmt.Sprintf("Skipping blacklisted env var: %s", k),
				Status:      ServiceRunStatus_Running,
			}

			continue
		}

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
			Label:       s.GetFilePath(),
			Status:      ServiceRunStatus_Error,
			Err:         err,
		}

		return nil
	}

	err = dockerClient.ContainerStart(context.TODO(), containerId, container.StartOptions{})
	if err != nil {
		updates <- ServiceRunUpdate{
			ServiceName: s.Name,
			Label:       s.GetFilePath(),
			Status:      ServiceRunStatus_Error,
			Err:         err,
		}

		return nil
	}

	updates <- ServiceRunUpdate{
		ServiceName: s.Name,
		Label:       s.GetFilePath(),
		Message:     fmt.Sprintf("Service %s started", s.Name),
		Status:      ServiceRunStatus_Running,
	}

	// Attach to the container to get stdout and stderr
	attachOptions := container.AttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
	}

	attachResponse, err := dockerClient.ContainerAttach(context.TODO(), containerId, attachOptions)
	if err != nil {
		return fmt.Errorf("error attaching to container %s: %w", s.Name, err)
	}

	// Use a separate goroutine to handle the container's output
	go func() {
		defer attachResponse.Close()
		// Using io.Copy to send the output to a writer
		_, err := io.Copy(writerFunc(func(p []byte) (int, error) {
			updates <- ServiceRunUpdate{
				ServiceName: s.Name,
				Label:       s.GetFilePath(),
				Message:     string(p),
				Status:      ServiceRunStatus_Running,
			}

			return len(p), nil
		}), attachResponse.Reader)
		if err != nil {
			updates <- ServiceRunUpdate{
				ServiceName: s.Name,
				Label:       s.GetFilePath(),
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
				Label:       s.GetFilePath(),
				Err:         err,
				Status:      ServiceRunStatus_Error,
			}

			return err
		case okBody := <-okChan:
			if okBody.StatusCode != 0 {
				logOptions := container.LogsOptions{ShowStdout: true, ShowStderr: true, Tail: "20"}

				logReader, err := dockerClient.ContainerLogs(context.Background(), containerId, logOptions)
				if err != nil {
					return err
				}

				// Create a buffer to hold the logs
				var logs bytes.Buffer
				if _, err := stdcopy.StdCopy(&logs, &logs, logReader); err != nil {
					return fmt.Errorf("error reading logs for service %s: %w", s.Name, err)
				}

				err = fmt.Errorf("service %s exited with non 0 status\n %s", s.Name, logs.String())

				updates <- ServiceRunUpdate{
					ServiceName: s.Name,
					Label:       s.GetFilePath(),
					Err:         err,
					Status:      ServiceRunStatus_Error,
				}

				return err
			} else {
				updates <- ServiceRunUpdate{
					Label:       s.GetFilePath(),
					ServiceName: s.Name,
					Message:     "Service successfully exited",
					Status:      ServiceRunStatus_Done,
				}
			}

			return nil
		case <-stop:
			if err := dockerClient.ContainerStop(context.Background(), containerId, container.StopOptions{}); err != nil {
				updates <- ServiceRunUpdate{
					Label:       s.GetFilePath(),
					ServiceName: s.Name,
					Status:      ServiceRunStatus_Error,
					Err:         err,
				}

				return nil
			}
		}
	}
}

// FIXME: Duplicate code from service.go
func (s *Batch) BuildImage(fs afero.Fs, logs io.Writer) error {
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

		err := fs.Remove(tmpDockerFile.Name())
		if err != nil {
			logger.Errorf("unable to remove temporary dockerfile %s: %s", tmpDockerFile.Name(), err)
		}
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
