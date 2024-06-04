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

package local

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/nitrictech/cli/pkg/view/tui"
	viewr "github.com/nitrictech/cli/pkg/view/tui/components/view"
	"github.com/nitrictech/cli/pkg/view/tui/teax"
)

type LocalCloudStartModel struct {
	spinner spinner.Model
	status  CloudStartupStatus
}

type CloudStartupStatus int

const (
	Starting CloudStartupStatus = iota
	Done
)

type LocalCloudStartStatusMsg struct {
	Status CloudStartupStatus
}

var (
	titleStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(tui.Colors.White).
			Background(tui.Colors.Blue).
			MarginRight(2)
	spinnerStyle = lipgloss.NewStyle().Foreground(tui.Colors.Purple)
)

var _ tea.Model = &TuiModel{}

func (t *LocalCloudStartModel) Init() tea.Cmd {
	return tea.Batch(t.spinner.Tick)
}

func (t *LocalCloudStartModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch typ := msg.(type) {
	case tea.KeyMsg:
		keyMsg := msg.(tea.KeyMsg)

		switch keyMsg.String() {
		case "ctrl+c", "q":
			return t, teax.Quit
		}
	case LocalCloudStartStatusMsg:
		t.status = typ.Status

		if t.status == Done {
			return t, teax.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		t.spinner, cmd = t.spinner.Update(msg)
		cmds = append(cmds, cmd)
	default:
		break
	}

	return t, tea.Batch(cmds...)
}

func (t *LocalCloudStartModel) View() string {
	v := viewr.New()
	if t.status != Done {
		v.Add("%s Starting Local Cloud, if this is your first run this may take a few minutes", t.spinner.View())
	} else {
		v.Add("Local cloud started successfully")
	}

	return v.Render()
}

func NewLocalCloudStartModel() *LocalCloudStartModel {

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle

	return &LocalCloudStartModel{
		status:  Starting,
		spinner: s,
	}
}
