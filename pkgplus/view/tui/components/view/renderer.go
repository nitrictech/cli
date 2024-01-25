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

// Renderer helps build styled views with rows and inline fragments
//
// Example:
// const newView = view.New()
// newView.AddRow(
//
//	view.NewFragment(mySpinner.View()).WithStyle(),
//	view.NewFragment().WithStyle(),
//
// )
//
// newView.AddRow(
//
//	view.NewFragment(textPrompt.View()).WithStyle(),
//	view.NewFragment().WithStyle(),
//
// )
//
// newView.AddRow(
//
//	view.NewFragment().WithStyle(),
//	view.NewFragment().WithStyle(),
//
// )
type Renderer struct {
	style lipgloss.Style
	strings.Builder
	Rows []*Row
}

// AddRow add a new row of fragments to this rendered, similar to a display: block CSS element
func (r *Renderer) AddRow(fragments ...*Fragment) *Row {
	newRow := &Row{
		Fragments: lo.Compact(fragments),
	}
	r.Rows = append(r.Rows, newRow)

	return newRow
}

// TODO: make break configurable
var defaultLineBreak = NewFragment("\n")

// Break returns a standard fragment used for line breaks
func Break() *Fragment {
	return defaultLineBreak
}

// When conditionally returns a fragment if the bool is true
func When(when bool, trueFrag *Fragment) *Fragment {
	if when {
		return trueFrag
	}

	return nil
}

// WhenOr conditionally selects from two fragments, returning one
//
// when: the bool to use for determination of the correct fragment
// trueFrag: the fragment returned when true
// falseFrag: the fragment returned when false
func WhenOr(when bool, trueFrag *Fragment, falseFrag *Fragment) *Fragment {
	if when {
		return trueFrag
	}

	return falseFrag
}

// Render the entire view applying all styles and row breaks and returning the resulting string
func (r *Renderer) Render() string {
	rows := lo.Map(r.Rows, func(r *Row, i int) string {
		return r.Render()
	})

	return r.style.Render(strings.Join(rows, "\n"))
}

type RendererOption = func(r *Renderer)

// WithStyle applies/updates the style of the view
func WithStyle(style lipgloss.Style) RendererOption {
	return func(r *Renderer) {
		r.style = style
	}
}

// WithStyle applies/updates the style of the view
func (r *Renderer) WithStyle(style lipgloss.Style) *Renderer {
	r.style = style
	return r
}

// New creates a new view renderer
func New(options ...RendererOption) *Renderer {
	renderer := &Renderer{
		Rows: []*Row{},
	}

	for _, o := range options {
		o(renderer)
	}

	return renderer
}
