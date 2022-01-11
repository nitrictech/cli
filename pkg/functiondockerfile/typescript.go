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
	"io"
	"strings"

	"github.com/nitrictech/boxygen/pkg/backend/dockerfile"
	"github.com/nitrictech/newcli/pkg/stack"
)

func typescriptGenerator(f *stack.Function, version, provider string, w io.Writer) error {
	con, err := dockerfile.NewContainer(dockerfile.NewContainerOpts{
		From:   "node:alpine",
		Ignore: []string{"node_modules/", ".nitric/", ".git/", ".idea/"},
	})
	if err != nil {
		return err
	}

	con.Run(dockerfile.RunOptions{Command: []string{"yarn", "global", "add", "typescript"}})
	con.Run(dockerfile.RunOptions{Command: []string{"yarn", "global", "add", "ts-node"}})
	con.Copy(dockerfile.CopyOptions{Src: "package.json *.lock *-lock.json", Dest: "/"})
	con.Run(dockerfile.RunOptions{Command: []string{"yarn", "import", "||", "echo", "Lockfile already exists"}})
	con.Run(dockerfile.RunOptions{Command: []string{
		"set", "-ex;",
		"yarn", "install", "--production", "--frozen-lockfile", "--cache-folder", "/tmp/.cache;",
		"rm", "-rf", "/tmp/.cache;"}})

	withMembrane(con, version, provider)

	con.Copy(dockerfile.CopyOptions{Src: ".", Dest: "."})
	con.Config(dockerfile.ConfigOptions{
		Cmd: []string{"ts-node", "-T", f.Handler},
	})
	_, err = w.Write([]byte(strings.Join(con.Lines(), "\n")))
	return err
}

// typescriptDevBaseGenerator generates a base image for code-as-config
func typescriptDevBaseGenerator(w io.Writer) error {
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
