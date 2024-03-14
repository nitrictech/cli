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

package netx

import (
	"bufio"
	"os"
	"strings"

	"github.com/hashicorp/consul/sdk/freeport"

	"github.com/nitrictech/nitric/core/pkg/logger"
)

// TakePort is just a wrapper around freeport.TakePort() that changes the
// stderr output to pterm.Debug
func TakePort(n int) ([]int, error) {
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	defer r.Close()

	stderr := os.Stderr
	os.Stderr = w
	ports, err := freeport.Take(n)
	os.Stderr = stderr

	w.Close()

	in := bufio.NewScanner(r)

	for in.Scan() {
		logger.Debug(strings.TrimPrefix(in.Text(), "[INFO] "))
	}

	return ports, err
}
