package stack_up

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pterm/pterm"
	"github.com/samber/lo"

	"github.com/nitrictech/cli/pkgplus/view/tui/reactive"
	deploymentspb "github.com/nitrictech/nitric/core/pkg/proto/deployments/v1"
	"github.com/nitrictech/pearls/pkg/tui"
	"github.com/nitrictech/pearls/pkg/tui/view"
)

type Resource struct {
	name       string
	message    string
	action     deploymentspb.ResourceDeploymentAction
	status     deploymentspb.ResourceDeploymentStatus
	startTime  time.Time
	finishTime time.Time
	children   []*Resource
}

type Model struct {
	stack              *Resource
	updatesChan        <-chan *deploymentspb.DeploymentUpEvent
	errorChan          <-chan error
	providerStdoutChan <-chan string
	providerStdout     []string
	errs               []error

	spinner        spinner.Model
	resourcesTable table.Model
}

var _ tea.Model = Model{}

func (m Model) Init() tea.Cmd {
	m.errs = make([]error, 0)
	return tea.Batch(
		m.spinner.Tick,
		reactive.AwaitChannel(m.updatesChan),
		reactive.AwaitChannel(m.errorChan),
		reactive.AwaitChannel(m.providerStdoutChan),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		default:
			m.resourcesTable, cmd = m.resourcesTable.Update(msg)
			return m, cmd
		}

	case reactive.ChanMsg[string]:
		if !msg.Ok {
			break
		}

		m.providerStdout = append(m.providerStdout, msg.Value)

		return m, reactive.AwaitChannel(msg.Source)
	case reactive.ChanMsg[*deploymentspb.DeploymentUpEvent]:

		// the source channel is close
		if !msg.Ok {
			return m, tea.Quit
		}

		switch content := msg.Value.Content.(type) {
		case *deploymentspb.DeploymentUpEvent_Update:

			if content.Update == nil || content.Update.Id == nil {
				break
			}

			name := content.Update.SubResource
			if name == "" {
				name = fmt.Sprintf("%s::%s", content.Update.Id.Type.String(), content.Update.Id.Name)
			}

			parent := m.stack
			if content.Update.SubResource != "" {
				nitricResource, found := lo.Find(m.stack.children, func(r *Resource) bool {
					return r.name == fmt.Sprintf("%s::%s", content.Update.Id.Type.String(), content.Update.Id.Name)
				})

				if !found {
					m.errs = append(m.errs, fmt.Errorf("received update for resource [%s], without associated nitric parent resource", content.Update.SubResource))
					return m, tea.Quit
				}

				parent = nitricResource
			}

			existingChild, found := lo.Find(parent.children, func(item *Resource) bool {
				return item.name == name
			})

			now := time.Now()

			if !found {
				existingChild = &Resource{
					name:      name,
					action:    content.Update.Action,
					startTime: now,
				}

				parent.children = append(parent.children, existingChild)
			}

			if content.Update.Status == deploymentspb.ResourceDeploymentStatus_FAILED || content.Update.Status == deploymentspb.ResourceDeploymentStatus_SUCCESS || content.Update.Action == deploymentspb.ResourceDeploymentAction_SAME {
				existingChild.finishTime = now
			}

			// update its status
			existingChild.status = content.Update.Status
			existingChild.message = content.Update.Message
		default:
			// discard for now
			pterm.Error.Println("unknown update type")
		}

		return m, reactive.AwaitChannel(msg.Source)
	case reactive.ChanMsg[error]:
		m.errs = append(m.errs, msg.Value)
		return m, nil
	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
	default:
		m.resourcesTable, cmd = m.resourcesTable.Update(msg)
	}
	return m, cmd
}

