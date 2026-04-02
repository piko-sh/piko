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
	goast "go/ast"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestSemanticAnalyser_Enter_NilNode(t *testing.T) {
	diagnostics := make([]*ast_domain.Diagnostic, 0)
	ctx := NewRootAnalysisContext(
		&diagnostics,
		"test/package",
		"testpkg",
		"test.go",
		"test.piko",
	)

	resolver := &TypeResolver{inspector: nil}
	analyser := NewSemanticAnalyser(resolver, ctx, nil, nil, nil, nil, SemanticAnalyserConfig{})

	visitor, err := analyser.Enter(context.Background(), nil)

	if err != nil {
		t.Errorf("Expected no error for nil node, got: %v", err)
	}
	if visitor != nil {
		t.Error("Expected nil visitor for nil node")
	}
	if len(diagnostics) != 0 {
		t.Errorf("Expected no diagnostics for nil node, got %d", len(diagnostics))
	}
}

func TestSemanticAnalyser_Enter_SimpleNode(t *testing.T) {
	diagnostics := make([]*ast_domain.Diagnostic, 0)
	ctx := NewRootAnalysisContext(
		&diagnostics,
		"test/package",
		"testpkg",
		"test.go",
		"test.piko",
	)

	resolver := &TypeResolver{inspector: nil}
	analyser := NewSemanticAnalyser(resolver, ctx, nil, nil, nil, nil, SemanticAnalyserConfig{})

	node := &ast_domain.TemplateNode{
		TagName:  "div",
		NodeType: ast_domain.NodeElement,
		Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	visitor, err := analyser.Enter(context.Background(), node)

	if err != nil {
		t.Errorf("Expected no error for simple node, got: %v", err)
	}
	if visitor == nil {
		t.Error("Expected visitor for child nodes, got nil")
	}
	if len(diagnostics) != 0 {
		t.Errorf("Expected no diagnostics for simple node, got %d", len(diagnostics))
	}
}

func TestSemanticAnalyser_Exit(t *testing.T) {
	ctx := NewRootAnalysisContext(
		new([]*ast_domain.Diagnostic),
		"test/package",
		"testpkg",
		"test.go",
		"test.piko",
	)

	resolver := &TypeResolver{inspector: nil}
	analyser := NewSemanticAnalyser(resolver, ctx, nil, nil, nil, nil, SemanticAnalyserConfig{})

	node := &ast_domain.TemplateNode{
		TagName:  "div",
		NodeType: ast_domain.NodeElement,
	}

	err := analyser.Exit(context.Background(), node)

	if err != nil {
		t.Errorf("Expected no error from Exit, got: %v", err)
	}
}

func TestSemanticAnalyser_newVisitorForChild(t *testing.T) {
	parentCtx := NewRootAnalysisContext(
		new([]*ast_domain.Diagnostic),
		"test/package",
		"testpkg",
		"test.go",
		"test.piko",
	)

	resolver := &TypeResolver{inspector: nil}
	parentAnalyser := NewSemanticAnalyser(resolver, parentCtx, nil, nil, nil, nil, SemanticAnalyserConfig{})

	childCtx := &AnalysisContext{
		Symbols:                  NewSymbolTable(parentCtx.Symbols),
		Diagnostics:              parentCtx.Diagnostics,
		CurrentGoFullPackagePath: parentCtx.CurrentGoFullPackagePath,
		CurrentGoPackageName:     parentCtx.CurrentGoPackageName,
		CurrentGoSourcePath:      parentCtx.CurrentGoSourcePath,
		SFCSourcePath:            parentCtx.SFCSourcePath,
		Logger:                   parentCtx.Logger,
	}
	childAnalyser := parentAnalyser.newVisitorForChild(childCtx, nil, nil)

	if childAnalyser.typeResolver != parentAnalyser.typeResolver {
		t.Error("Child analyser should share typeResolver with parent")
	}
	if childAnalyser.contextManager != parentAnalyser.contextManager {
		t.Error("Child analyser should share contextManager with parent")
	}
	if childAnalyser.attributeManager != parentAnalyser.attributeManager {
		t.Error("Child analyser should share attributeManager with parent")
	}
	if childAnalyser.keyAnalyser != parentAnalyser.keyAnalyser {
		t.Error("Child analyser should share keyAnalyser with parent")
	}
	if childAnalyser.internalsManager != parentAnalyser.internalsManager {
		t.Error("Child analyser should share internalsManager with parent")
	}

	if childAnalyser.ctx == parentAnalyser.ctx {
		t.Error("Child analyser should have different context than parent")
	}

	if childAnalyser.depth != parentAnalyser.depth+1 {
		t.Errorf("Child depth should be parent depth + 1, got %d, expected %d", childAnalyser.depth, parentAnalyser.depth+1)
	}
}

func TestSemanticAnalyser_SpecialistsCreated(t *testing.T) {
	ctx := NewRootAnalysisContext(
		new([]*ast_domain.Diagnostic),
		"test/package",
		"testpkg",
		"test.go",
		"test.piko",
	)

	resolver := &TypeResolver{inspector: nil}
	analyser := NewSemanticAnalyser(resolver, ctx, nil, nil, nil, nil, SemanticAnalyserConfig{})

	if analyser.typeResolver == nil {
		t.Error("typeResolver should not be nil")
	}
	if analyser.contextManager == nil {
		t.Error("contextManager should not be nil")
	}
	if analyser.attributeManager == nil {
		t.Error("attributeManager should not be nil")
	}
	if analyser.keyAnalyser == nil {
		t.Error("keyAnalyser should not be nil")
	}
	if analyser.internalsManager == nil {
		t.Error("internalsManager should not be nil")
	}
	if analyser.ctx == nil {
		t.Error("ctx should not be nil")
	}
}

func TestResolveAndValidate(t *testing.T) {
	tests := []struct {
		directive        *ast_domain.Directive
		validateFunction func(*ast_domain.Directive, *AnalysisContext)
		name             string
		expectValidation bool
	}{
		{
			name:             "nil directive",
			directive:        nil,
			validateFunction: nil,
			expectValidation: false,
		},
		{
			name: "directive with nil expression",
			directive: &ast_domain.Directive{
				Type:       ast_domain.DirectiveIf,
				Expression: nil,
				Location:   ast_domain.Location{Line: 1, Column: 1, Offset: 0},
			},
			validateFunction: nil,
			expectValidation: false,
		},
		{
			name: "directive with expression",
			directive: &ast_domain.Directive{
				Type:       ast_domain.DirectiveIf,
				Expression: &ast_domain.Identifier{Name: "condition"},
				Location:   ast_domain.Location{Line: 1, Column: 1, Offset: 0},
			},
			validateFunction: nil,
			expectValidation: false,
		},
		{
			name: "directive with validation function",
			directive: &ast_domain.Directive{
				Type:       ast_domain.DirectiveIf,
				Expression: &ast_domain.Identifier{Name: "condition"},
				Location:   ast_domain.Location{Line: 1, Column: 1, Offset: 0},
			},
			validateFunction: func(d *ast_domain.Directive, ctx *AnalysisContext) {},
			expectValidation: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewRootAnalysisContext(
				new([]*ast_domain.Diagnostic),
				"test/package",
				"testpkg",
				"test.go",
				"test.piko",
			)

			ctx.Symbols.Define(Symbol{
				Name:     "condition",
				TypeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: nil, PackageAlias: "", CanonicalPackagePath: "", IsSynthetic: false, IsExportedPackageSymbol: false, InitialPackagePath: "", InitialFilePath: ""},
			})

			resolver := &TypeResolver{inspector: nil}
			resolveAndValidate(context.Background(), tt.directive, ctx, resolver, tt.validateFunction)
			if tt.directive != nil && tt.directive.Expression != nil {
				if tt.directive.GoAnnotations == nil {
					t.Error("Expected GoAnnotations to be set after resolveAndValidate")
				}
			}
		})
	}
}

