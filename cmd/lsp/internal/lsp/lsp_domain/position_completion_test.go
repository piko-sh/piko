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
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestSplitLines(t *testing.T) {
	testCases := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "empty content",
			content:  "",
			expected: []string{},
		},
		{
			name:     "single line no newline",
			content:  "hello",
			expected: []string{"hello"},
		},
		{
			name:     "single line with newline",
			content:  "hello\n",
			expected: []string{"hello"},
		},
		{
			name:     "multiple lines",
			content:  "line1\nline2\nline3",
			expected: []string{"line1", "line2", "line3"},
		},
		{
			name:     "multiple lines with trailing newline",
			content:  "line1\nline2\n",
			expected: []string{"line1", "line2"},
		},
		{
			name:     "empty lines",
			content:  "line1\n\nline3",
			expected: []string{"line1", "", "line3"},
		},
		{
			name:     "carriage returns stripped",
			content:  "line1\r\nline2\r\n",
			expected: []string{"line1", "line2"},
		},
		{
			name:     "only newlines",
			content:  "\n\n\n",
			expected: []string{"", "", ""},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := splitLines([]byte(tc.content))
			if len(result) != len(tc.expected) {
				t.Errorf("splitLines(%q) returned %d lines, want %d", tc.content, len(result), len(tc.expected))
				return
			}
			for i := range tc.expected {
				if string(result[i]) != tc.expected[i] {
					t.Errorf("splitLines(%q)[%d] = %q, want %q", tc.content, i, string(result[i]), tc.expected[i])
				}
			}
		})
	}
}

func TestExtractBaseExpression(t *testing.T) {
	testCases := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "empty text",
			text:     "",
			expected: "",
		},
		{
			name:     "simple identifier",
			text:     "state",
			expected: "state",
		},
		{
			name:     "member access",
			text:     "state.name",
			expected: "state.name",
		},
		{
			name:     "chained member access",
			text:     "state.user.name",
			expected: "state.user.name",
		},
		{
			name:     "with function call",
			text:     "getUser()",
			expected: "getUser()",
		},
		{
			name:     "function call with member access",
			text:     "getUser().name",
			expected: "getUser().name",
		},
		{
			name:     "array access",
			text:     "users[0]",
			expected: "users[0]",
		},
		{
			name:     "complex expression",
			text:     "state.users[0].name",
			expected: "state.users[0].name",
		},
		{
			name:     "nested function calls",
			text:     "outer(inner())",
			expected: "outer(inner())",
		},
		{
			name:     "expression after space",
			text:     "x = state",
			expected: "state",
		},
		{
			name:     "expression after operator",
			text:     "x + state",
			expected: "state",
		},
		{
			name:     "expression in template",
			text:     "{{ state",
			expected: "state",
		},
		{
			name:     "identifier with underscore",
			text:     "user_name",
			expected: "user_name",
		},
		{
			name:     "identifier with numbers",
			text:     "user123",
			expected: "user123",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractBaseExpression([]byte(tc.text))
			if result != tc.expected {
				t.Errorf("extractBaseExpression(%q) = %q, want %q", tc.text, result, tc.expected)
			}
		})
	}
}

func TestFindExpressionStart(t *testing.T) {
	testCases := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "simple identifier",
			text:     "state",
			expected: 0,
		},
		{
			name:     "identifier after space",
			text:     "x state",
			expected: 2,
		},
		{
			name:     "after equals",
			text:     "x=state",
			expected: 2,
		},
		{
			name:     "with parens",
			text:     "func()",
			expected: 0,
		},
		{
			name:     "with brackets",
			text:     "arr[0]",
			expected: 0,
		},
		{
			name:     "nested parens",
			text:     "outer(inner())",
			expected: 0,
		},
		{
			name:     "after open paren at depth zero",
			text:     "(state",
			expected: 1,
		},
		{
			name:     "with dots",
			text:     "a.b.c",
			expected: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := findExpressionStart([]byte(tc.text))
			if result != tc.expected {
				t.Errorf("findExpressionStart(%q) = %d, want %d", tc.text, result, tc.expected)
			}
		})
	}
}

func TestIsIdentChar(t *testing.T) {
	testCases := []struct {
		name     string
		char     byte
		expected bool
	}{
		{name: "lowercase a", char: 'a', expected: true},
		{name: "lowercase z", char: 'z', expected: true},
		{name: "uppercase A", char: 'A', expected: true},
		{name: "uppercase Z", char: 'Z', expected: true},
		{name: "digit 0", char: '0', expected: true},
		{name: "digit 9", char: '9', expected: true},
		{name: "underscore", char: '_', expected: true},
		{name: "space", char: ' ', expected: false},
		{name: "dot", char: '.', expected: false},
		{name: "open paren", char: '(', expected: false},
		{name: "close paren", char: ')', expected: false},
		{name: "open bracket", char: '[', expected: false},
		{name: "close bracket", char: ']', expected: false},
		{name: "equals", char: '=', expected: false},
		{name: "plus", char: '+', expected: false},
		{name: "hyphen", char: '-', expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isIdentChar(tc.char)
			if result != tc.expected {
				t.Errorf("isIdentChar(%c) = %v, want %v", tc.char, result, tc.expected)
			}
		})
	}
}

func TestFindLastOccurrence(t *testing.T) {
	testCases := []struct {
		name     string
		s        string
		substr   string
		expected int
	}{
		{
			name:     "not found",
			s:        "hello world",
			substr:   "foo",
			expected: -1,
		},
		{
			name:     "found at start",
			s:        "hello world",
			substr:   "hello",
			expected: 0,
		},
		{
			name:     "found at end",
			s:        "hello world",
			substr:   "world",
			expected: 6,
		},
		{
			name:     "multiple occurrences returns last",
			s:        "hello hello hello",
			substr:   "hello",
			expected: 12,
		},
		{
			name:     "single char",
			s:        "abcabc",
			substr:   "c",
			expected: 5,
		},
		{
			name:     "empty string",
			s:        "",
			substr:   "x",
			expected: -1,
		},
		{
			name:     "empty substr",
			s:        "hello",
			substr:   "",
			expected: 5,
		},
		{
			name:     "substr equals string",
			s:        "hello",
			substr:   "hello",
			expected: 0,
		},
		{
			name:     "substr longer than string",
			s:        "hi",
			substr:   "hello",
			expected: -1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := findLastOccurrence(tc.s, tc.substr)
			if result != tc.expected {
				t.Errorf("findLastOccurrence(%q, %q) = %d, want %d", tc.s, tc.substr, result, tc.expected)
			}
		})
	}
}

func TestHasClosingQuote(t *testing.T) {
	testCases := []struct {
		name     string
		text     string
		expected bool
	}{
		{
			name:     "empty text",
			text:     "",
			expected: false,
		},
		{
			name:     "no quotes",
			text:     "hello world",
			expected: false,
		},
		{
			name:     "has double quote",
			text:     `hello "world`,
			expected: true,
		},
		{
			name:     "quote at start",
			text:     `"hello`,
			expected: true,
		},
		{
			name:     "quote at end",
			text:     `hello"`,
			expected: true,
		},
		{
			name:     "only quote",
			text:     `"`,
			expected: true,
		},
		{
			name:     "single quote not detected",
			text:     `hello'world`,
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := hasClosingQuote(tc.text)
			if result != tc.expected {
				t.Errorf("hasClosingQuote(%q) = %v, want %v", tc.text, result, tc.expected)
			}
		})
	}
}

func TestFindEventHandlerContext(t *testing.T) {
	testCases := []struct {
		name           string
		line           string
		expectedPrefix string
		cursorPosition int
		expectedIndex  int
	}{
		{
			name:           "not in event handler",
			line:           `<div class="test">`,
			cursorPosition: 10,
			expectedIndex:  -1,
			expectedPrefix: "",
		},
		{
			name:           "in click handler",
			line:           `<button p-on:click="handleClick`,
			cursorPosition: 31,
			expectedIndex:  20,
			expectedPrefix: "handleClick",
		},
		{
			name:           "in submit handler",
			line:           `<form p-on:submit="onSub`,
			cursorPosition: 24,
			expectedIndex:  19,
			expectedPrefix: "onSub",
		},
		{
			name:           "closed handler",
			line:           `<button p-on:click="handleClick" class`,
			cursorPosition: 38,
			expectedIndex:  -1,
			expectedPrefix: "",
		},
		{
			name:           "empty handler value",
			line:           `<button p-on:click="`,
			cursorPosition: 20,
			expectedIndex:  20,
			expectedPrefix: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			index, prefix := findEventHandlerContext(tc.line, tc.cursorPosition)
			if index != tc.expectedIndex {
				t.Errorf("findEventHandlerContext() index = %d, want %d", index, tc.expectedIndex)
			}
			if prefix != tc.expectedPrefix {
				t.Errorf("findEventHandlerContext() prefix = %q, want %q", prefix, tc.expectedPrefix)
			}
		})
	}
}

