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
	"fmt"
	goast "go/ast"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

func TestKeyAnalyser_AnalyseAndSetEffectiveKey(t *testing.T) {
	t.Run("NodeWithoutPFor", func(t *testing.T) {
		analyser := newKeyAnalyser(nil)
		ctx := createTestAnalysisContext()
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
		}

		analyser.AnalyseAndSetEffectiveKey(node, nil, ctx, 0)

		if node.GoAnnotations != nil && node.GoAnnotations.EffectiveKeyExpression != nil {
			t.Error("Expected no EffectiveKeyExpression for node without p-for")
		}
	})

	t.Run("NodeWithExplicitPKey", func(t *testing.T) {
		analyser := newKeyAnalyser(nil)
		ctx := createTestAnalysisContext()
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			DirFor: &ast_domain.Directive{
				Type:          ast_domain.DirectiveFor,
				RawExpression: "item in items",
			},
			DirKey: &ast_domain.Directive{
				Type:          ast_domain.DirectiveKey,
				RawExpression: "item.id",
			},
		}

		analyser.AnalyseAndSetEffectiveKey(node, nil, ctx, 0)

		if node.GoAnnotations != nil && node.GoAnnotations.EffectiveKeyExpression != nil {
			t.Error("Expected no EffectiveKeyExpression when p-key is provided")
		}
	})

	t.Run("InvalidForExpression", func(t *testing.T) {
		analyser := newKeyAnalyser(nil)
		ctx := createTestAnalysisContext()
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			DirFor: &ast_domain.Directive{
				Type:          ast_domain.DirectiveFor,
				RawExpression: "invalid",
				Expression: &ast_domain.Identifier{
					Name: "invalid",
				},
			},
		}

		analyser.AnalyseAndSetEffectiveKey(node, nil, ctx, 0)
	})

	t.Run("ForExpressionWithoutAnnotation", func(t *testing.T) {
		analyser := newKeyAnalyser(nil)
		ctx := createTestAnalysisContext()
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			DirFor: &ast_domain.Directive{
				Type:          ast_domain.DirectiveFor,
				RawExpression: "item in items",
				Expression: &ast_domain.ForInExpression{
					ItemVariable: &ast_domain.Identifier{Name: "item"},
					Collection:   &ast_domain.Identifier{Name: "items"},
				},
			},
			Key: &ast_domain.StringLiteral{Value: "r.0"},
		}

		analyser.AnalyseAndSetEffectiveKey(node, nil, ctx, 0)

		if node.GoAnnotations != nil && node.GoAnnotations.EffectiveKeyExpression != nil {
			t.Error("Expected no EffectiveKeyExpression when collection has no annotation")
		}
	})

	t.Run("ForExpressionWithoutResolvedType", func(t *testing.T) {
		analyser := newKeyAnalyser(nil)
		ctx := createTestAnalysisContext()
		collectionExpr := &ast_domain.Identifier{Name: "items"}
		collectionExpr.GoAnnotations = &ast_domain.GoGeneratorAnnotation{EffectiveKeyExpression: nil, DynamicCollectionInfo: nil, StaticCollectionLiteral: nil, ParentTypeName: nil, BaseCodeGenVarName: nil, GeneratedSourcePath: nil, DynamicAttributeOrigins: nil, ResolvedType: nil, Symbol: nil, PartialInfo: nil, PropDataSource: nil, OriginalSourcePath: nil, OriginalPackageAlias: nil, FieldTag: nil, SourceInvocationKey: nil, StaticCollectionData: nil, Srcset: nil, Stringability: 0, IsStatic: false, NeedsCSRF: false, NeedsRuntimeSafetyCheck: false, IsStructurallyStatic: false, IsPointerToStringable: false, IsCollectionCall: false, IsHybridCollection: false, IsMapAccess: false}
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			DirFor: &ast_domain.Directive{
				Type:          ast_domain.DirectiveFor,
				RawExpression: "item in items",
				Expression: &ast_domain.ForInExpression{
					ItemVariable: &ast_domain.Identifier{Name: "item"},
					Collection:   collectionExpr,
				},
			},
			Key: &ast_domain.StringLiteral{Value: "r.0"},
		}

		analyser.AnalyseAndSetEffectiveKey(node, nil, ctx, 0)

		if node.GoAnnotations != nil && node.GoAnnotations.EffectiveKeyExpression != nil {
			t.Error("Expected no EffectiveKeyExpression when collection has no resolved type")
		}
	})
}

func TestKeyAnalyser_extractChildPositionFromKey(t *testing.T) {
	tests := []struct {
		keyExpr         ast_domain.Expression
		name            string
		expectedLiteral string
		expectedParts   int
	}{
		{
			name:          "nil expression",
			keyExpr:       nil,
			expectedParts: 0,
		},
		{
			name: "template literal with colon pattern",
			keyExpr: &ast_domain.TemplateLiteral{
				Parts: []ast_domain.TemplateLiteralPart{
					{IsLiteral: true, Literal: "r.0:0:"},
					{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "row"}},
					{IsLiteral: true, Literal: ":0."},
					{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "cell"}},
				},
			},
			expectedParts:   1,
			expectedLiteral: ":0.",
		},
		{
			name: "template literal with single colon",
			keyExpr: &ast_domain.TemplateLiteral{
				Parts: []ast_domain.TemplateLiteralPart{
					{IsLiteral: true, Literal: "prefix."},
					{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "item"}},
					{IsLiteral: true, Literal: ":1."},
				},
			},
			expectedParts:   1,
			expectedLiteral: ":1.",
		},
		{
			name: "template literal with multiple colons - extracts last",
			keyExpr: &ast_domain.TemplateLiteral{
				Parts: []ast_domain.TemplateLiteralPart{
					{IsLiteral: true, Literal: "r:0:"},
					{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "x"}},
					{IsLiteral: true, Literal: ":2:3."},
				},
			},
			expectedParts:   1,
			expectedLiteral: ":3.",
		},
		{
			name: "template literal with no colon",
			keyExpr: &ast_domain.TemplateLiteral{
				Parts: []ast_domain.TemplateLiteralPart{
					{IsLiteral: true, Literal: "prefix."},
					{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "item"}},
					{IsLiteral: true, Literal: ".suffix"},
				},
			},
			expectedParts: 0,
		},
		{
			name:          "string literal - not a template",
			keyExpr:       &ast_domain.StringLiteral{Value: "r.0:0:"},
			expectedParts: 0,
		},
		{
			name:          "identifier - not a template",
			keyExpr:       &ast_domain.Identifier{Name: "item"},
			expectedParts: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyser := newKeyAnalyser(nil)
			parts := analyser.extractChildPositionFromKey(tt.keyExpr)

			if len(parts) != tt.expectedParts {
				t.Errorf("Expected %d parts, got %d", tt.expectedParts, len(parts))
				return
			}
			if tt.expectedParts > 0 && parts[0].IsLiteral {
				if parts[0].Literal != tt.expectedLiteral {
					t.Errorf("Expected literal '%s', got '%s'", tt.expectedLiteral, parts[0].Literal)
				}
			}
		})
	}
}