func TestIsIterable(t *testing.T) {
	tests := []struct {
		typeInfo *ast_domain.ResolvedTypeInfo
		name     string
		expected bool
	}{
		{
			name:     "nil type info",
			typeInfo: nil,
			expected: false,
		},
		{
			name: "array type",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.ArrayType{},
			},
			expected: true,
		},
		{
			name: "map type",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.MapType{},
			},
			expected: true,
		},
		{
			name: "string type",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("string"),
			},
			expected: true,
		},
		{
			name: "int type - not iterable",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("int"),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isIterable(tt.typeInfo)
			if result != tt.expected {
				t.Errorf("Expected isIterable to return %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestStringPtr(t *testing.T) {
	input := "test string"
	result := &input

	if *result != input {
		t.Errorf("Expected dereferenced value to be %q, got %q", input, *result)
	}
}

func TestCreateEmptyAnnotationResult(t *testing.T) {
	t.Parallel()

	t.Run("returns result wrapping provided AST", func(t *testing.T) {
		t.Parallel()

		templateAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{TagName: "div", NodeType: ast_domain.NodeElement},
			},
		}
		diagnostics := []*ast_domain.Diagnostic{
			ast_domain.NewDiagnostic(ast_domain.Warning, "test warn", "expr", ast_domain.Location{}, ""),
		}

		result, diagnostics, err := createEmptyAnnotationResult(templateAST, diagnostics)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Same(t, templateAST, result.AnnotatedAST)
		assert.Nil(t, result.VirtualModule)
		assert.Empty(t, result.StyleBlock)
		assert.Nil(t, result.AssetRefs)
		assert.Nil(t, result.CustomTags)
		assert.Nil(t, result.UniqueInvocations)
		assert.Nil(t, result.AssetDependencies)
		assert.Nil(t, result.AnalysisMap)
		assert.Empty(t, result.ClientScript)
		assert.Len(t, diagnostics, 1)
		assert.Equal(t, "test warn", diagnostics[0].Message)
	})

	t.Run("handles nil AST and empty diagnostics", func(t *testing.T) {
		t.Parallel()

		result, diagnostics, err := createEmptyAnnotationResult(nil, nil)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Nil(t, result.AnnotatedAST)
		assert.Nil(t, diagnostics)
	})
}

