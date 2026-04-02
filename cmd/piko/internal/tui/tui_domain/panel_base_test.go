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

package tui_domain

import (
	"strings"
	"testing"
)

func TestNewBasePanel(t *testing.T) {
	panel := NewBasePanel("test-id", "Test Title")

	if panel.ID() != "test-id" {
		t.Errorf("expected ID 'test-id', got %q", panel.ID())
	}
	if panel.Title() != "Test Title" {
		t.Errorf("expected Title 'Test Title', got %q", panel.Title())
	}
	if panel.Focused() {
		t.Error("expected not focused initially")
	}
	if panel.Width() != 0 {
		t.Error("expected width 0 initially")
	}
	if panel.Height() != 0 {
		t.Error("expected height 0 initially")
	}
	if panel.Cursor() != 0 {
		t.Error("expected cursor 0 initially")
	}
	if panel.ScrollOffset() != 0 {
		t.Error("expected scrollOffset 0 initially")
	}
}

func TestBasePanel_SetFocused(t *testing.T) {
	panel := NewBasePanel("id", "Title")

	panel.SetFocused(true)
	if !panel.Focused() {
		t.Error("expected focused after SetFocused(true)")
	}

	panel.SetFocused(false)
	if panel.Focused() {
		t.Error("expected not focused after SetFocused(false)")
	}
}

func TestBasePanel_SetSize(t *testing.T) {
	panel := NewBasePanel("id", "Title")

	panel.SetSize(100, 50)
	if panel.Width() != 100 {
		t.Errorf("expected width 100, got %d", panel.Width())
	}
	if panel.Height() != 50 {
		t.Errorf("expected height 50, got %d", panel.Height())
	}
}

func TestBasePanel_ContentDimensions(t *testing.T) {
	panel := NewBasePanel("id", "Title")
	panel.SetSize(100, 50)

	contentWidth := panel.ContentWidth()
	contentHeight := panel.ContentHeight()

	if contentWidth >= 100 {
		t.Errorf("expected contentWidth < 100, got %d", contentWidth)
	}
	if contentHeight >= 50 {
		t.Errorf("expected contentHeight < 50, got %d", contentHeight)
	}
}

func TestBasePanel_ContentDimensions_Minimum(t *testing.T) {
	panel := NewBasePanel("id", "Title")
	panel.SetSize(2, 2)

	if panel.ContentWidth() < 0 {
		t.Error("ContentWidth should not be negative")
	}
	if panel.ContentHeight() < 0 {
		t.Error("ContentHeight should not be negative")
	}
}

func TestBasePanel_CursorAndScroll(t *testing.T) {
	panel := NewBasePanel("id", "Title")

	panel.SetCursor(5)
	if panel.Cursor() != 5 {
		t.Errorf("expected cursor 5, got %d", panel.Cursor())
	}

	panel.SetScrollOffset(10)
	if panel.ScrollOffset() != 10 {
		t.Errorf("expected scrollOffset 10, got %d", panel.ScrollOffset())
	}
}

func TestBasePanel_KeyMap(t *testing.T) {
	panel := NewBasePanel("id", "Title")

	bindings := []KeyBinding{
		{Key: "j", Description: "Down"},
		{Key: "k", Description: "Up"},
	}
	panel.SetKeyMap(bindings)

	result := panel.KeyMap()
	if len(result) != 2 {
		t.Errorf("expected 2 bindings, got %d", len(result))
	}
}

func TestBasePanel_Init(t *testing.T) {
	panel := NewBasePanel("id", "Title")

	command := panel.Init()
	if command != nil {
		t.Error("expected nil command from Init")
	}
}

func TestBasePanel_HandleNavigation(t *testing.T) {
	testCases := []struct {
		name           string
		key            string
		initialCursor  int
		itemCount      int
		expectedCursor int
		handled        bool
	}{
		{
			name:           "move down with j",
			key:            "j",
			initialCursor:  0,
			itemCount:      5,
			expectedCursor: 1,
			handled:        true,
		},
		{
			name:           "move up with k",
			key:            "k",
			initialCursor:  2,
			itemCount:      5,
			expectedCursor: 1,
			handled:        true,
		},
		{
			name:           "go to top with g",
			key:            "g",
			initialCursor:  3,
			itemCount:      5,
			expectedCursor: 0,
			handled:        true,
		},
		{
			name:           "go to bottom with G",
			key:            "G",
			initialCursor:  0,
			itemCount:      5,
			expectedCursor: 4,
			handled:        true,
		},
		{
			name:           "cannot go below first item",
			key:            "k",
			initialCursor:  0,
			itemCount:      5,
			expectedCursor: 0,
			handled:        true,
		},
		{
			name:           "cannot go above last item",
			key:            "j",
			initialCursor:  4,
			itemCount:      5,
			expectedCursor: 4,
			handled:        true,
		},
		{
			name:           "unknown key not handled",
			key:            "x",
			initialCursor:  2,
			itemCount:      5,
			expectedCursor: 2,
			handled:        false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			panel := NewBasePanel("id", "Title")
			panel.SetSize(80, 20)
			panel.SetCursor(tc.initialCursor)

			message := createTestKeyMessage(tc.key)
			handled := panel.HandleNavigation(message, tc.itemCount)

			if handled != tc.handled {
				t.Errorf("expected handled=%v, got %v", tc.handled, handled)
			}
			if panel.Cursor() != tc.expectedCursor {
				t.Errorf("expected cursor=%d, got %d", tc.expectedCursor, panel.Cursor())
			}
		})
	}
}

