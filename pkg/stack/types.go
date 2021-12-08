package stack

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Function struct {
	// The location of the function handler
	// relative to context
	Handler string `yaml:"handler"`

	// TODO(Angus) not in the TypeScript
	Memory int `yaml:"memory,omitempty"`

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
}

type Container struct {
	Dockerfile string   `yaml:"dockerfile"`
	Args       []string `yaml:"args,omitempty"`
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
	Name        string                 `yaml:"name"`
	Functions   map[string]Function    `yaml:"functions,omitempty"`
	Collections map[string]interface{} `yaml:"collections,omitempty"`
	Containers  map[string]Container   `yaml:"containers,omitempty"`
	Buckets     map[string]interface{} `yaml:"buckets,omitempty"`
	Topics      map[string]interface{} `yaml:"topics,omitempty"`
	Queues      map[string]interface{} `yaml:"queues,omitempty"`
	Schedules   map[string]Schedule    `yaml:"schedules,omitempty"`
	Apis        map[string]string      `yaml:"apis,omitempty"`
	Sites       map[string]Site        `yaml:"sites,omitempty"`
	EntryPoints map[string]Entrypoint  `yaml:"entrypoints,omitempty"`
}

func FromFile(name string) (*Stack, error) {
	yamlFile, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, err
	}

	stack := &Stack{}
	err = yaml.Unmarshal(yamlFile, stack)
	if err != nil {
		return nil, err
	}
	return stack, nil
}

func (s *Stack) ToFile(name string) error {
	b, err := yaml.Marshal(s)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(name, b, 0)
}
