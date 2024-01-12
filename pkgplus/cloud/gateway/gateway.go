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

package gateway

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/asaskevich/EventBus"
	"github.com/fasthttp/router"
	"github.com/fasthttp/websocket"
	"github.com/google/uuid"
	"github.com/pterm/pterm"
	"github.com/samber/lo"
	"github.com/valyala/fasthttp"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/nitrictech/cli/pkgplus/cloud/apis"
	"github.com/nitrictech/cli/pkgplus/cloud/http"
	"github.com/nitrictech/cli/pkgplus/cloud/websockets"

	"github.com/nitrictech/cli/pkg/utils"
	base_http "github.com/nitrictech/nitric/cloud/common/runtime/gateway"

	workersHttp "github.com/nitrictech/nitric/core/pkg/workers/http"

	"github.com/nitrictech/nitric/core/pkg/gateway"
	apispb "github.com/nitrictech/nitric/core/pkg/proto/apis/v1"
	schedulespb "github.com/nitrictech/nitric/core/pkg/proto/schedules/v1"
	topicspb "github.com/nitrictech/nitric/core/pkg/proto/topics/v1"
	websocketspb "github.com/nitrictech/nitric/core/pkg/proto/websockets/v1"
)

type apiServer struct {
	lis net.Listener
	srv *fasthttp.Server
}

type socketServer struct {
	lis net.Listener
	srv *fasthttp.Server

	workerCount int
}

var upgrader = websocket.FastHTTPUpgrader{}

type LocalGatewayService struct {
	apiServers       []*apiServer
	httpServers      []*apiServer
	apis             []string
	httpWorkers      []string
	websocketWorkers []string
	socketServer     map[string]*socketServer
	serviceServer    *fasthttp.Server
	websocketPlugin  *websockets.LocalWebsocketService
	serviceListener  net.Listener
	gateway.UnimplementedGatewayPlugin
	stop chan bool

	options *gateway.GatewayStartOpts
	bus     EventBus.Bus
}

var _ gateway.GatewayService = &LocalGatewayService{}

const localApiRequestTopic = "local_api_request"

type ApiRequestState struct {
	Api      string
	ReqCtx   *fasthttp.RequestCtx
	HttpResp *apispb.HttpResponse
}

func (s *LocalGatewayService) publishState(state ApiRequestState) {
	s.bus.Publish(localApiRequestTopic, state)
}

func (s *LocalGatewayService) SubscribeToApiRequestCtx(subscriberFunction func(state ApiRequestState)) {
	s.bus.Subscribe(localApiRequestTopic, subscriberFunction)
}

// GetTriggerAddress - Returns the base address built-in nitric services, like schedules and topics, will be exposed on.
func (s *LocalGatewayService) GetTriggerAddress() string {
	if s.serviceListener != nil {
		return strings.Replace(s.serviceListener.Addr().String(), "[::]", "localhost", 1)
	}

	return ""
}

func (s *LocalGatewayService) GetApiAddresses() map[string]string {
	addresses := make(map[string]string)

	if len(s.apiServers) > 0 && len(s.apis) == len(s.apiServers) {
		for idx, api := range s.apis {
			addresses[api] = strings.Replace(s.apiServers[idx].lis.Addr().String(), "[::]", "localhost", 1)
		}
	}

	return addresses
}

func (s *LocalGatewayService) GetHttpWorkerAddresses() map[string]string {
	addresses := make(map[string]string)

	for idx, host := range s.httpWorkers {
		addresses[host] = strings.Replace(s.httpServers[idx].lis.Addr().String(), "[::]", "localhost", 1)
	}

	return addresses
}

func (s *LocalGatewayService) GetWebsocketAddresses() map[string]string {
	addresses := make(map[string]string)

	for socket, srv := range s.socketServer {
		if srv.workerCount > 0 {
			srvAddress := strings.Replace(srv.lis.Addr().String(), "[::]", "localhost", 1)
			addresses[socket] = srvAddress
		}
	}

	return addresses
}

