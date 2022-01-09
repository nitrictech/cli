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

package functiondockerfile

import (
	"bytes"
	"testing"

	"github.com/nitrictech/newcli/pkg/stack"
)

func Test_typescriptGenerator(t *testing.T) {
	w := &bytes.Buffer{}
	f := &stack.Function{
		Handler: "functions/list.ts",
	}
	if err := typescriptGenerator(f, "v1.2.3", "aws", w); err != nil {
		t.Errorf("typescriptGenerator() error = %v", err)
		return
	}
	wantW := `FROM node:alpine
RUN yarn global add typescript
RUN yarn global add ts-node
COPY package.json *.lock *-lock.json /
RUN yarn import || echo Lockfile already exists
RUN set -ex; yarn install --production --frozen-lockfile --cache-folder /tmp/.cache; rm -rf /tmp/.cache;
ADD https://github.com/nitrictech/nitric/releases/download/v1.2.3/membrane-aws /usr/local/bin/membrane
RUN chmod +x-rw /usr/local/bin/membrane
ENTRYPOINT ["/usr/local/bin/membrane"]
COPY . .
CMD ["ts-node", "-T", "functions/list.ts"]`

	if wantW != w.String() {
		t.Errorf("typescriptGenerator() = %v, want %v", w.String(), wantW)
	}
}
