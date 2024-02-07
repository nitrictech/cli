package services

import (
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/samber/lo"

	"github.com/nitrictech/cli/pkgplus/cloud"
	"github.com/nitrictech/cli/pkgplus/project"
	"github.com/nitrictech/cli/pkgplus/view/tui"
	"github.com/nitrictech/cli/pkgplus/view/tui/commands/local"
	"github.com/nitrictech/cli/pkgplus/view/tui/components/view"
	"github.com/nitrictech/cli/pkgplus/view/tui/fragments"
	"github.com/nitrictech/cli/pkgplus/view/tui/reactive"
	"github.com/nitrictech/cli/pkgplus/view/tui/teax"
)

type Model struct {
	stopChan           chan<- bool
	updateChan         <-chan project.ServiceRunUpdate
	localServicesModel tea.Model

	windowSize tea.WindowSizeMsg

	serviceStatus     map[string]project.ServiceRunUpdate
	serviceRunUpdates []project.ServiceRunUpdate
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
	case tea.WindowSizeMsg:
		m.windowSize = msg
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			func() {
				m.stopChan <- true
			}()
			return m, teax.Quit
		}
	case reactive.ChanMsg[project.ServiceRunUpdate]:
		// we know we have a service update
		m.serviceStatus[msg.Value.ServiceName] = msg.Value
		m.serviceRunUpdates = append(m.serviceRunUpdates, msg.Value)

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

var serviceColors = []lipgloss.CompleteColor{
	tui.Colors.Blue,
	tui.Colors.Purple,
	tui.Colors.Teal,
	tui.Colors.Red,
	tui.Colors.Orange,
	tui.Colors.Green,
}

func tail(text string, take int) string {
	if take < 1 {
		return text
	}

	lines := strings.Split(text, "\n")
	if len(lines) < 1 {
		return ""
	}

	start := lo.Max([]int{0, len(lines) - take})

	return strings.Join(lines[start:], "\n")
}

func (m Model) View() string {
	heightStyle := lipgloss.NewStyle().MaxHeight(m.windowSize.Height - 4)
	lv := view.New(view.WithStyle(heightStyle))
	rv := view.New(view.WithStyle(lipgloss.NewStyle().BorderForeground(tui.Colors.Gray).Border(lipgloss.NormalBorder(), false, false, false, true).PaddingLeft(1).MarginLeft(1)))

	if len(m.serviceStatus) == 0 {
		lv.Addln("No service found in project, check your nitric.yaml file contains at least one valid 'match' pattern.")
	} else {
		lv.Add("%d", len(m.serviceStatus)).WithStyle(lipgloss.NewStyle().Bold(true).Foreground(tui.Colors.Purple))
		lv.Addln(" services registered with local nitric server")
	}

	rv.Addln(fragments.Tag("logs"))

	svcColors := map[string]lipgloss.CompleteColor{}
	serviceNames := lo.Keys(m.serviceStatus)

	slices.Sort(serviceNames)

	for idx, svcName := range serviceNames {
		svcColors[svcName] = serviceColors[idx%len(serviceColors)]
	}

	for i, update := range m.serviceRunUpdates {
		statusColor := tui.Colors.Gray
		if update.Status == project.ServiceRunStatus(project.ServiceBuildStatus_Error) {
			statusColor = tui.Colors.Red
		}
		rv.Add("%s: ", update.Filepath).WithStyle(lipgloss.NewStyle().Foreground(svcColors[update.ServiceName]))
		rv.Add(strings.TrimSpace(update.Message)).WithStyle(lipgloss.NewStyle().Foreground(statusColor))
		if i < len(m.serviceRunUpdates)-1 {
			rv.Break()
		}
	}

	lv.Addln(m.localServicesModel.View())

	lv.Addln("Press 'q' to quit")

	sideBySide := lipgloss.JoinHorizontal(lipgloss.Top, lv.Render(), tail(rv.Render(), m.windowSize.Height-4))

	return lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(tui.Colors.Gray).Render(sideBySide)
}

func NewModel(stopChannel chan<- bool, updateChannel <-chan project.ServiceRunUpdate, localCloud *cloud.LocalCloud, dashboardUrl string) Model {
	localServicesModel := local.NewTuiModel(localCloud, dashboardUrl)

	return Model{
		stopChan:           stopChannel,
		localServicesModel: localServicesModel,
		updateChan:         updateChannel,
		serviceStatus:      make(map[string]project.ServiceRunUpdate),
		serviceRunUpdates:  []project.ServiceRunUpdate{},
	}
}
