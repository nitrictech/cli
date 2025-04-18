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

package stack_new

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/samber/lo"
	"github.com/spf13/afero"

	"github.com/nitrictech/cli/pkg/project"
	"github.com/nitrictech/cli/pkg/project/stack"
	clitui "github.com/nitrictech/cli/pkg/view/tui"
	tui "github.com/nitrictech/cli/pkg/view/tui"
	validators "github.com/nitrictech/cli/pkg/view/tui/commands/stack"
	"github.com/nitrictech/cli/pkg/view/tui/components/list"
	"github.com/nitrictech/cli/pkg/view/tui/components/listprompt"
	"github.com/nitrictech/cli/pkg/view/tui/components/textprompt"
	"github.com/nitrictech/cli/pkg/view/tui/components/validation"
	"github.com/nitrictech/cli/pkg/view/tui/components/view"
	"github.com/nitrictech/cli/pkg/view/tui/teax"
)

type (
	errMsg error
)

type NewStackStatus int

const (
	NameInput NewStackStatus = iota
	ProviderInput
	Pending
	Done
	Error
)

// Model - represents the state of the new stack creation operation
type Model struct {
	windowSize tea.WindowSizeMsg

	namePrompt     textprompt.TextPrompt
	providerPrompt listprompt.ListPrompt
	spinner        spinner.Model
	status         NewStackStatus
	provider       string
	projectConfig  *project.ProjectConfiguration
	nonInteractive bool

	newStackFilePath string

	fs afero.Fs

	err error
}

// StackName - returns the stack name entered by the user
func (m Model) StackName() string {
	return m.namePrompt.Value()
}

// ProviderName returns the stack cloud name selected by the user
func (m Model) ProviderName() string {
	return m.providerPrompt.Choice()
}

// Init initializes the model, used by Bubbletea
func (m Model) Init() tea.Cmd {
	if m.err != nil {
		return teax.Quit
	}

	if m.nonInteractive {
		return tea.Batch(m.spinner.Tick, m.createStack())
	}

	return tea.Batch(tea.ClearScreen, m.namePrompt.Init(), m.providerPrompt.Init())
}

