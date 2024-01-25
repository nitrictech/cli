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
	tea "github.com/charmbracelet/bubbletea"

	"github.com/nitrictech/cli/pkgplus/view/tui/components/inlinelist"
	"github.com/nitrictech/cli/pkgplus/view/tui/components/listprompt"
)

// Model - represents the state of the stack selection list
type Model struct {
	listModel tea.Model
}

// Init initializes the model, used by Bubbletea
func (m Model) Init() tea.Cmd {
	return nil
}

// Update the model based on a message
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	}

	m.listModel, cmd = m.listModel.Update(msg)
	if m.listModel.(listprompt.Model).IsComplete() {
		return m, tea.Quit
	}

	return m, cmd
}

func (m Model) View() string {
	return m.listModel.View()
}

func (m Model) Choice() string {
	return m.listModel.(listprompt.Model).Choice()
}

type Args struct {
	StackList []inlinelist.ListItem
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

var _ inlinelist.ListItem = StackListItem{}

func New(args Args) Model {
	listModel := listprompt.New(listprompt.Args{
		Items:  args.StackList,
		Tag:    "stack",
		Prompt: "Which stack would you like to update?",
	})

	return Model{
		listModel: listModel,
	}
}
