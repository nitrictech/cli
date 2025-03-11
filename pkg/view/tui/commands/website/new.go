package add_website

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nitrictech/cli/pkg/project"
	"github.com/nitrictech/cli/pkg/view/tui"
	"github.com/nitrictech/cli/pkg/view/tui/components/list"
	"github.com/nitrictech/cli/pkg/view/tui/components/listprompt"
	"github.com/nitrictech/cli/pkg/view/tui/components/textprompt"
	"github.com/nitrictech/cli/pkg/view/tui/components/validation"
	"github.com/nitrictech/cli/pkg/view/tui/components/view"
	"github.com/nitrictech/cli/pkg/view/tui/teax"
	"github.com/samber/lo"
	"github.com/spf13/afero"
)

type step int

const (
	StepName step = iota
	StepPath
	StepFramework
	StepPackageManager
	StepCreatingWebsite
	StepUpdatingConfig
	StepDone
)

// Args holds the arguments required for the website project creation
type Args struct {
	WebsiteName   string
	WebsitePath   string
	FrameworkName string
	Force         bool
}

var packageManagers = []string{"npm", "yarn", "pnpm"} // We will filter this based on what’s installed

type websiteCreateResultMsg struct {
	err error
}

type configUpdatedResultMsg struct {
	err error
}

type Model struct {
	windowSize      tea.WindowSizeMsg
	step            step
	frameworkPrompt listprompt.ListPrompt
	packagePrompt   listprompt.ListPrompt
	err             error
	namePrompt      textprompt.TextPrompt
	pathPrompt      textprompt.TextPrompt
	spinner         spinner.Model

	config        *project.ProjectConfiguration
	existingPaths []string

	fs afero.Fs
}

func New(fs afero.Fs, args Args) (Model, error) {
	config, err := project.ConfigurationFromFile(fs, "")
	tui.CheckErr(err)

	nameValidator := validation.ComposeValidators(websiteNameValidators...)
	nameInFlightValidator := validation.ComposeValidators(websiteNameInFlightValidators...)

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
		step = StepPath
		pathPrompt.Focus()
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
		step = StepFramework
	}

	frameworkItems := []list.ListItem{}
	for _, framework := range frameworks {
		frameworkItems = append(frameworkItems, &framework)
	}

	frameworkPrompt := listprompt.NewListPrompt(listprompt.ListPromptArgs{
		Prompt: "Which framework would you like to use?",
		Tag:    "frmwk",
		Items:  frameworkItems,
	})

	if args.FrameworkName != "" {
		frameworkPrompt.SetChoice(args.FrameworkName)

		if args.WebsitePath != "" {
			step = StepPackageManager
		} else {
			step = StepPath
		}
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = highlightStyle

	return Model{
		step:            step,
		namePrompt:      namePrompt,
		frameworkPrompt: frameworkPrompt,
		packagePrompt: listprompt.NewListPrompt(listprompt.ListPromptArgs{
			Prompt: "Which package manager would you like to use?",
			Tag:    "pkgm",
			Items:  list.StringsToListItems(getAvailablePackageManagers()),
		}),
		pathPrompt:    pathPrompt,
		config:        config,
		fs:            fs,
		spinner:       s,
		existingPaths: existingPaths,
		err:           nil,
	}, nil
}

