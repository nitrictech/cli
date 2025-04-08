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

package add_website

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/charmbracelet/lipgloss"
	"github.com/samber/lo"
	"github.com/spf13/afero"

	"github.com/nitrictech/cli/pkg/project"
	"github.com/nitrictech/cli/pkg/view/tui"
	"github.com/nitrictech/cli/pkg/view/tui/components/list"
	"github.com/nitrictech/cli/pkg/view/tui/components/listprompt"
	"github.com/nitrictech/cli/pkg/view/tui/components/textprompt"
	"github.com/nitrictech/cli/pkg/view/tui/components/validation"
	"github.com/nitrictech/cli/pkg/view/tui/components/view"
	"github.com/nitrictech/cli/pkg/view/tui/teax"
)

type step int

const (
	StepName step = iota
	StepPath
	StepTool
	StepPackageManager
	StepPort
	StepRunningToolCommand
	StepDone
)

// Args holds the arguments required for the website project creation
type Args struct {
	WebsiteName string
	WebsitePath string
	ToolName    string
}

var packageManagers = []string{"npm", "yarn", "pnpm"} // We will filter this based on what's installed

type configUpdatedResultMsg struct {
	err error
}

type commandResultMsg struct {
	err    error
	exited bool
	msg    string
}

type Model struct {
	windowSize    tea.WindowSizeMsg
	step          step
	toolPrompt    listprompt.ListPrompt
	packagePrompt listprompt.ListPrompt
	err           error
	namePrompt    textprompt.TextPrompt
	pathPrompt    textprompt.TextPrompt
	portPrompt    textprompt.TextPrompt

	config        *project.ProjectConfiguration
	existingPaths []string

	fs afero.Fs
}

