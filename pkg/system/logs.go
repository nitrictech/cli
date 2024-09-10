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

package system

import (
	"sync"

	"github.com/asaskevich/EventBus"
)

// SystemLogsService - An EventBus service for handling logging of events and exceptions
// So they can be subscribed to and displayed in the CLI
type SystemLogsService struct {
	bus EventBus.Bus
}

const logTopic = "system_logs"

var (
	instance *SystemLogsService
	once     sync.Once
)

func getInstance() *SystemLogsService {
	once.Do(func() {
		instance = &SystemLogsService{
			bus: EventBus.New(),
		}
	})

	return instance
}

func Log(msg string) {
	s := getInstance()
	s.bus.Publish(logTopic, msg)
}

func SubscribeToLogs(subscription func(string)) {
	s := getInstance()
	_ = s.bus.Subscribe(logTopic, subscription)
}

// func SubscribeToLogsAsRunUpdate() chan project.ServiceRunUpdate {
// 	s := getInstance()

// 	runUpdateChan := make(chan project.ServiceRunUpdate)

// 	_ = s.bus.Subscribe(logTopic, func(msg string) {
// 		runUpdateChan <- project.ServiceRunUpdate{
// 			ServiceName: "nitric",
// 			Label:       "nitric",
// 			Status:      project.ServiceRunStatus_Running,
// 			Message:     msg,
// 		}
// 	})

// 	return runUpdateChan
// }
