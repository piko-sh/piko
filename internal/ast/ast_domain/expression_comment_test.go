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

package ast_domain

import (
	"context"
	"testing"
)

func TestLexComment_BasicCommentHandling(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		input          string
		wantTokenTypes []tokenType
		wantValues     []string
	}{
		{
			name:           "comment before identifier",
			input:          "/* comment */ a",
			wantTokenTypes: []tokenType{tokenIdent, tokenEOF},
			wantValues:     []string{"a", ""},
		},
		{
			name:           "comment between operators",
			input:          "a /* comment */ + b",
			wantTokenTypes: []tokenType{tokenIdent, tokenSymbol, tokenIdent, tokenEOF},
			wantValues:     []string{"a", "+", "b", ""},
		},
		{
			name:           "multiple comments",
			input:          "/* c1 */ a /* c2 */ + /* c3 */ b",
			wantTokenTypes: []tokenType{tokenIdent, tokenSymbol, tokenIdent, tokenEOF},
			wantValues:     []string{"a", "+", "b", ""},
		},
		{
			name:           "empty comment",
			input:          "/**/",
			wantTokenTypes: []tokenType{tokenEOF},
			wantValues:     []string{""},
		},
		{
			name:           "comment with asterisks",
			input:          "/* * ** *** */",
			wantTokenTypes: []tokenType{tokenEOF},
			wantValues:     []string{""},
		},
		{
			name:           "comment at end",
			input:          "a /* comment */",
			wantTokenTypes: []tokenType{tokenIdent, tokenEOF},
			wantValues:     []string{"a", ""},
		},
		{
			name:           "comment only",
			input:          "/* just a comment */",
			wantTokenTypes: []tokenType{tokenEOF},
			wantValues:     []string{""},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tokens := lexInto(context.Background(), tc.input, nil)

			if len(tokens) != len(tc.wantTokenTypes) {
				t.Fatalf("token count mismatch: got %d, want %d", len(tokens), len(tc.wantTokenTypes))
			}

			for i, tok := range tokens {
				if tok.Type != tc.wantTokenTypes[i] {
					t.Errorf("token[%d] type: got %v, want %v", i, tok.Type, tc.wantTokenTypes[i])
				}
				gotVal := tok.getValue(tc.input)
				if gotVal != tc.wantValues[i] {
					t.Errorf("token[%d] value: got %q, want %q", i, gotVal, tc.wantValues[i])
				}
			}
		})
	}
}

func TestLexComment_MultiLineTracking(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		wantLine int
		wantCol  int
	}{
		{
			name:     "single line comment",
			input:    "/* comment */ a",
			wantLine: 1,
			wantCol:  15,
		},
		{
			name:     "multi-line comment single newline",
			input:    "/* line1\nline2 */ a",
			wantLine: 2,
			wantCol:  10,
		},
		{
			name:     "newline then comment then identifier",
			input:    "\n/* comment\n*/ b",
			wantLine: 3,
			wantCol:  4,
		},
		{
			name:     "multiple newlines in comment",
			input:    "/*\n\n\n*/ a",
			wantLine: 4,
			wantCol:  4,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tokens := lexInto(context.Background(), tc.input, nil)

			var targetTok lexerToken
			for _, tok := range tokens {
				if tok.Type == tokenIdent {
					targetTok = tok
					break
				}
			}

			if targetTok.Type != tokenIdent {
				t.Fatalf("expected to find an identifier token")
			}
			if targetTok.Location.Line != tc.wantLine {
				t.Errorf("line: got %d, want %d", targetTok.Location.Line, tc.wantLine)
			}
			if targetTok.Location.Column != tc.wantCol {
				t.Errorf("column: got %d, want %d", targetTok.Location.Column, tc.wantCol)
			}
		})
	}
}

