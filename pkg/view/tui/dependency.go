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

package tui

import (
	"errors"
	"os/exec"

	"github.com/spf13/cobra"
)

type DependencyError struct {
	details string
	assist  string
}

func (d *DependencyError) Error() string {
	return d.details
}

func (d *DependencyError) Assist() string {
	return d.assist
}

type Dependency = func() *DependencyError

func atLeastOne(d ...Dependency) Dependency {
	return func() *DependencyError {
		if len(d) == 0 {
			return nil
		}

		// Extract the preference dependency
		primary := d[0]()
		if primary == nil {
			return nil
		}

		// Check the rest of the dependencies
		for _, dep := range d[1:] {
			if dep() == nil {
				return nil
			}
		}

		return primary
	}
}

var (
	dockerAvailable       bool
	dockerBuildxAvailable bool
	podmanAvailable       bool
)

func DockerAvailable() error {
	if dockerAvailable {
		return nil
	}

	err := exec.Command("docker", "version").Run()
	if err != nil {
		return err
	}

	dockerAvailable = true

	return nil
}

func DockerBuildxAvailable() error {
	if dockerBuildxAvailable {
		return nil
	}

	err := exec.Command("docker", "buildx", "version").Run()
	if err != nil {
		return err
	}

	dockerBuildxAvailable = true

	return nil
}

func PodmanAvailable() error {
	if podmanAvailable {
		return nil
	}

	err := exec.Command("podman", "version").Run()
	if err != nil {
		return err
	}

	podmanAvailable = true

	return nil
}

func RequireDocker() *DependencyError {
	if dockerAvailable {
		return nil
	}

	err := DockerAvailable()
	if err != nil {
		depErr := DependencyError{
			details: err.Error(),
			assist:  "Docker is required, see https://docs.docker.com/engine/install/ for docker installation instructions",
		}

		return &depErr
	}

	err = DockerBuildxAvailable()
	if err != nil {
		depErr := DependencyError{
			details: err.Error(),
			assist:  "docker buildx is required to run this command. For installation instructions see: https://github.com/docker/buildx",
		}

		return &depErr
	}

	dockerBuildxAvailable = true

	return nil
}

func RequirePodman() *DependencyError {
	if podmanAvailable {
		return nil
	}

	err := PodmanAvailable()
	if err != nil {
		depErr := DependencyError{
			details: err.Error(),
			assist:  "Podman is required, see https://docs.docker.com/engine/install/ for docker installation instructions",
		}

		return &depErr
	}

	podmanAvailable = true

	return nil
}

var RequireContainerBuilder = atLeastOne(RequireDocker, RequirePodman)

// AddDependencyCheck - Wraps a cobra command with a pre-run that
// will check for dependencies
func AddDependencyCheck(cmd *cobra.Command, deps ...Dependency) *cobra.Command {
	cmd.PreRun = func(cmd *cobra.Command, args []string) {
		err := checkDependencies(deps...)
		CheckErr(err)
	}

	return cmd
}

func checkDependencies(deps ...Dependency) error {
	if len(deps) == 0 {
		return nil
	}

	for _, p := range deps {
		err := p()
		if err != nil {
			return errors.New(err.Assist())
		}
	}

	return nil
}
