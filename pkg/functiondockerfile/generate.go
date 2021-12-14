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
	"errors"
	"fmt"
	"io"
	"path"

	"github.com/nitrictech/boxygen/pkg/backend/dockerfile"
	"github.com/nitrictech/newcli/pkg/stack"
)

type FunctionDockerfile interface {
	Generate(io.Writer) error
}

func withMembrane(con dockerfile.ContainerState, version, provider string) {
	fetchFrom := fmt.Sprintf("https://github.com/nitrictech/nitric/releases/download/%s/membrane-%s", version, provider)
	if version == "latest" {
		fetchFrom = fmt.Sprintf("https://github.com/nitrictech/nitric/releases/%s/download/membrane-%s", version, provider)
	}
	con.Add(dockerfile.AddOptions{Src: fetchFrom, Dest: "/usr/local/bin/membrane"})
	con.Run(dockerfile.RunOptions{Command: []string{"chmod", "+x-rw", "/usr/local/bin/membrane"}})
	con.Config(dockerfile.ConfigOptions{
		Entrypoint: []string{"/usr/local/bin/membrane"},
	})
}

func Generate(f *stack.Function, version, provider string, fwriter io.Writer) error {
	switch path.Ext(f.Handler) {
	case ".js":
		return javascriptGenerator(f, version, provider, fwriter)
	case ".ts":
		return typescriptGenerator(f, version, provider, fwriter)
	case ".go":
		return golangGenerator(f, version, provider, fwriter)
	case ".py":
		return pythonGenerator(f, version, provider, fwriter)
	case ".jar":
		return javaGenerator(f, version, provider, fwriter)
	}
	return errors.New("could not build dockerfile from " + f.Handler + ", extension not supported")
}
