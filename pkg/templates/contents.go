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
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/hashicorp/go-getter"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/nitrictech/cli/pkg/utils"
)

const (
	rawGitHubURL        = "https://raw.githubusercontent.com"
	templatesRepo       = "nitrictech/stack-templates"
	templatesRepoGitURL = "github.com/nitrictech/stack-templates.git"
)

type TemplateInfo struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

type repository struct {
	Templates []TemplateInfo
}

type Downloader interface {
	Names() ([]string, error)
	Get(name string) *TemplateInfo
	DownloadDirectoryContents(name string, destDir string, force bool) error
}

type downloader struct {
	configPath string
	newGetter  func(*getter.Client) utils.GetterClient
	repo       []TemplateInfo
}

var _ Downloader = &downloader{}

func NewDownloader() Downloader {
	return &downloader{
		configPath: path.Join(utils.NitricTemplatesDir(), "repositories.yml"),
		newGetter:  utils.NewGetter,
	}
}

func (d *downloader) Names() ([]string, error) {
	names := []string{}
	if len(d.repo) == 0 {
		err := d.repository()
		if err != nil {
			return nil, err
		}
	}
	for _, ti := range d.repo {
		names = append(names, ti.Name)
	}
	return names, nil
}

func (d *downloader) Get(name string) *TemplateInfo {
	for _, ti := range d.repo {
		if ti.Name == name {
			return &ti
		}
	}
	return nil
}

func (d *downloader) readTemplatesConfig() ([]TemplateInfo, error) {
	file, err := os.Open(d.configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	repo := repository{}
	if err := decoder.Decode(&repo); err != nil {
		return nil, errors.WithMessage(err, "repository file "+d.configPath)
	}

	return repo.Templates, nil
}

func officialStackName(name string) string {
	return "official/" + name
}

func (d *downloader) repository() error {
	src := rawGitHubURL + "/" + path.Join(templatesRepo, "main/repository.yaml")
	client := d.newGetter(&getter.Client{
		Ctx: context.Background(),
		//define the destination to where the directory will be stored. This will create the directory if it doesnt exist
		Dst: d.configPath,
		//the repository with a subdirectory I would like to clone only
		Src:  src,
		Mode: getter.ClientModeFile,
		//define the type of detectors go getter should use, in this case only github is needed

		//provide the getter needed to download the files
		Getters: map[string]getter.Getter{
			"https": &getter.HttpGetter{},
		},
	})

	// download file
	if err := client.Get(); err != nil {
		return fmt.Errorf("error getting path %s: %w", src, err)
	}

	list, err := d.readTemplatesConfig()
	if err != nil {
		return err
	}

	d.repo = []TemplateInfo{}
	for _, template := range list {
		d.repo = append(d.repo, TemplateInfo{
			Name: officialStackName(template.Name),
			Path: filepath.Clean(template.Path),
		})
	}

	return nil
}

func (d *downloader) DownloadDirectoryContents(name string, destDir string, force bool) error {
	_, err := os.Stat(destDir)
	if err == nil && !force {
		return errors.New("stack directory already exists and isn't empty, choose a different name or use the --force flag to create in a non-empty directory")
	}

	template := d.Get(name)
	if template == nil {
		return fmt.Errorf("template %s not found", name)
	}

	client := d.newGetter(&getter.Client{
		Ctx: context.Background(),
		//define the destination to where the directory will be stored. This will create the directory if it doesnt exist
		Dst: destDir,
		Dir: true,
		//the repository with a subdirectory I would like to clone only
		Src:  templatesRepoGitURL + "//" + template.Path,
		Mode: getter.ClientModeDir,
		//define the type of detectors go getter should use, in this case only github is needed
		Detectors: []getter.Detector{
			&getter.GitHubDetector{},
		},
		//provide the getter needed to download the files
		Getters: map[string]getter.Getter{
			"git": &getter.GitGetter{},
		},
	})

	err = client.Get()
	return errors.WithMessagef(err, "error getting path %s", templatesRepoGitURL+"//"+template.Path)
}
