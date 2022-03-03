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

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

	"github.com/nitrictech/cli/pkg/stack"
	"github.com/nitrictech/cli/pkg/templates"
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

var newProjectCmd = &cobra.Command{
	Use:   "new [projectName] [templateName]",
	Short: "create a new nitric project",
	Long:  `Creates a new Nitric project from a template.`,
	Run: func(cmd *cobra.Command, args []string) {
		answers := struct {
			ProjectName  string
			TemplateName string
		}{}

		downloadr := templates.NewDownloader()
		dirs, err := downloadr.Names()
		cobra.CheckErr(err)

		templateNameQu.Prompt = &survey.Select{
			Message: "Choose a template:",
			Options: dirs,
		}
		templateNameQu.Validate = func(ans interface{}) error {
			if len(args) < 2 {
				return nil
			}

			a, ok := ans.(string)
			if !ok {
				return errors.New("wrong type, need a string")
			}

			if downloadr.Get(a) == nil {
				return fmt.Errorf("%s not in %v", a, dirs)
			}
			return nil
		}

		qs := []*survey.Question{}
		if len(args) > 0 && stackNameQu.Validate(args[0]) == nil {
			answers.ProjectName = args[0]
		} else {
			qs = append(qs, &stackNameQu)
		}

		if len(args) > 1 && templateNameQu.Validate(args[1]) == nil {
			answers.TemplateName = args[1]
		} else {
			qs = append(qs, &templateNameQu)
			args = []string{} // reassign args to ensure validation works correctly.
		}

		if len(qs) > 0 {
			err = survey.Ask(qs, &answers)
			cobra.CheckErr(err)
		}

		err = downloadr.DownloadDirectoryContents(answers.TemplateName, "./"+answers.ProjectName, force)
		cobra.CheckErr(err)
		err = setStackName(answers.ProjectName)
		cobra.CheckErr(err)
	},
	Args: cobra.MaximumNArgs(2),
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
	stackFilePath := filepath.Join("./", name, "nitric.yaml")
	// Skip non nitric.yaml template renaming (config as code)
	if _, err := os.Stat(stackFilePath); errors.Is(err, os.ErrNotExist) {
		return nil
	}

	s, err := stack.FromFile(stackFilePath)
	if err != nil {
		return err
	}
	s.Name = name
	return s.ToFile(stackFilePath)
}
