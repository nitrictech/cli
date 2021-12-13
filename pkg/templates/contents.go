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
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v41/github"
	"golang.org/x/oauth2"
)

type RepoContent interface {
	ListSubDirectories(path string) ([]string, error)
	DownloadDirectoryContents(path string, destDirectory string, force bool) error
}

type repoContent struct {
	client *github.Client
	repo   string
	owner  string
}

func NewRepoContent(owner, repo string) RepoContent {
	ctx := context.Background()

	token := os.Getenv("GITHUB_AUTH_TOKEN")
	var tc *http.Client
	if token != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		tc = oauth2.NewClient(ctx, ts)
	} else {
		fmt.Println("Attempting to download from github without a GITHUB_AUTH_TOKEN")
	}

	return &repoContent{
		repo:   repo,
		owner:  owner,
		client: github.NewClient(tc),
	}
}

func (r *repoContent) ListSubDirectories(path string) ([]string, error) {
	_, directoryContent, _, err := r.client.Repositories.GetContents(context.Background(), r.owner, r.repo, path, nil)
	if err != nil {
		return nil, err
	}

	var subdirs []string
	for _, c := range directoryContent {
		switch *c.Type {
		case "file":
		case "dir":
			if !strings.HasPrefix(*c.Path, ".") && strings.Contains(*c.Path, "-stack") {
				subdirs = append(subdirs, *c.Path)
			}
		}
	}
	return subdirs, nil
}

func (r *repoContent) DownloadDirectoryContents(path string, destDir string, force bool) error {
	_, err := os.Stat(destDir)
	if err == nil && !force {
		return errors.New("stack directory already exists and isn't empty, choose a different name or use the --force flag to create in a non-empty directory")
	}

	_, directoryContent, _, err := r.client.Repositories.GetContents(context.Background(), r.owner, r.repo, path, nil)
	if err != nil {
		return err
	}

	for _, c := range directoryContent {
		local := filepath.Join(destDir, strings.Replace(*c.Path, path, "", 1))

		switch *c.Type {
		case "file":
			r.downloadFile(c, local, force)
		case "dir":
			err = r.DownloadDirectoryContents(*c.Path, local, force)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *repoContent) downloadFile(content *github.RepositoryContent, localPath string, force bool) error {
	rc, _, err := r.client.Repositories.DownloadContents(context.Background(), r.owner, r.repo, *content.Path, nil)
	if err != nil {
		return err
	}
	defer rc.Close()

	b, err := ioutil.ReadAll(rc)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(localPath), 0755)
	if err != nil {
		return err
	}

	_, err = os.Stat(localPath)
	if err == nil && !force {
		return errors.New("file already exists re-run with --force to create, disregarding existing contents")
	}
	fmt.Println(localPath)
	f, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer f.Close()
	n, err := f.Write(b)
	if err != nil {
		return err
	}
	if n != *content.Size {
		return fmt.Errorf("number of bytes differ, %d vs %d\n", n, *content.Size)
	}
	return nil
}
