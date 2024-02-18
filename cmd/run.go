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

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/nitrictech/cli/pkg/cloud"
	docker "github.com/nitrictech/cli/pkg/docker"
	"github.com/nitrictech/cli/pkg/project"
	"github.com/nitrictech/cli/pkg/view/tui"
	"github.com/nitrictech/cli/pkg/view/tui/commands/build"
	"github.com/nitrictech/cli/pkg/view/tui/commands/services"
	"github.com/nitrictech/cli/pkg/view/tui/teax"
)

var runNoBrowser bool

var runCmd = &cobra.Command{
	Use:         "run",
	Short:       "Run your project locally for development and testing",
	Long:        `Run your project locally for development and testing`,
	Example:     `nitric run`,
	Annotations: map[string]string{"commonCommand": "yes"},
	RunE: func(cmd *cobra.Command, args []string) error {
		err := docker.VerifyDockerIsAvailable()
		tui.CheckErr(err)

		fs := afero.NewOsFs()

		proj, err := project.FromFile(fs, "")
		tui.CheckErr(err)

		// Start the local cloud service analogues
		localCloud, err := cloud.New()
		tui.CheckErr(err)

		updates, err := proj.BuildServices(fs)
		tui.CheckErr(err)

		prog := teax.NewProgram(build.NewModel(updates))
		// blocks but quits once the above updates channel is closed by the build process
		_, err = prog.Run()
		tui.CheckErr(err)

		// Run the app code (project services)
		stopChan := make(chan bool)
		updatesChan := make(chan project.ServiceRunUpdate)
		go func() {
			err := proj.RunServices(localCloud, stopChan, updatesChan)
			if err != nil {
				panic(err)
			}
		}()

		tui.CheckErr(err)

		// non-interactive environment
		if isNonInteractive() {
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
			runView := teax.NewProgram(services.NewModel(stopChan, updatesChan, localCloud, ""))

			_, _ = runView.Run()

			localCloud.Stop()
		}

		return nil
	},
	Args: cobra.ExactArgs(0),
}

func init() {
	runCmd.Flags().StringVarP(&envFile, "env-file", "e", "", "--env-file config/.my-env")
	runCmd.PersistentFlags().BoolVar(
		&runNoBrowser,
		"no-browser",
		false,
		"disable browser opening for local dashboard, note: in CI mode the browser opening feature is disabled",
	)
	rootCmd.AddCommand(tui.AddDependencyCheck(runCmd, tui.Docker))
}
