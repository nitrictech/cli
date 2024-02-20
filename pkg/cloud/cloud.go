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

package cloud

import (
	"fmt"
	"sync"

	"github.com/samber/lo"
	"google.golang.org/grpc"

	"github.com/nitrictech/cli/pkg/cloud/apis"
	"github.com/nitrictech/cli/pkg/cloud/gateway"
	"github.com/nitrictech/cli/pkg/cloud/http"
	"github.com/nitrictech/cli/pkg/cloud/keyvalue"
	"github.com/nitrictech/cli/pkg/cloud/queues"
	"github.com/nitrictech/cli/pkg/cloud/resources"
	"github.com/nitrictech/cli/pkg/cloud/schedules"
	"github.com/nitrictech/cli/pkg/cloud/secrets"
	"github.com/nitrictech/cli/pkg/cloud/storage"
	"github.com/nitrictech/cli/pkg/cloud/topics"
	"github.com/nitrictech/cli/pkg/cloud/websockets"
	"github.com/nitrictech/cli/pkg/grpcx"
	"github.com/nitrictech/cli/pkg/netx"
	"github.com/nitrictech/nitric/core/pkg/logger"
	"github.com/nitrictech/nitric/core/pkg/membrane"
)

type Subscribable[T any, A any] interface {
	SubscribeToState(fn func(T))
	SubscribeToAction(fn func(A)) // used to subscribe to api calls, ws messages, topic deliveries etc
}

type ServiceName = string

type LocalCloud struct {
	membraneLock sync.Mutex
	membranes    map[ServiceName]*membrane.Membrane

	Apis       *apis.LocalApiGatewayService
	KeyValue   *keyvalue.BoltDocService
	Gateway    *gateway.LocalGatewayService
	Http       *http.LocalHttpProxy
	Resources  *resources.LocalResourcesService
	Schedules  *schedules.LocalSchedulesService
	Secrets    *secrets.DevSecretService
	Storage    *storage.LocalStorageService
	Topics     *topics.LocalTopicsAndSubscribersService
	Websockets *websockets.LocalWebsocketService
	Queues     *queues.LocalQueuesService

	// Store all the plugins locally
}

// StartLocalNitric - starts the Nitric Server (membrane), including plugins and their local dependencies (e.g. local versions of cloud services
func (lc *LocalCloud) Stop() {
	for _, m := range lc.membranes {
		m.Stop()
	}

	err := lc.Gateway.Stop()
	if err != nil {
		logger.Errorf("Error stopping gateway: %s", err.Error())
	}
}

func (lc *LocalCloud) AddService(serviceName string) (int, error) {
	lc.membraneLock.Lock()
	defer lc.membraneLock.Unlock()

	if _, ok := lc.membranes[serviceName]; ok {
		return 0, fmt.Errorf("service %s already started", serviceName)
	}

	// get an available port
	ports, err := netx.TakePort(1)
	if err != nil {
		return 0, err
	}

	nitricMembraneServer, _ := membrane.New(&membrane.MembraneOptions{
		// worker/listener plugins (these delegate incoming events/requests to handlers written with nitric)
		ApiPlugin:               lc.Apis,
		HttpPlugin:              lc.Http,
		SchedulesPlugin:         lc.Schedules,
		TopicsListenerPlugin:    lc.Topics,
		StorageListenerPlugin:   lc.Storage,
		WebsocketListenerPlugin: lc.Websockets,

		// address used by nitric clients to connect to the membrane (e.g. SDKs)
		ServiceAddress: fmt.Sprintf("0.0.0.0:%d", ports[0]),

		// cloud service plugins
		SecretManagerPlugin: lc.Secrets,
		StoragePlugin:       lc.Storage,
		KeyValuePlugin:      lc.KeyValue,
		GatewayPlugin:       lc.Gateway,
		TopicsPlugin:        lc.Topics,
		ResourcesPlugin:     lc.Resources,
		WebsocketPlugin:     lc.Websockets,
		QueuesPlugin:        lc.Queues,

		MinWorkers: lo.ToPtr(0),

		SuppressLogs: false,
	})

	// Create a watcher that clears old resources when the service is restarted
	_, err = resources.NewServiceResourceRefresher(serviceName, resources.NewServiceResourceRefresherArgs{
		Resources:  lc.Resources,
		Apis:       lc.Apis,
		Schedules:  lc.Schedules,
		Http:       lc.Http,
		Listeners:  lc.Storage,
		Websockets: lc.Websockets,
		Topics:     lc.Topics,
		Storage:    lc.Storage,
	})
	if err != nil {
		return 0, err
	}

	go func() {
		interceptor, streamInterceptor := grpcx.CreateServiceNameInterceptor(serviceName)

		srv := grpc.NewServer(
			grpc.UnaryInterceptor(interceptor),
			grpc.StreamInterceptor(streamInterceptor),
		)

		err := nitricMembraneServer.Start(membrane.WithGrpcServer(srv))
		if err != nil {
			logger.Errorf("Error starting membrane: %s", err.Error())
		}
	}()

	lc.membranes[serviceName] = nitricMembraneServer

	return ports[0], nil
}

func New() (*LocalCloud, error) {
	localTopics, err := topics.NewLocalTopicsService()
	if err != nil {
		return nil, err
	}

	localWebsockets, err := websockets.NewLocalWebsocketService()
	if err != nil {
		return nil, err
	}

	localStorage, err := storage.NewLocalStorageService(storage.StorageOptions{
		AccessKey: "dummykey",
		SecretKey: "dummysecret",
	})
	if err != nil {
		return nil, err
	}

	localApis := apis.NewLocalApiGatewayService()

	localSchedules := schedules.NewLocalSchedulesService()
	localHttpProxy := http.NewLocalHttpProxyService()

	localSecrets, err := secrets.NewSecretService()
	if err != nil {
		return nil, err
	}

	localGateway, err := gateway.NewGateway()
	if err != nil {
		return nil, err
	}

	localResources := resources.NewLocalResourcesService(resources.LocalResourcesOptions{
		Gateway: localGateway,
	})

	keyvalueService, err := keyvalue.NewBoltService()
	if err != nil {
		return nil, err
	}

	localQueueService, err := queues.NewLocalQueuesService()
	if err != nil {
		return nil, err
	}

	return &LocalCloud{
		membranes:  make(map[string]*membrane.Membrane),
		Apis:       localApis,
		Http:       localHttpProxy,
		Resources:  localResources,
		Schedules:  localSchedules,
		Storage:    localStorage,
		Topics:     localTopics,
		Websockets: localWebsockets,
		Gateway:    localGateway,
		Secrets:    localSecrets,
		KeyValue:   keyvalueService,
		Queues:     localQueueService,
	}, nil
}
