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

package lsp_domain

import (
	"testing"

	"go.lsp.dev/protocol"
)

func TestScanCSSClassSelectors(t *testing.T) {
	testCases := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "single class",
			content:  ".foo { color: red; }",
			expected: []string{"foo"},
		},
		{
			name:     "multiple classes",
			content:  ".foo { } .bar { } .baz { }",
			expected: []string{"foo", "bar", "baz"},
		},
		{
			name:     "hyphenated class",
			content:  ".card-img { }",
			expected: []string{"card-img"},
		},
		{
			name:     "underscored class",
			content:  ".my_class { }",
			expected: []string{"my_class"},
		},
		{
			name:     "class with digits",
			content:  ".col-12 { }",
			expected: []string{"col-12"},
		},
		{
			name:     "compound selector",
			content:  ".foo.bar { }",
			expected: []string{"foo", "bar"},
		},
		{
			name:     "element with class",
			content:  "div.active { }",
			expected: []string{"active"},
		},
		{
			name:     "empty content",
			content:  "",
			expected: nil,
		},
		{
			name:     "no classes",
			content:  "div { color: red; }",
			expected: nil,
		},
		{
			name:     "nested in media query",
			content:  "@media (max-width: 768px) { .mobile { display: block; } }",
			expected: []string{"mobile"},
		},
		{
			name:     "duplicate class returns both",
			content:  ".foo { } .bar { } .foo { }",
			expected: []string{"foo", "bar", "foo"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matches := scanCSSClassSelectors(tc.content)
			if len(matches) != len(tc.expected) {
				t.Fatalf("scanCSSClassSelectors(%q) returned %d matches, want %d",
					tc.content, len(matches), len(tc.expected))
			}
			for i, match := range matches {
				if match.Name != tc.expected[i] {
					t.Errorf("match[%d].Name = %q, want %q", i, match.Name, tc.expected[i])
				}
			}
		})
	}
}

func TestScanCSSClassSelectors_ignores_non_class_dots(t *testing.T) {
	testCases := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "numeric value",
			content:  "div { font-size: 0.5em; }",
			expected: nil,
		},
		{
			name:     "bare decimal",
			content:  "div { opacity: .5; }",
			expected: nil,
		},
		{
			name:     "inside block comment",
			content:  "/* .hidden { } */ .visible { }",
			expected: []string{"visible"},
		},
		{
			name:     "inside single-quoted string",
			content:  "div::after { content: '.foo'; } .bar { }",
			expected: []string{"bar"},
		},
		{
			name:     "inside double-quoted string",
			content:  `div::after { content: ".foo"; } .bar { }`,
			expected: []string{"bar"},
		},
		{
			name:     "escaped character in string",
			content:  `div::after { content: "test\".foo"; } .bar { }`,
			expected: []string{"bar"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matches := scanCSSClassSelectors(tc.content)
			if len(matches) != len(tc.expected) {
				t.Fatalf("scanCSSClassSelectors(%q) returned %d matches, want %d",
					tc.content, len(matches), len(tc.expected))
			}
			for i, match := range matches {
				if match.Name != tc.expected[i] {
					t.Errorf("match[%d].Name = %q, want %q", i, match.Name, tc.expected[i])
				}
			}
		})
	}
}

func TestScanCSSClassSelectors_offsets(t *testing.T) {
	content := ".foo { } .bar { }"
	matches := scanCSSClassSelectors(content)

	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}

	if matches[0].DotOffset != 0 {
		t.Errorf("first match DotOffset = %d, want 0", matches[0].DotOffset)
	}
	if matches[1].DotOffset != 9 {
		t.Errorf("second match DotOffset = %d, want 9", matches[1].DotOffset)
	}
}

