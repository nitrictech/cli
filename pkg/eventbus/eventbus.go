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

package eventbus

import "github.com/asaskevich/EventBus"

var bus EventBus.Bus

func Bus() EventBus.Bus {
	if bus == nil {
		bus = EventBus.New()
	}

	return bus
}

var topicBus EventBus.Bus

func TopicBus() EventBus.Bus {
	if topicBus == nil {
		topicBus = EventBus.New()
	}

	return topicBus
}

var storageBus EventBus.Bus

func StorageBus() EventBus.Bus {
	if storageBus == nil {
		storageBus = EventBus.New()
	}

	return storageBus
}
