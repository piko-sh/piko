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

	"piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/js_ast"
)

func TestGetSymbolName(t *testing.T) {
	symbols := []ast.Symbol{
		{OriginalName: "alpha"},
		{OriginalName: "beta"},
		{OriginalName: "gamma"},
	}

	testCases := []struct {
		name     string
		expected string
		symbols  []ast.Symbol
		ref      ast.Ref
	}{
		{
			name:     "valid index returns original name",
			ref:      ast.Ref{InnerIndex: 0},
			symbols:  symbols,
			expected: "alpha",
		},
		{
			name:     "second index returns second name",
			ref:      ast.Ref{InnerIndex: 1},
			symbols:  symbols,
			expected: "beta",
		},
		{
			name:     "last valid index",
			ref:      ast.Ref{InnerIndex: 2},
			symbols:  symbols,
			expected: "gamma",
		},
		{
			name:     "out of range index returns empty string",
			ref:      ast.Ref{InnerIndex: 10},
			symbols:  symbols,
			expected: "",
		},
		{
			name:     "empty symbols returns empty string",
			ref:      ast.Ref{InnerIndex: 0},
			symbols:  []ast.Symbol{},
			expected: "",
		},
		{
			name:     "nil symbols returns empty string",
			ref:      ast.Ref{InnerIndex: 0},
			symbols:  nil,
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := getSymbolName(tc.ref, tc.symbols)
			if result != tc.expected {
				t.Errorf("getSymbolName() = %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestExtractBindingName(t *testing.T) {
	symbols := []ast.Symbol{
		{OriginalName: "myVar"},
		{OriginalName: "anotherVar"},
	}

	testCases := []struct {
		binding  js_ast.Binding
		name     string
		expected string
		symbols  []ast.Symbol
	}{
		{
			name: "BIdentifier binding returns symbol name",
			binding: js_ast.Binding{
				Data: &js_ast.BIdentifier{Ref: ast.Ref{InnerIndex: 0}},
			},
			symbols:  symbols,
			expected: "myVar",
		},
		{
			name: "BIdentifier with second index",
			binding: js_ast.Binding{
				Data: &js_ast.BIdentifier{Ref: ast.Ref{InnerIndex: 1}},
			},
			symbols:  symbols,
			expected: "anotherVar",
		},
		{
			name: "BArray binding returns empty string",
			binding: js_ast.Binding{
				Data: &js_ast.BArray{},
			},
			symbols:  symbols,
			expected: "",
		},
		{
			name: "BIdentifier with out-of-range ref returns empty string",
			binding: js_ast.Binding{
				Data: &js_ast.BIdentifier{Ref: ast.Ref{InnerIndex: 99}},
			},
			symbols:  symbols,
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractBindingName(tc.binding, tc.symbols)
			if result != tc.expected {
				t.Errorf("extractBindingName() = %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestExtractExportClauseNames(t *testing.T) {
	testCases := []struct {
		name     string
		clause   *js_ast.SExportClause
		expected []string
	}{
		{
			name: "single alias",
			clause: &js_ast.SExportClause{
				Items: []js_ast.ClauseItem{
					{Alias: "foo"},
				},
			},
			expected: []string{"foo"},
		},
		{
			name: "multiple aliases",
			clause: &js_ast.SExportClause{
				Items: []js_ast.ClauseItem{
					{Alias: "foo"},
					{Alias: "bar"},
					{Alias: "baz"},
				},
			},
			expected: []string{"foo", "bar", "baz"},
		},
		{
			name: "empty alias items are skipped",
			clause: &js_ast.SExportClause{
				Items: []js_ast.ClauseItem{
					{Alias: "foo"},
					{Alias: ""},
					{Alias: "bar"},
				},
			},
			expected: []string{"foo", "bar"},
		},
		{
			name: "no items returns empty slice",
			clause: &js_ast.SExportClause{
				Items: []js_ast.ClauseItem{},
			},
			expected: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractExportClauseNames(tc.clause)
			if len(result) != len(tc.expected) {
				t.Fatalf("len(result) = %d, want %d", len(result), len(tc.expected))
			}
			for i, name := range result {
				if name != tc.expected[i] {
					t.Errorf("result[%d] = %q, want %q", i, name, tc.expected[i])
				}
			}
		})
	}
}

func TestExtractExportedFunctionName(t *testing.T) {
	symbols := []ast.Symbol{
		{OriginalName: "handleClick"},
		{OriginalName: "processData"},
	}

	testCases := []struct {
		name       string
		jsFunction *js_ast.SFunction
		symbols    []ast.Symbol
		expected   []string
	}{
		{
			name: "exported function with name returns its name",
			jsFunction: &js_ast.SFunction{
				IsExport: true,
				Fn: js_ast.Fn{
					Name: &ast.LocRef{Ref: ast.Ref{InnerIndex: 0}},
				},
			},
			symbols:  symbols,
			expected: []string{"handleClick"},
		},
		{
			name: "non-exported function returns nil",
			jsFunction: &js_ast.SFunction{
				IsExport: false,
				Fn: js_ast.Fn{
					Name: &ast.LocRef{Ref: ast.Ref{InnerIndex: 0}},
				},
			},
			symbols:  symbols,
			expected: nil,
		},
		{
			name: "exported function without name returns nil",
			jsFunction: &js_ast.SFunction{
				IsExport: true,
				Fn:       js_ast.Fn{Name: nil},
			},
			symbols:  symbols,
			expected: nil,
		},
		{
			name: "exported function with out-of-range ref returns nil",
			jsFunction: &js_ast.SFunction{
				IsExport: true,
				Fn: js_ast.Fn{
					Name: &ast.LocRef{Ref: ast.Ref{InnerIndex: 99}},
				},
			},
			symbols:  symbols,
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractExportedFunctionName(tc.jsFunction, tc.symbols)
			if tc.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}
			if len(result) != len(tc.expected) {
				t.Fatalf("len(result) = %d, want %d", len(result), len(tc.expected))
			}
			for i, name := range result {
				if name != tc.expected[i] {
					t.Errorf("result[%d] = %q, want %q", i, name, tc.expected[i])
				}
			}
		})
	}
}

func TestExtractDefaultExportName(t *testing.T) {
	symbols := []ast.Symbol{
		{OriginalName: "myDefaultFn"},
	}

	testCases := []struct {
		name     string
		export   *js_ast.SExportDefault
		symbols  []ast.Symbol
		expected []string
	}{
		{
			name: "default export of named function",
			export: &js_ast.SExportDefault{
				Value: js_ast.Stmt{
					Data: &js_ast.SFunction{
						Fn: js_ast.Fn{
							Name: &ast.LocRef{Ref: ast.Ref{InnerIndex: 0}},
						},
					},
				},
			},
			symbols:  symbols,
			expected: []string{"myDefaultFn"},
		},
		{
			name: "default export of unnamed function returns nil",
			export: &js_ast.SExportDefault{
				Value: js_ast.Stmt{
					Data: &js_ast.SFunction{
						Fn: js_ast.Fn{Name: nil},
					},
				},
			},
			symbols:  symbols,
			expected: nil,
		},
		{
			name: "default export of non-function returns nil",
			export: &js_ast.SExportDefault{
				Value: js_ast.Stmt{
					Data: &js_ast.SLocal{},
				},
			},
			symbols:  symbols,
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractDefaultExportName(tc.export, tc.symbols)
			if tc.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}
			if len(result) != len(tc.expected) {
				t.Fatalf("len(result) = %d, want %d", len(result), len(tc.expected))
			}
			for i, name := range result {
				if name != tc.expected[i] {
					t.Errorf("result[%d] = %q, want %q", i, name, tc.expected[i])
				}
			}
		})
	}
}

func TestExtractLocalExportNames(t *testing.T) {
	symbols := []ast.Symbol{
		{OriginalName: "PI"},
		{OriginalName: "MAX_SIZE"},
	}

	testCases := []struct {
		name     string
		local    *js_ast.SLocal
		symbols  []ast.Symbol
		expected []string
	}{
		{
			name: "non-exported local returns nil",
			local: &js_ast.SLocal{
				IsExport: false,
				Decls: []js_ast.Decl{
					{Binding: js_ast.Binding{Data: &js_ast.BIdentifier{Ref: ast.Ref{InnerIndex: 0}}}},
				},
			},
			symbols:  symbols,
			expected: nil,
		},
		{
			name: "exported local with single binding",
			local: &js_ast.SLocal{
				IsExport: true,
				Decls: []js_ast.Decl{
					{Binding: js_ast.Binding{Data: &js_ast.BIdentifier{Ref: ast.Ref{InnerIndex: 0}}}},
				},
			},
			symbols:  symbols,
			expected: []string{"PI"},
		},
		{
			name: "exported local with multiple bindings",
			local: &js_ast.SLocal{
				IsExport: true,
				Decls: []js_ast.Decl{
					{Binding: js_ast.Binding{Data: &js_ast.BIdentifier{Ref: ast.Ref{InnerIndex: 0}}}},
					{Binding: js_ast.Binding{Data: &js_ast.BIdentifier{Ref: ast.Ref{InnerIndex: 1}}}},
				},
			},
			symbols:  symbols,
			expected: []string{"PI", "MAX_SIZE"},
		},
		{
			name: "exported local with non-identifier binding skips it",
			local: &js_ast.SLocal{
				IsExport: true,
				Decls: []js_ast.Decl{
					{Binding: js_ast.Binding{Data: &js_ast.BIdentifier{Ref: ast.Ref{InnerIndex: 0}}}},
					{Binding: js_ast.Binding{Data: &js_ast.BArray{}}},
				},
			},
			symbols:  symbols,
			expected: []string{"PI"},
		},
		{
			name: "exported local with empty decls",
			local: &js_ast.SLocal{
				IsExport: true,
				Decls:    []js_ast.Decl{},
			},
			symbols:  symbols,
			expected: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractLocalExportNames(tc.local, tc.symbols)
			if tc.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}
			if len(result) != len(tc.expected) {
				t.Fatalf("len(result) = %d, want %d", len(result), len(tc.expected))
			}
			for i, name := range result {
				if name != tc.expected[i] {
					t.Errorf("result[%d] = %q, want %q", i, name, tc.expected[i])
				}
			}
		})
	}
}

func TestExtractExportNamesFromStmt(t *testing.T) {
	symbols := []ast.Symbol{
		{OriginalName: "myFunc"},
		{OriginalName: "myConst"},
	}

	testCases := []struct {
		name      string
		statement js_ast.Stmt
		symbols   []ast.Symbol
		expected  []string
	}{
		{
			name: "SFunction export",
			statement: js_ast.Stmt{
				Data: &js_ast.SFunction{
					IsExport: true,
					Fn: js_ast.Fn{
						Name: &ast.LocRef{Ref: ast.Ref{InnerIndex: 0}},
					},
				},
			},
			symbols:  symbols,
			expected: []string{"myFunc"},
		},
		{
			name: "SExportClause",
			statement: js_ast.Stmt{
				Data: &js_ast.SExportClause{
					Items: []js_ast.ClauseItem{
						{Alias: "exportedA"},
						{Alias: "exportedB"},
					},
				},
			},
			symbols:  symbols,
			expected: []string{"exportedA", "exportedB"},
		},
		{
			name: "SLocal export",
			statement: js_ast.Stmt{
				Data: &js_ast.SLocal{
					IsExport: true,
					Decls: []js_ast.Decl{
						{Binding: js_ast.Binding{Data: &js_ast.BIdentifier{Ref: ast.Ref{InnerIndex: 1}}}},
					},
				},
			},
			symbols:  symbols,
			expected: []string{"myConst"},
		},
		{
			name: "SExportDefault with named function",
			statement: js_ast.Stmt{
				Data: &js_ast.SExportDefault{
					Value: js_ast.Stmt{
						Data: &js_ast.SFunction{
							Fn: js_ast.Fn{
								Name: &ast.LocRef{Ref: ast.Ref{InnerIndex: 0}},
							},
						},
					},
				},
			},
			symbols:  symbols,
			expected: []string{"myFunc"},
		},
		{
			name: "unrecognised statement type returns nil",
			statement: js_ast.Stmt{
				Data: &js_ast.SEmpty{},
			},
			symbols:  symbols,
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractExportNamesFromStmt(tc.statement, tc.symbols)
			if tc.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}
			if len(result) != len(tc.expected) {
				t.Fatalf("len(result) = %d, want %d", len(result), len(tc.expected))
			}
			for i, name := range result {
				if name != tc.expected[i] {
					t.Errorf("result[%d] = %q, want %q", i, name, tc.expected[i])
				}
			}
		})
	}
}

