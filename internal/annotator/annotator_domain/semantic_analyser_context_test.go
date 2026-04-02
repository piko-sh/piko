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
	"piko.sh/piko/internal/logger/logger_domain"
)

func TestNewContextManager(t *testing.T) {
	t.Parallel()

	t.Run("creates context manager with nil values", func(t *testing.T) {
		t.Parallel()

		cm := newContextManager(nil, nil)

		require.NotNil(t, cm)
		assert.Nil(t, cm.typeResolver)
		assert.Nil(t, cm.virtualModule)
	})

	t.Run("creates context manager with type resolver", func(t *testing.T) {
		t.Parallel()

		tr := &TypeResolver{inspector: nil}
		cm := newContextManager(tr, nil)

		require.NotNil(t, cm)
		assert.Same(t, tr, cm.typeResolver)
		assert.Nil(t, cm.virtualModule)
	})

	t.Run("creates context manager with virtual module", func(t *testing.T) {
		t.Parallel()

		vm := &annotator_dto.VirtualModule{}
		cm := newContextManager(nil, vm)

		require.NotNil(t, cm)
		assert.Nil(t, cm.typeResolver)
		assert.Same(t, vm, cm.virtualModule)
	})

	t.Run("creates context manager with both parameters", func(t *testing.T) {
		t.Parallel()

		tr := &TypeResolver{inspector: nil}
		vm := &annotator_dto.VirtualModule{
			ComponentsByHash: make(map[string]*annotator_dto.VirtualComponent),
		}
		cm := newContextManager(tr, vm)

		require.NotNil(t, cm)
		assert.Same(t, tr, cm.typeResolver)
		assert.Same(t, vm, cm.virtualModule)
	})
}

func TestContextSwitchResult(t *testing.T) {
	t.Parallel()

	t.Run("holds context and partial info", func(t *testing.T) {
		t.Parallel()

		ctx := createContextManagerTestContext()
		pInfo := &ast_domain.PartialInvocationInfo{
			InvocationKey:      "test-key",
			PartialPackageName: "partial_123",
		}

		result := &contextSwitchResult{
			newCtx:      ctx,
			activePInfo: pInfo,
		}

		assert.Same(t, ctx, result.newCtx)
		assert.Same(t, pInfo, result.activePInfo)
	})

	t.Run("allows nil values", func(t *testing.T) {
		t.Parallel()

		result := &contextSwitchResult{
			newCtx:      nil,
			activePInfo: nil,
		}

		assert.Nil(t, result.newCtx)
		assert.Nil(t, result.activePInfo)
	})
}

func TestNeedsContextSwitch(t *testing.T) {
	t.Parallel()

	t.Run("returns false for node without annotations", func(t *testing.T) {
		t.Parallel()

		cm := newContextManager(nil, createContextManagerVirtualModule())
		ctx := createContextManagerTestContext()
		node := &ast_domain.TemplateNode{
			GoAnnotations: nil,
		}

		result := cm.needsContextSwitch(node, ctx)

		assert.False(t, result)
	})

	t.Run("returns false when OriginalPackageAlias is nil", func(t *testing.T) {
		t.Parallel()

		cm := newContextManager(nil, createContextManagerVirtualModule())
		ctx := createContextManagerTestContext()
		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalPackageAlias: nil,
				OriginalSourcePath:   new("/test/main.pk"),
			},
		}

		result := cm.needsContextSwitch(node, ctx)

		assert.False(t, result)
	})

	t.Run("returns false when OriginalSourcePath is nil", func(t *testing.T) {
		t.Parallel()

		cm := newContextManager(nil, createContextManagerVirtualModule())
		ctx := createContextManagerTestContext()
		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalPackageAlias: new("partial_456"),
				OriginalSourcePath:   nil,
			},
		}

		result := cm.needsContextSwitch(node, ctx)

		assert.False(t, result)
	})

	t.Run("returns false when current component not found in module", func(t *testing.T) {
		t.Parallel()

		vm := &annotator_dto.VirtualModule{
			ComponentsByGoPath: make(map[string]*annotator_dto.VirtualComponent),
		}
		cm := newContextManager(nil, vm)
		ctx := createContextManagerTestContext()
		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalPackageAlias: new("partial_456"),
				OriginalSourcePath:   new("/test/partial.pk"),
			},
		}

		result := cm.needsContextSwitch(node, ctx)

		assert.False(t, result)
	})

	t.Run("returns false when hashed names match", func(t *testing.T) {
		t.Parallel()

		vm := createContextManagerVirtualModule()
		cm := newContextManager(nil, vm)
		ctx := createContextManagerTestContext()

		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalPackageAlias: new("main_abc123"),
				OriginalSourcePath:   new("/test/main.pk"),
			},
		}

		result := cm.needsContextSwitch(node, ctx)

		assert.False(t, result)
	})

	t.Run("returns true when hashed names differ", func(t *testing.T) {
		t.Parallel()

		vm := createContextManagerVirtualModule()
		cm := newContextManager(nil, vm)
		ctx := createContextManagerTestContext()

		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalPackageAlias: new("partial_456"),
				OriginalSourcePath:   new("/test/partial.pk"),
			},
		}

		result := cm.needsContextSwitch(node, ctx)

		assert.True(t, result)
	})
}

