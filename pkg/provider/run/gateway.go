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
// Copyright 2021 Nitric Pty Ltd.

package run

import (
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

//func apiWorkerFilter (apiName string) func(w worker.Worker) bool {
//	return func(w worker.Worker) bool {
//		if api, ok := w.(*worker.RouteWorker); ok {
//			return api.Api() == apiName
//		}

//		return false
//	}
//}

func (s *BaseHttpGateway) api(ctx *fasthttp.RequestCtx) {
	apiName := ctx.UserValue("name")
	// Rewrite the URL of the request to remove the /api/{name} subroute
	pathParts := utils.SplitPath(string(ctx.Path()))
	// remove first two path parts
	newPathParts := pathParts[2:]

	newPath := strings.Join(newPathParts, "/")

	// Rewrite the path
	ctx.URI().SetPath(newPath)

	httpReq := triggers.FromHttpRequest(ctx)

	s.pool.GetWorker(&worker.GetWorkerOptions{
		Http: httpReq,
		//Filter: apiWorkerFilter(apiName),
	})

	// Filter workers by a specific named API

}

func (s *BaseHttpGateway) schedule(ctx *fasthttp.RequestCtx) {
	scheduleName := ctx.UserValue("name")
	// Filter workers by schedule workers
}

//func (s *BaseHttpGateway) httpHandler(pool worker.WorkerPool) func(ctx *fasthttp.RequestCtx) {
//	return func(ctx *fasthttp.RequestCtx) {
//		if s.mw != nil {
//			if !s.mw(ctx, pool) {
//				// middleware has indicated that is has processed the request
//				// so we can exit here
//				return
//			}
//		}

//		httpTrigger := triggers.FromHttpRequest(ctx)
//		wrkr, err := pool.GetWorker(&worker.GetWorkerOptions{
//			Http: httpTrigger,
//		})

//		if err != nil {
//			ctx.Error("Unable to get worker to handle request", 500)
//			return
//		}

//		response, err := wrkr.HandleHttpRequest(httpTrigger)

//		if err != nil {
//			ctx.Error(fmt.Sprintf("Error handling HTTP Request: %v", err), 500)
//			return
//		}

//		if response.Header != nil {
//			response.Header.CopyTo(&ctx.Response.Header)
//		}

//		// Avoid content length header duplication
//		ctx.Response.Header.Del("Content-Length")
//		ctx.Response.SetStatusCode(response.StatusCode)
//		ctx.Response.SetBody(response.Body)
//	}
//}

func (s *BaseHttpGateway) Start(pool worker.WorkerPool) error {
	s.pool = pool

	// Setup routes
	r := router.New()
	// Make a request for an API gateway
	r.ANY("/apis/{name}/{any:*}", s.api)
	// TODO: Make a request to a specific registered function
	// r.ANY("/function/{name}/{any:*}", s.function)
	// Make a request to trigger a schedule
	r.POST("/schedules/{name}", s.schedule)

	s.server = &fasthttp.Server{
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
func New(mw HttpMiddleware) (gateway.GatewayService, error) {
	address := utils.GetEnv("GATEWAY_ADDRESS", ":9001")

	return &BaseHttpGateway{
		address: address,
	}, nil
}
