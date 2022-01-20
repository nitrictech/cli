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

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/apigatewayv2"
	awslambda "github.com/pulumi/pulumi-aws/sdk/v4/go/aws/lambda"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type ApiGatewayArgs struct {
	OpenAPISpec     *openapi3.T
	LambdaFunctions map[string]*Lambda
}

type ApiGateway struct {
	pulumi.ResourceState

	Name string
	Api  *apigatewayv2.Api
}

type nameArnPair struct {
	name      string
	invokeArn string
}

func newApiGateway(ctx *pulumi.Context, name string, args *ApiGatewayArgs, opts ...pulumi.ResourceOption) (*ApiGateway, error) {
	res := &ApiGateway{Name: name}
	err := ctx.RegisterComponentResource("nitric:api:AwsApiGateway", name, res, opts...)
	if err != nil {
		return nil, err
	}

	nameArnPairs := make([]interface{}, 0, len(args.LambdaFunctions))

	// collect name arn pairs for output iteration
	for k, v := range args.LambdaFunctions {
		nameArnPairs = append(nameArnPairs, v.Function.InvokeArn.ApplyT(func(arn string) nameArnPair {
			return nameArnPair{
				name:      k,
				invokeArn: arn,
			}
		}))
	}

	doc := pulumi.All(nameArnPairs...).ApplyT(func(pairs []interface{}) (string, error) {
		naps := make(map[string]string)

		for _, p := range pairs {
			fmt.Printf("Underlying Type: %T\n", p)
			if pair, ok := p.(nameArnPair); ok {
				naps[pair.name] = pair.invokeArn
			} else {
				// XXX: Should not occurr
				return "", fmt.Errorf("invalid data")
			}
		}

		for k, p := range args.OpenAPISpec.Paths {
			p.Get = awsOperation(p.Get, naps)
			p.Post = awsOperation(p.Post, naps)
			p.Patch = awsOperation(p.Patch, naps)
			p.Put = awsOperation(p.Put, naps)
			p.Delete = awsOperation(p.Delete, naps)
			args.OpenAPISpec.Paths[k] = p
		}

		b, err := args.OpenAPISpec.MarshalJSON()

		if err != nil {
			return "", err
		}

		return string(b), nil
	}).(pulumi.StringOutput)

	if err != nil {
		return nil, err
	}

	res.Api, err = apigatewayv2.NewApi(ctx, name, &apigatewayv2.ApiArgs{
		Body:         doc,
		ProtocolType: pulumi.String("HTTP"),
		Tags:         commonTags(ctx, name),
	}, pulumi.Parent(res))
	if err != nil {
		return nil, err
	}

	_, err = apigatewayv2.NewStage(ctx, name+"DefaultStage", &apigatewayv2.StageArgs{
		AutoDeploy: pulumi.BoolPtr(true),
		Name:       pulumi.String("$default"),
		ApiId:      res.Api.ID(),
		Tags:       commonTags(ctx, name+"DefaultStage"),
	}, pulumi.Parent(res))
	if err != nil {
		return nil, err
	}

	// Generate lambda permissions enabling the API Gateway to invoke the functions it targets
	for fName, fun := range args.LambdaFunctions {
		_, err = awslambda.NewPermission(ctx, name+fName, &awslambda.PermissionArgs{
			Function:  fun.Function.Name,
			Action:    pulumi.String("lambda:InvokeFunction"),
			Principal: pulumi.String("apigateway.amazonaws.com"),
			SourceArn: res.Api.ExecutionArn,
		}, pulumi.Parent(res))
		if err != nil {
			return nil, err
		}
	}

	return res, ctx.RegisterResourceOutputs(res, pulumi.Map{
		"name": pulumi.String(name),
		"api":  res.Api,
	})
}

func awsOperation(op *openapi3.Operation, funcs map[string]string) *openapi3.Operation {
	if op == nil {
		return nil
	}
	name := ""
	if v, ok := op.Extensions["x-nitric-target"]; ok {
		targetMap, isMap := v.(map[string]string)
		if isMap {
			name = targetMap["name"]
		}
	}
	if name == "" {
		return nil
	}
	if _, ok := funcs[name]; !ok {
		return nil
	}

	arn := funcs[name]
	op.Extensions["x-amazon-apigateway-integration"] = map[string]string{
		"type":                 "aws_proxy",
		"httpMethod":           "POST",
		"payloadFormatVersion": "2.0",
		// TODO: This might cause some trouble
		// Need to determine if the body of the..
		"uri": arn,
	}
	return op
}
