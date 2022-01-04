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

package local

import (
	"fmt"
	"path"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/hashicorp/consul/sdk/freeport"

	"github.com/nitrictech/newcli/pkg/stack"
)

func containerSubscriptions(s *stack.Stack) (map[string][]string, error) {
	// TODO implement me
	return map[string][]string{}, nil
}

func (l *local) function(deploymentName string, f *stack.Function) error {
	nitricRunDir := path.Join(l.s.Path(), runDir)
	ports, err := freeport.Take(1)
	if err != nil {
		return err
	}

	port := uint16(ports[0])
	imageName := f.ImageTagName(l.s, l.t.Provider)

	cID, err := l.cr.ContainerCreate(&container.Config{
		Image:  imageName,
		Labels: l.labels(deploymentName, "function"),
		ExposedPorts: nat.PortSet{
			nat.Port(fmt.Sprintf("%d/tcp", functionPort)): struct{}{},
		},
		Env: []string{
			"LOCAL_SUBSCRIPTIONS={}",
			"NITRIC_DEV_VOLUME=" + devVolume,
			"MINIO_ENDPOINT=" + fmt.Sprintf("http://minio-%s:9000", deploymentName),
			"MINIO_ACCESS_KEY=minioadmin",
			"MINIO_SECRET_KEY=minioadmin",
		},
	}, &container.HostConfig{
		PortBindings: nat.PortMap{
			nat.Port(fmt.Sprintf("%d/tcp", functionPort)): []nat.PortBinding{
				{
					HostPort: fmt.Sprintf("%d/tcp", port),
				},
			},
		},
		Mounts: []mount.Mount{
			{
				Type:   "bind",
				Source: nitricRunDir,
				Target: devVolume,
			},
		},
		NetworkMode: container.NetworkMode(l.network),
	}, &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			l.network: {Aliases: []string{f.Name()}},
		},
	}, imageName+"-"+deploymentName)
	if err != nil {
		return err
	}

	return l.cr.Start(cID)
}
