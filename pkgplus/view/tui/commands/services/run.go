package services

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/nitrictech/cli/pkgplus/cloud"
	"github.com/nitrictech/cli/pkgplus/project"
	"github.com/nitrictech/cli/pkgplus/view/tui/commands/local"
	"github.com/nitrictech/cli/pkgplus/view/tui/components/view"
	"github.com/nitrictech/cli/pkgplus/view/tui/reactive"
)

type Model struct {
	stopChan           chan<- bool
	updateChan         <-chan project.ServiceRunUpdate
	localServicesModel tea.Model

	serviceStatus map[string]project.ServiceRunUpdate
}

var _ tea.Model = (*Model)(nil)

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		reactive.AwaitChannel(m.updateChan),
		m.localServicesModel.Init(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			func() {
				m.stopChan <- true
			}()
			return m, tea.Quit
		}
	case reactive.ChanMsg[project.ServiceRunUpdate]:
		// we know we have a service update
		m.serviceStatus[msg.Value.ServiceName] = msg.Value

		return m, reactive.AwaitChannel(msg.Source)
	default:
		// give unknown messages to to sub model
		newLocalModel, cmd := m.localServicesModel.Update(msg)
		m.localServicesModel = newLocalModel

		return m, cmd
	}

	var cmd tea.Cmd
	m.localServicesModel, cmd = m.localServicesModel.Update(msg)

	return m, cmd
}

var headingStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFDF5"))

func (m Model) View() string {
	runView := view.New()

	runView.Addln(m.localServicesModel.View())

	runView.Addln("Running services").WithStyle(headingStyle)
	runView.Break()

	for _, service := range m.serviceStatus {
		runView.Addln("%s - %s", service.ServiceName, service.Status)

		if service.Err != nil {
			runView.Addln(service.Err.Error())
		}
	}

	return runView.Render()
}

func NewModel(stopChannel chan<- bool, updateChannel <-chan project.ServiceRunUpdate, localCloud *cloud.LocalCloud, dashboardUrl string) Model {
	localServicesModel := local.NewTuiModel(localCloud, dashboardUrl)

	return Model{
		stopChan:           stopChannel,
		localServicesModel: localServicesModel,
		updateChan:         updateChannel,
		serviceStatus:      make(map[string]project.ServiceRunUpdate),
	}
}
