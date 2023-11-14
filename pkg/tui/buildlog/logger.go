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

package buildlog

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/samber/lo"
)

type LogMessage struct {
	Function string
	Message  string
	Info     bool
}

type ErrorMessage struct {
	Error error
}

type FunctionsBuiltMessage struct{}

type LogWriter struct {
	Sub      chan tea.Msg
	Function string
}

func subscribeToChannel(sub chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-sub
	}
}

func (l *LogWriter) Write(b []byte) (int, error) {
	// Send a log message to the channel for every line in the message
	logLines := strings.Split(string(b), "\n")

	for _, line := range lo.Compact(logLines) {
		l.Sub <- LogMessage{
			Message:  line,
			Function: l.Function,
			Info:     strings.Contains(line, "CACHED") || strings.Contains(line, "DONE"),
		}
	}

	return len(b), nil
}
