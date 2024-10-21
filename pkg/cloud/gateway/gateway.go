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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/asaskevich/EventBus"
	"github.com/fasthttp/router"
	"github.com/fasthttp/websocket"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/valyala/fasthttp"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/nitrictech/cli/pkg/cloud/apis"
	"github.com/nitrictech/cli/pkg/cloud/batch"
	"github.com/nitrictech/cli/pkg/cloud/http"
	"github.com/nitrictech/cli/pkg/cloud/schedules"
	"github.com/nitrictech/cli/pkg/cloud/topics"
	"github.com/nitrictech/cli/pkg/cloud/websockets"
	"github.com/nitrictech/cli/pkg/netx"
	"github.com/nitrictech/cli/pkg/project/localconfig"
	"github.com/nitrictech/cli/pkg/system"
	"github.com/nitrictech/cli/pkg/view/tui"

	base_http "github.com/nitrictech/nitric/cloud/common/runtime/gateway"

	"github.com/nitrictech/nitric/core/pkg/gateway"
	apispb "github.com/nitrictech/nitric/core/pkg/proto/apis/v1"
	batchpb "github.com/nitrictech/nitric/core/pkg/proto/batch/v1"
	schedulespb "github.com/nitrictech/nitric/core/pkg/proto/schedules/v1"
	topicspb "github.com/nitrictech/nitric/core/pkg/proto/topics/v1"
	websocketspb "github.com/nitrictech/nitric/core/pkg/proto/websockets/v1"
)

type apiServer struct {
	lis            net.Listener
	srv            *fasthttp.Server
	tlsCredentials *TLSCredentials
	name           string // name of the API or host
}

type socketServer struct {
	lis net.Listener
	srv *fasthttp.Server

	workerCount int
}

type TLSCredentials struct {
	// CertFile - Path to the certificate file
	CertFile string
	// KeyFile - Path to the private key file
	KeyFile string
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
	apisPlugin       *apis.LocalApiGatewayService
	websocketPlugin  *websockets.LocalWebsocketService
	topicsPlugin     *topics.LocalTopicsAndSubscribersService
	schedulesPlugin  *schedules.LocalSchedulesService
	batchPlugin      *batch.LocalBatchService
	serviceListener  net.Listener

	localConfig localconfig.LocalConfiguration

	logWriter io.Writer

	ApiTlsCredentials *TLSCredentials

	lock sync.RWMutex
	gateway.UnimplementedGatewayPlugin
	stop chan bool

	options *gateway.GatewayStartOpts
	bus     EventBus.Bus
}

var _ gateway.GatewayService = &LocalGatewayService{}

// GetTriggerAddress - Returns the base address built-in nitric services, like schedules and topics, will be exposed on.
func (s *LocalGatewayService) GetTriggerAddress() string {
	if s.serviceListener != nil {
		return strings.Replace(s.serviceListener.Addr().String(), "[::]", "localhost", 1)
	}

	return ""
}

// GetApiAddresses - Returns a map of API names to their addresses, including protocol and port
func (s *LocalGatewayService) GetApiAddresses() map[string]string {
	s.lock.RLock()
	defer s.lock.RUnlock()

	addresses := make(map[string]string)

	if len(s.apiServers) > 0 && len(s.apis) == len(s.apiServers) {
		for _, srv := range s.apiServers {
			protocol := "http"
			if srv.tlsCredentials != nil {
				protocol = "https"
			}

			address := strings.Replace(srv.lis.Addr().String(), "[::]", "localhost", 1)

			addresses[srv.name] = fmt.Sprintf("%s://%s", protocol, address)
		}
	}

	return addresses
}

func (s *LocalGatewayService) GetHttpWorkerAddresses() map[string]string {
	s.lock.RLock()
	defer s.lock.RUnlock()

	addresses := make(map[string]string)

	if len(s.httpServers) > 0 && len(s.httpWorkers) == len(s.httpServers) {
		for _, srv := range s.httpServers {
			protocol := "http"
			if srv.tlsCredentials != nil {
				protocol = "https"
			}

			address := strings.Replace(srv.lis.Addr().String(), "[::]", "localhost", 1)

			addresses[srv.name] = fmt.Sprintf("%s://%s", protocol, address)
		}
	}

	return addresses
}

