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

package annotator_domain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestParseGoCompileError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		input        string
		wantFilePath string
		wantMessage  string
		wantLine     int
		wantColumn   int
		wantOK       bool
	}{
		{
			name:         "standard error format",
			input:        "/path/to/file.go:10:5: undefined: foo",
			wantOK:       true,
			wantFilePath: "/path/to/file.go",
			wantLine:     10,
			wantColumn:   5,
			wantMessage:  "undefined: foo",
		},
		{
			name:         "error with spaces in path",
			input:        "/path/with spaces/file.go:20:15: cannot use x",
			wantOK:       true,
			wantFilePath: "/path/with spaces/file.go",
			wantLine:     20,
			wantColumn:   15,
			wantMessage:  "cannot use x",
		},
		{
			name:         "error at line 1 column 1",
			input:        "/root/main.go:1:1: package clause expected",
			wantOK:       true,
			wantFilePath: "/root/main.go",
			wantLine:     1,
			wantColumn:   1,
			wantMessage:  "package clause expected",
		},
		{
			name:         "large line and column numbers",
			input:        "/code/large.go:9999:888: type mismatch",
			wantOK:       true,
			wantFilePath: "/code/large.go",
			wantLine:     9999,
			wantColumn:   888,
			wantMessage:  "type mismatch",
		},
		{
			name:         "multiline error keeps first line for location but full message",
			input:        "/path/file.go:5:10: cannot convert\n\thave int\n\twant string",
			wantOK:       true,
			wantFilePath: "/path/file.go",
			wantLine:     5,
			wantColumn:   10,
			wantMessage:  "cannot convert\n\thave int\n\twant string",
		},
		{
			name:   "invalid format - no line number",
			input:  "/path/to/file.go: undefined: foo",
			wantOK: false,
		},
		{
			name:   "invalid format - not a go file",
			input:  "/path/to/file.txt:10:5: some error",
			wantOK: false,
		},
		{
			name:   "invalid format - missing colon",
			input:  "/path/to/file.go 10:5: error",
			wantOK: false,
		},
		{
			name:   "invalid format - empty string",
			input:  "",
			wantOK: false,
		},
		{
			name:   "invalid format - just text",
			input:  "some random text",
			wantOK: false,
		},
		{
			name:   "invalid format - non-numeric line",
			input:  "/path/to/file.go:abc:5: error",
			wantOK: false,
		},
		{
			name:   "invalid format - non-numeric column",
			input:  "/path/to/file.go:10:xyz: error",
			wantOK: false,
		},
		{
			name:         "windows-style path",
			input:        "C:/Users/test/file.go:15:8: undefined: bar",
			wantOK:       true,
			wantFilePath: "C:/Users/test/file.go",
			wantLine:     15,
			wantColumn:   8,
			wantMessage:  "undefined: bar",
		},
		{
			name:         "relative path",
			input:        "./internal/pkg/file.go:42:12: syntax error",
			wantOK:       true,
			wantFilePath: "./internal/pkg/file.go",
			wantLine:     42,
			wantColumn:   12,
			wantMessage:  "syntax error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, ok := parseGoCompileError(tc.input)
			assert.Equal(t, tc.wantOK, ok)

			if tc.wantOK {
				assert.NotNil(t, result)
				assert.Equal(t, tc.wantFilePath, result.FilePath)
				assert.Equal(t, tc.wantLine, result.Line)
				assert.Equal(t, tc.wantColumn, result.Column)
				assert.Equal(t, tc.wantMessage, result.Message)
			}
		})
	}
}

