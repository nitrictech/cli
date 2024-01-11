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

package dashboard

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/nitrictech/cli/pkg/eventbus"
	"github.com/nitrictech/cli/pkgplus/collector"
	dashboard_events "github.com/nitrictech/cli/pkgplus/dashboard/dashboard_events"
	"github.com/nitrictech/cli/pkgplus/dashboard/history"
	websocketspb "github.com/nitrictech/nitric/core/pkg/proto/websockets/v1"
	"github.com/olahol/melody"

	"github.com/nitrictech/cli/pkg/browser"
	"github.com/nitrictech/cli/pkg/update"
	"github.com/nitrictech/cli/pkg/utils"
	"github.com/nitrictech/cli/pkg/version"
	"github.com/nitrictech/cli/pkgplus/cloud/apis"
	"github.com/nitrictech/cli/pkgplus/cloud/gateway"
	"github.com/nitrictech/cli/pkgplus/cloud/schedules"
	"github.com/nitrictech/cli/pkgplus/cloud/storage"
	"github.com/nitrictech/cli/pkgplus/cloud/topics"
	"github.com/nitrictech/cli/pkgplus/cloud/websockets"
	"github.com/nitrictech/cli/pkgplus/project"
)

type WebsocketSpec struct {
	Name   string   `json:"name,omitempty"`
	Events []string `json:"events,omitempty"`
}

type Dashboard struct {
	project              *project.Project
	history              *history.History
	storageService       *storage.LocalStorageService
	gatewayService       *gateway.LocalGatewayService
	apis                 []*openapi3.T
	// schedules            []*codeconfig.TopicResult
	// topics               []*codeconfig.TopicResult
	buckets              []string
	websockets           []WebsocketSpec
	envMap               map[string]string
	stackWebSocket       *melody.Melody
	historyWebSocket     *melody.Melody
	wsWebSocket          *melody.Melody
	websocketsInfo       map[string]*dashboard_events.WebsocketInfo
	resourcesLastUpdated time.Time
	// bucketNotifications  []*codeconfig.BucketNotification
	port                 int
	hasStarted           bool
	connected            bool
	noBrowser            bool
}


type Api struct {
	Name    string                 `json:"name,omitempty"`
	OpenApi map[string]interface{} `json:"spec,omitempty"` // not sure which spec version yet
}

type DashboardResponse struct {
	Apis                []*openapi3.T                    `json:"apis"`
	Buckets             []string                         `json:"buckets"`
	// Schedules           []*codeconfig.TopicResult        `json:"schedules"`
	// Topics              []*codeconfig.TopicResult        `json:"topics"`
	Websockets          []WebsocketSpec                  `json:"websockets"`
	ProjectName         string                           `json:"projectName"`
	ApiAddresses        map[string]string                `json:"apiAddresses"`
	WebsocketAddresses  map[string]string                `json:"websocketAddresses"`
	TriggerAddress      string                           `json:"triggerAddress"`
	StorageAddress      string                           `json:"storageAddress"`
	// BucketNotifications []*codeconfig.BucketNotification `json:"bucketNotifications"`
	CurrentVersion      string                           `json:"currentVersion"`
	LatestVersion       string                           `json:"latestVersion"`
	Connected           bool                             `json:"connected"`
}

type Bucket struct {
	Name         string     `json:"name,omitempty"`
	CreationDate *time.Time `json:"creationDate,omitempty"`
}

//go:embed dist/*
var content embed.FS

func (d *Dashboard) AddBucket(name string) {
	// reset buckets to allow for most recent resources only
	if !d.resourcesLastUpdated.IsZero() && time.Since(d.resourcesLastUpdated) > time.Second*5 {
		d.buckets = []string{}
	}

	for _, b := range d.buckets {
		if b == name {
			return
		}
	}

	d.buckets = append(d.buckets, name)

	d.resourcesLastUpdated = time.Now()

	err := d.sendStackUpdate()
	if err != nil {
		fmt.Printf("Error sending stack update: %v\n", err)
		return
	}
}

func (d *Dashboard) UpdateApis(state apis.State) {
	apiSpecs, _ := collector.ApisToOpenApiSpecs(state)

	d.apis = apiSpecs

	d.refresh()
}

