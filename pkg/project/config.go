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

package project

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"

	"github.com/nitrictech/cli/pkg/preview"
)

type RuntimeConfiguration struct {
	// Template dockerfile to use as the base for the custom runtime
	Dockerfile string
	// Directory path for the Docker build context for the custom runtime
	Context string
	// Additional args to pass to the custom runtime
	Args map[string]string
}

type BaseService interface {
	GetBasedir() string
	GetMatch() string
	GetRuntime() string
	GetStart() string
}

type BaseServiceConfiguration struct {
	// The base directory for source files
	Basedir string `yaml:"basedir"`

	// This is the string version
	Match string `yaml:"match"`

	// This is the custom runtime version (is custom if not nil, we auto-detect a standard language runtime)
	Runtime string `yaml:"runtime,omitempty"`

	// This is a command that will be use to run these services when using nitric start
	Start string `yaml:"start"`
}

func (b BaseServiceConfiguration) GetBasedir() string {
	return b.Basedir
}

func (b BaseServiceConfiguration) GetMatch() string {
	return b.Match
}

func (b BaseServiceConfiguration) GetRuntime() string {
	return b.Runtime
}

func (b BaseServiceConfiguration) GetStart() string {
	return b.Start
}

type ServiceConfiguration struct {
	BaseServiceConfiguration `yaml:",inline"`

	// This allows specifying a particular service type (e.g. "Job"), this is optional and custom service types can be defined for each stack
	Type string `yaml:"type,omitempty"`
}

type BatchConfiguration struct {
	BaseServiceConfiguration `yaml:",inline"`
}

type Build struct {
	Command string `yaml:"command"`
	Output  string `yaml:"output"`
}

type Dev struct {
	Command string `yaml:"command"`
}

type WebsiteConfiguration struct {
	BaseServiceConfiguration `yaml:",inline"`

	Build     Build  `yaml:"build"`
	Dev       Dev    `yaml:"dev"`
	IndexPage string `yaml:"index,omitempty"`
	ErrorPage string `yaml:"error,omitempty"`
}

type ProjectConfiguration struct {
	Name      string                          `yaml:"name"`
	Directory string                          `yaml:"-"`
	Services  []ServiceConfiguration          `yaml:"services"`
	Ports     map[string]int                  `yaml:"ports,omitempty"`
	Batches   []BatchConfiguration            `yaml:"batch-services"`
	Websites  []WebsiteConfiguration          `yaml:"websites"`
	Runtimes  map[string]RuntimeConfiguration `yaml:"runtimes,omitempty"`
	Preview   []preview.Feature               `yaml:"preview,omitempty"`
}

const defaultNitricYamlPath = "./nitric.yaml"

func (p ProjectConfiguration) ToFile(fs afero.Fs, filepath string) error {
	nitricYamlPath := defaultNitricYamlPath

	if filepath != "" {
		nitricYamlPath = filepath
	}

	projectBytes, err := yaml.Marshal(p)
	if err != nil {
		return err
	}

	if err = afero.WriteFile(fs, nitricYamlPath, projectBytes, os.ModePerm); err != nil {
		return err
	}

	return nil
}

func ConfigurationFromFile(fs afero.Fs, filePath string) (*ProjectConfiguration, error) {
	if filePath == "" {
		filePath = defaultNitricYamlPath
	}

	absProjectDir, err := filepath.Abs(filepath.Dir(filePath))
	if err != nil {
		return nil, err
	}

	// Check if the path is a directory
	info, err := fs.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("nitric.yaml not found in %s. Check that you are in the root directory of a nitric project", absProjectDir)
		}

		return nil, err
	}

	if info.IsDir() {
		return nil, fmt.Errorf("nitric.yaml path %s is a directory", filePath)
	}

	projectFileContents, err := afero.ReadFile(fs, filePath)
	if err != nil {
		return nil, fmt.Errorf("unable to read nitric.yaml: %w", err)
	}

	// TODO: Implement v0 yaml detection and provide link to the upgrade guide
	projectConfig := &ProjectConfiguration{}

	if err := yaml.Unmarshal(projectFileContents, projectConfig); err != nil {
		return nil, fmt.Errorf("unable to parse nitric.yaml: %w", err)
	}

	projectConfig.Directory = filepath.Dir(filePath)

	return projectConfig, nil
}