func TestBuildTranslationKeySet(t *testing.T) {
	t.Parallel()

	t.Run("returns nil for nil virtual component", func(t *testing.T) {
		t.Parallel()

		result := buildTranslationKeySet(nil)
		assert.Nil(t, result)
	})

	t.Run("returns nil for nil source", func(t *testing.T) {
		t.Parallel()

		vc := &annotator_dto.VirtualComponent{Source: nil}
		result := buildTranslationKeySet(vc)
		assert.Nil(t, result)
	})

	t.Run("returns nil when no local translations", func(t *testing.T) {
		t.Parallel()

		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				LocalTranslations: nil,
			},
		}
		result := buildTranslationKeySet(vc)
		assert.Nil(t, result)
	})

	t.Run("returns nil for empty translations map", func(t *testing.T) {
		t.Parallel()

		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				LocalTranslations: map[string]map[string]string{},
			},
		}
		result := buildTranslationKeySet(vc)
		assert.Nil(t, result)
	})

	t.Run("extracts keys from single locale", func(t *testing.T) {
		t.Parallel()

		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				LocalTranslations: map[string]map[string]string{
					"en": {
						"greeting": "Hello",
						"farewell": "Goodbye",
					},
				},
			},
		}

		result := buildTranslationKeySet(vc)

		require.NotNil(t, result)
		assert.Contains(t, result.LocalKeys, "greeting")
		assert.Contains(t, result.LocalKeys, "farewell")
		assert.Nil(t, result.GlobalKeys)
	})

	t.Run("merges keys from multiple locales", func(t *testing.T) {
		t.Parallel()

		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				LocalTranslations: map[string]map[string]string{
					"en": {"greeting": "Hello"},
					"fr": {"greeting": "Bonjour", "extra": "Supplément"},
				},
			},
		}

		result := buildTranslationKeySet(vc)

		require.NotNil(t, result)
		assert.Contains(t, result.LocalKeys, "greeting")
		assert.Contains(t, result.LocalKeys, "extra")
	})

	t.Run("returns nil when locales have empty key maps", func(t *testing.T) {
		t.Parallel()

		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				LocalTranslations: map[string]map[string]string{
					"en": {},
				},
			},
		}
		result := buildTranslationKeySet(vc)
		assert.Nil(t, result)
	})
}

func TestDetermineEffectiveKeyForChildren(t *testing.T) {
	t.Parallel()

	t.Run("returns EffectiveKeyExpression from GoAnnotations", func(t *testing.T) {
		t.Parallel()

		keyExpr := &ast_domain.Identifier{Name: "effectiveKey"}
		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				EffectiveKeyExpression: keyExpr,
			},
		}

		result := determineEffectiveKeyForChildren(node)
		assert.Same(t, keyExpr, result)
	})

	t.Run("returns node Key when no GoAnnotations effective key", func(t *testing.T) {
		t.Parallel()

		keyExpr := &ast_domain.Identifier{Name: "nodeKey"}
		node := &ast_domain.TemplateNode{Key: keyExpr}

		result := determineEffectiveKeyForChildren(node)
		assert.Same(t, keyExpr, result)
	})

	t.Run("returns nil when no key anywhere", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{TagName: "div"}
		result := determineEffectiveKeyForChildren(node)
		assert.Nil(t, result)
	})

	t.Run("prefers GoAnnotations key over node Key", func(t *testing.T) {
		t.Parallel()

		goKey := &ast_domain.Identifier{Name: "goKey"}
		nodeKey := &ast_domain.Identifier{Name: "nodeKey"}
		node := &ast_domain.TemplateNode{
			Key: nodeKey,
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				EffectiveKeyExpression: goKey,
			},
		}

		result := determineEffectiveKeyForChildren(node)
		assert.Same(t, goKey, result)
	})

	t.Run("returns nil when GoAnnotations exists but has no effective key", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				EffectiveKeyExpression: nil,
			},
		}
		result := determineEffectiveKeyForChildren(node)
		assert.Nil(t, result)
	})
}

func TestGetClientScriptLocation(t *testing.T) {
	t.Parallel()

	t.Run("returns default location for nil virtual component", func(t *testing.T) {
		t.Parallel()

		result := getClientScriptLocation(nil)
		assert.Equal(t, 1, result.Line)
		assert.Equal(t, 1, result.Column)
		assert.Equal(t, 0, result.Offset)
	})

	t.Run("returns default location for nil source", func(t *testing.T) {
		t.Parallel()

		vc := &annotator_dto.VirtualComponent{Source: nil}
		result := getClientScriptLocation(vc)
		assert.Equal(t, 1, result.Line)
		assert.Equal(t, 1, result.Column)
	})

	t.Run("returns default location for valid component", func(t *testing.T) {
		t.Parallel()

		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				SourcePath:   "/test/comp.piko",
				ClientScript: "export function handleClick() {}",
			},
		}
		result := getClientScriptLocation(vc)
		assert.Equal(t, 1, result.Line)
		assert.Equal(t, 1, result.Column)
	})
}

