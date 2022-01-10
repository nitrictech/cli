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
	"testing"

	"github.com/golang/mock/gomock"

	mock_containerengine "github.com/nitrictech/newcli/mocks/containerengine"
	"github.com/nitrictech/newcli/pkg/containerengine"
)

func TestCreateBaseDev(t *testing.T) {
	ctrl := gomock.NewController(t)
	me := mock_containerengine.NewMockContainerEngine(ctrl)
	me.EXPECT().Build(gomock.Any(), "path/to/stack", "nitric-ts-dev", map[string]string{})

	containerengine.MockEngine = me

	if err := CreateBaseDev("path/to/stack", map[string]string{"ts": "nitric-ts-dev"}); err != nil {
		t.Errorf("CreateBaseDev() error = %v", err)
	}
}
