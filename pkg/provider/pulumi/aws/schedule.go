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

package aws

import (
	"fmt"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/cloudwatch"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/sns"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/nitrictech/cli/pkg/cron"
	"github.com/nitrictech/cli/pkg/provider/pulumi/common"
)

type ScheduleArgs struct {
	Expression string
	TopicArn   pulumi.StringOutput
	TopicName  pulumi.StringInput
}

type Schedule struct {
	pulumi.ResourceState

	Name        string
	EventRule   *cloudwatch.EventRule
	EventTarget *cloudwatch.EventTarget
}

func (a *awsProvider) newSchedule(ctx *pulumi.Context, name string, args ScheduleArgs, opts ...pulumi.ResourceOption) (*Schedule, error) {
	res := &Schedule{Name: name}

	err := ctx.RegisterComponentResource("nitric:schedule:AwsSchedule", name, res, opts...)
	if err != nil {
		return nil, err
	}

	opts = append(opts, pulumi.Parent(res))

	awsCronValue, err := cron.ConvertToAWS(args.Expression)
	if err != nil {
		return nil, err
	}

	res.EventRule, err = cloudwatch.NewEventRule(ctx, name+"Schedule", &cloudwatch.EventRuleArgs{
		ScheduleExpression: pulumi.String(awsCronValue),
		Tags:               common.Tags(ctx, name+"Schedule"),
	}, opts...)
	if err != nil {
		return nil, err
	}

	res.EventTarget, err = cloudwatch.NewEventTarget(ctx, name+"Target", &cloudwatch.EventTargetArgs{
		Rule: res.EventRule.Name,
		Arn:  args.TopicArn,
	}, opts...)
	if err != nil {
		return nil, err
	}

	pdocJSON := args.TopicArn.ApplyT(func(arn string) (string, error) {
		pdoc, err := iam.GetPolicyDocument(ctx, &iam.GetPolicyDocumentArgs{
			PolicyId: to.StringPtr("__default_policy_ID"),
			Statements: []iam.GetPolicyDocumentStatement{
				{
					Sid:     to.StringPtr("__default_statement_ID"),
					Effect:  to.StringPtr("Allow"),
					Actions: []string{"SNS:Publish"},
					Principals: []iam.GetPolicyDocumentStatementPrincipal{
						{Type: "Service", Identifiers: []string{"events.amazonaws.com"}},
					},
					Resources: []string{arn},
				},
			},
		})
		if err != nil {
			return "", err
		}

		return pdoc.Json, nil
	}).(pulumi.StringInput)

	_, err = sns.NewTopicPolicy(ctx, fmt.Sprintf("%sTarget%vPolicy", name, args.TopicName), &sns.TopicPolicyArgs{
		Arn:    args.TopicArn,
		Policy: pdocJSON,
	}, opts...)

	return res, err
}
