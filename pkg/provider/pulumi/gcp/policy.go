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

package gcp

import (
	"fmt"

	"github.com/google/uuid"
	v1 "github.com/nitrictech/nitric/pkg/api/nitric/v1"
	"github.com/pkg/errors"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/firestore"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/projects"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/pubsub"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/secretmanager"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/storage"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Policy struct {
	pulumi.ResourceState

	Name         string
	RolePolicies []*projects.IAMMember
}

type StackResources struct {
	Topics      map[string]*pubsub.Topic
	Queues      map[string]*pubsub.Topic
	Buckets     map[string]*storage.Bucket
	Collections map[string]*firestore.Document
	Secrets     map[string]*secretmanager.Secret
}

type PrincipalMap = map[v1.ResourceType]map[string]*serviceaccount.Account

type PolicyArgs struct {
	Policy *v1.PolicyResource
	// Resources in the stack tha tmust be protected
	Resources *StackResources
	// Resources in the stack that may act as actors
	Principals PrincipalMap

	ProjectID pulumi.StringOutput
}

var gcpActionsMap map[v1.Action][]string = map[v1.Action][]string{
	v1.Action_BucketFileList: {
		"storage.objects.list",
	},
	v1.Action_BucketFileGet: {
		"storage.objects.get",
	},
	v1.Action_BucketFilePut: {
		"orgpolicy.policy.get",
		"storage.multipartUploads.abort",
		"storage.multipartUploads.create",
		"storage.multipartUploads.listParts",
		"storage.objects.create",
	},
	v1.Action_BucketFileDelete: {
		"storage.objects.delete",
	},
	v1.Action_TopicDetail: {},
	v1.Action_TopicEventPublish: {
		"pubsub.topics.publish",
	},
	v1.Action_TopicList: {
		"pubusb.topics.list",
	},
	v1.Action_QueueSend: {
		"pubsub.topics.publish",
	},
	v1.Action_QueueReceive: {
		"pubsub.topics.attachSubscription",
		"pubsub.snapshots.seek",
		"pubsub.subscriptions.consume",
	},
	v1.Action_QueueDetail: {},
	v1.Action_QueueList: {
		"pubsub.topics.list",
	},
	v1.Action_CollectionDocumentRead: {
		"appengine.applications.get",
		"datastore.databases.get",
		"datastore.databases.getMetadata",
		"datastore.entities.get",
		"datastore.indexes.get",
		"datastore.namespaces.get",
	},
	v1.Action_CollectionDocumentWrite: {
		"appengine.applications.get",
		"datastore.databases.list",
		"datastore.entities.list",
		"datastore.indexes.list",
		"datastore.namespaces.list",
	},
	v1.Action_CollectionQuery: {
		"appengine.applications.get",
		"datastore.databases.get",
		"datastore.databases.getMetadata",
		"datastore.entities.get",
		"datastore.indexes.get",
		"datastore.namespaces.get",
	},
	v1.Action_CollectionList: {
		"appengine.applications.get",
		"resourcemanager.projects.get",
		"resourcemanager.projects.list",
	},
	v1.Action_SecretAccess: {
		"resourcemanager.projects.get",
		"resourcemanager.projects.list",
		"secretmanager.locations.*",
		"secretmnager.secrets.get",
		"secretmanager.secrets.getIamPolicy",
		"secretmanager.version.get",
		"secretmanager.secrets.list",
		"secretmnager.versions.list",
	},
	v1.Action_SecretPut: {
		"resourcemanager.projects.get",
		"resourcemanager.projects.list",
		"secretmanager.versions.add",
	},
}

func actionsToGcpActions(actions []v1.Action) pulumi.StringArray {
	gcpActions := make(pulumi.StringArray, 0)

	for _, a := range actions {
		for _, ga := range gcpActionsMap[a] {
			gcpActions = append(gcpActions, pulumi.String(ga))
		}
	}

	return gcpActions
}

// Custom roles soft delete, so must have more randomization for each name, or there will be conflict on each deploy
func newCustomRoleName(princName string) string {
	id := uuid.New()
	return fmt.Sprintf("%s-%s", princName, id.String())
}

func newPolicy(ctx *pulumi.Context, name string, args *PolicyArgs, opts ...pulumi.ResourceOption) (*Policy, error) {
	res := &Policy{Name: name, RolePolicies: make([]*projects.IAMMember, 0)}
	err := ctx.RegisterComponentResource("nitric:func:GCPPolicy", name, res, opts...)
	if err != nil {
		return nil, err
	}

	actions := actionsToGcpActions(args.Policy.Actions)

	for _, principal := range args.Policy.Principals {
		sa := args.Principals[v1.ResourceType_Function][principal.Name]
		name := newCustomRoleName(principal.Name)

		role, err := projects.NewIAMCustomRole(ctx, name, &projects.IAMCustomRoleArgs{
			Permissions: actions,
			RoleId:      pulumi.String(name),
		}, append(opts, pulumi.Parent(res))...)
		if err != nil {
			return nil, err
		}

		_, err = projects.NewIAMMember(ctx, "", &projects.IAMMemberArgs{
			Member:  pulumi.Sprintf("serviceAccount:%s", sa.Email),
			Project: args.ProjectID,
			Role:    role.ID(),
		}, append(opts, pulumi.Parent(res))...)
		if err != nil {
			return nil, errors.WithMessage(err, "iam member "+principal.Name)
		}
	}

	return res, nil
}
