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
	"path"
	"time"

	"github.com/nitrictech/newcli/pkg/utils"
	"github.com/nitrictech/nitric/pkg/membrane"
	boltdb_service "github.com/nitrictech/nitric/pkg/plugins/document/boltdb"
	queue_service "github.com/nitrictech/nitric/pkg/plugins/queue/dev"
	secret_service "github.com/nitrictech/nitric/pkg/plugins/secret/dev"
	minio "github.com/nitrictech/nitric/pkg/plugins/storage/minio"
	nitric_utils "github.com/nitrictech/nitric/pkg/utils"
	"github.com/nitrictech/nitric/pkg/worker"
)

type LocalServices interface {
	Start() error
	Stop() error
	Running() bool
	Status() *LocalServicesStatus
}

type LocalServicesStatus struct {
	Running          bool   `yaml:"running"`
	RunDir           string `yaml:"runDir"`
	GatewayAddress   string `yaml:"gatewayAddress"`
	MembraneAddress  string `yaml:"membraneAddress"`
	MinioContainerID string `yaml:"minioContainerID"`
	MinioEndpoint    string `yaml:"minioEndpoint"`
}

type localServices struct {
	stackName string
	stackPath string
	mio       *MinioServer
	mem       *membrane.Membrane
	status    *LocalServicesStatus
}

func NewLocalServices(stackName, stackPath string) LocalServices {
	return &localServices{
		stackName: stackName,
		stackPath: stackPath,
		status: &LocalServicesStatus{
			Running:         false,
			RunDir:          path.Join(utils.NitricRunDir(), stackName),
			GatewayAddress:  nitric_utils.GetEnv("GATEWAY_ADDRESS", ":9001"),
			MembraneAddress: net.JoinHostPort("localhost", "50051"),
		},
	}
}

func (l *localServices) Stop() error {
	l.mem.Stop()
	return l.mio.Stop()
}

func (l *localServices) Running() bool {
	l.status.Running = false
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("0.0.0.0", "50051"), time.Second)
	if err == nil && conn != nil {
		defer conn.Close()
		l.status.Running = true
	}
	return l.status.Running
}

func (l *localServices) Status() *LocalServicesStatus {
	return l.status
}

func (l *localServices) Start() error {
	var err error
	l.mio, err = NewMinio(l.status.RunDir, "minio")
	if err != nil {
		return err
	}

	// start minio
	err = l.mio.Start()
	if err != nil {
		return err
	}
	l.status.MinioContainerID = l.mio.cid
	l.status.MinioEndpoint = fmt.Sprintf("localhost:%d", l.mio.GetApiPort())

	// Connect dev storage
	os.Setenv(minio.MINIO_ENDPOINT_ENV, l.status.MinioEndpoint)
	os.Setenv(minio.MINIO_ACCESS_KEY_ENV, "minioadmin")
	os.Setenv(minio.MINIO_SECRET_KEY_ENV, "minioadmin")
	sp, err := minio.New()
	if err != nil {
		return err
	}

	// Connect dev documents
	os.Setenv("LOCAL_DB_DIR", l.status.RunDir)
	dp, err := boltdb_service.New()
	if err != nil {
		return err
	}

	// Connect secrets plugin
	os.Setenv("LOCAL_SEC_DIR", l.status.RunDir)
	secp, err := secret_service.New()
	if err != nil {
		return err
	}

	// Connect queue plugin
	os.Setenv("LOCAL_QUEUE_DIR", l.status.RunDir)
	qp, err := queue_service.New()
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

	ev, err := NewEvents(pool)
	if err != nil {
		return err
	}

	// Start a new gateway plugin
	gw, err := NewGateway(l.status.GatewayAddress)
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
		QueuePlugin:             qp,
		StoragePlugin:           sp,
		DocumentPlugin:          dp,
		GatewayPlugin:           gw,
		EventsPlugin:            ev,
		Pool:                    pool,
		TolerateMissingServices: false,
	})
	if err != nil {
		return err
	}

	return l.mem.Start()
}
