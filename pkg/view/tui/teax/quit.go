package teax

import tea "github.com/charmbracelet/bubbletea"

type QuitMsg struct{}

func Quit() tea.Msg {
	return QuitMsg{}
}
