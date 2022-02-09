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

package utils

import (
	"fmt"

	"github.com/spf13/cast"
)

func ToStringMapStringMapStringE(i interface{}) (map[string]map[string]interface{}, error) {
	switch v := i.(type) {
	case map[string]map[string]interface{}:
		return v, nil
	case map[string]interface{}:
		var err error
		m := make(map[string]map[string]interface{})

		for k, val := range v {
			m[k], err = cast.ToStringMapE(val)
			if err != nil {
				return nil, err
			}
		}
		return m, nil
	default:
		return nil, fmt.Errorf("unable to cast %#v of type %T to map[string]map[string]interface{}", i, i)
	}
}
