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

func pythonGenerator(f *stack.Function, version, provider string, w io.Writer) error {
	con, err := dockerfile.NewContainer(dockerfile.NewContainerOpts{
		From:   "python:3.7-slim",
		Ignore: []string{"__pycache__/", "*.py[cod]", "*$py.class"},
	})
	if err != nil {
		return err
	}

	con.Run(dockerfile.RunOptions{Command: []string{"pip", "install", "--upgrade", "pip"}})
	con.Config(dockerfile.ConfigOptions{
		WorkingDir: "/",
	})
	con.Copy(dockerfile.CopyOptions{Src: "requirements.txt", Dest: "requirements.txt"})
	con.Run(dockerfile.RunOptions{Command: []string{"pip", "install", "--no-cache-dir", "-r", "requirements.txt"}})
	con.Copy(dockerfile.CopyOptions{Src: ".", Dest: "."})

	withMembrane(con, version, provider)

	con.Config(dockerfile.ConfigOptions{
		Env: map[string]string{
			"PYTHONPATH": "/app/:${PYTHONPATH}",
		},
		Ports: []int32{9001},
		Cmd:   []string{"python", f.Handler},
	})
	_, err = w.Write([]byte(strings.Join(con.Lines(), "\n")))
	return err
}