func TestKeyAnalyser_extractPathPartsForNestedLoop(t *testing.T) {
	tests := []struct {
		keyExpr       ast_domain.Expression
		validateParts func(*testing.T, []ast_domain.TemplateLiteralPart)
		name          string
		expectedParts int
	}{
		{
			name:          "nil expression",
			keyExpr:       nil,
			expectedParts: 0,
			validateParts: func(t *testing.T, parts []ast_domain.TemplateLiteralPart) {
				if parts != nil {
					t.Error("Expected nil parts for nil expression")
				}
			},
		},
		{
			name:          "string literal",
			keyExpr:       &ast_domain.StringLiteral{Value: "r.0:0:"},
			expectedParts: 1,
			validateParts: func(t *testing.T, parts []ast_domain.TemplateLiteralPart) {
				if !parts[0].IsLiteral || parts[0].Literal != "r.0:0:" {
					t.Errorf("Expected literal 'r.0:0:', got %v", parts[0])
				}
			},
		},
		{
			name: "template literal with mixed parts",
			keyExpr: &ast_domain.TemplateLiteral{
				Parts: []ast_domain.TemplateLiteralPart{
					{IsLiteral: true, Literal: "r.0:"},
					{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "__pikoLoopIdx"}},
				},
			},
			expectedParts: 2,
			validateParts: func(t *testing.T, parts []ast_domain.TemplateLiteralPart) {
				if !parts[0].IsLiteral || parts[0].Literal != "r.0:" {
					t.Errorf("Expected literal 'r.0:', got %v", parts[0])
				}
				if parts[1].IsLiteral {
					t.Error("Expected non-literal part for index variable")
				}
			},
		},
		{
			name: "template literal with empty parts",
			keyExpr: &ast_domain.TemplateLiteral{
				Parts: []ast_domain.TemplateLiteralPart{},
			},
			expectedParts: 0,
			validateParts: func(t *testing.T, parts []ast_domain.TemplateLiteralPart) {
				if len(parts) != 0 {
					t.Errorf("Expected empty parts slice, got %d parts", len(parts))
				}
			},
		},
		{
			name: "template literal with only literals",
			keyExpr: &ast_domain.TemplateLiteral{
				Parts: []ast_domain.TemplateLiteralPart{
					{IsLiteral: true, Literal: "r."},
					{IsLiteral: true, Literal: "0:"},
					{IsLiteral: true, Literal: "1."},
				},
			},
			expectedParts: 3,
			validateParts: func(t *testing.T, parts []ast_domain.TemplateLiteralPart) {
				for i, part := range parts {
					if !part.IsLiteral {
						t.Errorf("Expected all parts to be literal, part %d is not", i)
					}
				}
			},
		},
		{
			name: "template literal with only expressions",
			keyExpr: &ast_domain.TemplateLiteral{
				Parts: []ast_domain.TemplateLiteralPart{
					{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "__pikoLoopIdx"}},
					{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "__pikoLoopIdx2"}},
				},
			},
			expectedParts: 2,
			validateParts: func(t *testing.T, parts []ast_domain.TemplateLiteralPart) {
				for i, part := range parts {
					if part.IsLiteral {
						t.Errorf("Expected all parts to be expressions, part %d is literal", i)
					}
				}
			},
		},
		{
			name:          "identifier expression",
			keyExpr:       &ast_domain.Identifier{Name: "__pikoLoopIdx"},
			expectedParts: 1,
			validateParts: func(t *testing.T, parts []ast_domain.TemplateLiteralPart) {
				if parts[0].IsLiteral {
					t.Error("Expected non-literal part for identifier")
				}
			},
		},
		{
			name: "complex expression (binary)",
			keyExpr: &ast_domain.BinaryExpression{
				Left:     &ast_domain.Identifier{Name: "a"},
				Operator: ast_domain.OpPlus,
				Right:    &ast_domain.Identifier{Name: "b"},
			},
			expectedParts: 1,
			validateParts: func(t *testing.T, parts []ast_domain.TemplateLiteralPart) {
				if parts[0].IsLiteral {
					t.Error("Expected non-literal part for complex expression")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyser := newKeyAnalyser(nil)
			parts := analyser.extractPathPartsForNestedLoop(tt.keyExpr)

			if tt.expectedParts == 0 && parts == nil {
			} else if len(parts) != tt.expectedParts {
				t.Fatalf("Expected %d parts, got %d", tt.expectedParts, len(parts))
			}

			if tt.validateParts != nil {
				tt.validateParts(t, parts)
			}
		})
	}
}

