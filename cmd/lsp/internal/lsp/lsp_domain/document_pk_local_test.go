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
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestFindTypeDefinitionInAST(t *testing.T) {
	testCases := []struct {
		name       string
		source     string
		typeName   string
		wantName   string
		wantLine   int
		wantColumn int
		wantNil    bool
	}{
		{
			name: "finds a simple struct type",
			source: `package main

type MyStruct struct {
	Field1 string
}`,
			typeName:   "MyStruct",
			wantNil:    false,
			wantName:   "MyStruct",
			wantLine:   3,
			wantColumn: 6,
		},
		{
			name: "finds a type alias",
			source: `package main

type MyAlias = int`,
			typeName:   "MyAlias",
			wantNil:    false,
			wantName:   "MyAlias",
			wantLine:   3,
			wantColumn: 6,
		},
		{
			name: "returns nil for missing type",
			source: `package main

type Existing struct {}`,
			typeName: "Missing",
			wantNil:  true,
		},
		{
			name: "finds second type in file",
			source: `package main

type First struct {}
type Second struct {}`,
			typeName:   "Second",
			wantNil:    false,
			wantName:   "Second",
			wantLine:   4,
			wantColumn: 6,
		},
		{
			name: "ignores function declarations",
			source: `package main

func MyFunc() {}`,
			typeName: "MyFunc",
			wantNil:  true,
		},
		{
			name: "finds type in grouped declaration",
			source: `package main

type (
	Alpha struct {}
	Beta  struct {}
)`,
			typeName:   "Beta",
			wantNil:    false,
			wantName:   "Beta",
			wantLine:   5,
			wantColumn: 2,
		},
		{
			name: "finds interface type",
			source: `package main

type MyInterface interface {
	DoSomething()
}`,
			typeName:   "MyInterface",
			wantNil:    false,
			wantName:   "MyInterface",
			wantLine:   3,
			wantColumn: 6,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			fileAST, err := parser.ParseFile(fset, "test.go", tc.source, parser.AllErrors)
			if err != nil {
				t.Fatalf("failed to parse source: %v", err)
			}

			got := findTypeDefinitionInAST(fileAST, fset, tc.typeName)

			if tc.wantNil {
				if got != nil {
					t.Fatalf("expected nil, got %+v", got)
				}
				return
			}

			if got == nil {
				t.Fatal("expected non-nil result, got nil")
			}
			if got.Name != tc.wantName {
				t.Errorf("Name: got %q, want %q", got.Name, tc.wantName)
			}
			if got.Line != tc.wantLine {
				t.Errorf("Line: got %d, want %d", got.Line, tc.wantLine)
			}
			if got.Column != tc.wantColumn {
				t.Errorf("Column: got %d, want %d", got.Column, tc.wantColumn)
			}
		})
	}
}

func TestFindFunctionDefinitionInAST(t *testing.T) {
	testCases := []struct {
		name         string
		source       string
		functionName string
		wantName     string
		wantLine     int
		wantColumn   int
		wantNil      bool
	}{
		{
			name: "finds a simple function",
			source: `package main

func Hello() {}`,
			functionName: "Hello",
			wantNil:      false,
			wantName:     "Hello",
			wantLine:     3,
			wantColumn:   6,
		},
		{
			name: "returns nil when function not found",
			source: `package main

func Existing() {}`,
			functionName: "Missing",
			wantNil:      true,
		},
		{
			name: "finds second function",
			source: `package main

func First() {}
func Second() {}`,
			functionName: "Second",
			wantNil:      false,
			wantName:     "Second",
			wantLine:     4,
			wantColumn:   6,
		},
		{
			name: "ignores type declarations",
			source: `package main

type MyType struct {}`,
			functionName: "MyType",
			wantNil:      true,
		},
		{
			name: "finds method (FuncDecl with receiver)",
			source: `package main

type T struct{}

func (t T) MyMethod() {}`,
			functionName: "MyMethod",
			wantNil:      false,
			wantName:     "MyMethod",
			wantLine:     5,
			wantColumn:   12,
		},
		{
			name: "finds function with parameters and return",
			source: `package main

func Calculate(a, b int) int {
	return a + b
}`,
			functionName: "Calculate",
			wantNil:      false,
			wantName:     "Calculate",
			wantLine:     3,
			wantColumn:   6,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			fileAST, err := parser.ParseFile(fset, "test.go", tc.source, parser.AllErrors)
			if err != nil {
				t.Fatalf("failed to parse source: %v", err)
			}

			got := findFunctionDefinitionInAST(fileAST, fset, tc.functionName)

			if tc.wantNil {
				if got != nil {
					t.Fatalf("expected nil, got %+v", got)
				}
				return
			}

			if got == nil {
				t.Fatal("expected non-nil result, got nil")
			}
			if got.Name != tc.wantName {
				t.Errorf("Name: got %q, want %q", got.Name, tc.wantName)
			}
			if got.Line != tc.wantLine {
				t.Errorf("Line: got %d, want %d", got.Line, tc.wantLine)
			}
			if got.Column != tc.wantColumn {
				t.Errorf("Column: got %d, want %d", got.Column, tc.wantColumn)
			}
		})
	}
}

