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

	"github.com/google/go-cmp/cmp"

	"github.com/nitrictech/newcli/pkg/stack"
)

func TestGenerate(t *testing.T) {
	tests := []struct {
		name        string
		f           *stack.Function
		version     string
		provider    string
		wantFwriter string
	}{
		{
			name:     "ts",
			f:        &stack.Function{Handler: "functions/list.ts"},
			version:  "latest",
			provider: "azure",
			wantFwriter: `FROM node:alpine
RUN yarn global add typescript
RUN yarn global add ts-node
COPY package.json *.lock *-lock.json /
RUN yarn import || echo Lockfile already exists
RUN set -ex; yarn install --production --frozen-lockfile --cache-folder /tmp/.cache; rm -rf /tmp/.cache;
ADD https://github.com/nitrictech/nitric/releases/latest/download/membrane-azure /usr/local/bin/membrane
RUN chmod +x-rw /usr/local/bin/membrane
ENTRYPOINT ["/usr/local/bin/membrane"]
COPY . .
CMD ["ts-node", "-T", "functions/list.ts"]`,
		},
		{
			name:     "go",
			f:        &stack.Function{Handler: "pkg/handler/list.go"},
			version:  "v1.2.3",
			provider: "aws",
			wantFwriter: `FROM golang:alpine as build
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
CMD ["/bin/main"]`,
		},
		{
			name:     "python",
			f:        &stack.Function{Handler: "list.py"},
			version:  "v1.1.7",
			provider: "digitalocean",
			wantFwriter: `FROM python:3.7-slim
RUN pip install --upgrade pip
WORKDIR /
COPY requirements.txt requirements.txt
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
ADD https://github.com/nitrictech/nitric/releases/download/v1.1.7/membrane-digitalocean /usr/local/bin/membrane
RUN chmod +x-rw /usr/local/bin/membrane
ENTRYPOINT ["/usr/local/bin/membrane"]
EXPOSE 9001
ENV PYTHONPATH=/app/:${PYTHONPATH}
CMD ["python", "list.py"]`,
		},
		{
			name:     "js",
			f:        &stack.Function{Handler: "functions/list.js"},
			version:  "latest",
			provider: "gcp",
			wantFwriter: `FROM node:alpine
ADD https://github.com/nitrictech/nitric/releases/latest/download/membrane-gcp /usr/local/bin/membrane
RUN chmod +x-rw /usr/local/bin/membrane
ENTRYPOINT ["/usr/local/bin/membrane"]
COPY package.json *.lock *-lock.json /
RUN yarn import || echo Lockfile already exists
RUN set -ex; yarn install --production --frozen-lockfile --cache-folder /tmp/.cache; rm -rf /tmp/.cache;
COPY . .
CMD ["node", "functions/list.js"]`,
		},
		{
			name:     "java",
			f:        &stack.Function{Handler: "test.java", ComputeUnit: stack.ComputeUnit{Context: "testdata"}},
			version:  "latest",
			provider: "aws",
			wantFwriter: `FROM maven:3-openjdk-11 as build
COPY /pom.xml pom.xml
COPY /subdir/pom.xml subdir/pom.xml
RUN mvn de.qaware.maven:go-offline-maven-plugin:resolve-dependencies
COPY / .
COPY /subdir subdir
RUN mvn clean package
FROM adoptopenjdk/openjdk11:x86_64-alpine-jre-11.0.10_9
COPY --from=build test.java function.jar
WORKDIR /
EXPOSE 9001
CMD ["java", "-jar", "function.jar"]
ADD https://github.com/nitrictech/nitric/releases/latest/download/membrane-aws /usr/local/bin/membrane
RUN chmod +x-rw /usr/local/bin/membrane
ENTRYPOINT ["/usr/local/bin/membrane"]`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fwriter := &bytes.Buffer{}
			if tt.f.ComputeUnit.Context != "" {
				tt.f.ComputeUnit.ContextDirectory = tt.f.ComputeUnit.Context
			}
			tt.f.ComputeUnit.Name = "testfn"
			if err := Generate(tt.f, tt.version, tt.provider, fwriter); err != nil {
				t.Errorf("Generate() error = %v", err)
				return
			}
			if !cmp.Equal(fwriter.String(), tt.wantFwriter) {
				t.Error(cmp.Diff(tt.wantFwriter, fwriter.String()))
			}
		})
	}
}

func TestGenerateForCodeAsConfig(t *testing.T) {
	tests := []struct {
		name        string
		f           *stack.Function
		version     string
		provider    string
		wantFwriter string
	}{
		{
			name: "ts",
			f:    &stack.Function{Handler: "functions/list.ts"},
			wantFwriter: `FROM node:alpine
RUN yarn global add typescript ts-node nodemon
WORKDIR /app/
ENTRYPOINT ["ts-node"]`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fwriter := &bytes.Buffer{}
			tt.f.ComputeUnit = stack.ComputeUnit{Name: "testfn", ContextDirectory: "./"}
			if err := GenerateForCodeAsConfig(tt.f.Handler, fwriter); err != nil {
				t.Errorf("GenerateForCodeAsConfig() error = %v", err)
				return
			}
			if !cmp.Equal(fwriter.String(), tt.wantFwriter) {
				t.Error(cmp.Diff(tt.wantFwriter, fwriter.String()))
			}
		})
	}
}