func (s *LocalGatewayService) handleHttpProxyRequest(idx int) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		// TODO: Use port to map to the correct http worker
		// port := s.httpWorkers[idx]

		// TODO: Need to support multiple HTTP handlers
		// so a plugin wrapper will be required for this
		resp, err := s.options.HttpPlugin.HandleRequest(&ctx.Request)
		if err != nil {
			ctx.Error(fmt.Sprintf("Error handling HTTP Request: %v", err), 500)
			return
		}

		resp.CopyTo(&ctx.Response)
	}
}

func (s *LocalGatewayService) handleApiHttpRequest(idx int) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		if idx >= len(s.apis) {
			ctx.Error("Sorry, nitric is listening on this port but is waiting for an API to be available to handle, you may have removed an API during development this port will be assigned to an API when one becomes available", 404)
			return
		}

		apiName := s.apis[idx]

		headerMap := base_http.HttpHeadersToMap(&ctx.Request.Header)

		headers := map[string]*apispb.HeaderValue{}
		for k, v := range headerMap {
			headers[k] = &apispb.HeaderValue{Value: v}
		}

		query := map[string]*apispb.QueryValue{}

		ctx.QueryArgs().VisitAll(func(key []byte, val []byte) {
			k := string(key)

			if query[k] == nil {
				query[k] = &apispb.QueryValue{}
			}

			query[k].Value = append(query[k].Value, string(val))
		})

		path := string(ctx.URI().Path())

		_, err := url.Parse(path)
		if err != nil {
			ctx.Error(fmt.Sprintf("Bad Request: %v", err), 400)
			return
		}

		apiEvent := &apispb.ServerMessage{
			Content: &apispb.ServerMessage_HttpRequest{
				HttpRequest: &apispb.HttpRequest{
					Method:      string(ctx.Request.Header.Method()),
					Path:        path,
					Headers:     headers,
					QueryParams: query,
					PathParams:  map[string]string{},
					Body:        ctx.Request.Body(),
				},
			},
		}

		resp, err := s.options.ApiPlugin.HandleRequest(apiName, apiEvent)
		if err != nil {
			ctx.Error(fmt.Sprintf("Error handling HTTP Request: %v", err), 500)
			return
		}

		if http := resp.GetHttpResponse(); http != nil {
			// Copy headers across
			for k, v := range http.Headers {
				for _, val := range v.Value {
					ctx.Response.Header.Add(k, val)
				}
			}

			// Avoid content length header duplication
			ctx.Response.Header.Del("Content-Length")
			ctx.Response.SetStatusCode(int(http.Status))
			ctx.Response.SetBody(resp.GetHttpResponse().GetBody())

			// publish ctx for history
			s.publishState(ApiRequestState{
				Api:      apiName,
				ReqCtx:   ctx,
				HttpResp: http,
			})

			return
		}

		ctx.Error("Response was not a Http response", 500)
	}
}

