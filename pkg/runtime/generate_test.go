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

package runtime

import (
	"bytes"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGenerate(t *testing.T) {
	tsFile, err := os.ReadFile("typescript.dockerfile")
	if err != nil {
		t.Error(err)
	}

	goFile, err := os.ReadFile("golang.dockerfile")
	if err != nil {
		t.Error(err)
	}

	pythonFile, err := os.ReadFile("python.dockerfile")
	if err != nil {
		t.Error(err)
	}

	jsFile, err := os.ReadFile("javascript.dockerfile")
	if err != nil {
		t.Error(err)
	}

	tests := []struct {
		name        string
		handler     string
		wantFwriter string
	}{
		{
			name:        "ts",
			handler:     "functions/list.ts",
			wantFwriter: string(tsFile),
		},
		{
			name:        "go",
			handler:     "pkg/handler/list.go",
			wantFwriter: string(goFile),
		},
		{
			name:        "python",
			handler:     "list.py",
			wantFwriter: string(pythonFile),
		},
		{
			name:        "js",
			handler:     "functions/list.js",
			wantFwriter: string(jsFile),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fwriter := &bytes.Buffer{}
			rt, err := NewRunTimeFromHandler(tt.handler)
			if err != nil {
				t.Error(err)
			}
			if err := rt.BaseDockerFile(fwriter); err != nil {
				t.Errorf("Generate() error = %v", err)
				return
			}
			if !cmp.Equal(fwriter.String(), tt.wantFwriter) {
				t.Error(cmp.Diff(tt.wantFwriter, fwriter.String()))
			}
		})
	}
}
