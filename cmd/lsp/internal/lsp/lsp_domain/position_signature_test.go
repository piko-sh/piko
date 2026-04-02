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

func TestCountTopLevelCommas(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "no commas",
			input:    "hello world",
			expected: 0,
		},
		{
			name:     "single comma",
			input:    "a, b",
			expected: 1,
		},
		{
			name:     "multiple commas",
			input:    "a, b, c, d",
			expected: 3,
		},
		{
			name:     "empty string",
			input:    "",
			expected: 0,
		},
		{
			name:     "comma in parentheses ignored",
			input:    "func(a, b), c",
			expected: 1,
		},
		{
			name:     "nested parentheses",
			input:    "outer(inner(a, b), c), d",
			expected: 1,
		},
		{
			name:     "comma in brackets ignored",
			input:    "arr[a, b], c",
			expected: 1,
		},
		{
			name:     "comma in braces ignored",
			input:    "obj{a, b}, c",
			expected: 1,
		},
		{
			name:     "comma in double quoted string ignored",
			input:    `"a, b", c`,
			expected: 1,
		},
		{
			name:     "comma in single quoted string ignored",
			input:    `'a, b', c`,
			expected: 1,
		},
		{
			name:     "comma in raw string ignored",
			input:    "`a, b`, c",
			expected: 1,
		},
		{
			name:     "escaped quote in string",
			input:    `"a\", b", c`,
			expected: 1,
		},
		{
			name:     "complex expression",
			input:    `func(a, b), "x, y", arr[1, 2], c`,
			expected: 3,
		},
		{
			name:     "deeply nested",
			input:    "a(b(c(d, e), f), g), h",
			expected: 1,
		},
		{
			name:     "mixed nesting",
			input:    "a[b{c, d}], e(f, g), h",
			expected: 2,
		},
		{
			name:     "only commas",
			input:    ",,,",
			expected: 3,
		},
		{
			name:     "spaces around commas",
			input:    "a , b , c",
			expected: 2,
		},
		{
			name:     "string with escaped backslash",
			input:    `"a\\", b`,
			expected: 1,
		},
		{
			name:     "raw string with quotes inside",
			input:    "`\"a, b\"`, c",
			expected: 1,
		},
		{
			name:     "unclosed parenthesis still tracks depth",
			input:    "a(b, c",
			expected: 0,
		},
		{
			name:     "extra closing parenthesis clamped to zero",
			input:    "a), b",
			expected: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := countTopLevelCommas(tc.input)
			if result != tc.expected {
				t.Errorf("countTopLevelCommas(%q) = %d, want %d", tc.input, result, tc.expected)
			}
		})
	}
}

func TestCommaCounter_HandleEscape(t *testing.T) {
	testCases := []struct {
		name           string
		escapeNext     bool
		expectedReturn bool
		expectedState  bool
	}{
		{
			name:           "escape flag set",
			escapeNext:     true,
			expectedReturn: true,
			expectedState:  false,
		},
		{
			name:           "escape flag not set",
			escapeNext:     false,
			expectedReturn: false,
			expectedState:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := &commaCounter{escapeNext: tc.escapeNext}
			result := c.handleEscape()
			if result != tc.expectedReturn {
				t.Errorf("handleEscape() = %v, want %v", result, tc.expectedReturn)
			}
			if c.escapeNext != tc.expectedState {
				t.Errorf("escapeNext = %v, want %v", c.escapeNext, tc.expectedState)
			}
		})
	}
}

func TestCommaCounter_HandleRawString(t *testing.T) {
	testCases := []struct {
		name           string
		char           byte
		initialState   bool
		expectedReturn bool
		expectedState  bool
	}{
		{
			name:           "backtick starts raw string",
			char:           '`',
			initialState:   false,
			expectedReturn: true,
			expectedState:  true,
		},
		{
			name:           "backtick ends raw string",
			char:           '`',
			initialState:   true,
			expectedReturn: true,
			expectedState:  false,
		},
		{
			name:           "other char in raw string",
			char:           'a',
			initialState:   true,
			expectedReturn: true,
			expectedState:  true,
		},
		{
			name:           "other char outside raw string",
			char:           'a',
			initialState:   false,
			expectedReturn: false,
			expectedState:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := &commaCounter{inRawString: tc.initialState}
			result := c.handleRawString(tc.char)
			if result != tc.expectedReturn {
				t.Errorf("handleRawString(%c) = %v, want %v", tc.char, result, tc.expectedReturn)
			}
			if c.inRawString != tc.expectedState {
				t.Errorf("inRawString = %v, want %v", c.inRawString, tc.expectedState)
			}
		})
	}
}

