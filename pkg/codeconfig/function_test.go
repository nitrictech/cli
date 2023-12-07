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
	"fmt"
	"testing"

	"github.com/nitrictech/cli/pkg/project"
	v1 "github.com/nitrictech/nitric/core/pkg/api/nitric/v1"
)

type TestResource[T any] struct {
	name string
	res  T
}

func Test_addError(t *testing.T) {
	tests := []struct {
		name   string
		errors []error
		expect []string
	}{
		{
			name: "Test adding multiple errors",
			errors: []error{
				fmt.Errorf("error occurred 1"),
				fmt.Errorf("error occurred 2"),
			},
			expect: []string{
				"function test-function: error occurred 1",
				"function test-function: error occurred 2",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFunction("test-function", project.Function{Handler: "test-function"})

			for _, err := range tt.errors {
				f.AddError(err.Error())
			}

			if len(f.errors) != len(tt.expect) {
				t.Fatalf("Expected len = %d, Got len = %d", len(tt.expect), len(f.buckets))
			}

			for idx, err := range f.errors {
				if err != tt.expect[idx] {
					t.Fatalf("Expected = %s, Got = %s", err, tt.expect[idx])
				}
			}
		})
	}
}

func Test_addBucket(t *testing.T) {
	tests := []struct {
		name      string
		resources []TestResource[*v1.BucketResource]
		expect    int
	}{
		{
			name: "Test two different buckets",
			resources: []TestResource[*v1.BucketResource]{
				{
					name: "bucket-1",
					res:  &v1.BucketResource{},
				},
				{
					name: "bucket-2",
					res:  &v1.BucketResource{},
				},
			},
			expect: 2,
		},
		{
			name: "Test two buckets with same name",
			resources: []TestResource[*v1.BucketResource]{
				{
					name: "bucket-1",
					res:  &v1.BucketResource{},
				},
				{
					name: "bucket-1",
					res:  &v1.BucketResource{},
				},
			},
			expect: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFunction("test-function", project.Function{})

			for _, r := range tt.resources {
				f.AddBucket(r.name, r.res)
			}

			if len(f.errors) > 0 {
				t.Fatalf("Expected no errors but got errors: %s", f.errors)
			}

			if len(f.buckets) != tt.expect {
				t.Fatalf("Expected len = %d, Got len = %d", tt.expect, len(f.buckets))
			}
		})
	}
}

func Test_addSecret(t *testing.T) {
	tests := []struct {
		name      string
		resources []TestResource[*v1.SecretResource]
		expect    int
	}{
		{
			name: "Test two different secrets",
			resources: []TestResource[*v1.SecretResource]{
				{
					name: "secret-1",
					res:  &v1.SecretResource{},
				},
				{
					name: "secret-2",
					res:  &v1.SecretResource{},
				},
			},
			expect: 2,
		},
		{
			name: "Test two secrets with same name",
			resources: []TestResource[*v1.SecretResource]{
				{
					name: "secret-1",
					res:  &v1.SecretResource{},
				},
				{
					name: "secret-1",
					res:  &v1.SecretResource{},
				},
			},
			expect: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFunction("test-function", project.Function{})

			for _, r := range tt.resources {
				f.AddSecret(r.name, r.res)
			}

			if len(f.errors) > 0 {
				t.Fatalf("Expected no errors but got errors: %s", f.errors)
			}

			if len(f.secrets) != tt.expect {
				t.Fatalf("Expected len = %d, Got len = %d", tt.expect, len(f.secrets))
			}
		})
	}
}

