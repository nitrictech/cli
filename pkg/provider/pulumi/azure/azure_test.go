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

package azure

import (
	"testing"

	"github.com/hashicorp/go-version"
	"golang.org/x/exp/slices"
)

func Test_azureProvider_Plugins(t *testing.T) {
	want := []string{"azure-native", "azure", "azuread"}
	got := (&azureProvider{}).Plugins()

	for _, pl := range got {
		_, err := version.NewVersion(pl.Version)
		if err != nil {
			t.Error(err)
		}

		if !slices.Contains(want, pl.Name) {
			t.Errorf("azureProvider.Plugins() = %v not in want %v", pl, want)
		}
	}
}