func TestCommaCounter_HandleString(t *testing.T) {
	testCases := []struct {
		name           string
		char           byte
		inString       bool
		expectedReturn bool
		expectedState  bool
		expectEscape   bool
	}{
		{
			name:           "double quote starts string",
			char:           '"',
			inString:       false,
			expectedReturn: true,
			expectedState:  true,
		},
		{
			name:           "double quote ends string",
			char:           '"',
			inString:       true,
			expectedReturn: true,
			expectedState:  false,
		},
		{
			name:           "single quote starts string",
			char:           '\'',
			inString:       false,
			expectedReturn: true,
			expectedState:  true,
		},
		{
			name:           "single quote ends string",
			char:           '\'',
			inString:       true,
			expectedReturn: true,
			expectedState:  false,
		},
		{
			name:           "backslash in string sets escape",
			char:           '\\',
			inString:       true,
			expectedReturn: true,
			expectedState:  true,
			expectEscape:   true,
		},
		{
			name:           "other char in string",
			char:           'a',
			inString:       true,
			expectedReturn: true,
			expectedState:  true,
		},
		{
			name:           "other char outside string",
			char:           'a',
			inString:       false,
			expectedReturn: false,
			expectedState:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := &commaCounter{inString: tc.inString}
			result := c.handleString(tc.char)
			if result != tc.expectedReturn {
				t.Errorf("handleString(%c) = %v, want %v", tc.char, result, tc.expectedReturn)
			}
			if c.inString != tc.expectedState {
				t.Errorf("inString = %v, want %v", c.inString, tc.expectedState)
			}
			if c.escapeNext != tc.expectEscape {
				t.Errorf("escapeNext = %v, want %v", c.escapeNext, tc.expectEscape)
			}
		})
	}
}

func TestCommaCounter_HandleNesting(t *testing.T) {
	testCases := []struct {
		name          string
		char          byte
		initialDepth  int
		initialCount  int
		expectedDepth int
		expectedCount int
	}{
		{
			name:          "open paren increases depth",
			char:          '(',
			initialDepth:  0,
			expectedDepth: 1,
		},
		{
			name:          "open bracket increases depth",
			char:          '[',
			initialDepth:  0,
			expectedDepth: 1,
		},
		{
			name:          "open brace increases depth",
			char:          '{',
			initialDepth:  0,
			expectedDepth: 1,
		},
		{
			name:          "close paren decreases depth",
			char:          ')',
			initialDepth:  1,
			expectedDepth: 0,
		},
		{
			name:          "close bracket decreases depth",
			char:          ']',
			initialDepth:  1,
			expectedDepth: 0,
		},
		{
			name:          "close brace decreases depth",
			char:          '}',
			initialDepth:  1,
			expectedDepth: 0,
		},
		{
			name:          "close paren at zero depth stays zero",
			char:          ')',
			initialDepth:  0,
			expectedDepth: 0,
		},
		{
			name:          "comma at depth zero counts",
			char:          ',',
			initialDepth:  0,
			initialCount:  0,
			expectedDepth: 0,
			expectedCount: 1,
		},
		{
			name:          "comma at depth nonzero does not count",
			char:          ',',
			initialDepth:  1,
			initialCount:  0,
			expectedDepth: 1,
			expectedCount: 0,
		},
		{
			name:          "other char has no effect",
			char:          'a',
			initialDepth:  2,
			initialCount:  5,
			expectedDepth: 2,
			expectedCount: 5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := &commaCounter{depth: tc.initialDepth, count: tc.initialCount}
			c.handleNesting(tc.char)
			if c.depth != tc.expectedDepth {
				t.Errorf("depth = %d, want %d", c.depth, tc.expectedDepth)
			}
			if c.count != tc.expectedCount {
				t.Errorf("count = %d, want %d", c.count, tc.expectedCount)
			}
		})
	}
}