func TestCheckCSSClassDefinitionContext_static_class(t *testing.T) {
	testCases := []struct {
		name      string
		line      string
		cursor    int
		wantName  string
		wantFound bool
	}{
		{
			name:      "cursor on first class",
			line:      `<div class="foo bar">`,
			cursor:    13,
			wantName:  "foo",
			wantFound: true,
		},
		{
			name:      "cursor on second class",
			line:      `<div class="foo bar">`,
			cursor:    17,
			wantName:  "bar",
			wantFound: true,
		},
		{
			name:      "cursor on single class",
			line:      `<div class="active">`,
			cursor:    14,
			wantName:  "active",
			wantFound: true,
		},
		{
			name:      "cursor on whitespace between classes",
			line:      `<div class="foo bar">`,
			cursor:    15,
			wantFound: false,
		},
		{
			name:      "not a class attribute",
			line:      `<div id="foo">`,
			cursor:    10,
			wantFound: false,
		},
	}

	d := &document{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := d.checkCSSClassDefinitionContext(tc.line, tc.cursor, protocol.Position{})
			if tc.wantFound {
				if ctx == nil {
					t.Fatalf("expected context, got nil")
				}
				if ctx.Kind != PKDefCSSClass {
					t.Errorf("Kind = %d, want PKDefCSSClass", ctx.Kind)
				}
				if ctx.Name != tc.wantName {
					t.Errorf("Name = %q, want %q", ctx.Name, tc.wantName)
				}
			} else {
				if ctx != nil {
					t.Errorf("expected nil context, got %+v", ctx)
				}
			}
		})
	}
}

func TestCheckCSSClassDefinitionContext_p_class_shorthand(t *testing.T) {
	testCases := []struct {
		name      string
		line      string
		cursor    int
		wantName  string
		wantFound bool
	}{
		{
			name:      "p-class:active",
			line:      `<div p-class:active="isActive">`,
			cursor:    15,
			wantName:  "active",
			wantFound: true,
		},
		{
			name:      "p-class:is-visible",
			line:      `<div p-class:is-visible="show">`,
			cursor:    17,
			wantName:  "is-visible",
			wantFound: true,
		},
		{
			name:      "cursor before p-class",
			line:      `<div p-class:active="isActive">`,
			cursor:    3,
			wantFound: false,
		},
	}

	d := &document{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := d.checkCSSClassDefinitionContext(tc.line, tc.cursor, protocol.Position{})
			if tc.wantFound {
				if ctx == nil {
					t.Fatalf("expected context, got nil")
				}
				if ctx.Kind != PKDefCSSClass {
					t.Errorf("Kind = %d, want PKDefCSSClass", ctx.Kind)
				}
				if ctx.Name != tc.wantName {
					t.Errorf("Name = %q, want %q", ctx.Name, tc.wantName)
				}
			} else {
				if ctx != nil {
					t.Errorf("expected nil context, got %+v", ctx)
				}
			}
		})
	}
}

func TestCheckCSSClassDefinitionContext_directive_string(t *testing.T) {
	testCases := []struct {
		name      string
		line      string
		cursor    int
		wantName  string
		wantFound bool
	}{
		{
			name:      "p-class object syntax",
			line:      `<div p-class="{ 'active': true }">`,
			cursor:    18,
			wantName:  "active",
			wantFound: true,
		},
		{
			name:      "bind class string literal",
			line:      `<div :class="'card-img'">`,
			cursor:    16,
			wantName:  "card-img",
			wantFound: true,
		},
		{
			name:      "cursor outside string literal",
			line:      `<div p-class="{ 'active': true }">`,
			cursor:    25,
			wantFound: false,
		},
	}

	d := &document{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := d.checkCSSClassDefinitionContext(tc.line, tc.cursor, protocol.Position{})
			if tc.wantFound {
				if ctx == nil {
					t.Fatalf("expected context, got nil")
				}
				if ctx.Kind != PKDefCSSClass {
					t.Errorf("Kind = %d, want PKDefCSSClass", ctx.Kind)
				}
				if ctx.Name != tc.wantName {
					t.Errorf("Name = %q, want %q", ctx.Name, tc.wantName)
				}
			} else {
				if ctx != nil {
					t.Errorf("expected nil context, got %+v", ctx)
				}
			}
		})
	}
}

