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

package remote

import (
	"os"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/reflow/wordwrap"

	deploy "github.com/nitrictech/nitric/core/pkg/proto/deployments/v1"
)

type Model struct {
	Content string
	width   int
}

func NewOutputModel() (tea.Model, error) {
	err := os.Setenv("CLICOLOR_FORCE", "1")
	if err != nil {
		return nil, err
	}

	return Model{
		Content: "",
	}, nil
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(
			key.WithKeys("esc", "ctrl+c")),
		):
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, tea.ClearScreen
	case *deploy.DeployDownEvent_Message:
		m.Content = msg.Message.Message
	case *deploy.DeployUpEvent_Message:
		m.Content = msg.Message.Message
	}

	return m, nil
}

func (m Model) View() string {
	return wordwrap.String(m.Content, m.width)
}