func TestFindPartialNameContext(t *testing.T) {
	testCases := []struct {
		name           string
		line           string
		expectedPrefix string
		cursorPosition int
		expectedIndex  int
	}{
		{
			name:           "not in partial context",
			line:           `someFunction("test")`,
			cursorPosition: 15,
			expectedIndex:  -1,
			expectedPrefix: "",
		},
		{
			name:           "in reloadPartial single quote",
			line:           `reloadPartial('status_`,
			cursorPosition: 22,
			expectedIndex:  15,
			expectedPrefix: "status_",
		},
		{
			name:           "in reloadPartial double quote",
			line:           `reloadPartial("status_`,
			cursorPosition: 22,
			expectedIndex:  15,
			expectedPrefix: "status_",
		},
		{
			name:           "in reloadGroup single quote",
			line:           `reloadGroup('badge_`,
			cursorPosition: 19,
			expectedIndex:  13,
			expectedPrefix: "badge_",
		},
		{
			name:           "in reloadGroup double quote",
			line:           `reloadGroup("badge_`,
			cursorPosition: 19,
			expectedIndex:  13,
			expectedPrefix: "badge_",
		},
		{
			name:           "closed partial name",
			line:           `reloadPartial('status')`,
			cursorPosition: 23,
			expectedIndex:  -1,
			expectedPrefix: "",
		},
		{
			name:           "empty partial name",
			line:           `reloadPartial('`,
			cursorPosition: 15,
			expectedIndex:  15,
			expectedPrefix: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			index, prefix := findPartialNameContext(tc.line, tc.cursorPosition)
			if index != tc.expectedIndex {
				t.Errorf("findPartialNameContext() index = %d, want %d", index, tc.expectedIndex)
			}
			if prefix != tc.expectedPrefix {
				t.Errorf("findPartialNameContext() prefix = %q, want %q", prefix, tc.expectedPrefix)
			}
		})
	}
}

func TestFindRefsAccessContext(t *testing.T) {
	testCases := []struct {
		name           string
		line           string
		expectedPrefix string
		cursorPosition int
		expectedIndex  int
	}{
		{
			name:           "not in refs context",
			line:           `state.value`,
			cursorPosition: 11,
			expectedIndex:  -1,
			expectedPrefix: "",
		},
		{
			name:           "in refs context",
			line:           `refs.myInput`,
			cursorPosition: 12,
			expectedIndex:  5,
			expectedPrefix: "myInput",
		},
		{
			name:           "empty refs prefix",
			line:           `refs.`,
			cursorPosition: 5,
			expectedIndex:  5,
			expectedPrefix: "",
		},
		{
			name:           "refs with partial prefix",
			line:           `refs.inp`,
			cursorPosition: 8,
			expectedIndex:  5,
			expectedPrefix: "inp",
		},
		{
			name:           "refs followed by non-ident char",
			line:           `refs.myInput.`,
			cursorPosition: 13,
			expectedIndex:  -1,
			expectedPrefix: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			index, prefix := findRefsAccessContext(tc.line, tc.cursorPosition)
			if index != tc.expectedIndex {
				t.Errorf("findRefsAccessContext() index = %d, want %d", index, tc.expectedIndex)
			}
			if prefix != tc.expectedPrefix {
				t.Errorf("findRefsAccessContext() prefix = %q, want %q", prefix, tc.expectedPrefix)
			}
		})
	}
}

func TestFindStateAccessContext(t *testing.T) {
	testCases := []struct {
		name           string
		line           string
		expectedPrefix string
		cursorPosition int
		expectedIndex  int
	}{
		{
			name:           "not in state context",
			line:           `props.value`,
			cursorPosition: 11,
			expectedIndex:  -1,
			expectedPrefix: "",
		},
		{
			name:           "in state context",
			line:           `state.myValue`,
			cursorPosition: 13,
			expectedIndex:  6,
			expectedPrefix: "myValue",
		},
		{
			name:           "empty state prefix",
			line:           `state.`,
			cursorPosition: 6,
			expectedIndex:  6,
			expectedPrefix: "",
		},
		{
			name:           "state with partial prefix",
			line:           `state.coun`,
			cursorPosition: 10,
			expectedIndex:  6,
			expectedPrefix: "coun",
		},
		{
			name:           "state followed by non-ident char",
			line:           `state.count + 1`,
			cursorPosition: 12,
			expectedIndex:  -1,
			expectedPrefix: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			index, prefix := findStateAccessContext(tc.line, tc.cursorPosition)
			if index != tc.expectedIndex {
				t.Errorf("findStateAccessContext() index = %d, want %d", index, tc.expectedIndex)
			}
			if prefix != tc.expectedPrefix {
				t.Errorf("findStateAccessContext() prefix = %q, want %q", prefix, tc.expectedPrefix)
			}
		})
	}
}

func TestFindPropsAccessContext(t *testing.T) {
	testCases := []struct {
		name           string
		line           string
		expectedPrefix string
		cursorPosition int
		expectedIndex  int
	}{
		{
			name:           "not in props context",
			line:           `state.value`,
			cursorPosition: 11,
			expectedIndex:  -1,
			expectedPrefix: "",
		},
		{
			name:           "in props context",
			line:           `props.myProp`,
			cursorPosition: 12,
			expectedIndex:  6,
			expectedPrefix: "myProp",
		},
		{
			name:           "empty props prefix",
			line:           `props.`,
			cursorPosition: 6,
			expectedIndex:  6,
			expectedPrefix: "",
		},
		{
			name:           "props with partial prefix",
			line:           `props.tit`,
			cursorPosition: 9,
			expectedIndex:  6,
			expectedPrefix: "tit",
		},
		{
			name:           "props followed by non-ident char",
			line:           `props.title + " suffix"`,
			cursorPosition: 12,
			expectedIndex:  -1,
			expectedPrefix: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			index, prefix := findPropsAccessContext(tc.line, tc.cursorPosition)
			if index != tc.expectedIndex {
				t.Errorf("findPropsAccessContext() index = %d, want %d", index, tc.expectedIndex)
			}
			if prefix != tc.expectedPrefix {
				t.Errorf("findPropsAccessContext() prefix = %q, want %q", prefix, tc.expectedPrefix)
			}
		})
	}
}

func TestTryMemberAccessContext(t *testing.T) {
	testCases := []struct {
		name             string
		text             string
		expectedBaseExpr string
		expectedTrigger  completionTriggerKind
		expectedResult   bool
	}{
		{
			name:           "empty text",
			text:           "",
			expectedResult: false,
		},
		{
			name:           "no trailing dot",
			text:           "state",
			expectedResult: false,
		},
		{
			name:             "trailing dot",
			text:             "state.",
			expectedResult:   true,
			expectedTrigger:  triggerMemberAccess,
			expectedBaseExpr: "state",
		},
		{
			name:             "chained access",
			text:             "state.user.",
			expectedResult:   true,
			expectedTrigger:  triggerMemberAccess,
			expectedBaseExpr: "state.user",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &completionContext{}
			result := tryMemberAccessContext(ctx, []byte(tc.text))
			if result != tc.expectedResult {
				t.Errorf("tryMemberAccessContext() = %v, want %v", result, tc.expectedResult)
			}
			if result {
				if ctx.TriggerKind != tc.expectedTrigger {
					t.Errorf("TriggerKind = %v, want %v", ctx.TriggerKind, tc.expectedTrigger)
				}
				if ctx.BaseExpression != tc.expectedBaseExpr {
					t.Errorf("BaseExpression = %q, want %q", ctx.BaseExpression, tc.expectedBaseExpr)
				}
			}
		})
	}
}