func TestCreateForLoopContext(t *testing.T) {
	t.Parallel()

	t.Run("returns parent context when DirFor is nil", func(t *testing.T) {
		t.Parallel()

		cm := newContextManager(nil, nil)
		parentCtx := createContextManagerTestContext()
		node := &ast_domain.TemplateNode{
			DirFor: nil,
		}

		result, err := cm.CreateForLoopContext(context.Background(), node, parentCtx)

		require.NoError(t, err)
		assert.Same(t, parentCtx, result)
	})

	t.Run("returns parent context when expression is not ForInExpr", func(t *testing.T) {
		t.Parallel()

		cm := newContextManager(nil, nil)
		parentCtx := createContextManagerTestContext()
		node := &ast_domain.TemplateNode{
			DirFor: &ast_domain.Directive{
				Expression: &ast_domain.Identifier{Name: "notForInExpr"},
			},
		}

		result, err := cm.CreateForLoopContext(context.Background(), node, parentCtx)

		require.NoError(t, err)
		assert.Same(t, parentCtx, result)
	})

	t.Run("returns parent context when collection has no annotation", func(t *testing.T) {
		t.Parallel()

		cm := newContextManager(nil, nil)
		parentCtx := createContextManagerTestContext()
		node := &ast_domain.TemplateNode{
			DirFor: &ast_domain.Directive{
				Expression: &ast_domain.ForInExpression{
					Collection: &ast_domain.Identifier{Name: "items"},
				},
			},
		}

		result, err := cm.CreateForLoopContext(context.Background(), node, parentCtx)

		require.NoError(t, err)
		assert.Same(t, parentCtx, result)
	})

	t.Run("returns parent context when annotation has nil ResolvedType", func(t *testing.T) {
		t.Parallel()

		cm := newContextManager(nil, nil)
		parentCtx := createContextManagerTestContext()
		forExpr := &ast_domain.ForInExpression{
			Collection: &ast_domain.Identifier{
				Name: "items",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					ResolvedType: nil,
				},
			},
		}
		node := &ast_domain.TemplateNode{
			DirFor: &ast_domain.Directive{
				Expression: forExpr,
			},
		}

		result, err := cm.CreateForLoopContext(context.Background(), node, parentCtx)

		require.NoError(t, err)
		assert.Same(t, parentCtx, result)
	})
}

func TestDeterminePartialSelfContext(t *testing.T) {
	t.Parallel()

	t.Run("returns parent context when GoAnnotations is nil", func(t *testing.T) {
		t.Parallel()

		cm := newContextManager(nil, createContextManagerVirtualModule())
		parentCtx := createContextManagerTestContext()
		node := &ast_domain.TemplateNode{
			GoAnnotations: nil,
		}

		result := cm.DeterminePartialSelfContext(node, parentCtx)

		assert.Same(t, parentCtx, result)
	})

	t.Run("returns parent context when PartialInfo is nil", func(t *testing.T) {
		t.Parallel()

		cm := newContextManager(nil, createContextManagerVirtualModule())
		parentCtx := createContextManagerTestContext()
		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				PartialInfo: nil,
			},
		}

		result := cm.DeterminePartialSelfContext(node, parentCtx)

		assert.Same(t, parentCtx, result)
	})

	t.Run("returns parent context when partial component not found", func(t *testing.T) {
		t.Parallel()

		vm := &annotator_dto.VirtualModule{
			ComponentsByHash: make(map[string]*annotator_dto.VirtualComponent),
		}
		cm := newContextManager(nil, vm)
		parentCtx := createContextManagerTestContext()
		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				PartialInfo: &ast_domain.PartialInvocationInfo{
					PartialPackageName: "nonexistent_partial",
				},
			},
		}

		result := cm.DeterminePartialSelfContext(node, parentCtx)

		assert.Same(t, parentCtx, result)
	})

}

