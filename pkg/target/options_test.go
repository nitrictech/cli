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
	"reflect"
	"testing"

	"github.com/imdario/mergo"
	"github.com/spf13/viper"
)

func TestFromOptions(t *testing.T) {
	tests := []struct {
		name        string
		target      string
		provider    string
		region      string
		extraConfig []string
		config      map[string]map[string]interface{}
		want        *Target
	}{
		{
			name: "default",
			config: map[string]map[string]interface{}{
				"test": {
					"provider": "aws",
					"region":   "westus",
				},
			},
			want: &Target{Provider: Aws, Region: "us-east-1"},
		},
		{
			name:   "from config",
			target: "az",
			config: map[string]map[string]interface{}{
				"az": {
					"provider": "azure",
					"region":   "jioindiawest",
				},
				"aws": {
					"provider": "aws",
					"region":   "us-west-2",
				},
			},
			want: &Target{Provider: Azure, Region: "jioindiawest"},
		},
		{
			name:   "from config with extra",
			target: "azure",
			config: map[string]map[string]interface{}{
				"azure": {
					"provider":   "azure",
					"region":     "westus",
					"org":        "nitric.io",
					"adminemail": "a@b.io",
				},
			},
			want: &Target{
				Provider: Azure,
				Region:   "westus",
				Extra: map[string]interface{}{
					"org":        "nitric.io",
					"adminemail": "a@b.io",
				},
			},
		},
		{
			name:        "from args",
			provider:    "azure",
			region:      "eastus",
			extraConfig: []string{"org=nitric.io", "adminemail=a@b.io"},
			config: map[string]map[string]interface{}{
				"azure": {
					"provider": "azure",
					"region":   "westus",
				},
			},
			want: &Target{
				Provider: Azure,
				Region:   "eastus",
				Extra: map[string]interface{}{
					"org":        "nitric.io",
					"adminemail": "a@b.io",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target = tt.target
			provider = tt.provider
			region = tt.region
			extraConfig = tt.extraConfig
			EnsureDefaultConfig()
			err := mergo.Map(&tt.config, viper.Get("targets"))
			if err != nil {
				t.Errorf("mergo.Map() error = %v", err)
				return
			}
			viper.Set("targets", tt.config)
			got, err := FromOptions()
			if err != nil {
				t.Errorf("FromOptions() error = %v", err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FromOptions() = %v, want %v", got, tt.want)
			}
		})
	}
}
