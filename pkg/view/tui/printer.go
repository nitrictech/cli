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

package tui

import (
	"fmt"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"

	"github.com/nitrictech/cli/pkg/view/tui/components/view"
)

var (
	Debug = TagPrinter{
		Prefix: addPrefix("debug", Colors.White, Colors.Gray),
	}
	Error = TagPrinter{
		Prefix: addPrefix("error", Colors.White, Colors.Red),
	}
	Warning = TagPrinter{
		Prefix: addPrefix("warning", Colors.Black, Colors.Yellow),
	}

	width = 0
)

type TagPrinter struct {
	Prefix string
}

func (t *TagPrinter) Println(message string) {
	fmt.Println(t.Prefix, message)
}

func (t *TagPrinter) Printfln(message string, a ...interface{}) {
	fmt.Println(t.Prefix, fmt.Sprintf(message, a...))
}

func addPrefix(text string, foreground lipgloss.CompleteColor, background lipgloss.CompleteColor) string {
	if utf8.RuneCountInString(text)+2 > width {
		width = utf8.RuneCountInString(text) + 2
	}

	tagStyle := lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Foreground(foreground).Background(background)

	f := view.NewFragment(text).WithStyle(tagStyle)

	return f.Render()
}
