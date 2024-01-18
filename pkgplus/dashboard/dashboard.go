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
	"sync"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/olahol/melody"
	"github.com/spf13/afero"

	"github.com/nitrictech/cli/pkg/browser"
	"github.com/nitrictech/cli/pkg/eventbus"
	"github.com/nitrictech/cli/pkgplus/cloud"
	"github.com/nitrictech/cli/pkgplus/collector"
	websocketspb "github.com/nitrictech/nitric/core/pkg/proto/websockets/v1"

	"github.com/nitrictech/cli/pkg/update"
	"github.com/nitrictech/cli/pkg/utils"
	"github.com/nitrictech/cli/pkg/version"
	"github.com/nitrictech/cli/pkgplus/cloud/apis"
	"github.com/nitrictech/cli/pkgplus/cloud/gateway"
	"github.com/nitrictech/cli/pkgplus/cloud/resources"
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

type ScheduleSpec struct {
	Name       string `json:"name,omitempty"`
	Expression string `json:"expression,omitempty"`
	Rate       string `json:"rate,omitempty"`
}

type TopicSpec struct {
	Name            string `json:"name,omitempty"`
	SubscriberCount int    `json:"subscriberCount"`
}

type Dashboard struct {
	project              *project.Project
	storageService       *storage.LocalStorageService
	gatewayService       *gateway.LocalGatewayService
	apis                 []*openapi3.T
	schedules            []ScheduleSpec
	topics               []TopicSpec
	buckets              []string
	websockets           []WebsocketSpec
	envMap               map[string]string
	stackWebSocket       *melody.Melody
	historyWebSocket     *melody.Melody
	wsWebSocket          *melody.Melody
	websocketsInfo       map[string]*websockets.WebsocketInfo
	resourcesLastUpdated time.Time
	// bucketNotifications  []*codeconfig.BucketNotification
	port             int
	browserHasOpened bool
	connected        bool
	noBrowser        bool
	browserLock      sync.Mutex
}

type DashboardResponse struct {
	Apis               []*openapi3.T     `json:"apis"`
	Buckets            []string          `json:"buckets"`
	Schedules          []ScheduleSpec    `json:"schedules"`
	Topics             []TopicSpec       `json:"topics"`
	Websockets         []WebsocketSpec   `json:"websockets"`
	ProjectName        string            `json:"projectName"`
	ApiAddresses       map[string]string `json:"apiAddresses"`
	WebsocketAddresses map[string]string `json:"websocketAddresses"`
	TriggerAddress     string            `json:"triggerAddress"`
	StorageAddress     string            `json:"storageAddress"`
	// BucketNotifications []*codeconfig.BucketNotification `json:"bucketNotifications"`
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	Connected      bool   `json:"connected"`
}

type Bucket struct {
	Name         string     `json:"name,omitempty"`
	CreationDate *time.Time `json:"creationDate,omitempty"`
}

//go:embed dist/*
var content embed.FS

func (d *Dashboard) addBucket(name string) {
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

func (d *Dashboard) updateApis(state apis.State) {
	apiSpecs, _ := collector.ApisToOpenApiSpecs(state)

	d.apis = apiSpecs

	d.refresh()
}

func (d *Dashboard) updateWebsockets(state websockets.State) {
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

func (d *Dashboard) updateTopics(state topics.State) {
	topics := []TopicSpec{}

	for topic, count := range state {
		topics = append(topics, TopicSpec{
			Name:            topic,
			SubscriberCount: count,
		})
	}

	d.topics = topics

	d.refresh()
}

func (d *Dashboard) updateSchedules(state schedules.State) {
	schedules := []ScheduleSpec{}

	for _, schedule := range state {
		schedules = append(schedules, ScheduleSpec{
			Name:       schedule.GetScheduleName(),
			Expression: schedule.GetCron().GetExpression(),
			Rate:       schedule.GetEvery().GetRate(),
		})
	}

	d.schedules = schedules

	d.refresh()
}

func (d *Dashboard) refresh() {
	// TODO need to determine how to know if connected
	d.connected = true

	if !d.noBrowser && !d.browserHasOpened {
		d.openBrowser()
	}

	err := d.sendStackUpdate()
	if err != nil {
		fmt.Printf("Error sending stack update: %v\n", err)
		return
	}
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

	return nil
}

func (d *Dashboard) openBrowser() {
	d.browserLock.Lock()
	defer d.browserLock.Unlock()

	if d.browserHasOpened {
		return // Browser already opened
	}

	err := browser.Open(d.GetDashboardUrl())
	if err != nil {
		fmt.Printf("Error opening dashboard in browser: %v\n", err)
		return
	}

	d.browserHasOpened = true
}

func (d *Dashboard) GetDashboardUrl() string {
	return fmt.Sprintf("http://localhost:%s", strconv.Itoa(d.port))
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
		Apis:               d.apis,
		Topics:             d.topics,
		Buckets:            d.buckets,
		Schedules:          d.schedules,
		Websockets:         d.websockets,
		ProjectName:        d.project.Name,
		ApiAddresses:       d.gatewayService.GetApiAddresses(),
		WebsocketAddresses: d.gatewayService.GetWebsocketAddresses(),
		TriggerAddress:     d.gatewayService.GetTriggerAddress(),
		StorageAddress:     d.storageService.GetStorageEndpoint(),
		// BucketNotifications: d.bucketNotifications,
		CurrentVersion: currentVersion,
		LatestVersion:  latestVersion,
		Connected:      d.connected,
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
	response, err := d.ReadAllHistoryRecords()
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

func New(noBrowser bool, localCloud *cloud.LocalCloud) (*Dashboard, error) {
	fs := afero.NewOsFs()

	p, err := project.FromFile(fs, "")
	if err != nil {
		return nil, err
	}

	stackWebSocket := melody.New()
	historyWebSocket := melody.New()
	wsWebSocket := melody.New()

	dash := &Dashboard{
		project:          p,
		storageService:   localCloud.Storage,
		gatewayService:   localCloud.Gateway,
		apis:             []*openapi3.T{},
		envMap:           map[string]string{},
		stackWebSocket:   stackWebSocket,
		historyWebSocket: historyWebSocket,
		wsWebSocket:      wsWebSocket,
		// bucketNotifications: []*codeconfig.BucketNotification{},
		schedules:      []ScheduleSpec{},
		topics:         []TopicSpec{},
		websocketsInfo: map[string]*websockets.WebsocketInfo{},
		noBrowser:      noBrowser,
	}

	err = eventbus.Bus().Subscribe(resources.DeclareBucketTopic, dash.addBucket)
	if err != nil {
		return nil, err
	}

	localCloud.Apis.SubscribeToState(dash.updateApis)
	localCloud.Websockets.SubscribeToState(dash.updateWebsockets)
	localCloud.Schedules.SubscribeToState(dash.updateSchedules)
	localCloud.Topics.SubscribeToState(dash.updateTopics)

	// subscribe to history events from gateway
	localCloud.Apis.SubscribeToAction(dash.handleApiHistory)
	localCloud.Topics.SubscribeToAction(dash.handleTopicsHistory)
	localCloud.Schedules.SubscribeToAction(dash.handleSchedulesHistory)
	localCloud.Websockets.SubscribeToAction(dash.handleWebsocketEvents)

	return dash, nil
}
