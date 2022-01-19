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

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

func TestFromOptions(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		provider string
		region   string
		config   map[string]Target
		want     *Target
	}{
		{
			name: "default",
			want: &Target{Provider: Aws, Region: ""},
		},
		{
			name:   "from config",
			target: "aws",
			config: map[string]Target{"aws": {Provider: Aws, Region: "westus"}},
			want:   &Target{Provider: Aws, Region: "westus"},
		},
		{
			name:     "from args",
			provider: "azure",
			region:   "eastus",
			want:     &Target{Provider: Azure, Region: "eastus"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target = tt.target
			provider = tt.provider
			region = tt.region
			if tt.target != "" {
				current := viper.GetStringMap("targets")
				err := mapstructure.Decode(tt.config, &current)
				if err != nil {
					t.Error(err)
				}

				viper.Set("targets", current)
			}
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
