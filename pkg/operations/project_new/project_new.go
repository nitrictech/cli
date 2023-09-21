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

package project_new

import (
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/goombaio/namegenerator"
	"github.com/nitrictech/cli/pkg/project"
	"github.com/nitrictech/cli/pkg/templates"
	"github.com/nitrictech/cli/pkg/tui"
	"github.com/nitrictech/cli/pkg/tui/listprompt"
	"github.com/nitrictech/cli/pkg/tui/textprompt"
	"github.com/nitrictech/cli/pkg/utils"
)

type (
	errMsg error
)

type ProjectCreationStatus int

const (
	ToDo ProjectCreationStatus = iota
	Pending
	Done
	Error
)

var (
	force         bool
	nameRegex     = regexp.MustCompile(`^([a-zA-Z0-9-])*$`)
	projectNameQu = survey.Question{
		Name:   "projectName",
		Prompt: &survey.Input{Message: "What is the name of the project?"},
		// Validate: validateName,
	}
	templateNameQu = survey.Question{
		Name: "templateName",
	}
)

type Model struct {
	Name        string
	Template    string
	isValidName bool
	// textInput   textinput.Model
	namePrompt     textprompt.Model
	templatePrompt listprompt.Model
	projectStatus  ProjectCreationStatus

	spinner spinner.Model

	err error
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(tea.ClearScreen, m.namePrompt.Init(), m.templatePrompt.Init())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}

	case projectCreateResultMsg:
		if msg.err == nil {
			m.projectStatus = Done
		} else {
			m.projectStatus = Error
			m.err = msg.err
		}
		return m, nil

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	if !m.namePrompt.IsComplete() {
		m.namePrompt, cmd = m.namePrompt.Update(msg)
	} else if m.namePrompt.IsComplete() && !m.templatePrompt.IsComplete() {
		m.templatePrompt, cmd = m.templatePrompt.Update(msg)
		if m.templatePrompt.Choice() != "" {
			m.Name = m.namePrompt.Value()
			m.Template = m.templatePrompt.Choice()
			m.projectStatus = Pending
			return m, tea.Batch(m.spinner.Tick, m.createProject())
		}
	} else if m.projectStatus == Pending {
		m.spinner, cmd = m.spinner.Update(msg)
	}

	if m.projectStatus == Done {
		return m, tea.Quit
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

func tag(tag string) string {
	return lipgloss.NewStyle().Width(8).Background(tui.Colors.Purple).Foreground(tui.Colors.White).Align(lipgloss.Center).Render(tag)
}

func successMessage(projectPath string) string {
	var message strings.Builder

	message.WriteString(tag("proj"))
	message.WriteString(lipgloss.NewStyle().Bold(true).MarginLeft(2).Render("Project created!"))
	message.WriteString("\n\n")

	path := highlightStyle.Render(fmt.Sprintf("cd %s", projectPath))
	chatLink := highlightStyle.Render("https://nitric.io/chat")

	message.WriteString(lipgloss.NewStyle().MarginLeft(10).Render(fmt.Sprintf("Navigate to your project with %s\nInstall dependencies and you're ready to rock!\n\nNeed help? Come and chat %s", path, chatLink)))

	return message.String()
}

func (m Model) View() string {

	var view strings.Builder

	// Title
	view.WriteString(fmt.Sprintf("%sLet's get going!\n\n", titleStyle.Render("nitric")))

	// Name input
	view.WriteString(m.namePrompt.View())

	// Template selection input
	if m.namePrompt.IsComplete() {
		view.WriteString(m.templatePrompt.View())
	}

	// Creating Status
	if m.projectStatus == Pending {
		view.WriteString("\n\n")
		view.WriteString(tag("proj"))
		view.WriteString(fmt.Sprintf("  %s creating project...\n\n", m.spinner.View()))
	}

	// Done!
	if m.projectStatus == Done {
		view.WriteString("\n\n")
		view.WriteString(successMessage(fmt.Sprintf("./%s", m.Name)))
		view.WriteString("\n\n")
	} else {
		view.WriteString("\n\n(esc to quit)\n")
	}

	return view.String()
}

func New() *Model {
	seed := time.Now().UTC().UnixNano()
	nameGenerator := namegenerator.NewNameGenerator(seed)
	placeholderName := nameGenerator.Generate()

	namePrompt := textprompt.NewTextPrompt(textprompt.TextPromptArgs{
		Prompt:      "What should we name this project?",
		Tag:         "name",
		Placeholder: placeholderName,
		Validate:    validateName,
	})

	downloadr := templates.NewDownloader()
	templateNames, err := downloadr.Names()
	utils.CheckErr(err)

	templatePrompt := listprompt.New(listprompt.Args{
		Prompt:            "Which template should we start with?",
		Tag:               "tmpl",
		Items:             templateNames,
		MaxDisplayedItems: 5,
	})

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle

	namePrompt.Focus()

	return &Model{
		namePrompt:     *namePrompt,
		templatePrompt: *templatePrompt,
		spinner:        s,
		err:            nil,
	}
}

type projectCreateResultMsg struct {
	err error
}

func (m Model) createProject() tea.Cmd {
	return func() tea.Msg {
		cd, err := filepath.Abs(".")
		utils.CheckErr(err)

		projDir := path.Join(cd, m.Name)

		downloadr := templates.NewDownloader()
		err = downloadr.DownloadDirectoryContents(m.Template, projDir, force)
		utils.CheckErr(err)

		var p *project.Config

		// Load and update the project name in the template's nitric.yaml
		p, err = project.ConfigFromProjectPath(projDir)
		utils.CheckErr(err)
		p.Name = m.Name

		return projectCreateResultMsg{err: p.ToFile()}
	}
}

//
//func Run(ctx context.Context, args []string) {
//	answers := struct {
//		ProjectName  string
//		TemplateName string
//		FeedbackName string
//		Handlers     string
//	}{}
//
//	downloadr := templates.NewDownloader()
//	dirs, err := downloadr.Names()
//	utils.CheckErr(err)
//
//	templateNameQu.Prompt = &survey.Select{
//		Message: "Choose a template:",
//		Options: dirs,
//	}
//	templateNameQu.Validate = func(ans interface{}) error {
//		if len(args) < 2 {
//			return nil
//		}
//
//		a, ok := ans.(string)
//		if !ok {
//			return errors.New("wrong type, need a string")
//		}
//
//		if downloadr.Get(a) == nil {
//			return fmt.Errorf("%s not in %v", a, dirs)
//		}
//
//		return nil
//	}
//
//	qs := []*survey.Question{}
//
//	if len(args) > 0 {
//		if err := projectNameQu.Validate(args[0]); err != nil {
//			pterm.Error.PrintOnError(err)
//
//			qs = append(qs, &projectNameQu)
//		} else {
//			answers.ProjectName = args[0]
//		}
//	} else {
//		qs = append(qs, &projectNameQu)
//	}
//
//	if len(args) > 1 {
//		if err := templateNameQu.Validate(args[1]); err != nil {
//			pterm.Error.PrintOnError(err)
//
//			qs = append(qs, &templateNameQu)
//		} else {
//			answers.TemplateName = args[1]
//		}
//	} else {
//		qs = append(qs, &templateNameQu)
//		args = []string{} // reassign args to ensure validation works correctly.
//	}
//
//	if len(qs) > 0 {
//		err = survey.Ask(qs, &answers)
//		utils.CheckErr(err)
//	}
//
//	err = createNewProject(answers.ProjectName, answers.TemplateName)
//	utils.CheckErr(err)
//}
