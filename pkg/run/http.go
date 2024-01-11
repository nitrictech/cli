package run

import (
	"context"
	"fmt"
	"sync"

	httppb "github.com/nitrictech/nitric/core/pkg/proto/http/v1"
	"github.com/nitrictech/nitric/core/pkg/workers/http"
	"github.com/valyala/fasthttp"
)

type HostAddress = string

type LocalHttpProxy struct {
	httpWorkers map[HostAddress]*http.HttpServer

	httpWorkerLock sync.RWMutex
}

var _ httppb.HttpServer = (*LocalHttpProxy)(nil)

func (h *LocalHttpProxy) WorkerCount() int {
	h.httpWorkerLock.RLock()
	defer h.httpWorkerLock.RUnlock()

	return len(h.httpWorkers)
}

func (h *LocalHttpProxy) GetHttpWorkers() map[HostAddress]*http.HttpServer {
	h.httpWorkerLock.RLock()
	defer h.httpWorkerLock.RUnlock()

	return h.httpWorkers
}

// FIXME: Implement http server identification
func (h *LocalHttpProxy) HandleRequest(request *fasthttp.Request) (*fasthttp.Response, error) {
	h.httpWorkerLock.RLock()
	defer h.httpWorkerLock.RUnlock()

	host := string(request.Host())

	srv, ok := h.httpWorkers[host]
	if !ok {
		return nil, fmt.Errorf("No worker found for host: %s", host)
	}

	return srv.HandleRequest(request)
}

func (h *LocalHttpProxy) Proxy(ctx context.Context, req *httppb.HttpProxyRequest) (*httppb.HttpProxyResponse, error) {
	h.httpWorkerLock.Lock()
	defer h.httpWorkerLock.Unlock()
	host := req.GetHost()

	srv := http.New()

	// pass down the the original handler for port watching and management
	resp, err := srv.Proxy(ctx, req)
	if err != nil {
		return nil, err
	}

	// register the worker
	h.httpWorkers[host] = srv

	// return the original response
	return resp, nil
}

func NewLocalHttpGateway() *LocalHttpProxy {
	return &LocalHttpProxy{
		httpWorkers:    make(map[HostAddress]*http.HttpServer),
		httpWorkerLock: sync.RWMutex{},
	}
}
