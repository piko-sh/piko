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
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestConvertSeverityToLSP(t *testing.T) {
	testCases := []struct {
		name     string
		severity ast_domain.Severity
		want     protocol.DiagnosticSeverity
	}{
		{
			name:     "error maps to LSP error",
			severity: ast_domain.Error,
			want:     protocol.DiagnosticSeverityError,
		},
		{
			name:     "info maps to LSP information",
			severity: ast_domain.Info,
			want:     protocol.DiagnosticSeverityInformation,
		},
		{
			name:     "warning maps to LSP warning",
			severity: ast_domain.Warning,
			want:     protocol.DiagnosticSeverityWarning,
		},
		{
			name:     "unknown severity defaults to LSP warning",
			severity: ast_domain.Severity(99),
			want:     protocol.DiagnosticSeverityWarning,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := convertSeverityToLSP(tc.severity)
			if got != tc.want {
				t.Errorf("convertSeverityToLSP(%v) = %v, want %v", tc.severity, got, tc.want)
			}
		})
	}
}

func TestCalculateEndChar(t *testing.T) {
	testCases := []struct {
		diagnostic *ast_domain.Diagnostic
		name       string
		want       int
	}{
		{
			name: "expression with length adds to column",
			diagnostic: &ast_domain.Diagnostic{
				Location: ast_domain.Location{
					Line:   1,
					Column: 5,
				},
				Expression: "hello",
			},
			want: 9,
		},
		{
			name: "empty expression adds one",
			diagnostic: &ast_domain.Diagnostic{
				Location: ast_domain.Location{
					Line:   1,
					Column: 5,
				},
				Expression: "",
			},
			want: 5,
		},
		{
			name: "zero column with expression",
			diagnostic: &ast_domain.Diagnostic{
				Location: ast_domain.Location{
					Line:   1,
					Column: 0,
				},
				Expression: "abc",
			},
			want: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := calculateEndChar(tc.diagnostic)
			if got != tc.want {
				t.Errorf("calculateEndChar() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestIsDuplicateDiagnostic(t *testing.T) {
	testCases := []struct {
		setupSeen  map[string]struct{}
		diagnostic *ast_domain.Diagnostic
		name       string
		want       bool
	}{
		{
			name:      "first occurrence returns false",
			setupSeen: make(map[string]struct{}),
			diagnostic: &ast_domain.Diagnostic{
				Message:  "undefined variable",
				Location: ast_domain.Location{Line: 10, Column: 5},
				Severity: ast_domain.Error,
			},
			want: false,
		},
		{
			name: "second occurrence returns true",
			setupSeen: map[string]struct{}{
				"undefined variable:10:5:3": {},
			},
			diagnostic: &ast_domain.Diagnostic{
				Message:  "undefined variable",
				Location: ast_domain.Location{Line: 10, Column: 5},
				Severity: ast_domain.Error,
			},
			want: true,
		},
		{
			name: "different message is not a duplicate",
			setupSeen: map[string]struct{}{
				"undefined variable:10:5:3": {},
			},
			diagnostic: &ast_domain.Diagnostic{
				Message:  "type mismatch",
				Location: ast_domain.Location{Line: 10, Column: 5},
				Severity: ast_domain.Error,
			},
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := isDuplicateDiagnostic(tc.diagnostic, tc.setupSeen)
			if got != tc.want {
				t.Errorf("isDuplicateDiagnostic() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestShouldIncludeDiagnostic(t *testing.T) {
	testCases := []struct {
		name       string
		diagnostic *ast_domain.Diagnostic
		absPath    string
		want       bool
	}{
		{
			name: "matching path returns true",
			diagnostic: &ast_domain.Diagnostic{
				SourcePath: "/home/user/project/main.pk",
				Message:    "some error",
			},
			absPath: "/home/user/project/main.pk",
			want:    true,
		},
		{
			name: "non-matching path returns false",
			diagnostic: &ast_domain.Diagnostic{
				SourcePath: "/home/user/project/other.pk",
				Message:    "some error",
			},
			absPath: "/home/user/project/main.pk",
			want:    false,
		},
		{
			name: "empty path matches empty source path",
			diagnostic: &ast_domain.Diagnostic{
				SourcePath: "",
				Message:    "some error",
			},
			absPath: "",
			want:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldIncludeDiagnostic(context.Background(), tc.diagnostic, tc.absPath)
			if got != tc.want {
				t.Errorf("shouldIncludeDiagnostic() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestBuildLSPDiagnostic(t *testing.T) {
	testCases := []struct {
		name       string
		diagnostic *ast_domain.Diagnostic
		wantCode   any
		wantSource string
		wantStartL uint32
		wantStartC uint32
		wantEndL   uint32
		wantEndC   uint32
	}{
		{
			name: "with code populates code field",
			diagnostic: &ast_domain.Diagnostic{
				Message:    "type mismatch",
				Expression: "myVar",
				Code:       "E001",
				SourcePath: "/test.pk",
				Severity:   ast_domain.Error,
				Location:   ast_domain.Location{Line: 10, Column: 5},
			},
			wantCode:   "E001",
			wantSource: "piko",
			wantStartL: 9,
			wantStartC: 4,
			wantEndL:   9,
			wantEndC:   9,
		},
		{
			name: "without code leaves code nil",
			diagnostic: &ast_domain.Diagnostic{
				Message:    "warning message",
				Expression: "expr",
				Code:       "",
				SourcePath: "/test.pk",
				Severity:   ast_domain.Warning,
				Location:   ast_domain.Location{Line: 3, Column: 1},
			},
			wantCode:   nil,
			wantSource: "piko",
			wantStartL: 2,
			wantStartC: 0,
			wantEndL:   2,
			wantEndC:   4,
		},
		{
			name: "with expression sets end character by expression length",
			diagnostic: &ast_domain.Diagnostic{
				Message:    "info diagnostic",
				Expression: "longExpression",
				Code:       "",
				SourcePath: "/test.pk",
				Severity:   ast_domain.Info,
				Location:   ast_domain.Location{Line: 1, Column: 1},
			},
			wantCode:   nil,
			wantSource: "piko",
			wantStartL: 0,
			wantStartC: 0,
			wantEndL:   0,
			wantEndC:   14,
		},
		{
			name: "empty expression adds one to end character",
			diagnostic: &ast_domain.Diagnostic{
				Message:    "empty expression diagnostic",
				Expression: "",
				Code:       "W002",
				SourcePath: "/test.pk",
				Severity:   ast_domain.Warning,
				Location:   ast_domain.Location{Line: 7, Column: 3},
			},
			wantCode:   "W002",
			wantSource: "piko",
			wantStartL: 6,
			wantStartC: 2,
			wantEndL:   6,
			wantEndC:   3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildLSPDiagnostic(tc.diagnostic)

			if got.Range.Start.Line != tc.wantStartL {
				t.Errorf("Start.Line = %d, want %d", got.Range.Start.Line, tc.wantStartL)
			}
			if got.Range.Start.Character != tc.wantStartC {
				t.Errorf("Start.Character = %d, want %d", got.Range.Start.Character, tc.wantStartC)
			}
			if got.Range.End.Line != tc.wantEndL {
				t.Errorf("End.Line = %d, want %d", got.Range.End.Line, tc.wantEndL)
			}
			if got.Range.End.Character != tc.wantEndC {
				t.Errorf("End.Character = %d, want %d", got.Range.End.Character, tc.wantEndC)
			}
			if got.Source != tc.wantSource {
				t.Errorf("Source = %q, want %q", got.Source, tc.wantSource)
			}
			if got.Message != tc.diagnostic.Message {
				t.Errorf("Message = %q, want %q", got.Message, tc.diagnostic.Message)
			}
			if got.Code != tc.wantCode {
				t.Errorf("Code = %v, want %v", got.Code, tc.wantCode)
			}
		})
	}
}

func TestConvertRelatedInfoToLSP(t *testing.T) {
	testCases := []struct {
		diagnostic *ast_domain.Diagnostic
		name       string
		wantLen    int
		wantNil    bool
	}{
		{
			name: "empty related info returns nil",
			diagnostic: &ast_domain.Diagnostic{
				SourcePath:  "/test.pk",
				RelatedInfo: nil,
			},
			wantNil: true,
			wantLen: 0,
		},
		{
			name: "single related info",
			diagnostic: &ast_domain.Diagnostic{
				SourcePath: "/test.pk",
				RelatedInfo: []ast_domain.DiagnosticRelatedInfo{
					{
						Message:  "declared here",
						Location: ast_domain.Location{Line: 5, Column: 3},
					},
				},
			},
			wantNil: false,
			wantLen: 1,
		},
		{
			name: "multiple related info entries",
			diagnostic: &ast_domain.Diagnostic{
				SourcePath: "/test.pk",
				RelatedInfo: []ast_domain.DiagnosticRelatedInfo{
					{
						Message:  "first usage",
						Location: ast_domain.Location{Line: 2, Column: 1},
					},
					{
						Message:  "second usage",
						Location: ast_domain.Location{Line: 8, Column: 10},
					},
					{
						Message:  "declared here",
						Location: ast_domain.Location{Line: 1, Column: 1},
					},
				},
			},
			wantNil: false,
			wantLen: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := convertRelatedInfoToLSP(tc.diagnostic)

			if tc.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %v", got)
				}
				return
			}

			if got == nil {
				t.Fatal("expected non-nil result")
			}

			if len(got) != tc.wantLen {
				t.Fatalf("len(result) = %d, want %d", len(got), tc.wantLen)
			}

			for i, ri := range got {
				expectedURI := protocol.DocumentURI("file://" + tc.diagnostic.SourcePath)
				if ri.Location.URI != expectedURI {
					t.Errorf("entry[%d].Location.URI = %q, want %q", i, ri.Location.URI, expectedURI)
				}

				srcRI := tc.diagnostic.RelatedInfo[i]
				wantStartLine := uint32(srcRI.Location.Line - 1)
				wantStartChar := uint32(srcRI.Location.Column - 1)
				wantEndLine := uint32(srcRI.Location.Line - 1)
				wantEndChar := uint32(srcRI.Location.Column)

				if ri.Location.Range.Start.Line != wantStartLine {
					t.Errorf("entry[%d].Start.Line = %d, want %d", i, ri.Location.Range.Start.Line, wantStartLine)
				}
				if ri.Location.Range.Start.Character != wantStartChar {
					t.Errorf("entry[%d].Start.Character = %d, want %d", i, ri.Location.Range.Start.Character, wantStartChar)
				}
				if ri.Location.Range.End.Line != wantEndLine {
					t.Errorf("entry[%d].End.Line = %d, want %d", i, ri.Location.Range.End.Line, wantEndLine)
				}
				if ri.Location.Range.End.Character != wantEndChar {
					t.Errorf("entry[%d].End.Character = %d, want %d", i, ri.Location.Range.End.Character, wantEndChar)
				}
				if ri.Message != srcRI.Message {
					t.Errorf("entry[%d].Message = %q, want %q", i, ri.Message, srcRI.Message)
				}
			}
		})
	}
}

func TestConvertDiagnosticsToLSP(t *testing.T) {
	testCases := []struct {
		validate func(t *testing.T, result []protocol.Diagnostic)
		name     string
		absPath  string
		input    []*ast_domain.Diagnostic
		wantLen  int
	}{
		{
			name:    "empty input returns empty slice",
			input:   []*ast_domain.Diagnostic{},
			absPath: "/test.pk",
			wantLen: 0,
			validate: func(t *testing.T, result []protocol.Diagnostic) {
				if result == nil {
					t.Error("expected non-nil empty slice")
				}
			},
		},
		{
			name: "single matching diagnostic",
			input: []*ast_domain.Diagnostic{
				{
					Message:    "undefined variable",
					Expression: "x",
					SourcePath: "/test.pk",
					Severity:   ast_domain.Error,
					Location:   ast_domain.Location{Line: 1, Column: 1},
				},
			},
			absPath: "/test.pk",
			wantLen: 1,
			validate: func(t *testing.T, result []protocol.Diagnostic) {
				if result[0].Message != "undefined variable" {
					t.Errorf("Message = %q, want %q", result[0].Message, "undefined variable")
				}
				if result[0].Source != "piko" {
					t.Errorf("Source = %q, want %q", result[0].Source, "piko")
				}
			},
		},
		{
			name: "filters by path",
			input: []*ast_domain.Diagnostic{
				{
					Message:    "error in target file",
					Expression: "a",
					SourcePath: "/test.pk",
					Severity:   ast_domain.Error,
					Location:   ast_domain.Location{Line: 1, Column: 1},
				},
				{
					Message:    "error in other file",
					Expression: "b",
					SourcePath: "/other.pk",
					Severity:   ast_domain.Error,
					Location:   ast_domain.Location{Line: 2, Column: 1},
				},
				{
					Message:    "another error in target file",
					Expression: "c",
					SourcePath: "/test.pk",
					Severity:   ast_domain.Warning,
					Location:   ast_domain.Location{Line: 3, Column: 1},
				},
			},
			absPath: "/test.pk",
			wantLen: 2,
			validate: func(t *testing.T, result []protocol.Diagnostic) {
				if result[0].Message != "error in target file" {
					t.Errorf("result[0].Message = %q, want %q", result[0].Message, "error in target file")
				}
				if result[1].Message != "another error in target file" {
					t.Errorf("result[1].Message = %q, want %q", result[1].Message, "another error in target file")
				}
			},
		},
		{
			name: "deduplicates identical diagnostics",
			input: []*ast_domain.Diagnostic{
				{
					Message:    "duplicate error",
					Expression: "x",
					SourcePath: "/test.pk",
					Severity:   ast_domain.Error,
					Location:   ast_domain.Location{Line: 5, Column: 10},
				},
				{
					Message:    "duplicate error",
					Expression: "x",
					SourcePath: "/test.pk",
					Severity:   ast_domain.Error,
					Location:   ast_domain.Location{Line: 5, Column: 10},
				},
			},
			absPath: "/test.pk",
			wantLen: 1,
			validate: func(t *testing.T, result []protocol.Diagnostic) {
				if result[0].Message != "duplicate error" {
					t.Errorf("Message = %q, want %q", result[0].Message, "duplicate error")
				}
			},
		},
		{
			name: "mixed filtering and deduplication",
			input: []*ast_domain.Diagnostic{
				{
					Message:    "first error",
					Expression: "a",
					SourcePath: "/test.pk",
					Severity:   ast_domain.Error,
					Location:   ast_domain.Location{Line: 1, Column: 1},
				},
				{
					Message:    "wrong file",
					Expression: "b",
					SourcePath: "/other.pk",
					Severity:   ast_domain.Error,
					Location:   ast_domain.Location{Line: 2, Column: 1},
				},
				{
					Message:    "first error",
					Expression: "a",
					SourcePath: "/test.pk",
					Severity:   ast_domain.Error,
					Location:   ast_domain.Location{Line: 1, Column: 1},
				},
				{
					Message:    "second error",
					Expression: "c",
					SourcePath: "/test.pk",
					Severity:   ast_domain.Warning,
					Location:   ast_domain.Location{Line: 3, Column: 5},
				},
				{
					Message:    "also wrong file",
					Expression: "d",
					SourcePath: "/another.pk",
					Severity:   ast_domain.Info,
					Location:   ast_domain.Location{Line: 4, Column: 1},
				},
			},
			absPath: "/test.pk",
			wantLen: 2,
			validate: func(t *testing.T, result []protocol.Diagnostic) {
				if result[0].Message != "first error" {
					t.Errorf("result[0].Message = %q, want %q", result[0].Message, "first error")
				}
				if result[0].Severity != protocol.DiagnosticSeverityError {
					t.Errorf("result[0].Severity = %v, want %v", result[0].Severity, protocol.DiagnosticSeverityError)
				}
				if result[1].Message != "second error" {
					t.Errorf("result[1].Message = %q, want %q", result[1].Message, "second error")
				}
				if result[1].Severity != protocol.DiagnosticSeverityWarning {
					t.Errorf("result[1].Severity = %v, want %v", result[1].Severity, protocol.DiagnosticSeverityWarning)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := convertDiagnosticsToLSP(context.Background(), tc.input, tc.absPath)

			if len(got) != tc.wantLen {
				t.Fatalf("len(result) = %d, want %d", len(got), tc.wantLen)
			}

			if tc.validate != nil {
				tc.validate(t, got)
			}
		})
	}
}
