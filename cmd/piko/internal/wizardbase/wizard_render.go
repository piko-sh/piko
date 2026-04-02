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
	"fmt"
	"strings"
)

const (
	// twoStringFmt is the format string for joining two strings with a space.
	twoStringFmt = "%s %s"
)

// CheckboxItem represents a single item in a checkbox list.
type CheckboxItem struct {
	// Label is the display text for the item.
	Label string

	// Selected indicates whether the item is checked.
	Selected bool
}

// RenderCheckboxList renders a multi-select checkbox list with a "Continue"
// button, including cursor highlighting and check/uncheck indicators.
//
// Takes items ([]CheckboxItem) which are the items to render.
// Takes cursor (int) which is the currently highlighted position.
//
// Returns string which is the rendered list ready for display.
func RenderCheckboxList(items []CheckboxItem, cursor int) string {
	var s strings.Builder
	for i, item := range items {
		current := " "
		if cursor == i {
			current = CursorStyle.Render(">")
		}
		check := "[ ]"
		if item.Selected {
			check = "[x]"
		}
		line := fmt.Sprintf("%s %s %s", current, check, item.Label)
		if cursor == i {
			_, _ = fmt.Fprintf(&s, "%s", SelectedStyle.Render(line))
		} else {
			s.WriteString(line)
		}
		s.WriteString("\n")
	}
	s.WriteString("\n")
	continueIndex := len(items)
	if cursor == continueIndex {
		_, _ = fmt.Fprintf(&s, "%s", SelectedStyle.Render(
			fmt.Sprintf(twoStringFmt, CursorStyle.Render(">"), "Continue"),
		))
	} else {
		s.WriteString("  Continue")
	}
	s.WriteString("\n")
	return s.String()
}

// RenderChoiceList renders a single-select choice list with cursor
// highlighting.
//
// Takes choices ([]string) which are the options to display.
// Takes cursor (int) which is the highlighted position.
//
// Returns string which is the rendered list.
func RenderChoiceList(choices []string, cursor int) string {
	var s strings.Builder
	for i, choice := range choices {
		current := " "
		if cursor == i {
			current = CursorStyle.Render(">")
			_, _ = fmt.Fprintf(&s, "%s", SelectedStyle.Render(
				fmt.Sprintf(twoStringFmt, current, choice),
			))
		} else {
			_, _ = fmt.Fprintf(&s, twoStringFmt, current, choice)
		}
		s.WriteString("\n")
	}
	return s.String()
}

// RenderYesNo renders a binary Yes/No choice with cursor highlighting.
//
// Takes cursor (int) which is the highlighted position (0 = Yes, 1 = No).
//
// Returns string which is the rendered Yes/No prompt.
func RenderYesNo(cursor int) string {
	return RenderChoiceList([]string{"Yes", "No"}, cursor)
}

// RenderSpinnerLine renders a spinner followed by a message.
//
// Takes spinnerView (string) which is the spinner's rendered output.
// Takes message (string) which is the text to display next to the spinner.
//
// Returns string which is the combined spinner and message.
func RenderSpinnerLine(spinnerView, message string) string {
	return fmt.Sprintf(twoStringFmt, spinnerView, message)
}
