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
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var deploymentCmd = &cobra.Command{
	Use:   "deployment",
	Short: "Work with a deployment",
	Long: `Delopy a project, e.g.
	nitric deployment create
	nitric deployment delete
	nitric deployment list
`,
}

var deploymentCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "create a new application deployment",
	Long:  `Creates a new Nitric application deployment.`,
	Run: func(cmd *cobra.Command, args []string) {
		notice := color.New(color.Bold, color.FgGreen).PrintlnFunc()
		notice("Don't forget this... %v")
	},
	Args: cobra.MaximumNArgs(2),
}

var deploymentListCmd = &cobra.Command{
	Use:   "list [name]",
	Short: "list deployments done for this stack",
	Long:  `Lists Nitric application deployments done for this stack.`,
	Run: func(cmd *cobra.Command, args []string) {
		notice := color.New(color.Bold, color.FgGreen).PrintlnFunc()
		notice("Don't forget this... %v")
	},
	Args: cobra.MaximumNArgs(2),
}

func RootCommand() *cobra.Command {
	deploymentCmd.AddCommand(deploymentCreateCmd)
	deploymentCmd.AddCommand(deploymentListCmd)
	return deploymentCmd
}
