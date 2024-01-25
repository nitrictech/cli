package build

import (
	"sort"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/samber/lo"

	"github.com/nitrictech/cli/pkgplus/project"
	tui "github.com/nitrictech/cli/pkgplus/view/tui/components"
	"github.com/nitrictech/cli/pkgplus/view/tui/components/view"
	"github.com/nitrictech/cli/pkgplus/view/tui/reactive"
)

type Model struct {
	serviceBuildUpdates map[string]project.ServiceBuildUpdate

	serviceBuildUpdatesChannel chan project.ServiceBuildUpdate

	spinner spinner.Model
}

var _ tea.Model = (*Model)(nil)

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		reactive.AwaitChannel(m.serviceBuildUpdatesChannel),
		m.spinner.Tick,
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case reactive.ChanMsg[project.ServiceBuildUpdate]:
		// channel closed, the build is complete.
		if !msg.Ok {
			return m, tea.Quit
		}

		m.serviceBuildUpdates[msg.Value.ServiceName] = msg.Value

		// resubscribe to the messages originating channel
		return m, reactive.AwaitChannel(msg.Source)
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)

		return m, cmd
	}

	return m, nil
}

var (
	headingStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFDF5"))
	inProgStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#0000A0"))
	doneStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#00A000"))
	messageStyle = lipgloss.NewStyle().MarginLeft(2).Foreground(tui.Colors.Gray)
)

func (m Model) View() string {
	buildView := view.NewRenderer()

	buildView.AddRow(
		view.NewFragment("Building services"+m.spinner.View()).WithStyle(headingStyle),
		view.Break(),
	)

	serviceNames := lo.Keys(m.serviceBuildUpdates)

	sort.Strings(serviceNames)

	for _, serviceName := range serviceNames {
		service := m.serviceBuildUpdates[serviceName]

		serviceProgStyle := inProgStyle
		if service.Status == project.ServiceBuildStatus_Complete {
			serviceProgStyle = doneStyle
		}

		buildView.AddRow(
			view.NewFragment(serviceName),
			view.NewFragment(" "),
			view.NewFragment(service.Status).WithStyle(serviceProgStyle),
			view.Break(),
			view.NewFragment(service.Message).WithStyle(messageStyle),
		)
	}

	return buildView.Render()
}

func NewModel(serviceBuildUpdates chan project.ServiceBuildUpdate) Model {
	return Model{
		spinner:                    spinner.New(spinner.WithSpinner(spinner.Ellipsis)),
		serviceBuildUpdatesChannel: serviceBuildUpdates,
		serviceBuildUpdates:        make(map[string]project.ServiceBuildUpdate),
	}
}