func TestExtractTextBetweenPositions(t *testing.T) {
	testCases := []struct {
		name     string
		content  string
		expected string
		start    protocol.Position
		end      protocol.Position
	}{
		{
			name:     "same line extraction",
			content:  "hello world",
			start:    protocol.Position{Line: 0, Character: 0},
			end:      protocol.Position{Line: 0, Character: 5},
			expected: "hello",
		},
		{
			name:     "same line middle extraction",
			content:  "hello world",
			start:    protocol.Position{Line: 0, Character: 6},
			end:      protocol.Position{Line: 0, Character: 11},
			expected: "world",
		},
		{
			name:     "multi line extraction",
			content:  "line1\nline2\nline3",
			start:    protocol.Position{Line: 0, Character: 3},
			end:      protocol.Position{Line: 2, Character: 3},
			expected: "e1\nline2\nlin",
		},
		{
			name:     "empty extraction same position",
			content:  "hello",
			start:    protocol.Position{Line: 0, Character: 2},
			end:      protocol.Position{Line: 0, Character: 2},
			expected: "",
		},
		{
			name:     "start line out of bounds",
			content:  "hello",
			start:    protocol.Position{Line: 5, Character: 0},
			end:      protocol.Position{Line: 5, Character: 3},
			expected: "",
		},
		{
			name:     "end line out of bounds",
			content:  "hello",
			start:    protocol.Position{Line: 0, Character: 0},
			end:      protocol.Position{Line: 5, Character: 3},
			expected: "",
		},
		{
			name:     "start character out of bounds",
			content:  "hello",
			start:    protocol.Position{Line: 0, Character: 10},
			end:      protocol.Position{Line: 0, Character: 15},
			expected: "",
		},
		{
			name:     "end character out of bounds",
			content:  "hello",
			start:    protocol.Position{Line: 0, Character: 0},
			end:      protocol.Position{Line: 0, Character: 15},
			expected: "",
		},
		{
			name:     "full first line",
			content:  "first\nsecond",
			start:    protocol.Position{Line: 0, Character: 0},
			end:      protocol.Position{Line: 0, Character: 5},
			expected: "first",
		},
		{
			name:     "extract across two lines",
			content:  "hello\nworld",
			start:    protocol.Position{Line: 0, Character: 2},
			end:      protocol.Position{Line: 1, Character: 3},
			expected: "llo\nwor",
		},
		{
			name:     "multiple middle lines",
			content:  "aaa\nbbb\nccc\nddd",
			start:    protocol.Position{Line: 0, Character: 1},
			end:      protocol.Position{Line: 3, Character: 2},
			expected: "aa\nbbb\nccc\ndd",
		},
		{
			name:     "empty content",
			content:  "",
			start:    protocol.Position{Line: 0, Character: 0},
			end:      protocol.Position{Line: 0, Character: 0},
			expected: "",
		},
		{
			name:     "single character",
			content:  "x",
			start:    protocol.Position{Line: 0, Character: 0},
			end:      protocol.Position{Line: 0, Character: 1},
			expected: "x",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractTextBetweenPositions([]byte(tc.content), tc.start, tc.end)
			if result != tc.expected {
				t.Errorf("extractTextBetweenPositions() = %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestCountActiveParameter_NilCallExpr(t *testing.T) {
	position := protocol.Position{Line: 0, Character: 5}
	content := []byte("test content")

	result := countActiveParameter(nil, position, content)
	if result != 0 {
		t.Errorf("countActiveParameter(nil, ...) = %d, want 0", result)
	}
}

func TestCommaCounter_ProcessChar_IntegrationSequence(t *testing.T) {
	testCases := []struct {
		name          string
		input         string
		expectedCount int
	}{
		{
			name:          "simple sequence",
			input:         "a, b, c",
			expectedCount: 2,
		},
		{
			name:          "string then comma",
			input:         `"x, y", z`,
			expectedCount: 1,
		},
		{
			name:          "raw string then comma",
			input:         "`x, y`, z",
			expectedCount: 1,
		},
		{
			name:          "nested parens then comma",
			input:         "f(a, b), c",
			expectedCount: 1,
		},
		{
			name:          "escape in string",
			input:         `"\", a", b`,
			expectedCount: 1,
		},
		{
			name:          "all bracket types",
			input:         "([{a, b}]), c",
			expectedCount: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := &commaCounter{}
			for i := 0; i < len(tc.input); i++ {
				c.processChar(tc.input[i])
			}
			if c.count != tc.expectedCount {
				t.Errorf("processChar sequence for %q: count = %d, want %d", tc.input, c.count, tc.expectedCount)
			}
		})
	}
}

func TestCallExprFinder_Visit(t *testing.T) {
	testCases := []struct {
		expression  ast_domain.Expression
		name        string
		targetPos   protocol.Position
		expectMatch bool
	}{
		{
			name:        "nil expression does not panic",
			expression:  nil,
			targetPos:   protocol.Position{Line: 0, Character: 5},
			expectMatch: false,
		},
		{
			name: "non-CallExpr is ignored",
			expression: &ast_domain.Identifier{
				Name:             "foo",
				RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
				SourceLength:     3,
			},
			targetPos:   protocol.Position{Line: 0, Character: 1},
			expectMatch: false,
		},
		{
			name: "CallExpr containing position sets bestMatch",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{
					Name:             "myFunc",
					RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
					SourceLength:     6,
				},
				LparenLocation: ast_domain.Location{Line: 1, Column: 7},
				RparenLocation: ast_domain.Location{Line: 1, Column: 15},
				SourceLength:   16,
			},
			targetPos:   protocol.Position{Line: 0, Character: 10},
			expectMatch: true,
		},
		{
			name: "CallExpr not containing position does not set bestMatch",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{
					Name:             "myFunc",
					RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
					SourceLength:     6,
				},
				LparenLocation: ast_domain.Location{Line: 1, Column: 7},
				RparenLocation: ast_domain.Location{Line: 1, Column: 15},
				SourceLength:   16,
			},
			targetPos:   protocol.Position{Line: 5, Character: 0},
			expectMatch: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			finder := &callExprFinder{
				targetPosition: tc.targetPos,
			}
			finder.visit(tc.expression)

			if tc.expectMatch && finder.bestMatch == nil {
				t.Error("expected bestMatch to be set")
			}
			if !tc.expectMatch && finder.bestMatch != nil {
				t.Errorf("expected bestMatch to be nil, got %T", finder.bestMatch)
			}
		})
	}
}

