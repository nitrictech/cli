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
	"strings"

	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/strslice"

	"github.com/nitrictech/boxygen/pkg/backend/dockerfile"
)

type typescript struct {
	rte     RuntimeExt
	handler string
}

var _ Runtime = &typescript{}

func (t *typescript) DevImageName() string {
	return fmt.Sprintf("nitric-%s-dev", t.rte)
}

func (t *typescript) ContainerName() string {
	return strings.Replace(filepath.Base(t.handler), filepath.Ext(t.handler), "", 1)
}

func (t *typescript) FunctionDockerfile(funcCtxDir, version, provider string, w io.Writer) error {
	con, err := dockerfile.NewContainer(dockerfile.NewContainerOpts{
		From:   "node:alpine",
		Ignore: []string{"node_modules/", ".nitric/", ".git/", ".idea/"},
	})
	if err != nil {
		return err
	}

	con.Run(dockerfile.RunOptions{Command: []string{"yarn", "global", "add", "typescript"}})
	con.Run(dockerfile.RunOptions{Command: []string{"yarn", "global", "add", "ts-node"}})
	err = con.Copy(dockerfile.CopyOptions{Src: "package.json *.lock *-lock.json", Dest: "/"})
	if err != nil {
		return err
	}
	con.Run(dockerfile.RunOptions{Command: []string{"yarn", "import", "||", "echo", "Lockfile already exists"}})
	con.Run(dockerfile.RunOptions{Command: []string{
		"set", "-ex;",
		"yarn", "install", "--production", "--frozen-lockfile", "--cache-folder", "/tmp/.cache;",
		"rm", "-rf", "/tmp/.cache;"}})

	withMembrane(con, version, provider)

	err = con.Copy(dockerfile.CopyOptions{Src: ".", Dest: "."})
	if err != nil {
		return err
	}
	con.Config(dockerfile.ConfigOptions{
		Cmd: []string{"ts-node", "-T", t.handler},
	})
	_, err = w.Write([]byte(strings.Join(con.Lines(), "\n")))
	return err
}

func (t *typescript) FunctionDockerfileForCodeAsConfig(w io.Writer) error {
	con, err := dockerfile.NewContainer(dockerfile.NewContainerOpts{
		From:   "node:alpine",
		Ignore: []string{"node_modules/", ".nitric/", ".git/", ".idea/"},
	})
	if err != nil {
		return err
	}

	con.Run(dockerfile.RunOptions{Command: []string{"yarn", "global", "add", "typescript", "ts-node", "nodemon"}})
	con.Config(dockerfile.ConfigOptions{
		Entrypoint: []string{"ts-node"},
		WorkingDir: "/app/",
	})

	_, err = w.Write([]byte(strings.Join(con.Lines(), "\n")))
	return err
}

func (t *typescript) LaunchOptsForFunctionCollect(runCtx string) (LaunchOpts, error) {
	return LaunchOpts{
		Image:      t.DevImageName(),
		Entrypoint: strslice.StrSlice{"ts-node"},
		Cmd:        strslice.StrSlice{"-T", "/app/" + filepath.ToSlash(t.handler)},
		TargetWD:   "/app",
		Mounts: []mount.Mount{
			{
				Type:   "bind",
				Source: runCtx,
				Target: "/app",
			},
		},
	}, nil
}

func (t *typescript) LaunchOptsForFunction(runCtx string) (LaunchOpts, error) {
	return LaunchOpts{
		TargetWD: "/app",
		Mounts: []mount.Mount{
			{
				Type:   "bind",
				Source: runCtx,
				Target: "/app",
			},
		},
		Entrypoint: strslice.StrSlice{"nodemon"},
		Cmd:        strslice.StrSlice{"--watch", "/app/**", "--ext", "ts,js,json", "--exec", "ts-node -T " + "/app/" + t.handler},
	}, nil
}
