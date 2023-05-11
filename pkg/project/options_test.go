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
	"path/filepath"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/imdario/mergo"

	v1 "github.com/nitrictech/nitric/core/pkg/api/nitric/v1"
)

func TestFromConfig(t *testing.T) {
	pkgAbsPath, err := filepath.Abs("../../pkg")
	if err != nil {
		t.Fatal(err)
	}

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
				BaseConfig: BaseConfig{
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
				BaseConfig: BaseConfig{
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
		{
			name: "dockerfile container",
			proj: &Config{
				BaseConfig: BaseConfig{
					Name: "docker",
					Dir:  "../../pkg",
					Containers: []DockerConfig{
						{
							Dockerfile: "runtime/typescript.dockerfile",
							Args: map[string]string{
								"MY_SCOPE": "test123",
							},
						},
					},
				},
			},
			want: &Project{
				Dir:  "../../pkg",
				Name: "docker",
				Functions: map[string]Function{
					"typescript.dockerfile-3f87c7b9702a57a8fa2968ad77981d0ae6a2f78eee7f82f4247082792ac1ac9a": {
						Handler:    "",
						Dockerfile: filepath.Join(pkgAbsPath, "runtime/typescript.dockerfile"),
						Args: map[string]string{
							"MY_SCOPE": "test123",
						},
						Context: pkgAbsPath,
						ComputeUnit: ComputeUnit{
							Name: "typescript.dockerfile-3f87c7b9702a57a8fa2968ad77981d0ae6a2f78eee7f82f4247082792ac1ac9a",
						},
					},
				},
				Policies: []*v1.PolicyResource{},
			},
		},
		{
			name: "dockerfile container glob",
			proj: &Config{
				BaseConfig: BaseConfig{
					Name: "docker",
					Dir:  "../../pkg",
					Containers: []DockerConfig{
						{
							Dockerfile: "runtime/*.dockerfile",
							Args: map[string]string{
								"MY_SCOPE": "all",
							},
						},
						{
							Dockerfile: "runtime/golang.dockerfile",
							Args: map[string]string{
								"MY_SCOPE": "go",
							},
						},
					},
				},
			},
			want: &Project{
				Dir:  "../../pkg",
				Name: "docker",
				Functions: map[string]Function{
					"csharp.dockerfile-1f62f9efd086c5ef3709c9322f68c8f4e7999701c8d3623a0f185b4ec8d97c49": {
						Handler:    "",
						Dockerfile: filepath.Join(pkgAbsPath, "runtime/csharp.dockerfile"),
						Args: map[string]string{
							"MY_SCOPE": "all",
						},
						Context: pkgAbsPath,
						ComputeUnit: ComputeUnit{
							Name: "csharp.dockerfile-1f62f9efd086c5ef3709c9322f68c8f4e7999701c8d3623a0f185b4ec8d97c49",
						},
					},
					"golang.dockerfile-3361d6856b455289dd07b4186858389285bb1b4d52cdcefcda174b066173a2ae": {
						Handler:    "",
						Dockerfile: filepath.Join(pkgAbsPath, "runtime/golang.dockerfile"),
						Args: map[string]string{
							"MY_SCOPE": "go",
						},
						Context: pkgAbsPath,
						ComputeUnit: ComputeUnit{
							Name: "golang.dockerfile-3361d6856b455289dd07b4186858389285bb1b4d52cdcefcda174b066173a2ae",
						},
					},
					"javascript.dockerfile-484e6c8c5505ffd10acedb2b23f2b7c4cafbf026ffaa4c817d007e36d4a3ec3c": {
						Handler:    "",
						Dockerfile: filepath.Join(pkgAbsPath, "runtime/javascript.dockerfile"),
						Args: map[string]string{
							"MY_SCOPE": "all",
						},
						Context: pkgAbsPath,
						ComputeUnit: ComputeUnit{
							Name: "javascript.dockerfile-484e6c8c5505ffd10acedb2b23f2b7c4cafbf026ffaa4c817d007e36d4a3ec3c",
						},
					},
					"python.dockerfile-a351536d73ca631b449872b8e19f216f029c029d3a871136148b31a0691bba5b": {
						Handler:    "",
						Dockerfile: filepath.Join(pkgAbsPath, "runtime/python.dockerfile"),
						Args: map[string]string{
							"MY_SCOPE": "all",
						},
						Context: pkgAbsPath,
						ComputeUnit: ComputeUnit{
							Name: "python.dockerfile-a351536d73ca631b449872b8e19f216f029c029d3a871136148b31a0691bba5b",
						},
					},
					"typescript.dockerfile-3f87c7b9702a57a8fa2968ad77981d0ae6a2f78eee7f82f4247082792ac1ac9a": {
						Handler:    "",
						Dockerfile: filepath.Join(pkgAbsPath, "runtime/typescript.dockerfile"),
						Args: map[string]string{
							"MY_SCOPE": "all",
						},
						Context: pkgAbsPath,
						ComputeUnit: ComputeUnit{
							Name: "typescript.dockerfile-3f87c7b9702a57a8fa2968ad77981d0ae6a2f78eee7f82f4247082792ac1ac9a",
						},
					},
				},
				Policies: []*v1.PolicyResource{},
			},
		},
		{
			name: "docker image",
			proj: &Config{
				BaseConfig: BaseConfig{
					Name: "docker-image",
					Dir:  "../../pkg",
					Containers: []DockerConfig{
						{
							Image: "python:3.9-alpine",
							Args: map[string]string{
								"MY_SCOPE": "all",
							},
						},
					},
				},
			},
			want: &Project{
				Dir:  "../../pkg",
				Name: "docker-image",
				Functions: map[string]Function{
					"python:3.9-alpine": {
						Handler: "",
						Image:   "python:3.9-alpine",
						Args: map[string]string{
							"MY_SCOPE": "all",
						},
						Context: "",
						ComputeUnit: ComputeUnit{
							Name: "python",
						},
					},
				},
				Policies: []*v1.PolicyResource{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := New(BaseConfig{})

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