func TestTryDirectiveContext(t *testing.T) {
	testCases := []struct {
		name            string
		text            string
		expectedPrefix  string
		expectedTrigger completionTriggerKind
		expectedResult  bool
	}{
		{
			name:           "empty text",
			text:           "",
			expectedResult: false,
		},
		{
			name:           "single char",
			text:           "p",
			expectedResult: false,
		},
		{
			name:           "no directive prefix",
			text:           "class=",
			expectedResult: false,
		},
		{
			name:            "directive prefix only",
			text:            "p-",
			expectedResult:  true,
			expectedTrigger: triggerDirective,
			expectedPrefix:  "",
		},
		{
			name:            "directive in HTML context",
			text:            "<div p-",
			expectedResult:  true,
			expectedTrigger: triggerDirective,
			expectedPrefix:  "",
		},
		{
			name:            "partial directive name - sh",
			text:            "<div p-sh",
			expectedResult:  true,
			expectedTrigger: triggerDirective,
			expectedPrefix:  "sh",
		},
		{
			name:            "partial directive name - show",
			text:            "<div p-show",
			expectedResult:  true,
			expectedTrigger: triggerDirective,
			expectedPrefix:  "show",
		},
		{
			name:            "partial directive name - for",
			text:            "  p-for",
			expectedResult:  true,
			expectedTrigger: triggerDirective,
			expectedPrefix:  "for",
		},
		{
			name:            "partial directive name with hyphen",
			text:            "<span p-else-",
			expectedResult:  true,
			expectedTrigger: triggerDirective,
			expectedPrefix:  "else-",
		},
		{
			name:            "partial directive name - else-if",
			text:            "p-else-if",
			expectedResult:  true,
			expectedTrigger: triggerDirective,
			expectedPrefix:  "else-if",
		},
		{
			name:           "p- not at word boundary",
			text:           "xp-if",
			expectedResult: false,
		},
		{
			name:            "p- after newline",
			text:            "\n  p-",
			expectedResult:  true,
			expectedTrigger: triggerDirective,
			expectedPrefix:  "",
		},
		{
			name:           "invalid chars after p-",
			text:           "p-=",
			expectedResult: false,
		},
		{
			name:           "p- in middle of word",
			text:           "abcp-if",
			expectedResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &completionContext{}
			result := tryDirectiveContext(ctx, []byte(tc.text))
			if result != tc.expectedResult {
				t.Errorf("tryDirectiveContext() = %v, want %v", result, tc.expectedResult)
			}
			if result {
				if ctx.TriggerKind != tc.expectedTrigger {
					t.Errorf("TriggerKind = %v, want %v", ctx.TriggerKind, tc.expectedTrigger)
				}
				if ctx.Prefix != tc.expectedPrefix {
					t.Errorf("Prefix = %q, want %q", ctx.Prefix, tc.expectedPrefix)
				}
			}
		})
	}
}

func TestTryPartialAliasContext(t *testing.T) {
	testCases := []struct {
		name            string
		line            string
		expectedPrefix  string
		cursorPosition  int
		expectedTrigger completionTriggerKind
		expectedResult  bool
	}{
		{
			name:           "not in partial alias",
			line:           `class="test"`,
			cursorPosition: 8,
			expectedResult: false,
		},
		{
			name:            "in partial alias",
			line:            `is="status_`,
			cursorPosition:  11,
			expectedResult:  true,
			expectedTrigger: triggerPartialAlias,
			expectedPrefix:  "status_",
		},
		{
			name:           "closed partial alias",
			line:           `is="status"`,
			cursorPosition: 11,
			expectedResult: false,
		},
		{
			name:            "empty partial alias",
			line:            `is="`,
			cursorPosition:  4,
			expectedResult:  true,
			expectedTrigger: triggerPartialAlias,
			expectedPrefix:  "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &completionContext{}
			result := tryPartialAliasContext(ctx, tc.line, tc.cursorPosition)
			if result != tc.expectedResult {
				t.Errorf("tryPartialAliasContext() = %v, want %v", result, tc.expectedResult)
			}
			if result {
				if ctx.TriggerKind != tc.expectedTrigger {
					t.Errorf("TriggerKind = %v, want %v", ctx.TriggerKind, tc.expectedTrigger)
				}
				if ctx.Prefix != tc.expectedPrefix {
					t.Errorf("Prefix = %q, want %q", ctx.Prefix, tc.expectedPrefix)
				}
			}
		})
	}
}

func TestCompletionTriggerKind_Values(t *testing.T) {

	testCases := []struct {
		name  string
		kind  completionTriggerKind
		value int
	}{
		{name: "triggerScope", kind: triggerScope, value: 0},
		{name: "triggerMemberAccess", kind: triggerMemberAccess, value: 1},
		{name: "triggerPartialAlias", kind: triggerPartialAlias, value: 2},
		{name: "triggerDirective", kind: triggerDirective, value: 3},
		{name: "triggerDirectiveValue", kind: triggerDirectiveValue, value: 4},
		{name: "triggerEventHandler", kind: triggerEventHandler, value: 5},
		{name: "triggerPartialName", kind: triggerPartialName, value: 6},
		{name: "triggerRefAccess", kind: triggerRefAccess, value: 7},
		{name: "triggerStateAccessJS", kind: triggerStateAccessJS, value: 8},
		{name: "triggerPropsAccessJS", kind: triggerPropsAccessJS, value: 9},
		{name: "triggerPikoNamespace", kind: triggerPikoNamespace, value: 10},
		{name: "triggerPikoSubNamespace", kind: triggerPikoSubNamespace, value: 11},
		{name: "triggerActionNamespace", kind: triggerActionNamespace, value: 12},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if int(tc.kind) != tc.value {
				t.Errorf("%s = %d, want %d", tc.name, tc.kind, tc.value)
			}
		})
	}
}

func TestTryPikoNamespaceContext(t *testing.T) {
	testCases := []struct {
		name              string
		textBeforeCursor  string
		expectedNamespace string
		expectedPrefix    string
		expectedTrigger   completionTriggerKind
		expectMatch       bool
	}{
		{
			name:             "piko. triggers namespace completion",
			textBeforeCursor: "piko.",
			expectMatch:      true,
			expectedTrigger:  triggerPikoNamespace,
			expectedPrefix:   "",
		},
		{
			name:             "piko.na triggers namespace completion with prefix",
			textBeforeCursor: "piko.na",
			expectMatch:      true,
			expectedTrigger:  triggerPikoNamespace,
			expectedPrefix:   "na",
		},
		{
			name:              "piko.nav. triggers sub-namespace completion",
			textBeforeCursor:  "piko.nav.",
			expectMatch:       true,
			expectedTrigger:   triggerPikoSubNamespace,
			expectedNamespace: "nav",
			expectedPrefix:    "",
		},
		{
			name:              "piko.nav.na triggers sub-namespace completion with prefix",
			textBeforeCursor:  "piko.nav.na",
			expectMatch:       true,
			expectedTrigger:   triggerPikoSubNamespace,
			expectedNamespace: "nav",
			expectedPrefix:    "na",
		},
		{
			name:              "piko.form. triggers sub-namespace completion",
			textBeforeCursor:  "piko.form.",
			expectMatch:       true,
			expectedTrigger:   triggerPikoSubNamespace,
			expectedNamespace: "form",
			expectedPrefix:    "",
		},
		{
			name:             "piko without dot doesn't trigger",
			textBeforeCursor: "piko",
			expectMatch:      false,
		},
		{
			name:             "xpiko. doesn't trigger (word boundary)",
			textBeforeCursor: "xpiko.",
			expectMatch:      false,
		},
		{
			name:             "piko. in expression context",
			textBeforeCursor: `p-on:click="piko.`,
			expectMatch:      true,
			expectedTrigger:  triggerPikoNamespace,
			expectedPrefix:   "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var ctx completionContext
			matched := tryPikoNamespaceContext(&ctx, []byte(tc.textBeforeCursor))

			if matched != tc.expectMatch {
				t.Errorf("tryPikoNamespaceContext() matched = %v, want %v", matched, tc.expectMatch)
				return
			}

			if !tc.expectMatch {
				return
			}

			if ctx.TriggerKind != tc.expectedTrigger {
				t.Errorf("TriggerKind = %v, want %v", ctx.TriggerKind, tc.expectedTrigger)
			}
			if ctx.Namespace != tc.expectedNamespace {
				t.Errorf("Namespace = %q, want %q", ctx.Namespace, tc.expectedNamespace)
			}
			if ctx.Prefix != tc.expectedPrefix {
				t.Errorf("Prefix = %q, want %q", ctx.Prefix, tc.expectedPrefix)
			}
		})
	}
}

func TestFindLastDotIndex(t *testing.T) {
	testCases := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "empty text",
			text:     "",
			expected: -1,
		},
		{
			name:     "no dot",
			text:     "state",
			expected: -1,
		},
		{
			name:     "single dot",
			text:     "state.name",
			expected: 5,
		},
		{
			name:     "multiple dots returns last",
			text:     "a.b.c",
			expected: 3,
		},
		{
			name:     "dot at start",
			text:     ".abc",
			expected: 0,
		},
		{
			name:     "dot at end",
			text:     "abc.",
			expected: 3,
		},
		{
			name:     "only dot",
			text:     ".",
			expected: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := findLastDotIndex([]byte(tc.text))
			if result != tc.expected {
				t.Errorf("findLastDotIndex(%q) = %d, want %d", tc.text, result, tc.expected)
			}
		})
	}
}

