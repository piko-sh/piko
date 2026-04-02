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
	"errors"
	"testing"
)

func TestAssertionError_Error(t *testing.T) {
	testCases := []struct {
		name     string
		err      AssertionError
		expected string
	}{
		{
			name: "with expected and actual",
			err: AssertionError{
				Selector: "#my-element",
				Expected: "hello",
				Actual:   "world",
				Message:  "text mismatch",
			},
			expected: `text mismatch: expected "hello", got "world" (selector: #my-element)`,
		},
		{
			name: "without expected and actual",
			err: AssertionError{
				Selector: ".button",
				Message:  "element not found",
			},
			expected: `element not found (selector: .button)`,
		},
		{
			name: "with only expected (actual empty)",
			err: AssertionError{
				Selector: "div",
				Expected: "foo",
				Actual:   "",
				Message:  "missing content",
			},
			expected: `missing content (selector: div)`,
		},
		{
			name: "with only actual (expected empty)",
			err: AssertionError{
				Selector: "span",
				Expected: "",
				Actual:   "bar",
				Message:  "unexpected content",
			},
			expected: `unexpected content (selector: span)`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.err.Error()
			if result != tc.expected {
				t.Errorf("AssertionError.Error() = %q, expected %q", result, tc.expected)
			}
		})
	}
}

func TestAssertionError_ImplementsError(t *testing.T) {

	var _ error = &AssertionError{}
}

func TestElementCountMatches(t *testing.T) {
	testCases := []struct {
		err      error
		name     string
		count    int
		expected int
		want     bool
	}{
		{name: "exact match", count: 3, expected: 3, err: nil, want: true},
		{name: "mismatch", count: 2, expected: 3, err: nil, want: false},
		{name: "expected zero with error", count: 0, expected: 0, err: errors.New("not found"), want: true},
		{name: "expected nonzero with error", count: 0, expected: 3, err: errors.New("not found"), want: false},
		{name: "at least one with count >= 1", count: 5, expected: -1, err: nil, want: true},
		{name: "at least one with count == 1", count: 1, expected: -1, err: nil, want: true},
		{name: "at least one with count == 0", count: 0, expected: -1, err: nil, want: false},
		{name: "zero count matches zero expected", count: 0, expected: 0, err: nil, want: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := elementCountMatches(tc.count, tc.expected, tc.err)
			if result != tc.want {
				t.Errorf("elementCountMatches(%d, %d, %v) = %v, want %v", tc.count, tc.expected, tc.err, result, tc.want)
			}
		})
	}
}

func TestNewElementCountError(t *testing.T) {
	testCases := []struct {
		name               string
		selector           string
		wantExpectedString string
		expected           int
		actual             int
	}{
		{
			name:               "standard count",
			selector:           ".item",
			expected:           3,
			actual:             5,
			wantExpectedString: "3 elements",
		},
		{
			name:               "at least one",
			selector:           ".item",
			expected:           -1,
			actual:             0,
			wantExpectedString: "at least 1 element",
		},
		{
			name:               "zero expected",
			selector:           ".gone",
			expected:           0,
			actual:             2,
			wantExpectedString: "0 elements",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := newElementCountError(tc.selector, tc.expected, tc.actual)
			if err.Selector != tc.selector {
				t.Errorf("Selector = %q, expected %q", err.Selector, tc.selector)
			}
			if err.Expected != tc.wantExpectedString {
				t.Errorf("Expected = %q, expected %q", err.Expected, tc.wantExpectedString)
			}
			if err.Message != "element count mismatch" {
				t.Errorf("Message = %q, expected %q", err.Message, "element count mismatch")
			}
		})
	}
}

func TestAssertionHandlers_AllKeysAreAssertionActions(t *testing.T) {
	for action := range assertionHandlers {
		if !IsAssertionAction(action) {
			t.Errorf("assertionHandlers has key %q but IsAssertionAction returns false", action)
		}
	}
}

func TestIsAssertionAction_NonAssertionActions(t *testing.T) {
	nonAssertions := []string{
		"click", "navigate", "fill", "press", "wait",
		"submit", "hover", "focus", "blur", "", "unknown",
	}
	for _, action := range nonAssertions {
		if IsAssertionAction(action) {
			t.Errorf("IsAssertionAction(%q) should be false", action)
		}
	}
}

