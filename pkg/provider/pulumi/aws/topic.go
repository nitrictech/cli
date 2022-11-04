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
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/sfn"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/sns"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/nitrictech/cli/pkg/project"
	"github.com/nitrictech/cli/pkg/provider/pulumi/common"
)

type Topic struct {
	pulumi.ResourceState
	Name string
	Sns  *sns.Topic
	Sfn  *sfn.StateMachine
}

type TopicArgs struct {
	StackID pulumi.StringInput
	Topic   project.Topic
}

func newTopic(ctx *pulumi.Context, name string, args *TopicArgs, opts ...pulumi.ResourceOption) (*Topic, error) {
	res := &Topic{Name: name}

	err := ctx.RegisterComponentResource("nitric:topic:AwsSnsTopic", name, res, opts...)
	if err != nil {
		return nil, err
	}

	// create the SNS topic
	res.Sns, err = sns.NewTopic(ctx, name, &sns.TopicArgs{
		Tags: common.Tags(ctx, args.StackID, name),
	}, pulumi.Parent(res))
	if err != nil {
		return nil, err
	}

	// create a State Machine to support delayed messaging
	// unfortunately we cannot create a single dynamic state machine that uses
	// the topicArn as input so we need to create one per topic
	// Note this is going to be better for security
	r, _ := json.Marshal(map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			{
				"Sid":    "",
				"Effect": "Allow",
				"Principal": map[string]interface{}{
					"Service": "states.amazonaws.com",
				},
				"Action": "sts:AssumeRole",
			},
		},
	})

	sfnRole, err := iam.NewRole(ctx, fmt.Sprintf("%s-delay-ctrl", name), &iam.RoleArgs{
		AssumeRolePolicy: pulumi.String(r),
	})
	if err != nil {
		return nil, errors.WithMessage(err, "topic delay controller role")
	}

	policy := res.Sns.Arn.ApplyT(func(arn string) (string, error) {
		rp, err := json.Marshal(map[string]interface{}{
			"Version": "2012-10-17",
			"Statement": []map[string]interface{}{
				{
					"Sid":      "",
					"Effect":   "Allow",
					"Action":   []string{"sns:Publish"},
					"Resource": arn,
				},
			},
		})

		return string(rp), err
	})

	// Enable a role with publish access to this stacks topics only
	_, err = iam.NewRolePolicy(ctx, fmt.Sprintf("%s-delay-ctrl", name), &iam.RolePolicyArgs{
		Role: sfnRole,
		// TODO: Limit to only this stacks topics (deployed above)
		Policy: policy,
	})
	if err != nil {
		return nil, errors.WithMessage(err, "topic delay controller role policy")
	}

	sfnDef := res.Sns.Arn.ApplyT(func(arn string) (string, error) {
		def, err := json.Marshal(map[string]interface{}{
			"Comment": "",
			"StartAt": "Wait",
			"States": map[string]interface{}{
				"Wait": map[string]string{
					"Type":        "Wait",
					"SecondsPath": "$.seconds",
					"Next":        "Publish",
				},
				"Publish": map[string]interface{}{
					"Type":     "Task",
					"Resource": "arn:aws:states:::sns:publish",
					"Parameters": map[string]string{
						"TopicArn":  arn,
						"Message.$": "$.message",
					},
					"End": true,
				},
			},
		})

		return string(def), err
	}).(pulumi.StringOutput)

	// Deploy a delay manager using AWS step functions
	// This will enable runtime delaying of event
	res.Sfn, err = sfn.NewStateMachine(ctx, fmt.Sprintf("%s-delay-ctrl", name), &sfn.StateMachineArgs{
		RoleArn: sfnRole.Arn,
		// Apply the same name as the topic to the state machine
		Tags:       common.Tags(ctx, args.StackID, fmt.Sprintf("%s", name)),
		Definition: sfnDef,
	})
	if err != nil {
		return nil, errors.WithMessage(err, "topic delay controller")
	}

	return res, nil
}
