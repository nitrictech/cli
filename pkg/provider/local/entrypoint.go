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
	"strings"

	nettypes "github.com/containers/podman/v3/libpod/network/types"
	"github.com/containers/podman/v3/pkg/specgen"
	"github.com/hashicorp/consul/sdk/freeport"
	"github.com/nitrictech/newcli/pkg/stack"
	"github.com/nitrictech/newcli/pkg/utils"
)

const (
	httpPort        = 80
	nginxConfigFile = "nginx.conf"
)

func createNginxConfig(e *stack.Entrypoint, s *stack.Stack) (string, error) {
	configLines := []string{
		"events {}",
		"http {",
		"include mime.types;",
		"server {",
	}

	for location, p := range e.Paths {
		switch p.Type {
		case "site":
			configLines = append(configLines, "location "+location+" {")
			configLines = append(configLines, "root /www/"+p.Target+";")
			configLines = append(configLines, "try_files $uri $uri/ /index.html;")
		case "api":
			configLines = append(configLines, "location "+location+" {")
			configLines = append(configLines, fmt.Sprintf("proxy_pass http://api-%s:%d;", p.Target, gatewayPort))
		case "function", "container":
			configLines = append(configLines, "location "+location+" {")
			configLines = append(configLines, fmt.Sprintf("proxy_pass http://%s:%d;", p.Target, functionPort))
		default:
			return "", fmt.Errorf("endpoint path %s type incorrect %s", location, p.Type)
		}
	}

	configLines = append(configLines, "}", "}")

	return strings.Join(configLines, "\n"), nil
}

func (l *local) entrypoint(deploymentName, entrypointName string, e *stack.Entrypoint) error {
	ports, err := freeport.Take(1)
	if err != nil {
		return err
	}

	port := uint16(ports[0])

	err = l.cr.Pull("nginx")
	if err != nil {
		return err
	}

	s := specgen.NewSpecGenerator("nginx", false)
	s.Name = "entry-" + entrypointName + "-" + deploymentName
	s.Labels = l.labels(deploymentName, "gateway")
	if l.network != "bridge" {
		//		s.Aliases = map[string][]string{
		//			l.network: {"api-" + apiName},
		//		}
		s.ContainerNetworkConfig.CNINetworks = []string{l.network}
	}
	s.PortMappings = []nettypes.PortMapping{
		{
			ContainerPort: httpPort,
			HostPort:      port,
			Protocol:      "tcp",
		},
	}
	s.PublishExposedPorts = true
	s.Expose = map[uint16]string{
		httpPort: "tcp",
	}

	cID, err := l.cr.CreateWithSpec(s)
	if err != nil {
		return err
	}

	config, err := createNginxConfig(e, l.s)
	if err != nil {
		return err
	}

	configTarReader, err := utils.TarReaderFromString(nginxConfigFile, config)
	if err != nil {
		return err
	}

	err = l.cr.CopyFromArchive(cID, "/etc/nginx/", configTarReader)
	if err != nil {
		return err
	}

	err = l.cr.Start(cID)
	if err != nil {
		return err
	}

	for k, s := range l.s.Sites {
		err = l.cr.ContainerExec(cID, []string{"mkdir", "-p", "/www/" + k}, "/")
		if err != nil {
			return err
		}

		asssetTarReader, err := utils.TarReaderFromPath(s.AssetPath)
		if err != nil {
			return err
		}
		err = l.cr.CopyFromArchive(cID, "/www/"+k, asssetTarReader)
		if err != nil {
			return err
		}

		err = l.cr.ContainerExec(cID, []string{"chmod", "-R", "755", "/www/"}, "/")
		if err != nil {
			return err
		}
	}

	return nil
}
