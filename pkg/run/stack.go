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

	"github.com/pterm/pterm"

	"github.com/nitrictech/nitric/pkg/worker"
)

type RunStackState struct {
	apis map[string]int
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

func NewStackState() *RunStackState {
	return &RunStackState{
		apis: map[string]int{},
	}
}
