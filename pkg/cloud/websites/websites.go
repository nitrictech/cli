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
	"errors"
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
	*WebsitePb

	Name            string
	Directory       string
	OutputDirectory string
	DevURL          string
	URL             string
}

type (
	WebsiteName   = string
	State         = map[WebsiteName]Website
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

	// Emulates the CDN URL used in a deployed environment
	publicUrl := fmt.Sprintf("http://localhost:%d/%s", l.port, strings.TrimPrefix(website.BasePath, "/"))

	l.state[website.Name] = Website{
		WebsitePb: website.WebsitePb,
		Name:      website.Name,
		DevURL:    website.DevURL,
		Directory: website.Directory,
		URL:       publicUrl,
	}

	l.publishState()
}

type staticSiteHandler struct {
	website    *Website
	port       int
	devURL     string
	isStartCmd bool
}

func (h staticSiteHandler) serveProxy(res http.ResponseWriter, req *http.Request) {
	if h.devURL == "" {
		http.Error(res, "The dev URL is not set for this website", http.StatusInternalServerError)
		return
	}

	targetUrl, err := url.Parse(h.devURL)
	if err != nil {
		http.Error(res, fmt.Sprintf("Invalid dev URL '%s': %v", h.devURL, err), http.StatusInternalServerError)
		return
	}

	// ignore proxy errors like unsupported protocol
	if targetUrl == nil || targetUrl.Scheme == "" {
		return
	}

	// Reverse proxy request
	proxy := httputil.NewSingleHostReverseProxy(targetUrl)

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		if err != nil {
			var opErr *net.OpError

			if errors.As(err, &opErr) && opErr.Op == "dial" {
				http.Error(w, "Connection to the dev server was refused. Check the URL and server status.", http.StatusServiceUnavailable)
			} else {
				http.Error(w, err.Error(), http.StatusBadGateway)
			}
		}
	}
	proxy.ServeHTTP(res, req)
}

func (h staticSiteHandler) serveStatic(res http.ResponseWriter, req *http.Request) {
	path := filepath.Join(h.website.OutputDirectory, req.URL.Path)

	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// if the file doesn't exist, serve the error page with a 404 status code
			http.ServeFile(res, req, filepath.Join(h.website.OutputDirectory, h.website.ErrorDocument))

			return
		}

		http.Error(res, err.Error(), http.StatusInternalServerError)

		return
	}

	if fi.IsDir() {
		http.ServeFile(res, req, filepath.Join(h.website.OutputDirectory, h.website.IndexDocument))

		return
	}

	http.FileServer(http.Dir(h.website.OutputDirectory)).ServeHTTP(res, req)
}

// ServeHTTP - Serve a static website from the local filesystem
func (h staticSiteHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	// If the website is running (i.e. start mode), proxy the request to the dev server
	if h.isStartCmd {
		h.serveProxy(res, req)
		return
	}

	h.serveStatic(res, req)
}

// Start - Start the local website service
func (l *LocalWebsiteService) Start(websites []Website) error {
	newLis, err := netx.GetNextListener(netx.MinPort(5000))
	if err != nil {
		return err
	}

	l.port = newLis.Addr().(*net.TCPAddr).Port

	_ = newLis.Close()

	mux := http.NewServeMux()

	// Register the API proxy handler
	mux.HandleFunc("/api/{name}/", func(res http.ResponseWriter, req *http.Request) {
		apiName := req.PathValue("name")

		apiAddress := l.getApiAddress(apiName)
		if apiAddress == "" {
			http.Error(res, fmt.Sprintf("api %s not found", apiName), http.StatusNotFound)
			return
		}

		targetPath := strings.TrimPrefix(req.URL.Path, fmt.Sprintf("/api/%s", apiName))
		targetUrl, _ := url.Parse(apiAddress)

		proxy := httputil.NewSingleHostReverseProxy(targetUrl)
		req.URL.Path = targetPath

		proxy.ServeHTTP(res, req)
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