func TestIsValidIdentifierPrefix(t *testing.T) {
	testCases := []struct {
		name     string
		text     string
		expected bool
	}{
		{
			name:     "empty text",
			text:     "",
			expected: true,
		},
		{
			name:     "lowercase letters",
			text:     "abc",
			expected: true,
		},
		{
			name:     "uppercase letters",
			text:     "ABC",
			expected: true,
		},
		{
			name:     "digits",
			text:     "123",
			expected: true,
		},
		{
			name:     "underscores",
			text:     "a_b_c",
			expected: true,
		},
		{
			name:     "mixed valid chars",
			text:     "myVar_123",
			expected: true,
		},
		{
			name:     "contains dot",
			text:     "a.b",
			expected: false,
		},
		{
			name:     "contains space",
			text:     "a b",
			expected: false,
		},
		{
			name:     "contains hyphen",
			text:     "a-b",
			expected: false,
		},
		{
			name:     "contains equals",
			text:     "a=b",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isValidIdentifierPrefix([]byte(tc.text))
			if result != tc.expected {
				t.Errorf("isValidIdentifierPrefix(%q) = %v, want %v", tc.text, result, tc.expected)
			}
		})
	}
}

func TestFindLastPDash(t *testing.T) {
	testCases := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "empty text",
			text:     "",
			expected: -1,
		},
		{
			name:     "single char",
			text:     "p",
			expected: -1,
		},
		{
			name:     "p- at start",
			text:     "p-if",
			expected: 0,
		},
		{
			name:     "p- after space",
			text:     " p-show",
			expected: 1,
		},
		{
			name:     "multiple p- returns last",
			text:     "p-if p-show",
			expected: 5,
		},
		{
			name:     "no p-",
			text:     "class",
			expected: -1,
		},
		{
			name:     "just p-",
			text:     "p-",
			expected: 0,
		},
		{
			name:     "p without dash",
			text:     "prefix",
			expected: -1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := findLastPDash([]byte(tc.text))
			if result != tc.expected {
				t.Errorf("findLastPDash(%q) = %d, want %d", tc.text, result, tc.expected)
			}
		})
	}
}

func TestIsAttrBoundary(t *testing.T) {
	testCases := []struct {
		name     string
		char     byte
		expected bool
	}{
		{name: "space", char: ' ', expected: true},
		{name: "tab", char: '\t', expected: true},
		{name: "newline", char: '\n', expected: true},
		{name: "carriage return", char: '\r', expected: true},
		{name: "less than", char: '<', expected: true},
		{name: "letter a", char: 'a', expected: false},
		{name: "digit 0", char: '0', expected: false},
		{name: "equals", char: '=', expected: false},
		{name: "double quote", char: '"', expected: false},
		{name: "hyphen", char: '-', expected: false},
		{name: "greater than", char: '>', expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isAttrBoundary(tc.char)
			if result != tc.expected {
				t.Errorf("isAttrBoundary(%q) = %v, want %v", tc.char, result, tc.expected)
			}
		})
	}
}

func TestIsValidDirectivePrefix(t *testing.T) {
	testCases := []struct {
		name     string
		text     string
		expected bool
	}{
		{
			name:     "empty text",
			text:     "",
			expected: true,
		},
		{
			name:     "lowercase letters",
			text:     "show",
			expected: true,
		},
		{
			name:     "with hyphen",
			text:     "else-if",
			expected: true,
		},
		{
			name:     "with digits",
			text:     "abc123",
			expected: true,
		},
		{
			name:     "uppercase letters",
			text:     "ABC",
			expected: true,
		},
		{
			name:     "contains underscore",
			text:     "a_b",
			expected: false,
		},
		{
			name:     "contains dot",
			text:     "a.b",
			expected: false,
		},
		{
			name:     "contains space",
			text:     "a b",
			expected: false,
		},
		{
			name:     "contains equals",
			text:     "show=",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isValidDirectivePrefix([]byte(tc.text))
			if result != tc.expected {
				t.Errorf("isValidDirectivePrefix(%q) = %v, want %v", tc.text, result, tc.expected)
			}
		})
	}
}

func TestIsDirectiveNameChar(t *testing.T) {
	testCases := []struct {
		name     string
		char     byte
		expected bool
	}{
		{name: "lowercase a", char: 'a', expected: true},
		{name: "lowercase z", char: 'z', expected: true},
		{name: "uppercase A", char: 'A', expected: true},
		{name: "uppercase Z", char: 'Z', expected: true},
		{name: "digit 0", char: '0', expected: true},
		{name: "digit 9", char: '9', expected: true},
		{name: "hyphen", char: '-', expected: true},
		{name: "underscore", char: '_', expected: false},
		{name: "dot", char: '.', expected: false},
		{name: "space", char: ' ', expected: false},
		{name: "equals", char: '=', expected: false},
		{name: "colon", char: ':', expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isDirectiveNameChar(tc.char)
			if result != tc.expected {
				t.Errorf("isDirectiveNameChar(%q) = %v, want %v", tc.char, result, tc.expected)
			}
		})
	}
}

func TestExtractDirectiveValuePrefix(t *testing.T) {
	testCases := []struct {
		name           string
		text           string
		expectedPrefix string
		expectedInside bool
	}{
		{
			name:           "empty text (inside quotes)",
			text:           "",
			expectedPrefix: "",
			expectedInside: true,
		},
		{
			name:           "text without closing quote",
			text:           "state.value",
			expectedPrefix: "state.value",
			expectedInside: true,
		},
		{
			name:           "text with closing quote",
			text:           `state.value" class`,
			expectedPrefix: "",
			expectedInside: false,
		},
		{
			name:           "partial expression",
			text:           "sta",
			expectedPrefix: "sta",
			expectedInside: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prefix, inside := extractDirectiveValuePrefix([]byte(tc.text))
			if prefix != tc.expectedPrefix {
				t.Errorf("prefix = %q, want %q", prefix, tc.expectedPrefix)
			}
			if inside != tc.expectedInside {
				t.Errorf("inside = %v, want %v", inside, tc.expectedInside)
			}
		})
	}
}

func TestIsValidActionPrefix(t *testing.T) {
	testCases := []struct {
		name     string
		text     string
		expected bool
	}{
		{
			name:     "empty text",
			text:     "",
			expected: true,
		},
		{
			name:     "simple identifier",
			text:     "customer",
			expected: true,
		},
		{
			name:     "dotted path",
			text:     "customer.create",
			expected: true,
		},
		{
			name:     "with underscore",
			text:     "my_action",
			expected: true,
		},
		{
			name:     "with digits",
			text:     "action123",
			expected: true,
		},
		{
			name:     "contains space",
			text:     "a b",
			expected: false,
		},
		{
			name:     "contains hyphen",
			text:     "a-b",
			expected: false,
		},
		{
			name:     "contains equals",
			text:     "a=b",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isValidActionPrefix([]byte(tc.text))
			if result != tc.expected {
				t.Errorf("isValidActionPrefix(%q) = %v, want %v", tc.text, result, tc.expected)
			}
		})
	}
}

func TestFindPatternEnd(t *testing.T) {
	testCases := []struct {
		name     string
		text     string
		pattern  string
		expected int
	}{
		{
			name:     "pattern found at start",
			text:     "piko.nav",
			pattern:  "piko.",
			expected: 5,
		},
		{
			name:     "pattern not found",
			text:     "state.value",
			pattern:  "piko.",
			expected: -1,
		},
		{
			name:     "pattern at word boundary after space",
			text:     "x = piko.nav",
			pattern:  "piko.",
			expected: 9,
		},
		{
			name:     "pattern not at word boundary",
			text:     "xpiko.nav",
			pattern:  "piko.",
			expected: -1,
		},
		{
			name:     "pattern at start of text (word boundary)",
			text:     "action.create",
			pattern:  "action.",
			expected: 7,
		},
		{
			name:     "multiple occurrences returns last",
			text:     "piko.a piko.b",
			pattern:  "piko.",
			expected: 12,
		},
		{
			name:     "pattern equals text",
			text:     "piko.",
			pattern:  "piko.",
			expected: 5,
		},
		{
			name:     "text shorter than pattern",
			text:     "pik",
			pattern:  "piko.",
			expected: -1,
		},
		{
			name:     "pattern after equals sign (word boundary)",
			text:     `p-on:click="piko.`,
			pattern:  "piko.",
			expected: 17,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := findPatternEnd([]byte(tc.text), tc.pattern)
			if result != tc.expected {
				t.Errorf("findPatternEnd(%q, %q) = %d, want %d", tc.text, tc.pattern, result, tc.expected)
			}
		})
	}
}

