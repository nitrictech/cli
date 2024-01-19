package apis

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/asaskevich/EventBus"
	"github.com/pterm/pterm"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc/metadata"

	"github.com/nitrictech/cli/pkgplus/grpcx"
	"github.com/nitrictech/cli/pkgplus/streams"
	apispb "github.com/nitrictech/nitric/core/pkg/proto/apis/v1"
	"github.com/nitrictech/nitric/core/pkg/workers/apis"
)

type ApiName = string
type ServiceName = string
type State = map[ApiName]map[ServiceName][]*apispb.RegistrationRequest

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

func (l *LocalApiGatewayService) registerApiWorker(serviceName string, registrationRequest *apispb.RegistrationRequest) {
	l.apiRegLock.Lock()

	if l.state[registrationRequest.Api] == nil {
		l.state[registrationRequest.Api] = make(map[string][]*apispb.RegistrationRequest)
	}

	l.state[registrationRequest.Api][serviceName] = append(l.state[registrationRequest.Api][serviceName], registrationRequest)

	l.apiRegLock.Unlock()

	l.publishState()
}

func (l *LocalApiGatewayService) unregisterApiWorker(serviceName string, registrationRequest *apispb.RegistrationRequest) {
	l.apiRegLock.Lock()
	defer func() {
		l.apiRegLock.Unlock()
		l.publishState()
	}()

	l.state[registrationRequest.Api][serviceName] = slices.DeleteFunc(l.state[registrationRequest.Api][serviceName], func(item *apispb.RegistrationRequest) bool {
		return item == registrationRequest
	})
}

func (l *LocalApiGatewayService) Serve(stream apispb.Api_ServeServer) error {
	streamMetadata, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		return fmt.Errorf("missing expected metadata")
	}
	pterm.Error.Printfln("%+v", streamMetadata)

	serviceName := strings.Join(streamMetadata.Get(grpcx.ServiceNameKey), "")

	if serviceName == "" {
		return fmt.Errorf("missing expected service name")
	}

	peekableStream := streams.NewPeekableStreamServer[*apispb.ServerMessage, *apispb.ClientMessage](stream)

	firstRequest, err := peekableStream.Peek()
	if err != nil {
		return err
	}

	if firstRequest.GetRegistrationRequest() == nil {
		return fmt.Errorf("first request must be a Registration Request")
	}

	// register the api
	l.registerApiWorker(serviceName, firstRequest.GetRegistrationRequest())
	defer l.unregisterApiWorker(serviceName, firstRequest.GetRegistrationRequest())

	return l.RouteWorkerManager.Serve(peekableStream)
}

func NewLocalApiGatewayService() *LocalApiGatewayService {
	return &LocalApiGatewayService{
		RouteWorkerManager: apis.New(),
		state:              State{},
		bus:                EventBus.New(),
	}
}
