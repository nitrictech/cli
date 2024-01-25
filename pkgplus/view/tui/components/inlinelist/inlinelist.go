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

package inlinelist

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/paginator"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	tui "github.com/nitrictech/cli/pkgplus/view/tui/components"
	"github.com/nitrictech/cli/pkgplus/view/tui/components/view"
)

type ListItem interface {
	GetItemValue() string
	GetItemDescription() string
}
type Model struct {
	cursor             int
	Items              []ListItem
	MaxDisplayedItems  int
	firstDisplayedItem int
	choice             string
	Paginator          paginator.Model
}

type Args struct {
	Items             []ListItem
	MaxDisplayedItems int
}

func New(args Args) Model {
	p := paginator.New()
	p.Type = paginator.Dots
	p.ActiveDot = activePaginationDot.String()
	p.InactiveDot = inactivePaginationDot.String()

	return Model{
		cursor:             0,
		firstDisplayedItem: 0,
		Paginator:          p,
		Items:              args.Items,
		MaxDisplayedItems:  args.MaxDisplayedItems,
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m Model) Init() tea.Cmd {
	return nil
}

var (
	bullet                   = "•"
	cursorIconOffset         = lipgloss.NewStyle().MarginLeft(2)
	selected                 = lipgloss.NewStyle().Bold(true).Foreground(tui.Colors.Blue)
	unselected               = cursorIconOffset.Copy().Foreground(tui.Colors.White)
	descriptionStyle         = cursorIconOffset.Copy().Foreground(tui.Colors.Gray)
	descriptionSelectedStyle = cursorIconOffset.Copy().Foreground(tui.Colors.Blue)
	inactivePaginationDot    = cursorIconOffset.Copy().Foreground(tui.Colors.Gray).SetString(bullet)
	activePaginationDot      = cursorIconOffset.Copy().Foreground(tui.Colors.White).SetString(bullet)
)

func (m Model) View() string {
	listView := view.New()
	maxDisplayedItems := min(m.MaxDisplayedItems, len(m.Items))

	for i := 0; i < maxDisplayedItems; i++ {
		listView.AddRow(
			view.WhenOr(
				i+m.firstDisplayedItem == m.cursor,
				view.NewFragment(fmt.Sprintf("→ %s", m.Items[i+m.firstDisplayedItem].GetItemValue())).WithStyle(selected),
				view.NewFragment(m.Items[i+m.firstDisplayedItem].GetItemValue()).WithStyle(unselected),
			),
		)

		if m.Items[i+m.firstDisplayedItem].GetItemDescription() != "" {
			listView.AddRow(view.WhenOr(
				i+m.firstDisplayedItem == m.cursor,
				view.NewFragment(m.Items[i+m.firstDisplayedItem].GetItemDescription()).WithStyle(descriptionSelectedStyle),
				view.NewFragment(m.Items[i+m.firstDisplayedItem].GetItemDescription()).WithStyle(descriptionStyle),
			),
				view.Break())
		}
	}

	if maxDisplayedItems < len(m.Items) {
		m.Paginator.TotalPages = (len(m.Items) + maxDisplayedItems - 1) / maxDisplayedItems
		m.Paginator.Page = max(0, m.cursor/maxDisplayedItems)

		listView.AddRow(view.NewFragment(m.Paginator.View()))
	}

	return listView.Render()
}

type UpdateListItemsMsg []ListItem

// UpdateInlineList does the same thing as Update, without erasing the component's type.
//
// useful when composing this model into another model
func (m Model) UpdateInlineList(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, tui.KeyMap.Quit):
			return m, tea.Quit
		case key.Matches(msg, tui.KeyMap.Enter):
			m.choice = m.Items[m.cursor].GetItemValue()
		case key.Matches(msg, tui.KeyMap.Down):
			return m.CursorDown(), nil
		case key.Matches(msg, tui.KeyMap.Up):
			return m.CursorUp(), nil
		}
	}

	return m, nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m.UpdateInlineList(msg)
}

func (m Model) UpdateItems(items []ListItem) Model {
	m.Items = items
	m.cursor = 0
	m.firstDisplayedItem = 0

	return m
}

func (m Model) CursorUp() Model {
	m.cursor--
	if m.cursor < 0 {
		m.cursor = len(m.Items) - 1
	}

	return m.refreshViewCursor()
}

func (m Model) CursorDown() Model {
	m.cursor = (m.cursor + 1) % len(m.Items)

	return m.refreshViewCursor()
}

// lastDisplayedItem returns the index of the last item currently visible in the list
func (m Model) lastDisplayedItem() int {
	return m.firstDisplayedItem + (m.MaxDisplayedItems - 1)
}

func (m Model) refreshViewCursor() Model {
	for m.cursor > m.lastDisplayedItem() {
		m.firstDisplayedItem++
	}

	for m.cursor < m.firstDisplayedItem {
		m.firstDisplayedItem--
	}

	return m
}

func (m Model) Choice() string {
	return m.choice
}

func (m *Model) SetChoice(choice string) {
	m.choice = choice
}
