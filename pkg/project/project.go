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
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/samber/lo"
	"github.com/spf13/afero"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	goruntime "runtime"

	"github.com/nitrictech/cli/pkg/cloud"
	"github.com/nitrictech/cli/pkg/collector"
	"github.com/nitrictech/cli/pkg/project/runtime"
	"github.com/nitrictech/nitric/core/pkg/logger"
	apispb "github.com/nitrictech/nitric/core/pkg/proto/apis/v1"
	httppb "github.com/nitrictech/nitric/core/pkg/proto/http/v1"
	resourcespb "github.com/nitrictech/nitric/core/pkg/proto/resources/v1"
	schedulespb "github.com/nitrictech/nitric/core/pkg/proto/schedules/v1"
	storagepb "github.com/nitrictech/nitric/core/pkg/proto/storage/v1"
	topicspb "github.com/nitrictech/nitric/core/pkg/proto/topics/v1"
	websocketspb "github.com/nitrictech/nitric/core/pkg/proto/websockets/v1"
)

const tempBuildDir = "./.nitric/build"

// BuildImage - Builds the docker image for the service

type Project struct {
	Name      string
	Directory string

	services []Service
}

func (p *Project) GetServices() []Service {
	return p.services
}

// BuildServices - Builds all the services in the project
func (p *Project) BuildServices(fs afero.Fs) (chan ServiceBuildUpdate, error) {
	updatesChan := make(chan ServiceBuildUpdate)

	if len(p.services) == 0 {
		return nil, fmt.Errorf("no services found in project, nothing to build. This may indicate misconfigured `match` patterns in your nitric.yaml file")
	}

	maxConcurrentBuilds := make(chan struct{}, min(goruntime.NumCPU(), goruntime.GOMAXPROCS(0)))

	waitGroup := sync.WaitGroup{}

	for _, service := range p.services {
		waitGroup.Add(1)
		// Create writer
		serviceBuildUpdateWriter := &serviceBuildUpdateWriter{
			buildUpdateChan: updatesChan,
			serviceName:     service.Name,
		}

		go func(svc Service, writer io.Writer) {
			// Acquire a token by filling the maxConcurrentBuilds channel
			// this will block once the buffer is full
			maxConcurrentBuilds <- struct{}{}

			// Start goroutine
			if err := svc.BuildImage(fs, writer); err != nil {
				updatesChan <- ServiceBuildUpdate{
					ServiceName: svc.Name,
					Err:         err,
					Message:     err.Error(),
					Status:      ServiceBuildStatus_Error,
				}
			} else {
				updatesChan <- ServiceBuildUpdate{
					ServiceName: svc.Name,
					Message:     "Build Complete",
					Status:      ServiceBuildStatus_Complete,
				}
			}

			// release our lock
			<-maxConcurrentBuilds

			waitGroup.Done()
		}(service, serviceBuildUpdateWriter)
	}

	go func() {
		waitGroup.Wait()
		// Drain the semaphore to make sure all goroutines have finished
		for i := 0; i < cap(maxConcurrentBuilds); i++ {
			maxConcurrentBuilds <- struct{}{}
		}

		close(updatesChan)
	}()

	return updatesChan, nil
}