func TestIsDirectiveAttribute(t *testing.T) {
	testCases := []struct {
		name     string
		text     string
		expected bool
	}{
		{
			name:     "p-if",
			text:     "p-if",
			expected: true,
		},
		{
			name:     "p-show",
			text:     "p-show",
			expected: true,
		},
		{
			name:     "p-bind:class",
			text:     "p-bind:class",
			expected: true,
		},
		{
			name:     "p-on:click",
			text:     "p-on:click",
			expected: true,
		},
		{
			name:     "p- after space",
			text:     " p-if",
			expected: true,
		},
		{
			name:     "class (not a directive)",
			text:     "class",
			expected: false,
		},
		{
			name:     "empty text",
			text:     "",
			expected: false,
		},
		{
			name:     "single char",
			text:     "p",
			expected: false,
		},
		{
			name:     "not at word boundary",
			text:     "xp-if",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isDirectiveAttribute([]byte(tc.text))
			if result != tc.expected {
				t.Errorf("isDirectiveAttribute(%q) = %v, want %v", tc.text, result, tc.expected)
			}
		})
	}
}

func TestFindDirectiveValueStart(t *testing.T) {
	testCases := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "p-if value start",
			text:     `p-if="`,
			expected: 6,
		},
		{
			name:     "p-show value start",
			text:     `p-show="`,
			expected: 8,
		},
		{
			name:     "p-bind:class value start",
			text:     `p-bind:class="`,
			expected: 14,
		},
		{
			name:     "no directive",
			text:     `class="`,
			expected: -1,
		},
		{
			name:     "no equals quote",
			text:     `p-if`,
			expected: -1,
		},
		{
			name:     "empty text",
			text:     "",
			expected: -1,
		},
		{
			name:     "text too short",
			text:     "ab",
			expected: -1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := findDirectiveValueStart([]byte(tc.text))
			if result != tc.expected {
				t.Errorf("findDirectiveValueStart(%q) = %d, want %d", tc.text, result, tc.expected)
			}
		})
	}
}

func TestTryDirectiveValueContext(t *testing.T) {
	testCases := []struct {
		name            string
		text            string
		expectedPrefix  string
		expectedTrigger completionTriggerKind
		expectedResult  bool
	}{
		{
			name:            "inside p-if value",
			text:            `p-if="sta`,
			expectedResult:  true,
			expectedTrigger: triggerDirectiveValue,
			expectedPrefix:  "sta",
		},
		{
			name:            "empty p-show value",
			text:            `p-show="`,
			expectedResult:  true,
			expectedTrigger: triggerDirectiveValue,
			expectedPrefix:  "",
		},
		{
			name:           "closed directive value",
			text:           `p-if="state.visible" class`,
			expectedResult: false,
		},
		{
			name:           "not in directive",
			text:           `class="test`,
			expectedResult: false,
		},
		{
			name:           "empty text",
			text:           "",
			expectedResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &completionContext{}
			result := tryDirectiveValueContext(ctx, []byte(tc.text))
			if result != tc.expectedResult {
				t.Errorf("tryDirectiveValueContext() = %v, want %v", result, tc.expectedResult)
			}
			if result {
				if ctx.TriggerKind != tc.expectedTrigger {
					t.Errorf("TriggerKind = %v, want %v", ctx.TriggerKind, tc.expectedTrigger)
				}
				if ctx.Prefix != tc.expectedPrefix {
					t.Errorf("Prefix = %q, want %q", ctx.Prefix, tc.expectedPrefix)
				}
				if !ctx.InDirective {
					t.Error("expected InDirective to be true")
				}
			}
		})
	}
}

func TestTryActionNamespaceContext(t *testing.T) {
	testCases := []struct {
		name             string
		textBeforeCursor string
		expectedPrefix   string
		expectMatch      bool
	}{
		{
			name:             "action. triggers completion",
			textBeforeCursor: "action.",
			expectMatch:      true,
			expectedPrefix:   "",
		},
		{
			name:             "action.cus triggers completion with prefix",
			textBeforeCursor: "action.cus",
			expectMatch:      true,
			expectedPrefix:   "cus",
		},
		{
			name:             "action.customer. triggers with dotted prefix",
			textBeforeCursor: "action.customer.",
			expectMatch:      true,
			expectedPrefix:   "customer.",
		},
		{
			name:             "action.customer.cr triggers with partial name",
			textBeforeCursor: "action.customer.cr",
			expectMatch:      true,
			expectedPrefix:   "customer.cr",
		},
		{
			name:             "action without dot doesn't trigger",
			textBeforeCursor: "action",
			expectMatch:      false,
		},
		{
			name:             "xaction. doesn't trigger (word boundary)",
			textBeforeCursor: "xaction.",
			expectMatch:      false,
		},
		{
			name:             "action. in expression context",
			textBeforeCursor: `p-on:click.prevent="action.`,
			expectMatch:      true,
			expectedPrefix:   "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var ctx completionContext
			matched := tryActionNamespaceContext(&ctx, []byte(tc.textBeforeCursor))

			if matched != tc.expectMatch {
				t.Errorf("tryActionNamespaceContext() matched = %v, want %v", matched, tc.expectMatch)
				return
			}

			if !tc.expectMatch {
				return
			}

			if ctx.TriggerKind != triggerActionNamespace {
				t.Errorf("TriggerKind = %v, want %v", ctx.TriggerKind, triggerActionNamespace)
			}
			if ctx.Prefix != tc.expectedPrefix {
				t.Errorf("Prefix = %q, want %q", ctx.Prefix, tc.expectedPrefix)
			}
		})
	}
}

func TestTryEventHandlercompletionContext(t *testing.T) {
	testCases := []struct {
		name            string
		line            string
		expectedPrefix  string
		cursorPosition  int
		expectedTrigger completionTriggerKind
		expectedResult  bool
	}{
		{
			name:           "no event handler present",
			line:           `<div class="test">`,
			cursorPosition: 10,
			expectedResult: false,
		},
		{
			name:            "inside click handler value",
			line:            `<button p-on:click="handle`,
			cursorPosition:  26,
			expectedResult:  true,
			expectedTrigger: triggerEventHandler,
			expectedPrefix:  "handle",
		},
		{
			name:            "empty click handler value",
			line:            `<button p-on:click="`,
			cursorPosition:  20,
			expectedResult:  true,
			expectedTrigger: triggerEventHandler,
			expectedPrefix:  "",
		},
		{
			name:           "handler already closed",
			line:           `<button p-on:click="handleClick" class`,
			cursorPosition: 38,
			expectedResult: false,
		},
		{
			name:            "inside submit handler",
			line:            `<form p-on:submit="onSub`,
			cursorPosition:  24,
			expectedResult:  true,
			expectedTrigger: triggerEventHandler,
			expectedPrefix:  "onSub",
		},
		{
			name:            "inside input handler",
			line:            `<input p-on:input="val`,
			cursorPosition:  22,
			expectedResult:  true,
			expectedTrigger: triggerEventHandler,
			expectedPrefix:  "val",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &completionContext{}
			result := tryEventHandlercompletionContext(ctx, tc.line, tc.cursorPosition)
			if result != tc.expectedResult {
				t.Errorf("tryEventHandlercompletionContext() = %v, want %v", result, tc.expectedResult)
			}
			if result {
				if ctx.TriggerKind != tc.expectedTrigger {
					t.Errorf("TriggerKind = %v, want %v", ctx.TriggerKind, tc.expectedTrigger)
				}
				if ctx.Prefix != tc.expectedPrefix {
					t.Errorf("Prefix = %q, want %q", ctx.Prefix, tc.expectedPrefix)
				}
			}
		})
	}
}

func TestTryPartialNamecompletionContext(t *testing.T) {
	testCases := []struct {
		name            string
		line            string
		expectedPrefix  string
		cursorPosition  int
		expectedTrigger completionTriggerKind
		expectedResult  bool
	}{
		{
			name:           "not inside partial name call",
			line:           `someFunction("test")`,
			cursorPosition: 15,
			expectedResult: false,
		},
		{
			name:            "inside reloadPartial with single quote",
			line:            `reloadPartial('status_`,
			cursorPosition:  22,
			expectedResult:  true,
			expectedTrigger: triggerPartialName,
			expectedPrefix:  "status_",
		},
		{
			name:            "inside reloadGroup with double quote",
			line:            `reloadGroup("badge_`,
			cursorPosition:  19,
			expectedResult:  true,
			expectedTrigger: triggerPartialName,
			expectedPrefix:  "badge_",
		},
		{
			name:           "closed partial name call",
			line:           `reloadPartial('status')`,
			cursorPosition: 23,
			expectedResult: false,
		},
		{
			name:            "empty partial name",
			line:            `reloadPartial('`,
			cursorPosition:  15,
			expectedResult:  true,
			expectedTrigger: triggerPartialName,
			expectedPrefix:  "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &completionContext{}
			result := tryPartialNamecompletionContext(ctx, tc.line, tc.cursorPosition)
			if result != tc.expectedResult {
				t.Errorf("tryPartialNamecompletionContext() = %v, want %v", result, tc.expectedResult)
			}
			if result {
				if ctx.TriggerKind != tc.expectedTrigger {
					t.Errorf("TriggerKind = %v, want %v", ctx.TriggerKind, tc.expectedTrigger)
				}
				if ctx.Prefix != tc.expectedPrefix {
					t.Errorf("Prefix = %q, want %q", ctx.Prefix, tc.expectedPrefix)
				}
			}
		})
	}
}

