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

	"github.com/nitrictech/cli/pkg/view/tui/components/view"
)

// ColorPalette of standard CLI UI colors
type ColorPalette struct {
	White         lipgloss.CompleteAdaptiveColor
	Gray          lipgloss.CompleteAdaptiveColor
	Black         lipgloss.CompleteAdaptiveColor
	Red           lipgloss.CompleteAdaptiveColor
	Orange        lipgloss.CompleteAdaptiveColor
	Yellow        lipgloss.CompleteAdaptiveColor
	Green         lipgloss.CompleteAdaptiveColor
	Teal          lipgloss.CompleteAdaptiveColor
	Blue          lipgloss.CompleteAdaptiveColor
	Purple        lipgloss.CompleteAdaptiveColor
	Text          lipgloss.CompleteAdaptiveColor
	TextMuted     lipgloss.CompleteAdaptiveColor
	TextHighlight lipgloss.CompleteAdaptiveColor
	TextActive    lipgloss.CompleteAdaptiveColor
}

var (
	white  = lipgloss.CompleteColor{TrueColor: "#FFFFFF", ANSI256: "255", ANSI: "15"}
	gray   = lipgloss.CompleteColor{TrueColor: "#696969", ANSI256: "250", ANSI: "7"}
	black  = lipgloss.CompleteColor{TrueColor: "#000000", ANSI256: "16", ANSI: "0"}
	red    = lipgloss.CompleteColor{TrueColor: "#E91E63", ANSI256: "197", ANSI: "1"}
	orange = lipgloss.CompleteColor{TrueColor: "#F97316", ANSI256: "208", ANSI: "3"}
	yellow = lipgloss.CompleteColor{TrueColor: "#FDE047", ANSI256: "220", ANSI: "11"}
	green  = lipgloss.CompleteColor{TrueColor: "#22C55E", ANSI256: "47", ANSI: "10"}
	teal   = lipgloss.CompleteColor{TrueColor: "#32D0D1", ANSI256: "51", ANSI: "14"}
	blue   = lipgloss.CompleteColor{TrueColor: "#2563EB", ANSI256: "21", ANSI: "4"}
	purple = lipgloss.CompleteColor{TrueColor: "#C27AFA", ANSI256: "99", ANSI: "13"}
)

// Colors contains our standard UI colors for CLI output
var Colors *ColorPalette = &ColorPalette{
	White:  lipgloss.CompleteAdaptiveColor{Light: white, Dark: white},
	Gray:   lipgloss.CompleteAdaptiveColor{Light: gray, Dark: gray},
	Black:  lipgloss.CompleteAdaptiveColor{Light: black, Dark: black},
	Red:    lipgloss.CompleteAdaptiveColor{Light: red, Dark: red},
	Orange: lipgloss.CompleteAdaptiveColor{Light: orange, Dark: orange},
	Yellow: lipgloss.CompleteAdaptiveColor{Light: yellow, Dark: yellow},
	Green:  lipgloss.CompleteAdaptiveColor{Light: green, Dark: green},
	Teal:   lipgloss.CompleteAdaptiveColor{Light: teal, Dark: teal},
	Blue:   lipgloss.CompleteAdaptiveColor{Light: blue, Dark: blue},
	Purple: lipgloss.CompleteAdaptiveColor{Light: purple, Dark: purple},
	Text: lipgloss.CompleteAdaptiveColor{
		Light: black,
		Dark:  white,
	},
	TextMuted: lipgloss.CompleteAdaptiveColor{
		Light: lipgloss.CompleteColor{TrueColor: "#4E4E4E", ANSI256: "242", ANSI: "8"},
		Dark:  lipgloss.CompleteColor{TrueColor: "#9E9E9E", ANSI256: "249", ANSI: "7"},
	},
	TextHighlight: lipgloss.CompleteAdaptiveColor{
		Light: blue,
		Dark:  purple,
	},
	TextActive: lipgloss.CompleteAdaptiveColor{
		Light: blue,
		Dark:  blue,
	},
}

// DebugColors returns the entire color palette as a string
//
// Use for testing various terminals to confirm output
func DebugColors() string {
	standardWidth := lipgloss.NewStyle().Width(8)

	colorView := view.New(view.WithStyle(lipgloss.NewStyle().Margin(1, 0)))

	colorView.Addln("Color Palette Debug").WithStyle(lipgloss.NewStyle().Bold(true))
	colorView.Break()

	headerStyle := standardWidth.Bold(true)

	colorView.Add("Color").WithStyle(headerStyle)
	colorView.Add("True").WithStyle(headerStyle)
	colorView.Add("Light").WithStyle(headerStyle)
	colorView.Add("Dark").WithStyle(headerStyle)
	colorView.Add("ANSI256").WithStyle(headerStyle)
	colorView.Add("ANSI").WithStyle(headerStyle)

	v := reflect.ValueOf(*Colors)

	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		value := v.Field(i).Interface()

		switch c := value.(type) {
		case lipgloss.CompleteColor:
			back := standardWidth.Align(lipgloss.Center).Background(c)
			fore := standardWidth.Width(8).Foreground(c)

			lightOnBack := back.Foreground(Colors.White)
			darkOnBack := back.Foreground(Colors.Black)

			ANSI256 := standardWidth.Align(lipgloss.Center).Background(lipgloss.Color(c.ANSI256))
			ANSI := standardWidth.Align(lipgloss.Center).Background(lipgloss.Color(c.ANSI))

			colorView.Add(field.Name).WithStyle(fore)
			colorView.Add(field.Name).WithStyle(back)
			colorView.Add(field.Name).WithStyle(lightOnBack)
			colorView.Add(field.Name).WithStyle(darkOnBack)
			colorView.Add(field.Name).WithStyle(ANSI256)
			colorView.Add(field.Name).WithStyle(ANSI)
			colorView.Break()
		}
	}

	return colorView.Render()
}
