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
	"crypto/md5"
	"encoding/hex"
	"encoding/json"

	"github.com/asaskevich/EventBus"
	"github.com/nitrictech/cli/pkgplus/cloud/gateway"
	"github.com/nitrictech/cli/pkgplus/grpcx"
	resourcespb "github.com/nitrictech/nitric/core/pkg/proto/resources/v1"
)

type ResourceName = string

type LocalResourcesState struct {
	Buckets     *ResourceRegistrar[resourcespb.BucketResource]
	Collections *ResourceRegistrar[resourcespb.CollectionResource]
	Policies    *ResourceRegistrar[resourcespb.PolicyResource]
	Secrets     *ResourceRegistrar[resourcespb.SecretResource]
	Topics      *ResourceRegistrar[resourcespb.TopicResource]
}

type LocalResourcesService struct {
	gateway *gateway.LocalGatewayService

	state LocalResourcesState

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

func (s *LocalResourcesService) SubscribeToState(fn func(lrs LocalResourcesState)) {
	s.bus.Subscribe(localResourcesTopic, fn)
}

// policyResourceName generates a unique name for a policy resource by hashing the policy document
func policyResourceName(policy *resourcespb.PolicyResource) (string, error) {
	policyDoc, err := json.Marshal(policy)
	if err != nil {
		return "", err
	}

	hasher := md5.New()
	hasher.Write(policyDoc)

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func (l *LocalResourcesService) Declare(ctx context.Context, req *resourcespb.ResourceDeclareRequest) (*resourcespb.ResourceDeclareResponse, error) {
	serviceName, err := grpcx.GetServiceNameFromIncomingContext(ctx)
	if err != nil {
		return nil, err
	}

	switch req.Id.Type {
	case resourcespb.ResourceType_Bucket:
		err = l.state.Buckets.Register(req.Id.Name, serviceName, req.GetBucket())
	case resourcespb.ResourceType_Collection:
		err = l.state.Collections.Register(req.Id.Name, serviceName, req.GetCollection())
	case resourcespb.ResourceType_Policy:
		// Services don't know their own name, so forgetful 🙄, that's ok, we'll add it here.
		for _, principal := range req.GetPolicy().Principals {
			if principal.Type == resourcespb.ResourceType_Service {
				principal.Name = serviceName
			}
		}

		policyName, policyErr := policyResourceName(req.GetPolicy())
		if policyErr != nil {
			return nil, policyErr
		}

		err = l.state.Policies.Register(policyName, serviceName, req.GetPolicy())
	case resourcespb.ResourceType_Secret:
		err = l.state.Secrets.Register(req.Id.Name, serviceName, req.GetSecret())
	case resourcespb.ResourceType_Topic:
		err = l.state.Topics.Register(req.Id.Name, serviceName, req.GetTopic())
	}
	if err != nil {
		return nil, err
	}

	l.bus.Publish(localResourcesTopic, l.state)

	return &resourcespb.ResourceDeclareResponse{}, nil
}

// ClearServiceResources - Clear all resources registered by a service, typically done when the service terminates or is restarted
func (l *LocalResourcesService) ClearServiceResources(serviceName string) {
	l.state.Buckets.ClearRequestingService(serviceName)
	l.state.Collections.ClearRequestingService(serviceName)
	l.state.Policies.ClearRequestingService(serviceName)
	l.state.Secrets.ClearRequestingService(serviceName)
	l.state.Topics.ClearRequestingService(serviceName)
}

// TODO: Refactor Declare and Details into their respected resources contracts (e.g. Storage/Apis/Collections etc.)
func NewLocalResourcesService(opts LocalResourcesOptions) *LocalResourcesService {
	return &LocalResourcesService{
		state: LocalResourcesState{
			Buckets:     NewResourceRegistrar[resourcespb.BucketResource](),
			Collections: NewResourceRegistrar[resourcespb.CollectionResource](),
			Policies:    NewResourceRegistrar[resourcespb.PolicyResource](),
			Secrets:     NewResourceRegistrar[resourcespb.SecretResource](),
			Topics:      NewResourceRegistrar[resourcespb.TopicResource](),
		},
		gateway: opts.Gateway,
		bus:     EventBus.New(),
	}
}