func TestTryRefsAccesscompletionContext(t *testing.T) {
	testCases := []struct {
		name            string
		line            string
		expectedPrefix  string
		cursorPosition  int
		expectedTrigger completionTriggerKind
		expectedResult  bool
	}{
		{
			name:           "not in refs context",
			line:           `state.value`,
			cursorPosition: 11,
			expectedResult: false,
		},
		{
			name:            "after refs. with prefix",
			line:            `refs.myInput`,
			cursorPosition:  12,
			expectedResult:  true,
			expectedTrigger: triggerRefAccess,
			expectedPrefix:  "myInput",
		},
		{
			name:            "after refs. with no prefix",
			line:            `refs.`,
			cursorPosition:  5,
			expectedResult:  true,
			expectedTrigger: triggerRefAccess,
			expectedPrefix:  "",
		},
		{
			name:           "refs followed by non-ident char",
			line:           `refs.myInput.`,
			cursorPosition: 13,
			expectedResult: false,
		},
		{
			name:            "partial prefix after refs.",
			line:            `refs.inp`,
			cursorPosition:  8,
			expectedResult:  true,
			expectedTrigger: triggerRefAccess,
			expectedPrefix:  "inp",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &completionContext{}
			result := tryRefsAccesscompletionContext(ctx, tc.line, tc.cursorPosition)
			if result != tc.expectedResult {
				t.Errorf("tryRefsAccesscompletionContext() = %v, want %v", result, tc.expectedResult)
			}
			if result {
				if ctx.TriggerKind != tc.expectedTrigger {
					t.Errorf("TriggerKind = %v, want %v", ctx.TriggerKind, tc.expectedTrigger)
				}
				if ctx.Prefix != tc.expectedPrefix {
					t.Errorf("Prefix = %q, want %q", ctx.Prefix, tc.expectedPrefix)
				}
			}
		})
	}
}

func TestTryStateAccesscompletionContext(t *testing.T) {

	content := "<template>\n<div>hello</div>\n</template>\n<script>\nstate.coun\n</script>"
	document := newTestDocumentBuilder().
		WithContent(content).
		Build()

	testCases := []struct {
		name            string
		lineString      string
		expectedPrefix  string
		cursorPosition  int
		expectedTrigger completionTriggerKind
		position        protocol.Position
		expectedResult  bool
	}{
		{
			name:           "state. outside script block",
			lineString:     "state.coun",
			cursorPosition: 10,
			position:       protocol.Position{Line: 0, Character: 10},
			expectedResult: false,
		},
		{
			name:            "state. inside client script block",
			lineString:      "state.coun",
			cursorPosition:  10,
			position:        protocol.Position{Line: 4, Character: 10},
			expectedResult:  true,
			expectedTrigger: triggerStateAccessJS,
			expectedPrefix:  "coun",
		},
		{
			name:           "no state. pattern present",
			lineString:     "props.value",
			cursorPosition: 11,
			position:       protocol.Position{Line: 4, Character: 11},
			expectedResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &completionContext{}
			result := tryStateAccesscompletionContext(ctx, document, tc.position, tc.lineString, tc.cursorPosition)
			if result != tc.expectedResult {
				t.Errorf("tryStateAccesscompletionContext() = %v, want %v", result, tc.expectedResult)
			}
			if result {
				if ctx.TriggerKind != tc.expectedTrigger {
					t.Errorf("TriggerKind = %v, want %v", ctx.TriggerKind, tc.expectedTrigger)
				}
				if ctx.Prefix != tc.expectedPrefix {
					t.Errorf("Prefix = %q, want %q", ctx.Prefix, tc.expectedPrefix)
				}
			}
		})
	}
}

func TestTryPropsAccesscompletionContext(t *testing.T) {

	content := "<template>\n<div>hello</div>\n</template>\n<script>\nprops.tit\n</script>"
	document := newTestDocumentBuilder().
		WithContent(content).
		Build()

	testCases := []struct {
		name            string
		lineString      string
		expectedPrefix  string
		cursorPosition  int
		expectedTrigger completionTriggerKind
		position        protocol.Position
		expectedResult  bool
	}{
		{
			name:           "props. outside script block",
			lineString:     "props.tit",
			cursorPosition: 9,
			position:       protocol.Position{Line: 0, Character: 9},
			expectedResult: false,
		},
		{
			name:            "props. inside client script block",
			lineString:      "props.tit",
			cursorPosition:  9,
			position:        protocol.Position{Line: 4, Character: 9},
			expectedResult:  true,
			expectedTrigger: triggerPropsAccessJS,
			expectedPrefix:  "tit",
		},
		{
			name:           "no props. pattern present",
			lineString:     "state.count",
			cursorPosition: 11,
			position:       protocol.Position{Line: 4, Character: 11},
			expectedResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &completionContext{}
			result := tryPropsAccesscompletionContext(ctx, document, tc.position, tc.lineString, tc.cursorPosition)
			if result != tc.expectedResult {
				t.Errorf("tryPropsAccesscompletionContext() = %v, want %v", result, tc.expectedResult)
			}
			if result {
				if ctx.TriggerKind != tc.expectedTrigger {
					t.Errorf("TriggerKind = %v, want %v", ctx.TriggerKind, tc.expectedTrigger)
				}
				if ctx.Prefix != tc.expectedPrefix {
					t.Errorf("Prefix = %q, want %q", ctx.Prefix, tc.expectedPrefix)
				}
			}
		})
	}
}

func TestAnalyseCompletionContext(t *testing.T) {
	testCases := []struct {
		name              string
		content           string
		expectedPrefix    string
		expectedBaseExpr  string
		expectedTrigger   completionTriggerKind
		position          protocol.Position
		expectedDirective bool
	}{
		{
			name:            "empty document returns scope trigger",
			content:         "",
			position:        protocol.Position{Line: 0, Character: 0},
			expectedTrigger: triggerScope,
		},
		{
			name:            "cursor beyond line count returns scope trigger",
			content:         "hello",
			position:        protocol.Position{Line: 5, Character: 0},
			expectedTrigger: triggerScope,
		},
		{
			name:            "cursor beyond line length returns scope trigger",
			content:         "hello",
			position:        protocol.Position{Line: 0, Character: 100},
			expectedTrigger: triggerScope,
		},
		{
			name:             "member access with trailing dot",
			content:          "state.",
			position:         protocol.Position{Line: 0, Character: 6},
			expectedTrigger:  triggerMemberAccess,
			expectedPrefix:   "",
			expectedBaseExpr: "state",
		},
		{
			name:             "member access with partial name",
			content:          "state.us",
			position:         protocol.Position{Line: 0, Character: 8},
			expectedTrigger:  triggerMemberAccess,
			expectedPrefix:   "us",
			expectedBaseExpr: "state",
		},
		{
			name:            "directive prefix p-",
			content:         "<div p-",
			position:        protocol.Position{Line: 0, Character: 7},
			expectedTrigger: triggerDirective,
			expectedPrefix:  "",
		},
		{
			name:            "directive prefix p-sh",
			content:         "<div p-sh",
			position:        protocol.Position{Line: 0, Character: 9},
			expectedTrigger: triggerDirective,
			expectedPrefix:  "sh",
		},
		{
			name:              "directive value context p-if equals quote",
			content:           `<div p-if="sta`,
			position:          protocol.Position{Line: 0, Character: 14},
			expectedTrigger:   triggerDirectiveValue,
			expectedPrefix:    "sta",
			expectedDirective: true,
		},
		{
			name:            "partial alias context is equals quote",
			content:         `<div is="status_`,
			position:        protocol.Position{Line: 0, Character: 16},
			expectedTrigger: triggerPartialAlias,
			expectedPrefix:  "status_",
		},
		{
			name:            "piko namespace",
			content:         "piko.",
			position:        protocol.Position{Line: 0, Character: 5},
			expectedTrigger: triggerPikoNamespace,
			expectedPrefix:  "",
		},
		{
			name:            "action namespace",
			content:         "action.cus",
			position:        protocol.Position{Line: 0, Character: 10},
			expectedTrigger: triggerActionNamespace,
			expectedPrefix:  "cus",
		},
		{
			name:            "plain text returns scope trigger",
			content:         "hello world",
			position:        protocol.Position{Line: 0, Character: 5},
			expectedTrigger: triggerScope,
		},
		{
			name:             "multiline document second line with member access",
			content:          "first line\nstate.",
			position:         protocol.Position{Line: 1, Character: 6},
			expectedTrigger:  triggerMemberAccess,
			expectedPrefix:   "",
			expectedBaseExpr: "state",
		},
		{
			name:              "event handler triggers directive value first due to priority",
			content:           `<button p-on:click="handle`,
			position:          protocol.Position{Line: 0, Character: 26},
			expectedTrigger:   triggerDirectiveValue,
			expectedPrefix:    "handle",
			expectedDirective: true,
		},
		{
			name:             "refs. triggers member access first due to priority",
			content:          `refs.myInput`,
			position:         protocol.Position{Line: 0, Character: 12},
			expectedTrigger:  triggerMemberAccess,
			expectedPrefix:   "myInput",
			expectedBaseExpr: "refs",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document := newTestDocumentBuilder().
				WithContent(tc.content).
				Build()
			ctx := analyseCompletionContext(document, tc.position)

			if ctx.TriggerKind != tc.expectedTrigger {
				t.Errorf("TriggerKind = %v, want %v", ctx.TriggerKind, tc.expectedTrigger)
			}
			if ctx.Prefix != tc.expectedPrefix {
				t.Errorf("Prefix = %q, want %q", ctx.Prefix, tc.expectedPrefix)
			}
			if tc.expectedBaseExpr != "" && ctx.BaseExpression != tc.expectedBaseExpr {
				t.Errorf("BaseExpression = %q, want %q", ctx.BaseExpression, tc.expectedBaseExpr)
			}
			if tc.expectedDirective && !ctx.InDirective {
				t.Error("expected InDirective to be true")
			}
		})
	}
}

