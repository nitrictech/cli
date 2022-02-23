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
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types/mount"

	"github.com/nitrictech/boxygen/pkg/backend/dockerfile"
)

type Runtime interface {
	DevImageName() string
	ContainerName() string
	FunctionDockerfile(funcCtxDir, version, provider string, w io.Writer) error
	FunctionDockerfileForCodeAsConfig(w io.Writer) error // FunctionDockerfileForCodeAsConfig generates a base image for code-as-config
	LaunchOptsForFunction(stackDir string) (LaunchOpts, error)
	LaunchOptsForFunctionCollect(stackDir string) (LaunchOpts, error)
}

type RuntimeExt string

const (
	RuntimeTypescript RuntimeExt = "ts"
	RuntimeJavascript RuntimeExt = "js"
	RuntimePython     RuntimeExt = "py"
	RuntimeGolang     RuntimeExt = "go"
	RuntimeJava       RuntimeExt = "java"

	RuntimeUnknown RuntimeExt = ""
)

type LaunchOpts struct {
	Image      string
	TargetWD   string
	Entrypoint []string
	Cmd        []string
	Mounts     []mount.Mount
}

func NewRunTimeFromHandler(handler string) (Runtime, error) {
	rt := RuntimeExt(strings.Replace(filepath.Ext(handler), ".", "", -1))

	switch rt {
	case RuntimeGolang:
		return &golang{rte: rt, handler: handler}, nil
	case RuntimeJavascript:
		return &javascript{rte: rt, handler: handler}, nil
	case RuntimePython:
		return &python{rte: rt, handler: handler}, nil
	case RuntimeTypescript:
		return &typescript{rte: rt, handler: handler}, nil
	case RuntimeJava:
		return &java{rte: rt, handler: handler}, nil
	default:
		return nil, errors.New("runtime '" + string(rt) + "' not supported")
	}
}

func withMembrane(con dockerfile.ContainerState, version, provider string) {
	membraneName := "membrane-" + provider
	fetchFrom := fmt.Sprintf("https://github.com/nitrictech/nitric/releases/download/%s/%s", version, membraneName)
	if version == "latest" {
		fetchFrom = fmt.Sprintf("https://github.com/nitrictech/nitric/releases/%s/download/%s", version, membraneName)
	}
	con.Add(dockerfile.AddOptions{Src: fetchFrom, Dest: "/usr/local/bin/membrane"})
	con.Run(dockerfile.RunOptions{Command: []string{"chmod", "+x-rw", "/usr/local/bin/membrane"}})
	con.Config(dockerfile.ConfigOptions{
		Entrypoint: []string{"/usr/local/bin/membrane"},
	})
}
