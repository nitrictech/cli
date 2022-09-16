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
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/nitrictech/cli/pkg/pflagext"
	"github.com/nitrictech/cli/pkg/utils"
)

var stack string

func GetStacks() ([]string, error) {
	stackFiles, err := utils.GlobInDir(".", "nitric-*.yaml")
	if err != nil {
		return nil, err
	}

	stacks := []string{}

	for _, sf := range stackFiles {
		stacks = append(stacks, strings.TrimSuffix(strings.TrimPrefix(sf, "nitric-"), ".yaml"))
	}

	return stacks, nil
}

// Assume the project is in the currentDirectory
func ConfigFromOptions() (*Config, error) {
	sName := stack // Default to the supplied stack
	if sName == "" {
		stacks, err := GetStacks()
		if err != nil {
			return nil, err
		}

		if len(stacks) == 1 {
			// If there is only one stack use that
			sName = stacks[0]
		} else if len(stacks) > 0 {
			// List all the other stacks
			err = survey.AskOne(&survey.Select{
				Message: "Which stack do you wish to deploy?",
				Options: stacks,
			}, &sName)

			if err != nil {
				return nil, err
			}
		} else {
			return nil, errors.New("No nitric stacks found, run `nitric stack new` to create a new stack")
		}
	}

	return configFromFile("nitric-" + sName + ".yaml")
}

func configFromFile(file string) (*Config, error) {
	s := &Config{}

	yamlFile, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("no nitric stack found (unable to find %s). If you haven't created a stack yet, run `nitric stack new` to get started", file)
	}

	err = yaml.Unmarshal(yamlFile, s)

	return s, err
}

func AddOptions(cmd *cobra.Command, providerOnly bool) error {
	stacks, err := GetStacks()
	if err != nil {
		return err
	}

	cmd.Flags().VarP(pflagext.NewStringEnumVar(&stack, stacks, ""), "stack", "s", "use this to refer to a stack configuration nitric-<stackname>.yaml")

	return cmd.RegisterFlagCompletionFunc("stack", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return stacks, cobra.ShellCompDirectiveDefault
	})
}