func TestCollectNodeDirectives(t *testing.T) {
	t.Run("collects all standard directives", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			DirIf:       &ast_domain.Directive{Type: ast_domain.DirectiveIf},
			DirElseIf:   &ast_domain.Directive{Type: ast_domain.DirectiveElseIf},
			DirFor:      &ast_domain.Directive{Type: ast_domain.DirectiveFor},
			DirShow:     &ast_domain.Directive{Type: ast_domain.DirectiveShow},
			DirModel:    &ast_domain.Directive{Type: ast_domain.DirectiveModel},
			DirText:     &ast_domain.Directive{Type: ast_domain.DirectiveText},
			DirHTML:     &ast_domain.Directive{Type: ast_domain.DirectiveHTML},
			DirClass:    &ast_domain.Directive{Type: ast_domain.DirectiveClass},
			DirStyle:    &ast_domain.Directive{Type: ast_domain.DirectiveStyle},
			DirKey:      nil,
			DirContext:  nil,
			DirScaffold: nil,
		}

		directives := collectNodeDirectives(node)

		if len(directives) < 12 {
			t.Errorf("expected at least 12 directives in slice, got %d", len(directives))
		}

		nonNil := 0
		for _, d := range directives {
			if d != nil {
				nonNil++
			}
		}
		if nonNil != 9 {
			t.Errorf("expected 9 non-nil directives, got %d", nonNil)
		}
	})

	t.Run("collects bind directives", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			Binds: map[string]*ast_domain.Directive{
				"class": {Type: ast_domain.DirectiveBind, Arg: "class"},
				"style": {Type: ast_domain.DirectiveBind, Arg: "style"},
			},
		}

		directives := collectNodeDirectives(node)
		bindCount := 0
		for _, d := range directives {
			if d != nil && d.Type == ast_domain.DirectiveBind {
				bindCount++
			}
		}
		if bindCount != 2 {
			t.Errorf("expected 2 bind directives, got %d", bindCount)
		}
	})

	t.Run("collects event directives", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			OnEvents: map[string][]ast_domain.Directive{
				"click": {{Type: ast_domain.DirectiveOn, Arg: "click"}},
				"submit": {
					{Type: ast_domain.DirectiveOn, Arg: "submit"},
					{Type: ast_domain.DirectiveOn, Arg: "submit"},
				},
			},
		}

		directives := collectNodeDirectives(node)
		eventCount := 0
		for _, d := range directives {
			if d != nil && d.Type == ast_domain.DirectiveOn {
				eventCount++
			}
		}
		if eventCount != 3 {
			t.Errorf("expected 3 event directives, got %d", eventCount)
		}
	})

	t.Run("empty node returns only nil standard slots", func(t *testing.T) {
		node := &ast_domain.TemplateNode{}
		directives := collectNodeDirectives(node)
		for _, d := range directives {
			if d != nil {
				t.Error("expected all directives from an empty node to be nil")
				break
			}
		}
	})
}

func TestProcessNodeDynamicAttrs(t *testing.T) {
	t.Run("adds dynamic attribute expressions to range map", func(t *testing.T) {
		expression := &ast_domain.Identifier{
			Name:         "title",
			SourceLength: 5,
			RelativeLocation: ast_domain.Location{
				Line:   0,
				Column: 0,
			},
		}
		node := &ast_domain.TemplateNode{
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{
					Name:       "title",
					Expression: expression,
					Location: ast_domain.Location{
						Line:   1,
						Column: 10,
					},
				},
			},
		}

		rangeMap := make(map[ast_domain.Expression]protocol.Range)
		processNodeDynamicAttrs(node, rangeMap)

		if len(rangeMap) != 1 {
			t.Errorf("expected 1 entry in range map, got %d", len(rangeMap))
		}
		if _, ok := rangeMap[expression]; !ok {
			t.Error("expected expression to be present in range map")
		}
	})

	t.Run("empty dynamic attributes produce empty range map", func(t *testing.T) {
		node := &ast_domain.TemplateNode{}
		rangeMap := make(map[ast_domain.Expression]protocol.Range)
		processNodeDynamicAttrs(node, rangeMap)

		if len(rangeMap) != 0 {
			t.Errorf("expected empty range map, got %d entries", len(rangeMap))
		}
	})
}

func TestProcessNodeRichText(t *testing.T) {
	t.Run("adds non-literal rich text expressions", func(t *testing.T) {
		expression := &ast_domain.Identifier{
			Name:         "userName",
			SourceLength: 8,
			RelativeLocation: ast_domain.Location{
				Line:   0,
				Column: 0,
			},
		}
		node := &ast_domain.TemplateNode{
			RichText: []ast_domain.TextPart{
				{IsLiteral: true, Literal: "Hello, "},
				{
					IsLiteral:  false,
					Expression: expression,
					Location: ast_domain.Location{
						Line:   1,
						Column: 8,
					},
				},
			},
		}

		rangeMap := make(map[ast_domain.Expression]protocol.Range)
		processNodeRichText(node, rangeMap)

		if len(rangeMap) != 1 {
			t.Errorf("expected 1 entry in range map, got %d", len(rangeMap))
		}
		if _, ok := rangeMap[expression]; !ok {
			t.Error("expected expression to be present in range map")
		}
	})

	t.Run("literal-only rich text produces empty range map", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			RichText: []ast_domain.TextPart{
				{IsLiteral: true, Literal: "Just text"},
			},
		}

		rangeMap := make(map[ast_domain.Expression]protocol.Range)
		processNodeRichText(node, rangeMap)

		if len(rangeMap) != 0 {
			t.Errorf("expected empty range map, got %d entries", len(rangeMap))
		}
	})

	t.Run("empty rich text produces empty range map", func(t *testing.T) {
		node := &ast_domain.TemplateNode{}
		rangeMap := make(map[ast_domain.Expression]protocol.Range)
		processNodeRichText(node, rangeMap)

		if len(rangeMap) != 0 {
			t.Errorf("expected empty range map, got %d entries", len(rangeMap))
		}
	})
}

func TestProcessNodeDirectives(t *testing.T) {
	t.Run("adds directive with non-synthetic location", func(t *testing.T) {
		expression := &ast_domain.Identifier{
			Name:         "visible",
			SourceLength: 7,
			RelativeLocation: ast_domain.Location{
				Line:   0,
				Column: 0,
			},
		}
		node := &ast_domain.TemplateNode{
			DirIf: &ast_domain.Directive{
				Type:       ast_domain.DirectiveIf,
				Expression: expression,
				Location: ast_domain.Location{
					Line:   5,
					Column: 10,
				},
				AttributeRange: ast_domain.Range{
					Start: ast_domain.Location{Line: 5, Column: 1},
					End:   ast_domain.Location{Line: 5, Column: 20},
				},
			},
		}

		rangeMap := make(map[ast_domain.Expression]protocol.Range)
		processNodeDirectives(node, rangeMap)

		if len(rangeMap) != 1 {
			t.Errorf("expected 1 entry in range map, got %d", len(rangeMap))
		}
	})

	t.Run("skips directive with synthetic location", func(t *testing.T) {
		expression := &ast_domain.Identifier{
			Name:         "hidden",
			SourceLength: 6,
		}
		node := &ast_domain.TemplateNode{
			DirIf: &ast_domain.Directive{
				Type:       ast_domain.DirectiveIf,
				Expression: expression,
				Location: ast_domain.Location{
					Line:   1,
					Column: 1,
				},
				AttributeRange: ast_domain.Range{
					Start: ast_domain.Location{Line: 0, Column: 0},
					End:   ast_domain.Location{Line: 0, Column: 0},
				},
			},
		}

		rangeMap := make(map[ast_domain.Expression]protocol.Range)
		processNodeDirectives(node, rangeMap)

		if len(rangeMap) != 0 {
			t.Errorf("expected empty range map for synthetic location, got %d entries", len(rangeMap))
		}
	})

	t.Run("skips nil directives", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			DirIf:   nil,
			DirShow: nil,
		}

		rangeMap := make(map[ast_domain.Expression]protocol.Range)
		processNodeDirectives(node, rangeMap)

		if len(rangeMap) != 0 {
			t.Errorf("expected empty range map for nil directives, got %d entries", len(rangeMap))
		}
	})
}

