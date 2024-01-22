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
	"context"

	"github.com/asaskevich/EventBus"
	"github.com/nitrictech/cli/pkgplus/cloud/gateway"
	"github.com/nitrictech/cli/pkgplus/grpcx"
	resourcespb "github.com/nitrictech/nitric/core/pkg/proto/resources/v1"
)

type ResourceName = string

type LocalResourcesService struct {
	gateway *gateway.LocalGatewayService

	Apis        ResourceRegistrar[resourcespb.ApiResource]
	Buckets     ResourceRegistrar[resourcespb.BucketResource]
	Collections ResourceRegistrar[resourcespb.CollectionResource]
	Policies    ResourceRegistrar[resourcespb.PolicyResource]
	Secrets     ResourceRegistrar[resourcespb.SecretResource]
	Topics      ResourceRegistrar[resourcespb.TopicResource]

	bus EventBus.Bus
}

type LocalResourcesOptions struct {
	Gateway *gateway.LocalGatewayService
}

/*
var _ resourcespb.ResourcesServer = &LocalResourcesService{}

func (l *LocalResourcesService) getApiDetails(name string) (*resourcespb.ResourceDetailsResponse, error) {
	gatewayUri, ok := l.gateway.GetApiAddresses()[name]
	if !ok {
		return nil, fmt.Errorf("api %s does not exist", name)
	}

	// if !l.isStart {
	// 	gatewayUri = strings.Replace(gatewayUri, "localhost", "host.docker.internal", 1)
	// }

	return &resourcespb.ResourceDetailsResponse{
		Id:       name,
		Provider: "dev",
		Service:  "Api",
		Details: &resourcespb.ResourceDetailsResponse_Api{
			Api: &resourcespb.ApiResourceDetails{
				Url: fmt.Sprintf("http://%s", gatewayUri),
			},
		},
	}, nil
}

func (l *LocalResourcesService) getWebsocketDetails(name string) (*resourcespb.ResourceDetailsResponse, error) {
	gatewayUri, ok := l.gateway.GetWebsocketAddresses()[name]
	if !ok {
		return nil, fmt.Errorf("api %s does not exist", name)
	}

	// if !r.isStart {
	// 	gatewayUri = strings.Replace(gatewayUri, "localhost", "host.docker.internal", 1)
	// }

	return &resourcespb.ResourceDetailsResponse{
		Id:       name,
		Provider: "dev",
		Service:  "Websocket",
		Details: &resourcespb.ResourceDetailsResponse_Websocket{
			Websocket: &resourcespb.WebsocketResourceDetails{
				Url: fmt.Sprintf("ws://%s", gatewayUri),
			},
		},
	}, nil
}

*/

func (l *LocalResourcesService) Details(ctx context.Context, req *resourcespb.ResourceDetailsRequest) (*resourcespb.ResourceDetailsResponse, error) {
	// switch req.Resource.Type {
	// case resourcespb.ResourceType_Api:
	// 	return l.getApiDetails(req.Resource.Name)
	// case resourcespb.ResourceType_Websocket:
	// 	return l.getWebsocketDetails(req.Resource.Name)
	// default:
	// 	return nil, fmt.Errorf("unsupported resource type %s", req.Resource.Type)
	// }
	// TODO Refactor Declare and Details into their respected resources contracts (e.g. Storage/Apis/Collections etc.)
	return nil, nil
}

const localResourcesTopic = "local_resources"

// func (s *LocalResourcesService) SubscribeToState(fn func(State)) {
// 	s.bus.Subscribe(localResourcesTopic, fn)
// }

func (l *LocalResourcesService) Declare(ctx context.Context, req *resourcespb.ResourceDeclareRequest) (*resourcespb.ResourceDeclareResponse, error) {
	serviceName, err := grpcx.GetServiceNameFromIncomingContext(ctx)
	if err != nil {
		return nil, err
	}

	switch req.Id.Type {
	case resourcespb.ResourceType_Api:
		err = l.Apis.Register(req.Id.Name, serviceName, req.GetApi())
	case resourcespb.ResourceType_Bucket:
		// eventbus.Bus().Publish(DeclareBucketTopic, req.Id.Name)
		err = l.Buckets.Register(req.Id.Name, serviceName, req.GetBucket())
	case resourcespb.ResourceType_Collection:
		err = l.Collections.Register(req.Id.Name, serviceName, req.GetCollection())
	case resourcespb.ResourceType_Policy:
		err = l.Policies.Register(req.Id.Name, serviceName, req.GetPolicy())
	case resourcespb.ResourceType_Secret:
		err = l.Secrets.Register(req.Id.Name, serviceName, req.GetSecret())
	case resourcespb.ResourceType_Topic:
		err = l.Topics.Register(req.Id.Name, serviceName, req.GetTopic())
	}
	if err != nil {
		return nil, err
	}

	l.bus.Publish(localResourcesTopic, l)

	return &resourcespb.ResourceDeclareResponse{}, nil
}

// ClearServiceResources - Clear all resources registered by a service, typically done when the service terminates or is restarted
func (l *LocalResourcesService) ClearServiceResources(serviceName string) {
	l.Apis.ClearRequestingService(serviceName)
	l.Buckets.ClearRequestingService(serviceName)
	l.Collections.ClearRequestingService(serviceName)
	l.Policies.ClearRequestingService(serviceName)
	l.Secrets.ClearRequestingService(serviceName)
	l.Topics.ClearRequestingService(serviceName)
}

// TODO: Refactor Declare and Details into their respected resources contracts (e.g. Storage/Apis/Collections etc.)
func NewLocalResourcesService(opts LocalResourcesOptions) *LocalResourcesService {
	return &LocalResourcesService{
		gateway: opts.Gateway,
		bus:     EventBus.New(),
	}
}
