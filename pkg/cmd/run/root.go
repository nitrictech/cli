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
	"path"
	"path/filepath"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/nitrictech/newcli/pkg/build"
	"github.com/nitrictech/newcli/pkg/run"
	"github.com/nitrictech/newcli/pkg/tasklet"
)

var runCmd = &cobra.Command{
	Use:   "run [entrypointsGlob]",
	Short: "run a nitric stack",
	Long: `Run a nitric stack locally for
	development/testing
`,
	Run: func(cmd *cobra.Command, args []string) {
		term := make(chan os.Signal, 1)
		signal.Notify(term, os.Interrupt, syscall.SIGTERM)
		signal.Notify(term, os.Interrupt, syscall.SIGINT)

		stackPath, err := filepath.Abs(".")
		cobra.CheckErr(err)

		files, err := filepath.Glob(filepath.Join(stackPath, args[0]))
		cobra.CheckErr(err)
		if len(files) == 0 {
			err = errors.New("No files where found with glob, try a new pattern")
			cobra.CheckErr(err)
		}

		createBaseImage := tasklet.Runner{
			StartMsg: "Creating Dev Image",
			Runner: func(tCtx tasklet.TaskletContext) error {
				return build.CreateBaseDev(stackPath, map[string]string{
					"ts": "nitric-ts-dev",
				})
			},
			StopMsg: "Created Dev Image!",
		}
		tasklet.MustRun(createBaseImage, tasklet.Opts{Signal: term})

		ls := run.NewLocalServices(path.Base(stackPath), stackPath)

		memerr := make(chan error)

		startLocalServices := tasklet.Runner{
			StartMsg: "Starting Local Services",
			Runner: func(tCtx tasklet.TaskletContext) error {
				go func(errch chan error) {
					errch <- ls.Start()
				}(memerr)

				for {
					if ls.Running() {
						break
					}
					time.Sleep(time.Second)
				}
				return nil
			},
			StopMsg: "Started Local Services!",
		}
		tasklet.MustRun(startLocalServices, tasklet.Opts{Signal: term})

		var functions []*run.Function

		startFunctions := tasklet.Runner{
			StartMsg: "Starting Functions",
			Runner: func(tCtx tasklet.TaskletContext) error {
				functions, err = run.FunctionsFromHandlers(stackPath, files)
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

		fmt.Println("Local running, use ctrl-C to stop")

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

		// Stop the membrane
		cobra.CheckErr(ls.Stop())
	},
	Args: cobra.MaximumNArgs(1),
}

func RootCommand() *cobra.Command {
	return runCmd
}
