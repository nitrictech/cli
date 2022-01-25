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
					"container_helper": {
						Handler:     "container_helper.go",
						ComputeUnit: ComputeUnit{Name: "container_helper", ContextDirectory: "."},
					},
					"function_helpers": {
						Handler:     "function_helpers.go",
						ComputeUnit: ComputeUnit{Name: "function_helpers", ContextDirectory: "."},
					},
					"function_helpers_test": {
						Handler:     "function_helpers_test.go",
						ComputeUnit: ComputeUnit{Name: "function_helpers_test", ContextDirectory: "."},
					},
					"options": {
						Handler:     "options.go",
						ComputeUnit: ComputeUnit{Name: "options", ContextDirectory: "."},
					},
					"options_test": {
						Handler:     "options_test.go",
						ComputeUnit: ComputeUnit{Name: "options_test", ContextDirectory: "."},
					},
					"types": {
						Handler:     "types.go",
						ComputeUnit: ComputeUnit{Name: "types", ContextDirectory: "."},
					},
				},
			},
		},
		{
			name:      "glob",
			glob:      []string{"utils/*.go"},
			stackPath: "../../pkg",
			want: &Stack{
				Dir:  "../../pkg",
				Name: "pkg",
				Functions: map[string]Function{
					"errors": {
						Handler:     "utils/errors.go",
						ComputeUnit: ComputeUnit{Name: "errors", ContextDirectory: "../../pkg"},
					},
					"fileinfo": {
						Handler:     "utils/fileinfo.go",
						ComputeUnit: ComputeUnit{Name: "fileinfo", ContextDirectory: "../../pkg"},
					},
					"getter": {
						Handler:     "utils/getter.go",
						ComputeUnit: ComputeUnit{Name: "getter", ContextDirectory: "../../pkg"},
					},
					"glob": {
						Handler:     "utils/glob.go",
						ComputeUnit: ComputeUnit{Name: "glob", ContextDirectory: "../../pkg"},
					},
					"paths": {
						Handler:     "utils/paths.go",
						ComputeUnit: ComputeUnit{Name: "paths", ContextDirectory: "../../pkg"},
					},
					"runtime": {
						Handler:     "utils/runtime.go",
						ComputeUnit: ComputeUnit{Name: "runtime", ContextDirectory: "../../pkg"},
					},
					"tar": {
						Handler:     "utils/tar.go",
						ComputeUnit: ComputeUnit{Name: "tar", ContextDirectory: "../../pkg"},
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
					"types": {
						Handler:     "stack/types.go",
						ComputeUnit: ComputeUnit{Name: "types", ContextDirectory: "../../pkg"},
					},
					"options": {
						Handler:     "stack/options.go",
						ComputeUnit: ComputeUnit{Name: "options", ContextDirectory: "../../pkg"},
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
			got, err := FromGlobArgs(tt.glob)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromGlobArgs() error = %v, wantErr %v", err, tt.wantErr)
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
