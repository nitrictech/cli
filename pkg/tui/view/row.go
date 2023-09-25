package view

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/samber/lo"
)

type Row struct {
	style     lipgloss.Style
	Fragments []*Fragment
}

// Add an inline fragment to this row
func (r *Row) Add(fragments ...*Fragment) {
	r.Fragments = append(r.Fragments, fragments...)
}

func (r *Row) WithStyle(style lipgloss.Style) *Row {
	r.style = style
	return r
}

// Render the row as a string
func (r *Row) Render() string {
	fragments := lo.Map(r.Fragments, func(f *Fragment, i int) string {
		return f.Render()
	})

	return r.style.Render(strings.Join(fragments, ""))
}
