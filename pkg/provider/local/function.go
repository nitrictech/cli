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

	nettypes "github.com/containers/podman/v3/libpod/network/types"
	"github.com/containers/podman/v3/pkg/specgen"
	"github.com/hashicorp/consul/sdk/freeport"
	"github.com/opencontainers/runtime-spec/specs-go"

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

	s := specgen.NewSpecGenerator(imageName, false)
	s.Name = imageName + "-" + deploymentName
	s.Labels = l.labels(deploymentName, "function")
	s.PortMappings = []nettypes.PortMapping{
		{
			ContainerPort: functionPort,
			HostPort:      port,
			Protocol:      "tcp",
		},
	}

	s.PublishExposedPorts = true
	s.Expose = map[uint16]string{
		functionPort: "tcp",
	}

	s.Env = map[string]string{
		"LOCAL_SUBSCRIPTIONS": "{}",
		"NITRIC_DEV_VOLUME":   devVolume,
		"MINIO_ENDPOINT":      fmt.Sprintf("http://minio-%s:9000", deploymentName),
		"MINIO_ACCESS_KEY":    "minioadmin",
		"MINIO_SECRET_KEY":    "minioadmin",
	}

	if l.network != "bridge" {
		s.Aliases = map[string][]string{
			l.network: {f.Name()},
		}
		s.ContainerNetworkConfig.CNINetworks = []string{l.network}
	}
	s.Mounts = []specs.Mount{
		{
			Type:        "bind",
			Source:      nitricRunDir,
			Destination: devVolume,
		},
	}

	cID, err := l.cr.CreateWithSpec(s)
	if err != nil {
		return err
	}
	return l.cr.Start(cID)
}
