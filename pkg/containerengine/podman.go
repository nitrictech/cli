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

package containerengine

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"

	"github.com/nitrictech/newcli/pkg/utils"
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

	//export DOCKER_HOST=unix:///run/user/1000/podman/podman.sock
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}
	// Test the connection
	_, err = cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		fmt.Println("podman socket not running, please execute 'sudo systemctl start podman.socket'")
		return nil, err
	}
	fmt.Println("podman found")

	return &podman{docker: &docker{cli: cli}}, err
}

func (p *podman) Type() string {
	return "podman"
}

func (p *podman) Build(dockerfile, path, imageTag string, buildArgs map[string]string, excludes []string) error {
	return p.docker.Build(dockerfile, path, imageTag, buildArgs, excludes)
}

func (p *podman) ListImages(stackName, containerName string) ([]Image, error) {
	return p.docker.ListImages(stackName, containerName)
}

func (p *podman) ImagePull(rawImage string) error {
	return p.docker.ImagePull(rawImage)
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

func (p *podman) Stop(nameOrID string, timeout *time.Duration) error {
	return p.docker.Stop(nameOrID, timeout)
}

func (p *podman) ContainerWait(containerID string, condition container.WaitCondition) (<-chan container.ContainerWaitOKBody, <-chan error) {
	return p.docker.ContainerWait(containerID, condition)
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

func (p *podman) Logger(stackPath string) ContainerLogger {
	logPath, _ := utils.NewNitricLogFile(stackPath)
	return &jsonfile{logPath: logPath}
}

type jsonfile struct {
	logPath string
}

func (j *jsonfile) Config() *container.LogConfig {
	return &container.LogConfig{
		Type: "json-file",
		Config: map[string]string{
			"path": j.logPath,
		},
	}
}

func (j *jsonfile) Stop() error  { return nil }
func (j *jsonfile) Start() error { return nil }
