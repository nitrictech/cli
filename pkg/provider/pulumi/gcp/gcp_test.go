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
	"reflect"
	"sync"
	"testing"

	"github.com/pulumi/pulumi-docker/sdk/v3/go/docker"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/cloudrun"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/pubsub"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/storage"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"

	"github.com/nitrictech/cli/pkg/project"
	"github.com/nitrictech/cli/pkg/provider/pulumi/common"
	"github.com/nitrictech/cli/pkg/stack"
	v1 "github.com/nitrictech/nitric/pkg/api/nitric/v1"
)

type mocks int

// Create the mock.
func (mocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	outputs := args.Inputs.Mappable()
	fmt.Println(args.TypeToken)
	switch args.TypeToken {
	case "gcp:cloudrun/service:Service":
		outputs["statuses"] = []map[string]string{{"url": "test/url"}}
	}
	outputs["name"] = args.Name
	return args.Name + "_id", resource.NewPropertyMapFromMap(outputs), nil
}

func (mocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	outputs := map[string]interface{}{}
	return resource.NewPropertyMapFromMap(outputs), nil
}

func TestGCP(t *testing.T) {
	s := project.New(&project.Config{Name: "atest", Dir: "."})
	s.Topics = map[string]project.Topic{"sales": {}}
	s.Buckets = map[string]project.Bucket{"money": {}}
	s.Queues = map[string]project.Queue{"checkout": {}}
	s.Functions = map[string]project.Function{
		"runnner": {
			Handler: "functions/create/main.go",
			ComputeUnit: project.ComputeUnit{
				Name:     "runner",
				Triggers: project.Triggers{Topics: []string{"sales"}},
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

	a := &gcpProvider{
		proj: s,
		sc: &stack.Config{
			Provider: stack.Aws,
			Region:   "mock",
		},
		token:              &oauth2.Token{AccessToken: "testing-token"},
		projectId:          "test-project-id",
		projectNumber:      "test-project-number",
		buckets:            map[string]*storage.Bucket{},
		topics:             map[string]*pubsub.Topic{},
		queueTopics:        map[string]*pubsub.Topic{},
		queueSubscriptions: map[string]*pubsub.Subscription{},
		images: map[string]*common.Image{
			"runner": {
				DockerImage: &docker.Image{
					ImageName: pulumi.Sprintf("docker.io/nitrictech/runner:latest"),
				},
			},
		},
		cloudRunners: map[string]*CloudRunner{},
	}

	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		err := a.Deploy(ctx)
		assert.NoError(t, err)

		var wg sync.WaitGroup

		wg.Add(1)
		a.topics["sales"].Name.ApplyT(func(name string) error {
			assert.Equal(t, "sales", name, "topic has the wrong name %s!=%s", "sales", name)
			wg.Done()
			return nil
		})

		wg.Add(1)
		a.topics["sales"].Labels.ApplyT(func(tags map[string]string) error {
			expectTags := map[string]string{"x-nitric-name": "sales", "x-nitric-project": "atest", "x-nitric-stack": "atest-deploy"}
			assert.Equal(t, expectTags, tags, "topic has the wrong tags %s!=%s", expectTags, tags)
			wg.Done()
			return nil
		})

		wg.Add(1)
		a.buckets["money"].Name.ApplyT(func(name string) error {
			assert.Equal(t, "money", name, "bucket has the wrong name %s!=%s", "money", name)
			wg.Done()
			return nil
		})

		wg.Add(1)
		a.buckets["money"].Labels.ApplyT(func(tags map[string]string) error {
			expectTags := map[string]string{"x-nitric-name": "money", "x-nitric-project": "atest", "x-nitric-stack": "atest-deploy"}
			assert.Equal(t, expectTags, tags, "money has the wrong tags %s!=%s", expectTags, tags)
			wg.Done()
			return nil
		})

		wg.Add(1)
		a.queueTopics["checkout"].Name.ApplyT(func(name string) error {
			assert.Equal(t, "checkout", name, "queueTopic has the wrong name %s!=%s", "checkout", name)
			wg.Done()
			return nil
		})

		wg.Add(1)
		a.queueSubscriptions["checkout"].Name.ApplyT(func(name string) error {
			assert.Equal(t, "checkout-sub", name, "queueSubscription has the wrong name %s!=%s", "checkout-sub", name)
			wg.Done()
			return nil
		})

		wg.Add(1)
		a.cloudRunners["runner"].Service.Name.ApplyT(func(name string) error {
			assert.Equal(t, "runner", name, "cloudRunner has the wrong name %s!=%s", "runner", name)
			wg.Done()
			return nil
		})

		wg.Add(1)
		a.cloudRunners["runner"].Service.Template.Spec().Containers().Index(pulumi.Int(0)).ApplyT(func(c cloudrun.ServiceTemplateSpecContainer) error {
			assert.Equal(t, 9001, *c.Ports[0].ContainerPort)
			assert.Equal(t, "docker.io/nitrictech/runner:latest", c.Image)
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
		t       *stack.Config
		wantErr bool
	}{
		{
			name: "valid",
			t: &stack.Config{
				Provider: stack.Gcp,
				Region:   "us-west4",
				Extra: map[string]interface{}{
					"project": "foo",
				},
			},
		},
		{
			name:    "invalid",
			t:       &stack.Config{Provider: stack.Gcp, Region: "pole-north-right-next-to-santa"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(nil, tt.t, map[string]string{})
			if err := a.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("gcpProvider.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_gcpProvider_Plugins(t *testing.T) {
	want := []common.Plugin{
		{Name: "gcp", Version: "v6.12.0"},
		{Name: "random", Version: "v4.4.2"},
	}
	got := (&gcpProvider{}).Plugins()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("gcpProvider.Plugins() = %v, want %v", got, want)
	}
}