func (p *Project) collectServiceRequirements(service Service) (*collector.ServiceRequirements, error) {
	serviceRequirements := collector.NewServiceRequirements(service.Name, service.GetFilePath(), service.Type)

	// start a grpc service with this registered
	grpcServer := grpc.NewServer()

	resourcespb.RegisterResourcesServer(grpcServer, serviceRequirements)
	apispb.RegisterApiServer(grpcServer, serviceRequirements.ApiServer)
	schedulespb.RegisterSchedulesServer(grpcServer, serviceRequirements)
	// topicspb.RegisterTopicsServer(grpcServer, serviceRequirements)
	topicspb.RegisterSubscriberServer(grpcServer, serviceRequirements)
	websocketspb.RegisterWebsocketHandlerServer(grpcServer, serviceRequirements)
	storagepb.RegisterStorageListenerServer(grpcServer, serviceRequirements)
	httppb.RegisterHttpServer(grpcServer, serviceRequirements)

	listener, err := net.Listen("tcp", ":")
	if err != nil {
		return nil, err
	}

	// register non-blocking
	go func() {
		err := grpcServer.Serve(listener)
		if err != nil {
			logger.Errorf("unable to start local Nitric collection server: %s", err)
		}
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
		return nil, fmt.Errorf("unable to split host and port for local Nitric collection server: %w", err)
	}

	err = service.RunContainer(stopChannel, updatesChannel, WithNitricPort(port), WithNitricEnvironment("build"))
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

	for _, service := range p.services {
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
	stopChannels := lo.FanOut[bool](len(p.services), 1, stop)

	group, _ := errgroup.WithContext(context.TODO())

	for i, service := range p.services {
		idx := i
		svc := service

		// start the service with the given file reference from its projects CWD
		group.Go(func() error {
			port, err := localCloud.AddService(svc.filepath)
			if err != nil {
				return err
			}

			return svc.Run(stopChannels[idx], updates, map[string]string{
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
	stopChannels := lo.FanOut[bool](len(p.services), 1, stop)

	group, _ := errgroup.WithContext(context.TODO())

	for i, service := range p.services {
		idx := i
		svc := service

		group.Go(func() error {
			port, err := localCloud.AddService(svc.filepath)
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

	matches := map[string]string{}

	for _, serviceSpec := range projectConfig.Services {
		files, err := afero.Glob(fs, serviceSpec.Match)
		if err != nil {
			return nil, fmt.Errorf("unable to match service files for pattern %s: %w", serviceSpec.Match, err)
		}

		for _, f := range files {
			relativeServiceEntrypointPath, _ := filepath.Rel(projectConfig.Directory, f)

			serviceName := projectConfig.pathToNormalizedServiceName(relativeServiceEntrypointPath)

			var buildContext *runtime.RuntimeBuildContext

			if serviceSpec.Runtime != "" {
				// We have a custom runtime
				customRuntime, ok := projectConfig.Runtimes[serviceSpec.Runtime]
				if !ok {
					return nil, fmt.Errorf("unable to find runtime %s", serviceSpec.Runtime)
				}

				buildContext, err = runtime.NewBuildContext(
					relativeServiceEntrypointPath,
					customRuntime.Dockerfile,
					customRuntime.Args,
					// TODO: Get other entrypoint files as ignores
					[]string{},
					fs,
				)
				if err != nil {
					return nil, fmt.Errorf("unable to create build context for custom service file %s: %w", f, err)
				}
			} else {
				buildContext, err = runtime.NewBuildContext(
					relativeServiceEntrypointPath,
					"",
					map[string]string{},
					// TODO: Get other entrypoint files as ignores
					[]string{},
					fs,
				)
				if err != nil {
					return nil, fmt.Errorf("unable to create build context for service file %s: %w", f, err)
				}
			}

			if matches[f] != "" {
				return nil, fmt.Errorf("service file %s matched by multiple patterns: %s and %s, services must only be matched by a single pattern", f, matches[f], serviceSpec.Match)
			}

			matches[f] = serviceSpec.Match

			newService := Service{
				Name:         serviceName,
				filepath:     f,
				buildContext: *buildContext,
				Type:         serviceSpec.Type,
				startCmd:     serviceSpec.Start,
			}

			if serviceSpec.Type == "" {
				serviceSpec.Type = "default"
			}

			services = append(services, newService)
		}
	}

	return &Project{
		Name:      projectConfig.Name,
		Directory: projectConfig.Directory,
		services:  services,
	}, nil
}

// FromFile - Loads a nitric project from a nitric.yaml file
// If no filepath is provided, the default location './nitric.yaml' is used
func FromFile(fs afero.Fs, filepath string) (*Project, error) {
	projectConfig, err := ConfigurationFromFile(fs, filepath)
	if err != nil {
		return nil, fmt.Errorf("unable to load nitric.yaml, are you currently in a nitric project?: %w", err)
	}

	return fromProjectConfiguration(projectConfig, fs)
}