func TestDetermineNodeContext(t *testing.T) {
	t.Parallel()

	t.Run("returns parent context for node without annotations", func(t *testing.T) {
		t.Parallel()

		cm := newContextManager(nil, createContextManagerVirtualModule())
		parentCtx := createContextManagerTestContext()
		node := &ast_domain.TemplateNode{
			GoAnnotations: nil,
		}

		resultCtx, resultPInfo := cm.DetermineNodeContext(node, parentCtx, nil, 0)

		assert.Same(t, parentCtx, resultCtx)
		assert.Nil(t, resultPInfo)
	})

	t.Run("returns parent context with existing partial info", func(t *testing.T) {
		t.Parallel()

		cm := newContextManager(nil, createContextManagerVirtualModule())
		parentCtx := createContextManagerTestContext()
		existingPInfo := &ast_domain.PartialInvocationInfo{
			InvocationKey:      "existing-key",
			PartialPackageName: "existing_partial",
		}
		node := &ast_domain.TemplateNode{
			GoAnnotations: nil,
		}

		resultCtx, resultPInfo := cm.DetermineNodeContext(node, parentCtx, existingPInfo, 0)

		assert.Same(t, parentCtx, resultCtx)
		assert.Same(t, existingPInfo, resultPInfo)
	})

	t.Run("updates partial info when node is partial root", func(t *testing.T) {
		t.Parallel()

		cm := newContextManager(nil, createContextManagerVirtualModule())
		parentCtx := createContextManagerTestContext()
		nodePInfo := &ast_domain.PartialInvocationInfo{
			InvocationKey:      "node-key",
			PartialPackageName: "node_partial",
		}
		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				PartialInfo: nodePInfo,
			},
		}

		resultCtx, resultPInfo := cm.DetermineNodeContext(node, parentCtx, nil, 0)

		assert.Same(t, parentCtx, resultCtx)
		assert.Same(t, nodePInfo, resultPInfo)
	})
}

func TestTryContextSwitch(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when context switch not needed", func(t *testing.T) {
		t.Parallel()

		cm := newContextManager(nil, createContextManagerVirtualModule())
		parentCtx := createContextManagerTestContext()
		node := &ast_domain.TemplateNode{
			GoAnnotations: nil,
		}

		result := cm.tryContextSwitch(node, parentCtx, nil, 0)

		assert.Nil(t, result)
	})

	t.Run("returns nil when switched component not found", func(t *testing.T) {
		t.Parallel()

		vm := createContextManagerVirtualModule()
		cm := newContextManager(nil, vm)
		parentCtx := createContextManagerTestContext()

		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalPackageAlias: new("nonexistent_789"),
				OriginalSourcePath:   new("/test/nonexistent.pk"),
			},
		}

		result := cm.tryContextSwitch(node, parentCtx, nil, 0)

		assert.Nil(t, result)
	})

}

func TestCreateSwitchedContext(t *testing.T) {
	t.Parallel()

	t.Run("creates context with component package info", func(t *testing.T) {
		t.Parallel()

		cm := newContextManager(nil, nil)
		parentCtx := createContextManagerTestContext()
		vc := &annotator_dto.VirtualComponent{
			CanonicalGoPackagePath: "test/switched",
			VirtualGoFilePath:      "/virtual/switched.go",
			RewrittenScriptAST: &goast.File{
				Name: goast.NewIdent("switched"),
			},
		}

		result := cm.createSwitchedContext(parentCtx, vc, "/test/switched.pk")

		require.NotNil(t, result)
		assert.Equal(t, "test/switched", result.CurrentGoFullPackagePath)
		assert.Equal(t, "switched", result.CurrentGoPackageName)
		assert.Equal(t, "/virtual/switched.go", result.CurrentGoSourcePath)
		assert.Equal(t, "/test/switched.pk", result.SFCSourcePath)
	})

	t.Run("handles nil RewrittenScriptAST", func(t *testing.T) {
		t.Parallel()

		cm := newContextManager(nil, nil)
		parentCtx := createContextManagerTestContext()
		vc := &annotator_dto.VirtualComponent{
			CanonicalGoPackagePath: "test/noast",
			VirtualGoFilePath:      "/virtual/noast.go",
			RewrittenScriptAST:     nil,
		}

		result := cm.createSwitchedContext(parentCtx, vc, "/test/noast.pk")

		require.NotNil(t, result)
		assert.Equal(t, "test/noast", result.CurrentGoFullPackagePath)
		assert.Empty(t, result.CurrentGoPackageName)
	})
}

