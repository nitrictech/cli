package fragments

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/nitrictech/cli/pkgplus/view/tui"
	"github.com/nitrictech/cli/pkgplus/view/tui/components/view"
)

// StatusNode assists with rendering a tree of nodes, each with an optional status
// e.g.
// Api::main
//
//	├─aws:apigatewayv2/api:Api::main                               updated (3s)
//	├─aws:lambda/permission:Permission::maintwilight-sun_services- unchanged (0s)
//	│   hello
//	└─aws:apigatewayv2/stage:Stage::mainDefaultStage               unchanged (0s)
//
// KeyValueStore::cache
//
//	└─aws:dynamodb/table:Table::cache                              unchanged (0s)
type StatusNode struct {
	name     string
	status   string
	children []*StatusNode
}

const (
	indent      = 2
	linkWidth   = 2
	node        = "├─"
	wrapped     = "│ "
	lastNode    = "└─"
	lastWrapped = "  "
)

func wrap(text string, width int, indent int) []string {
	if width <= 0 {
		width = 1
	}

	parts := strings.Split(lipgloss.NewStyle().Width(width).Render(text), "\n")
	if indent == 0 || len(parts) <= 1 {
		return parts
	}
	first := parts[0]
	rest := strings.Join(parts[1:], "")
	indented := strings.Split(lipgloss.NewStyle().Width(width-indent).Render(rest), "\n")
	for i, line := range indented {
		indented[i] = strings.Repeat(" ", indent) + line
	}
	return append([]string{first}, indented...)
}

func (n StatusNode) Name() string {
	return n.name
}

func (n StatusNode) Status() string {
	return n.status
}

func (n StatusNode) Children() []*StatusNode {
	return n.children
}

// Render this node as a tree
// maxNameWidth sets the maximum width of the names of nodes in the tree.
// the total width is maxNameWidth + 1 + maxStatusWidth
func (n StatusNode) Render(maxWidth int) string {
	return n.render(true, maxWidth, 0)
}

const statusWidth = 20

var linkStyle = lipgloss.NewStyle().MarginLeft(1).Foreground(tui.Colors.Blue)

func (n StatusNode) render(omitSelf bool, width int, depth int) string {
	addDepth := 1
	if omitSelf {
		addDepth = 0
	}

	v := view.New()

	if !omitSelf {
		nameWidth := width - statusWidth
		nameParts := wrap(n.name, nameWidth-((depth-1)*(indent+linkWidth)), indent)
		nameStyle := lipgloss.NewStyle().Foreground(tui.Colors.Gray)
		if depth == 0 {
			nameStyle = lipgloss.NewStyle().Foreground(tui.Colors.Blue)
		}
		for i, namePart := range nameParts {
			v.Add(namePart).WithStyle(nameStyle)
			if i == 0 {
				v.Add(" ").WithStyle(nameStyle)
				v.Addln(n.status).WithStyle(nameStyle.Copy().Width(statusWidth))
			} else {
				v.Break()
			}
		}
	}

	for i, child := range n.children {
		last := i == len(n.children)-1
		childParts := strings.Split(strings.TrimSpace(child.render(false, width, depth+addDepth)), "\n")
		for ic, childPart := range childParts {
			if !omitSelf {
				if last {
					if ic == 0 {
						v.Add(lastNode).WithStyle(linkStyle)
					} else {
						v.Add(lastWrapped).WithStyle(linkStyle)
					}
				} else {
					if ic == 0 {
						v.Add(node).WithStyle(linkStyle)
					} else {
						v.Add(wrapped).WithStyle(linkStyle)
					}
				}
			}

			v.Addln(childPart)
		}
	}

	return v.Render()
}

func NewStatusNode(name string, status string) *StatusNode {
	return &StatusNode{
		name:     strings.ReplaceAll(name, "\n", ""),
		status:   status,
		children: make([]*StatusNode, 0),
	}
}

func (s *StatusNode) AddNode(name string, status string) *StatusNode {
	child := NewStatusNode(name, status)
	s.children = append(s.children, child)
	return child
}
