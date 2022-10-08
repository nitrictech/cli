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

package common

import (
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/nitrictech/cli/pkg/project"
)

func TestDockerfile(t *testing.T) {
	tests := []struct {
		name     string
		c        project.Compute
		want     string
		wantBody string
		wantErr  error
	}{
		{
			name: "function",
			c: &project.Function{
				Handler:     "functions/list.ts",
				ComputeUnit: project.ComputeUnit{Name: "list"},
			},
			want: ".nitric/list.Dockerfile",
			wantBody: `FROM test
			CMD ["test"]
			`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, err := dockerfile(".", `FROM test
			CMD ["test"]
			`, tt.c)
			if (err != nil) != (tt.wantErr != nil) {
				t.Errorf("dockerfile() error = %v", err)
			}

			if !strings.Contains(fn, tt.want) {
				t.Errorf("%s != %s", tt.want, fn)
			}

			if tt.wantBody != "" {
				contents, err := os.ReadFile(fn)
				if err != nil {
					t.Error(err)
				}

				if !cmp.Equal(tt.wantBody, string(contents)) {
					t.Error(cmp.Diff(tt.wantBody, string(contents)))
				}
			}

			_ = os.Remove(".dockerignore")
			_ = os.RemoveAll(".nitric")
		})
	}
}
