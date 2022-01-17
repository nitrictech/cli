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

	"github.com/nitrictech/boxygen/pkg/backend/dockerfile"
	"github.com/nitrictech/newcli/pkg/stack"
	"github.com/nitrictech/newcli/pkg/utils"
)

type FunctionDockerfile interface {
	Generate(io.Writer) error
}

func withMembrane(con dockerfile.ContainerState, version, provider string) {
	membraneName := "membrane-" + provider
	if provider == "local" {
		membraneName = "membrane-dev"
	}
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

var generators = map[utils.Runtime]func(f *stack.Function, version, provider string, w io.Writer) error{
	utils.RuntimeGolang:     golangGenerator,
	utils.RuntimeJava:       javaGenerator,
	utils.RuntimeJavascript: javascriptGenerator,
	utils.RuntimeTypescript: typescriptGenerator,
	utils.RuntimePython:     pythonGenerator,
}

func Generate(f *stack.Function, version, provider string, fwriter io.Writer) error {
	rt, err := utils.NewRunTimeFromFilename(f.Handler)
	if err != nil {
		return err
	}
	generator, ok := generators[rt]
	if generator == nil || !ok {
		return errors.New("could not build dockerfile from " + f.Handler + ", extension not supported")
	}
	return generator(f, version, provider, fwriter)
}

// GenerateForCodeAsConfig dockerfiles for code-as-config
// These will initially be generated without the membrane
func GenerateForCodeAsConfig(handler string, fwriter io.Writer) error {
	rt, err := utils.NewRunTimeFromFilename(handler)
	if err != nil {
		return err
	}
	switch rt {
	case utils.RuntimeJavascript:
		fallthrough
	case utils.RuntimeTypescript:
		return typescriptDevBaseGenerator(fwriter)
	}

	return errors.New("could not build dockerfile from " + handler + ", extension not supported")
}
