package services

import (
	"fmt"
	"sync"

	"github.com/nitrictech/cli/pkgplus/cloud/apis"
	"github.com/nitrictech/cli/pkgplus/cloud/http"
	"github.com/nitrictech/cli/pkgplus/cloud/resources"
	"github.com/nitrictech/cli/pkgplus/cloud/schedules"
	"github.com/nitrictech/cli/pkgplus/cloud/storage"
	"github.com/nitrictech/cli/pkgplus/cloud/topics"
	"github.com/nitrictech/cli/pkgplus/cloud/websockets"
)

type ServiceState struct {
	serviceName string

	resourcesPlugin *resources.LocalResourcesService

	lock             sync.RWMutex
	apiWorkers       int
	scheduleWorkers  int
	httpWorkers      int
	listenerWorkers  int
	websocketWorkers int
}

type UpdateArgs struct {
	apiState             apis.State
	schedulesState       schedules.State
	websocketState       websockets.State
	bucketListenersState storage.State
	topicSubscriberState topics.State
	httpState            http.State
}

func (s *ServiceState) allWorkerCount() int {
	return s.apiWorkers + s.scheduleWorkers + s.httpWorkers + s.listenerWorkers + s.websocketWorkers
}

func (s *ServiceState) updatesWorkers(update UpdateArgs) {
	s.lock.Lock()
	defer s.lock.Unlock()
	previous := s.allWorkerCount()

	if update.apiState != nil {
		s.apiWorkers = 0
		for _, api := range update.apiState {
			s.apiWorkers += len(api[s.serviceName])
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

	// When the worker count for a service is 0, we can assume that the service is not running.
	// Typically this happens during a hot-reload/restarting a service and means the policies should be reset, since new policy requests will be coming in.
	if previous > 0 && s.allWorkerCount() == 0 {
		s.resourcesPlugin.ClearServiceResources(s.serviceName)
	}
}

type NewServiceStateArgs struct {
	resources *resources.LocalResourcesService

	apis      *apis.LocalApiGatewayService
	schedules *schedules.LocalSchedulesService
	http      *http.LocalHttpProxy
	listeners *storage.LocalStorageService
	websocket *websockets.LocalWebsocketService
	topics    *topics.LocalTopicsAndSubscribersService
	storage   *storage.LocalStorageService
}

func NewServiceState(serviceName string, args NewServiceStateArgs) (*ServiceState, error) {
	if args.resources == nil || args.apis == nil || args.schedules == nil || args.http == nil || args.listeners == nil || args.websocket == nil {
		return nil, fmt.Errorf("all service plugins are required")
	}

	serviceState := &ServiceState{
		serviceName:     serviceName,
		resourcesPlugin: args.resources,
		lock:            sync.RWMutex{},
	}

	args.apis.SubscribeToState(func(s apis.State) {
		serviceState.updatesWorkers(UpdateArgs{
			apiState: s,
		})
	})

	args.http.SubscribeToState(func(s http.State) {
		serviceState.updatesWorkers(UpdateArgs{
			httpState: s,
		})
	})

	args.websocket.SubscribeToState(func(s websockets.State) {
		serviceState.updatesWorkers(UpdateArgs{
			websocketState: s,
		})
	})

	args.schedules.SubscribeToState(func(s schedules.State) {
		serviceState.updatesWorkers(UpdateArgs{
			schedulesState: s,
		})
	})

	args.topics.SubscribeToState(func(s topics.State) {
		serviceState.updatesWorkers(UpdateArgs{
			topicSubscriberState: s,
		})
	})

	args.storage.SubscribeToState(func(s storage.State) {
		serviceState.updatesWorkers(UpdateArgs{
			bucketListenersState: s,
		})
	})

	return serviceState, nil
}
