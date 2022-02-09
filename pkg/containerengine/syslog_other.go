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

//go:build !windows
// +build !windows

package containerengine

import (
	"fmt"
	"log"
	slog "log/syslog"
	"net"
	"os"
	"path"

	"github.com/docker/docker/api/types/container"
	"github.com/hashicorp/consul/sdk/freeport"
	"gopkg.in/mcuadros/go-syslog.v2"

	"github.com/nitrictech/cli/pkg/utils"
)

type localSyslog struct {
	logPath string
	file    *os.File
	port    int
	server  *syslog.Server
}

func newSyslog(logPath string) ContainerLogger {
	return &localSyslog{logPath: logPath}
}

func (s *localSyslog) Stop() error {
	errList := utils.NewErrorList()

	errList.Add(s.server.Kill())
	s.server.Wait()
	errList.Add(s.file.Close())

	return errList.Aggregate()
}

func (s *localSyslog) Config() *container.LogConfig {
	return &container.LogConfig{
		Type: "syslog",
		Config: map[string]string{
			"syslog-address": "udp://" + net.JoinHostPort("localhost", fmt.Sprint(s.port)),
			"tag":            "{{.ImageName}}/{{.Name}}/{{.ID}}",
		},
	}
}

func (s *localSyslog) Start() error {
	ports, err := freeport.Take(1)
	if err != nil {
		return err
	}
	s.port = ports[0]

	channel := make(syslog.LogPartsChannel)
	handler := syslog.NewChannelHandler(channel)

	s.server = syslog.NewServer()
	s.server.SetFormat(syslog.Automatic)
	s.server.SetHandler(handler)
	err = s.server.ListenUDP(net.JoinHostPort("0.0.0.0", fmt.Sprint(s.port)))
	if err != nil {
		return err
	}

	err = s.server.Boot()
	if err != nil {
		return err
	}

	err = os.MkdirAll(path.Dir(s.logPath), 0777)
	if err != nil {
		return err
	}

	s.file, err = os.OpenFile(s.logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}

	go func(channel syslog.LogPartsChannel) {
		for logParts := range channel {
			fmt.Fprintf(s.file, "%s %s %s\n", logParts["timestamp"], logParts["tag"], logParts["content"])
		}
	}(channel)

	// set up the log client
	logwriter, err := slog.Dial("udp", net.JoinHostPort("localhost", fmt.Sprint(s.port)), slog.LOG_DEBUG, "nitric-run")
	if err != nil {
		return err
	}
	log.SetOutput(logwriter)

	return nil
}
