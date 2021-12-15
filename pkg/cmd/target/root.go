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

package target

import (
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nitrictech/newcli/pkg/output"
	"github.com/nitrictech/newcli/pkg/target"
)

var targetCmd = &cobra.Command{
	Use:   "target",
	Short: "work with target objects",
	Long: `Choose an action to perform on a target, e.g.
	nitric target list
`,
}

var targetListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured targets",
	Long:  `Lists configured taregts.`,
	Run: func(cmd *cobra.Command, args []string) {
		targets := map[string]target.Target{}
		cobra.CheckErr(mapstructure.Decode(viper.GetStringMap("targets"), &targets))
		output.Print(targets)
	},
	Args: cobra.MaximumNArgs(0),
}

func RootCommand() *cobra.Command {
	targetCmd.AddCommand(targetListCmd)
	return targetCmd
}
