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
	"path/filepath"

	"gopkg.in/yaml.v2"
)

// TODO: Move this into remote provider logic
// A provider should be able to produce it's own template for a valid stack specification as stack specifications may change over time
type customProvider struct {
	*nitricDeployment
}

// FIXME: Prompting to create a new stack state and memory and persisting to a file should be two separte functions
func (g *customProvider) AskAndSave() error {
	b, err := yaml.Marshal(g.sfc)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(g.cfc.ProjectDir(), fmt.Sprintf("nitric-%s.yaml", g.sfc.Name)), b, 0o644)
}
