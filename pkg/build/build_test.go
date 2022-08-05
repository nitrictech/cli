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
	"io/ioutil"
	"os"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/nitrictech/cli/mocks/mock_containerengine"
	"github.com/nitrictech/cli/pkg/containerengine"
	"github.com/nitrictech/cli/pkg/project"
)

func TestCreateBaseDev(t *testing.T) {
	ctrl := gomock.NewController(t)
	me := mock_containerengine.NewMockContainerEngine(ctrl)

	dir, err := ioutil.TempDir("", "test-nitric-build")
	if err != nil {
		t.Error(err)
	}

	defer os.RemoveAll(dir)

	s := project.New(&project.Config{Name: "", Dir: dir})
	s.Functions = map[string]project.Function{"foo": {Handler: "functions/list.ts"}}

	me.EXPECT().Build(gomock.Any(), dir, "nitric-ts-dev", map[string]string{}, []string{"node_modules/", ".nitric/", ".git/", ".idea/"})

	containerengine.DiscoveredEngine = me

	if err := CreateBaseDev(s); err != nil {
		t.Errorf("CreateBaseDev() error = %v", err)
	}
}
