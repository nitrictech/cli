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

	"github.com/nitrictech/cli/pkgplus/browser"
	"github.com/nitrictech/cli/pkgplus/cloud"
	"github.com/nitrictech/cli/pkgplus/collector"
	"github.com/nitrictech/cli/pkgplus/netx"
	resourcespb "github.com/nitrictech/nitric/core/pkg/proto/resources/v1"
	websocketspb "github.com/nitrictech/nitric/core/pkg/proto/websockets/v1"

	"github.com/nitrictech/cli/pkgplus/cloud/apis"
	"github.com/nitrictech/cli/pkgplus/cloud/gateway"
	"github.com/nitrictech/cli/pkgplus/cloud/resources"
	"github.com/nitrictech/cli/pkgplus/cloud/schedules"
	"github.com/nitrictech/cli/pkgplus/cloud/storage"
	"github.com/nitrictech/cli/pkgplus/cloud/topics"
	"github.com/nitrictech/cli/pkgplus/cloud/websockets"
	"github.com/nitrictech/cli/pkgplus/project"
	"github.com/nitrictech/cli/pkgplus/update"
	"github.com/nitrictech/cli/pkgplus/version"
)

type BaseResourceSpec struct {
	Name string `json:"name"`
	// TODO: Remove this field
	RequestingServices []string `json:"requestingServices"`
}

type ApiSpec struct {
	*BaseResourceSpec

	OpenApiSpec *openapi3.T `json:"spec"`
}
type WebsocketSpec struct {
	*BaseResourceSpec

	Events []string `json:"events,omitempty"`

	Targets map[string]string `json:"targets,omitempty"`
}

type ServiceSpec struct {
	*BaseResourceSpec

	FilePath string `json:"filePath"`
}

type ScheduleSpec struct {
	*BaseResourceSpec

	Expression string `json:"expression,omitempty"`
	Rate       string `json:"rate,omitempty"`
	Target     string `json:"target,omitempty"`
}

type TopicSpec struct {
	*BaseResourceSpec
}

type BucketSpec struct {
	*BaseResourceSpec
}

type NotifierSpec struct {
	Bucket string `json:"bucket"`
	Target string `json:"target"`
}

type SubscriberSpec struct {
	Topic  string `json:"topic"`
	Target string `json:"target"`
}

type PolicyResource struct {
	Name string `json:"name"`
	Type string `json:"type"`
}
type PolicySpec struct {
	*BaseResourceSpec

	Principals []PolicyResource `json:"principals"`
	Actions    []string         `json:"actions"`
	Resources  []PolicyResource `json:"resources"`
}

type Dashboard struct {
	resourcesLock  sync.Mutex
	project        *project.Project
	storageService *storage.LocalStorageService
	gatewayService *gateway.LocalGatewayService
	apis           []ApiSpec
	schedules      []ScheduleSpec
	topics         []*TopicSpec
	buckets        []*BucketSpec
	websockets     []WebsocketSpec
	subscriptions  []*SubscriberSpec
	notifications  []*NotifierSpec

	policies map[string]PolicySpec
	envMap   map[string]string

	stackWebSocket   *melody.Melody
	historyWebSocket *melody.Melody
	wsWebSocket      *melody.Melody
	websocketsInfo   map[string]*websockets.WebsocketInfo
	port             int
	browserHasOpened bool
	noBrowser        bool
	browserLock      sync.Mutex
	debouncedUpdate  func()
}

type DashboardResponse struct {
	Apis          []ApiSpec         `json:"apis"`
	Buckets       []*BucketSpec     `json:"buckets"`
	Schedules     []ScheduleSpec    `json:"schedules"`
	Topics        []*TopicSpec      `json:"topics"`
	Websockets    []WebsocketSpec   `json:"websockets"`
	Subscriptions []*SubscriberSpec `json:"subscriptions"`
	Notifications []*NotifierSpec   `json:"notifications"`

	Services []*ServiceSpec `json:"services"`

	Policies           map[string]PolicySpec `json:"policies"`
	ProjectName        string                `json:"projectName"`
	ApiAddresses       map[string]string     `json:"apiAddresses"`
	WebsocketAddresses map[string]string     `json:"websocketAddresses"`
	TriggerAddress     string                `json:"triggerAddress"`
	StorageAddress     string                `json:"storageAddress"`
	CurrentVersion     string                `json:"currentVersion"`
	LatestVersion      string                `json:"latestVersion"`
	Connected          bool                  `json:"connected"`
}