func TestBuildPartialInvocationMap(t *testing.T) {
	t.Parallel()

	t.Run("returns existing map when pInfo is nil", func(t *testing.T) {
		t.Parallel()

		ctx := NewRootAnalysisContext(new([]*ast_domain.Diagnostic), "test/pkg", "testpkg", "test.go", "test.piko")
		resolver := &TypeResolver{inspector: nil}
		analyser := NewSemanticAnalyser(resolver, ctx, nil, nil, nil, nil, SemanticAnalyserConfig{})

		result := analyser.buildPartialInvocationMap(nil)
		assert.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("returns existing map when pInfo has empty package name", func(t *testing.T) {
		t.Parallel()

		ctx := NewRootAnalysisContext(new([]*ast_domain.Diagnostic), "test/pkg", "testpkg", "test.go", "test.piko")
		resolver := &TypeResolver{inspector: nil}
		analyser := NewSemanticAnalyser(resolver, ctx, nil, nil, nil, nil, SemanticAnalyserConfig{})

		pInfo := &ast_domain.PartialInvocationInfo{PartialPackageName: "", InvocationKey: "inv_abc"}
		result := analyser.buildPartialInvocationMap(pInfo)
		assert.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("creates new map with active pInfo", func(t *testing.T) {
		t.Parallel()

		ctx := NewRootAnalysisContext(new([]*ast_domain.Diagnostic), "test/pkg", "testpkg", "test.go", "test.piko")
		resolver := &TypeResolver{inspector: nil}
		analyser := NewSemanticAnalyser(resolver, ctx, nil, nil, nil, nil, SemanticAnalyserConfig{})

		pInfo := &ast_domain.PartialInvocationInfo{PartialPackageName: "card_pkg", InvocationKey: "inv_123", PartialAlias: "card"}
		result := analyser.buildPartialInvocationMap(pInfo)
		require.Len(t, result, 1)
		assert.Same(t, pInfo, result["card_pkg"])
	})

	t.Run("preserves existing entries and adds new", func(t *testing.T) {
		t.Parallel()

		ctx := NewRootAnalysisContext(new([]*ast_domain.Diagnostic), "test/pkg", "testpkg", "test.go", "test.piko")
		resolver := &TypeResolver{inspector: nil}
		analyser := NewSemanticAnalyser(resolver, ctx, nil, nil, nil, nil, SemanticAnalyserConfig{})

		existingPInfo := &ast_domain.PartialInvocationInfo{PartialPackageName: "header_pkg", InvocationKey: "inv_001"}
		analyser.partialInvocationMap["header_pkg"] = existingPInfo

		newPInfo := &ast_domain.PartialInvocationInfo{PartialPackageName: "card_pkg", InvocationKey: "inv_002"}
		result := analyser.buildPartialInvocationMap(newPInfo)
		require.Len(t, result, 2)
		assert.Same(t, existingPInfo, result["header_pkg"])
		assert.Same(t, newPInfo, result["card_pkg"])
	})

	t.Run("does not modify original map", func(t *testing.T) {
		t.Parallel()

		ctx := NewRootAnalysisContext(new([]*ast_domain.Diagnostic), "test/pkg", "testpkg", "test.go", "test.piko")
		resolver := &TypeResolver{inspector: nil}
		analyser := NewSemanticAnalyser(resolver, ctx, nil, nil, nil, nil, SemanticAnalyserConfig{})

		pInfo := &ast_domain.PartialInvocationInfo{PartialPackageName: "card_pkg", InvocationKey: "inv_123"}
		_ = analyser.buildPartialInvocationMap(pInfo)
		assert.Empty(t, analyser.partialInvocationMap)
	})
}

func TestNewSemanticAnalyser_Extended(t *testing.T) {
	t.Parallel()

	t.Run("initialises all fields correctly", func(t *testing.T) {
		t.Parallel()

		ctx := NewRootAnalysisContext(new([]*ast_domain.Diagnostic), "test/pkg", "testpkg", "test.go", "test.piko")
		resolver := &TypeResolver{inspector: nil}
		analysisMap := make(map[*ast_domain.TemplateNode]*AnalysisContext)
		pInfo := &ast_domain.PartialInvocationInfo{PartialPackageName: "card", InvocationKey: "inv_001"}
		config := SemanticAnalyserConfig{MainComponentHash: "main_abc"}

		sa := NewSemanticAnalyser(resolver, ctx, pInfo, nil, nil, analysisMap, config)

		require.NotNil(t, sa)
		assert.Same(t, resolver, sa.typeResolver)
		assert.Same(t, ctx, sa.ctx)
		assert.Same(t, pInfo, sa.currentPartialInfo)
		assert.NotNil(t, sa.analysisMap)
		assert.NotNil(t, sa.contextManager)
		assert.NotNil(t, sa.attributeManager)
		assert.NotNil(t, sa.keyAnalyser)
		assert.NotNil(t, sa.internalsManager)
		assert.Nil(t, sa.parentEffectiveKey)
		assert.Equal(t, 0, sa.depth)
		assert.NotNil(t, sa.partialInvocationMap)
		assert.Empty(t, sa.partialInvocationMap)
	})
}

func TestSemanticAnalyser_newVisitorForChild_WithPartialInfo(t *testing.T) {
	t.Parallel()

	t.Run("extends partial invocation map when pInfo has package name", func(t *testing.T) {
		t.Parallel()

		ctx := NewRootAnalysisContext(new([]*ast_domain.Diagnostic), "test/pkg", "testpkg", "test.go", "test.piko")
		resolver := &TypeResolver{inspector: nil}
		parent := NewSemanticAnalyser(resolver, ctx, nil, nil, nil, nil, SemanticAnalyserConfig{})

		pInfo := &ast_domain.PartialInvocationInfo{PartialPackageName: "card_pkg", InvocationKey: "inv_123"}
		childCtx := ctx.ForChildScope()
		child := parent.newVisitorForChild(childCtx, pInfo, nil)

		require.Len(t, child.partialInvocationMap, 1)
		assert.Same(t, pInfo, child.partialInvocationMap["card_pkg"])
		assert.Empty(t, parent.partialInvocationMap)
	})

	t.Run("sets parent effective key", func(t *testing.T) {
		t.Parallel()

		ctx := NewRootAnalysisContext(new([]*ast_domain.Diagnostic), "test/pkg", "testpkg", "test.go", "test.piko")
		resolver := &TypeResolver{inspector: nil}
		parent := NewSemanticAnalyser(resolver, ctx, nil, nil, nil, nil, SemanticAnalyserConfig{})

		parentKey := &ast_domain.Identifier{Name: "key"}
		childCtx := ctx.ForChildScope()
		child := parent.newVisitorForChild(childCtx, nil, parentKey)
		assert.Same(t, parentKey, child.parentEffectiveKey)
	})

	t.Run("increments depth", func(t *testing.T) {
		t.Parallel()

		ctx := NewRootAnalysisContext(new([]*ast_domain.Diagnostic), "test/pkg", "testpkg", "test.go", "test.piko")
		resolver := &TypeResolver{inspector: nil}
		parent := NewSemanticAnalyser(resolver, ctx, nil, nil, nil, nil, SemanticAnalyserConfig{})

		childCtx := ctx.ForChildScope()
		child := parent.newVisitorForChild(childCtx, nil, nil)
		grandchild := child.newVisitorForChild(childCtx.ForChildScope(), nil, nil)

		assert.Equal(t, 0, parent.depth)
		assert.Equal(t, 1, child.depth)
		assert.Equal(t, 2, grandchild.depth)
	})
}

func TestIsIterable_AdditionalCases(t *testing.T) {
	t.Parallel()

	t.Run("nil type expression returns false", func(t *testing.T) {
		t.Parallel()
		result := isIterable(&ast_domain.ResolvedTypeInfo{TypeExpression: nil})
		assert.False(t, result)
	})

	t.Run("bool type is not iterable", func(t *testing.T) {
		t.Parallel()
		result := isIterable(&ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("bool")})
		assert.False(t, result)
	})

	t.Run("float64 type is not iterable", func(t *testing.T) {
		t.Parallel()
		result := isIterable(&ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("float64")})
		assert.False(t, result)
	})

	t.Run("struct type is not iterable", func(t *testing.T) {
		t.Parallel()
		result := isIterable(&ast_domain.ResolvedTypeInfo{TypeExpression: &goast.StructType{Fields: &goast.FieldList{}}})
		assert.False(t, result)
	})

	t.Run("pointer type is not iterable", func(t *testing.T) {
		t.Parallel()
		result := isIterable(&ast_domain.ResolvedTypeInfo{TypeExpression: &goast.StarExpr{X: goast.NewIdent("int")}})
		assert.False(t, result)
	})
}

func TestLogAnalysisContext(t *testing.T) {
	t.Parallel()

	t.Run("does not panic for nil context", func(t *testing.T) {
		t.Parallel()
		assert.NotPanics(t, func() {
			logAnalysisContext(nil, "test")
		})
	})

	t.Run("does not panic for valid context", func(t *testing.T) {
		t.Parallel()
		diagnostics := &[]*ast_domain.Diagnostic{}
		ctx := NewRootAnalysisContext(diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")
		assert.NotPanics(t, func() {
			logAnalysisContext(ctx, "Test Title")
		})
	})
}

func TestBuildAnnotationResult(t *testing.T) {
	t.Parallel()

	t.Run("builds result with all fields", func(t *testing.T) {
		t.Parallel()

		flatAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{{TagName: "div"}},
		}
		vm := &annotator_dto.VirtualModule{}
		linkResult := &annotator_dto.LinkingResult{
			CombinedCSS:       "body { color: red; }",
			UniqueInvocations: []*annotator_dto.PartialInvocation{{InvocationKey: "inv1"}},
		}
		analysisMap := make(map[*ast_domain.TemplateNode]*AnalysisContext)
		mainComp := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				ClientScript: "export function onClick() {}",
			},
		}

		result := buildAnnotationResult(flatAST, vm, linkResult, analysisMap, mainComp)

		require.NotNil(t, result)
		assert.Same(t, flatAST, result.AnnotatedAST)
		assert.Same(t, vm, result.VirtualModule)
		assert.Equal(t, "body { color: red; }", result.StyleBlock)
		assert.Equal(t, analysisMap, result.AnalysisMap)
		assert.Equal(t, "export function onClick() {}", result.ClientScript)
		assert.Len(t, result.UniqueInvocations, 1)
	})
}

