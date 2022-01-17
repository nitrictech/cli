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
	"fmt"
	"strings"
	"time"

	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"

	"github.com/nitrictech/nitric/pkg/plugins/gateway"
	"github.com/nitrictech/nitric/pkg/triggers"
	"github.com/nitrictech/nitric/pkg/utils"
	"github.com/nitrictech/nitric/pkg/worker"
)

type HttpMiddleware func(*fasthttp.RequestCtx, worker.WorkerPool) bool

type BaseHttpGateway struct {
	address string
	server  *fasthttp.Server
	gateway.UnimplementedGatewayPlugin

	pool worker.WorkerPool
}

func apiWorkerFilter(apiName string) func(w worker.Worker) bool {
	return func(w worker.Worker) bool {
		if api, ok := w.(*worker.RouteWorker); ok {
			return api.Api() == apiName
		}

		return false
	}
}

func (s *BaseHttpGateway) api(ctx *fasthttp.RequestCtx) {
	apiName := ctx.UserValue("name").(string)
	// Rewrite the URL of the request to remove the /api/{name} subroute
	pathParts := utils.SplitPath(string(ctx.Path()))
	// remove first two path parts
	newPathParts := pathParts[2:]

	newPath := strings.Join(newPathParts, "/")

	// Rewrite the path
	ctx.URI().SetPath(newPath)

	httpReq := triggers.FromHttpRequest(ctx)

	worker, err := s.pool.GetWorker(&worker.GetWorkerOptions{
		Http:   httpReq,
		Filter: apiWorkerFilter(apiName),
	})

	if err != nil {
		ctx.Error("worker not found for api", 404)
		return
	}

	resp, err := worker.HandleHttpRequest(httpReq)

	if err != nil {
		ctx.Error(fmt.Sprintf("Error handling HTTP Request: %v", err), 500)
		return
	}

	if resp.Header != nil {
		resp.Header.CopyTo(&ctx.Response.Header)
	}

	// Avoid content length header duplication
	ctx.Response.Header.Del("Content-Length")
	ctx.Response.SetStatusCode(resp.StatusCode)
	ctx.Response.SetBody(resp.Body)
}

func (s *BaseHttpGateway) Start(pool worker.WorkerPool) error {
	s.pool = pool

	// Setup routes
	r := router.New()
	// Make a request for an API gateway
	r.ANY("/apis/{name}/{any:*}", s.api)

	s.server = &fasthttp.Server{
		ReadTimeout:     time.Second * 1,
		IdleTimeout:     time.Second * 1,
		CloseOnShutdown: true,
		Handler:         r.Handler,
	}

	return s.server.ListenAndServe(s.address)
}

func (s *BaseHttpGateway) Stop() error {
	if s.server != nil {
		return s.server.Shutdown()
	}
	return nil
}

// Create new HTTP gateway
// XXX: No External Args for function atm (currently the plugin loader does not pass any argument information)
func NewGateway() (gateway.GatewayService, error) {
	address := utils.GetEnv("GATEWAY_ADDRESS", ":9001")

	return &BaseHttpGateway{
		address: address,
	}, nil
}
