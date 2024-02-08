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

package queues

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/google/uuid"
	queuespb "github.com/nitrictech/nitric/core/pkg/proto/queues/v1"
	"github.com/samber/lo"
)

type (
	queueName   = string
	serviceName = string
)

type State = map[queueName]map[serviceName]int

type Lease struct {
	Id     string
	Expiry time.Time
}

type QueueItem struct {
	lease *Lease
	task  *structpb.Struct
}

type LocalQueuesService struct {
	queueLock sync.Mutex

	queues map[queueName][]*QueueItem
}

var (
	_                        queuespb.QueuesServer = (*LocalQueuesService)(nil)
	defaultVisibilityTimeout                       = 30 * time.Second
)

func (l *LocalQueuesService) ensureQueue(queueName string) {
	if _, ok := l.queues[queueName]; !ok {
		l.queues[queueName] = []*QueueItem{}
	}
}

// Send messages to a queue
func (l *LocalQueuesService) Send(ctx context.Context, req *queuespb.QueueSendRequestBatch) (*queuespb.QueueSendResponse, error) {
	l.queueLock.Lock()
	defer l.queueLock.Unlock()
	l.ensureQueue(req.QueueName)

	// queue the payloads
	l.queues[req.QueueName] = append(l.queues[req.QueueName], lo.Map(req.Requests, func(task *queuespb.QueueSendRequest, idx int) *QueueItem {
		return &QueueItem{
			task: task.Payload,
		}
	})...)

	return &queuespb.QueueSendResponse{}, nil
}

// Receive message(s) from a queue
func (l *LocalQueuesService) Receive(ctx context.Context, req *queuespb.QueueReceiveRequest) (*queuespb.QueueReceiveResponse, error) {
	l.queueLock.Lock()
	defer l.queueLock.Unlock()
	l.ensureQueue(req.QueueName)

	if req.Depth < 1 {
		return nil, fmt.Errorf("invalid depth: %d cannot be less than one", req.Depth)
	} else if req.Depth > 10 {
		return nil, fmt.Errorf("invalid depth: %d cannot be greater than 10", req.Depth)
	}

	resp := &queuespb.QueueReceiveResponse{
		Tasks: []*queuespb.ReceivedTask{},
	}

	// remove the leased tasks from the queue
	for _, queueItem := range l.queues[req.QueueName] {
		if queueItem.lease != nil && queueItem.lease.Expiry.After(time.Now()) {
			// the task is still leased, so it's not available
			continue
		}

		queueItem.lease = &Lease{
			Id:     uuid.New().String(),
			Expiry: time.Now().Add(defaultVisibilityTimeout),
		}

		resp.Tasks = append(resp.Tasks, &queuespb.ReceivedTask{
			LeaseId: queueItem.lease.Id,
			Payload: queueItem.task,
		})

		if len(resp.Tasks) >= int(req.Depth) {
			break
		}
	}

	return resp, nil
}

// Complete an item previously popped from a queue
func (l *LocalQueuesService) Complete(ctx context.Context, req *queuespb.QueueCompleteRequest) (*queuespb.QueueCompleteResponse, error) {
	l.queueLock.Lock()
	defer l.queueLock.Unlock()
	l.ensureQueue(req.QueueName)

	// find the leased task
	for i, queueItem := range l.queues[req.QueueName] {
		if queueItem.lease != nil && queueItem.lease.Id == req.LeaseId {
			if time.Now().Before(queueItem.lease.Expiry) {
				// remove the leased task
				l.queues[req.QueueName] = append(l.queues[req.QueueName][:i], l.queues[req.QueueName][i+1:]...)
				return &queuespb.QueueCompleteResponse{}, nil
			}
			return nil, fmt.Errorf("LeaseId: %s expired at %s, current time %s", req.LeaseId, queueItem.lease.Expiry, time.Now())
		}
	}

	return nil, fmt.Errorf("LeaseId: %s not found", req.LeaseId)
}

// Create new Dev EventService
func NewLocalQueuesService() (*LocalQueuesService, error) {
	queueService := &LocalQueuesService{
		queues: map[queueName][]*QueueItem{},
	}

	return queueService, nil
}