func TestKeyAnalyser_extractPathPartsFromKey(t *testing.T) {
	tests := []struct {
		keyExpr       ast_domain.Expression
		validateParts func(*testing.T, []ast_domain.TemplateLiteralPart)
		name          string
		expectedParts int
		expectNil     bool
	}{
		{
			name:      "nil expression",
			keyExpr:   nil,
			expectNil: true,
		},
		{
			name:          "pure string literal",
			keyExpr:       &ast_domain.StringLiteral{Value: "r.0:0:"},
			expectedParts: 1,
			validateParts: func(t *testing.T, parts []ast_domain.TemplateLiteralPart) {
				if !parts[0].IsLiteral || parts[0].Literal != "r.0:0:" {
					t.Errorf("Expected literal 'r.0:0:', got %v", parts[0])
				}
			},
		},
		{
			name: "template literal with variable - strips last variable",
			keyExpr: &ast_domain.TemplateLiteral{
				Parts: []ast_domain.TemplateLiteralPart{
					{IsLiteral: true, Literal: "r.0:"},
					{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "item"}},
				},
			},
			expectedParts: 1,
			validateParts: func(t *testing.T, parts []ast_domain.TemplateLiteralPart) {
				if !parts[0].IsLiteral || parts[0].Literal != "r.0:" {
					t.Errorf("Expected literal 'r.0:' after stripping variable, got %v", parts[0])
				}
			},
		},
		{
			name:      "identifier only - no path",
			keyExpr:   &ast_domain.Identifier{Name: "item"},
			expectNil: true,
		},
		{
			name: "template literal with all literals - keeps all",
			keyExpr: &ast_domain.TemplateLiteral{
				Parts: []ast_domain.TemplateLiteralPart{
					{IsLiteral: true, Literal: "r."},
					{IsLiteral: true, Literal: "0:"},
				},
			},
			expectedParts: 2,
			validateParts: func(t *testing.T, parts []ast_domain.TemplateLiteralPart) {
				for _, part := range parts {
					if !part.IsLiteral {
						t.Error("Expected all parts to be literal")
					}
				}
			},
		},
		{
			name: "template literal with multiple variables - strips only last",
			keyExpr: &ast_domain.TemplateLiteral{
				Parts: []ast_domain.TemplateLiteralPart{
					{IsLiteral: true, Literal: "r."},
					{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "__pikoLoopIdx"}},
					{IsLiteral: true, Literal: ":0."},
					{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "item"}},
				},
			},
			expectedParts: 3,
			validateParts: func(t *testing.T, parts []ast_domain.TemplateLiteralPart) {

				if len(parts) != 3 {
					t.Fatalf("Expected 3 parts after stripping last variable, got %d", len(parts))
				}
			},
		},
		{
			name: "template literal empty parts",
			keyExpr: &ast_domain.TemplateLiteral{
				Parts: []ast_domain.TemplateLiteralPart{},
			},
			expectNil: true,
		},
		{
			name: "complex expression - treated as opaque",
			keyExpr: &ast_domain.BinaryExpression{
				Left:     &ast_domain.Identifier{Name: "a"},
				Operator: ast_domain.OpPlus,
				Right:    &ast_domain.Identifier{Name: "b"},
			},
			expectedParts: 1,
			validateParts: func(t *testing.T, parts []ast_domain.TemplateLiteralPart) {
				if parts[0].IsLiteral {
					t.Error("Expected non-literal part for complex expression")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyser := newKeyAnalyser(nil)
			parts := analyser.extractPathPartsFromKey(tt.keyExpr)

			if tt.expectNil {
				if parts != nil {
					t.Errorf("Expected nil parts, got %d parts", len(parts))
				}
				return
			}

			if len(parts) != tt.expectedParts {
				t.Fatalf("Expected %d parts, got %d", tt.expectedParts, len(parts))
			}

			if tt.validateParts != nil {
				tt.validateParts(t, parts)
			}
		})
	}
}

