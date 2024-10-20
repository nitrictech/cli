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

package batch

import (
	"context"
	"fmt"
	"maps"
	"sync"

	"github.com/asaskevich/EventBus"

	"github.com/nitrictech/cli/pkg/cloud/errorsx"
	"github.com/nitrictech/cli/pkg/grpcx"
	"github.com/nitrictech/cli/pkg/validation"
	"github.com/nitrictech/nitric/core/pkg/logger"
	batchpb "github.com/nitrictech/nitric/core/pkg/proto/batch/v1"
	resourcespb "github.com/nitrictech/nitric/core/pkg/proto/resources/v1"
	"github.com/nitrictech/nitric/core/pkg/workers/jobs"
)

type BatchRunner func(req *batchpb.JobSubmitRequest) error

type (
	jobName     = string
	serviceName = string
)

type ActionState struct {
	JobName string
	Payload string
	Success bool
}

type (
	State             = map[jobName]map[serviceName]int
	LocalBatchService struct {
		*jobs.JobManager
		batchpb.UnimplementedBatchServer

		errorLogger errorsx.ServiceErrorLogger
		state       State
		batchLock   sync.RWMutex

		bus EventBus.Bus
	}
)

var (
	_ batchpb.BatchServer = (*LocalBatchService)(nil)
	_ batchpb.JobServer   = (*LocalBatchService)(nil)
)

const (
	localBatchTopic         = "local-batch"
	localBatchDeliveryTopic = "local-batch-delivery"
)

func (l *LocalBatchService) SubscribeToState(subscriberFunction func(State)) {
	// ignore the error, it's only returned if the fn param isn't a function
	_ = l.bus.Subscribe(localBatchTopic, subscriberFunction)
}

func (l *LocalBatchService) publishState() {
	l.bus.Publish(localBatchTopic, l.GetState())
}

func (l *LocalBatchService) GetState() State {
	return maps.Clone(l.state)
}

func (l *LocalBatchService) publishAction(action ActionState) {
	l.bus.Publish(localBatchDeliveryTopic, action)
}

func (l *LocalBatchService) SubscribeToAction(subscription func(ActionState)) {
	// ignore the error, it's only returned if the fn param isn't a function
	_ = l.bus.Subscribe(localBatchDeliveryTopic, subscription)
}

func (l *LocalBatchService) registerJob(serviceName string, registration *batchpb.RegistrationRequest) {
	l.batchLock.Lock()
	defer l.batchLock.Unlock()

	if !validation.IsValidResourceName(registration.JobName) {
		l.errorLogger(
			serviceName,
			fmt.Errorf("invalid name: \"%s\" for %s resource", registration.JobName, resourcespb.ResourceType_Job),
		)
		return
	}

	if l.state[registration.JobName] == nil {
		l.state[registration.JobName] = make(map[string]int)
	}

	l.state[registration.JobName][serviceName]++

	l.publishState()
}

func (l *LocalBatchService) unregisterJob(serviceName string, registration *batchpb.RegistrationRequest) {
	l.batchLock.Lock()
	defer l.batchLock.Unlock()

	if l.state[registration.JobName] == nil {
		l.state[registration.JobName] = make(map[string]int)
	}

	l.state[registration.JobName][serviceName]--

	if l.state[registration.JobName][serviceName] == 0 {
		delete(l.state, registration.JobName)
	}

	l.publishState()
}

func (l *LocalBatchService) HandleJob(stream batchpb.Job_HandleJobServer) error {
	serviceName, err := grpcx.GetServiceNameFromStream(stream)
	if err != nil {
		return err
	}

	peekableStream := grpcx.NewPeekableStreamServer[*batchpb.ServerMessage, *batchpb.ClientMessage](stream)

	firstRequest, err := peekableStream.Peek()
	if err != nil {
		return err
	}

	if firstRequest.GetRegistrationRequest() == nil {
		return fmt.Errorf("first request must be a registration request")
	}

	err = stream.Send(&batchpb.ServerMessage{
		Id: firstRequest.Id,
		Content: &batchpb.ServerMessage_RegistrationResponse{
			RegistrationResponse: &batchpb.RegistrationResponse{},
		},
	})
	if err != nil {
		return err
	}

	// Keep track of our local batch subscriptions
	l.registerJob(serviceName, firstRequest.GetRegistrationRequest())
	defer l.unregisterJob(serviceName, firstRequest.GetRegistrationRequest())

	return l.JobManager.HandleJob(peekableStream)
}

func (l *LocalBatchService) SubmitJob(ctx context.Context, req *batchpb.JobSubmitRequest) (*batchpb.JobSubmitResponse, error) {
	go func() {
		json, err := req.Data.GetStruct().MarshalJSON()
		if err != nil {
			logger.Errorf("Error marshalling job request data: %s", err.Error())
		}

		_, err = l.HandleJobRequest(&batchpb.ServerMessage{
			Content: &batchpb.ServerMessage_JobRequest{
				JobRequest: &batchpb.JobRequest{
					JobName: req.GetJobName(),
					Data:    req.Data,
				},
			},
		})
		if err != nil {
			logger.Errorf("Error handling job request: %s", err.Error())

			l.publishAction(ActionState{
				JobName: req.GetJobName(),
				Success: false,
				Payload: string(json),
			})

			return
		}

		l.publishAction(ActionState{
			JobName: req.GetJobName(),
			Success: true,
			Payload: string(json),
		})
	}()

	return &batchpb.JobSubmitResponse{}, nil
}

func NewLocalBatchService(errorLogger errorsx.ServiceErrorLogger) *LocalBatchService {
	return &LocalBatchService{
		errorLogger: errorLogger,
		JobManager:  jobs.New(),
		state:       make(map[string]map[string]int),
		bus:         EventBus.New(),
	}
}