func (s *LocalGatewayService) GetWebsocketAddresses() map[string]string {
	s.lock.RLock()
	defer s.lock.RUnlock()

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
		port := s.httpWorkers[idx]

		// set port so http plugin can find server from state
		requestCopy := &fasthttp.Request{}
		ctx.Request.CopyTo(requestCopy)
		requestCopy.URI().SetHost(port)
		// TODO: Need to support multiple HTTP handlers
		// so a plugin wrapper will be required for this
		resp, err := s.options.HttpPlugin.HandleRequest(requestCopy)
		if err != nil {
			ctx.Error(fmt.Sprintf("Error handling HTTP Request: %v", err), 500)
			return
		}

		resp.CopyTo(&ctx.Response)
	}
}

func (s *LocalGatewayService) handleApiHttpRequest(apiName string) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		if !s.apiServerExists(apiName) {
			ctx.Error("Sorry, nitric is listening on this port but is waiting for an API to be available to handle requests, you may have removed an API during development this port will be assigned to an API when one becomes available", 404)
			return
		}

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
			s.apisPlugin.PublishActionState(apis.ApiRequestState{
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

		if resp.GetWebsocketEventResponse() == nil || (resp.GetWebsocketEventResponse().GetConnectionResponse() != nil && resp.GetWebsocketEventResponse().GetConnectionResponse().Reject) {
			// close the connection
			ctx.Error("Connection Refused", 500)
			return
		}

		err = upgrader.Upgrade(ctx, func(ws *websocket.Conn) {
			// generate a new connection ID for this client
			defer func() {
				// close within the websocket plugin will also call ws.Close
				_, err = s.websocketPlugin.CloseConnection(ctx, &websocketspb.WebsocketCloseConnectionRequest{
					ConnectionId: connectionId,
					SocketName:   socketName,
				})
				if err != nil {
					tui.Error.Println(err.Error())
					return
				}
			}()

			err = s.websocketPlugin.RegisterConnection(socketName, connectionId, ws)
			if err != nil {
				tui.Error.Println(err.Error())
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
					tui.Error.Println(err.Error())
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
				tui.Error.Println(err.Error())
				return
			}
		})
		if err != nil {
			if _, ok := err.(websocket.HandshakeError); ok {
				tui.Error.Println(err.Error())
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

	_, err = s.topicsPlugin.Publish(ctx, &topicspb.TopicPublishRequest{
		TopicName: topicName,
		Message: &topicspb.TopicMessage{
			Content: &topicspb.TopicMessage_StructPayload{
				StructPayload: structPayload,
			},
		},
	})
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

	_, err := s.schedulesPlugin.HandleRequest(msg)
	if err != nil {
		ctx.Error(fmt.Sprintf("Error handling schedule trigger: %v", err), 500)
		return
	}

	ctx.SuccessString("text/plain", "Successfully triggered schedule")
}

func (s *LocalGatewayService) handleBatchJobTrigger(ctx *fasthttp.RequestCtx) {
	jobName := ctx.UserValue("name").(string)

	// Get the incoming data as JobData_Struct
	payload := map[string]interface{}{}

	err := json.Unmarshal(ctx.Request.Body(), &payload)
	if err != nil {
		ctx.Error(fmt.Sprintf("Error parsing JSON: %v", err), 400)
		return
	}

	st, err := structpb.NewStruct(payload)
	if err != nil {
		ctx.Error(fmt.Sprintf("Error serializing job message from payload: %v", err), 400)
		return
	}

	jobSubmitRequest := &batchpb.JobSubmitRequest{
		JobName: jobName,
		Data:    &batchpb.JobData{Data: &batchpb.JobData_Struct{Struct: st}},
	}

	_, err = s.batchPlugin.SubmitJob(context.Background(), jobSubmitRequest)
	if err != nil {
		ctx.Error(fmt.Sprintf("Error handling batch job trigger: %v", err), 500)
		return
	}

	ctx.SuccessString("text/plain", "Successfully triggered job")
}

func (s *LocalGatewayService) refreshApis(apiState apis.State) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// api has been removed
	if len(apiState) < len(s.apiServers) {
		// shutdown the apis that have been removed
		s.apiServers = lo.Filter(s.apiServers, func(item *apiServer, index int) bool {
			_, exists := apiState[item.name]

			if !exists {
				shutdownServer(item.srv)
			}

			return exists
		})
	}

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

	err := s.createApiServers()
	if err != nil {
		system.Log(fmt.Sprintf("error creating api servers: %s", err.Error()))
	}
}

func (s *LocalGatewayService) refreshHttpWorkers(state http.State) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.httpWorkers = make([]string, 0)

	// http server has been removed
	if len(state) < len(s.httpServers) {
		// shutdown the http servers that have been removed
		s.httpServers = lo.Filter(s.httpServers, func(item *apiServer, index int) bool {
			_, exists := state[item.name]

			if !exists {
				shutdownServer(item.srv)
			}

			return exists
		})
	}

	uniqHttpWorkers := lo.Reduce(lo.Keys(state), func(agg []string, host string, idx int) []string {
		if !lo.Contains(agg, host) {
			agg = append(agg, host)
		}

		return agg
	}, []string{})

	// sort the Http Worker Ports lowest to highest
	sort.Strings(uniqHttpWorkers)

	s.httpWorkers = append(s.httpWorkers, uniqHttpWorkers...)

	err := s.createHttpServers()
	if err != nil {
		system.Log(fmt.Sprintf("error creating http servers: %s", err.Error()))
	}
}

func (s *LocalGatewayService) refreshWebsocketWorkers(state websockets.State) {
	s.lock.Lock()

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

	// TODO move thread-safe lists/maps to own type so no deadlocks are possible
	s.lock.Unlock()

	err := s.createWebsocketServers()
	if err != nil {
		system.Log(fmt.Sprintf("error creating websocket servers: %s", err.Error()))
	}
}

func (s *LocalGatewayService) createApiServers() error {
	// create an api server for every API worker
	for _, apiName := range s.apis {
		if s.apiServerExists(apiName) {
			continue
		}

		lis, err := getListener(s.localConfig.Apis, apiName)
		if err != nil {
			return err
		}

		fhttp := &fasthttp.Server{
			ReadTimeout:     time.Second * 1,
			IdleTimeout:     time.Second * 1,
			CloseOnShutdown: true,
			ReadBufferSize:  8192,
			Handler:         s.handleApiHttpRequest(apiName),
			Logger:          log.New(s.logWriter, fmt.Sprintf("%s: ", lis.Addr().String()), 0),
		}

		srv := &apiServer{
			lis:            lis,
			srv:            fhttp,
			tlsCredentials: s.ApiTlsCredentials,
			name:           apiName,
		}

		// get a free port and listen on that for this API
		go func(srv *apiServer) {
			var err error
			if srv.tlsCredentials != nil {
				err = srv.srv.ServeTLS(srv.lis, srv.tlsCredentials.CertFile, srv.tlsCredentials.KeyFile)
			} else {
				err = srv.srv.Serve(srv.lis)
			}

			if err != nil {
				fmt.Println(err)
			}
		}(srv)

		s.apiServers = append(s.apiServers, srv)
	}

	return nil
}

func (s *LocalGatewayService) apiServerExists(apiName string) bool {
	return lo.SomeBy(s.apiServers, func(as *apiServer) bool {
		return as.name == apiName
	})
}

func getListener(mapping map[string]localconfig.LocalResourceConfiguration, name string) (net.Listener, error) {
	if config, exists := mapping[name]; exists {
		if config.Port != 0 {
			list, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Port))
			if err != nil {
				return nil, fmt.Errorf("error mapping %s to port %d, %s", name, config.Port, err.Error())
			}

			return list, nil
		}
	}

	return netx.GetNextListener()
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

			lis, err := getListener(s.localConfig.Websockets, sock)
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

	s.websocketPlugin.SetServers(s.GetWebsocketAddresses())

	return nil
}

