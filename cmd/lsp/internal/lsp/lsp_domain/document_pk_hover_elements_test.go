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

func TestCheckPikoElementHoverContext_OpeningTag(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		line         string
		expectedName string
		cursor       int
	}{
		{
			name:         "piko:img opening tag",
			line:         `<piko:img src="/image.jpg" />`,
			cursor:       5,
			expectedName: "piko:img",
		},
		{
			name:         "piko:svg opening tag",
			line:         `<piko:svg src="/icon.svg" class="icon" />`,
			cursor:       8,
			expectedName: "piko:svg",
		},
		{
			name:         "piko:a opening tag",
			line:         `<piko:a href="/about">Link</piko:a>`,
			cursor:       3,
			expectedName: "piko:a",
		},
		{
			name:         "piko:video opening tag",
			line:         `<piko:video src="/video.mp4" controls />`,
			cursor:       7,
			expectedName: "piko:video",
		},
		{
			name:         "piko:slot opening tag",
			line:         `<piko:slot name="header" />`,
			cursor:       6,
			expectedName: "piko:slot",
		},
		{
			name:         "piko:content opening tag",
			line:         `<piko:content />`,
			cursor:       10,
			expectedName: "piko:content",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := &document{}
			position := protocol.Position{Line: 0, Character: uint32(tc.cursor)}

			ctx := d.checkPikoElementHoverContext(tc.line, tc.cursor, position)

			if ctx == nil {
				t.Fatalf("expected context, got nil")
			}
			if ctx.Name != tc.expectedName {
				t.Errorf("expected name %q, got %q", tc.expectedName, ctx.Name)
			}
			if ctx.Kind != PKDefPikoElement {
				t.Errorf("expected kind PKDefPikoElement, got %v", ctx.Kind)
			}
		})
	}
}

func TestCheckPikoElementHoverContext_ClosingTag(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		line         string
		expectedName string
		cursor       int
	}{
		{
			name:         "piko:img closing tag",
			line:         `</piko:img>`,
			cursor:       5,
			expectedName: "piko:img",
		},
		{
			name:         "piko:a closing tag in context",
			line:         `<piko:a href="/about">Link</piko:a>`,
			cursor:       30,
			expectedName: "piko:a",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := &document{}
			position := protocol.Position{Line: 0, Character: uint32(tc.cursor)}

			ctx := d.checkPikoElementHoverContext(tc.line, tc.cursor, position)

			if ctx == nil {
				t.Fatalf("expected context, got nil")
			}
			if ctx.Name != tc.expectedName {
				t.Errorf("expected name %q, got %q", tc.expectedName, ctx.Name)
			}
		})
	}
}

func TestCheckPikoElementHoverContext_NoMatch(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		line   string
		cursor int
	}{
		{
			name:   "cursor on attribute value",
			line:   `<piko:img src="/image.jpg" />`,
			cursor: 18,
		},
		{
			name:   "cursor on regular element",
			line:   `<div class="container">`,
			cursor: 3,
		},
		{
			name:   "unknown piko tag",
			line:   `<piko:unknown attr="value" />`,
			cursor: 7,
		},
		{
			name:   "cursor before tag",
			line:   `   <piko:img src="/image.jpg" />`,
			cursor: 1,
		},
		{
			name:   "empty line",
			line:   ``,
			cursor: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := &document{}
			position := protocol.Position{Line: 0, Character: uint32(tc.cursor)}

			ctx := d.checkPikoElementHoverContext(tc.line, tc.cursor, position)

			if ctx != nil {
				t.Errorf("expected nil context, got %+v", ctx)
			}
		})
	}
}

func TestGetPikoElementHover(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name               string
		elementName        string
		expectHeader       string
		expectExample      string
		expectDocumentsURL string
		expectRequired     []string
		expectOptional     []string
	}{
		{
			name:               "piko:img hover",
			elementName:        "piko:img",
			expectHeader:       "## `<piko:img>`",
			expectRequired:     []string{"src"},
			expectOptional:     []string{"alt", "sizes", "widths", "densities", "formats", "variant", "cms-media"},
			expectExample:      `<piko:img src="/images/hero.jpg"`,
			expectDocumentsURL: "/docs/api/tags/piko-img",
		},
		{
			name:               "piko:svg hover",
			elementName:        "piko:svg",
			expectHeader:       "## `<piko:svg>`",
			expectRequired:     []string{"src"},
			expectOptional:     []string{"class", "width", "height", "fill"},
			expectExample:      `<piko:svg src="/icons/menu.svg" class="icon" width="24" height="24"></piko:svg>`,
			expectDocumentsURL: "/docs/api/tags/piko-svg",
		},
		{
			name:               "piko:slot hover",
			elementName:        "piko:slot",
			expectHeader:       "## `<piko:slot>`",
			expectRequired:     []string{},
			expectOptional:     []string{"name"},
			expectExample:      `<piko:slot name="header"></piko:slot>`,
			expectDocumentsURL: "/docs/api/tags/piko-slot",
		},
		{
			name:               "piko:content hover",
			elementName:        "piko:content",
			expectHeader:       "## `<piko:content>`",
			expectRequired:     []string{},
			expectOptional:     []string{},
			expectExample:      `<piko:content />`,
			expectDocumentsURL: "/docs/api/tags/piko-content",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := &document{}
			ctx := &PKHoverContext{
				Kind: PKDefPikoElement,
				Name: tc.elementName,
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 1},
					End:   protocol.Position{Line: 0, Character: uint32(1 + len(tc.elementName))},
				},
			}

			hover, err := d.getPikoElementHover(ctx)

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

			for _, attr := range tc.expectRequired {
				if !strings.Contains(content, "`"+attr+"`") {
					t.Errorf("expected required attribute %q in content:\n%s", attr, content)
				}
			}

			for _, attr := range tc.expectOptional {
				if !strings.Contains(content, "`"+attr+"`") {
					t.Errorf("expected optional attribute %q in content:\n%s", attr, content)
				}
			}

			if tc.expectExample != "" && !strings.Contains(content, tc.expectExample) {
				t.Errorf("expected example %q in content:\n%s", tc.expectExample, content)
			}

			if tc.expectDocumentsURL != "" && !strings.Contains(content, tc.expectDocumentsURL) {
				t.Errorf("expected docs URL %q in content:\n%s", tc.expectDocumentsURL, content)
			}
		})
	}
}

