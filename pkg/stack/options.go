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

package stack

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pterm/pterm"

	"github.com/nitrictech/cli/pkg/project"
	"github.com/nitrictech/cli/pkg/runtime"
	"github.com/nitrictech/cli/pkg/utils"
)

var (
	stackPath string
)

func FromConfig(p *project.Config) (*Stack, error) {
	s := New(p.Name, p.Dir)

	for _, g := range p.Handlers {
		maybeFile := filepath.Join(s.Dir, g)
		if _, err := os.Stat(maybeFile); err != nil {
			fs, err := utils.GlobInDir(stackPath, g)
			if err != nil {
				return nil, err
			}
			for _, f := range fs {
				fn := FunctionFromHandler(f, s.Dir)
				s.Functions[fn.Name] = fn
			}
		} else {
			fn := FunctionFromHandler(g, s.Dir)
			s.Functions[fn.Name] = fn
		}
	}

	if len(s.Functions) == 0 {
		return nil, fmt.Errorf("no functions were found with the glob '%s', try a new pattern", strings.Join(p.Handlers, ","))
	}

	return s, nil
}

func FunctionFromHandler(h, stackDir string) Function {
	pterm.Debug.Println("Using function from " + h)
	rt, _ := runtime.NewRunTimeFromHandler(h)
	fn := Function{
		ComputeUnit: ComputeUnit{Name: rt.ContainerName()},
		Handler:     h,
	}

	return fn
}