// Update the model based on a message
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowSize = msg

		if m.windowSize.Height < 15 {
			m.providerPrompt.SetMinimized(true)
			m.providerPrompt.SetMaxDisplayedItems(m.windowSize.Height - 1)
		} else {
			m.providerPrompt.SetMinimized(false)
			maxItems := ((m.windowSize.Height - 3) / 3) // make room for the exit message
			m.providerPrompt.SetMaxDisplayedItems(maxItems)
		}

		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, teax.Quit
		}

	case stackCreateResultMsg:
		if msg.err == nil {
			m.status = Done
			m.newStackFilePath = msg.filePath
		} else {
			m.status = Error
			m.err = msg.err
		}

		return m, teax.Quit

	case errMsg:
		m.err = msg
		return m, teax.Quit
	case textprompt.CompleteMsg:
		if msg.ID == m.namePrompt.ID {
			m.namePrompt.Blur()

			m.status = ProviderInput
		}

		return m, nil
	}

	// Deal with the various steps in the process from data capture to building the project
	// FIXME: don't switch on status here, only look for Msg types. Update may not be called when the status changes.
	switch m.status {
	case NameInput:
		m.namePrompt, cmd = m.namePrompt.UpdateTextPrompt(msg)
	case ProviderInput:
		m.providerPrompt, cmd = m.providerPrompt.UpdateListPrompt(msg)

		if m.providerPrompt.Choice() != "" {
			m.provider = m.providerPrompt.Choice()

			m.status = Pending

			return m, m.createStack()
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
	spinnerStyle   = lipgloss.NewStyle().Foreground(tui.Colors.TextHighlight)
	highlightStyle = lipgloss.NewStyle().Foreground(tui.Colors.TextHighlight)
)

var (
	errorTag                 = lipgloss.NewStyle().Background(tui.Colors.Red).Foreground(tui.Colors.White).PaddingLeft(2).PaddingRight(2).Align(lipgloss.Center)
	errorText                = lipgloss.NewStyle().PaddingLeft(2).Foreground(tui.Colors.Red)
	tagStyle                 = lipgloss.NewStyle().Width(8).Background(tui.Colors.Purple).Foreground(tui.Colors.White).Align(lipgloss.Center)
	leftMarginStyle          = lipgloss.NewStyle().MarginLeft(2)
	stackCreatedHeadingStyle = lipgloss.NewStyle().Bold(true).MarginLeft(2)
)

func (m Model) View() string {
	v := view.New()

	if m.err != nil {
		v.Add("error").WithStyle(errorTag)
		v.Addln(m.err.Error()).WithStyle(errorText)
		v.Break()

		return v.Render()
	}

	if !m.nonInteractive {
		v.Add("nitric").WithStyle(titleStyle)
		v.Addln("Let's get deployed!")
		v.Break()

		v.Addln(m.namePrompt.View())

		// Cloud selection input
		if m.status >= ProviderInput {
			v.Add(m.providerPrompt.View())
		}
	}

	// Creating Status
	if m.status == Pending {
		v.Break()

		v.Add("stack").WithStyle(tagStyle)
		v.Add(m.spinner.View()).WithStyle(leftMarginStyle)
		v.Addln(" creating stack...")
		v.Break()
	}

	// Done!
	if m.status == Done {
		v.Break()
		v.Add("stack").WithStyle(tagStyle)
		v.Addln("Stack Created!").WithStyle(stackCreatedHeadingStyle)

		indent := view.New(view.WithStyle(lipgloss.NewStyle().MarginLeft(10)))

		indent.Add("Your new stack is available at ")
		indent.Addln(m.newStackFilePath).WithStyle(highlightStyle)
		indent.Break()

		indent.Addln("Check the file for any additional configuration required.")
		indent.Break()

		indent.Add("Then deploy your stack using ")
		indent.Addln("nitric up").WithStyle(highlightStyle)

		indent.Add("Need help? Come and chat ")
		indent.Addln("https://nitric.io/chat 💬").WithStyle(highlightStyle)

		v.Add(indent.Render())
	} else if m.windowSize.Height > 10 {
		v.Break()
		v.Break()
		v.Add("(esc to quit)").WithStyle(lipgloss.NewStyle().Foreground(tui.Colors.TextMuted))
	}

	return v.Render()
}

type Args struct {
	StackName    string
	ProviderName string
	Force        bool
}

type ListItem struct {
	Value       string
	Description string
}

func (m *ListItem) GetItemValue() string {
	return m.Value
}

func (m *ListItem) GetItemDescription() string {
	return ""
}

func stackNameExistsValidator(projectDir string) validation.StringValidator {
	return func(stackName string) error {
		_, err := os.Stat(filepath.Join(projectDir, fmt.Sprintf("nitric.%s.yaml", stackName)))
		if err == nil {
			return fmt.Errorf(`stack with the name "%s" already exists. Choose a different name or use the --force flag to create`, stackName)
		}

		return nil
	}
}

const (
	Aws     = "Pulumi AWS"
	Azure   = "Pulumi Azure"
	Gcp     = "Pulumi Google Cloud"
	AwsTf   = "Terraform AWS (Preview)"
	AzureTf = "Terraform Azure (Preview)"
	GcpTf   = "Terraform Google Cloud (Preview)"
)

var availableProviders = []string{Aws, Gcp, Azure, AwsTf, GcpTf, AzureTf}

func New(fs afero.Fs, args Args) Model {
	// Load and update the project name in the template's nitric.yaml
	projectConfig, err := project.ConfigurationFromFile(fs, "")
	clitui.CheckErr(err)

	if !args.Force {
		validators.ProjectNameValidators = append(validators.ProjectNameValidators, stackNameExistsValidator(projectConfig.Directory))
	}

	nameValidator := validation.ComposeValidators(validators.ProjectNameValidators...)
	nameInFlightValidator := validation.ComposeValidators(validators.ProjectNameValidators...)

	namePrompt := textprompt.NewTextPrompt("stackName", textprompt.TextPromptArgs{
		Prompt:            "What should we name this stack?",
		Tag:               "name",
		Validator:         nameValidator,
		Placeholder:       "dev",
		InFlightValidator: nameInFlightValidator,
	})
	namePrompt.Focus()

	providerPrompt := listprompt.NewListPrompt(listprompt.ListPromptArgs{
		Prompt: "Which provider do you want to deploy with?",
		Tag:    "prov",
		Items:  list.StringsToListItems(availableProviders),
	})

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle

	// prefill values from CLI args
	if args.StackName != "" {
		if err := nameValidator(args.StackName); err != nil {
			return Model{
				err: err,
			}
		}

		namePrompt.SetValue(args.StackName)
	}

	if args.ProviderName != "" {
		validProviders := lo.Map(availableProviders, func(p string, _ int) string {
			return providerLabelToValue(p)
		})

		if !lo.Contains(validProviders, args.ProviderName) {
			return Model{
				err: fmt.Errorf("cloud name is not valid, must be one of: %v", strings.Join(validProviders, ", ")),
			}
		}

		providerPrompt.SetChoice(args.ProviderName)
	}

	isNonInteractive := false
	stackStatus := NameInput

	if args.StackName != "" {
		stackStatus = ProviderInput

		namePrompt.Blur()
	}

	return Model{
		fs:             fs,
		namePrompt:     namePrompt,
		providerPrompt: providerPrompt,
		nonInteractive: isNonInteractive,
		status:         stackStatus,
		projectConfig:  projectConfig,
		spinner:        s,
		err:            nil,
	}
}

type stackCreateResultMsg struct {
	err      error
	filePath string
}

func providerLabelToValue(provider string) string {
	switch provider {
	case Aws:
		return "aws"
	case Azure:
		return "azure"
	case Gcp:
		return "gcp"
	case AwsTf:
		return "aws-tf"
	case GcpTf:
		return "gcp-tf"
	case AzureTf:
		return "azure-tf"
	}

	return strings.ToLower(provider)
}

// createStack returns a command that will create the stack on disk using the inputs gathered
func (m Model) createStack() tea.Cmd {
	return func() tea.Msg {
		filePath, err := stack.NewStackFile(m.fs, providerLabelToValue(m.provider), m.StackName(), "")

		return stackCreateResultMsg{
			err:      err,
			filePath: filePath,
		}
	}
}
