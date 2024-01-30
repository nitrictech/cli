package stack_down

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pterm/pterm"
	"github.com/samber/lo"

	tui "github.com/nitrictech/cli/pkgplus/view/tui"
	"github.com/nitrictech/cli/pkgplus/view/tui/commands/stack"
	"github.com/nitrictech/cli/pkgplus/view/tui/components/view"
	"github.com/nitrictech/cli/pkgplus/view/tui/reactive"
	deploymentspb "github.com/nitrictech/nitric/core/pkg/proto/deployments/v1"
)

type Model struct {
	stack              *stack.Resource
	updatesChan        <-chan *deploymentspb.DeploymentDownEvent
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
	case reactive.ChanMsg[*deploymentspb.DeploymentDownEvent]:

		// the source channel is close
		if !msg.Ok {
			return m, tea.Quit
		}

		switch content := msg.Value.Content.(type) {
		case *deploymentspb.DeploymentDownEvent_Update:

			if content.Update == nil || content.Update.Id == nil {
				break
			}

			name := content.Update.SubResource
			if name == "" {
				name = fmt.Sprintf("%s::%s", content.Update.Id.Type.String(), content.Update.Id.Name)
			}

			parent := m.stack
			if content.Update.SubResource != "" {
				nitricResource, found := lo.Find(m.stack.Children, func(r *stack.Resource) bool {
					return r.Name == fmt.Sprintf("%s::%s", content.Update.Id.Type.String(), content.Update.Id.Name)
				})

				if !found {
					nitricResource = &stack.Resource{
						Name:     fmt.Sprintf("%s::%s", content.Update.Id.Type.String(), content.Update.Id.Name),
						Message:  "",
						Action:   content.Update.Action,
						Status:   content.Update.Status,
						Children: make([]*stack.Resource, 0),
					}

					// Add it from the given parent details
					m.stack.Children = append(m.stack.Children, nitricResource)
				}

				parent = nitricResource
			}

			existingChild, found := lo.Find(parent.Children, func(item *stack.Resource) bool {
				return item.Name == name
			})

			now := time.Now()

			if !found {
				existingChild = &stack.Resource{
					Name:      name,
					Action:    content.Update.Action,
					StartTime: now,
				}

				parent.Children = append(parent.Children, existingChild)
			}

			if content.Update.Status == deploymentspb.ResourceDeploymentStatus_FAILED || content.Update.Status == deploymentspb.ResourceDeploymentStatus_SUCCESS || content.Update.Action == deploymentspb.ResourceDeploymentAction_SAME {
				existingChild.FinishTime = now
			}

			// update its status
			existingChild.Status = content.Update.Status
			existingChild.Message = content.Update.Message
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

const maxOutputLines = 5

var (
	titleStyle          = lipgloss.NewStyle().Foreground(tui.Colors.Purple).Bold(true)
	terminalBorderStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(tui.Colors.Gray)
	errorStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
)

func (m Model) View() string {
	// print the stack?
	v := view.New()

	v.Addln("nitric down%s", m.spinner.View()).WithStyle(titleStyle)
	v.Break()

	rows := []table.Row{}

	for _, child := range m.stack.Children {
		// print the child
		rows = append(rows, table.Row{
			lipgloss.NewStyle().Bold(true).Foreground(tui.Colors.Blue).Render(child.Name),
			"", // "", verbMap[child.action][child.status],
		})

		for ix, grandchild := range child.Children {
			linkChar := lo.Ternary(ix < len(child.Children)-1, "├─", "└─")

			resourceTime := lo.Ternary(grandchild.FinishTime.IsZero(), time.Since(grandchild.StartTime).Round(time.Second), grandchild.FinishTime.Sub(grandchild.StartTime))

			rows = append(rows, table.Row{
				lipgloss.NewStyle().MarginLeft(1).Foreground(tui.Colors.Blue).Render(linkChar) + lipgloss.NewStyle().Foreground(tui.Colors.Gray).Render(grandchild.Name),
				stack.VerbMap[grandchild.Action][grandchild.Status] + fmt.Sprintf(" (%s)", resourceTime.Round(time.Second)),
			})
		}
	}
	m.resourcesTable.SetRows(rows)

	v.Addln(m.resourcesTable.View())

	// Provider Stdout and Stderr rendering
	if len(m.providerStdout) > 0 {
		v.Addln("Provider Output:").WithStyle(lipgloss.NewStyle().Foreground(tui.Colors.Gray))

		providerTerm := view.New(view.WithStyle(terminalBorderStyle))

		for _, line := range m.providerStdout[max(0, len(m.providerStdout)-maxOutputLines):] {
			providerTerm.Addln(line).WithStyle(lipgloss.NewStyle().Width(98))
		}

		v.Addln(providerTerm.Render())
		v.Break()
	}

	for _, e := range m.errs[max(0, len(m.errs)-maxOutputLines):] {
		v.Addln("Error: %s", e.Error()).WithStyle(errorStyle)
	}

	return v.Render()
}

func New(updatesChan <-chan *deploymentspb.DeploymentDownEvent, providerStdoutChan <-chan string, errorChan <-chan error) Model {
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
		stack: &stack.Resource{
			Name:     "stack",
			Message:  "",
			Children: make([]*stack.Resource, 0),
		},
	}
}