func TestCheckConsoleMessage_WithPageHelper(t *testing.T) {
	ph := &PageHelper{
		stopChan:      make(chan struct{}),
		consoleLogs:   []string{},
		consoleLogsV2: []ConsoleLog{},
	}

	ph.recordConsoleLog("test error message", "error")
	ph.recordConsoleLog("test warning", "warn")
	ph.recordConsoleLog("info message", "info")

	ctx := &ActionContext{PageHelper: ph}

	if err := CheckConsoleMessage(ctx, "error", "error message"); err != nil {
		t.Errorf("expected match for error level + message, got: %v", err)
	}

	if err := CheckConsoleMessage(ctx, "warn", ""); err != nil {
		t.Errorf("expected match for warn level, got: %v", err)
	}

	if err := CheckConsoleMessage(ctx, "", "info"); err != nil {
		t.Errorf("expected match for info message, got: %v", err)
	}

	if err := CheckConsoleMessage(ctx, "debug", "not here"); err == nil {
		t.Error("expected error for non-matching console message")
	}
}

func TestCheckConsoleMessage_NilPageHelper(t *testing.T) {
	ctx := &ActionContext{}
	err := CheckConsoleMessage(ctx, "", "")
	if err == nil {
		t.Error("expected error when PageHelper is nil")
	}
}

func TestCheckNoConsoleMessage_NoMatch(t *testing.T) {
	ph := &PageHelper{
		stopChan:      make(chan struct{}),
		consoleLogs:   []string{},
		consoleLogsV2: []ConsoleLog{},
	}

	ph.recordConsoleLog("info log", "info")
	ctx := &ActionContext{PageHelper: ph}

	if err := CheckNoConsoleMessage(ctx, "error", ""); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestCheckNoConsoleMessage_MatchFound(t *testing.T) {
	ph := &PageHelper{
		stopChan:      make(chan struct{}),
		consoleLogs:   []string{},
		consoleLogsV2: []ConsoleLog{},
	}

	ph.recordConsoleLog("something bad", "error")
	ctx := &ActionContext{PageHelper: ph}

	if err := CheckNoConsoleMessage(ctx, "error", ""); err == nil {
		t.Error("expected error when matching console message is found")
	}
}

func TestCheckNoConsoleMessage_NilPageHelper(t *testing.T) {
	ctx := &ActionContext{}
	err := CheckNoConsoleMessage(ctx, "", "")
	if err == nil {
		t.Error("expected error when PageHelper is nil")
	}
}

func TestCheckNoConsoleErrors_NoErrors(t *testing.T) {
	ph := &PageHelper{
		stopChan:      make(chan struct{}),
		consoleLogs:   []string{},
		consoleLogsV2: []ConsoleLog{},
	}

	ph.recordConsoleLog("normal log", "info")
	ctx := &ActionContext{PageHelper: ph}

	if err := CheckNoConsoleErrors(ctx); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestCheckNoConsoleErrors_WithError(t *testing.T) {
	ph := &PageHelper{
		stopChan:      make(chan struct{}),
		consoleLogs:   []string{},
		consoleLogsV2: []ConsoleLog{},
	}

	ph.recordConsoleLog("error occurred", "error")
	ctx := &ActionContext{PageHelper: ph}

	if err := CheckNoConsoleErrors(ctx); err == nil {
		t.Error("expected error when console error exists")
	}
}

func TestCheckNoConsoleWarnings_NoWarnings(t *testing.T) {
	ph := &PageHelper{
		stopChan:      make(chan struct{}),
		consoleLogs:   []string{},
		consoleLogsV2: []ConsoleLog{},
	}

	ph.recordConsoleLog("normal log", "info")
	ctx := &ActionContext{PageHelper: ph}

	if err := CheckNoConsoleWarnings(ctx); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestCheckNoConsoleWarnings_WithWarning(t *testing.T) {
	ph := &PageHelper{
		stopChan:      make(chan struct{}),
		consoleLogs:   []string{},
		consoleLogsV2: []ConsoleLog{},
	}

	ph.recordConsoleLog("deprecated feature", "warn")
	ctx := &ActionContext{PageHelper: ph}

	if err := CheckNoConsoleWarnings(ctx); err == nil {
		t.Error("expected error when console warning exists")
	}
}
