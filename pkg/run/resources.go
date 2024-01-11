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
	"context"
	"fmt"
	"strings"

	resourcespb "github.com/nitrictech/nitric/core/pkg/proto/resources/v1"
)

type RunResourcesService struct {
	localSrvcs *localServices
	isStart    bool
}

var _ resourcespb.ResourcesServer = &RunResourcesService{}

func (r *RunResourcesService) getApiDetails(name string) (*resourcespb.ResourceDetailsResponse, error) {
	gatewayUri, ok := r.localSrvcs.gateway.GetApiAddresses()[name]
	if !ok {
		return nil, fmt.Errorf("api %s does not exist", name)
	}

	if !r.isStart {
		gatewayUri = strings.Replace(gatewayUri, "localhost", "host.docker.internal", 1)
	}

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

func (r *RunResourcesService) getWebsocketDetails(name string) (*resourcespb.ResourceDetailsResponse, error) {
	gatewayUri, ok := r.localSrvcs.gateway.GetWebsocketAddresses()[name]
	if !ok {
		return nil, fmt.Errorf("api %s does not exist", name)
	}

	if !r.isStart {
		gatewayUri = strings.Replace(gatewayUri, "localhost", "host.docker.internal", 1)
	}

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

func (r *RunResourcesService) Details(ctx context.Context, req *resourcespb.ResourceDetailsRequest) (*resourcespb.ResourceDetailsResponse, error) {
	switch req.Resource.Type {
	case resourcespb.ResourceType_Api:
		return r.getApiDetails(req.Resource.Name)
	case resourcespb.ResourceType_Websocket:
		return r.getWebsocketDetails(req.Resource.Name)
	default:
		return nil, fmt.Errorf("unsupported resource type %s", req.Resource.Type)
	}
}

func (r *RunResourcesService) Declare(ctx context.Context, req *resourcespb.ResourceDeclareRequest) (*resourcespb.ResourceDeclareResponse, error) {
	// resource := req.Resource

	// switch resource.Type {
	// case resourcespb.ResourceType_Bucket:
	// 	r.localSrvcs.dashboard.AddBucket(resource.GetName())
	// }

	return &resourcespb.ResourceDeclareResponse{}, nil
}

func NewResources(ls *localServices, isStart bool) *RunResourcesService {
	return &RunResourcesService{
		localSrvcs: ls,
		isStart:    isStart,
	}
}