func TestPopulateSwitchedContext(t *testing.T) {
	t.Parallel()

	t.Run("identifies switch to active partial correctly", func(t *testing.T) {
		t.Parallel()

		activePInfo := &ast_domain.PartialInvocationInfo{
			InvocationKey:      "active-key",
			PartialPackageName: "partial_123",
		}
		vc := &annotator_dto.VirtualComponent{
			HashedName: "partial_123",
		}

		isSwitchingToActivePartial := activePInfo != nil && vc.HashedName == activePInfo.PartialPackageName

		assert.True(t, isSwitchingToActivePartial)
	})

	t.Run("identifies switch to different component correctly", func(t *testing.T) {
		t.Parallel()

		activePInfo := &ast_domain.PartialInvocationInfo{
			InvocationKey:      "other-key",
			PartialPackageName: "other_partial",
		}
		vc := &annotator_dto.VirtualComponent{
			HashedName: "partial_123",
		}

		isSwitchingToActivePartial := activePInfo != nil && vc.HashedName == activePInfo.PartialPackageName

		assert.False(t, isSwitchingToActivePartial)
	})

	t.Run("handles nil active partial info", func(t *testing.T) {
		t.Parallel()

		var activePInfo *ast_domain.PartialInvocationInfo
		vc := &annotator_dto.VirtualComponent{
			HashedName: "partial_123",
		}

		isSwitchingToActivePartial := activePInfo != nil && vc.HashedName == activePInfo.PartialPackageName

		assert.False(t, isSwitchingToActivePartial)
	})
}

func TestCreateForLoopVisitorContext(t *testing.T) {
	t.Parallel()

	t.Run("returns parent context when DirFor is nil", func(t *testing.T) {
		t.Parallel()

		cm := newContextManager(nil, nil)
		parentCtx := createContextManagerTestContext()
		node := &ast_domain.TemplateNode{
			DirFor: nil,
		}

		result, err := cm.CreateForLoopVisitorContext(context.Background(), node, parentCtx, 0)

		require.NoError(t, err)
		assert.Same(t, parentCtx, result)
	})

	t.Run("returns parent context when expression is not ForInExpr", func(t *testing.T) {
		t.Parallel()

		cm := newContextManager(nil, nil)
		parentCtx := createContextManagerTestContext()
		node := &ast_domain.TemplateNode{
			DirFor: &ast_domain.Directive{
				RawExpression: "invalid",
				Expression:    &ast_domain.Identifier{Name: "notForInExpr"},
			},
		}

		result, err := cm.CreateForLoopVisitorContext(context.Background(), node, parentCtx, 0)

		require.NoError(t, err)
		assert.Same(t, parentCtx, result)
	})

	t.Run("returns parent context when collection has no annotation", func(t *testing.T) {
		t.Parallel()

		cm := newContextManager(nil, nil)
		parentCtx := createContextManagerTestContext()
		node := &ast_domain.TemplateNode{
			DirFor: &ast_domain.Directive{
				RawExpression: "item in items",
				Expression: &ast_domain.ForInExpression{
					Collection: &ast_domain.Identifier{Name: "items"},
				},
			},
		}

		result, err := cm.CreateForLoopVisitorContext(context.Background(), node, parentCtx, 0)

		require.NoError(t, err)
		assert.Same(t, parentCtx, result)
	})
}