func TestExtractSearchTerms(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		contains []string
		excludes []string
	}{
		{
			name:     "extracts method call pattern",
			input:    "undefined: user.GetProfile",
			contains: []string{"user.GetProfile"},
		},
		{
			name:     "extracts multiple method calls",
			input:    "cannot call service.Fetch and db.Query",
			contains: []string{"service.Fetch", "db.Query"},
		},
		{
			name:     "extracts identifiers",
			input:    "undefined: myVariable",
			contains: []string{"myVariable"},
			excludes: []string{"undefined"},
		},
		{
			name:     "filters out noise words",
			input:    "cannot use int as string in argument to call",
			excludes: []string{"cannot", "use", "as", "in", "argument", "to", "call", "int", "string"},
		},
		{
			name:     "filters out Go keywords",
			input:    "func type var const if else for range",
			excludes: []string{"func", "type", "var", "const", "if", "else", "for", "range"},
		},
		{
			name:     "extracts CamelCase identifiers",
			input:    "undefined: MyCustomType",
			contains: []string{"MyCustomType"},
		},
		{
			name:     "extracts snake_case identifiers",
			input:    "undefined: my_variable_name",
			contains: []string{"my_variable_name"},
		},
		{
			name:     "filters short words",
			input:    "a b c is ok",
			excludes: []string{"a", "b", "c"},
		},
		{
			name:     "method call takes priority over parts",
			input:    "undefined: editorService.GetFolderAncestors",
			contains: []string{"editorService.GetFolderAncestors"},
		},
		{
			name:     "empty string returns empty slice",
			input:    "",
			contains: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := extractSearchTerms(tc.input)

			for _, expected := range tc.contains {
				assert.Contains(t, result, expected, "should contain %q", expected)
			}

			for _, excluded := range tc.excludes {
				assert.NotContains(t, result, excluded, "should not contain %q", excluded)
			}
		})
	}
}

func TestIsNoiseWord(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		word     string
		expected bool
	}{

		{name: "single char is noise", word: "a", expected: true},
		{name: "empty string is noise", word: "", expected: true},

		{name: "type is noise", word: "type", expected: true},
		{name: "func is noise", word: "func", expected: true},
		{name: "var is noise", word: "var", expected: true},
		{name: "const is noise", word: "const", expected: true},
		{name: "if is noise", word: "if", expected: true},
		{name: "else is noise", word: "else", expected: true},
		{name: "for is noise", word: "for", expected: true},
		{name: "range is noise", word: "range", expected: true},
		{name: "return is noise", word: "return", expected: true},
		{name: "nil is noise", word: "nil", expected: true},
		{name: "true is noise", word: "true", expected: true},
		{name: "false is noise", word: "false", expected: true},

		{name: "int is noise", word: "int", expected: true},
		{name: "string is noise", word: "string", expected: true},
		{name: "bool is noise", word: "bool", expected: true},
		{name: "error is noise", word: "error", expected: true},
		{name: "any is noise", word: "any", expected: true},

		{name: "undefined is noise", word: "undefined", expected: true},
		{name: "cannot is noise", word: "cannot", expected: true},
		{name: "use is noise", word: "use", expected: true},
		{name: "as is noise", word: "as", expected: true},
		{name: "in is noise", word: "in", expected: true},
		{name: "to is noise", word: "to", expected: true},
		{name: "field is noise", word: "field", expected: true},
		{name: "method is noise", word: "method", expected: true},

		{name: "myVar is not noise", word: "myVar", expected: false},
		{name: "CustomType is not noise", word: "CustomType", expected: false},
		{name: "getData is not noise", word: "getData", expected: false},
		{name: "user_id is not noise", word: "user_id", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := isNoiseWord(tc.word)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFindTermInSource(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		source     string
		term       string
		wantExpr   string
		wantLine   int
		wantColumn int
		wantLength int
	}{
		{
			name:       "finds term on first line",
			source:     "hello world",
			term:       "world",
			wantLine:   1,
			wantColumn: 7,
			wantLength: 5,
			wantExpr:   "world",
		},
		{
			name:       "finds term on second line",
			source:     "first line\nsecond line with target",
			term:       "target",
			wantLine:   2,
			wantColumn: 18,
			wantLength: 6,
			wantExpr:   "target",
		},
		{
			name:       "prefers code lines over comments",
			source:     "// comment with myFunc\nmyFunc := 42",
			term:       "myFunc",
			wantLine:   2,
			wantColumn: 1,
			wantLength: 6,
			wantExpr:   "myFunc",
		},
		{
			name:       "skips comment lines",
			source:     "// myVar in comment\ncode line\nmyVar := 1",
			term:       "myVar",
			wantLine:   3,
			wantColumn: 1,
			wantLength: 5,
			wantExpr:   "myVar",
		},
		{
			name:       "finds in block comment start",
			source:     "/* myFunc here */\nother line",
			term:       "myFunc",
			wantLine:   1,
			wantColumn: 4,
			wantLength: 6,
			wantExpr:   "myFunc",
		},
		{
			name:       "falls back to comment if only match",
			source:     "// only in comment myTerm",
			term:       "myTerm",
			wantLine:   1,
			wantColumn: 20,
			wantLength: 6,
			wantExpr:   "myTerm",
		},
		{
			name:       "term not found returns zero line",
			source:     "some code here",
			term:       "notfound",
			wantLine:   0,
			wantColumn: 0,
			wantLength: 0,
			wantExpr:   "",
		},
		{
			name:       "empty source returns zero line",
			source:     "",
			term:       "anything",
			wantLine:   0,
			wantColumn: 0,
			wantLength: 0,
			wantExpr:   "",
		},
		{
			name:       "empty term returns zero line",
			source:     "some source",
			term:       "",
			wantLine:   1,
			wantColumn: 1,
			wantLength: 0,
			wantExpr:   "",
		},
		{
			name:       "finds at start of line",
			source:     "line1\ntarget at start",
			term:       "target",
			wantLine:   2,
			wantColumn: 1,
			wantLength: 6,
			wantExpr:   "target",
		},
		{
			name:       "finds at end of line",
			source:     "ends with target",
			term:       "target",
			wantLine:   1,
			wantColumn: 11,
			wantLength: 6,
			wantExpr:   "target",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := findTermInSource(tc.source, tc.term)
			assert.Equal(t, tc.wantLine, result.Line)
			assert.Equal(t, tc.wantColumn, result.Column)
			assert.Equal(t, tc.wantLength, result.Length)
			assert.Equal(t, tc.wantExpr, result.Expression)
		})
	}
}

