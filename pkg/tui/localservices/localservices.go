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

package localservices

import (
	"fmt"
	"time"

	"github.com/bep/debounce"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pterm/pterm"

	"github.com/nitrictech/cli/pkg/dashboard"
	"github.com/nitrictech/cli/pkg/output"
	"github.com/nitrictech/cli/pkg/project"
	"github.com/nitrictech/cli/pkg/run"
	"github.com/nitrictech/pearls/pkg/tui"
	"github.com/nitrictech/pearls/pkg/tui/view"
)

var (
	textStyle    = lipgloss.NewStyle().Foreground(tui.Colors.White).Align(lipgloss.Left)
	warningStyle = lipgloss.NewStyle().Foreground(tui.Colors.Blue)
	helpStyle    = lipgloss.NewStyle().Foreground(tui.Colors.Gray)
)

type Model struct {
	sub chan tea.Msg

	viewport viewport.Model

	help help.Model

	envMap    map[string]string
	project   *project.Project
	noBrowser bool
	quitting  bool

	Ready      FunctionsReadyMessage
	StackState *run.RunStackState
	Error      error
	Warnings   []string
}

type ModelArgs struct {
	Envs      map[string]string
	Project   *project.Project
	Sub       chan tea.Msg
	NoBrowser bool
}

func New(args ModelArgs) Model {
	if args.Envs == nil {
		args.Envs = make(map[string]string)
	}

	// Need to initialise the viewport size here as an initial window size isn't sent in update
	// This is because it is a nested model
	otherContentLines := 12

	w, h, err := pterm.GetTerminalSize()
	vp := viewport.New(w, h-otherContentLines)

	return Model{
		envMap:    args.Envs,
		project:   args.Project,
		sub:       args.Sub,
		noBrowser: args.NoBrowser,
		viewport:  vp,
		Error:     err,
	}
}

func startLocalServices(sub chan tea.Msg, project *project.Project, envMap map[string]string, noBrowser bool) tea.Msg {
	sub <- StartingFunctionsMessage{}

	dash, err := dashboard.New(project, envMap)
	if err != nil {
		return ErrorMessage{
			Error: err,
		}
	}

	ls := run.NewLocalServices(project, false, dash)
	if ls.Running() {
		return ErrorMessage{
			Error: fmt.Errorf("only one instance of Nitric can be run locally at a time, please check that you have ended all other instances and try again"),
		}
	}

	pool := run.NewRunProcessPool()

	membraneErr := make(chan error)
	go func(errch chan error) {
		errch <- ls.Start(pool, true)
	}(membraneErr)

	for {
		select {
		case memErr := <-membraneErr:
			// catch any early errors from Start()
			if memErr != nil {
				return ErrorMessage{Error: memErr}
			}
		default:
		}

		if ls.Running() {
			break
		}

		time.Sleep(time.Second)
	}

	functions, err := run.FunctionsFromHandlers(project)
	if err != nil {
		return ErrorMessage{Error: err}
	}

	for _, f := range functions {
		err = f.Start(envMap)
		if err != nil {
			return ErrorMessage{Error: err}
		}
	}

	sub <- FunctionsStartedMessage{}

	stackState := run.NewStackState(project)
	sub <- StackUpdateMessage{
		StackState: stackState,
	}

	err = ls.Refresh()
	if err != nil {
		return ErrorMessage{Error: err}
	}

	stackState.Update(pool, ls)
	sub <- StackUpdateMessage{
		StackState: stackState,
	}

	// Create a debouncer for the refresh and remove locking
	debounced := debounce.New(500 * time.Millisecond)

	// React to worker pool state and update services table
	pool.Listen(func(we run.WorkerEvent) {
		debounced(func() {
			err := ls.Refresh()
			if err != nil {
				sub <- ErrorMessage{Error: err}
				return
			}

			if !dash.HasStarted() {
				// Start local dashboard
				err = dash.Serve(ls.GetStorageService(), noBrowser || output.CI)
				if err != nil {
					sub <- ErrorMessage{Error: err}
					return
				}
			}

			stackState.Update(pool, ls)

			sub <- StackUpdateMessage{
				StackState: stackState,
			}
		})
	})

	memErr := <-membraneErr
	if memErr != nil {
		return ErrorMessage{Error: memErr}
	}

	return FunctionsReadyMessage{
		LocalServices: ls,
		Functions:     functions,
	}
}

