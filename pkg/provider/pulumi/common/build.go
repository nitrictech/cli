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

package common

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/nitrictech/cli/pkg/project"
	"github.com/nitrictech/cli/pkg/runtime"
	"github.com/nitrictech/cli/pkg/utils"
)

func dynamicDockerfile(dir, name string) (*os.File, error) {
	// create a more stable file name for the hashing
	err := os.MkdirAll(filepath.Join(dir, ".nitric"), os.ModePerm)
	if err != nil {
		return nil, err
	}

	return os.Create(filepath.Join(dir, ".nitric", name+".Dockerfile"))
}

func dockerfile(projDir, provider string, c project.Compute) (string, error) {
	switch x := c.(type) {
	case *project.Container:
		return x.Dockerfile, nil

	case *project.Function:
		fh, err := dynamicDockerfile(projDir, x.Name)
		if err != nil {
			return "", err
		}

		rt, err := runtime.NewRunTimeFromHandler(x.Handler)
		if err != nil {
			return "", err
		}

		err = rt.FunctionDockerfile(projDir, project.DefaultMembraneVersion, provider, fh)
		if err != nil {
			return "", err
		}

		err = os.WriteFile(fh.Name()+".dockerignore", []byte(strings.Join(rt.BuildIgnore(), "\n")), 0o644)
		if err != nil {
			return "", err
		}

		fh.Close()

		return fh.Name(), nil
	}

	return "", utils.NewNotSupportedErr("only Function and Containers supported")
}
