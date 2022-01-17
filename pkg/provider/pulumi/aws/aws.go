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
	"context"
	"io/ioutil"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/dynamodb"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/ecr"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/s3"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/sns"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/sqs"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/nitrictech/newcli/pkg/provider/pulumi/types"
	"github.com/nitrictech/newcli/pkg/stack"
	"github.com/nitrictech/newcli/pkg/target"
	"github.com/nitrictech/newcli/pkg/utils"
)

type awsProvider struct {
	s      *stack.Stack
	t      *target.Target
	tmpDir string
}

func New(s *stack.Stack, t *target.Target) types.PulumiProvider {
	return &awsProvider{s: s, t: t}
}

func (a *awsProvider) PluginName() string {
	return "aws"
}

func (a *awsProvider) PluginVersion() string {
	return "v4.0.0"
}

func (a *awsProvider) Configure(ctx context.Context, autoStack *auto.Stack) error {
	if a.t.Region != "" {
		return autoStack.SetConfig(ctx, "aws:region", auto.ConfigValue{Value: a.t.Region})
	}
	return nil
}

func commonTags(ctx *pulumi.Context, name string) pulumi.StringMap {
	return pulumi.StringMap{
		"x-nitric-project":       pulumi.String(ctx.Project()),
		"x-nitric-stack":         pulumi.String(ctx.Stack()),
		"x-nitric-resource-name": pulumi.String(name),
	}
}

func (a *awsProvider) Deploy(ctx *pulumi.Context) error {
	var err error
	a.tmpDir, err = ioutil.TempDir("", ctx.Stack()+"-*")
	if err != nil {
		return err
	}

	topics := map[string]*sns.Topic{}
	for k := range a.s.Topics {
		topics[k], err = sns.NewTopic(ctx, k, &sns.TopicArgs{Tags: commonTags(ctx, k)})
		if err != nil {
			return errors.WithMessage(err, "sns topic "+k)
		}
	}

	buckets := map[string]*s3.Bucket{}
	for k := range a.s.Buckets {
		buckets[k], err = s3.NewBucket(ctx, k, &s3.BucketArgs{
			Tags: commonTags(ctx, k),
		})
		if err != nil {
			return errors.WithMessage(err, "s3 bucket "+k)
		}
	}

	queues := map[string]*sqs.Queue{}
	for k := range a.s.Queues {
		queues[k], err = sqs.NewQueue(ctx, k, &sqs.QueueArgs{
			Tags: commonTags(ctx, k),
		})
		if err != nil {
			return errors.WithMessage(err, "sqs queue "+k)
		}
	}

	dbs := map[string]*dynamodb.Table{}
	for k := range a.s.Collections {
		dbs[k], err = dynamodb.NewTable(ctx, "mytable", &dynamodb.TableArgs{
			Attributes: dynamodb.TableAttributeArray{
				&dynamodb.TableAttributeArgs{
					Name: pulumi.String("_pk"),
					Type: pulumi.String("S"),
				},
				&dynamodb.TableAttributeArgs{
					Name: pulumi.String("_sk"),
					Type: pulumi.String("S"),
				},
			},
			HashKey:     pulumi.String("_pk"),
			RangeKey:    pulumi.String("_sk"),
			BillingMode: pulumi.String("PAY_PER_REQUEST"),
			Tags:        commonTags(ctx, k),
		})
		if err != nil {
			return errors.WithMessage(err, "dynamodb table "+k)
		}
	}

	for k, s := range a.s.Schedules {
		if len(topics) > 0 && s.Target.Type == "topic" && s.Target.Name != "" {
			err := a.schedule(ctx, k, s.Expression, topics[s.Target.Name])
			if err != nil {
				return errors.WithMessage(err, "schedule "+k)
			}
		}
	}

	for k, s := range a.s.Sites {
		err := a.site(ctx, k, &s)
		if err != nil {
			return errors.WithMessage(err, "site "+k)
		}
	}

	authToken, err := ecr.GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenArgs{})
	if err != nil {
		return err
	}

	funcs := map[string]*Lambda{}
	for k, f := range a.s.Functions {
		image, err := newECRImage(ctx, f.Name(), &ECRImageArgs{
			LocalImageName:  f.ImageTagName(a.s, ""),
			SourceImageName: f.ImageTagName(a.s, a.t.Provider),
			AuthToken:       authToken,
			TempDir:         a.tmpDir})
		if err != nil {
			return errors.WithMessage(err, "function image tag "+f.Name())
		}
		funcs[k], err = newLambda(ctx, k, &LambdaArgs{
			Topics:      topics,
			DockerImage: image.DockerImage,
			Compute:     &f,
		})
		if err != nil {
			return errors.WithMessage(err, "lambda function "+f.Name())
		}
	}

	for k, c := range a.s.Containers {
		image, err := newECRImage(ctx, c.Name(), &ECRImageArgs{
			LocalImageName:  c.ImageTagName(a.s, ""),
			SourceImageName: c.ImageTagName(a.s, a.t.Provider),
			AuthToken:       authToken,
			TempDir:         a.tmpDir})
		if err != nil {
			return errors.WithMessage(err, "function image tag "+c.Name())
		}
		funcs[k], err = newLambda(ctx, k, &LambdaArgs{
			Topics:      topics,
			DockerImage: image.DockerImage,
			Compute:     &c,
		})
		if err != nil {
			return errors.WithMessage(err, "lambda container "+c.Name())
		}
	}
	apiGateways := map[string]*ApiGateway{}
	for k, apiFile := range a.s.Apis {
		apiGateways[k], err = newApiGateway(ctx, k, &ApiGatewayArgs{
			ApiFilePath:     path.Join(a.s.Path(), apiFile),
			LambdaFunctions: funcs})
		if err != nil {
			return errors.WithMessage(err, "gateway "+k)
		}
	}

	for k, v := range a.s.EntryPoints {
		err = a.entrypoint(ctx, k, &v)
		if err != nil {
			return errors.WithMessage(err, "entrypoint "+k)
		}
	}

	return nil
}

func (a *awsProvider) CleanUp() {
	if a.tmpDir != "" {
		os.Remove(a.tmpDir)
	}
}

func (a *awsProvider) site(ctx *pulumi.Context, name string, o interface{}) error {
	return utils.NewNotSupportedErr("site not supported on AWS yet")
}

func (a *awsProvider) entrypoint(ctx *pulumi.Context, name string, o interface{}) error {
	return utils.NewNotSupportedErr("entrypoint not supported on AWS yet")
}
