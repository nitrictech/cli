package codeconfig

import (
	"fmt"
	"sync"

	v1 "github.com/nitrictech/nitric/core/pkg/api/nitric/v1"
)

type Websocket struct {
	name             string
	function         *FunctionDependencies
	connectWorker    *v1.WebsocketWorker
	disconnectWorker *v1.WebsocketWorker
	messageWorker    *v1.WebsocketWorker
	// workers []*v1.WebsocketWorker
	lock sync.RWMutex
}

func newWebsocket(name string, function *FunctionDependencies) *Websocket {
	return &Websocket{
		name:     name,
		function: function,
	}
}

func (a *Websocket) AddWorker(worker *v1.WebsocketWorker) {
	a.lock.Lock()
	defer a.lock.Unlock()

	switch worker.Event {
	case v1.WebsocketEvent_Connect:
		if a.connectWorker != nil {
			a.function.AddError(fmt.Sprintf("has registered multiple connect workers for socket: %s", a.name))
			return
		}

		a.connectWorker = worker
	case v1.WebsocketEvent_Disconnect:
		if a.disconnectWorker != nil {
			a.function.AddError(fmt.Sprintf("has registered multiple disconnect workers for socket: %s", a.name))
			return
		}

		a.disconnectWorker = worker
	case v1.WebsocketEvent_Message:
		if a.messageWorker != nil {
			a.function.AddError(fmt.Sprintf("has registered multiple message workers for socket: %s", a.name))
			return
		}

		a.messageWorker = worker
	default:
		a.function.AddError(fmt.Sprintf("has registered an invalid event type for socket: %s", a.name))
	}
}
