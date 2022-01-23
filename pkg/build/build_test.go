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

	"github.com/nitrictech/newcli/mocks/mock_containerengine"
	"github.com/nitrictech/newcli/pkg/containerengine"
	"github.com/nitrictech/newcli/pkg/stack"
	"github.com/nitrictech/newcli/pkg/target"
)

func TestCreateBaseDev(t *testing.T) {
	ctrl := gomock.NewController(t)
	me := mock_containerengine.NewMockContainerEngine(ctrl)

	dir, err := ioutil.TempDir("", "test-nitric-build")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	me.EXPECT().Build(gomock.Any(), dir, "nitric-ts-dev", map[string]string{})

	containerengine.DiscoveredEngine = me

	if err := CreateBaseDev(dir, map[string]string{"ts": "nitric-ts-dev"}); err != nil {
		t.Errorf("CreateBaseDev() error = %v", err)
	}
}

func TestCreate(t *testing.T) {
	ctrl := gomock.NewController(t)
	me := mock_containerengine.NewMockContainerEngine(ctrl)
	me.EXPECT().Build(gomock.Any(), "", "corp-abc-dev:123456", map[string]string{"PROVIDER": "aws"})
	me.EXPECT().Build("Dockerfile.custom", "", "corp-xyz-dev:444444", map[string]string{"PROVIDER": "aws"})

	containerengine.MockEngine = me

	s := &stack.Stack{
		Name: "test-stack",
		Functions: map[string]stack.Function{
			"list": {
				Handler: "functions/list.ts",
				ComputeUnit: stack.ComputeUnit{
					Tag: "corp-abc-dev:123456",
				},
				BuildScripts: []string{"ls"},
			},
		},
		Containers: map[string]stack.Container{
			"doit": {
				Dockerfile: "Dockerfile.custom",
				ComputeUnit: stack.ComputeUnit{
					Tag: "corp-xyz-dev:444444",
				},
			},
		},
	}

	if err := Create(s, &target.Target{Provider: "aws", Region: "eastus"}); err != nil {
		t.Errorf("CreateBaseDev() error = %v", err)
	}
}