func TestDefineItemVariable(t *testing.T) {
	t.Parallel()

	t.Run("does nothing when ItemVariable is nil", func(t *testing.T) {
		t.Parallel()

		tr := &TypeResolver{inspector: nil}
		cm := newContextManager(tr, nil)
		parentCtx := createContextManagerTestContext()
		loopCtx := parentCtx.ForChildScope()
		forExpr := &ast_domain.ForInExpression{
			ItemVariable: nil,
			Collection:   &ast_domain.Identifier{Name: "items"},
		}
		collectionAnn := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.ArrayType{Elt: goast.NewIdent("string")},
			},
		}

		cm.defineItemVariable(context.Background(), forExpr, parentCtx, loopCtx, collectionAnn, "/test.pk")

		assert.Empty(t, loopCtx.Symbols.AllSymbolNames())
	})

	t.Run("defines item variable and enriches annotation", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		cm := newContextManager(h.Resolver, newTestHarnessVirtualModule())
		parentCtx := h.Context
		loopCtx := parentCtx.ForChildScope()
		itemVar := &ast_domain.Identifier{Name: "item"}
		forExpr := &ast_domain.ForInExpression{
			ItemVariable: itemVar,
			Collection:   &ast_domain.Identifier{Name: "items"},
		}
		collectionAnn := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.ArrayType{Elt: goast.NewIdent("string")},
			},
		}

		cm.defineItemVariable(context.Background(), forExpr, parentCtx, loopCtx, collectionAnn, "/test.pk")

		symbolNames := loopCtx.Symbols.AllSymbolNames()
		assert.Contains(t, symbolNames, "item")

		require.NotNil(t, itemVar.GoAnnotations, "item variable should have GoAnnotations set")
		require.NotNil(t, itemVar.GoAnnotations.ResolvedType)
		require.NotNil(t, itemVar.GoAnnotations.Symbol)
		assert.Equal(t, "item", itemVar.GoAnnotations.Symbol.Name)
		require.NotNil(t, itemVar.GoAnnotations.BaseCodeGenVarName)
		assert.Equal(t, "item", *itemVar.GoAnnotations.BaseCodeGenVarName)
	})
}

func TestDefineIndexVariable(t *testing.T) {
	t.Parallel()

	t.Run("does nothing when IndexVariable is nil", func(t *testing.T) {
		t.Parallel()

		tr := &TypeResolver{inspector: nil}
		cm := newContextManager(tr, nil)
		parentCtx := createContextManagerTestContext()
		loopCtx := parentCtx.ForChildScope()
		forExpr := &ast_domain.ForInExpression{
			IndexVariable: nil,
			Collection:    &ast_domain.Identifier{Name: "items"},
		}
		collectionAnn := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.ArrayType{Elt: goast.NewIdent("string")},
			},
		}

		cm.defineIndexVariable(forExpr, parentCtx, loopCtx, collectionAnn, "/test.pk")

		assert.Empty(t, loopCtx.Symbols.AllSymbolNames())
	})

	t.Run("defines index variable and enriches annotation", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		cm := newContextManager(h.Resolver, newTestHarnessVirtualModule())
		parentCtx := h.Context
		loopCtx := parentCtx.ForChildScope()
		indexVar := &ast_domain.Identifier{Name: "index"}
		forExpr := &ast_domain.ForInExpression{
			IndexVariable: indexVar,
			Collection:    &ast_domain.Identifier{Name: "items"},
		}
		collectionAnn := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.ArrayType{Elt: goast.NewIdent("string")},
			},
		}

		cm.defineIndexVariable(forExpr, parentCtx, loopCtx, collectionAnn, "/test.pk")

		symbolNames := loopCtx.Symbols.AllSymbolNames()
		assert.Contains(t, symbolNames, "index")

		require.NotNil(t, indexVar.GoAnnotations, "index variable should have GoAnnotations set")
		require.NotNil(t, indexVar.GoAnnotations.ResolvedType)
		identifier, ok := indexVar.GoAnnotations.ResolvedType.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "int", identifier.Name)

		require.NotNil(t, indexVar.GoAnnotations.Symbol)
		assert.Equal(t, "index", indexVar.GoAnnotations.Symbol.Name)
	})

	t.Run("defines map index variable with key type", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		cm := newContextManager(h.Resolver, newTestHarnessVirtualModule())
		parentCtx := h.Context
		loopCtx := parentCtx.ForChildScope()
		indexVar := &ast_domain.Identifier{Name: "key"}
		forExpr := &ast_domain.ForInExpression{
			IndexVariable: indexVar,
			Collection:    &ast_domain.Identifier{Name: "myMap"},
		}
		collectionAnn := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.MapType{
					Key:   goast.NewIdent("string"),
					Value: goast.NewIdent("int"),
				},
			},
		}

		cm.defineIndexVariable(forExpr, parentCtx, loopCtx, collectionAnn, "/test.pk")

		require.NotNil(t, indexVar.GoAnnotations)
		require.NotNil(t, indexVar.GoAnnotations.ResolvedType)
		identifier, ok := indexVar.GoAnnotations.ResolvedType.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "string", identifier.Name)
	})
}