func TestExtractExportsFromAST(t *testing.T) {
	testCases := []struct {
		name     string
		tree     *js_ast.AST
		expected []string
	}{
		{
			name: "empty AST returns empty slice",
			tree: &js_ast.AST{
				Parts:   []js_ast.Part{},
				Symbols: []ast.Symbol{},
			},
			expected: []string{},
		},
		{
			name: "single exported function",
			tree: &js_ast.AST{
				Symbols: []ast.Symbol{
					{OriginalName: "onClick"},
				},
				Parts: []js_ast.Part{
					{
						Stmts: []js_ast.Stmt{
							{
								Data: &js_ast.SFunction{
									IsExport: true,
									Fn: js_ast.Fn{
										Name: &ast.LocRef{Ref: ast.Ref{InnerIndex: 0}},
									},
								},
							},
						},
					},
				},
			},
			expected: []string{"onClick"},
		},
		{
			name: "deduplicates repeated names",
			tree: &js_ast.AST{
				Symbols: []ast.Symbol{
					{OriginalName: "handleClick"},
				},
				Parts: []js_ast.Part{
					{
						Stmts: []js_ast.Stmt{
							{
								Data: &js_ast.SFunction{
									IsExport: true,
									Fn: js_ast.Fn{
										Name: &ast.LocRef{Ref: ast.Ref{InnerIndex: 0}},
									},
								},
							},
							{
								Data: &js_ast.SExportClause{
									Items: []js_ast.ClauseItem{
										{Alias: "handleClick"},
									},
								},
							},
						},
					},
				},
			},
			expected: []string{"handleClick"},
		},
		{
			name: "multiple exports across parts",
			tree: &js_ast.AST{
				Symbols: []ast.Symbol{
					{OriginalName: "funcA"},
					{OriginalName: "constB"},
				},
				Parts: []js_ast.Part{
					{
						Stmts: []js_ast.Stmt{
							{
								Data: &js_ast.SFunction{
									IsExport: true,
									Fn: js_ast.Fn{
										Name: &ast.LocRef{Ref: ast.Ref{InnerIndex: 0}},
									},
								},
							},
						},
					},
					{
						Stmts: []js_ast.Stmt{
							{
								Data: &js_ast.SLocal{
									IsExport: true,
									Decls: []js_ast.Decl{
										{Binding: js_ast.Binding{Data: &js_ast.BIdentifier{Ref: ast.Ref{InnerIndex: 1}}}},
									},
								},
							},
						},
					},
				},
			},
			expected: []string{"funcA", "constB"},
		},
		{
			name: "non-exported statements produce no names",
			tree: &js_ast.AST{
				Symbols: []ast.Symbol{
					{OriginalName: "internal"},
				},
				Parts: []js_ast.Part{
					{
						Stmts: []js_ast.Stmt{
							{
								Data: &js_ast.SFunction{
									IsExport: false,
									Fn: js_ast.Fn{
										Name: &ast.LocRef{Ref: ast.Ref{InnerIndex: 0}},
									},
								},
							},
						},
					},
				},
			},
			expected: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractExportsFromAST(tc.tree)
			if len(result) != len(tc.expected) {
				t.Fatalf("len(result) = %d, want %d; result=%v", len(result), len(tc.expected), result)
			}
			for i, name := range result {
				if name != tc.expected[i] {
					t.Errorf("result[%d] = %q, want %q", i, name, tc.expected[i])
				}
			}
		})
	}
}
