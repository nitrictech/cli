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
	"log"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/nitrictech/cli/pkg/build"
	"github.com/nitrictech/cli/pkg/codeconfig"
	"github.com/nitrictech/cli/pkg/output"
	"github.com/nitrictech/cli/pkg/project"
	"github.com/nitrictech/cli/pkg/provider"
	"github.com/nitrictech/cli/pkg/provider/types"
	"github.com/nitrictech/cli/pkg/target"
	"github.com/nitrictech/cli/pkg/tasklet"
)

var stackName string

var stackCmd = &cobra.Command{
	Use:   "stack",
	Short: "Manage stacks",
	Long: `Manage stacks.

A stack is a named update target, and a single project may have many of them.

The stack commands generally need 3 things:
1. a target (either explicitly with "-t <targetname> or defined in the config)
2. a stack name (either explicitly with -n <stack name> or use the default name of "dep")
3. a project configuration (seed config from nitric.yaml and the remainder is automatically collected from the code in functions).`,
	Example: `nitric stack up
nitric stack down
nitric stack list
`,
}

var stackUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Create or Update a new application stack",
	Long:  `Updates a Nitric application stack.`,
	Example: `# Configured default handlerGlob (stack in the current directory).
nitric stack up -t aws

# use a custom stack name
nitric stack up -n prod -t aws`,
	Run: func(cmd *cobra.Command, args []string) {
		t, err := target.FromOptions()
		cobra.CheckErr(err)

		config, err := project.ConfigFromFile()
		cobra.CheckErr(err)

		proj, err := project.FromConfig(config)
		cobra.CheckErr(err)

		log.SetOutput(output.NewPtermWriter(pterm.Debug))

		codeAsConfig := tasklet.Runner{
			StartMsg: "Gathering configuration from code..",
			Runner: func(_ output.Progress) error {
				proj, err = codeconfig.Populate(proj)
				return err
			},
			StopMsg: "Configuration gathered",
		}
		tasklet.MustRun(codeAsConfig, tasklet.Opts{})

		p, err := provider.NewProvider(proj, t)
		cobra.CheckErr(err)

		buildImages := tasklet.Runner{
			StartMsg: "Building Images",
			Runner: func(_ output.Progress) error {
				return build.Create(proj, t)
			},
			StopMsg: "Images built",
		}
		tasklet.MustRun(buildImages, tasklet.Opts{})

		d := &types.Deployment{}
		deploy := tasklet.Runner{
			StartMsg: "Deploying..",
			Runner: func(progress output.Progress) error {
				d, err = p.Apply(progress, stackName)
				return err
			},
			StopMsg: "Stack",
		}
		tasklet.MustRun(deploy, tasklet.Opts{SuccessPrefix: "Deployed"})

		rows := [][]string{{"API", "Endpoint"}}
		for k, v := range d.ApiEndpoints {
			rows = append(rows, []string{k, v})
		}
		_ = pterm.DefaultTable.WithBoxed().WithData(rows).Render()
	},
	Args:    cobra.MinimumNArgs(0),
	Aliases: []string{"up"},
}

var stackDeleteCmd = &cobra.Command{
	Use:   "down",
	Short: "Brings downs an application stack",
	Long:  `Brings downs a Nitric application stack.`,
	Example: `nitric stack down
nitric stack down -t prod
nitric stack down -n prod-aws -t prod
`,
	Run: func(cmd *cobra.Command, args []string) {
		t, err := target.FromOptions()
		cobra.CheckErr(err)

		config, err := project.ConfigFromFile()
		cobra.CheckErr(err)

		proj, err := project.FromConfig(config)
		cobra.CheckErr(err)

		p, err := provider.NewProvider(proj, t)
		cobra.CheckErr(err)

		deploy := tasklet.Runner{
			StartMsg: "Deleting..",
			Runner: func(progress output.Progress) error {
				return p.Delete(progress, stackName)
			},
			StopMsg: "Stack",
		}
		tasklet.MustRun(deploy, tasklet.Opts{
			SuccessPrefix: "Deleted",
		})
	},
	Args: cobra.ExactArgs(0),
}

var stackListCmd = &cobra.Command{
	Use:   "list",
	Short: "list stacks for a project",
	Long:  `Lists Nitric application stacks for a project.`,
	Example: `nitric list
nitric stack list -t prod
`,
	Run: func(cmd *cobra.Command, args []string) {
		t, err := target.FromOptions()
		cobra.CheckErr(err)

		config, err := project.ConfigFromFile()
		cobra.CheckErr(err)

		proj, err := project.FromConfig(config)
		cobra.CheckErr(err)

		p, err := provider.NewProvider(proj, t)
		cobra.CheckErr(err)

		deps, err := p.List()
		cobra.CheckErr(err)

		output.Print(deps)
	},
	Args:    cobra.ExactArgs(0),
	Aliases: []string{"ls"},
}

func RootCommand() *cobra.Command {
	stackCmd.AddCommand(stackUpdateCmd)
	stackUpdateCmd.Flags().StringVarP(&stackName, "name", "n", "dep", "the name of the project")
	cobra.CheckErr(target.AddOptions(stackUpdateCmd, false))

	stackCmd.AddCommand(stackDeleteCmd)
	stackDeleteCmd.Flags().StringVarP(&stackName, "name", "n", "dep", "the name of the project")
	cobra.CheckErr(target.AddOptions(stackDeleteCmd, false))

	stackCmd.AddCommand(stackListCmd)
	cobra.CheckErr(target.AddOptions(stackListCmd, false))
	return stackCmd
}