func TestIsCommentLine(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		line     string
		expected bool
	}{

		{name: "line comment with //", line: "// this is a comment", expected: true},
		{name: "line comment with leading spaces", line: "   // indented comment", expected: true},
		{name: "line comment with leading tabs", line: "\t// tabbed comment", expected: true},

		{name: "block comment start", line: "/* block comment */", expected: true},
		{name: "block comment start with spaces", line: "  /* block start", expected: true},

		{name: "code line", line: "x := 42", expected: false},
		{name: "code with inline comment", line: "x := 42 // inline", expected: false},
		{name: "empty line", line: "", expected: false},
		{name: "whitespace only", line: "   ", expected: false},
		{name: "string containing //", line: `s := "// not a comment"`, expected: false},
		{name: "slash without second slash", line: "x := a / b", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := isCommentLine(tc.line)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestEstimateLineFromGeneratedLine(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		vc            *annotator_dto.VirtualComponent
		name          string
		generatedLine int
		expectedLine  int
	}{
		{
			name:          "with script location uses offset",
			generatedLine: 50,
			vc: &annotator_dto.VirtualComponent{
				Source: &annotator_dto.ParsedComponent{
					Script: &annotator_dto.ParsedScript{
						ScriptStartLocation: ast_domain.Location{Line: 10},
					},
				},
			},
			expectedLine: 50,
		},
		{
			name:          "with script location clamps negative offset",
			generatedLine: 5,
			vc: &annotator_dto.VirtualComponent{
				Source: &annotator_dto.ParsedComponent{
					Script: &annotator_dto.ParsedScript{
						ScriptStartLocation: ast_domain.Location{Line: 10},
					},
				},
			},
			expectedLine: 11,
		},
		{
			name:          "without script block uses generated line",
			generatedLine: 25,
			vc: &annotator_dto.VirtualComponent{
				Source: &annotator_dto.ParsedComponent{
					Script: nil,
				},
			},
			expectedLine: 25,
		},
		{
			name:          "with zero script start line uses generated line",
			generatedLine: 30,
			vc: &annotator_dto.VirtualComponent{
				Source: &annotator_dto.ParsedComponent{
					Script: &annotator_dto.ParsedScript{
						ScriptStartLocation: ast_domain.Location{Line: 0},
					},
				},
			},
			expectedLine: 30,
		},
		{
			name:          "empty source uses generated line",
			generatedLine: 15,
			vc: &annotator_dto.VirtualComponent{
				Source: &annotator_dto.ParsedComponent{},
			},
			expectedLine: 15,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := estimateLineFromGeneratedLine(tc.generatedLine, tc.vc)
			assert.Equal(t, tc.expectedLine, result)
		})
	}
}

func TestExtractSearchTermsDeduplicates(t *testing.T) {
	t.Parallel()

	input := "undefined: myFunc and also myFunc again"
	result := extractSearchTerms(input)

	count := 0
	for _, term := range result {
		if term == "myFunc" {
			count++
		}
	}
	assert.Equal(t, 1, count, "myFunc should only appear once in results")
}

