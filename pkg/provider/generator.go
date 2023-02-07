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

package provider

import (
	"fmt"

	"github.com/nitrictech/cli/pkg/provider/pulumi"
	"github.com/nitrictech/cli/pkg/provider/remote"
	"github.com/nitrictech/cli/pkg/provider/types"
	"github.com/pterm/pterm"
)

func NewProvider(cfc types.ConfigFromCode, name, provider string, envMap map[string]string, opts *types.ProviderOpts) (types.Provider, error) {
	switch provider {
	case types.Aws, types.Azure, types.Gcp:
		pterm.Warning.Print(fmt.Sprintf(`Provider %s has been deprecated and may be unavailable in future releases.
Provider should be updated to nitric/%s@0.22.0 for more information see: https://nitric.io/blog/new-providers
		`, provider, provider))
		return pulumi.New(cfc, name, provider, envMap, opts)
	default:
		return remote.New(cfc, name, provider, envMap, opts)
	}
}