func TestFindAnyFieldDefinition(t *testing.T) {
	testCases := []struct {
		name       string
		source     string
		fieldName  string
		wantName   string
		wantLine   int
		wantColumn int
		wantNil    bool
	}{
		{
			name: "finds field in single struct",
			source: `package main

type State struct {
	Title string
	Count int
}`,
			fieldName:  "Title",
			wantNil:    false,
			wantName:   "Title",
			wantLine:   4,
			wantColumn: 2,
		},
		{
			name: "finds field in second struct",
			source: `package main

type First struct {
	Alpha string
}

type Second struct {
	Beta int
}`,
			fieldName:  "Beta",
			wantNil:    false,
			wantName:   "Beta",
			wantLine:   8,
			wantColumn: 2,
		},
		{
			name: "returns nil when field not found",
			source: `package main

type State struct {
	Title string
}`,
			fieldName: "Missing",
			wantNil:   true,
		},
		{
			name: "returns nil for non-struct types",
			source: `package main

type MyAlias = string`,
			fieldName: "SomeField",
			wantNil:   true,
		},
		{
			name:      "returns nil when no type declarations exist",
			source:    `package main`,
			fieldName: "Field",
			wantNil:   true,
		},
		{
			name: "finds field across grouped type declarations",
			source: `package main

type (
	Alpha struct {
		Name string
	}
	Beta struct {
		Age int
	}
)`,
			fieldName:  "Age",
			wantNil:    false,
			wantName:   "Age",
			wantLine:   8,
			wantColumn: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			fileAST, err := parser.ParseFile(fset, "test.go", tc.source, parser.AllErrors)
			if err != nil {
				t.Fatalf("failed to parse source: %v", err)
			}

			got := findAnyFieldDefinition(fileAST, fset, tc.fieldName)

			if tc.wantNil {
				if got != nil {
					t.Fatalf("expected nil, got %+v", got)
				}
				return
			}

			if got == nil {
				t.Fatal("expected non-nil result, got nil")
			}
			if got.Name != tc.wantName {
				t.Errorf("Name: got %q, want %q", got.Name, tc.wantName)
			}
			if got.Line != tc.wantLine {
				t.Errorf("Line: got %d, want %d", got.Line, tc.wantLine)
			}
			if got.Column != tc.wantColumn {
				t.Errorf("Column: got %d, want %d", got.Column, tc.wantColumn)
			}
		})
	}
}

