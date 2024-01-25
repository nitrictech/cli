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

package view

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/samber/lo"
)

// Row represents a row of UI fragments, i.e. breaks on either side.
type Row struct {
	style     lipgloss.Style
	Fragments []*Fragment
}

// Add an inline fragment to this row
func (r *Row) Add(fragments ...*Fragment) {
	r.Fragments = append(r.Fragments, fragments...)
}

// WithStyle applies/updates the style of the row
func (r *Row) WithStyle(style lipgloss.Style) *Row {
	r.style = style
	return r
}

// Render the row as a string
func (r *Row) Render() string {
	fragments := lo.Map(r.Fragments, func(f *Fragment, i int) string {
		return f.Render()
	})

	return r.style.Render(strings.Join(fragments, ""))
}