func TestCreateForLoopVisitorContext_WithAnnotatedCollection(t *testing.T) {
	t.Parallel()

	t.Run("creates loop context with item and index variables", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		vm := newTestHarnessVirtualModule()
		cm := newContextManager(h.Resolver, vm)
		parentCtx := h.Context

		forExpr := &ast_domain.ForInExpression{
			ItemVariable:  &ast_domain.Identifier{Name: "item"},
			IndexVariable: &ast_domain.Identifier{Name: "index"},
			Collection:    &ast_domain.Identifier{Name: "items"},
		}
		setAnnotationOnExpression(forExpr, &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.ArrayType{Elt: goast.NewIdent("string")},
			},
		})

		node := &ast_domain.TemplateNode{
			DirFor: &ast_domain.Directive{
				RawExpression: "item, index in items",
				Expression:    forExpr,
			},
		}

		result, err := cm.CreateForLoopVisitorContext(context.Background(), node, parentCtx, 0)

		require.NoError(t, err)
		assert.NotSame(t, parentCtx, result)
		symbolNames := result.Symbols.AllSymbolNames()
		assert.Contains(t, symbolNames, "item")
		assert.Contains(t, symbolNames, "index")
	})

	t.Run("creates loop context with only item variable", func(t *testing.T) {
		t.Parallel()

		h := newTypeResolverTestHarness()
		vm := newTestHarnessVirtualModule()
		cm := newContextManager(h.Resolver, vm)
		parentCtx := h.Context

		forExpr := &ast_domain.ForInExpression{
			ItemVariable: &ast_domain.Identifier{Name: "val"},
			Collection:   &ast_domain.Identifier{Name: "data"},
		}
		setAnnotationOnExpression(forExpr, &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.MapType{
					Key:   goast.NewIdent("string"),
					Value: goast.NewIdent("int"),
				},
			},
		})

		node := &ast_domain.TemplateNode{
			DirFor: &ast_domain.Directive{
				RawExpression: "val in data",
				Expression:    forExpr,
			},
		}

		result, err := cm.CreateForLoopVisitorContext(context.Background(), node, parentCtx, 0)

		require.NoError(t, err)
		assert.NotSame(t, parentCtx, result)
		symbolNames := result.Symbols.AllSymbolNames()
		assert.Contains(t, symbolNames, "val")
	})
}

func createContextManagerTestContext() *AnalysisContext {
	return &AnalysisContext{
		Symbols:                  NewSymbolTable(nil),
		Diagnostics:              new([]*ast_domain.Diagnostic),
		CurrentGoFullPackagePath: "test/main",
		CurrentGoPackageName:     "main",
		CurrentGoSourcePath:      "/test/main.go",
		SFCSourcePath:            "/test/main.pk",
		Logger:                   logger_domain.GetLogger("test"),
	}
}

func createContextManagerVirtualModule() *annotator_dto.VirtualModule {
	return &annotator_dto.VirtualModule{
		Graph: &annotator_dto.ComponentGraph{
			PathToHashedName: map[string]string{
				"/test/main.pk": "main_abc123",
			},
			HashedNameToPath: map[string]string{
				"main_abc123": "/test/main.pk",
			},
		},
		ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
			"main_abc123": {
				HashedName:             "main_abc123",
				CanonicalGoPackagePath: "test/main",
				VirtualGoFilePath:      "/virtual/main.go",
				RewrittenScriptAST: &goast.File{
					Name: goast.NewIdent("main"),
				},
				Source: &annotator_dto.ParsedComponent{
					SourcePath: "/test/main.pk",
				},
			},
		},
		ComponentsByGoPath: map[string]*annotator_dto.VirtualComponent{
			"test/main": {
				HashedName:             "main_abc123",
				CanonicalGoPackagePath: "test/main",
			},
		},
	}
}

func TestCreateForLoopContext_NoDirFor(t *testing.T) {
	t.Parallel()

	vm := createContextManagerVirtualModule()
	h := newTypeResolverTestHarness()
	cm := newContextManager(h.Resolver, vm)

	parentCtx := NewRootAnalysisContext(
		h.Diagnostics,
		"test/main",
		"main",
		"/virtual/main.go",
		"/test/main.pk",
	)

	node := &ast_domain.TemplateNode{
		DirFor: nil,
	}

	resultCtx, err := cm.CreateForLoopContext(context.Background(), node, parentCtx)

	require.NoError(t, err)
	assert.Same(t, parentCtx, resultCtx, "should return parent context when DirFor is nil")
}