func TestLexComment_ErrorCases(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		input     string
		wantError string
	}{
		{
			name:      "unterminated comment at end",
			input:     "/* never closed",
			wantError: "unterminated multi-line comment",
		},
		{
			name:      "unterminated after partial parse",
			input:     "a + /* never closed",
			wantError: "unterminated multi-line comment",
		},
		{
			name:      "just comment start",
			input:     "/*",
			wantError: "unterminated multi-line comment",
		},
		{
			name:      "stray close at start",
			input:     "*/",
			wantError: "unexpected '*/' without matching '/*'",
		},
		{
			name:      "stray close in expression",
			input:     "a + */ b",
			wantError: "unexpected '*/' without matching '/*'",
		},
		{
			name:      "comment start with single char remaining",
			input:     "/*x",
			wantError: "unterminated multi-line comment",
		},
		{
			name:      "nested comment attempt leaves stray close",
			input:     "/* outer /* inner */ still */ oops",
			wantError: "unexpected '*/' without matching '/*'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tokens := lexInto(context.Background(), tc.input, nil)

			var foundError bool
			var gotError string
			for _, tok := range tokens {
				if tok.Type == tokenError {
					foundError = true
					gotError = tok.errorMessage
					break
				}
			}

			if !foundError {
				t.Fatalf("expected error token, got none")
			}
			if gotError != tc.wantError {
				t.Errorf("error message: got %q, want %q", gotError, tc.wantError)
			}
		})
	}
}

func TestLexComment_NoFalsePositives(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		input          string
		wantTokenTypes []tokenType
		wantValues     []string
	}{
		{
			name:           "division operator",
			input:          "a / b",
			wantTokenTypes: []tokenType{tokenIdent, tokenSymbol, tokenIdent, tokenEOF},
			wantValues:     []string{"a", "/", "b", ""},
		},
		{
			name:           "multiplication operator",
			input:          "a * b",
			wantTokenTypes: []tokenType{tokenIdent, tokenSymbol, tokenIdent, tokenEOF},
			wantValues:     []string{"a", "*", "b", ""},
		},
		{
			name:           "comment-like string double quotes",
			input:          `"/* not a comment */"`,
			wantTokenTypes: []tokenType{tokenString, tokenEOF},
			wantValues:     []string{`"/* not a comment */"`, ""},
		},
		{
			name:           "comment-like string single quotes",
			input:          `'*/'`,
			wantTokenTypes: []tokenType{tokenString, tokenEOF},
			wantValues:     []string{`'*/'`, ""},
		},
		{
			name:           "space prevents comment",
			input:          "a / * b",
			wantTokenTypes: []tokenType{tokenIdent, tokenSymbol, tokenSymbol, tokenIdent, tokenEOF},
			wantValues:     []string{"a", "/", "*", "b", ""},
		},
		{
			name:           "nested comments not supported",
			input:          "/* outer /* inner */ still",
			wantTokenTypes: []tokenType{tokenIdent, tokenEOF},
			wantValues:     []string{"still", ""},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tokens := lexInto(context.Background(), tc.input, nil)

			for i, tok := range tokens {
				if tok.Type == tokenError {
					t.Fatalf("unexpected error at token[%d]: %s", i, tok.errorMessage)
				}
			}

			if len(tokens) != len(tc.wantTokenTypes) {
				t.Fatalf("token count mismatch: got %d, want %d\ntokens: %v", len(tokens), len(tc.wantTokenTypes), tokens)
			}

			for i, tok := range tokens {
				if tok.Type != tc.wantTokenTypes[i] {
					t.Errorf("token[%d] type: got %v, want %v", i, tok.Type, tc.wantTokenTypes[i])
				}
				gotVal := tok.getValue(tc.input)
				if gotVal != tc.wantValues[i] {
					t.Errorf("token[%d] value: got %q, want %q", i, gotVal, tc.wantValues[i])
				}
			}
		})
	}
}

func TestLexComment_ParserIntegration(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "comment in addition",
			input:   "a /* plus */ + b",
			wantErr: false,
		},
		{
			name:    "comment in function call",
			input:   "fn(/* argument */ a, /* argument */ b)",
			wantErr: false,
		},
		{
			name:    "comment in array literal",
			input:   "[/* first */ 1, /* second */ 2]",
			wantErr: false,
		},
		{
			name:    "comment in object literal",
			input:   "{/* key */ a: /* value */ 1}",
			wantErr: false,
		},
		{
			name:    "comment before dot access",
			input:   "obj /* comment */ .field",
			wantErr: false,
		},
		{
			name:    "unterminated should error",
			input:   "a + /* oops",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			parser := NewExpressionParser(context.Background(), tc.input, "test")
			_, diagnostics := parser.ParseExpression(context.Background())
			hasErr := HasErrors(diagnostics)

			if hasErr != tc.wantErr {
				t.Errorf("parse error: got %v (diagnostics=%v), want error=%v", hasErr, diagnostics, tc.wantErr)
			}
		})
	}
}
