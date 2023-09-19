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

package cmd

import (
	"errors"
	"regexp"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

	"github.com/nitrictech/cli/pkg/operations/project_new"
)

var (
	force         bool
	nameRegex     = regexp.MustCompile(`^([a-zA-Z0-9-])*$`)
	projectNameQu = survey.Question{
		Name:     "projectName",
		Prompt:   &survey.Input{Message: "What is the name of the project?"},
		Validate: validateName,
	}
	templateNameQu = survey.Question{
		Name: "templateName",
	}
)

var newCmd = &cobra.Command{
	Use:   "new [projectName] [templateName]",
	Short: "Create a new project",
	Long:  `Creates a new Nitric project from a template.`,
	Example: `# For an interactive command that will ask the required questions
nitric new

# For a non-interactive command use the arguments.
nitric new hello-world "official/TypeScript - Starter" `,
	Run: func(cmd *cobra.Command, args []string) {
		project_new.Run(cmd.Context(), args)
	},
	Args: cobra.MaximumNArgs(2),
}

func validateName(val interface{}) error {
	name, ok := val.(string)
	if !ok {
		return errors.New("project name must be a string")
	}

	if name == "" {
		return errors.New("project name can not be empty")
	}

	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") || !nameRegex.MatchString(name) {
		return errors.New("invalid project name, only letters, numbers and dashes are supported")
	}

	return nil
}

func init() {
	newCmd.Flags().BoolVarP(&force, "force", "f", false, "force project creation, even in non-empty directories.")
	rootCmd.AddCommand(newCmd)
}
