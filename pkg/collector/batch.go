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
	"errors"
	"fmt"
	"sync"

	"google.golang.org/grpc"

	"github.com/nitrictech/cli/pkg/validation"
	"github.com/nitrictech/cli/pkg/view/tui/components/view"
	apispb "github.com/nitrictech/nitric/core/pkg/proto/apis/v1"
	batchpb "github.com/nitrictech/nitric/core/pkg/proto/batch/v1"
	httppb "github.com/nitrictech/nitric/core/pkg/proto/http/v1"
	resourcespb "github.com/nitrictech/nitric/core/pkg/proto/resources/v1"
	schedulespb "github.com/nitrictech/nitric/core/pkg/proto/schedules/v1"
	storagepb "github.com/nitrictech/nitric/core/pkg/proto/storage/v1"
	topicspb "github.com/nitrictech/nitric/core/pkg/proto/topics/v1"
	websocketspb "github.com/nitrictech/nitric/core/pkg/proto/websockets/v1"
)

type BatchRequirements struct {
	batchName string
	batchFile string

	resourceLock sync.Mutex

	buckets        map[string]*resourcespb.BucketResource
	keyValueStores map[string]*resourcespb.KeyValueStoreResource
	topics         map[string]*resourcespb.TopicResource
	queues         map[string]*resourcespb.QueueResource
	sqlDatabases   map[string]*resourcespb.SqlDatabaseResource
	secrets        map[string]*resourcespb.SecretResource

	jobs        map[string]*resourcespb.JobResource
	jobHandlers map[string]*batchpb.RegistrationRequest

	policies []*resourcespb.PolicyResource

	errors []error
	topicspb.UnimplementedTopicsServer
	storagepb.UnimplementedStorageListenerServer
	websocketspb.UnimplementedWebsocketServer

	ApiServer apispb.ApiServer
}

// Error - Returns an error if any requirements have been registered incorrectly, such as duplicates
func (s *BatchRequirements) Error() error {
	if len(s.errors) > 0 {
		errorView := view.New()
		errorView.Addln("Errors found in batch %s", s.batchFile)

		for _, err := range s.errors {
			errorView.Addln("- %s", err.Error())
		}

		return errors.New(errorView.Render())
	}

	return nil
}

// TODO: Remove when databases are no longer in preview
func (s *BatchRequirements) HasDatabases() bool {
	return len(s.sqlDatabases) > 0
}

func (s *BatchRequirements) RegisterServices(grpcServer *grpc.Server) {
	batchpb.RegisterJobServer(grpcServer, s)
	resourcespb.RegisterResourcesServer(grpcServer, s)
	apispb.RegisterApiServer(grpcServer, s.ApiServer)
	schedulespb.RegisterSchedulesServer(grpcServer, s)
	topicspb.RegisterTopicsServer(grpcServer, s)
	topicspb.RegisterSubscriberServer(grpcServer, s)
	websocketspb.RegisterWebsocketHandlerServer(grpcServer, s)
	storagepb.RegisterStorageListenerServer(grpcServer, s)
	httppb.RegisterHttpServer(grpcServer, s)
}

func (s *BatchRequirements) Declare(ctx context.Context, req *resourcespb.ResourceDeclareRequest) (*resourcespb.ResourceDeclareResponse, error) {
	s.resourceLock.Lock()
	defer s.resourceLock.Unlock()

	if !validation.IsValidResourceName(req.Id.GetName()) {
		s.errors = append(s.errors, validation.NewResourceNameViolationError(req.Id.Name, req.Id.Type.String()))
	}

	switch req.Id.Type {
	case resourcespb.ResourceType_Bucket:
		// Add a bucket
		s.buckets[req.Id.GetName()] = req.GetBucket()
	case resourcespb.ResourceType_KeyValueStore:
		// Add a key/value store
		s.keyValueStores[req.Id.GetName()] = req.GetKeyValueStore()
	case resourcespb.ResourceType_Api:
		// Discard and ignore for batches
	case resourcespb.ResourceType_ApiSecurityDefinition:
		// Discard and ignore for batches
	case resourcespb.ResourceType_Secret:
		// Add a secret
		s.secrets[req.Id.GetName()] = req.GetSecret()
	case resourcespb.ResourceType_SqlDatabase:
		// Add a sql database
		s.sqlDatabases[req.Id.GetName()] = req.GetSqlDatabase()
	case resourcespb.ResourceType_Policy:
		// Services don't know their own name, so we need to add it here
		if req.GetPolicy().GetPrincipals() == nil {
			req.GetPolicy().Principals = []*resourcespb.ResourceIdentifier{{
				Name: s.batchName,
				Type: resourcespb.ResourceType_Batch,
			}}
		} else {
			for _, principal := range req.GetPolicy().GetPrincipals() {
				if principal.GetName() == "" && principal.GetType() == resourcespb.ResourceType_Service {
					principal.Name = s.batchName
					principal.Type = resourcespb.ResourceType_Batch
				}
			}
		}

		// Add a policy
		s.policies = append(s.policies, req.GetPolicy())
	case resourcespb.ResourceType_Topic:
		// add a topic
		s.topics[req.Id.GetName()] = req.GetTopic()
	case resourcespb.ResourceType_Queue:
		// add a queue
		s.queues[req.Id.GetName()] = req.GetQueue()
	case resourcespb.ResourceType_Job:
		// add a job
		s.jobs[req.Id.GetName()] = req.GetJob()
	}

	return &resourcespb.ResourceDeclareResponse{}, nil
}