func (d *Dashboard) UpdateWebsockets(state websockets.State) {
	wsSpec := []WebsocketSpec{}

	for name, ws := range state {
		spec := WebsocketSpec{
			Name: name,
		}

		for _, eventType := range ws {
			switch eventType {
			case websocketspb.WebsocketEventType_Connect:
				spec.Events = append(spec.Events, "connect")
			case websocketspb.WebsocketEventType_Disconnect:
				spec.Events = append(spec.Events, "disconnect")
			case websocketspb.WebsocketEventType_Message:
				spec.Events = append(spec.Events, "message")
			}
		}

		wsSpec = append(wsSpec, spec)
	}

	d.websockets = wsSpec

	d.refresh()
}

func (d *Dashboard) UpdateTopics(state topics.State) {
	// TODO
	// for name, topic := range state {
	// 	println(name)
	// 	println(topic)
	// }

	// d.refresh()
}

func (d *Dashboard) UpdateSchedules(state schedules.State) {
	// TODO
	// for name, schedule := range state {
	// 	println(name)
	// 	println(schedule)
	// }

	// d.refresh()
}

func (d *Dashboard) refresh() {
	err := d.sendStackUpdate()
	if err != nil {
		fmt.Printf("Error sending stack update: %v\n", err)
		return
	}
}

func (d *Dashboard) RefreshHistory(_ *history.HistoryEvent[any]) error {
	return d.sendHistoryUpdate()
}

func (d *Dashboard) UpdateWebsocketInfoCount(socket string, count int) error {
	if d.websocketsInfo[socket] == nil {
		d.websocketsInfo[socket] = &dashboard_events.WebsocketInfo{}
	}

	d.websocketsInfo[socket].ConnectionCount = count

	err := d.sendWebsocketsUpdate()
	if err != nil {
		return err
	}

	return nil
}

func (d *Dashboard) AddWebsocketInfoMessage(socket string, message dashboard_events.WebsocketMessage) error {
	if d.websocketsInfo[socket] == nil {
		d.websocketsInfo[socket] = &dashboard_events.WebsocketInfo{}
	}
	
	d.websocketsInfo[socket].Messages = append([]dashboard_events.WebsocketMessage{message}, d.websocketsInfo[socket].Messages...)

	err := d.sendWebsocketsUpdate()
	if err != nil {
		return err
	}

	return nil
}

func (d *Dashboard) SetConnected(connected bool) {
	d.connected = connected
}

