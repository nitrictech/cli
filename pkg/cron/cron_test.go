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

package cron

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRateToCron(t *testing.T) {
	tests := []struct {
		name    string
		rate    string
		want    string
		wantErr error
	}{
		{
			name:    "not enough parts",
			rate:    "x",
			want:    "",
			wantErr: errors.New("not enough parts to rate expression x"),
		},
		{
			name:    "not a valid value",
			rate:    "x 45",
			wantErr: errors.New("invalid rate expression x 45; strconv.Atoi: parsing \"x\": invalid syntax"),
		},
		{
			name:    "not a valid unit",
			rate:    "45 x",
			wantErr: errors.New("invalid rate expression 45 x; x must be one of [minutes, hours, days]"),
		},
		{
			name: "45 minutes",
			rate: "45 minutes",
			want: "*/45 * * * *",
		},
		{
			name: "6 hours",
			rate: "6 hours",
			want: "0 */6 * * *",
		},
		{
			name: "3 days",
			rate: "3 days",
			want: "0 0 */3 * *",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RateToCron(tt.rate)
			if tt.wantErr != nil {
				require.EqualError(t, err, tt.wantErr.Error())
				return
			}
			if got != tt.want {
				t.Errorf("RateToCron() = %v, want %v", got, tt.want)
			}
		})
	}
}