func (s *BatchRequirements) HandleJob(stream batchpb.Job_HandleJobServer) error {
	s.resourceLock.Lock()
	defer s.resourceLock.Unlock()

	msg, err := stream.Recv()
	if err != nil {
		return err
	}

	registrationRequest := msg.GetRegistrationRequest()
	if registrationRequest == nil {
		return fmt.Errorf("first message must be a registration request")
	}

	s.jobHandlers[registrationRequest.JobName] = registrationRequest

	return stream.Send(&batchpb.ServerMessage{
		Content: &batchpb.ServerMessage_RegistrationResponse{
			RegistrationResponse: &batchpb.RegistrationResponse{},
		},
	})
}

func (s *BatchRequirements) HandleEvents(stream websocketspb.WebsocketHandler_HandleEventsServer) error {
	s.resourceLock.Lock()
	defer s.resourceLock.Unlock()

	_, err := stream.Recv()
	if err != nil {
		return err
	}

	s.errors = append(s.errors, fmt.Errorf("websocket handler declared in Batch %s, batches cannot handle Websocket events", s.batchFile))

	return stream.Send(&websocketspb.ServerMessage{
		Content: &websocketspb.ServerMessage_RegistrationResponse{
			RegistrationResponse: &websocketspb.RegistrationResponse{},
		},
	})
}

func (s *BatchRequirements) Proxy(stream httppb.Http_ProxyServer) error {
	s.resourceLock.Lock()
	defer s.resourceLock.Unlock()

	_, err := stream.Recv()
	if err != nil {
		return err
	}

	s.errors = append(s.errors, fmt.Errorf("HTTP Proxy declared in Batch %s, batches cannot handle HTTP servers", s.batchFile))

	return nil
}

func (s *BatchRequirements) Serve(stream apispb.Api_ServeServer) error {
	s.resourceLock.Lock()
	defer s.resourceLock.Unlock()

	_, err := stream.Recv()
	if err != nil {
		return err
	}

	s.errors = append(s.errors, fmt.Errorf("API route declared in Batch %s, batches cannot handle API requests", s.batchFile))

	// Send a registration response
	return stream.Send(&apispb.ServerMessage{
		Content: &apispb.ServerMessage_RegistrationResponse{
			RegistrationResponse: &apispb.RegistrationResponse{},
		},
	})
}

func (s *BatchRequirements) Schedule(stream schedulespb.Schedules_ScheduleServer) error {
	s.resourceLock.Lock()
	defer s.resourceLock.Unlock()

	_, err := stream.Recv()
	if err != nil {
		return err
	}

	s.errors = append(s.errors, fmt.Errorf("Schedule declared in Batch %s, batches cannot currently handle schedules", s.batchFile))

	return stream.Send(&schedulespb.ServerMessage{
		Content: &schedulespb.ServerMessage_RegistrationResponse{
			RegistrationResponse: &schedulespb.RegistrationResponse{},
		},
	})
}

func (s *BatchRequirements) Subscribe(stream topicspb.Subscriber_SubscribeServer) error {
	s.resourceLock.Lock()
	defer s.resourceLock.Unlock()

	_, err := stream.Recv()
	if err != nil {
		return err
	}

	s.errors = append(s.errors, fmt.Errorf("topic subscription declared in Batch %s, batches cannot handle topic subscriptions", s.batchFile))

	return stream.Send(&topicspb.ServerMessage{
		Content: &topicspb.ServerMessage_RegistrationResponse{
			RegistrationResponse: &topicspb.RegistrationResponse{},
		},
	})
}

func NewBatchRequirements(serviceName string, serviceFile string) *BatchRequirements {
	requirements := &BatchRequirements{
		batchName:      serviceName,
		batchFile:      serviceFile,
		resourceLock:   sync.Mutex{},
		jobHandlers:    make(map[string]*batchpb.RegistrationRequest),
		jobs:           make(map[string]*resourcespb.JobResource),
		buckets:        make(map[string]*resourcespb.BucketResource),
		keyValueStores: make(map[string]*resourcespb.KeyValueStoreResource),
		topics:         make(map[string]*resourcespb.TopicResource),
		policies:       []*resourcespb.PolicyResource{},
		secrets:        make(map[string]*resourcespb.SecretResource),
		sqlDatabases:   make(map[string]*resourcespb.SqlDatabaseResource),
		queues:         make(map[string]*resourcespb.QueueResource),
		errors:         []error{},
	}

	return requirements
}
