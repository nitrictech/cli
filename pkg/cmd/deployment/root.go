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

	"github.com/nitrictech/newcli/pkg/output"
	"github.com/nitrictech/newcli/pkg/provider"
	"github.com/nitrictech/newcli/pkg/stack"
	"github.com/nitrictech/newcli/pkg/target"
)

var deploymentCmd = &cobra.Command{
	Use:   "deployment",
	Short: "Work with a deployment",
	Long: `Delopy a project, e.g.
	nitric deployment create
	nitric deployment delete
	nitric deployment list
	nitric deployment describe
`,
}

var deploymentApplyCmd = &cobra.Command{
	Use:   "apply [name]",
	Short: "Create or Update a new application deployment",
	Long:  `Applies a Nitric application deployment.`,
	Run: func(cmd *cobra.Command, args []string) {
		t, err := target.FromOptions()
		cobra.CheckErr(err)
		s, err := stack.FromOptions()
		cobra.CheckErr(err)
		p, err := provider.NewProvider(s, t)
		cobra.CheckErr(err)
		cobra.CheckErr(p.Apply(args[0]))
	},
	Args: cobra.ExactArgs(1),
}

var deploymentDeleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete an application deployment",
	Long:  `Delete a Nitric application deployment.`,
	Run: func(cmd *cobra.Command, args []string) {
		t, err := target.FromOptions()
		cobra.CheckErr(err)
		s, err := stack.FromOptions()
		cobra.CheckErr(err)
		p, err := provider.NewProvider(s, t)
		cobra.CheckErr(err)
		cobra.CheckErr(p.Delete(args[0]))
	},
	Args: cobra.ExactArgs(1),
}

var deploymentListCmd = &cobra.Command{
	Use:   "list",
	Short: "list deployments for a stack",
	Long:  `Lists Nitric application deployments for a stack.`,
	Run: func(cmd *cobra.Command, args []string) {
		t, err := target.FromOptions()
		cobra.CheckErr(err)
		s, err := stack.FromOptions()
		cobra.CheckErr(err)
		p, err := provider.NewProvider(s, t)
		cobra.CheckErr(err)
		deps, err := p.List()
		cobra.CheckErr(err)
		output.Print(deps)
	},
	Args: cobra.MaximumNArgs(0),
}

func RootCommand() *cobra.Command {
	deploymentCmd.AddCommand(deploymentApplyCmd)
	cobra.CheckErr(target.AddOptions(deploymentApplyCmd, false))
	stack.AddOptions(deploymentApplyCmd)

	deploymentCmd.AddCommand(deploymentDeleteCmd)
	cobra.CheckErr(target.AddOptions(deploymentDeleteCmd, false))
	stack.AddOptions(deploymentDeleteCmd)

	deploymentCmd.AddCommand(deploymentListCmd)
	stack.AddOptions(deploymentListCmd)
	cobra.CheckErr(target.AddOptions(deploymentListCmd, false))
	return deploymentCmd
}
