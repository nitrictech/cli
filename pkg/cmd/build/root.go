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

package build

import (
	"github.com/spf13/cobra"

	"github.com/nitrictech/newcli/pkg/build"
	"github.com/nitrictech/newcli/pkg/output"
	"github.com/nitrictech/newcli/pkg/stack"
	"github.com/nitrictech/newcli/pkg/target"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Work with a build",
	Long: `Build a project, e.g.
	nitric build create
`,
}

var buildCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "create a new application build",
	Long:  `Creates a new Nitric application build.`,
	Run: func(cmd *cobra.Command, args []string) {
		t := target.FromOptions()
		s, err := stack.FromOptions()
		cobra.CheckErr(err)
		cobra.CheckErr(build.Create(s, t))
	},
	Args: cobra.MaximumNArgs(0),
}

var buildListCmd = &cobra.Command{
	Use:   "list",
	Short: "list builds done for this stack",
	Long:  `Lists Nitric application builds done for this stack.`,
	Run: func(cmd *cobra.Command, args []string) {
		s, err := stack.FromOptions()
		cobra.CheckErr(err)
		out, err := build.List(s)
		cobra.CheckErr(err)
		output.Print(out)
	},
	Args: cobra.MaximumNArgs(0),
}

func RootCommand() *cobra.Command {
	buildCmd.AddCommand(buildCreateCmd)
	cobra.CheckErr(target.AddOptions(buildCreateCmd, true))
	stack.AddOptions(buildCreateCmd)
	buildCmd.AddCommand(buildListCmd)
	stack.AddOptions(buildListCmd)
	return buildCmd
}
