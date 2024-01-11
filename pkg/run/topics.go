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

package run

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nitrictech/cli/pkg/dashboard/history"
	"github.com/nitrictech/cli/pkg/eventbus"
	"google.golang.org/grpc/codes"

	grpc_errors "github.com/nitrictech/nitric/core/pkg/grpc/errors"
	topicspb "github.com/nitrictech/nitric/core/pkg/proto/topics/v1"
	"github.com/nitrictech/nitric/core/pkg/workers/topics"
)

type WorkerPoolEventService struct {
	*topics.SubscriberManager
	subscribers map[string]int

	subscribersLock sync.RWMutex
}

var _ topicspb.TopicsServer = (*WorkerPoolEventService)(nil)
var _ topicspb.SubscriberServer = (*WorkerPoolEventService)(nil)

func (s *WorkerPoolEventService) GetSubscribers() map[string]int {
	s.subscribersLock.RLock()
	defer s.subscribersLock.RUnlock()

	return s.subscribers
}

func (s *WorkerPoolEventService) registerSubscriber(registration *topicspb.RegistrationRequest) {
	s.subscribersLock.Lock()
	defer s.subscribersLock.Unlock()

	s.subscribers[registration.TopicName]++
}

func (s *WorkerPoolEventService) unregisterSubscriber(registration *topicspb.RegistrationRequest) {
	s.subscribersLock.Lock()
	defer s.subscribersLock.Unlock()

	s.subscribers[registration.TopicName]--
}

// Subscribe to a topic and handle incoming messages
func (s *WorkerPoolEventService) Subscribe(stream topicspb.Subscriber_SubscribeServer) error {

	firstRequest, err := stream.Recv()
	if err != nil {
		return err
	}

	if firstRequest.GetRegistrationRequest() == nil {
		// first request MUST be a registration request
		return fmt.Errorf("first request must be a registration request")
	}

	stream.Send(&topicspb.ServerMessage{
		Id: firstRequest.Id,
		Content: &topicspb.ServerMessage_RegistrationResponse{
			RegistrationResponse: &topicspb.RegistrationResponse{},
		},
	})

	// Keep track of our local topic subscriptions
	s.registerSubscriber(firstRequest.GetRegistrationRequest())
	defer s.unregisterSubscriber(firstRequest.GetRegistrationRequest())

	// we've got the worker details, lets get the subcribed
	topicName := firstRequest.GetRegistrationRequest().TopicName

	eventbus.TopicBus().SubscribeAsync(topicName, func(req *topicspb.ServerMessage) {
		err := stream.Send(req)
		if err != nil {
			fmt.Println("problem sending the event")
		}
	}, false)

	for {
		// log responses
		// problem processing the event
		msg, err := stream.Recv()
		if err != nil {
			return err
		}

		resp := msg.GetMessageResponse()
		if resp == nil {
			return fmt.Errorf("expected message response")
		}

		// TODO: Add successfully handled history event
		eventbus.Bus().Publish(history.AddRecordTopic, &history.HistoryEvent[history.TopicEvent]{
			Time:       time.Now().UnixMilli(),
			RecordType: history.TOPIC,
			Event: history.TopicEvent{
				Id:    msg.Id,
				Topic: topicName,
				Result: &history.TopicSubscriberResultEvent{
					Success: msg.GetMessageResponse().Success,
				},
			},
		})
	}
}

func (s *WorkerPoolEventService) deliverEvent(ctx context.Context, req *topicspb.TopicPublishRequest) error {
	jsonPayload, err := req.Message.GetStructPayload().MarshalJSON()
	if err != nil {
		return err
	}

	// Other message brokers generate their own IDs, we simulate that with a basic uuid.
	messageId := uuid.New().String()

	// Send to dashboard here.... (assign an ID to the individual)
	eventbus.Bus().Publish(history.AddRecordTopic, &history.HistoryEvent[history.TopicEvent]{
		Time:       time.Now().UnixMilli(),
		RecordType: history.TOPIC,
		Event: history.TopicEvent{
			Id:      messageId,
			Topic:   req.TopicName,
			Publish: &history.TopicPublishEvent{Payload: string(jsonPayload)},
		},
	})

	fmt.Printf("Publishing to %s topic, %d subscriber(s)\n", req.TopicName, s.WorkerCount())

	eventbus.TopicBus().Publish(req.TopicName, &topicspb.ServerMessage{
		Id: messageId,
		Content: &topicspb.ServerMessage_MessageRequest{
			MessageRequest: &topicspb.MessageRequest{
				TopicName: req.TopicName,
				Message:   req.Message,
			},
		},
	})

	return nil
}

// Publish a message to a given topic
func (s *WorkerPoolEventService) Publish(ctx context.Context, req *topicspb.TopicPublishRequest) (*topicspb.TopicPublishResponse, error) {
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
func NewEvents() (*WorkerPoolEventService, error) {
	return &WorkerPoolEventService{
		subscribers: make(map[string]int),
	}, nil
}
