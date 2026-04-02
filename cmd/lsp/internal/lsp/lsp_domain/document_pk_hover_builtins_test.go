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

func TestCheckBuiltinHoverContext_CoreFunctions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		line         string
		expectedName string
		cursor       int
	}{
		{
			name:         "len function",
			line:         `p-if="len(state.items) > 0"`,
			cursor:       7,
			expectedName: "len",
		},
		{
			name:         "cap function",
			line:         `{{ cap(state.buffer) }}`,
			cursor:       4,
			expectedName: "cap",
		},
		{
			name:         "append function",
			line:         `{{ append(state.items, newItem) }}`,
			cursor:       5,
			expectedName: "append",
		},
		{
			name:         "min function",
			line:         `p-text="min(a, b)"`,
			cursor:       10,
			expectedName: "min",
		},
		{
			name:         "max function",
			line:         `:value="max(state.score, 100)"`,
			cursor:       9,
			expectedName: "max",
		},
		{
			name:         "T translation function",
			line:         `p-text="T('welcome')"`,
			cursor:       8,
			expectedName: "T",
		},
		{
			name:         "LT local translation function",
			line:         `p-text="LT('form.label')"`,
			cursor:       9,
			expectedName: "LT",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := &document{}
			position := protocol.Position{Line: 0, Character: uint32(tc.cursor)}

			ctx := d.checkBuiltinHoverContext(tc.line, tc.cursor, position)

			if ctx == nil {
				t.Fatalf("expected context, got nil")
			}
			if ctx.Name != tc.expectedName {
				t.Errorf("expected name %q, got %q", tc.expectedName, ctx.Name)
			}
			if ctx.Kind != PKDefBuiltinFunction {
				t.Errorf("expected kind PKDefBuiltinFunction, got %v", ctx.Kind)
			}
		})
	}
}

func TestCheckBuiltinHoverContext_TypeCoercion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		line         string
		expectedName string
		cursor       int
	}{
		{
			name:         "string coercion",
			line:         `p-text="string(state.count)"`,
			cursor:       10,
			expectedName: "string",
		},
		{
			name:         "int coercion",
			line:         `{{ int(state.floatVal) }}`,
			cursor:       5,
			expectedName: "int",
		},
		{
			name:         "int64 coercion",
			line:         `{{ int64(value) }}`,
			cursor:       5,
			expectedName: "int64",
		},
		{
			name:         "float coercion",
			line:         `{{ float(state.intVal) }}`,
			cursor:       5,
			expectedName: "float",
		},
		{
			name:         "float64 coercion",
			line:         `{{ float64(value) }}`,
			cursor:       6,
			expectedName: "float64",
		},
		{
			name:         "bool coercion",
			line:         `p-if="bool(state.enabled)"`,
			cursor:       8,
			expectedName: "bool",
		},
		{
			name:         "decimal coercion",
			line:         `{{ decimal(state.price) }}`,
			cursor:       6,
			expectedName: "decimal",
		},
		{
			name:         "bigint coercion",
			line:         `{{ bigint(state.largeId) }}`,
			cursor:       6,
			expectedName: "bigint",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := &document{}
			position := protocol.Position{Line: 0, Character: uint32(tc.cursor)}

			ctx := d.checkBuiltinHoverContext(tc.line, tc.cursor, position)

			if ctx == nil {
				t.Fatalf("expected context, got nil")
			}
			if ctx.Name != tc.expectedName {
				t.Errorf("expected name %q, got %q", tc.expectedName, ctx.Name)
			}
		})
	}
}

func TestCheckBuiltinHoverContext_NoMatch(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		line   string
		cursor int
	}{
		{
			name:   "cursor on arguments not function name",
			line:   `p-if="len(state.items) > 0"`,
			cursor: 15,
		},
		{
			name:   "variable named len (no parenthesis)",
			line:   `p-text="len"`,
			cursor: 9,
		},
		{
			name:   "method call not builtin",
			line:   `{{ state.len() }}`,
			cursor: 11,
		},
		{
			name:   "unknown function",
			line:   `{{ customFunc(x) }}`,
			cursor: 6,
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

			ctx := d.checkBuiltinHoverContext(tc.line, tc.cursor, position)

			if ctx != nil {
				t.Errorf("expected nil context, got %+v", ctx)
			}
		})
	}
}

