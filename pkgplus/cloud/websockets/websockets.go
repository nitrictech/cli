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

package websockets

import (
	"context"
	"fmt"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/asaskevich/EventBus"
	"github.com/fasthttp/websocket"

	"github.com/nitrictech/cli/pkgplus/grpcx"

	nitricws "github.com/nitrictech/nitric/core/pkg/proto/websockets/v1"
	"github.com/nitrictech/nitric/core/pkg/workers/websockets"
)

type (
	socketName  = string
	serviceName = string
)

type State = map[socketName]map[serviceName][]nitricws.WebsocketEventType

type LocalWebsocketService struct {
	*websockets.WebsocketManager
	connections map[string]map[string]*websocket.Conn
	state       State
	lock        sync.RWMutex

	servers map[string]string

	bus EventBus.Bus
}

var (
	_ nitricws.WebsocketServer        = (*LocalWebsocketService)(nil)
	_ nitricws.WebsocketHandlerServer = (*LocalWebsocketService)(nil)
)

const (
	localWebsocketActionTopic = "local_websocket_action"
	localWebsocketTopic       = "local_websocket_gateway"
)

type WebsocketMessage struct {
	Data         string    `json:"data,omitempty"`
	Time         time.Time `json:"time,omitempty"`
	ConnectionID string    `json:"connectionId,omitempty"`
}

type WebsocketInfo struct {
	ConnectionCount int                `json:"connectionCount,omitempty"`
	Messages        []WebsocketMessage `json:"messages,omitempty"`
}

type ActionType string

const (
	INFO    ActionType = "info"
	MESSAGE ActionType = "message"
)

type EventItem interface {
	WebsocketMessage | WebsocketInfo | any
}
type WebsocketAction[Event EventItem] struct {
	Name  string     `json:"name"`
	Event Event      `json:"event"`
	Type  ActionType `json:"-"`
}

func (r *LocalWebsocketService) SubscribeToState(subscription func(map[string]map[string][]nitricws.WebsocketEventType)) {
	r.bus.Subscribe(localWebsocketTopic, subscription)
}

func (r *LocalWebsocketService) GetState() State {
	r.lock.RLock()
	defer r.lock.RUnlock()

	return r.state
}

func (r *LocalWebsocketService) SetServers(server map[string]string) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.servers = server
}

func (r *LocalWebsocketService) publishAction(action WebsocketAction[EventItem]) {
	r.bus.Publish(localWebsocketActionTopic, action)
}

func (r *LocalWebsocketService) SubscribeToAction(subscription func(WebsocketAction[EventItem])) {
	r.bus.Subscribe(localWebsocketActionTopic, subscription)
}

func (r *LocalWebsocketService) registerWebsocketWorker(serviceName string, registration *nitricws.RegistrationRequest) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.state[registration.SocketName] == nil {
		r.state[registration.SocketName] = make(map[string][]nitricws.WebsocketEventType, 0)
	}

	r.state[registration.SocketName][serviceName] = append(r.state[registration.SocketName][serviceName], registration.EventType)

	r.bus.Publish(localWebsocketTopic, r.state)
}

func (r *LocalWebsocketService) unRegisterWebsocketWorker(serviceName string, registration *nitricws.RegistrationRequest) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.state[registration.SocketName] == nil {
		return
	}

	for i, w := range r.state[registration.SocketName][serviceName] {
		if w == registration.EventType {
			r.state[registration.SocketName][serviceName] = append(r.state[registration.SocketName][serviceName][:i], r.state[registration.SocketName][serviceName][i+1:]...)
			break
		}
	}

	if len(r.state[registration.SocketName]) == 0 {
		delete(r.state, registration.SocketName)
	}

	r.bus.Publish(localWebsocketTopic, r.state)
}

func (r *LocalWebsocketService) HandleEvents(stream nitricws.WebsocketHandler_HandleEventsServer) error {
	serviceName, err := grpcx.GetServiceNameFromStream(stream)
	if err != nil {
		return err
	}

	peekableStream := grpcx.NewPeekableStreamServer[*nitricws.ServerMessage, *nitricws.ClientMessage](stream)

	firstRequest, err := peekableStream.Peek()
	if err != nil {
		return err
	}

	if firstRequest.GetRegistrationRequest() == nil {
		return fmt.Errorf("first request must be a Registration Request")
	}

	// register the websocket
	r.registerWebsocketWorker(serviceName, firstRequest.GetRegistrationRequest())
	defer r.unRegisterWebsocketWorker(serviceName, firstRequest.GetRegistrationRequest())

	return r.WebsocketManager.HandleEvents(peekableStream)
}

func (r *LocalWebsocketService) RegisterConnection(socket string, connectionId string, connection *websocket.Conn) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.connections[socket] == nil {
		r.connections[socket] = make(map[string]*websocket.Conn)
	}

	r.connections[socket][connectionId] = connection

	r.publishAction(WebsocketAction[EventItem]{
		Name: socket,
		Type: INFO,
		Event: WebsocketInfo{
			ConnectionCount: len(r.connections[socket]),
		},
	})

	return nil
}

func (r *LocalWebsocketService) Details(ctx context.Context, req *nitricws.WebsocketDetailsRequest) (*nitricws.WebsocketDetailsResponse, error) {
	gatewayUri, ok := r.servers[req.SocketName]
	if !ok {
		return nil, fmt.Errorf("websocket %s does not exist", req.SocketName)
	}

	return &nitricws.WebsocketDetailsResponse{
		Url: fmt.Sprintf("ws://%s", gatewayUri),
	}, nil
}

func (r *LocalWebsocketService) Send(ctx context.Context, req *nitricws.WebsocketSendRequest) (*nitricws.WebsocketSendResponse, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	conn, ok := r.connections[req.SocketName][req.ConnectionId]
	if !ok {
		return nil, fmt.Errorf("could not get connection " + req.ConnectionId)
	}

	// Determine if the message is a binary message
	isBinary := isBinaryString(req.Data)

	if isBinary {
		// binary is not supported by AWS, so tell user
		req.Data = []byte("Binary messages are not currently supported by AWS")
	}

	err := conn.WriteMessage(websocket.TextMessage, req.Data)
	if err != nil {
		return nil, err
	}

	r.publishAction(WebsocketAction[EventItem]{
		Name: req.SocketName,
		Type: MESSAGE,
		Event: WebsocketMessage{
			Data:         string(req.Data),
			Time:         time.Now(),
			ConnectionID: req.ConnectionId,
		},
	})

	return &nitricws.WebsocketSendResponse{}, nil
}

func (r *LocalWebsocketService) Close(ctx context.Context, req *nitricws.WebsocketCloseRequest) (*nitricws.WebsocketCloseResponse, error) {
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

	r.publishAction(WebsocketAction[EventItem]{
		Name: req.SocketName,
		Type: INFO,
		Event: WebsocketInfo{
			ConnectionCount: len(r.connections[req.SocketName]),
		},
	})

	return &nitricws.WebsocketCloseResponse{}, nil
}

func NewLocalWebsocketService() (*LocalWebsocketService, error) {
	return &LocalWebsocketService{
		WebsocketManager: websockets.NewWebsocketManager(),
		connections:      make(map[string]map[string]*websocket.Conn),
		lock:             sync.RWMutex{},
		state:            make(map[string]map[string][]nitricws.WebsocketEventType),
		bus:              EventBus.New(),
	}, nil
}

func isBinaryString(data []byte) bool {
	return !utf8.Valid(data)
}
