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
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/nitrictech/cli/pkgplus/cloud"
	"github.com/nitrictech/cli/pkgplus/project"
	"github.com/nitrictech/cli/pkgplus/view/tui"
	"github.com/nitrictech/cli/pkgplus/view/tui/commands/services"
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
		// log.SetOutput(output.NewPtermWriter(pterm.Debug))
		// log.SetFlags(0)
		fs := afero.NewOsFs()

		proj, err := project.FromFile(fs, "")
		tui.CheckErr(err)

		// Start the local cloud service analogues
		localCloud, err := cloud.New()
		tui.CheckErr(err)

		// Run the app code (project services)
		stopChan := make(chan bool)
		updatesChan := make(chan project.ServiceRunUpdate)
		go func() {
			err := proj.RunServicesWithCommand(localCloud, stopChan, updatesChan)
			if err != nil {
				panic(err)
			}
		}()

		tui.CheckErr(err)

		runView := tea.NewProgram(services.NewModel(stopChan, updatesChan, localCloud))

		_, _ = runView.Run()
		// cobra.CheckErr(err)
		localCloud.Stop()

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