func TestValidateForDirective(t *testing.T) {
	t.Parallel()

	t.Run("produces diagnostic when expression is not ForInExpr", func(t *testing.T) {
		t.Parallel()

		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")

		d := &ast_domain.Directive{
			Type:          ast_domain.DirectiveFor,
			Expression:    &ast_domain.Identifier{Name: "notAForExpr"},
			RawExpression: "notAForExpr",
			Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		}

		validateForDirective(d, ctx)

		require.Len(t, diagnostics, 1)
		assert.Contains(t, diagnostics[0].Message, "p-for expression is not a valid 'in' loop")
		assert.Equal(t, ast_domain.Error, diagnostics[0].Severity)
	})

	t.Run("produces diagnostic when collection is not iterable", func(t *testing.T) {
		t.Parallel()

		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")

		collectionExpr := &ast_domain.Identifier{Name: "myBool"}
		collectionExpr.SetGoAnnotation(&ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("bool"),
			},
		})

		d := &ast_domain.Directive{
			Type: ast_domain.DirectiveFor,
			Expression: &ast_domain.ForInExpression{
				ItemVariable: &ast_domain.Identifier{Name: "item"},
				Collection:   collectionExpr,
			},
			RawExpression: "item in myBool",
			Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		}

		validateForDirective(d, ctx)

		require.Len(t, diagnostics, 1)
		assert.Contains(t, diagnostics[0].Message, "Cannot loop over type 'bool'")
		assert.Equal(t, ast_domain.Error, diagnostics[0].Severity)
	})

	t.Run("no diagnostic when collection is iterable", func(t *testing.T) {
		t.Parallel()

		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")

		collectionExpr := &ast_domain.Identifier{Name: "items"}
		collectionExpr.SetGoAnnotation(&ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.ArrayType{Elt: goast.NewIdent("string")},
			},
		})

		d := &ast_domain.Directive{
			Type: ast_domain.DirectiveFor,
			Expression: &ast_domain.ForInExpression{
				ItemVariable: &ast_domain.Identifier{Name: "item"},
				Collection:   collectionExpr,
			},
			RawExpression: "item in items",
			Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		}

		validateForDirective(d, ctx)

		assert.Empty(t, diagnostics)
	})

	t.Run("no diagnostic when collection has nil annotation", func(t *testing.T) {
		t.Parallel()

		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")

		d := &ast_domain.Directive{
			Type: ast_domain.DirectiveFor,
			Expression: &ast_domain.ForInExpression{
				ItemVariable: &ast_domain.Identifier{Name: "item"},
				Collection:   &ast_domain.Identifier{Name: "unknown"},
			},
			RawExpression: "item in unknown",
			Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		}

		validateForDirective(d, ctx)

		assert.Empty(t, diagnostics)
	})

	t.Run("no diagnostic when collection has nil resolved type", func(t *testing.T) {
		t.Parallel()

		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")

		collectionExpr := &ast_domain.Identifier{Name: "things"}
		collectionExpr.SetGoAnnotation(&ast_domain.GoGeneratorAnnotation{
			ResolvedType: nil,
		})

		d := &ast_domain.Directive{
			Type: ast_domain.DirectiveFor,
			Expression: &ast_domain.ForInExpression{
				ItemVariable: &ast_domain.Identifier{Name: "item"},
				Collection:   collectionExpr,
			},
			RawExpression: "item in things",
			Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		}

		validateForDirective(d, ctx)

		assert.Empty(t, diagnostics)
	})
}

