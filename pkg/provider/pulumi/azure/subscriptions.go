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
	"bytes"
	"context"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/eventgrid/2018-01-01/eventgrid"
	pulumiEventgrid "github.com/pulumi/pulumi-azure/sdk/v4/go/azure/eventgrid"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type SubscriptionsArgs struct {
	ResourceGroupName pulumi.StringInput
	Apps              map[string]*ContainerApp
}

type Subscriptions struct {
	pulumi.ResourceState

	Name string
}

func newSubscriptions(ctx *pulumi.Context, name string, args *SubscriptionsArgs, opts ...pulumi.ResourceOption) (*Subscriptions, error) {
	res := &Subscriptions{Name: name}

	err := ctx.RegisterComponentResource("nitric:api:AzureApiManagement", name, res, opts...)
	if err != nil {
		return nil, err
	}

	for _, app := range args.Apps {
		if len(app.Subscriptions) == 0 {
			continue
		}

		hostUrl := app.App.LatestRevisionFqdn.ApplyT(func(fqdn string) (string, error) {
			_ = ctx.Log.Info("waiting for "+app.Name+" to start before creating subscriptions", &pulumi.LogArgs{Ephemeral: true})

			// Get the full URL of the deployed container
			hostUrl := "https://" + fqdn

			hCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			// Poll the URL until the host has started.
			for {
				// Provide data in the expected shape. The content is current not important.
				empty := ""
				dummyEvgt := eventgrid.Event{
					ID:          &empty,
					Data:        &empty,
					EventType:   &empty,
					Subject:     &empty,
					DataVersion: &empty,
				}

				jsonStr, err := dummyEvgt.MarshalJSON()
				if err != nil {
					return "", err
				}

				body := bytes.NewBuffer(jsonStr)

				req, err := http.NewRequestWithContext(hCtx, "POST", hostUrl, body)
				if err != nil {
					return "", err
				}

				// TODO: Implement a membrane health check handler in the Membrane and trigger that instead.
				// Set event type header to simulate a subscription validation event.
				// These events are automatically resolved by the Membrane and won't be processed by handlers.
				req.Header.Set("aeg-event-type", "SubscriptionValidation")
				req.Header.Set("Content-Type", "application/json")
				client := &http.Client{
					Timeout: 10 * time.Second,
				}

				resp, err := client.Do(req)
				if err == nil {
					resp.Body.Close()
					break
				}
			}

			return hostUrl, nil
		}).(pulumi.StringOutput)

		_ = ctx.Log.Info("creating subscriptions for "+app.Name, &pulumi.LogArgs{})

		for subName, sub := range app.Subscriptions {
			_, err = pulumiEventgrid.NewEventSubscription(ctx, resourceName(ctx, app.Name+"-"+subName, EventSubscriptionRT), &pulumiEventgrid.EventSubscriptionArgs{
				Scope: sub.ID(),
				WebhookEndpoint: pulumiEventgrid.EventSubscriptionWebhookEndpointArgs{
					Url: pulumi.Sprintf("%s/x-nitric-subscription/%s", hostUrl, subName),
					// TODO: Reduce event chattiness here and handle internally in the Azure AppService HTTP Gateway?
					MaxEventsPerBatch:         pulumi.Int(1),
					ActiveDirectoryAppIdOrUri: app.Sp.ClientID,
					ActiveDirectoryTenantId:   app.Sp.TenantID,
				},
				RetryPolicy: pulumiEventgrid.EventSubscriptionRetryPolicyArgs{
					MaxDeliveryAttempts: pulumi.Int(30),
					EventTimeToLive:     pulumi.Int(5),
				},
			})
			if err != nil {
				return nil, err
			}
		}
	}

	return res, nil
}
