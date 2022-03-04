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

package stack

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/imdario/mergo"

	"github.com/nitrictech/cli/pkg/project"
)

func TestFromConfig(t *testing.T) {
	tests := []struct {
		name    string
		proj    *project.Config
		want    *Stack
		wantErr bool
	}{
		{
			name: "glob - current dir",
			proj: &project.Config{
				Name:     "stack",
				Dir:      ".",
				Handlers: []string{"*.go"},
			},
			want: &Stack{
				Dir:  ".",
				Name: "stack",
				Functions: map[string]Function{
					"stack": {
						Handler:     "types.go",
						ComputeUnit: ComputeUnit{Name: "stack"},
					},
				},
			},
		},
		{
			name: "files",
			proj: &project.Config{
				Name:     "pkg",
				Dir:      "../../pkg",
				Handlers: []string{"stack/types.go", "stack/options.go"},
			},
			want: &Stack{
				Dir:  "../../pkg",
				Name: "pkg",
				Functions: map[string]Function{
					"stack": {
						Handler:     "stack/options.go",
						ComputeUnit: ComputeUnit{Name: "stack"},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := New("", "")
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
