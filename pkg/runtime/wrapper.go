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

package runtime

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/nitrictech/cli/pkg/containerengine"
)

const wrapperDockerFile = `
ARG BASE_IMAGE

FROM ${BASE_IMAGE}

ARG MEMBRANE_URI

ADD ${MEMBRANE_URI} /bin/membrane

RUN chmod +x-rw /bin/membrane

CMD [%s]
ENTRYPOINT ["/bin/membrane"]
`

// CmdFromImage - Takes the existing Entrypoint and CMD from and image and makes it a new CMD to be wrapped by a new entrypoint
func cmdFromImage(ce containerengine.ContainerEngine, imageName string) ([]string, error) {
	ii, err := ce.Inspect(imageName)
	if err != nil {
		return nil, err
	}

	// Get the new cmd
	cmds := append(ii.Config.Entrypoint, ii.Config.Cmd...)

	execCmds := make([]string, 0)
	for _, cmd := range cmds {
		execCmds = append(execCmds, fmt.Sprintf("\"%s\"", cmd))
	}

	return execCmds, nil
}

type WrappedBuildInput struct {
	Args       map[string]string
	Dockerfile string
}

func WrapperBuildArgs(imageName string, provider string, version string) (*WrappedBuildInput, error) {
	ce, err := containerengine.Discover()
	if err != nil {
		return nil, err
	}

	cmd, err := cmdFromImage(ce, imageName)
	if err != nil {
		return nil, err
	}

	membraneName := "membrane-" + provider
	fetchFrom := fmt.Sprintf("https://github.com/nitrictech/nitric/releases/download/%s/%s", version, membraneName)

	if version == "latest" {
		fetchFrom = fmt.Sprintf("https://github.com/nitrictech/nitric/releases/%s/download/%s", version, membraneName)
	}

	return &WrappedBuildInput{
		Dockerfile: fmt.Sprintf(wrapperDockerFile, strings.Join(cmd, ",")),
		Args: map[string]string{
			"MEMBRANE_URI": fetchFrom,
			"BASE_IMAGE":   imageName,
		},
	}, nil
}
