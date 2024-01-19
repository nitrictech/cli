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
	"os"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-getter"
	"gopkg.in/yaml.v2"

	"github.com/nitrictech/cli/mocks/mock_utils"
	"github.com/nitrictech/cli/pkgplus/project/templates"
)

func TestRepository(t *testing.T) {
	ctrl := gomock.NewController(t)
	mgetter := mock_utils.NewMockGetterClient(ctrl)

	fh, err := os.CreateTemp("", "repository.*")
	if err != nil {
		t.Error(err)
	}

	configPath := fh.Name()

	t.Cleanup(func() { os.Remove(fh.Name()) })

	mgetter.EXPECT().Get().Do(func() {
		encoder := yaml.NewEncoder(fh)
		repo := repository{
			Templates: []TemplateInfo{
				{
					Name: "Java Stack (Multi Module)",
					Path: "./java-stack-multi",
				},
				{
					Name: "Go Stack",
					Path: "./go-stack",
				},
			},
		}
		if err := encoder.Encode(&repo); err != nil {
			t.Error(err)
		}

		fh.Close()
	})

	d := &downloader{
		configPath: configPath,
		newGetter: func(c *getter.Client) templates.GetterClient {
			return mgetter
		},
	}

	err = d.repository()
	if err != nil {
		t.Errorf("downloader.repository() error = %v", err)
		return
	}

	wantRepo := []TemplateInfo{
		{
			Name: "official/Java Stack (Multi Module)",
			Path: "java-stack-multi",
		},
		{
			Name: "official/Go Stack",
			Path: "go-stack",
		},
	}

	if !reflect.DeepEqual(d.repo, wantRepo) {
		t.Errorf("downloader.repository() = %v, want %v", d.repo, wantRepo)
	}
}
