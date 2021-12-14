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
	"path"
	"regexp"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

	"github.com/nitrictech/newcli/pkg/stack"
	"github.com/nitrictech/newcli/pkg/templates"
)

var (
	force       bool
	nameRegex   = regexp.MustCompile(`^([a-zA-Z0-9-])*$`)
	stackNameQu = survey.Question{
		Name:     "stackName",
		Prompt:   &survey.Input{Message: "What is the name of the stack?"},
		Validate: validateName,
	}
	templateNameQu = survey.Question{
		Name: "templateName",
	}
)

var stackCmd = &cobra.Command{
	Use:   "stack",
	Short: "work with stack objects",
	Long: `Choose an action to perform on a stack, e.g.
nitric stack create
`,
}

var stackCreateCmd = &cobra.Command{
	Use:   "create [name] [template]",
	Short: "create a new application stack",
	Long:  `Creates a new Nitric application stack from a template.`,
	Run: func(cmd *cobra.Command, args []string) {
		answers := struct {
			StackName    string
			TemplateName string
		}{}

		rc := templates.NewRepoContent("nitrictech", "stack-templates")
		dirs, err := rc.ListSubDirectories("")
		cobra.CheckErr(err)
		templateNameQu.Prompt = &survey.Select{
			Message: "Choose a template:",
			Options: dirs,
			Default: "go-stack",
		}

		var qs = []*survey.Question{}

		if len(args) > 0 && stackNameQu.Validate(args[0]) == nil {
			answers.StackName = args[0]
		} else {
			qs = append(qs, &stackNameQu)
		}

		if len(args) > 1 && templateNameQu.Validate(args[1]) == nil {
			answers.TemplateName = args[1]
		} else {
			qs = append(qs, &templateNameQu)
		}

		if len(qs) > 0 {
			err = survey.Ask(qs, &answers)
			cobra.CheckErr(err)
		}

		err = rc.DownloadDirectoryContents(answers.TemplateName, "./"+answers.StackName, force)
		cobra.CheckErr(err)
		err = setStackName(answers.StackName)
		cobra.CheckErr(err)
	},
	Args: cobra.MaximumNArgs(2),
}

func RootCommand() *cobra.Command {
	stackCreateCmd.Flags().BoolVarP(&force, "force", "f", false, "force stack creation, even in non-empty directories.")
	stackCmd.AddCommand(stackCreateCmd)
	return stackCmd
}

func validateName(val interface{}) error {
	name, ok := val.(string)
	if !ok {
		return errors.New("stack name must be a string")
	}
	if name == "" {
		return errors.New("stack name can not be empty")
	}
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") || !nameRegex.MatchString(name) {
		return errors.New("invalid stack name, only letters, numbers and dashes are supported")
	}
	return nil
}

func setStackName(name string) error {
	stackFilePath := path.Join("./", name, "nitric.yaml")
	s, err := stack.FromFile(stackFilePath)
	if err != nil {
		return err
	}
	s.Name = name
	return s.ToFile(stackFilePath)
}
