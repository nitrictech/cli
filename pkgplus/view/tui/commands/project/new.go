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

package project

import (
	"fmt"
	"path"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/spf13/afero"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/goombaio/namegenerator"

	"github.com/nitrictech/cli/pkgplus/project"
	"github.com/nitrictech/cli/pkgplus/project/templates"
	tui "github.com/nitrictech/cli/pkgplus/view/tui"
	"github.com/nitrictech/cli/pkgplus/view/tui/components/list"
	"github.com/nitrictech/cli/pkgplus/view/tui/components/listprompt"
	"github.com/nitrictech/cli/pkgplus/view/tui/components/textprompt"
	"github.com/nitrictech/cli/pkgplus/view/tui/components/validation"
	"github.com/nitrictech/cli/pkgplus/view/tui/components/view"
	"github.com/nitrictech/cli/pkgplus/view/tui/teax"
)

type (
	errMsg error
)

type NewProjectStatus int

const (
	NameInput NewProjectStatus = iota
	TemplateInput
	Pending
	Done
	Error
)

// Model - represents the state of the new project creation operation
type Model struct {
	namePrompt     textprompt.TextPrompt
	templatePrompt listprompt.ListPrompt
	spinner        spinner.Model
	status         NewProjectStatus
	nonInteractive bool

	downloader templates.Downloader

	fs afero.Fs

	err error
}

// ProjectName - returns the project name entered by the user
func (m Model) ProjectName() string {
	return m.namePrompt.Value()
}

// TemplateName returns the project template name selected by the user
func (m Model) TemplateName() string {
	template := m.downloader.GetByLabel(m.templatePrompt.Choice())

	return template.Name
}

// Init initializes the model, used by Bubbletea
func (m Model) Init() tea.Cmd {
	if m.err != nil {
		return teax.Quit
	}

	if m.nonInteractive {
		return tea.Batch(m.spinner.Tick, m.createProject(m.fs))
	}

	return tea.Batch(tea.ClearScreen, m.namePrompt.Init(), m.templatePrompt.Init())
}

// Update the model based on a message
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, teax.Quit
		}

	case projectCreateResultMsg:
		if msg.err == nil {
			m.status = Done
		} else {
			m.status = Error
			m.err = msg.err
		}

		return m, nil

	case errMsg:
		m.err = msg
		return m, teax.Quit
	case textprompt.CompleteMsg:
		if msg.ID == m.namePrompt.ID {
			m.namePrompt.Blur()
			m.status = TemplateInput
		}

		return m, nil
	}

	// Deal with the various steps in the process from data capture to building the project
	switch m.status {
	case NameInput:
		m.namePrompt, cmd = m.namePrompt.UpdateTextPrompt(msg)
	case TemplateInput:
		m.templatePrompt, cmd = m.templatePrompt.UpdateListPrompt(msg)
		if m.templatePrompt.Choice() != "" {
			m.status = Pending
			return m, tea.Batch(m.spinner.Tick, m.createProject(m.fs))
		}
	case Pending:
		m.spinner, cmd = m.spinner.Update(msg)
	case Done:
		return m, teax.Quit
	}

	return m, cmd
}

var (
	titleStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(tui.Colors.White).
			Background(tui.Colors.Blue).
			MarginRight(2)
	spinnerStyle   = lipgloss.NewStyle().Foreground(tui.Colors.Purple)
	highlightStyle = lipgloss.NewStyle().Foreground(tui.Colors.Purple)
)

var (
	errorTagStyle           = lipgloss.NewStyle().Background(tui.Colors.Red).Foreground(tui.Colors.White).PaddingLeft(2).PaddingRight(2).Align(lipgloss.Center)
	errorTextStyle          = lipgloss.NewStyle().PaddingLeft(2).Foreground(tui.Colors.Red)
	tagStyle                = lipgloss.NewStyle().Width(8).Background(tui.Colors.Purple).Foreground(tui.Colors.White).Align(lipgloss.Center)
	projCreatedHeadingStyle = lipgloss.NewStyle().Bold(true).MarginLeft(2)
)

