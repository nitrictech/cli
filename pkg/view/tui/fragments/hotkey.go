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

package fragments

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/nitrictech/cli/pkg/view/tui"
	"github.com/nitrictech/cli/pkg/view/tui/components/view"
)

// Hotkey renders a hotkey fragment e.g. q: quit
func Hotkey(key string, description string) string {
	keyView := view.NewFragment(fmt.Sprintf("%s:", key)).WithStyle(lipgloss.NewStyle().Foreground(tui.Colors.Text)).Render()
	descriptionView := view.NewFragment(description).WithStyle(lipgloss.NewStyle().Foreground(tui.Colors.TextMuted)).Render()

	return fmt.Sprintf("%s %s", keyView, descriptionView)
}
