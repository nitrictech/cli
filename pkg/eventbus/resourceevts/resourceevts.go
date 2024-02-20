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

package resourceevts

import (
	"net"

	"github.com/asaskevich/EventBus"

	"github.com/nitrictech/nitric/core/pkg/gateway"
)

type LocalInfrastructureState struct {
	GatewayStartOpts   *gateway.GatewayStartOpts
	TriggerAddress     string
	StorageAddress     string
	ApiAddresses       map[string]string
	WebSocketAddresses map[string]string
	ServiceListener    net.Listener
}

const localInfrastructureTopic = "local_infrastructure"

var localInfrastructureStateBus EventBus.Bus

func localInfrastructure() EventBus.Bus {
	if localInfrastructureStateBus == nil {
		localInfrastructureStateBus = EventBus.New()
	}

	return localInfrastructureStateBus
}

func Subscribe(fn func(opts *LocalInfrastructureState)) error {
	return localInfrastructure().Subscribe(localInfrastructureTopic, fn)
}

func SubscribeAsync(fn func(opts *LocalInfrastructureState)) error {
	return localInfrastructure().SubscribeAsync(localInfrastructureTopic, fn, false)
}

func SubscribeOnce(fn func(opts *LocalInfrastructureState)) error {
	return localInfrastructure().SubscribeOnce(localInfrastructureTopic, fn)
}

func NewSubChannel() chan LocalInfrastructureState {
	ch := make(chan LocalInfrastructureState)
	_ = SubscribeAsync(func(opts *LocalInfrastructureState) {
		ch <- *opts
	})

	return ch
}

func Publish(evt LocalInfrastructureState) {
	localInfrastructure().Publish(localInfrastructureTopic, evt)
}
