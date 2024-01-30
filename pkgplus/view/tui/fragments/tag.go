package fragments

import (
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/nitrictech/cli/pkgplus/view/tui"
	"github.com/nitrictech/cli/pkgplus/view/tui/components/view"
)

var (
	width = 0
)

// CustomTag renders a tag with the given text, foreground, background and width
// e.g. CustomTag("hello", tui.Colors.White, tui.Colors.Purple, 8)
// Use Tag() for a standard tag.
func CustomTag(text string, foreground lipgloss.CompleteColor, background lipgloss.CompleteColor) string {
	if utf8.RuneCountInString(text)+2 > width {
		width = utf8.RuneCountInString(text) + 2
	}
	tagStyle := lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Foreground(foreground).Background(background)
	f := view.NewFragment(text).WithStyle(tagStyle)
	return f.Render()
}

// Tag renders a standard tag with the given title
func Tag(text string) string {
	return CustomTag(text, tui.Colors.White, tui.Colors.Purple)
}

// NitricTag renders a standard tag with the title "nitric"
func NitricTag() string {
	return CustomTag("nitric", tui.Colors.White, tui.Colors.Blue)
}

func ErrorTag() string {
	return CustomTag("error", tui.Colors.White, tui.Colors.Red)
}

// TagWidth returns the width of tags, which auto adjusts based on the longest tag rendered
func TagWidth() int {
	return width
}
