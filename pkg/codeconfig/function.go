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
	"sync"

	"github.com/nitrictech/cli/pkg/preview"
	"github.com/nitrictech/cli/pkg/project"
	apispb "github.com/nitrictech/nitric/core/pkg/proto/apis/v1"
	httppb "github.com/nitrictech/nitric/core/pkg/proto/http/v1"
	resourcespb "github.com/nitrictech/nitric/core/pkg/proto/resources/v1"
	schedulespb "github.com/nitrictech/nitric/core/pkg/proto/schedules/v1"
	storagepb "github.com/nitrictech/nitric/core/pkg/proto/storage/v1"
	topicspb "github.com/nitrictech/nitric/core/pkg/proto/topics/v1"
	websocketspb "github.com/nitrictech/nitric/core/pkg/proto/websockets/v1"
)

// FunctionDependencies - Stores information about a Nitric Function, and it's dependencies
type FunctionDependencies struct {
	name                string
	functionConfig      project.Function
	apis                map[string]*Api
	websockets          map[string]*Websocket
	subscriptions       map[string]*topicspb.RegistrationRequest
	schedules           map[string]*schedulespb.RegistrationRequest
	httpWorkers         map[string]*httppb.HttpProxyRequest
	buckets             map[string]*resourcespb.BucketResource
	topics              map[string]*resourcespb.TopicResource
	collections         map[string]*resourcespb.CollectionResource
	policies            []*resourcespb.PolicyResource
	secrets             map[string]*resourcespb.SecretResource
	bucketNotifications map[string][]*storagepb.RegistrationRequest
	errors              []string
	lock                sync.RWMutex
}

func (a *FunctionDependencies) AddError(err string) {
	a.errors = append(a.errors, fmt.Sprintf("function %s: %s", a.functionConfig.Handler, err))
}

// AddPolicy - Adds an access policy dependency to the function
func (a *FunctionDependencies) AddPolicy(p *resourcespb.PolicyResource) {
	a.lock.Lock()
	defer a.lock.Unlock()

	for _, p := range p.Principals {
		// If provided a blank function principal assume its for this function
		if p.Type == resourcespb.ResourceType_Function && p.Name == "" {
			p.Name = a.name
		}
	}

	a.policies = append(a.policies, p)
}

func (a *FunctionDependencies) AddApiSecurityDefinitions(name string, sds map[string]*resourcespb.ApiSecurityDefinitionResource) {
	a.lock.Lock()
	defer a.lock.Unlock()

	if a.apis[name] == nil {
		a.apis[name] = newApi(a)
	}

	for n, sd := range sds {
		a.apis[name].AddSecurityDefinition(n, sd)
	}
}

func (a *FunctionDependencies) AddApiSecurity(name string, security map[string]*resourcespb.ApiScopes) {
	a.lock.Lock()
	defer a.lock.Unlock()

	if a.apis[name] == nil {
		a.apis[name] = newApi(a)
	}

	for n, scopes := range security {
		a.apis[name].AddSecurity(n, scopes.Scopes)
	}
}

func (a *FunctionDependencies) AddApiHandler(aw *apispb.RegistrationRequest) {
	a.lock.Lock()
	defer a.lock.Unlock()

	if len(a.httpWorkers) > 0 {
		a.AddError("APIs cannot be defined in functions that already contain HTTP proxies")
		return
	}

	// Check that there are no APIs in this function that have the same path
	// TODO: allow two APIs to support matching paths without issue
	for _, api := range a.apis {
		for _, wkr := range api.workers {
			if matchingWorkers(aw, wkr) {
				a.AddError("APIs cannot share paths within the same function")
				return
			}
		}
	}

	if a.apis[aw.Api] == nil {
		a.apis[aw.Api] = newApi(a)
	}

	a.apis[aw.Api].AddWorker(aw)
}

func (a *FunctionDependencies) AddWebsocketHandler(ws *websocketspb.RegistrationRequest) {
	a.lock.Lock()
	defer a.lock.Unlock()

	if !a.functionConfig.Project.IsPreviewFeatureEnabled(preview.Feature_Websockets) {
		a.AddError(`websockets are currently in preview if you would like to enable them add
preview-features:
  - websockets
to your nitric.yaml file.
		`)

		return
	}

	if a.websockets[ws.GetSocketName()] == nil {
		a.websockets[ws.GetSocketName()] = newWebsocket(ws.GetSocketName(), a)
	}

	a.websockets[ws.GetSocketName()].AddWorker(ws)
}

