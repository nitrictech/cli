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

package localservices

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/nitrictech/cli/pkg/run"
)

type ErrorMessage struct {
	Error error
}

type WarningMessage struct {
	Warning string
}

type StartingFunctionsMessage struct{}

type FunctionsStartedMessage struct{}

type QuitMessage struct{}

type StackUpdateMessage struct {
	StackState *run.RunStackState
}

type FunctionsReadyMessage struct {
	LocalServices run.LocalServices
	Functions     []*run.Function
}

func subscribeToChannel[T any](sub chan T) tea.Cmd {
	return func() tea.Msg {
		return <-sub
	}
}
