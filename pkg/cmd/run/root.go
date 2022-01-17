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
	"github.com/nitrictech/newcli/pkg/provider/run"
	"github.com/nitrictech/nitric/pkg/membrane"
	boltdb_service "github.com/nitrictech/nitric/pkg/plugins/document/boltdb"
	minio "github.com/nitrictech/nitric/pkg/plugins/storage/minio"
	"github.com/nitrictech/nitric/pkg/worker"
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

		ctx, err := filepath.Abs(".")
		cobra.CheckErr(err)

		files, err := filepath.Glob(filepath.Join(ctx, args[0]))
		cobra.CheckErr(err)

		build.CreateBaseDev(ctx, map[string]string{
			"ts": "nitric-ts-dev",
		})
		cobra.CheckErr(err)

		mio, err := run.NewMinio("./.nitric/run", "test-run")
		cobra.CheckErr(err)

		// start minio
		mio.Start()
		cobra.CheckErr(err)

		// Connect dev storage
		os.Setenv(minio.MINIO_ENDPOINT_ENV, fmt.Sprintf("localhost:%d", mio.GetApiPort()))
		os.Setenv(minio.MINIO_ACCESS_KEY_ENV, "minioadmin")
		os.Setenv(minio.MINIO_SECRET_KEY_ENV, "minioadmin")
		sp, err := minio.New()
		cobra.CheckErr(err)

		// Connect dev documents
		dp, err := boltdb_service.New()
		cobra.CheckErr(err)

		// Create a new Worker Pool
		// TODO: We may want to override GetWorker on the default ProcessPool
		// For now we'll use the default and expand from there
		pool := worker.NewProcessPool(&worker.ProcessPoolOptions{
			MinWorkers: 0,
			MaxWorkers: 100,
		})

		// Start a new gateway plugin
		gw, err := run.NewGateway()
		cobra.CheckErr(err)

		// Prepare development membrane to start
		// This will start a single membrane that all
		// running functions will connect to
		mem, err := membrane.New(&membrane.MembraneOptions{
			ServiceAddress:          "0.0.0.0:50051",
			ChildCommand:            []string{"echo", "running membrane 🚀"},
			StoragePlugin:           sp,
			DocumentPlugin:          dp,
			GatewayPlugin:           gw,
			Pool:                    pool,
			TolerateMissingServices: true,
		})
		cobra.CheckErr(err)

		memerr := make(chan error)
		go func(errch chan error) {
			errch <- mem.Start()
		}(memerr)

		time.Sleep(time.Second * time.Duration(2))

		functions, err := run.FunctionsFromHandlers(ctx, files)
		cobra.CheckErr(err)

		for _, f := range functions {
			err = f.Start()
			cobra.CheckErr(err)
		}

		fmt.Println("Local running, use ctrl-C to stop")

		select {
		case membraneError := <-memerr:
			fmt.Println(errors.WithMessage(membraneError, "membrane error, exiting"))
		case sigTerm := <-term:
			fmt.Printf("Received %v, exiting\n", sigTerm)
		}

		for _, f := range functions {
			f.Stop()
		}

		// Stop the membrane
		mem.Stop()
		// Stop the minio server
		mio.Stop()
	},
	Args: cobra.MaximumNArgs(1),
}

func RootCommand() *cobra.Command {
	return runCmd
}
