package view

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

type Fragment struct {
	content any
	style   lipgloss.Style
}

// Render this fragment as a string, applying its style
func (f Fragment) Render() string {
	return f.style.Render(fmt.Sprint(f.content))
}

func (f Fragment) String() string {
	return f.Render()
}

// WithStyle adds a style to this fragment, which will be used when rendering
func (f *Fragment) WithStyle(style lipgloss.Style) *Fragment {
	f.style = style
	return f
}

func NewFragment(content any) *Fragment {
	return &Fragment{
		content: content,
	}
}
