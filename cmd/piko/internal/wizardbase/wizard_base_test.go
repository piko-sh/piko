// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package wizardbase

import (
	"testing"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
)

func TestNewWizardBase_SpinnerInitialised(t *testing.T) {
	t.Parallel()

	wb := NewWizardBase()
	if wb.Step != 0 {
		t.Errorf("Step = %d, want 0", wb.Step)
	}
	if wb.Cursor != 0 {
		t.Errorf("Cursor = %d, want 0", wb.Cursor)
	}
	if wb.Aborted {
		t.Error("Aborted should be false")
	}

	if wb.Spinner.View() == "" {
		t.Error("Spinner.View() should not be empty")
	}
}

func TestHandleAbort(t *testing.T) {
	t.Parallel()

	wb := NewWizardBase()
	command := wb.HandleAbort()

	if !wb.Aborted {
		t.Error("Aborted should be true after HandleAbort")
	}
	if command == nil {
		t.Error("HandleAbort should return a non-nil command")
	}
}

func TestHandleNavigation_Down(t *testing.T) {
	t.Parallel()

	wb := NewWizardBase()
	wb.Cursor = 0

	handled := wb.HandleNavigation(tea.KeyPressMsg{Code: 'j', Text: "j"}, 2)
	if !handled {
		t.Error("HandleNavigation should return true for 'j'")
	}
	if wb.Cursor != 1 {
		t.Errorf("Cursor = %d, want 1", wb.Cursor)
	}
}

func TestHandleNavigation_DownArrow(t *testing.T) {
	t.Parallel()

	wb := NewWizardBase()
	wb.Cursor = 0

	handled := wb.HandleNavigation(tea.KeyPressMsg{Code: tea.KeyDown, Text: "down"}, 2)
	if !handled {
		t.Error("HandleNavigation should return true for 'down'")
	}
	if wb.Cursor != 1 {
		t.Errorf("Cursor = %d, want 1", wb.Cursor)
	}
}

func TestHandleNavigation_Up(t *testing.T) {
	t.Parallel()

	wb := NewWizardBase()
	wb.Cursor = 2

	handled := wb.HandleNavigation(tea.KeyPressMsg{Code: 'k', Text: "k"}, 3)
	if !handled {
		t.Error("HandleNavigation should return true for 'k'")
	}
	if wb.Cursor != 1 {
		t.Errorf("Cursor = %d, want 1", wb.Cursor)
	}
}

func TestHandleNavigation_UpArrow(t *testing.T) {
	t.Parallel()

	wb := NewWizardBase()
	wb.Cursor = 2

	handled := wb.HandleNavigation(tea.KeyPressMsg{Code: tea.KeyUp, Text: "up"}, 3)
	if !handled {
		t.Error("HandleNavigation should return true for 'up'")
	}
	if wb.Cursor != 1 {
		t.Errorf("Cursor = %d, want 1", wb.Cursor)
	}
}

func TestHandleNavigation_AtUpperBound(t *testing.T) {
	t.Parallel()

	wb := NewWizardBase()
	wb.Cursor = 3

	handled := wb.HandleNavigation(tea.KeyPressMsg{Code: 'j', Text: "j"}, 3)
	if !handled {
		t.Error("HandleNavigation should return true for 'j'")
	}
	if wb.Cursor != 3 {
		t.Errorf("Cursor = %d, want 3 (should not exceed maxCursor)", wb.Cursor)
	}
}

func TestHandleNavigation_AtLowerBound(t *testing.T) {
	t.Parallel()

	wb := NewWizardBase()
	wb.Cursor = 0

	handled := wb.HandleNavigation(tea.KeyPressMsg{Code: 'k', Text: "k"}, 3)
	if !handled {
		t.Error("HandleNavigation should return true for 'k'")
	}
	if wb.Cursor != 0 {
		t.Errorf("Cursor = %d, want 0 (should not go below 0)", wb.Cursor)
	}
}

func TestHandleNavigation_UnhandledKey(t *testing.T) {
	t.Parallel()

	wb := NewWizardBase()
	wb.Cursor = 1

	handled := wb.HandleNavigation(tea.KeyPressMsg{Code: 'x', Text: "x"}, 3)
	if handled {
		t.Error("HandleNavigation should return false for unhandled key")
	}
	if wb.Cursor != 1 {
		t.Errorf("Cursor = %d, want 1 (should not change)", wb.Cursor)
	}
}

func TestHandleToggle_WithinBounds(t *testing.T) {
	t.Parallel()

	wb := NewWizardBase()
	wb.Selected = []bool{false, false, false}
	wb.Cursor = 1

	toggled := wb.HandleToggle()
	if !toggled {
		t.Error("HandleToggle should return true when cursor is within bounds")
	}
	if !wb.Selected[1] {
		t.Error("Selected[1] should be true after toggle")
	}

	toggled = wb.HandleToggle()
	if !toggled {
		t.Error("HandleToggle should return true on second toggle")
	}
	if wb.Selected[1] {
		t.Error("Selected[1] should be false after second toggle")
	}
}

func TestHandleToggle_OutOfBounds(t *testing.T) {
	t.Parallel()

	wb := NewWizardBase()
	wb.Selected = []bool{false, false}
	wb.Cursor = 2

	toggled := wb.HandleToggle()
	if toggled {
		t.Error("HandleToggle should return false when cursor is out of bounds")
	}
}

func TestHandleToggle_EmptySelected(t *testing.T) {
	t.Parallel()

	wb := NewWizardBase()
	wb.Cursor = 0

	toggled := wb.HandleToggle()
	if toggled {
		t.Error("HandleToggle should return false when Selected is nil")
	}
}

func TestAnySelected_NoneSelected(t *testing.T) {
	t.Parallel()

	wb := NewWizardBase()
	wb.Selected = []bool{false, false, false}

	if wb.AnySelected() {
		t.Error("AnySelected should return false when nothing is selected")
	}
}

func TestAnySelected_OneSelected(t *testing.T) {
	t.Parallel()

	wb := NewWizardBase()
	wb.Selected = []bool{false, true, false}

	if !wb.AnySelected() {
		t.Error("AnySelected should return true when at least one item is selected")
	}
}

func TestAnySelected_NilSlice(t *testing.T) {
	t.Parallel()

	wb := NewWizardBase()

	if wb.AnySelected() {
		t.Error("AnySelected should return false for nil slice")
	}
}

func TestUpdateSpinner_DoesNotPanic(t *testing.T) {
	t.Parallel()

	wb := NewWizardBase()
	command := wb.UpdateSpinner(spinner.TickMsg{})

	_ = command
}
