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
	"path/filepath"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/nitrictech/newcli/pkg/build"
	"github.com/nitrictech/newcli/pkg/run"
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

		err = build.CreateBaseDev(stackPath, map[string]string{
			"ts": "nitric-ts-dev",
		})
		cobra.CheckErr(err)

		ls := run.NewLocalServices(stackPath)
		memerr := make(chan error)
		go func(errch chan error) {
			errch <- ls.Start()
		}(memerr)

		for {
			if ls.Running() {
				break
			}
			time.Sleep(time.Second)
		}

		functions, err := run.FunctionsFromHandlers(stackPath, files)
		cobra.CheckErr(err)

		for _, f := range functions {
			err = f.Start()
			cobra.CheckErr(err)
		}

		time.Sleep(time.Second * 2)
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
