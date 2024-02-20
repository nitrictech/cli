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

package teax

import tea "github.com/charmbracelet/bubbletea"

type fullHeightModel struct {
	tea.Model
	quitting bool
}

var _ tea.Model = fullHeightModel{}

func (q fullHeightModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case QuitMsg:
		q.quitting = true

		return q, tea.Quit
	}

	var cmd tea.Cmd
	q.Model, cmd = q.Model.Update(msg)

	return q, cmd
}

func (q fullHeightModel) FullView() string {
	return q.Model.View()
}

func (q fullHeightModel) View() string {
	if q.quitting {
		return ""
	}

	return q.Model.View()
}
