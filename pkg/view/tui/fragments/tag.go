// Copyright Nitric Pty Ltd.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fragments

import (
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"

	"github.com/nitrictech/cli/pkg/view/tui"
	"github.com/nitrictech/cli/pkg/view/tui/components/view"
)

var width = 0

// CustomTag renders a tag with the given text, foreground, background and width
// e.g. CustomTag("hello", tui.Colors.White, tui.Colors.Purple, 8)
// Use Tag() for a standard tag.
func CustomTag(text string, foreground lipgloss.CompleteAdaptiveColor, background lipgloss.CompleteAdaptiveColor) string {
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
