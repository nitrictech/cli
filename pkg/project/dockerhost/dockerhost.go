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

package dockerhost

import (
	goruntime "runtime"

	"github.com/nitrictech/nitric/core/pkg/env"
)

func GetInternalDockerHost() string {
	dockerHost := "host.docker.internal"

	if goruntime.GOOS == "linux" {
		host := env.GetEnv("NITRIC_DOCKER_HOST", "172.17.0.1")

		return host.String()
	}

	return dockerHost
}
