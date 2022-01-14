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
	"runtime"
	"syscall"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/strslice"
	"github.com/nitrictech/newcli/pkg/build"
	"github.com/nitrictech/newcli/pkg/containerengine"
	"github.com/nitrictech/newcli/pkg/provider/run"
	"github.com/nitrictech/nitric/pkg/membrane"
	boltdb_service "github.com/nitrictech/nitric/pkg/plugins/document/boltdb"
	gateway_plugin "github.com/nitrictech/nitric/pkg/plugins/gateway/dev"
	minio "github.com/nitrictech/nitric/pkg/plugins/storage/minio"
	"github.com/nitrictech/nitric/pkg/worker"
	"github.com/spf13/cobra"
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

		files, err := filepath.Glob(args[0])

		if err != nil {
			cobra.CheckErr(err)
		}

		ce, err := containerengine.Discover()

		// Prepare development images
		ctx, _ := filepath.Abs(".")
		if err := build.CreateBaseDev(ctx, map[string]string{
			"ts": "nitric-ts-dev",
		}); err != nil {
			cobra.CheckErr(err)
		}

		mio, err := run.NewMinio("./.nitric/run", "test-run")

		if err != nil {
			cobra.CheckErr(err)
		}

		// start minio
		if err := mio.Start(); err != nil {
			cobra.CheckErr(err)
		}

		if err := mio.Stop(); err != nil {
			cobra.CheckErr(err)
		}
		// Connect dev storage
		os.Setenv(minio.MINIO_ENDPOINT_ENV, fmt.Sprintf("localhost:%d", mio.GetApiPort()))
		os.Setenv(minio.MINIO_ACCESS_KEY_ENV, "minioadmin")
		os.Setenv(minio.MINIO_SECRET_KEY_ENV, "minioadmin")
		sp, err := minio.New()
		if err != nil {
			cobra.CheckErr(err)
		}

		// Connect dev documents
		dp, err := boltdb_service.New()
		if err != nil {
			cobra.CheckErr(err)
		}

		// Create a new Worker Pool
		// TODO: We may want to override GetWorker on the default ProcessPool
		// For now we'll use the default and expand from there
		pool := worker.NewProcessPool(&worker.ProcessPoolOptions{
			MinWorkers: 0,
			MaxWorkers: 100,
		})

		// Start a new gateway plugin
		gw, err := gateway_plugin.New()

		if err != nil {
			cobra.CheckErr(err)
		}

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

		if err != nil {
			cobra.CheckErr(err)
		}

		memerr := make(chan error)
		go func(errch chan error) {
			errch <- mem.Start()
		}(memerr)

		time.Sleep(time.Second * time.Duration(2))

		// Start the functions
		// cids := make([]string, 0)
		fmt.Println("found files:", files)
		for i, f := range files {
			hostConfig := &container.HostConfig{
				AutoRemove: true,
				Mounts: []mount.Mount{
					{
						Type:   "bind",
						Source: ctx,
						Target: "/app",
					},
				},
			}
			if runtime.GOOS == "linux" {
				// setup host.docker.internal to route to host gateway
				// to access rpc server hosted by local CLI run
				hostConfig.ExtraHosts = []string{"host.docker.internal:172.17.0.1"}
			}

			cID, err := ce.ContainerCreate(&container.Config{
				Image: "nitric-ts-dev", // Select an image to use based on the handler
				// Set the address to the bound port
				Env:        []string{fmt.Sprintf("SERVICE_ADDRESS=host.docker.internal:%d", 50051)},
				Entrypoint: strslice.StrSlice{"nodemon"},
				Cmd:        strslice.StrSlice{"--watch", "/app/**", "--ext", "ts,json", "--exec", "ts-node -T " + "/app/" + f},
			}, hostConfig, nil, fmt.Sprintf("test-container-%d", i))

			if err != nil {
				cobra.CheckErr(err)
			}

			// cids = append(cids, cID)

			err = ce.Start(cID)
		}

		fmt.Println("Local running, use ctrl-C to stop")

		select {
		case membraneError := <-memerr:
			fmt.Println(fmt.Sprintf("Membrane Error: %v, exiting", membraneError))
		case sigTerm := <-term:
			fmt.Println(fmt.Sprintf("Received %v, exiting", sigTerm))
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
