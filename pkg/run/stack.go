package run

import (
	"fmt"

	"github.com/nitrictech/nitric/pkg/worker"
	"github.com/pterm/pterm"
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
