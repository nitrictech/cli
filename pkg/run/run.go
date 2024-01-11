//go:build ignore
// +build ignore

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
	"time"

	"github.com/nitrictech/cli/pkg/eventbus/resourceevts"
	"github.com/nitrictech/nitric/core/pkg/membrane"
)

type LocalServices interface {
	Start(suppressLogs bool) error
	Stop() error
	Running() bool
	Status() *LocalServicesStatus
	Refresh() error
	Apis() map[string]string
	HttpWorkers() map[string]string
	Websockets() map[string]string
	TriggerAddress() string
}

type LocalServicesStatus struct {
	// RunDir string `yaml:"runDir"`
	// GatewayAddress  string `yaml:"gatewayAddress"`
	MembraneAddress string `yaml:"membraneAddress"`
}

type localServices struct {
	membrane   *membrane.Membrane
	status     *LocalServicesStatus
	gateway    *LocalGatewayService
	isStartCmd bool
}

func NewLocalServices(isStartCmd bool) LocalServices {
	return &localServices{
		isStartCmd: isStartCmd,
		status: &LocalServicesStatus{
			// RunDir:          NITRIC_LOCAL_RUN_DIR,
			MembraneAddress: net.JoinHostPort("localhost", "50051"),
		},
	}
}

func (l *localServices) TriggerAddress() string {
	if l.gateway != nil {
		return l.gateway.GetTriggerAddress()
	}

	return ""
}

func (l *localServices) Refresh() error {
	if l.gateway != nil {
		err := l.gateway.Refresh()
		if err != nil {
			return err
		}
	}

	resourceevts.Publish(resourceevts.LocalInfrastructureState{
		TriggerAddress:     l.TriggerAddress(),
		ApiAddresses:       l.Apis(),
		WebSocketAddresses: l.Websockets(),
		StorageAddress:     l.Status().StorageEndpoint,
		ServiceListener:    l.gateway.serviceListener,
	})

	return nil
}

func (l *localServices) Stop() error {
	l.membrane.Stop()
	// return l.storageService.StopSeaweed()
}

func (l *localServices) Running() bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("0.0.0.0", "50051"), time.Second)
	if err == nil && conn != nil {
		defer conn.Close()

		return true
	}

	return false
}

func (l *localServices) Apis() map[string]string {
	if l.gateway != nil {
		return l.gateway.GetApiAddresses()
	}

	return nil
}

func (l *localServices) HttpWorkers() map[string]string {
	if l.gateway != nil {
		return l.gateway.GetHttpWorkerAddresses()
	}

	return nil
}

func (l *localServices) Websockets() map[string]string {
	if l.gateway != nil {
		return l.gateway.GetWebsocketAddresses()
	}

	return nil
}

func (l *localServices) Status() *LocalServicesStatus {
	return l.status
}

func (l *localServices) Start(suppressLogs bool) error {
	var err error

	// l.storage, err = NewSeaweed()
	// if err != nil {
	// 	return err
	// }

	// start seaweed server
	err = l.storage.Start()
	if err != nil {
		return err
	}

	l.status.StorageEndpoint = fmt.Sprintf("http://localhost:%d", l.storage.GetApiPort())

	l.storageService, err = NewStorage(StorageOptions{
		AccessKey: "dummykey",
		SecretKey: "dummysecret",
		Endpoint:  l.status.StorageEndpoint,
	})
	if err != nil {
		return err
	}

	dp, err := NewBoltService()
	if err != nil {
		return err
	}

	secp, err := NewSecretService()
	if err != nil {
		return err
	}

	ev, err := NewEvents()
	if err != nil {
		return err
	}

	wsPlugin, _ := NewRunWebsocketService()

	// Start a new gateway plugin
	l.gateway, err = NewGateway(wsPlugin)
	if err != nil {
		return err
	}

	events, err := NewEvents()
	if err != nil {
		return err
	}
	// l.gateway.dash = l.dashboard

	// Prepare development membrane to start
	// This will start a single membrane that all
	// running functions will connect to
	l.membrane, err = membrane.New(&membrane.MembraneOptions{
		ApiPlugin:               NewLocalApiGateway(),
		HttpPlugin:              NewLocalHttpGateway(),
		SchedulesPlugin:         NewLocalSchedules(),
		TopicsListenerPlugin:    events,
		StorageListenerPlugin:   l.storageService,
		WebsocketListenerPlugin: wsPlugin,
		// HttpPlugin: New,

		ServiceAddress:          "0.0.0.0:50051",
		SecretManagerPlugin:     secp,
		StoragePlugin:           l.storageService,
		DocumentPlugin:          dp,
		GatewayPlugin:           l.gateway,
		TopicsPlugin:            ev,
		ResourcesPlugin:         NewResources(l, l.isStartCmd),
		WebsocketPlugin:         wsPlugin,
		TolerateMissingServices: false,
		SuppressLogs:            suppressLogs,
	})
	if err != nil {
		return err
	}

	return l.membrane.Start()
}

func (l *localServices) GetStorageService() *RunStorageService {
	return l.storageService
}
