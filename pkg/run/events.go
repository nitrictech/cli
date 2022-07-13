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
	"encoding/json"
	"fmt"

	"github.com/nitrictech/nitric/pkg/plugins/errors"
	"github.com/nitrictech/nitric/pkg/plugins/errors/codes"
	"github.com/nitrictech/nitric/pkg/plugins/events"
	"github.com/nitrictech/nitric/pkg/triggers"
	"github.com/nitrictech/nitric/pkg/worker"
)

type WorkerPoolEventService struct {
	events.UnimplementedeventsPlugin
	pool worker.WorkerPool
}

// Publish a message to a given topic
func (s *WorkerPoolEventService) Publish(topic string, event *events.NitricEvent) error {
	newErr := errors.ErrorsWithScope(
		"WorkerPoolEventService.Publish",
		map[string]interface{}{
			"topic": topic,
			"event": event,
		},
	)

	requestId := event.ID
	// payloadType := event.PayloadType
	payload := event.Payload

	marshaledPayload, err := json.Marshal(payload)
	if err != nil {
		return newErr(
			codes.Internal,
			"error marshalling event payload",
			err,
		)
	}

	evt := &triggers.Event{
		ID:      requestId,
		Topic:   topic,
		Payload: marshaledPayload,
	}

	// get all scribers to this event
	targets := s.pool.GetWorkers(&worker.GetWorkerOptions{
		Event: evt,
	})

	fmt.Printf("Publishing to %s topic, %d subscriber(s)\n", topic, len(targets))
	for _, target := range targets {
		err = target.HandleEvent(evt)
		if err != nil {
			// this is likely an error in the user's handler, we don't want it to bring the server down.
			// just log and move on.
			fmt.Println(err)
		}
	}

	return nil
}

// Create new Dev EventService
func NewEvents(pool worker.WorkerPool) (events.EventService, error) {
	return &WorkerPoolEventService{
		pool: pool,
	}, nil
}