func TestKeyAnalyser_replaceItemVariablesWithIndexVariables(t *testing.T) {
	tests := []struct {
		setupContext  func(*AnalysisContext)
		validateParts func(*testing.T, []ast_domain.TemplateLiteralPart)
		name          string
		parts         []ast_domain.TemplateLiteralPart
		expectedParts int
	}{
		{
			name:          "empty parts",
			parts:         []ast_domain.TemplateLiteralPart{},
			expectedParts: 0,
		},
		{
			name: "literal parts only - all kept",
			parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: true, Literal: "r.0:"},
				{IsLiteral: true, Literal: "0:"},
			},
			expectedParts: 2,
			validateParts: func(t *testing.T, parts []ast_domain.TemplateLiteralPart) {
				for i, part := range parts {
					if !part.IsLiteral {
						t.Errorf("Expected part %d to be literal", i)
					}
				}
			},
		},
		{
			name: "item variable filtered out",
			parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: true, Literal: "r.0:"},
				{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "item"}},
			},
			expectedParts: 1,
			validateParts: func(t *testing.T, parts []ast_domain.TemplateLiteralPart) {
				if !parts[0].IsLiteral {
					t.Error("Expected remaining part to be literal")
				}
			},
		},
		{
			name: "index variable kept with annotation from symbol table",
			parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: true, Literal: "r.0:"},
				{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "__pikoLoopIdx"}},
			},
			setupContext: func(ctx *AnalysisContext) {
				ctx.Symbols.Define(Symbol{
					Name:           "__pikoLoopIdx",
					CodeGenVarName: "__pikoLoopIdx",
					TypeInfo: &ast_domain.ResolvedTypeInfo{
						TypeExpression: goast.NewIdent("int"),
					},
				})
			},
			expectedParts: 2,
			validateParts: func(t *testing.T, parts []ast_domain.TemplateLiteralPart) {
				if parts[1].IsLiteral {
					t.Error("Expected index variable part to be non-literal")
				}
				identifier, ok := parts[1].Expression.(*ast_domain.Identifier)
				if !ok {
					t.Error("Expected identifier expression")
				} else if identifier.GoAnnotations == nil {
					t.Error("Expected GoAnnotations to be set from symbol table")
				}
			},
		},
		{
			name: "index variable kept even without symbol table entry",
			parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: true, Literal: "r.0:"},
				{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "__pikoLoopIdx"}},
			},
			expectedParts: 2,
			validateParts: func(t *testing.T, parts []ast_domain.TemplateLiteralPart) {
				if parts[1].IsLiteral {
					t.Error("Expected index variable part to be non-literal")
				}
			},
		},
		{
			name: "multiple item variables filtered",
			parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: true, Literal: "r."},
				{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "row"}},
				{IsLiteral: true, Literal: ":0."},
				{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "cell"}},
			},
			expectedParts: 2,
			validateParts: func(t *testing.T, parts []ast_domain.TemplateLiteralPart) {
				for _, part := range parts {
					if !part.IsLiteral {
						t.Error("Expected only literal parts after filtering item variables")
					}
				}
			},
		},
		{
			name: "mixed: literals, index variables, item variables",
			parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: true, Literal: "r."},
				{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "__pikoLoopIdx"}},
				{IsLiteral: true, Literal: ":0."},
				{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "item"}},
			},
			setupContext: func(ctx *AnalysisContext) {
				ctx.Symbols.Define(Symbol{
					Name:           "__pikoLoopIdx",
					CodeGenVarName: "__pikoLoopIdx",
					TypeInfo: &ast_domain.ResolvedTypeInfo{
						TypeExpression: goast.NewIdent("int"),
					},
				})
			},
			expectedParts: 3,
			validateParts: func(t *testing.T, parts []ast_domain.TemplateLiteralPart) {
				if !parts[0].IsLiteral {
					t.Error("Expected first part to be literal")
				}
				if parts[1].IsLiteral {
					t.Error("Expected second part to be index variable (non-literal)")
				}
				if !parts[2].IsLiteral {
					t.Error("Expected third part to be literal")
				}
			},
		},
		{
			name: "non-identifier expression kept",
			parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: true, Literal: "r."},
				{IsLiteral: false, Expression: &ast_domain.BinaryExpression{
					Left:     &ast_domain.Identifier{Name: "a"},
					Operator: ast_domain.OpPlus,
					Right:    &ast_domain.Identifier{Name: "b"},
				}},
			},
			expectedParts: 2,
			validateParts: func(t *testing.T, parts []ast_domain.TemplateLiteralPart) {
				if parts[1].IsLiteral {
					t.Error("Expected non-literal part for complex expression")
				}
			},
		},
		{
			name: "multiple index variables with different suffixes",
			parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "__pikoLoopIdx"}},
				{IsLiteral: true, Literal: "."},
				{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "__pikoLoopIdx2"}},
			},
			setupContext: func(ctx *AnalysisContext) {
				ctx.Symbols.Define(Symbol{
					Name:           "__pikoLoopIdx",
					CodeGenVarName: "__pikoLoopIdx",
					TypeInfo: &ast_domain.ResolvedTypeInfo{
						TypeExpression: goast.NewIdent("int"),
					},
				})
				ctx.Symbols.Define(Symbol{
					Name:           "__pikoLoopIdx2",
					CodeGenVarName: "__pikoLoopIdx2",
					TypeInfo: &ast_domain.ResolvedTypeInfo{
						TypeExpression: goast.NewIdent("int"),
					},
				})
			},
			expectedParts: 3,
			validateParts: func(t *testing.T, parts []ast_domain.TemplateLiteralPart) {
				if parts[0].IsLiteral || parts[2].IsLiteral {
					t.Error("Expected index variables to be kept")
				}
				if !parts[1].IsLiteral {
					t.Error("Expected middle part to be literal")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyser := newKeyAnalyser(nil)
			ctx := createTestAnalysisContext()
			if tt.setupContext != nil {
				tt.setupContext(ctx)
			}

			result := analyser.replaceItemVariablesWithIndexVariables(tt.parts, ctx)

			if len(result) != tt.expectedParts {
				t.Fatalf("Expected %d parts, got %d", tt.expectedParts, len(result))
			}

			if tt.validateParts != nil {
				tt.validateParts(t, result)
			}
		})
	}
}

