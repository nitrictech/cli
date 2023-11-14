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

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/samber/lo"

	"github.com/nitrictech/cli/pkg/codeconfig"
	"github.com/nitrictech/cli/pkg/project"
	"github.com/nitrictech/cli/pkg/provider"
	"github.com/nitrictech/cli/pkg/provider/types"
	"github.com/nitrictech/cli/pkg/utils"
	"github.com/nitrictech/pearls/pkg/tui"
	"github.com/nitrictech/pearls/pkg/tui/inlinelist"
	"github.com/nitrictech/pearls/pkg/tui/listprompt"
	"github.com/nitrictech/pearls/pkg/tui/textprompt"
	"github.com/nitrictech/pearls/pkg/tui/validation"
	"github.com/nitrictech/pearls/pkg/tui/view"
)

type (
	errMsg error
)

type NewStackStatus int

const (
	NameInput NewStackStatus = iota
	ProviderInput
	RegionInput
	GCPProjectInput
	AzureOrgInput
	AzureAdminEmailInput
	Pending
	Done
	Error
)

// Model - represents the state of the new stack creation operation
type Model struct {
	namePrompt            textprompt.Model
	providerPrompt        listprompt.Model
	regionPrompt          listprompt.Model
	gcpProjectPrompt      textprompt.Model
	azureOrgPrompt        textprompt.Model
	azureAdminEmailPrompt textprompt.Model
	spinner               spinner.Model
	status                NewStackStatus
	provider              types.Provider
	projectConfig         *project.Config
	nonInteractive        bool

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

// Region returns the stack cloud region selected by the user
func (m Model) Region() string {
	return m.regionPrompt.Choice()
}

// Init initializes the model, used by Bubbletea
func (m Model) Init() tea.Cmd {
	if m.err != nil {
		return tea.Quit
	}

	if m.nonInteractive {
		return tea.Batch(m.spinner.Tick, m.createStack())
	}

	return tea.Batch(tea.ClearScreen, m.namePrompt.Init(), m.providerPrompt.Init(), m.regionPrompt.Init(), m.gcpProjectPrompt.Init())
}

// Update the model based on a message
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}

	case stackCreateResultMsg:
		if msg.err == nil {
			m.status = Done
		} else {
			m.status = Error
			m.err = msg.err
		}

		return m, nil

	case errMsg:
		m.err = msg
		return m, tea.Quit
	case textprompt.CompleteMsg:
		if msg.ID == m.namePrompt.ID {
			m.namePrompt.Blur()

			m.status = ProviderInput
		}

		if msg.ID == m.gcpProjectPrompt.ID {
			m.provider.SetStackConfigProp("gcp-project-id", m.gcpProjectPrompt.Value())
			m.gcpProjectPrompt.Blur()

			m.status = Pending

			return m, tea.Batch(m.spinner.Tick, m.createStack())
		}

		if msg.ID == m.azureOrgPrompt.ID {
			m.provider.SetStackConfigProp("org", m.azureOrgPrompt.Value())
			m.azureOrgPrompt.Blur()

			m.status = AzureAdminEmailInput
		}

		if msg.ID == m.azureAdminEmailPrompt.ID {
			m.provider.SetStackConfigProp("adminemail", m.azureAdminEmailPrompt.Value())
			m.azureOrgPrompt.Blur()

			m.status = Pending

			return m, tea.Batch(m.spinner.Tick, m.createStack())
		}

		return m, nil
	}

	// Deal with the various steps in the process from data capture to building the project
	switch m.status {
	case NameInput:
		m.namePrompt, cmd = m.namePrompt.UpdateTextPrompt(msg)
	case ProviderInput:
		m.providerPrompt, cmd = m.providerPrompt.UpdateListPrompt(msg)

		if m.providerPrompt.Choice() != "" {
			m.provider = getProviderFromCloud(m.projectConfig, m.StackName(), m.providerPrompt.Choice())

			m.regionPrompt = m.regionPrompt.UpdateItems(getRegionList(m.provider))

			m.status = RegionInput
		}
	case RegionInput:
		m.regionPrompt, cmd = m.regionPrompt.UpdateListPrompt(msg)
		if m.regionPrompt.Choice() != "" {
			m.provider.SetStackConfigProp("region", m.Region())

			if m.ProviderName() == "gcp" {
				m.status = GCPProjectInput
			} else if m.ProviderName() == "azure" {
				m.status = AzureOrgInput
			} else {
				m.status = Pending
				return m, tea.Batch(m.spinner.Tick, m.createStack())
			}
		}
	case GCPProjectInput:
		m.gcpProjectPrompt, cmd = m.gcpProjectPrompt.UpdateTextPrompt(msg)
	case AzureOrgInput:
		m.azureOrgPrompt, cmd = m.azureOrgPrompt.UpdateTextPrompt(msg)
	case AzureAdminEmailInput:
		m.azureAdminEmailPrompt, cmd = m.azureAdminEmailPrompt.UpdateTextPrompt(msg)
	case Pending:
		m.spinner, cmd = m.spinner.Update(msg)
	case Done:
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

var (
	errorTagStyle            = lipgloss.NewStyle().Background(tui.Colors.Red).Foreground(tui.Colors.White).PaddingLeft(2).PaddingRight(2).Align(lipgloss.Center)
	errorTextStyle           = lipgloss.NewStyle().PaddingLeft(2).Foreground(tui.Colors.Red)
	tagStyle                 = lipgloss.NewStyle().Width(8).Background(tui.Colors.Purple).Foreground(tui.Colors.White).Align(lipgloss.Center)
	stackCreatedHeadingStyle = lipgloss.NewStyle().Bold(true).MarginLeft(2)
)

func (m Model) View() string {
	stackView := view.New()

	if m.err != nil {
		stackView.AddRow(
			view.NewFragment("error").WithStyle(errorTagStyle),
			view.NewFragment(m.err.Error()).WithStyle(errorTextStyle),
			view.Break(),
		)

		return stackView.Render()
	}

	if !m.nonInteractive {
		stackView.AddRow(
			view.NewFragment("nitric").WithStyle(titleStyle),
			view.NewFragment("Let's get deployed!"),
			view.Break(),
		)

		stackView.AddRow(
			view.NewFragment(m.namePrompt.View()),
		)

		// Cloud selection input
		if m.status >= ProviderInput {
			stackView.AddRow(
				view.NewFragment(m.providerPrompt.View()),
			)
		}

		if m.status >= RegionInput {
			stackView.AddRow(
				view.NewFragment(m.regionPrompt.View()),
			)
		}

		if m.ProviderName() == "gcp" && lo.Contains([]NewStackStatus{GCPProjectInput, Pending, Done}, m.status) {
			stackView.AddRow(
				view.NewFragment(m.gcpProjectPrompt.View()),
			)
		}

		if m.ProviderName() == "azure" {
			if lo.Contains([]NewStackStatus{AzureOrgInput, AzureAdminEmailInput, Pending, Done}, m.status) {
				stackView.AddRow(
					view.NewFragment(m.azureOrgPrompt.View()),
				)
			}

			if lo.Contains([]NewStackStatus{AzureAdminEmailInput, Pending, Done}, m.status) {
				stackView.AddRow(
					view.NewFragment(m.azureAdminEmailPrompt.View()),
				)
			}
		}
	}

	// Creating Status
	if m.status == Pending {
		stackView.AddRow(
			view.Break(),
			view.NewFragment("stack").WithStyle(tagStyle),
			view.NewFragment(m.spinner.View()).WithStyle(lipgloss.NewStyle().MarginLeft(2)),
			view.NewFragment(" creating stack..."),
			view.Break(),
		)
	}

	// Done!
	if m.status == Done {
		stackView.AddRow(
			view.Break(),
			view.NewFragment("stack").WithStyle(tagStyle),
			view.NewFragment("Stack Created!").WithStyle(stackCreatedHeadingStyle),
			view.Break(),
		)

		shiftRight := lipgloss.NewStyle().MarginLeft(10)

		stackView.AddRow(
			view.NewFragment("Deploy your stack with "),
			view.NewFragment("nitric up").WithStyle(highlightStyle),
		).WithStyle(shiftRight)

		stackView.AddRow(
			view.NewFragment("Need help? Come and chat "),
			view.NewFragment("https://nitric.io/chat").WithStyle(highlightStyle),
			view.Break(),
		).WithStyle(shiftRight)
	} else {
		stackView.AddRow(
			view.Break(),
			view.NewFragment("(esc to quit)").WithStyle(lipgloss.NewStyle().Foreground(tui.Colors.Gray)),
		)
	}

	return stackView.Render()
}

type Args struct {
	StackName    string
	ProviderName string
	Force        bool
}

type RegionItem struct {
	Value       string
	Description string
}

func (m *RegionItem) GetItemValue() string {
	return m.Value
}

func (m *RegionItem) GetItemDescription() string {
	return ""
}

func stackNameExistsValidator(projectDir string) validation.StringValidator {
	return func(stackName string) error {
		_, err := os.Stat(filepath.Join(projectDir, fmt.Sprintf("nitric-%s.yaml", stackName)))
		if err == nil {
			return fmt.Errorf(`stack with the name "%s" already exists. Choose a different name or use the --force flag to create`, stackName)
		}

		return nil
	}
}

func New(args Args) Model {
	// check if in a nitric project directory
	projDir, err := filepath.Abs(".")
	utils.CheckErr(err)

	// Load and update the project name in the template's nitric.yaml
	projectConfig, err := project.ConfigFromProjectPath(projDir)
	utils.CheckErr(err)

	if !args.Force {
		projectNameValidators = append(projectNameValidators, stackNameExistsValidator(projectConfig.Dir))
	}

	nameValidator := validation.ComposeValidators(projectNameValidators...)
	nameInFlightValidator := validation.ComposeValidators(projectNameInFlightValidators...)
	azureOrgNameValidator := validation.ComposeValidators(azureOrgNameValidators...)
	adminEmailValidator := validation.ComposeValidators(adminEmailValidators...)
	gcpProjectIDValidator := validation.ComposeValidators(gcpProjectIDValidators...)

	noopValidator := func(s string) error { return nil }

	namePrompt := textprompt.NewTextPrompt("stackName", textprompt.TextPromptArgs{
		Prompt:            "What should we name this stack?",
		Tag:               "name",
		Validator:         nameValidator,
		Placeholder:       "dev",
		InFlightValidator: nameInFlightValidator,
	})
	namePrompt.Focus()

	providerPrompt := listprompt.New(listprompt.Args{
		Prompt: "Which provider do you want to deploy with?",
		Tag:    "prov",
		Items:  listprompt.ConvertStringsToListItems(types.Providers),
	})

	regionPrompt := listprompt.New(listprompt.Args{
		Prompt: "Which region should the stack deploy to?",
		Tag:    "region",
		Items:  []inlinelist.ListItem{},
	})

	gcpProjectPrompt := textprompt.NewTextPrompt("project", textprompt.TextPromptArgs{
		Prompt:            "Provide the gcp project ID to deploy to",
		Tag:               "proj",
		Validator:         gcpProjectIDValidator,
		InFlightValidator: noopValidator,
	})
	gcpProjectPrompt.Focus()

	azureOrgPrompt := textprompt.NewTextPrompt("org", textprompt.TextPromptArgs{
		Prompt:            "Provide the organisation to associate with the API",
		Tag:               "org",
		Validator:         azureOrgNameValidator,
		InFlightValidator: noopValidator,
	})
	azureOrgPrompt.Focus()

	azureAdminEmailPrompt := textprompt.NewTextPrompt("adminEmail", textprompt.TextPromptArgs{
		Prompt:            "Provide the Admin email to associate with the API",
		Tag:               "email",
		Validator:         adminEmailValidator,
		InFlightValidator: noopValidator,
	})
	azureAdminEmailPrompt.Focus()

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

	var provider types.Provider

	if args.ProviderName != "" {
		if !lo.Contains([]string{"aws", "azure", "gcp"}, args.ProviderName) {
			return Model{
				err: fmt.Errorf("cloud name is not valid, must be aws, azure or gcp"),
			}
		}

		provider = getProviderFromCloud(projectConfig, args.StackName, args.ProviderName)

		providerPrompt.SetChoice(args.ProviderName)

		regionPrompt = regionPrompt.UpdateItems(getRegionList(provider))
	}

	isNonInteractive := false
	stackStatus := NameInput

	if args.StackName != "" {
		stackStatus = ProviderInput

		namePrompt.Blur()
	}

	if args.ProviderName != "" {
		stackStatus = RegionInput
	}

	return Model{
		namePrompt:            namePrompt,
		providerPrompt:        providerPrompt,
		regionPrompt:          regionPrompt,
		gcpProjectPrompt:      gcpProjectPrompt,
		azureOrgPrompt:        azureOrgPrompt,
		azureAdminEmailPrompt: azureAdminEmailPrompt,
		nonInteractive:        isNonInteractive,
		status:                stackStatus,
		projectConfig:         projectConfig,
		provider:              provider,
		spinner:               s,
		err:                   nil,
	}
}

type stackCreateResultMsg struct {
	err error
}

func getProviderFromCloud(config *project.Config, stackName string, cloud string) types.Provider {
	cc, err := codeconfig.New(project.New(config.BaseConfig), map[string]string{})
	utils.CheckErr(err)

	provider, err := provider.NewProvider(cc, stackName, cloud, map[string]string{}, &types.ProviderOpts{SkipChecks: true})
	utils.CheckErr(err)

	return provider
}

// createStack returns a command that will create the stack on disk using the inputs gathered
func getRegionList(provider types.Provider) []inlinelist.ListItem {
	listItems := []inlinelist.ListItem{}

	for _, region := range provider.SupportedRegions() {
		listItems = append(listItems, region)
	}

	return listItems
}

// createStack returns a command that will create the stack on disk using the inputs gathered
func (m Model) createStack() tea.Cmd {
	return func() tea.Msg {
		return stackCreateResultMsg{err: m.provider.ToFile()}
	}
}
