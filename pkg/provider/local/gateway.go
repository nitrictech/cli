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
	"bytes"
	"os"
	"path"

	nettypes "github.com/containers/podman/v3/libpod/network/types"
	"github.com/containers/podman/v3/pkg/specgen"
	"github.com/hashicorp/consul/sdk/freeport"
	"github.com/jhoonb/archivex"
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

	s := specgen.NewSpecGenerator(devAPIGatewayImageName, false)
	s.Name = "api-" + apiName + "-" + deploymentName
	s.Labels = l.labels(deploymentName, "gateway")
	s.PortMappings = []nettypes.PortMapping{
		{
			ContainerPort: gatewayPort,
			HostPort:      port,
			Protocol:      "tcp",
		},
	}
	s.PublishExposedPorts = true
	s.Expose = map[uint16]string{
		gatewayPort: "tcp",
	}
	if l.network != "bridge" {
		s.Aliases = map[string][]string{
			l.network: {"api-" + apiName},
		}
		s.ContainerNetworkConfig.CNINetworks = []string{l.network}
	}
	cID, err := l.cr.CreateWithSpec(s)
	if err != nil {
		return err
	}

	// Create staging dir for the build and add the api spec to be loaded by the gateway server
	dirName := createAPIDirectory(apiName)

	err = copyFile(apiDocument, path.Join(dirName, "openapi.json"))
	if err != nil {
		return err
	}

	tar := new(archivex.TarFile)
	apiSpecTarReader := bytes.Buffer{}
	err = tar.CreateWriter(apiName+".tar", &apiSpecTarReader)
	if err != nil {
		return err
	}
	err = tar.AddAll(dirName, false)
	if err != nil {
		return err
	}
	tar.Close()

	// Write the open api file to this api gateway source
	err = l.cr.CopyFromArchive(cID, "/", &apiSpecTarReader)
	if err != nil {
		return err
	}

	return l.cr.Start(cID)

	/* TODO do we need this?
	return containerResult{
		name: api.name,
		type: "api",
		container,
		ports: [port],
	},nil
	*/
}
