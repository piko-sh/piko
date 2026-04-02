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
	"testing"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func TestResolveIdentifierFromFallbackContext(t *testing.T) {
	intTypeInfo := &ast_domain.ResolvedTypeInfo{
		TypeExpression: ast.NewIdent("int"),
	}
	stringTypeInfo := &ast_domain.ResolvedTypeInfo{
		TypeExpression: ast.NewIdent("string"),
	}

	testCases := []struct {
		setupDoc   func() *document
		name       string
		identifier string
		wantType   string
		wantNil    bool
	}{
		{
			name:       "nil AnalysisMap returns nil",
			identifier: "foo",
			setupDoc: func() *document {
				return newTestDocumentBuilder().Build()
			},
			wantNil: true,
		},
		{
			name:       "empty AnalysisMap returns nil",
			identifier: "foo",
			setupDoc: func() *document {
				return newTestDocumentBuilder().
					WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{}).
					Build()
			},
			wantNil: true,
		},
		{
			name:       "context with nil Symbols is skipped",
			identifier: "foo",
			setupDoc: func() *document {
				node := newTestNode("div", 1, 1)
				return newTestDocumentBuilder().
					WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
						node: {Symbols: nil},
					}).
					Build()
			},
			wantNil: true,
		},
		{
			name:       "nil context entry is skipped",
			identifier: "foo",
			setupDoc: func() *document {
				node := newTestNode("div", 1, 1)
				return newTestDocumentBuilder().
					WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
						node: nil,
					}).
					Build()
			},
			wantNil: true,
		},
		{
			name:       "symbol not found in any context returns nil",
			identifier: "notExists",
			setupDoc: func() *document {
				node := newTestNode("div", 1, 1)
				st := annotator_domain.NewSymbolTable(nil)
				st.Define(annotator_domain.Symbol{
					Name:     "other",
					TypeInfo: intTypeInfo,
				})
				return newTestDocumentBuilder().
					WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
						node: {Symbols: st},
					}).
					Build()
			},
			wantNil: true,
		},
		{
			name:       "symbol found with nil TypeInfo returns nil",
			identifier: "noType",
			setupDoc: func() *document {
				node := newTestNode("div", 1, 1)
				st := annotator_domain.NewSymbolTable(nil)
				st.Define(annotator_domain.Symbol{
					Name:     "noType",
					TypeInfo: nil,
				})
				return newTestDocumentBuilder().
					WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
						node: {Symbols: st},
					}).
					Build()
			},
			wantNil: true,
		},
		{
			name:       "symbol found in first context returns its TypeInfo",
			identifier: "counter",
			setupDoc: func() *document {
				node := newTestNode("div", 1, 1)
				st := annotator_domain.NewSymbolTable(nil)
				st.Define(annotator_domain.Symbol{
					Name:     "counter",
					TypeInfo: intTypeInfo,
				})
				return newTestDocumentBuilder().
					WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
						node: {Symbols: st},
					}).
					Build()
			},
			wantNil:  false,
			wantType: "int",
		},
		{
			name:       "symbol found in second context when first has no match",
			identifier: "name",
			setupDoc: func() *document {
				node1 := newTestNode("div", 1, 1)
				node2 := newTestNode("span", 2, 1)

				st1 := annotator_domain.NewSymbolTable(nil)
				st1.Define(annotator_domain.Symbol{
					Name:     "other",
					TypeInfo: intTypeInfo,
				})

				st2 := annotator_domain.NewSymbolTable(nil)
				st2.Define(annotator_domain.Symbol{
					Name:     "name",
					TypeInfo: stringTypeInfo,
				})

				return newTestDocumentBuilder().
					WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
						node1: {Symbols: st1},
						node2: {Symbols: st2},
					}).
					Build()
			},
			wantNil:  false,
			wantType: "string",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document := tc.setupDoc()
			result := document.resolveIdentifierFromFallbackContext(context.Background(), tc.identifier)

			if tc.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			identifier, ok := result.TypeExpression.(*ast.Ident)
			if !ok {
				t.Fatalf("TypeExpr is %T, want *ast.Ident", result.TypeExpression)
			}
			if identifier.Name != tc.wantType {
				t.Errorf("TypeExpr name = %q, want %q", identifier.Name, tc.wantType)
			}
		})
	}
}

