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

var dockerAvailable bool
var dockerBuildxAvailable bool
var podmanAvailable bool

func DockerAvailable() (bool, error) {
	if dockerAvailable {
		return true, nil
	}

	err := exec.Command("docker", "version").Run()
	if err == nil {
		dockerAvailable = true
	}

	return dockerAvailable, err
}

func DockerBuildxAvailable() (bool, error) {
	if dockerBuildxAvailable {
		return true, nil
	}

	err := exec.Command("docker", "buildx", "version").Run()
	if err == nil {
		dockerBuildxAvailable = true
	}

	return dockerBuildxAvailable, err
}

func PodmanAvailable() (bool, error) {
	if podmanAvailable {
		return true, nil
	}

	err := exec.Command("podman", "version").Run()
	if err == nil {
		podmanAvailable = true
	}

	return podmanAvailable, err
}

type Dependency interface {
	// Check if the dependency is met
	Check() error
	// If the dependency is not met, provide a message to the user to assist in installing it
	Assist() string
}

type ContainerToolDependency struct {
	message string
}

func (c *ContainerToolDependency) Check() error {
	_, err := DockerAvailable()
	if err != nil {
		_, podErr := PodmanAvailable()
		if podErr != nil {
			c.message = "Docker or Podman is required, see https://docs.docker.com/engine/install/ for docker installation instructions"
			return err
		} else {
			// Use podman
			return nil
		}
	}

	_, err = DockerBuildxAvailable()
	if err != nil {
		c.message = "docker buildx is required to run this command. For installation instructions see: https://github.com/docker/buildx"
		return err
	}

	return nil
}

func (c *ContainerToolDependency) Assist() string {
	return c.message
}

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

	missing := make([]Dependency, 0)

	for _, p := range deps {
		err := p.Check()
		if err != nil {
			missing = append(missing, p)
		}
	}

	if len(missing) > 0 {
		for _, p := range missing {
			err := errors.New(p.Assist())
			if err != nil {
				return err
			}
		}
	}

	return nil
}