// websocket request handler
// TODO: Add broadcast capability
func (s *LocalGatewayService) handleWebsocketRequest(socketName string) func(ctx *fasthttp.RequestCtx) {
	return func(ctx *fasthttp.RequestCtx) {
		upgrader.CheckOrigin = func(ctx *fasthttp.RequestCtx) bool {
			return true
		}

		connectionId := uuid.New().String()

		query := map[string]*websocketspb.QueryValue{}

		ctx.QueryArgs().VisitAll(func(key []byte, val []byte) {
			k := string(key)

			if query[k] == nil {
				query[k] = &websocketspb.QueryValue{}
			}

			query[k].Value = append(query[k].Value, string(val))
		})

		resp, err := s.options.WebsocketListenerPlugin.HandleRequest(&websocketspb.ServerMessage{
			Content: &websocketspb.ServerMessage_WebsocketEventRequest{
				WebsocketEventRequest: &websocketspb.WebsocketEventRequest{
					SocketName: socketName,
					WebsocketEvent: &websocketspb.WebsocketEventRequest_Connection{
						Connection: &websocketspb.WebsocketConnectionEvent{
							QueryParams: query,
						},
					},
					ConnectionId: connectionId,
				},
			},
		})
		if err != nil {
			return
		}

		if resp.GetWebsocketEventResponse() == nil && resp.GetWebsocketEventResponse().GetConnectionResponse() != nil && resp.GetWebsocketEventResponse().GetConnectionResponse().Reject {
			// close the connection
			ctx.Error("Connection Refused", 500)
			return
		}

		err = upgrader.Upgrade(ctx, func(ws *websocket.Conn) {
			// generate a new connection ID for this client
			defer func() {
				// close within the websocket plugin will also call ws.Close
				_, err = s.websocketPlugin.Close(ctx, &websocketspb.WebsocketCloseRequest{
					ConnectionId: connectionId,
					SocketName:   socketName,
				})
				if err != nil {
					pterm.Error.Println(err)
					return
				}
			}()

			err = s.websocketPlugin.RegisterConnection(socketName, connectionId, ws)
			if err != nil {
				pterm.Error.Println(err)
				return
			}

			// Handshake successful send a registration message with connection ID to the socket worker
			for {
				// We have successfully connected a new client
				// We can read/write messages to/from this client
				// Need to create a unique ID for this connection and store in a central location
				// This will allow connected clients to message eachother and broadcast to all clients as well
				// We'll only read new messages on this connection here, writing will be done by a separate runtime API
				_, message, err := ws.ReadMessage()
				if err != nil {
					log.Println("read:", err)
					break
				}

				_, err = s.options.WebsocketListenerPlugin.HandleRequest(&websocketspb.ServerMessage{
					Content: &websocketspb.ServerMessage_WebsocketEventRequest{
						WebsocketEventRequest: &websocketspb.WebsocketEventRequest{
							SocketName:   socketName,
							ConnectionId: connectionId,
							WebsocketEvent: &websocketspb.WebsocketEventRequest_Message{
								Message: &websocketspb.WebsocketMessageEvent{
									Body: message,
								},
							},
						},
					},
				})
				if err != nil {
					pterm.Error.Println(err)
					return
				}
			}

			_, err = s.options.WebsocketListenerPlugin.HandleRequest(&websocketspb.ServerMessage{
				Content: &websocketspb.ServerMessage_WebsocketEventRequest{
					WebsocketEventRequest: &websocketspb.WebsocketEventRequest{
						SocketName:   socketName,
						ConnectionId: connectionId,
						WebsocketEvent: &websocketspb.WebsocketEventRequest_Disconnection{
							Disconnection: &websocketspb.WebsocketDisconnectionEvent{},
						},
					},
				},
			})
			if err != nil {
				pterm.Error.Println(err)
				return
			}
		})

		if err != nil {
			if _, ok := err.(websocket.HandshakeError); ok {
				pterm.Error.Println(err)
			}

			return
		}
	}
}

func (s *LocalGatewayService) handleTopicRequest(ctx *fasthttp.RequestCtx) {
	topicName := ctx.UserValue("name").(string)

	// Get the incoming data as JSON
	payload := map[string]interface{}{}
	err := json.Unmarshal(ctx.Request.Body(), &payload)
	if err != nil {
		ctx.Error(fmt.Sprintf("Error parsing JSON: %v", err), 400)
		return
	}

	structPayload, err := structpb.NewStruct(payload)
	if err != nil {
		ctx.Error(fmt.Sprintf("Error serializing topic message from payload: %v", err), 400)
		return
	}

	msg := &topicspb.ServerMessage{
		Content: &topicspb.ServerMessage_MessageRequest{
			MessageRequest: &topicspb.MessageRequest{
				TopicName: topicName,
				Message: &topicspb.Message{
					Content: &topicspb.Message_StructPayload{
						StructPayload: structPayload,
					},
				},
			},
		},
	}

	_, err = s.options.TopicsListenerPlugin.HandleRequest(msg)
	if err != nil {
		ctx.Error(fmt.Sprintf("Error handling topic request: %v", err), 500)
		return
	}

	ctx.SuccessString("text/plain", "Successfully delivered message to topic")
}

