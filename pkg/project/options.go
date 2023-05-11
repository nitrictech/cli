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

package project

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pterm/pterm"

	"github.com/nitrictech/cli/pkg/runtime"
	"github.com/nitrictech/cli/pkg/utils"
)

func FromConfig(c *Config) (*Project, error) {
	p := New(c.BaseConfig)

	for _, h := range c.ConcreteHandlers {
		fs, err := utils.GlobInDir(p.Dir, h.Match)
		if err != nil {
			return nil, err
		}

		for _, f := range fs {
			fn, err := FunctionFromHandler(f, h.Type)
			if err != nil {
				return nil, err
			}

			p.Functions[fn.Name] = fn
		}
	}

	for _, c := range c.Containers {
		if c.Dockerfile != "" {
			// Get the absolute path of docker context
			dockerContext, err := filepath.Abs(filepath.Join(p.Dir, c.Context))
			if err != nil {
				return nil, err
			}

			fs, err := utils.GlobInDir(p.Dir, c.Dockerfile)
			if err != nil {
				return nil, err
			}

			for _, f := range fs {
				absPath, err := filepath.Abs(filepath.Join(p.Dir, f))
				if err != nil {
					return nil, err
				}

				fn, err := FunctionFromContainer(DockerConfig{
					Dockerfile: absPath,
					Args:       c.Args,
					Context:    dockerContext,
					Nitric:     c.Nitric,
				})
				if err != nil {
					return nil, err
				}

				p.Functions[fn.Name] = fn
			}
		} else if c.Image != "" {
			imageRef, err := utils.ParseDockerImage(c.Image)
			if err != nil {
				return nil, err
			}

			// TODO add non nitric wrapper later
			p.Functions[c.Image] = Function{
				ComputeUnit: ComputeUnit{
					Name: imageRef.Name,
				},
				Image: c.Image,
				Args:  c.Args,
			}
		}
	}

	if len(p.Functions) == 0 {
		if len(c.ConcreteHandlers) > 0 {
			return nil, fmt.Errorf("no functions were found within match on handlers '%+v' in dir '%s', try a new pattern", c.ConcreteHandlers, p.Dir)
		}

		// in preparation for non nitric wrapper
		if len(p.Containers) > 0 {
			return p, nil
		}

		return nil, fmt.Errorf("no containers were found, try a new pattern")
	}

	return p, nil
}

func FunctionFromHandler(h string, t string) (Function, error) {
	_, err := utils.ParseDockerImage(filepath.Base(h))
	if err != nil {
		return Function{}, fmt.Errorf("handler filepath \"%s\" is invalid, must be valid ASCII containing lowercase and uppercase letters, digits, underscores, periods and hyphens", h)
	}

	pterm.Debug.Println("Using function from " + h)

	rt, err := runtime.NewRunTimeFromHandler(h)
	if err != nil {
		return Function{}, err
	}

	return Function{
		ComputeUnit: ComputeUnit{
			Name: rt.ContainerName(),
			Type: t,
		},
		Handler: h,
	}, nil
}

func FunctionFromContainer(c DockerConfig) (Function, error) {
	pterm.Debug.Println("Using container from dockerfile: " + c.Dockerfile)

	// // Read the contents of the file
	fileContent, err := os.ReadFile(c.Dockerfile)
	if err != nil {
		return Function{}, err
	}

	hash := sha256.Sum256(fileContent)
	hashValue := hex.EncodeToString(hash[:])
	imageName := fmt.Sprintf("%s-%s", strings.ToLower(filepath.Base(c.Dockerfile)), hashValue)

	return Function{
		ComputeUnit: ComputeUnit{
			Name: imageName,
		},
		Dockerfile: c.Dockerfile,
		Context:    c.Context,
		Args:       c.Args,
	}, nil
}
