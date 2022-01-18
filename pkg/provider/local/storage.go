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
	"os"
	"path"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/hashicorp/consul/sdk/freeport"
	"github.com/pkg/errors"
)

func (l *local) storage(deploymentName string) error {
	nitricRunDir := path.Join(l.s.Path(), runDir)
	os.MkdirAll(nitricRunDir, runPerm)
	for bName := range l.s.Buckets {
		os.MkdirAll(path.Join(nitricRunDir, "buckets", bName), runPerm)
	}
	ports, err := freeport.Take(2)
	if err != nil {
		return errors.WithMessage(err, "freeport.Take")
	}

	port := uint16(ports[0])
	consolePort := uint16(ports[1])

	minioImage := "minio/minio:latest"
	err = l.cr.ImagePull(minioImage)
	if err != nil {
		return err
	}

	cID, err := l.cr.ContainerCreate(&container.Config{
		Image:  minioImage,
		Cmd:    []string{"minio", "server", "/nitric/buckets", "--console-address", fmt.Sprintf(":%d", consolePort)},
		Labels: l.labels(deploymentName, "storage"),
		ExposedPorts: nat.PortSet{
			nat.Port(fmt.Sprintf("%d/tcp", minioPort)):        struct{}{},
			nat.Port(fmt.Sprintf("%d/tcp", minioConsolePort)): struct{}{},
		},
	}, &container.HostConfig{
		PortBindings: nat.PortMap{
			nat.Port(fmt.Sprintf("%d/tcp", minioPort)): []nat.PortBinding{
				{
					HostPort: fmt.Sprintf("%d", port),
				},
			},
			nat.Port(fmt.Sprintf("%d/tcp", minioConsolePort)): []nat.PortBinding{
				{
					HostPort: fmt.Sprintf("%d", consolePort),
				},
			},
		},
		Mounts: []mount.Mount{
			{
				Source: nitricRunDir,
				Type:   mount.TypeBind,
				Target: devVolume,
			},
		},
		NetworkMode: container.NetworkMode(l.network),
	}, &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{},
	}, "minio-"+deploymentName)
	if err != nil {
		return err
	}
	return l.cr.Start(cID)
}
