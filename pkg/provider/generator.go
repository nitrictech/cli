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

package provider

import (
	"fmt"

	"github.com/nitrictech/cli/pkg/project"
	"github.com/nitrictech/cli/pkg/provider/pulumi"
	"github.com/nitrictech/cli/pkg/provider/types"
	"github.com/nitrictech/cli/pkg/stack"
	"github.com/nitrictech/cli/pkg/utils"
)

func getProviderOpts[T interface{}](opts []interface{}) *T {
	for _, o := range opts {
		t, ok := o.(*T)
		if ok {
			return t
		}
	}
	return nil
}

func NewProvider(p *project.Project, s *stack.Config, envMap map[string]string, opts ...interface{}) (types.Provider, error) {
	switch s.Provider {
	case stack.Aws, stack.Azure, stack.Digitalocean, stack.Gcp:
		pulumiOpts := getProviderOpts[pulumi.PulumiOpts](opts)
		return pulumi.New(p, s, envMap, pulumiOpts)
	default:
		return nil, utils.NewNotSupportedErr(fmt.Sprintf("provider %s is not supported", s.Provider))
	}
}