func TestKeyAnalyser_buildExpressionFromParts(t *testing.T) {
	tests := []struct {
		validateResult func(*testing.T, ast_domain.Expression)
		name           string
		expectedType   string
		parts          []ast_domain.TemplateLiteralPart
	}{
		{
			name:         "empty parts - returns empty string literal",
			parts:        []ast_domain.TemplateLiteralPart{},
			expectedType: "*ast_domain.StringLiteral",
			validateResult: func(t *testing.T, expression ast_domain.Expression) {
				strLit, ok := expression.(*ast_domain.StringLiteral)
				if !ok {
					t.Fatalf("Expected StringLiteral, got %T", expression)
				}
				if strLit.Value != "" {
					t.Errorf("Expected empty string, got '%s'", strLit.Value)
				}
			},
		},
		{
			name: "only literals - merges into single string literal",
			parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: true, Literal: "r."},
				{IsLiteral: true, Literal: "0:"},
				{IsLiteral: true, Literal: "1."},
			},
			expectedType: "*ast_domain.StringLiteral",
			validateResult: func(t *testing.T, expression ast_domain.Expression) {
				strLit, ok := expression.(*ast_domain.StringLiteral)
				if !ok {
					t.Fatalf("Expected StringLiteral, got %T", expression)
				}
				if strLit.Value != "r.0:1." {
					t.Errorf("Expected 'r.0:1.', got '%s'", strLit.Value)
				}
			},
		},
		{
			name: "single literal - returns string literal",
			parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: true, Literal: "static"},
			},
			expectedType: "*ast_domain.StringLiteral",
			validateResult: func(t *testing.T, expression ast_domain.Expression) {
				strLit, ok := expression.(*ast_domain.StringLiteral)
				if !ok {
					t.Fatalf("Expected StringLiteral, got %T", expression)
				}
				if strLit.Value != "static" {
					t.Errorf("Expected 'static', got '%s'", strLit.Value)
				}
			},
		},
		{
			name: "single dynamic part - returns expression directly",
			parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "__pikoLoopIdx"}},
			},
			expectedType: "*ast_domain.Identifier",
			validateResult: func(t *testing.T, expression ast_domain.Expression) {
				identifier, ok := expression.(*ast_domain.Identifier)
				if !ok {
					t.Fatalf("Expected Identifier, got %T", expression)
				}
				if identifier.Name != "__pikoLoopIdx" {
					t.Errorf("Expected '__pikoLoopIdx', got '%s'", identifier.Name)
				}
			},
		},
		{
			name: "mixed literal and dynamic - creates template literal",
			parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: true, Literal: "r.0:"},
				{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "__pikoLoopIdx"}},
			},
			expectedType: "*ast_domain.TemplateLiteral",
			validateResult: func(t *testing.T, expression ast_domain.Expression) {
				templ, ok := expression.(*ast_domain.TemplateLiteral)
				if !ok {
					t.Fatalf("Expected TemplateLiteral, got %T", expression)
				}
				if len(templ.Parts) != 2 {
					t.Errorf("Expected 2 parts, got %d", len(templ.Parts))
				}
			},
		},
		{
			name: "consecutive literals merged - creates template literal",
			parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: true, Literal: "a"},
				{IsLiteral: true, Literal: "b"},
				{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "x"}},
				{IsLiteral: true, Literal: "c"},
				{IsLiteral: true, Literal: "d"},
			},
			expectedType: "*ast_domain.TemplateLiteral",
			validateResult: func(t *testing.T, expression ast_domain.Expression) {
				templ, ok := expression.(*ast_domain.TemplateLiteral)
				if !ok {
					t.Fatalf("Expected TemplateLiteral, got %T", expression)
				}

				if len(templ.Parts) != 3 {
					t.Errorf("Expected 3 parts after merging, got %d", len(templ.Parts))
				}
				if !templ.Parts[0].IsLiteral || templ.Parts[0].Literal != "ab" {
					t.Errorf("Expected first part to be 'ab', got %v", templ.Parts[0])
				}
				if !templ.Parts[2].IsLiteral || templ.Parts[2].Literal != "cd" {
					t.Errorf("Expected third part to be 'cd', got %v", templ.Parts[2])
				}
			},
		},
		{
			name: "multiple dynamic parts - creates template literal",
			parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "a"}},
				{IsLiteral: true, Literal: "."},
				{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "b"}},
			},
			expectedType: "*ast_domain.TemplateLiteral",
			validateResult: func(t *testing.T, expression ast_domain.Expression) {
				templ, ok := expression.(*ast_domain.TemplateLiteral)
				if !ok {
					t.Fatalf("Expected TemplateLiteral, got %T", expression)
				}
				if len(templ.Parts) != 3 {
					t.Errorf("Expected 3 parts, got %d", len(templ.Parts))
				}
			},
		},
		{
			name: "dynamic followed by literal - creates template literal",
			parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "index"}},
				{IsLiteral: true, Literal: ".suffix"},
			},
			expectedType: "*ast_domain.TemplateLiteral",
			validateResult: func(t *testing.T, expression ast_domain.Expression) {
				templ, ok := expression.(*ast_domain.TemplateLiteral)
				if !ok {
					t.Fatalf("Expected TemplateLiteral, got %T", expression)
				}
				if len(templ.Parts) != 2 {
					t.Errorf("Expected 2 parts, got %d", len(templ.Parts))
				}
			},
		},
		{
			name: "complex expression as dynamic part",
			parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: true, Literal: "prefix."},
				{IsLiteral: false, Expression: &ast_domain.BinaryExpression{
					Left:     &ast_domain.Identifier{Name: "a"},
					Operator: ast_domain.OpPlus,
					Right:    &ast_domain.Identifier{Name: "b"},
				}},
			},
			expectedType: "*ast_domain.TemplateLiteral",
			validateResult: func(t *testing.T, expression ast_domain.Expression) {
				templ, ok := expression.(*ast_domain.TemplateLiteral)
				if !ok {
					t.Fatalf("Expected TemplateLiteral, got %T", expression)
				}
				if len(templ.Parts) != 2 {
					t.Errorf("Expected 2 parts, got %d", len(templ.Parts))
				}
				_, ok = templ.Parts[1].Expression.(*ast_domain.BinaryExpression)
				if !ok {
					t.Error("Expected second part to contain BinaryExpr")
				}
			},
		},
		{
			name: "empty literals filtered out during merge",
			parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: true, Literal: ""},
				{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "x"}},
				{IsLiteral: true, Literal: ""},
			},
			expectedType: "*ast_domain.Identifier",
			validateResult: func(t *testing.T, expression ast_domain.Expression) {

				identifier, ok := expression.(*ast_domain.Identifier)
				if !ok {
					t.Fatalf("Expected Identifier after empty literal filtering, got %T", expression)
				}
				if identifier.Name != "x" {
					t.Errorf("Expected 'x', got '%s'", identifier.Name)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyser := newKeyAnalyser(nil)
			loc := ast_domain.Location{Line: 1, Column: 1, Offset: 0}

			expression := analyser.buildExpressionFromParts(tt.parts, loc)

			if expression == nil {
				t.Fatal("Expected non-nil expression")
			}

			actualType := fmt.Sprintf("%T", expression)
			if actualType != tt.expectedType {
				t.Errorf("Expected type %s, got %s", tt.expectedType, actualType)
			}

			if tt.validateResult != nil {
				tt.validateResult(t, expression)
			}
		})
	}
}

