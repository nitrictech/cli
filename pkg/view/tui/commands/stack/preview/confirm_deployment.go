package stack_preview

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	tui "github.com/nitrictech/cli/pkg/view/tui"
	"github.com/nitrictech/cli/pkg/view/tui/components/listprompt"
	"github.com/nitrictech/cli/pkg/view/tui/teax"
)

type ConfirmDeploymentModel struct {
	windowSize tea.WindowSizeMsg

	confirmPrompt listprompt.ListPrompt
}

// Init initializes the model, used by Bubbletea
func (m ConfirmDeploymentModel) Init() tea.Cmd {
	return nil
}

// Update the model based on a message
func (m ConfirmDeploymentModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowSize = msg

		if m.windowSize.Height < 7 {
			m.confirmPrompt.SetMinimized(true)
			m.confirmPrompt.SetMaxDisplayedItems(m.windowSize.Height - 1)
		} else {
			m.confirmPrompt.SetMinimized(false)
			maxItems := ((m.windowSize.Height - 1) / 3) // make room for the exit message
			m.confirmPrompt.SetMaxDisplayedItems(maxItems)
		}

		return m, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, tui.KeyMap.Quit):
			return m, teax.Quit
		}
	}

	m.confirmPrompt, cmd = m.confirmPrompt.UpdateListPrompt(msg)
	if m.confirmPrompt.IsComplete() {
		return m, teax.Quit
	}

	return m, cmd
}

func (m ConfirmDeploymentModel) View() string {
	return m.confirmPrompt.View()
}

func (m ConfirmDeploymentModel) Choice() string {
	return m.confirmPrompt.Choice()
}

func NewConfirmDeployment(args listprompt.ListPromptArgs) *ConfirmDeploymentModel {
	confirmPrompt := listprompt.NewListPrompt(args)

	return &ConfirmDeploymentModel{
		confirmPrompt: confirmPrompt,
	}
}