type Bucket struct {
	Name         string     `json:"name,omitempty"`
	CreationDate *time.Time `json:"creationDate,omitempty"`
}

//go:embed dist/*
var content embed.FS

func (d *Dashboard) getServices() ([]*ServiceSpec, error) {
	serviceSpecs := []*ServiceSpec{}

	for _, service := range d.project.GetServices() {
		absPath, err := service.GetAbsoluteFilePath()
		if err != nil {
			return nil, err
		}

		serviceSpecs = append(serviceSpecs, &ServiceSpec{
			BaseResourceSpec: &BaseResourceSpec{
				Name: service.GetFilePath(),
			},
			FilePath: absPath,
		})
	}

	return serviceSpecs, nil
}

func (d *Dashboard) updateResources(lrs resources.LocalResourcesState) {
	d.resourcesLock.Lock()
	defer d.resourcesLock.Unlock()

	d.buckets = []*BucketSpec{}
	d.topics = []*TopicSpec{}
	d.policies = map[string]PolicySpec{}

	for bucketName, resource := range lrs.Buckets.GetAll() {
		exists := lo.ContainsBy(d.buckets, func(item *BucketSpec) bool {
			return item.Name == bucketName
		})

		if !exists {
			d.buckets = append(d.buckets, &BucketSpec{
				BaseResourceSpec: &BaseResourceSpec{
					Name:               bucketName,
					RequestingServices: resource.RequestingServices,
				},
			})
		}
	}

	for topicName, resource := range lrs.Topics.GetAll() {
		exists := lo.ContainsBy(d.topics, func(item *TopicSpec) bool {
			return item.Name == topicName
		})

		if !exists {
			d.topics = append(d.topics, &TopicSpec{
				BaseResourceSpec: &BaseResourceSpec{
					Name:               topicName,
					RequestingServices: resource.RequestingServices,
				},
			})
		}
	}

	for policyName, policy := range lrs.Policies.GetAll() {
		d.policies[policyName] = PolicySpec{
			BaseResourceSpec: &BaseResourceSpec{
				Name:               policyName,
				RequestingServices: policy.RequestingServices,
			},
			Actions: lo.Map(policy.Resource.Actions, func(item resourcespb.Action, index int) string {
				return item.String()
			}),
			Resources: lo.Map(policy.Resource.Resources, func(item *resourcespb.ResourceIdentifier, index int) PolicyResource {
				return PolicyResource{
					Name: item.Name,
					Type: strings.ToLower(item.Type.String()),
				}
			}),
			Principals: lo.Map(policy.Resource.Principals, func(item *resourcespb.ResourceIdentifier, index int) PolicyResource {
				return PolicyResource{
					Name: item.Name,
					Type: strings.ToLower(item.Type.String()),
				}
			}),
		}
	}

	d.refresh()
}

func (d *Dashboard) updateApis(state apis.State) {
	d.resourcesLock.Lock()
	defer d.resourcesLock.Unlock()

	apiSpecs := []ApiSpec{}

	for apiName, rr := range state {
		resources := lo.Keys(rr)
		apiSpec := ApiSpec{
			BaseResourceSpec: &BaseResourceSpec{
				Name:               apiName,
				RequestingServices: resources,
			},
		}

		spec, _ := collector.ApiToOpenApiSpec(rr, &collector.ProjectErrors{})

		if spec != nil {
			// set title to api name
			spec.Info.Title = apiName

			apiSpec.OpenApiSpec = spec
		}

		apiSpecs = append(apiSpecs, apiSpec)
	}

	d.apis = apiSpecs

	d.refresh()
}

func (d *Dashboard) updateWebsockets(state websockets.State) {
	d.resourcesLock.Lock()
	defer d.resourcesLock.Unlock()

	wsSpec := []WebsocketSpec{}

	for name, ws := range state {
		spec := WebsocketSpec{
			BaseResourceSpec: &BaseResourceSpec{
				Name:               name,
				RequestingServices: lo.Uniq(lo.Keys(ws)),
			},
			Targets: map[string]string{},
		}

		for target, serviceWs := range ws {
			for _, eventType := range serviceWs {
				switch eventType {
				case websocketspb.WebsocketEventType_Connect:
					spec.Targets["connect"] = target
					spec.Events = append(spec.Events, "connect")
				case websocketspb.WebsocketEventType_Disconnect:
					spec.Targets["disconnect"] = target
					spec.Events = append(spec.Events, "disconnect")
				case websocketspb.WebsocketEventType_Message:
					spec.Targets["message"] = target
					spec.Events = append(spec.Events, "message")
				}
			}
		}

		wsSpec = append(wsSpec, spec)
	}

	d.websockets = wsSpec

	d.refresh()
}

