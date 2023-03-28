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

package project

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/imdario/mergo"

	v1 "github.com/nitrictech/nitric/core/pkg/api/nitric/v1"
)

func TestFromConfig(t *testing.T) {
	tests := []struct {
		name    string
		proj    *Config
		want    *Project
		wantErr bool
	}{
		{
			name: "glob - current dir",
			proj: &Config{
				ConcreteHandlers: []*HandlerConfig{{
					BaseComputeConfig: BaseComputeConfig{
						Type: "default",
					},
					Match: "*.go",
				}},
				BaseConfig: &BaseConfig{
					Name:     "project",
					Dir:      ".",
					Handlers: []any{"*.go"},
				},
			},
			want: &Project{
				Dir:  ".",
				Name: "project",
				Functions: map[string]Function{
					"project": {
						Handler: "types.go",
						ComputeUnit: ComputeUnit{
							Name: "project",
							Type: "default",
						},
					},
				},
				Policies: []*v1.PolicyResource{},
			},
		},
		{
			name: "files",
			proj: &Config{
				ConcreteHandlers: []*HandlerConfig{{
					BaseComputeConfig: BaseComputeConfig{
						Type: "default",
					},
					Match: "stack/types.go",
				}, {
					BaseComputeConfig: BaseComputeConfig{
						Type: "default",
					},
					Match: "stack/options.go",
				}},
				BaseConfig: &BaseConfig{
					Name:     "pkg",
					Dir:      "../../pkg",
					Handlers: []any{"stack/types.go", "stack/options.go"},
				},
			},
			want: &Project{
				Dir:  "../../pkg",
				Name: "pkg",
				Functions: map[string]Function{
					"stack": {
						Handler: "stack/options.go",
						ComputeUnit: ComputeUnit{
							Name: "stack",
							Type: "default",
						},
					},
				},
				Policies: []*v1.PolicyResource{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := New(&BaseConfig{})

			err := mergo.Merge(want, tt.want, mergo.WithOverrideEmptySlice, mergo.WithOverride)
			if err != nil {
				t.Fatal(err)
			}

			got, err := FromConfig(tt.proj)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromOptions() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if !reflect.DeepEqual(want, got) {
				t.Error(cmp.Diff(want, got))
			}
		})
	}
}
