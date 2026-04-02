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
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// WizardBase provides shared state for interactive wizard flows. Embed
// this struct in your model to get standard cursor, selection, step,
// abort, and spinner state management.
type WizardBase struct {
	// Selected tracks which items are checked in multi-select steps.
	Selected []bool

	// Spinner shows a loading animation during async steps.
	Spinner spinner.Model

	// Step is the current position in the wizard flow.
	Step int

	// Cursor is the zero-based index of the highlighted item.
	Cursor int

	// Aborted indicates whether the user cancelled the wizard.
	Aborted bool
}

// NewWizardBase creates a WizardBase with a spinner initialised to
// the standard Piko wizard style (dot spinner, blue foreground).
//
// Returns WizardBase which is ready to use.
func NewWizardBase() WizardBase {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	return WizardBase{Spinner: s}
}

// HandleAbort sets the Aborted flag and returns a quit command.
//
// Returns tea.Cmd which is tea.Quit.
func (w *WizardBase) HandleAbort() tea.Cmd {
	w.Aborted = true
	return tea.Quit
}

// HandleNavigation processes up/k and down/j keys against the given
// maximum cursor position (inclusive).
//
// Takes message (tea.KeyPressMsg) which is the key event to process.
// Takes maxCursor (int) which is the maximum cursor position (inclusive).
//
// Returns bool which is true if the key was handled.
func (w *WizardBase) HandleNavigation(message tea.KeyPressMsg, maxCursor int) bool {
	switch message.String() {
	case "up", "k":
		if w.Cursor > 0 {
			w.Cursor--
		}
		return true
	case "down", "j":
		if w.Cursor < maxCursor {
			w.Cursor++
		}
		return true
	}
	return false
}

// HandleToggle toggles the Selected flag at the current cursor
// position, provided the cursor is within the Selected slice bounds.
//
// Returns bool which is true if a toggle occurred.
func (w *WizardBase) HandleToggle() bool {
	if w.Cursor < len(w.Selected) {
		w.Selected[w.Cursor] = !w.Selected[w.Cursor]
		return true
	}
	return false
}

// AnySelected returns true if at least one item in Selected is checked.
//
// Returns bool which is true when any selection is active.
func (w *WizardBase) AnySelected() bool {
	for _, selected := range w.Selected {
		if selected {
			return true
		}
	}
	return false
}

// UpdateSpinner delegates a message to the spinner and returns the
// resulting command.
//
// Takes message (tea.Msg) which is the message to forward.
//
// Returns tea.Cmd which is the spinner's tick command.
func (w *WizardBase) UpdateSpinner(message tea.Msg) tea.Cmd {
	var command tea.Cmd
	w.Spinner, command = w.Spinner.Update(message)
	return command
}