func createTestAnalysisContext() *AnalysisContext {
	return &AnalysisContext{
		Symbols:                  NewSymbolTable(nil),
		Diagnostics:              new([]*ast_domain.Diagnostic),
		CurrentGoFullPackagePath: "test/package",
		CurrentGoPackageName:     "test",
		CurrentGoSourcePath:      "/test/file.go",
		SFCSourcePath:            "/test/file.phtml",
		Logger:                   logger_domain.GetLogger("test"),
	}
}

func TestNeedsDotSeparator(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		parts    []ast_domain.TemplateLiteralPart
		expected bool
	}{
		{
			name:     "empty parts returns true",
			parts:    []ast_domain.TemplateLiteralPart{},
			expected: true,
		},
		{
			name: "last part is non-literal returns true",
			parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: true, Literal: "r.0:"},
				{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "index"}},
			},
			expected: true,
		},
		{
			name: "last part is literal ending with dot returns false",
			parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: true, Literal: "r.0."},
			},
			expected: false,
		},
		{
			name: "last part is literal not ending with dot returns true",
			parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: true, Literal: "r.0:"},
			},
			expected: true,
		},
		{
			name: "single literal with just a dot returns false",
			parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: true, Literal: "."},
			},
			expected: false,
		},
		{
			name: "single literal empty string returns true",
			parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: true, Literal: ""},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := needsDotSeparator(tc.parts)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCloneIdentifier(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		result := cloneIdentifier(nil)

		assert.Nil(t, result)
	})

	t.Run("clones all fields", func(t *testing.T) {
		t.Parallel()

		original := &ast_domain.Identifier{
			Name:             "myVar",
			RelativeLocation: ast_domain.Location{Line: 5, Column: 10, Offset: 42},
			SourceLength:     5,
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				Stringability: 1,
			},
		}

		clone := cloneIdentifier(original)

		require.NotNil(t, clone)
		assert.Equal(t, original.Name, clone.Name)
		assert.Equal(t, original.RelativeLocation, clone.RelativeLocation)
		assert.Equal(t, original.SourceLength, clone.SourceLength)
		assert.Equal(t, original.GoAnnotations, clone.GoAnnotations)
	})

	t.Run("clone is a separate object", func(t *testing.T) {
		t.Parallel()

		original := &ast_domain.Identifier{
			Name:         "index",
			SourceLength: 3,
		}

		clone := cloneIdentifier(original)

		require.NotNil(t, clone)

		clone.Name = "modified"
		assert.Equal(t, "index", original.Name)
		assert.Equal(t, "modified", clone.Name)
	})
}

func TestReplaceBlankIdentifierInIdentifier(t *testing.T) {
	t.Parallel()

	t.Run("blank identifier is replaced with clone of new variable", func(t *testing.T) {
		t.Parallel()

		blank := &ast_domain.Identifier{Name: "_"}
		newVar := &ast_domain.Identifier{
			Name:         "__pikoLoopIdx",
			SourceLength: 13,
		}

		result := replaceBlankIdentifierInIdentifier(blank, newVar)

		require.NotNil(t, result)
		identifier, ok := result.(*ast_domain.Identifier)
		require.True(t, ok)
		assert.Equal(t, "__pikoLoopIdx", identifier.Name)

		assert.NotSame(t, newVar, identifier)
	})

	t.Run("non-blank identifier is returned as-is", func(t *testing.T) {
		t.Parallel()

		original := &ast_domain.Identifier{Name: "item"}
		newVar := &ast_domain.Identifier{Name: "__pikoLoopIdx"}

		result := replaceBlankIdentifierInIdentifier(original, newVar)

		require.NotNil(t, result)
		identifier, ok := result.(*ast_domain.Identifier)
		require.True(t, ok)
		assert.Equal(t, "item", identifier.Name)
		assert.Same(t, original, identifier)
	})
}

func TestCreateIndexVariableIdentifier(t *testing.T) {
	t.Parallel()

	t.Run("creates identifier with correct name and annotations", func(t *testing.T) {
		t.Parallel()

		typeInfo := &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("int"),
		}

		identifier := createIndexVariableIdentifier("__pikoLoopIdx", typeInfo)

		require.NotNil(t, identifier)
		assert.Equal(t, "__pikoLoopIdx", identifier.Name)
		assert.Equal(t, 0, identifier.SourceLength)

		require.NotNil(t, identifier.GoAnnotations)
		assert.Equal(t, int(inspector_dto.StringablePrimitive), identifier.GoAnnotations.Stringability)
		require.NotNil(t, identifier.GoAnnotations.BaseCodeGenVarName)
		assert.Equal(t, "__pikoLoopIdx", *identifier.GoAnnotations.BaseCodeGenVarName)
		assert.Equal(t, typeInfo, identifier.GoAnnotations.ResolvedType)
	})

	t.Run("annotations have correct default values", func(t *testing.T) {
		t.Parallel()

		typeInfo := &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("string"),
		}

		identifier := createIndexVariableIdentifier("index", typeInfo)

		require.NotNil(t, identifier.GoAnnotations)
		assert.Nil(t, identifier.GoAnnotations.EffectiveKeyExpression)
		assert.Nil(t, identifier.GoAnnotations.DynamicCollectionInfo)
		assert.Nil(t, identifier.GoAnnotations.StaticCollectionLiteral)
		assert.Nil(t, identifier.GoAnnotations.ParentTypeName)
		assert.Nil(t, identifier.GoAnnotations.Symbol)
		assert.Nil(t, identifier.GoAnnotations.PartialInfo)
		assert.False(t, identifier.GoAnnotations.IsStatic)
		assert.False(t, identifier.GoAnnotations.NeedsCSRF)
		assert.False(t, identifier.GoAnnotations.NeedsRuntimeSafetyCheck)
		assert.False(t, identifier.GoAnnotations.IsPointerToStringable)
		assert.False(t, identifier.GoAnnotations.IsCollectionCall)
		assert.False(t, identifier.GoAnnotations.IsMapAccess)
	})
}

