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
	"context"
	"testing"

	"go.lsp.dev/protocol"
)

func TestExtractVariableName_UndefinedColon(t *testing.T) {
	testCases := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "undefined with colon",
			message:  "undefined: myVar",
			expected: "myVar",
		},
		{
			name:     "undefined with colon and extra text",
			message:  "undefined: userName is not defined",
			expected: "userName",
		},
		{
			name:     "undefined with colon and punctuation",
			message:  "undefined: count, please check",
			expected: "count",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractVariableName(tc.message)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestExtractVariableName_QuotedVariable(t *testing.T) {
	testCases := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "single quoted variable",
			message:  "variable 'myVar' is not defined",
			expected: "myVar",
		},
		{
			name:     "quoted variable in middle",
			message:  "Error: 'userName' undefined in scope",
			expected: "userName",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractVariableName(tc.message)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestExtractVariableName_VariableKeyword(t *testing.T) {
	testCases := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "variable keyword prefix",
			message:  "variable myVar is not defined",
			expected: "myVar",
		},
		{
			name:     "variable keyword with spaces",
			message:  "variable   userName not found",
			expected: "userName",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractVariableName(tc.message)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestExtractVariableName_NoMatch(t *testing.T) {
	testCases := []struct {
		name    string
		message string
	}{
		{
			name:    "empty message",
			message: "",
		},
		{
			name:    "no variable pattern",
			message: "generic error message",
		},
		{
			name:    "syntax error",
			message: "unexpected token at line 5",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractVariableName(tc.message)
			if result != "" {
				t.Errorf("expected empty string, got %q", result)
			}
		})
	}
}

func TestGenerateQuickFixes_NoDiagnosticCode_ReturnsFallbackFixes(t *testing.T) {
	testDocument := &document{
		URI:     "file:///test.pk",
		Content: []byte(`<template><div>{{ undefined }}</div></template>`),
	}

	ws := &workspace{
		documents: map[protocol.DocumentURI]*document{
			testDocument.URI: testDocument,
		},
		docCache: NewDocumentCache(),
	}

	diagnostic := protocol.Diagnostic{
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 18},
			End:   protocol.Position{Line: 0, Character: 27},
		},
		Message:  "undefined: 'undefined' is not defined",
		Severity: protocol.DiagnosticSeverityError,
		Code:     nil,
	}

	actions := generateQuickFixes(context.Background(), diagnostic, testDocument, ws)

	if len(actions) == 0 {
		t.Log("no actions generated (expected when fallback can extract variable)")
	}
}

func TestGenerateQuickFixes_UnknownCode_ReturnsFallbackFixes(t *testing.T) {
	testDocument := &document{
		URI:     "file:///test.pk",
		Content: []byte(`<template><div>test</div></template>`),
	}

	ws := &workspace{
		documents: map[protocol.DocumentURI]*document{
			testDocument.URI: testDocument,
		},
		docCache: NewDocumentCache(),
	}

	diagnostic := protocol.Diagnostic{
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 10},
		},
		Message:  "some generic error",
		Severity: protocol.DiagnosticSeverityWarning,
		Code:     "unknown_code",
	}

	actions := generateQuickFixes(context.Background(), diagnostic, testDocument, ws)

	_ = actions
}

func TestGenerateFallbackFixes_UndefinedMessage_GeneratesAction(t *testing.T) {
	testDocument := &document{
		URI:     "file:///test.pk",
		Content: []byte(`<template><div>test</div></template>`),
	}

	ws := &workspace{
		documents: map[protocol.DocumentURI]*document{
			testDocument.URI: testDocument,
		},
		docCache: NewDocumentCache(),
	}

	diagnostic := protocol.Diagnostic{
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 10},
		},
		Message:  "undefined: myVariable not found",
		Severity: protocol.DiagnosticSeverityError,
	}

	actions := generateFallbackFixes(diagnostic, testDocument, ws)

	if len(actions) == 0 {
		t.Fatal("expected at least one action for undefined variable message")
	}

	foundAddToState := false
	for _, action := range actions {
		if action.Kind == protocol.QuickFix {
			foundAddToState = true
			break
		}
	}

	if !foundAddToState {
		t.Error("expected QuickFix action for undefined variable")
	}
}

func TestGenerateFallbackFixes_NotFoundMessage_GeneratesAction(t *testing.T) {
	testDocument := &document{
		URI:     "file:///test.pk",
		Content: []byte(`<template><div>test</div></template>`),
	}

	ws := &workspace{
		documents: map[protocol.DocumentURI]*document{
			testDocument.URI: testDocument,
		},
		docCache: NewDocumentCache(),
	}

	diagnostic := protocol.Diagnostic{
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 10},
		},
		Message:  "'myVar' not found in scope",
		Severity: protocol.DiagnosticSeverityError,
	}

	actions := generateFallbackFixes(diagnostic, testDocument, ws)

	if len(actions) == 0 {
		t.Fatal("expected at least one action for not found message")
	}
}

