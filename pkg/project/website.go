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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Website struct {
	Name string

	// the base directory for the website source files
	basedir string

	// the build command to build the website
	buildCmd string

	// the path to the website source files
	outputPath string

	// index page for the website
	indexPage string

	// error page for the website
	errorPage string
}

func (s *Website) GetOutputPath() string {
	return filepath.Join(s.basedir, s.outputPath)
}

func (s *Website) GetAbsoluteOutputPath() (string, error) {
	return filepath.Abs(s.GetOutputPath())
}

// Run - runs the website using the provided command. TODO
func (s *Website) Run(stop <-chan bool, updates chan<- ServiceRunUpdate, env map[string]string) error {
	return nil
}

// Build - builds the website using the provided command
func (s *Website) Build(updates chan ServiceBuildUpdate, env map[string]string) error {
	if s.buildCmd == "" {
		return fmt.Errorf("no build command provided for website %s", s.basedir)
	}

	commandParts := strings.Split(s.buildCmd, " ")
	cmd := exec.Command(
		commandParts[0],
		commandParts[1:]...,
	)

	cmd.Env = append([]string{}, os.Environ()...)
	cmd.Dir = s.basedir

	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	cmd.Stdout = &serviceBuildUpdateWriter{
		buildUpdateChan: updates,
		serviceName:     s.Name,
	}

	cmd.Stderr = &serviceBuildUpdateWriter{
		buildUpdateChan: updates,
		serviceName:     s.Name,
	}

	errChan := make(chan error)

	go func() {
		err := cmd.Start()
		if err != nil {
			errChan <- fmt.Errorf("error building website %s: %w", s.Name, err)
		} else {
			updates <- ServiceBuildUpdate{
				ServiceName: s.Name,
				Status:      ServiceBuildStatus_InProgress,
				Message:     fmt.Sprintf("building website %s", s.GetOutputPath()),
			}
		}

		err = cmd.Wait()
		if err != nil {
			// provide runtime errors as a run update rather than as a fatal error
			updates <- ServiceBuildUpdate{
				ServiceName: s.Name,
				Status:      ServiceBuildStatus_Error,
				Err:         err,
			}
		}

		errChan <- nil
	}()

	// go func(cmd *exec.Cmd) {
	// 	<-stop

	// 	err := cmd.Process.Signal(syscall.SIGTERM)
	// 	if err != nil {
	// 		_ = cmd.Process.Kill()
	// 	}
	// }(cmd)

	err := <-errChan

	if err != nil {
		updates <- ServiceBuildUpdate{
			ServiceName: s.Name,
			Status:      ServiceBuildStatus_Error,
			Err:         err,
		}
	} else {
		// updates <- ServiceBuildUpdate{
		// 	ServiceName: s.Name,
		// 	Status:      ServiceBuildStatus_Complete,
		// 	Message:     fmt.Sprintf("website %s built successfully", s.Name),
		// }
	}

	return err
}
