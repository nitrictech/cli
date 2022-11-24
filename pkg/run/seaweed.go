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
	"io"
	"os"

	"github.com/pkg/errors"

	"github.com/seaweedfs/seaweedfs/weed/command"
	flag "github.com/seaweedfs/seaweedfs/weed/util/fla9"

	"github.com/nitrictech/cli/pkg/utils"
)

type SeaweedServer struct {
	apiPort int // external API port from the seaweedfs s3 API gateway
}

const (
	bucketsDir = "./.nitric/buckets"
	logsDir    = "/.nitric/logs"
	runPerm    = os.ModePerm // NOTE: octal notation is important here!!!
)

// Start - Start the local SeaweedFS server
func (m *SeaweedServer) Start() error {
	err := os.MkdirAll(bucketsDir, runPerm)
	if err != nil {
		return errors.WithMessage(err, "os.MkdirAll")
	}

	ports, err := utils.Take(1)
	if err != nil {
		return errors.WithMessage(err, "freeport.Take")
	}

	port := uint16(ports[0])

	go func() {
		// FIXME: magic number 26 is the seaweedFS server command
		// We may want to fork this to publically expose these commands
		srvCmd := command.Commands[26]

		parentArgs := []string{
			"server",
			"-alsologtostderr=false",
			fmt.Sprintf("-logdir=%s", logsDir),
		}

		cmdArgs := []string{
			"server",
			fmt.Sprintf("-dir=%s", bucketsDir),
			"-s3",
			fmt.Sprintf("-s3.port=%d", port),
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

	m.apiPort = int(port)

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
	return &SeaweedServer{}, nil
}