func TestGetBuiltinHover(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name               string
		builtinName        string
		expectHeader       string
		expectSig          string
		expectAccepts      string
		expectReturns      string
		expectDocumentsURL string
		expectExample      bool
	}{
		{
			name:               "len hover",
			builtinName:        "len",
			expectHeader:       "## `len`",
			expectSig:          "len(x) int",
			expectAccepts:      "Array, slice, map, or string",
			expectReturns:      "int",
			expectExample:      true,
			expectDocumentsURL: "/docs/api/builtins/len",
		},
		{
			name:               "T hover",
			builtinName:        "T",
			expectHeader:       "## `T`",
			expectSig:          "T(key string, fallback ...string) string",
			expectAccepts:      "Translation key",
			expectReturns:      "string",
			expectExample:      true,
			expectDocumentsURL: "/docs/api/builtins/T",
		},
		{
			name:               "string hover",
			builtinName:        "string",
			expectHeader:       "## `string`",
			expectSig:          "string(x) string",
			expectAccepts:      "int, uint, float, bool",
			expectReturns:      "string",
			expectExample:      true,
			expectDocumentsURL: "/docs/api/builtins/string",
		},
		{
			name:               "decimal hover",
			builtinName:        "decimal",
			expectHeader:       "## `decimal`",
			expectSig:          "decimal(x) maths.Decimal",
			expectAccepts:      "int, uint, byte, rune, string",
			expectReturns:      "maths.Decimal",
			expectExample:      true,
			expectDocumentsURL: "/docs/api/builtins/decimal",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := &document{}
			ctx := &PKHoverContext{
				Kind: PKDefBuiltinFunction,
				Name: tc.builtinName,
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 0},
					End:   protocol.Position{Line: 0, Character: uint32(len(tc.builtinName))},
				},
			}

			hover, err := d.getBuiltinHover(ctx)

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
			if !strings.Contains(content, tc.expectSig) {
				t.Errorf("expected signature %q in content:\n%s", tc.expectSig, content)
			}
			if !strings.Contains(content, tc.expectAccepts) {
				t.Errorf("expected accepts %q in content:\n%s", tc.expectAccepts, content)
			}
			if !strings.Contains(content, tc.expectReturns) {
				t.Errorf("expected returns %q in content:\n%s", tc.expectReturns, content)
			}
			if tc.expectExample && !strings.Contains(content, "**Example:**") {
				t.Errorf("expected example in content:\n%s", content)
			}
			if tc.expectDocumentsURL != "" && !strings.Contains(content, tc.expectDocumentsURL) {
				t.Errorf("expected docs URL %q in content:\n%s", tc.expectDocumentsURL, content)
			}
		})
	}
}

func TestGetBuiltinHover_UnknownBuiltin(t *testing.T) {
	t.Parallel()

	d := &document{}
	ctx := &PKHoverContext{
		Kind: PKDefBuiltinFunction,
		Name: "nonexistent",
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 11},
		},
	}

	hover, err := d.getBuiltinHover(ctx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hover != nil {
		t.Errorf("expected nil hover for unknown builtin, got %+v", hover)
	}
}

func TestFindBuiltinAtCursor(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		line         string
		expectedName string
		cursor       int
		expectedEnd  int
	}{
		{
			name:         "len at start",
			line:         "len(x)",
			cursor:       1,
			expectedName: "len",
			expectedEnd:  3,
		},
		{
			name:         "T with space before paren",
			line:         "T ('key')",
			cursor:       0,
			expectedName: "T",
			expectedEnd:  1,
		},
		{
			name:         "append in expression",
			line:         "result = append(slice, item)",
			cursor:       12,
			expectedName: "append",
			expectedEnd:  15,
		},
		{
			name:         "no builtin - variable",
			line:         "myLen = 5",
			cursor:       3,
			expectedName: "",
			expectedEnd:  0,
		},
		{
			name:         "no builtin - method on object",
			line:         "obj.len()",
			cursor:       5,
			expectedName: "",
			expectedEnd:  0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			name, _, end := findBuiltinAtCursor(tc.line, tc.cursor)

			if name != tc.expectedName {
				t.Errorf("expected name %q, got %q", tc.expectedName, name)
			}
			if tc.expectedName != "" && end != tc.expectedEnd {
				t.Errorf("expected end %d, got %d", tc.expectedEnd, end)
			}
		})
	}
}

func TestPikoBuiltinDocuments_AllBuiltinsHaveRequiredFields(t *testing.T) {
	t.Parallel()

	for name, document := range pikoBuiltinDocumentations {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if document.Name == "" {
				t.Errorf("builtin %q has empty Name", name)
			}
			if document.Name != name {
				t.Errorf("builtin %q has mismatched Name %q", name, document.Name)
			}
			if document.Signature == "" {
				t.Errorf("builtin %q has empty Signature", name)
			}
			if document.Description == "" {
				t.Errorf("builtin %q has empty Description", name)
			}
			if document.Example == "" {
				t.Errorf("builtin %q has empty Example", name)
			}
			if document.DocumentsURL == "" {
				t.Errorf("builtin %q has empty DocumentsURL", name)
			}
		})
	}
}

func TestIsIdentifierChar(t *testing.T) {
	t.Parallel()

	validChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"
	invalidChars := " \t\n()[]{}.,;:+-*/<>="

	for _, c := range validChars {
		if !isIdentifierChar(byte(c)) {
			t.Errorf("expected %q to be valid identifier char", string(c))
		}
	}

	for _, c := range invalidChars {
		if isIdentifierChar(byte(c)) {
			t.Errorf("expected %q to be invalid identifier char", string(c))
		}
	}
}