func (s *LocalGatewayService) handleSchedulesTrigger(ctx *fasthttp.RequestCtx) {
	scheduleName := ctx.UserValue("name").(string)

	msg := &schedulespb.ServerMessage{
		Content: &schedulespb.ServerMessage_IntervalRequest{
			IntervalRequest: &schedulespb.IntervalRequest{
				ScheduleName: scheduleName,
			},
		},
	}

	_, err := s.options.SchedulesPlugin.HandleRequest(msg)
	if err != nil {
		ctx.Error(fmt.Sprintf("Error handling schedule trigger: %v", err), 500)
		return
	}

	ctx.SuccessString("text/plain", "Successfully triggered schedule")
}

func (s *LocalGatewayService) refreshApis(apiState apis.State) {
	s.apis = make([]string, 0)

	uniqApis := lo.Reduce(lo.Keys(apiState), func(agg []string, apiName string, idx int) []string {
		if !lo.Contains(agg, apiName) {
			agg = append(agg, apiName)
		}

		return agg
	}, []string{})

	// sort the APIs by alphabetical order
	sort.Strings(uniqApis)

	s.apis = append(s.apis, uniqApis...)

	s.createApiServers()
}

func (s *LocalGatewayService) refreshHttpWorkers(state http.State) {
	s.httpWorkers = make([]string, 0)
	var uniqHttpWorkers []string

	uniqHttpWorkers = lo.Reduce(lo.Keys(state), func(agg []string, host string, idx int) []string {
		if !lo.Contains(agg, host) {
			agg = append(agg, host)
		}

		return agg
	}, []string{})

	// sort the Http Worker Ports lowest to highest
	sort.Strings(uniqHttpWorkers)

	s.httpWorkers = append(s.httpWorkers, uniqHttpWorkers...)
}

func (s *LocalGatewayService) refreshWebsocketWorkers(state websockets.State) {
	s.websocketWorkers = make([]string, 0)

	websockets := lo.Reduce(lo.Keys(state), func(agg []string, socketName string, idx int) []string {
		if !lo.Contains(agg, socketName) {
			agg = append(agg, socketName)
		}

		return agg
	}, []string{})

	// sort the Http Worker Ports lowest to highest
	sort.Strings(websockets)

	s.websocketWorkers = append(s.websocketWorkers, websockets...)

	s.createWebsocketServers()
}

func (s *LocalGatewayService) createApiServers() error {
	// create an api server for every API worker
	for len(s.apiServers) < len(s.apis) {
		fhttp := &fasthttp.Server{
			ReadTimeout:     time.Second * 1,
			IdleTimeout:     time.Second * 1,
			CloseOnShutdown: true,
			Handler:         s.handleApiHttpRequest(len(s.apiServers)),
		}
		// Expand servers to account for apis
		lis, err := utils.GetNextListener()
		if err != nil {
			return err
		}

		srv := &apiServer{
			lis: lis,
			srv: fhttp,
		}

		// get a free port and listen on that for this API
		go func(srv *apiServer) {
			err := srv.srv.Serve(srv.lis)
			if err != nil {
				fmt.Println(err)
			}
		}(srv)

		s.apiServers = append(s.apiServers, srv)
	}

	return nil
}

func (s *LocalGatewayService) createWebsocketServers() error {
	if s.socketServer == nil {
		s.socketServer = make(map[string]*socketServer)
	}

	for _, sock := range s.websocketWorkers {
		currSocket, ok := s.socketServer[sock]

		if !ok {
			fhttp := &fasthttp.Server{
				ReadTimeout:     time.Second * 1,
				IdleTimeout:     time.Second * 1,
				CloseOnShutdown: true,
				Handler:         s.handleWebsocketRequest(sock),
			}

			lis, err := utils.GetNextListener()
			if err != nil {
				return err
			}

			srv := &socketServer{
				lis:         lis,
				srv:         fhttp,
				workerCount: 0,
			}

			go func(srv *socketServer) {
				err := srv.srv.Serve(srv.lis)
				if err != nil {
					fmt.Println(err)
				}
			}(srv)

			currSocket = srv
			// append to the server collection
			s.socketServer[sock] = currSocket

			// this is a brand new server we need to start up
			// lets start it and add it to the active list of servers
			// we can then filter the servers by their active worker count
			currSocket.workerCount = 0
		}

		currSocket.workerCount = currSocket.workerCount + 1
	}

	return nil
}

