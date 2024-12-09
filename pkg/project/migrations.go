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
	"fmt"
	"io"
	"os"
	goruntime "runtime"
	"strings"
	"sync"

	"github.com/docker/docker/api/types/container"
	"github.com/spf13/afero"

	"github.com/nitrictech/cli/pkg/cloud/sql"
	"github.com/nitrictech/cli/pkg/collector"
	"github.com/nitrictech/cli/pkg/docker"
	"github.com/nitrictech/cli/pkg/project/dockerhost"
	"github.com/nitrictech/cli/pkg/project/runtime"
	"github.com/nitrictech/nitric/core/pkg/logger"
	resourcespb "github.com/nitrictech/nitric/core/pkg/proto/resources/v1"
)

type LocalMigration struct {
	DatabaseName     string
	ConnectionString string
}

type DatabaseMigrationState struct {
	*LocalMigration
}

func migrationImageName(dbName string) string {
	return fmt.Sprintf("%s-migrations", dbName)
}

func BuildAndRunMigrations(fs afero.Fs, servers map[string]*sql.DatabaseServer, databasesToMigrate map[string]*resourcespb.SqlDatabaseResource, useBuilder bool) error {
	serviceRequirements := collector.MakeDatabaseServiceRequirements(databasesToMigrate)

	migrationImageContexts, err := collector.GetMigrationImageBuildContexts(serviceRequirements, []*collector.BatchRequirements{}, fs)
	if err != nil {
		return fmt.Errorf("failed to get migration image build contexts: %w", err)
	}

	if len(migrationImageContexts) > 0 {
		updates, err := BuildMigrationImages(fs, migrationImageContexts, useBuilder)
		if err != nil {
			return err
		}

		// wait for updates to complete
		for update := range updates {
			if update.Err != nil {
				return fmt.Errorf("failed to build migration image: %w", update.Err)
			}
		}

		err = RunMigrations(servers)
		if err != nil {
			return fmt.Errorf("failed to run migrations: %w", err)
		}
	}

	return nil
}

func BuildMigrationImage(fs afero.Fs, dbName string, buildContext *runtime.RuntimeBuildContext, logs io.Writer, useBuilder bool) error {
	tempBuildDir := GetTempBuildDir()
	svcName := migrationImageName(dbName)

	dockerClient, err := docker.New()
	if err != nil {
		return err
	}

	err = fs.MkdirAll(tempBuildDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to create temporary build directory %s: %w", tempBuildDir, err)
	}

	tmpDockerFile, err := afero.TempFile(fs, tempBuildDir, fmt.Sprintf("%s-*.dockerfile", svcName))
	if err != nil {
		return fmt.Errorf("unable to create temporary dockerfile for service %s: %w", svcName, err)
	}

	if err := afero.WriteFile(fs, tmpDockerFile.Name(), []byte(buildContext.DockerfileContents), os.ModePerm); err != nil {
		return fmt.Errorf("unable to write temporary dockerfile for service %s: %w", svcName, err)
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
		buildContext.BaseDirectory,
		svcName,
		docker.WithBuildArgs(buildContext.BuildArguments),
		docker.WithExcludes(strings.Split(buildContext.IgnoreFileContents, "\n")),
		docker.WithLogger(logs),
		docker.WithBuilder(useBuilder),
	)
	if err != nil {
		return err
	}

	return nil
}

// FIXME: This is essentially a copy of the project.BuildServiceImages function
func BuildMigrationImages(fs afero.Fs, migrationBuildContexts map[string]*runtime.RuntimeBuildContext, useBuilder bool) (chan ServiceBuildUpdate, error) {
	updatesChan := make(chan ServiceBuildUpdate)

	maxConcurrentBuilds := make(chan struct{}, min(goruntime.NumCPU(), goruntime.GOMAXPROCS(0)))

	waitGroup := sync.WaitGroup{}

	for dbName, buildContext := range migrationBuildContexts {
		waitGroup.Add(1)

		serviceBuildUpdateWriter := NewBuildUpdateWriter(migrationImageName(dbName), updatesChan)

		go func(dbName string, buildContext *runtime.RuntimeBuildContext, writer io.Writer) {
			// Acquire a token by filling the maxConcurrentBuilds channel
			// this will block once the buffer is full
			maxConcurrentBuilds <- struct{}{}

			svcName := migrationImageName(dbName)

			// Start goroutine
			if err := BuildMigrationImage(fs, dbName, buildContext, writer, useBuilder); err != nil {
				updatesChan <- ServiceBuildUpdate{
					ServiceName: svcName,
					Err:         err,
					Message:     err.Error(),
					Status:      ServiceBuildStatus_Error,
				}
			} else {
				updatesChan <- ServiceBuildUpdate{
					ServiceName: svcName,
					Message:     "Build Complete",
					Status:      ServiceBuildStatus_Complete,
				}
			}

			// release our lock
			<-maxConcurrentBuilds

			waitGroup.Done()
		}(dbName, buildContext, serviceBuildUpdateWriter)
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

// Run the migrations
func RunMigration(databaseName string, connectionString string) error {
	client, err := docker.New()
	if err != nil {
		return err
	}

	// Run the migrations
	imageName := migrationImageName(databaseName)

	// Update connection string for docker host...
	dockerHost := dockerhost.GetInternalDockerHost()

	dockerConnectionString := strings.Replace(connectionString, "localhost", dockerHost, 1)

	// Create the container
	containerId, err := client.ContainerCreate(&container.Config{
		Image: imageName,
		Env: []string{
			fmt.Sprintf("NITRIC_DB_NAME=%s", databaseName),
			fmt.Sprintf("DB_URL=%s", dockerConnectionString),
		},
	}, &container.HostConfig{
		AutoRemove: true,
	}, nil, fmt.Sprintf("nitric-%s-migrations-local-sql", databaseName))
	if err != nil {
		return err
	}

	// Start the container
	err = client.ContainerStart(context.Background(), containerId, container.StartOptions{})
	if err != nil {
		return err
	}

	return nil
}

func RunMigrations(servers map[string]*sql.DatabaseServer) error {
	var wg sync.WaitGroup

	errChan := make(chan error, len(servers))

	for name, mig := range servers {
		wg.Add(1)

		go func(dbName string, connectionString string) {
			defer wg.Done()

			err := RunMigration(dbName, connectionString)
			if err != nil {
				errChan <- err
			}
		}(name, mig.ConnectionString)
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}
