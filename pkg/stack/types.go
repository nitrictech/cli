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

package stack

import (
	"io/ioutil"
	"path"
	"path/filepath"

	"github.com/getkin/kin-openapi/openapi3"
	"gopkg.in/yaml.v2"
)

type Triggers struct {
	Topics []string `yaml:"topics,omitempty"`
}

type ComputeUnit struct {
	name string `yaml:"-"` //nolint:structcheck,unused

	contextDirectory string `yaml:"-"` //nolint:structcheck,unused

	// Context is the directory containing the code for the function
	Context string `yaml:"context,omitempty"`

	// Triggers used to invoke this compute unit, e.g. Topic Subscriptions
	Triggers Triggers `yaml:"triggers,omitempty"`

	// The memory of the compute instance in MB
	Memory int `yaml:"memory,omitempty"`

	// The minimum number of instances to keep alive
	MinScale int `yaml:"minScale,omitempty"`

	// The maximum number of instances to scale to
	MaxScale int `yaml:"maxScale,omitempty"`

	// Allow the user to specify a custom unique tag for the function
	Tag string `yaml:"tag,omitempty"`
}

type Function struct {
	// The location of the function handler
	// relative to context
	Handler string `yaml:"handler"`

	// The build pack version of the membrane used for the function build
	Version string `yaml:"version,omitempty"`

	// Scripts that will be executed by the nitric
	// build process before beginning the docker build
	BuildScripts []string `yaml:"buildScripts,omitempty"`

	// files to exclude from final build
	Excludes []string `yaml:"excludes,omitempty"`

	// The most requests a single function instance should handle
	MaxRequests int `yaml:"maxRequests,omitempty"`

	// Simple configuration to determine if the function should be directly
	// invokable without authentication
	// would use public, but its reserved by typescript
	External bool `yaml:"external"`

	ComputeUnit `yaml:",inline"`
}

type Container struct {
	Dockerfile string   `yaml:"dockerfile"`
	Args       []string `yaml:"args,omitempty"`

	ComputeUnit `yaml:",inline"`
}

type Compute interface {
	Name() string
	ImageTagName(s *Stack, provider string) string
	Unit() *ComputeUnit
}

// A subset of a NitricEvent
// excluding it's requestId
// This will be generated based on the scedule
type ScheduleEvent struct {
	PayloadType string                 `yaml:"payloadType"`
	Payload     map[string]interface{} `yaml:"payload,omitempty"`
}

type ScheduleTarget struct {
	Type string `yaml:"type"` // TODO(Angus) check type: 'topic'; // ; | "queue"
	Name string `yaml:"name"`
}

type Schedule struct {
	Expression string `yaml:"expression"`

	// The Topic to be targeted for schedule
	Target ScheduleTarget `yaml:"target"`
	Event  ScheduleEvent  `yaml:"event"`
}

type Collection struct{}

type Bucket struct{}

type Topic struct{}

type Queue struct{}

// A static site deployment with Nitric
// We also support server rendered applications
type Site struct {
	// Base path of the site
	// Will be used to execute scripts
	Path string `yaml:"path"`
	// Path to get assets to upload
	// this will be relative to path
	AssetPath string `yaml:"assetPath"`
	// Build scripts to execute before upload
	BuildScripts []string `yaml:"buildScripts,omitempty"`
}

type EntrypointPath struct {
	Target string `yaml:"target"`
	Type   string `yaml:"type"` // 'site' | 'api' | 'function' | 'container';
}

type Entrypoint struct {
	Domains []string                  `yaml:"domains,omitempty"`
	Paths   map[string]EntrypointPath `yaml:"paths,omitempty"`
}

type Stack struct {
	dir         string
	Name        string                 `yaml:"name"`
	Functions   map[string]Function    `yaml:"functions,omitempty"`
	Collections map[string]Collection  `yaml:"collections,omitempty"`
	Containers  map[string]Container   `yaml:"containers,omitempty"`
	Buckets     map[string]Bucket      `yaml:"buckets,omitempty"`
	Topics      map[string]Topic       `yaml:"topics,omitempty"`
	Queues      map[string]Queue       `yaml:"queues,omitempty"`
	Schedules   map[string]Schedule    `yaml:"schedules,omitempty"`
	apiDocs     map[string]*openapi3.T `yaml:"-"`
	Apis        map[string]string      `yaml:"apis,omitempty"`
	Sites       map[string]Site        `yaml:"sites,omitempty"`
	EntryPoints map[string]Entrypoint  `yaml:"entrypoints,omitempty"`
}

func (s *Stack) SetApiDoc(name string, doc *openapi3.T) {
	if s.apiDocs == nil {
		s.apiDocs = make(map[string]*openapi3.T)
	}

	s.apiDocs[name] = doc
}

func FromFile(name string) (*Stack, error) {
	yamlFile, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, err
	}
	dir, err := filepath.Abs(path.Dir(name))
	if err != nil {
		return nil, err
	}
	stack := &Stack{dir: dir}
	err = yaml.Unmarshal(yamlFile, stack)
	if err != nil {
		return nil, err
	}
	for name, fn := range stack.Functions {
		fn.name = name
		if fn.Context != "" {
			fn.contextDirectory = path.Join(stack.Path(), fn.Context)
		} else {
			fn.contextDirectory = stack.Path()
		}
		stack.Functions[name] = fn
	}
	for name, c := range stack.Containers {
		c.name = name
		if c.Context != "" {
			c.contextDirectory = path.Join(stack.Path(), c.Context)
		} else {
			c.contextDirectory = stack.Path()
		}
		stack.Containers[name] = c
	}

	// Attempt to populate documents from api file references
	for k, v := range stack.Apis {
		if doc, err := openapi3.NewLoader().LoadFromFile(filepath.Join(stack.dir, v)); err != nil {
			return nil, err
		} else {
			if stack.apiDocs == nil {
				stack.apiDocs = make(map[string]*openapi3.T)
			}

			stack.apiDocs[k] = doc
		}
	}

	return stack, nil
}

func (s *Stack) Path() string {
	return s.dir
}

func (s *Stack) ToFile(name string) error {
	b, err := yaml.Marshal(s)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(name, b, 0)
}
