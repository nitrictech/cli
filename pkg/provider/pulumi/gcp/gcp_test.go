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
	"sync"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/cloudrun"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/pubsub"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/secretmanager"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/storage"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"
	"golang.org/x/oauth2"

	"github.com/nitrictech/cli/pkg/project"
	"github.com/nitrictech/cli/pkg/provider/pulumi/common"
	"github.com/nitrictech/cli/pkg/provider/types"
	v1 "github.com/nitrictech/nitric/pkg/api/nitric/v1"
	"github.com/nitrictech/pulumi-docker-buildkit/sdk/v0.1.17/dockerbuildkit"
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
	p := project.New(&project.Config{Name: "atest", Dir: "."})
	p.Topics = map[string]project.Topic{"sales": {}}
	p.Buckets = map[string]project.Bucket{"money": {}}
	p.Queues = map[string]project.Queue{"checkout": {}}
	p.Functions = map[string]project.Function{
		"runnner": {
			Handler: "functions/create/main.go",
			ComputeUnit: project.ComputeUnit{
				Name:     "runner",
				Triggers: project.Triggers{Topics: []string{"sales"}},
			},
		},
	}
	p.Policies = []*v1.PolicyResource{
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
	p.Secrets = map[string]project.Secret{
		"hush": {},
	}

	projectName := p.Name
	stackName := p.Name + "-deploy"

	g := &gcpProvider{
		proj:   p,
		envMap: map[string]string{},
		sc: &gcpStackConfig{
			Name:     "deploy",
			Provider: types.Gcp,
			Region:   "mock",
			Config:   map[string]gcpFunctionConfig{},
		},
		buckets:            map[string]*storage.Bucket{},
		topics:             map[string]*pubsub.Topic{},
		queueTopics:        map[string]*pubsub.Topic{},
		queueSubscriptions: map[string]*pubsub.Subscription{},
		images:             map[string]*common.Image{},
		cloudRunners:       map[string]*CloudRunner{},
		secrets:            map[string]*secretmanager.Secret{},
	}
	g.token = &oauth2.Token{AccessToken: "testing-token"}
	g.projectID = "test-project-id"
	g.projectNumber = "test-project-number"
	g.images = map[string]*common.Image{
		"runner": {
			DockerImage: &dockerbuildkit.Image{
				Name:       pulumi.Sprintf("docker.io/nitrictech/runner:latest"),
				RepoDigest: pulumi.Sprintf("docker.io/nitrictech/runner:latest@sha:foo"),
			},
		},
	}
	g.cloudRunners = map[string]*CloudRunner{}

	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		err := g.Deploy(ctx)
		assert.NoError(t, err)

		var wg sync.WaitGroup

		wg.Add(1)
		g.topics["sales"].Name.ApplyT(func(name string) error {
			assert.Equal(t, "sales", name, "topic has the wrong name %s!=%s", "sales", name)
			wg.Done()

			return nil
		})

		wg.Add(1)
		g.topics["sales"].Labels.ApplyT(func(tags map[string]string) error {
			expectTags := map[string]string{"x-nitric-name": "sales", "x-nitric-project": "atest", "x-nitric-stack": "atest-deploy", "x-nitric-stack-name": "atest-deploy"}
			assert.Equal(t, expectTags, tags, "topic has the wrong tags %s!=%s", expectTags, tags)
			wg.Done()

			return nil
		})

		wg.Add(1)
		g.buckets["money"].Name.ApplyT(func(name string) error {
			assert.Equal(t, "money", name, "bucket has the wrong name %s!=%s", "money", name)
			wg.Done()

			return nil
		})

		wg.Add(1)
		g.buckets["money"].Labels.ApplyT(func(tags map[string]string) error {
			expectTags := map[string]string{"x-nitric-name": "money", "x-nitric-project": "atest", "x-nitric-stack": "atest-deploy", "x-nitric-stack-name": "atest-deploy"}
			assert.Equal(t, expectTags, tags, "money has the wrong tags %s!=%s", expectTags, tags)
			wg.Done()

			return nil
		})

		wg.Add(1)
		g.secrets["hush"].Labels.ApplyT(func(tags map[string]string) error {
			expectTags := map[string]string{"x-nitric-name": "hush", "x-nitric-project": "atest", "x-nitric-stack": "atest-deploy", "x-nitric-stack-name": "atest-deploy"}
			assert.Equal(t, expectTags, tags, "hush has the wrong tags %s!=%s", expectTags, tags)
			wg.Done()

			return nil
		})

		wg.Add(1)
		g.queueTopics["checkout"].Name.ApplyT(func(name string) error {
			assert.Equal(t, "checkout", name, "queueTopic has the wrong name %s!=%s", "checkout", name)
			wg.Done()

			return nil
		})

		wg.Add(1)
		g.queueSubscriptions["checkout"].Name.ApplyT(func(name string) error {
			assert.Equal(t, "checkout-sub", name, "queueSubscription has the wrong name %s!=%s", "checkout-sub", name)
			wg.Done()

			return nil
		})

		wg.Add(1)
		g.cloudRunners["runner"].Service.Name.ApplyT(func(name string) error {
			assert.Equal(t, "runner", name, "cloudRunner has the wrong name %s!=%s", "runner", name)
			wg.Done()

			return nil
		})

		wg.Add(1)
		g.cloudRunners["runner"].Service.Template.Spec().Containers().Index(pulumi.Int(0)).ApplyT(func(c cloudrun.ServiceTemplateSpecContainer) error {
			assert.Equal(t, 9001, *c.Ports[0].ContainerPort)
			assert.Equal(t, "docker.io/nitrictech/runner:latest@sha:foo", c.Image)
			wg.Done()

			return nil
		})

		wg.Wait()

		return nil
	}, pulumi.WithMocks(projectName, stackName, mocks(0)))
	assert.NoError(t, err)

	g.CleanUp()
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		sc      *gcpStackConfig
		wantErr bool
	}{
		{
			name: "valid",
			sc: &gcpStackConfig{
				Provider: types.Gcp,
				Region:   "us-west4",
				Project:  "foo",
			},
		},
		{
			name:    "invalid",
			sc:      &gcpStackConfig{Provider: types.Gcp, Region: "pole-north-right-next-to-santa"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &gcpProvider{sc: tt.sc}
			if err := a.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("gcpProvider.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_gcpProvider_Plugins(t *testing.T) {
	want := []string{"gcp", "random"}

	got := (&gcpProvider{}).Plugins()

	for _, pl := range got {
		_, err := version.NewVersion(pl.Version)
		if err != nil {
			t.Error(err)
		}

		if !slices.Contains(want, pl.Name) {
			t.Errorf("gcpProvider.Plugins() = %v not in want %v", pl, want)
		}
	}
}
