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

	"github.com/google/uuid"
	"github.com/samber/lo"

	queuespb "github.com/nitrictech/nitric/core/pkg/proto/queues/v1"
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
	lease   *Lease
	message *queuespb.QueueMessage
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
func (l *LocalQueuesService) Enqueue(ctx context.Context, req *queuespb.QueueEnqueueRequest) (*queuespb.QueueEnqueueResponse, error) {
	l.queueLock.Lock()
	defer l.queueLock.Unlock()
	l.ensureQueue(req.QueueName)

	// queue the payloads
	l.queues[req.QueueName] = append(l.queues[req.QueueName], lo.Map(req.Messages, func(task *queuespb.QueueMessage, idx int) *QueueItem {
		return &QueueItem{
			message: task,
		}
	})...)

	return &queuespb.QueueEnqueueResponse{}, nil
}

// Receive message(s) from a queue
func (l *LocalQueuesService) Dequeue(ctx context.Context, req *queuespb.QueueDequeueRequest) (*queuespb.QueueDequeueResponse, error) {
	l.queueLock.Lock()
	defer l.queueLock.Unlock()
	l.ensureQueue(req.QueueName)

	if req.Depth < 1 {
		return nil, fmt.Errorf("invalid depth: %d cannot be less than one", req.Depth)
	} else if req.Depth > 10 {
		return nil, fmt.Errorf("invalid depth: %d cannot be greater than 10", req.Depth)
	}

	resp := &queuespb.QueueDequeueResponse{
		Messages: []*queuespb.DequeuedMessage{},
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

		resp.Messages = append(resp.Messages, &queuespb.DequeuedMessage{
			LeaseId: queueItem.lease.Id,
			Message: queueItem.message,
		})

		if len(resp.Messages) >= int(req.Depth) {
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

	completeTime := time.Now()

	// find the leased task
	for i, queueItem := range l.queues[req.QueueName] {
		if queueItem.lease != nil && queueItem.lease.Id == req.LeaseId {
			if completeTime.Before(queueItem.lease.Expiry) {
				// remove the leased task
				l.queues[req.QueueName] = append(l.queues[req.QueueName][:i], l.queues[req.QueueName][i+1:]...)
				return &queuespb.QueueCompleteResponse{}, nil
			}

			return nil, fmt.Errorf("LeaseId: %s expired at %s, current time %s", req.LeaseId, queueItem.lease.Expiry, completeTime)
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
