// Copyright Nitric Pty Ltd.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package iox

import (
	"io"
	"strings"
)

type channelWriter struct {
	out chan<- string
}

func (cw channelWriter) Write(bytes []byte) (int, error) {
	lines := strings.Split(string(bytes), "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		cw.out <- line
	}

	return len(bytes), nil
}

func NewChannelWriter(channel chan<- string) io.Writer {
	return channelWriter{
		out: channel,
	}
}
