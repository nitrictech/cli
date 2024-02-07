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

package topics

import (
	"context"
	"fmt"
	"maps"
	"sync"
	"time"

	"github.com/asaskevich/EventBus"
	"google.golang.org/grpc/codes"

	"github.com/nitrictech/cli/pkgplus/grpcx"

	grpc_errors "github.com/nitrictech/nitric/core/pkg/grpc/errors"
	topicspb "github.com/nitrictech/nitric/core/pkg/proto/topics/v1"
	"github.com/nitrictech/nitric/core/pkg/workers/topics"
)

type (
	topicName   = string
	serviceName = string
)

type State = map[topicName]map[serviceName]int

type LocalTopicsAndSubscribersService struct {
	*topics.SubscriberManager
	subscribers State

	subscribersLock sync.RWMutex

	bus EventBus.Bus
}

type ActionState struct {
	TopicName string
	Payload   string
	Success   bool
}

var (
	_ topicspb.TopicsServer     = (*LocalTopicsAndSubscribersService)(nil)
	_ topicspb.SubscriberServer = (*LocalTopicsAndSubscribersService)(nil)
)

const localTopicsTopic = "local_topics"

const localTopicsDeliveryTopic = "local_topics_delivery"

func (s *LocalTopicsAndSubscribersService) publishState() {
	s.bus.Publish(localTopicsTopic, maps.Clone(s.subscribers))
}

func (s *LocalTopicsAndSubscribersService) SubscribeToState(subscription func(State)) {
	s.bus.Subscribe(localTopicsTopic, subscription)
}

func (s *LocalTopicsAndSubscribersService) publishAction(action ActionState) {
	s.bus.Publish(localTopicsDeliveryTopic, action)
}

func (s *LocalTopicsAndSubscribersService) SubscribeToAction(subscription func(ActionState)) {
	s.bus.Subscribe(localTopicsDeliveryTopic, subscription)
}

func (s *LocalTopicsAndSubscribersService) GetSubscribers() map[string]map[string]int {
	s.subscribersLock.RLock()
	defer s.subscribersLock.RUnlock()

	return s.subscribers
}

func (s *LocalTopicsAndSubscribersService) registerSubscriber(serviceName string, registration *topicspb.RegistrationRequest) {
	s.subscribersLock.Lock()
	defer s.subscribersLock.Unlock()

	if s.subscribers[registration.TopicName] == nil {
		s.subscribers[registration.TopicName] = make(map[string]int)
	}

	s.subscribers[registration.TopicName][serviceName]++

	s.publishState()
}

func (s *LocalTopicsAndSubscribersService) unregisterSubscriber(serviceName string, registration *topicspb.RegistrationRequest) {
	s.subscribersLock.Lock()
	defer s.subscribersLock.Unlock()

	if s.subscribers[registration.TopicName] == nil {
		s.subscribers[registration.TopicName] = make(map[string]int)
	}

	s.subscribers[registration.TopicName][serviceName]--

	if s.subscribers[registration.TopicName][serviceName] == 0 {
		delete(s.subscribers, registration.TopicName)
	}

	s.publishState()
}

// Subscribe to a topic and handle incoming messages
func (s *LocalTopicsAndSubscribersService) Subscribe(stream topicspb.Subscriber_SubscribeServer) error {
	serviceName, err := grpcx.GetServiceNameFromStream(stream)
	if err != nil {
		return err
	}

	peekableStream := grpcx.NewPeekableStreamServer[*topicspb.ServerMessage, *topicspb.ClientMessage](stream)

	firstRequest, err := peekableStream.Peek()
	if err != nil {
		return err
	}

	if firstRequest.GetRegistrationRequest() == nil {
		return fmt.Errorf("first request must be a registration request")
	}

	// TODO: move to common validation decorators and send grpc invalid argument error
	if firstRequest.GetRegistrationRequest().TopicName == "" {
		return fmt.Errorf("topic name must be specified")
	}

	stream.Send(&topicspb.ServerMessage{
		Id: firstRequest.Id,
		Content: &topicspb.ServerMessage_RegistrationResponse{
			RegistrationResponse: &topicspb.RegistrationResponse{},
		},
	})

	// Keep track of our local topic subscriptions
	s.registerSubscriber(serviceName, firstRequest.GetRegistrationRequest())
	defer s.unregisterSubscriber(serviceName, firstRequest.GetRegistrationRequest())

	return s.SubscriberManager.Subscribe(peekableStream)
}

func (s *LocalTopicsAndSubscribersService) deliverEvent(ctx context.Context, req *topicspb.TopicPublishRequest) error {
	msg := &topicspb.ServerMessage{
		Content: &topicspb.ServerMessage_MessageRequest{
			MessageRequest: &topicspb.MessageRequest{
				TopicName: req.TopicName,
				Message: &topicspb.Message{
					Content: &topicspb.Message_StructPayload{
						StructPayload: req.Message.GetStructPayload(),
					},
				},
			},
		},
	}
	resp, err := s.SubscriberManager.HandleRequest(msg)
	if err != nil {
		return err
	}

	json, err := req.Message.GetStructPayload().MarshalJSON()
	if err != nil {
		return err
	}

	s.publishAction(ActionState{
		TopicName: req.TopicName,
		Success:   resp.GetMessageResponse().GetSuccess(),
		Payload:   string(json),
	})

	return err
}

// Publish a message to a given topic
func (s *LocalTopicsAndSubscribersService) Publish(ctx context.Context, req *topicspb.TopicPublishRequest) (*topicspb.TopicPublishResponse, error) {
	newErr := grpc_errors.ErrorsWithScope("WorkerPoolEventService.Publish")

	if req.Delay != nil {
		// TODO: Implement a signal from the front end that allows for the early release of delayed events (by their ID)
		// FIXME: We want the event to appear straight away in the history table (maybe as a new event type that counts down)
		go func(evt *topicspb.TopicPublishRequest) {
			// Wait to deliver the events
			time.Sleep(req.Delay.AsDuration())
			s.deliverEvent(ctx, evt)
		}(req)
	} else {
		err := s.deliverEvent(ctx, req)
		if err != nil {
			return nil, newErr(
				codes.Internal,
				"could not publish event",
				err,
			)
		}
	}

	return &topicspb.TopicPublishResponse{}, nil
}

// Create new Dev EventService
func NewLocalTopicsService() (*LocalTopicsAndSubscribersService, error) {
	return &LocalTopicsAndSubscribersService{
		SubscriberManager: topics.New(),
		subscribersLock:   sync.RWMutex{},
		subscribers:       make(map[string]map[string]int),
		bus:               EventBus.New(),
	}, nil
}
