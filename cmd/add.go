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

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/nitrictech/cli/pkg/view/tui"
	add_website "github.com/nitrictech/cli/pkg/view/tui/commands/website"
	"github.com/nitrictech/cli/pkg/view/tui/teax"
)

// addCmd acts as a parent command for adding different types of resources
// e.g., websites or other components in the future.
var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add new resources to your Nitric project",
	Long: `Add new components such as websites to an existing Nitric project.
Run 'nitric add website' to add a new website.`,
	Example: `# Add a new website interactively
nitric add website`,
}

var addWebsiteCmd = &cobra.Command{
	Use:   "website [websiteName] [toolName]",
	Short: "Add a new website to your Nitric project",
	Long:  `Add a new website to your Nitric project, with optional tool selection.`,
	Example: `# Interactive website addition
nitric add website

# Add a new website with a specific tool
nitric add website my-site astro`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fs := afero.NewOsFs()

		websiteName := ""
		if len(args) >= 1 {
			websiteName = args[0]
		}

		toolName := ""
		if len(args) >= 2 {
			toolName = args[1]
		}

		if !tui.IsTerminal() {
			return fmt.Errorf("non-interactive mode is not supported by this command")
		}

		websitePath, err := cmd.Flags().GetString("path")
		if err != nil {
			return fmt.Errorf("failed to get path flag: %w", err)
		}

		websiteModel, err := add_website.New(fs, add_website.Args{
			WebsiteName: websiteName,
			ToolName:    toolName,
			WebsitePath: websitePath,
		})
		tui.CheckErr(err)

		if _, err := teax.NewProgram(websiteModel).Run(); err != nil {
			return err
		}

		return nil
	},
	Args: cobra.MaximumNArgs(2),
}

func init() {
	rootCmd.AddCommand(addCmd)

	addCmd.AddCommand(addWebsiteCmd)

	addStackCmd := &cobra.Command{
		Use:   "stack [stackName] [providerName]",
		Short: newStackCmd.Short,
		Long:  newStackCmd.Long,
		RunE:  newStackCmd.RunE,
		Args:  newStackCmd.Args,
	}
	addStackCmd.Flags().AddFlagSet(newStackCmd.Flags())
	addCmd.AddCommand(addStackCmd)

	addWebsiteCmd.Flags().StringP("path", "p", "", "base url path for the website, e.g. /my-site")
}
