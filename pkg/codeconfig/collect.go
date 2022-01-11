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

package codeconfig

import (
	"fmt"
	"net"
	"path"
	"runtime"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/strslice"
	"google.golang.org/grpc"

	v1 "github.com/nitrictech/apis/go/nitric/v1"
	"github.com/nitrictech/newcli/pkg/containerengine"
	"github.com/nitrictech/newcli/pkg/utils"
)

func ImageNameFromExt(ext string) string {
	return "nitric-" + strings.Replace(ext, ".", "", 1) + "-dev"
}

func containerNameFromHandler(handler string) string {
	return strings.Replace(path.Base(handler), path.Ext(handler), "", 1)
}

// Collect - Collects information about a function for a nitric stack
// ctx - base context for the application
// handler - the specific handler for the application
// stack - the stack to add the information to
func Collect(ctx string, handler string, stack *Stack) error {
	// 0 - create a new function
	fun := NewFunction()
	// 1 - start the server on a free port
	srv := NewServer(fun)
	grpcSrv := grpc.NewServer()

	v1.RegisterResourceServiceServer(grpcSrv, srv)
	v1.RegisterFaasServiceServer(grpcSrv, srv)

	// 1a - run server non-blocking
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		return err
	}

	defer lis.Close()

	errChan := make(chan error)
	go func(errChan chan error) {
		errChan <- grpcSrv.Serve(lis)
	}(errChan)

	// 2 - run the handler in a container
	// 2a - Specify the service bind as the port with the docker gateway IP (running in bride mode)
	// Select an image to use based on the handler
	ce, err := containerengine.Discover()
	if err != nil {
		return err
	}

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
		Image: ImageNameFromExt(path.Ext(handler)),
		// Set the address to the bound port
		Env: []string{"SERVICE_ADDRESS=host.docker.internal:50051"},
		Cmd: strslice.StrSlice{"-T", handler},
	}, hostConfig, nil, containerNameFromHandler(handler))
	if err != nil {
		return err
	}

	err = ce.Start(cID)
	if err != nil {
		return err
	}

	waitChan, cErrChan := ce.ContainerWait(cID, container.WaitConditionNextExit)
	select {
	case done := <-waitChan:
		msg := ""
		if done.Error != nil {
			msg = done.Error.Message
		}
		if msg != "" || done.StatusCode != 0 {
			err = utils.WrapError(err, fmt.Errorf("error executing container (code %d) %s", done.StatusCode, msg))
		}
	case cErr := <-cErrChan:
		err = utils.WrapError(err, cErr)
	}

	// 3 - When the container exits stop the server
	grpcSrv.Stop()
	err = utils.WrapError(err, <-errChan)

	// 4 - Add the function to the stack
	stack.AddFunction(fun)

	return err
}
