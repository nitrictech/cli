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
	"github.com/nitrictech/cli/pkg/view/tui"
	add_website "github.com/nitrictech/cli/pkg/view/tui/commands/website"
	"github.com/nitrictech/cli/pkg/view/tui/teax"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
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

var forceAddWebsite bool

var addWebsiteCmd = &cobra.Command{
	Use:   "website [websiteName] [frameworkName]",
	Short: "Add a new website to your Nitric project",
	Long:  `Add a new website to your Nitric project, with optional framework selection.`,
	Example: `# Interactive website addition
nitric add website

# Non-interactive
nitric add website my-site astro`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fs := afero.NewOsFs()

		websiteName := ""
		if len(args) >= 1 {
			websiteName = args[0]
		}

		frameworkName := ""
		if len(args) >= 2 {
			frameworkName = args[1]
		}

		if !tui.IsTerminal() && (websiteName == "" || frameworkName == "") {
			return fmt.Errorf(`non-interactive environment detected, please provide all mandatory arguments e.g. nitric add website my-site nextjs`)
		}

		// get base url path for the website
		websitePath, err := cmd.Flags().GetString("path")
		if err != nil {
			return fmt.Errorf("failed to get path flag: %w", err)
		}

		websiteModel, err := add_website.New(fs, add_website.Args{
			WebsiteName:   websiteName,
			FrameworkName: frameworkName,
			WebsitePath:   websitePath,
			Force:         forceAddWebsite,
		})
		tui.CheckErr(err)

		if _, err := teax.NewProgram(websiteModel, tea.WithANSICompressor()).Run(); err != nil {
			return err
		}

		return nil
	},
	Args: cobra.MaximumNArgs(2),
}

// init registers the 'add' command with the root command
func init() {
	rootCmd.AddCommand(addCmd)

	// Add subcommands under 'add'
	addCmd.AddCommand(addWebsiteCmd)

	// Add flag for --path, the base url path for the website
	addWebsiteCmd.Flags().StringP("path", "p", "", "base url path for the website, e.g. /my-site")

	addWebsiteCmd.Flags().BoolVarP(&forceAddWebsite, "force", "f", false, "force website creation, even if conflicts exist.")
}