func (d *Dashboard) Start() error {
	// Get the embedded files from the 'dist' directory
	staticFiles, err := fs.Sub(content, "dist")
	if err != nil {
		return err
	}

	fs := http.FileServer(http.FS(staticFiles))

	// Serve the files using the http package
	http.Handle("/", fs)

	// handle websocket
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		err := d.stackWebSocket.HandleRequest(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// Register a handler for when a client connects
	d.stackWebSocket.HandleConnect(func(s *melody.Session) {
		// Send a welcome message to the client
		err := d.sendStackUpdate()
		if err != nil {
			log.Fatal(err)
		}
	})

	d.stackWebSocket.HandleMessage(func(s *melody.Session, msg []byte) {
		err := d.stackWebSocket.Broadcast(msg)
		if err != nil {
			log.Print(err)
		}
	})

	// handle history websocket
	http.HandleFunc("/history", func(w http.ResponseWriter, r *http.Request) {
		err := d.historyWebSocket.HandleRequest(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	d.historyWebSocket.HandleConnect(func(s *melody.Session) {
		// Send a welcome message to the client
		err := d.sendHistoryUpdate()
		if err != nil {
			log.Fatal(err)
		}
	})

	d.historyWebSocket.HandleMessage(func(s *melody.Session, msg []byte) {
		err := d.historyWebSocket.Broadcast(msg)
		if err != nil {
			log.Print(err)
		}
	})

	http.HandleFunc("/api/history", d.createHistoryHttpHandler())

	// Define an API route under /call to proxy communication between app and apis
	http.HandleFunc("/api/call/", d.createCallProxyHttpHandler())

	http.HandleFunc("/api/storage", d.handleStorage())

	// handle websockets
	http.HandleFunc("/ws-info", func(w http.ResponseWriter, r *http.Request) {
		err := d.wsWebSocket.HandleRequest(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	http.HandleFunc("/api/ws-clear-messages", d.handleWebsocketMessagesClear())

	d.wsWebSocket.HandleConnect(func(s *melody.Session) {
		// Send a welcome message to the client
		err := d.sendWebsocketsUpdate()
		if err != nil {
			log.Fatal(err)
		}
	})

	d.wsWebSocket.HandleMessage(func(s *melody.Session, msg []byte) {
		err := d.wsWebSocket.Broadcast(msg)
		if err != nil {
			log.Print(err)
		}
	})

	// using ephemeral ports, we will redirect to the dashboard on main api 4000
	dashListener, err := utils.GetNextListener(utils.MinPort(49152), utils.MaxPort(65535))
	if err != nil {
		return err
	}

	serveFn := func() {
		err = http.Serve(dashListener, nil)
		if err != nil {
			log.Fatal(err)
		}
	}
	go serveFn()

	d.port = dashListener.Addr().(*net.TCPAddr).Port

	// open browser
	if !d.noBrowser {
		err := browser.Open(d.GetDashboardUrl())
		if err != nil {
			return err
		}
	}

	d.hasStarted = true

	return nil
}

func (d *Dashboard) GetDashboardUrl() string {
	return fmt.Sprintf("http://localhost:%s", strconv.Itoa(d.port))
}

func (d *Dashboard) HasStarted() bool {
	return d.hasStarted
}

func handleResponseWriter(w http.ResponseWriter, data []byte) {
	_, err := w.Write(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (d *Dashboard) sendStackUpdate() error {
	currentVersion := strings.TrimPrefix(version.Version, "v")
	latestVersion := update.FetchLatestVersion()

	response := &DashboardResponse{
		Apis:                d.apis,
		// Topics:              d.topics,
		Buckets:             d.buckets,
		// Schedules:           d.schedules,
		Websockets:          d.websockets,
		ProjectName:         d.project.Name,
		ApiAddresses:        d.gatewayService.GetApiAddresses(),
		WebsocketAddresses:  d.gatewayService.GetWebsocketAddresses(),
		TriggerAddress:      d.gatewayService.GetTriggerAddress(),
		StorageAddress:      d.storageService.GetStorageEndpoint(),
		// BucketNotifications: d.bucketNotifications,
		CurrentVersion:      currentVersion,
		LatestVersion:       latestVersion,
		Connected:           d.connected,
	}

	// Encode the response as JSON
	jsonData, err := json.Marshal(response)
	if err != nil {
		return err
	}

	err = d.stackWebSocket.Broadcast(jsonData)

	return err
}

func (d *Dashboard) sendHistoryUpdate() error {
	response, err := d.history.ReadAllHistoryRecords()
	if err != nil {
		return err
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		return err
	}

	return d.historyWebSocket.Broadcast(jsonData)
}

func (d *Dashboard) sendWebsocketsUpdate() error {
	jsonData, err := json.Marshal(d.websocketsInfo)
	if err != nil {
		return err
	}

	err = d.wsWebSocket.Broadcast(jsonData)

	return err
}

func New(p *project.Project, noBrowser bool, storageService *storage.LocalStorageService, gatewayService *gateway.LocalGatewayService) (*Dashboard, error) {
	stackWebSocket := melody.New()
	historyWebSocket := melody.New()
	wsWebSocket := melody.New()

	dash := &Dashboard{
		project:             p,
		history:             history.New(p.Directory),
		storageService:      storageService,
		gatewayService:      gatewayService,
		apis:                []*openapi3.T{},
		envMap:              map[string]string{},
		stackWebSocket:      stackWebSocket,
		historyWebSocket:    historyWebSocket,
		wsWebSocket:         wsWebSocket,
		// bucketNotifications: []*codeconfig.BucketNotification{},
		// schedules:           []*codeconfig.TopicResult{},
		// topics:              []*codeconfig.TopicResult{},
		websocketsInfo:      map[string]*dashboard_events.WebsocketInfo{},
		hasStarted:          false,
		noBrowser:           noBrowser,
	}

	err := eventbus.Bus().Subscribe(dashboard_events.AddBucketTopic, dash.AddBucket)
	if err != nil {
		return nil, err
	}

	err = eventbus.Bus().Subscribe(history.AddRecordTopic, dash.RefreshHistory)
	if err != nil {
		return nil, err
	}

	err = eventbus.Bus().Subscribe(dashboard_events.AddWebsocketInfoTopic, dash.AddWebsocketInfoMessage)
	if err != nil {
		return nil, err
	}

	err = eventbus.Bus().Subscribe(dashboard_events.UpdateWebsocketInfoCountTopic, dash.UpdateWebsocketInfoCount)
	if err != nil {
		return nil, err
	}

	return dash, nil
}
