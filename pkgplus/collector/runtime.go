// Copyright Nitric Pty Ltd.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package collector

import (
	"context"

	apispb "github.com/nitrictech/nitric/core/pkg/proto/apis/v1"
	queuespb "github.com/nitrictech/nitric/core/pkg/proto/queues/v1"
	storagepb "github.com/nitrictech/nitric/core/pkg/proto/storage/v1"
	topicspb "github.com/nitrictech/nitric/core/pkg/proto/topics/v1"
	websocketspb "github.com/nitrictech/nitric/core/pkg/proto/websockets/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func illegalRuntimeMethodCall(serviceName string, serviceCall string) error {
	return status.Errorf(
		codes.FailedPrecondition,
		"illegal runtime method call %s made by %s at build time: see nitric documentation %s for more details",
		serviceCall, serviceName, "https://nitric.io/docs/assets/resources-overview#rules",
	)
}

var _ topicspb.TopicsServer = (*ServiceRequirements)(nil)
var _ storagepb.StorageServer = (*ServiceRequirements)(nil)
var _ queuespb.QueuesServer = (*ServiceRequirements)(nil)
var _ apispb.ApiServer = (*ServiceRequirements)(nil)
var _ websocketspb.WebsocketServer = (*ServiceRequirements)(nil)

func (s *ServiceRequirements) Publish(context.Context, *topicspb.TopicPublishRequest) (*topicspb.TopicPublishResponse, error) {
	return nil, illegalRuntimeMethodCall(s.serviceName, "Topic::Publish")
}

// Retrieve an item from a bucket
func (s *ServiceRequirements) Read(context.Context, *storagepb.StorageReadRequest) (*storagepb.StorageReadResponse, error) {
	return nil, illegalRuntimeMethodCall(s.serviceName, "Storage::Read")
}

// Store an item to a bucket
func (s *ServiceRequirements) Write(context.Context, *storagepb.StorageWriteRequest) (*storagepb.StorageWriteResponse, error) {
	return nil, illegalRuntimeMethodCall(s.serviceName, "Storage::Write")
}

// Delete an item from a bucket
func (s *ServiceRequirements) Delete(context.Context, *storagepb.StorageDeleteRequest) (*storagepb.StorageDeleteResponse, error) {
	return nil, illegalRuntimeMethodCall(s.serviceName, "Storage::Delete")
}

// Generate a pre-signed URL for direct operations on an item
func (s *ServiceRequirements) PreSignUrl(context.Context, *storagepb.StoragePreSignUrlRequest) (*storagepb.StoragePreSignUrlResponse, error) {
	return nil, illegalRuntimeMethodCall(s.serviceName, "Storage::PreSignUrl")
}

// List blobs currently in the bucket
func (s *ServiceRequirements) ListBlobs(context.Context, *storagepb.StorageListBlobsRequest) (*storagepb.StorageListBlobsResponse, error) {
	return nil, illegalRuntimeMethodCall(s.serviceName, "Storage::ListBlobs")
}

// Determine is an object exists in a bucket
func (s *ServiceRequirements) Exists(context.Context, *storagepb.StorageExistsRequest) (*storagepb.StorageExistsResponse, error) {
	return nil, illegalRuntimeMethodCall(s.serviceName, "Storage::Exists")
}

// Send messages to a queue
func (s *ServiceRequirements) Send(context.Context, *queuespb.QueueSendRequestBatch) (*queuespb.QueueSendResponse, error) {
	return nil, illegalRuntimeMethodCall(s.serviceName, "Queue::Send")

}

// Receive message(s) from a queue
func (s *ServiceRequirements) Receive(context.Context, *queuespb.QueueReceiveRequest) (*queuespb.QueueReceiveResponse, error) {
	return nil, illegalRuntimeMethodCall(s.serviceName, "Queue::Receive")
}

// Complete an item previously popped from a queue
func (s *ServiceRequirements) Complete(context.Context, *queuespb.QueueCompleteRequest) (*queuespb.QueueCompleteResponse, error) {
	return nil, illegalRuntimeMethodCall(s.serviceName, "Queue::Complete")
}
