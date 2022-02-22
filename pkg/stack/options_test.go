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
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/imdario/mergo"
)

func newFakeStack(name, dir string) *Stack {
	s := &Stack{
		Name:   name,
		Dir:    dir,
		Loaded: true,
		Collections: map[string]Collection{
			"dollars": {},
		},
		Containers: map[string]Container{
			"thing": {
				Dockerfile: "containerfile",
				Args:       []string{"-x", "-y"},
				ComputeUnit: ComputeUnit{
					Name:    "thing",
					Context: "feat5",
					Memory:  4096,
					Triggers: Triggers{
						[]string{"spiders"},
					},
				},
			},
		},
		Buckets: map[string]Bucket{
			"big": {},
			"red": {},
		},
		Topics: map[string]Topic{
			"pollies": {},
		},
		Queues: map[string]Queue{
			"covid": {},
		},
		Schedules: map[string]Schedule{
			"firstly": {
				Expression: "@daily",
				Event: ScheduleEvent{
					PayloadType: "?",
					Payload: map[string]interface{}{
						"a": "value",
					},
				},
				Target: ScheduleTarget{Type: "y", Name: "x"},
			},
		},
		Apis: map[string]string{
			"main": "main.json",
		},
		ApiDocs: map[string]*openapi3.T{
			"main": {
				ExtensionProps: openapi3.ExtensionProps{Extensions: map[string]interface{}{}},
				OpenAPI:        "3.0.1",
				Components: openapi3.Components{
					ExtensionProps: openapi3.ExtensionProps{Extensions: map[string]interface{}{}},
				},
				Info: &openapi3.Info{
					Title:          "test dummy",
					Version:        "v1",
					ExtensionProps: openapi3.ExtensionProps{Extensions: map[string]interface{}{}},
				},
				Paths: openapi3.Paths{},
			},
		},
		Functions: map[string]Function{
			"listr": {
				Version:      "v1.2.3",
				BuildScripts: []string{"make generate"},
				Excludes:     []string{"data/"},
				MaxRequests:  3490,
				External:     false,
				Handler:      "list.go",
				ComputeUnit: ComputeUnit{
					Name:    "listr",
					Context: "feat5",
					Memory:  4096,
					Triggers: Triggers{
						[]string{"spiders"},
					},
				},
			},
		},
	}
	for k, v := range s.Functions {
		v.SetContextDirectory(dir)
		s.Functions[k] = v
	}
	for k, v := range s.Containers {
		v.SetContextDirectory(dir)
		s.Containers[k] = v
	}
	return s
}

func TestFromOptions(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "nitric-cli-test-*")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(tmpDir)

	s := newFakeStack("test", tmpDir)

	err = s.ToFile(filepath.Join(tmpDir, "nitric.yaml"))
	if err != nil {
		t.Error(err)
	}

	stackPath = tmpDir
	newS, err := FromOptions([]string{})
	if err != nil {
		t.Error(err)
	}
	if !cmp.Equal(s, newS, cmpopts.IgnoreFields(Stack{}, "Policies")) {
		t.Error(cmp.Diff(s, newS, cmpopts.IgnoreFields(Stack{}, "Policies")))
	}
}

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
						ComputeUnit: ComputeUnit{Name: "stack", ContextDirectory: "."},
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
					"utils": {
						Handler:     "utils/paths.go",
						ComputeUnit: ComputeUnit{Name: "utils", ContextDirectory: "../../pkg"},
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
						ComputeUnit: ComputeUnit{Name: "stack", ContextDirectory: "../../pkg"},
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
	expectGlobs := []string{"functions/*/*.go", "functions/*.ts"}
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
