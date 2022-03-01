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
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/imdario/mergo"
)

func TestFromGlobArgs(t *testing.T) {
	tests := []struct {
		name      string
		glob      []string
		stackPath string
		want      *Stack
		wantErr   bool
	}{
		{
			name:      "glob - current dir",
			glob:      []string{"*.go"},
			stackPath: ".",
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
			name:      "files",
			glob:      []string{"stack/types.go", "stack/options.go"},
			stackPath: "../../pkg",
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
			stackPath = tt.stackPath
			want := New("", "")
			err := mergo.Merge(want, tt.want)
			if err != nil {
				t.Fatal(err)
			}
			got, err := FromOptions(tt.glob)
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

func TestFromOptionsMinimal(t *testing.T) {
	tests := []struct {
		name      string
		stackPath string
		wantDir   string
		wantName  string
	}{
		{
			name:      "current dir",
			stackPath: ".",
			wantDir:   ".",
			wantName:  "stack",
		},
		{
			name:      "relative",
			stackPath: "../../pkg/cron",
			wantDir:   "../../pkg/cron",
			wantName:  "cron",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stackPath = tt.stackPath
			got, err := FromOptionsMinimal()
			if err != nil {
				t.Errorf("FromOptionsMinimal() error = %v", err)
				return
			}
			if got.Dir != tt.wantDir {
				t.Errorf("FromOptionsMinimal() got.Dir = %s, wantDir %v", got.Dir, tt.wantDir)
			}
			if got.Name != tt.wantName {
				t.Errorf("FromOptionsMinimal() got.Name = %s, wantName %v", got.Name, tt.wantName)
			}
		})
	}
}

func TestEnsureRuntimeDefaults(t *testing.T) {
	want := true

	if got := EnsureRuntimeDefaults(); got != want {
		t.Errorf("EnsureRuntimeDefaults() = %v, want %v", got, want)
	}
	expectGlobs := []string{"functions/*/*.go", "functions/*.ts", "functions/*.js"}
	sort.SliceStable(expectGlobs, func(i, j int) bool {
		return expectGlobs[i] < expectGlobs[j]
	})
	globs := defaultGlobsFromConfig()
	sort.SliceStable(globs, func(i, j int) bool {
		return globs[i] < globs[j]
	})

	if !cmp.Equal(expectGlobs, globs) {
		t.Error(cmp.Diff(expectGlobs, globs))
	}
}
