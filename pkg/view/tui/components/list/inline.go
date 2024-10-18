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

package list

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/paginator"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	tui "github.com/nitrictech/cli/pkg/view/tui"
	"github.com/nitrictech/cli/pkg/view/tui/components/view"
	"github.com/nitrictech/cli/pkg/view/tui/teax"
)

type InlineList struct {
	Items     SliceView[ListItem]
	choice    string
	minimized bool
	Paginator paginator.Model
}

type InlineListArgs struct {
	Items             []ListItem
	Minimized         bool
	MaxDisplayedItems int
}

func NewInlineList(args InlineListArgs) InlineList {
	p := paginator.New()
	p.Type = paginator.Dots
	p.ActiveDot = activePaginationDot.String()
	p.InactiveDot = inactivePaginationDot.String()

	items := NewSliceView(args.Items)
	items.SetMaxDisplayedItems(args.MaxDisplayedItems)

	return InlineList{
		Paginator: p,
		Items:     items,
		minimized: args.Minimized,
	}
}

func (m InlineList) Init() tea.Cmd {
	return nil
}

var (
	bullet                   = "•"
	cursorIconOffset         = lipgloss.NewStyle().MarginLeft(2)
	selected                 = lipgloss.NewStyle().Bold(true).Foreground(tui.Colors.TextActive)
	unselected               = cursorIconOffset.Copy().Foreground(tui.Colors.Text)
	descriptionStyle         = cursorIconOffset.Copy().Foreground(tui.Colors.TextMuted)
	descriptionSelectedStyle = cursorIconOffset.Copy().Foreground(tui.Colors.TextActive)
	inactivePaginationDot    = cursorIconOffset.Copy().Foreground(tui.Colors.TextMuted).SetString(bullet)
	activePaginationDot      = cursorIconOffset.Copy().Foreground(tui.Colors.Text).SetString(bullet)
)

func (m InlineList) View() string {
	listView := view.New()

	for i, item := range m.Items.View() {
		isSelected := m.Items.IsChoiceRelative(i)

		if isSelected {
			listView.Addln("→ %s", item.GetItemValue()).WithStyle(selected)
		} else {
			listView.Addln(item.GetItemValue()).WithStyle(unselected)
		}

		// Skip rendering the description if the list is minimized
		if m.minimized {
			continue
		}

		if item.GetItemDescription() != "" {
			if isSelected {
				listView.Addln(item.GetItemDescription()).WithStyle(descriptionSelectedStyle)
			} else {
				listView.Addln(item.GetItemDescription()).WithStyle(descriptionStyle)
			}

			listView.Break()
		}
	}

	if m.IsPaginationVisible() {
		m.Paginator.TotalPages = (len(m.Items.All()) + m.Items.NumDisplayed() - 1) / m.Items.NumDisplayed()
		m.Paginator.Page = max(0, m.Items.Cursor()/m.Items.NumDisplayed())

		listView.Addln(m.Paginator.View())
	}

	return strings.TrimSuffix(listView.Render(), "\n")
}

func (m InlineList) IsPaginationVisible() bool {
	return m.Items.NumDisplayed() < len(m.Items.All())
}

type UpdateListItemsMsg []ListItem

// UpdateInlineList does the same thing as Update, without erasing the component's type.
//
// useful when composing this model into another model
func (m InlineList) UpdateInlineList(msg tea.Msg) (InlineList, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, tui.KeyMap.Quit):
			return m, teax.Quit
		case key.Matches(msg, tui.KeyMap.Enter):
			m.choice = m.Items.Choice().GetItemValue()
		case key.Matches(msg, tui.KeyMap.Down):
			m.Items.CursorNext()
			return m, nil
		case key.Matches(msg, tui.KeyMap.Up):
			m.Items.CursorBack()
			return m, nil
		}
	}

	return m, nil
}

func (m InlineList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m.UpdateInlineList(msg)
}

func (m InlineList) Choice() string {
	return m.choice
}

func (m *InlineList) SetChoice(choice string) {
	m.choice = choice
}

func (m *InlineList) SetMaxDisplayedItems(maxDisplayedItems int) {
	m.Items.SetMaxDisplayedItems(maxDisplayedItems)
}

func (m *InlineList) SetMinimized(minimized bool) {
	m.minimized = minimized
}
