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
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/olahol/melody"

	"github.com/nitrictech/cli/pkg/codeconfig"
	"github.com/nitrictech/cli/pkg/project"
	"github.com/nitrictech/cli/pkg/run"
	"github.com/nitrictech/cli/pkg/utils"
)

type dashboard struct {
	project        *project.Project
	apis           []*openapi3.T
	schedules      []*codeconfig.TopicResult
	envMap         map[string]string
	melody         *melody.Melody
	triggerAddress string
	apiAddresses   map[string]string
}

type Api struct {
	Name    string                 `json:"name,omitempty"`
	OpenApi map[string]interface{} `json:"spec,omitempty"` // not sure which spec version yet
}

type DashResponse struct {
	Apis           []*openapi3.T             `json:"apis,omitempty"`
	Schedules      []*codeconfig.TopicResult `json:"schedules,omitempty"`
	ProjectName    string                    `json:"projectName,omitempty"`
	ApiAddresses   map[string]string         `json:"apiAddresses,omitempty"`
	TriggerAddress string                    `json:"triggerAddress,omitempty"`
}

//go:embed dist/*
var content embed.FS

func New(p *project.Project, envMap map[string]string) (*dashboard, error) {
	m := melody.New()

	return &dashboard{
		project: p,
		apis:    []*openapi3.T{},
		envMap:  envMap,
		melody:  m,
	}, nil
}

func (d *dashboard) Refresh(ls run.LocalServices) error {
	cc, err := codeconfig.New(d.project, d.envMap)
	if err != nil {
		return err
	}

	pool := ls.GetWorkerPool()

	spec, err := cc.SpecFromWorkerPool(pool)
	if err != nil {
		return err
	}

	d.triggerAddress = ls.TriggerAddress()
	d.apiAddresses = ls.Apis()
	d.apis = spec.Apis
	d.schedules = spec.Shedules

	err = d.sendUpdate()
	if err != nil {
		return err
	}

	return nil
}

type Message struct {
	Text string `json:"text"`
}

func (d *dashboard) Serve() (*int, error) {
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
		err := d.melody.HandleRequest(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// Register a handler for when a client connects
	d.melody.HandleConnect(func(s *melody.Session) {
		// Send a welcome message to the client
		err := d.sendUpdate()
		if err != nil {
			log.Fatal(err)
		}
	})

	d.melody.HandleMessage(func(s *melody.Session, msg []byte) {
		err := d.melody.Broadcast(msg)
		if err != nil {
			log.Print(err)
		}
	})

	// Define an API route under /call to proxy communication between app and apis
	http.HandleFunc("/call/", func(w http.ResponseWriter, r *http.Request) {
		// Set CORs headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// find call callAddress
		callAddress := r.Header.Get("X-Nitric-Local-Call-Address")

		// Remove "/call" prefix from URL path
		path := strings.TrimPrefix(r.URL.Path, "/call/")

		// Build proxy request URL with query parameters
		query := r.URL.RawQuery
		if query != "" {
			query = "?" + query
		}
		url := fmt.Sprintf("http://%s/%s%s", callAddress, path, query)

		// Create a new request object
		req, err := http.NewRequest(r.Method, url, r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Copy the headers from the original request to the new request
		for key, value := range r.Header {
			req.Header.Set(key, value[0])
		}

		// Send the new request and handle the response
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		// Copy the headers from the response to the response writer
		for key, value := range resp.Header {
			w.Header().Set(key, value[0])
		}

		// Copy the status code from the response to the response writer
		w.WriteHeader(resp.StatusCode)

		// Copy the response body to the response writer
		_, err = io.Copy(w, resp.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
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

	return &port, nil
}

func (d *dashboard) sendUpdate() error {
	// ignore if no apis
	if len(d.apis) == 0 {
		return nil
	}

	response := &DashResponse{
		Apis:           d.apis,
		Schedules:      d.schedules,
		ProjectName:    d.project.Name,
		ApiAddresses:   d.apiAddresses,
		TriggerAddress: d.triggerAddress,
	}

	// Encode the response as JSON
	jsonData, err := json.Marshal(response)
	if err != nil {
		return err
	}

	err = d.melody.Broadcast(jsonData)

	return err
}
