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
	"fmt"

	"github.com/pterm/pterm"
	"github.com/samber/lo"

	"github.com/nitrictech/cli/pkg/utils"
)

var stackPath string

func FromConfig(c *Config) (*Project, error) {
	p := New(c.BaseConfig)

	for _, h := range c.ConcreteHandlers {
		fs, err := utils.GlobInDir(stackPath, h.Match)
		if err != nil {
			return nil, err
		}

		for _, f := range fs {
			fn, err := FunctionFromHandler(f, h)
			if err != nil {
				return nil, err
			}

			fn.Project = p

			rt, err := fn.GetRuntime()
			if err != nil {
				return nil, err
			}

			fn.Name = rt.ContainerName()
			fn.Project = p

			p.Functions[rt.ContainerName()] = fn
		}
	}

	if len(p.Functions) == 0 {
		handlerMatches := lo.Reduce(c.ConcreteHandlers, func(agg string, handler *HandlerConfig, idx int) string {
			if agg == "" {
				return handler.Match
			}
			return fmt.Sprintf("%s, %s", agg, handler.Match)
		}, "")

		return nil, fmt.Errorf("no functions were found within match on handlers '%s' in dir '%s', try a new pattern", handlerMatches, p.Dir)
	}

	return p, nil
}

func FunctionFromHandler(handlerFile string, config *HandlerConfig) (Function, error) {
	pterm.Debug.Println("Using function from " + handlerFile)

	return Function{
		Handler: handlerFile,
		Config:  config,
	}, nil
}