func TestResolveIdentifierFromNode(t *testing.T) {
	intTypeInfo := &ast_domain.ResolvedTypeInfo{
		TypeExpression: ast.NewIdent("int"),
	}

	testCases := []struct {
		setupDoc   func() (*document, *ast_domain.TemplateNode)
		name       string
		identifier string
		wantType   string
		wantNil    bool
	}{
		{
			name:       "node not in AnalysisMap returns nil",
			identifier: "foo",
			setupDoc: func() (*document, *ast_domain.TemplateNode) {
				targetNode := newTestNode("div", 1, 1)
				document := newTestDocumentBuilder().
					WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{}).
					Build()
				return document, targetNode
			},
			wantNil: true,
		},
		{
			name:       "nil analysis context returns nil",
			identifier: "foo",
			setupDoc: func() (*document, *ast_domain.TemplateNode) {
				targetNode := newTestNode("div", 1, 1)
				document := newTestDocumentBuilder().
					WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
						targetNode: nil,
					}).
					Build()
				return document, targetNode
			},
			wantNil: true,
		},
		{
			name:       "nil Symbols in context returns nil",
			identifier: "foo",
			setupDoc: func() (*document, *ast_domain.TemplateNode) {
				targetNode := newTestNode("div", 1, 1)
				document := newTestDocumentBuilder().
					WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
						targetNode: {Symbols: nil},
					}).
					Build()
				return document, targetNode
			},
			wantNil: true,
		},
		{
			name:       "symbol not found returns nil",
			identifier: "missing",
			setupDoc: func() (*document, *ast_domain.TemplateNode) {
				targetNode := newTestNode("div", 1, 1)
				st := annotator_domain.NewSymbolTable(nil)
				st.Define(annotator_domain.Symbol{
					Name:     "existing",
					TypeInfo: intTypeInfo,
				})
				document := newTestDocumentBuilder().
					WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
						targetNode: {Symbols: st},
					}).
					Build()
				return document, targetNode
			},
			wantNil: true,
		},
		{
			name:       "symbol found with nil TypeInfo returns nil",
			identifier: "noType",
			setupDoc: func() (*document, *ast_domain.TemplateNode) {
				targetNode := newTestNode("div", 1, 1)
				st := annotator_domain.NewSymbolTable(nil)
				st.Define(annotator_domain.Symbol{
					Name:     "noType",
					TypeInfo: nil,
				})
				document := newTestDocumentBuilder().
					WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
						targetNode: {Symbols: st},
					}).
					Build()
				return document, targetNode
			},
			wantNil: true,
		},
		{
			name:       "symbol found returns its TypeInfo",
			identifier: "count",
			setupDoc: func() (*document, *ast_domain.TemplateNode) {
				targetNode := newTestNode("div", 1, 1)
				st := annotator_domain.NewSymbolTable(nil)
				st.Define(annotator_domain.Symbol{
					Name:     "count",
					TypeInfo: intTypeInfo,
				})
				document := newTestDocumentBuilder().
					WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
						targetNode: {Symbols: st},
					}).
					Build()
				return document, targetNode
			},
			wantNil:  false,
			wantType: "int",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document, targetNode := tc.setupDoc()
			result := document.resolveIdentifierFromNode(context.Background(), tc.identifier, targetNode)

			if tc.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			identifier, ok := result.TypeExpression.(*ast.Ident)
			if !ok {
				t.Fatalf("TypeExpr is %T, want *ast.Ident", result.TypeExpression)
			}
			if identifier.Name != tc.wantType {
				t.Errorf("TypeExpr name = %q, want %q", identifier.Name, tc.wantType)
			}
		})
	}
}

