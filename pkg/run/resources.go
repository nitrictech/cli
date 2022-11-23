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

	"github.com/nitrictech/nitric/pkg/providers/common"
)

type RunResourcesService struct {
	gatewayUri string
}

var _ common.ResourceService = &RunResourcesService{}

func (r *RunResourcesService) getApiDetails(name string) (*common.DetailsResponse[any], error) {
	return &common.DetailsResponse[any]{
		Id:       name,
		Provider: "dev",
		Service:  "Api",
		Detail: common.ApiDetails{
			URL: fmt.Sprintf("%s/apis/%s", r.gatewayUri, name),
		},
	}, nil
}

func (r *RunResourcesService) Details(typ common.ResourceType, name string) (*common.DetailsResponse[any], error) {
	switch typ {
	case common.ResourceType_Api:
		return r.getApiDetails(name)
	default:
		return nil, fmt.Errorf("unsupported resource type %s", typ)
	}
}

func NewResources(gatewayUri string) common.ResourceService {
	return &RunResourcesService{
		gatewayUri: gatewayUri,
	}
}
