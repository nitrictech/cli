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

package resources

import (
	"fmt"
	"sync"

	"github.com/nitrictech/cli/pkg/cloud/apis"
	"github.com/nitrictech/cli/pkg/cloud/batch"
	"github.com/nitrictech/cli/pkg/cloud/http"
	"github.com/nitrictech/cli/pkg/cloud/schedules"
	"github.com/nitrictech/cli/pkg/cloud/storage"
	"github.com/nitrictech/cli/pkg/cloud/topics"
	"github.com/nitrictech/cli/pkg/cloud/websockets"
)

type ServiceResourceRefresher struct {
	serviceName string

	resourcesPlugin *LocalResourcesService
	httpProxyPlugin *http.LocalHttpProxy

	lock              sync.RWMutex
	apiWorkers        int
	batchWorkers      int
	scheduleWorkers   int
	httpWorkers       int
	listenerWorkers   int
	subscriberWorkers int
	websocketWorkers  int
}

type UpdateArgs struct {
	apiState             apis.State
	batchState           batch.State
	schedulesState       schedules.State
	websocketState       websockets.State
	bucketListenersState storage.State
	topicSubscriberState topics.State
	httpState            http.State
}

func (s *ServiceResourceRefresher) allWorkerCount() int {
	return s.apiWorkers + s.scheduleWorkers + s.httpWorkers + s.listenerWorkers + s.subscriberWorkers + s.websocketWorkers + s.batchWorkers
}

func (s *ServiceResourceRefresher) updatesWorkers(update UpdateArgs) {
	s.lock.Lock()
	defer s.lock.Unlock()
	previous := s.allWorkerCount()

	if update.apiState != nil {
		s.apiWorkers = 0
		for _, api := range update.apiState {
			s.apiWorkers += len(api[s.serviceName])
		}
	}

	if update.topicSubscriberState != nil {
		s.subscriberWorkers = 0
		for _, topic := range update.topicSubscriberState {
			s.subscriberWorkers += topic[s.serviceName]
		}
	}

	if update.schedulesState != nil {
		s.scheduleWorkers = 0
		for _, schedule := range update.schedulesState {
			if schedule.ServiceName == s.serviceName {
				s.scheduleWorkers += 1
			}
		}
	}

	if update.websocketState != nil {
		s.websocketWorkers = 0
		for _, websocket := range update.websocketState {
			s.websocketWorkers += len(websocket[s.serviceName])
		}
	}

	if update.bucketListenersState != nil {
		s.listenerWorkers = 0
		for _, listenerCounts := range update.bucketListenersState {
			s.listenerWorkers += listenerCounts[s.serviceName]
		}
	}

	if update.httpState != nil {
		s.httpWorkers = 0
		for _, httpApi := range update.httpState {
			if httpApi.ServiceName == s.serviceName {
				s.httpWorkers += 1
			}
		}
	}

	if update.batchState != nil {
		s.batchWorkers = 0
		for _, batch := range update.batchState {
			s.batchWorkers += batch[s.serviceName]
		}
	}

	// When the worker count for a service is 0, we can assume that the service is not running.
	// Typically this happens during a hot-reload/restarting a service and means the policies should be reset, since new policy requests will be coming in.
	if previous > 0 && s.allWorkerCount() == 0 {
		s.resourcesPlugin.ClearServiceResources(s.serviceName)
	}
}

type NewServiceResourceRefresherArgs struct {
	Resources *LocalResourcesService

	Apis       *apis.LocalApiGatewayService
	Schedules  *schedules.LocalSchedulesService
	Http       *http.LocalHttpProxy
	Listeners  *storage.LocalStorageService
	Websockets *websockets.LocalWebsocketService
	Topics     *topics.LocalTopicsAndSubscribersService
	Storage    *storage.LocalStorageService
	BatchJobs  *batch.LocalBatchService
}

func NewServiceResourceRefresher(serviceName string, args NewServiceResourceRefresherArgs) (*ServiceResourceRefresher, error) {
	if args.Resources == nil || args.Apis == nil || args.Schedules == nil || args.Http == nil || args.Listeners == nil || args.Websockets == nil || args.BatchJobs == nil {
		return nil, fmt.Errorf("all service plugins are required")
	}

	serviceState := &ServiceResourceRefresher{
		serviceName:     serviceName,
		resourcesPlugin: args.Resources,
		httpProxyPlugin: args.Http,
		lock:            sync.RWMutex{},
	}

	args.Apis.SubscribeToState(func(s apis.State) {
		serviceState.updatesWorkers(UpdateArgs{
			apiState: s,
		})
	})

	args.BatchJobs.SubscribeToState(func(s batch.State) {
		serviceState.updatesWorkers(UpdateArgs{
			batchState: s,
		})
	})

	args.Http.SubscribeToState(func(s http.State) {
		serviceState.updatesWorkers(UpdateArgs{
			httpState: s,
		})
	})

	args.Websockets.SubscribeToState(func(s websockets.State) {
		serviceState.updatesWorkers(UpdateArgs{
			websocketState: s,
		})
	})

	args.Schedules.SubscribeToState(func(s schedules.State) {
		serviceState.updatesWorkers(UpdateArgs{
			schedulesState: s,
		})
	})

	args.Topics.SubscribeToState(func(s topics.State) {
		serviceState.updatesWorkers(UpdateArgs{
			topicSubscriberState: s,
		})
	})

	args.Storage.SubscribeToState(func(s storage.State) {
		serviceState.updatesWorkers(UpdateArgs{
			bucketListenersState: s,
		})
	})

	return serviceState, nil
}
