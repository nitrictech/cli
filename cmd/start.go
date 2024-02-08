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
	"os"
	"os/signal"
	"syscall"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/nitrictech/cli/pkgplus/cloud"
	"github.com/nitrictech/cli/pkgplus/dashboard"
	"github.com/nitrictech/cli/pkgplus/project"
	"github.com/nitrictech/cli/pkgplus/view/tui"
	"github.com/nitrictech/cli/pkgplus/view/tui/commands/services"
	"github.com/nitrictech/cli/pkgplus/view/tui/fragments"
	"github.com/nitrictech/cli/pkgplus/view/tui/teax"
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

		fmt.Print(fragments.NitricTag())
		fmt.Println(" start")
		fmt.Println()

		// Start the local cloud service analogues
		localCloud, err := cloud.New()
		tui.CheckErr(err)

		fmt.Println("local nitric server started")

		// Start dashboard
		dash, err := dashboard.New(startNoBrowser, localCloud, proj)
		tui.CheckErr(err)

		err = dash.Start()
		tui.CheckErr(err)

		bold := lipgloss.NewStyle().Bold(true).Foreground(tui.Colors.Purple)
		numServices := fmt.Sprintf("%d", len(proj.GetServices()))

		fmt.Print("found ")
		fmt.Print(bold.Render(numServices))
		fmt.Print(" services in project\n")

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

		// non-interactive environment
		if CI || !tui.IsTerminal() {
			go func() {
				sigChan := make(chan os.Signal, 1)
				signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

				// Wait for a signal
				<-sigChan

				// Send stop signal to stopChan
				close(stopChan)

				localCloud.Stop()
			}()

			for {
				select {
				case update := <-updatesChan:
					fmt.Printf("%s [%s]: %s", update.ServiceName, update.Status, update.Message)
				case <-stopChan:
					fmt.Println("Shutting down services - exiting")
				}
			}
		} else {
			// interactive environment
			runView := teax.NewProgram(services.NewModel(stopChan, updatesChan, localCloud, dash.GetDashboardUrl()))

			_, err = runView.Run()
			tui.CheckErr(err)

			localCloud.Stop()
		}

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