func Test_addTopic(t *testing.T) {
	tests := []struct {
		name      string
		resources []TestResource[*v1.TopicResource]
		expect    int
	}{
		{
			name: "Test two different topics",
			resources: []TestResource[*v1.TopicResource]{
				{
					name: "secret-1",
					res:  &v1.TopicResource{},
				},
				{
					name: "secret-2",
					res:  &v1.TopicResource{},
				},
			},
			expect: 2,
		},
		{
			name: "Test two topics with same name",
			resources: []TestResource[*v1.TopicResource]{
				{
					name: "topic-1",
					res:  &v1.TopicResource{},
				},
				{
					name: "topic-1",
					res:  &v1.TopicResource{},
				},
			},
			expect: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFunction("test-function", project.Function{})

			for _, r := range tt.resources {
				f.AddTopic(r.name, r.res)
			}

			if len(f.errors) > 0 {
				t.Fatalf("Expected no errors but got errors: %s", f.errors)
			}

			if len(f.topics) != tt.expect {
				t.Fatalf("Expected len = %d, Got len = %d", tt.expect, len(f.topics))
			}
		})
	}
}

func Test_addCollection(t *testing.T) {
	tests := []struct {
		name      string
		resources []TestResource[*v1.CollectionResource]
		expect    int
	}{
		{
			name: "Test two different collections",
			resources: []TestResource[*v1.CollectionResource]{
				{
					name: "collection-1",
					res:  &v1.CollectionResource{},
				},
				{
					name: "collection-2",
					res:  &v1.CollectionResource{},
				},
			},
			expect: 2,
		},
		{
			name: "Test two collections with same name",
			resources: []TestResource[*v1.CollectionResource]{
				{
					name: "collection-1",
					res:  &v1.CollectionResource{},
				},
				{
					name: "collection-1",
					res:  &v1.CollectionResource{},
				},
			},
			expect: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFunction("test-function", project.Function{})

			for _, r := range tt.resources {
				f.AddCollection(r.name, r.res)
			}

			if len(f.errors) > 0 {
				t.Fatalf("Expected no errors but got errors: %s", f.errors)
			}

			if len(f.collections) != tt.expect {
				t.Fatalf("Expected len = %d, Got len = %d", tt.expect, len(f.collections))
			}
		})
	}
}

func Test_addQueue(t *testing.T) {
	tests := []struct {
		name      string
		resources []TestResource[*v1.QueueResource]
		expect    int
	}{
		{
			name: "Test two different queues",
			resources: []TestResource[*v1.QueueResource]{
				{
					name: "queue-1",
					res:  &v1.QueueResource{},
				},
				{
					name: "queue-2",
					res:  &v1.QueueResource{},
				},
			},
			expect: 2,
		},
		{
			name: "Test two queues with same name",
			resources: []TestResource[*v1.QueueResource]{
				{
					name: "queue-1",
					res:  &v1.QueueResource{},
				},
				{
					name: "queue-1",
					res:  &v1.QueueResource{},
				},
			},
			expect: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFunction("test-function", project.Function{})

			for _, r := range tt.resources {
				f.AddQueue(r.name, r.res)
			}

			if len(f.errors) > 0 {
				t.Fatalf("Expected no errors but got errors: %s", f.errors)
			}

			if len(f.queues) != tt.expect {
				t.Fatalf("Expected len = %d, Got len = %d", tt.expect, len(f.queues))
			}
		})
	}
}

