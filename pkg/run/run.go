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
	"net"
	"os"
	"time"

	"github.com/nitrictech/nitric/pkg/membrane"
	boltdb_service "github.com/nitrictech/nitric/pkg/plugins/document/boltdb"
	secret_service "github.com/nitrictech/nitric/pkg/plugins/secret/dev"
	minio "github.com/nitrictech/nitric/pkg/plugins/storage/minio"
	"github.com/nitrictech/nitric/pkg/worker"
)

type LocalServices interface {
	Start() error
	Stop() error
	Running() bool
}

type localServices struct {
	stackPath string
	mio       *MinioServer
	mem       *membrane.Membrane
}

func NewLocalServices(stackPath string) LocalServices {
	return &localServices{
		stackPath: stackPath,
	}
}

func (l *localServices) Stop() error {
	l.mem.Stop()
	return l.mio.Stop()
}

func (l *localServices) Running() bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("0.0.0.0", "50051"), time.Second)
	if err != nil {
		return false
	}
	if conn != nil {
		defer conn.Close()
		return true
	}
	return false
}

func (l *localServices) Start() error {
	var err error
	l.mio, err = NewMinio("./.nitric/run", "test-run")
	if err != nil {
		return err
	}

	// start minio
	err = l.mio.Start()
	if err != nil {
		return err
	}

	// Connect dev storage
	os.Setenv(minio.MINIO_ENDPOINT_ENV, fmt.Sprintf("localhost:%d", l.mio.GetApiPort()))
	os.Setenv(minio.MINIO_ACCESS_KEY_ENV, "minioadmin")
	os.Setenv(minio.MINIO_SECRET_KEY_ENV, "minioadmin")
	sp, err := minio.New()
	if err != nil {
		return err
	}

	// Connect dev documents
	os.Setenv("LOCAL_DB_DIR", "./.nitric/run")
	dp, err := boltdb_service.New()
	if err != nil {
		return err
	}

	// Connect secrets plugin
	os.Setenv("LOCAL_SEC_DIR", "./.nitric/run")
	secp, err := secret_service.New()
	if err != nil {
		return err
	}

	// Create a new Worker Pool
	// TODO: We may want to override GetWorker on the default ProcessPool
	// For now we'll use the default and expand from there
	pool := worker.NewProcessPool(&worker.ProcessPoolOptions{
		MinWorkers: 0,
		MaxWorkers: 100,
	})

	// Start a new gateway plugin
	gw, err := NewGateway()
	if err != nil {
		return err
	}

	// Prepare development membrane to start
	// This will start a single membrane that all
	// running functions will connect to
	l.mem, err = membrane.New(&membrane.MembraneOptions{
		ServiceAddress:          "0.0.0.0:50051",
		ChildCommand:            []string{"echo", "running membrane 🚀"},
		SecretPlugin:            secp,
		StoragePlugin:           sp,
		DocumentPlugin:          dp,
		GatewayPlugin:           gw,
		Pool:                    pool,
		TolerateMissingServices: true,
	})
	if err != nil {
		return err
	}

	return l.mem.Start()
}
