package codeconfig

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"

	"google.golang.org/grpc"

	v1 "github.com/nitrictech/apis/go/nitric/v1"
	"github.com/nitrictech/newcli/pkg/utils"
)

// Collect - Collects information about a function for a nitric stack
// ctx - base context for the application
// handler - the specific handler for the application
// stack - the stack to add the information to
func Collect(ctx string, handler string, stack *Stack) error {
	// 0 - create a new function
	fun := NewFunction()
	// 1 - start the server on a free port
	srv := New(fun)
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
	var cmd *exec.Cmd = nil

	var localIp = utils.GetLocalIP()

	if strings.HasSuffix(handler, ".ts") {
		cmd = exec.Command(
			"docker",
			"run",
			"--rm",
			// setup host.docker.internal to route to host gateway
			// to access rpc server hosted by local CLI run
			"--add-host", "host.docker.internal:"+localIp,
			// Set the address to the bound port
			"-e", "SERVICE_ADDRESS=host.docker.internal:50051",
			// Set the volume mount to bound context
			"-v", fmt.Sprintf("%s:/app/", ctx),
			"nitric-ts-dev", "-T", handler,
		)
	} else {
		return fmt.Errorf("unsupported artifact")
	}

	err = cmd.Run()

	if err != nil {
		return err
	}

	// 3 - When the container exits stop the server
	grpcSrv.Stop()
	err = <-errChan

	// 4 - Add the function to the stack
	stack.AddFunction(fun)

	return err
}
