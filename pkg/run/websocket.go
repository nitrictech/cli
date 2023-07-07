package run

import (
	"context"
	"fmt"
	"sync"

	"github.com/fasthttp/websocket"
	nitricws "github.com/nitrictech/nitric/core/pkg/plugins/websocket"
)

type RunWebsocketService struct {
	nitricws.WebsocketService
	connections map[string]map[string]*websocket.Conn
	lock        sync.RWMutex
}

func (r *RunWebsocketService) RegisterConnection(socket string, connectionId string, connection *websocket.Conn) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.connections[socket] == nil {
		r.connections[socket] = make(map[string]*websocket.Conn)
	}

	r.connections[socket][connectionId] = connection
}

func (r *RunWebsocketService) Send(ctx context.Context, socket string, connectionId string, message []byte) error {
	r.lock.RLock()
	defer r.lock.RUnlock()

	conn, ok := r.connections[socket][connectionId]
	if !ok {
		return fmt.Errorf("could not get connection " + connectionId)
	}

	err := conn.WriteMessage(websocket.TextMessage, message)
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

	// write a close message
	err := conn.WriteMessage(websocket.CloseMessage, nil)
	if err != nil {
		return err
	}

	// force close the connection
	err = conn.Close()
	if err != nil {
		return err
	}

	// delete the connection from the pool
	delete(r.connections[socket], connectionId)

	return nil
}

func NewRunWebsocketService() (*RunWebsocketService, error) {
	return &RunWebsocketService{
		connections: make(map[string]map[string]*websocket.Conn),
		lock:        sync.RWMutex{},
	}, nil
}