func (m Model) View() string {
	v := view.New()

	if m.err != nil {
		v.Add("error").WithStyle(errorTagStyle)
		v.Addln(m.err.Error()).WithStyle(errorTextStyle)

		// TODO: this shouldn't be needed but without it the line doesn't print
		v.Break()

		return v.Render()
	}

	if !m.nonInteractive {
		v.Add("nitric").WithStyle(titleStyle)
		v.Addln("Let's get going!")
		v.Break()

		v.Addln(m.namePrompt.View())

		// Template selection input
		if m.status >= TemplateInput {
			v.Addln(m.templatePrompt.View())
		}
	}

	// Creating Status
	if m.status == Pending {
		v.Break()
		v.Add("proj").WithStyle(tagStyle)
		v.Add(m.spinner.View()).WithStyle(lipgloss.NewStyle().MarginLeft(2))
		v.Addln(" creating project...")
		v.Break()
	}

	// Done!
	if m.status == Done {
		v.Break()
		v.Add("proj").WithStyle(tagStyle)
		v.Addln("Project Created!").WithStyle(projCreatedHeadingStyle)
		v.Break()

		indent := view.New(view.WithStyle(lipgloss.NewStyle().MarginLeft(10)))

		indent.Add("Navigate to your project with ")
		indent.Addln("cd ./%s", m.ProjectName()).WithStyle(highlightStyle)

		indent.Addln("Install dependencies and you're ready to rock! 🪨")
		indent.Break()

		indent.Add("Need help? Come and chat ")
		indent.Addln("https://nitric.io/chat").WithStyle(highlightStyle)

		v.Addln(indent.Render())
	} else {
		v.Break()
		v.Addln("(esc to quit)").WithStyle(lipgloss.NewStyle().Foreground(tui.Colors.Gray))
	}

	return v.Render()
}

type Args struct {
	ProjectName  string
	TemplateName string
}

type TemplateItem struct {
	Value       string
	Description string
}

func (m *TemplateItem) GetItemValue() string {
	return m.Value
}

func (m *TemplateItem) GetItemDescription() string {
	return ""
}

func New(fs afero.Fs, args Args) (Model, error) {
	seed := time.Now().UTC().UnixNano()
	nameGenerator := namegenerator.NewNameGenerator(seed)
	placeholderName := nameGenerator.Generate()

	nameValidator := validation.ComposeValidators(projectNameValidators...)
	nameInFlightValidator := validation.ComposeValidators(projectNameInFlightValidators...)

	namePrompt := textprompt.NewTextPrompt("projectName", textprompt.TextPromptArgs{
		Prompt:            "What should we name this project?",
		Tag:               "name",
		Placeholder:       placeholderName,
		Validator:         nameValidator,
		InFlightValidator: nameInFlightValidator,
	})
	namePrompt.Focus()

	downloadr := templates.NewDownloader()
	templates, err := downloadr.Templates()
	if err != nil {
		return Model{}, err
	}

	templateItems := []list.ListItem{}

	for _, template := range templates {
		templateItems = append(templateItems, &TemplateItem{Value: template.Label})
	}

	templatePrompt := listprompt.NewListPrompt(listprompt.ListPromptArgs{
		Prompt:            "Which template should we start with?",
		Tag:               "tmpl",
		Items:             templateItems,
		MaxDisplayedItems: len(templates),
	})

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle

	// prefill values from CLI args
	if args.ProjectName != "" {
		if err := nameValidator(args.ProjectName); err != nil {
			return Model{
				err: err,
			}, nil
		}

		namePrompt.SetValue(args.ProjectName)
	}

	if args.TemplateName != "" {
		template := downloadr.Get(args.TemplateName)
		if template == nil {
			return Model{
				err: fmt.Errorf("template \"%s\" could not be found", args.TemplateName),
			}, nil
		}

		templatePrompt.SetChoice(template.Label)
	}

	isNonInteractive := false
	projectStatus := NameInput

	if args.ProjectName != "" {
		projectStatus = TemplateInput
	}

	if args.TemplateName != "" {
		isNonInteractive = true
		projectStatus = Pending
	}

	return Model{
		namePrompt:     namePrompt,
		templatePrompt: templatePrompt,
		nonInteractive: isNonInteractive,
		status:         projectStatus,
		spinner:        s,
		err:            nil,
		fs:             fs,
		downloader:     downloadr,
	}, nil
}

type projectCreateResultMsg struct {
	err error
}

// createProject returns a command that will create the project on disk using the inputs gathered
func (m Model) createProject(fs afero.Fs) tea.Cmd {
	return func() tea.Msg {
		cd, err := filepath.Abs(".")
		if err != nil {
			// TODO: make sure this error is handled in the view output.
			return projectCreateResultMsg{
				err: err,
			}
		}

		projDir := path.Join(cd, m.ProjectName())

		downloadr := templates.NewDownloader()
		if err = downloadr.DownloadDirectoryContents(m.TemplateName(), projDir, false); err != nil {
			return projectCreateResultMsg{
				err: err,
			}
		}

		yamlPath := filepath.Join(projDir, "./nitric.yaml")

		// Load and update the project name in the template's nitric.yaml
		p, err := project.ConfigurationFromFile(fs, yamlPath)
		if err != nil {
			return projectCreateResultMsg{
				err: err,
			}
		}

		p.Name = m.ProjectName()

		return projectCreateResultMsg{err: p.ToFile(fs, yamlPath)}
	}
}
