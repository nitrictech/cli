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
	"sync"

	"github.com/asaskevich/EventBus"
	"github.com/nitrictech/cli/pkg/netx"
	"github.com/nitrictech/nitric/core/pkg/logger"
)

type (
	WebsiteName = string
	State       = map[WebsiteName]string
)

type LocalWebsiteService struct {
	websiteRegLock sync.RWMutex
	state          State

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
func (l *LocalWebsiteService) register(websiteName string, port int) {
	if _, exists := l.state[websiteName]; exists {
		logger.Warnf("Website %s is already registered", websiteName)
		return
	}

	l.websiteRegLock.Lock()
	defer l.websiteRegLock.Unlock()

	l.state[websiteName] = fmt.Sprintf("http://localhost:%d", port)

	l.publishState()
}

// deregister - Deregister a website
func (l *LocalWebsiteService) deregister(websiteName string) {
	l.websiteRegLock.Lock()
	defer l.websiteRegLock.Unlock()

	delete(l.state, websiteName)

	l.publishState()
}

// Serve - Serve a website from the local filesystem
func (l *LocalWebsiteService) Serve(websiteName string, path string) error {
	// serve the website from path using http server
	fs := http.FileServer(http.Dir(path))

	// Create a new ServeMux to handle the request
	mux := http.NewServeMux()
	mux.Handle("/", fs)

	// get an available port
	ports, err := netx.TakePort(1)
	if err != nil {
		return err
	}

	port := ports[0] // Take the first available port

	// Start the HTTP server on the assigned port
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		l.deregister(websiteName)
		return fmt.Errorf("failed to start server on port %d: %w", port, err)
	}

	go func() {
		if err := http.Serve(listener, mux); err != nil {
			logger.Errorf("Error serving website %s: %s", websiteName, err.Error())
			l.deregister(websiteName)
		}
	}()

	l.register(websiteName, port)

	return nil
}

func NewLocalWebsitesService() *LocalWebsiteService {
	return &LocalWebsiteService{
		state: State{},
		bus:   EventBus.New(),
	}
}