func TestCreateForLoopContext_ForExprNotForInExpr(t *testing.T) {
	t.Parallel()

	vm := createContextManagerVirtualModule()
	h := newTypeResolverTestHarness()
	cm := newContextManager(h.Resolver, vm)

	parentCtx := NewRootAnalysisContext(
		h.Diagnostics,
		"test/main",
		"main",
		"/virtual/main.go",
		"/test/main.pk",
	)

	node := &ast_domain.TemplateNode{
		DirFor: &ast_domain.Directive{
			Expression: &ast_domain.Identifier{Name: "something"},
		},
	}

	resultCtx, err := cm.CreateForLoopContext(context.Background(), node, parentCtx)

	require.NoError(t, err)
	assert.Same(t, parentCtx, resultCtx, "should return parent context when expression is not ForInExpr")
}

func TestCreateForLoopContext_NilAnnotation(t *testing.T) {
	t.Parallel()

	vm := createContextManagerVirtualModule()
	h := newTypeResolverTestHarness()
	cm := newContextManager(h.Resolver, vm)

	parentCtx := NewRootAnalysisContext(
		h.Diagnostics,
		"test/main",
		"main",
		"/virtual/main.go",
		"/test/main.pk",
	)

	forExpr := &ast_domain.ForInExpression{
		ItemVariable: &ast_domain.Identifier{Name: "item"},
		Collection:   &ast_domain.Identifier{Name: "items"},
	}

	node := &ast_domain.TemplateNode{
		DirFor: &ast_domain.Directive{
			Expression: forExpr,
		},
	}

	resultCtx, err := cm.CreateForLoopContext(context.Background(), node, parentCtx)

	require.NoError(t, err)
	assert.Same(t, parentCtx, resultCtx, "should return parent context when annotation is nil")
}

func TestCreateForLoopContext_WithAnnotatedCollection(t *testing.T) {
	t.Parallel()

	vm := createContextManagerVirtualModule()
	h := newTypeResolverTestHarness()
	cm := newContextManager(h.Resolver, vm)

	parentCtx := NewRootAnalysisContext(
		h.Diagnostics,
		"test/main",
		"main",
		"/virtual/main.go",
		"/test/main.pk",
	)

	forExpr := &ast_domain.ForInExpression{
		ItemVariable: &ast_domain.Identifier{Name: "item"},
		Collection:   &ast_domain.Identifier{Name: "items"},
	}

	setAnnotationOnExpression(forExpr, &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression: &goast.ArrayType{Elt: goast.NewIdent("string")},
		},
	})

	node := &ast_domain.TemplateNode{
		DirFor: &ast_domain.Directive{
			Expression: forExpr,
		},
	}

	resultCtx, err := cm.CreateForLoopContext(context.Background(), node, parentCtx)

	require.NoError(t, err)
	assert.NotSame(t, parentCtx, resultCtx, "should return a new child context")

	itemSym, found := resultCtx.Symbols.Find("item")
	assert.True(t, found, "item variable should be defined")
	if found {
		require.NotNil(t, itemSym.TypeInfo)
		identifier, ok := itemSym.TypeInfo.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "string", identifier.Name)
	}
}

func TestDeterminePartialSelfContext_NoPartialInfo(t *testing.T) {
	t.Parallel()

	vm := createContextManagerVirtualModuleWithPartial()
	h := newTypeResolverTestHarness()
	cm := newContextManager(h.Resolver, vm)

	parentCtx := NewRootAnalysisContext(
		h.Diagnostics,
		"test/main",
		"main",
		"/virtual/main.go",
		"/test/main.pk",
	)

	node := &ast_domain.TemplateNode{
		GoAnnotations: nil,
	}

	resultCtx := cm.DeterminePartialSelfContext(node, parentCtx)
	assert.Same(t, parentCtx, resultCtx, "should return parent context when no partial info")
}

func TestDeterminePartialSelfContext_PartialNotInModule(t *testing.T) {
	t.Parallel()

	vm := createContextManagerVirtualModule()
	h := newTypeResolverTestHarness()
	cm := newContextManager(h.Resolver, vm)

	parentCtx := NewRootAnalysisContext(
		h.Diagnostics,
		"test/main",
		"main",
		"/virtual/main.go",
		"/test/main.pk",
	)

	node := &ast_domain.TemplateNode{
		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
			PartialInfo: &ast_domain.PartialInvocationInfo{
				PartialPackageName: "nonexistent_partial",
			},
		},
	}

	resultCtx := cm.DeterminePartialSelfContext(node, parentCtx)
	assert.Same(t, parentCtx, resultCtx, "should return parent context when partial not found in module")
}

