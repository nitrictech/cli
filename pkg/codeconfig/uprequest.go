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

package codeconfig

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/imdario/mergo"
	multierror "github.com/missionMeteora/toolkit/errors"
	"github.com/nitrictech/cli/pkg/cron"
	deploymentspb "github.com/nitrictech/nitric/core/pkg/proto/deployments/v1"
	resourcespb "github.com/nitrictech/nitric/core/pkg/proto/resources/v1"
	schedulespb "github.com/nitrictech/nitric/core/pkg/proto/schedules/v1"
	"github.com/samber/lo"
)

type upRequestBuilder struct {
	projName  string
	resources map[resourcespb.ResourceType]map[string]*deploymentspb.Resource
}

func (b *upRequestBuilder) set(r *deploymentspb.Resource) {
	if _, ok := b.resources[r.Type]; !ok {
		b.resources[r.Type] = map[string]*deploymentspb.Resource{}
	}

	if _, ok := b.resources[r.Type][r.Name]; !ok {
		b.resources[r.Type][r.Name] = r
	} else {
		current := b.resources[r.Type][r.Name]
		if err := mergo.Merge(current, r, mergo.WithAppendSlice); err != nil {
			current = r
		}

		b.resources[r.Type][r.Name] = current
	}
}

func ValidateUpRequest(request *deploymentspb.DeployUpRequest) error {
	errors := []string{}

	websockets := lo.Filter(request.Spec.Resources, func(res *deploymentspb.Resource, idx int) bool {
		return res.Type == resourcespb.ResourceType_Websocket
	})

	for _, ws := range websockets {
		if ws.GetWebsocket().ConnectTarget == nil || ws.GetWebsocket().DisconnectTarget == nil || ws.GetWebsocket().MessageTarget == nil {
			errors = append(errors, fmt.Sprintf("socket: %s, is missing handlers. Sockets must have handlers for connect/disconnect/message events", ws.Name))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("application contains errors:\n %s", strings.Join(errors, "\n"))
	}

	return nil
}

func (b *upRequestBuilder) Output() *deploymentspb.DeployUpRequest {
	res := []*deploymentspb.Resource{}

	for _, resMap := range b.resources {
		for _, r := range resMap {
			res = append(res, r)
		}
	}

	return &deploymentspb.DeployUpRequest{
		Spec: &deploymentspb.Spec{
			Resources: res,
		},
	}
}

func md5Hash(b []byte) string {
	hasher := md5.New()
	hasher.Write(b)

	return hex.EncodeToString(hasher.Sum(nil))
}

func policyResourceName(policy *resourcespb.PolicyResource) (string, error) {
	policyDoc, err := json.Marshal(policy)
	if err != nil {
		return "", err
	}

	return md5Hash(policyDoc), nil
}

func (c *codeConfig) ToUpRequest() (*deploymentspb.DeployUpRequest, error) {
	builder := &upRequestBuilder{
		projName:  c.initialProject.Name,
		resources: map[resourcespb.ResourceType]map[string]*deploymentspb.Resource{},
	}
	errs := multierror.ErrorList{}

	for _, f := range c.functions {
		for bucketName := range f.buckets {
			notifications := []*deploymentspb.BucketNotificationTarget{}

			for _, v := range f.bucketNotifications[bucketName] {
				notifications = append(notifications, &deploymentspb.BucketNotificationTarget{
					Config: v,
					Target: &deploymentspb.BucketNotificationTarget_ExecutionUnit{
						ExecutionUnit: f.name,
					},
				})
			}

			res := &deploymentspb.Resource{
				Name: bucketName,
				Type: resourcespb.ResourceType_Bucket,
				Config: &deploymentspb.Resource_Bucket{
					Bucket: &deploymentspb.Bucket{
						Notifications: notifications,
					},
				},
			}

			builder.set(res)
		}

		for k := range f.collections {
			builder.set(&deploymentspb.Resource{
				Name: k,
				Type: resourcespb.ResourceType_Collection,
				Config: &deploymentspb.Resource_Collection{
					Collection: &deploymentspb.Collection{},
				},
			})
		}

		for topicName := range f.topics {
			subs := []*deploymentspb.SubscriptionTarget{}

			for _, v := range f.subscriptions {
				if v.GetTopicName() == topicName {
					subs = append(subs, &deploymentspb.SubscriptionTarget{
						Target: &deploymentspb.SubscriptionTarget_ExecutionUnit{
							ExecutionUnit: f.name,
						},
					})
				}
			}

			res := &deploymentspb.Resource{
				Name: topicName,
				Type: resourcespb.ResourceType_Topic,
				Config: &deploymentspb.Resource_Topic{
					Topic: &deploymentspb.Topic{
						// TODO: Determine if this will successfully merge between multiple functions
						Subscriptions: subs,
					},
				},
			}
			builder.set(res)
		}

		for k := range f.secrets {
			res := &deploymentspb.Resource{
				Name:   k,
				Type:   resourcespb.ResourceType_Secret,
				Config: &deploymentspb.Resource_Secret{},
			}
			builder.set(res)
		}

		for k := range f.apis {
			spec, err := c.apiSpec(k, nil)
			if err != nil {
				errs.Push(fmt.Errorf("could not build spec for api: %s; %w", k, err))
				continue
			}

			apiBody, err := json.Marshal(spec)
			if err != nil {
				errs.Push(err)
				continue
			}

			builder.set(&deploymentspb.Resource{
				Name: k,
				Type: resourcespb.ResourceType_Api,
				Config: &deploymentspb.Resource_Api{
					Api: &deploymentspb.Api{
						Document: &deploymentspb.Api_Openapi{
							Openapi: string(apiBody),
						},
					},
				},
			})
		}

		// Create websockets and attach relevant workers for this function
		for k, ws := range f.websockets {
			// Collect all sockets organised by name and even type
			deploymentspbWebsocket := &deploymentspb.Websocket{}

			if ws.connectWorker != nil {
				deploymentspbWebsocket.ConnectTarget = &deploymentspb.WebsocketTarget{
					Target: &deploymentspb.WebsocketTarget_ExecutionUnit{
						ExecutionUnit: f.name,
					},
				}
			}

			if ws.disconnectWorker != nil {
				deploymentspbWebsocket.DisconnectTarget = &deploymentspb.WebsocketTarget{
					Target: &deploymentspb.WebsocketTarget_ExecutionUnit{
						ExecutionUnit: f.name,
					},
				}
			}

			if ws.messageWorker != nil {
				deploymentspbWebsocket.MessageTarget = &deploymentspb.WebsocketTarget{
					Target: &deploymentspb.WebsocketTarget_ExecutionUnit{
						ExecutionUnit: f.name,
					},
				}
			}

			builder.set(&deploymentspb.Resource{
				Name: k,
				Type: resourcespb.ResourceType_Websocket,
				Config: &deploymentspb.Resource_Websocket{
					Websocket: deploymentspbWebsocket,
				},
			})
		}

		// This will produce a compacted map of policy resources with colliding principals and actions
		// we'll compact all these resources into a single policy object
		compactedPoliciesByKey := lo.GroupBy(f.policies, func(item *resourcespb.PolicyResource) string {
			// get the princpals and actions as a unique key (make sure they're sorted for consistency)
			principalNames := lo.Reduce(item.Principals, func(agg []string, principal *resourcespb.Resource, idx int) []string {
				return append(agg, principal.Name)
			}, []string{})
			slices.Sort(principalNames)

			principals := strings.Join(principalNames, ":")

			slices.Sort(item.Actions)
			actions := lo.Reduce(item.Actions, func(agg string, action resourcespb.Action, idx int) string {
				return agg + action.String()
			}, "")

			return principals + "-" + actions
		})

		compactedPolicies := []*resourcespb.PolicyResource{}
		// for each key of the compacted policies we want to make a single policy object that appends all of the policies resources together
		for _, pols := range compactedPoliciesByKey {
			newPol := pols[0]

			for _, pol := range pols[1:] {
				newPol.Resources = append(newPol.Resources, pol.Resources...)
			}

			compactedPolicies = append(compactedPolicies, newPol)
		}

		dedupedPolicies := map[string]*resourcespb.PolicyResource{}

		for _, v := range compactedPolicies {
			policyName, err := policyResourceName(v)
			if err != nil {
				return nil, err
			}

			dedupedPolicies[policyName] = v
		}

		for policyName, v := range dedupedPolicies {
			principals := []*deploymentspb.Resource{}
			resources := []*deploymentspb.Resource{}

			for _, r := range v.Resources {
				resources = append(resources, &deploymentspb.Resource{
					Name: r.Name,
					Type: r.Type,
				})
			}

			for _, p := range v.Principals {
				principals = append(principals, &deploymentspb.Resource{
					Name: p.Name,
					Type: p.Type,
				})
			}

			builder.set(&deploymentspb.Resource{
				Name: policyName,
				Type: resourcespb.ResourceType_Policy,
				Config: &deploymentspb.Resource_Policy{
					Policy: &deploymentspb.Policy{
						Principals: principals,
						Actions:    v.Actions,
						Resources:  resources,
					},
				},
			})
		}

		for k, v := range f.schedules {
			var exp string
			switch v.Cadence.(type) {
			case *schedulespb.RegistrationRequest_Cron:
				exp = v.GetCron().GetExpression()
			default:
				e, err := cron.RateToCron(v.GetRate().Rate)
				if err != nil {
					errs.Push(fmt.Errorf("schedule expression %s is invalid; %w", v.GetRate().Rate, err))
					continue
				}

				exp = e
			}

			builder.set(&deploymentspb.Resource{
				Name: k,
				Type: resourcespb.ResourceType_Schedule,
				Config: &deploymentspb.Resource_Schedule{
					Schedule: &deploymentspb.Schedule{
						Cron: exp,
						Target: &deploymentspb.ScheduleTarget{
							Target: &deploymentspb.ScheduleTarget_ExecutionUnit{
								ExecutionUnit: f.name,
							},
						},
					},
				},
			})
		}

		for range f.httpWorkers {
			builder.set(&deploymentspb.Resource{
				Name: f.name,
				Type: resourcespb.ResourceType_Http,
				Config: &deploymentspb.Resource_Http{
					Http: &deploymentspb.Http{
						Target: &deploymentspb.HttpTarget{
							Target: &deploymentspb.HttpTarget_ExecutionUnit{
								ExecutionUnit: f.name,
							},
						},
					},
				},
			})
		}

		// Get the original function config
		fun := c.initialProject.Functions[f.name]

		builder.set(&deploymentspb.Resource{
			Name: f.name,
			Type: resourcespb.ResourceType_Function,
			Config: &deploymentspb.Resource_ExecutionUnit{
				ExecutionUnit: &deploymentspb.ExecutionUnit{
					Source: &deploymentspb.ExecutionUnit_Image{
						Image: &deploymentspb.ImageSource{
							Uri: fmt.Sprintf("%s-%s", c.initialProject.Name, f.name),
						},
					},
					Workers: int32(f.WorkerCount()),
					Type:    fun.Config.Type,
					Env:     c.envMap,
				},
			},
		})
	}

	if errs.Err() != nil {
		return nil, errs.Err()
	}

	out := builder.Output()

	err := ValidateUpRequest(out)
	if err != nil {
		return nil, err
	}

	return out, nil
}