func (d *Dashboard) updateTopicSubscriptions(state topics.State) {
	d.resourcesLock.Lock()
	defer d.resourcesLock.Unlock()

	d.subscriptions = []*SubscriberSpec{}

	for topicName, functions := range state {
		for functionName := range functions {
			d.subscriptions = append(d.subscriptions, &SubscriberSpec{
				Topic:  topicName,
				Target: functionName,
			})
		}
	}

	d.refresh()
}

func (d *Dashboard) updateSchedules(state schedules.State) {
	d.resourcesLock.Lock()
	defer d.resourcesLock.Unlock()

	schedules := []ScheduleSpec{}

	for _, srvc := range state {
		requestingServices := []string{srvc.ServiceName}

		schedules = append(schedules, ScheduleSpec{
			BaseResourceSpec: &BaseResourceSpec{
				Name:               srvc.Schedule.GetScheduleName(),
				RequestingServices: requestingServices,
			},
			Expression: srvc.Schedule.GetCron().GetExpression(),
			Rate:       srvc.Schedule.GetEvery().GetRate(),
			Target:     srvc.ServiceName,
		})
	}

	d.schedules = schedules

	d.refresh()
}

func (d *Dashboard) updateBucketNotifications(state storage.State) {
	d.resourcesLock.Lock()
	defer d.resourcesLock.Unlock()

	d.notifications = []*NotifierSpec{}

	for bucketName, functions := range state {
		for functionName, count := range functions {
			if count > 0 {
				d.notifications = append(d.notifications, &NotifierSpec{
					Bucket: bucketName,
					Target: functionName,
				})
			}
		}
	}

	d.refresh()
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
	notificationsRegistered := len(d.notifications) > 0

	return apisRegistered || websocketsRegistered || topicsRegistered || schedulesRegistered || notificationsRegistered
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

	services, err := d.getServices()
	if err != nil {
		return err
	}

	response := &DashboardResponse{
		Apis:               d.apis,
		Topics:             d.topics,
		Buckets:            d.buckets,
		Schedules:          d.schedules,
		Websockets:         d.websockets,
		Policies:           d.policies,
		Services:           services,
		Subscriptions:      d.subscriptions,
		Notifications:      d.notifications,
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

func New(noBrowser bool, localCloud *cloud.LocalCloud, project *project.Project) (*Dashboard, error) {
	stackWebSocket := melody.New()
	historyWebSocket := melody.New()
	wsWebSocket := melody.New()

	dash := &Dashboard{
		project:          project,
		storageService:   localCloud.Storage,
		gatewayService:   localCloud.Gateway,
		apis:             []ApiSpec{},
		envMap:           map[string]string{},
		stackWebSocket:   stackWebSocket,
		historyWebSocket: historyWebSocket,
		wsWebSocket:      wsWebSocket,
		buckets:          []*BucketSpec{},
		schedules:        []ScheduleSpec{},
		topics:           []*TopicSpec{},
		subscriptions:    []*SubscriberSpec{},
		notifications:    []*NotifierSpec{},
		websockets:       []WebsocketSpec{},
		policies:         map[string]PolicySpec{},
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

	// subscribe to resource, used for visualizer and buckets
	localCloud.Resources.SubscribeToState(dash.updateResources)

	localCloud.Apis.SubscribeToState(dash.updateApis)
	localCloud.Websockets.SubscribeToState(dash.updateWebsockets)
	localCloud.Schedules.SubscribeToState(dash.updateSchedules)
	localCloud.Topics.SubscribeToState(dash.updateTopicSubscriptions)
	localCloud.Storage.SubscribeToState(dash.updateBucketNotifications)

	// subscribe to history events from gateway
	localCloud.Apis.SubscribeToAction(dash.handleApiHistory)
	localCloud.Topics.SubscribeToAction(dash.handleTopicsHistory)
	localCloud.Schedules.SubscribeToAction(dash.handleSchedulesHistory)
	localCloud.Websockets.SubscribeToAction(dash.handleWebsocketEvents)

	return dash, nil
}