func TestTryCSSClassValueContext(t *testing.T) {
	testCases := []struct {
		name           string
		line           string
		cursorPosition int
		wantMatch      bool
		wantPrefix     string
	}{
		{
			name:           "inside class attr value - empty",
			line:           `<div class="">`,
			cursorPosition: 12,
			wantMatch:      true,
			wantPrefix:     "",
		},
		{
			name:           "inside class attr value - partial word",
			line:           `<div class="fo">`,
			cursorPosition: 14,
			wantMatch:      true,
			wantPrefix:     "fo",
		},
		{
			name:           "inside class attr value - after space",
			line:           `<div class="foo ">`,
			cursorPosition: 16,
			wantMatch:      true,
			wantPrefix:     "",
		},
		{
			name:           "inside class attr value - second word",
			line:           `<div class="foo ba">`,
			cursorPosition: 18,
			wantMatch:      true,
			wantPrefix:     "ba",
		},
		{
			name:           "outside class attr - after closing quote",
			line:           `<div class="foo" id="">`,
			cursorPosition: 21,
			wantMatch:      false,
		},
		{
			name:           "not a class attribute",
			line:           `<div id="foo">`,
			cursorPosition: 12,
			wantMatch:      false,
		},
		{
			name:           "before class attr",
			line:           `<div class="foo">`,
			cursorPosition: 5,
			wantMatch:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &completionContext{}
			matched := tryCSSClassValueContext(ctx, tc.line, tc.cursorPosition)
			if matched != tc.wantMatch {
				t.Fatalf("tryCSSClassValueContext() = %v, want %v", matched, tc.wantMatch)
			}
			if matched {
				if ctx.TriggerKind != triggerCSSClassValue {
					t.Errorf("TriggerKind = %d, want triggerCSSClassValue", ctx.TriggerKind)
				}
				if ctx.Prefix != tc.wantPrefix {
					t.Errorf("Prefix = %q, want %q", ctx.Prefix, tc.wantPrefix)
				}
			}
		})
	}
}

func TestFindWordAtOffset(t *testing.T) {
	testCases := []struct {
		name     string
		text     string
		offset   int
		expected string
	}{
		{
			name:     "single word",
			text:     "hello",
			offset:   2,
			expected: "hello",
		},
		{
			name:     "first of two words",
			text:     "foo bar",
			offset:   1,
			expected: "foo",
		},
		{
			name:     "second of two words",
			text:     "foo bar",
			offset:   5,
			expected: "bar",
		},
		{
			name:     "on space",
			text:     "foo bar",
			offset:   3,
			expected: "",
		},
		{
			name:     "at end of text",
			text:     "foo",
			offset:   3,
			expected: "foo",
		},
		{
			name:     "empty text",
			text:     "",
			offset:   0,
			expected: "",
		},
		{
			name:     "negative offset",
			text:     "foo",
			offset:   -1,
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := findWordAtOffset(tc.text, tc.offset)
			if result != tc.expected {
				t.Errorf("findWordAtOffset(%q, %d) = %q, want %q",
					tc.text, tc.offset, result, tc.expected)
			}
		})
	}
}

func TestExtractStringLiteralAtCursor(t *testing.T) {
	testCases := []struct {
		name     string
		text     string
		cursor   int
		expected string
	}{
		{
			name:     "cursor inside string",
			text:     "{ 'active': true }",
			cursor:   4,
			expected: "active",
		},
		{
			name:     "cursor on first char",
			text:     "'foo'",
			cursor:   1,
			expected: "foo",
		},
		{
			name:     "cursor outside string",
			text:     "{ 'active': true }",
			cursor:   12,
			expected: "",
		},
		{
			name:     "empty string",
			text:     "''",
			cursor:   1,
			expected: "",
		},
		{
			name:     "second string",
			text:     "'foo', 'bar'",
			cursor:   9,
			expected: "bar",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractStringLiteralAtCursor(tc.text, tc.cursor)
			if result != tc.expected {
				t.Errorf("extractStringLiteralAtCursor(%q, %d) = %q, want %q",
					tc.text, tc.cursor, result, tc.expected)
			}
		})
	}
}

func TestExtractLastWord(t *testing.T) {
	testCases := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "empty",
			text:     "",
			expected: "",
		},
		{
			name:     "single word",
			text:     "foo",
			expected: "foo",
		},
		{
			name:     "ends with space",
			text:     "foo ",
			expected: "",
		},
		{
			name:     "two words",
			text:     "foo ba",
			expected: "ba",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractLastWord(tc.text)
			if result != tc.expected {
				t.Errorf("extractLastWord(%q) = %q, want %q", tc.text, result, tc.expected)
			}
		})
	}
}
