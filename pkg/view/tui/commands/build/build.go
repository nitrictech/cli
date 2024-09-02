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

package build

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/samber/lo"

	projservice "github.com/nitrictech/cli/pkg/project/service"
	tui "github.com/nitrictech/cli/pkg/view/tui"
	"github.com/nitrictech/cli/pkg/view/tui/components/view"
	"github.com/nitrictech/cli/pkg/view/tui/fragments"
	"github.com/nitrictech/cli/pkg/view/tui/reactive"
	"github.com/nitrictech/cli/pkg/view/tui/teax"
)

type Model struct {
	title               string
	serviceBuildUpdates map[string][]projservice.ServiceBuildUpdate
	windowSize          tea.WindowSizeMsg

	serviceBuildUpdatesChannel <-chan projservice.ServiceBuildUpdate

	spinner spinner.Model

	Err error
}

var _ tea.Model = (*Model)(nil)

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		reactive.AwaitChannel(m.serviceBuildUpdatesChannel),
		m.spinner.Tick,
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, tui.KeyMap.Quit):
			return m, teax.Quit
		}
	case tea.WindowSizeMsg:
		m.windowSize = msg
	case reactive.ChanMsg[projservice.ServiceBuildUpdate]:
		// channel closed, the build is complete.
		if !msg.Ok {
			return m, teax.Quit
		}

		if m.serviceBuildUpdates[msg.Value.ServiceName] == nil {
			m.serviceBuildUpdates[msg.Value.ServiceName] = make([]projservice.ServiceBuildUpdate, 0)
		}

		m.serviceBuildUpdates[msg.Value.ServiceName] = append(m.serviceBuildUpdates[msg.Value.ServiceName], msg.Value)

		if msg.Value.Err != nil {
			m.Err = msg.Value.Err
			return m, teax.Quit
		}

		// resubscribe to the messages originating channel
		return m, reactive.AwaitChannel(msg.Source)
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)

		return m, cmd
	}

	return m, nil
}

func (m *Model) AllDone() bool {
	for _, serviceUpdates := range m.serviceBuildUpdates {
		for _, update := range serviceUpdates {
			if update.Status == projservice.ServiceBuildStatus_Skipped {
				continue
			}

			if update.Status == projservice.ServiceBuildStatus_Complete {
				continue
			}

			if update.Status == projservice.ServiceBuildStatus_Error {
				continue
			}

			return false
		}
	}

	return true
}

func (m Model) View() string {
	// setting the max width prevents unexpected newlines when the text is wrapped automatically by the terminal.
	v := view.New(view.WithStyle(lipgloss.NewStyle().Width(m.windowSize.Width)))
	v.Add(fragments.Tag("build"))

	v.Add(fmt.Sprintf("  %s", m.title))

	if !m.AllDone() {
		v.Add(m.spinner.View())
	}

	v.Break()

	gap := strings.Builder{}
	for i := 0; i < fragments.TagWidth()+2; i++ {
		gap.WriteString(" ")
	}

	serviceNames := lo.Keys(m.serviceBuildUpdates)

	sort.Strings(serviceNames)

	serviceUpdates := view.New(view.WithStyle(lipgloss.NewStyle().MarginLeft(fragments.TagWidth() + 2)))
	serviceUpdates.Break()

	for _, serviceName := range serviceNames {
		service := m.serviceBuildUpdates[serviceName]

		if len(service) == 0 {
			continue
		}

		latestUpdate := service[len(service)-1]

		if latestUpdate.Status != projservice.ServiceBuildStatus_Skipped {
			statusColor := tui.Colors.Gray
			if latestUpdate.Status == projservice.ServiceBuildStatus_Complete {
				statusColor = tui.Colors.Green
			} else if latestUpdate.Status == projservice.ServiceBuildStatus_InProgress {
				statusColor = tui.Colors.Blue
			} else if latestUpdate.Status == projservice.ServiceBuildStatus_Error {
				statusColor = tui.Colors.Red
			}

			serviceUpdates.Add("%s ", serviceName)
			serviceUpdates.Addln(strings.ToLower(string(latestUpdate.Status))).WithStyle(lipgloss.NewStyle().Foreground(statusColor))
		}

		if m.Err != nil {
			for _, update := range service {
				messageLines := strings.Split(strings.TrimSpace(update.Message), "\n")
				if len(messageLines) > 0 && update.Status != projservice.ServiceBuildStatus_Complete && latestUpdate.Status != projservice.ServiceBuildStatus_Skipped {
					serviceUpdates.Addln("  %s", messageLines[len(messageLines)-1]).WithStyle(lipgloss.NewStyle().Foreground(tui.Colors.Gray))
				}
			}
		} else {
			messageLines := strings.Split(strings.TrimSpace(latestUpdate.Message), "\n")
			if len(messageLines) > 0 && latestUpdate.Status != projservice.ServiceBuildStatus_Complete && latestUpdate.Status != projservice.ServiceBuildStatus_Skipped {
				serviceUpdates.Addln("  %s", messageLines[len(messageLines)-1]).WithStyle(lipgloss.NewStyle().Foreground(tui.Colors.Gray))
			}
		}
	}

	v.Add(serviceUpdates.Render())

	return v.Render()
}

func NewModel(serviceBuildUpdates <-chan projservice.ServiceBuildUpdate, title string) Model {
	return Model{
		title:                      title,
		spinner:                    spinner.New(spinner.WithSpinner(spinner.Ellipsis)),
		serviceBuildUpdatesChannel: serviceBuildUpdates,
		serviceBuildUpdates:        make(map[string][]projservice.ServiceBuildUpdate),
	}
}
