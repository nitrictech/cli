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
	"github.com/nitrictech/newcli/pkg/provider/local"
	"github.com/nitrictech/newcli/pkg/provider/types"
	"github.com/nitrictech/newcli/pkg/stack"
	"github.com/nitrictech/newcli/pkg/target"
)

func NewProvider(s *stack.Stack, t *target.Target) (types.Provider, error) {
	switch t.Provider {
	case "local":
		return local.New(s, t)
	default:
		return nil, utils.NewNotSupportedErr(fmt.Sprintf("provider %s is not supported", t.Provider))
	}
}
