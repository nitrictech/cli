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

package deployment

import (
	"github.com/spf13/cobra"

	"github.com/nitrictech/cli/pkg/build"
	"github.com/nitrictech/cli/pkg/codeconfig"
	"github.com/nitrictech/cli/pkg/output"
	"github.com/nitrictech/cli/pkg/provider"
	"github.com/nitrictech/cli/pkg/stack"
	"github.com/nitrictech/cli/pkg/target"
	"github.com/nitrictech/cli/pkg/tasklet"
)

var deploymentName string

var deploymentCmd = &cobra.Command{
	Use:   "deployment",
	Short: "Work with a deployment",
	Long: `Stack deployment command set.

The deployment commands generally need 3 things:
1. a target (either explicitly with "-t <targetname> or defined in the config)
2. a deployment name (either explicitly with -n <deployment name> or use the default name of "dep")
3. a stack definition, this could be a supplied by any of the following ways:
  - Provide a nitric.yaml configuration file, the path to which is provided either explicitly with "-s <stack dir> or in the current director.
  - A Glob to the functions from which the configuration is derived.
	`,
	Example: `nitric deployment apply
nitric deployment delete
nitric deployment list
`,
}

var deploymentApplyCmd = &cobra.Command{
	Use:   "apply [handlerGlob]",
	Short: "Create or Update a new application deployment",
	Long:  `Applies a Nitric application deployment.`,
	Example: `nitric deployment apply

# When using a nitric.yaml
nitric deployment apply -n prod-aws -s ../project/ -t prod

# When using code-as-config, specify the functions.
nitric deployment apply -n prod-aws -s ../project/ -t prod "functions/*.ts"
		`,
	Run: func(cmd *cobra.Command, args []string) {
		t, err := target.FromOptions()
		cobra.CheckErr(err)

		s, err := stack.FromOptions()
		if err != nil && len(args) > 0 {
			s, err = stack.FromGlobArgs(args)
			cobra.CheckErr(err)

			s, err = codeconfig.Populate(s)
		}
		cobra.CheckErr(err)

		p, err := provider.NewProvider(s, t)
		cobra.CheckErr(err)

		buildImages := tasklet.Runner{
			StartMsg: "Building Images",
			Runner: func(tCtx tasklet.TaskletContext) error {
				return build.Create(s, t)
			},
			StopMsg: "Images built!",
		}
		tasklet.MustRun(buildImages, tasklet.Opts{})

		deploy := tasklet.Runner{
			StartMsg: "Deploying..",
			Runner: func(tCtx tasklet.TaskletContext) error {
				return p.Apply(deploymentName)
			},
			StopMsg: "Deployment complete!",
		}
		tasklet.MustRun(deploy, tasklet.Opts{})
	},
	Args: cobra.MinimumNArgs(0),
}

var deploymentDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete an application deployment",
	Long:  `Delete a Nitric application deployment.`,
	Example: `nitric deployment delete
nitric deployment delete -s ../project/ -t prod
nitric deployment delete -n prod-aws -s ../project/ -t prod
`,
	Run: func(cmd *cobra.Command, args []string) {
		t, err := target.FromOptions()
		cobra.CheckErr(err)

		s, err := stack.FromOptionsMinimal()
		cobra.CheckErr(err)

		p, err := provider.NewProvider(s, t)
		cobra.CheckErr(err)

		deploy := tasklet.Runner{
			StartMsg: "Deleting..",
			Runner: func(tCtx tasklet.TaskletContext) error {
				return p.Delete(deploymentName)
			},
			StopMsg: "Deployment deleted!",
		}
		tasklet.MustRun(deploy, tasklet.Opts{})

	},
	Args: cobra.ExactArgs(0),
}

var deploymentListCmd = &cobra.Command{
	Use:   "list",
	Short: "list deployments for a stack",
	Long:  `Lists Nitric application deployments for a stack.`,
	Example: `nitric list
nitric deployment list -s ../project/ -t prod
`,
	Run: func(cmd *cobra.Command, args []string) {
		t, err := target.FromOptions()
		cobra.CheckErr(err)

		s, err := stack.FromOptionsMinimal()
		cobra.CheckErr(err)

		p, err := provider.NewProvider(s, t)
		cobra.CheckErr(err)

		deps, err := p.List()
		cobra.CheckErr(err)

		output.Print(deps)
	},
	Args: cobra.ExactArgs(0),
}

func RootCommand() *cobra.Command {
	deploymentCmd.AddCommand(deploymentApplyCmd)
	deploymentApplyCmd.Flags().StringVarP(&deploymentName, "name", "n", "dep", "the name of the deployment")
	cobra.CheckErr(target.AddOptions(deploymentApplyCmd, false))
	stack.AddOptions(deploymentApplyCmd)

	deploymentCmd.AddCommand(deploymentDeleteCmd)
	deploymentDeleteCmd.Flags().StringVarP(&deploymentName, "name", "n", "dep", "the name of the deployment")
	cobra.CheckErr(target.AddOptions(deploymentDeleteCmd, false))
	stack.AddOptions(deploymentDeleteCmd)

	deploymentCmd.AddCommand(deploymentListCmd)
	stack.AddOptions(deploymentListCmd)
	cobra.CheckErr(target.AddOptions(deploymentListCmd, false))
	return deploymentCmd
}
