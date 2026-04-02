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
	"strings"
	"testing"

	"go.lsp.dev/protocol"
)

func TestCheckDirectiveHoverContext_ControlFlow(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		line         string
		expectedName string
		cursor       int
	}{
		{
			name:         "p-if directive",
			line:         `<div p-if="state.visible">`,
			cursor:       7,
			expectedName: "p-if",
		},
		{
			name:         "p-else-if directive",
			line:         `<div p-else-if="state.other">`,
			cursor:       10,
			expectedName: "p-else-if",
		},
		{
			name:         "p-else directive",
			line:         `<div p-else>`,
			cursor:       7,
			expectedName: "p-else",
		},
		{
			name:         "p-for directive",
			line:         `<li p-for="item in items">`,
			cursor:       6,
			expectedName: "p-for",
		},
		{
			name:         "p-show directive",
			line:         `<span p-show="state.isVisible">`,
			cursor:       8,
			expectedName: "p-show",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := &document{}
			position := protocol.Position{Line: 0, Character: uint32(tc.cursor)}

			ctx := d.checkDirectiveHoverContext(tc.line, tc.cursor, position)

			if ctx == nil {
				t.Fatalf("expected context, got nil")
			}
			if ctx.Name != tc.expectedName {
				t.Errorf("expected name %q, got %q", tc.expectedName, ctx.Name)
			}
			if ctx.Kind != PKDefDirective {
				t.Errorf("expected kind PKDefDirective, got %v", ctx.Kind)
			}
		})
	}
}

func TestCheckDirectiveHoverContext_DataBinding(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		line         string
		expectedName string
		cursor       int
	}{
		{
			name:         "p-text directive",
			line:         `<span p-text="state.name">`,
			cursor:       8,
			expectedName: "p-text",
		},
		{
			name:         "p-html directive",
			line:         `<div p-html="state.content">`,
			cursor:       7,
			expectedName: "p-html",
		},
		{
			name:         "p-model directive",
			line:         `<input p-model="state.value" />`,
			cursor:       10,
			expectedName: "p-model",
		},
		{
			name:         "p-bind with argument",
			line:         `<a p-bind:href="state.url">`,
			cursor:       6,
			expectedName: "p-bind",
		},
		{
			name:         "p-bind:src",
			line:         `<img p-bind:src="state.image" />`,
			cursor:       10,
			expectedName: "p-bind",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := &document{}
			position := protocol.Position{Line: 0, Character: uint32(tc.cursor)}

			ctx := d.checkDirectiveHoverContext(tc.line, tc.cursor, position)

			if ctx == nil {
				t.Fatalf("expected context, got nil")
			}
			if ctx.Name != tc.expectedName {
				t.Errorf("expected name %q, got %q", tc.expectedName, ctx.Name)
			}
		})
	}
}

func TestCheckDirectiveHoverContext_Events(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		line         string
		expectedName string
		cursor       int
	}{
		{
			name:         "p-on:click",
			line:         `<button p-on:click="handleClick">`,
			cursor:       10,
			expectedName: "p-on",
		},
		{
			name:         "p-on:submit.prevent",
			line:         `<form p-on:submit.prevent="submitForm">`,
			cursor:       8,
			expectedName: "p-on",
		},
		{
			name:         "p-on with multiple modifiers",
			line:         `<form p-on:submit.prevent.stop="submit">`,
			cursor:       12,
			expectedName: "p-on",
		},
		{
			name:         "p-event custom",
			line:         `<comp p-event:update="handle">`,
			cursor:       9,
			expectedName: "p-event",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := &document{}
			position := protocol.Position{Line: 0, Character: uint32(tc.cursor)}

			ctx := d.checkDirectiveHoverContext(tc.line, tc.cursor, position)

			if ctx == nil {
				t.Fatalf("expected context, got nil")
			}
			if ctx.Name != tc.expectedName {
				t.Errorf("expected name %q, got %q", tc.expectedName, ctx.Name)
			}
		})
	}
}

