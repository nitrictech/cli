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
	"reflect"

	"github.com/nitrictech/cli/pkg/utils"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// Config shared by all compute types
type BaseComputeConfig struct {
	Type string `yaml:"type"`
}

type HandlerConfig struct {
	BaseComputeConfig
	Match string
}

// TODO: Determine best way to use generic mixed type constraint when deserializing
// type Handler interface {
// 	string | HandlerConfig
// }

type BaseConfig struct {
	Name       string         `yaml:"name"`
	Dir        string         `yaml:"-"`
	Handlers   []any          `yaml:"handlers"`
	Containers []DockerConfig `yaml:"containers" validate:"handlers_containers_required,validate_docker_object,dive"`
}

type DockerConfig struct {
	BaseComputeConfig `yaml:",inline"`
	Dockerfile        string            `yaml:"dockerfile"`
	Image             string            `yaml:"image" validate:"omitempty,validate_docker_image"`
	Context           string            `yaml:"context"`
	Args              map[string]string `yaml:"args"`
	Nitric            bool              `yaml:"nitric"`
}

type Config struct {
	BaseConfig       `yaml:",inline"`
	ConcreteHandlers []*HandlerConfig `yaml:"-"`
}

func validateConfig(config *Config) error {
	validate := validator.New()

	err := validate.RegisterValidation("handlers_containers_required", func(fl validator.FieldLevel) bool {
		return len(config.Handlers) > 0 || len(config.Containers) > 0
	})
	if err != nil {
		return err
	}

	err = validate.RegisterValidation("validate_docker_object", func(fl validator.FieldLevel) bool {
		for _, dc := range config.Containers {
			// a dockerfile or image must be set
			if dc.Dockerfile == "" && dc.Image == "" {
				return false
			}

			// a dockerfile cannot be set with an image
			if dc.Dockerfile != "" && dc.Image != "" {
				return false
			}
		}

		return true
	})
	if err != nil {
		return err
	}

	err = validate.RegisterValidation("validate_docker_image", func(fl validator.FieldLevel) bool {
		image, err := utils.ParseDockerImage(fl.Field().String())
		if err != nil {
			return false
		}

		return image.Name != ""
	})
	if err != nil {
		return err
	}

	err = validate.Struct(config)
	if err != nil {
		valErrors := err.(validator.ValidationErrors)
		for _, ve := range valErrors {
			switch ve.Tag() {
			case "handlers_containers_required":
				return errors.New("handlers or containers must be provided")
			case "validate_docker_object":
				return errors.New("container must have a dockerfile or image key")
			case "required":
				return errors.Errorf(`field %s must be provided`, ve.Field())
			}
		}

		return errors.WithMessage(valErrors, "there was an error validating the nitric config")
	}

	return nil
}

func (p *Config) ToFile() error {
	if p.Dir == "" || p.Name == "" {
		return errors.New("fields Dir and Name must be provided")
	}

	b, err := yaml.Marshal(p)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(p.Dir, "nitric.yaml"), b, 0o644)
}

// configFromBaseConfig - Unwraps Generic configs (e.g. Handler) and populates missing defaults (e.g. Type)
func configFromBaseConfig(base BaseConfig) (*Config, error) {
	newConfig := &Config{
		BaseConfig:       base,
		ConcreteHandlers: make([]*HandlerConfig, 0),
	}

	for _, h := range base.Handlers {
		if str, isString := h.(string); isString {
			// if its a basic string populate with default handler config
			newConfig.ConcreteHandlers = append(newConfig.ConcreteHandlers, &HandlerConfig{
				BaseComputeConfig: BaseComputeConfig{
					Type: "default",
				},
				Match: str,
			})
		} else if m, isMap := h.(map[any]any); isMap {
			// otherwise extract its map configuration
			// TODO: Check and validate the map properties
			newConfig.ConcreteHandlers = append(newConfig.ConcreteHandlers, &HandlerConfig{
				BaseComputeConfig: BaseComputeConfig{
					Type: m["type"].(string),
				},
				Match: m["match"].(string),
			})
		} else {
			return nil, fmt.Errorf("invalid handler config provided: %+v %s", h, reflect.TypeOf(h))
		}
	}

	return newConfig, nil
}

// ConfigFromProjectPath - loads the config nitric.yaml file from the project path, defaults to the current working directory
func ConfigFromProjectPath(projPath string) (*Config, error) {
	if projPath == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}

		projPath = wd
	}

	absDir, err := filepath.Abs(projPath)
	if err != nil {
		return nil, err
	}

	p := BaseConfig{
		Dir: absDir,
	}

	yamlFile, err := os.ReadFile(filepath.Join(projPath, "nitric.yaml"))
	if err != nil {
		return nil, errors.WithMessage(err, "No nitric project found (unable to find nitric.yaml). If you haven't created a project yet, run `nitric new` to get started")
	}

	err = yaml.Unmarshal(yamlFile, &p)
	if err != nil {
		return nil, err
	}

	config, err := configFromBaseConfig(p)
	if err != nil {
		return nil, err
	}

	err = validateConfig(config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
