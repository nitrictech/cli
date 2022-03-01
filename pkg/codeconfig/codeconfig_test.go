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

package codeconfig

import (
	"reflect"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func Test_splitPath(t *testing.T) {
	tests := []struct {
		name       string
		workerPath string
		want       string
		want1      openapi3.Parameters
	}{
		{
			name:       "simple",
			workerPath: "/orders",
			want:       "/orders",
			want1:      openapi3.Parameters{},
		},
		{
			name:       "trailing slash",
			workerPath: "/orders/",
			want:       "/orders",
			want1:      openapi3.Parameters{},
		},
		{
			name:       "with param",
			workerPath: "/orders/:id",
			want:       "/orders/{id}",
			want1: openapi3.Parameters{
				&openapi3.ParameterRef{
					Value: &openapi3.Parameter{
						In:       "path",
						Name:     "id",
						Required: true,
						Schema: &openapi3.SchemaRef{
							Value: &openapi3.Schema{
								Type: "string",
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := splitPath(tt.workerPath)
			if got != tt.want {
				t.Errorf("splitPath() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("splitPath() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