func TestExtractSearchTermsMethodCallPriority(t *testing.T) {
	t.Parallel()

	input := "undefined: user.GetProfile"
	result := extractSearchTerms(input)

	if len(result) > 0 {
		assert.Equal(t, "user.GetProfile", result[0], "method call should be first")
	}
}

func TestFindVirtualComponentForGeneratedFile(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		vm            *annotator_dto.VirtualModule
		name          string
		generatedPath string
		wantHash      string
		wantNil       bool
	}{
		{
			name:          "exact match on VirtualGoFilePath",
			generatedPath: "/virtual/component.go",
			vm: &annotator_dto.VirtualModule{
				ComponentsByGoPath: map[string]*annotator_dto.VirtualComponent{
					"test/pkg": {
						HashedName:        "comp_hash",
						VirtualGoFilePath: "/virtual/component.go",
					},
				},
			},
			wantNil:  false,
			wantHash: "comp_hash",
		},
		{
			name:          "match by directory fallback",
			generatedPath: "/virtual/other_file.go",
			vm: &annotator_dto.VirtualModule{
				ComponentsByGoPath: map[string]*annotator_dto.VirtualComponent{
					"test/pkg": {
						HashedName:        "dir_match_hash",
						VirtualGoFilePath: "/virtual/component.go",
					},
				},
			},
			wantNil:  false,
			wantHash: "dir_match_hash",
		},
		{
			name:          "no match returns nil",
			generatedPath: "/completely/different/path.go",
			vm: &annotator_dto.VirtualModule{
				ComponentsByGoPath: map[string]*annotator_dto.VirtualComponent{
					"test/pkg": {
						HashedName:        "no_match",
						VirtualGoFilePath: "/virtual/component.go",
					},
				},
			},
			wantNil: true,
		},
		{
			name:          "empty module returns nil",
			generatedPath: "/virtual/component.go",
			vm: &annotator_dto.VirtualModule{
				ComponentsByGoPath: map[string]*annotator_dto.VirtualComponent{},
			},
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := findVirtualComponentForGeneratedFile(tc.generatedPath, tc.vm)

			if tc.wantNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tc.wantHash, result.HashedName)
			}
		})
	}
}

func TestConvertTypeInspectorErrorToDiagnostics(t *testing.T) {
	t.Parallel()

	t.Run("nil error returns nil diagnostics", func(t *testing.T) {
		t.Parallel()

		result := convertTypeInspectorErrorToDiagnostics(
			context.Background(),
			nil,
			&annotator_dto.VirtualModule{},
			nil,
		)

		assert.Nil(t, result)
	})

	t.Run("nil virtual module returns nil diagnostics", func(t *testing.T) {
		t.Parallel()

		result := convertTypeInspectorErrorToDiagnostics(
			context.Background(),
			errors.New("some error"),
			nil,
			nil,
		)

		assert.Nil(t, result)
	})

	t.Run("unparseable error message produces no diagnostics", func(t *testing.T) {
		t.Parallel()

		result := convertTypeInspectorErrorToDiagnostics(
			context.Background(),
			errors.New("not a Go compiler error format"),
			&annotator_dto.VirtualModule{
				ComponentsByGoPath: map[string]*annotator_dto.VirtualComponent{},
			},
			newTestFSReader(),
		)

		assert.Empty(t, result)
	})

	t.Run("strips errors found during package loading prefix", func(t *testing.T) {
		t.Parallel()

		vm := &annotator_dto.VirtualModule{
			ComponentsByGoPath: map[string]*annotator_dto.VirtualComponent{},
		}

		result := convertTypeInspectorErrorToDiagnostics(
			context.Background(),
			errors.New("errors found during package loading: /test/file.go:5:3: undefined: myVar"),
			vm,
			newTestFSReader(),
		)

		assert.NotEmpty(t, result)
		assert.Contains(t, result[0].Message, "undefined: myVar")
	})

	t.Run("strips failed to load packages from source prefix", func(t *testing.T) {
		t.Parallel()

		vm := &annotator_dto.VirtualModule{
			ComponentsByGoPath: map[string]*annotator_dto.VirtualComponent{},
		}

		result := convertTypeInspectorErrorToDiagnostics(
			context.Background(),
			errors.New("failed to load packages from source: /test/file.go:10:1: syntax error"),
			vm,
			newTestFSReader(),
		)

		assert.NotEmpty(t, result)
		assert.Contains(t, result[0].Message, "syntax error")
	})

	t.Run("parses multiple semicolon-separated errors", func(t *testing.T) {
		t.Parallel()

		vm := &annotator_dto.VirtualModule{
			ComponentsByGoPath: map[string]*annotator_dto.VirtualComponent{},
		}

		result := convertTypeInspectorErrorToDiagnostics(
			context.Background(),
			errors.New("/test/a.go:1:1: error one; /test/b.go:2:2: error two"),
			vm,
			newTestFSReader(),
		)

		assert.Len(t, result, 2)
	})
}

