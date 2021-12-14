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

package build

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

type ContainerRuntime interface {
	Build(dockerfile, path, imageTag, provider string, buildArgs map[string]string) error
}

func DiscoverContainerRuntime() (ContainerRuntime, error) {
	cmd := exec.Command("podman", "--version")
	err := cmd.Run()
	if err == nil {
		return &podman{}, nil
	}
	cmd = exec.Command("docker", "--version")
	err = cmd.Run()
	if err == nil {
		return &docker{}, nil
	}
	return nil, errors.New("neither podman nor docker found")
}

type podman struct{}

var _ ContainerRuntime = &podman{}

func (p *podman) Build(dockerfile, path, imageTag, provider string, buildArgs map[string]string) error {
	args := []string{"build", path, "-f", dockerfile, "-t", imageTag, "--progress", "plain", "--build-arg=PROVIDER=" + provider}

	for key, val := range buildArgs {
		args = append(args, "--build-arg="+key+"="+val)
	}
	fmt.Println("podman", args)
	cmd := exec.Command("podman", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type docker struct{}

var _ ContainerRuntime = &docker{}

func (d *docker) Build(dockerfile, path, imageTag, provider string, buildArgs map[string]string) error {
	args := []string{"build", path, "-f", dockerfile, "-t", imageTag, "--progress", "plain", "--build-arg PROVIDER=" + provider}

	for key, val := range buildArgs {
		args = append(args, "--build-arg "+key+"="+val)
	}
	fmt.Println("docker", args)
	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = append(os.Environ(), "DOCKER_BUILDKIT=1")
	return cmd.Run()
}
