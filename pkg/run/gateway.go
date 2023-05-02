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
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"

	"github.com/nitrictech/cli/pkg/utils"
	base_http "github.com/nitrictech/nitric/cloud/common/runtime/gateway"
	v1 "github.com/nitrictech/nitric/core/pkg/api/nitric/v1"
	"github.com/nitrictech/nitric/core/pkg/plugins/gateway"
	"github.com/nitrictech/nitric/core/pkg/worker"
	"github.com/nitrictech/nitric/core/pkg/worker/pool"
)

type HttpMiddleware func(*fasthttp.RequestCtx, pool.WorkerPool) bool

type apiServer struct {
	lis net.Listener
	srv *fasthttp.Server

	workerCount int
}

type BaseHttpGateway struct {
	apiServer       map[string]*apiServer
	serviceServer   *fasthttp.Server
	serviceListener net.Listener
	gateway.UnimplementedGatewayPlugin
	stop     chan bool
	pool     pool.WorkerPool
	dashPort int
}

var _ gateway.GatewayService = &BaseHttpGateway{}

func apiWorkerFilter(apiName string) func(w worker.Worker) bool {
	return func(w worker.Worker) bool {
		if api, ok := w.(*worker.RouteWorker); ok {
			return api.Api() == apiName
		}

		return false
	}
}

// GetTriggerAddress - Returns the address built-in nitric services
// this can be used to publishing messages to topics or triggering schedules
func (s *BaseHttpGateway) GetTriggerAddress() string {
	if s.serviceListener != nil {
		return strings.Replace(s.serviceListener.Addr().String(), "[::]", "localhost", 1)
	}

	return ""
}

func (s *BaseHttpGateway) GetApiAddresses() map[string]string {
	addresses := make(map[string]string)

	for api, srv := range s.apiServer {
		if srv.workerCount > 0 {
			srvAddress := strings.Replace(srv.lis.Addr().String(), "[::]", "localhost", 1)
			addresses[api] = srvAddress
		}
	}

	return addresses
}

func (s *BaseHttpGateway) handleHttpRequest(apiName string) func(ctx *fasthttp.RequestCtx) {
	return func(ctx *fasthttp.RequestCtx) {
		headerMap := base_http.HttpHeadersToMap(&ctx.Request.Header)

		headers := map[string]*v1.HeaderValue{}
		for k, v := range headerMap {
			headers[k] = &v1.HeaderValue{Value: v}
		}

		query := map[string]*v1.QueryValue{}

		ctx.QueryArgs().VisitAll(func(key []byte, val []byte) {
			k := string(key)

			if query[k] == nil {
				query[k] = &v1.QueryValue{}
			}

			query[k].Value = append(query[k].Value, string(val))
		})

		httpTrigger := &v1.TriggerRequest{
			Data: ctx.Request.Body(),
			Context: &v1.TriggerRequest_Http{
				Http: &v1.HttpTriggerContext{
					Method:      string(ctx.Request.Header.Method()),
					Path:        string(ctx.URI().PathOriginal()),
					Headers:     headers,
					QueryParams: query,
				},
			},
		}

		worker, err := s.pool.GetWorker(&pool.GetWorkerOptions{
			Trigger: httpTrigger,
			Filter:  apiWorkerFilter(apiName),
		})
		if err != nil {
			ctx.Redirect(fmt.Sprintf("http://localhost:%v/not-found", s.dashPort), fasthttp.StatusTemporaryRedirect)
			return
		}

		resp, err := worker.HandleTrigger(context.TODO(), httpTrigger)
		if err != nil {
			ctx.Error(fmt.Sprintf("Error handling HTTP Request: %v", err), 500)
			return
		}

		if http := resp.GetHttp(); http != nil {
			// Copy headers across
			for k, v := range http.Headers {
				for _, val := range v.Value {
					ctx.Response.Header.Add(k, val)
				}
			}

			// Avoid content length header duplication
			ctx.Response.Header.Del("Content-Length")
			ctx.Response.SetStatusCode(int(http.Status))
			ctx.Response.SetBody(resp.Data)

			return
		}

		ctx.Error("Response was not a Http response", 500)
	}
}

func (s *BaseHttpGateway) handleTopicRequest(ctx *fasthttp.RequestCtx) {
	topicName := ctx.UserValue("name").(string)

	trigger := &v1.TriggerRequest{
		Data: ctx.Request.Body(),
		Context: &v1.TriggerRequest_Topic{
			Topic: &v1.TopicTriggerContext{
				Topic: topicName,
			},
		},
	}

	ws := s.pool.GetWorkers(&pool.GetWorkerOptions{
		Trigger: trigger,
	})

	if len(ws) == 0 {
		ctx.Error("no subscribers found for topic", 404)
	}

	errList := make([]error, 0)

	for _, w := range ws {
		if _, err := w.HandleTrigger(context.TODO(), trigger); err != nil {
			errList = append(errList, err)
		}
	}

	ctx.Success("text/plain", []byte(fmt.Sprintf("%d successful & %d failed deliveries", len(ws)-len(errList), len(errList))))
}

type BucketNotificationPayload struct {
	Key  string `json:"key"`
	Type string `json:"type"`
}