func TestDeterminePartialSelfContext_PartialFound(t *testing.T) {
	t.Parallel()

	vm := createContextManagerVirtualModuleWithPartial()
	h := newTypeResolverTestHarness()
	cm := newContextManager(h.Resolver, vm)

	parentCtx := NewRootAnalysisContext(
		h.Diagnostics,
		"test/main",
		"main",
		"/virtual/main.go",
		"/test/main.pk",
	)

	node := &ast_domain.TemplateNode{
		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
			PartialInfo: &ast_domain.PartialInvocationInfo{
				PartialPackageName: "partial_123",
			},
		},
	}

	resultCtx := cm.DeterminePartialSelfContext(node, parentCtx)
	assert.NotSame(t, parentCtx, resultCtx, "should return a new partial context")
	assert.Equal(t, "test/partial", resultCtx.CurrentGoFullPackagePath)
	assert.Equal(t, "partial", resultCtx.CurrentGoPackageName)
}

func TestDetermineNodeContext_NoPartialInfo(t *testing.T) {
	t.Parallel()

	vm := createContextManagerVirtualModule()
	h := newTypeResolverTestHarness()
	cm := newContextManager(h.Resolver, vm)

	parentCtx := NewRootAnalysisContext(
		h.Diagnostics,
		"test/main",
		"main",
		"/virtual/main.go",
		"/test/main.pk",
	)

	node := &ast_domain.TemplateNode{
		GoAnnotations: nil,
	}

	resultCtx, resultPInfo := cm.DetermineNodeContext(node, parentCtx, nil, 0)

	assert.Same(t, parentCtx, resultCtx)
	assert.Nil(t, resultPInfo)
}

func TestDetermineNodeContext_WithPartialInfo(t *testing.T) {
	t.Parallel()

	vm := createContextManagerVirtualModule()
	h := newTypeResolverTestHarness()
	cm := newContextManager(h.Resolver, vm)

	parentCtx := NewRootAnalysisContext(
		h.Diagnostics,
		"test/main",
		"main",
		"/virtual/main.go",
		"/test/main.pk",
	)

	pInfo := &ast_domain.PartialInvocationInfo{
		PartialPackageName: "partial_123",
		InvocationKey:      "invkey_1",
	}

	node := &ast_domain.TemplateNode{
		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
			PartialInfo: pInfo,
		},
	}

	_, resultPInfo := cm.DetermineNodeContext(node, parentCtx, nil, 0)

	require.NotNil(t, resultPInfo)
	assert.Equal(t, "partial_123", resultPInfo.PartialPackageName)
	assert.Equal(t, "invkey_1", resultPInfo.InvocationKey)
}

func createContextManagerVirtualModuleWithPartial() *annotator_dto.VirtualModule {
	vm := createContextManagerVirtualModule()
	vm.ComponentsByHash["partial_123"] = &annotator_dto.VirtualComponent{
		HashedName:             "partial_123",
		CanonicalGoPackagePath: "test/partial",
		VirtualGoFilePath:      "/virtual/partial.go",
		RewrittenScriptAST: &goast.File{
			Name: goast.NewIdent("partial"),
		},
		Source: &annotator_dto.ParsedComponent{
			SourcePath: "/test/partial.pk",
		},
	}
	vm.ComponentsByGoPath["test/partial"] = vm.ComponentsByHash["partial_123"]
	return vm
}

func TestNewContextManager_Constructor(t *testing.T) {
	t.Parallel()

	t.Run("nil parameters accepted", func(t *testing.T) {
		t.Parallel()

		cm := newContextManager(nil, nil)

		require.NotNil(t, cm)
		assert.Nil(t, cm.typeResolver)
		assert.Nil(t, cm.virtualModule)
	})

	t.Run("stores references", func(t *testing.T) {
		t.Parallel()

		tr := &TypeResolver{}
		vm := &annotator_dto.VirtualModule{
			ComponentsByHash:   map[string]*annotator_dto.VirtualComponent{},
			ComponentsByGoPath: map[string]*annotator_dto.VirtualComponent{},
			Graph:              nil,
		}
		cm := newContextManager(tr, vm)

		require.NotNil(t, cm)
		assert.Same(t, tr, cm.typeResolver)
		assert.Same(t, vm, cm.virtualModule)
	})
}
