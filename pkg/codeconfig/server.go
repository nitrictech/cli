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
	"context"
	"fmt"

	apispb "github.com/nitrictech/nitric/core/pkg/proto/apis/v1"
	httppb "github.com/nitrictech/nitric/core/pkg/proto/http/v1"
	resourcespb "github.com/nitrictech/nitric/core/pkg/proto/resources/v1"
	schedulespb "github.com/nitrictech/nitric/core/pkg/proto/schedules/v1"
	storagepb "github.com/nitrictech/nitric/core/pkg/proto/storage/v1"
	topicspb "github.com/nitrictech/nitric/core/pkg/proto/topics/v1"
	websocketspb "github.com/nitrictech/nitric/core/pkg/proto/websockets/v1"
)

type Server struct {
	name     string
	function *FunctionDependencies

	// Leave resourcespb::Details unimplemented
	resourcespb.UnimplementedResourcesServer
}

var _ storagepb.StorageListenerServer = (*Server)(nil)
var _ topicspb.SubscriberServer = (*Server)(nil)
var _ websocketspb.WebsocketHandlerServer = (*Server)(nil)
var _ schedulespb.SchedulesServer = (*Server)(nil)
var _ apispb.ApiServer = (*Server)(nil)
var _ httppb.HttpServer = (*Server)(nil)
var _ resourcespb.ResourcesServer = (*Server)(nil)

// Listen for storage notifications
func (s *Server) Listen(stream storagepb.StorageListener_ListenServer) error {
	firstRequest, err := stream.Recv()
	if err != nil {
		return err
	}

	registrationRequest := firstRequest.GetRegistrationRequest()
	if registrationRequest == nil {
		return fmt.Errorf("")
	}

	s.function.AddBucketNotificationHandler(registrationRequest)
	return nil
}

// Subscribe to topics
func (s *Server) Subscribe(stream topicspb.Subscriber_SubscribeServer) error {
	firstRequest, err := stream.Recv()
	if err != nil {
		return err
	}

	registrationRequest := firstRequest.GetRegistrationRequest()
	if registrationRequest == nil {
		return fmt.Errorf("")
	}

	s.function.AddSubscriptionHandler(registrationRequest)
	return nil
}

// Handle websocket events
func (s *Server) HandleEvents(stream websocketspb.WebsocketHandler_HandleEventsServer) error {
	firstRequest, err := stream.Recv()
	if err != nil {
		return err
	}

	registrationRequest := firstRequest.GetRegistrationRequest()
	if registrationRequest == nil {
		return fmt.Errorf("")
	}

	s.function.AddWebsocketHandler(registrationRequest)
	return nil
}

// Make schedule
func (s *Server) Schedule(stream schedulespb.Schedules_ScheduleServer) error {
	firstRequest, err := stream.Recv()
	if err != nil {
		return err
	}

	registrationRequest := firstRequest.GetRegistrationRequest()
	if registrationRequest == nil {
		return fmt.Errorf("")
	}

	s.function.AddScheduleHandler(registrationRequest)
	return nil
}

func (s *Server) Serve(stream apispb.Api_ServeServer) error {
	firstRequest, err := stream.Recv()
	if err != nil {
		return err
	}

	registrationRequest := firstRequest.GetRegistrationRequest()
	if registrationRequest == nil {
		return fmt.Errorf("")
	}

	s.function.AddApiHandler(registrationRequest)
	return nil
}

func (s *Server) Proxy(ctx context.Context, req *httppb.HttpProxyRequest) (*httppb.HttpProxyResponse, error) {
	s.function.AddHttpWorker(req)
	return &httppb.HttpProxyResponse{}, nil
}

// // Declare - Accepts resource declarations, adding them as dependencies to the Function
func (s *Server) Declare(ctx context.Context, req *resourcespb.ResourceDeclareRequest) (*resourcespb.ResourceDeclareResponse, error) {
	switch req.Resource.Type {
	case resourcespb.ResourceType_Bucket:
		s.function.AddBucket(req.Resource.Name, req.GetBucket())
	case resourcespb.ResourceType_Collection:
		s.function.AddCollection(req.Resource.Name, req.GetCollection())
	case resourcespb.ResourceType_Topic:
		s.function.AddTopic(req.Resource.Name, req.GetTopic())
	case resourcespb.ResourceType_Policy:
		s.function.AddPolicy(req.GetPolicy())
	case resourcespb.ResourceType_Secret:
		s.function.AddSecret(req.Resource.Name, req.GetSecret())
	case resourcespb.ResourceType_Api:
		// FIXME: Make sure this is correct
		// s.function.AddApiSecurityDefinitions(req.Resource.Name, req.GetApi().SecurityDefinitions)
		s.function.AddApiSecurity(req.Resource.Name, req.GetApi().Security)
	case resourcespb.ResourceType_Websocket:
		// TODO: Add websocket configuration here when available
		break
	}

	return &resourcespb.ResourceDeclareResponse{}, nil
}

// 	return &v1.ResourceDeclareResponse{}, nil
// }

// NewServer - Creates a new deployment server
func NewServer(name string, function *FunctionDependencies) *Server {
	return &Server{
		name:     name,
		function: function,
	}
}