func TestFindMainComponent(t *testing.T) {
	t.Parallel()

	t.Run("returns component when path exists", func(t *testing.T) {
		t.Parallel()

		expectedComp := &annotator_dto.VirtualComponent{
			HashedName: "comp_abc",
		}
		vm := &annotator_dto.VirtualModule{
			Graph: &annotator_dto.ComponentGraph{
				PathToHashedName: map[string]string{
					"/app/page.piko": "comp_abc",
				},
			},
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				"comp_abc": expectedComp,
			},
		}

		result, err := findMainComponent(context.Background(), vm, "/app/page.piko")

		require.NoError(t, err)
		assert.Same(t, expectedComp, result)
	})

	t.Run("returns error when path not in graph", func(t *testing.T) {
		t.Parallel()

		vm := &annotator_dto.VirtualModule{
			Graph: &annotator_dto.ComponentGraph{
				PathToHashedName: map[string]string{},
			},
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
		}

		result, err := findMainComponent(context.Background(), vm, "/missing/path.piko")

		assert.Nil(t, result)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "entry point path '/missing/path.piko' not found")
	})

	t.Run("returns error when hash not in components map", func(t *testing.T) {
		t.Parallel()

		vm := &annotator_dto.VirtualModule{
			Graph: &annotator_dto.ComponentGraph{
				PathToHashedName: map[string]string{
					"/app/page.piko": "comp_missing",
				},
			},
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
		}

		result, err := findMainComponent(context.Background(), vm, "/app/page.piko")

		assert.Nil(t, result)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "could not find main virtual component for hash 'comp_missing'")
	})
}

func TestCreatePKValidator(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when no client script", func(t *testing.T) {
		t.Parallel()

		comp := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				ClientScript: "",
				SourcePath:   "/test/comp.piko",
			},
		}

		result := createPKValidator(context.Background(), comp)
		assert.Nil(t, result)
	})

	t.Run("creates validator when client script exists", func(t *testing.T) {
		t.Parallel()

		comp := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				ClientScript: "export function handleClick() {}",
				SourcePath:   "/test/comp.piko",
			},
		}

		result := createPKValidator(context.Background(), comp)
		require.NotNil(t, result)
	})

	t.Run("registers imported partial aliases", func(t *testing.T) {
		t.Parallel()

		comp := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				ClientScript: "export function handleClick() {}",
				SourcePath:   "/test/comp.piko",
				PikoImports: []annotator_dto.PikoImport{
					{Alias: "card"},
					{Alias: "header"},
				},
			},
		}

		result := createPKValidator(context.Background(), comp)
		require.NotNil(t, result)
	})
}

