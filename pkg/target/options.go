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

package target

import (
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nitrictech/newcli/pkg/pflagext"
)

const (
	Aws             = "aws"
	Azure           = "azure"
	Gcp             = "gcp"
	Digitalocean    = "digitalocean"
	DefaultTarget   = Aws
	DefaultProvider = Aws
)

var (
	target    string
	provider  string
	region    string
	Providers = []string{Aws, Azure, Gcp, Digitalocean}
)

func FromOptions() *Target {
	t := Target{}
	if target == "" {
		target = DefaultTarget
		t.Provider = DefaultProvider
		if provider != "" {
			t.Provider = provider
		}
		if region != "" {
			t.Region = region
		}
	} else {
		targets := map[string]Target{}
		cobra.CheckErr(mapstructure.Decode(viper.GetStringMap("targets"), &targets))
		t = targets[target]
	}
	return &t
}

func AddOptions(cmd *cobra.Command, providerOnly bool) error {
	targetsMap := viper.GetStringMap("targets")
	targets := []string{}
	for k := range targetsMap {
		targets = append(targets, k)
	}

	cmd.Flags().VarP(pflagext.NewStringEnumVar(&target, targets, Aws), "target", "t", "use this to refer to a target in the configuration")
	err := cmd.RegisterFlagCompletionFunc("target", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return targets, cobra.ShellCompDirectiveDefault
	})
	if err != nil {
		return err
	}

	cmd.Flags().VarP(pflagext.NewStringEnumVar(&provider, Providers, Aws), "provider", "p", "the provider to deploy to")
	err = cmd.RegisterFlagCompletionFunc("provider", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return Providers, cobra.ShellCompDirectiveDefault
	})
	if err != nil {
		return err
	}

	if !providerOnly {
		cmd.Flags().StringVarP(&region, "region", "r", "", "the region to deploy to")
	}
	return nil
}
