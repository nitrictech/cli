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

package templates

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/hashicorp/go-getter"
	"gopkg.in/yaml.v2"

	"github.com/nitrictech/newcli/pkg/utils"
)

type Template struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

type TemplatesConfig struct {
	Templates []Template
}

const (
	rawGitHubURL        = "https://raw.githubusercontent.com"
	templatesRepo       = "nitrictech/stack-templates"
	templatesRepoGitURL = "github.com/nitrictech/stack-templates.git"
)

var configPath = path.Join(utils.NitricHome(), "store", "repositories.yml")

func ReadTemplatesConfig() (*TemplatesConfig, error) {
	var config *TemplatesConfig

	// Open YAML file
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Decode YAML file to struct
	if file != nil {
		decoder := yaml.NewDecoder(file)
		if err := decoder.Decode(&config); err != nil {
			return nil, err
		}
	}

	return config, nil
}

func officialStackName(name string) string {
	return "official/" + name
}

func ListTemplates() (TemplatesConfig, error) {
	client := &getter.Client{
		Ctx: context.Background(),
		//define the destination to where the directory will be stored. This will create the directory if it doesnt exist
		Dst: configPath,
		//the repository with a subdirectory I would like to clone only
		Src:  rawGitHubURL + "/" + path.Join(templatesRepo, "main/repository.yaml"),
		Mode: getter.ClientModeFile,
		//define the type of detectors go getter should use, in this case only github is needed

		//provide the getter needed to download the files
		Getters: map[string]getter.Getter{
			"https": &getter.HttpGetter{},
		},
	}

	// download file
	if err := client.Get(); err != nil {
		return TemplatesConfig{}, fmt.Errorf("Error getting path %s: %v", client.Src, err)
	}

	var config, err = ReadTemplatesConfig()

	if err != nil {
		return TemplatesConfig{}, err
	}

	if config.Templates == nil {
		return TemplatesConfig{}, errors.New("Templates array does not exist in respositories.yml")
	}

	var transformedConfig = TemplatesConfig{}

	for _, template := range config.Templates {
		transformedTemplate := Template{Name: officialStackName(template.Name), Path: filepath.Base(template.Path)}
		transformedConfig.Templates = append(transformedConfig.Templates, transformedTemplate)
	}

	return transformedConfig, nil
}

func DownloadDirectoryContents(templatePath string, destDir string, force bool) error {
	_, err := os.Stat(destDir)
	if err == nil && !force {
		return errors.New("stack directory already exists and isn't empty, choose a different name or use the --force flag to create in a non-empty directory")
	}

	client := &getter.Client{
		Ctx: context.Background(),
		//define the destination to where the directory will be stored. This will create the directory if it doesnt exist
		Dst: destDir,
		Dir: true,
		//the repository with a subdirectory I would like to clone only
		Src:  templatesRepoGitURL + "//" + templatePath,
		Mode: getter.ClientModeDir,
		//define the type of detectors go getter should use, in this case only github is needed
		Detectors: []getter.Detector{
			&getter.GitHubDetector{},
		},
		//provide the getter needed to download the files
		Getters: map[string]getter.Getter{
			"git": &getter.GitGetter{},
		},
	}

	// TODO add spinner

	// downloads files
	if err := client.Get(); err != nil {
		return fmt.Errorf("Error getting path %s: %v", client.Src, err)
	}

	return nil
}
