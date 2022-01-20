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

	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/cloudwatch"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/sns"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/nitrictech/newcli/pkg/cron"
)

func (a *awsProvider) schedule(ctx *pulumi.Context, name, expression string, topic *sns.Topic) error {
	awsCronValue, err := cron.ConvertToAWS(expression)
	if err != nil {
		return err
	}

	eventRule, err := cloudwatch.NewEventRule(ctx, name+"Schedule", &cloudwatch.EventRuleArgs{
		ScheduleExpression: pulumi.String("cron(" + awsCronValue + ")"),
		Tags:               commonTags(ctx, name+"Schedule"),
	})
	if err != nil {
		return err
	}

	_, err = cloudwatch.NewEventTarget(ctx, name+"Target", &cloudwatch.EventTargetArgs{
		Rule: eventRule.Name,
		Arn:  topic.Arn,
	})
	if err != nil {
		return err
	}

	rolepolicyJSON, err := json.Marshal(map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []interface{}{
			map[string]interface{}{
				"SID":    "__default_statement_ID",
				"Effect": "Allow",
				"Action": []string{"SNS:Publish*"},
				"Principal": map[string]interface{}{
					"Service": "events.amazonaws.com",
				},
				"Resource": topic.Arn.ToStringOutput(),
			},
		},
	})
	if err != nil {
		return err
	}

	_, err = sns.NewTopicPolicy(ctx, fmt.Sprintf("%sTarget%vPolicy", name, topic.Name), &sns.TopicPolicyArgs{
		Arn:    topic.Arn,
		Policy: pulumi.String(rolepolicyJSON),
	})

	return err
}
