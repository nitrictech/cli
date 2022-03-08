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
	"github.com/pkg/errors"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/cloudrun"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/pubsub"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/nitrictech/cli/pkg/project"
	"github.com/nitrictech/cli/pkg/provider/pulumi/common"
	"github.com/nitrictech/cli/pkg/utils"
)

type CloudRunnerArgs struct {
	Location       pulumi.StringInput
	ProjectId      string
	Compute        project.Compute
	Image          *common.Image
	ServiceAccount *serviceaccount.Account
	Topics         map[string]*pubsub.Topic
}

type CloudRunner struct {
	pulumi.ResourceState

	Name    string
	Service *cloudrun.Service
	Url     pulumi.StringInput
}

func (g *gcpProvider) newCloudRunner(ctx *pulumi.Context, name string, args *CloudRunnerArgs, opts ...pulumi.ResourceOption) (*CloudRunner, error) {
	res := &CloudRunner{
		Name: name,
	}
	err := ctx.RegisterComponentResource("nitric:func:GCPCloudRunner", name, res, opts...)
	if err != nil {
		return nil, err
	}
	// // Give project editor permissions
	// // FIXME: Trim this down
	// _, err = projects.NewIAMMember(ctx, name+"-editor", &projects.IAMMemberArgs{
	// 	Role: pulumi.String("roles/editor"),
	// 	// Get the cloudrun service account email
	// 	Member:  pulumi.Sprintf("serviceAccount:%s", sa.Email),
	// 	Project: pulumi.String(args.ProjectId),
	// }, append(opts, pulumi.Parent(res))...)
	// if err != nil {
	// 	return nil, errors.WithMessage(err, "iam member "+name)
	// }

	// // Give secret accessor permissions
	// _, err = projects.NewIAMMember(ctx, name+"-secret-access", &projects.IAMMemberArgs{
	// 	Role: pulumi.String("roles/secretmanager.secretAccessor"),
	// 	// Get the cloudrun service account email
	// 	Member:  pulumi.Sprintf("serviceAccount:%s", sa.Email),
	// 	Project: pulumi.String(args.ProjectId),
	// }, append(opts, pulumi.Parent(res))...)
	// if err != nil {
	// 	return nil, errors.WithMessage(err, "iam member "+name)
	// }

	// Deploy the func
	memory := common.IntValueOrDefault(args.Compute.Unit().Memory, 128)
	maxScale := common.IntValueOrDefault(args.Compute.Unit().MaxScale, 10)
	minScale := common.IntValueOrDefault(args.Compute.Unit().MinScale, 0)
	res.Service, err = cloudrun.NewService(ctx, name, &cloudrun.ServiceArgs{
		Location: pulumi.String(g.sc.Region),
		Project:  pulumi.String(args.ProjectId),
		Template: cloudrun.ServiceTemplateArgs{
			Metadata: cloudrun.ServiceTemplateMetadataArgs{
				Annotations: pulumi.StringMap{
					"autoscaling.knative.dev/minScale": pulumi.Sprintf("%d", minScale),
					"autoscaling.knative.dev/maxScale": pulumi.Sprintf("%d", maxScale),
				},
			},
			Spec: cloudrun.ServiceTemplateSpecArgs{
				ServiceAccountName: args.ServiceAccount.Email,
				Containers: cloudrun.ServiceTemplateSpecContainerArray{
					cloudrun.ServiceTemplateSpecContainerArgs{
						Image: args.Image.DockerImage.ImageName, // TODO check
						Ports: cloudrun.ServiceTemplateSpecContainerPortArray{
							cloudrun.ServiceTemplateSpecContainerPortArgs{
								ContainerPort: pulumi.Int(9001),
							},
						},
						Resources: cloudrun.ServiceTemplateSpecContainerResourcesArgs{
							Limits: pulumi.StringMap{"memory": pulumi.Sprintf("%dMi", memory)},
						},
					},
				},
			},
		},
	}, append(opts, pulumi.Parent(res))...)
	if err != nil {
		return nil, errors.WithMessage(err, "iam member "+name)
	}

	res.Url = res.Service.Statuses.ApplyT(func(ss []cloudrun.ServiceStatus) (string, error) {
		if len(ss) == 0 {
			return "", errors.New("serviceStatus is empty")
		}
		return *ss[0].Url, nil
	}).(pulumi.StringInput)

	// wire up its subscriptions
	if len(args.Compute.Unit().Triggers.Topics) > 0 {
		// Create an account for invoking this func via subscriptions
		// TODO: Do we want to make this one account for subscription in future
		// TODO: We will likely configure this via eventarc in the future
		invokerAccount, err := serviceaccount.NewAccount(ctx, name+"subacct", &serviceaccount.AccountArgs{
			// accountId accepts a max of 30 chars, limit our generated name to this length
			AccountId: pulumi.String(utils.StringTrunc(name, 30-8) + "subacct"),
		}, append(opts, pulumi.Parent(res))...)
		if err != nil {
			return nil, errors.WithMessage(err, "invokerAccount "+name)
		}

		// Apply permissions for the above account to the newly deployed cloud run service
		_, err = cloudrun.NewIamMember(ctx, name+"-subrole", &cloudrun.IamMemberArgs{
			Member:   pulumi.Sprintf("serviceAccount:%s", invokerAccount.Email),
			Role:     pulumi.String("roles/run.invoker"),
			Service:  res.Service.Name,
			Location: res.Service.Location,
		}, append(opts, pulumi.Parent(res))...)
		if err != nil {
			return nil, errors.WithMessage(err, "iam member "+name)
		}

		for _, t := range args.Compute.Unit().Triggers.Topics {
			topic, ok := args.Topics[t]
			if ok {
				_, err = pubsub.NewSubscription(ctx, name+"-"+t+"-sub", &pubsub.SubscriptionArgs{
					Topic:              topic.Name,
					AckDeadlineSeconds: pulumi.Int(0),
					RetryPolicy: pubsub.SubscriptionRetryPolicyArgs{
						MinimumBackoff: pulumi.String("15s"),
						MaximumBackoff: pulumi.String("600s"),
					},
					PushConfig: pubsub.SubscriptionPushConfigArgs{
						OidcToken: pubsub.SubscriptionPushConfigOidcTokenArgs{
							ServiceAccountEmail: invokerAccount.Email,
						},
						PushEndpoint: res.Url,
					},
				}, append(opts, pulumi.Parent(res))...)
				if err != nil {
					return nil, errors.WithMessage(err, "subscription "+name+"-"+t+"-sub")
				}
			}
		}
	}

	return res, ctx.RegisterResourceOutputs(res, pulumi.Map{
		"name":    pulumi.String(res.Name),
		"service": res.Service,
		"url":     res.Url,
	})
}
