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
	"strings"
	"testing"
)

func TestRenderCheckboxList_UncheckedItems(t *testing.T) {
	t.Parallel()

	items := []CheckboxItem{
		{Label: "Option A", Selected: false},
		{Label: "Option B", Selected: false},
	}
	result := RenderCheckboxList(items, 3)

	if !strings.Contains(result, "[ ] Option A") {
		t.Error("should contain unchecked Option A")
	}
	if !strings.Contains(result, "[ ] Option B") {
		t.Error("should contain unchecked Option B")
	}
	if !strings.Contains(result, "Continue") {
		t.Error("should contain Continue button")
	}
}

func TestRenderCheckboxList_CheckedItems(t *testing.T) {
	t.Parallel()

	items := []CheckboxItem{
		{Label: "Option A", Selected: true},
		{Label: "Option B", Selected: false},
	}
	result := RenderCheckboxList(items, 3)

	if !strings.Contains(result, "[x] Option A") {
		t.Error("should contain checked Option A")
	}
	if !strings.Contains(result, "[ ] Option B") {
		t.Error("should contain unchecked Option B")
	}
}

func TestRenderCheckboxList_CursorOnItem(t *testing.T) {
	t.Parallel()

	items := []CheckboxItem{
		{Label: "Option A", Selected: false},
		{Label: "Option B", Selected: false},
	}
	result := RenderCheckboxList(items, 0)

	if !strings.Contains(result, ">") {
		t.Error("should contain cursor indicator '>'")
	}
}

func TestRenderCheckboxList_CursorOnContinue(t *testing.T) {
	t.Parallel()

	items := []CheckboxItem{
		{Label: "Option A", Selected: false},
	}
	result := RenderCheckboxList(items, 1)

	if !strings.Contains(result, "Continue") {
		t.Error("should contain Continue button")
	}
	if !strings.Contains(result, ">") {
		t.Error("should contain cursor indicator on Continue")
	}
}

func TestRenderCheckboxList_EmptyItems(t *testing.T) {
	t.Parallel()

	result := RenderCheckboxList(nil, 0)

	if !strings.Contains(result, "Continue") {
		t.Error("should still contain Continue button even with no items")
	}
}

func TestRenderChoiceList_CursorPosition(t *testing.T) {
	t.Parallel()

	choices := []string{"First", "Second", "Third"}

	result := RenderChoiceList(choices, 1)

	if !strings.Contains(result, "First") {
		t.Error("should contain First")
	}
	if !strings.Contains(result, "Second") {
		t.Error("should contain Second")
	}
	if !strings.Contains(result, "Third") {
		t.Error("should contain Third")
	}
	if !strings.Contains(result, ">") {
		t.Error("should contain cursor indicator")
	}
}

func TestRenderChoiceList_AllChoicesPresent(t *testing.T) {
	t.Parallel()

	choices := []string{"Alpha", "Beta"}

	for i := range choices {
		result := RenderChoiceList(choices, i)
		for _, c := range choices {
			if !strings.Contains(result, c) {
				t.Errorf("cursor=%d: should contain %q", i, c)
			}
		}
	}
}

func TestRenderYesNo(t *testing.T) {
	t.Parallel()

	result := RenderYesNo(0)
	if !strings.Contains(result, "Yes") {
		t.Error("should contain Yes")
	}
	if !strings.Contains(result, "No") {
		t.Error("should contain No")
	}
	if !strings.Contains(result, ">") {
		t.Error("should contain cursor indicator")
	}
}

func TestRenderYesNo_CursorOnNo(t *testing.T) {
	t.Parallel()

	result := RenderYesNo(1)
	if !strings.Contains(result, "Yes") {
		t.Error("should contain Yes")
	}
	if !strings.Contains(result, "No") {
		t.Error("should contain No")
	}
}

func TestRenderSpinnerLine(t *testing.T) {
	t.Parallel()

	result := RenderSpinnerLine("⠋", "Loading...")
	if result != "⠋ Loading..." {
		t.Errorf("RenderSpinnerLine = %q, want %q", result, "⠋ Loading...")
	}
}
