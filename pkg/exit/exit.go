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

package exit

import (
	"sync"

	"github.com/asaskevich/EventBus"
)

// ExitService - Service for handling application exit events
type ExitService struct {
	bus EventBus.Bus
}

const exitTopic = "local_exit_cli"

var (
	instance *ExitService
	once     sync.Once
)

func GetExitService() *ExitService {
	once.Do(func() {
		instance = &ExitService{
			bus: EventBus.New(),
		}
	})

	return instance
}

func (s *ExitService) Exit(err error) {
	s.bus.Publish(exitTopic, err)
}

func (s *ExitService) SubscribeToExit(subscription func(err error)) {
	_ = s.bus.SubscribeOnce(exitTopic, subscription)
}
