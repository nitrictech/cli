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
	"time"
	"unicode/utf8"

	"github.com/fasthttp/websocket"

	"github.com/nitrictech/cli/pkg/dashboard"
	nitricws "github.com/nitrictech/nitric/core/pkg/plugins/websocket"
)

type RunWebsocketService struct {
	nitricws.WebsocketService
	connections map[string]map[string]*websocket.Conn
	lock        sync.RWMutex
	dash        *dashboard.Dashboard
}

func (r *RunWebsocketService) RegisterConnection(socket string, connectionId string, connection *websocket.Conn) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.connections[socket] == nil {
		r.connections[socket] = make(map[string]*websocket.Conn)
	}

	r.connections[socket][connectionId] = connection

	err := r.dash.UpdateWebsocketInfoCount(socket, len(r.connections[socket]))
	if err != nil {
		return err
	}

	return nil
}

func (r *RunWebsocketService) Send(ctx context.Context, socket string, connectionId string, message []byte) error {
	r.lock.RLock()
	defer r.lock.RUnlock()

	conn, ok := r.connections[socket][connectionId]
	if !ok {
		return fmt.Errorf("could not get connection " + connectionId)
	}

	// Determine if the message is a binary message
	isBinary := isBinaryString(message)

	if isBinary {
		// binary is not supported by AWS, so tell user
		message = []byte("Binary messages are not currently supported by AWS")
	}

	err := conn.WriteMessage(websocket.TextMessage, message)
	if err != nil {
		return err
	}

	infoMessage := dashboard.WebsocketMessage{
		Data:         string(message),
		Time:         time.Now(),
		ConnectionID: connectionId,
	}

	err = r.dash.AddWebsocketInfoMessage(socket, infoMessage)
	if err != nil {
		return err
	}

	return nil
}

func (r *RunWebsocketService) Close(ctx context.Context, socket string, connectionId string) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	conn, ok := r.connections[socket][connectionId]
	if !ok {
		return fmt.Errorf("could not get connection")
	}

	// force close the connection
	err := conn.Close()
	if err != nil {
		return err
	}

	// delete the connection from the pool
	delete(r.connections[socket], connectionId)

	err = r.dash.UpdateWebsocketInfoCount(socket, len(r.connections[socket]))
	if err != nil {
		return err
	}

	return nil
}

func NewRunWebsocketService(dash *dashboard.Dashboard) (*RunWebsocketService, error) {
	return &RunWebsocketService{
		connections: make(map[string]map[string]*websocket.Conn),
		lock:        sync.RWMutex{},
		dash:        dash,
	}, nil
}

func isBinaryString(data []byte) bool {
	return !utf8.Valid(data)
}
