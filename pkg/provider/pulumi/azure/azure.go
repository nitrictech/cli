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
	"context"
	_ "embed"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/golangci/golangci-lint/pkg/sliceutil"
	"github.com/pkg/errors"
	"github.com/pulumi/pulumi-azure/sdk/v4/go/azure/core"
	"github.com/pulumi/pulumi-azure/sdk/v4/go/azure/eventgrid"
	"github.com/pulumi/pulumi-azure/sdk/v4/go/azure/keyvault"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/nitrictech/cli/pkg/provider/pulumi/common"
	"github.com/nitrictech/cli/pkg/stack"
	"github.com/nitrictech/cli/pkg/target"
	"github.com/nitrictech/cli/pkg/utils"
)

type azureProvider struct {
	s          *stack.Stack
	t          *target.Target
	tmpDir     string
	org        string
	adminEmail string
}

var (
	//go:embed pulumi-azure-version.txt
	azurePluginVersion string
	//go:embed pulumi-azuread-version.txt
	azureADPluginVersion string
	//go:embed pulumi-azure-native-version.txt
	azureNativePluginVersion string
)

func New(s *stack.Stack, t *target.Target) common.PulumiProvider {
	return &azureProvider{s: s, t: t}
}

func (a *azureProvider) Plugins() []common.Plugin {
	return []common.Plugin{
		{
			Name:    "azure-native",
			Version: strings.TrimSpace(azureNativePluginVersion),
		},
		{
			Name:    "azure",
			Version: strings.TrimSpace(azurePluginVersion),
		},
		{
			Name:    "azuread",
			Version: strings.TrimSpace(azureADPluginVersion),
		},
	}
}

func (a *azureProvider) SupportedRegions() []string {
	return []string{
		"eastus2",
	}
}

func (a *azureProvider) Validate() error {
	errList := utils.NewErrorList()

	if a.t.Region == "" {
		errList.Add(fmt.Errorf("target %s requires \"region\"", a.t.Provider))
	} else if !sliceutil.Contains(a.SupportedRegions(), a.t.Region) {
		errList.Add(utils.NewNotSupportedErr(fmt.Sprintf("region %s not supported on provider %s", a.t.Region, a.t.Provider)))
	}

	if _, ok := a.t.Extra["org"]; !ok {
		errList.Add(fmt.Errorf("target %s requires \"org\"", a.t.Provider))
	} else {
		a.org = a.t.Extra["org"].(string)
	}

	if _, ok := a.t.Extra["adminemail"]; !ok {
		errList.Add(fmt.Errorf("target %s requires \"adminemail\"", a.t.Provider))
	} else {
		a.adminEmail = a.t.Extra["adminemail"].(string)
	}

	return errList.Aggregate()
}

func (a *azureProvider) Configure(ctx context.Context, autoStack *auto.Stack) error {
	if a.t.Region != "" {
		err := autoStack.SetConfig(ctx, "azure:location", auto.ConfigValue{Value: a.t.Region})
		if err != nil {
			return err
		}
		err = autoStack.SetConfig(ctx, "azure-native:location", auto.ConfigValue{Value: a.t.Region})
		if err != nil {
			return err
		}
		return nil
	}
	region, err := autoStack.GetConfig(ctx, "azure-native:location")
	if err != nil {
		return err
	}
	a.t.Region = region.Value
	return nil
}

