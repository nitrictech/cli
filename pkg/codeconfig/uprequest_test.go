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

	v1 "github.com/nitrictech/nitric/core/pkg/api/nitric/v1"
)

func Test_checkDuplicateBucketNotifications(t *testing.T) {
	tests := []struct {
		name       string
		notifications []*v1.BucketNotificationWorker
		want       error
	}{
		{
			name:       "Test with no overlaps no error is returned",
			notifications: []*v1.BucketNotificationWorker{
				{
					Config: &v1.BucketNotificationConfig{
						EventType: v1.EventType_Created,
						EventFilter: "/product",
					},
				},
				{
					Config: &v1.BucketNotificationConfig{
						EventType: v1.EventType_Deleted,
						EventFilter: "/product",
					},
				},
				{
					Config: &v1.BucketNotificationConfig{
						EventType: v1.EventType_Created,
						EventFilter: "/photos",
					},
				},
				{
					Config: &v1.BucketNotificationConfig{
						EventType: v1.EventType_Deleted,
						EventFilter: "/photos",
					},
				},
				{
					Config: &v1.BucketNotificationConfig{
						EventType: v1.EventType_Created,
						EventFilter: "/users",
					},
				},
				{
					Config: &v1.BucketNotificationConfig{
						EventType: v1.EventType_Deleted,
						EventFilter: "/users",
					},
				},
			},
			want:       nil,
		},
		{
			name:       "Test overlaps for same event type",
			notifications: []*v1.BucketNotificationWorker{
				{
					Config: &v1.BucketNotificationConfig{
						EventType: v1.EventType_Created,
						EventFilter: "/product",
					},
				},
				{
					Config: &v1.BucketNotificationConfig{
						EventType: v1.EventType_Deleted,
						EventFilter: "/product",
					},
				},
				{
					Config: &v1.BucketNotificationConfig{
						EventType: v1.EventType_Created,
						EventFilter: "/photos",
					},
				},
				{
					Config: &v1.BucketNotificationConfig{
						EventType: v1.EventType_Created,
						EventFilter: "/product/images",
					},
				},
			},
			want: fmt.Errorf("overlapping prefixes in notifications for bucket"),
		},
		{
			name:       "Test overlaps for wildcard prefx",
			notifications: []*v1.BucketNotificationWorker{
				{
					Config: &v1.BucketNotificationConfig{
						EventType: v1.EventType_Created,
						EventFilter: "*",
					},
				},
				{
					Config: &v1.BucketNotificationConfig{
						EventType: v1.EventType_Created,
						EventFilter: "/product",
					},
				},
			},
			want: fmt.Errorf("overlapping prefixes in notifications for bucket"),
		},
		{
			name:       "Test overlap for different event types",
			notifications: []*v1.BucketNotificationWorker{
				{
					Config: &v1.BucketNotificationConfig{
						EventType: v1.EventType_Created,
						EventFilter: "/product",
					},
				},
				{
					Config: &v1.BucketNotificationConfig{
						EventType: v1.EventType_Deleted,
						EventFilter: "/product/photos/",
					},
				},
			},
			want: nil,
		},
		{
			name:       "Test overlap for strings but not prefixes",
			notifications: []*v1.BucketNotificationWorker{
				{
					Config: &v1.BucketNotificationConfig{
						EventType: v1.EventType_Created,
						EventFilter: "/photos",
					},
				},
				{
					Config: &v1.BucketNotificationConfig{
						EventType: v1.EventType_Created,
						EventFilter: "/product/photos/",
					},
				},
			},
			want: nil,
		},
		{
			name:       "empty array",
			notifications: []*v1.BucketNotificationWorker{},
			want:       nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkDuplicateBucketNotifications(tt.notifications)
			if got == nil && tt.want == nil {
				return
			}
			if got.Error() != tt.want.Error() {
				t.Errorf("checkDuplicateBucketNotifications() got = %v, want %v", got, tt.want)
			}
		})
	}
}