func New(fs afero.Fs, args Args) (Model, error) {
	config, err := project.ConfigurationFromFile(fs, "")
	tui.CheckErr(err)

	// collect existing website names
	existingNames := []string{}
	for _, website := range config.Websites {
		existingNames = append(existingNames, website.Basedir)
	}

	// If website name is provided in args, validate it first
	if args.WebsiteName != "" {
		// Normalize the provided name
		normalizedName := strings.TrimPrefix(args.WebsiteName, "./")

		// Check if it's a duplicate
		for _, name := range existingNames {
			existingName := strings.TrimPrefix(name, "./")
			if existingName == normalizedName {
				return Model{}, fmt.Errorf("website name '%s' already exists", normalizedName)
			}
		}

		// Validate the format
		if !WebsiteNameRegex.MatchString(normalizedName) {
			return Model{}, fmt.Errorf("website name can only contain letters, numbers, underscores and hyphens")
		}

		// Check if name starts with a valid character
		if !WebsiteNameStartRegex.MatchString(normalizedName) {
			return Model{}, fmt.Errorf("website name must start with a letter or number")
		}

		// Check if name ends with a valid character
		if !WebsiteNameEndRegex.MatchString(normalizedName) {
			return Model{}, fmt.Errorf("website name cannot end with a hyphen")
		}

		// Check if directory already exists
		if _, err := fs.Stat(normalizedName); err == nil {
			return Model{}, fmt.Errorf("website directory '%s' already exists", normalizedName)
		} else if !os.IsNotExist(err) {
			return Model{}, fmt.Errorf("failed to check website directory: %w", err)
		}
	}

	nameValidator := validation.ComposeValidators(WebsiteNameValidators(existingNames)...)
	nameInFlightValidator := validation.ComposeValidators(WebsiteNameInFlightValidators()...)

	namePrompt := textprompt.NewTextPrompt("websiteName", textprompt.TextPromptArgs{
		Prompt:            "What would you like to name your website?",
		Tag:               "name",
		Validator:         nameValidator,
		Placeholder:       "",
		InFlightValidator: nameInFlightValidator,
	})

	// collect existing paths
	existingPaths := []string{}

	for _, website := range config.Websites {
		if website.Path == "" {
			existingPaths = append(existingPaths, "/")
			continue
		}

		existingPaths = append(existingPaths, website.Path)
	}

	pathValidator := validation.ComposeValidators(WebsiteURLPathValidators(existingPaths)...)
	pathInFlightValidator := validation.ComposeValidators(WebsiteURLPathInFlightValidators(existingPaths)...)

	pathPrompt := textprompt.NewTextPrompt("path", textprompt.TextPromptArgs{
		Prompt:            "What path would you like to use?",
		Tag:               "path",
		Validator:         pathValidator,
		Placeholder:       "",
		InFlightValidator: pathInFlightValidator,
	})

	step := StepName

	namePrompt.Focus()

	if args.WebsiteName != "" {
		namePrompt.SetValue(args.WebsiteName)
		namePrompt.Blur()

		if len(existingPaths) > 0 {
			step = StepPath

			pathPrompt.Focus()
		} else {
			step = StepTool
		}
	}

	if args.WebsitePath != "" {
		// check if the path is already in use
		if lo.Contains(existingPaths, args.WebsitePath) {
			return Model{}, fmt.Errorf("path %s is already in use", args.WebsitePath)
		}
		// check if the path is valid
		if err := pathValidator(args.WebsitePath); err != nil {
			return Model{}, fmt.Errorf("path %s is invalid: %w", args.WebsitePath, err)
		}

		pathPrompt.SetValue(args.WebsitePath)
		pathPrompt.Blur()

		step = StepTool
	}

	toolItems := []list.ListItem{}
	for _, tool := range tools {
		toolItems = append(toolItems, &tool)
	}

	toolPrompt := listprompt.NewListPrompt(listprompt.ListPromptArgs{
		Prompt: "Choose your site setup:",
		Tag:    "setup",
		Items:  toolItems,
	})

	if args.ToolName != "" {
		tool, found := lo.Find(tools, func(f Tool) bool { return f.Value == args.ToolName })
		if !found {
			return Model{}, fmt.Errorf("tool '%s' not found", args.ToolName)
		}

		toolPrompt.SetChoice(tool.Name)

		if args.WebsitePath != "" {
			if !tool.SkipPackageManagerPrompt {
				step = StepPackageManager
			} else {
				step = StepTool
			}
		} else {
			step = StepPath
		}
	}

	portValidator := validation.ComposeValidators(PortValidators()...)
	portInFlightValidator := validation.ComposeValidators(PortInFlightValidators()...)

	portPrompt := textprompt.NewTextPrompt("port", textprompt.TextPromptArgs{
		Prompt:            "What port would you like to use for development?",
		Tag:               "port",
		Validator:         portValidator,
		Placeholder:       "3000",
		InFlightValidator: portInFlightValidator,
	})

	return Model{
		step:       step,
		namePrompt: namePrompt,
		toolPrompt: toolPrompt,
		packagePrompt: listprompt.NewListPrompt(listprompt.ListPromptArgs{
			Prompt: "Which package manager would you like to use?",
			Tag:    "pkgm",
			Items:  list.StringsToListItems(getAvailablePackageManagers()),
		}),
		pathPrompt:    pathPrompt,
		portPrompt:    portPrompt,
		config:        config,
		fs:            fs,
		existingPaths: existingPaths,
		err:           nil,
	}, nil
}