var verbMap = map[deploymentspb.ResourceDeploymentAction]map[deploymentspb.ResourceDeploymentStatus]string{
	deploymentspb.ResourceDeploymentAction_CREATE: {
		deploymentspb.ResourceDeploymentStatus_PENDING:     "create",
		deploymentspb.ResourceDeploymentStatus_IN_PROGRESS: "creating",
		deploymentspb.ResourceDeploymentStatus_FAILED:      "creation failed",
		deploymentspb.ResourceDeploymentStatus_SUCCESS:     "created",
	},
	deploymentspb.ResourceDeploymentAction_DELETE: {
		deploymentspb.ResourceDeploymentStatus_PENDING:     "delete",
		deploymentspb.ResourceDeploymentStatus_SUCCESS:     "deleted",
		deploymentspb.ResourceDeploymentStatus_IN_PROGRESS: "deleting",
		deploymentspb.ResourceDeploymentStatus_FAILED:      "failed to delete",
	},
	deploymentspb.ResourceDeploymentAction_REPLACE: {
		deploymentspb.ResourceDeploymentStatus_PENDING:     "replace",
		deploymentspb.ResourceDeploymentStatus_SUCCESS:     "replaced",
		deploymentspb.ResourceDeploymentStatus_IN_PROGRESS: "replacing",
		deploymentspb.ResourceDeploymentStatus_FAILED:      "failed to replace",
	},
	deploymentspb.ResourceDeploymentAction_UPDATE: {
		deploymentspb.ResourceDeploymentStatus_PENDING:     "update",
		deploymentspb.ResourceDeploymentStatus_SUCCESS:     "updated",
		deploymentspb.ResourceDeploymentStatus_IN_PROGRESS: "updating",
		deploymentspb.ResourceDeploymentStatus_FAILED:      "failed to update",
	},
	deploymentspb.ResourceDeploymentAction_SAME: {
		deploymentspb.ResourceDeploymentStatus_PENDING:     "unchanged",
		deploymentspb.ResourceDeploymentStatus_SUCCESS:     "unchanged",
		deploymentspb.ResourceDeploymentStatus_IN_PROGRESS: "unchanged",
		deploymentspb.ResourceDeploymentStatus_FAILED:      "unchanged",
	},
}

const maxOutputLines = 5

func (m Model) View() string {
	// print the stack?
	treeView := view.New()

	treeView.AddRow(
		view.NewFragment("Nitric Up"+m.spinner.View()).WithStyle(lipgloss.NewStyle().Foreground(tui.Colors.Purple).Bold(true)),
		view.Break(),
	)

	rows := []table.Row{}

	for _, child := range m.stack.children {
		// print the child
		rows = append(rows, table.Row{
			lipgloss.NewStyle().Bold(true).Foreground(tui.Colors.Blue).Render(child.name),
			"", // "", verbMap[child.action][child.status],
		})

		for ix, grandchild := range child.children {
			linkChar := lo.Ternary(ix < len(child.children)-1, "├─", "└─")

			resourceTime := lo.Ternary(grandchild.finishTime.IsZero(), time.Since(grandchild.startTime).Round(time.Second), grandchild.finishTime.Sub(grandchild.startTime))

			rows = append(rows, table.Row{
				lipgloss.NewStyle().MarginLeft(1).Foreground(tui.Colors.Blue).Render(linkChar) + lipgloss.NewStyle().Foreground(tui.Colors.Gray).Render(grandchild.name),
				verbMap[grandchild.action][grandchild.status] + fmt.Sprintf(" (%s)", resourceTime.Round(time.Second)),
			})
		}
	}
	m.resourcesTable.SetRows(rows)

	treeView.AddRow(view.NewFragment(m.resourcesTable.View()))

	// Provider Stdout and Stderr rendering
	if len(m.providerStdout) > 0 {
		tealForTim := lipgloss.NewStyle().Foreground(tui.Colors.Gray)

		providerTermView := view.New()

		treeView.AddRow(
			view.NewFragment("Provider Output:").WithStyle(tealForTim),
		)

		for _, line := range m.providerStdout[max(0, len(m.providerStdout)-maxOutputLines):] {
			providerTermView.AddRow(
				view.NewFragment(line).WithStyle(lipgloss.NewStyle().Width(98)),
			)
		}

		borderStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(tui.Colors.Gray)

		treeView.AddRow(
			view.NewFragment(providerTermView.Render()).WithStyle(borderStyle),
			view.Break(),
		)
	}

	for _, e := range m.errs[max(0, len(m.errs)-maxOutputLines):] {
		treeView.AddRow(
			view.NewFragment("Error:"),
			view.NewFragment(e.Error()),
		).WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("205")))
	}

	return treeView.Render()
}

func New(updatesChan <-chan *deploymentspb.DeploymentUpEvent, providerStdoutChan <-chan string, errorChan <-chan error) Model {
	return Model{
		resourcesTable: table.New(
			table.WithColumns([]table.Column{
				{
					Title: "Name",
					Width: 80,
				},
				{
					Title: "Status",
					Width: 20,
				},
			}),
			table.WithStyles(table.Styles{
				Selected: table.DefaultStyles().Cell,
				Header:   table.DefaultStyles().Header,
				Cell:     table.DefaultStyles().Cell,
			}),
		),
		spinner:            spinner.New(spinner.WithSpinner(spinner.Ellipsis)),
		updatesChan:        updatesChan,
		providerStdoutChan: providerStdoutChan,
		errorChan:          errorChan,
		stack: &Resource{
			name:     "stack",
			message:  "",
			children: make([]*Resource, 0),
		},
	}
}