func TestProcessExpressionsInNode(t *testing.T) {
	t.Run("processes all expression types from a node", func(t *testing.T) {
		dynExpr := &ast_domain.Identifier{
			Name:         "title",
			SourceLength: 5,
			RelativeLocation: ast_domain.Location{
				Line:   0,
				Column: 0,
			},
		}
		dirExpr := &ast_domain.Identifier{
			Name:         "isVisible",
			SourceLength: 9,
			RelativeLocation: ast_domain.Location{
				Line:   0,
				Column: 0,
			},
		}
		richExpr := &ast_domain.Identifier{
			Name:         "greeting",
			SourceLength: 8,
			RelativeLocation: ast_domain.Location{
				Line:   0,
				Column: 0,
			},
		}

		node := &ast_domain.TemplateNode{
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{
					Name:       "title",
					Expression: dynExpr,
					Location:   ast_domain.Location{Line: 1, Column: 5},
				},
			},
			DirIf: &ast_domain.Directive{
				Type:       ast_domain.DirectiveIf,
				Expression: dirExpr,
				Location:   ast_domain.Location{Line: 1, Column: 20},
				AttributeRange: ast_domain.Range{
					Start: ast_domain.Location{Line: 1, Column: 15},
					End:   ast_domain.Location{Line: 1, Column: 30},
				},
			},
			RichText: []ast_domain.TextPart{
				{
					IsLiteral:  false,
					Expression: richExpr,
					Location:   ast_domain.Location{Line: 2, Column: 1},
				},
			},
		}

		rangeMap := make(map[ast_domain.Expression]protocol.Range)
		processExpressionsInNode(node, rangeMap)

		if len(rangeMap) != 3 {
			t.Errorf("expected 3 entries in range map, got %d", len(rangeMap))
		}
	})

	t.Run("empty node produces empty range map", func(t *testing.T) {
		node := &ast_domain.TemplateNode{}
		rangeMap := make(map[ast_domain.Expression]protocol.Range)
		processExpressionsInNode(node, rangeMap)

		if len(rangeMap) != 0 {
			t.Errorf("expected empty range map, got %d entries", len(rangeMap))
		}
	})
}

func TestAddExpressionTreeToRangeMap(t *testing.T) {
	t.Run("adds single identifier expression", func(t *testing.T) {
		expression := &ast_domain.Identifier{
			Name:         "count",
			SourceLength: 5,
			RelativeLocation: ast_domain.Location{
				Line:   0,
				Column: 0,
			},
		}
		baseLocation := ast_domain.Location{Line: 3, Column: 10}
		rangeMap := make(map[ast_domain.Expression]protocol.Range)

		addExpressionTreeToRangeMap(expression, baseLocation, rangeMap)

		if len(rangeMap) != 1 {
			t.Errorf("expected 1 entry in range map, got %d", len(rangeMap))
		}
		r, ok := rangeMap[expression]
		if !ok {
			t.Fatal("expected expression to be in range map")
		}

		if r.Start.Line != 2 || r.Start.Character != 9 {
			t.Errorf("expected start position (2, 9), got (%d, %d)", r.Start.Line, r.Start.Character)
		}
	})

	t.Run("adds member expression tree with base and property", func(t *testing.T) {
		base := &ast_domain.Identifier{
			Name:         "state",
			SourceLength: 5,
			RelativeLocation: ast_domain.Location{
				Line:   0,
				Column: 0,
			},
		}
		prop := &ast_domain.Identifier{
			Name:         "name",
			SourceLength: 4,
			RelativeLocation: ast_domain.Location{
				Line:   0,
				Column: 6,
			},
		}
		memberExpr := &ast_domain.MemberExpression{
			Base:     base,
			Property: prop,
			RelativeLocation: ast_domain.Location{
				Line:   0,
				Column: 0,
			},
			SourceLength: 10,
		}

		baseLocation := ast_domain.Location{Line: 1, Column: 1}
		rangeMap := make(map[ast_domain.Expression]protocol.Range)

		addExpressionTreeToRangeMap(memberExpr, baseLocation, rangeMap)

		if len(rangeMap) != 3 {
			t.Errorf("expected 3 entries in range map, got %d", len(rangeMap))
		}
	})

	t.Run("handles nil expression gracefully", func(t *testing.T) {
		baseLocation := ast_domain.Location{Line: 1, Column: 1}
		rangeMap := make(map[ast_domain.Expression]protocol.Range)

		addExpressionTreeToRangeMap(nil, baseLocation, rangeMap)

		if len(rangeMap) != 0 {
			t.Errorf("expected empty range map for nil expression, got %d entries", len(rangeMap))
		}
	})
}

func TestBuildExpressionRangeMap(t *testing.T) {
	docPath := "test/file.pkc"

	t.Run("collects expressions from matching document path", func(t *testing.T) {
		expression := &ast_domain.Identifier{
			Name:         "count",
			SourceLength: 5,
			RelativeLocation: ast_domain.Location{
				Line:   0,
				Column: 0,
			},
		}
		node := &ast_domain.TemplateNode{
			TagName:  "div",
			NodeType: ast_domain.NodeElement,
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalSourcePath: &docPath,
			},
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{
					Name:       "count",
					Expression: expression,
					Location:   ast_domain.Location{Line: 1, Column: 5},
				},
			},
		}
		tree := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{node},
		}

		rangeMap := buildExpressionRangeMap(tree, docPath)

		if len(rangeMap) != 1 {
			t.Errorf("expected 1 entry in range map, got %d", len(rangeMap))
		}
	})

	t.Run("skips nodes from different document path", func(t *testing.T) {
		expression := &ast_domain.Identifier{
			Name:         "count",
			SourceLength: 5,
		}
		node := &ast_domain.TemplateNode{
			TagName:  "div",
			NodeType: ast_domain.NodeElement,
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalSourcePath: new("other/file.pkc"),
			},
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{
					Name:       "count",
					Expression: expression,
					Location:   ast_domain.Location{Line: 1, Column: 5},
				},
			},
		}
		tree := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{node},
		}

		rangeMap := buildExpressionRangeMap(tree, docPath)

		if len(rangeMap) != 0 {
			t.Errorf("expected empty range map for non-matching path, got %d entries", len(rangeMap))
		}
	})

	t.Run("skips nodes without GoAnnotations", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			TagName:  "div",
			NodeType: ast_domain.NodeElement,
		}
		tree := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{node},
		}

		rangeMap := buildExpressionRangeMap(tree, docPath)

		if len(rangeMap) != 0 {
			t.Errorf("expected empty range map for node without annotations, got %d entries", len(rangeMap))
		}
	})

	t.Run("skips nodes with nil OriginalSourcePath", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			TagName:       "div",
			NodeType:      ast_domain.NodeElement,
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{},
		}
		tree := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{node},
		}

		rangeMap := buildExpressionRangeMap(tree, docPath)

		if len(rangeMap) != 0 {
			t.Errorf("expected empty range map for nil source path, got %d entries", len(rangeMap))
		}
	})

	t.Run("processes child nodes recursively", func(t *testing.T) {
		childExpr := &ast_domain.Identifier{
			Name:         "name",
			SourceLength: 4,
			RelativeLocation: ast_domain.Location{
				Line:   0,
				Column: 0,
			},
		}
		child := &ast_domain.TemplateNode{
			TagName:  "span",
			NodeType: ast_domain.NodeElement,
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalSourcePath: &docPath,
			},
			RichText: []ast_domain.TextPart{
				{
					IsLiteral:  false,
					Expression: childExpr,
					Location:   ast_domain.Location{Line: 2, Column: 3},
				},
			},
		}
		parent := &ast_domain.TemplateNode{
			TagName:  "div",
			NodeType: ast_domain.NodeElement,
			Children: []*ast_domain.TemplateNode{child},
		}
		tree := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{parent},
		}

		rangeMap := buildExpressionRangeMap(tree, docPath)

		if len(rangeMap) != 1 {
			t.Errorf("expected 1 entry from child node, got %d", len(rangeMap))
		}
	})
}
