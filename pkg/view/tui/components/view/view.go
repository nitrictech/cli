package view

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type View struct {
	style     lipgloss.Style
	fragments []*Fragment
	newline   string
}

// Add formats according to a format specifier and appends the resulting string to the view.
func (v *View) Add(format string, a ...any) *Fragment {
	f := NewFragment(fmt.Sprintf(format, a...))
	v.fragments = append(v.fragments, f)
	return f
}

// Break appends a new line fragment to the view.
func (v *View) Break() {
	v.fragments = append(v.fragments, NewFragment(v.newline))
}

// Addln formats according to a format specifier and appends the resulting string to the view, followed by a new line.
func (v *View) Addln(format string, a ...any) *Fragment {
	fragment := v.Add(format, a...)
	v.Break()
	return fragment
}

// Render the view as a string, applying the style.
func (v *View) Render() string {
	builder := strings.Builder{}

	for _, fragment := range v.fragments {
		builder.WriteString(fragment.Render())
	}

	return v.style.Render(builder.String())
}

// ViewOption is a function that configures a view.
type ViewOption func(*View)

// WithNewline sets a custom newline string for the view.
func WithNewline(newline string) ViewOption {
	return func(v *View) {
		v.newline = newline
	}
}

// WithNewline sets a custom newline string for the view.
func (v *View) WithNewline(newline string) *View {
	v.newline = newline
	return v
}

// WithStyle applies/updates the style of the view.
func WithStyle(style lipgloss.Style) ViewOption {
	return func(v *View) {
		v.style = style
	}
}

// WithStyle applies/updates the style of the view.
func (v *View) WithStyle(style lipgloss.Style) *View {
	v.style = style
	return v
}

// New returns a new view.
func New(opts ...ViewOption) *View {
	v := &View{
		fragments: []*Fragment{},
		style:     lipgloss.NewStyle(),
		newline:   "\n",
	}

	for _, opt := range opts {
		opt(v)
	}

	return v
}
