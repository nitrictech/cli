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

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// FullViewProgram is a program that will print the full view for the model as the program terminates.
//
// Bubbletea programs limit the output of the view to the terminal size, which fixes issues with rerendering,
// but results in any off-screen output being lost when the program exits.
type FullViewProgram struct {
	*tea.Program
}

func (p *FullViewProgram) Run() (tea.Model, error) {
	model, err := p.Program.Run()

	tea.Batch()

	quittingModel := model.(fullHeightModel)

	quittingModel.quitting = false
	fmt.Println(quittingModel.View())

	return quittingModel.Model, err
}

func NewProgram(model tea.Model, opts ...tea.ProgramOption) *FullViewProgram {
	return &FullViewProgram{tea.NewProgram(fullHeightModel{model, false}, opts...)}
}