func TestBasePanel_HandleNavigation_PageUpDown(t *testing.T) {
	panel := NewBasePanel("id", "Title")
	panel.SetSize(80, 20)

	itemCount := 50
	contentHeight := panel.ContentHeight()

	panel.SetCursor(0)
	message := createTestKeyMessage("pgdown")
	panel.HandleNavigation(message, itemCount)
	if panel.Cursor() != contentHeight {
		t.Errorf("page down: expected cursor=%d, got %d", contentHeight, panel.Cursor())
	}

	panel.SetCursor(contentHeight * 2)
	message = createTestKeyMessage("pgup")
	panel.HandleNavigation(message, itemCount)
	if panel.Cursor() != contentHeight {
		t.Errorf("page up: expected cursor=%d, got %d", contentHeight, panel.Cursor())
	}
}

func TestBasePanel_RenderFrame(t *testing.T) {
	panel := NewBasePanel("id", "Test Panel")
	panel.SetSize(40, 10)

	content := "Line 1\nLine 2\nLine 3"
	result := panel.RenderFrame(content)

	if result == "" {
		t.Error("expected non-empty frame")
	}
	if !strings.Contains(result, "Test Panel") {
		t.Error("expected frame to contain title")
	}
}

func TestBasePanel_RenderFrame_Focused(t *testing.T) {
	panel := NewBasePanel("id", "Test Panel")
	panel.SetSize(40, 10)
	panel.SetFocused(true)

	content := "Content"
	result := panel.RenderFrame(content)

	if result == "" {
		t.Error("expected non-empty frame")
	}
}

func TestStatusIndicator(t *testing.T) {
	testCases := []struct {
		status ResourceStatus
	}{
		{status: ResourceStatusHealthy},
		{status: ResourceStatusDegraded},
		{status: ResourceStatusUnhealthy},
		{status: ResourceStatusPending},
		{status: ResourceStatusUnknown},
	}

	for _, tc := range testCases {
		t.Run(tc.status.String(), func(t *testing.T) {
			result := StatusIndicator(tc.status)
			if result == "" {
				t.Error("expected non-empty indicator")
			}
		})
	}
}

func TestStatusStyle(t *testing.T) {
	testCases := []struct {
		status ResourceStatus
	}{
		{status: ResourceStatusHealthy},
		{status: ResourceStatusDegraded},
		{status: ResourceStatusUnhealthy},
		{status: ResourceStatusPending},
		{status: ResourceStatusUnknown},
	}

	for _, tc := range testCases {
		t.Run(tc.status.String(), func(t *testing.T) {
			style := StatusStyle(tc.status)

			result := style.Render("test")
			if result == "" {
				t.Error("expected styled output")
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
		maxWidth int
	}{
		{
			name:     "no truncation needed",
			input:    "short",
			maxWidth: 10,
			expected: "short",
		},
		{
			name:     "exact length",
			input:    "exact",
			maxWidth: 5,
			expected: "exact",
		},
		{
			name:     "needs truncation",
			input:    "this is a long string",
			maxWidth: 10,
			expected: "this is...",
		},
		{
			name:     "very short max width",
			input:    "hello",
			maxWidth: 2,
			expected: "he",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := TruncateString(tc.input, tc.maxWidth)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestPadRight(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		width    int
		checkLen int
	}{
		{
			name:     "needs padding",
			input:    "hi",
			width:    5,
			checkLen: 5,
		},
		{
			name:     "exact width",
			input:    "hello",
			width:    5,
			checkLen: 5,
		},
		{
			name:     "longer than width truncates",
			input:    "hello world",
			width:    5,
			checkLen: 5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := PadRight(tc.input, tc.width)

			if result == "" {
				t.Error("expected non-empty result")
			}
		})
	}
}

func TestRenderMetadataRow(t *testing.T) {
	testCases := []struct {
		name     string
		key      string
		value    string
		selected bool
		focused  bool
	}{
		{
			name:     "not selected",
			key:      "Name",
			value:    "test-value",
			selected: false,
			focused:  false,
		},
		{
			name:     "selected not focused",
			key:      "Status",
			value:    "running",
			selected: true,
			focused:  false,
		},
		{
			name:     "selected and focused",
			key:      "Type",
			value:    "deployment",
			selected: true,
			focused:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := MetadataRowConfig{
				Selected:     tc.selected,
				Focused:      tc.focused,
				ContentWidth: 80,
			}
			result := RenderMetadataRow(tc.key, tc.value, config)

			if result == "" {
				t.Error("expected non-empty result")
			}
			if !strings.Contains(result, tc.key) {
				t.Errorf("expected result to contain key %q", tc.key)
			}
		})
	}
}

func TestRenderMetadataRow_LongValue(t *testing.T) {
	config := MetadataRowConfig{
		ContentWidth: 30,
	}
	longValue := "this is a very long value that should be truncated"
	result := RenderMetadataRow("Key", longValue, config)

	if result == "" {
		t.Error("expected non-empty result")
	}
}
