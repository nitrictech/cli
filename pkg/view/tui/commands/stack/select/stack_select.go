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

package stack_select

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/nitrictech/cli/pkg/view/tui"
	"github.com/nitrictech/cli/pkg/view/tui/components/list"
	"github.com/nitrictech/cli/pkg/view/tui/components/listprompt"
	"github.com/nitrictech/cli/pkg/view/tui/teax"
)

// Model - represents the state of the stack selection list
type Model struct {
	windowSize tea.WindowSizeMsg

	stackPrompt listprompt.ListPrompt
}

// Init initializes the model, used by Bubbletea
func (m Model) Init() tea.Cmd {
	return nil
}

// Update the model based on a message
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowSize = msg

		if m.windowSize.Height < 7 {
			m.stackPrompt.SetMinimized(true)
			m.stackPrompt.SetMaxDisplayedItems(m.windowSize.Height - 1)
		} else {
			m.stackPrompt.SetMinimized(false)
			maxItems := ((m.windowSize.Height - 1) / 3) // make room for the exit message
			m.stackPrompt.SetMaxDisplayedItems(maxItems)
		}

		return m, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, tui.KeyMap.Quit):
			return m, teax.Quit
		}
	}

	m.stackPrompt, cmd = m.stackPrompt.UpdateListPrompt(msg)
	if m.stackPrompt.IsComplete() {
		return m, teax.Quit
	}

	return m, cmd
}

func (m Model) View() string {
	return m.stackPrompt.View()
}

func (m Model) Choice() string {
	return m.stackPrompt.Choice()
}

type Args struct {
	Prompt    string
	StackList []list.ListItem
}

type StackListItem struct {
	Name     string
	Provider string
}

func (s StackListItem) GetItemValue() string {
	return s.Name
}

func (s StackListItem) GetItemDescription() string {
	return s.Provider
}

var _ list.ListItem = StackListItem{}

func New(args Args) Model {
	prompt := args.Prompt
	if prompt == "" {
		prompt = "Select a stack"
	}

	stackPrompt := listprompt.NewListPrompt(listprompt.ListPromptArgs{
		Items:  args.StackList,
		Tag:    "stack",
		Prompt: prompt,
	})

	return Model{
		stackPrompt: stackPrompt,
	}
}
