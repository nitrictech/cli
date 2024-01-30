package fragments

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/nitrictech/cli/pkgplus/view/tui"
	"github.com/nitrictech/cli/pkgplus/view/tui/components/view"
)

type ErrorListOptions struct {
	heading string
}

type ErrorListOption = func(*ErrorListOptions) *ErrorListOptions

// WithCustomHeading sets a custom heading for the error list
func WithCustomHeading(heading string) ErrorListOption {
	return func(ol *ErrorListOptions) *ErrorListOptions {
		ol.heading = heading
		return ol
	}
}

func WithoutHeading(ol *ErrorListOptions) *ErrorListOptions {
	ol.heading = ""
	return ol
}

// ErrorList renders a list of errors as a dot point list
func ErrorList(errs []error, opts ...ErrorListOption) string {
	v := view.New()

	ol := &ErrorListOptions{
		heading: lipgloss.NewStyle().Width(10).Align(lipgloss.Center).Bold(true).Foreground(tui.Colors.White).Background(tui.Colors.Red).Render("Errors"),
	}

	for _, opt := range opts {
		ol = opt(ol)
	}

	for _, err := range errs {
		v.Addln(" - %s", err.Error()).WithStyle(lipgloss.NewStyle().Foreground(tui.Colors.Red))
	}

	return v.Render()
}