func (m Model) Init() tea.Cmd {
	if m.err != nil {
		return teax.Quit
	}

	// if m.nonInteractive {
	// 	return tea.Batch(m.spinner.Tick, m.createStack())
	// }

	return tea.Batch(
		tea.ClearScreen,
		m.namePrompt.Init(),
		m.pathPrompt.Init(),
		m.frameworkPrompt.Init(),
		m.packagePrompt.Init(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowSize = msg

		if m.windowSize.Height < 15 {
			m.frameworkPrompt.SetMinimized(true)
			m.frameworkPrompt.SetMaxDisplayedItems(m.windowSize.Height - 1)
			m.packagePrompt.SetMinimized(true)
			m.packagePrompt.SetMaxDisplayedItems(m.windowSize.Height - 1)
		} else {
			m.frameworkPrompt.SetMinimized(false)
			maxItems := ((m.windowSize.Height - 3) / 3) // make room for the exit message
			m.frameworkPrompt.SetMaxDisplayedItems(maxItems)
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
				m.step = StepFramework
			}
		} else if msg.ID == m.pathPrompt.ID {
			m.pathPrompt.Blur()
			m.step = StepFramework
		}

		return m, nil
	case websiteCreateResultMsg:
		if msg.err == nil {
			m.step = StepUpdatingConfig
		} else {
			m.step = StepDone
			m.err = msg.err

			return m, teax.Quit
		}
	case configUpdatedResultMsg:
		if msg.err == nil {
			m.step = StepDone
		} else {
			m.step = StepDone
			m.err = msg.err

		}

		return m, teax.Quit
	}

	switch m.step {
	case StepName:
		m.namePrompt, cmd = m.namePrompt.UpdateTextPrompt(msg)
	case StepPath:
		m.pathPrompt, cmd = m.pathPrompt.UpdateTextPrompt(msg)
	case StepFramework:
		m.frameworkPrompt, cmd = m.frameworkPrompt.UpdateListPrompt(msg)

		if m.frameworkPrompt.Choice() != "" {
			framework, err := m.getSelectedFramework()
			if err != nil {
				m.err = err

				return m, teax.Quit
			}

			if framework.Value == "hugo" {
				m.packagePrompt.SetChoice("hugo")
			}

			// if the framework is not package manager based, we need to run the command
			if framework.Value == "hugo" {
				m.step = StepCreatingWebsite

				return m, m.runCommand()
			}

			m.step = StepPackageManager
		}
	case StepPackageManager:
		m.packagePrompt, cmd = m.packagePrompt.UpdateListPrompt(msg)

		if m.packagePrompt.Choice() != "" {
			m.step = StepCreatingWebsite

			return m, m.runCommand()
		}
	case StepUpdatingConfig:
		return m, m.updateConfig()
	case StepCreatingWebsite:
		m.spinner, cmd = m.spinner.Update(msg)
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

	if m.step >= StepFramework {
		v.Addln(m.frameworkPrompt.View())
		v.Break()
	}

	if m.step >= StepPackageManager && m.packagePrompt.Choice() != "hugo" {
		v.Addln(m.packagePrompt.View())
		v.Break()
	}

	if m.step == StepCreatingWebsite || m.step == StepUpdatingConfig {
		v.Break()

		v.Add("site").WithStyle(tagStyle)
		v.Add(m.spinner.View()).WithStyle(leftMarginStyle)
		v.Addln(" creating website...")
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

		installCommand := fmt.Sprintf("%s install", m.packagePrompt.Choice())

		indent.Break()

		indent.Add("Install dependencies with ")
		indent.Add(installCommand).WithStyle(highlightStyle)
		indent.Addln(" and you're ready to rock! 🪨")
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

// help to get the selected framework
func (m Model) getSelectedFramework() (Framework, error) {
	framework, ok := lo.Find(frameworks, func(f Framework) bool {
		return f.Name == m.frameworkPrompt.Choice()
	})
	if !ok {
		return Framework{}, fmt.Errorf("framework %s not found", framework)

	}

	return framework, nil
}

func (m Model) runCommand() tea.Cmd {
	done := make(chan websiteCreateResultMsg)

	framework, err := m.getSelectedFramework()
	if err != nil {
		m.err = err
		return teax.Quit
	}

	cmd := framework.GetCreateCommand(m.packagePrompt.Choice(), m.namePrompt.Value())

	go func() {
		parts := strings.Fields(cmd)
		command := parts[0]
		args := parts[1:]

		c := exec.Command(command, args...)
		c.Stderr = os.Stderr // or a buffer if you want to capture output

		err := c.Run()
		if err != nil {
			err = fmt.Errorf("failed to run website command: %w", err)
		}

		done <- websiteCreateResultMsg{err: err}
		close(done)
	}()

	return tea.Batch(
		func() tea.Msg {
			return m.spinner.Tick() // This keeps the spinner ticking
		},
		waitForCommand(done),
	)
}

// This function waits for the external command to finish and sends a message when done
func waitForCommand(done chan websiteCreateResultMsg) tea.Cmd {
	return func() tea.Msg {
		// Blocking call to receive the result from the channel
		result := <-done
		return result // This sends the result back into the update loop
	}
}

// update the nitric.yaml config file with website
func (m Model) updateConfig() tea.Cmd {
	return func() tea.Msg {
		var framework Framework

		for _, f := range frameworks {
			if f.Name == m.frameworkPrompt.Choice() {
				framework = f
				break
			}
		}

		path := m.pathPrompt.Value()

		website := project.WebsiteConfiguration{
			Basedir: fmt.Sprintf("./%s", m.namePrompt.Value()),
			Build: project.Build{
				Command: framework.GetBuildCommand(m.packagePrompt.Choice(), path),
				Output:  framework.OutputDir,
			},
			Dev: project.Dev{
				Command: framework.GetDevCommand(m.packagePrompt.Choice(), path),
				URL:     framework.DevURL,
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
	var available []string
	for _, pm := range packageManagers {
		if _, err := exec.LookPath(pm); err == nil {
			available = append(available, pm)
		}
	}
	return available
}