// AddSubscriptionHandler - registers a handler in the function that subscribes to a topic of events
func (a *FunctionDependencies) AddSubscriptionHandler(sw *topicspb.RegistrationRequest) {
	a.lock.Lock()
	defer a.lock.Unlock()

	// TODO: Determine if this subscription handler has a write policy to the same topic
	if a.subscriptions[sw.TopicName] != nil {
		// return a new error
		a.AddError(fmt.Sprintf("declared multiple subscriptions for topic %s, only one subscription per topic is allowed per function", sw.TopicName))
		return
	}

	a.subscriptions[sw.TopicName] = sw
}

func (a *FunctionDependencies) WorkerCount() int {
	workerCount := 0

	for _, v := range a.websockets {
		workerCount = workerCount + v.WorkerCount()
	}

	for _, v := range a.apis {
		workerCount = workerCount + len(v.workers)
	}

	workerCount = workerCount + len(a.subscriptions) + len(a.schedules) + len(a.bucketNotifications) + len(a.httpWorkers)

	return workerCount
}

// AddScheduleHandler - registers a handler in the function that runs on a schedule
func (a *FunctionDependencies) AddScheduleHandler(sw *schedulespb.RegistrationRequest) {
	a.lock.Lock()
	defer a.lock.Unlock()

	if a.schedules[sw.ScheduleName] != nil {
		a.AddError(fmt.Sprintf("declared schedule %s multiple times", sw.ScheduleName))
		return
	}

	a.schedules[sw.ScheduleName] = sw
}

// AddHttpWorker - registers a handler in the function that listens on a port
func (a *FunctionDependencies) AddHttpWorker(hw *httppb.HttpProxyRequest) {
	a.lock.Lock()
	defer a.lock.Unlock()

	if !a.functionConfig.Project.IsPreviewFeatureEnabled(preview.Feature_Http) {
		a.AddError(`HTTP Proxies are currently in preview if you would like to enable them add
preview-features:
  - http
to your nitric.yaml file.
		`)

		return
	}

	if len(a.httpWorkers) > 0 {
		a.AddError("declared multiple http proxies, only one http proxy is allowed per function")
	}

	if len(a.apis) > 0 {
		a.AddError("declared a HTTP Proxy, but already declares an API. Function can only handle one")
		return
	}

	a.httpWorkers[hw.GetHost()] = hw
}

// AddBucketNotificationHandler - registers a handler in the function that is triggered by bucket events
func (a *FunctionDependencies) AddBucketNotificationHandler(nw *storagepb.RegistrationRequest) {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.bucketNotifications[nw.GetBucketName()] = append(a.bucketNotifications[nw.GetBucketName()], nw)
}

// AddBucket - adds a storage bucket dependency to the function
func (a *FunctionDependencies) AddBucket(name string, b *resourcespb.BucketResource) {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.buckets[name] = b
}

// AddTopic - adds a pub/sub topic dependency to the function
func (a *FunctionDependencies) AddTopic(name string, t *resourcespb.TopicResource) {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.topics[name] = t
}

// AddCollection - adds a document database collection dependency to the function
func (a *FunctionDependencies) AddCollection(name string, c *resourcespb.CollectionResource) {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.collections[name] = c
}

func (a *FunctionDependencies) AddSecret(name string, s *resourcespb.SecretResource) {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.secrets[name] = s
}

// NewFunction - creates a new Nitric Function, ready to register handlers and dependencies.
func NewFunction(name string, projectFunction project.Function) *FunctionDependencies {
	return &FunctionDependencies{
		name:                name,
		functionConfig:      projectFunction,
		apis:                make(map[string]*Api),
		websockets:          make(map[string]*Websocket),
		subscriptions:       make(map[string]*topicspb.RegistrationRequest),
		httpWorkers:         make(map[string]*httppb.HttpProxyRequest),
		schedules:           make(map[string]*schedulespb.RegistrationRequest),
		buckets:             make(map[string]*resourcespb.BucketResource),
		topics:              make(map[string]*resourcespb.TopicResource),
		collections:         make(map[string]*resourcespb.CollectionResource),
		secrets:             make(map[string]*resourcespb.SecretResource),
		bucketNotifications: make(map[string][]*storagepb.RegistrationRequest),
		policies:            make([]*resourcespb.PolicyResource, 0),
		errors:              []string{},
	}
}
