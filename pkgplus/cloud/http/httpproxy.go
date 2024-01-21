package http

import (
	"context"
	"fmt"
	"maps"
	"sync"

	"github.com/asaskevich/EventBus"
	"github.com/valyala/fasthttp"

	"github.com/nitrictech/cli/pkgplus/grpcx"
	httppb "github.com/nitrictech/nitric/core/pkg/proto/http/v1"
	"github.com/nitrictech/nitric/core/pkg/workers/http"
)

type HttpProxyService struct {
	serviceName string
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

func (l *LocalHttpProxy) SubscribeToState(fn func(map[HostAddress]*HttpProxyService)) {
	l.bus.Subscribe(localHttpProxyTopic, fn)
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

func (h *LocalHttpProxy) Proxy(ctx context.Context, req *httppb.HttpProxyRequest) (*httppb.HttpProxyResponse, error) {
	serviceName, err := grpcx.GetServiceNameFromIncomingContext(ctx)
	if err != nil {
		return nil, err
	}

	h.httpWorkerLock.Lock()
	defer h.httpWorkerLock.Unlock()
	host := req.GetHost()

	srv := http.New()

	// pass down the the original handler for port watching and management
	resp, err := srv.Proxy(ctx, req)
	if err != nil {
		return nil, err
	}

	h.state[host] = &HttpProxyService{
		server:      srv,
		serviceName: serviceName,
	}

	h.publishState()

	// return the original response
	return resp, nil
}

func NewLocalHttpProxyService() *LocalHttpProxy {
	return &LocalHttpProxy{
		state:          make(State),
		httpWorkerLock: sync.RWMutex{},
		bus:            EventBus.New(),
	}
}
