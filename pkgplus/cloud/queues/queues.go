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
	"slices"
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

type LeasedTask struct {
	leasedTime time.Time
	content    *structpb.Struct
}

type LocalQueuesService struct {
	queueLock sync.Mutex

	leased map[queueName]map[string]*LeasedTask
	queues map[queueName][]*structpb.Struct
}

var (
	_ queuespb.QueuesServer = (*LocalQueuesService)(nil)
)

func (l *LocalQueuesService) ensureQueue(queueName string) {
	if _, ok := l.queues[queueName]; !ok {
		l.queues[queueName] = []*structpb.Struct{}
	}

	if _, ok := l.leased[queueName]; !ok {
		l.leased[queueName] = map[string]*LeasedTask{}
	}
}

// Send messages to a queue
func (l *LocalQueuesService) Send(ctx context.Context, req *queuespb.QueueSendRequestBatch) (*queuespb.QueueSendResponse, error) {
	l.queueLock.Lock()
	defer l.queueLock.Unlock()
	l.ensureQueue(req.QueueName)

	// queue the payloads
	l.queues[req.QueueName] = append(l.queues[req.QueueName], lo.Map(req.Requests, func(task *queuespb.QueueSendRequest, idx int) *structpb.Struct {
		return task.Payload
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

	leasedTasks := l.queues[req.QueueName][0:lo.Max([]int{len(l.queues[req.QueueName]), int(req.Depth)})]

	resp := &queuespb.QueueReceiveResponse{
		Tasks: []*queuespb.ReceivedTask{},
	}

	// remove the leased tasks from the queue
	for _, leasedTask := range leasedTasks {
		l.queues[req.QueueName] = slices.DeleteFunc(l.queues[req.QueueName], func(item *structpb.Struct) bool {
			return item == leasedTask
		})

		leaseId := uuid.New()

		l.leased[req.QueueName][leaseId.String()] = &LeasedTask{
			leasedTime: time.Now(),
			content:    leasedTask,
		}

		resp.Tasks = append(resp.Tasks, &queuespb.ReceivedTask{
			LeaseId: leaseId.String(),
			Payload: leasedTask,
		})
	}

	return resp, nil
}

// Complete an item previously popped from a queue
func (l *LocalQueuesService) Complete(ctx context.Context, req *queuespb.QueueCompleteRequest) (*queuespb.QueueCompleteResponse, error) {
	l.queueLock.Lock()
	defer l.queueLock.Unlock()
	l.ensureQueue(req.QueueName)

	_, ok := l.leased[req.QueueName][req.LeaseId]

	if !ok {
		return nil, fmt.Errorf("LeaseId: %s not found", req.LeaseId)
	}

	// remove the leased task
	delete(l.leased[req.QueueName], req.LeaseId)

	return &queuespb.QueueCompleteResponse{}, nil
}

func (l *LocalQueuesService) Requeue() {
	l.queueLock.Lock()
	defer l.queueLock.Unlock()

	for queueName, queue := range l.leased {
		for serviceName, task := range queue {
			if time.Since(task.leasedTime) > (time.Second * 30) {
				// re-queue the task
				l.queues[queueName] = append(l.queues[queueName], task.content)
				delete(l.leased[queueName], serviceName)
			}
		}
	}

}

// Create new Dev EventService
func NewLocalQueuesService() (*LocalQueuesService, error) {
	queueService := &LocalQueuesService{
		queues: map[queueName][]*structpb.Struct{},
		leased: map[queueName]map[string]*LeasedTask{},
	}

	go func() {
		requeueTimer := time.NewTicker(time.Second * 30)
		defer requeueTimer.Stop()

		for range requeueTimer.C {
			queueService.Requeue()
		}
	}()

	return queueService, nil
}
