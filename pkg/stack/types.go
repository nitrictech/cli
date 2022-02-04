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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"

	"github.com/getkin/kin-openapi/openapi3"
	v1 "github.com/nitrictech/nitric/pkg/api/nitric/v1"
	"gopkg.in/yaml.v2"
)

type Triggers struct {
	Topics []string `yaml:"topics,omitempty"`
}

type ComputeUnit struct {
	Name string `yaml:"-"`

	// This is the stack.Dir + Context
	ContextDirectory string `yaml:"-"`

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
	ImageTagName(s *Stack, provider string) string
	SetContextDirectory(stackDir string)
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

type Stack struct {
	Dir         string                 `yaml:"-"`
	Name        string                 `yaml:"name"`
	Functions   map[string]Function    `yaml:"functions,omitempty"`
	Collections map[string]Collection  `yaml:"collections,omitempty"`
	Containers  map[string]Container   `yaml:"containers,omitempty"`
	Buckets     map[string]Bucket      `yaml:"buckets,omitempty"`
	Topics      map[string]Topic       `yaml:"topics,omitempty"`
	Queues      map[string]Queue       `yaml:"queues,omitempty"`
	Schedules   map[string]Schedule    `yaml:"schedules,omitempty"`
	ApiDocs     map[string]*openapi3.T `yaml:"-"`
	Apis        map[string]string      `yaml:"apis,omitempty"`
	// TODO: Not currently supported by nitric.yaml configuration (but is technically definable using the proto model)
	// We may want to decouple the definition from contracts at a later stage
	// but re-using the contract here provides us a serializable entity with no
	// repetition/redefinition
	// NOTE: if we want to use the proto definition here we would need support for yaml parsing to use customisable tags
	Policies []*v1.PolicyResource `yaml:"-"`
}

func New(name, dir string) *Stack {
	return &Stack{
		Name:        name,
		Dir:         dir,
		Containers:  map[string]Container{},
		Collections: map[string]Collection{},
		Functions:   map[string]Function{},
		Buckets:     map[string]Bucket{},
		Topics:      map[string]Topic{},
		Queues:      map[string]Queue{},
		Schedules:   map[string]Schedule{},
		Apis:        map[string]string{},
		ApiDocs:     map[string]*openapi3.T{},
		Policies:    make([]*v1.PolicyResource, 0),
	}
}

// Compute default policies for a stack
func calculateDefaultPolicies(s *Stack) []*v1.PolicyResource {
	// TODO: Calculate policies for stacks loaded from a file
	return []*v1.PolicyResource{}
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
	stack := &Stack{Dir: dir}
	err = yaml.Unmarshal(yamlFile, stack)
	if err != nil {
		return nil, err
	}
	for name, fn := range stack.Functions {
		fn.Name = name
		fn.SetContextDirectory(stack.Dir)
		stack.Functions[name] = fn
	}
	for name, c := range stack.Containers {
		c.Name = name
		c.SetContextDirectory(stack.Dir)
		stack.Containers[name] = c
	}

	// Attempt to populate documents from api file references
	for k, v := range stack.Apis {
		if doc, err := openapi3.NewLoader().LoadFromFile(filepath.Join(stack.Dir, v)); err != nil {
			return nil, err
		} else {
			if stack.ApiDocs == nil {
				stack.ApiDocs = make(map[string]*openapi3.T)
			}

			stack.ApiDocs[k] = doc
		}
	}

	// Calculate default policies
	stack.Policies = calculateDefaultPolicies(stack)

	return stack, nil
}

func (s *Stack) ToFile(name string) error {
	b, err := yaml.Marshal(s)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path.Join(s.Dir, name), b, 0644)
	if err != nil {
		return err
	}

	for apiName, apiFile := range s.Apis {
		apiPath := path.Join(s.Dir, apiFile)
		doc, ok := s.ApiDocs[apiName]
		if !ok {
			return fmt.Errorf("apiDoc %s does not exist", apiPath)
		}
		docJ, err := json.Marshal(doc)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(apiPath, docJ, 0644)
		if err != nil {
			return err
		}
	}
	return nil
}