func (s *LocalGatewayService) createHttpServers() error {
	// create an api server for every API worker
	for len(s.httpServers) < len(s.httpWorkers) {
		fhttp := &fasthttp.Server{
			ReadTimeout:     time.Second * 1,
			IdleTimeout:     time.Second * 1,
			CloseOnShutdown: true,
			Handler:         s.handleHttpProxyRequest(len(s.httpServers)),
		}
		// Expand servers to account for apis
		lis, err := utils.GetNextListener()
		if err != nil {
			return err
		}

		srv := &apiServer{
			lis: lis,
			srv: fhttp,
		}

		// get a free port and listen on that for this API
		go func(srv *apiServer) {
			err := srv.srv.Serve(srv.lis)
			if err != nil {
				fmt.Println(err)
			}
		}(srv)

		s.httpServers = append(s.httpServers, srv)
	}

	return nil
}

const nameParam = "{name}"

const (
	topicPath    = "/topic/" + nameParam
	schedulePath = "/schedules/" + nameParam
)

func (s *LocalGatewayService) GetTopicTriggerUrl(topicName string) string {
	// TODO: do the path build with the topicPath var
	endpoint, _ := url.JoinPath("http://"+s.GetTriggerAddress(), strings.Replace(topicPath, nameParam, topicName, 1))
	return endpoint
}

func (s *LocalGatewayService) GetScheduleManualTriggerUrl(scheduleName string) string {
	endpoint, _ := url.JoinPath("http://"+s.GetTriggerAddress(), strings.Replace(schedulePath, nameParam, scheduleName, 1))
	return endpoint
}

func (s *LocalGatewayService) Start(opts *gateway.GatewayStartOpts) error {
	var err error
	// Assign the pool and block
	s.options = opts
	s.stop = make(chan bool)

	// Setup routes
	r := router.New()
	// Publish to a topic
	r.POST(topicPath, s.handleTopicRequest)
	r.POST(schedulePath, s.handleSchedulesTrigger)

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

	if apiPlugin, ok := s.options.ApiPlugin.(*apis.LocalApiGatewayService); ok {
		apiPlugin.SubscribeToState(func(state apis.State) {
			s.refreshApis(state)
		})
	}

	if websocketPlugin, ok := s.options.WebsocketListenerPlugin.(*websockets.LocalWebsocketService); ok {
		websocketPlugin.SubscribeToState(func(state map[string][]websocketspb.WebsocketEventType) {
			s.refreshWebsocketWorkers(state)
		})
	}

	if httpProxyPlugin, ok := s.options.HttpPlugin.(*http.LocalHttpProxy); ok {
		httpProxyPlugin.SubscribeToState(func(state map[string]*workersHttp.HttpServer) {
			s.refreshHttpWorkers(state)
		})
	}

	go func() {
		_ = s.serviceServer.Serve(s.serviceListener)
	}()

	// block on a stop signal
	<-s.stop

	return nil
}

func (s *LocalGatewayService) Stop() error {
	for _, s := range s.apiServers {
		// shutdown all the servers
		// this will allow Start to exit
		_ = s.srv.Shutdown()
	}

	s.stop <- true

	return nil
}

// Create new HTTP gateway
// XXX: No External Args for function atm (currently the plugin loader does not pass any argument information)
func NewGateway(wsPlugin *websockets.LocalWebsocketService) (*LocalGatewayService, error) {
	return &LocalGatewayService{
		websocketPlugin: wsPlugin,
		bus:             EventBus.New(),
	}, nil
}
