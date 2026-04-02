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

package browser_provider_chromedp

import (
	"strings"
	"testing"
	"time"
)

func TestHandleSetFiles_NoFilesNoValue(t *testing.T) {
	step := &BrowserStep{Action: "setFiles", Selector: "input[type=file]"}
	err := handleSetFiles(nil, step)
	if err == nil {
		t.Fatal("expected error when no files and no value")
	}
	if !strings.Contains(err.Error(), "setFiles requires") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleWait_InvalidValue(t *testing.T) {
	testCases := []struct {
		name  string
		value string
	}{
		{name: "empty string", value: ""},
		{name: "non-numeric", value: "abc"},
		{name: "float", value: "1.5"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			step := &BrowserStep{Action: "wait", Value: tc.value}
			err := handleWait(nil, step)
			if err == nil {
				t.Fatal("expected error for invalid wait value")
			}
			if !strings.Contains(err.Error(), "invalid wait value") {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestHandleWait_ValidZero(t *testing.T) {
	step := &BrowserStep{Action: "wait", Value: "0"}
	err := handleWait(nil, step)
	if err != nil {
		t.Fatalf("unexpected error for wait 0: %v", err)
	}
}

func TestHandlePress_NoKeyNoValue(t *testing.T) {
	step := &BrowserStep{Action: "press"}
	err := handlePress(nil, step)
	if err == nil {
		t.Fatal("expected error when no key and no value")
	}
	if !strings.Contains(err.Error(), "press requires") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleType_EmptyValue(t *testing.T) {
	step := &BrowserStep{Action: "type"}
	err := handleType(nil, step)
	if err == nil {
		t.Fatal("expected error when value is empty")
	}
	if !strings.Contains(err.Error(), "type requires") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleKeyDown_NoKeyNoValue(t *testing.T) {
	step := &BrowserStep{Action: "keyDown"}
	err := handleKeyDown(nil, step)
	if err == nil {
		t.Fatal("expected error when no key and no value")
	}
	if !strings.Contains(err.Error(), "keyDown requires") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleKeyUp_NoKeyNoValue(t *testing.T) {
	step := &BrowserStep{Action: "keyUp"}
	err := handleKeyUp(nil, step)
	if err == nil {
		t.Fatal("expected error when no key and no value")
	}
	if !strings.Contains(err.Error(), "keyUp requires") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleSetCursor_EmptySelector(t *testing.T) {
	step := &BrowserStep{Action: "setCursor"}
	err := handleSetCursor(nil, step)
	if err == nil {
		t.Fatal("expected error when selector is empty")
	}
	if !strings.Contains(err.Error(), "setCursor requires") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleSetSelection_EmptySelector(t *testing.T) {
	step := &BrowserStep{Action: "setSelection"}
	err := handleSetSelection(nil, step)
	if err == nil {
		t.Fatal("expected error when selector is empty")
	}
	if !strings.Contains(err.Error(), "setSelection requires") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleSelectAll_EmptySelector(t *testing.T) {
	step := &BrowserStep{Action: "selectAll"}
	err := handleSelectAll(nil, step)
	if err == nil {
		t.Fatal("expected error when selector is empty")
	}
	if !strings.Contains(err.Error(), "selectAll requires") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleScrollIntoView_EmptySelector(t *testing.T) {
	step := &BrowserStep{Action: "scrollIntoView"}
	err := handleScrollIntoView(nil, step)
	if err == nil {
		t.Fatal("expected error when selector is empty")
	}
	if !strings.Contains(err.Error(), "scrollIntoView requires") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleSetAttribute_EmptySelector(t *testing.T) {
	step := &BrowserStep{Action: "setAttribute", Name: "data-foo", Value: "bar"}
	err := handleSetAttribute(nil, step)
	if err == nil {
		t.Fatal("expected error when selector is empty")
	}
	if !strings.Contains(err.Error(), "setAttribute requires 'selector'") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleSetAttribute_EmptyName(t *testing.T) {
	step := &BrowserStep{Action: "setAttribute", Selector: "#el", Value: "bar"}
	err := handleSetAttribute(nil, step)
	if err == nil {
		t.Fatal("expected error when name is empty")
	}
	if !strings.Contains(err.Error(), "setAttribute requires 'name'") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleRemoveAttribute_EmptySelector(t *testing.T) {
	step := &BrowserStep{Action: "removeAttribute", Name: "data-foo"}
	err := handleRemoveAttribute(nil, step)
	if err == nil {
		t.Fatal("expected error when selector is empty")
	}
	if !strings.Contains(err.Error(), "removeAttribute requires 'selector'") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleRemoveAttribute_EmptyName(t *testing.T) {
	step := &BrowserStep{Action: "removeAttribute", Selector: "#el"}
	err := handleRemoveAttribute(nil, step)
	if err == nil {
		t.Fatal("expected error when name is empty")
	}
	if !strings.Contains(err.Error(), "removeAttribute requires 'name'") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleSetViewport_InvalidDimensions(t *testing.T) {
	testCases := []struct {
		name        string
		errContains string
		width       int
		height      int
	}{
		{name: "zero width", width: 0, height: 600, errContains: "positive 'width'"},
		{name: "negative width", width: -1, height: 600, errContains: "positive 'width'"},
		{name: "zero height", width: 800, height: 0, errContains: "positive 'height'"},
		{name: "negative height", width: 800, height: -1, errContains: "positive 'height'"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			step := &BrowserStep{Action: "setViewport", Width: tc.width, Height: tc.height}
			err := handleSetViewport(nil, step)
			if err == nil {
				t.Fatal("expected error for invalid dimensions")
			}
			if !strings.Contains(err.Error(), tc.errContains) {
				t.Errorf("expected error containing %q, got: %v", tc.errContains, err)
			}
		})
	}
}

func TestHandleComment_ReturnsNil(t *testing.T) {
	err := handleComment(nil, &BrowserStep{Action: "comment", Value: "some comment"})
	if err != nil {
		t.Fatalf("handleComment should return nil, got: %v", err)
	}
}

func TestHandleClearConsole_NilPageHelper(t *testing.T) {
	ctx := &ActionContext{}
	err := handleClearConsole(ctx, &BrowserStep{})
	if err != nil {
		t.Fatalf("handleClearConsole with nil PageHelper should not error, got: %v", err)
	}
}

func TestGetStepTimeout_Unit(t *testing.T) {
	testCases := []struct {
		name     string
		timeout  int
		expected time.Duration
	}{
		{name: "zero uses default", timeout: 0, expected: DefaultAssertionTimeout},
		{name: "negative uses default", timeout: -1, expected: DefaultAssertionTimeout},
		{name: "positive uses custom", timeout: 3000, expected: 3000 * time.Millisecond},
		{name: "small positive", timeout: 100, expected: 100 * time.Millisecond},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			step := &BrowserStep{Timeout: tc.timeout}
			result := getStepTimeout(step)
			if result != tc.expected {
				t.Errorf("getStepTimeout() = %v, expected %v", result, tc.expected)
			}
		})
	}
}

func TestExecuteStep_UnknownAction(t *testing.T) {
	ctx := &ActionContext{}
	step := &BrowserStep{Action: "nonExistentAction"}
	err := ExecuteStep(ctx, step)
	if err == nil {
		t.Fatal("expected error for unknown action")
	}
	if !strings.Contains(err.Error(), "unknown or unhandled action") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExecuteAssertion_UnknownAction(t *testing.T) {
	ctx := &ActionContext{}
	step := &BrowserStep{Action: "nonExistentAssertion"}
	err := ExecuteAssertion(ctx, step)
	if err == nil {
		t.Fatal("expected error for unknown assertion action")
	}
	if !strings.Contains(err.Error(), "unknown assertion action") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestStepHandlers_AllNonNil(t *testing.T) {
	for action, handler := range stepHandlers {
		if handler == nil {
			t.Errorf("stepHandlers[%q] is nil", action)
		}
	}
}

func TestAssertionHandlers_AllNonNil(t *testing.T) {
	for action, handler := range assertionHandlers {
		if handler == nil {
			t.Errorf("assertionHandlers[%q] is nil", action)
		}
	}
}

func TestAssertionHandlers_SyncWithIsAssertionAction(t *testing.T) {
	for action := range assertionHandlers {
		if !IsAssertionAction(action) {
			t.Errorf("assertionHandlers has key %q but IsAssertionAction returns false", action)
		}
	}
}

func TestHandlePress_UsesValueWhenKeyEmpty(t *testing.T) {

	step := &BrowserStep{Action: "press", Key: "Enter"}

	resolved := step.Key
	if resolved == "" {
		resolved = step.Value
	}
	if resolved == "" {
		t.Error("expected resolved key to be non-empty")
	}

	step2 := &BrowserStep{Action: "press", Key: "", Value: "Tab"}
	resolved2 := step2.Key
	if resolved2 == "" {
		resolved2 = step2.Value
	}
	if resolved2 != "Tab" {
		t.Errorf("expected resolved key to be %q, got %q", "Tab", resolved2)
	}
}
