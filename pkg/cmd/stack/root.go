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
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/AlecAivazis/survey/v2"
	"github.com/joho/godotenv"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/nitrictech/cli/pkg/build"
	"github.com/nitrictech/cli/pkg/codeconfig"
	"github.com/nitrictech/cli/pkg/output"
	"github.com/nitrictech/cli/pkg/project"
	"github.com/nitrictech/cli/pkg/provider"
	"github.com/nitrictech/cli/pkg/provider/types"
	"github.com/nitrictech/cli/pkg/stack"
	"github.com/nitrictech/cli/pkg/tasklet"
	"github.com/nitrictech/cli/pkg/utils"
)

var (
	confirmDown bool
	force       bool
	envFile     string
)

var stackCmd = &cobra.Command{
	Use:   "stack",
	Short: "Manage stacks (the deployed app containing multiple resources e.g. collection, bucket, topic)",
	Long: `Manage stacks (the deployed app containing multiple resources e.g. collection, bucket, topic).

A stack is a named update target, and a single project may have many of them.`,
	Example: `nitric stack up
nitric stack down
nitric stack list
`,
}

var newStackCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a new Nitric stack",
	Long:  `Creates a new Nitric stack.`,
	Run: func(cmd *cobra.Command, args []string) {
		name := ""
		err := survey.AskOne(&survey.Input{
			Message: "What do you want to call your new stack?",
		}, &name)
		cobra.CheckErr(err)

		pName := ""
		err = survey.AskOne(&survey.Select{
			Message: "Which Cloud do you wish to deploy to?",
			Default: stack.Aws,
			Options: stack.Providers,
		}, &pName)
		cobra.CheckErr(err)

		pc, err := project.ConfigFromProjectPath("")
		cobra.CheckErr(err)

		prov, err := provider.NewProvider(project.New(pc), &stack.Config{Name: name, Provider: pName}, map[string]string{}, &types.ProviderOpts{})
		cobra.CheckErr(err)

		sc, err := prov.Ask()
		cobra.CheckErr(err)

		err = sc.ToFile(filepath.Join(pc.Dir, fmt.Sprintf("nitric-%s.yaml", sc.Name)))
		cobra.CheckErr(err)
	},
	Args:        cobra.MaximumNArgs(2),
	Annotations: map[string]string{"commonCommand": "yes"},
}

var stackUpdateCmd = &cobra.Command{
	Use:     "update [-s stack]",
	Short:   "Create or update a deployed stack",
	Long:    `Create or update a deployed stack`,
	Example: `nitric stack update -s aws`,
	Run: func(cmd *cobra.Command, args []string) {

		//FIXME: Remove this error once multi-architecture support is complete
		if runtime.GOARCH != "amd64" {
			cobra.CheckErr(fmt.Errorf("only x86_64 CPU architectures are supported for the `nitric up` command currently.\nSee https://github.com/nitrictech/nitric/issues/283 for updated status on multi-architecture support"))
		}

		s, err := stack.ConfigFromOptions()
		cobra.CheckErr(err)

		config, err := project.ConfigFromProjectPath("")
		cobra.CheckErr(err)

		proj, err := project.FromConfig(config)
		cobra.CheckErr(err)

		log.SetOutput(output.NewPtermWriter(pterm.Debug))

		envFiles := utils.FilesExisting(".env", ".env.production", envFile)
		envMap := map[string]string{}
		if len(envFiles) > 0 {
			envMap, err = godotenv.Read(envFiles...)
			cobra.CheckErr(err)
		}

		codeAsConfig := tasklet.Runner{
			StartMsg: "Gathering configuration from code..",
			Runner: func(_ output.Progress) error {
				proj, err = codeconfig.Populate(proj, envMap)
				return err
			},
			StopMsg: "Configuration gathered",
		}
		tasklet.MustRun(codeAsConfig, tasklet.Opts{})

		p, err := provider.NewProvider(proj, s, envMap, &types.ProviderOpts{Force: force})
		cobra.CheckErr(err)

		if err := p.TryPullImages(); err != nil {
			pterm.Info.Print(err)
		}

		buildImages := tasklet.Runner{
			StartMsg: "Building Images",
			Runner: func(_ output.Progress) error {
				return build.Create(proj, s)
			},
			StopMsg: "Images built",
		}
		tasklet.MustRun(buildImages, tasklet.Opts{})

		d := &types.Deployment{}
		deploy := tasklet.Runner{
			StartMsg: "Deploying..",
			Runner: func(progress output.Progress) error {
				d, err = p.Up(progress)
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
	Use:   "down [-s stack]",
	Short: "Undeploy a previously deployed stack, deleting resources",
	Long:  `Undeploy a previously deployed stack, deleting resources`,
	Example: `nitric stack down -s aws

# To not be prompted, use -y
nitric stack down -e aws -y`,
	Run: func(cmd *cobra.Command, args []string) {
		if !confirmDown {
			confirm := ""
			err := survey.AskOne(&survey.Select{
				Message: "Warning - This operation will destroy your stack, all deployed resources will be removed. Are you sure you want to proceed?",
				Default: "No",
				Options: []string{"Yes", "No"},
			}, &confirm)
			cobra.CheckErr(err)
			if confirm != "Yes" {
				pterm.Info.Println("Cancelling command")
				os.Exit(0)
			}
		}

		s, err := stack.ConfigFromOptions()
		cobra.CheckErr(err)

		config, err := project.ConfigFromProjectPath("")
		cobra.CheckErr(err)

		proj, err := project.FromConfig(config)
		cobra.CheckErr(err)

		p, err := provider.NewProvider(proj, s, map[string]string{}, &types.ProviderOpts{Force: true})
		cobra.CheckErr(err)

		deploy := tasklet.Runner{
			StartMsg: "Deleting..",
			Runner: func(progress output.Progress) error {
				if err := p.TryPullImages(); err != nil {
					pterm.Info.Print(err)
				}

				return p.Down(progress)
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
	Use:   "list [-s stack]",
	Short: "List all project stacks and their status",
	Long:  `List all project stacks and their status`,
	Example: `nitric stack list

nitric stack list -s aws
`,
	Run: func(cmd *cobra.Command, args []string) {
		s, err := stack.ConfigFromOptions()
		cobra.CheckErr(err)

		config, err := project.ConfigFromProjectPath("")
		cobra.CheckErr(err)

		proj, err := project.FromConfig(config)
		cobra.CheckErr(err)

		p, err := provider.NewProvider(proj, s, map[string]string{}, &types.ProviderOpts{})
		cobra.CheckErr(err)

		deps, err := p.List()
		cobra.CheckErr(err)

		output.Print(deps)
	},
	Args:    cobra.ExactArgs(0),
	Aliases: []string{"ls"},
}

func RootCommand() *cobra.Command {
	stackCmd.AddCommand(newStackCmd)

	stackCmd.AddCommand(stackUpdateCmd)
	stackUpdateCmd.Flags().StringVarP(&envFile, "env-file", "e", "", "--env-file config/.my-env")
	stackUpdateCmd.Flags().BoolVarP(&force, "force", "f", false, "force override previous deployment")
	cobra.CheckErr(stack.AddOptions(stackUpdateCmd, false))

	stackCmd.AddCommand(stackDeleteCmd)
	stackDeleteCmd.Flags().BoolVarP(&confirmDown, "yes", "y", false, "confirm the destruction of the stack")
	cobra.CheckErr(stack.AddOptions(stackDeleteCmd, false))

	stackCmd.AddCommand(stackListCmd)
	cobra.CheckErr(stack.AddOptions(stackListCmd, false))

	return stackCmd
}
