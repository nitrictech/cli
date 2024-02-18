package http

import (
	"fmt"
	"maps"
	"sync"

	"github.com/asaskevich/EventBus"
	"github.com/valyala/fasthttp"

	"github.com/nitrictech/cli/pkg/grpcx"
	httppb "github.com/nitrictech/nitric/core/pkg/proto/http/v1"
	"github.com/nitrictech/nitric/core/pkg/workers/http"
)

type HttpProxyService struct {
	ServiceName string
	server      *http.HttpServer
}

type HostAddress = string

type State = map[HostAddress]*HttpProxyService

type LocalHttpProxy struct {
	state          State
	httpWorkerLock sync.RWMutex
	bus            EventBus.Bus
}

const localHttpProxyTopic = "local_http_proxy"

func (l *LocalHttpProxy) publishState() {
	l.bus.Publish(localHttpProxyTopic, maps.Clone(l.state))
}

func (l *LocalHttpProxy) SubscribeToState(fn func(State)) {
	// ignore the error, it's only returned if the fn param isn't a function
	_ = l.bus.Subscribe(localHttpProxyTopic, fn)
}

var _ httppb.HttpServer = (*LocalHttpProxy)(nil)

func (h *LocalHttpProxy) WorkerCount() int {
	h.httpWorkerLock.RLock()
	defer h.httpWorkerLock.RUnlock()

	return len(h.state)
}

func (h *LocalHttpProxy) GetState() State {
	h.httpWorkerLock.RLock()
	defer h.httpWorkerLock.RUnlock()

	return h.state
}

// FIXME: Implement http server identification
func (h *LocalHttpProxy) HandleRequest(request *fasthttp.Request) (*fasthttp.Response, error) {
	h.httpWorkerLock.RLock()
	defer h.httpWorkerLock.RUnlock()

	host := string(request.Host())

	service, ok := h.state[host]
	if !ok {
		return nil, fmt.Errorf("no worker found for host: %s", host)
	}

	return service.server.HandleRequest(request)
}

func (h *LocalHttpProxy) registerHttpProxy(host string, service *HttpProxyService) {
	h.httpWorkerLock.Lock()
	defer h.httpWorkerLock.Unlock()

	// pterm.Error.Printfln("got a http proxy")

	h.state[host] = service

	h.publishState()
}

func (h *LocalHttpProxy) unregisterHttpProxy(host string) {
	h.httpWorkerLock.Lock()
	defer h.httpWorkerLock.Unlock()

	delete(h.state, host)

	h.publishState()
}

func (h *LocalHttpProxy) Proxy(stream httppb.Http_ProxyServer) error {
	serviceName, err := grpcx.GetServiceNameFromStream(stream)
	if err != nil {
		return err
	}

	peekableStream := grpcx.NewPeekableStreamServer(stream)

	firstRequest, err := peekableStream.Peek()
	if err != nil {
		return err
	}

	if firstRequest.Request == nil {
		return fmt.Errorf("first request must be a proxy request")
	}

	host := firstRequest.Request.GetHost()
	srv := http.New()

	h.registerHttpProxy(host, &HttpProxyService{
		server:      srv,
		ServiceName: serviceName,
	})
	defer h.unregisterHttpProxy(host)

	// pass down the the original handler for port watching and management
	// let the proxy manage the connection
	return srv.Proxy(peekableStream)
}

func NewLocalHttpProxyService() *LocalHttpProxy {
	return &LocalHttpProxy{
		state:          make(State),
		httpWorkerLock: sync.RWMutex{},
		bus:            EventBus.New(),
	}
}