func Test_addBucketNotificationHandler(t *testing.T) {
	tests := []struct {
		name      string
		resources []*v1.BucketNotificationWorker
		expect    []int
	}{
		{
			name: "Test two different bucket notifications with same bucket",
			resources: []*v1.BucketNotificationWorker{
				{
					Bucket: "bucket-1",
					Config: &v1.BucketNotificationConfig{
						NotificationType:         v1.BucketNotificationType_Created,
						NotificationPrefixFilter: "*",
					},
				},
				{
					Bucket: "bucket-1",
					Config: &v1.BucketNotificationConfig{
						NotificationType:         v1.BucketNotificationType_Created,
						NotificationPrefixFilter: "*",
					},
				},
			},
			expect: []int{2},
		},
		{
			name: "Test two bucket notifications with different bucket",
			resources: []*v1.BucketNotificationWorker{
				{
					Bucket: "bucket-1",
					Config: &v1.BucketNotificationConfig{
						NotificationType:         v1.BucketNotificationType_Created,
						NotificationPrefixFilter: "*",
					},
				},
				{
					Bucket: "bucket-2",
					Config: &v1.BucketNotificationConfig{
						NotificationType:         v1.BucketNotificationType_Created,
						NotificationPrefixFilter: "*",
					},
				},
			},
			expect: []int{1, 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFunction("test-function", project.Function{})

			for _, r := range tt.resources {
				f.AddBucketNotificationHandler(r)
			}

			if len(f.errors) > 0 {
				t.Fatalf("Expected no errors but got errors: %s", f.errors)
			}

			if len(tt.expect) != len(f.bucketNotifications) {
				t.Fatalf("Expected map to be len = %d, Got len = %d", len(tt.expect), len(f.bucketNotifications))
			}

			count := 0
			for _, n := range f.bucketNotifications {
				size := tt.expect[count]
				if len(n) != size {
					t.Fatalf("Expected map[%d] to be len = %d, Got len = %d", count, size, len(n))
				}

				count++
			}
		})
	}
}

func Test_addScheduleHandler(t *testing.T) {
	tests := []struct {
		name      string
		resources []*v1.ScheduleWorker
		expect    int
		errs      []string
	}{
		{
			name: "Test two different schedules",
			resources: []*v1.ScheduleWorker{
				{
					Key: "schedule-1",
				},
				{
					Key: "schedule-2",
				},
			},
			expect: 2,
			errs:   []string{},
		},
		{
			name: "Test two schedules with same name",
			resources: []*v1.ScheduleWorker{
				{
					Key: "schedule-1",
				},
				{
					Key: "schedule-1",
				},
			},
			expect: 1,
			errs:   []string{"function test-function: declared schedule schedule-1 multiple times"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFunction("test-function", project.Function{Handler: "test-function"})

			for _, r := range tt.resources {
				f.AddScheduleHandler(r)
			}

			if len(f.schedules) != tt.expect {
				t.Fatalf("Expected len = %d, Got len = %d", tt.expect, len(f.schedules))
			}

			if len(f.errors) != len(tt.errs) {
				t.Fatalf("Expected err length = %d, Got err length = %d", len(tt.errs), len(f.errors))
			}

			for idx, err := range f.errors {
				if err != tt.errs[idx] {
					t.Fatalf("Expected = %s, Got = %s", err, tt.errs[idx])
				}
			}
		})
	}
}

func Test_addSubscriptionHandler(t *testing.T) {
	tests := []struct {
		name      string
		resources []*v1.SubscriptionWorker
		expect    int
		errs      []string
	}{
		{
			name: "Test two different topics",
			resources: []*v1.SubscriptionWorker{
				{
					Topic: "topic-1",
				},
				{
					Topic: "topic-2",
				},
			},
			expect: 2,
			errs:   []string{},
		},
		{
			name: "Test two topics with same name",
			resources: []*v1.SubscriptionWorker{
				{
					Topic: "topic-1",
				},
				{
					Topic: "topic-1",
				},
			},
			expect: 1,
			errs:   []string{"function test-function: declared multiple subscriptions for topic topic-1, only one subscription per topic is allowed per function"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFunction("test-function", project.Function{Handler: "test-function"})

			for _, r := range tt.resources {
				f.AddSubscriptionHandler(r)
			}

			if len(f.subscriptions) != tt.expect {
				t.Fatalf("Expected len = %d, Got len = %d", tt.expect, len(f.subscriptions))
			}

			if len(f.errors) != len(tt.errs) {
				t.Fatalf("Expected err length = %d, Got err length = %d", len(tt.errs), len(f.errors))
			}

			for idx, err := range f.errors {
				if err != tt.errs[idx] {
					t.Fatalf("Expected = %s, Got = %s", err, tt.errs[idx])
				}
			}
		})
	}
}

