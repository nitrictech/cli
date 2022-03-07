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
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Name     string   `yaml:"name"`
	Dir      string   `yaml:"-"`
	Handlers []string `yaml:"handlers"`
}

func (p *Config) ToFile() error {
	if p.Dir == "" || p.Name == "" {
		return errors.New("fields Dir and Name must be provided")
	}

	b, err := yaml.Marshal(p)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(p.Dir, "nitric.yaml"), b, 0644)
}

// Assume the project is in the currentDirectory
func ConfigFromFile() (*Config, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	absDir, err := filepath.Abs(wd)
	if err != nil {
		return nil, err
	}

	p := &Config{
		Dir: absDir,
	}
	yamlFile, err := ioutil.ReadFile(filepath.Join(wd, "nitric.yaml"))
	if err != nil {
		return nil, errors.WithMessage(err, "No nitric project found (unable to find nitric.yaml). If you haven't created a project yet, run `nitric new` to get started")
	}

	err = yaml.Unmarshal(yamlFile, p)
	if err != nil {
		return nil, err
	}
	return p, nil
}
