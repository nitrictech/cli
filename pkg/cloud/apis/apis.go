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

package apis

import (
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/asaskevich/EventBus"
	"github.com/valyala/fasthttp"

	"github.com/nitrictech/cli/pkg/grpcx"
	apispb "github.com/nitrictech/nitric/core/pkg/proto/apis/v1"
	"github.com/nitrictech/nitric/core/pkg/workers/apis"
)

type (
	ApiName     = string
	ServiceName = string
	State       = map[ApiName]map[ServiceName][]*apispb.RegistrationRequest
)

type ApiRequestState struct {
	Api      string
	ReqCtx   *fasthttp.RequestCtx
	HttpResp *apispb.HttpResponse
}
type LocalApiGatewayService struct {
	*apis.RouteWorkerManager

	apiRegLock sync.RWMutex
	state      State

	bus EventBus.Bus
}

func deepCopyApiMap(originalMap map[ApiName]map[ServiceName][]*apispb.RegistrationRequest) map[ApiName]map[ServiceName][]*apispb.RegistrationRequest {
	copiedMap := make(map[ApiName]map[ServiceName][]*apispb.RegistrationRequest)

	for apiName, serviceMap := range originalMap {
		copiedMap[apiName] = make(map[ServiceName][]*apispb.RegistrationRequest)

		for serviceName, requests := range serviceMap {
			copiedRequests := make([]*apispb.RegistrationRequest, len(requests))
			copy(copiedRequests, requests)
			copiedMap[apiName][serviceName] = copiedRequests
		}
	}

	return copiedMap
}

const localApiGatewayTopic = "local_api_gateway"

const localApiRequestTopic = "local_api_gateway_request"

func (l *LocalApiGatewayService) publishState() {
	l.bus.Publish(localApiGatewayTopic, l.GetState())
}

var _ apispb.ApiServer = (*LocalApiGatewayService)(nil)

func (l *LocalApiGatewayService) SubscribeToState(subscriberFunction func(State)) {
	// ignore the error, it's only returned if the fn param isn't a function
	_ = l.bus.Subscribe(localApiGatewayTopic, subscriberFunction)
}

func (l *LocalApiGatewayService) PublishActionState(state ApiRequestState) {
	l.bus.Publish(localApiRequestTopic, state)
}

func (l *LocalApiGatewayService) SubscribeToAction(subscription func(ApiRequestState)) {
	// ignore the error, it's only returned if the fn param isn't a function
	_ = l.bus.Subscribe(localApiRequestTopic, subscription)
}

// GetState - Returns a copy of internal state
func (l *LocalApiGatewayService) GetState() State {
	l.apiRegLock.RLock()
	defer l.apiRegLock.RUnlock()

	return deepCopyApiMap(l.state)
}

func (l *LocalApiGatewayService) registerApiWorker(serviceName string, registrationRequest *apispb.RegistrationRequest) error {
	l.apiRegLock.Lock()

	if !strings.HasPrefix(registrationRequest.Path, "/") {
		return fmt.Errorf("service %s attempted to register path '%s' which is missing a leading slash", registrationRequest.Api, registrationRequest.Path)
	}

	if l.state[registrationRequest.Api] == nil {
		l.state[registrationRequest.Api] = make(map[string][]*apispb.RegistrationRequest)
	}

	l.state[registrationRequest.Api][serviceName] = append(l.state[registrationRequest.Api][serviceName], registrationRequest)

	l.apiRegLock.Unlock()

	l.publishState()

	return nil
}

func (l *LocalApiGatewayService) unregisterApiWorker(serviceName string, registrationRequest *apispb.RegistrationRequest) {
	l.apiRegLock.Lock()
	defer func() {
		l.apiRegLock.Unlock()
		l.publishState()
	}()

	l.state[registrationRequest.Api][serviceName] = slices.DeleteFunc(l.state[registrationRequest.Api][serviceName], func(item *apispb.RegistrationRequest) bool {
		return item == registrationRequest
	})

	if len(l.state[registrationRequest.Api][serviceName]) == 0 {
		delete(l.state[registrationRequest.Api], serviceName)
	}

	if len(l.state[registrationRequest.Api]) == 0 {
		delete(l.state, registrationRequest.Api)
	}
}

func (l *LocalApiGatewayService) Serve(stream apispb.Api_ServeServer) error {
	serviceName, err := grpcx.GetServiceNameFromStream(stream)
	if err != nil {
		return err
	}

	peekableStream := grpcx.NewPeekableStreamServer[*apispb.ServerMessage, *apispb.ClientMessage](stream)

	firstRequest, err := peekableStream.Peek()
	if err != nil {
		return err
	}

	if firstRequest.GetRegistrationRequest() == nil {
		return fmt.Errorf("first request must be a Registration Request")
	}

	// register the api
	err = l.registerApiWorker(serviceName, firstRequest.GetRegistrationRequest())
	if err != nil {
		return err
	}

	defer l.unregisterApiWorker(serviceName, firstRequest.GetRegistrationRequest())

	return l.RouteWorkerManager.Serve(peekableStream)
}

func NewLocalApiGatewayService() *LocalApiGatewayService {
	return &LocalApiGatewayService{
		RouteWorkerManager: apis.New(),
		state:              State{},
		bus:                EventBus.New(),
	}
}
