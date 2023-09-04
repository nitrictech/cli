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
	"os/exec"
	"runtime"
	"strconv"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/olahol/melody"

	"github.com/nitrictech/cli/pkg/codeconfig"
	"github.com/nitrictech/cli/pkg/project"
	"github.com/nitrictech/cli/pkg/utils"
	"github.com/nitrictech/nitric/core/pkg/plugins/storage"
	"github.com/nitrictech/nitric/core/pkg/worker/pool"
)

type WebsocketMessage struct {
	Data         string    `json:"data,omitempty"`
	Time         time.Time `json:"time,omitempty"`
	ConnectionID string    `json:"connectionId,omitempty"`
}
type WebsocketInfo struct {
	ConnectionCount int                `json:"connectionCount,omitempty"`
	Messages        []WebsocketMessage `json:"messages,omitempty"`
}

type Dashboard struct {
	project              *project.Project
	apis                 []*openapi3.T
	schedules            []*codeconfig.TopicResult
	topics               []*codeconfig.TopicResult
	buckets              []string
	websockets           []*codeconfig.WebsocketResult
	envMap               map[string]string
	stackWebSocket       *melody.Melody
	historyWebSocket     *melody.Melody
	wsWebSocket          *melody.Melody
	triggerAddress       string
	storageAddress       string
	apiAddresses         map[string]string
	websocketAddresses   map[string]string
	websocketsInfo       map[string]*WebsocketInfo
	resourcesLastUpdated time.Time
	bucketNotifications  []*codeconfig.BucketNotification
	noBrowser            bool
}

type Api struct {
	Name    string                 `json:"name,omitempty"`
	OpenApi map[string]interface{} `json:"spec,omitempty"` // not sure which spec version yet
}

type DashboardResponse struct {
	Apis                []*openapi3.T                    `json:"apis,omitempty"`
	Buckets             []string                         `json:"buckets,omitempty"`
	Schedules           []*codeconfig.TopicResult        `json:"schedules,omitempty"`
	Topics              []*codeconfig.TopicResult        `json:"topics,omitempty"`
	Websockets          []*codeconfig.WebsocketResult    `json:"websockets,omitempty"`
	ProjectName         string                           `json:"projectName,omitempty"`
	ApiAddresses        map[string]string                `json:"apiAddresses,omitempty"`
	WebsocketAddresses  map[string]string                `json:"websocketAddresses,omitempty"`
	TriggerAddress      string                           `json:"triggerAddress,omitempty"`
	StorageAddress      string                           `json:"storageAddress,omitempty"`
	BucketNotifications []*codeconfig.BucketNotification `json:"bucketNotifications,omitempty"`
}

type Bucket struct {
	Name         string     `json:"name,omitempty"`
	CreationDate *time.Time `json:"creationDate,omitempty"`
}

type RefreshOptions struct {
	Pool               pool.WorkerPool
	TriggerAddress     string
	StorageAddress     string
	ApiAddresses       map[string]string
	WebSocketAddresses map[string]string
	ServiceListener    net.Listener
}

//go:embed dist/*
var content embed.FS

func New(p *project.Project, envMap map[string]string, noBrowser bool) (*Dashboard, error) {
	stackWebSocket := melody.New()

	historyWebSocket := melody.New()

	wsWebSocket := melody.New()

	return &Dashboard{
		project:             p,
		apis:                []*openapi3.T{},
		envMap:              envMap,
		stackWebSocket:      stackWebSocket,
		historyWebSocket:    historyWebSocket,
		wsWebSocket:         wsWebSocket,
		bucketNotifications: []*codeconfig.BucketNotification{},
		schedules:           []*codeconfig.TopicResult{},
		topics:              []*codeconfig.TopicResult{},
		websocketsInfo:      map[string]*WebsocketInfo{},
		noBrowser:           noBrowser,
	}, nil
}

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
}