func TestGetPikoElementHover_UnknownElement(t *testing.T) {
	t.Parallel()

	d := &document{}
	ctx := &PKHoverContext{
		Kind: PKDefPikoElement,
		Name: "piko:nonexistent",
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 1},
			End:   protocol.Position{Line: 0, Character: 17},
		},
	}

	hover, err := d.getPikoElementHover(ctx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hover != nil {
		t.Errorf("expected nil hover for unknown element, got %+v", hover)
	}
}

func TestFormatPikoElementDocumentation(t *testing.T) {
	t.Parallel()

	document := pikoElementDoc{
		Name:        "piko:test",
		Description: "A test element for testing purposes",
		RequiredAttrs: []pikoAttrDoc{
			{Name: "required", Type: "string", Description: "A required attribute"},
		},
		OptionalAttrs: []pikoAttrDoc{
			{Name: "optional", Type: "boolean", Description: "An optional attribute"},
		},
		Example:      `<piko:test required="value" />`,
		DocumentsURL: "/docs/api/tags/piko-test",
	}

	content := formatPikoElementDocumentation(document)

	expectedParts := []string{
		"## `<piko:test>`",
		"A test element for testing purposes",
		"**Required:**",
		"`required` (string) - A required attribute",
		"**Optional:**",
		"`optional` (boolean) - An optional attribute",
		"**Example:**",
		"```html",
		`<piko:test required="value" />`,
		"---",
		"[Documentation](/docs/api/tags/piko-test)",
	}

	for _, part := range expectedParts {
		if !strings.Contains(content, part) {
			t.Errorf("expected %q in formatted content:\n%s", part, content)
		}
	}
}

func TestPikoElementDocuments_AllElementsHaveRequiredFields(t *testing.T) {
	t.Parallel()

	for name, document := range pikoElementDocuments {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if document.Name == "" {
				t.Errorf("element %q has empty Name", name)
			}
			if document.Name != name {
				t.Errorf("element %q has mismatched Name %q", name, document.Name)
			}
			if document.Description == "" {
				t.Errorf("element %q has empty Description", name)
			}
			if document.Example == "" {
				t.Errorf("element %q has empty Example", name)
			}
			if document.DocumentsURL == "" {
				t.Errorf("element %q has empty DocumentsURL", name)
			}
		})
	}
}

func TestExtractPikoTagName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		line          string
		expectedName  string
		startPosition int
		expectedEnd   int
	}{
		{
			name:          "piko:img",
			line:          "piko:img src=",
			startPosition: 0,
			expectedName:  "piko:img",
			expectedEnd:   8,
		},
		{
			name:          "server_slot",
			line:          "server_slot>",
			startPosition: 0,
			expectedName:  "server_slot",
			expectedEnd:   11,
		},
		{
			name:          "piko:a with space",
			line:          "piko:a href=",
			startPosition: 0,
			expectedName:  "piko:a",
			expectedEnd:   6,
		},
		{
			name:          "empty at end of line",
			line:          "",
			startPosition: 0,
			expectedName:  "",
			expectedEnd:   0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			name, end := extractPikoTagName(tc.line, tc.startPosition)

			if name != tc.expectedName {
				t.Errorf("expected name %q, got %q", tc.expectedName, name)
			}
			if end != tc.expectedEnd {
				t.Errorf("expected end %d, got %d", tc.expectedEnd, end)
			}
		})
	}
}

func TestFindPikoTagStart(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		line     string
		cursor   int
		expected int
	}{
		{
			name:     "opening piko tag",
			line:     "<piko:img src=",
			cursor:   5,
			expected: 1,
		},
		{
			name:     "closing piko tag",
			line:     "</piko:img>",
			cursor:   5,
			expected: 2,
		},
		{
			name:     "no piko tag",
			line:     "<div class=",
			cursor:   5,
			expected: -1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := findPikoTagStart(tc.line, tc.cursor)

			if result != tc.expected {
				t.Errorf("expected %d, got %d", tc.expected, result)
			}
		})
	}
}

func TestPikoTimelineElementDocuments_AllElementsHaveRequiredFields(t *testing.T) {
	t.Parallel()

	for name, document := range pikoTimelineElementDocuments {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if document.Name == "" {
				t.Errorf("element %q has empty Name", name)
			}
			if document.Name != name {
				t.Errorf("element %q has mismatched Name %q", name, document.Name)
			}
			if document.Description == "" {
				t.Errorf("element %q has empty Description", name)
			}
			if document.Example == "" {
				t.Errorf("element %q has empty Example", name)
			}
		})
	}
}
