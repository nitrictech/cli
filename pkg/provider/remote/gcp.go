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

	"github.com/nitrictech/cli/pkg/provider/types"
)

// TODO: Move this into remote provider logic
// A provider should be able to produce it's own template for a valid stack specification as stack specifications may change over time
type gcpProvider struct {
	*nitricDeployment
}

var gcpSupportedRegions = []types.RegionItem{
	{Value: "us-west2", Description: "US West (Los Angeles)"},
	{Value: "us-west3", Description: "US West (Salt Lake City)"},
	{Value: "us-west4", Description: "US West (Las Vegas)"},
	{Value: "us-central1", Description: "US Central (Iowa)"},
	{Value: "us-east1", Description: "US East (South Carolina)"},
	{Value: "us-east4", Description: "US East (Northern Virginia)"},
	{Value: "europe-west1", Description: "Europe West (Belgium)"},
	{Value: "europe-west2", Description: "Europe West (London)"},
	{Value: "asia-east1", Description: "Asia East (Taiwan)"},
	{Value: "australia-southeast1", Description: "Australia Southeast (Sydney)"},
}

func (g *gcpProvider) SupportedRegions() []types.RegionItem {
	return gcpSupportedRegions
}

func (g *gcpProvider) ToFile() error {
	b, err := yaml.Marshal(g.sfc)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(g.cfc.ProjectDir(), fmt.Sprintf("nitric-%s.yaml", g.sfc.Name)), b, 0o644)
}
