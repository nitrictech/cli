package inlinelist

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nitrictech/cli/pkg/tui"
)

type Model struct {
	cursor             int
	Items              []string
	MaxDisplayedItems  int
	firstDisplayedItem int
	choice             string
}

type ModelArgs struct {
	Items             []string
	MaxDisplayedItems int
}

func New(args ModelArgs) Model {
	return Model{
		cursor:             0,
		firstDisplayedItem: 0,
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

func (m Model) Init() tea.Cmd {
	return nil
}

var (
	listItemStyle     = lipgloss.NewStyle().Foreground(tui.Colors.Gray)
	selectedItemStyle = listItemStyle.Copy().Bold(true).Foreground(tui.Colors.White)
)

func (m Model) View() string {
	var view strings.Builder
	for i := 0; i < min(m.MaxDisplayedItems, len(m.Items)); i++ {
		if i+m.firstDisplayedItem == m.cursor {
			view.WriteString(selectedItemStyle.Render(fmt.Sprintf("→ %s", m.Items[i+m.firstDisplayedItem])))
		} else {
			view.WriteString(listItemStyle.Render(fmt.Sprintf("  %s", m.Items[i+m.firstDisplayedItem])))
		}
		view.WriteString("\n")
	}
	return fmt.Sprintf("%s\n", view.String())
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, tui.KeyMap.Quit):
			return m, tea.Quit
		case key.Matches(msg, tui.KeyMap.Enter):
			m.choice = m.Items[m.cursor]
		case key.Matches(msg, tui.KeyMap.Down):
			return m.CursorDown(), nil
		case key.Matches(msg, tui.KeyMap.Up):
			return m.CursorUp(), nil
		}
	}

	return m, nil
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
