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

const (
	ansiRedFg   = "\x1b[31m"
	ansiBlueFg  = "\x1b[34m"
	ansiBoldOn  = "\x1b[1m"
	ansiResetFg = "\x1b[0m"
)

func TestTextWidth(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected int
	}{
		{name: "empty", input: "", expected: 0},
		{name: "ascii", input: "hello", expected: 5},
		{name: "ansi only", input: ansiRedFg + ansiResetFg, expected: 0},
		{name: "ansi wrapped ascii", input: ansiRedFg + "abc" + ansiResetFg, expected: 3},
		{name: "ansi wrapped multi-style", input: ansiRedFg + "ab" + ansiBoldOn + "cd" + ansiResetFg, expected: 4},
		{name: "emoji single", input: "📄", expected: 2},
		{name: "emoji with text", input: "file 📄 ok", expected: 10},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := TextWidth(tc.input)
			if got != tc.expected {
				t.Errorf("TextWidth(%q) = %d, want %d", tc.input, got, tc.expected)
			}
		})
	}
}

func TestTruncateANSIPlain(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
		maxWidth int
	}{
		{name: "no truncation needed", input: "short", maxWidth: 10, expected: "short"},
		{name: "exact length", input: "exact", maxWidth: 5, expected: "exact"},
		{name: "needs truncation", input: "this is a long string", maxWidth: 10, expected: "this is..."},
		{name: "very short max width", input: "hello", maxWidth: 2, expected: "he"},
		{name: "zero max width", input: "hello", maxWidth: 0, expected: ""},
		{name: "negative max width", input: "hello", maxWidth: -1, expected: ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := TruncateANSI(tc.input, tc.maxWidth)
			if got != tc.expected {
				t.Errorf("TruncateANSI(%q, %d) = %q, want %q", tc.input, tc.maxWidth, got, tc.expected)
			}
		})
	}
}

func TestTruncateANSIWide(t *testing.T) {

	threeEmojis := "📄🌐🔌"
	if TextWidth(threeEmojis) != 6 {
		t.Fatalf("setup: expected width 6, got %d", TextWidth(threeEmojis))
	}

	got := TruncateANSI(threeEmojis, 4)
	if TextWidth(got) > 4 {
		t.Errorf("TruncateANSI(threeEmojis, 4) produced width %d > 4: %q", TextWidth(got), got)
	}
}

func TestTruncateANSIStyled(t *testing.T) {
	styled := ansiRedFg + "this is a long string" + ansiResetFg

	got := TruncateANSI(styled, 10)
	if TextWidth(got) > 10 {
		t.Errorf("truncated styled string has width %d, want <=10: %q", TextWidth(got), got)
	}
	if !strings.Contains(got, ansiRedFg) {
		t.Errorf("expected truncated output to retain SGR open sequence, got %q", got)
	}
}

func TestPadRightANSI(t *testing.T) {
	testCases := []struct {
		name      string
		input     string
		width     int
		wantWidth int
	}{
		{name: "pad short", input: "hi", width: 5, wantWidth: 5},
		{name: "exact", input: "hello", width: 5, wantWidth: 5},
		{name: "truncate long", input: "hello world", width: 5, wantWidth: 5},
		{name: "zero width", input: "hello", width: 0, wantWidth: 0},
		{name: "ansi short", input: ansiBlueFg + "hi" + ansiResetFg, width: 5, wantWidth: 5},
		{name: "emoji short", input: "📄", width: 5, wantWidth: 5},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := PadRightANSI(tc.input, tc.width)
			gotWidth := TextWidth(got)
			if gotWidth != tc.wantWidth {
				t.Errorf("PadRightANSI(%q, %d) visible width = %d, want %d (output %q)", tc.input, tc.width, gotWidth, tc.wantWidth, got)
			}
		})
	}
}

func TestPadRightANSIPreservesStyle(t *testing.T) {
	styled := ansiBlueFg + "hi" + ansiResetFg

	got := PadRightANSI(styled, 8)
	if !strings.Contains(got, ansiBlueFg) {
		t.Errorf("expected padded output to retain SGR sequence, got %q", got)
	}
	if TextWidth(got) != 8 {
		t.Errorf("padded width = %d, want 8: %q", TextWidth(got), got)
	}
}
