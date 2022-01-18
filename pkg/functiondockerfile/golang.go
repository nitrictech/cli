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

func golangGenerator(f *stack.Function, version, provider string, w io.Writer) error {
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
	buildCon.Run(dockerfile.RunOptions{Command: []string{"CGO_ENABLED=0", "GOOS=linux", "go", "build", "-o", "/bin/main", f.Handler}})

	con, err := dockerfile.NewContainer(dockerfile.NewContainerOpts{
		From:   "alpine",
		Ignore: []string{},
	})
	if err != nil {
		return err
	}

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
