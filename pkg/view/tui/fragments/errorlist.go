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
	"github.com/charmbracelet/lipgloss"

	"github.com/nitrictech/cli/pkg/view/tui"
	"github.com/nitrictech/cli/pkg/view/tui/components/view"
)

type ErrorListOptions struct {
	heading string
}

type ErrorListOption = func(*ErrorListOptions) *ErrorListOptions

// WithCustomHeading sets a custom heading for the error list
func WithCustomHeading(heading string) ErrorListOption {
	return func(ol *ErrorListOptions) *ErrorListOptions {
		ol.heading = heading
		return ol
	}
}

func WithoutHeading(ol *ErrorListOptions) *ErrorListOptions {
	ol.heading = ""
	return ol
}

// ErrorList renders a list of errors as a dot point list
func ErrorList(errs []error, opts ...ErrorListOption) string {
	v := view.New()

	ol := &ErrorListOptions{
		heading: lipgloss.NewStyle().Width(10).Align(lipgloss.Center).Bold(true).Foreground(tui.Colors.White).Background(tui.Colors.Red).Render("Errors"),
	}

	for _, opt := range opts {
		ol = opt(ol)
	}

	for _, err := range errs {
		v.Addln(" - %s", err.Error()).WithStyle(lipgloss.NewStyle().Foreground(tui.Colors.Red))
	}

	return v.Render()
}
