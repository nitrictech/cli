package pulumi

import (
	"fmt"

	"github.com/nitrictech/cli/pkg/output"
	"github.com/nitrictech/cli/pkg/provider/types"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/events"
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
