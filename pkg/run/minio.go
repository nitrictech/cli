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
package run

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/hashicorp/consul/sdk/freeport"
	"github.com/pkg/errors"

	"github.com/nitrictech/newcli/pkg/containerengine"
)

type MinioServer struct {
	dir     string
	name    string
	cid     string
	ce      containerengine.ContainerEngine
	apiPort int
}

const (
	minioImage       = "minio/minio:latest"
	devVolume        = "/nitric/"
	runPerm          = os.ModePerm // NOTE: octal notation is important here!!!
	LabelRunID       = "io.nitric-run-id"
	LabelStackName   = "io.nitric-stack"
	LabelType        = "io.nitric-type"
	minioPort        = 9000
	minioConsolePort = 9001 // TODO: Determine if we would like to expose the console

)

// StartMinio -
func (m *MinioServer) Start() error {
	runDir, err := filepath.Abs(m.dir)
	if err != nil {
		return err
	}

	err = os.MkdirAll(runDir, runPerm)
	if err != nil {
		return errors.WithMessage(err, "mkdirall")
	}

	// TODO: Create new buckets on the fly
	//for bName := range l.s.Buckets {
	//	os.MkdirAll(path.Join(nitricRunDir, "buckets", bName), runPerm)
	//}
	ports, err := freeport.Take(2)
	if err != nil {
		return errors.WithMessage(err, "freeport.Take")
	}

	port := uint16(ports[0])
	consolePort := uint16(ports[1])

	err = m.ce.ImagePull(minioImage)
	if err != nil {
		return err
	}

	cID, err := m.ce.ContainerCreate(&container.Config{
		Image: minioImage,
		Cmd:   []string{"minio", "server", "/nitric/buckets", "--console-address", fmt.Sprintf(":%d", consolePort)},
		ExposedPorts: nat.PortSet{
			nat.Port(fmt.Sprintf("%d/tcp", minioPort)):        struct{}{},
			nat.Port(fmt.Sprintf("%d/tcp", minioConsolePort)): struct{}{},
		},
	}, &container.HostConfig{
		AutoRemove: true,
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
				Source: runDir,
				Type:   mount.TypeBind,
				Target: devVolume,
			},
		},
		LogConfig:   *m.ce.Logger(m.dir).Config(),
		NetworkMode: container.NetworkMode("bridge"),
	}, &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{},
	}, "minio-"+m.name)
	if err != nil {
		return err
	}
	m.cid = cID
	m.apiPort = minioPort

	return m.ce.Start(cID)
}

func (m *MinioServer) GetApiPort() int {
	return m.apiPort
}

func (m *MinioServer) Stop() error {
	return m.ce.Stop(m.cid, nil)
}

func NewMinio(dir string, name string) (*MinioServer, error) {
	ce, err := containerengine.Discover()
	if err != nil {
		return nil, err
	}

	return &MinioServer{
		ce:   ce,
		dir:  dir,
		name: name,
	}, nil
}
