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
	_ "embed"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/samber/lo"
	"github.com/spf13/afero"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	goruntime "runtime"

	"github.com/nitrictech/cli/pkg/cloud"
	"github.com/nitrictech/cli/pkg/cloud/websites"
	"github.com/nitrictech/cli/pkg/collector"
	"github.com/nitrictech/cli/pkg/preview"
	"github.com/nitrictech/cli/pkg/project/localconfig"
	"github.com/nitrictech/cli/pkg/project/runtime"
	"github.com/nitrictech/nitric/core/pkg/logger"
	deploymentpb "github.com/nitrictech/nitric/core/pkg/proto/deployments/v1"
)

type Project struct {
	Name        string
	Directory   string
	Preview     []preview.Feature
	LocalConfig localconfig.LocalConfiguration

	services []Service
	batches  []Batch
	websites []Website
}

func (p *Project) GetServices() []Service {
	return p.services
}

func (p *Project) GetBatchServices() []Batch {
	return p.batches
}

func (p *Project) GetWebsites() []Website {
	return p.websites
}

// TODO: Reduce duplicate code
// BuildBatches - Builds all the batches in the project
func (p *Project) BuildBatches(fs afero.Fs, useBuilder bool) (chan ServiceBuildUpdate, error) {
	updatesChan := make(chan ServiceBuildUpdate)

	maxConcurrentBuilds := make(chan struct{}, min(goruntime.NumCPU(), goruntime.GOMAXPROCS(0)))

	waitGroup := sync.WaitGroup{}

	for _, batch := range p.batches {
		waitGroup.Add(1)
		// Create writer
		serviceBuildUpdateWriter := &serviceBuildUpdateWriter{
			buildUpdateChan: updatesChan,
			serviceName:     batch.Name,
		}

		go func(svc Batch, writer io.Writer) {
			// Acquire a token by filling the maxConcurrentBuilds channel
			// this will block once the buffer is full
			maxConcurrentBuilds <- struct{}{}

			// Start goroutine
			if err := svc.BuildImage(fs, writer, useBuilder); err != nil {
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
		}(batch, serviceBuildUpdateWriter)
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

// BuildServices - Builds all the services in the project
func (p *Project) BuildServices(fs afero.Fs, useBuilder bool) (chan ServiceBuildUpdate, error) {
	updatesChan := make(chan ServiceBuildUpdate)

	maxConcurrentBuilds := make(chan struct{}, min(goruntime.NumCPU(), goruntime.GOMAXPROCS(0)))

	waitGroup := sync.WaitGroup{}

	for _, service := range p.services {
		waitGroup.Add(1)
		// Create writer
		serviceBuildUpdateWriter := NewBuildUpdateWriter(service.Name, updatesChan)

		go func(svc Service, writer io.Writer) {
			// Acquire a token by filling the maxConcurrentBuilds channel
			// this will block once the buffer is full
			maxConcurrentBuilds <- struct{}{}

			// Start goroutine
			if err := svc.BuildImage(fs, writer, useBuilder); err != nil {
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

// BuildWebsites - Builds all the websites in the project via build command
func (p *Project) BuildWebsites(env map[string]string) (chan ServiceBuildUpdate, error) {
	updatesChan := make(chan ServiceBuildUpdate)

	maxConcurrentBuilds := make(chan struct{}, min(goruntime.NumCPU(), goruntime.GOMAXPROCS(0)))

	waitGroup := sync.WaitGroup{}

	for _, website := range p.websites {
		waitGroup.Add(1)
		// Create writer
		serviceBuildUpdateWriter := NewBuildUpdateWriter(website.Name, updatesChan)

		go func(site Website, writer io.Writer) {
			// Acquire a token by filling the maxConcurrentBuilds channel
			// this will block once the buffer is full
			maxConcurrentBuilds <- struct{}{}

			// Start goroutine
			if err := site.Build(updatesChan, env); err != nil {
				updatesChan <- ServiceBuildUpdate{
					ServiceName: site.Name,
					Err:         err,
					Message:     err.Error(),
					Status:      ServiceBuildStatus_Error,
				}
			} else {
				updatesChan <- ServiceBuildUpdate{
					ServiceName: site.Name,
					Message:     "Build Complete",
					Status:      ServiceBuildStatus_Complete,
				}
			}

			// release our lock
			<-maxConcurrentBuilds

			waitGroup.Done()
		}(website, serviceBuildUpdateWriter)
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

	serviceRequirements.RegisterServices(grpcServer)

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
		// TODO: elevate env for tmp diretory and reuse
		tmpCollectDir := "./.nitric/collect"

		err := os.MkdirAll(tmpCollectDir, os.ModePerm)
		if err != nil {
			log.Fatalf("unable to create collect log directory %s", err)
		}

		// Create a temporary log file for this service
		logFile, err := afero.TempFile(afero.NewOsFs(), tmpCollectDir, fmt.Sprintf("nitric-%s-*.log", service.Name))
		if err != nil {
			log.Fatalf("unable to create collect log file: %s", err)
		}

		defer logFile.Close()

		for update := range updatesChannel {
			_, err = logFile.WriteString(update.Message)
			if err != nil {
				log.Fatalf("unable to write update log %s", err)
			}

			if update.Err != nil {
				_, err = logFile.WriteString(update.Err.Error())
				if err != nil {
					log.Fatalf("unable to write update error log %s", err)
				}
			}
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

	if serviceRequirements.HasDatabases() && !slices.Contains(p.Preview, preview.Feature_SqlDatabases) {
		return nil, fmt.Errorf("service %s requires a database, but the project does not have the 'sql-databases' preview feature enabled. Please add sql-databases to the preview field of your nitric.yaml file to enable this feature", service.GetFilePath())
	}

	return serviceRequirements, nil
}

func (p *Project) collectBatchRequirements(service Batch) (*collector.BatchRequirements, error) {
	serviceRequirements := collector.NewBatchRequirements(service.Name, service.GetFilePath())

	// start a grpc service with this registered
	grpcServer := grpc.NewServer()

	serviceRequirements.RegisterServices(grpcServer)

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

	if serviceRequirements.HasDatabases() && !slices.Contains(p.Preview, preview.Feature_SqlDatabases) {
		return nil, fmt.Errorf("service %s requires a database, but the project does not have the 'sql-databases' preview feature enabled. Please add sql-databases to the preview field of your nitric.yaml file to enable this feature", service.filepath)
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

func (p *Project) CollectBatchRequirements() ([]*collector.BatchRequirements, error) {
	allBatchRequirements := []*collector.BatchRequirements{}
	batchErrors := []error{}

	reqLock := sync.Mutex{}
	errorLock := sync.Mutex{}
	wg := sync.WaitGroup{}

	for _, batch := range p.batches {
		b := batch

		wg.Add(1)

		go func(s Batch) {
			defer wg.Done()

			batchRequirements, err := p.collectBatchRequirements(s)
			if err != nil {
				errorLock.Lock()
				defer errorLock.Unlock()

				batchErrors = append(batchErrors, err)

				return
			}

			reqLock.Lock()
			defer reqLock.Unlock()

			allBatchRequirements = append(allBatchRequirements, batchRequirements)
		}(b)
	}

	wg.Wait()

	if len(batchErrors) > 0 {
		return nil, errors.Join(batchErrors...)
	}

	return allBatchRequirements, nil
}

func (p *Project) CollectWebsiteRequirements() ([]*deploymentpb.Website, error) {
	allWebsiteRequirements := []*deploymentpb.Website{}

	for _, site := range p.websites {
		outputDir, err := site.GetAbsoluteOutputPath()
		if err != nil {
			return nil, fmt.Errorf("unable to get absolute output path for website %s: %w", site.basedir, err)
		}

		allWebsiteRequirements = append(allWebsiteRequirements, &deploymentpb.Website{
			BasePath:      site.path,
			IndexDocument: site.indexPage,
			ErrorDocument: site.errorPage,
			AssetSource: &deploymentpb.Website_LocalDirectory{
				LocalDirectory: outputDir,
			},
		})
	}

	return allWebsiteRequirements, nil
}

// DefaultMigrationImage - Returns the default migration image name for the project
// Also returns ok if image is required or not
func (p *Project) DefaultMigrationImage(fs afero.Fs) (string, bool) {
	ok, _ := afero.DirExists(fs, "./migrations")

	return fmt.Sprintf("%s-nitric-migrations", p.Name), ok
}

// RunServicesWithCommand - Runs all the services locally using a startup command
// use the stop channel to stop all running services
func (p *Project) RunServicesWithCommand(localCloud *cloud.LocalCloud, stop <-chan bool, updates chan<- ServiceRunUpdate, env map[string]string) error {
	stopChannels := lo.FanOut[bool](len(p.services), 1, stop)

	group, _ := errgroup.WithContext(context.TODO())

	for i, service := range p.services {
		idx := i
		svc := service

		// start the service with the given file reference from its projects CWD
		group.Go(func() error {
			port, err := localCloud.AddService(svc.GetFilePath())
			if err != nil {
				return fmt.Errorf("unable to add service %s: %w", svc.GetFilePath(), err)
			}

			envVariables := map[string]string{
				"PYTHONUNBUFFERED":   "TRUE", // ensure all print statements print immediately for python
				"NITRIC_ENVIRONMENT": "run",
				"SERVICE_ADDRESS":    "localhost:" + strconv.Itoa(port),
			}

			for key, value := range env {
				envVariables[key] = value
			}

			err = svc.Run(stopChannels[idx], updates, envVariables)
			if err != nil {
				return fmt.Errorf("%s: %w", svc.GetFilePath(), err)
			}

			return nil
		})
	}

	return group.Wait()
}

// RunBatchesWithCommand - Runs all the batches locally using a startup command
// use the stop channel to stop all running batches
func (p *Project) RunBatchesWithCommand(localCloud *cloud.LocalCloud, stop <-chan bool, updates chan<- ServiceRunUpdate, env map[string]string) error {
	stopChannels := lo.FanOut[bool](len(p.batches), 1, stop)

	group, _ := errgroup.WithContext(context.TODO())

	for i, service := range p.batches {
		idx := i
		svc := service

		// start the service with the given file reference from its projects CWD
		group.Go(func() error {
			port, err := localCloud.AddBatch(svc.GetFilePath())
			if err != nil {
				return fmt.Errorf("unable to add batch %s: %w", svc.GetFilePath(), err)
			}

			envVariables := map[string]string{
				"PYTHONUNBUFFERED":   "TRUE", // ensure all print statements print immediately for python
				"NITRIC_ENVIRONMENT": "run",
				"SERVICE_ADDRESS":    "localhost:" + strconv.Itoa(port),
			}

			for key, value := range env {
				envVariables[key] = value
			}

			err = svc.Run(stopChannels[idx], updates, envVariables)
			if err != nil {
				return fmt.Errorf("%s: %w", svc.GetFilePath(), err)
			}

			return nil
		})
	}

	return group.Wait()
}

// RunBatches - Runs all the batches as containers
// use the stop channel to stop all running batches
func (p *Project) RunBatches(localCloud *cloud.LocalCloud, stop <-chan bool, updates chan<- ServiceRunUpdate, env map[string]string) error {
	stopChannels := lo.FanOut[bool](len(p.batches), 1, stop)

	group, _ := errgroup.WithContext(context.TODO())

	for i, service := range p.batches {
		idx := i
		svc := service

		group.Go(func() error {
			port, err := localCloud.AddBatch(svc.GetFilePath())
			if err != nil {
				return err
			}

			return svc.RunContainer(stopChannels[idx], updates, WithNitricPort(strconv.Itoa(port)), WithEnvVars(env))
		})
	}

	return group.Wait()
}

// RunServices - Runs all the services as containers
// use the stop channel to stop all running services
func (p *Project) RunServices(localCloud *cloud.LocalCloud, stop <-chan bool, updates chan<- ServiceRunUpdate, env map[string]string) error {
	stopChannels := lo.FanOut[bool](len(p.services), 1, stop)

	group, _ := errgroup.WithContext(context.TODO())

	for i, service := range p.services {
		idx := i
		svc := service

		group.Go(func() error {
			port, err := localCloud.AddService(svc.GetFilePath())
			if err != nil {
				return err
			}

			return svc.RunContainer(stopChannels[idx], updates, WithNitricPort(strconv.Itoa(port)), WithEnvVars(env))
		})
	}

	return group.Wait()
}

// RunWebsites - Runs all the websites as http servers
// TODO this has duplicate code with CollectWebsiteRequirements
func (p *Project) RunWebsites(localCloud *cloud.LocalCloud) error {
	sites := []websites.Website{}

	// register websites with the local cloud
	for _, site := range p.websites {
		outputDir, err := site.GetAbsoluteOutputPath()
		if err != nil {
			return fmt.Errorf("unable to get absolute output path for website %s: %w", site.basedir, err)
		}

		absBaseDir, err := site.GetAbsoluteBaseDirectory()
		if err != nil {
			return fmt.Errorf("unable to get absolute directory for website %s: %w", site.basedir, err)
		}

		sites = append(sites, websites.Website{
			Name:            site.Name,
			DevURL:          site.devURL,
			Directory:       absBaseDir,
			OutputDirectory: outputDir,
			WebsitePb: &websites.WebsitePb{
				BasePath:      site.path,
				IndexDocument: site.indexPage,
				ErrorDocument: site.errorPage,
				AssetSource: &deploymentpb.Website_LocalDirectory{
					LocalDirectory: outputDir,
				},
			},
		})
	}

	return localCloud.Websites.Start(sites)
}

// RunWebsitesWithCommand - Runs all the websites using a startup command
// use the stop channel to stop all running websites
func (p *Project) RunWebsitesWithCommand(localCloud *cloud.LocalCloud, stop <-chan bool, updates chan<- ServiceRunUpdate, env map[string]string) error {
	stopChannels := lo.FanOut[bool](len(p.websites), 1, stop)

	group, _ := errgroup.WithContext(context.TODO())

	for i, site := range p.websites {
		idx := i
		s := site

		// start the service with the given file reference from its projects CWD
		group.Go(func() error {
			envVariables := map[string]string{
				"PYTHONUNBUFFERED":   "TRUE", // ensure all print statements print immediately for python
				"NITRIC_ENVIRONMENT": "run",
			}

			for key, value := range env {
				envVariables[key] = value
			}

			err := s.Run(stopChannels[idx], updates, envVariables)
			if err != nil {
				return fmt.Errorf("%s: %w", s.Name, err)
			}

			return nil
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
func fromProjectConfiguration(projectConfig *ProjectConfiguration, localConfig *localconfig.LocalConfiguration, fs afero.Fs) (*Project, error) {
	services := []Service{}
	batches := []Batch{}
	websites := []Website{}

	matches := map[string]string{}

	baseServices := []BaseService{}
	for _, serviceSpec := range projectConfig.Services {
		baseServices = append(baseServices, serviceSpec)
	}

	for _, batchSpec := range projectConfig.Batches {
		baseServices = append(baseServices, batchSpec)
	}

	for _, baseService := range baseServices {
		serviceMatch := filepath.Join(baseService.GetBasedir(), baseService.GetMatch())

		files, err := afero.Glob(fs, serviceMatch)
		if err != nil {
			return nil, fmt.Errorf("unable to match service files for pattern %s: %w", serviceMatch, err)
		}

		if len(files) == 0 {
			return nil, fmt.Errorf("unable to match service files for pattern %s. This may indicate misconfigured `match` patterns in your nitric.yaml file", serviceMatch)
		}

		for _, f := range files {
			relativeServiceEntrypointPath, _ := filepath.Rel(filepath.Join(projectConfig.Directory, baseService.GetBasedir()), f)
			projectRelativeServiceFile := filepath.Join(projectConfig.Directory, f)

			serviceName := projectConfig.pathToNormalizedServiceName(projectRelativeServiceFile)

			var buildContext *runtime.RuntimeBuildContext

			otherEntryPointFiles := lo.Filter(files, func(file string, index int) bool {
				return file != f
			})

			if baseService.GetRuntime() != "" {
				// We have a custom runtime
				customRuntime, ok := projectConfig.Runtimes[baseService.GetRuntime()]
				if !ok {
					return nil, fmt.Errorf("unable to find runtime %s", baseService.GetRuntime())
				}

				buildContext, err = runtime.NewBuildContext(
					relativeServiceEntrypointPath,
					customRuntime.Dockerfile,
					// will default to the project directory if not set
					lo.Ternary(customRuntime.Context != "", customRuntime.Context, baseService.GetBasedir()),
					customRuntime.Args,
					otherEntryPointFiles,
					fs,
				)
				if err != nil {
					return nil, fmt.Errorf("unable to create build context for custom service file %s: %w", f, err)
				}
			} else {
				buildContext, err = runtime.NewBuildContext(
					relativeServiceEntrypointPath,
					"",
					baseService.GetBasedir(),
					map[string]string{},
					otherEntryPointFiles,
					fs,
				)
				if err != nil {
					return nil, fmt.Errorf("unable to create build context for service file %s: %w", f, err)
				}
			}

			if matches[f] != "" {
				return nil, fmt.Errorf("service file %s matched by multiple patterns: %s and %s, services must only be matched by a single pattern", f, matches[f], baseService.GetMatch())
			}

			matches[f] = baseService.GetMatch()

			relativeFilePath, err := filepath.Rel(baseService.GetBasedir(), f)
			if err != nil {
				return nil, fmt.Errorf("unable to get relative file path for service %s: %w", f, err)
			}

			if svc, ok := baseService.(ServiceConfiguration); ok {
				newService := Service{
					Name:         serviceName,
					filepath:     relativeFilePath,
					basedir:      baseService.GetBasedir(),
					buildContext: *buildContext,
					Type:         svc.Type,
					startCmd:     svc.Start,
				}

				if svc.Type == "" {
					svc.Type = "default"
				}

				services = append(services, newService)
			} else if batch, ok := baseService.(BatchConfiguration); ok {
				newBatch := Batch{
					Name:         serviceName,
					basedir:      batch.Basedir,
					filepath:     relativeFilePath,
					buildContext: *buildContext,
					runCmd:       batch.Start,
				}

				batches = append(batches, newBatch)
			}
		}
	}

	for _, websiteSpec := range projectConfig.Websites {
		if websiteSpec.Build.Output == "" {
			return nil, fmt.Errorf("no build output provided for website %s", websiteSpec.GetBasedir())
		}

		// apply defaults and validate website configuration
		if websiteSpec.Path == "" {
			websiteSpec.Path = "/"
		} else if !strings.HasPrefix(websiteSpec.Path, "/") {
			return nil, fmt.Errorf("invalid website path %s, must start with a /", websiteSpec.Path)
		}

		if websiteSpec.IndexPage == "" {
			websiteSpec.IndexPage = "index.html"
		} else if !strings.HasSuffix(websiteSpec.IndexPage, ".html") {
			return nil, fmt.Errorf("invalid index page %s, must end with .html", websiteSpec.IndexPage)
		}

		if websiteSpec.ErrorPage == "" {
			websiteSpec.ErrorPage = "404.html"
		} else if !strings.HasSuffix(websiteSpec.ErrorPage, ".html") {
			return nil, fmt.Errorf("invalid error page %s, must end with .html", websiteSpec.ErrorPage)
		}

		// Get the absolute path directly
		absBaseDir, err := filepath.Abs(filepath.Join(projectConfig.Directory, websiteSpec.GetBasedir()))
		if err != nil {
			return nil, fmt.Errorf("unable to get absolute path for website %s: %w", websiteSpec.GetBasedir(), err)
		}

		// Generate the website name from the base directory
		websiteName := fmt.Sprintf("websites_%s", strings.ToLower(filepath.Base(absBaseDir)))

		websites = append(websites, Website{
			Name:       websiteName,
			basedir:    websiteSpec.GetBasedir(),
			path:       websiteSpec.Path,
			outputPath: websiteSpec.Build.Output,
			buildCmd:   websiteSpec.Build.Command,
			devCmd:     websiteSpec.Dev.Command,
			devURL:     websiteSpec.Dev.URL,
			indexPage:  websiteSpec.IndexPage,
			errorPage:  websiteSpec.ErrorPage,
		})
	}

	// website validation
	if len(websites) > 0 {
		if !slices.Contains(projectConfig.Preview, preview.Feature_Websites) {
			return nil, fmt.Errorf("project contains websites, but the project does not have the 'websites' preview feature enabled. Please add websites to the preview field of your nitric.yaml file to enable this feature")
		}

		// check for duplicate paths in websites and error
		siteDuplicates := lo.FindDuplicatesBy(websites, func(website Website) string {
			return website.path
		})

		if len(siteDuplicates) > 0 {
			duplicatePaths := lo.Map(siteDuplicates, func(website Website, i int) string {
				return website.path
			})

			return nil, fmt.Errorf("duplicate website paths found: %s", strings.Join(duplicatePaths, ", "))
		}

		// check that there is a root website
		_, found := lo.Find(websites, func(website Website) bool {
			return website.path == "/"
		})
		if !found {
			return nil, fmt.Errorf("no root website found, please add a website with path /")
		}

		// ensure there /api path is not used
		_, found = lo.Find(websites, func(website Website) bool {
			return strings.TrimSuffix(website.path, "/") == "/api"
		})
		if found {
			return nil, fmt.Errorf("path /api is reserved for API rewrites to APIs, please use a different path")
		}
	}

	// create an empty local configuration if none is provided
	if localConfig == nil {
		localConfig = &localconfig.LocalConfiguration{}
	}

	project := &Project{
		Name:        projectConfig.Name,
		Directory:   projectConfig.Directory,
		Preview:     projectConfig.Preview,
		LocalConfig: *localConfig,
		services:    services,
		batches:     batches,
		websites:    websites,
	}

	if len(project.batches) > 0 && !slices.Contains(project.Preview, preview.Feature_BatchServices) {
		return nil, fmt.Errorf("project contains batch services, but the project does not have the 'batch-services' preview feature enabled. Please add batch-services to the preview field of your nitric.yaml file to enable this feature")
	}

	return project, nil
}

// FromFile - Loads a nitric project from a nitric.yaml file
// If no filepath is provided, the default location './nitric.yaml' is used
func FromFile(fs afero.Fs, filepath string) (*Project, error) {
	projectConfig, err := ConfigurationFromFile(fs, filepath)
	if err != nil {
		return nil, fmt.Errorf("error loading nitric.yaml: %w", err)
	}

	// load local configuration
	localConfig, err := localconfig.LocalConfigurationFromFile(fs, "")
	if err != nil {
		return nil, fmt.Errorf("error loading local.nitric.yaml: %w", err)
	}

	return fromProjectConfiguration(projectConfig, localConfig, fs)
}
