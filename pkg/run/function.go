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
	"log"
	goruntime "runtime"
	"time"

	"github.com/docker/docker/api/types/container"

	"github.com/nitrictech/cli/pkg/containerengine"
	"github.com/nitrictech/cli/pkg/output"
	"github.com/nitrictech/cli/pkg/runtime"
	"github.com/nitrictech/cli/pkg/stack"
)

type Function struct {
	handler string
	runCtx  string
	rt      runtime.Runtime
	ce      containerengine.ContainerEngine
	// Container id populated after a call to Start
	cid string
}

func (f *Function) Name() string {
	return f.rt.ContainerName()
}

func (f *Function) Start() error {
	launchOpts, err := f.rt.LaunchOptsForFunction(f.runCtx)
	if err != nil {
		return err
	}

	hc := &container.HostConfig{
		AutoRemove: true,
		Mounts:     launchOpts.Mounts,
		LogConfig:  *f.ce.Logger(f.runCtx).Config(),
	}

	if goruntime.GOOS == "linux" {
		// setup host.docker.internal to route to host gateway
		// to access rpc server hosted by local CLI run
		hc.ExtraHosts = []string{"host.docker.internal:172.17.0.1"}
	}

	cc := &container.Config{
		Image: f.rt.DevImageName(), // Select an image to use based on the handler
		// Set the address to the bound port
		Env: []string{
			fmt.Sprintf("SERVICE_ADDRESS=host.docker.internal:%d", 50051),
			fmt.Sprintf("NITRIC_SERVICE_PORT=%d", 50051),
			fmt.Sprintf("NITRIC_SERVICE_HOST=%s", "host.docker.internal"),
		},
		Entrypoint: launchOpts.Entrypoint,
		Cmd:        launchOpts.Cmd,
		WorkingDir: launchOpts.TargetWD,
	}

	if output.VerboseLevel > 1 {
		log.Default().Print(containerengine.Cli(cc, hc))
	}

	cID, err := f.ce.ContainerCreate(cc, hc, nil, f.Name())
	if err != nil {
		return err
	}

	f.cid = cID

	return f.ce.Start(cID)
}

func (f *Function) Stop() error {
	timeout := time.Second * 5
	return f.ce.Stop(f.cid, &timeout)
}

type FunctionOpts struct {
	Handler         string
	RunCtx          string
	ContainerEngine containerengine.ContainerEngine
}

func newFunction(opts FunctionOpts) (*Function, error) {
	rt, err := runtime.NewRunTimeFromHandler(opts.Handler)
	if err != nil {
		return nil, err
	}

	return &Function{
		rt:      rt,
		handler: opts.Handler,
		runCtx:  opts.RunCtx,
		ce:      opts.ContainerEngine,
	}, nil
}

func FunctionsFromHandlers(s *stack.Stack) ([]*Function, error) {
	funcs := make([]*Function, 0, len(s.Functions))
	ce, err := containerengine.Discover()
	if err != nil {
		return nil, err
	}

	for _, f := range s.Functions {
		relativeHandlerPath, err := f.RelativeHandlerPath(s)
		if err != nil {
			return nil, err
		}

		if f, err := newFunction(FunctionOpts{
			RunCtx:          s.Dir,
			Handler:         relativeHandlerPath,
			ContainerEngine: ce,
		}); err != nil {
			return nil, err
		} else {
			funcs = append(funcs, f)
		}
	}

	return funcs, nil
}
