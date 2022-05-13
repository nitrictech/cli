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

package azure

import (
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	v1 "github.com/nitrictech/nitric/pkg/api/nitric/v1"
	"github.com/pkg/errors"
	apimanagement "github.com/pulumi/pulumi-azure-native/sdk/go/azure/apimanagement/v20201201"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	v1 "github.com/nitrictech/nitric/pkg/api/nitric/v1"
)

type AzureApiManagementArgs struct {
	ResourceGroupName   pulumi.StringInput
	OrgName             pulumi.StringInput
	AdminEmail          pulumi.StringInput
	OpenAPISpec         *openapi3.T
	Apps                map[string]*ContainerApp
	SecurityDefinitions map[string]*v1.ApiSecurityDefinition
}

type AzureApiManagement struct {
	pulumi.ResourceState

	Name    string
	Api     *apimanagement.Api
	Service *apimanagement.ApiManagementService
}

const policyTemplate = `<policies><inbound><base /><set-backend-service base-url="https://%s" /></inbound><backend><base /></backend><outbound><base /></outbound><on-error><base /></on-error></policies>`

const oidcTemplate = `<policies><inbound><base /><validate-jwt header-name=”Authorization” failed-validation-httpcode=”401″ failed-validation-error-message=”Unauthorized. Access token is missing or invalid.”>  
<openid-config url=”%s/.well-known/openid-configuration” />  
   <required-claims>  
	  <claim name=”aud” match="any" separator=",">  
		 <value>%s</value>  
	  </claim>  
   </required-claims>  
</validate-jwt> </inbound><backend><base /></backend><outbound><base /></outbound><on-error><base /></on-error></policies>`

func newAzureApiManagement(ctx *pulumi.Context, name string, args *AzureApiManagementArgs, opts ...pulumi.ResourceOption) (*AzureApiManagement, error) {
	res := &AzureApiManagement{Name: name}
	err := ctx.RegisterComponentResource("nitric:api:AzureApiManagement", name, res, opts...)
	if err != nil {
		return nil, err
	}

	res.Service, err = apimanagement.NewApiManagementService(ctx, resourceName(ctx, name, ApiManagementRT), &apimanagement.ApiManagementServiceArgs{
		ResourceGroupName: args.ResourceGroupName,
		PublisherEmail:    args.AdminEmail,
		PublisherName:     args.OrgName,
		Sku: apimanagement.ApiManagementServiceSkuPropertiesArgs{
			Name:     pulumi.String("Consumption"),
			Capacity: pulumi.Int(0),
		},
	})
	if err != nil {
		return nil, err
	}

	displayName := name + "-api"
	if args.OpenAPISpec.Info != nil && args.OpenAPISpec.Info.Title != "" {
		displayName = args.OpenAPISpec.Info.Title
	}
	b, err := args.OpenAPISpec.MarshalJSON()
	if err != nil {
		return nil, err
	}

	res.Api, err = apimanagement.NewApi(ctx, resourceName(ctx, name, ApiRT), &apimanagement.ApiArgs{
		DisplayName:          pulumi.String(displayName),
		Protocols:            apimanagement.ProtocolArray{"https"},
		ApiId:                pulumi.String(name),
		Format:               pulumi.String("openapi+json"),
		Path:                 pulumi.String("/"),
		ResourceGroupName:    args.ResourceGroupName,
		SubscriptionRequired: pulumi.Bool(false),
		ServiceName:          res.Service.Name,
		// XXX: Do we need to stringify this?
		// Not need to transform the original spec,
		// the mapping occurs as part of the operation policies below
		Value: pulumi.String(string(b)),
	})
	if err != nil {
		return nil, err
	}

	ctx.Export("api:"+name, res.Api.ServiceUrl)

	for _, pathItem := range args.OpenAPISpec.Paths {
		for _, op := range pathItem.Operations() {
			if v, ok := op.Extensions["x-nitric-target"]; ok {
				target := ""
				targetMap, isMap := v.(map[string]string)
				if !isMap {
					continue
				}
				target = targetMap["name"]
				app, ok := args.Apps[target]
				if !ok {
					continue
				}

				// this.api.id returns a URL path, which is the incorrect value here.
				//   We instead need the value passed to apiId in the api creation above.
				// However, we want to maintain the pulumi dependency, so we need to keep the 'apply' call.
				apiId := res.Api.ID().ToStringOutput().ApplyT(func(id string) string {
					return name
				}).(pulumi.StringOutput)

				_ = ctx.Log.Info("op policy "+op.OperationID+" , name "+name, &pulumi.LogArgs{Ephemeral: true})

				_, err = apimanagement.NewApiOperationPolicy(ctx, resourceName(ctx, name+"-"+op.OperationID, ApiOperationPolicyRT), &apimanagement.ApiOperationPolicyArgs{
					ResourceGroupName: args.ResourceGroupName,
					ApiId:             apiId,
					ServiceName:       res.Service.Name,
					OperationId:       pulumi.String(op.OperationID),
					PolicyId:          pulumi.String("policy"),
					Format:            pulumi.String("xml"),
					Value:             pulumi.Sprintf(policyTemplate, app.App.LatestRevisionFqdn),
				})

				// Add an api operation policy if we have a security definition available
				if sec, ok := op.Extensions["x-nitric-security"]; ok {
					if secName, ok := sec.(string); ok {
						sd := args.SecurityDefinitions[secName]

						apimanagement.NewApiOperationPolicy(ctx, resourceName(ctx, name+"-"+op.OperationID+"-sec", ApiOperationPolicyRT), &apimanagement.ApiOperationPolicyArgs{
							ResourceGroupName: args.ResourceGroupName,
							ApiId:             apiId,
							ServiceName:       res.Service.Name,
							OperationId:       pulumi.String(op.OperationID),
							PolicyId:          pulumi.String("policy"),
							Format:            pulumi.String("xml"),
							Value:             pulumi.Sprintf(oidcTemplate, sd.GetJwt().Issuer, strings.Join(sd.GetJwt().Audiences, ",")),
						})
					}
				}

				if err != nil {
					return nil, errors.WithMessage(err, "NewApiOperationPolicy "+op.OperationID)
				}
			}
		}
	}

	return res, ctx.RegisterResourceOutputs(res, pulumi.Map{
		"name":    pulumi.String(name),
		"service": res.Service,
		"api":     res.Api,
	})
}
