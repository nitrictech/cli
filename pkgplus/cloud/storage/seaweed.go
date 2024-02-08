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

package storage

import (
	"fmt"
	"io"
	"os"

	"github.com/pkg/errors"

	"github.com/seaweedfs/seaweedfs/weed/command"
	flag "github.com/seaweedfs/seaweedfs/weed/util/fla9"

	"github.com/nitrictech/cli/pkgplus/cloud/env"
	"github.com/nitrictech/cli/pkgplus/netx"
)

type SeaweedServer struct {
	apiPort    int // external API port from the seaweedfs s3 API gateway
	bucketsDir string
	logsDir    string
}

const (
	runPerm = os.ModePerm // NOTE: octal notation is important here!!!
)

// Start - Start the local SeaweedFS server
func (m *SeaweedServer) Start() error {
	ports, err := netx.TakePort(6)
	if err != nil {
		return errors.WithMessage(err, "freeport.Take")
	}

	// unique ports needed for all elements of seaweedfs to allow multiple instances to run in parallel
	masterPort := uint16(ports[0])
	volumePort := uint16(ports[1])
	s3Port := uint16(ports[2])
	masterGrpcPort := uint16(ports[3])
	volumeGrpcPort := uint16(ports[4])
	filerPort := uint16(ports[5])

	go func() {
		// FIXME: magic number 26 is the seaweedFS server command
		// We may want to fork this to publicly expose these commands
		srvCmd := command.Commands[26]

		parentArgs := []string{
			"server",
			"-alsologtostderr=false",
			fmt.Sprintf("-logdir=%s", m.logsDir),
		}

		cmdArgs := []string{
			"server",
			fmt.Sprintf("-dir=%s", m.bucketsDir),
			fmt.Sprintf("-master.port=%d", masterPort),
			fmt.Sprintf("-master.port.grpc=%d", masterGrpcPort),
			"-s3",
			fmt.Sprintf("-s3.port=%d", s3Port),
			"-volume",
			"-volume.max=300",
			fmt.Sprintf("-volume.port=%d", volumePort),
			fmt.Sprintf("-volume.port.grpc=%d", volumeGrpcPort),
			fmt.Sprintf("-filer.port=%d", filerPort),
		}

		origOsArgs := os.Args
		os.Args = parentArgs

		flag.Parse()

		// restore original os.args
		os.Args = origOsArgs

		// run the seaweedfs server command
		_ = srvCmd.Flag.Parse(cmdArgs[1:])
		srvCmd.Flag.SetOutput(io.Discard)
		otherArgs := srvCmd.Flag.Args()

		_ = srvCmd.Run(srvCmd, otherArgs)
	}()

	m.apiPort = int(s3Port)

	return nil
}

func (m *SeaweedServer) GetApiPort() int {
	return m.apiPort
}

func (m *SeaweedServer) Stop() error {
	// TODO: Implement explicit stop
	// currently this is not required as the implementations is embedded and will
	// respect process signals
	return nil
}

func NewSeaweed() (*SeaweedServer, error) {
	bDir := env.LOCAL_BUCKETS_DIR.String()

	if err := os.MkdirAll(bDir, runPerm); err != nil {
		return nil, errors.WithMessage(err, "os.MkdirAll")
	}

	lDir := env.LOCAL_SEAWEED_LOGS_DIR.String()

	if err := os.MkdirAll(lDir, runPerm); err != nil {
		return nil, errors.WithMessage(err, "os.MkdirAll")
	}

	return &SeaweedServer{
		logsDir:    lDir,
		bucketsDir: bDir,
	}, nil
}
