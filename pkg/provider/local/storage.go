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

	nettypes "github.com/containers/podman/v3/libpod/network/types"
	"github.com/containers/podman/v3/pkg/specgen"
	"github.com/hashicorp/consul/sdk/freeport"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
)

func (l *local) storage(deploymentName string) error {
	// Ensure the buckets directory exists
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
	err = l.cr.Pull(minioImage)
	if err != nil {
		return err
	}

	s := specgen.NewSpecGenerator(minioImage, false)
	s.Name = "minio-" + deploymentName
	s.Labels = l.labels(deploymentName, "storage")
	s.Command = []string{"minio", "server", "/nitric/buckets", "--console-address", fmt.Sprintf(":%d", consolePort)}
	s.Mounts = []specs.Mount{
		{
			Destination: devVolume,
			Source:      nitricRunDir, // volume.name,
			Type:        "bind",
		},
	}
	s.PortMappings = []nettypes.PortMapping{
		{
			ContainerPort: minioPort,
			HostPort:      port,
			Protocol:      "tcp",
		},
		{
			ContainerPort: minioConsolePort,
			HostPort:      consolePort,
			Protocol:      "tcp",
		},
	}
	s.ContainerNetworkConfig.CNINetworks = []string{l.network}
	s.PublishExposedPorts = true
	s.Expose = map[uint16]string{
		minioPort:        "tcp",
		minioConsolePort: "tcp",
	}
	cID, err := l.cr.CreateWithSpec(s)
	if err != nil {
		return errors.WithMessage(err, "create")
	}
	return l.cr.Start(cID)
}