func TestReplaceBlankIdentifierInExpr(t *testing.T) {
	t.Parallel()

	t.Run("nil expression returns nil", func(t *testing.T) {
		t.Parallel()

		newVar := &ast_domain.Identifier{Name: "__pikoLoopIdx"}

		result := replaceBlankIdentifierInExpr(nil, newVar)

		assert.Nil(t, result)
	})

	t.Run("blank identifier in template literal is replaced", func(t *testing.T) {
		t.Parallel()

		newVar := &ast_domain.Identifier{
			Name:         "__pikoLoopIdx",
			SourceLength: 13,
		}
		literal := &ast_domain.TemplateLiteral{
			Parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: true, Literal: "r.0."},
				{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "_"}},
			},
		}

		result := replaceBlankIdentifierInExpr(literal, newVar)

		require.NotNil(t, result)
		tl, ok := result.(*ast_domain.TemplateLiteral)
		require.True(t, ok)
		require.Equal(t, 2, len(tl.Parts))
		identifier, ok := tl.Parts[1].Expression.(*ast_domain.Identifier)
		require.True(t, ok)
		assert.Equal(t, "__pikoLoopIdx", identifier.Name)
	})

	t.Run("non-blank identifier in template literal is unchanged", func(t *testing.T) {
		t.Parallel()

		newVar := &ast_domain.Identifier{Name: "__pikoLoopIdx"}
		literal := &ast_domain.TemplateLiteral{
			Parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: true, Literal: "r.0."},
				{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "item"}},
			},
		}

		result := replaceBlankIdentifierInExpr(literal, newVar)

		assert.Same(t, literal, result)
	})

	t.Run("unknown expression type is returned as-is", func(t *testing.T) {
		t.Parallel()

		newVar := &ast_domain.Identifier{Name: "__pikoLoopIdx"}
		expression := &ast_domain.StringLiteral{Value: "static"}

		result := replaceBlankIdentifierInExpr(expression, newVar)

		assert.Same(t, expression, result)
	})
}

func TestKeyAnalyser_annotateEffectiveKey(t *testing.T) {
	t.Parallel()

	ka := newKeyAnalyser(nil)

	t.Run("nil expression does not panic", func(t *testing.T) {
		t.Parallel()

		ka.annotateEffectiveKey(nil)
	})

	t.Run("template literal gets string annotation", func(t *testing.T) {
		t.Parallel()

		tl := &ast_domain.TemplateLiteral{
			Parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: true, Literal: "r.0."},
			},
		}

		ka.annotateEffectiveKey(tl)

		require.NotNil(t, tl.GoAnnotations)
		require.NotNil(t, tl.GoAnnotations.ResolvedType)
		identifier, ok := tl.GoAnnotations.ResolvedType.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "string", identifier.Name)
	})

	t.Run("string literal gets string annotation", func(t *testing.T) {
		t.Parallel()

		sl := &ast_domain.StringLiteral{Value: "static-key"}

		ka.annotateEffectiveKey(sl)

		require.NotNil(t, sl.GoAnnotations)
		require.NotNil(t, sl.GoAnnotations.ResolvedType)
		identifier, ok := sl.GoAnnotations.ResolvedType.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "string", identifier.Name)
	})

	t.Run("other expression type is not annotated", func(t *testing.T) {
		t.Parallel()

		id := &ast_domain.Identifier{Name: "index"}

		ka.annotateEffectiveKey(id)

		assert.Nil(t, id.GoAnnotations)
	})
}

func TestKeyAnalyser_attachEffectiveKeyToNode(t *testing.T) {
	t.Parallel()

	ka := newKeyAnalyser(nil)

	t.Run("creates GoAnnotations if nil", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{}
		key := &ast_domain.StringLiteral{Value: "r.0"}

		ka.attachEffectiveKeyToNode(node, key)

		require.NotNil(t, node.GoAnnotations)
		assert.Equal(t, key, node.GoAnnotations.EffectiveKeyExpression)
	})

	t.Run("uses existing GoAnnotations", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				Stringability: 1,
			},
		}
		key := &ast_domain.StringLiteral{Value: "r.0"}

		ka.attachEffectiveKeyToNode(node, key)

		assert.Equal(t, key, node.GoAnnotations.EffectiveKeyExpression)

		assert.Equal(t, 1, node.GoAnnotations.Stringability)
	})
}

func TestKeyAnalyser_fixBlankIdentifierInChildKeys(t *testing.T) {
	t.Parallel()

	t.Run("nil node does not panic", func(t *testing.T) {
		t.Parallel()

		ka := newKeyAnalyser(nil)
		newVar := &ast_domain.Identifier{Name: "__pikoLoopIdx"}

		assert.NotPanics(t, func() {
			ka.fixBlankIdentifierInChildKeys(nil, newVar)
		})
	})

	t.Run("nil newIndexVar does not panic", func(t *testing.T) {
		t.Parallel()

		ka := newKeyAnalyser(nil)
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
		}

		assert.NotPanics(t, func() {
			ka.fixBlankIdentifierInChildKeys(node, nil)
		})
	})

	t.Run("replaces blank identifier in child key", func(t *testing.T) {
		t.Parallel()

		ka := newKeyAnalyser(nil)
		newVar := &ast_domain.Identifier{
			Name:         "__pikoLoopIdx",
			SourceLength: 13,
		}
		child := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "span",
			Key:      &ast_domain.Identifier{Name: "_"},
		}
		parent := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			Children: []*ast_domain.TemplateNode{child},
		}

		ka.fixBlankIdentifierInChildKeys(parent, newVar)

		identifier, ok := child.Key.(*ast_domain.Identifier)
		require.True(t, ok)
		assert.Equal(t, "__pikoLoopIdx", identifier.Name)
	})

	t.Run("does not descend into children with DirFor", func(t *testing.T) {
		t.Parallel()

		ka := newKeyAnalyser(nil)
		newVar := &ast_domain.Identifier{Name: "__pikoLoopIdx"}
		grandchild := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "em",
			Key:      &ast_domain.Identifier{Name: "_"},
		}
		child := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "span",
			DirFor:   &ast_domain.Directive{Type: ast_domain.DirectiveFor, RawExpression: "item in items"},
			Children: []*ast_domain.TemplateNode{grandchild},
		}
		parent := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			Children: []*ast_domain.TemplateNode{child},
		}

		ka.fixBlankIdentifierInChildKeys(parent, newVar)

		identifier, ok := grandchild.Key.(*ast_domain.Identifier)
		require.True(t, ok)
		assert.Equal(t, "_", identifier.Name)
	})
}

