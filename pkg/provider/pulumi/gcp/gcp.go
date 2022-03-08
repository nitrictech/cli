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
	"crypto/md5"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/getkin/kin-openapi/openapi2conv"
	"github.com/golangci/golangci-lint/pkg/sliceutil"
	"github.com/pkg/errors"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/cloudscheduler"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/firestore"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/organizations"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/pubsub"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/secretmanager"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/storage"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/nitrictech/cli/pkg/project"
	"github.com/nitrictech/cli/pkg/provider/pulumi/common"
	"github.com/nitrictech/cli/pkg/stack"
	"github.com/nitrictech/cli/pkg/utils"
	v1 "github.com/nitrictech/nitric/pkg/api/nitric/v1"
)

type gcpProvider struct {
	sc         *stack.Config
	proj       *project.Project
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

func New(s *project.Project, t *stack.Config) common.PulumiProvider {
	return &gcpProvider{
		proj:               s,
		sc:                 t,
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

func md5Hash(b []byte) string {
	hasher := md5.New()
	hasher.Write(b)
	return hex.EncodeToString(hasher.Sum(nil))
}

func policyResourceName(policy *v1.PolicyResource) (string, error) {
	policyDoc, err := json.Marshal(policy)
	if err != nil {
		return "", err
	}

	return md5Hash(policyDoc), nil
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

func (a *gcpProvider) Ask() (*stack.Config, error) {
	answers := struct {
		Region  string
		Project string
	}{}
	qs := []*survey.Question{
		{
			Name: "region",
			Prompt: &survey.Select{
				Message: "select the region",
				Options: a.SupportedRegions(),
			},
		},
		{
			Name: "project",
			Prompt: &survey.Input{
				Message: "Provide the gcp project to use",
			},
		},
	}
	sc := &stack.Config{
		Name:     a.sc.Name,
		Provider: a.sc.Provider,
		Extra:    map[string]interface{}{},
	}

	err := survey.Ask(qs, &answers)
	if err != nil {
		return nil, err
	}

	sc.Region = answers.Region
	sc.Extra["project"] = answers.Project

	return sc, nil
}

func (g *gcpProvider) Validate() error {
	errList := utils.NewErrorList()

	if g.sc.Region == "" {
		errList.Add(fmt.Errorf("target %s requires \"region\"", g.sc.Provider))
	} else if !sliceutil.Contains(g.SupportedRegions(), g.sc.Region) {
		errList.Add(utils.NewNotSupportedErr(fmt.Sprintf("region %s not supported on provider %s", g.sc.Region, g.sc.Provider)))
	}

	if _, ok := g.sc.Extra["project"]; !ok {
		errList.Add(fmt.Errorf("target %s requires GCP \"project\"", g.sc.Provider))
	} else {
		g.gcpProject = g.sc.Extra["project"].(string)
	}

	return errList.Aggregate()
}

func (g *gcpProvider) Configure(ctx context.Context, autoStack *auto.Stack) error {
	err := autoStack.SetConfig(ctx, "gcp:region", auto.ConfigValue{Value: g.sc.Region})
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

	for key := range g.proj.Buckets {
		g.buckets[key], err = storage.NewBucket(ctx, key, &storage.BucketArgs{
			Location: pulumi.String(g.sc.Region),
			Project:  pulumi.String(g.projectId),
			Labels:   common.Tags(ctx, key),
		}, defaultResourceOptions)
		if err != nil {
			return err
		}
	}

	for key := range g.proj.Topics {
		g.topics[key], err = pubsub.NewTopic(ctx, key, &pubsub.TopicArgs{
			Name:   pulumi.String(key),
			Labels: common.Tags(ctx, key),
		}, defaultResourceOptions)
		if err != nil {
			return err
		}
	}

	for key := range g.proj.Queues {
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

	for k, sched := range g.proj.Schedules {
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

	principalMap := make(PrincipalMap, len(g.proj.Computes()))

	for _, c := range g.proj.Computes() {
		if _, ok := g.images[c.Unit().Name]; !ok {
			g.images[c.Unit().Name], err = common.NewImage(ctx, c.Unit().Name+"Image", &common.ImageArgs{
				LocalImageName:  c.ImageTagName(g.proj, ""),
				SourceImageName: c.ImageTagName(g.proj, g.sc.Provider),
				RepositoryUrl:   pulumi.Sprintf("gcr.io/%s/%s", g.projectId, c.ImageTagName(g.proj, g.sc.Provider)),
				Username:        pulumi.String("oauth2accesstoken"),
				Password:        pulumi.String(g.token.AccessToken),
				Server:          pulumi.String("https://gcr.io"),
				TempDir:         g.tmpDir,
			}, defaultResourceOptions)
			if err != nil {
				return errors.WithMessage(err, "function image tag "+c.Unit().Name)
			}
		}
		// Create a service account for this cloud run instance
		sa, err := serviceaccount.NewAccount(ctx, c.Unit().Name+"-acct", &serviceaccount.AccountArgs{
			AccountId: pulumi.String(utils.StringTrunc(c.Unit().Name, 30-5) + "-acct"),
		})
		if err != nil {
			return errors.WithMessage(err, "function serviceaccount "+c.Unit().Name)
		}

		g.cloudRunners[c.Unit().Name], err = g.newCloudRunner(ctx, c.Unit().Name, &CloudRunnerArgs{
			Location:       pulumi.String(g.sc.Region),
			ProjectId:      g.projectId,
			Topics:         g.topics,
			Compute:        c,
			Image:          g.images[c.Unit().Name],
			ServiceAccount: sa,
		}, defaultResourceOptions)
		if err != nil {
			return err
		}

		principalMap[v1.ResourceType_Function][c.Unit().Name] = sa
	}

	for k, doc := range g.proj.ApiDocs {
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

	for _, p := range g.proj.Policies {
		if len(p.Actions) == 0 {
			_ = ctx.Log.Debug("policy has no actions "+fmt.Sprint(p), &pulumi.LogArgs{Ephemeral: true})
		}
		policyName, err := policyResourceName(p)
		if err != nil {
			return err
		}
		if _, err := newPolicy(ctx, policyName, &PolicyArgs{
			Policy: p,
			Resources: &StackResources{
				Topics:      g.topics,
				Queues:      g.queueTopics,
				Buckets:     g.buckets,
				Collections: map[string]*firestore.Document{},
				Secrets:     map[string]*secretmanager.Secret{},
			},
			Principals: principalMap,
		}); err != nil {
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
