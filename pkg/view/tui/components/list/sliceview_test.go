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

package list_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nitrictech/cli/pkg/view/tui/components/list"
)

func TestNewSliceView(t *testing.T) {
	items := []string{"item1", "item2", "item3"}
	view := list.NewSliceView(items)

	assert.Equal(t, 3, view.NumDisplayed(), "Expected number of displayed items to be 3")
	assert.Equal(t, 0, view.Cursor(), "Expected initial cursor position to be 0")
	assert.Equal(t, items, view.All(), "Expected all items to match input items")
}

func TestCursorMovement(t *testing.T) {
	items := []string{"item1", "item2", "item3"}
	view := list.NewSliceView(items)

	view.CursorNext()
	assert.Equal(t, 1, view.Cursor(), "Expected cursor to move down to 1")

	view.CursorNext()
	assert.Equal(t, 2, view.Cursor(), "Expected cursor to move down to 2")

	view.CursorNext()
	assert.Equal(t, 0, view.Cursor(), "Expected cursor to wrap around to 0")

	view.CursorBack()
	assert.Equal(t, 2, view.Cursor(), "Expected cursor to wrap around up to 2")

	view.CursorBack()
	assert.Equal(t, 1, view.Cursor(), "Expected cursor to move up to 1")
}

func TestSetMaxDisplayedItems(t *testing.T) {
	items := []string{"item1", "item2", "item3", "item4", "item5"}
	view := list.NewSliceView(items)

	view.SetMaxDisplayedItems(3)
	assert.Equal(t, 3, view.MaxDisplayedItems(), "Expected max displayed items to be set to 3")
	assert.Equal(t, 3, view.NumDisplayed(), "Expected number of displayed items to be 3")
}

func TestShrinkGrowMaxDisplayedItems(t *testing.T) {
	items := []string{"item1", "item2", "item3", "item4", "item5"}
	view := list.NewSliceView(items)
	view.SetCursor(2)

	view.SetMaxDisplayedItems(4)
	assert.Equal(t, 4, view.NumDisplayed(), "Expected number of displayed items to be 4")
	assert.Equal(t, []string{"item1", "item2", "item3", "item4"}, view.View(), "Expected view to display the first 4 items")

	view.SetMaxDisplayedItems(3)
	assert.Equal(t, 3, view.NumDisplayed(), "Expected number of displayed items to be 3")
	assert.Equal(t, []string{"item1", "item2", "item3"}, view.View(), "Expect the view end to be trimmed by 1")

	view.SetMaxDisplayedItems(2)
	assert.Equal(t, 2, view.NumDisplayed(), "Expected number of displayed items to be 2")
	assert.Equal(t, []string{"item2", "item3"}, view.View(), "Expected view to move by 1 to keep the cursor in view")

	view.SetMaxDisplayedItems(1)
	assert.Equal(t, 1, view.NumDisplayed(), "Expected number of displayed items to be 1")
	assert.Equal(t, []string{"item3"}, view.View(), "Expected view to move by 1 to keep the cursor in view")

	view.SetMaxDisplayedItems(2)
	assert.Equal(t, 2, view.NumDisplayed(), "Expected number of displayed items to be 2")
	assert.Equal(t, []string{"item3", "item4"}, view.View(), "Expected view to expand by 1, keeping the cursor in view")

	view.SetMaxDisplayedItems(3)
	assert.Equal(t, 3, view.NumDisplayed(), "Expected number of displayed items to be 3")
	assert.Equal(t, []string{"item3", "item4", "item5"}, view.View(), "Expected view to expand by 1, keeping the cursor in view")

	view.SetMaxDisplayedItems(4)
	assert.Equal(t, 4, view.NumDisplayed(), "Expected number of displayed items to be 4")
	assert.Equal(t, []string{"item2", "item3", "item4", "item5"}, view.View(), "Expected view to move by 1 to keep the cursor in view, without going over the end")
}

func TestSetCursor(t *testing.T) {
	items := []string{"item1", "item2", "item3"}
	view := list.NewSliceView(items)

	view.SetCursor(2)
	assert.Equal(t, 2, view.Cursor(), "Expected cursor to be set to 2")
	assert.Equal(t, "item3", view.Choice(), "Expected choice to be 'item3'")
}

func TestView(t *testing.T) {
	items := []string{"item1", "item2", "item3", "item4", "item5"}
	view := list.NewSliceView(items)

	view.SetMaxDisplayedItems(3)
	assert.Equal(t, []string{"item1", "item2", "item3"}, view.View(), "Expected view to display the first 3 items")

	view.SetCursor(4)
	assert.Equal(t, "item5", view.Choice(), "Expected choice to be 'item5'")
	assert.Equal(t, []string{"item3", "item4", "item5"}, view.View(), "Expected view to update to show last 3 items")
}
