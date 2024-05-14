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
	"fmt"
	"io"
	"os"
	goruntime "runtime"
	"strings"
	"sync"

	"github.com/spf13/afero"

	"github.com/nitrictech/cli/pkg/docker"
	"github.com/nitrictech/cli/pkg/project/runtime"
	"github.com/nitrictech/nitric/core/pkg/logger"
)

func BuildMigrationImage(fs afero.Fs, dbName string, buildContext *runtime.RuntimeBuildContext, logs io.Writer) error {
	svcName := fmt.Sprintf("%s-migrations", dbName)

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
func BuildMigrationImages(fs afero.Fs, migrationBuildContexts map[string]*runtime.RuntimeBuildContext) (chan ServiceBuildUpdate, error) {
	updatesChan := make(chan ServiceBuildUpdate)

	maxConcurrentBuilds := make(chan struct{}, min(goruntime.NumCPU(), goruntime.GOMAXPROCS(0)))

	waitGroup := sync.WaitGroup{}

	for dbName, buildContext := range migrationBuildContexts {
		waitGroup.Add(1)

		serviceBuildUpdateWriter := &serviceBuildUpdateWriter{
			buildUpdateChan: updatesChan,
			serviceName:     fmt.Sprintf("%s-migrations", dbName),
		}

		go func(dbName string, buildContext *runtime.RuntimeBuildContext, writer io.Writer) {
			// Acquire a token by filling the maxConcurrentBuilds channel
			// this will block once the buffer is full
			maxConcurrentBuilds <- struct{}{}

			svcName := fmt.Sprintf("%s-migrations", dbName)

			// Start goroutine
			if err := BuildMigrationImage(fs, dbName, buildContext, writer); err != nil {
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
