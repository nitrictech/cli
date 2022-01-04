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

// +build linux

package containerengine

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
)

// use docker client to podman socket.
type podman struct {
	*docker
}

var _ ContainerEngine = &podman{}

func newPodman() (ContainerEngine, error) {
	cmd := exec.Command("podman", "--version")
	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	// make sure that the podman-docker package has been installed.
	out := &bytes.Buffer{}
	cmd = exec.Command("docker", "--version")
	cmd.Stdout = out
	err = cmd.Run()
	if err != nil {
		return nil, errors.WithMessage(err, "the podman-docker package is required")
	}
	if !strings.Contains(out.String(), "podman") {
		// this is the actual docker cli installed as well, return an error here and just use docker.
		return nil, errors.New("both podman and docker found, will use docker")
	}

	cmd = exec.Command("sudo", "systemctl", "is-active", "podman.socket")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil || cmd.ProcessState.ExitCode() != 0 {
		fmt.Println("podman.socket not available, starting..")
		cmd = exec.Command("sudo", "systemctl", "start", "podman.socket")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			return nil, err
		}
	}

	//socket := "unix:" + os.Getenv("XDG_RUNTIME_DIR") + "/podman/podman.sock"
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}
	fmt.Println("podman found")

	return &podman{docker: &docker{cli: cli}}, err
}

func (p *podman) Build(dockerfile, path, imageTag, provider string, buildArgs map[string]string) error {
	args := []string{"build", path, "-f", dockerfile, "-t", imageTag, "--progress", "plain", "--build-arg=PROVIDER=" + provider}

	for key, val := range buildArgs {
		args = append(args, "--build-arg="+key+"="+val)
	}
	cmd := exec.Command("podman", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (p *podman) ListImages(stackName, containerName string) ([]Image, error) {
	return p.docker.ListImages(stackName, containerName)
}

func (p *podman) Pull(rawImage string) error {
	return p.docker.Pull(rawImage)
}

func (p *podman) NetworkCreate(name string) error {
	return p.docker.NetworkCreate(name)
}

func (p *podman) ContainerCreate(config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, name string) (string, error) {
	return p.docker.ContainerCreate(config, hostConfig, networkingConfig, name)
}

func (p *podman) Start(nameOrID string) error {
	return p.docker.Start(nameOrID)
}

func (p *podman) CopyFromArchive(nameOrID string, path string, reader io.Reader) error {
	return p.docker.CopyFromArchive(nameOrID, path, reader)
}

func (p *podman) ContainersListByLabel(match map[string]string) ([]types.Container, error) {
	return p.docker.ContainersListByLabel(match)
}

func (p *podman) RemoveByLabel(name, value string) error {
	return p.docker.RemoveByLabel(name, value)
}

func (p *podman) ContainerExec(containerName string, cmd []string, workingDir string) error {
	return p.docker.ContainerExec(containerName, cmd, workingDir)
}
