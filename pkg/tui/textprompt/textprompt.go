package textprompt

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nitrictech/cli/pkg/tui"
)

type (
	errMsg error
)

type Model struct {
	textInput textinput.Model
	Prompt    string
	Tag       string
	Validate  ValidateFunc
	focus     bool
	complete  bool
	previous  string

	err error
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, tui.KeyMap.Quit):
			return m, tea.Quit
		case key.Matches(msg, tui.KeyMap.Enter):
			if m.textInput.Value() == "" {
				m.textInput.SetValue(m.textInput.Placeholder)
			}
			m.err = m.Validate(m.textInput.Value(), false)
			if m.err == nil {
				m.Complete()
			}
		}

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	m.textInput, cmd = m.textInput.Update(msg)

	// only clear/update error messages if the input has changed
	if m.previous != m.textInput.Value() {
		if m.textInput.Value() != "" {
			m.err = m.Validate(m.textInput.Value(), true)
		} else {
			m.err = nil
		}
	}
	m.previous = m.textInput.Value()

	return m, cmd
}

var (
	labelStyle  = lipgloss.NewStyle().MarginTop(1)
	tagStyle    = lipgloss.NewStyle().Background(tui.Colors.Purple).Foreground(tui.Colors.White).Width(8).Align(lipgloss.Center)
	promptStyle = lipgloss.NewStyle().MarginLeft(2)
	inputStyle  = lipgloss.NewStyle().MarginLeft(10)
	textStyle   = lipgloss.NewStyle().Foreground(tui.Colors.Gray).MarginLeft(10)
	errorStyle  = lipgloss.NewStyle().Foreground(tui.Colors.Red).Margin(1, 0, 0, 10).Italic(true)
)

func (m Model) View() string {
	var view strings.Builder

	// Label
	tag := tagStyle.Render(m.Tag)
	prompt := promptStyle.Render(m.Prompt)
	view.WriteString(labelStyle.Render(fmt.Sprintf("%s%s", tag, prompt)))

	view.WriteString("\n\n")

	// Input/Text
	if m.complete {
		view.WriteString(textStyle.Render(m.textInput.Value()))
	} else {
		view.WriteString(inputStyle.Render(m.textInput.View()))
	}

	// Error
	if m.err != nil {
		view.WriteString(errorStyle.Render(m.err.Error()))
	}

	view.WriteString("\n")
	return view.String()
}

// Focus sets the focus state on the model. When the model is in focus it can
// receive keyboard input and the cursor will be shown.
func (m *Model) Focus() tea.Cmd {
	m.focus = true
	return m.textInput.Focus()
}

// Blur removes the focus state on the model.  When the model is blurred it can
// not receive keyboard input and the cursor will be hidden.
func (m *Model) Blur() {
	m.focus = false
	m.textInput.Blur()
}

func (m *Model) Complete() {
	m.complete = true
	m.Blur()
}

func (m Model) IsComplete() bool {
	return m.complete
}

func (m Model) Value() string {
	return m.textInput.Value()
}

// ValidateFunc is a function that returns an error if the input is invalid.
type ValidateFunc func(string, bool) error

type TextPromptArgs struct {
	Placeholder string
	Validate    ValidateFunc
	Prompt      string
	Tag         string
}

func NewTextPrompt(args TextPromptArgs) *Model {

	ti := textinput.New()
	ti.CharLimit = 156
	ti.Width = 20
	ti.Placeholder = args.Placeholder

	return &Model{
		textInput: ti,
		complete:  false,
		Prompt:    args.Prompt,
		Tag:       args.Tag,
		Validate:  args.Validate,
		err:       nil,
	}
}