func (d *Dashboard) Refresh(opts *RefreshOptions) error {
	cc, err := codeconfig.New(d.project, d.envMap)
	if err != nil {
		return err
	}

	spec, err := cc.SpecFromWorkerPool(opts.Pool)
	if err != nil {
		return err
	}

	d.apis = spec.Apis
	d.schedules = spec.Schedules
	d.topics = spec.Topics
	d.bucketNotifications = spec.BucketNotifications
	d.websockets = spec.WebSockets

	d.triggerAddress = opts.TriggerAddress
	d.apiAddresses = opts.ApiAddresses
	d.websocketAddresses = opts.WebSocketAddresses
	d.storageAddress = opts.StorageAddress

	err = d.sendStackUpdate()
	if err != nil {
		return err
	}

	err = d.sendHistoryUpdate()
	if err != nil {
		return err
	}

	return nil
}

func (d *Dashboard) RefreshHistory() error {
	return d.sendHistoryUpdate()
}

func (d *Dashboard) UpdateWebsocketInfoCount(socket string, count int) error {
	if d.websocketsInfo[socket] == nil {
		d.websocketsInfo[socket] = &WebsocketInfo{}
	}

	d.websocketsInfo[socket].ConnectionCount = count

	err := d.sendWebsocketsUpdate()
	if err != nil {
		return err
	}

	return nil
}

func (d *Dashboard) AddWebsocketInfoMessage(socket string, message WebsocketMessage) error {
	d.websocketsInfo[socket].Messages = append([]WebsocketMessage{message}, d.websocketsInfo[socket].Messages...)

	err := d.sendWebsocketsUpdate()
	if err != nil {
		return err
	}

	return nil
}

func (d *Dashboard) Serve(storagePlugin storage.StorageService) (*int, error) {
	// Get the embedded files from the 'dist' directory
	staticFiles, err := fs.Sub(content, "dist")
	if err != nil {
		return nil, err
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

	http.HandleFunc("/api/history", d.handleHistory())

	// Define an API route under /call to proxy communication between app and apis
	http.HandleFunc("/api/call/", d.handleCallProxy())

	http.HandleFunc("/api/storage", d.handleStorage(storagePlugin))

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
		return nil, err
	}

	serveFn := func() {
		err = http.Serve(dashListener, nil)
		if err != nil {
			log.Fatal(err)
		}
	}
	go serveFn()

	port := dashListener.Addr().(*net.TCPAddr).Port

	// open browser
	if !d.noBrowser {
		err = openBrowser(fmt.Sprintf("http://localhost:%s", strconv.Itoa(port)))
		if err != nil {
			return nil, err
		}
	}

	return &port, nil
}

func handleResponseWriter(w http.ResponseWriter, data []byte) {
	_, err := w.Write(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (d *Dashboard) sendStackUpdate() error {
	response := &DashboardResponse{
		Apis:                d.apis,
		Topics:              d.topics,
		Buckets:             d.buckets,
		Schedules:           d.schedules,
		Websockets:          d.websockets,
		ProjectName:         d.project.Name,
		ApiAddresses:        d.apiAddresses,
		WebsocketAddresses:  d.websocketAddresses,
		TriggerAddress:      d.triggerAddress,
		StorageAddress:      d.storageAddress,
		BucketNotifications: d.bucketNotifications,
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
	// Define an API route under /call to proxy communication between app and apis
	response, err := d.project.History.ReadAllHistoryRecords()
	if err != nil {
		return err
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		return err
	}

	err = d.historyWebSocket.Broadcast(jsonData)

	return err
}

func (d *Dashboard) sendWebsocketsUpdate() error {
	jsonData, err := json.Marshal(d.websocketsInfo)
	if err != nil {
		return err
	}

	err = d.wsWebSocket.Broadcast(jsonData)

	return err
}

func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform")
	}

	err := cmd.Start()
	if err != nil {
		return err
	}

	return nil
}
