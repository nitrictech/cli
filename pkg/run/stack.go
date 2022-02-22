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
	"fmt"
	"strings"

	"github.com/pterm/pterm"

	"github.com/nitrictech/nitric/pkg/worker"
)

type RunStackState struct {
	apis      map[string]int
	subs      map[string]int
	schedules map[string]int
}

func (r *RunStackState) UpdateFromWorkerEvent(evt WorkerEvent) {
	if evt.Type == WorkerEventType_Add {
		switch evt.Worker.(type) {
		case *worker.RouteWorker:
			w := evt.Worker.(*worker.RouteWorker)

			if _, ok := r.apis[w.Api()]; !ok {
				r.apis[w.Api()] = 1
			} else {
				r.apis[w.Api()] = r.apis[w.Api()] + 1
			}
		case *worker.SubscriptionWorker:
			w := evt.Worker.(*worker.SubscriptionWorker)

			if _, ok := r.subs[w.Topic()]; !ok {
				r.subs[w.Topic()] = 1
			} else {
				r.subs[w.Topic()] = r.subs[w.Topic()] + 1
			}
		case *worker.ScheduleWorker:
			w := evt.Worker.(*worker.ScheduleWorker)

			if _, ok := r.schedules[w.Key()]; !ok {
				r.schedules[w.Key()] = 1
			} else {
				r.schedules[w.Key()] = r.schedules[w.Key()] + 1
			}
		}
	} else if evt.Type == WorkerEventType_Remove {
		switch evt.Worker.(type) {
		case *worker.RouteWorker:
			w := evt.Worker.(*worker.RouteWorker)

			r.apis[w.Api()] = r.apis[w.Api()] - 1

			if r.apis[w.Api()] <= 0 {
				// Remove the key if the reference count is 0 or less
				delete(r.apis, w.Api())
			}
		case *worker.SubscriptionWorker:
			w := evt.Worker.(*worker.SubscriptionWorker)

			r.subs[w.Topic()] = r.subs[w.Topic()] - 1

			if r.subs[w.Topic()] <= 0 {
				// Remove the key if the reference count is 0 or less
				delete(r.subs, w.Topic())
			}
		case *worker.ScheduleWorker:
			w := evt.Worker.(*worker.ScheduleWorker)

			r.schedules[w.Key()] = r.schedules[w.Key()] - 1

			if r.schedules[w.Key()] <= 0 {
				// Remove the key if the reference count is 0 or less
				delete(r.schedules, w.Key())
			}
		}
	}
}

func (r *RunStackState) ApiTable(port int) string {
	tableData := pterm.TableData{{"Api", "Endpoint"}}

	for k := range r.apis {
		tableData = append(tableData, []string{
			k, fmt.Sprintf("http://localhost:%d/apis/%s", port, k),
		})
	}

	str, _ := pterm.DefaultTable.WithHasHeader().WithData(tableData).Srender()

	return str
}

func (r *RunStackState) TopicTable(port int) string {
	tableData := pterm.TableData{{"Topic", "Endpoint"}}

	for k := range r.subs {
		tableData = append(tableData, []string{
			k, fmt.Sprintf("http://localhost:%d/topics/%s", port, k),
		})
	}

	str, _ := pterm.DefaultTable.WithHasHeader().WithData(tableData).Srender()

	return str
}

func (r *RunStackState) SchedulesTable(port int) string {
	tableData := pterm.TableData{{"Schedule", "Endpoint"}}

	for k := range r.schedules {
		nKey := strings.ToLower(strings.ReplaceAll(k, " ", "-"))
		tableData = append(tableData, []string{
			k, fmt.Sprintf("http://localhost:%d/topics/%s", port, nKey),
		})
	}

	str, _ := pterm.DefaultTable.WithHasHeader().WithData(tableData).Srender()

	return str
}

func NewStackState() *RunStackState {
	return &RunStackState{
		apis:      map[string]int{},
		subs:      map[string]int{},
		schedules: map[string]int{},
	}
}