func (s *LocalGatewayService) createHttpServers() error {
	// Expand servers to account for apis
	lis, err := netx.GetNextListener()
	if err != nil {
		return err
	}

	// create an api server for every API worker
	for len(s.httpServers) < len(s.httpWorkers) {
		fhttp := &fasthttp.Server{
			ReadTimeout:     time.Second * 1,
			IdleTimeout:     time.Second * 1,
			CloseOnShutdown: true,
			ReadBufferSize:  8192,
			Handler:         s.handleHttpProxyRequest(len(s.httpServers)),
			Logger:          log.New(s.logWriter, fmt.Sprintf("%s: ", lis.Addr().String()), 0),
		}

		srv := &apiServer{
			lis:            lis,
			srv:            fhttp,
			tlsCredentials: s.ApiTlsCredentials,
			name:           s.httpWorkers[len(s.httpServers)],
		}

		// get a free port and listen on that for this API
		go func(srv *apiServer) {
			var err error
			if srv.tlsCredentials != nil {
				err = srv.srv.ServeTLS(srv.lis, srv.tlsCredentials.CertFile, srv.tlsCredentials.KeyFile)
			} else {
				err = srv.srv.Serve(srv.lis)
			}

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
	topicPath    = "/topics/" + nameParam
	schedulePath = "/schedules/" + nameParam
	batchPath    = "/jobs/" + nameParam
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

func (s *LocalGatewayService) GetBatchTriggerUrl(jobName string) string {
	endpoint, _ := url.JoinPath("http://"+s.GetTriggerAddress(), strings.Replace(batchPath, nameParam, jobName, 1))
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
	r.POST(batchPath, s.handleBatchJobTrigger)

	s.serviceServer = &fasthttp.Server{
		ReadTimeout:     time.Second * 1,
		IdleTimeout:     time.Second * 1,
		CloseOnShutdown: true,
		ReadBufferSize:  8192,
		Handler:         r.Handler,
	}

	s.serviceListener, err = netx.GetNextListener()
	if err != nil {
		return err
	}

	if apiPlugin, ok := s.options.ApiPlugin.(*apis.LocalApiGatewayService); ok {
		apiPlugin.SubscribeToState(func(state apis.State) {
			s.refreshApis(state)
		})

		s.apisPlugin = apiPlugin
	}

	if topicsPlugin, ok := s.options.TopicsListenerPlugin.(*topics.LocalTopicsAndSubscribersService); ok {
		s.topicsPlugin = topicsPlugin
	}

	if schedulesPlugin, ok := s.options.SchedulesPlugin.(*schedules.LocalSchedulesService); ok {
		s.schedulesPlugin = schedulesPlugin
	}

	if websocketPlugin, ok := s.options.WebsocketListenerPlugin.(*websockets.LocalWebsocketService); ok {
		websocketPlugin.SubscribeToState(func(state map[string]map[string][]websocketspb.WebsocketEventType) {
			s.refreshWebsocketWorkers(state)
		})

		s.websocketPlugin = websocketPlugin
	}

	if httpProxyPlugin, ok := s.options.HttpPlugin.(*http.LocalHttpProxy); ok {
		httpProxyPlugin.SubscribeToState(func(state map[string]*http.HttpProxyService) {
			s.refreshHttpWorkers(state)
		})
	}

	return s.serviceServer.Serve(s.serviceListener)
}

func shutdownServer(srv *fasthttp.Server) {
	// Shutdown the server
	// This will allow Start to exit
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()

	_ = srv.ShutdownWithContext(ctx)
}

func (s *LocalGatewayService) Stop() error {
	// Shutdown all the api servers
	for _, as := range s.apiServers {
		shutdownServer(as.srv)
	}

	// Shutdown all the http servers
	for _, hs := range s.httpServers {
		shutdownServer(hs.srv)
	}

	// Shutdown all the websocket servers
	for _, ss := range s.socketServer {
		shutdownServer(ss.srv)
	}

	if s.serviceServer != nil {
		return s.serviceServer.Shutdown()
	}

	return nil
}

type NewGatewayOpts struct {
	TLSCredentials *TLSCredentials
	LogWriter      io.Writer
	LocalConfig    localconfig.LocalConfiguration
	BatchPlugin    *batch.LocalBatchService
}

// Create new HTTP gateway
// XXX: No External Args for function atm (currently the plugin loader does not pass any argument information)
func NewGateway(opts NewGatewayOpts) (*LocalGatewayService, error) {
	return &LocalGatewayService{
		ApiTlsCredentials: opts.TLSCredentials,
		bus:               EventBus.New(),
		logWriter:         opts.LogWriter,
		localConfig:       opts.LocalConfig,
		batchPlugin:       opts.BatchPlugin,
	}, nil
}