func TestRunASTTraversal(t *testing.T) {
	t.Parallel()

	t.Run("returns nil for empty AST", func(t *testing.T) {
		t.Parallel()

		ctx := NewRootAnalysisContext(new([]*ast_domain.Diagnostic), "test/pkg", "testpkg", "test.go", "test.piko")
		resolver := &TypeResolver{inspector: nil}
		visitor := NewSemanticAnalyser(resolver, ctx, nil, nil, nil, nil, SemanticAnalyserConfig{})

		tree := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{},
		}

		err := runASTTraversal(context.Background(), tree, visitor)
		assert.NoError(t, err)
	})

	t.Run("traverses simple nodes", func(t *testing.T) {
		t.Parallel()

		ctx := NewRootAnalysisContext(new([]*ast_domain.Diagnostic), "test/pkg", "testpkg", "test.go", "test.piko")
		resolver := &TypeResolver{inspector: nil}
		analysisMap := make(map[*ast_domain.TemplateNode]*AnalysisContext)
		visitor := NewSemanticAnalyser(resolver, ctx, nil, nil, nil, analysisMap, SemanticAnalyserConfig{})

		node := &ast_domain.TemplateNode{
			TagName:  "div",
			NodeType: ast_domain.NodeElement,
			Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		}
		tree := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{node},
		}

		err := runASTTraversal(context.Background(), tree, visitor)
		assert.NoError(t, err)
		assert.Contains(t, analysisMap, node)
	})
}

func TestSemanticAnalyser_Enter_PopulatesAnalysisMap(t *testing.T) {
	t.Parallel()

	ctx := NewRootAnalysisContext(new([]*ast_domain.Diagnostic), "test/pkg", "testpkg", "test.go", "test.piko")
	resolver := &TypeResolver{inspector: nil}
	analysisMap := make(map[*ast_domain.TemplateNode]*AnalysisContext)
	analyser := NewSemanticAnalyser(resolver, ctx, nil, nil, nil, analysisMap, SemanticAnalyserConfig{})

	node := &ast_domain.TemplateNode{
		TagName:  "span",
		NodeType: ast_domain.NodeElement,
		Location: ast_domain.Location{Line: 5, Column: 3, Offset: 42},
	}

	visitor, err := analyser.Enter(context.Background(), node)

	require.NoError(t, err)
	require.NotNil(t, visitor)
	assert.Contains(t, analysisMap, node)
	assert.NotNil(t, analysisMap[node])
}

func TestValidateForDirective_RejectsEventPlaceholder(t *testing.T) {
	t.Parallel()

	diagnostics := make([]*ast_domain.Diagnostic, 0)
	ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")

	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveFor,
		RawExpression: "$event in items",
		Expression:    &ast_domain.Identifier{Name: "$event"},
		Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	validateForDirective(d, ctx)

	require.Len(t, diagnostics, 1)
	assert.Equal(t, ast_domain.Error, diagnostics[0].Severity)
	assert.Contains(t, diagnostics[0].Message, "$event can only be used in p-on or p-event handlers")
}

func TestValidateForDirective_RejectsFormPlaceholder(t *testing.T) {
	t.Parallel()

	diagnostics := make([]*ast_domain.Diagnostic, 0)
	ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")

	d := &ast_domain.Directive{
		Type:          ast_domain.DirectiveFor,
		RawExpression: "$form in items",
		Expression:    &ast_domain.Identifier{Name: "$form"},
		Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	validateForDirective(d, ctx)

	require.Len(t, diagnostics, 1)
	assert.Equal(t, ast_domain.Error, diagnostics[0].Severity)
	assert.Contains(t, diagnostics[0].Message, "$form can only be used in p-on or p-event handlers")
}

func TestValidateForDirective_IterableMapType(t *testing.T) {
	t.Parallel()

	diagnostics := make([]*ast_domain.Diagnostic, 0)
	ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")

	collectionExpr := &ast_domain.Identifier{Name: "myMap"}
	collectionExpr.SetGoAnnotation(&ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression: &goast.MapType{
				Key:   goast.NewIdent("string"),
				Value: goast.NewIdent("int"),
			},
		},
	})

	d := &ast_domain.Directive{
		Type: ast_domain.DirectiveFor,
		Expression: &ast_domain.ForInExpression{
			ItemVariable:  &ast_domain.Identifier{Name: "val"},
			IndexVariable: &ast_domain.Identifier{Name: "key"},
			Collection:    collectionExpr,
		},
		RawExpression: "val, key in myMap",
		Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	validateForDirective(d, ctx)

	assert.Empty(t, diagnostics)
}

func TestValidateForDirective_IterableStringType(t *testing.T) {
	t.Parallel()

	diagnostics := make([]*ast_domain.Diagnostic, 0)
	ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")

	collectionExpr := &ast_domain.Identifier{Name: "myString"}
	collectionExpr.SetGoAnnotation(&ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("string"),
		},
	})

	d := &ast_domain.Directive{
		Type: ast_domain.DirectiveFor,
		Expression: &ast_domain.ForInExpression{
			ItemVariable: &ast_domain.Identifier{Name: "char"},
			Collection:   collectionExpr,
		},
		RawExpression: "char in myString",
		Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	validateForDirective(d, ctx)

	assert.Empty(t, diagnostics)
}

func TestValidateForDirective_NonIterableStructType(t *testing.T) {
	t.Parallel()

	diagnostics := make([]*ast_domain.Diagnostic, 0)
	ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")

	collectionExpr := &ast_domain.Identifier{Name: "myStruct"}
	collectionExpr.SetGoAnnotation(&ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("int"),
		},
	})

	d := &ast_domain.Directive{
		Type: ast_domain.DirectiveFor,
		Expression: &ast_domain.ForInExpression{
			ItemVariable: &ast_domain.Identifier{Name: "item"},
			Collection:   collectionExpr,
		},
		RawExpression: "item in myStruct",
		Location:      ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	validateForDirective(d, ctx)

	require.Len(t, diagnostics, 1)
	assert.Contains(t, diagnostics[0].Message, "Cannot loop over type 'int'")
}

