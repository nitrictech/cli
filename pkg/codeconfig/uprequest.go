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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/imdario/mergo"
	multierror "github.com/missionMeteora/toolkit/errors"

	"github.com/nitrictech/cli/pkg/cron"
	deploy "github.com/nitrictech/nitric/core/pkg/api/nitric/deploy/v1"
	v1 "github.com/nitrictech/nitric/core/pkg/api/nitric/v1"
)

type upRequestBuilder struct {
	projName  string
	resources []*deploy.Resource
	idx       map[v1.ResourceType]map[string]int
}

func (b *upRequestBuilder) set(r *deploy.Resource) {
	if _, ok := b.idx[r.Type]; !ok {
		b.idx[r.Type] = map[string]int{}
	}

	if _, ok := b.idx[r.Type][r.Name]; !ok {
		b.idx[r.Type][r.Name] = 0
		b.resources = append(b.resources, r)
	} else {
		current := b.resources[b.idx[r.Type][r.Name]]

		if err := mergo.Merge(&current, r); err != nil {
			current = r
		}

		b.resources[b.idx[r.Type][r.Name]] = current
	}
}

func (b *upRequestBuilder) Output() *deploy.DeployUpRequest {
	return &deploy.DeployUpRequest{
		Spec: &deploy.Spec{
			Resources: b.resources,
		},
		Attributes: map[string]string{
			"x-nitric-project": b.projName,
		},
	}
}

func (c *codeConfig) ToUpRequest() (*deploy.DeployUpRequest, error) {
	builder := &upRequestBuilder{
		projName:  c.initialProject.Name,
		resources: []*deploy.Resource{},
		idx:       map[v1.ResourceType]map[string]int{},
	}
	errs := multierror.ErrorList{}

	for _, f := range c.functions {
		for k := range f.buckets {
			builder.set(&deploy.Resource{
				Name: k,
				Type: v1.ResourceType_Bucket,
				Config: &deploy.Resource_Bucket{
					Bucket: &deploy.Bucket{},
				},
			})
		}

		for k := range f.collections {
			builder.set(&deploy.Resource{
				Name: k,
				Type: v1.ResourceType_Collection,
				Config: &deploy.Resource_Collection{
					Collection: &deploy.Collection{},
				},
			})
		}

		for k := range f.queues {
			builder.set(&deploy.Resource{
				Name: k,
				Type: v1.ResourceType_Queue,
				Config: &deploy.Resource_Queue{
					Queue: &deploy.Queue{},
				},
			})
		}

		for k := range f.topics {
			builder.set(&deploy.Resource{
				Name: k,
				Type: v1.ResourceType_Topic,
				Config: &deploy.Resource_Topic{
					Topic: &deploy.Topic{},
				},
			})
		}

		for k := range f.secrets {
			errs.Push(fmt.Errorf("secrets are not supported in code config (%s)", k))
		}

		for k := range f.apis {
			spec, err := c.apiSpec(k)
			if err != nil {
				errs.Push(fmt.Errorf("could not build spec for api: %s; %w", k, err))
				continue
			}

			apiBody, err := json.Marshal(spec)
			if err != nil {
				errs.Push(err)
				continue
			}

			builder.set(&deploy.Resource{
				Name: k,
				Type: v1.ResourceType_Api,
				Config: &deploy.Resource_Api{
					Api: &deploy.Api{
						Document: &deploy.Api_Openapi{
							Openapi: string(apiBody),
						},
					},
				},
			})
		}

		for _, v := range f.policies {
			principals := []*deploy.Resource{}
			resources := []*deploy.Resource{}

			for _, r := range v.Resources {
				resources = append(resources, &deploy.Resource{
					Name: r.Name,
					Type: r.Type,
				})
			}

			for _, p := range v.Principals {
				principals = append(principals, &deploy.Resource{
					Name: p.Name,
					Type: p.Type,
				})
			}

			builder.set(&deploy.Resource{
				Type: v1.ResourceType_Policy,
				Config: &deploy.Resource_Policy{
					Policy: &deploy.Policy{
						Principals: principals,
						Actions:    v.Actions,
						Resources:  resources,
					},
				},
			})
		}

		for k, v := range f.schedules {
			// Create a new topic target
			// replace spaced with hyphens
			topicName := strings.ToLower(strings.ReplaceAll(k, " ", "-"))

			builder.set(&deploy.Resource{
				Name: topicName,
				Type: v1.ResourceType_Topic,
				Config: &deploy.Resource_Topic{
					Topic: &deploy.Topic{},
				},
			})

			var exp string
			switch v.Cadence.(type) {
			case *v1.ScheduleWorker_Cron:
				exp = v.GetCron().Cron
			default:
				e, err := cron.RateToCron(v.GetRate().Rate)
				if err != nil {
					errs.Push(fmt.Errorf("schedule expresson %s is invalid; %w", v.GetRate().Rate, err))
					continue
				}

				exp = e
			}

			builder.set(&deploy.Resource{
				Name: k,
				Type: v1.ResourceType_Schedule,
				Config: &deploy.Resource_Schedule{
					Schedule: &deploy.Schedule{
						Cron: exp,
						Target: &deploy.ScheduleTarget{
							Target: &deploy.ScheduleTarget_ExecutionUnit{
								ExecutionUnit: k,
							},
						},
					},
				},
			})
		}

		builder.set(&deploy.Resource{
			Name: f.name,
			Type: v1.ResourceType_Function,
			Config: &deploy.Resource_ExecutionUnit{
				ExecutionUnit: &deploy.ExecutionUnit{
					Source: &deploy.ExecutionUnit_Image{
						Image: &deploy.ImageSource{
							Uri: fmt.Sprintf("%s-%s", c.initialProject.Name, f.name),
						},
					},
					Workers: int32(f.WorkerCount()),
				},
			},
		})
	}

	return builder.Output(), errs.Err()
}
