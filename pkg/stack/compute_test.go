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

package stack

import (
	"reflect"
	"testing"
)

func TestCompute(t *testing.T) {
	s := &Stack{Dir: "../run", Name: "test"}
	cu := ComputeUnit{
		Name: "unit",
	}

	for _, c := range []Compute{&Container{ComputeUnit: cu}, &Function{ComputeUnit: cu}} {
		gotImageName := c.ImageTagName(s, "aws")
		if gotImageName != "test-unit-aws" {
			t.Error("imageTagName", gotImageName)
		}

		if !reflect.DeepEqual(c.Unit(), &cu) {
			t.Error("unit", c.Unit())
		}
	}
}