func TestCheckDirectiveHoverContext_Styling(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		line         string
		expectedName string
		cursor       int
	}{
		{
			name:         "p-class directive",
			line:         `<div p-class="{ active: true }">`,
			cursor:       7,
			expectedName: "p-class",
		},
		{
			name:         "p-style directive",
			line:         `<div p-style="{ color: 'red' }">`,
			cursor:       7,
			expectedName: "p-style",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := &document{}
			position := protocol.Position{Line: 0, Character: uint32(tc.cursor)}

			ctx := d.checkDirectiveHoverContext(tc.line, tc.cursor, position)

			if ctx == nil {
				t.Fatalf("expected context, got nil")
			}
			if ctx.Name != tc.expectedName {
				t.Errorf("expected name %q, got %q", tc.expectedName, ctx.Name)
			}
		})
	}
}

func TestCheckDirectiveHoverContext_Reference(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		line         string
		expectedName string
		cursor       int
	}{
		{
			name:         "p-ref directive",
			line:         `<canvas p-ref="myCanvas">`,
			cursor:       10,
			expectedName: "p-ref",
		},
		{
			name:         "p-slot directive",
			line:         `<div p-slot="header">`,
			cursor:       7,
			expectedName: "p-slot",
		},
		{
			name:         "p-key directive",
			line:         `<li p-for="i in items" p-key="i.id">`,
			cursor:       26,
			expectedName: "p-key",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := &document{}
			position := protocol.Position{Line: 0, Character: uint32(tc.cursor)}

			ctx := d.checkDirectiveHoverContext(tc.line, tc.cursor, position)

			if ctx == nil {
				t.Fatalf("expected context, got nil")
			}
			if ctx.Name != tc.expectedName {
				t.Errorf("expected name %q, got %q", tc.expectedName, ctx.Name)
			}
		})
	}
}

func TestCheckDirectiveHoverContext_NoMatch(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		line   string
		cursor int
	}{
		{
			name:   "cursor on attribute value",
			line:   `<div p-if="state.visible">`,
			cursor: 15,
		},
		{
			name:   "cursor on regular attribute",
			line:   `<div class="container">`,
			cursor: 8,
		},
		{
			name:   "unknown directive",
			line:   `<div p-unknown="value">`,
			cursor: 7,
		},
		{
			name:   "empty line",
			line:   ``,
			cursor: 0,
		},
		{
			name:   "no directive",
			line:   `<div>Hello</div>`,
			cursor: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := &document{}
			position := protocol.Position{Line: 0, Character: uint32(tc.cursor)}

			ctx := d.checkDirectiveHoverContext(tc.line, tc.cursor, position)

			if ctx != nil {
				t.Errorf("expected nil context, got %+v", ctx)
			}
		})
	}
}

func TestGetDirectiveHover(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		directiveName   string
		expectHeader    string
		expectSyntax    string
		expectExample   bool
		expectModifiers bool
	}{
		{
			name:          "p-if hover",
			directiveName: "p-if",
			expectHeader:  "## `p-if`",
			expectSyntax:  `p-if="expression"`,
			expectExample: true,
		},
		{
			name:            "p-on hover with modifiers",
			directiveName:   "p-on",
			expectHeader:    "## `p-on`",
			expectSyntax:    `p-on:event[.modifier...]="handler"`,
			expectExample:   true,
			expectModifiers: true,
		},
		{
			name:          "p-for hover",
			directiveName: "p-for",
			expectHeader:  "## `p-for`",
			expectSyntax:  `p-for="item in items"`,
			expectExample: true,
		},
		{
			name:          "p-text hover",
			directiveName: "p-text",
			expectHeader:  "## `p-text`",
			expectExample: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := &document{}
			ctx := &PKHoverContext{
				Kind: PKDefDirective,
				Name: tc.directiveName,
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 5},
					End:   protocol.Position{Line: 0, Character: uint32(5 + len(tc.directiveName))},
				},
			}

			hover, err := d.getDirectiveHover(ctx)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if hover == nil {
				t.Fatalf("expected hover, got nil")
			}

			content := hover.Contents.Value
			if !strings.Contains(content, tc.expectHeader) {
				t.Errorf("expected header %q in content:\n%s", tc.expectHeader, content)
			}

			if tc.expectSyntax != "" && !strings.Contains(content, tc.expectSyntax) {
				t.Errorf("expected syntax %q in content:\n%s", tc.expectSyntax, content)
			}

			if tc.expectExample && !strings.Contains(content, "**Example:**") {
				t.Errorf("expected example section in content:\n%s", content)
			}

			if tc.expectModifiers && !strings.Contains(content, "**Modifiers:**") {
				t.Errorf("expected modifiers section in content:\n%s", content)
			}
		})
	}
}

