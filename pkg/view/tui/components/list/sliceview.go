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

package list

// SliceView sliding window view of a list of items
// e.g. a list of items that can be displayed in a terminal window, but not all at once
// It keeps track of the cursor position and the view window, ensuring the cursor is always visible in the sub-slice of items
type SliceView[T any] struct {
	items []T
	// max is the maximum number of items to display, if 0 or less, it will display all items
	max int
	// cursor is the index of the currently selected item
	cursor int
	// viewCursor is the index of the first item displayed in the view
	viewCursor int
}

func (m *SliceView[T]) UpdateItems(items []T) {
	m.items = items
	m.cursor = 0
	m.viewCursor = 0
}

// NumDisplayed is the number of items that will be displayed
// based on max and total number of items
func (m SliceView[T]) NumDisplayed() int {
	if m.max <= 0 {
		return len(m.items)
	}

	return min(m.max, len(m.items))
}

func NewSliceView[T any](items []T) SliceView[T] {
	return SliceView[T]{
		items:      items,
		max:        len(items),
		cursor:     0,
		viewCursor: 0,
	}
}

func (m *SliceView[T]) scrollIntoView(i int) {
	vStart, vEnd := m.ViewBounds()

	if vEnd >= len(m.items) {
		m.viewCursor = len(m.items) - m.NumDisplayed()
	}

	if vStart < 0 {
		m.viewCursor = 0
	}

	if i < vStart {
		m.viewCursor = i
	} else if i >= vEnd {
		m.viewCursor = i - m.NumDisplayed() + 1
	}
}

// ViewBounds returns the start and end index of the view
func (m SliceView[T]) ViewBounds() (int, int) {
	return m.viewCursor, m.viewCursor + m.NumDisplayed()
}

func (m *SliceView[T]) IsInView(i int) bool {
	return i >= m.viewCursor && i < m.viewCursor+m.NumDisplayed()
}

// SetCursor sets the cursor to the specified index, updating the view as needed
// if the cursor is out of bounds, it will wrap around
func (m *SliceView[T]) SetCursor(cursor int) {
	// Clamp the cursor to the bounds of the items
	if cursor < 0 {
		cursor = len(m.items) - 1
	} else if cursor >= len(m.items) {
		cursor = 0
	}

	m.cursor = cursor

	m.scrollIntoView(cursor)
}

// CursorUp moves the cursor up, wrapping around if it reaches the top
// it also updates the view to ensure the cursor is always visible
func (m *SliceView[T]) CursorBack() {
	m.SetCursor(m.cursor - 1)
}

// CursorDown moves the cursor down, wrapping around if it reaches the bottom
// it also updates the view to ensure the cursor is always visible
func (m *SliceView[T]) CursorNext() {
	m.SetCursor(m.cursor + 1)
}

// Choice returns the currently selected item
func (m SliceView[T]) Choice() T {
	return m.items[m.cursor]
}

// SetMaxDisplayedItems sets the maximum number of items to display
// if max is 0 or less, it will display all items
func (m *SliceView[T]) SetMaxDisplayedItems(max int) {
	m.max = min(max, len(m.items))

	m.scrollIntoView(m.cursor)
}

// All returns all items
func (m SliceView[T]) All() []T {
	return m.items
}

// Cursor returns the index of the currently selected item
func (m SliceView[T]) Cursor() int {
	return m.cursor
}

// MaxDisplayedItems returns the maximum number of items to display
// Use NumDisplayed to get the actual number of items that will be displayed
func (m SliceView[T]) MaxDisplayedItems() int {
	return m.max
}

// IsChoice returns true if the index is the currently selected item
func (m SliceView[T]) IsChoice(i int) bool {
	return i == m.cursor
}

// IsChoiceRelative returns true if the index is the currently selected item
// where the index is relative to the view sub-slice
func (m SliceView[T]) IsChoiceRelative(relativeI int) bool {
	i := relativeI + m.viewCursor
	return m.IsChoice(i)
}

// View returns the items that should be displayed
func (m SliceView[T]) View() []T {
	if m.NumDisplayed() == len(m.items) {
		return m.items
	}

	return m.items[m.viewCursor : m.viewCursor+m.NumDisplayed()]
}

// ViewOffset returns the index of the first item displayed in the view
func (m SliceView[T]) ViewOffset() int {
	return m.viewCursor
}
