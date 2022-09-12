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
			wantBody: `FROM node:alpine as layer-build
RUN yarn global add typescript @vercel/ncc
COPY package.json *.lock *-lock.json /
RUN yarn import || echo Lockfile already exists
RUN set -ex; yarn install --production --frozen-lockfile --cache-folder /tmp/.cache; rm -rf /tmp/.cache;
COPY . .
RUN test -f tsconfig.json || echo '{"compilerOptions":{"esModuleInterop":true,"target":"es2015","moduleResolution":"node"}}' > tsconfig.json
RUN ncc build functions/list.ts -m --v8-cache -o lib/
FROM node:alpine as layer-final
COPY --from=layer-build package.json package.json
COPY --from=layer-build node_modules/ node_modules/
COPY --from=layer-build lib/ /
ADD https://github.com/nitrictech/nitric/releases/download/v0.18.0/membrane-aws /usr/local/bin/membrane
RUN chmod +x-rw /usr/local/bin/membrane
ENTRYPOINT ["/usr/local/bin/membrane"]
CMD ["node", "index.js"]`,
		},
		{
			name: "container",
			c: &project.Container{
				Dockerfile:  "Dockerfile.custom",
				ComputeUnit: project.ComputeUnit{Name: "custom"},
			},
			want: "Dockerfile.custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, err := dockerfile(".", "aws", tt.c)
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