func (m Model) Init() tea.Cmd {
	if m.err != nil {
		return teax.Quit
	}

	return tea.Batch(
		tea.ClearScreen,
		m.namePrompt.Init(),
		m.pathPrompt.Init(),
		m.toolPrompt.Init(),
		m.packagePrompt.Init(),
		m.portPrompt.Init(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowSize = msg

		if m.windowSize.Height < 15 {
			m.toolPrompt.SetMinimized(true)
			m.toolPrompt.SetMaxDisplayedItems(m.windowSize.Height - 1)
			m.packagePrompt.SetMinimized(true)
			m.packagePrompt.SetMaxDisplayedItems(m.windowSize.Height - 1)
		} else {
			m.toolPrompt.SetMinimized(false)
			maxItems := ((m.windowSize.Height - 3) / 3) // make room for the exit message
			m.toolPrompt.SetMaxDisplayedItems(maxItems)
			m.packagePrompt.SetMinimized(false)
			m.packagePrompt.SetMaxDisplayedItems(maxItems)
		}

		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, teax.Quit
		}
	case textprompt.CompleteMsg:
		if msg.ID == m.namePrompt.ID {
			m.namePrompt.Blur()

			if len(m.existingPaths) > 0 {
				m.step = StepPath
				m.pathPrompt.Focus()
			} else {
				m.step = StepTool
			}
		} else if msg.ID == m.pathPrompt.ID {
			m.pathPrompt.Blur()
			m.step = StepTool
		} else if msg.ID == m.portPrompt.ID {
			m.portPrompt.Blur()
			m.step = StepRunningToolCommand

			// Run the command directly
			return m, m.runCommand()
		}

		return m, nil
	case configUpdatedResultMsg:
		if msg.err == nil {
			m.step = StepDone
		} else {
			m.step = StepDone
			m.err = msg.err
		}

		return m, teax.Quit
	case commandResultMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, teax.Quit
		}

		if msg.exited {
			return m, teax.Quit
		}

		// Command completed successfully, update config
		return m, m.updateConfig()
	}

	switch m.step {
	case StepName:
		m.namePrompt, cmd = m.namePrompt.UpdateTextPrompt(msg)
	case StepPath:
		m.pathPrompt, cmd = m.pathPrompt.UpdateTextPrompt(msg)
	case StepTool:
		m.toolPrompt, cmd = m.toolPrompt.UpdateListPrompt(msg)

		if m.toolPrompt.Choice() != "" {
			tool, err := m.getSelectedTool()
			if err != nil {
				m.err = err

				return m, teax.Quit
			}

			// if the tool is not package manager based, we need to run the command
			if tool.SkipPackageManagerPrompt {
				m.packagePrompt.SetChoice(tool.Value)
				m.step = StepPort
				m.portPrompt.Focus()
			} else {
				m.step = StepPackageManager
			}
		}
	case StepPackageManager:
		m.packagePrompt, cmd = m.packagePrompt.UpdateListPrompt(msg)

		if m.packagePrompt.Choice() != "" {
			m.step = StepPort
			m.portPrompt.Focus()
		}
	case StepPort:
		m.portPrompt, cmd = m.portPrompt.UpdateTextPrompt(msg)
	}

	return m, cmd
}

var (
	errorTagStyle       = lipgloss.NewStyle().Background(tui.Colors.Red).Foreground(tui.Colors.White).PaddingLeft(2).PaddingRight(2).Align(lipgloss.Center)
	errorTextStyle      = lipgloss.NewStyle().PaddingLeft(2).Foreground(tui.Colors.Red)
	tagStyle            = lipgloss.NewStyle().Width(8).Background(tui.Colors.Purple).Foreground(tui.Colors.White).Align(lipgloss.Center)
	leftMarginStyle     = lipgloss.NewStyle().MarginLeft(2)
	createdHeadingStyle = lipgloss.NewStyle().Bold(true).MarginLeft(2)
	highlightStyle      = lipgloss.NewStyle().Foreground(tui.Colors.TextHighlight)
)

func (m Model) View() string {
	v := view.New(view.WithStyle(lipgloss.NewStyle()))

	// clear the screen manually with a print, fixes a bug where the screen is not cleared on exit due to ExecProcess
	if m.step == StepDone {
		fmt.Println("\033c")
	}

	if m.err != nil {
		v.Add("error").WithStyle(errorTagStyle)
		v.Addln(m.err.Error()).WithStyle(errorTextStyle)

		v.Break()

		return v.Render()
	}

	v.Addln(m.namePrompt.View())

	if m.step >= StepPath && len(m.existingPaths) > 0 {
		v.Addln(m.pathPrompt.View())
		v.Break()
	}

	if m.step >= StepTool {
		v.Addln(m.toolPrompt.View())
		v.Break()
	}

	if m.step >= StepPackageManager {
		tool, _ := m.getSelectedTool()

		if !tool.SkipPackageManagerPrompt {
			v.Addln(m.packagePrompt.View())
			v.Break()
		}
	}

	if m.step >= StepPort {
		v.Addln(m.portPrompt.View())
	}

	if m.step == StepRunningToolCommand {
		v.Break()

		v.Add("site").WithStyle(tagStyle)
		v.Addln("Running site setup 👇").WithStyle(leftMarginStyle)
		v.Break()
	}

	if m.step == StepDone {
		v.Break()
		v.Add("site").WithStyle(tagStyle)
		v.Addln("Website Created!").WithStyle(createdHeadingStyle)
		v.Break()

		indent := view.New(view.WithStyle(lipgloss.NewStyle().MarginLeft(10)))

		indent.Add("Navigate to your website with ")
		indent.Addln("cd ./%s", m.namePrompt.Value()).WithStyle(highlightStyle)

		indent.Break()

		indent.Add("Need help? Come and chat ")
		indent.Addln("https://nitric.io/chat").WithStyle(highlightStyle)

		v.Addln(indent.Render())
	} else if m.windowSize.Height > 10 {
		v.Break()
		v.Break()
		v.Add("(esc to quit)").WithStyle(lipgloss.NewStyle().Foreground(tui.Colors.TextMuted))
	}

	return v.Render()
}

