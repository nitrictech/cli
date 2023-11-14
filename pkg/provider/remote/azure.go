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

	"github.com/AlecAivazis/survey/v2"
	"gopkg.in/yaml.v2"

	"github.com/nitrictech/cli/pkg/provider/types"
)

// TODO: Move this into remote provider logic
// A provider should be able to produce it's own template for a valid stack specification as stack specifications may change over time
type azureProvider struct {
	*nitricDeployment
}

var azureSupportedRegions = []types.RegionItem{
	{Value: "canadacentral", Description: "Central Canada"},
	{Value: "eastasia", Description: "East Asia"},
	{Value: "eastus", Description: "Eastern United States"},
	{Value: "eastus2", Description: "Eastern United States 2"},
	{Value: "germanywestcentral", Description: "Central Germany"},
	{Value: "japaneast", Description: "Japan East"},
	{Value: "northeurope", Description: "Northern Europe"},
	{Value: "uksouth", Description: "Southern United Kingdom"},
	{Value: "westeurope", Description: "Western Europe"},
	{Value: "westus", Description: "Western United States"},
}

func (g *azureProvider) SupportedRegions() []types.RegionItem {
	return azureSupportedRegions
}

func (g *azureProvider) ToFile() error {
	b, err := yaml.Marshal(g.sfc)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(g.cfc.ProjectDir(), fmt.Sprintf("nitric-%s.yaml", g.sfc.Name)), b, 0o644)
}

func (a *azureProvider) AskAndSave() error {
	answers := struct {
		Region     string
		Org        string
		AdminEmail string
	}{}
	qs := []*survey.Question{
		{
			Name: "org",
			Prompt: &survey.Input{
				Message: "Provide the organisation to associate with the API",
			},
		},
		{
			Name: "adminEmail",
			Prompt: &survey.Input{
				Message: "Provide the adminEmail to associate with the API",
			},
		},
	}

	err := survey.Ask(qs, &answers)
	if err != nil {
		return err
	}

	a.sfc.Props["region"] = answers.Region
	a.sfc.Props["adminemail"] = answers.AdminEmail
	a.sfc.Props["org"] = answers.Org

	b, err := yaml.Marshal(a.sfc)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(a.cfc.ProjectDir(), fmt.Sprintf("nitric-%s.yaml", a.sfc.Name)), b, 0o644)
}
