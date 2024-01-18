package apis

import (
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/asaskevich/EventBus"
	"github.com/valyala/fasthttp"

	"github.com/nitrictech/cli/pkgplus/streams"
	apispb "github.com/nitrictech/nitric/core/pkg/proto/apis/v1"
	"github.com/nitrictech/nitric/core/pkg/workers/apis"
)

type ApiName = string

type State = map[ApiName][]*apispb.RegistrationRequest

type ApiRequestState struct {
	Api      string
	ReqCtx   *fasthttp.RequestCtx
	HttpResp *apispb.HttpResponse
}
type LocalApiGatewayService struct {
	*apis.RouteWorkerManager

	apiRegLock sync.RWMutex
	state      State

	bus EventBus.Bus
}

const localApiGatewayTopic = "local_api_gateway"

const localApiRequestTopic = "local_api_gateway_request"

func (l *LocalApiGatewayService) publishState() {
	l.bus.Publish(localApiGatewayTopic, l.GetState())
}

var _ apispb.ApiServer = (*LocalApiGatewayService)(nil)

func (l *LocalApiGatewayService) SubscribeToState(subscriberFunction func(State)) {
	l.bus.Subscribe(localApiGatewayTopic, subscriberFunction)
}

func (l *LocalApiGatewayService) PublishActionState(state ApiRequestState) {
	l.bus.Publish(localApiRequestTopic, state)
}

func (l *LocalApiGatewayService) SubscribeToAction(subscription func(ApiRequestState)) {
	l.bus.Subscribe(localApiRequestTopic, subscription)
}

// GetState - Returns a copy of internal state
func (l *LocalApiGatewayService) GetState() State {
	l.apiRegLock.RLock()
	defer l.apiRegLock.RUnlock()

	return maps.Clone(l.state)
}

func (l *LocalApiGatewayService) registerApiWorker(registrationRequest *apispb.RegistrationRequest) {
	l.apiRegLock.Lock()
	l.state[registrationRequest.Api] = append(l.state[registrationRequest.Api], registrationRequest)
	l.apiRegLock.Unlock()

	l.publishState()
}

func (l *LocalApiGatewayService) unregisterApiWorker(registrationRequest *apispb.RegistrationRequest) {
	l.apiRegLock.Lock()
	defer func() {
		l.apiRegLock.Unlock()
		l.publishState()
	}()

	l.state[registrationRequest.Api] = slices.DeleteFunc(l.state[registrationRequest.Api], func(item *apispb.RegistrationRequest) bool {
		return item == registrationRequest
	})

	// Remove the key if registrations is 0
	if len(l.state[registrationRequest.Api]) == 0 {
		delete(l.state, registrationRequest.Api)
	}
}

func (l *LocalApiGatewayService) Serve(stream apispb.Api_ServeServer) error {
	peekableStream := streams.NewPeekableStreamServer[*apispb.ServerMessage, *apispb.ClientMessage](stream)

	firstRequest, err := peekableStream.Peek()
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

func NewLocalApiGatewayService() *LocalApiGatewayService {
	return &LocalApiGatewayService{
		RouteWorkerManager: apis.New(),
		state:              State{},
		bus:                EventBus.New(),
	}
}