// help to get the selected tool
func (m Model) getSelectedTool() (Tool, error) {
	tool, ok := lo.Find(tools, func(f Tool) bool {
		return f.Name == m.toolPrompt.Choice()
	})
	if !ok {
		return Tool{}, fmt.Errorf("tool %s not found", m.toolPrompt.Choice())
	}

	return tool, nil
}

func (m Model) runCommand() tea.Cmd {
	tool, err := m.getSelectedTool()
	if err != nil {
		m.err = err
		return teax.Quit
	}

	// if the tool has skip package manager prompt, check the tool exists and print the install guide
	if tool.SkipPackageManagerPrompt {
		if _, err := exec.LookPath(tool.Value); err != nil {
			return func() tea.Msg {
				return commandResultMsg{
					err: fmt.Errorf("tool %s not found, please install it using the following guide: %s", tool.Value, tool.InstallLink),
					msg: "Tool not found",
				}
			}
		}
	}

	cmd := tool.GetCreateCommand(m.packagePrompt.Choice(), m.namePrompt.Value())
	parts := strings.Fields(cmd)
	command := parts[0]
	args := parts[1:]

	c := exec.Command(command, args...)

	// Return a command that will execute the process
	return tea.ExecProcess(c, func(err error) tea.Msg {
		// If there was an error running the command
		if err != nil {
			return commandResultMsg{err: fmt.Errorf("failed to run website command: %w", err), msg: "Failed to create website"}
		}

		// Check if the website directory was created
		websiteDir := m.namePrompt.Value()
		if _, err := m.fs.Stat(websiteDir); err != nil {
			if os.IsNotExist(err) {
				return commandResultMsg{exited: true, msg: fmt.Sprintf("website directory '%s' was not created", websiteDir)}
			}

			return commandResultMsg{err: fmt.Errorf("failed to check website directory: %w", err), msg: "Failed to verify website creation"}
		}

		// If we get here, the website was created successfully
		return commandResultMsg{msg: "Website created successfully"}
	})
}

// update the nitric.yaml config file with website
func (m Model) updateConfig() tea.Cmd {
	return func() tea.Msg {
		var tool Tool

		for _, f := range tools {
			if f.Name == m.toolPrompt.Choice() {
				tool = f
				break
			}
		}

		path := m.pathPrompt.Value()
		port := m.portPrompt.Value()

		website := project.WebsiteConfiguration{
			Basedir: fmt.Sprintf("./%s", m.namePrompt.Value()),
			Build: project.Build{
				Command: tool.GetBuildCommand(m.packagePrompt.Choice(), path),
				Output:  tool.OutputDir,
			},
			Dev: project.Dev{
				Command: tool.GetDevCommand(m.packagePrompt.Choice(), port),
				URL:     tool.GetDevURL(port),
			},
			Path: path,
		}

		m.config.Websites = append(m.config.Websites, website)

		return configUpdatedResultMsg{
			err: m.config.ToFile(m.fs, ""),
		}
	}
}

// getAvailablePackageManagers filters package managers that exist on the system
func getAvailablePackageManagers() []string {
	available := []string{}

	for _, pm := range packageManagers {
		if _, err := exec.LookPath(pm); err == nil {
			available = append(available, pm)
		}
	}

	return available
}
