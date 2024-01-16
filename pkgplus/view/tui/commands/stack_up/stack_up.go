package stack_up

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nitrictech/cli/pkgplus/view/tui/reactive"
	deploymentspb "github.com/nitrictech/nitric/core/pkg/proto/deployments/v1"
	"github.com/nitrictech/pearls/pkg/tui/view"
	"github.com/pterm/pterm"
	"github.com/samber/lo"
)

type Resource struct {
	name     string
	message  string
	action   string
	status   string
	children []*Resource
}

type Model struct {
	stack       *Resource
	updatesChan <-chan *deploymentspb.DeploymentUpEvent
	errorChan   <-chan error
	err         error
}

var _ tea.Model = Model{}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		reactive.AwaitChannel(m.updatesChan),
		reactive.AwaitChannel(m.errorChan),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case reactive.ChanMsg[*deploymentspb.DeploymentUpEvent]:

		// the source channel is close
		if !msg.Ok {
			return m, tea.Quit
		}

		switch content := msg.Value.Content.(type) {
		case *deploymentspb.DeploymentUpEvent_Update:

			// deets
			if content.Update == nil || content.Update.Id == nil {
				pterm.Error.Printfln("got a bad update %+v", content.Update)
				break
			}

			name := content.Update.SubResource
			if name == "" {
				name = content.Update.Id.Name
			}

			parent := m.stack
			if content.Update.SubResource != "" {
				nitricResource, found := lo.Find(m.stack.children, func(r *Resource) bool {
					return r.name == content.Update.Id.Name
				})

				if !found {
					// create it?
					m.stack.children = append(m.stack.children, &Resource{
						name:     content.Update.Id.Name,
						children: make([]*Resource, 0),
						status:   "unknown",
						action:   content.Update.Action.String(), // FIXME: this is the child's action not the parents...
						message:  "child reported before parent, this shouldn't happen?",
					})
				}

				parent = nitricResource
			}

			existingChild, found := lo.Find(parent.children, func(item *Resource) bool {
				return item.name == name
			})

			if !found {
				existingChild = &Resource{
					name:   name,
					action: content.Update.Action.String(),
				}

				parent.children = append(parent.children, existingChild)
			}

			// update its status
			existingChild.status = content.Update.Message
			existingChild.message = content.Update.Status.String()
		default:
			// discard for now
			pterm.Error.Println("unknown update type")
		}

		// let the good times roll
		return m, reactive.AwaitChannel(msg.Source)
	case reactive.ChanMsg[error]:
		m.err = msg.Value
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) View() string {
	// print the stack?
	treeView := view.New()

	treeView.AddRow(
		view.NewFragment("Nitric Up"),
		view.Break(),
	)

	if m.err != nil {
		treeView.AddRow(
			view.NewFragment("Error:"),
			view.NewFragment(m.err.Error()),
		).WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("205")))
	}

	for _, child := range m.stack.children {
		// print the child
		treeView.AddRow(
			view.NewFragment(child.name),
			view.NewFragment(" - "),
			view.NewFragment(child.action).WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("205"))),
			view.NewFragment(" - "),
			view.NewFragment(child.status),
		).WithStyle(lipgloss.NewStyle().Bold(true))
		for ix, grandchild := range child.children {

			linkChar := lo.Ternary(ix < len(child.children)-1, "├─", "└─")

			treeView.AddRow(
				view.NewFragment(linkChar),
				view.NewFragment(grandchild.name),
				view.NewFragment(" - "),
				view.NewFragment(grandchild.action).WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("205"))),
				view.NewFragment(" - "),
				view.NewFragment(grandchild.status),
			).WithStyle(lipgloss.NewStyle().MarginLeft(1))
		}
	}

	treeView.AddRow()

	return treeView.Render()
}

func New(updatesChan <-chan *deploymentspb.DeploymentUpEvent, errorChan <-chan error) Model {
	return Model{
		updatesChan: updatesChan,
		errorChan:   errorChan,
		stack: &Resource{
			name:     "stack",
			message:  "",
			children: make([]*Resource, 0),
		},
	}
}
