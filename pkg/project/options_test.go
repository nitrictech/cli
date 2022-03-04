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
				Name:     "stack",
				Dir:      ".",
				Handlers: []string{"*.go"},
			},
			want: &Project{
				Dir:  ".",
				Name: "project",
				Functions: map[string]Function{
					"project": {
						Handler:     "types.go",
						ComputeUnit: ComputeUnit{Name: "project"},
					},
				},
			},
		},
		{
			name: "files",
			proj: &Config{
				Name:     "pkg",
				Dir:      "../../pkg",
				Handlers: []string{"stack/types.go", "stack/options.go"},
			},
			want: &Project{
				Dir:  "../../pkg",
				Name: "pkg",
				Functions: map[string]Function{
					"project": {
						Handler:     "project/options.go",
						ComputeUnit: ComputeUnit{Name: "project"},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := New(&Config{})
			err := mergo.Merge(want, tt.want)
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
