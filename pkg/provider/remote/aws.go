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
type awsProvider struct {
	*nitricDeployment
}

var awsSupportedRegions = []types.RegionItem{
	{Value: "us-east-1", Description: "N. Virginia, USA"},
	{Value: "us-west-1", Description: "N. California, USA"},
	{Value: "us-west-2", Description: "Oregon, USA"},
	{Value: "eu-west-1", Description: "Ireland"},
	{Value: "eu-central-1", Description: "Frankfurt, Germany"},
	{Value: "ap-southeast-1", Description: "Singapore"},
	{Value: "ap-northeast-1", Description: "Tokyo, Japan"},
	{Value: "ap-southeast-2", Description: "Sydney, Australia"},
	{Value: "ap-northeast-2", Description: "Seoul, South Korea"},
	{Value: "sa-east-1", Description: "Sao Paulo, Brazil"},
	{Value: "cn-north-1", Description: "Beijing, China"},
	{Value: "ap-south-1", Description: "Mumbai, India"},
}

func (g *awsProvider) SupportedRegions() []types.RegionItem {
	return awsSupportedRegions
}

func (g *awsProvider) ToFile() error {
	b, err := yaml.Marshal(g.sfc)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(g.cfc.ProjectDir(), fmt.Sprintf("nitric-%s.yaml", g.sfc.Name)), b, 0o644)
}
