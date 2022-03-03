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
	"os"
	"path/filepath"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nitrictech/cli/pkg/pflagext"
	"github.com/nitrictech/cli/pkg/runtime"
	"github.com/nitrictech/cli/pkg/utils"
)

var (
	stackPath string
)

func EnsureRuntimeDefaults() bool {
	defaults := map[string]map[string]interface{}{
		"ts": {
			"functionglob": "functions/*.ts",
		},
		"js": {
			"functionglob": "functions/*.js",
		},
		"go": {
			"functionglob": "functions/*/*.go",
		},
	}
	written := false
	runtime, err := utils.ToStringMapStringMapStringE(viper.Get("runtime"))
	if err != nil {
		fmt.Println("ERROR: runtime configuration in the wrong format")
		return false
	}

	for rtName, rt := range defaults {
		if _, ok := runtime[rtName]; !ok {
			runtime[rtName] = rt
			written = true
		}
	}
	if written {
		viper.Set("runtime", runtime)
	}
	return written
}

func defaultGlobsFromConfig() []string {
	globs := []string{}
	runtime, err := utils.ToStringMapStringMapStringE(viper.Get("runtime"))
	if err != nil {
		return globs
	}
	for _, rt := range runtime {
		globs = append(globs, rt["functionglob"].(string))
	}

	return globs
}

func FromOptions(glob []string) (*Project, error) {
	s, err := FromOptionsMinimal()
	if err != nil {
		return nil, err
	}

	if len(glob) == 0 {
		glob = defaultGlobsFromConfig()
	}

	for _, g := range glob {
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
		return nil, fmt.Errorf("no functions were found with the glob '%s', try a new pattern", strings.Join(glob, ","))
	}

	return s, nil
}

func FromOptionsMinimal() (*Project, error) {
	ss, err := os.Stat(stackPath)
	if err != nil {
		return nil, err
	}

	sDir := stackPath
	if !ss.IsDir() {
		sDir = filepath.Dir(stackPath)
	}

	// get the abs dir in case user provides "."
	absDir, err := filepath.Abs(sDir)
	if err != nil {
		return nil, err
	}
	s := New(filepath.Base(absDir), sDir)

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

func AddOptions(cmd *cobra.Command) {
	wd, err := os.Getwd()
	cobra.CheckErr(err)
	cmd.Flags().VarP(pflagext.NewPathVar(&stackPath, pflagext.AllowFileAndDir, wd), "stack", "s", "path to the stack")
}