func Test_addWebsocketHandler(t *testing.T) {
	tests := []struct {
		name      string
		resources []*v1.WebsocketWorker
		expect    []int
		errs      []string
	}{
		{
			name: "Test two different websockets",
			resources: []*v1.WebsocketWorker{
				{
					Socket: "websocket-1",
				},
				{
					Socket: "websocket-2",
				},
			},
			expect: []int{1, 1},
			errs:   []string{},
		},
		{
			name: "Test two websockets with same name different events",
			resources: []*v1.WebsocketWorker{
				{
					Socket: "websocket-1",
					Event:  v1.WebsocketEvent_Connect,
				},
				{
					Socket: "websocket-1",
					Event:  v1.WebsocketEvent_Disconnect,
				},
			},
			expect: []int{2},
			errs:   []string{},
		},
		{
			name: "Test two websockets with same name same events",
			resources: []*v1.WebsocketWorker{
				{
					Socket: "websocket-1",
					Event:  v1.WebsocketEvent_Connect,
				},
				{
					Socket: "websocket-1",
					Event:  v1.WebsocketEvent_Connect,
				},
			},
			expect: []int{1},
			errs:   []string{"function test-function: has registered multiple connect workers for socket: websocket-1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFunction("test-function", project.Function{
				Handler: "test-function",
				Project: &project.Project{
					PreviewFeatures: []string{"websockets"},
				},
			})

			for _, r := range tt.resources {
				f.AddWebsocketHandler(r)
			}

			if len(tt.expect) != len(f.websockets) {
				t.Fatalf("Expected map to be len = %d, Got len = %d", len(tt.expect), len(f.websockets))
			}

			count := 0
			for _, n := range f.websockets {
				size := tt.expect[count]
				if n.WorkerCount() != size {
					t.Fatalf("Expected map[%d] to be len = %d, Got len = %d", count, size, n.WorkerCount())
				}

				count++
			}

			if len(f.errors) != len(tt.errs) {
				t.Fatalf("Expected err length = %d, Got err length = %d", len(tt.errs), len(f.errors))
			}

			for idx, err := range f.errors {
				if err != tt.errs[idx] {
					t.Fatalf("Expected = %s, Got = %s", err, tt.errs[idx])
				}
			}
		})
	}
}

func Test_workerCount(t *testing.T) {
	tests := []struct {
		name          string
		proxies       []*v1.HttpWorker
		apis          []*v1.ApiWorker
		subscriptions []*v1.SubscriptionWorker
		websockets    []*v1.WebsocketWorker
		schedules     []*v1.ScheduleWorker
		expect        int
	}{
		{
			name: "Test http contributes to the worker count",
			websockets: []*v1.WebsocketWorker{
				{
					Socket: "websocket-1",
					Event:  v1.WebsocketEvent_Connect,
				},
			},
			proxies: []*v1.HttpWorker{
				{
					Port: 3000,
				},
			},
			subscriptions: []*v1.SubscriptionWorker{
				{
					Topic: "topic-1",
				},
			},
			schedules: []*v1.ScheduleWorker{
				{
					Key: "schedule-1",
				},
			},
			expect: 4,
		},
		{
			name: "Test each api resource contributes to the worker count",
			websockets: []*v1.WebsocketWorker{
				{
					Socket: "websocket-1",
					Event:  v1.WebsocketEvent_Connect,
				},
			},
			apis: []*v1.ApiWorker{
				{
					Api: "api-1",
				},
			},
			subscriptions: []*v1.SubscriptionWorker{
				{
					Topic: "topic-1",
				},
			},
			schedules: []*v1.ScheduleWorker{
				{
					Key: "schedule-1",
				},
			},
			expect: 4,
		},
		{
			name: "Test two different websocket events",
			websockets: []*v1.WebsocketWorker{
				{
					Socket: "websocket-1",
					Event:  v1.WebsocketEvent_Connect,
				},
				{
					Socket: "websocket-1",
					Event:  v1.WebsocketEvent_Disconnect,
				},
			},
			expect: 2,
		},
		{
			name: "Test three api methods",
			apis: []*v1.ApiWorker{
				{
					Api:     "api-1",
					Path:    "/",
					Methods: []string{"GET"},
				},
				{
					Api:     "api-1",
					Path:    "/",
					Methods: []string{"PUT"},
				},
				{
					Api:     "api-1",
					Path:    "/",
					Methods: []string{"POST"},
				},
			},
			expect: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFunction("test-function", project.Function{
				Handler: "test-function",
				Project: &project.Project{
					PreviewFeatures: []string{"websockets", "http"},
				},
			})

			for _, a := range tt.apis {
				f.AddApiHandler(a)
			}

			for _, w := range tt.websockets {
				f.AddWebsocketHandler(w)
			}

			for _, p := range tt.proxies {
				f.AddHttpWorker(p)
			}

			for _, s := range tt.subscriptions {
				f.AddSubscriptionHandler(s)
			}

			for _, s := range tt.schedules {
				f.AddScheduleHandler(s)
			}

			if f.WorkerCount() != tt.expect {
				t.Fatalf("Expected = %d, Got = %d", tt.expect, f.WorkerCount())
			}
		})
	}
}

