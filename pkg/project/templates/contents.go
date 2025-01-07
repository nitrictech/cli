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
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-getter"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/nitrictech/cli/pkg/paths"
)

const (
	rawGitHubURL        = "https://raw.githubusercontent.com"
	templatesRepo       = "nitrictech/examples"
	templatesRepoGitURL = "github.com/nitrictech/examples.git"
)

type TemplateInfo struct {
	Name  string `yaml:"name"`
	Label string `yaml:"label"`
	Desc  string `yaml:"desc"`
	Path  string `yaml:"path"`
}

type repository struct {
	Templates []TemplateInfo
}

type Downloader interface {
	Templates() ([]TemplateInfo, error)
	Get(name string) *TemplateInfo
	GetByLabel(label string) *TemplateInfo
	DownloadDirectoryContents(name string, destDir string, force bool) error
}

type downloader struct {
	configPath string
	newGetter  func(*getter.Client) GetterClient
	repo       []TemplateInfo
}

var _ Downloader = &downloader{}

func NewDownloader() Downloader {
	return &downloader{
		configPath: filepath.Join(paths.NitricTemplatesDir(), "cli-templates.yaml"),
		newGetter:  NewGetter,
	}
}

func (d *downloader) lazyLoadTemplates() error {
	if len(d.repo) == 0 {
		err := d.repository()
		if err != nil {
			if strings.Contains(err.Error(), "git must be available and on the PATH") {
				return errors.WithMessage(err, "please refer to the installation instructions - https://nitric.io/docs/installation")
			}

			return err
		}
	}

	return nil
}

func (d *downloader) Templates() ([]TemplateInfo, error) {
	err := d.lazyLoadTemplates()
	if err != nil {
		return nil, err
	}

	return d.repo, nil
}

func (d *downloader) Get(name string) *TemplateInfo {
	err := d.lazyLoadTemplates()
	if err != nil {
		return nil
	}

	for _, ti := range d.repo {
		if ti.Name == name {
			return &ti
		}
	}

	return nil
}

func (d *downloader) GetByLabel(label string) *TemplateInfo {
	err := d.lazyLoadTemplates()
	if err != nil {
		return nil
	}

	for _, ti := range d.repo {
		if ti.Label == label {
			return &ti
		}
	}

	return nil
}

func (d *downloader) readTemplatesConfig(client GetterClient, retryCount int) ([]TemplateInfo, error) {
	file, err := os.Open(d.configPath)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	decoder := yaml.NewDecoder(file)
	repo := repository{}

	if err := decoder.Decode(&repo); err != nil {
		// if an error occurs while decoding the yaml file, delete the file and try again based on the retry count
		if retryCount > 0 {
			// close the file before deleting it
			file.Close()

			err = os.Remove(d.configPath)
			if err != nil {
				return nil, errors.WithMessage(err, "repository file "+d.configPath)
			}

			err = client.Get()
			if err != nil {
				return nil, errors.WithMessage(err, "repository file "+d.configPath)
			}

			return d.readTemplatesConfig(client, retryCount-1)
		}

		return nil, errors.WithMessage(err, "repository file "+d.configPath)
	}

	return repo.Templates, nil
}

func (d *downloader) repository() error {
	src := rawGitHubURL + "/" + filepath.Join(templatesRepo, "main/cli-templates.yaml")

	client := d.newGetter(&getter.Client{
		Ctx: context.Background(),
		// define the destination to where the directory will be stored. This will create the directory if it doesnt exist
		Dst: d.configPath,
		// the repository with a subdirectory I would like to clone only
		Src:  src,
		Mode: getter.ClientModeFile,
		// define the type of detectors go getter should use, in this case only github is needed

		// provide the getter needed to download the files
		Getters: map[string]getter.Getter{
			"https": &getter.HttpGetter{
				DoNotCheckHeadFirst: true,
			},
		},
	})

	// download file
	if err := client.Get(); err != nil {
		return err
	}

	list, err := d.readTemplatesConfig(client, 1)
	if err != nil {
		return err
	}

	d.repo = []TemplateInfo{}

	for _, template := range list {
		d.repo = append(d.repo, TemplateInfo{
			Name:  template.Name,
			Label: template.Label,
			Desc:  template.Desc,
			Path:  filepath.Clean(template.Path),
		})
	}

	return nil
}

func (d *downloader) DownloadDirectoryContents(name string, destDir string, force bool) error {
	_, err := os.Stat(destDir)
	if err == nil && !force {
		return errors.New("project directory already exists and isn't empty, choose a different name or use the --force flag to create in a non-empty directory")
	}

	template := d.Get(name)
	if template == nil {
		return fmt.Errorf("template %s not found", name)
	}

	client := d.newGetter(&getter.Client{
		Ctx: context.Background(),
		// define the destination to where the directory will be stored. This will create the directory if it doesnt exist
		Dst: destDir,
		Dir: true,
		// the repository with a subdirectory I would like to clone only
		Src:  templatesRepoGitURL + "//" + strings.ReplaceAll(template.Path, "\\", "/"),
		Mode: getter.ClientModeDir,
		// define the type of detectors go getter should use, in this case only github is needed
		Detectors: []getter.Detector{
			&getter.GitHubDetector{},
		},
		// provide the getter needed to download the files
		Getters: map[string]getter.Getter{
			"git": &getter.GitGetter{},
		},
	})

	err = client.Get()

	if err != nil && strings.Contains(err.Error(), "git must be available and on the PATH") {
		return errors.WithMessage(err, "please refer to the installation instructions - https://nitric.io/docs/installation")
	}

	return errors.WithMessagef(err, "error getting path %s//%s", templatesRepoGitURL, template.Path)
}