func (m Model) Init() tea.Cmd {
	return func() tea.Msg {
		return startLocalServices(m.sub, m.project, m.envMap, m.noBrowser)
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case ErrorMessage:
		m.Error = msg.Error
		cmds = append(cmds, subscribeToChannel(m.sub))
	case StartingFunctionsMessage:
		cmds = append(cmds, subscribeToChannel(m.sub))
	case FunctionsStartedMessage:
		cmds = append(cmds, subscribeToChannel(m.sub))
	case WarningMessage:
		m.Warnings = append(m.Warnings, msg.Warning)
		cmds = append(cmds, subscribeToChannel(m.sub))
	case StackUpdateMessage:
		m.StackState = msg.StackState
		cmds = append(cmds, subscribeToChannel(m.sub))
	case FunctionsReadyMessage:
		m.Ready = msg
		cmds = append(cmds, subscribeToChannel(m.sub))
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, LocalServicesKeys.Quit):
			m.quitting = true

			for _, f := range m.Ready.Functions {
				err := f.Stop()
				m.Error = err
			}

			if m.Ready.LocalServices != nil {
				err := m.Ready.LocalServices.Stop()
				m.Error = err
			}

			return m, tea.Quit
		case key.Matches(msg, LocalServicesKeys.Down):
			m.viewport.LineDown(1)
			return m, nil
		case key.Matches(msg, LocalServicesKeys.Up):
			m.viewport.LineUp(1)
			return m, nil
		}
	case tea.WindowSizeMsg:
		// the number of lines that content other than the viewport fills.
		otherContentLines := 10

		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - otherContentLines

		if m.viewport.Height < 0 {
			m.viewport.Height = 0
		}

		m.viewport.GotoTop()

		return m, tea.ClearScreen
	}

	_, content := m.getViewportContent()
	m.viewport.SetContent(content)

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	renderer := view.New().WithStyle(textStyle)

	if m.Error != nil {
		renderer.AddRow(
			view.NewFragment("An error occurred whilst starting local services:"),
			view.Break(),
			view.NewFragment(m.Error),
			view.Break(),
		)

		return renderer.Render()
	}

	if m.quitting {
		renderer.AddRow(
			view.NewFragment("Shutting down local services..."),
			view.Break(),
		)

		return renderer.Render()
	}

	tables := 0
	if m.StackState != nil {
		tables = len(m.StackState.Tables())
	}

	if m.StackState != nil {
		count, _ := m.getViewportContent()

		renderer.AddRow(
			view.WhenOr(
				tables != 0,
				view.NewFragment("Application is running!"),
				view.NewFragment("Waiting for your application to start..."),
			),
			view.Break(),
			view.Break(),
			view.NewFragment(m.viewport.View()),
			view.Break(),
			view.Break(),
			view.WhenOr(count > m.viewport.Height,
				view.NewFragment(m.help.FullHelpView(LocalServicesKeys.FullHelp())).WithStyle(helpStyle),
				view.NewFragment(m.help.ShortHelpView(LocalServicesKeys.ShortHelp())).WithStyle(helpStyle),
			),
			view.Break(),
		)
	}

	for _, warning := range m.Warnings {
		renderer.AddRow(view.NewFragment(warning)).WithStyle(warningStyle)
	}

	return renderer.Render()
}

func (m Model) getViewportContent() (int, string) {
	viewport := view.New()

	if m.StackState == nil {
		return 0, viewport.Render()
	}

	tables := m.StackState.Tables()
	rowCount := 0

	for _, tbl := range tables {
		rowCount = rowCount + len(tbl.Rows()) + 3 // # of rows + the header (2) + the break (2)
		viewport.AddRow(
			view.NewFragment(tbl.View()),
		)
	}

	return rowCount, viewport.Render()
}
