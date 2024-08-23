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

package localconfig

import (
	"fmt"
	"os"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

type LocalResourceConfiguration struct {
	// This port is used to map the local
	Port int `yaml:"port"`
}

type LocalConfiguration struct {
	Apis       map[string]LocalResourceConfiguration `yaml:"apis"`
	Websockets map[string]LocalResourceConfiguration `yaml:"websockets"`
}

const defaultLocalNitricYamlPath = "./local.nitric.yaml"

func LocalConfigurationFromFile(fs afero.Fs, filePath string) (*LocalConfiguration, error) {
	if filePath == "" {
		filePath = defaultLocalNitricYamlPath
	}

	// Check if the path is a directory
	info, err := fs.Stat(defaultLocalNitricYamlPath)
	if err != nil {
		if os.IsNotExist(err) {
			// ignore if the file does not exist, it is optional
			return nil, nil
		}

		return nil, err
	}

	if info.IsDir() {
		return nil, fmt.Errorf("local.nitric.yaml path %s is a directory", filePath)
	}

	localConfigFileContents, err := afero.ReadFile(fs, filePath)
	if err != nil {
		return nil, fmt.Errorf("unable to read local.nitric.yaml: %w", err)
	}

	localConfig := &LocalConfiguration{}

	if err := yaml.Unmarshal(localConfigFileContents, localConfig); err != nil {
		return nil, fmt.Errorf("unable to parse local.nitric.yaml: %w", err)
	}

	return localConfig, nil
}