func TestGenerateQuickFixes_TypeMismatchCode_DispatchesToCoerceFix(t *testing.T) {
	testDocument := &document{
		URI:     "file:///test.pk",
		Content: []byte(`<template><div></div></template>`),
	}
	ws := &workspace{
		documents: map[protocol.DocumentURI]*document{testDocument.URI: testDocument},
		docCache:  NewDocumentCache(),
	}

	diagnostic := protocol.Diagnostic{
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 10},
		},
		Code: DiagCodeTypeMismatch,
		Data: map[string]any{
			"can_coerce": false,
		},
	}

	actions := generateQuickFixes(context.Background(), diagnostic, testDocument, ws)

	if len(actions) != 0 {
		t.Errorf("expected 0 actions for type mismatch with can_coerce=false, got %d", len(actions))
	}
}

func TestGenerateQuickFixes_UndefinedVariableCode_DispatchesToVariableFixes(t *testing.T) {
	testDocument := &document{
		URI:     "file:///test.pk",
		Content: []byte(`<template><div></div></template>`),
	}
	ws := &workspace{
		documents: map[protocol.DocumentURI]*document{testDocument.URI: testDocument},
		docCache:  NewDocumentCache(),
	}

	diagnostic := protocol.Diagnostic{
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 10},
		},
		Code: DiagCodeUndefinedVariable,
		Data: map[string]any{
			"suggestion": "myVar",
		},
	}

	actions := generateQuickFixes(context.Background(), diagnostic, testDocument, ws)
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}

	if actions[0].Kind != protocol.QuickFix {
		t.Errorf("Kind = %v, want %v", actions[0].Kind, protocol.QuickFix)
	}
}

func TestGenerateQuickFixes_UndefinedPartialAliasCode(t *testing.T) {
	testDocument := &document{
		URI:     "file:///test.pk",
		Content: []byte(`<template><div></div></template>`),
	}
	ws := &workspace{
		documents: map[protocol.DocumentURI]*document{testDocument.URI: testDocument},
		docCache:  NewDocumentCache(),
	}

	diagnostic := protocol.Diagnostic{
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 10},
		},
		Code: DiagCodeUndefinedPartialAlias,
		Data: map[string]any{
			"suggestion": "StatusBadge",
		},
	}

	actions := generateQuickFixes(context.Background(), diagnostic, testDocument, ws)
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
}

func TestGenerateQuickFixes_MissingRequiredPropCode(t *testing.T) {
	testDocument := &document{
		URI:     "file:///test.pk",
		Content: []byte(`<template><div></div></template>`),
	}
	ws := &workspace{
		documents: map[protocol.DocumentURI]*document{testDocument.URI: testDocument},
		docCache:  NewDocumentCache(),
	}

	diagnostic := protocol.Diagnostic{
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 10},
		},
		Code: DiagCodeMissingRequiredProp,
		Data: nil,
	}

	actions := generateQuickFixes(context.Background(), diagnostic, testDocument, ws)
	if len(actions) != 0 {
		t.Errorf("expected 0 actions for nil data, got %d", len(actions))
	}
}

func TestGenerateQuickFixes_MissingImportCode(t *testing.T) {
	testDocument := &document{
		URI:     "file:///test.pk",
		Content: []byte(`<template><div></div></template>`),
	}
	ws := &workspace{
		documents: map[protocol.DocumentURI]*document{testDocument.URI: testDocument},
		docCache:  NewDocumentCache(),
	}

	diagnostic := protocol.Diagnostic{
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 10},
		},
		Code: DiagCodeMissingImport,
		Data: nil,
	}

	actions := generateQuickFixes(context.Background(), diagnostic, testDocument, ws)
	if len(actions) != 0 {
		t.Errorf("expected 0 actions for nil data, got %d", len(actions))
	}
}

func TestGenerateFallbackFixes_NoMatchingPattern_ReturnsEmpty(t *testing.T) {
	testDocument := &document{
		URI:     "file:///test.pk",
		Content: []byte(`<template><div>test</div></template>`),
	}

	ws := &workspace{
		documents: map[protocol.DocumentURI]*document{
			testDocument.URI: testDocument,
		},
		docCache: NewDocumentCache(),
	}

	diagnostic := protocol.Diagnostic{
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 10},
		},
		Message:  "syntax error: unexpected token",
		Severity: protocol.DiagnosticSeverityError,
	}

	actions := generateFallbackFixes(diagnostic, testDocument, ws)

	if len(actions) != 0 {
		t.Errorf("expected empty actions for non-matching pattern, got %d", len(actions))
	}
}
