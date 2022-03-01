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
	"reflect"
	"sync"
	"testing"

	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/dynamodb"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/s3"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/sns"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/sqs"
	"github.com/pulumi/pulumi-docker/sdk/v3/go/docker"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"

	"github.com/nitrictech/cli/pkg/provider/pulumi/common"
	"github.com/nitrictech/cli/pkg/stack"
	"github.com/nitrictech/cli/pkg/target"
	v1 "github.com/nitrictech/nitric/pkg/api/nitric/v1"
)

type mocks int

// Create the mock.
func (mocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	outputs := args.Inputs.Mappable()
	fmt.Println(args.TypeToken)
	switch args.TypeToken {
	case "aws:sns/topic:Topic":
		outputs["arn"] = "test-arn"
	case "aws:s3/bucket:Bucket":
		outputs["bucket"] = args.Name
	}
	outputs["name"] = args.Name
	return args.Name + "_id", resource.NewPropertyMapFromMap(outputs), nil
}

func (mocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	outputs := map[string]interface{}{}
	return resource.NewPropertyMapFromMap(outputs), nil
}

func TestAWS(t *testing.T) {
	s := stack.New("atest", ".")
	s.Topics = map[string]stack.Topic{"sales": {}}
	s.Buckets = map[string]stack.Bucket{"money": {}}
	s.Queues = map[string]stack.Queue{"checkout": {}}
	s.Collections = map[string]stack.Collection{"customer": {}}
	s.Schedules = map[string]stack.Schedule{
		"daily": {
			Expression: "@daily",
			Target:     stack.ScheduleTarget{Type: "topic", Name: "sales"},
			Event:      stack.ScheduleEvent{PayloadType: "?"},
		},
	}
	s.Functions = map[string]stack.Function{
		"runnner": {
			Handler: "functions/create/main.go",
			ComputeUnit: stack.ComputeUnit{
				Name:     "runner",
				Triggers: stack.Triggers{Topics: []string{"sales"}},
			},
		},
	}
	s.Policies = []*v1.PolicyResource{
		{
			Principals: []*v1.Resource{
				{
					Type: v1.ResourceType_Function,
					Name: "runner",
				},
			},
			Actions: []v1.Action{
				v1.Action_BucketFileGet, v1.Action_BucketFileList,
			},
			Resources: []*v1.Resource{
				{
					Type: v1.ResourceType_Bucket,
					Name: "money",
				},
			},
		},
	}

	projectName := s.Name
	stackName := s.Name + "-deploy"

	a := &awsProvider{
		s: s,
		t: &target.Target{
			Provider: target.Aws,
			Region:   "mock",
		},
		topics:      map[string]*sns.Topic{},
		buckets:     map[string]*s3.Bucket{},
		queues:      map[string]*sqs.Queue{},
		collections: map[string]*dynamodb.Table{},
		schedules:   map[string]*Schedule{},
		images: map[string]*common.Image{
			"runner": {
				DockerImage: &docker.Image{
					ImageName: pulumi.Sprintf("docker.io/nitrictech/runner:latest"),
				},
			},
		},
		funcs: map[string]*Lambda{},
	}

	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		err := a.Deploy(ctx)
		assert.NoError(t, err)

		var wg sync.WaitGroup

		wg.Add(1)
		pulumi.All(a.rg.Name, a.rg.ResourceQuery.Query()).ApplyT(func(all []interface{}) error {
			name := all[0].(string)
			query := all[1].(string)
			expectQuery := `{"ResourceTypeFilters":["AWS::AllSupported"],"TagFilters":[{"Key":"x-nitric-stack","Values":["atest-deploy"]}]}`

			assert.Equal(t, stackName, name, "resourceGroup has the wrong name %s!=%s", stackName, name)
			assert.Equal(t, expectQuery, query, "resourceGroup has the wrong query %s!=%s", expectQuery, query)

			wg.Done()
			return nil
		})

		wg.Add(1)
		a.topics["sales"].Name.ApplyT(func(name string) error {
			assert.Equal(t, "sales", name, "topic has the wrong name %s!=%s", "sales", name)
			wg.Done()
			return nil
		})

		wg.Add(1)
		a.topics["sales"].Tags.ApplyT(func(tags map[string]string) error {
			expectTags := map[string]string{"x-nitric-name": "sales", "x-nitric-project": "atest", "x-nitric-stack": "atest-deploy"}
			assert.Equal(t, expectTags, tags, "topic has the wrong tags %s!=%s", expectTags, tags)
			wg.Done()
			return nil
		})

		wg.Add(1)
		a.buckets["money"].Bucket.ApplyT(func(name string) error {
			assert.Equal(t, "money", name, "bucket has the wrong name %s!=%s", "money", name)
			wg.Done()
			return nil
		})

		wg.Add(1)
		a.buckets["money"].Tags.ApplyT(func(tags map[string]string) error {
			expectTags := map[string]string{"x-nitric-name": "money", "x-nitric-project": "atest", "x-nitric-stack": "atest-deploy"}
			assert.Equal(t, expectTags, tags, "money has the wrong tags %s!=%s", expectTags, tags)
			wg.Done()
			return nil
		})

		wg.Add(1)
		a.queues["checkout"].Name.ApplyT(func(name string) error {
			assert.Equal(t, "checkout", name, "queue has the wrong name %s!=%s", "checkout", name)
			wg.Done()
			return nil
		})

		wg.Add(1)
		a.queues["checkout"].Tags.ApplyT(func(tags map[string]string) error {
			expectTags := map[string]string{"x-nitric-name": "checkout", "x-nitric-project": "atest", "x-nitric-stack": "atest-deploy"}
			assert.Equal(t, expectTags, tags, "checkout has the wrong tags %s!=%s", expectTags, tags)
			wg.Done()
			return nil
		})

		wg.Add(1)
		a.collections["customer"].Tags.ApplyT(func(tags map[string]string) error {
			expectTags := map[string]string{"x-nitric-name": "customer", "x-nitric-project": "atest", "x-nitric-stack": "atest-deploy"}
			assert.Equal(t, expectTags, tags, "customer has the wrong tags %s!=%s", expectTags, tags)
			wg.Done()
			return nil
		})

		wg.Add(1)
		a.collections["customer"].Attributes.ApplyT(func(attrs []dynamodb.TableAttribute) error {
			expectAttrs := []dynamodb.TableAttribute{
				{
					Name: "_pk",
					Type: "S",
				},
				{
					Name: "_sk",
					Type: "S",
				},
			}

			assert.Equal(t, expectAttrs, attrs, "customer table has the wrong attrs %s!=%s", expectAttrs, attrs)
			wg.Done()
			return nil
		})

		wg.Add(1)
		pulumi.All(a.funcs["runner"].Function.ImageUri, a.funcs["runner"].Function.Role, a.funcs["runner"].Role.Arn).ApplyT(func(all []interface{}) error {
			imageUri := all[0].(*string)
			fRole := all[1].(string)
			roleArn := all[2].(string)

			assert.Equal(t, "docker.io/nitrictech/runner:latest", *imageUri, "wrong imageUri %s!=%s", "", *imageUri)
			assert.Equal(t, roleArn, fRole, "wrong role %s!=%s", roleArn, fRole)
			wg.Done()
			return nil
		})

		wg.Add(1)
		pulumi.All(a.schedules["daily"].EventRule.ScheduleExpression, a.schedules["daily"].EventTarget.Arn, a.topics["sales"].Arn).ApplyT(func(all []interface{}) error {
			expr := all[0].(*string)
			arn := all[1].(string)
			topicArn := all[2].(string)

			assert.Equal(t, "cron(0 0 * * ? *)", *expr, "wrong expression %s!=%s", "", *expr)
			assert.Equal(t, topicArn, arn, "wrong arn %s!=%s", topicArn, arn)
			wg.Done()
			return nil
		})

		wg.Wait()
		return nil
	}, pulumi.WithMocks(projectName, stackName, mocks(0)))
	assert.NoError(t, err)

	a.CleanUp()
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		t       *target.Target
		wantErr bool
	}{
		{
			name: "valid",
			t:    &target.Target{Provider: target.Aws, Region: "us-west-1"},
		},
		{
			name:    "invalid",
			t:       &target.Target{Provider: target.Aws, Region: "pole-north-right-next-to-santa"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(nil, tt.t)
			if err := a.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("awsProvider.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_awsProvider_Plugins(t *testing.T) {
	want := []common.Plugin{
		{Name: "aws", Version: "v4.37.5"},
	}
	got := (&awsProvider{}).Plugins()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("awsProvider.Plugins() = %v, want %v", got, want)
	}
}
