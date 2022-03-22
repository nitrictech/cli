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

	"github.com/ettle/strcase"
	"github.com/google/uuid"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/projects"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/pubsub"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/secretmanager"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/storage"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	v1 "github.com/nitrictech/nitric/pkg/api/nitric/v1"
)

type Policy struct {
	pulumi.ResourceState

	Name         string
	RolePolicies []*projects.IAMMember
}

type StackResources struct {
	Topics  map[string]*pubsub.Topic
	Queues  map[string]*pubsub.Topic
	Buckets map[string]*storage.Bucket
	Secrets map[string]*secretmanager.Secret
}

type PrincipalMap = map[v1.ResourceType]map[string]*serviceaccount.Account

type PolicyArgs struct {
	Policy *v1.PolicyResource
	// Resources in the stack that must be protected
	Resources *StackResources
	// Resources in the stack that may act as actors
	Principals PrincipalMap

	ProjectID pulumi.StringInput
}

var gcpActionsMap map[v1.Action][]string = map[v1.Action][]string{
	v1.Action_BucketFileList: {
		"storage.objects.list",
	},
	v1.Action_BucketFileGet: {
		"storage.objects.get",
		"iam.serviceAccounts.signBlob",
	},
	v1.Action_BucketFilePut: {
		"orgpolicy.policy.get",
		"storage.multipartUploads.abort",
		"storage.multipartUploads.create",
		"storage.multipartUploads.listParts",
		"storage.objects.create",
		"iam.serviceAccounts.signBlob",
	},
	v1.Action_BucketFileDelete: {
		"storage.objects.delete",
	},
	v1.Action_TopicDetail: {},
	v1.Action_TopicEventPublish: {
		"pubsub.topics.publish",
	},
	v1.Action_TopicList: {
		"pubsub.topics.list",
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
	v1.Action_CollectionDocumentDelete: {
		"appengine.applications.get",
		"datastore.databases.get",
		"datastore.indexes.get",
		"datastore.namespaces.get",
		"datastore.entities.delete",
	},
	v1.Action_CollectionDocumentRead: {
		"appengine.applications.get",
		"datastore.databases.get",
		"datastore.entities.get",
		"datastore.indexes.get",
		"datastore.namespaces.get",
		"datastore.entities.list",
	},
	v1.Action_CollectionDocumentWrite: {
		"appengine.applications.get",
		"datastore.indexes.list",
		"datastore.namespaces.list",
		"datastore.entities.create",
		"datastore.entities.update",
	},
	v1.Action_CollectionQuery: {
		"appengine.applications.get",
		"datastore.databases.get",
		"datastore.entities.get",
		"datastore.entities.list",
		"datastore.indexes.get",
		"datastore.namespaces.get",
	},
	v1.Action_CollectionList: {
		"appengine.applications.get",
	},
	v1.Action_SecretAccess: {
		"secretmanager.locations.*",
		"secretmanager.secrets.get",
		"secretmanager.secrets.getIamPolicy",
		"secretmanager.version.get",
		"secretmanager.secrets.list",
		"secretmanager.versions.list",
	},
	v1.Action_SecretPut: {
		"secretmanager.versions.add",
	},
}

var collectionActions []string = nil

func getCollectionActions() []string {
	if collectionActions == nil {
		collectionActions = make([]string, 0)
		collectionActions = append(collectionActions, gcpActionsMap[v1.Action_CollectionDocumentRead]...)
		collectionActions = append(collectionActions, gcpActionsMap[v1.Action_CollectionDocumentWrite]...)
		collectionActions = append(collectionActions, gcpActionsMap[v1.Action_CollectionDocumentDelete]...)
	}

	return collectionActions
}

func filterCollectionActions(actions pulumi.StringArray) pulumi.StringArrayOutput {
	arr, _ := actions.ToStringArrayOutput().ApplyT(func(actions []string) []string {
		filteredActions := []string{}

		for _, a := range actions {
			for _, ca := range getCollectionActions() {
				if a == ca {
					filteredActions = append(filteredActions, a)
					break
				}
			}
		}

		return filteredActions
	}).(pulumi.StringArrayOutput)

	return arr
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

	rolePolicy, err := projects.NewIAMCustomRole(ctx, name, &projects.IAMCustomRoleArgs{
		Title:       pulumi.String(name),
		Permissions: actions,
		RoleId:      pulumi.String(strcase.ToCamel(name)),
	}, pulumi.Parent(res))

	if err != nil {
		return nil, err
	}

	for _, principal := range args.Policy.Principals {
		sa := args.Principals[v1.ResourceType_Function][principal.Name]
		name := newCustomRoleName(principal.Name)

		for _, resource := range args.Policy.Resources {
			memberName := fmt.Sprintf("%s-%s", principal.Name, resource.Name)
			memberId := pulumi.Sprintf("serviceAccount:%s", sa.Email)

			switch resource.Type {
			case v1.ResourceType_Bucket:
				b := args.Resources.Buckets[resource.Name]

				_, err = storage.NewBucketIAMMember(ctx, memberName, &storage.BucketIAMMemberArgs{
					Bucket: b.Name,
					Member: memberId,
					Role:   rolePolicy.Name,
				}, pulumi.Parent(res))

				if err != nil {
					return nil, err
				}

			case v1.ResourceType_Collection:
				collActions := filterCollectionActions(actions)

				collRole, err := projects.NewIAMCustomRole(ctx, name, &projects.IAMCustomRoleArgs{
					Title:       pulumi.String(name),
					Permissions: collActions,
					RoleId:      pulumi.String(strcase.ToCamel(name)),
				}, pulumi.Parent(res))

				if err != nil {
					return nil, err
				}

				_, err = projects.NewIAMMember(ctx, memberName, &projects.IAMMemberArgs{
					Member:  memberId,
					Project: args.ProjectID,
					Role:    collRole.Name,
				}, pulumi.Parent(res))

				if err != nil {
					return nil, err
				}

			case v1.ResourceType_Queue:
				q := args.Resources.Queues[resource.Name]

				_, err = pubsub.NewTopicIAMMember(ctx, memberName, &pubsub.TopicIAMMemberArgs{
					Topic:  q.Name,
					Member: memberId,
					Role:   rolePolicy.Name,
				}, pulumi.Parent(res))

				if err != nil {
					return nil, err
				}

			case v1.ResourceType_Topic:
				t := args.Resources.Topics[resource.Name]

				_, err = pubsub.NewTopicIAMMember(ctx, memberName, &pubsub.TopicIAMMemberArgs{
					Topic:  t.Name,
					Member: memberId,
					Role:   rolePolicy.Name,
				}, pulumi.Parent(res))

				if err != nil {
					return nil, err
				}

			case v1.ResourceType_Secret:
				s := args.Resources.Secrets[resource.Name]

				_, err = secretmanager.NewSecretIamMember(ctx, memberName, &secretmanager.SecretIamMemberArgs{
					SecretId: s.SecretId,
					Member:   memberId,
					Role:     rolePolicy.Name,
				}, pulumi.Parent(res))

				if err != nil {
					return nil, err
				}
			}
		}
	}

	return res, nil
}
