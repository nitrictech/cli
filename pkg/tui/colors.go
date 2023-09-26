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
	"reflect"

	"github.com/charmbracelet/lipgloss"

	"github.com/nitrictech/cli/pkg/tui/view"
)

// ColorPalette of standard CLI UI colors
type ColorPalette struct {
	White  lipgloss.CompleteColor
	Gray   lipgloss.CompleteColor
	Black  lipgloss.CompleteColor
	Red    lipgloss.CompleteColor
	Orange lipgloss.CompleteColor
	Yellow lipgloss.CompleteColor
	Green  lipgloss.CompleteColor
	Teal   lipgloss.CompleteColor
	Blue   lipgloss.CompleteColor
	Purple lipgloss.CompleteColor
}

// Colors contains our standard UI colors for CLI output
var Colors *ColorPalette = &ColorPalette{
	White:  lipgloss.CompleteColor{TrueColor: "#FFFFFF", ANSI256: "255", ANSI: "15"},
	Gray:   lipgloss.CompleteColor{TrueColor: "#696969", ANSI256: "250", ANSI: "7"},
	Black:  lipgloss.CompleteColor{TrueColor: "#000000", ANSI256: "16", ANSI: "0"},
	Red:    lipgloss.CompleteColor{TrueColor: "#E91E63", ANSI256: "197", ANSI: "1"},
	Orange: lipgloss.CompleteColor{TrueColor: "#F97316", ANSI256: "208", ANSI: "3"},
	Yellow: lipgloss.CompleteColor{TrueColor: "#FDE047", ANSI256: "220", ANSI: "11"},
	Green:  lipgloss.CompleteColor{TrueColor: "#22C55E", ANSI256: "47", ANSI: "10"},
	Teal:   lipgloss.CompleteColor{TrueColor: "#32D0D1", ANSI256: "51", ANSI: "14"},
	Blue:   lipgloss.CompleteColor{TrueColor: "#2563EB", ANSI256: "21", ANSI: "4"},
	Purple: lipgloss.CompleteColor{TrueColor: "#C27AFA", ANSI256: "99", ANSI: "13"},
}

// DebugColors returns the entire color palette as a string
//
// Use for testing various terminals to confirm output
func DebugColors() string {
	standardWidth := lipgloss.NewStyle().Width(8)

	colorView := view.New().WithStyle(lipgloss.NewStyle().Margin(1, 0))

	colorView.AddRow(
		view.NewFragment("Color Palette Debug").WithStyle(lipgloss.NewStyle().Bold(true)),
		view.Break(),
	)

	headerStyle := standardWidth.Copy().Bold(true)

	colorView.AddRow(
		view.NewFragment("Color").WithStyle(headerStyle),
		view.NewFragment("True").WithStyle(headerStyle),
		view.NewFragment("Light").WithStyle(headerStyle),
		view.NewFragment("Dark").WithStyle(headerStyle),
		view.NewFragment("ANSI256").WithStyle(headerStyle),
		view.NewFragment("ANSI").WithStyle(headerStyle),
	)

	v := reflect.ValueOf(*Colors)

	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		value := v.Field(i).Interface()

		switch c := value.(type) {
		case lipgloss.CompleteColor:
			back := standardWidth.Copy().Align(lipgloss.Center).Background(c)
			fore := standardWidth.Copy().Width(8).Foreground(c)

			lightOnBack := back.Copy().Foreground(Colors.White)
			darkOnBack := back.Copy().Foreground(Colors.Black)

			ANSI256 := standardWidth.Copy().Align(lipgloss.Center).Background(lipgloss.Color(c.ANSI256))
			ANSI := standardWidth.Copy().Align(lipgloss.Center).Background(lipgloss.Color(c.ANSI))

			colorView.AddRow(
				view.NewFragment(field.Name).WithStyle(fore),
				view.NewFragment(field.Name).WithStyle(back),
				view.NewFragment(field.Name).WithStyle(lightOnBack),
				view.NewFragment(field.Name).WithStyle(darkOnBack),
				view.NewFragment(field.Name).WithStyle(ANSI256),
				view.NewFragment(field.Name).WithStyle(ANSI),
			)
		}
	}

	return colorView.Render()
}