func TestMapGoErrorToDiagnostic(t *testing.T) {
	t.Parallel()

	t.Run("maps error to source file when virtual component found with nil fsReader skips file read", func(t *testing.T) {
		t.Parallel()

		goErr := &goCompileError{
			FilePath: "/virtual/component.go",
			Line:     10,
			Column:   5,
			Message:  "undefined: myFunc",
		}

		vm := &annotator_dto.VirtualModule{
			ComponentsByGoPath: map[string]*annotator_dto.VirtualComponent{
				"test/pkg": {
					HashedName:        "test_hash",
					VirtualGoFilePath: "/virtual/component.go",
					Source: &annotator_dto.ParsedComponent{
						SourcePath: "/test/component.piko",
						Script: &annotator_dto.ParsedScript{
							ScriptStartLocation: ast_domain.Location{Line: 5},
						},
					},
				},
			},
		}

		fsReader := newTestFSReader()

		diagnostic, ok := mapGoErrorToDiagnostic(context.Background(), goErr, vm, fsReader)

		assert.True(t, ok)
		assert.NotNil(t, diagnostic)
		assert.Equal(t, "/test/component.piko", diagnostic.SourcePath)
		assert.Equal(t, "undefined: myFunc", diagnostic.Message)
		assert.Equal(t, ast_domain.Error, diagnostic.Severity)
	})

	t.Run("falls back to Go file diagnostic when no virtual component found", func(t *testing.T) {
		t.Parallel()

		goErr := &goCompileError{
			FilePath: "/some/other/file.go",
			Line:     5,
			Column:   3,
			Message:  "type mismatch",
		}

		vm := &annotator_dto.VirtualModule{
			ComponentsByGoPath: map[string]*annotator_dto.VirtualComponent{},
		}

		fsReader := newTestFSReader()

		diagnostic, ok := mapGoErrorToDiagnostic(context.Background(), goErr, vm, fsReader)

		assert.True(t, ok)
		assert.NotNil(t, diagnostic)
		assert.Equal(t, "/some/other/file.go", diagnostic.SourcePath)
		assert.Equal(t, 5, diagnostic.Location.Line)
		assert.Equal(t, 3, diagnostic.Location.Column)
	})
}

func TestCreateDiagnosticForGoFile(t *testing.T) {
	t.Parallel()

	fsReader := newTestFSReader()

	t.Run("uses error file path when no overlay matches", func(t *testing.T) {
		t.Parallel()

		goErr := &goCompileError{
			FilePath: "/src/main.go",
			Line:     1,
			Column:   1,
			Message:  "package clause expected",
		}

		vm := &annotator_dto.VirtualModule{
			SourceOverlay: nil,
		}

		diagnostic, ok := createDiagnosticForGoFile(context.Background(), goErr, vm, fsReader)

		assert.True(t, ok)
		assert.NotNil(t, diagnostic)
		assert.Equal(t, "/src/main.go", diagnostic.SourcePath)
		assert.Equal(t, 1, diagnostic.Location.Line)
	})

	t.Run("uses overlay path when it matches", func(t *testing.T) {
		t.Parallel()

		goErr := &goCompileError{
			FilePath: "/src/main.go",
			Line:     10,
			Column:   5,
			Message:  "undefined: foo",
		}

		vm := &annotator_dto.VirtualModule{
			SourceOverlay: map[string][]byte{
				"/src/main.go": {},
			},
		}

		diagnostic, ok := createDiagnosticForGoFile(context.Background(), goErr, vm, fsReader)

		assert.True(t, ok)
		assert.NotNil(t, diagnostic)
		assert.Equal(t, "/src/main.go", diagnostic.SourcePath)
	})

	t.Run("diagnostic has error severity", func(t *testing.T) {
		t.Parallel()

		goErr := &goCompileError{
			FilePath: "/src/main.go",
			Line:     1,
			Column:   1,
			Message:  "test error",
		}

		diagnostic, ok := createDiagnosticForGoFile(context.Background(), goErr, nil, fsReader)

		assert.True(t, ok)
		assert.Equal(t, ast_domain.Error, diagnostic.Severity)
	})

	t.Run("finds expression from source file when fsReader returns content", func(t *testing.T) {
		t.Parallel()

		goErr := &goCompileError{
			FilePath: "/src/main.go",
			Line:     2,
			Column:   5,
			Message:  "undefined: myFunc",
		}

		readerWithFile := newTestFSReader()
		readerWithFile.addFile("/src/main.go", "package main\nmyFunc := 42\n")

		diagnostic, ok := createDiagnosticForGoFile(context.Background(), goErr, nil, readerWithFile)

		assert.True(t, ok)
		assert.NotNil(t, diagnostic)
		assert.Equal(t, "myFunc", diagnostic.Expression)
	})
}