func TestFindEnclosingCallExpr(t *testing.T) {
	testCases := []struct {
		tree         *ast_domain.TemplateAST
		name         string
		content      []byte
		position     protocol.Position
		expectNonNil bool
	}{
		{
			name:         "empty AST returns nil",
			tree:         newTestAnnotatedAST(),
			position:     protocol.Position{Line: 0, Character: 5},
			content:      []byte("myFunc(a, b)"),
			expectNonNil: false,
		},
		{
			name: "finds call expression containing position",
			tree: func() *ast_domain.TemplateAST {
				node := newTestNodeMultiLine("div", 1, 1, 3, 20)
				node.DirIf = &ast_domain.Directive{
					Expression: &ast_domain.CallExpression{
						Callee: &ast_domain.Identifier{
							Name:             "myFunc",
							RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
							SourceLength:     6,
						},
						Args: []ast_domain.Expression{
							&ast_domain.Identifier{
								Name:             "a",
								RelativeLocation: ast_domain.Location{Line: 0, Column: 7},
								SourceLength:     1,
							},
						},
						LparenLocation:   ast_domain.Location{Line: 1, Column: 7},
						RparenLocation:   ast_domain.Location{Line: 1, Column: 10},
						RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
						SourceLength:     10,
					},
				}
				return newTestAnnotatedAST(node)
			}(),
			position:     protocol.Position{Line: 0, Character: 8},
			content:      []byte("myFunc(a)"),
			expectNonNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			callExpr, _ := findEnclosingCallExpr(tc.tree, tc.position, tc.content)
			if tc.expectNonNil && callExpr == nil {
				t.Error("expected non-nil call expression")
			}
			if !tc.expectNonNil && callExpr != nil {
				t.Errorf("expected nil, got %+v", callExpr)
			}
		})
	}
}
