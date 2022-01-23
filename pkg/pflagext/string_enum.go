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

package pflagext

import (
	"fmt"
	"strings"
)

type stringEnum struct {
	Allowed []string
	ValueP  *string
}

// NewStringEnumVar give a list of allowed flag parameters, where the second argument is the default
func NewStringEnumVar(value *string, allowed []string, d string) *stringEnum {
	*value = d
	return &stringEnum{
		Allowed: allowed,
		ValueP:  value,
	}
}

func (e *stringEnum) String() string {
	return *e.ValueP
}

func (e *stringEnum) Set(p string) error {
	isIncluded := func(opts []string, val string) bool {
		for _, opt := range opts {
			if val == opt {
				return true
			}
		}
		return false
	}
	if !isIncluded(e.Allowed, p) {
		return fmt.Errorf("%s is not included in %s", p, strings.Join(e.Allowed, ","))
	}
	*e.ValueP = p
	return nil
}

func (e *stringEnum) Type() string {
	return "stringEnumVar"
}