func TestFindFieldDefinitionInType(t *testing.T) {
	testCases := []struct {
		name       string
		source     string
		typeName   string
		fieldName  string
		wantName   string
		wantLine   int
		wantColumn int
		wantNil    bool
	}{
		{
			name: "finds field in struct",
			source: `package main

type State struct {
	Title string
	Count int
}`,
			typeName:   "State",
			fieldName:  "Count",
			wantNil:    false,
			wantName:   "Count",
			wantLine:   5,
			wantColumn: 2,
		},
		{
			name: "returns nil when field not found",
			source: `package main

type State struct {
	Title string
}`,
			typeName:  "State",
			fieldName: "Missing",
			wantNil:   true,
		},
		{
			name: "returns nil for non-struct type",
			source: `package main

type MyInterface interface {
	DoSomething()
}`,
			typeName:  "MyInterface",
			fieldName: "DoSomething",
			wantNil:   true,
		},
		{
			name: "finds first of multiple fields on same line",
			source: `package main

type Pair struct {
	X, Y int
}`,
			typeName:   "Pair",
			fieldName:  "X",
			wantNil:    false,
			wantName:   "X",
			wantLine:   4,
			wantColumn: 2,
		},
		{
			name: "finds second of multiple fields on same line",
			source: `package main

type Pair struct {
	X, Y int
}`,
			typeName:   "Pair",
			fieldName:  "Y",
			wantNil:    false,
			wantName:   "Y",
			wantLine:   4,
			wantColumn: 5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			fileAST, err := parser.ParseFile(fset, "test.go", tc.source, parser.AllErrors)
			if err != nil {
				t.Fatalf("failed to parse source: %v", err)
			}

			typeSpec := findTypeSpecByName(fileAST, tc.typeName)
			if typeSpec == nil {
				t.Fatalf("type %q not found in source", tc.typeName)
			}

			got := findFieldDefinitionInType(typeSpec, fset, tc.fieldName)

			if tc.wantNil {
				if got != nil {
					t.Fatalf("expected nil, got %+v", got)
				}
				return
			}

			if got == nil {
				t.Fatal("expected non-nil result, got nil")
			}
			if got.Name != tc.wantName {
				t.Errorf("Name: got %q, want %q", got.Name, tc.wantName)
			}
			if got.Line != tc.wantLine {
				t.Errorf("Line: got %d, want %d", got.Line, tc.wantLine)
			}
			if got.Column != tc.wantColumn {
				t.Errorf("Column: got %d, want %d", got.Column, tc.wantColumn)
			}
		})
	}
}

