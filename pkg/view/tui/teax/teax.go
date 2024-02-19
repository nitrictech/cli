package teax

import tea "github.com/charmbracelet/bubbletea"

type fullHeightModel struct {
	tea.Model
	quitting bool
}

var _ tea.Model = fullHeightModel{}

func (q fullHeightModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case QuitMsg:
		q.quitting = true

		return q, tea.Quit
	}

	var cmd tea.Cmd
	q.Model, cmd = q.Model.Update(msg)

	return q, cmd
}

func (q fullHeightModel) FullView() string {
	return q.Model.View()
}

func (q fullHeightModel) View() string {
	if q.quitting {
		return ""
	}

	return q.Model.View()
}
