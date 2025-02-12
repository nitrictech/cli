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

package websites

import (
	"fmt"
	"maps"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/asaskevich/EventBus"
	"github.com/nitrictech/cli/pkg/netx"
	deploymentspb "github.com/nitrictech/nitric/core/pkg/proto/deployments/v1"
)

type WebsitePb = deploymentspb.Website

type Website struct {
	WebsitePb

	Name   string
	DevURL string
}

type (
	WebsiteName   = string
	State         = map[WebsiteName]string
	GetApiAddress = func(apiName string) string
)

type LocalWebsiteService struct {
	websiteRegLock sync.RWMutex
	state          State
	port           int
	getApiAddress  GetApiAddress
	isStartCmd     bool

	bus EventBus.Bus
}

const localWebsitesTopic = "local_websites"

func (l *LocalWebsiteService) publishState() {
	l.bus.Publish(localWebsitesTopic, maps.Clone(l.state))
}

func (l *LocalWebsiteService) SubscribeToState(fn func(State)) {
	// ignore the error, it's only returned if the fn param isn't a function
	_ = l.bus.Subscribe(localWebsitesTopic, fn)
}

// register - Register a new website
func (l *LocalWebsiteService) register(website Website) {
	l.websiteRegLock.Lock()
	defer l.websiteRegLock.Unlock()

	l.state[website.Name] = fmt.Sprintf("http://localhost:%d/%s", l.port, strings.TrimPrefix(website.BasePath, "/"))

	l.publishState()
}

// deregister - Deregister a website
func (l *LocalWebsiteService) deregister(websiteName string) {
	l.websiteRegLock.Lock()
	defer l.websiteRegLock.Unlock()

	delete(l.state, websiteName)

	l.publishState()
}

type staticSiteHandler struct {
	website    *Website
	port       int
	devURL     string
	isStartCmd bool
}

// ServeHTTP - Serve a static website from the local filesystem
func (h staticSiteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// if start command just proxy the request to the dev url
	if h.isStartCmd {
		// Target backend API server
		target, err := url.Parse(h.devURL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// ignore proxy errors like unsupported protocol
		if target == nil || target.Scheme == "" {
			return
		}

		// Reverse proxy request
		proxy := httputil.NewSingleHostReverseProxy(target)
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
			}
		}
		proxy.ServeHTTP(w, r)

		return
	}

	path := filepath.Join(h.website.OutputDirectory, r.URL.Path)

	// check whether a file exists or is a directory at the given path
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// if the file doesn't exist, serve the error page with a 404 status code
			http.ServeFile(w, r, filepath.Join(h.website.OutputDirectory, h.website.ErrorDocument))
			return
		}

		// if we got an error (that wasn't that the file doesn't exist) stating the
		// file, return a 500 internal server error and stop
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if fi.IsDir() {
		http.ServeFile(w, r, filepath.Join(h.website.OutputDirectory, h.website.IndexDocument))
		return
	}

	// otherwise, use http.FileServer to serve the static file
	http.FileServer(http.Dir(h.website.OutputDirectory)).ServeHTTP(w, r)
}

// Serve - Serve a website from the local filesystem
func (l *LocalWebsiteService) Start(websites []Website) error {
	newLis, err := netx.GetNextListener(netx.MinPort(5000))
	if err != nil {
		return err
	}

	l.port = newLis.Addr().(*net.TCPAddr).Port

	_ = newLis.Close()

	// Initialize the multiplexer only if websites will be served
	mux := http.NewServeMux()

	// Register the API handler
	mux.HandleFunc("/api/{name}/", func(w http.ResponseWriter, r *http.Request) {
		// get the api name from the request path
		apiName := r.PathValue("name")

		// get the address of the api
		apiAddress := l.getApiAddress(apiName)
		if apiAddress == "" {
			http.Error(w, fmt.Sprintf("api %s not found", apiName), http.StatusNotFound)
			return
		}

		// Strip /api/{name}/ from the URL path
		newPath := strings.TrimPrefix(r.URL.Path, fmt.Sprintf("/api/%s", apiName))

		// Target backend API server
		target, _ := url.Parse(apiAddress)

		// Reverse proxy request
		proxy := httputil.NewSingleHostReverseProxy(target)
		r.URL.Path = newPath

		// Forward the modified request to the backend
		proxy.ServeHTTP(w, r)
	})

	// Register the SPA handler for each website
	for i := range websites {
		website := &websites[i]
		spa := staticSiteHandler{website: website, port: l.port, devURL: website.DevURL, isStartCmd: l.isStartCmd}

		if website.BasePath == "/" {
			mux.Handle("/", spa)
		} else {
			mux.Handle(website.BasePath+"/", http.StripPrefix(website.BasePath+"/", spa))
		}
	}

	// Start the server with the multiplexer
	go func() {
		addr := fmt.Sprintf(":%d", l.port)
		if err := http.ListenAndServe(addr, mux); err != nil {
			fmt.Printf("Failed to start server: %s\n", err)
		}
	}()

	// Register the websites
	for _, website := range websites {
		l.register(website)
	}

	return nil
}

func NewLocalWebsitesService(getApiAddress GetApiAddress, isStartCmd bool) *LocalWebsiteService {
	return &LocalWebsiteService{
		state:         State{},
		bus:           EventBus.New(),
		getApiAddress: getApiAddress,
		isStartCmd:    isStartCmd,
	}
}
