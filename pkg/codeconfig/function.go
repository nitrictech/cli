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

package codeconfig

import (
	"fmt"
	"strings"
	"sync"

	"github.com/nitrictech/cli/pkg/utils"
	v1 "github.com/nitrictech/nitric/core/pkg/api/nitric/v1"
)

type Api struct {
	securityDefinitions map[string]*v1.ApiSecurityDefinition
	security            map[string][]string
	workers             []*v1.ApiWorker
	lock                sync.RWMutex
}

func (a *Api) String() string {
	return fmt.Sprintf("workers: %+v", a.workers)
}

func newApi() *Api {
	return &Api{
		workers:             make([]*v1.ApiWorker, 0),
		securityDefinitions: make(map[string]*v1.ApiSecurityDefinition),
		security:            make(map[string][]string),
	}
}

func normalizePath(path string) string {
	parts := utils.SplitPath(path)
	for i, part := range parts {
		if strings.HasPrefix(part, ":") {
			parts[i] = ":param"

			continue
		}

		parts[i] = strings.ToLower(part)
	}

	return strings.Join(parts, "/")
}

func matchingWorkers(a *v1.ApiWorker, b *v1.ApiWorker) bool {
	if normalizePath(a.GetPath()) == normalizePath(b.GetPath()) {
		for _, aMethod := range a.GetMethods() {
			for _, bMethod := range b.GetMethods() {
				if aMethod == bMethod {
					return true
				}
			}
		}
	}

	return false
}

func (a *Api) AddWorker(worker *v1.ApiWorker) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	// Ensure the worker is unique
	for _, existing := range a.workers {
		if matchingWorkers(existing, worker) {
			return fmt.Errorf("overlapping worker %v already registered, can't add new worker %v", existing, worker)
		}
	}

	a.workers = append(a.workers, worker)

	return nil
}

func (a *Api) AddSecurityDefinition(name string, sd *v1.ApiSecurityDefinition) {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.securityDefinitions[name] = sd
}

func (a *Api) AddSecurity(name string, scopes []string) {
	a.lock.Lock()
	defer a.lock.Unlock()

	if scopes != nil {
		a.security[name] = scopes
	} else {
		// default to empty scopes for a nil assignment
		a.security[name] = []string{}
	}
}

// FunctionDependencies - Stores information about a Nitric Function, and it's dependencies
type FunctionDependencies struct {
	name          string
	apis          map[string]*Api
	subscriptions map[string]*v1.SubscriptionWorker
	schedules     map[string]*v1.ScheduleWorker
	buckets       map[string]*v1.BucketResource
	topics        map[string]*v1.TopicResource
	collections   map[string]*v1.CollectionResource
	queues        map[string]*v1.QueueResource
	policies      []*v1.PolicyResource
	secrets       map[string]*v1.SecretResource
	lock          sync.RWMutex
}

// AddPolicy - Adds an access policy dependency to the function
func (a *FunctionDependencies) AddPolicy(p *v1.PolicyResource) {
	a.lock.Lock()
	defer a.lock.Unlock()

	for _, p := range p.Principals {
		// If provided a blank function principal assume its for this function
		if p.Type == v1.ResourceType_Function && p.Name == "" {
			p.Name = a.name
		}
	}

	a.policies = append(a.policies, p)
}

func (a *FunctionDependencies) AddApiSecurityDefinitions(name string, sds map[string]*v1.ApiSecurityDefinition) {
	a.lock.Lock()
	defer a.lock.Unlock()

	if a.apis[name] == nil {
		a.apis[name] = newApi()
	}

	for n, sd := range sds {
		a.apis[name].AddSecurityDefinition(n, sd)
	}
}

func (a *FunctionDependencies) AddApiSecurity(name string, security map[string]*v1.ApiScopes) {
	a.lock.Lock()
	defer a.lock.Unlock()

	if a.apis[name] == nil {
		a.apis[name] = newApi()
	}

	for n, scopes := range security {
		a.apis[name].AddSecurity(n, scopes.Scopes)
	}
}

func (a *FunctionDependencies) AddApiHandler(aw *v1.ApiWorker) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	if a.apis[aw.Api] == nil {
		a.apis[aw.Api] = newApi()
	}

	return a.apis[aw.Api].AddWorker(aw)
}

// AddSubscriptionHandler - registers a handler in the function that subscribes to a topic of events
func (a *FunctionDependencies) AddSubscriptionHandler(sw *v1.SubscriptionWorker) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	// TODO: Determine if this subscription handler has a write policy to the same topic
	if a.subscriptions[sw.Topic] != nil {
		// return a new error
		return fmt.Errorf("subscription already declared for topic %s, only one subscription per topic is allowed per application", sw.Topic)
	}

	// This maps to a trigger worker for this application
	a.subscriptions[sw.Topic] = sw

	return nil
}

func (a *FunctionDependencies) WorkerCount() int {
	workerCount := 0

	for _, v := range a.apis {
		workerCount = workerCount + len(v.workers)
	}

	workerCount = workerCount + len(a.subscriptions) + len(a.schedules)

	return workerCount
}

// AddScheduleHandler - registers a handler in the function that runs on a schedule
func (a *FunctionDependencies) AddScheduleHandler(sw *v1.ScheduleWorker) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	if a.schedules[sw.Key] != nil {
		return fmt.Errorf("schedule %s already exists", sw.Key)
	}

	a.schedules[sw.GetKey()] = sw

	return nil
}

// AddBucket - adds a storage bucket dependency to the function
func (a *FunctionDependencies) AddBucket(name string, b *v1.BucketResource) {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.buckets[name] = b
}

// AddTopic - adds a pub/sub topic dependency to the function
func (a *FunctionDependencies) AddTopic(name string, t *v1.TopicResource) {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.topics[name] = t
}

// AddCollection - adds a document database collection dependency to the function
func (a *FunctionDependencies) AddCollection(name string, c *v1.CollectionResource) {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.collections[name] = c
}

// AddQueue - adds a queue dependency to the function
func (a *FunctionDependencies) AddQueue(name string, q *v1.QueueResource) {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.queues[name] = q
}

func (a *FunctionDependencies) AddSecret(name string, s *v1.SecretResource) {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.secrets[name] = s
}

// NewFunction - creates a new Nitric Function, ready to register handlers and dependencies.
func NewFunction(name string) *FunctionDependencies {
	return &FunctionDependencies{
		name:          name,
		apis:          make(map[string]*Api),
		subscriptions: make(map[string]*v1.SubscriptionWorker),
		schedules:     make(map[string]*v1.ScheduleWorker),
		buckets:       make(map[string]*v1.BucketResource),
		topics:        make(map[string]*v1.TopicResource),
		collections:   make(map[string]*v1.CollectionResource),
		queues:        make(map[string]*v1.QueueResource),
		secrets:       make(map[string]*v1.SecretResource),
		policies:      make([]*v1.PolicyResource, 0),
	}
}
