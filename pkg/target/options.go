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
	"errors"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nitrictech/cli/pkg/pflagext"
	"github.com/nitrictech/cli/pkg/utils"
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
	target      string
	provider    string
	region      string
	extraConfig []string
	Providers   = []string{Aws, Azure, Gcp, Digitalocean}
)

func EnsureDefaultConfig() bool {
	written := false

	targets, err := utils.ToStringMapStringMapStringE(viper.Get("targets"))
	if err != nil {
		targets = map[string]map[string]interface{}{}
	}

	if _, ok := targets[Aws]; !ok {
		targets[Aws] = map[string]interface{}{
			"provider": Aws,
			"region":   "us-east-1",
		}
		viper.Set("targets", targets)
		written = true
	}

	if _, ok := targets[Azure]; !ok {
		targets[Azure] = map[string]interface{}{
			"adminemail": "admin@example.com",
			"org":        "example.com",
			"region":     "eastus2",
			"provider":   Azure,
		}
		viper.Set("targets", targets)
		written = true
	}

	return written
}

func AllFromConfig() (map[string]Target, error) {
	tsMap, err := utils.ToStringMapStringMapStringE(viper.Get("targets"))
	if err != nil {
		return nil, err
	}

	targets := map[string]Target{}
	for name, tMap := range tsMap {
		t := Target{}
		err := mapstructure.Decode(tMap, &t)
		if err != nil {
			return nil, err
		}

		if len(tMap) > 2 {
			// Decode the "extra" map for provider specific values
			delete(tMap, "provider")
			delete(tMap, "region")
			err := mapstructure.Decode(tMap, &t.Extra)
			if err != nil {
				return nil, err
			}
		}
		targets[name] = t
	}

	return targets, nil
}

func FromOptions() (*Target, error) {
	if target == "" {
		target = DefaultTarget
	}

	targets, err := AllFromConfig()
	if err != nil {
		return nil, err
	}
	t, ok := targets[target]
	if !ok {
		return nil, errors.New("target " + target + " not in config")
	}

	if provider != "" {
		t.Provider = provider
	}

	if region != "" {
		t.Region = region
	}

	if len(extraConfig) > 0 && t.Extra == nil {
		t.Extra = map[string]interface{}{}
	}
	for _, c := range extraConfig {
		sc := strings.Split(c, "=")
		if len(sc) == 2 {
			t.Extra[sc[0]] = sc[1]
		}
	}

	return &t, nil
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

	cmd.Flags().VarP(pflagext.NewStringEnumVar(&provider, Providers, ""), "provider", "p", "the provider to deploy to")
	err = cmd.RegisterFlagCompletionFunc("provider", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return Providers, cobra.ShellCompDirectiveDefault
	})
	if err != nil {
		return err
	}

	if !providerOnly {
		cmd.Flags().StringVarP(&region, "region", "r", "", "the region to deploy to")
		cmd.Flags().StringSliceVarP(&extraConfig, "extra", "e", nil, "provider specific extra config")
	}
	return nil
}
