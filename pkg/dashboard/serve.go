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
	"io/fs"
	"log"
	"net"
	"net/http"

	"github.com/nitrictech/cli/pkg/utils"
)

//go:embed dist/*
var content embed.FS

func Serve() (*int, error) {
	// Get the embedded files from the 'dist' directory
	staticFiles, err := fs.Sub(content, "dist")
	if err != nil {
		return nil, err
	}

	// Serve the files using the http package
	http.Handle("/", http.FileServer(http.FS(staticFiles)))

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