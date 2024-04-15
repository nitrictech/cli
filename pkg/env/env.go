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

package env

import (
	"os"

	"github.com/joho/godotenv"
)

var defaultEnv = ".env"

func ReadEnv(filePath string) (map[string]string, error) {
	file, err := os.OpenFile(filePath, os.O_RDONLY|os.O_CREATE, 0o666)
	if err != nil {
		return nil, err
	}

	return godotenv.Parse(file)
}

func ReadLocalEnv(additionalFilePaths ...string) (map[string]string, error) {
	envVariables, err := ReadEnv(defaultEnv)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	if envVariables == nil {
		envVariables = map[string]string{}
	}

	for _, filePath := range additionalFilePaths {
		additionalEnvVariables, err := ReadEnv(filePath)
		if err != nil {
			return nil, err
		}

		for key, value := range additionalEnvVariables {
			envVariables[key] = value
		}
	}

	return envVariables, nil
}

func LoadLocalEnv(additionalFilePaths ...string) error {
	paths := append(additionalFilePaths, defaultEnv)
	return godotenv.Load(paths...)
}
