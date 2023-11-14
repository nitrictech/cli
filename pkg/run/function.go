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
	goruntime "runtime"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"

	"github.com/nitrictech/cli/pkg/containerengine"
	"github.com/nitrictech/cli/pkg/project"
	"github.com/nitrictech/cli/pkg/runtime"
	"github.com/nitrictech/cli/pkg/utils"
)

type Function struct {
	projectName string
	handler     string
	name        string
	runCtx      string
	rt          runtime.Runtime
	ce          containerengine.ContainerEngine
	// Container id populated after a call to Start
	cid string
}

func (f *Function) Name() string {
	return f.name
}

func (f *Function) Start(envMap map[string]string) error {
	hc := &container.HostConfig{
		AutoRemove: true,
		LogConfig:  *f.ce.Logger(f.runCtx).Config(),
	}

	if goruntime.GOOS == "linux" {
		// setup host.docker.internal to route to host gateway
		// to access rpc server hosted by local CLI run
		hc.ExtraHosts = []string{"host.docker.internal:172.17.0.1"}
	}

	env := []string{
		"NITRIC_ENVIRONMENT=run",
		fmt.Sprintf("SERVICE_ADDRESS=host.docker.internal:%d", 50051),
		fmt.Sprintf("NITRIC_SERVICE_PORT=%d", 50051),
		fmt.Sprintf("NITRIC_SERVICE_HOST=%s", "host.docker.internal"),
	}

	for k, v := range envMap {
		env = append(env, k+"="+v)
	}

	cc := &container.Config{
		Image: fmt.Sprintf("%s-%s", f.projectName, f.Name()), // Select an image to use based on the handler
		Env:   env,
	}

	// Set additional configuration for http proxy
	hostProxyPort := envMap["NITRIC_HTTP_PROXY_PORT"]
	if hostProxyPort == "" {
		randomPort, err := utils.Take(1)
		if err != nil {
			return err
		}

		cc.Env = append(cc.Env, fmt.Sprintf("NITRIC_HTTP_PROXY_PORT=%d", randomPort[0]))
		hostProxyPort = fmt.Sprint(randomPort[0])
	}

	hc.PortBindings = nat.PortMap{
		nat.Port(hostProxyPort): []nat.PortBinding{
			{
				HostPort: hostProxyPort,
			},
		},
	}

	cc.ExposedPorts = nat.PortSet{
		nat.Port(hostProxyPort): struct{}{},
	}

	cID, err := f.ce.ContainerCreate(cc, hc, nil, strings.Join([]string{f.projectName, "run", f.Name()}, "-"))
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
	Name            string
	ProjectName     string
	Handler         string
	RunCtx          string
	Runtime         runtime.Runtime
	ContainerEngine containerengine.ContainerEngine
}

func newFunction(opts FunctionOpts) (*Function, error) {
	return &Function{
		name:        opts.Name,
		rt:          opts.Runtime,
		projectName: opts.ProjectName,
		handler:     opts.Handler,
		runCtx:      opts.RunCtx,
		ce:          opts.ContainerEngine,
	}, nil
}

func FunctionsFromHandlers(p *project.Project) ([]*Function, error) {
	funcs := make([]*Function, 0, len(p.Functions))

	if len(p.Functions) == 0 {
		return funcs, nil
	}

	ce, err := containerengine.Discover()
	if err != nil {
		return nil, err
	}

	for _, f := range p.Functions {
		relativeHandlerPath, err := f.RelativeHandlerPath(p)
		if err != nil {
			return nil, err
		}

		runtime, err := f.GetRuntime()
		if err != nil {
			return nil, err
		}

		if f, err := newFunction(FunctionOpts{
			Name:            f.Name,
			Runtime:         runtime,
			RunCtx:          p.Dir,
			Handler:         relativeHandlerPath,
			ContainerEngine: ce,
			ProjectName:     p.Name,
		}); err != nil {
			return nil, err
		} else {
			funcs = append(funcs, f)
		}
	}

	return funcs, nil
}