func (a *azureProvider) Deploy(ctx *pulumi.Context) error {
	var err error
	a.tmpDir, err = ioutil.TempDir("", ctx.Stack()+"-*")
	if err != nil {
		return err
	}

	current, err := core.LookupSubscription(ctx, nil, nil)
	if err != nil {
		return err
	}

	rg, err := core.NewResourceGroup(ctx, resourceName(ctx, "", ResourceGroupRT), &core.ResourceGroupArgs{
		Location: pulumi.String(a.t.Region),
		Tags:     common.Tags(ctx, ctx.Stack()),
	})
	if err != nil {
		return errors.WithMessage(err, "resource group create")
	}

	contAppsArgs := &ContainerAppsArgs{
		ResourceGroupName: rg.Name,
		Location:          rg.Location,
		SubscriptionID:    pulumi.String(current.Id),
		Topics:            map[string]*eventgrid.Topic{},
	}

	// Create a stack level keyvault if secrets are enabled
	// At the moment secrets have no config level setting
	kvName := resourceName(ctx, "", KeyVaultRT)
	kv, err := keyvault.NewKeyVault(ctx, kvName, &keyvault.KeyVaultArgs{
		Location:                rg.Location,
		ResourceGroupName:       rg.Name,
		SkuName:                 pulumi.String("standard"),
		TenantId:                pulumi.String(current.TenantId),
		EnableRbacAuthorization: pulumi.Bool(true),
		Tags:                    common.Tags(ctx, kvName),
	})
	if err != nil {
		return err
	}
	contAppsArgs.KVaultName = kv.Name

	if len(a.s.Buckets) > 0 || len(a.s.Queues) > 0 {
		sr, err := a.newStorageResources(ctx, "storage", &StorageArgs{ResourceGroupName: rg.Name})
		if err != nil {
			return errors.WithMessage(err, "storage create")
		}
		contAppsArgs.StorageAccountBlobEndpoint = sr.Account.PrimaryBlobEndpoint
		contAppsArgs.StorageAccountQueueEndpoint = sr.Account.PrimaryQueueEndpoint
	}

	for k := range a.s.Topics {
		contAppsArgs.Topics[k], err = eventgrid.NewTopic(ctx, resourceName(ctx, k, EventGridRT), &eventgrid.TopicArgs{
			ResourceGroupName: rg.Name,
			Location:          rg.Location,
			Tags:              common.Tags(ctx, k),
		})
		if err != nil {
			return errors.WithMessage(err, "eventgrid topic "+k)
		}
	}

	if len(a.s.Collections) > 0 {
		mc, err := a.newMongoCollections(ctx, "mongodb", &MongoCollectionsArgs{
			ResourceGroupName: rg.Name,
		})
		if err != nil {
			return errors.WithMessage(err, "mongodb collections")
		}
		contAppsArgs.MongoDatabaseName = mc.MongoDB.Name
		contAppsArgs.MongoDatabaseConnectionString = mc.Account.ConnectionStrings.Index(pulumi.Int(0))
	}

	var apps *ContainerApps
	if len(a.s.Functions) > 0 || len(a.s.Containers) > 0 {
		apps, err = a.newContainerApps(ctx, "containerApps", contAppsArgs)
		if err != nil {
			return errors.WithMessage(err, "containerApps")
		}
	}

	_, err = newSubscriptions(ctx, "subscriptions", &SubscriptionsArgs{
		ResourceGroupName: rg.Name,
		Apps:              apps.Apps,
	})
	if err != nil {
		return errors.WithMessage(err, "subscripitons")
	}

	// TODO: Add schedule support
	// NOTE: Currently CRONTAB support is required, we either need to revisit the design of
	// our scheduled expressions or implement a workaround or request a feature.
	if len(a.s.Schedules) > 0 {
		_ = ctx.Log.Warn("Schedules are not currently supported for Azure deployments", &pulumi.LogArgs{})
	}

	for k, v := range a.s.ApiDocs {
		_, err = newAzureApiManagement(ctx, k, &AzureApiManagementArgs{
			ResourceGroupName: rg.Name,
			OrgName:           pulumi.String(a.org),
			AdminEmail:        pulumi.String(a.adminEmail),
			OpenAPISpec:       v,
			Apps:              apps.Apps,
		})
		if err != nil {
			return errors.WithMessage(err, "gateway "+k)
		}
	}

	return nil
}

func (a *azureProvider) CleanUp() {
	if a.tmpDir != "" {
		os.Remove(a.tmpDir)
	}
}
