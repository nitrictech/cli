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
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/getkin/kin-openapi/openapi2conv"
	"github.com/golangci/golangci-lint/pkg/sliceutil"
	"github.com/pkg/errors"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/cloudscheduler"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/organizations"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/pubsub"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/storage"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/nitrictech/cli/pkg/provider/pulumi/common"
	"github.com/nitrictech/cli/pkg/stack"
	"github.com/nitrictech/cli/pkg/target"
	"github.com/nitrictech/cli/pkg/utils"
)

type gcpProvider struct {
	s          *stack.Stack
	t          *target.Target
	tmpDir     string
	gcpProject string

	token         *oauth2.Token
	projectNumber string
	projectId     string

	buckets            map[string]*storage.Bucket
	topics             map[string]*pubsub.Topic
	queueTopics        map[string]*pubsub.Topic
	queueSubscriptions map[string]*pubsub.Subscription
	images             map[string]*common.Image
	cloudRunners       map[string]*CloudRunner
}

//go:embed pulumi-gcp-version.txt
var gcpPluginVersion string

func New(s *stack.Stack, t *target.Target) common.PulumiProvider {
	return &gcpProvider{
		s:                  s,
		t:                  t,
		buckets:            map[string]*storage.Bucket{},
		topics:             map[string]*pubsub.Topic{},
		queueTopics:        map[string]*pubsub.Topic{},
		queueSubscriptions: map[string]*pubsub.Subscription{},
		images:             map[string]*common.Image{},
		cloudRunners:       map[string]*CloudRunner{},
	}
}

func (g *gcpProvider) Plugins() []common.Plugin {
	return []common.Plugin{
		{
			Name:    "gcp",
			Version: strings.TrimSpace(gcpPluginVersion),
		},
	}
}

func (g *gcpProvider) SupportedRegions() []string {
	return []string{
		"us-west2",
		"us-west3",
		"us-west4",
		"us-central1",
		"us-east1",
		"us-east4",
		"europe-west1",
		"europe-west2",
		"asia-east1",
		"australia-southeast1",
	}
}

func (g *gcpProvider) Validate() error {
	errList := utils.NewErrorList()

	if g.t.Region == "" {
		errList.Add(fmt.Errorf("target %s requires \"region\"", g.t.Provider))
	} else if !sliceutil.Contains(g.SupportedRegions(), g.t.Region) {
		errList.Add(utils.NewNotSupportedErr(fmt.Sprintf("region %s not supported on provider %s", g.t.Region, g.t.Provider)))
	}

	if _, ok := g.t.Extra["project"]; !ok {
		errList.Add(fmt.Errorf("target %s requires GCP \"project\"", g.t.Provider))
	} else {
		g.gcpProject = g.t.Extra["project"].(string)
	}

	return errList.Aggregate()
}

func (g *gcpProvider) Configure(ctx context.Context, autoStack *auto.Stack) error {
	err := autoStack.SetConfig(ctx, "gcp:region", auto.ConfigValue{Value: g.t.Region})
	if err != nil {
		return err
	}
	return autoStack.SetConfig(ctx, "gcp:project", auto.ConfigValue{Value: g.gcpProject})
}