func TestFixBlankIdentifierInNodeKey(t *testing.T) {
	t.Parallel()

	t.Run("nil node does not panic", func(t *testing.T) {
		t.Parallel()

		newVar := &ast_domain.Identifier{Name: "__pikoLoopIdx"}

		assert.NotPanics(t, func() {
			fixBlankIdentifierInNodeKey(nil, newVar)
		})
	})

	t.Run("node with nil key is unchanged", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
		}
		newVar := &ast_domain.Identifier{Name: "__pikoLoopIdx"}

		fixBlankIdentifierInNodeKey(node, newVar)

		assert.Nil(t, node.Key)
	})

	t.Run("node with blank key is replaced", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			Key:      &ast_domain.Identifier{Name: "_"},
		}
		newVar := &ast_domain.Identifier{Name: "__pikoLoopIdx"}

		fixBlankIdentifierInNodeKey(node, newVar)

		identifier, ok := node.Key.(*ast_domain.Identifier)
		require.True(t, ok)
		assert.Equal(t, "__pikoLoopIdx", identifier.Name)
	})

	t.Run("node with DirFor stops recursion", func(t *testing.T) {
		t.Parallel()

		child := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "span",
			Key:      &ast_domain.Identifier{Name: "_"},
		}
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			DirFor:   &ast_domain.Directive{Type: ast_domain.DirectiveFor, RawExpression: "item in items"},
			Children: []*ast_domain.TemplateNode{child},
		}
		newVar := &ast_domain.Identifier{Name: "__pikoLoopIdx"}

		fixBlankIdentifierInNodeKey(node, newVar)

		identifier, ok := child.Key.(*ast_domain.Identifier)
		require.True(t, ok)
		assert.Equal(t, "_", identifier.Name)
	})
}

func TestKeyAnalyser_appendIndexToKeyParts(t *testing.T) {
	t.Parallel()

	ka := newKeyAnalyser(nil)

	t.Run("empty parts gets dot and index", func(t *testing.T) {
		t.Parallel()

		indexVar := &ast_domain.Identifier{Name: "__pikoLoopIdx"}

		result := ka.appendIndexToKeyParts(nil, indexVar)

		require.Len(t, result, 2)
		assert.True(t, result[0].IsLiteral)
		assert.Equal(t, ".", result[0].Literal)
		assert.False(t, result[1].IsLiteral)
		identifier, ok := result[1].Expression.(*ast_domain.Identifier)
		require.True(t, ok)
		assert.Equal(t, "__pikoLoopIdx", identifier.Name)
	})

	t.Run("parts ending with dot do not get extra dot", func(t *testing.T) {
		t.Parallel()

		parts := []ast_domain.TemplateLiteralPart{
			{IsLiteral: true, Literal: "r.0.", Expression: nil, RelativeLocation: ast_domain.Location{Line: 0, Column: 0, Offset: 0}},
		}
		indexVar := &ast_domain.Identifier{Name: "__pikoLoopIdx"}

		result := ka.appendIndexToKeyParts(parts, indexVar)

		require.Len(t, result, 2)
		assert.True(t, result[0].IsLiteral)
		assert.Equal(t, "r.0.", result[0].Literal)
		assert.False(t, result[1].IsLiteral)
	})

	t.Run("parts ending with non-dot literal get dot separator", func(t *testing.T) {
		t.Parallel()

		parts := []ast_domain.TemplateLiteralPart{
			{IsLiteral: true, Literal: "r.0:", Expression: nil, RelativeLocation: ast_domain.Location{Line: 0, Column: 0, Offset: 0}},
		}
		indexVar := &ast_domain.Identifier{Name: "__pikoLoopIdx"}

		result := ka.appendIndexToKeyParts(parts, indexVar)

		require.Len(t, result, 3)
		assert.True(t, result[0].IsLiteral)
		assert.Equal(t, "r.0:", result[0].Literal)
		assert.True(t, result[1].IsLiteral)
		assert.Equal(t, ".", result[1].Literal)
		assert.False(t, result[2].IsLiteral)
	})
}

func TestCreateStringTypeAnnotation(t *testing.T) {
	t.Parallel()

	t.Run("returns annotation with string type", func(t *testing.T) {
		t.Parallel()

		ann := createStringTypeAnnotation()

		require.NotNil(t, ann)
		require.NotNil(t, ann.ResolvedType)
		assert.Equal(t, 1, ann.Stringability)
		assert.Nil(t, ann.EffectiveKeyExpression)
		assert.Nil(t, ann.BaseCodeGenVarName)
		assert.False(t, ann.IsStatic)
		assert.False(t, ann.NeedsCSRF)
	})
}

func TestNewKeyAnalyser_Constructor(t *testing.T) {
	t.Parallel()

	t.Run("nil resolver accepted", func(t *testing.T) {
		t.Parallel()

		ka := newKeyAnalyser(nil)

		require.NotNil(t, ka)
		assert.Nil(t, ka.typeResolver)
	})

	t.Run("stores resolver reference", func(t *testing.T) {
		t.Parallel()

		tr := &TypeResolver{}
		ka := newKeyAnalyser(tr)

		require.NotNil(t, ka)
		assert.Same(t, tr, ka.typeResolver)
	})
}
