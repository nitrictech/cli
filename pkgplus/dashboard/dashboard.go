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
	"github.com/samber/lo"
	"github.com/spf13/afero"

	"github.com/nitrictech/cli/pkgplus/browser"
	"github.com/nitrictech/cli/pkgplus/cloud"
	"github.com/nitrictech/cli/pkgplus/collector"
	"github.com/nitrictech/cli/pkgplus/netx"
	apispb "github.com/nitrictech/nitric/core/pkg/proto/apis/v1"
	websocketspb "github.com/nitrictech/nitric/core/pkg/proto/websockets/v1"

	"github.com/nitrictech/cli/pkgplus/cloud/apis"
	"github.com/nitrictech/cli/pkgplus/cloud/gateway"
	"github.com/nitrictech/cli/pkgplus/cloud/schedules"
	"github.com/nitrictech/cli/pkgplus/cloud/storage"
	"github.com/nitrictech/cli/pkgplus/cloud/topics"
	"github.com/nitrictech/cli/pkgplus/cloud/websockets"
	"github.com/nitrictech/cli/pkgplus/project"
	"github.com/nitrictech/cli/pkgplus/update"
	"github.com/nitrictech/cli/pkgplus/version"
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

type BucketSpec struct {
	Name              string `json:"name,omitempty"`
	NotificationCount int    `json:"notificationCount"`
}

type Dashboard struct {
	project                    *project.Project
	storageService             *storage.LocalStorageService
	gatewayService             *gateway.LocalGatewayService
	apis                       []*openapi3.T
	schedules                  []ScheduleSpec
	topics                     []TopicSpec
	buckets                    []*BucketSpec
	websockets                 []WebsocketSpec
	envMap                     map[string]string
	stackWebSocket             *melody.Melody
	historyWebSocket           *melody.Melody
	wsWebSocket                *melody.Melody
	websocketsInfo             map[string]*websockets.WebsocketInfo
	bucketResourcesLastUpdated time.Time
	port                       int
	browserHasOpened           bool
	noBrowser                  bool
	browserLock                sync.Mutex
	debouncedUpdate            func()
}

type DashboardResponse struct {
	Apis               []*openapi3.T     `json:"apis"`
	Buckets            []*BucketSpec     `json:"buckets"`
	Schedules          []ScheduleSpec    `json:"schedules"`
	Topics             []TopicSpec       `json:"topics"`
	Websockets         []WebsocketSpec   `json:"websockets"`
	ProjectName        string            `json:"projectName"`
	ApiAddresses       map[string]string `json:"apiAddresses"`
	WebsocketAddresses map[string]string `json:"websocketAddresses"`
	TriggerAddress     string            `json:"triggerAddress"`
	StorageAddress     string            `json:"storageAddress"`
	CurrentVersion     string            `json:"currentVersion"`
	LatestVersion      string            `json:"latestVersion"`
	Connected          bool              `json:"connected"`
}

type Bucket struct {
	Name         string     `json:"name,omitempty"`
	CreationDate *time.Time `json:"creationDate,omitempty"`
}

//go:embed dist/*
var content embed.FS

func (d *Dashboard) addBucket(name string) {
	// reset buckets to allow for most recent resources only
	if !d.bucketResourcesLastUpdated.IsZero() && time.Since(d.bucketResourcesLastUpdated) > time.Second*5 {
		d.buckets = []*BucketSpec{}
	}

	for _, b := range d.buckets {
		if b.Name == name {
			return
		}
	}

	d.buckets = append(d.buckets, &BucketSpec{
		Name:              name,
		NotificationCount: 0,
	})

	d.bucketResourcesLastUpdated = time.Now()

	d.refresh()
}

func (d *Dashboard) updateApis(state apis.State) {
	apis := make(map[string][]*apispb.RegistrationRequest, 0)

	for apiName, api := range state {
		for _, routes := range lo.Values(api) {
			apis[apiName] = append(apis[apiName], routes...)
		}
	}

	apiSpecs, _ := collector.ApisToOpenApiSpecs(apis, &collector.ProjectErrors{})

	d.apis = apiSpecs

	d.refresh()
}

