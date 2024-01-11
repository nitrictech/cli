package run

import (
	"fmt"
	"sync"

	apispb "github.com/nitrictech/nitric/core/pkg/proto/apis/v1"
	"github.com/nitrictech/nitric/core/pkg/workers/apis"
)

type LocalApiGateway struct {
	*apis.RouteWorkerManager

	apiRegLock sync.RWMutex
	apis       map[string][]*apispb.RegistrationRequest
}

var _ apispb.ApiServer = (*LocalApiGateway)(nil)

func (l *LocalApiGateway) GetApis() map[string][]*apispb.RegistrationRequest {
	l.apiRegLock.RLock()
	defer l.apiRegLock.RUnlock()

	return l.apis
}

func (l *LocalApiGateway) registerApiWorker(registrationRequest *apispb.RegistrationRequest) {
	l.apiRegLock.Lock()
	defer l.apiRegLock.Unlock()

	registrations := l.apis[registrationRequest.Api]
	registrations = append(registrations, registrationRequest)
}

func (l *LocalApiGateway) unregisterApiWorker(registrationRequest *apispb.RegistrationRequest) {
	l.apiRegLock.Lock()
	defer l.apiRegLock.Unlock()

	registrations := l.apis[registrationRequest.Api]
	for i, r := range registrations {
		if r == registrationRequest {
			registrations = append(registrations[:i], registrations[i+1:]...)
			break
		}
	}
}

func (l *LocalApiGateway) Serve(stream apispb.Api_ServeServer) error {
	peekableStream := NewPeekableStreamServer[*apispb.ServerMessage, *apispb.ClientMessage](stream)

	firstRequest, err := peekableStream.Recv()
	if err != nil {
		return err
	}

	if firstRequest.GetRegistrationRequest() == nil {
		return fmt.Errorf("first request must be a Registration Request")
	}

	// register the api
	l.registerApiWorker(firstRequest.GetRegistrationRequest())
	defer l.unregisterApiWorker(firstRequest.GetRegistrationRequest())

	return l.RouteWorkerManager.Serve(peekableStream)
}

func NewLocalApiGateway() *LocalApiGateway {
	return &LocalApiGateway{
		RouteWorkerManager: apis.New(),
		apis:               map[string][]*apispb.RegistrationRequest{},
	}
}
