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
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/strslice"

	"github.com/nitrictech/newcli/pkg/containerengine"
	"github.com/nitrictech/newcli/pkg/utils"
)

type Function struct {
	handler string
	runCtx  string
	runtime utils.Runtime
	ce      containerengine.ContainerEngine
	// Container id populated after a call to Start
	cid string
}

type LaunchOpts struct {
	Entrypoint []string
	Cmd        []string
}

func launchOptsForFunction(f *Function) (LaunchOpts, error) {
	opts := LaunchOpts{Entrypoint: strslice.StrSlice{"nodemon"}}
	rt, err := utils.NewRunTimeFromFilename(f.handler)
	if err != nil {
		return opts, err
	}
	switch rt {
	case utils.RuntimeJavascript, utils.RuntimeTypescript:
		opts.Cmd = strslice.StrSlice{"--watch", "/app/**", "--ext", "ts,js,json", "--exec", "ts-node -T " + "/app/" + f.handler}
	default:
		return opts, errors.New("could not get launchOpts from " + f.handler + ", runtime not supported")
	}
	return opts, nil
}

func (f *Function) Name() string {
	return strings.Replace(filepath.Base(f.handler), filepath.Ext(f.handler), "", 1)
}

func (f *Function) Start() error {
	hostConfig := &container.HostConfig{
		AutoRemove: true,
		Mounts: []mount.Mount{
			{
				Type:   "bind",
				Source: f.runCtx,
				Target: "/app",
			},
		},
		LogConfig: *f.ce.Logger(f.runCtx).Config(),
	}
	if runtime.GOOS == "linux" {
		// setup host.docker.internal to route to host gateway
		// to access rpc server hosted by local CLI run
		hostConfig.ExtraHosts = []string{"host.docker.internal:172.17.0.1"}
	}

	launchOpts, err := launchOptsForFunction(f)
	if err != nil {
		return err
	}

	cID, err := f.ce.ContainerCreate(&container.Config{
		Image: f.runtime.DevImageName(), // Select an image to use based on the handler
		// Set the address to the bound port
		Env:        []string{fmt.Sprintf("SERVICE_ADDRESS=host.docker.internal:%d", 50051)},
		Entrypoint: launchOpts.Entrypoint,
		Cmd:        launchOpts.Cmd,
	}, hostConfig, nil, f.Name())
	if err != nil {
		return err
	}

	f.cid = cID

	return f.ce.Start(cID)
}

func (f *Function) Stop() error {
	return f.ce.Stop(f.cid, nil)
}

type FunctionOpts struct {
	Handler         string
	RunCtx          string
	ContainerEngine containerengine.ContainerEngine
}

func newFunction(opts FunctionOpts) (*Function, error) {
	runtime, err := utils.NewRunTimeFromFilename(opts.Handler)
	if err != nil {
		return nil, err
	}

	return &Function{
		runtime: runtime,
		handler: opts.Handler,
		runCtx:  opts.RunCtx,
		ce:      opts.ContainerEngine,
	}, nil
}

func FunctionsFromHandlers(runCtx string, handlers []string) ([]*Function, error) {
	funcs := make([]*Function, 0, len(handlers))
	ce, err := containerengine.Discover()
	if err != nil {
		return nil, err
	}

	for _, h := range handlers {
		relativeHandlerPath, _ := filepath.Rel(runCtx, h)

		if f, err := newFunction(FunctionOpts{
			RunCtx:          runCtx,
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