func TestIsIterable_ChannelType(t *testing.T) {
	t.Parallel()

	result := isIterable(&ast_domain.ResolvedTypeInfo{
		TypeExpression: &goast.ChanType{
			Dir:   goast.RECV,
			Value: goast.NewIdent("int"),
		},
	})
	assert.False(t, result)
}

func TestIsIterable_InterfaceType(t *testing.T) {
	t.Parallel()

	result := isIterable(&ast_domain.ResolvedTypeInfo{
		TypeExpression: &goast.InterfaceType{Methods: &goast.FieldList{}},
	})
	assert.False(t, result)
}

func TestIsIterable_FuncType(t *testing.T) {
	t.Parallel()

	result := isIterable(&ast_domain.ResolvedTypeInfo{
		TypeExpression: &goast.FuncType{},
	})
	assert.False(t, result)
}

func TestSemanticAnalyser_Enter_NodeWithDirIf(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	analysisMap := make(map[*ast_domain.TemplateNode]*AnalysisContext)
	analyser := NewSemanticAnalyser(h.Resolver, h.Context, nil, nil, nil, analysisMap, SemanticAnalyserConfig{})

	condExpr := &ast_domain.Identifier{Name: "someVar"}
	node := &ast_domain.TemplateNode{
		TagName:  "div",
		NodeType: ast_domain.NodeElement,
		Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		DirIf: &ast_domain.Directive{
			Type:          ast_domain.DirectiveIf,
			Expression:    condExpr,
			RawExpression: "someVar",
			Location:      ast_domain.Location{Line: 1, Column: 5, Offset: 4},
		},
	}

	visitor, err := analyser.Enter(context.Background(), node)

	require.NoError(t, err)
	require.NotNil(t, visitor)
	assert.Contains(t, analysisMap, node)
}

func TestSemanticAnalyser_Enter_NodeWithDynamicAttributes(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	analysisMap := make(map[*ast_domain.TemplateNode]*AnalysisContext)
	analyser := NewSemanticAnalyser(h.Resolver, h.Context, nil, nil, nil, analysisMap, SemanticAnalyserConfig{})

	node := &ast_domain.TemplateNode{
		TagName:  "div",
		NodeType: ast_domain.NodeElement,
		Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		DynamicAttributes: []ast_domain.DynamicAttribute{
			{
				Name:          "title",
				Expression:    &ast_domain.StringLiteral{Value: "hello"},
				RawExpression: "'hello'",
				Location:      ast_domain.Location{Line: 1, Column: 10, Offset: 9},
			},
		},
	}

	visitor, err := analyser.Enter(context.Background(), node)

	require.NoError(t, err)
	require.NotNil(t, visitor)
	assert.Contains(t, analysisMap, node)
}

func TestSemanticAnalyser_Enter_NilAnalysisMap(t *testing.T) {
	t.Parallel()

	ctx := NewRootAnalysisContext(new([]*ast_domain.Diagnostic), "test/pkg", "testpkg", "test.go", "test.piko")
	resolver := &TypeResolver{inspector: nil}
	analyser := NewSemanticAnalyser(resolver, ctx, nil, nil, nil, nil, SemanticAnalyserConfig{})

	node := &ast_domain.TemplateNode{
		TagName:  "div",
		NodeType: ast_domain.NodeElement,
		Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	visitor, err := analyser.Enter(context.Background(), node)

	require.NoError(t, err)
	require.NotNil(t, visitor)
}

func TestSemanticAnalyser_Enter_NodeWithDirKey(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	analysisMap := make(map[*ast_domain.TemplateNode]*AnalysisContext)
	analyser := NewSemanticAnalyser(h.Resolver, h.Context, nil, nil, nil, analysisMap, SemanticAnalyserConfig{})

	node := &ast_domain.TemplateNode{
		TagName:  "li",
		NodeType: ast_domain.NodeElement,
		Location: ast_domain.Location{Line: 3, Column: 1, Offset: 0},
		Key:      &ast_domain.Identifier{Name: "item"},
	}

	visitor, err := analyser.Enter(context.Background(), node)

	require.NoError(t, err)
	require.NotNil(t, visitor)
	assert.Contains(t, analysisMap, node)
}

func TestSemanticAnalyser_Enter_NodeWithGoAnnotationsKey(t *testing.T) {
	t.Parallel()

	h := newTypeResolverTestHarness()
	analysisMap := make(map[*ast_domain.TemplateNode]*AnalysisContext)
	analyser := NewSemanticAnalyser(h.Resolver, h.Context, nil, nil, nil, analysisMap, SemanticAnalyserConfig{})

	keyExpr := &ast_domain.Identifier{Name: "effectiveKey"}
	node := &ast_domain.TemplateNode{
		TagName:  "div",
		NodeType: ast_domain.NodeElement,
		Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
			EffectiveKeyExpression: keyExpr,
		},
	}

	visitor, err := analyser.Enter(context.Background(), node)

	require.NoError(t, err)
	require.NotNil(t, visitor)

	childAnalyser, ok := visitor.(*SemanticAnalyser)
	require.True(t, ok)
	assert.Same(t, keyExpr, childAnalyser.parentEffectiveKey)
}
