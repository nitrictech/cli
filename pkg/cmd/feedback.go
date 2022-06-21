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
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/nitrictech/cli/pkg/ghissue"
)

var feedbackCmd = &cobra.Command{
	Use:     "feedback",
	Short:   "Provide feedback on your experience with nitric",
	Long:    `Provide feedback on your experience with nitric.`,
	Example: `nitric feedback`,
	Run: func(cmd *cobra.Command, args []string) {
		answers := struct {
			Repo  string
			Kind  string
			Title string
			Body  string
		}{}

		d, err := ghissue.Gather()
		cobra.CheckErr(err)

		diag, err := yaml.Marshal(d)
		cobra.CheckErr(err)

		qs := []*survey.Question{
			{
				Name: "repo",
				Prompt: &survey.Select{
					Message: "What is the name of the repo?",
					Options: []string{"cli", "nitric", "docs", "apis", "node-sdk", "go-sdk"},
				},
			},
			{
				Name: "kind",
				Prompt: &survey.Select{
					Message: "What kind of feedback do you want to give?",
					Options: []string{"bug", "feature-request", "question"},
				},
			},
			{
				Name: "title",
				Prompt: &survey.Input{
					Message: "How would you like to title your feedback?",
				},
			},
			{
				Name: "body",
				Prompt: &survey.Editor{
					Message:       "Please write your feedback",
					Default:       string(diag),
					HideDefault:   true,
					AppendDefault: true,
				},
			},
		}
		err = survey.Ask(qs, &answers)
		cobra.CheckErr(err)

		pterm.Info.Println("Please create a github issue by clicking on the link below")
		fmt.Println(ghissue.IssueLink(answers.Repo, answers.Kind, answers.Title, answers.Body))
	},
}