func TestFindRenderReturnType(t *testing.T) {
	testCases := []struct {
		name   string
		source string
		want   string
	}{
		{
			name: "finds Render return type",
			source: `package main

func Render() Response {
	return Response{}
}`,
			want: "Response",
		},
		{
			name: "returns empty when no Render function exists",
			source: `package main

func Other() string { return "" }`,
			want: "",
		},
		{
			name: "returns empty when Render has no return values",
			source: `package main

func Render() {}`,
			want: "",
		},
		{
			name: "ignores non-ident return type",
			source: `package main

func Render() *Response {
	return nil
}`,
			want: "",
		},
		{
			name: "finds Render among other functions",
			source: `package main

func Setup() {}

func Render() State {
	return State{}
}

func Cleanup() {}`,
			want: "State",
		},
		{
			name: "returns first return type when multiple exist",
			source: `package main

func Render() (Model, error) {
	return Model{}, nil
}`,
			want: "Model",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fileAST := parseGoSource(t, tc.source)
			got := findRenderReturnType(fileAST)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFindRenderPropsType(t *testing.T) {
	testCases := []struct {
		name             string
		source           string
		wantPackageAlias string
		wantNil          bool
	}{
		{
			name: "finds props type as second parameter (simple ident)",
			source: `package main

func Render(ctx Context, props MyProps) {}`,
			wantNil:          false,
			wantPackageAlias: "",
		},
		{
			name: "finds props type as second parameter (selector expr)",
			source: `package main

func Render(ctx Context, props pkg.MyProps) {}`,
			wantNil:          false,
			wantPackageAlias: "pkg",
		},
		{
			name: "returns nil when Render has fewer than two params",
			source: `package main

func Render(ctx Context) {}`,
			wantNil: true,
		},
		{
			name: "returns nil when no Render function",
			source: `package main

func Other(a, b int) {}`,
			wantNil: true,
		},
		{
			name: "unwraps pointer to get underlying type",
			source: `package main

func Render(ctx Context, props *MyProps) {}`,
			wantNil:          false,
			wantPackageAlias: "",
		},
		{
			name: "unwraps pointer to selector expr",
			source: `package main

func Render(ctx Context, props *pkg.MyProps) {}`,
			wantNil:          false,
			wantPackageAlias: "pkg",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fileAST := parseGoSource(t, tc.source)
			got := findRenderPropsType(fileAST)

			if tc.wantNil {
				if got != nil {
					t.Fatalf("expected nil, got %+v", got)
				}
				return
			}

			if got == nil {
				t.Fatal("expected non-nil result, got nil")
			}
			if got.PackageAlias != tc.wantPackageAlias {
				t.Errorf("PackageAlias: got %q, want %q", got.PackageAlias, tc.wantPackageAlias)
			}
		})
	}
}

func TestBuildResolvedTypeFromExpr(t *testing.T) {
	testCases := []struct {
		expression       ast.Expr
		name             string
		wantPackageAlias string
		wantNil          bool
	}{
		{
			name:       "nil input returns nil",
			expression: nil,
			wantNil:    true,
		},
		{
			name:             "ast.Ident returns empty pkg alias",
			expression:       &ast.Ident{Name: "MyType"},
			wantNil:          false,
			wantPackageAlias: "",
		},
		{
			name: "ast.SelectorExpr returns package alias",
			expression: &ast.SelectorExpr{
				X:   &ast.Ident{Name: "pkg"},
				Sel: &ast.Ident{Name: "Type"},
			},
			wantNil:          false,
			wantPackageAlias: "pkg",
		},
		{
			name: "ast.StarExpr unwraps to Ident",
			expression: &ast.StarExpr{
				X: &ast.Ident{Name: "MyType"},
			},
			wantNil:          false,
			wantPackageAlias: "",
		},
		{
			name: "ast.StarExpr unwraps to SelectorExpr",
			expression: &ast.StarExpr{
				X: &ast.SelectorExpr{
					X:   &ast.Ident{Name: "pkg"},
					Sel: &ast.Ident{Name: "Type"},
				},
			},
			wantNil:          false,
			wantPackageAlias: "pkg",
		},
		{
			name: "double pointer unwraps fully",
			expression: &ast.StarExpr{
				X: &ast.StarExpr{
					X: &ast.Ident{Name: "DeepType"},
				},
			},
			wantNil:          false,
			wantPackageAlias: "",
		},
		{
			name:             "default case returns resolved info with empty alias",
			expression:       &ast.ArrayType{Elt: &ast.Ident{Name: "int"}},
			wantNil:          false,
			wantPackageAlias: "",
		},
		{
			name: "SelectorExpr with non-ident X returns empty alias",
			expression: &ast.SelectorExpr{
				X:   &ast.ParenExpr{X: &ast.Ident{Name: "pkg"}},
				Sel: &ast.Ident{Name: "Type"},
			},
			wantNil:          false,
			wantPackageAlias: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildResolvedTypeFromExpr(tc.expression)

			if tc.wantNil {
				if got != nil {
					t.Fatalf("expected nil, got %+v", got)
				}
				return
			}

			if got == nil {
				t.Fatal("expected non-nil result, got nil")
			}
			if got.PackageAlias != tc.wantPackageAlias {
				t.Errorf("PackageAlias: got %q, want %q", got.PackageAlias, tc.wantPackageAlias)
			}
			if got.TypeExpression == nil {
				t.Error("TypeExpr should not be nil")
			}
		})
	}
}

func TestFormatTypeExpr(t *testing.T) {
	testCases := []struct {
		name       string
		expression ast.Expr
		want       string
	}{
		{
			name:       "ident",
			expression: &ast.Ident{Name: "string"},
			want:       "string",
		},
		{
			name: "selector expr",
			expression: &ast.SelectorExpr{
				X:   &ast.Ident{Name: "time"},
				Sel: &ast.Ident{Name: "Duration"},
			},
			want: "time.Duration",
		},
		{
			name: "star expr",
			expression: &ast.StarExpr{
				X: &ast.Ident{Name: "int"},
			},
			want: "*int",
		},
		{
			name: "slice type (nil length)",
			expression: &ast.ArrayType{
				Len: nil,
				Elt: &ast.Ident{Name: "byte"},
			},
			want: "[]byte",
		},
		{
			name: "array type (with length)",
			expression: &ast.ArrayType{
				Len: &ast.BasicLit{},
				Elt: &ast.Ident{Name: "int"},
			},
			want: "[...]int",
		},
		{
			name: "map type",
			expression: &ast.MapType{
				Key:   &ast.Ident{Name: "string"},
				Value: &ast.Ident{Name: "int"},
			},
			want: "map[string]int",
		},
		{
			name: "nested pointer to selector",
			expression: &ast.StarExpr{
				X: &ast.SelectorExpr{
					X:   &ast.Ident{Name: "pkg"},
					Sel: &ast.Ident{Name: "Type"},
				},
			},
			want: "*pkg.Type",
		},
		{
			name: "slice of pointers",
			expression: &ast.ArrayType{
				Len: nil,
				Elt: &ast.StarExpr{
					X: &ast.Ident{Name: "Item"},
				},
			},
			want: "[]*Item",
		},
		{
			name: "map with complex value",
			expression: &ast.MapType{
				Key: &ast.Ident{Name: "string"},
				Value: &ast.ArrayType{
					Len: nil,
					Elt: &ast.Ident{Name: "int"},
				},
			},
			want: "map[string][]int",
		},
		{
			name:       "unknown type returns question mark",
			expression: &ast.FuncType{},
			want:       "?",
		},
		{
			name: "selector with non-ident X returns question mark",
			expression: &ast.SelectorExpr{
				X:   &ast.ParenExpr{X: &ast.Ident{Name: "a"}},
				Sel: &ast.Ident{Name: "B"},
			},
			want: "?",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatTypeExpr(tc.expression)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFindTypeSpecByName(t *testing.T) {
	testCases := []struct {
		name     string
		source   string
		typeName string
		wantName string
		wantNil  bool
	}{
		{
			name: "finds type by name",
			source: `package main

type MyStruct struct {
	Field string
}`,
			typeName: "MyStruct",
			wantNil:  false,
			wantName: "MyStruct",
		},
		{
			name: "returns nil when type not found",
			source: `package main

type Other struct {}`,
			typeName: "Missing",
			wantNil:  true,
		},
		{
			name:     "returns nil in empty file",
			source:   `package main`,
			typeName: "Anything",
			wantNil:  true,
		},
		{
			name: "finds type in grouped declaration",
			source: `package main

type (
	Alpha struct {}
	Beta  struct {}
)`,
			typeName: "Alpha",
			wantNil:  false,
			wantName: "Alpha",
		},
		{
			name: "ignores function declarations",
			source: `package main

func MyFunc() {}`,
			typeName: "MyFunc",
			wantNil:  true,
		},
		{
			name: "ignores var declarations",
			source: `package main

var MyVar = 42`,
			typeName: "MyVar",
			wantNil:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fileAST := parseGoSource(t, tc.source)
			got := findTypeSpecByName(fileAST, tc.typeName)

			if tc.wantNil {
				if got != nil {
					t.Fatalf("expected nil, got type spec with name %q", got.Name.Name)
				}
				return
			}

			if got == nil {
				t.Fatal("expected non-nil result, got nil")
			}
			if got.Name.Name != tc.wantName {
				t.Errorf("Name: got %q, want %q", got.Name.Name, tc.wantName)
			}
		})
	}
}

func TestFormatStructPreview(t *testing.T) {
	testCases := []struct {
		name      string
		source    string
		typeName  string
		want      string
		maxFields int
	}{
		{
			name: "all fields shown when under limit",
			source: `package main

type State struct {
	Title string
	Count int
}`,
			typeName:  "State",
			maxFields: 5,
			want: "type State struct {\n" +
				"    Title                string\n" +
				"    Count                int\n" +
				"}",
		},
		{
			name: "truncated fields show remaining count",
			source: `package main

type State struct {
	A string
	B int
	C bool
	D float64
}`,
			typeName:  "State",
			maxFields: 2,
			want: "type State struct {\n" +
				"    A                    string\n" +
				"    B                    int\n" +
				"    ... (2 more fields)\n" +
				"}",
		},
		{
			name: "zero max fields shows only truncation message",
			source: `package main

type State struct {
	A string
	B int
}`,
			typeName:  "State",
			maxFields: 0,
			want: "type State struct {\n" +
				"    ... (2 more fields)\n" +
				"}",
		},
		{
			name: "single field struct",
			source: `package main

type Single struct {
	Value string
}`,
			typeName:  "Single",
			maxFields: 10,
			want: "type Single struct {\n" +
				"    Value                string\n" +
				"}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fileAST := parseGoSource(t, tc.source)
			typeSpec := findTypeSpecByName(fileAST, tc.typeName)
			if typeSpec == nil {
				t.Fatalf("type %q not found", tc.typeName)
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				t.Fatalf("type %q is not a struct", tc.typeName)
			}

			got := formatStructPreview(tc.typeName, structType, tc.maxFields)
			if got != tc.want {
				t.Errorf("got:\n%s\n\nwant:\n%s", got, tc.want)
			}
		})
	}
}

func TestCountStructFields(t *testing.T) {
	testCases := []struct {
		name     string
		source   string
		typeName string
		want     int
	}{
		{
			name: "counts all named fields",
			source: `package main

type State struct {
	Title string
	Count int
	Active bool
}`,
			typeName: "State",
			want:     3,
		},
		{
			name: "counts multi-name field declarations",
			source: `package main

type Pair struct {
	X, Y int
}`,
			typeName: "Pair",
			want:     2,
		},
		{
			name: "empty struct returns zero",
			source: `package main

type Empty struct {}`,
			typeName: "Empty",
			want:     0,
		},
		{
			name: "embedded fields have no names",
			source: `package main

type Base struct {
	Name string
}

type Child struct {
	Base
	Age int
}`,
			typeName: "Child",
			want:     1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fileAST := parseGoSource(t, tc.source)
			typeSpec := findTypeSpecByName(fileAST, tc.typeName)
			if typeSpec == nil {
				t.Fatalf("type %q not found", tc.typeName)
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				t.Fatalf("type %q is not a struct", tc.typeName)
			}

			got := countStructFields(structType)
			if got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestFormatLocalFieldLine(t *testing.T) {
	testCases := []struct {
		name      string
		source    string
		typeName  string
		fieldName string
		want      string
	}{
		{
			name: "simple field with short name",
			source: `package main

type State struct {
	Title string
}`,
			typeName:  "State",
			fieldName: "Title",
			want:      "Title                string",
		},
		{
			name: "field with tag",
			source: `package main

type State struct {
	Name string ` + "`" + `json:"name"` + "`" + `
}`,
			typeName:  "State",
			fieldName: "Name",
			want:      "Name                 string `json:\"name\"`",
		},
		{
			name: "field with long name exceeding pad width",
			source: `package main

type State struct {
	VeryLongFieldNameThatExceedsPadding string
}`,
			typeName:  "State",
			fieldName: "VeryLongFieldNameThatExceedsPadding",
			want:      "VeryLongFieldNameThatExceedsPadding string",
		},
		{
			name: "field with exactly pad width name",
			source: `package main

type State struct {
	TwentyCharFieldNames string
}`,
			typeName:  "State",
			fieldName: "TwentyCharFieldNames",

			want: "TwentyCharFieldNames string",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			fileAST, err := parser.ParseFile(fset, "test.go", tc.source, parser.AllErrors)
			if err != nil {
				t.Fatalf("failed to parse source: %v", err)
			}

			typeSpec := findTypeSpecByName(fileAST, tc.typeName)
			if typeSpec == nil {
				t.Fatalf("type %q not found", tc.typeName)
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				t.Fatalf("type %q is not a struct", tc.typeName)
			}

			var targetField *ast.Field
			for _, field := range structType.Fields.List {
				for _, fieldIdent := range field.Names {
					if fieldIdent.Name == tc.fieldName {
						targetField = field
					}
				}
			}
			if targetField == nil {
				t.Fatalf("field %q not found in type %q", tc.fieldName, tc.typeName)
			}

			got := formatLocalFieldLine(targetField, tc.fieldName)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestGetLocalTypePreview(t *testing.T) {
	testCases := []struct {
		name      string
		source    string
		typeName  string
		wantHas   string
		maxFields int
		wantEmpty bool
	}{
		{
			name: "returns preview for existing struct type",
			source: `package main

type State struct {
	Title string
	Count int
}`,
			typeName:  "State",
			maxFields: 5,
			wantEmpty: false,
			wantHas:   "type State struct {",
		},
		{
			name: "returns empty for missing type",
			source: `package main

type Other struct {}`,
			typeName:  "Missing",
			maxFields: 5,
			wantEmpty: true,
		},
		{
			name: "returns empty for non-struct type",
			source: `package main

type MyAlias = string`,
			typeName:  "MyAlias",
			maxFields: 5,
			wantEmpty: true,
		},
		{
			name: "returns empty for interface type",
			source: `package main

type MyInterface interface {
	DoSomething()
}`,
			typeName:  "MyInterface",
			maxFields: 5,
			wantEmpty: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			fileAST, err := parser.ParseFile(fset, "test.go", tc.source, parser.AllErrors)
			if err != nil {
				t.Fatalf("failed to parse source: %v", err)
			}

			got := getLocalTypePreview(fileAST, fset, tc.typeName, tc.maxFields)

			if tc.wantEmpty {
				if got != "" {
					t.Fatalf("expected empty string, got %q", got)
				}
				return
			}

			if got == "" {
				t.Fatal("expected non-empty string, got empty")
			}
			if tc.wantHas != "" && !strings.Contains(got, tc.wantHas) {
				t.Errorf("expected result to contain %q, got %q", tc.wantHas, got)
			}
		})
	}
}

func TestBuildProtocolLocation(t *testing.T) {
	testCases := []struct {
		name          string
		uri           protocol.DocumentURI
		symbolName    string
		line          int
		column        int
		wantStartLine uint32
		wantStartChar uint32
		wantEndLine   uint32
		wantEndChar   uint32
	}{
		{
			name:          "converts 1-based to 0-based positions",
			uri:           protocol.DocumentURI("file:///test.pk"),
			symbolName:    "MyType",
			line:          10,
			column:        5,
			wantStartLine: 9,
			wantStartChar: 4,
			wantEndLine:   9,
			wantEndChar:   10,
		},
		{
			name:          "line 1 column 1 becomes 0 0",
			uri:           protocol.DocumentURI("file:///test.pk"),
			symbolName:    "X",
			line:          1,
			column:        1,
			wantStartLine: 0,
			wantStartChar: 0,
			wantEndLine:   0,
			wantEndChar:   1,
		},
		{
			name:          "empty symbol name produces zero-width range",
			uri:           protocol.DocumentURI("file:///test.pk"),
			symbolName:    "",
			line:          5,
			column:        3,
			wantStartLine: 4,
			wantStartChar: 2,
			wantEndLine:   4,
			wantEndChar:   2,
		},
		{
			name:          "long symbol name extends end character",
			uri:           protocol.DocumentURI("file:///some/path.pk"),
			symbolName:    "VeryLongSymbolName",
			line:          20,
			column:        10,
			wantStartLine: 19,
			wantStartChar: 9,
			wantEndLine:   19,
			wantEndChar:   27,
		},
		{
			name:          "preserves document URI",
			uri:           protocol.DocumentURI("file:///my/component.pk"),
			symbolName:    "Foo",
			line:          3,
			column:        6,
			wantStartLine: 2,
			wantStartChar: 5,
			wantEndLine:   2,
			wantEndChar:   8,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildProtocolLocation(tc.uri, tc.symbolName, tc.line, tc.column)

			if got == nil {
				t.Fatal("expected non-nil result, got nil")
			}

			if got.URI != tc.uri {
				t.Errorf("URI: got %q, want %q", got.URI, tc.uri)
			}
			if got.Range.Start.Line != tc.wantStartLine {
				t.Errorf("Start.Line: got %d, want %d", got.Range.Start.Line, tc.wantStartLine)
			}
			if got.Range.Start.Character != tc.wantStartChar {
				t.Errorf("Start.Character: got %d, want %d", got.Range.Start.Character, tc.wantStartChar)
			}
			if got.Range.End.Line != tc.wantEndLine {
				t.Errorf("End.Line: got %d, want %d", got.Range.End.Line, tc.wantEndLine)
			}
			if got.Range.End.Character != tc.wantEndChar {
				t.Errorf("End.Character: got %d, want %d", got.Range.End.Character, tc.wantEndChar)
			}
		})
	}
}

func TestBuildResolvedTypeFromExprPreservesTypeExpr(t *testing.T) {
	identifier := &ast.Ident{Name: "SomeType"}
	got := buildResolvedTypeFromExpr(identifier)

	if got == nil {
		t.Fatal("expected non-nil result")
	}

	resultIdent, ok := got.TypeExpression.(*ast.Ident)
	if !ok {
		t.Fatal("expected TypeExpr to be *ast.Ident")
	}
	if resultIdent.Name != "SomeType" {
		t.Errorf("TypeExpr name: got %q, want %q", resultIdent.Name, "SomeType")
	}

	if got.CanonicalPackagePath != "" {
		t.Errorf("CanonicalPackagePath: got %q, want empty", got.CanonicalPackagePath)
	}
}

func TestParseOriginalScriptBlock(t *testing.T) {
	testCases := []struct {
		document  *document
		wantCheck func(t *testing.T, result *scriptParseResult)
		name      string
		wantErr   bool
		wantNil   bool
	}{
		{
			name: "empty content returns error",
			document: newTestDocumentBuilder().
				WithURI("file:///project/page.pk").
				WithContent("").
				Build(),
			wantErr: true,
		},
		{
			name: "valid SFC with Go script block returns parsed result",
			document: func() *document {
				content := `<script lang="go">
package main

type State struct {
	Title string
}

func Render() State {
	return State{}
}
</script>
<template>
<div>{{ state.Title }}</div>
</template>`
				return newTestDocumentBuilder().
					WithURI("file:///project/page.pk").
					WithContent(content).
					Build()
			}(),
			wantErr: false,
			wantNil: false,
			wantCheck: func(t *testing.T, result *scriptParseResult) {
				t.Helper()
				if result.AST == nil {
					t.Error("expected non-nil AST")
				}
				if result.Fset == nil {
					t.Error("expected non-nil Fset")
				}
				if result.Offset == nil {
					t.Error("expected non-nil Offset")
				}
			},
		},
		{
			name: "SFC without Go script block returns error",
			document: func() *document {
				content := `<template>
<div>hello</div>
</template>`
				return newTestDocumentBuilder().
					WithURI("file:///project/page.pk").
					WithContent(content).
					Build()
			}(),
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.document.parseOriginalScriptBlock()

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.wantNil {
				if result != nil {
					t.Errorf("expected nil result, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result, got nil")
			}

			if tc.wantCheck != nil {
				tc.wantCheck(t, result)
			}
		})
	}
}

func TestFindLocalSymbolDefinition(t *testing.T) {

	content := `<script lang="go">
package main

type State struct {
	Title string
	Count int
}

func Helper() {}
</script>
<template>
<div>hello</div>
</template>`

	document := newTestDocumentBuilder().
		WithURI("file:///project/page.pk").
		WithContent(content).
		Build()

	testCases := []struct {
		name       string
		symbolName string
		wantNil    bool
	}{
		{
			name:       "finds type definition",
			symbolName: "State",
			wantNil:    false,
		},
		{
			name:       "finds field definition",
			symbolName: "Title",
			wantNil:    false,
		},
		{
			name:       "finds function definition",
			symbolName: "Helper",
			wantNil:    false,
		},
		{
			name:       "returns nil for non-existent symbol",
			symbolName: "NonExistent",
			wantNil:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := document.findLocalSymbolDefinition(context.Background(), tc.symbolName)

			if tc.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result, got nil")
			}

			if result.URI != document.URI {
				t.Errorf("URI = %q, want %q", result.URI, document.URI)
			}
		})
	}
}

func TestFindLocalSymbolDefinition_NonPKFile(t *testing.T) {
	document := newTestDocumentBuilder().
		WithURI("file:///project/page.go").
		WithContent("package main").
		Build()

	result := document.findLocalSymbolDefinition(context.Background(), "Anything")
	if result != nil {
		t.Errorf("expected nil for non-.pk file, got %+v", result)
	}
}

func TestFindStateTypeDefinition(t *testing.T) {
	testCases := []struct {
		name    string
		content string
		uri     protocol.DocumentURI
		wantNil bool
	}{
		{
			name: "finds state type via Render return type",
			content: `<script lang="go">
package main

type PageState struct {
	Title string
}

func Render() PageState {
	return PageState{}
}
</script>
<template>
<div>hello</div>
</template>`,
			uri:     "file:///project/page.pk",
			wantNil: false,
		},
		{
			name: "returns nil when Render has no return type",
			content: `<script lang="go">
package main

func Render() {}
</script>
<template>
<div>hello</div>
</template>`,
			uri:     "file:///project/page.pk",
			wantNil: true,
		},
		{
			name:    "returns nil for non-pk file",
			content: "package main",
			uri:     "file:///project/page.go",
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document := newTestDocumentBuilder().
				WithURI(tc.uri).
				WithContent(tc.content).
				Build()

			result := document.findStateTypeDefinition(context.Background())

			if tc.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result, got nil")
			}

			if result.URI != tc.uri {
				t.Errorf("URI = %q, want %q", result.URI, tc.uri)
			}
		})
	}
}

func TestFindLocalPKTypeDefinition(t *testing.T) {
	content := `<script lang="go">
package main

type Response struct {
	Message string
}
</script>
<template>
<div>hello</div>
</template>`

	document := newTestDocumentBuilder().
		WithURI("file:///project/page.pk").
		WithContent(content).
		Build()

	testCases := []struct {
		name     string
		typeName string
		wantNil  bool
	}{
		{
			name:     "finds existing type",
			typeName: "Response",
			wantNil:  false,
		},
		{
			name:     "returns nil for non-existent type",
			typeName: "Missing",
			wantNil:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := document.findLocalPKTypeDefinition(context.Background(), tc.typeName)

			if tc.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result, got nil")
			}
		})
	}
}

var _ *ast_domain.ResolvedTypeInfo
