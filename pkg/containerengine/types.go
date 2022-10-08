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
	"errors"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
)

var DiscoveredEngine ContainerEngine

type Image struct {
	ID         string `yaml:"id"`
	Repository string `yaml:"repository,omitempty"`
	Tag        string `yaml:"tag,omitempty"`
	CreatedAt  string `yaml:"createdAt,omitempty"`
}

type ContainerLogger interface {
	Start() error
	Stop() error
	Config() *container.LogConfig
}

type ContainerEngine interface {
	Type() string
	Build(dockerfile, path, imageTag string, buildArgs map[string]string, excludes []string) error
	ListImages(stackName, containerName string) ([]Image, error)
	Inspect(imageName string) (types.ImageInspect, error)
	ImagePull(rawImage string, opts types.ImagePullOptions) error
	ContainerCreate(config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, name string) (string, error)
	Start(nameOrID string) error
	Stop(nameOrID string, timeout *time.Duration) error
	ContainerWait(containerID string, condition container.WaitCondition) (<-chan container.ContainerWaitOKBody, <-chan error)
	RemoveByLabel(labels map[string]string) error
	ContainerLogs(containerID string, opts types.ContainerLogsOptions) (io.ReadCloser, error)
	Logger(stackPath string) ContainerLogger
	Version() string
}

func Discover() (ContainerEngine, error) {
	if DiscoveredEngine != nil {
		return DiscoveredEngine, nil
	}

	dk, err := newDocker()
	if err == nil {
		DiscoveredEngine = dk
		return dk, nil
	}

	return nil, errors.New("Nitric relies on Docker to containerize your project. Please refer to the installation instructions - https://nitric.io/docs/installation")
}

func Cli(cc *container.Config, hc *container.HostConfig) string {
	cmd := []string{"docker", "run"}

	if cc.Tty {
		cmd = append(cmd, "-t")
	}

	if len(cc.Entrypoint) > 0 {
		cmd = append(cmd, "--entrypoint")
		cmd = append(cmd, cc.Entrypoint...)
	}

	if cc.WorkingDir != "" {
		cmd = append(cmd, "-w", cc.WorkingDir)
	}

	for _, v := range hc.Mounts {
		cmd = append(cmd, "-v", v.Source+":"+v.Target)
	}

	for _, e := range cc.Env {
		cmd = append(cmd, "-e", e)
	}

	for _, h := range hc.ExtraHosts {
		cmd = append(cmd, "--add-host", h)
	}

	if hc.AutoRemove {
		cmd = append(cmd, "--rm")
	}

	if cc.AttachStdout {
		cmd = append(cmd, "-a", "stdout")
	}

	if cc.AttachStdin {
		cmd = append(cmd, "-a", "stdin")
	}

	if cc.AttachStderr {
		cmd = append(cmd, "-a", "stderr")
	}

	cmd = append(cmd, cc.Image)
	cmd = append(cmd, cc.Cmd...)

	return strings.Join(cmd, " ")
}