func Test_addApiHandler(t *testing.T) {
	tests := []struct {
		name      string
		resources []*v1.ApiWorker
		expect    []int
		errs      []string
	}{
		{
			name: "Test two different apis with different paths",
			resources: []*v1.ApiWorker{
				{
					Api:  "api-1",
					Path: "/one",
				},
				{
					Api:  "api-2",
					Path: "/two",
				},
			},
			expect: []int{1, 1},
			errs:   []string{},
		},
		{
			name: "Test two apis with same name different paths",
			resources: []*v1.ApiWorker{
				{
					Api:  "api-1",
					Path: "/one",
				},
				{
					Api:  "api-1",
					Path: "/two",
				},
			},
			expect: []int{2},
			errs:   []string{},
		},
		{
			name: "Test two apis with different name same paths same methods",
			resources: []*v1.ApiWorker{
				{
					Api:     "api-1",
					Path:    "/one",
					Methods: []string{"GET"},
				},
				{
					Api:     "api-2",
					Path:    "/one",
					Methods: []string{"GET"},
				},
			},
			expect: []int{1},
			errs:   []string{"function test-function: APIs cannot share paths within the same function"},
		},
		{
			name: "Test two apis with different name same paths different methods",
			resources: []*v1.ApiWorker{
				{
					Api:     "api-1",
					Path:    "/one",
					Methods: []string{"GET"},
				},
				{
					Api:     "api-2",
					Path:    "/one",
					Methods: []string{"POST"},
				},
			},
			expect: []int{1, 1},
			errs:   []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFunction("test-function", project.Function{
				Handler: "test-function",
			})

			for _, r := range tt.resources {
				f.AddApiHandler(r)
			}

			if len(tt.expect) != len(f.apis) {
				t.Fatalf("Expected map to be len = %d, Got len = %d", len(tt.expect), len(f.apis))
			}

			count := 0
			for _, n := range f.apis {
				size := tt.expect[count]
				if n.WorkerCount() != size {
					t.Fatalf("Expected map[%d] to be len = %d, Got len = %d", count, size, n.WorkerCount())
				}

				count++
			}

			if len(f.errors) != len(tt.errs) {
				t.Fatalf("Expected err length = %d, Got err length = %d", len(tt.errs), len(f.errors))
			}

			for idx, err := range f.errors {
				if err != tt.errs[idx] {
					t.Fatalf("Expected = %s, Got = %s", err, tt.errs[idx])
				}
			}
		})
	}
}
