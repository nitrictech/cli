package stack

import (
	"time"

	deploymentspb "github.com/nitrictech/nitric/core/pkg/proto/deployments/v1"
)

type Resource struct {
	Name       string
	Message    string
	Action     deploymentspb.ResourceDeploymentAction
	Status     deploymentspb.ResourceDeploymentStatus
	StartTime  time.Time
	FinishTime time.Time
	Children   []*Resource
}
