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
	"context"
	"fmt"
	"sync"
	"unicode/utf8"

	"github.com/fasthttp/websocket"

	nitricws "github.com/nitrictech/nitric/core/pkg/proto/websockets/v1"
	"github.com/nitrictech/nitric/core/pkg/workers/websockets"
)

type RunWebsocketService struct {
	*websockets.WebsocketManager
	connections map[string]map[string]*websocket.Conn
	workers     map[string][]nitricws.WebsocketEventType
	lock        sync.RWMutex
}

var _ nitricws.WebsocketServer = (*RunWebsocketService)(nil)
var _ nitricws.WebsocketHandlerServer = (*RunWebsocketService)(nil)

func (r *RunWebsocketService) GetWebsocketWorkers() map[string][]nitricws.WebsocketEventType {
	r.lock.RLock()
	defer r.lock.RUnlock()

	return r.workers
}

func (r *RunWebsocketService) registerWebsocketWorker(registration *nitricws.RegistrationRequest) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.workers[registration.SocketName] == nil {
		r.workers[registration.SocketName] = make([]nitricws.WebsocketEventType, 0)
	}

	r.workers[registration.SocketName] = append(r.workers[registration.SocketName], registration.EventType)
}

func (r *RunWebsocketService) unRegisterWebsocketWorker(registration *nitricws.RegistrationRequest) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.workers[registration.SocketName] == nil {
		return
	}

	for i, w := range r.workers[registration.SocketName] {
		if w == registration.EventType {
			r.workers[registration.SocketName] = append(r.workers[registration.SocketName][:i], r.workers[registration.SocketName][i+1:]...)
			break
		}
	}

	if len(r.workers[registration.SocketName]) == 0 {
		delete(r.workers, registration.SocketName)
	}
}

func (r *RunWebsocketService) HandleEvents(stream nitricws.WebsocketHandler_HandleEventsServer) error {
	peekableStream := NewPeekableStreamServer[*nitricws.ServerMessage, *nitricws.ClientMessage](stream)

	firstRequest, err := peekableStream.Peek()
	if err != nil {
		return err
	}

	if firstRequest.GetRegistrationRequest() == nil {
		return fmt.Errorf("first request must be a Registration Request")
	}

	// register the api
	r.registerWebsocketWorker(firstRequest.GetRegistrationRequest())
	defer r.unRegisterWebsocketWorker(firstRequest.GetRegistrationRequest())

	return r.WebsocketManager.HandleEvents(peekableStream)
}

func (r *RunWebsocketService) RegisterConnection(socket string, connectionId string, connection *websocket.Conn) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.connections[socket] == nil {
		r.connections[socket] = make(map[string]*websocket.Conn)
	}

	r.connections[socket][connectionId] = connection

	// FIXME: Use topics
	// err := r.dash.UpdateWebsocketInfoCount(socket, len(r.connections[socket]))
	// if err != nil {
	// 	return err
	// }

	return nil
}

func (r *RunWebsocketService) Send(ctx context.Context, req *nitricws.WebsocketSendRequest) (*nitricws.WebsocketSendResponse, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	conn, ok := r.connections[req.SocketName][req.ConnectionId]
	if !ok {
		return nil, fmt.Errorf("could not get connection " + req.ConnectionId)
	}

	err := conn.WriteMessage(websocket.TextMessage, req.Data)
	if err != nil {
		return nil, err
	}

	// FIXME: Use topics
	// infoMessage := dashboard.WebsocketMessage{
	// 	Data:         string(req.Data),
	// 	Time:         time.Now(),
	// 	ConnectionID: req.ConnectionId,
	// }

	// err = r.dash.AddWebsocketInfoMessage(req.SocketName, infoMessage)
	// if err != nil {
	// 	return nil, err
	// }

	return &nitricws.WebsocketSendResponse{}, nil
}

func (r *RunWebsocketService) Close(ctx context.Context, req *nitricws.WebsocketCloseRequest) (*nitricws.WebsocketCloseResponse, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	conn, ok := r.connections[req.SocketName][req.ConnectionId]
	if !ok {
		return nil, fmt.Errorf("could not get connection")
	}

	// force close the connection
	err := conn.Close()
	if err != nil {
		return nil, err
	}

	// delete the connection from the pool
	delete(r.connections[req.SocketName], req.ConnectionId)

	// FIXME: Use topics
	// err = r.dash.UpdateWebsocketInfoCount(req.SocketName, len(r.connections[req.SocketName]))
	// if err != nil {
	// 	return nil, err
	// }

	return &nitricws.WebsocketCloseResponse{}, nil
}

func NewRunWebsocketService() (*RunWebsocketService, error) {
	return &RunWebsocketService{
		connections: make(map[string]map[string]*websocket.Conn),
		lock:        sync.RWMutex{},
		workers:     make(map[string][]nitricws.WebsocketEventType),
	}, nil
}

func isBinaryString(data []byte) bool {
	return !utf8.Valid(data)
}