func (d *Dashboard) updateWebsockets(state websockets.State) {
	wsSpec := []WebsocketSpec{}

	for name, ws := range state {
		spec := WebsocketSpec{
			Name: name,
		}

		for _, serviceWs := range ws {
			for _, eventType := range serviceWs {
				switch eventType {
				case websocketspb.WebsocketEventType_Connect:
					spec.Events = append(spec.Events, "connect")
				case websocketspb.WebsocketEventType_Disconnect:
					spec.Events = append(spec.Events, "disconnect")
				case websocketspb.WebsocketEventType_Message:
					spec.Events = append(spec.Events, "message")
				}
			}
		}

		wsSpec = append(wsSpec, spec)
	}

	d.websockets = wsSpec

	d.refresh()
}

func (d *Dashboard) updateTopics(state topics.State) {
	topics := []TopicSpec{}

	for topic, services := range state {
		count := 0
		for _, serviceSubCount := range services {
			count += serviceSubCount
		}
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
			Name:       schedule.Schedule.GetScheduleName(),
			Expression: schedule.Schedule.GetCron().GetExpression(),
			Rate:       schedule.Schedule.GetEvery().GetRate(),
		})
	}

	d.schedules = schedules

	d.refresh()
}

func (d *Dashboard) updateBucketNotifications(state storage.State) {
	var performUpdate bool

	for bucketName, serviceListnerCount := range state {
		count := 0
		for _, serviceCount := range serviceListnerCount {
			count += serviceCount
		}
		_, idx, found := lo.FindIndexOf[*BucketSpec](d.buckets, func(item *BucketSpec) bool {
			return item.Name == bucketName
		})

		if found && d.buckets[idx] != nil {
			d.buckets[idx].NotificationCount = count
			performUpdate = true
		}
	}

	if performUpdate {
		d.refresh()
	}
}

func (d *Dashboard) refresh() {
	if !d.noBrowser && !d.browserHasOpened {
		d.openBrowser()
	}

	d.debouncedUpdate()
}

func (d *Dashboard) isConnected() bool {
	apisRegistered := len(d.apis) > 0
	websocketsRegistered := len(d.websockets) > 0
	topicsRegistered := len(d.topics) > 0
	schedulesRegistered := len(d.schedules) > 0

	bucketNotificationsRegistered := false

	// Note: buckets arent completely removed at the moment, but the NotificationCount is.
	for _, bs := range d.buckets {
		if bs.NotificationCount > 0 {
			return true
		}
	}

	return apisRegistered || websocketsRegistered || topicsRegistered || schedulesRegistered || bucketNotificationsRegistered
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
	dashListener, err := netx.GetNextListener(netx.MinPort(49152), netx.MaxPort(65535))
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
		CurrentVersion:     currentVersion,
		LatestVersion:      latestVersion,
		Connected:          d.isConnected(),
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
		schedules:        []ScheduleSpec{},
		topics:           []TopicSpec{},
		websockets:       []WebsocketSpec{},
		websocketsInfo:   map[string]*websockets.WebsocketInfo{},
		noBrowser:        noBrowser,
	}

	debouncedUpdate, _ := lo.NewDebounce(300*time.Millisecond, func() {
		err := dash.sendStackUpdate()
		if err != nil {
			fmt.Printf("Error sending stack update: %v\n", err)
			return
		}
	})

	dash.debouncedUpdate = debouncedUpdate

	// FIXME:
	// err = eventbus.Bus().Subscribe(resources.DeclareBucketTopic, dash.addBucket)
	// if err != nil {
	// 	return nil, err
	// }

	localCloud.Apis.SubscribeToState(dash.updateApis)
	localCloud.Websockets.SubscribeToState(dash.updateWebsockets)
	localCloud.Schedules.SubscribeToState(dash.updateSchedules)
	localCloud.Topics.SubscribeToState(dash.updateTopics)
	localCloud.Storage.SubscribeToState(dash.updateBucketNotifications)

	// subscribe to history events from gateway
	localCloud.Apis.SubscribeToAction(dash.handleApiHistory)
	localCloud.Topics.SubscribeToAction(dash.handleTopicsHistory)
	localCloud.Schedules.SubscribeToAction(dash.handleSchedulesHistory)
	localCloud.Websockets.SubscribeToAction(dash.handleWebsocketEvents)

	return dash, nil
}
