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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/strslice"

	"github.com/nitrictech/boxygen/pkg/backend/dockerfile"
	"github.com/nitrictech/cli/pkg/utils"
)

type golang struct {
	rte     RuntimeExt
	handler string
}

var _ Runtime = &golang{}

func (t *golang) DevImageName() string {
	return fmt.Sprintf("nitric-%s-dev", t.rte)
}

func (t *golang) BuildIgnore() []string {
	return []string{}
}

func (t *golang) ContainerName() string {
	// get the abs dir in case user provides "."
	absH, err := filepath.Abs(t.handler)
	if err != nil {
		return ""
	}

	return filepath.Base(filepath.Dir(absH))
}

func (t *golang) FunctionDockerfile(funcCtxDir, version, provider string, w io.Writer) error {
	buildCon, err := dockerfile.NewContainer(dockerfile.NewContainerOpts{
		From:   "golang:alpine",
		As:     "build",
		Ignore: []string{},
	})
	if err != nil {
		return err
	}

	buildCon.Run(dockerfile.RunOptions{Command: []string{"apk", "update"}})
	buildCon.Run(dockerfile.RunOptions{Command: []string{"apk", "upgrade"}})
	buildCon.Run(dockerfile.RunOptions{Command: []string{"apk", "add", "--no-cache", "git", "gcc", "g++", "make"}})
	buildCon.Config(dockerfile.ConfigOptions{
		WorkingDir: "/app/",
	})

	err = buildCon.Copy(dockerfile.CopyOptions{Src: "go.mod *.sum", Dest: "./"})
	if err != nil {
		return err
	}
	buildCon.Run(dockerfile.RunOptions{Command: []string{"go", "mod", "download"}})
	err = buildCon.Copy(dockerfile.CopyOptions{Src: ".", Dest: "."})
	if err != nil {
		return err
	}

	buildCon.Run(dockerfile.RunOptions{Command: []string{"CGO_ENABLED=0", "GOOS=linux", "go", "build", "-o", "/bin/main", t.handler}})

	con, err := dockerfile.NewContainer(dockerfile.NewContainerOpts{
		From:   "alpine",
		Ignore: []string{},
	})
	if err != nil {
		return err
	}

	withMembrane(con, version, provider)

	err = con.Copy(dockerfile.CopyOptions{Src: "/bin/main", Dest: "/bin/main", From: "build"})
	if err != nil {
		return err
	}
	con.Run(dockerfile.RunOptions{Command: []string{"chmod", "+x-rw", "/bin/main"}})
	con.Config(dockerfile.ConfigOptions{
		Ports:      []int32{9001},
		WorkingDir: "/",
		Cmd:        []string{"/bin/main"},
	})

	_, err = w.Write([]byte(strings.Join(append(buildCon.Lines(), con.Lines()...), "\n")))
	return err
}

func (t *golang) FunctionDockerfileForCodeAsConfig(w io.Writer) error {
	con, err := dockerfile.NewContainer(dockerfile.NewContainerOpts{
		From:   "golang:alpine",
		Ignore: []string{},
	})
	if err != nil {
		return err
	}
	con.Run(dockerfile.RunOptions{Command: []string{"go", "install", "github.com/asalkeld/CompileDaemon@d4b10de"}})

	_, err = w.Write([]byte(strings.Join(con.Lines(), "\n")))
	return err
}

func (t *golang) LaunchOptsForFunctionCollect(runCtx string) (LaunchOpts, error) {
	module, err := utils.GoModule(runCtx)
	if err != nil {
		return LaunchOpts{}, err
	}
	return LaunchOpts{
		Image:    t.DevImageName(),
		TargetWD: filepath.Join("/go/src", module),
		Cmd:      strslice.StrSlice{"go", "run", "./" + filepath.ToSlash(t.handler)},
		Mounts: []mount.Mount{
			{
				Type:   "bind",
				Source: filepath.Join(os.Getenv("GOPATH"), "pkg"),
				Target: "/go/pkg",
			},
			{
				Type:   "bind",
				Source: runCtx,
				Target: filepath.Join("/go/src", module),
			},
		},
	}, nil
}

func (t *golang) LaunchOptsForFunction(runCtx string) (LaunchOpts, error) {
	module, err := utils.GoModule(runCtx)
	if err != nil {
		return LaunchOpts{}, err
	}
	containerRunCtx := filepath.Join("/go/src", module)
	relHandler := t.handler
	if strings.HasPrefix(t.handler, runCtx) {
		relHandler, err = filepath.Rel(runCtx, t.handler)
		if err != nil {
			return LaunchOpts{}, err
		}
	}

	opts := LaunchOpts{
		TargetWD: containerRunCtx,
		Cmd: strslice.StrSlice{
			"/go/bin/CompileDaemon",
			"-verbose",
			"-exclude-dir=.git",
			"-exclude-dir=.nitric",
			"-directory=.",
			fmt.Sprintf("-build=go build -o %s ./%s", t.ContainerName(), relHandler),
			"-command=./" + t.ContainerName(),
		},
		Mounts: []mount.Mount{
			{
				Type:   "bind",
				Source: filepath.Join(os.Getenv("GOPATH"), "pkg"),
				Target: "/go/pkg",
			},
			{
				Type:   "bind",
				Source: runCtx,
				Target: containerRunCtx,
			},
		},
	}

	return opts, nil
}
