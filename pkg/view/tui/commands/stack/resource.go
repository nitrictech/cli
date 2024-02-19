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

var VerbMap = map[deploymentspb.ResourceDeploymentAction]map[deploymentspb.ResourceDeploymentStatus]string{
	deploymentspb.ResourceDeploymentAction_CREATE: {
		deploymentspb.ResourceDeploymentStatus_PENDING:     "create",
		deploymentspb.ResourceDeploymentStatus_IN_PROGRESS: "creating",
		deploymentspb.ResourceDeploymentStatus_FAILED:      "creation failed",
		deploymentspb.ResourceDeploymentStatus_SUCCESS:     "created",
	},
	deploymentspb.ResourceDeploymentAction_DELETE: {
		deploymentspb.ResourceDeploymentStatus_PENDING:     "delete",
		deploymentspb.ResourceDeploymentStatus_SUCCESS:     "deleted",
		deploymentspb.ResourceDeploymentStatus_IN_PROGRESS: "deleting",
		deploymentspb.ResourceDeploymentStatus_FAILED:      "failed to delete",
	},
	deploymentspb.ResourceDeploymentAction_REPLACE: {
		deploymentspb.ResourceDeploymentStatus_PENDING:     "replace",
		deploymentspb.ResourceDeploymentStatus_SUCCESS:     "replaced",
		deploymentspb.ResourceDeploymentStatus_IN_PROGRESS: "replacing",
		deploymentspb.ResourceDeploymentStatus_FAILED:      "failed to replace",
	},
	deploymentspb.ResourceDeploymentAction_UPDATE: {
		deploymentspb.ResourceDeploymentStatus_PENDING:     "update",
		deploymentspb.ResourceDeploymentStatus_SUCCESS:     "updated",
		deploymentspb.ResourceDeploymentStatus_IN_PROGRESS: "updating",
		deploymentspb.ResourceDeploymentStatus_FAILED:      "failed to update",
	},
	deploymentspb.ResourceDeploymentAction_SAME: {
		deploymentspb.ResourceDeploymentStatus_PENDING:     "unchanged",
		deploymentspb.ResourceDeploymentStatus_SUCCESS:     "unchanged",
		deploymentspb.ResourceDeploymentStatus_IN_PROGRESS: "unchanged",
		deploymentspb.ResourceDeploymentStatus_FAILED:      "unchanged",
	},
}
