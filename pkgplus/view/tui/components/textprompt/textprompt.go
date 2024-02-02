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

package textprompt

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	tui "github.com/nitrictech/cli/pkgplus/view/tui"
	"github.com/nitrictech/cli/pkgplus/view/tui/components/validation"
	"github.com/nitrictech/cli/pkgplus/view/tui/components/view"
	"github.com/nitrictech/cli/pkgplus/view/tui/teax"
)

type (
	errMsg error
)

type TextPrompt struct {
	ID               string
	textInput        textinput.Model
	Prompt           string
	Tag              string
	validate         validation.StringValidator
	validateInFlight validation.StringValidator
	focus            bool
	previous         string

	err error
}

func (m TextPrompt) Init() tea.Cmd {
	return textinput.Blink
}

type CompleteMsg struct {
	ID    string
	Value string
}

func (m *TextPrompt) submit() tea.Cmd {
	return func() tea.Msg {
		return CompleteMsg{
			ID:    m.ID,
			Value: m.textInput.Value(),
		}
	}
}

func (m TextPrompt) UpdateTextPrompt(msg tea.Msg) (TextPrompt, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, tui.KeyMap.Quit):
			return m, teax.Quit
		case key.Matches(msg, tui.KeyMap.Enter):
			if m.textInput.Value() == "" {
				m.textInput.SetValue(m.textInput.Placeholder)
			}

			m.err = m.validate(m.textInput.Value())

			if m.err == nil {
				m.textInput.Blur()
				return m, m.submit()
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
			m.err = m.validateInFlight(m.textInput.Value())
		} else {
			m.err = nil
		}
	}

	m.previous = m.textInput.Value()

	return m, cmd
}

func (m TextPrompt) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m.UpdateTextPrompt(msg)
}

var (
	tagStyle        = lipgloss.NewStyle().Background(tui.Colors.Purple).Foreground(tui.Colors.White).Width(8).Align(lipgloss.Center)
	promptStyle     = lipgloss.NewStyle().MarginLeft(2)
	shiftRightStyle = lipgloss.NewStyle().MarginLeft(10)
	textStyle       = lipgloss.NewStyle().Foreground(tui.Colors.Gray)
	errorStyle      = lipgloss.NewStyle().Foreground(tui.Colors.Red).Italic(true).MarginTop(1)
)

func (m TextPrompt) View() string {
	v := view.New()

	v.Add(m.Tag).WithStyle(tagStyle, lipgloss.NewStyle().MarginTop(1))
	v.Addln(m.Prompt).WithStyle(promptStyle)
	v.Break()

	field := view.New(view.WithStyle(shiftRightStyle))

	if m.textInput.Focused() {
		field.Addln(m.textInput.View())
	} else {
		field.Addln(m.textInput.Value()).WithStyle(textStyle)
	}

	if m.err != nil {
		field.Addln(m.err.Error()).WithStyle(errorStyle)
	}

	v.Add(field.Render())

	return v.Render()
}

// Focus sets the focus state on the model. When the model is in focus it can
// receive keyboard input and the cursor will be shown.
func (m *TextPrompt) Focus() tea.Cmd {
	m.focus = true
	return m.textInput.Focus()
}

// Blur removes the focus state on the model.  When the model is blurred it can
// not receive keyboard input and the cursor will be hidden.
func (m *TextPrompt) Blur() {
	m.focus = false
	m.textInput.Blur()
}

func (m *TextPrompt) SetValue(value string) {
	m.textInput.SetValue(value)
}

func (m TextPrompt) Value() string {
	return m.textInput.Value()
}

type TextPromptArgs struct {
	ID                string
	Placeholder       string
	Validator         validation.StringValidator
	InFlightValidator validation.StringValidator
	Prompt            string
	Tag               string
}

func NewTextPrompt(id string, args TextPromptArgs) TextPrompt {
	ti := textinput.New()
	ti.CharLimit = 156
	ti.Width = 20
	ti.Placeholder = args.Placeholder

	return TextPrompt{
		ID:               id,
		textInput:        ti,
		Prompt:           args.Prompt,
		Tag:              args.Tag,
		validate:         args.Validator,
		validateInFlight: args.InFlightValidator,
		err:              nil,
	}
}
