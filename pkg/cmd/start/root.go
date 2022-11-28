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

package start

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/nitrictech/cli/pkg/output"
	"github.com/nitrictech/cli/pkg/project"
	"github.com/nitrictech/cli/pkg/run"
	"github.com/nitrictech/cli/pkg/tasklet"
)

var startCmd = &cobra.Command{
	Use:         "start",
	Short:       "Run nitric services locally for development and testing",
	Long:        `Run nitric services locally for development and testing`,
	Example:     `nitric start`,
	Annotations: map[string]string{"commonCommand": "yes"},
	Run: func(cmd *cobra.Command, args []string) {
		term := make(chan os.Signal, 1)
		signal.Notify(term, syscall.SIGTERM, syscall.SIGINT)

		// Divert default log output to pterm debug
		log.SetOutput(output.NewPtermWriter(pterm.Debug))
		log.SetFlags(0)

		ls := run.NewLocalServices(&project.Project{
			Name: "local",
		})
		if ls.Running() {
			pterm.Error.Println("Only one instance of Nitric can be run locally at a time, please check that you have ended all other instances and try again")
			os.Exit(2)
		}

		memerr := make(chan error)
		pool := run.NewRunProcessPool()

		startLocalServices := tasklet.Runner{
			StartMsg: "Starting Local Services",
			Runner: func(progress output.Progress) error {
				go func(errch chan error) {
					errch <- ls.Start(pool)
				}(memerr)

				for {
					select {
					case err := <-memerr:
						// catch any early errors from Start()
						if err != nil {
							return err
						}
					default:
					}
					if ls.Running() {
						break
					}
					progress.Busyf("Waiting for Local Services to be ready")
					time.Sleep(time.Second)
				}
				return nil
			},
			StopMsg: "Started Local Services!",
		}
		tasklet.MustRun(startLocalServices, tasklet.Opts{
			Signal: term,
		})

		pterm.DefaultBasicText.Println("Local running, use ctrl-C to stop")

		stackState := run.NewStackState()

		area, _ := pterm.DefaultArea.Start()
		lck := sync.Mutex{}
		// React to worker pool state and update services table
		pool.Listen(func(we run.WorkerEvent) {
			lck.Lock()
			defer lck.Unlock()
			// area.Clear()

			err := ls.Refresh()
			if err != nil {
				cobra.CheckErr(err)
			}

			area.Update(err)
			stackState.Update(pool, ls)

			tables := []string{}
			table, rows := stackState.ApiTable(9001)
			if rows > 0 {
				tables = append(tables, table)
			}

			table, rows = stackState.TopicTable(9001)
			if rows > 0 {
				tables = append(tables, table)
			}

			table, rows = stackState.SchedulesTable(9001)
			if rows > 0 {
				tables = append(tables, table)
			}
			area.Update(strings.Join(tables, "\n\n"))
		})

		select {
		case membraneError := <-memerr:
			fmt.Println(errors.WithMessage(membraneError, "membrane error, exiting"))
		case <-term:
			fmt.Println("Shutting down services - exiting")
		}

		_ = area.Stop()
		// Stop the membrane
		cobra.CheckErr(ls.Stop())
	},
	Args: cobra.ExactArgs(0),
}

func RootCommand() *cobra.Command {
	return startCmd
}
