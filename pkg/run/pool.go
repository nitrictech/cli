package run

import "github.com/nitrictech/nitric/pkg/worker"

type WorkerEventType string

const (
	WorkerEventType_Add    WorkerEventType = "add"
	WorkerEventType_Remove WorkerEventType = "remove"
)

type WorkerEvent struct {
	Type   WorkerEventType
	Worker worker.Worker
}

type WorkerListener = func(WorkerEvent)

type RunProcessPool struct {
	worker.WorkerPool
	listeners []WorkerListener
}

func (r *RunProcessPool) notifyListeners(evt WorkerEvent) {
	for _, l := range r.listeners {
		l(evt)
	}
}

func (r *RunProcessPool) AddWorker(w worker.Worker) error {
	if err := r.WorkerPool.AddWorker(w); err != nil {
		return err
	}
	// notify listener of successfully added worker
	r.notifyListeners(WorkerEvent{
		Type:   WorkerEventType_Add,
		Worker: w,
	})
	return nil
}

func (r *RunProcessPool) RemoveWorker(w worker.Worker) error {
	if err := r.WorkerPool.RemoveWorker(w); err != nil {
		return err
	}
	// notify listener of successfully removed worker
	r.notifyListeners(WorkerEvent{
		Type:   WorkerEventType_Remove,
		Worker: w,
	})
	return nil
}

func (r *RunProcessPool) Listen(l WorkerListener) {
	r.listeners = append(r.listeners, l)
}

func NewRunProcessPool() *RunProcessPool {
	return &RunProcessPool{
		listeners: make([]WorkerListener, 0),
		WorkerPool: worker.NewProcessPool(&worker.ProcessPoolOptions{
			MinWorkers: 0,
			MaxWorkers: 100,
		}),
	}
}
