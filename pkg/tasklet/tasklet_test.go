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

package tasklet

import (
	"errors"
	"testing"
	"time"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name    string
		runner  Runner
		opts    Opts
		wantErr error
	}{
		{
			name: "success no opts",
			runner: Runner{
				StartMsg: "start test 22",
				Runner: func(tc TaskletContext) error {
					if tc.Spinner().Text != "start test 22" {
						return errors.New("start text not set")
					}
					return nil
				},
			},
		},
		{
			name: "fail no opts",
			runner: Runner{
				Runner: func(tc TaskletContext) error { return errors.New("bang!") },
			},
			wantErr: errors.New("bang!"),
		},
		{
			name: "timeout",
			runner: Runner{
				Runner: func(tc TaskletContext) error {
					time.Sleep(time.Minute)
					return nil
				},
			},
			opts:    Opts{Timeout: time.Second * 2},
			wantErr: errors.New("tasklet timedout after 2s"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Run(tt.runner, tt.opts)
			if (err != nil) != (tt.wantErr != nil) {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
			if (err != nil) && (tt.wantErr != nil) && err.Error() != tt.wantErr.Error() {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
