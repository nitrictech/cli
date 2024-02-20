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

package resources

import (
	"maps"
	"slices"
	"sync"

	"github.com/samber/lo"

	"github.com/nitrictech/nitric/core/pkg/logger"
)

type ResourceRegister[R any] struct {
	RequestingServices []string
	Resource           *R
}

type ResourceRegistrar[R any] struct {
	lock      sync.RWMutex
	resources map[ResourceName]*ResourceRegister[R]
}

func (r *ResourceRegistrar[R]) isAlreadyRegistered(name string, requestingService string) bool {
	_, exists := r.resources[name]
	if exists {
		duplicate := slices.Contains(r.resources[name].RequestingServices, requestingService)
		return duplicate
	}

	return false
}

func (r *ResourceRegistrar[R]) Register(name string, requestingService string, resource *R) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	_, exists := r.resources[name]
	if exists {
		if r.isAlreadyRegistered(name, requestingService) {
			logger.Debugf("resource %s registered multiple times for service %s", name, requestingService)
			return nil
		}

		// already registered, by another service, add this service to the list
		r.resources[name].RequestingServices = append(r.resources[name].RequestingServices, requestingService)

		return nil
	}

	// new resource, register it
	r.resources[name] = &ResourceRegister[R]{
		RequestingServices: []string{requestingService},
		Resource:           resource,
	}

	return nil
}

func (r *ResourceRegistrar[R]) Get(resourceName string) *R {
	r.lock.RLock()
	defer r.lock.RUnlock()

	registration, ok := r.resources[resourceName]
	if !ok {
		return nil
	}

	return registration.Resource
}

func (r *ResourceRegistrar[R]) GetAll() map[ResourceName]*ResourceRegister[R] {
	r.lock.RLock()
	defer r.lock.RUnlock()

	return maps.Clone(r.resources)
}

func (r *ResourceRegistrar[R]) GetRequestingServices(name string) []string {
	r.lock.RLock()
	defer r.lock.RUnlock()

	registration, ok := r.resources[name]
	if !ok {
		return []string{}
	}

	return registration.RequestingServices
}

// ClearRequestingService - Remove a requesting service from all resources, if it was the only requestor for a resource, the resource is also removed
func (r *ResourceRegistrar[R]) ClearRequestingService(requestingService string) {
	r.lock.Lock()
	defer r.lock.Unlock()

	for name, registration := range r.resources {
		registration.RequestingServices = lo.Filter(registration.RequestingServices, func(item string, index int) bool {
			return item != requestingService
		})

		if len(registration.RequestingServices) == 0 {
			delete(r.resources, name)
		}
	}
}

func NewResourceRegistrar[R any]() *ResourceRegistrar[R] {
	return &ResourceRegistrar[R]{
		resources: make(map[string]*ResourceRegister[R], 0),
	}
}
