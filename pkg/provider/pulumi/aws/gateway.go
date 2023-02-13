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
	"net/url"
	"path"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/apigatewayv2"
	awslambda "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/lambda"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/segmentio/encoding/json"

	"github.com/nitrictech/cli/pkg/provider/pulumi/common"
	v1 "github.com/nitrictech/nitric/core/pkg/api/nitric/v1"
)

type ApiGatewayArgs struct {
	SecurityDefintions map[string]*v1.ApiSecurityDefinition
	OpenAPISpec        *openapi3.T
	LambdaFunctions    map[string]*Lambda
	StackID            pulumi.StringInput
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

	opts = append(opts, pulumi.Parent(res))

	nameArnPairs := make([]interface{}, 0, len(args.LambdaFunctions))

	// augment open api spec with security definitions
	for sn, sd := range args.SecurityDefintions {
		if args.OpenAPISpec.Components.SecuritySchemes == nil {
			args.OpenAPISpec.Components.SecuritySchemes = make(openapi3.SecuritySchemes)
		}

		// if it's a JWT security definition

		if sd.GetJwt() != nil {
			issuerUrl, err := url.Parse(sd.GetJwt().GetIssuer())
			if err != nil {
				return nil, err
			}

			issuerUrl.Path = path.Join(issuerUrl.Path, ".well-known/openid-configuration")

			args.OpenAPISpec.Components.SecuritySchemes[sn] = &openapi3.SecuritySchemeRef{
				Value: &openapi3.SecurityScheme{
					Type:             "openIdConnect",
					OpenIdConnectUrl: issuerUrl.String(),
					ExtensionProps: openapi3.ExtensionProps{
						Extensions: map[string]interface{}{
							"x-amazon-apigateway-authorizer": map[string]interface{}{
								"type": "jwt",
								"jwtConfiguration": map[string]interface{}{
									"audience": sd.GetJwt().Audiences,
								},
								"identitySource": "$request.header.Authorization",
							},
						},
					},
				},
			}
		} else {
			return nil, fmt.Errorf("unsupported security definition supplied")
		}
	}

	// collect name arn pairs for output iteration
	for k, v := range args.LambdaFunctions {
		nameArnPairs = append(nameArnPairs, pulumi.All(k, v.Function.InvokeArn).ApplyT(func(args []interface{}) nameArnPair {
			name := args[0].(string)
			arn := args[1].(string)

			return nameArnPair{
				name:      name,
				invokeArn: arn,
			}
		}))
	}

	doc := pulumi.All(nameArnPairs...).ApplyT(func(pairs []interface{}) (string, error) {
		naps := make(map[string]string)

		for _, p := range pairs {
			if pair, ok := p.(nameArnPair); ok {
				naps[pair.name] = pair.invokeArn
			} else {
				// XXX: Should not occur
				return "", fmt.Errorf("invalid data %T %v", p, p)
			}
		}

		for k, p := range args.OpenAPISpec.Paths {
			p.Get = awsOperation(p.Get, naps)
			p.Post = awsOperation(p.Post, naps)
			p.Patch = awsOperation(p.Patch, naps)
			p.Put = awsOperation(p.Put, naps)
			p.Delete = awsOperation(p.Delete, naps)
			p.Options = awsOperation(p.Options, naps)
			args.OpenAPISpec.Paths[k] = p
		}

		// augment the api specs with security definitions where available
		b, err := json.Marshal(args.OpenAPISpec)
		if err != nil {
			return "", err
		}

		return string(b), nil
	}).(pulumi.StringOutput)

	res.Api, err = apigatewayv2.NewApi(ctx, name, &apigatewayv2.ApiArgs{
		Body:         doc,
		ProtocolType: pulumi.String("HTTP"),
		Tags:         common.Tags(ctx, args.StackID, name),
	}, opts...)
	if err != nil {
		return nil, err
	}

	_, err = apigatewayv2.NewStage(ctx, name+"DefaultStage", &apigatewayv2.StageArgs{
		AutoDeploy: pulumi.BoolPtr(true),
		Name:       pulumi.String("$default"),
		ApiId:      res.Api.ID(),
		Tags:       common.Tags(ctx, args.StackID, name+"DefaultStage"),
	}, opts...)
	if err != nil {
		return nil, err
	}

	// Generate lambda permissions enabling the API Gateway to invoke the functions it targets
	for fName, fun := range args.LambdaFunctions {
		_, err = awslambda.NewPermission(ctx, name+fName, &awslambda.PermissionArgs{
			Function:  fun.Function.Name,
			Action:    pulumi.String("lambda:InvokeFunction"),
			Principal: pulumi.String("apigateway.amazonaws.com"),
			SourceArn: pulumi.Sprintf("%s/*/*/*", res.Api.ExecutionArn),
		}, opts...)
		if err != nil {
			return nil, err
		}
	}

	endPoint := res.Api.ApiEndpoint.ApplyT(func(ep string) string {
		return ep
	}).(pulumi.StringInput)

	ctx.Export("api:"+name, endPoint)

	return res, nil
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
