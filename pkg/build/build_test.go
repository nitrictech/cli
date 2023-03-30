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

package build

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/nitrictech/cli/mocks/mock_containerengine"
	"github.com/nitrictech/cli/pkg/containerengine"
	"github.com/nitrictech/cli/pkg/project"
)

func TestBuildBaseImagesWithHandlers(t *testing.T) {
	ctrl := gomock.NewController(t)
	me := mock_containerengine.NewMockContainerEngine(ctrl)

	dir, err := os.MkdirTemp("", "test-nitric-build")
	if err != nil {
		t.Error(err)
	}

	defer os.RemoveAll(dir)

	s := project.New(project.BaseConfig{Name: "", Dir: dir})
	s.Functions = map[string]project.Function{"foo": {Handler: "functions/list.ts", ComputeUnit: project.ComputeUnit{Name: "foo"}}}

	me.EXPECT().Build(gomock.Any(), dir, "-foo", gomock.Any(), []string{
		".nitric/", "!.nitric/*.yaml", ".git/", ".idea/", ".vscode/", ".github/", "*.dockerfile", "*.dockerignore", "node_modules/",
	})

	containerengine.DiscoveredEngine = me

	if err := BuildBaseImages(s); err != nil {
		t.Errorf("CreateBaseDev() error = %v", err)
	}
}

func TestBuildBaseImagesWithContainers(t *testing.T) {
	ctrl := gomock.NewController(t)
	me := mock_containerengine.NewMockContainerEngine(ctrl)

	dir, err := os.MkdirTemp("", "test-nitric-build-container")
	if err != nil {
		t.Error(err)
	}

	dockerfile := "FROM node:alpine"

	fh, err := os.Create(filepath.Join(dir, "test.dockerfile"))
	if err != nil {
		t.Error(err)
	}

	_, err = fh.Write([]byte(dockerfile))
	if err != nil {
		t.Error(err)
	}

	fh.Close()

	defer os.RemoveAll(dir)

	hash := sha256.Sum256([]byte(dockerfile))
	hashValue := hex.EncodeToString(hash[:])
	imageName := fmt.Sprintf("%s-%s", "test.dockerfile", hashValue)

	s := project.New(project.BaseConfig{Name: "", Dir: dir})
	s.Functions = map[string]project.Function{imageName: {Dockerfile: "test.dockerfile", Context: dir, ComputeUnit: project.ComputeUnit{Name: imageName}}}

	me.EXPECT().Build(gomock.Any(), dir, "-"+imageName, gomock.Any(), []string{})

	me.EXPECT().TagImageToNitricName("-"+imageName, "")

	containerengine.DiscoveredEngine = me

	if err := BuildBaseImages(s); err != nil {
		t.Errorf("CreateBaseDev() error = %v", err)
	}
}
