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

package output

import (
	"bytes"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/nitrictech/cli/pkg/containerengine"
	"github.com/nitrictech/cli/pkg/target"
)

func Test_printStruct(t *testing.T) {
	tests := []struct {
		name   string
		object interface{}
		expect string
	}{
		{
			name:   "json tags",
			object: target.Target{Provider: "azure", Region: "somewhere"},
			expect: `+----------+-----------+
| PROVIDER | azure     |
| REGION   | somewhere |
+----------+-----------+
`,
		},
		{
			name:   "yaml tags",
			object: containerengine.Image{ID: "34234242", Repository: "my-app", Tag: "latest"},
			expect: `+------------+----------+
| ID         | 34234242 |
| REPOSITORY | my-app   |
| TAG        | latest   |
| CREATEDAT  |          |
+------------+----------+
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			printStruct(tt.object, buf)
			if !cmp.Equal(tt.expect, buf.String()) {
				t.Error(cmp.Diff(tt.expect, buf.String()))
			}
		})
	}
}

func Test_printList(t *testing.T) {
	tests := []struct {
		name   string
		object []target.Target
		expect string
	}{
		{
			name: "json tags",
			object: []target.Target{
				{Provider: "azure", Region: "somewhere"},
				{Provider: "local"},
			},
			expect: `+----------+-----------+
| PROVIDER | REGION    |
+----------+-----------+
| azure    | somewhere |
| local    |           |
+----------+-----------+
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			sort.SliceStable(tt.object, func(i, j int) bool {
				return strings.Compare(tt.object[i].Provider, tt.object[j].Provider) < 0
			})
			printList(tt.object, buf)
			if !cmp.Equal(tt.expect, buf.String()) {
				t.Error(cmp.Diff(tt.expect, buf.String()))
			}
		})
	}
}

func Test_printMap(t *testing.T) {
	tests := []struct {
		name    string
		object  interface{}
		wantOut string
	}{
		{
			name: "json tags",
			object: map[string]target.Target{
				"t1":    {Provider: "azure", Region: "somewhere"},
				"local": {Provider: "local"},
			},
			wantOut: `+-------+----------+-----------+
| KEY   | PROVIDER | REGION    |
+-------+----------+-----------+
| local | local    |           |
| t1    | azure    | somewhere |
+-------+----------+-----------+
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			printMap(tt.object, out)
			if !cmp.Equal(tt.wantOut, out.String()) {
				t.Error(cmp.Diff(tt.wantOut, out.String()))
			}
		})
	}
}
