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
	"github.com/spf13/viper"
)

var MockEngine ContainerEngine

type Image struct {
	ID         string `yaml:"id"`
	Repository string `yaml:"repository,omitempty"`
	Tag        string `yaml:"tag,omitempty"`
	CreatedAt  string `yaml:"createdAt,omitempty"`
}

type ContainerEngine interface {
	Build(dockerfile, path, imageTag string, buildArgs map[string]string) error
	ListImages(stackName, containerName string) ([]Image, error)
	ImagePull(rawImage string) error
	NetworkCreate(name string) error
	ContainerCreate(config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, name string) (string, error)
	Start(nameOrID string) error
	Stop(nameOrID string, timeout *time.Duration) error
	ContainerWait(containerID string, condition container.WaitCondition) (<-chan container.ContainerWaitOKBody, <-chan error)
	CopyFromArchive(nameOrID string, path string, reader io.Reader) error
	ContainersListByLabel(match map[string]string) ([]types.Container, error)
	RemoveByLabel(name, value string) error
	ContainerExec(containerName string, cmd []string, workingDir string) error
}

func Discover() (ContainerEngine, error) {
	if MockEngine != nil {
		// for unit testing
		return MockEngine, nil
	}
	pm, err := newPodman()
	if err == nil {
		return pm, nil
	}
	dk, err := newDocker()
	if err == nil {
		return dk, nil
	}
	return nil, errors.New("neither podman nor docker found")
}

func buildTimeout() time.Duration {
	return viper.GetDuration("build_timeout")
}
