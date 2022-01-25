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
	"errors"
	"testing"
)

func TestErrorList_Error(t *testing.T) {
	tests := []struct {
		name             string
		errs             []error
		wantError        string
		wantAggregateNil bool
	}{
		{
			name:      "one",
			errs:      []error{errors.New("one")},
			wantError: "one",
		},
		{
			name:             "nil",
			errs:             []error{},
			wantError:        "",
			wantAggregateNil: true,
		},
		{
			name:      "multiple",
			errs:      []error{errors.New("one"), errors.New("two")},
			wantError: "one\ntwo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			el := NewErrorList()
			for _, e := range tt.errs {
				el.Add(e)
			}
			if tt.wantAggregateNil != (el.Aggregate() == nil) {
				t.Errorf("ErrorList.Aggregat() = %v, want %v", el.Aggregate(), tt.wantAggregateNil)
			}
			if got := el.Error(); got != tt.wantError {
				t.Errorf("ErrorList.Error() = %v, want %v", got, tt.wantError)
			}
		})
	}
}
