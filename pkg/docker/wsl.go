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

package docker

import (
	"context"
	"net"
	"strings"
)

// IsDockerRunningInWSL2 checks if Docker is running in WSL2
func IsDockerRunningInWSL2(dockerClient *Docker) (bool, error) {
	// Get system information
	info, err := dockerClient.Info(context.Background())
	if err != nil {
		return false, err
	}

	// Check for WSL2 indicators in the kernel version string
	if strings.HasSuffix(info.KernelVersion, "microsoft-standard-WSL2") {
		return true, nil
	}

	return false, nil
}

// GetNonLoopbackLocalIPForWSL returns the non loopback local IP of the host interface eth0, used for running docker in linux for WSL support
func GetNonLoopbackLocalIPForWSL() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, iface := range interfaces {
		if iface.Name == "eth0" {
			addrs, err := iface.Addrs()
			if err != nil {
				return ""
			}

			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4() != nil {
						return ipnet.IP.String()
					}
				}
			}
		}
	}

	return ""
}