func TestResolveFieldOnType(t *testing.T) {
	testCases := []struct {
		name      string
		setupDoc  func() *document
		baseType  *ast_domain.ResolvedTypeInfo
		fieldName string
		wantNil   bool
	}{
		{
			name: "nil TypeInspector returns nil",
			setupDoc: func() *document {
				return newTestDocumentBuilder().Build()
			},
			baseType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: ast.NewIdent("MyStruct"),
			},
			fieldName: "Field",
			wantNil:   true,
		},
		{
			name: "nil baseType returns nil",
			setupDoc: func() *document {
				return newTestDocumentBuilder().
					WithTypeInspector(&mockTypeInspector{}).
					Build()
			},
			baseType:  nil,
			fieldName: "Field",
			wantNil:   true,
		},
		{
			name: "nil TypeExpr on baseType returns nil",
			setupDoc: func() *document {
				return newTestDocumentBuilder().
					WithTypeInspector(&mockTypeInspector{}).
					Build()
			},
			baseType:  &ast_domain.ResolvedTypeInfo{TypeExpression: nil},
			fieldName: "Field",
			wantNil:   true,
		},
		{
			name: "empty AnalysisMap yields no packagePath and returns nil",
			setupDoc: func() *document {
				return newTestDocumentBuilder().
					WithTypeInspector(&mockTypeInspector{}).
					WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{}).
					Build()
			},
			baseType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: ast.NewIdent("MyStruct"),
			},
			fieldName: "Field",
			wantNil:   true,
		},
		{
			name: "all nil contexts yield no packagePath and returns nil",
			setupDoc: func() *document {
				node := newTestNode("div", 1, 1)
				return newTestDocumentBuilder().
					WithTypeInspector(&mockTypeInspector{}).
					WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
						node: nil,
					}).
					Build()
			},
			baseType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: ast.NewIdent("MyStruct"),
			},
			fieldName: "Field",
			wantNil:   true,
		},
		{
			name: "inspector returns nil fieldInfo returns nil",
			setupDoc: func() *document {
				node := newTestNode("div", 1, 1)
				return newTestDocumentBuilder().
					WithTypeInspector(&mockTypeInspector{
						FindFieldInfoFunc: func(_ context.Context, _ ast.Expr, _, _, _ string) *inspector_dto.FieldInfo {
							return nil
						},
					}).
					WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
						node: {
							CurrentGoFullPackagePath: "example.com/pkg",
							CurrentGoSourcePath:      "/src/file.go",
						},
					}).
					Build()
			},
			baseType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: ast.NewIdent("MyStruct"),
			},
			fieldName: "Missing",
			wantNil:   true,
		},
		{
			name: "inspector returns fieldInfo builds ResolvedTypeInfo",
			setupDoc: func() *document {
				node := newTestNode("div", 1, 1)
				return newTestDocumentBuilder().
					WithTypeInspector(&mockTypeInspector{
						FindFieldInfoFunc: func(_ context.Context, _ ast.Expr, _, _, _ string) *inspector_dto.FieldInfo {
							return &inspector_dto.FieldInfo{
								Type:                 ast.NewIdent("string"),
								PackageAlias:         "fmt",
								CanonicalPackagePath: "fmt",
							}
						},
					}).
					WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
						node: {
							CurrentGoFullPackagePath: "example.com/pkg",
							CurrentGoSourcePath:      "/src/file.go",
						},
					}).
					Build()
			},
			baseType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: ast.NewIdent("MyStruct"),
			},
			fieldName: "Name",
			wantNil:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document := tc.setupDoc()
			result := document.resolveFieldOnType(context.Background(), tc.baseType, tc.fieldName)

			if tc.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if result.PackageAlias != "fmt" {
				t.Errorf("PackageAlias = %q, want %q", result.PackageAlias, "fmt")
			}
			if result.CanonicalPackagePath != "fmt" {
				t.Errorf("CanonicalPackagePath = %q, want %q", result.CanonicalPackagePath, "fmt")
			}
		})
	}
}

func TestResolveExpressionFromText_EmptyString(t *testing.T) {
	document := newTestDocumentBuilder().Build()
	result := document.resolveExpressionFromText(context.Background(), "", protocol.Position{})
	if result != nil {
		t.Errorf("expected nil for empty expression, got %v", result)
	}
}

func TestResolveFirstSegment_SpecialIdentifiers(t *testing.T) {

	document := newTestDocumentBuilder().Build()

	testCases := []struct {
		name       string
		identifier string
	}{
		{name: "state returns nil when no script block", identifier: "state"},
		{name: "props returns nil when no script block", identifier: "props"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := document.resolveFirstSegment(context.Background(), tc.identifier, protocol.Position{})
			if result != nil {
				t.Errorf("expected nil for %q on empty document, got %v", tc.identifier, result)
			}
		})
	}
}