func TestMapGeneratedLineToSource(t *testing.T) {
	t.Parallel()

	t.Run("finds term in PK source file when source exists", func(t *testing.T) {
		t.Parallel()

		goErr := &goCompileError{
			FilePath: "/virtual/component.go",
			Line:     15,
			Column:   3,
			Message:  "undefined: MyHelper",
		}

		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/test/component.pk",
				Script: &annotator_dto.ParsedScript{
					ScriptStartLocation: ast_domain.Location{Line: 5},
				},
			},
		}

		fsReader := newTestFSReader()
		fsReader.addFile("/test/component.pk", "<template>\n  <div>{{ MyHelper() }}</div>\n</template>\n<script>\nfunc MyHelper() string {\n  return \"hello\"\n}\n</script>\n")

		result := mapGeneratedLineToSource(context.Background(), goErr, vc, nil, fsReader)

		assert.Equal(t, "MyHelper", result.Expression)
		assert.Greater(t, result.Line, 0)
		assert.Greater(t, result.Length, 0)
	})

	t.Run("falls back to estimated line when source file cannot be read", func(t *testing.T) {
		t.Parallel()

		goErr := &goCompileError{
			FilePath: "/virtual/component.go",
			Line:     20,
			Column:   5,
			Message:  "undefined: unknownThing",
		}

		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/test/missing.pk",
				Script: &annotator_dto.ParsedScript{
					ScriptStartLocation: ast_domain.Location{Line: 10},
				},
			},
		}

		fsReader := newTestFSReader()

		result := mapGeneratedLineToSource(context.Background(), goErr, vc, nil, fsReader)

		assert.Empty(t, result.Expression)
		assert.Equal(t, 1, result.Column)

		assert.Equal(t, 20, result.Line)
	})

	t.Run("falls back to estimated line when term not found in source", func(t *testing.T) {
		t.Parallel()

		goErr := &goCompileError{
			FilePath: "/virtual/component.go",
			Line:     15,
			Column:   3,
			Message:  "undefined: totallyAbsentSymbol",
		}

		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/test/component.pk",
				Script: &annotator_dto.ParsedScript{
					ScriptStartLocation: ast_domain.Location{Line: 8},
				},
			},
		}

		fsReader := newTestFSReader()
		fsReader.addFile("/test/component.pk", "<template>\n  <div>Hello</div>\n</template>\n<script>\nfunc Render() string { return \"\" }\n</script>\n")

		result := mapGeneratedLineToSource(context.Background(), goErr, vc, nil, fsReader)

		assert.Empty(t, result.Expression)
		assert.Equal(t, 1, result.Column)
	})

	t.Run("falls back to estimated line when source path is empty", func(t *testing.T) {
		t.Parallel()

		goErr := &goCompileError{
			FilePath: "/virtual/component.go",
			Line:     25,
			Column:   1,
			Message:  "undefined: thing",
		}

		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "",
				Script: &annotator_dto.ParsedScript{
					ScriptStartLocation: ast_domain.Location{Line: 5},
				},
			},
		}

		fsReader := newTestFSReader()

		result := mapGeneratedLineToSource(context.Background(), goErr, vc, nil, fsReader)

		assert.Empty(t, result.Expression)
		assert.Equal(t, 1, result.Column)

		assert.Equal(t, 20, result.Line)
	})
}
