package tui

import "github.com/charmbracelet/lipgloss"

type ColorPallet struct {
	White  lipgloss.CompleteColor
	Gray   lipgloss.CompleteColor
	Purple lipgloss.CompleteColor
	Blue   lipgloss.CompleteColor
	Red    lipgloss.CompleteColor
}

var (
	Colors *ColorPallet = &ColorPallet{
		White:  lipgloss.CompleteColor{TrueColor: "#FFFFFF", ANSI256: "255", ANSI: "15"},
		Gray:   lipgloss.CompleteColor{TrueColor: "#696969", ANSI256: "250", ANSI: "7"},
		Purple: lipgloss.CompleteColor{TrueColor: "#C27AFA", ANSI256: "99", ANSI: "13"},
		Blue:   lipgloss.CompleteColor{TrueColor: "#2C40F7", ANSI256: "21", ANSI: "4"},
		Red:    lipgloss.CompleteColor{TrueColor: "#E91E63", ANSI256: "197", ANSI: "1"},
	}
)
