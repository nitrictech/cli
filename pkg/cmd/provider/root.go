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

package provider

import (
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var providerCmd = &cobra.Command{
	Use:   "provider",
	Short: "Work with a provider",
	Long: `List availabe providers, e.g.
	nitric provider list
`,
}

var providerListCmd = &cobra.Command{
	Use:   "list",
	Short: "list providers",
	Long:  `Lists Nitric providers.`,
	Run: func(cmd *cobra.Command, args []string) {
		notice := color.New(color.Bold, color.FgGreen).PrintlnFunc()
		notice("Don't forget this... %v")
	},
	Args: cobra.MaximumNArgs(2),
}

func RootCommand() *cobra.Command {
	providerCmd.AddCommand(providerListCmd)
	return providerCmd
}
