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
	Build(dockerfile, path, imageTag string, buildArgs map[string]string, excludes []string, buildLogger io.Writer) error
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

type ContainerEngineArgs struct {
	Output io.Writer
	Errors io.Writer
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

	return nil, errors.New("Nitric relies on Docker to be running before containerizing your project. Please refer to our installation instructions - https://nitric.io/docs/installation")
}
