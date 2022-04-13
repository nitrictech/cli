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
	"path/filepath"
	osruntime "runtime"
	"strings"

	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/go-connections/nat"

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
	return commonIgnore
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
	css := dockerfile.NewStateStore()

	buildCon, err := css.NewContainer(dockerfile.NewContainerOpts{
		From:   "golang:alpine",
		As:     layerBuild,
		Ignore: t.BuildIgnore(),
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

	buildCon.Run(dockerfile.RunOptions{
		Command: []string{
			"go", "build", "-o", "/bin/main", "./" + filepath.Dir(t.handler) + "/...",
		},
	})

	con, err := css.NewContainer(dockerfile.NewContainerOpts{
		From:   "alpine",
		As:     layerFinal,
		Ignore: t.BuildIgnore(),
	})
	if err != nil {
		return err
	}

	withMembrane(con, version, provider)

	err = con.Copy(dockerfile.CopyOptions{Src: "/bin/main", Dest: "/bin/main", From: buildCon.Name()})
	if err != nil {
		return err
	}

	con.Run(dockerfile.RunOptions{Command: []string{"chmod", "+x-rw", "/bin/main"}})
	con.Run(dockerfile.RunOptions{Command: []string{"apk", "add", "--no-cache", "tzdata"}})

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
		Ignore: t.BuildIgnore(),
	})
	if err != nil {
		return err
	}

	con.Run(dockerfile.RunOptions{Command: []string{"apk", "add", "--no-cache", "git", "gcc", "g++", "make", "musl-dev"}})
	con.Run(dockerfile.RunOptions{Command: []string{"go", "install", "github.com/asalkeld/CompileDaemon@master"}})
	// Allow delve to run on Alpine based containers.
	con.Run(dockerfile.RunOptions{Command: []string{"go", "install", "github.com/go-delve/delve/cmd/dlv@latest"}})

	_, err = w.Write([]byte(strings.Join(con.Lines(), "\n")))

	return err
}

func (t *golang) LaunchOptsForFunctionCollect(runCtx string) (LaunchOpts, error) {
	module, err := utils.GoModule(runCtx)
	if err != nil {
		return LaunchOpts{}, err
	}

	goPath, err := utils.GoPath()
	if err != nil {
		return LaunchOpts{}, err
	}

	return LaunchOpts{
		Image:    t.DevImageName(),
		TargetWD: filepath.ToSlash(filepath.Join("/go/src", module)),
		Cmd:      strslice.StrSlice{"go", "run", "./" + filepath.ToSlash(filepath.Dir(t.handler)) + "/..."},
		Mounts: []mount.Mount{
			{
				Type:   "bind",
				Source: filepath.Join(goPath, "pkg"),
				Target: "/go/pkg",
			},
			{
				Type:   "bind",
				Source: runCtx,
				Target: filepath.ToSlash(filepath.Join("/go/src", module)),
			},
		},
	}, nil
}

func (t *golang) LaunchOptsForFunction(runCtx string) (LaunchOpts, error) {
	module, err := utils.GoModule(runCtx)
	if err != nil {
		return LaunchOpts{}, err
	}

	containerRunCtx := filepath.ToSlash(filepath.Join("/go/src", module))
	relHandler := t.handler

	if strings.HasPrefix(t.handler, runCtx) {
		relHandler, err = filepath.Rel(runCtx, t.handler)
		if err != nil {
			return LaunchOpts{}, err
		}
	}

	goPath, err := utils.GoPath()
	if err != nil {
		return LaunchOpts{}, err
	}

	opts := LaunchOpts{
		TargetWD: containerRunCtx,
		Cmd: strslice.StrSlice{
			"/go/bin/CompileDaemon",
			"-verbose",
			"-exclude-dir=.git",
			"-exclude-dir=.nitric",
			"-directory=.",
			"-polling-interval=2500",
			fmt.Sprintf("-polling=%t", osruntime.GOOS == "windows"),
			fmt.Sprintf("-build=go build -gcflags=all=-N -gcflags=all=-l -o %s ./%s/...", t.ContainerName(), filepath.ToSlash(filepath.Dir(relHandler))),
			fmt.Sprintf("-command=dlv exec ./%v -l=0.0.0.0:2345 --continue --accept-multiclient --headless --api-version=2", t.ContainerName()),
		},
		PortBindings: nat.PortMap{
			"2345/tcp": []nat.PortBinding{
				{
					HostIP: "0.0.0.0",
					HostPort: "2345",
				},
			},
		},
		Mounts: []mount.Mount{
			{
				Type:   "bind",
				Source: filepath.Join(goPath, "pkg"),
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
