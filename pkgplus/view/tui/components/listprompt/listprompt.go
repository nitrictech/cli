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

package listprompt

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	tui "github.com/nitrictech/cli/pkgplus/view/tui/components"
	"github.com/nitrictech/cli/pkgplus/view/tui/components/list"
	"github.com/nitrictech/cli/pkgplus/view/tui/components/view"
)

type ListPrompt struct {
	Prompt    string
	listInput list.InlineList
	Tag       string
}

func (m ListPrompt) Init() tea.Cmd {
	return nil
}

func (m ListPrompt) UpdateItems(items []list.ListItem) ListPrompt {
	m.listInput = m.listInput.UpdateItems(items)

	return m
}

func (m ListPrompt) UpdateListPrompt(msg tea.Msg) (ListPrompt, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, tui.KeyMap.Quit):
			return m, tea.Quit
		}
	}

	m.listInput, cmd = m.listInput.UpdateInlineList(msg)

	return m, cmd
}

func (m ListPrompt) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m.UpdateListPrompt(msg)
}

func (m ListPrompt) IsComplete() bool {
	return m.listInput.Choice() != ""
}

func (m ListPrompt) Choice() string {
	return m.listInput.Choice()
}

func (m *ListPrompt) SetChoice(choice string) {
	m.listInput.SetChoice(choice)
}

var (
	tagStyle    = lipgloss.NewStyle().Background(tui.Colors.Purple).Foreground(tui.Colors.White).Width(8).Align(lipgloss.Center)
	promptStyle = lipgloss.NewStyle().MarginLeft(2)
	inputStyle  = lipgloss.NewStyle().MarginLeft(8)
	textStyle   = lipgloss.NewStyle().Foreground(tui.Colors.Gray).MarginLeft(10)
)

func (m ListPrompt) View() string {
	listView := view.New()

	// render the list header
	listView.Add(m.Tag).WithStyle(tagStyle, lipgloss.NewStyle().MarginTop(1))
	listView.Addln(m.Prompt).WithStyle(promptStyle)
	listView.Break()

	// render the list
	if m.Choice() == "" {
		listView.Addln(m.listInput.View()).WithStyle(inputStyle)
	} else {
		listView.Addln(m.Choice()).WithStyle(textStyle)
	}

	return listView.Render()
}

type ListPromptArgs struct {
	MaxDisplayedItems int
	Items             []list.ListItem
	Prompt            string
	Tag               string
}

func NewListPrompt(args ListPromptArgs) ListPrompt {
	if args.MaxDisplayedItems < 1 {
		args.MaxDisplayedItems = 5
	}

	listInput := list.NewInlineList(list.InlineListArgs{
		Items:             args.Items,
		MaxDisplayedItems: args.MaxDisplayedItems,
	})

	return ListPrompt{
		Prompt:    args.Prompt,
		listInput: listInput,
		Tag:       args.Tag,
	}
}
