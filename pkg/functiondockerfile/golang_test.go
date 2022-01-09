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

func Test_golangGenerator(t *testing.T) {
	w := &bytes.Buffer{}
	f := &stack.Function{
		Handler: "pkg/handler/list.go",
	}
	if err := golangGenerator(f, "v1.2.3", "aws", w); err != nil {
		t.Errorf("golangGenerator() error = %v", err)
		return
	}
	wantW := `FROM golang:alpine as build
RUN apk update
RUN apk upgrade
RUN apk add --no-cache git gcc g++ make
WORKDIR /app/
COPY go.mod *.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/main pkg/handler/list.go
FROM alpine
COPY --from=build /bin/main /bin/main
RUN chmod +x-rw /bin/main
WORKDIR /
EXPOSE 9001
CMD ["/bin/main"]`

	if wantW != w.String() {
		t.Errorf("golangGenerator() = %v, want %v", w.String(), wantW)
	}
}
