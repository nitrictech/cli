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
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/nitrictech/cli/pkg/output"
	"github.com/nitrictech/cli/pkg/utils"
	"github.com/nitrictech/cli/pkgplus/cloud"
	"github.com/nitrictech/cli/pkgplus/dashboard"
	"github.com/nitrictech/cli/pkgplus/view/tui"
	"github.com/nitrictech/cli/pkgplus/view/tui/commands/local"
)

var startNoBrowser bool

var startCmd = &cobra.Command{
	Use:         "start",
	Short:       "Run nitric services locally for development and testing",
	Long:        `Run nitric services locally for development and testing`,
	Example:     `nitric start`,
	Annotations: map[string]string{"commonCommand": "yes"},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Divert default log output to pterm debug
		log.SetOutput(output.NewPtermWriter(pterm.Debug))
		log.SetFlags(0)

		if !utils.IsTerminal() && !output.CI {
			fmt.Println("")
			pterm.Warning.Println("non-interactive environment detected, switching to non-interactive mode.")
			output.CI = true
		}

		// if output.CI {
		// 	return start.RunNonInteractive(startNoBrowser)
		// }

		localCloud, err := cloud.New()
		tui.CheckErr(err)

		// create dashboard
		dash, err := dashboard.New(startNoBrowser, localCloud)
		cobra.CheckErr(err)

		// fs := afero.NewOsFs()

		// _, err := project.FromFile(fs, "")
		// tui.CheckErr(err)

		// // create dashboard, we will start it once an application is connected
		// dash, err = dashboard.New(startNoBrowser, localCloud.Storage)
		// tui.CheckErr(err)

		// Start dashboard
		err = dash.Start()
		cobra.CheckErr(err)

		// Start a new tea app
		go func() {
			cliView := tea.NewProgram(local.NewTuiModel(localCloud, dash.GetDashboardUrl()))

			_, _ = cliView.Run()
			localCloud.Stop()
		}()

		err = localCloud.Start()
		tui.CheckErr(err)

		return nil
	},
	Args: cobra.ExactArgs(0),
}

func init() {
	startCmd.PersistentFlags().BoolVar(
		&startNoBrowser,
		"no-browser",
		false,
		"disable browser opening for local dashboard, note: in CI mode the browser opening feature is disabled",
	)

	rootCmd.AddCommand(startCmd)
}
