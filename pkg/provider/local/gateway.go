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
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/hashicorp/consul/sdk/freeport"

	"github.com/nitrictech/newcli/pkg/utils"
)

const (
	gatewayPort            = 8080
	functionPort           = 9001
	devAPIGatewayImageName = "nitricimages/dev-api-gateway"
)

func createAPIDirectory(apiName string) string {
	os.MkdirAll(path.Join(stagingAPIDir, apiName), 0755)
	return path.Join(stagingAPIDir, apiName)
}

func (l *local) gateway(deploymentName, apiName, apiFile string) error {
	apiDocument := path.Join(l.s.Path(), apiFile)
	ports, err := freeport.Take(1)
	if err != nil {
		return err
	}
	port := uint16(ports[0])

	err = l.cr.Pull(devAPIGatewayImageName)
	if err != nil {
		return err
	}

	cID, err := l.cr.ContainerCreate(&container.Config{
		Image:  devAPIGatewayImageName,
		Labels: l.labels(deploymentName, "gateway"),
		ExposedPorts: nat.PortSet{
			nat.Port(fmt.Sprintf("%d/tcp", gatewayPort)): struct{}{},
		},
	}, &container.HostConfig{
		PortBindings: nat.PortMap{
			nat.Port(fmt.Sprintf("%d/tcp", gatewayPort)): []nat.PortBinding{
				{
					HostPort: fmt.Sprintf("%d", port),
				},
			},
		},
		NetworkMode: container.NetworkMode(l.network),
	}, &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			l.network: {Aliases: []string{"api-" + apiName}},
		},
	}, "api-"+apiName+"-"+deploymentName)
	if err != nil {
		return err
	}

	// Create staging dir for the build and add the api spec to be loaded by the gateway server
	dirName := createAPIDirectory(apiName)
	err = copyFile(apiDocument, path.Join(dirName, "openapi.json"))
	if err != nil {
		return err
	}

	apiSpecTarReader, err := utils.TarReaderFromPath(dirName)
	if err != nil {
		return err
	}

	// Write the open api file to this api gateway source
	err = l.cr.CopyFromArchive(cID, "/", apiSpecTarReader)
	if err != nil {
		return err
	}

	return l.cr.Start(cID)
}
