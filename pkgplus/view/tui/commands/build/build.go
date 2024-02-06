package build

import (
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/samber/lo"

	"github.com/nitrictech/cli/pkgplus/project"
	tui "github.com/nitrictech/cli/pkgplus/view/tui"
	"github.com/nitrictech/cli/pkgplus/view/tui/components/view"
	"github.com/nitrictech/cli/pkgplus/view/tui/fragments"
	"github.com/nitrictech/cli/pkgplus/view/tui/reactive"
	"github.com/nitrictech/cli/pkgplus/view/tui/teax"
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
			return m, teax.Quit
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

func (m *Model) AllDone() bool {
	for _, service := range m.serviceBuildUpdates {
		if service.Status == project.ServiceBuildStatus_Complete {
			continue
		}
		if service.Status == project.ServiceBuildStatus_Error {
			continue
		}
		return false
	}

	return true
}

func (m Model) View() string {

	v := view.New()
	v.Break()
	v.Add(fragments.Tag("build"))
	v.Add("  Building services")
	if !m.AllDone() {
		v.Add(m.spinner.View())
	}
	v.Break()

	gap := strings.Builder{}
	for i := 0; i < fragments.TagWidth()+2; i++ {
		gap.WriteString(" ")
	}

	serviceNames := lo.Keys(m.serviceBuildUpdates)

	sort.Strings(serviceNames)

	serviceUpdates := view.New(view.WithStyle(lipgloss.NewStyle().MarginLeft(fragments.TagWidth() + 2)))
	serviceUpdates.Break()
	for _, serviceName := range serviceNames {
		service := m.serviceBuildUpdates[serviceName]

		statusColor := tui.Colors.Gray
		if service.Status == project.ServiceBuildStatus_Complete {
			statusColor = tui.Colors.Green
		} else if service.Status == project.ServiceBuildStatus_InProgress {
			statusColor = tui.Colors.Blue
		} else if service.Status == project.ServiceBuildStatus_Error {
			statusColor = tui.Colors.Red
		}

		messageLines := strings.Split(strings.TrimSpace(service.Message), "\n")

		serviceUpdates.Add("%s ", serviceName)
		serviceUpdates.Addln(strings.ToLower(string(service.Status))).WithStyle(lipgloss.NewStyle().Foreground(statusColor))
		if len(messageLines) > 0 && service.Status != project.ServiceBuildStatus_Complete {
			serviceUpdates.Addln("  %s", messageLines[len(messageLines)-1]).WithStyle(lipgloss.NewStyle().Foreground(tui.Colors.Gray))
		}
	}
	v.Add(serviceUpdates.Render())

	return v.Render()
}

func NewModel(serviceBuildUpdates chan project.ServiceBuildUpdate) Model {
	return Model{
		spinner:                    spinner.New(spinner.WithSpinner(spinner.Ellipsis)),
		serviceBuildUpdatesChannel: serviceBuildUpdates,
		serviceBuildUpdates:        make(map[string]project.ServiceBuildUpdate),
	}
}