func notificationStringToNotificationType(notificationType string) (v1.BucketNotificationType, error) {
	switch strings.ToLower(notificationType) {
	case "created":
		return v1.BucketNotificationType_Created, nil
	case "deleted":
		return v1.BucketNotificationType_Deleted, nil
	default:
		return v1.BucketNotificationType_All, fmt.Errorf("unsupported event type: %s", notificationType)
	}
}

func (s *BaseHttpGateway) handleBucketNotificationRequest(ctx *fasthttp.RequestCtx) {
	bucketName := ctx.UserValue("name").(string)

	var payload BucketNotificationPayload

	err := json.Unmarshal(ctx.Request.Body(), &payload)
	if err != nil {
		ctx.Error(err.Error(), 400)
	}

	notificationType, err := notificationStringToNotificationType(payload.Type)
	if err != nil {
		ctx.Error(err.Error(), 400)
		return
	}

	trigger := &v1.TriggerRequest{
		Context: &v1.TriggerRequest_Notification{
			Notification: &v1.NotificationTriggerContext{
				Source: bucketName,
				Notification: &v1.NotificationTriggerContext_Bucket{
					Bucket: &v1.BucketNotification{
						Key:  payload.Key,
						Type: notificationType,
					},
				},
			},
		},
	}

	w, err := s.pool.GetWorker(&pool.GetWorkerOptions{
		Trigger: trigger,
		Filter: func(w worker.Worker) bool {
			_, ok := w.(*worker.BucketNotificationWorker)
			return ok
		},
	})
	if err != nil {
		ctx.Error("could not get bucket notification for bucket", 404)
		return
	}

	resp, err := w.HandleTrigger(context.TODO(), trigger)
	if err != nil || !resp.GetNotification().Success {
		ctx.Error("bucket notification failed", 500)
		return
	}

	ctx.Success("text/plain", []byte("success"))
}

// Update the gateway and API based on the worker pool
func (s *BaseHttpGateway) Refresh() error {
	// instansiate servers if not already done
	if s.apiServer == nil {
		s.apiServer = make(map[string]*apiServer)
	}

	for _, srv := range s.apiServer {
		// reset all server worker counts
		srv.workerCount = 0
	}

	for _, w := range s.pool.GetWorkers(&pool.GetWorkerOptions{}) {
		if api, ok := w.(*worker.RouteWorker); ok {
			currApi, ok := s.apiServer[api.Api()]

			if !ok {
				fhttp := &fasthttp.Server{
					ReadTimeout:     time.Second * 1,
					IdleTimeout:     time.Second * 1,
					CloseOnShutdown: true,
					Handler:         s.handleHttpRequest(api.Api()),
				}

				lis, err := utils.GetNextListener()
				if err != nil {
					return err
				}

				srv := &apiServer{
					lis:         lis,
					srv:         fhttp,
					workerCount: 0,
				}

				// get a free port and listen on that for this API
				go func(srv *apiServer) {
					err := srv.srv.Serve(srv.lis)
					if err != nil {
						fmt.Println(err)
					}
				}(srv)

				currApi = srv
				// append to the server collection
				s.apiServer[api.Api()] = currApi

				// this is a brand new server we need to start up
				// lets start it and add it to the active list of servers
				// we can then filter the servers by their active worker count
				currApi.workerCount = 0
			}

			// Increment this APIs worker count
			// this will be used to filter displayed APIs
			currApi.workerCount = currApi.workerCount + 1
		}
	}

	return nil
}

func (s *BaseHttpGateway) Start(pool pool.WorkerPool) error {
	var err error
	// Assign the pool and block
	s.pool = pool
	s.stop = make(chan bool)

	// Setup routes
	r := router.New()
	// Publish to a topic
	r.POST("/topic/{name}", s.handleTopicRequest)
	// Publish to a bucket notification
	r.POST("/notification/bucket/{name}", s.handleBucketNotificationRequest)

	r.NotFound = func(ctx *fasthttp.RequestCtx) {
		if string(ctx.Path()) == "/" {
			ctx.Redirect(fmt.Sprintf("http://localhost:%v", s.dashPort), fasthttp.StatusMovedPermanently)
		} else {
			ctx.Redirect(fmt.Sprintf("http://localhost:%v/not-found", s.dashPort), fasthttp.StatusTemporaryRedirect)
		}
	}

	s.serviceServer = &fasthttp.Server{
		ReadTimeout:     time.Second * 1,
		IdleTimeout:     time.Second * 1,
		CloseOnShutdown: true,
		Handler:         r.Handler,
	}

	s.serviceListener, err = utils.GetNextListener()
	if err != nil {
		return err
	}

	go func() {
		_ = s.serviceServer.Serve(s.serviceListener)
	}()

	// block on a stop signal
	<-s.stop

	return nil
}

func (s *BaseHttpGateway) Stop() error {
	for _, s := range s.apiServer {
		// shutdown all the servers
		// this will allow Start to exit
		_ = s.srv.Shutdown()
	}

	s.stop <- true

	return nil
}

// Create new HTTP gateway
// XXX: No External Args for function atm (currently the plugin loader does not pass any argument information)
func NewGateway() (*BaseHttpGateway, error) {
	return &BaseHttpGateway{}, nil
}
