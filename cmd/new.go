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

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/nitrictech/cli/pkg/view/tui"
	"github.com/nitrictech/cli/pkg/view/tui/commands/project"
	"github.com/nitrictech/cli/pkg/view/tui/teax"
)

var force bool

var newCmd = &cobra.Command{
	Use:   "new [projectName] [templateName]",
	Short: "Create a new project",
	Long:  `Creates a new Nitric project from a template.`,
	Example: `# For an interactive command that will ask the required questions
nitric new

# For a non-interactive command use the arguments.
nitric new hello-world "official/TypeScript - Starter" `,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName := ""
		if len(args) >= 1 {
			projectName = args[0]
		}

		templateName := ""
		if len(args) >= 2 {
			templateName = args[1]
		}

		if !tui.IsTerminal() && (templateName == "" || projectName == "") {
			return fmt.Errorf(`non-interactive environment detected, please provide all mandatory arguments e.g. nitric new hello-world "official/TypeScript - Starter"`)
		}

		projectModel, err := project.New(afero.NewOsFs(), project.Args{
			ProjectName:  projectName,
			TemplateName: templateName,
			Force:        force,
		})
		tui.CheckErr(err)

		// TODO add --force
		if _, err := teax.NewProgram(projectModel, tea.WithANSICompressor()).Run(); err != nil {
			return err
		}

		return nil
	},
	Args: cobra.MaximumNArgs(2),
}

func init() {
	newCmd.Flags().BoolVarP(&force, "force", "f", false, "force project creation, even in non-empty directories.")
	rootCmd.AddCommand(newCmd)
}
