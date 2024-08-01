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

package collector

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	apispb "github.com/nitrictech/nitric/core/pkg/proto/apis/v1"
	kvstorepb "github.com/nitrictech/nitric/core/pkg/proto/kvstore/v1"
	queuespb "github.com/nitrictech/nitric/core/pkg/proto/queues/v1"
	secretspb "github.com/nitrictech/nitric/core/pkg/proto/secrets/v1"
	sqlpb "github.com/nitrictech/nitric/core/pkg/proto/sql/v1"
	storagepb "github.com/nitrictech/nitric/core/pkg/proto/storage/v1"
	topicspb "github.com/nitrictech/nitric/core/pkg/proto/topics/v1"
	websocketspb "github.com/nitrictech/nitric/core/pkg/proto/websockets/v1"
)

func (s *ServiceRequirements) illegalRuntimeMethodCall(serviceCall string) error {
	return status.Errorf(
		codes.FailedPrecondition,
		"illegal runtime method call %s made by %s at build time: see nitric documentation %s for more details",
		serviceCall, s.serviceFile, "https://nitric.io/docs/assets/resources-overview#rules",
	)
}

var (
	_ topicspb.TopicsServer         = (*ServiceRequirements)(nil)
	_ storagepb.StorageServer       = (*ServiceRequirements)(nil)
	_ queuespb.QueuesServer         = (*ServiceRequirements)(nil)
	_ apispb.ApiServer              = (*ServiceRequirements)(nil)
	_ kvstorepb.KvStoreServer       = (*ServiceRequirements)(nil)
	_ sqlpb.SqlServer               = (*ServiceRequirements)(nil)
	_ websocketspb.WebsocketServer  = (*ServiceRequirements)(nil)
	_ secretspb.SecretManagerServer = (*ServiceRequirements)(nil)
)

// API
func (s *ServiceRequirements) ApiDetails(context.Context, *apispb.ApiDetailsRequest) (*apispb.ApiDetailsResponse, error) {
	return nil, s.illegalRuntimeMethodCall("Api::ApiDetails")
}

// Websockets
func (s *ServiceRequirements) SocketDetails(context.Context, *websocketspb.WebsocketDetailsRequest) (*websocketspb.WebsocketDetailsResponse, error) {
	return nil, s.illegalRuntimeMethodCall("Websocket::Websocket")
}

func (s *ServiceRequirements) SendMessage(context.Context, *websocketspb.WebsocketSendRequest) (*websocketspb.WebsocketSendResponse, error) {
	return nil, s.illegalRuntimeMethodCall("Websocket::Websocket")
}

func (s *ServiceRequirements) CloseConnection(context.Context, *websocketspb.WebsocketCloseConnectionRequest) (*websocketspb.WebsocketCloseConnectionResponse, error) {
	return nil, s.illegalRuntimeMethodCall("Websocket::Websocket")
}

// Secrets
func (s *ServiceRequirements) Access(context.Context, *secretspb.SecretAccessRequest) (*secretspb.SecretAccessResponse, error) {
	return nil, s.illegalRuntimeMethodCall("Secrets::Access")
}

func (s *ServiceRequirements) Put(context.Context, *secretspb.SecretPutRequest) (*secretspb.SecretPutResponse, error) {
	return nil, s.illegalRuntimeMethodCall("Secrets::Put")
}

// KeyValue
func (s *ServiceRequirements) GetValue(context.Context, *kvstorepb.KvStoreGetValueRequest) (*kvstorepb.KvStoreGetValueResponse, error) {
	return nil, s.illegalRuntimeMethodCall("KeyValue::GetValue")
}

func (s *ServiceRequirements) SetValue(context.Context, *kvstorepb.KvStoreSetValueRequest) (*kvstorepb.KvStoreSetValueResponse, error) {
	return nil, s.illegalRuntimeMethodCall("KeyValue::SetValue")
}

func (s *ServiceRequirements) ScanKeys(*kvstorepb.KvStoreScanKeysRequest, kvstorepb.KvStore_ScanKeysServer) error {
	return s.illegalRuntimeMethodCall("KeyValue::ScanKeys")
}

func (s *ServiceRequirements) DeleteKey(context.Context, *kvstorepb.KvStoreDeleteKeyRequest) (*kvstorepb.KvStoreDeleteKeyResponse, error) {
	return nil, s.illegalRuntimeMethodCall("KeyValue::DeleteKey")
}

// SQL
func (s *ServiceRequirements) ConnectionString(context.Context, *sqlpb.SqlConnectionStringRequest) (*sqlpb.SqlConnectionStringResponse, error) {
	return nil, s.illegalRuntimeMethodCall("Sql::ConnectionString")
}

// Topics
func (s *ServiceRequirements) Publish(context.Context, *topicspb.TopicPublishRequest) (*topicspb.TopicPublishResponse, error) {
	return nil, s.illegalRuntimeMethodCall("Topic::Publish")
}

// Buckets
func (s *ServiceRequirements) Read(context.Context, *storagepb.StorageReadRequest) (*storagepb.StorageReadResponse, error) {
	return nil, s.illegalRuntimeMethodCall("Storage::Read")
}

func (s *ServiceRequirements) Write(context.Context, *storagepb.StorageWriteRequest) (*storagepb.StorageWriteResponse, error) {
	return nil, s.illegalRuntimeMethodCall("Storage::Write")
}

func (s *ServiceRequirements) Delete(context.Context, *storagepb.StorageDeleteRequest) (*storagepb.StorageDeleteResponse, error) {
	return nil, s.illegalRuntimeMethodCall("Storage::Delete")
}

func (s *ServiceRequirements) PreSignUrl(context.Context, *storagepb.StoragePreSignUrlRequest) (*storagepb.StoragePreSignUrlResponse, error) {
	return nil, s.illegalRuntimeMethodCall("Storage::PreSignUrl")
}

func (s *ServiceRequirements) ListBlobs(context.Context, *storagepb.StorageListBlobsRequest) (*storagepb.StorageListBlobsResponse, error) {
	return nil, s.illegalRuntimeMethodCall("Storage::ListBlobs")
}

func (s *ServiceRequirements) Exists(context.Context, *storagepb.StorageExistsRequest) (*storagepb.StorageExistsResponse, error) {
	return nil, s.illegalRuntimeMethodCall("Storage::Exists")
}

// Queues
func (s *ServiceRequirements) Enqueue(context.Context, *queuespb.QueueEnqueueRequest) (*queuespb.QueueEnqueueResponse, error) {
	return nil, s.illegalRuntimeMethodCall("Queue::Enqueue")
}

func (s *ServiceRequirements) Dequeue(context.Context, *queuespb.QueueDequeueRequest) (*queuespb.QueueDequeueResponse, error) {
	return nil, s.illegalRuntimeMethodCall("Queue::Dequeue")
}

func (s *ServiceRequirements) Complete(context.Context, *queuespb.QueueCompleteRequest) (*queuespb.QueueCompleteResponse, error) {
	return nil, s.illegalRuntimeMethodCall("Queue::Complete")
}
