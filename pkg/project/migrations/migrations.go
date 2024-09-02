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

package migrations

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

	"github.com/nitrictech/cli/pkg/docker"
	"github.com/nitrictech/cli/pkg/project/runtime"
	"github.com/nitrictech/cli/pkg/project/service"
	"github.com/nitrictech/nitric/core/pkg/logger"
)

type LocalMigration struct {
	DatabaseName     string
	ConnectionString string
}

func migrationImageName(dbName string) string {
	return fmt.Sprintf("%s-migrations", dbName)
}

func BuildMigrationImage(fs afero.Fs, dbName string, buildContext *runtime.RuntimeBuildContext, logs io.Writer) error {
	tempBuildDir := service.GetTempBuildDir()
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
		buildContext.BuildArguments,
		strings.Split(buildContext.IgnoreFileContents, "\n"),
		logs,
	)
	if err != nil {
		return err
	}

	return nil
}

// FIXME: This is essentially a copy of the project.BuildServiceImages function
func BuildMigrationImages(fs afero.Fs, migrationBuildContexts map[string]*runtime.RuntimeBuildContext) (chan service.ServiceBuildUpdate, error) {
	updatesChan := make(chan service.ServiceBuildUpdate)

	maxConcurrentBuilds := make(chan struct{}, min(goruntime.NumCPU(), goruntime.GOMAXPROCS(0)))

	waitGroup := sync.WaitGroup{}

	for dbName, buildContext := range migrationBuildContexts {
		waitGroup.Add(1)

		serviceBuildUpdateWriter := service.NewBuildUpdateWriter(migrationImageName(dbName), updatesChan)

		go func(dbName string, buildContext *runtime.RuntimeBuildContext, writer io.Writer) {
			// Acquire a token by filling the maxConcurrentBuilds channel
			// this will block once the buffer is full
			maxConcurrentBuilds <- struct{}{}

			svcName := migrationImageName(dbName)

			// Start goroutine
			if err := BuildMigrationImage(fs, dbName, buildContext, writer); err != nil {
				updatesChan <- service.ServiceBuildUpdate{
					ServiceName: svcName,
					Err:         err,
					Message:     err.Error(),
					Status:      service.ServiceBuildStatus_Error,
				}
			} else {
				updatesChan <- service.ServiceBuildUpdate{
					ServiceName: svcName,
					Message:     "Build Complete",
					Status:      service.ServiceBuildStatus_Complete,
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

	// Create the container
	containerId, err := client.ContainerCreate(&container.Config{
		Image: imageName,
		Env: []string{
			fmt.Sprintf("NITRIC_DB_NAME=%s", databaseName),
			fmt.Sprintf("DB_URL=%s", connectionString),
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

func RunMigrations(migrations []LocalMigration) error {
	var wg sync.WaitGroup

	errChan := make(chan error, len(migrations))

	for _, mig := range migrations {
		wg.Add(1)

		go func(dbName string, connectionString string) {
			defer wg.Done()

			err := RunMigration(dbName, connectionString)
			if err != nil {
				errChan <- err
			}
		}(mig.DatabaseName, mig.ConnectionString)
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
