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

package run

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/nitrictech/cli/pkg/build"
	"github.com/nitrictech/cli/pkg/codeconfig"
	"github.com/nitrictech/cli/pkg/containerengine"
	"github.com/nitrictech/cli/pkg/output"
	"github.com/nitrictech/cli/pkg/run"
	"github.com/nitrictech/cli/pkg/stack"
	"github.com/nitrictech/cli/pkg/tasklet"
)

var runCmd = &cobra.Command{
	Use:   "run [handlerGlob]",
	Short: "run a nitric stack",
	Long: `Run a nitric stack locally for development or testing
`,
	Example: `# use a nitric.yaml or configured default handlerGlob (stack in the current directory).
nitric run

# use an explicit handlerGlob (stack in the current directory)
nitric run "functions/*.ts"

# use an explicit handlerGlob and explicit stack directory
nitric run -s ../projectX/ "functions/*.ts"`,
	Run: func(cmd *cobra.Command, args []string) {
		term := make(chan os.Signal, 1)
		signal.Notify(term, os.Interrupt, syscall.SIGTERM)
		signal.Notify(term, os.Interrupt, syscall.SIGINT)

		s, err := stack.FromOptions(args)
		cobra.CheckErr(err)
		if !s.Loaded {
			codeAsConfig := tasklet.Runner{
				StartMsg: "Gathering configuration from code..",
				Runner: func(_ output.Progress) error {
					s, err = codeconfig.Populate(s)
					return err
				},
				StopMsg: "Configuration gathered",
			}
			tasklet.MustRun(codeAsConfig, tasklet.Opts{LogToPterm: true})
		}

		ce, err := containerengine.Discover()
		cobra.CheckErr(err)

		logger := ce.Logger(s.Dir)
		cobra.CheckErr(logger.Start())

		createBaseImage := tasklet.Runner{
			StartMsg: "Creating Dev Image",
			Runner: func(_ output.Progress) error {
				return build.CreateBaseDev(s)
			},
			StopMsg: "Created Dev Image!",
		}
		tasklet.MustRun(createBaseImage, tasklet.Opts{Signal: term})

		ls := run.NewLocalServices(s.Name, s.Dir)
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
			Signal:     term,
			LogToPterm: true,
		})

		var functions []*run.Function

		startFunctions := tasklet.Runner{
			StartMsg: "Starting Functions",
			Runner: func(_ output.Progress) error {
				functions, err = run.FunctionsFromHandlers(s)
				if err != nil {
					return err
				}
				for _, f := range functions {
					err = f.Start()
					if err != nil {
						return err
					}
				}
				return nil
			},
			StopMsg: "Started Functions!",
		}
		tasklet.MustRun(startFunctions, tasklet.Opts{Signal: term})

		pterm.DefaultBasicText.Println("Local running, use ctrl-C to stop")

		stackState := run.NewStackState()

		area, _ := pterm.DefaultArea.Start()
		lck := sync.Mutex{}
		// React to worker pool state and update services table
		pool.Listen(func(we run.WorkerEvent) {
			lck.Lock()
			defer lck.Unlock()
			stackState.UpdateFromWorkerEvent(we)
			apiTable := stackState.ApiTable(9001)
			topicsTable := stackState.TopicTable(9001)
			schedTable := stackState.Schedules(9001)
			area.Update(apiTable, "\n\n", topicsTable, "\n\n", schedTable)
		})

		// TODO: revisit nitric.yaml support for this output
		// if len(apis) == 0 {
		// 	// if we have a nitric.yaml then ApiDocs will be empty
		// 	for a := range s.Apis {
		// 		apis[a] = fmt.Sprintf("http://127.0.0.1:9001/apis/%s", a)
		// 	}
		// }
		// output.Print(apis)

		select {
		case membraneError := <-memerr:
			fmt.Println(errors.WithMessage(membraneError, "membrane error, exiting"))
		case sigTerm := <-term:
			fmt.Printf("Received %v, exiting\n", sigTerm)
		}

		for _, f := range functions {
			if err = f.Stop(); err != nil {
				fmt.Println(f.Name(), " stop error ", err)
			}
		}

		_ = area.Stop()
		_ = logger.Stop()
		// Stop the membrane
		cobra.CheckErr(ls.Stop())
	},
	Args: cobra.MinimumNArgs(0),
}

func RootCommand() *cobra.Command {
	stack.AddOptions(runCmd)
	return runCmd
}
