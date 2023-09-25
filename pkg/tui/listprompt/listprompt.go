package listprompt

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nitrictech/cli/pkg/tui"
	"github.com/nitrictech/cli/pkg/tui/inlinelist"
	"github.com/nitrictech/cli/pkg/tui/view"
)

type Model struct {
	Prompt    string
	listInput inlinelist.Model
	Tag       string
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, tui.KeyMap.Quit):
			return m, tea.Quit
		}
	}

	m.listInput, cmd = m.listInput.Update(msg)

	return m, cmd
}

func (m Model) IsComplete() bool {
	return m.listInput.Choice() != ""
}

func (m Model) Choice() string {
	return m.listInput.Choice()
}

func (m *Model) SetChoice(choice string) {
	m.listInput.SetChoice(choice)
}

var (
	labelStyle  = lipgloss.NewStyle().MarginTop(1)
	tagStyle    = lipgloss.NewStyle().Background(tui.Colors.Purple).Foreground(tui.Colors.White).Width(8).Align(lipgloss.Center)
	promptStyle = lipgloss.NewStyle().MarginLeft(2)
	inputStyle  = lipgloss.NewStyle().MarginLeft(8)
	textStyle   = lipgloss.NewStyle().Foreground(tui.Colors.Gray).MarginLeft(10)
	errorStyle  = lipgloss.NewStyle().Foreground(tui.Colors.Red).Margin(1, 0, 0, 10).Italic(true)
)

func (m Model) View() string {
	listView := view.New()

	// render the list header
	listView.AddRow(
		view.NewFragment(m.Tag).WithStyle(tagStyle),
		view.NewFragment(m.Prompt).WithStyle(promptStyle),
		view.Break(),
	).WithStyle(labelStyle)

	// render the list view
	listView.AddRow(
		view.WhenOr(
			m.Choice() == "",
			view.NewFragment(m.listInput.View()).WithStyle(inputStyle),
			view.NewFragment(m.Choice()).WithStyle(textStyle),
		),
	)

	return listView.Render()
}

type Args struct {
	MaxDisplayedItems int
	Items             []string
	Prompt            string
	Tag               string
}

func New(args Args) Model {
	if args.MaxDisplayedItems < 1 {
		args.MaxDisplayedItems = 5
	}

	listInput := inlinelist.New(inlinelist.ModelArgs{
		Items:             args.Items,
		MaxDisplayedItems: args.MaxDisplayedItems,
	})

	return Model{
		Prompt:    args.Prompt,
		listInput: listInput,
		Tag:       args.Tag,
	}
}
