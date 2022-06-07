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
	"strings"

	"github.com/pkg/errors"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/cloudwatch"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/lambda"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/nitrictech/cli/pkg/cron"
	"github.com/nitrictech/cli/pkg/project"
	"github.com/nitrictech/cli/pkg/provider/pulumi/common"
)

type ScheduleArgs struct {
	Expression string
	Functions  map[string]*Lambda
	Schedule   project.Schedule
}

type Schedule struct {
	pulumi.ResourceState

	Name        string
	EventRule   *cloudwatch.EventRule
	EventTarget *cloudwatch.EventTarget
}

func (a *awsProvider) newSchedule(ctx *pulumi.Context, name string, args ScheduleArgs, opts ...pulumi.ResourceOption) (*Schedule, error) {
	res := &Schedule{Name: name}
	normalizedName := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	err := ctx.RegisterComponentResource("nitric:schedule:AwsSchedule", name, res, opts...)
	if err != nil {
		return nil, err
	}

	opts = append(opts, pulumi.Parent(res))

	awsCronValue, err := cron.ConvertToAWS(args.Expression)
	if err != nil {
		return nil, err
	}
	res.EventRule, err = cloudwatch.NewEventRule(ctx, normalizedName, &cloudwatch.EventRuleArgs{
		ScheduleExpression: pulumi.String(awsCronValue),
		Tags:               common.Tags(ctx, name),
	}, opts...)

	if err != nil {
		return nil, errors.WithMessage(err, "error creating CloudWatch EventRule")
	}

	var targetArn pulumi.StringInput = nil
	switch args.Schedule.Target.Type {
	case "function":
		if f, ok := args.Functions[args.Schedule.Target.Name]; ok {
			targetArn = f.Function.Arn
			// give the event rule created above permission to access this lambda

			_, err := lambda.NewPermission(ctx, normalizedName+"LambdaPermission", &lambda.PermissionArgs{
				Action:    pulumi.String("lambda:InvokeFunction"),
				Principal: pulumi.String("events.amazonaws.com"),
				SourceArn: res.EventRule.Arn,
				Function:  f.Function.Name,
			})

			if err != nil {
				return nil, err
			}
		}
	case "topic":
		return nil, fmt.Errorf("schedule to topic target support has been deprecated and removed")
	}

	if targetArn == nil {
		return nil, fmt.Errorf("unable to resolve schedule target")
	}

	res.EventTarget, err = cloudwatch.NewEventTarget(ctx, normalizedName, &cloudwatch.EventTargetArgs{
		Rule: res.EventRule.Name,
		Arn:  targetArn,
	}, opts...)
	if err != nil {
		return nil, err
	}

	return res, err
}
