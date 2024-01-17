package stack_up

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nitrictech/cli/pkgplus/view/tui/reactive"
	deploymentspb "github.com/nitrictech/nitric/core/pkg/proto/deployments/v1"
	"github.com/nitrictech/pearls/pkg/tui/view"
	"github.com/pterm/pterm"
	"github.com/samber/lo"
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
	stack       *Resource
	updatesChan <-chan *deploymentspb.DeploymentUpEvent
	errorChan   <-chan error
	err         error
	allMsgs     []string

	spinner        spinner.Model
	resourcesTable table.Model
}

var _ tea.Model = Model{}

func (m Model) Init() tea.Cmd {
	m.allMsgs = make([]string, 0)
	return tea.Batch(
		m.spinner.Tick,
		reactive.AwaitChannel(m.updatesChan),
		reactive.AwaitChannel(m.errorChan),
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
	case reactive.ChanMsg[*deploymentspb.DeploymentUpEvent]:

		// the source channel is close
		if !msg.Ok {
			return m, nil
		}

		switch content := msg.Value.Content.(type) {
		case *deploymentspb.DeploymentUpEvent_Update:

			if content.Update == nil || content.Update.Id == nil {
				break
			}

			m.allMsgs = append(m.allMsgs, fmt.Sprintf("%s -:- %+v", msg.Value.GetUpdate().Id.Type.String(), msg.Value))

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
					// // create it?
					// m.stack.children = append(m.stack.children, &Resource{
					// 	name:      fmt.Sprintf("%s::%s", content.Update.Id.Type.String(), content.Update.Id.Name),
					// 	children:  make([]*Resource, 0),
					// 	startTime: time.Now(),
					// 	status:    content.Update.Status, // Unknown what parent's status is.
					// 	action:    content.Update.Action, // FIXME: this is the child's action not the parents...
					// 	message:   "child reported before parent, this shouldn't happen?",
					// })
					m.err = fmt.Errorf("child reported before parent, this shouldn't happen?")
					return m, nil
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
		m.err = msg.Value
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

func (m Model) View() string {
	// print the stack?
	treeView := view.New()

	treeView.AddRow(
		view.NewFragment("Nitric Up"+m.spinner.View()),
		view.Break(),
	)

	for _, msg := range m.allMsgs {
		treeView.AddRow(
			view.NewFragment(msg),
		)
	}

	// table.New(table.WithKeyMap(
	// 	table.DefaultKeyMap(),
	// ))

	rows := []table.Row{}

	for _, child := range m.stack.children {
		// print the child
		rows = append(rows, table.Row{
			lipgloss.NewStyle().Bold(true).Render(child.name),
			"", // "", verbMap[child.action][child.status],
		})

		for ix, grandchild := range child.children {
			linkChar := lo.Ternary(ix < len(child.children)-1, "├─", "└─")

			resourceTime := lo.Ternary(grandchild.finishTime.IsZero(), time.Since(grandchild.startTime).Round(time.Second), grandchild.finishTime.Sub(grandchild.startTime))

			rows = append(rows, table.Row{
				lipgloss.NewStyle().MarginLeft(1).Render(linkChar + grandchild.name),
				verbMap[grandchild.action][grandchild.status] + fmt.Sprintf(" (%s)", resourceTime.Round(time.Second)),
			})
		}
	}
	m.resourcesTable.SetRows(rows)

	treeView.AddRow(view.NewFragment(m.resourcesTable.View()))

	if m.err != nil {
		treeView.AddRow(
			view.NewFragment("Error:"),
			view.NewFragment(m.err.Error()),
		).WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("205")))
	}

	return treeView.Render()
}

func New(updatesChan <-chan *deploymentspb.DeploymentUpEvent, errorChan <-chan error) Model {
	return Model{
		resourcesTable: table.New(
			table.WithColumns([]table.Column{
				{
					Title: "name",
					Width: 50,
				},
				{
					Title: "status",
					Width: 20,
				},
			}),
			table.WithHeight(20),
		),
		spinner:     spinner.New(spinner.WithSpinner(spinner.Ellipsis)),
		updatesChan: updatesChan,
		errorChan:   errorChan,
		stack: &Resource{
			name:     "stack",
			message:  "",
			children: make([]*Resource, 0),
		},
	}
}
