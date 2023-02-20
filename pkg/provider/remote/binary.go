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

package remote

import (
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/nitrictech/cli/pkg/output"
	"github.com/nitrictech/cli/pkg/provider/types"
	"github.com/nitrictech/cli/pkg/utils"
)

// Provides a binary remote provider type
type binaryRemoteDeployment struct {
	providerPath string
	*remoteDeployment
}

func (p *binaryRemoteDeployment) startProcess() (*os.Process, error) {
	cmd := exec.Command(p.providerPath)

	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	return cmd.Process, nil
}

func (p *binaryRemoteDeployment) Up(log output.Progress) (*types.Deployment, error) {
	// start the provider command
	process, err := p.startProcess()
	if err != nil {
		return nil, err
	}

	defer process.Kill() //nolint:errcheck

	return p.remoteDeployment.Up(log)
}

func (p *binaryRemoteDeployment) Down(log output.Progress) (*types.Summary, error) {
	// start the provider command
	process, err := p.startProcess()
	if err != nil {
		return nil, err
	}

	defer process.Kill() //nolint:errcheck

	return p.remoteDeployment.Down(log)
}

func isExecAny(mode os.FileMode) bool {
	return mode&0o111 != 0
}

func providerFilePath(prov *provider) string {
	provDir := utils.NitricProviderDir()

	return path.Join(provDir, prov.org, fmt.Sprintf("%s-%s", prov.name, prov.version))
}

func NewBinaryRemoteDeployment(cfc types.ConfigFromCode, sc *StackConfig, prov *provider, envMap map[string]string, opts *types.ProviderOpts) (types.Provider, error) {
	// See if the binary exists in NITRIC_HOME/providers
	providerPath := providerFilePath(prov)

	fi, err := os.Stat(providerPath)
	if err != nil {
		return nil, err
	}

	// Ensure the file is executable
	if !isExecAny(fi.Mode()) {
		return nil, fmt.Errorf("provider exists but is not executable")
	}

	// return a valid binary deployment
	return &binaryRemoteDeployment{
		providerPath: providerPath,
		remoteDeployment: &remoteDeployment{
			cfc: cfc,
			sfc: sc,
		},
	}, nil
}
