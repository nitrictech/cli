package localservices

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/nitrictech/pearls/pkg/tui"
)

func createTable(columns []table.Column, rows []table.Row) table.Model {
	headerStyle := lipgloss.NewStyle().Bold(true)
	headers := []table.Column{}

	for _, column := range columns {
		headers = append(headers, table.Column{Title: headerStyle.Render(column.Title), Width: column.Width})
	}

	t := table.New(
		table.WithColumns(headers),
		table.WithRows(rows),
		table.WithFocused(false),
		table.WithHeight(len(rows)+1),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(tui.Colors.White).
		BorderBottom(true).
		Bold(true)
	s.Selected = lipgloss.NewStyle()
	t.SetStyles(s)

	return t
}