func TestGetDirectiveHover_UnknownDirective(t *testing.T) {
	t.Parallel()

	d := &document{}
	ctx := &PKHoverContext{
		Kind: PKDefDirective,
		Name: "p-nonexistent",
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 5},
			End:   protocol.Position{Line: 0, Character: 18},
		},
	}

	hover, err := d.getDirectiveHover(ctx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hover != nil {
		t.Errorf("expected nil hover for unknown directive, got %+v", hover)
	}
}

func TestNormaliseDirectiveName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		input    string
		expected string
	}{
		{input: "p-if", expected: "p-if"},
		{input: "p-for", expected: "p-for"},
		{input: "p-bind:href", expected: "p-bind"},
		{input: "p-bind:src", expected: "p-bind"},
		{input: "p-on:click", expected: "p-on"},
		{input: "p-on:submit.prevent", expected: "p-on"},
		{input: "p-on:click.prevent.stop", expected: "p-on"},
		{input: "p-event:update", expected: "p-event"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			result := normaliseDirectiveName(tc.input)
			if result != tc.expected {
				t.Errorf("normaliseDirectiveName(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestPikoDirectiveDocuments_AllDirectivesHaveRequiredFields(t *testing.T) {
	t.Parallel()

	for name, document := range pikoDirectiveDocumentations {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if document.Name == "" {
				t.Errorf("directive %q has empty Name", name)
			}
			if document.Name != name {
				t.Errorf("directive %q has mismatched Name %q", name, document.Name)
			}
			if document.Description == "" {
				t.Errorf("directive %q has empty Description", name)
			}
			if document.DocumentsURL == "" {
				t.Errorf("directive %q has empty DocumentsURL", name)
			}
		})
	}
}

func TestFormatDirectiveDocumentation(t *testing.T) {
	t.Parallel()

	document := pikoDirectiveDocumentation{
		Name:         "p-test",
		Description:  "A test directive for testing",
		Syntax:       `p-test="value"`,
		Accepts:      "Test value",
		Example:      `<div p-test="foo"></div>`,
		Note:         "This is a test note.",
		DocumentsURL: "/docs/api/directives/p-test",
	}

	content := formatDirectiveDocumentation(document)

	expectedParts := []string{
		"## `p-test`",
		"A test directive for testing",
		"**Syntax:** `p-test=\"value\"`",
		"**Accepts:** Test value",
		"**Example:**",
		"```html",
		`<div p-test="foo"></div>`,
		"**Note:** This is a test note.",
		"---",
		"[Documentation](/docs/api/directives/p-test)",
	}

	for _, part := range expectedParts {
		if !strings.Contains(content, part) {
			t.Errorf("expected %q in formatted content:\n%s", part, content)
		}
	}
}

func TestFormatDirectiveDocumentation_WithModifiers(t *testing.T) {
	t.Parallel()

	document := pikoDirectiveDocumentation{
		Name:         "p-on",
		Description:  "Event handler",
		Modifiers:    []string{".prevent", ".stop"},
		DocumentsURL: "/docs/api/directives/p-on",
	}

	content := formatDirectiveDocumentation(document)

	expectedParts := []string{
		"**Modifiers:**",
		"`.prevent` - calls preventDefault()",
		"`.stop` - calls stopPropagation()",
	}

	for _, part := range expectedParts {
		if !strings.Contains(content, part) {
			t.Errorf("expected %q in formatted content:\n%s", part, content)
		}
	}
}