func (g *gcpProvider) Deploy(ctx *pulumi.Context) error {
	var err error
	g.tmpDir, err = ioutil.TempDir("", ctx.Stack()+"-*")
	if err != nil {
		return err
	}

	if g.token == nil { // for unit testing
		creds, err := google.FindDefaultCredentialsWithParams(context.Background(), google.CredentialsParams{
			Scopes: []string{
				"https://www.googleapis.com/auth/cloud-platform",
			}})
		if err != nil {
			return errors.WithMessage(err, "Unable to find credentials, try 'gcloud auth application-default login'")
		}

		g.token, err = creds.TokenSource.Token()
		if err != nil {
			return errors.WithMessage(err, "Unable to acquire token source")
		}
	}

	if g.projectId == "" {
		project, err := organizations.LookupProject(ctx, &organizations.LookupProjectArgs{
			ProjectId: &g.gcpProject,
		}, nil)
		if err != nil {
			return err
		}
		g.projectId = *project.ProjectId
		g.projectNumber = project.Number
	}

	nitricProj, err := newProject(ctx, "project", &ProjectArgs{
		ProjectId:     g.projectId,
		ProjectNumber: g.projectNumber,
	})
	if err != nil {
		return err
	}

	defaultResourceOptions := pulumi.DependsOn([]pulumi.Resource{nitricProj})

	for key := range g.s.Buckets {
		g.buckets[key], err = storage.NewBucket(ctx, key, &storage.BucketArgs{
			Location: pulumi.String(g.t.Region),
			Project:  pulumi.String(g.projectId),
			Labels:   common.Tags(ctx, key),
		}, defaultResourceOptions)
		if err != nil {
			return err
		}
	}

	for key := range g.s.Topics {
		g.topics[key], err = pubsub.NewTopic(ctx, key, &pubsub.TopicArgs{
			Name:   pulumi.String(key),
			Labels: common.Tags(ctx, key),
		}, defaultResourceOptions)
		if err != nil {
			return err
		}
	}

	for key := range g.s.Queues {
		g.queueTopics[key], err = pubsub.NewTopic(ctx, key, &pubsub.TopicArgs{
			Name:   pulumi.String(key),
			Labels: common.Tags(ctx, key),
		}, defaultResourceOptions)
		if err != nil {
			return err
		}

		g.queueSubscriptions[key], err = pubsub.NewSubscription(ctx, key+"-sub", &pubsub.SubscriptionArgs{
			Name:  pulumi.Sprintf("%s-nitricqueue", key),
			Topic: g.queueTopics[key].Name,
		}, defaultResourceOptions)
		if err != nil {
			return err
		}
	}

	for k, sched := range g.s.Schedules {
		if _, ok := g.topics[sched.Target.Name]; ok {
			payload := ""
			if len(sched.Event.Payload) > 0 {
				eventJSON, err := json.Marshal(sched.Event.Payload)
				if err != nil {
					return err
				}
				payload = base64.StdEncoding.EncodeToString(eventJSON)
			}

			_, err = cloudscheduler.NewJob(ctx, k, &cloudscheduler.JobArgs{
				TimeZone: pulumi.String("UTC"),
				PubsubTarget: cloudscheduler.JobPubsubTargetArgs{
					TopicName: pulumi.Sprintf("projects/%s/topics/%s", g.projectId, g.topics[sched.Target.Name].Name),
					Data:      pulumi.String(payload),
				},
				Schedule: pulumi.String(strings.ReplaceAll(sched.Expression, "'", "")),
			}, defaultResourceOptions)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("topic %s defined as target for schedule, but does not exist in the stack", sched.Target.Name)
		}
	}

	for _, c := range g.s.Computes() {
		if _, ok := g.images[c.Unit().Name]; !ok {
			g.images[c.Unit().Name], err = common.NewImage(ctx, c.Unit().Name+"Image", &common.ImageArgs{
				LocalImageName:  c.ImageTagName(g.s, ""),
				SourceImageName: c.ImageTagName(g.s, g.t.Provider),
				RepositoryUrl:   pulumi.Sprintf("gcr.io/%s/%s", g.projectId, c.ImageTagName(g.s, g.t.Provider)),
				Username:        pulumi.String("oauth2accesstoken"),
				Password:        pulumi.String(g.token.AccessToken),
				Server:          pulumi.String("https://gcr.io"),
				TempDir:         g.tmpDir,
			}, defaultResourceOptions)
			if err != nil {
				return errors.WithMessage(err, "function image tag "+c.Unit().Name)
			}
		}

		g.cloudRunners[c.Unit().Name], err = g.newCloudRunner(ctx, c.Unit().Name, &CloudRunnerArgs{
			Location:  pulumi.String(g.t.Region),
			ProjectId: g.projectId,
			Topics:    g.topics,
			Compute:   c,
			Image:     g.images[c.Unit().Name],
		}, defaultResourceOptions)
		if err != nil {
			return err
		}
	}

	for k, doc := range g.s.ApiDocs {
		v2doc, err := openapi2conv.FromV3(doc)
		if err != nil {
			return err
		}
		_, err = newApiGateway(ctx, k, &ApiGatewayArgs{
			Functions:   g.cloudRunners,
			OpenAPISpec: v2doc,
			ProjectId:   pulumi.String(g.projectId),
		}, defaultResourceOptions)
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *gcpProvider) CleanUp() {
	if g.tmpDir != "" {
		os.Remove(g.tmpDir)
	}
}
