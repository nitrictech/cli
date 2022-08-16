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

package pulumi

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/auto/events"

	"github.com/nitrictech/cli/pkg/output"
	"github.com/nitrictech/cli/pkg/provider/types"
)

// Use to collect summary from pulumi stack updates
type pulumiLogger struct {
	resources map[string]*types.ResourceState

	output.Progress
}

func (p *pulumiLogger) getResourceState(urn string) *types.ResourceState {
	if p.resources[urn] == nil {
		p.resources[urn] = &types.ResourceState{
			Errored:  false,
			Messages: make([]string, 0),
		}
	}

	return p.resources[urn]
}

func (p *pulumiLogger) CollectEvent(evt events.EngineEvent) {
	if p.resources == nil {
		p.resources = map[string]*types.ResourceState{}
	}

	if evt.DiagnosticEvent != nil && evt.DiagnosticEvent.URN != "" {
		rs := p.getResourceState(evt.DiagnosticEvent.URN)

		rs.Messages = append(rs.Messages, evt.DiagnosticEvent.Message)
	}

	if evt.ResOpFailedEvent != nil {
		rs := p.getResourceState(evt.ResOpFailedEvent.Metadata.URN)
		rs.Errored = true
		rs.OpType = fmt.Sprintf("%s", evt.ResOpFailedEvent.Metadata.Op)
	}

	if evt.ResOutputsEvent != nil {
		rs := p.getResourceState(evt.ResOutputsEvent.Metadata.URN)
		rs.OpType = fmt.Sprintf("%s", evt.ResOutputsEvent.Metadata.Op)
	}
}
