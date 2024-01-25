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
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Fragment represents a UI element
type Fragment struct {
	content any
	styles  []lipgloss.Style
}

// Render this fragment as a string, applying its style
func (f Fragment) Render() string {
	rendered := fmt.Sprint(f.content)

	for _, style := range f.styles {
		rendered = style.Render(rendered)
	}

	return rendered
}

// String returns the rendered fragment as a string
func (f Fragment) String() string {
	return f.Render()
}

// WithStyle adds a style to this fragment, which will be used when rendering
func (f *Fragment) WithStyle(style ...lipgloss.Style) *Fragment {
	f.styles = style
	return f
}

// NewFragment constructs a new fragment from its un-styled content
func NewFragment(content any) *Fragment {
	return &Fragment{
		content: content,
	}
}
