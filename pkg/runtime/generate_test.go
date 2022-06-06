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
	"path/filepath"
	"testing"

	"github.com/docker/docker/api/types/mount"
	"github.com/google/go-cmp/cmp"

	"github.com/nitrictech/cli/pkg/utils"
)

func TestGenerate(t *testing.T) {
	tests := []struct {
		name        string
		handler     string
		version     string
		provider    string
		wantFwriter string
	}{
		{
			name:     "ts",
			handler:  "functions/list.ts",
			version:  "latest",
			provider: "azure",
			wantFwriter: `FROM node:alpine as layer-build
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
ADD https://github.com/nitrictech/nitric/releases/latest/download/membrane-azure /usr/local/bin/membrane
RUN chmod +x-rw /usr/local/bin/membrane
ENTRYPOINT ["/usr/local/bin/membrane"]
CMD ["node", "index.js"]`,
		},
		{
			name:     "go",
			handler:  "pkg/handler/list.go",
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
RUN go build -o /bin/main ./pkg/handler/...
FROM alpine
ADD https://github.com/nitrictech/nitric/releases/download/v1.2.3/membrane-aws /usr/local/bin/membrane
RUN chmod +x-rw /usr/local/bin/membrane
ENTRYPOINT ["/usr/local/bin/membrane"]
COPY --from=build /bin/main /bin/main
RUN chmod +x-rw /bin/main
RUN apk add --no-cache tzdata
WORKDIR /
EXPOSE 9001
CMD ["/bin/main"]`,
		},
		{
			name:     "python",
			handler:  "list.py",
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
			handler:  "functions/list.js",
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
			handler:  "testdata/test.java",
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
COPY --from=build testdata/test.java function.jar
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
			rt, err := NewRunTimeFromHandler(tt.handler)
			if err != nil {
				t.Error(err)
			}
			if err := rt.FunctionDockerfile("testdata", tt.version, tt.provider, fwriter); err != nil {
				t.Errorf("Generate() error = %v", err)
				return
			}
			if !cmp.Equal(fwriter.String(), tt.wantFwriter) {
				t.Error(cmp.Diff(tt.wantFwriter, fwriter.String()))
			}
		})
	}
}

func TestGeneralFuncs(t *testing.T) {
	tests := []struct {
		handler       string
		containerName string
		devImageName  string
	}{
		{
			handler:       "functions/list.ts",
			containerName: "list",
			devImageName:  "nitric-ts-dev",
		},
		{
			handler:       "pkg/list/main.go",
			containerName: "list",
			devImageName:  "nitric-go-dev",
		},
		{
			handler:       "list.py",
			containerName: "list",
			devImageName:  "nitric-py-dev",
		},
		{
			handler:       "functions/list.js",
			containerName: "list",
			devImageName:  "nitric-js-dev",
		},
		{
			handler:       "testdata/test.java",
			containerName: "testdata",
			devImageName:  "nitric-java-dev",
		},
	}
	for _, tt := range tests {
		t.Run(tt.handler, func(t *testing.T) {
			rt, err := NewRunTimeFromHandler(tt.handler)
			if err != nil {
				t.Error(err)
			}

			if rt.ContainerName() != tt.containerName {
				t.Errorf("ContainerName() %s != %s", rt.ContainerName(), tt.containerName)
			}

			if rt.DevImageName() != tt.devImageName {
				t.Errorf("DevImageName() %s != %s", rt.DevImageName(), tt.devImageName)
			}
		})
	}
}

func TestGenerateForCodeAsConfig(t *testing.T) {
	tests := []struct {
		handler     string
		version     string
		provider    string
		wantFwriter string
	}{
		{
			handler: "functions/list.ts",
			wantFwriter: `FROM node:alpine
RUN yarn global add typescript ts-node nodemon
WORKDIR /app/
ENTRYPOINT ["ts-node"]`,
		},
		{
			handler: "functions/list.js",
			wantFwriter: `FROM node:alpine
RUN yarn global add nodemon
WORKDIR /app/
ENTRYPOINT ["node"]`,
		},
		{
			handler: "pkg/list/main.go",
			wantFwriter: `FROM golang:alpine
RUN apk add --no-cache git
RUN go install github.com/asalkeld/CompileDaemon@d4b10de`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.handler, func(t *testing.T) {
			fwriter := &bytes.Buffer{}
			rt, err := NewRunTimeFromHandler(tt.handler)
			if err != nil {
				t.Error(err)
			}
			if err := rt.FunctionDockerfileForCodeAsConfig(fwriter); err != nil {
				t.Errorf("GenerateForCodeAsConfig() error = %v", err)
				return
			}
			if !cmp.Equal(fwriter.String(), tt.wantFwriter) {
				t.Error(cmp.Diff(tt.wantFwriter, fwriter.String()))
			}
		})
	}
}

func TestLaunchOptsForFunction(t *testing.T) {
	goPath, err := utils.GoPath()
	if err != nil {
		t.Error(err)
	}

	tests := []struct {
		handler string
		runCtx  string
		opts    LaunchOpts
	}{
		{
			handler: "functions/list.ts",
			runCtx:  ".",
			opts: LaunchOpts{
				TargetWD:   "/app",
				Entrypoint: []string{"nodemon"},
				Cmd:        []string{"--watch", "/app/**", "--ext", "ts,js,json", "--exec", "ts-node -T /app/functions/list.ts"},
				Mounts:     []mount.Mount{{Type: "bind", Source: ".", Target: "/app"}},
			},
		},
		{
			handler: "functions/list.js",
			runCtx:  ".",
			opts: LaunchOpts{
				TargetWD:   "/app",
				Entrypoint: []string{"nodemon"},
				Cmd:        []string{"--watch", "/app/**", "--ext", "ts,js,json", "--exec", "node /app/functions/list.js"},
				Mounts:     []mount.Mount{{Type: "bind", Source: ".", Target: "/app"}},
			},
		},
		{
			handler: "main.go",
			runCtx:  "../../",
			opts: LaunchOpts{
				TargetWD: "/go/src/github.com/nitrictech/cli",
				Cmd: []string{
					"/go/bin/CompileDaemon",
					"-verbose",
					"-exclude-dir=.git",
					"-exclude-dir=.nitric",
					"-directory=.", "-polling=false", "-build=go build -buildvcs=false -o runtime ././...", "-command=./runtime"},
				Mounts: []mount.Mount{
					{
						Type: "bind", Source: filepath.Join(goPath, "pkg"), Target: "/go/pkg",
					},
					{
						Type: "bind", Source: "../../", Target: "/go/src/github.com/nitrictech/cli",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.handler, func(t *testing.T) {
			rt, err := NewRunTimeFromHandler(tt.handler)
			if err != nil {
				t.Error(err)
			}
			lo, err := rt.LaunchOptsForFunction(tt.runCtx)
			if err != nil {
				t.Errorf("GenerateForCodeAsConfig() error = %v", err)
				return
			}
			if !cmp.Equal(tt.opts, lo) {
				t.Error(cmp.Diff(tt.opts, lo))
			}
		})
	}
}

func TestLaunchOptsForFunctionCollect(t *testing.T) {
	goPath, err := utils.GoPath()
	if err != nil {
		t.Error(err)
	}

	tests := []struct {
		handler string
		runCtx  string
		opts    LaunchOpts
	}{
		{
			handler: "functions/list.ts",
			runCtx:  ".",
			opts: LaunchOpts{
				Image:      "nitric-ts-dev",
				TargetWD:   "/app",
				Entrypoint: []string{"ts-node"},
				Cmd:        []string{"-T", "/app/functions/list.ts"},
				Mounts:     []mount.Mount{{Type: "bind", Source: ".", Target: "/app"}},
			},
		},
		{
			handler: "functions/list.js",
			runCtx:  ".",
			opts: LaunchOpts{
				Image:      "nitric-js-dev",
				TargetWD:   "/app",
				Entrypoint: []string{"node"},
				Cmd:        []string{"/app/functions/list.js"},
				Mounts:     []mount.Mount{{Type: "bind", Source: ".", Target: "/app"}},
			},
		},
		{
			handler: "main.go",
			runCtx:  "../../",
			opts: LaunchOpts{
				Image:    "nitric-go-dev",
				TargetWD: "/go/src/github.com/nitrictech/cli",
				Cmd: []string{
					"go", "run", "././..."},
				Mounts: []mount.Mount{
					{
						Type: "bind", Source: filepath.Join(goPath, "pkg"), Target: "/go/pkg",
					},
					{
						Type: "bind", Source: "../../", Target: "/go/src/github.com/nitrictech/cli",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.handler, func(t *testing.T) {
			rt, err := NewRunTimeFromHandler(tt.handler)
			if err != nil {
				t.Error(err)
			}
			lo, err := rt.LaunchOptsForFunctionCollect(tt.runCtx)
			if err != nil {
				t.Errorf("LaunchOptsForFunctionCollect() error = %v", err)
				return
			}
			if !cmp.Equal(tt.opts, lo) {
				t.Error(cmp.Diff(tt.opts, lo))
			}
		})
	}
}
