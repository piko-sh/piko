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
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

func TestLinkingVisitor_Enter(t *testing.T) {
	t.Run("NilNode", func(t *testing.T) {
		visitor := createTestLinkingVisitor()

		childVisitor, err := visitor.Enter(context.Background(), nil)

		if err != nil {
			t.Errorf("Expected no error for nil node, got: %v", err)
		}
		if childVisitor != nil {
			t.Error("Expected nil visitor for nil node")
		}
	})

	t.Run("SimpleElement", func(t *testing.T) {
		visitor := createTestLinkingVisitor()
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		}

		childVisitor, err := visitor.Enter(context.Background(), node)

		if err != nil {
			t.Errorf("Expected no error for simple element, got: %v", err)
		}
		if childVisitor == nil {
			t.Error("Expected child visitor, got nil")
		}
	})

	t.Run("PartialInvocation", func(t *testing.T) {
		visitor := createTestLinkingVisitor()
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				PartialInfo: &ast_domain.PartialInvocationInfo{
					PartialPackageName: "partial_123",
					PartialAlias:       "myPartial",
					PassedProps:        make(map[string]ast_domain.PropValue),
				},
			},
			Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		}

		childVisitor, err := visitor.Enter(context.Background(), node)

		if err == nil {
			t.Error("Expected error for partial without proper virtual module setup")
		}
		if childVisitor != nil {
			t.Error("Expected nil visitor when error occurs")
		}
	})
}

func TestLinkingVisitor_Exit(t *testing.T) {
	t.Run("NoOpExit", func(t *testing.T) {
		visitor := createTestLinkingVisitor()
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
		}

		err := visitor.Exit(context.Background(), node)

		if err != nil {
			t.Errorf("Expected no error from Exit, got: %v", err)
		}
	})
}

func TestLinkingVisitor_NewVisitorForChild(t *testing.T) {
	t.Run("CreatesChildVisitor", func(t *testing.T) {
		parentVisitor := createTestLinkingVisitor()
		childCtx := &AnalysisContext{
			Symbols:                  NewSymbolTable(parentVisitor.ctx.Symbols),
			Diagnostics:              parentVisitor.ctx.Diagnostics,
			CurrentGoFullPackagePath: parentVisitor.ctx.CurrentGoFullPackagePath,
			CurrentGoPackageName:     parentVisitor.ctx.CurrentGoPackageName,
			CurrentGoSourcePath:      parentVisitor.ctx.CurrentGoSourcePath,
			SFCSourcePath:            parentVisitor.ctx.SFCSourcePath,
			Logger:                   parentVisitor.ctx.Logger,
		}

		childVisitor := parentVisitor.newVisitorForChild(childCtx, "")

		if childVisitor == nil {
			t.Fatal("Expected non-nil child visitor")
		}

		linkingChildVisitor, ok := childVisitor.(*linkingVisitor)
		if !ok {
			t.Fatal("Expected child visitor to be *linkingVisitor")
		}

		if linkingChildVisitor.typeResolver != parentVisitor.typeResolver {
			t.Error("Expected child to share typeResolver")
		}
		if linkingChildVisitor.virtualModule != parentVisitor.virtualModule {
			t.Error("Expected child to share virtualModule")
		}
		if linkingChildVisitor.state != parentVisitor.state {
			t.Error("Expected child to share state")
		}

		if linkingChildVisitor.ctx == parentVisitor.ctx {
			t.Error("Expected child to have different context")
		}

		if linkingChildVisitor.depth != parentVisitor.depth+1 {
			t.Errorf("Expected child depth to be %d, got %d", parentVisitor.depth+1, linkingChildVisitor.depth)
		}
	})
}

func TestLinkingVisitor_CreateContextForNode(t *testing.T) {
	t.Run("NodeWithoutOriginAnnotations", func(t *testing.T) {
		visitor := createTestLinkingVisitor()
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
		}

		resultCtx, err := visitor.createContextForNode(context.Background(), node, nil)

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if resultCtx != visitor.ctx {
			t.Error("Expected fallback to parent context when no origin annotations")
		}
	})

	t.Run("NodeWithOriginFromNonexistentComponent", func(t *testing.T) {
		visitor := createTestLinkingVisitor()
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalPackageAlias: new("nonexistent_hash"),
				OriginalSourcePath:   new("/test/nonexistent.piko"),
			},
		}

		_, err := visitor.createContextForNode(context.Background(), node, nil)

		if err == nil {
			t.Error("Expected error for nonexistent component")
		}
	})
}

func TestLinkingSharedState(t *testing.T) {
	t.Run("InitialState", func(t *testing.T) {
		state := &linkingSharedState{
			uniqueInvocations: make(map[string]*annotator_dto.PartialInvocation),
			invocationOrder:   make([]string, 0),
		}

		if state.uniqueInvocations == nil {
			t.Error("Expected non-nil uniqueInvocations map")
		}
		if state.invocationOrder == nil {
			t.Error("Expected non-nil invocationOrder slice")
		}
		if len(state.uniqueInvocations) != 0 {
			t.Errorf("Expected empty uniqueInvocations, got %d", len(state.uniqueInvocations))
		}
		if len(state.invocationOrder) != 0 {
			t.Errorf("Expected empty invocationOrder, got %d", len(state.invocationOrder))
		}
	})

	t.Run("TracksInvocations", func(t *testing.T) {
		state := &linkingSharedState{
			uniqueInvocations: make(map[string]*annotator_dto.PartialInvocation),
			invocationOrder:   make([]string, 0),
		}
		invocation := &annotator_dto.PartialInvocation{
			InvocationKey:     "partial_key1",
			PartialAlias:      "myPartial",
			PartialHashedName: "partial_123",
		}

		state.uniqueInvocations[invocation.InvocationKey] = invocation
		state.invocationOrder = append(state.invocationOrder, invocation.InvocationKey)

		if len(state.uniqueInvocations) != 1 {
			t.Errorf("Expected 1 invocation, got %d", len(state.uniqueInvocations))
		}
		if len(state.invocationOrder) != 1 {
			t.Errorf("Expected 1 order entry, got %d", len(state.invocationOrder))
		}
		if state.invocationOrder[0] != "partial_key1" {
			t.Errorf("Expected order entry 'partial_key1', got '%s'", state.invocationOrder[0])
		}
	})

	t.Run("PreventsDuplicateKeys", func(t *testing.T) {
		state := &linkingSharedState{
			uniqueInvocations: make(map[string]*annotator_dto.PartialInvocation),
			invocationOrder:   make([]string, 0),
		}
		key := "partial_key1"

		if _, exists := state.uniqueInvocations[key]; !exists {
			state.uniqueInvocations[key] = &annotator_dto.PartialInvocation{InvocationKey: key}
			state.invocationOrder = append(state.invocationOrder, key)
		}
		if _, exists := state.uniqueInvocations[key]; !exists {
			state.uniqueInvocations[key] = &annotator_dto.PartialInvocation{InvocationKey: key}
			state.invocationOrder = append(state.invocationOrder, key)
		}

		if len(state.uniqueInvocations) != 1 {
			t.Errorf("Expected 1 unique invocation, got %d", len(state.uniqueInvocations))
		}
		if len(state.invocationOrder) != 1 {
			t.Errorf("Expected 1 order entry, got %d", len(state.invocationOrder))
		}
	})
}

func TestLinkingVisitor_IsPartialInvocation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		node     *ast_domain.TemplateNode
		name     string
		expected bool
	}{
		{
			name: "node with PartialInfo is partial invocation",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{
						PartialAlias: "card",
					},
				},
			},
			expected: true,
		},
		{
			name: "node with GoAnnotations but no PartialInfo is not partial",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: nil,
				},
			},
			expected: false,
		},
		{
			name: "node without GoAnnotations is not partial",
			node: &ast_domain.TemplateNode{
				GoAnnotations: nil,
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			visitor := &linkingVisitor{}
			result := visitor.isPartialInvocation(tc.node)

			if result != tc.expected {
				t.Errorf("Expected isPartialInvocation to return %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestLinkingVisitor_HandleSlottedContentContextSwitch(t *testing.T) {
	t.Parallel()

	t.Run("returns current context for node without GoAnnotations", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitor()
		node := &ast_domain.TemplateNode{
			NodeType:      ast_domain.NodeElement,
			TagName:       "div",
			GoAnnotations: nil,
		}

		result := visitor.handleSlottedContentContextSwitch(context.Background(), node)

		if result != visitor.ctx {
			t.Error("Expected current context to be returned when no GoAnnotations")
		}
	})

	t.Run("returns current context when OriginalPackageAlias is nil and no PartialInfo", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitor()
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalPackageAlias: nil,
				PartialInfo:          nil,
			},
		}

		result := visitor.handleSlottedContentContextSwitch(context.Background(), node)

		if result != visitor.ctx {
			t.Error("Expected current context when OriginalPackageAlias is nil and no PartialInfo")
		}
	})
}

func TestLinkingVisitor_HandleForDirective(t *testing.T) {
	t.Parallel()

	t.Run("returns child context when no p-for directive", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitor()
		childCtx := visitor.ctx.ForChildScope()
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			DirFor:   nil,
		}

		result := visitor.handleForDirective(context.Background(), node, visitor.ctx, childCtx, "")

		if result != childCtx {
			t.Error("Expected child context to be returned when DirFor is nil")
		}
	})

	t.Run("returns child context when DirFor expression is not ForInExpr", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitor()
		childCtx := visitor.ctx.ForChildScope()
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			DirFor: &ast_domain.Directive{
				Type:       ast_domain.DirectiveFor,
				Expression: &ast_domain.Identifier{Name: "invalid"},
			},
		}

		result := visitor.handleForDirective(context.Background(), node, visitor.ctx, childCtx, "")

		if result != childCtx {
			t.Error("Expected child context when expression is not ForInExpr")
		}
	})
}

func TestLinkingVisitor_StoreUniqueInvocation(t *testing.T) {
	t.Parallel()

	t.Run("stores new invocation", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitor()
		node := &ast_domain.TemplateNode{
			TagName:  "div",
			Location: ast_domain.Location{Line: 5, Column: 1},
		}
		partialInfo := &ast_domain.PartialInvocationInfo{
			PartialAlias:        "card",
			PartialPackageName:  "partial_card",
			InvokerPackageAlias: "page_hash",
		}
		finalised := &finalisedInvocationData{
			canonicalKey:   "card_key_1",
			canonicalProps: map[string]ast_domain.PropValue{},
			dependsOn:      nil,
		}

		visitor.storeUniqueInvocation(context.Background(), node, partialInfo, finalised)

		if len(visitor.state.uniqueInvocations) != 1 {
			t.Errorf("Expected 1 invocation, got %d", len(visitor.state.uniqueInvocations))
		}
		if _, exists := visitor.state.uniqueInvocations["card_key_1"]; !exists {
			t.Error("Expected invocation to be stored with key 'card_key_1'")
		}
		if len(visitor.state.invocationOrder) != 1 {
			t.Errorf("Expected 1 order entry, got %d", len(visitor.state.invocationOrder))
		}
		if partialInfo.InvocationKey != "card_key_1" {
			t.Errorf("Expected partialInfo.InvocationKey to be 'card_key_1', got %q", partialInfo.InvocationKey)
		}
	})

	t.Run("does not overwrite existing invocation with same key", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitor()
		visitor.state.uniqueInvocations["existing_key"] = &annotator_dto.PartialInvocation{
			InvocationKey: "existing_key",
			PartialAlias:  "original",
		}
		visitor.state.invocationOrder = append(visitor.state.invocationOrder, "existing_key")

		node := &ast_domain.TemplateNode{TagName: "div"}
		partialInfo := &ast_domain.PartialInvocationInfo{
			PartialAlias: "different",
		}
		finalised := &finalisedInvocationData{
			canonicalKey:   "existing_key",
			canonicalProps: map[string]ast_domain.PropValue{},
		}

		visitor.storeUniqueInvocation(context.Background(), node, partialInfo, finalised)

		if len(visitor.state.uniqueInvocations) != 1 {
			t.Errorf("Expected 1 invocation, got %d", len(visitor.state.uniqueInvocations))
		}
		stored := visitor.state.uniqueInvocations["existing_key"]
		if stored.PartialAlias != "original" {
			t.Errorf("Expected original alias preserved, got %q", stored.PartialAlias)
		}
		if len(visitor.state.invocationOrder) != 1 {
			t.Errorf("Expected 1 order entry (no duplicate), got %d", len(visitor.state.invocationOrder))
		}
	})
}

func TestLinkingVisitor_SwitchToInvokerContext(t *testing.T) {
	t.Parallel()

	t.Run("returns current context when current component not found in GoPath map", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitor()
		visitor.virtualModule.ComponentsByGoPath = nil

		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				PartialInfo: &ast_domain.PartialInvocationInfo{
					InvokerPackageAlias: "invoker_hash",
				},
			},
		}

		result := visitor.switchToInvokerContext(context.Background(), node)

		assert.Equal(t, visitor.ctx, result)
	})

	t.Run("returns current context when invoker hash matches current component", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitorWithGoPath()
		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				PartialInfo: &ast_domain.PartialInvocationInfo{
					InvokerPackageAlias: "test_hash",
				},
			},
		}

		result := visitor.switchToInvokerContext(context.Background(), node)

		assert.Equal(t, visitor.ctx, result)
	})

	t.Run("returns current context when invoker component not found by hash", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitorWithGoPath()
		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				PartialInfo: &ast_domain.PartialInvocationInfo{
					InvokerPackageAlias: "nonexistent_invoker_hash",
				},
			},
		}

		result := visitor.switchToInvokerContext(context.Background(), node)

		assert.Equal(t, visitor.ctx, result)
	})

}

func TestLinkingVisitor_SwitchToOriginContext(t *testing.T) {
	t.Parallel()

	t.Run("returns current context when current component not found in GoPath map", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitor()
		visitor.virtualModule.ComponentsByGoPath = nil
		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalSourcePath: new("/test/origin.piko"),
			},
		}

		result := visitor.switchToOriginContext(context.Background(), node, "origin_hash")

		assert.Equal(t, visitor.ctx, result)
	})

	t.Run("returns current context when origin hash matches current component", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitorWithGoPath()
		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalSourcePath: new("/test/file.piko"),
			},
		}

		result := visitor.switchToOriginContext(context.Background(), node, "test_hash")

		assert.Equal(t, visitor.ctx, result)
	})

	t.Run("returns current context when origin component not found by hash", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitorWithGoPath()
		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalSourcePath: new("/test/unknown.piko"),
			},
		}

		result := visitor.switchToOriginContext(context.Background(), node, "nonexistent_hash")

		assert.Equal(t, visitor.ctx, result)
	})

}

func TestLinkingVisitor_PrepareLinkerContext(t *testing.T) {
	t.Parallel()

	t.Run("returns linker context when no DirFor", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitor()
		node := &ast_domain.TemplateNode{
			DirFor: nil,
		}
		partialInfo := &ast_domain.PartialInvocationInfo{}

		result := visitor.prepareLinkerContext(context.Background(), node, visitor.ctx, partialInfo)

		assert.Equal(t, visitor.ctx, result)
	})

	t.Run("returns linker context when DirFor expression is not ForInExpr", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitor()
		node := &ast_domain.TemplateNode{
			DirFor: &ast_domain.Directive{
				Type:       ast_domain.DirectiveFor,
				Expression: &ast_domain.Identifier{Name: "notForIn"},
			},
		}
		partialInfo := &ast_domain.PartialInvocationInfo{}

		result := visitor.prepareLinkerContext(context.Background(), node, visitor.ctx, partialInfo)

		assert.Equal(t, visitor.ctx, result)
	})
}

func TestLinkingVisitor_HandleSlottedContentContextSwitch_WithPartialInfo(t *testing.T) {
	t.Parallel()

	t.Run("delegates to switchToInvokerContext when PartialInfo is set and invoker matches current", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitorWithGoPath()

		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				PartialInfo: &ast_domain.PartialInvocationInfo{
					InvokerPackageAlias: "test_hash",
				},
			},
		}

		result := visitor.handleSlottedContentContextSwitch(context.Background(), node)

		assert.Equal(t, visitor.ctx, result, "when invoker matches current component, context should remain the same")
	})
}

func TestLinkingVisitor_HandleSlottedContentContextSwitch_WithOriginalPackageAlias(t *testing.T) {
	t.Parallel()

	t.Run("returns current context when OriginalPackageAlias matches current component", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitorWithGoPath()

		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalPackageAlias: new("test_hash"),
				OriginalSourcePath:   new("/test/file.piko"),
			},
		}

		result := visitor.handleSlottedContentContextSwitch(context.Background(), node)

		assert.Equal(t, visitor.ctx, result, "when origin matches current component, context should remain the same")
	})

	t.Run("returns current context when OriginalPackageAlias component not found", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitorWithGoPath()

		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalPackageAlias: new("nonexistent_hash"),
				OriginalSourcePath:   new("/test/missing.piko"),
			},
		}

		result := visitor.handleSlottedContentContextSwitch(context.Background(), node)

		assert.Equal(t, visitor.ctx, result, "when origin component not found, context should remain the same")
	})
}

func TestLinkingVisitor_CreateContextForNode_WithOriginAnnotations(t *testing.T) {
	t.Parallel()

	t.Run("returns error when component hash not found", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitorWithGoPath()
		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalPackageAlias: new("nonexistent_hash"),
				OriginalSourcePath:   new("/test/missing.piko"),
			},
		}

		_, err := visitor.createContextForNode(context.Background(), node, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "nonexistent_hash")
	})

	t.Run("falls back to parent context with missing GoAnnotations", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitorWithGoPath()
		node := &ast_domain.TemplateNode{
			TagName:       "div",
			GoAnnotations: nil,
		}

		resultCtx, err := visitor.createContextForNode(context.Background(), node, nil)

		require.NoError(t, err)
		assert.Equal(t, visitor.ctx, resultCtx)
	})

	t.Run("falls back to parent context with nil OriginalPackageAlias", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitorWithGoPath()
		node := &ast_domain.TemplateNode{
			TagName: "div",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalPackageAlias: nil,
				OriginalSourcePath:   nil,
			},
		}

		resultCtx, err := visitor.createContextForNode(context.Background(), node, nil)

		require.NoError(t, err)
		assert.Equal(t, visitor.ctx, resultCtx)
	})

	t.Run("falls back to parent context with nil OriginalSourcePath", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitorWithGoPath()
		node := &ast_domain.TemplateNode{
			TagName: "div",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalPackageAlias: new("test_hash"),
				OriginalSourcePath:   nil,
			},
		}

		resultCtx, err := visitor.createContextForNode(context.Background(), node, nil)

		require.NoError(t, err)
		assert.Equal(t, visitor.ctx, resultCtx)
	})
}

func TestLinkingVisitor_NewVisitorForChild_InvocationKey(t *testing.T) {
	t.Parallel()

	t.Run("propagates invocation key to child visitor", func(t *testing.T) {
		t.Parallel()

		parentVisitor := createTestLinkingVisitor()
		childCtx := parentVisitor.ctx.ForChildScope()

		childVisitor := parentVisitor.newVisitorForChild(childCtx, "my_invocation_key")

		linkingChild, ok := childVisitor.(*linkingVisitor)
		require.True(t, ok)
		assert.Equal(t, "my_invocation_key", linkingChild.currentInvocationKey)
	})

	t.Run("empty invocation key is propagated", func(t *testing.T) {
		t.Parallel()

		parentVisitor := createTestLinkingVisitor()
		childCtx := parentVisitor.ctx.ForChildScope()

		childVisitor := parentVisitor.newVisitorForChild(childCtx, "")

		linkingChild, ok := childVisitor.(*linkingVisitor)
		require.True(t, ok)
		assert.Equal(t, "", linkingChild.currentInvocationKey)
	})
}

func TestLinkingVisitor_Enter_SimpleElementWithAnnotations(t *testing.T) {
	t.Parallel()

	t.Run("simple element with GoAnnotations but no PartialInfo", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitor()
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				PartialInfo: nil,
			},
			Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		}

		childVisitor, err := visitor.Enter(context.Background(), node)

		require.NoError(t, err)
		assert.NotNil(t, childVisitor)
	})
}

func TestLinkingVisitor_HandleForDirective_WithValidForExpr(t *testing.T) {
	t.Parallel()

	t.Run("returns child context when DirFor has non-ForInExpr", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitor()
		childCtx := visitor.ctx.ForChildScope()
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "li",
			DirFor: &ast_domain.Directive{
				Type:       ast_domain.DirectiveFor,
				Expression: &ast_domain.StringLiteral{Value: "invalid"},
			},
		}

		result := visitor.handleForDirective(context.Background(), node, visitor.ctx, childCtx, "")

		assert.Equal(t, childCtx, result)
	})
}

func createTestLinkingVisitorWithGoPath() *linkingVisitor {
	diagnostics := make([]*ast_domain.Diagnostic, 0)
	ctx := &AnalysisContext{
		Symbols:                  NewSymbolTable(nil),
		Diagnostics:              &diagnostics,
		CurrentGoFullPackagePath: "test/pkg",
		CurrentGoPackageName:     "pkg",
		CurrentGoSourcePath:      "/test/file.go",
		SFCSourcePath:            "/test/file.piko",
		Logger:                   logger_domain.GetLogger("test"),
	}

	testComp := &annotator_dto.VirtualComponent{
		HashedName:             "test_hash",
		CanonicalGoPackagePath: "test/pkg",
		VirtualGoFilePath:      "/virtual/test.go",
		RewrittenScriptAST: &goast.File{
			Name: goast.NewIdent("pkg"),
		},
		Source: &annotator_dto.ParsedComponent{
			SourcePath: "/test/file.piko",
		},
	}

	vm := &annotator_dto.VirtualModule{
		ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
			"test_hash": testComp,
		},
		ComponentsByGoPath: map[string]*annotator_dto.VirtualComponent{
			"test/pkg": testComp,
		},
	}

	state := &linkingSharedState{
		uniqueInvocations: make(map[string]*annotator_dto.PartialInvocation),
		invocationOrder:   make([]string, 0),
	}

	return &linkingVisitor{

		typeResolver: &TypeResolver{
			inspector: &inspector_domain.MockTypeQuerier{
				GetImportsForFileFunc: func(_, _ string) map[string]string {
					return map[string]string{}
				},
			},
		},
		virtualModule: vm,
		diagnostics:   &diagnostics,
		ctx:           ctx,
		depth:         0,
		state:         state,
	}
}

func TestLinkingVisitor_SwitchToInvokerContext_FullSwitch(t *testing.T) {
	t.Parallel()

	t.Run("switches context when invoker is different from current component", func(t *testing.T) {
		t.Parallel()

		invokerComp := &annotator_dto.VirtualComponent{
			HashedName:             "invoker_hash",
			CanonicalGoPackagePath: "invoker/pkg",
			VirtualGoFilePath:      "/virtual/invoker.go",
			RewrittenScriptAST: &goast.File{
				Name: goast.NewIdent("invoker_pkg"),
			},
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/test/invoker.piko",
			},
		}

		currentComp := &annotator_dto.VirtualComponent{
			HashedName:             "current_hash",
			CanonicalGoPackagePath: "current/pkg",
			VirtualGoFilePath:      "/virtual/current.go",
			RewrittenScriptAST: &goast.File{
				Name: goast.NewIdent("current_pkg"),
			},
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/test/current.piko",
			},
		}

		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := &AnalysisContext{
			Symbols:                  NewSymbolTable(nil),
			Diagnostics:              &diagnostics,
			CurrentGoFullPackagePath: "current/pkg",
			CurrentGoPackageName:     "current_pkg",
			CurrentGoSourcePath:      "/virtual/current.go",
			SFCSourcePath:            "/test/current.piko",
			Logger:                   logger_domain.GetLogger("test"),
		}

		vm := &annotator_dto.VirtualModule{
			ComponentsByGoPath: map[string]*annotator_dto.VirtualComponent{
				"current/pkg": currentComp,
			},
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				"current_hash": currentComp,
				"invoker_hash": invokerComp,
			},
		}

		state := &linkingSharedState{
			uniqueInvocations: make(map[string]*annotator_dto.PartialInvocation),
			invocationOrder:   make([]string, 0),
		}

		mockInspector := &inspector_domain.MockTypeQuerier{
			GetImportsForFileFunc: func(_, _ string) map[string]string {
				return map[string]string{}
			},
			GetAllPackagesFunc: func() map[string]*inspector_dto.Package {
				return map[string]*inspector_dto.Package{}
			},
		}

		visitor := &linkingVisitor{

			typeResolver:  &TypeResolver{inspector: mockInspector},
			virtualModule: vm,
			diagnostics:   &diagnostics,
			ctx:           ctx,
			depth:         0,
			state:         state,
		}

		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				PartialInfo: &ast_domain.PartialInvocationInfo{
					InvokerPackageAlias: "invoker_hash",
				},
			},
		}

		result := visitor.switchToInvokerContext(context.Background(), node)

		assert.NotEqual(t, visitor.ctx, result, "context should be switched")
		assert.Equal(t, "invoker/pkg", result.CurrentGoFullPackagePath)
		assert.Equal(t, "invoker_pkg", result.CurrentGoPackageName)
		assert.Equal(t, "/virtual/invoker.go", result.CurrentGoSourcePath)
		assert.Equal(t, "/test/invoker.piko", result.SFCSourcePath)
	})

	t.Run("switches context when invoker has nil RewrittenScriptAST and no script", func(t *testing.T) {
		t.Parallel()

		invokerComp := &annotator_dto.VirtualComponent{
			HashedName:             "invoker_hash",
			CanonicalGoPackagePath: "invoker/pkg",
			VirtualGoFilePath:      "/virtual/invoker.go",
			RewrittenScriptAST:     nil,
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/test/invoker.piko",
			},
		}

		currentComp := &annotator_dto.VirtualComponent{
			HashedName:             "current_hash",
			CanonicalGoPackagePath: "current/pkg",
			VirtualGoFilePath:      "/virtual/current.go",
			RewrittenScriptAST: &goast.File{
				Name: goast.NewIdent("current_pkg"),
			},
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/test/current.piko",
			},
		}

		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := &AnalysisContext{
			Symbols:                  NewSymbolTable(nil),
			Diagnostics:              &diagnostics,
			CurrentGoFullPackagePath: "current/pkg",
			CurrentGoPackageName:     "current_pkg",
			CurrentGoSourcePath:      "/virtual/current.go",
			SFCSourcePath:            "/test/current.piko",
			Logger:                   logger_domain.GetLogger("test"),
		}

		vm := &annotator_dto.VirtualModule{
			ComponentsByGoPath: map[string]*annotator_dto.VirtualComponent{
				"current/pkg": currentComp,
			},
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				"current_hash": currentComp,
				"invoker_hash": invokerComp,
			},
		}

		state := &linkingSharedState{
			uniqueInvocations: make(map[string]*annotator_dto.PartialInvocation),
			invocationOrder:   make([]string, 0),
		}

		mockInspector := &inspector_domain.MockTypeQuerier{
			GetImportsForFileFunc: func(_, _ string) map[string]string {
				return map[string]string{}
			},
			GetAllPackagesFunc: func() map[string]*inspector_dto.Package {
				return map[string]*inspector_dto.Package{}
			},
		}

		visitor := &linkingVisitor{

			typeResolver:  &TypeResolver{inspector: mockInspector},
			virtualModule: vm,
			diagnostics:   &diagnostics,
			ctx:           ctx,
			depth:         0,
			state:         state,
		}

		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				PartialInfo: &ast_domain.PartialInvocationInfo{
					InvokerPackageAlias: "invoker_hash",
				},
			},
		}

		result := visitor.switchToInvokerContext(context.Background(), node)

		assert.NotEqual(t, visitor.ctx, result)
		assert.Equal(t, "invoker/pkg", result.CurrentGoFullPackagePath)
		assert.Equal(t, "", result.CurrentGoPackageName, "should be empty when RewrittenScriptAST is nil")
		assert.Equal(t, "/test/invoker.piko", result.SFCSourcePath)
	})
}

func TestLinkingVisitor_SwitchToOriginContext_FullSwitch(t *testing.T) {
	t.Parallel()

	t.Run("switches context to origin component", func(t *testing.T) {
		t.Parallel()

		originComp := &annotator_dto.VirtualComponent{
			HashedName:             "origin_hash",
			CanonicalGoPackagePath: "origin/pkg",
			VirtualGoFilePath:      "/virtual/origin.go",
			RewrittenScriptAST: &goast.File{
				Name: goast.NewIdent("origin_pkg"),
			},
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/test/origin.piko",
			},
		}

		currentComp := &annotator_dto.VirtualComponent{
			HashedName:             "current_hash",
			CanonicalGoPackagePath: "current/pkg",
			VirtualGoFilePath:      "/virtual/current.go",
			RewrittenScriptAST: &goast.File{
				Name: goast.NewIdent("current_pkg"),
			},
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/test/current.piko",
			},
		}

		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := &AnalysisContext{
			Symbols:                  NewSymbolTable(nil),
			Diagnostics:              &diagnostics,
			CurrentGoFullPackagePath: "current/pkg",
			CurrentGoPackageName:     "current_pkg",
			CurrentGoSourcePath:      "/virtual/current.go",
			SFCSourcePath:            "/test/current.piko",
			Logger:                   logger_domain.GetLogger("test"),
		}

		vm := &annotator_dto.VirtualModule{
			ComponentsByGoPath: map[string]*annotator_dto.VirtualComponent{
				"current/pkg": currentComp,
			},
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				"current_hash": currentComp,
				"origin_hash":  originComp,
			},
		}

		state := &linkingSharedState{
			uniqueInvocations: make(map[string]*annotator_dto.PartialInvocation),
			invocationOrder:   make([]string, 0),
		}

		visitor := &linkingVisitor{

			typeResolver: &TypeResolver{
				inspector: &inspector_domain.MockTypeQuerier{
					GetImportsForFileFunc: func(_, _ string) map[string]string {
						return map[string]string{}
					},
				},
			},
			virtualModule: vm,
			diagnostics:   &diagnostics,
			ctx:           ctx,
			depth:         0,
			state:         state,
		}

		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalSourcePath: new("/test/origin.piko"),
			},
		}

		result := visitor.switchToOriginContext(context.Background(), node, "origin_hash")

		assert.NotEqual(t, visitor.ctx, result, "context should be switched")
		assert.Equal(t, "origin/pkg", result.CurrentGoFullPackagePath)
		assert.Equal(t, "origin_pkg", result.CurrentGoPackageName)
		assert.Equal(t, "/virtual/origin.go", result.CurrentGoSourcePath)
		assert.Equal(t, "/test/origin.piko", result.SFCSourcePath)
	})

	t.Run("switches context when origin has nil RewrittenScriptAST", func(t *testing.T) {
		t.Parallel()

		originComp := &annotator_dto.VirtualComponent{
			HashedName:             "origin_hash",
			CanonicalGoPackagePath: "origin/pkg",
			VirtualGoFilePath:      "/virtual/origin.go",
			RewrittenScriptAST:     nil,
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/test/origin.piko",
			},
		}

		currentComp := &annotator_dto.VirtualComponent{
			HashedName:             "current_hash",
			CanonicalGoPackagePath: "current/pkg",
			VirtualGoFilePath:      "/virtual/current.go",
			RewrittenScriptAST: &goast.File{
				Name: goast.NewIdent("current_pkg"),
			},
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/test/current.piko",
			},
		}

		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := &AnalysisContext{
			Symbols:                  NewSymbolTable(nil),
			Diagnostics:              &diagnostics,
			CurrentGoFullPackagePath: "current/pkg",
			CurrentGoPackageName:     "current_pkg",
			CurrentGoSourcePath:      "/virtual/current.go",
			SFCSourcePath:            "/test/current.piko",
			Logger:                   logger_domain.GetLogger("test"),
		}

		vm := &annotator_dto.VirtualModule{
			ComponentsByGoPath: map[string]*annotator_dto.VirtualComponent{
				"current/pkg": currentComp,
			},
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				"current_hash": currentComp,
				"origin_hash":  originComp,
			},
		}

		state := &linkingSharedState{
			uniqueInvocations: make(map[string]*annotator_dto.PartialInvocation),
			invocationOrder:   make([]string, 0),
		}

		visitor := &linkingVisitor{

			typeResolver: &TypeResolver{
				inspector: &inspector_domain.MockTypeQuerier{
					GetImportsForFileFunc: func(_, _ string) map[string]string {
						return map[string]string{}
					},
				},
			},
			virtualModule: vm,
			diagnostics:   &diagnostics,
			ctx:           ctx,
			depth:         0,
			state:         state,
		}

		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalSourcePath: new("/test/origin.piko"),
			},
		}

		result := visitor.switchToOriginContext(context.Background(), node, "origin_hash")

		assert.NotEqual(t, visitor.ctx, result)
		assert.Equal(t, "origin/pkg", result.CurrentGoFullPackagePath)
		assert.Equal(t, "", result.CurrentGoPackageName, "should be empty when RewrittenScriptAST is nil")
	})
}

func TestLinkingVisitor_HandleForDirective_WithForInExpr(t *testing.T) {
	t.Parallel()

	t.Run("creates nested scope with item and index variables", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitor()
		visitor.ctx.Symbols.Define(Symbol{
			Name:           "items",
			CodeGenVarName: "items",
			TypeInfo:       newSimpleTypeInfo(&goast.ArrayType{Elt: goast.NewIdent("string")}),
		})
		childCtx := visitor.ctx.ForChildScope()
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "li",
			DirFor: &ast_domain.Directive{
				Type: ast_domain.DirectiveFor,
				Expression: &ast_domain.ForInExpression{
					ItemVariable:  &ast_domain.Identifier{Name: "item"},
					IndexVariable: &ast_domain.Identifier{Name: "index"},
					Collection:    &ast_domain.Identifier{Name: "items"},
				},
				Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
			},
		}

		result := visitor.handleForDirective(context.Background(), node, visitor.ctx, childCtx, "")

		assert.NotEqual(t, childCtx, result, "should return new nested scope context")
		_, itemFound := result.Symbols.Find("item")
		assert.True(t, itemFound, "item variable should be defined in nested scope")
		_, idxFound := result.Symbols.Find("index")
		assert.True(t, idxFound, "index variable should be defined in nested scope")
	})

	t.Run("creates nested scope with only item variable", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitor()
		visitor.ctx.Symbols.Define(Symbol{
			Name:           "data",
			CodeGenVarName: "data",
			TypeInfo:       newSimpleTypeInfo(&goast.ArrayType{Elt: goast.NewIdent("string")}),
		})
		childCtx := visitor.ctx.ForChildScope()
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "li",
			DirFor: &ast_domain.Directive{
				Type: ast_domain.DirectiveFor,
				Expression: &ast_domain.ForInExpression{
					ItemVariable:  &ast_domain.Identifier{Name: "val"},
					IndexVariable: nil,
					Collection:    &ast_domain.Identifier{Name: "data"},
				},
				Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
			},
		}

		result := visitor.handleForDirective(context.Background(), node, visitor.ctx, childCtx, "")

		assert.NotEqual(t, childCtx, result, "should return new nested scope context")
		_, valFound := result.Symbols.Find("val")
		assert.True(t, valFound, "item variable should be defined in nested scope")
	})

	t.Run("creates nested scope with only index variable", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitor()
		visitor.ctx.Symbols.Define(Symbol{
			Name:           "data",
			CodeGenVarName: "data",
			TypeInfo:       newSimpleTypeInfo(&goast.ArrayType{Elt: goast.NewIdent("string")}),
		})
		childCtx := visitor.ctx.ForChildScope()
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "li",
			DirFor: &ast_domain.Directive{
				Type: ast_domain.DirectiveFor,
				Expression: &ast_domain.ForInExpression{
					ItemVariable:  nil,
					IndexVariable: &ast_domain.Identifier{Name: "i"},
					Collection:    &ast_domain.Identifier{Name: "data"},
				},
				Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
			},
		}

		result := visitor.handleForDirective(context.Background(), node, visitor.ctx, childCtx, "")

		assert.NotEqual(t, childCtx, result, "should return new nested scope context")
		_, iFound := result.Symbols.Find("i")
		assert.True(t, iFound, "index variable should be defined in nested scope")
	})
}

func TestLinkingVisitor_CreateContextForNode_WithValidHash(t *testing.T) {
	t.Parallel()

	t.Run("creates new context when component found by hash", func(t *testing.T) {
		t.Parallel()

		partialComp := &annotator_dto.VirtualComponent{
			HashedName:             "partial_hash",
			CanonicalGoPackagePath: "partial/pkg",
			VirtualGoFilePath:      "/virtual/partial.go",
			RewrittenScriptAST: &goast.File{
				Name: goast.NewIdent("partial_pkg"),
			},
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/test/partial.piko",
			},
		}

		currentComp := &annotator_dto.VirtualComponent{
			HashedName:             "current_hash",
			CanonicalGoPackagePath: "current/pkg",
			VirtualGoFilePath:      "/virtual/current.go",
			RewrittenScriptAST: &goast.File{
				Name: goast.NewIdent("current_pkg"),
			},
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/test/current.piko",
			},
		}

		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := &AnalysisContext{
			Symbols:                  NewSymbolTable(nil),
			Diagnostics:              &diagnostics,
			CurrentGoFullPackagePath: "current/pkg",
			CurrentGoPackageName:     "current_pkg",
			CurrentGoSourcePath:      "/virtual/current.go",
			SFCSourcePath:            "/test/current.piko",
			Logger:                   logger_domain.GetLogger("test"),
		}

		vm := &annotator_dto.VirtualModule{
			ComponentsByGoPath: map[string]*annotator_dto.VirtualComponent{
				"current/pkg": currentComp,
			},
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				"current_hash": currentComp,
				"partial_hash": partialComp,
			},
		}

		state := &linkingSharedState{
			uniqueInvocations: make(map[string]*annotator_dto.PartialInvocation),
			invocationOrder:   make([]string, 0),
		}

		visitor := &linkingVisitor{

			typeResolver: &TypeResolver{
				inspector: &inspector_domain.MockTypeQuerier{
					GetImportsForFileFunc: func(_, _ string) map[string]string {
						return map[string]string{}
					},
				},
			},
			virtualModule: vm,
			diagnostics:   &diagnostics,
			ctx:           ctx,
			depth:         0,
			state:         state,
		}

		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalPackageAlias: new("partial_hash"),
				OriginalSourcePath:   new("/test/partial.piko"),
			},
		}

		resultCtx, err := visitor.createContextForNode(context.Background(), node, nil)

		require.NoError(t, err)
		assert.NotEqual(t, visitor.ctx, resultCtx, "context should be new for the partial")
		assert.Equal(t, "partial/pkg", resultCtx.CurrentGoFullPackagePath)
		assert.Equal(t, "partial_pkg", resultCtx.CurrentGoPackageName)
		assert.Equal(t, "/virtual/partial.go", resultCtx.CurrentGoSourcePath)
		assert.Equal(t, "/test/partial.piko", resultCtx.SFCSourcePath)
	})
}

func TestLinkingVisitor_PrepareLinkerContext_WithForInExpr(t *testing.T) {
	t.Parallel()

	t.Run("creates nested scope with loop variables for p-for and partial on same node", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitor()
		visitor.ctx.Symbols.Define(Symbol{
			Name:           "items",
			CodeGenVarName: "items",
			TypeInfo:       newSimpleTypeInfo(&goast.ArrayType{Elt: goast.NewIdent("string")}),
		})
		node := &ast_domain.TemplateNode{
			DirFor: &ast_domain.Directive{
				Type: ast_domain.DirectiveFor,
				Expression: &ast_domain.ForInExpression{
					ItemVariable:  &ast_domain.Identifier{Name: "item"},
					IndexVariable: &ast_domain.Identifier{Name: "index"},
					Collection:    &ast_domain.Identifier{Name: "items"},
				},
				Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
			},
		}
		pInfo := &ast_domain.PartialInvocationInfo{}

		result := visitor.prepareLinkerContext(context.Background(), node, visitor.ctx, pInfo)

		assert.NotEqual(t, visitor.ctx, result, "should return new nested scope context")
		_, itemFound := result.Symbols.Find("item")
		assert.True(t, itemFound, "item variable should be defined in nested scope")
		_, idxFound := result.Symbols.Find("index")
		assert.True(t, idxFound, "index variable should be defined in nested scope")
	})

	t.Run("creates nested scope with only item variable when index is nil", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitor()
		visitor.ctx.Symbols.Define(Symbol{
			Name:           "entries",
			CodeGenVarName: "entries",
			TypeInfo:       newSimpleTypeInfo(&goast.ArrayType{Elt: goast.NewIdent("string")}),
		})
		node := &ast_domain.TemplateNode{
			DirFor: &ast_domain.Directive{
				Type: ast_domain.DirectiveFor,
				Expression: &ast_domain.ForInExpression{
					ItemVariable:  &ast_domain.Identifier{Name: "entry"},
					IndexVariable: nil,
					Collection:    &ast_domain.Identifier{Name: "entries"},
				},
				Location: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
			},
		}
		pInfo := &ast_domain.PartialInvocationInfo{}

		result := visitor.prepareLinkerContext(context.Background(), node, visitor.ctx, pInfo)

		assert.NotEqual(t, visitor.ctx, result)
		_, entryFound := result.Symbols.Find("entry")
		assert.True(t, entryFound, "item variable should be defined")
	})
}

func createTestLinkingVisitor() *linkingVisitor {
	diagnostics := make([]*ast_domain.Diagnostic, 0)
	ctx := &AnalysisContext{
		Symbols:                  NewSymbolTable(nil),
		Diagnostics:              &diagnostics,
		CurrentGoFullPackagePath: "test/pkg",
		CurrentGoPackageName:     "pkg",
		CurrentGoSourcePath:      "/test/file.go",
		SFCSourcePath:            "/test/file.piko",
		Logger:                   logger_domain.GetLogger("test"),
	}

	vm := &annotator_dto.VirtualModule{
		ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
			"test_hash": {
				HashedName:             "test_hash",
				CanonicalGoPackagePath: "test/pkg",
				VirtualGoFilePath:      "/virtual/test.go",
				RewrittenScriptAST: &goast.File{
					Name: goast.NewIdent("pkg"),
				},
				Source: &annotator_dto.ParsedComponent{
					SourcePath: "/test/file.piko",
				},
			},
		},
	}

	state := &linkingSharedState{
		uniqueInvocations: make(map[string]*annotator_dto.PartialInvocation),
		invocationOrder:   make([]string, 0),
	}

	return &linkingVisitor{

		typeResolver: &TypeResolver{
			inspector: &inspector_domain.MockTypeQuerier{
				GetImportsForFileFunc: func(_, _ string) map[string]string {
					return map[string]string{}
				},
			},
		},
		virtualModule: vm,
		diagnostics:   &diagnostics,
		ctx:           ctx,
		depth:         0,
		state:         state,
	}
}

func TestLinkingVisitor_ProcessPartialLinking(t *testing.T) {
	t.Parallel()

	t.Run("returns error when partial component not found", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitor()
		pInfo := &ast_domain.PartialInvocationInfo{
			PartialPackageName: "nonexistent_hash",
			PartialAlias:       "missing",
			PassedProps:        make(map[string]ast_domain.PropValue),
		}

		_, err := visitor.processPartialLinking(context.Background(), pInfo, visitor.ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "nonexistent_hash")
	})

	t.Run("succeeds for valid partial with no props", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitorForPartialLinking()

		pInfo := &ast_domain.PartialInvocationInfo{
			PartialPackageName: "partial_abc",
			PartialAlias:       "Card",
			PassedProps:        make(map[string]ast_domain.PropValue),
			Location:           ast_domain.Location{Line: 10, Column: 1},
		}

		finalised, err := visitor.processPartialLinking(context.Background(), pInfo, visitor.ctx)

		require.NoError(t, err)
		require.NotNil(t, finalised)
		assert.NotEmpty(t, finalised.canonicalKey)
	})

	t.Run("succeeds for valid partial with passed props", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitorForPartialLinking()

		pInfo := &ast_domain.PartialInvocationInfo{
			PartialPackageName: "partial_abc",
			PartialAlias:       "Card",
			PassedProps: map[string]ast_domain.PropValue{
				"title": {
					Expression:   &ast_domain.StringLiteral{Value: "Hello"},
					Location:     ast_domain.Location{Line: 10, Column: 5},
					NameLocation: ast_domain.Location{Line: 10, Column: 1},
				},
			},
			Location: ast_domain.Location{Line: 10, Column: 1},
		}

		finalised, err := visitor.processPartialLinking(context.Background(), pInfo, visitor.ctx)

		require.NoError(t, err)
		require.NotNil(t, finalised)
		assert.NotEmpty(t, finalised.canonicalKey)
	})

	t.Run("returns error when invoker context is nil", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitorForPartialLinking()

		pInfo := &ast_domain.PartialInvocationInfo{
			PartialPackageName: "partial_abc",
			PartialAlias:       "Card",
			PassedProps:        make(map[string]ast_domain.PropValue),
			Location:           ast_domain.Location{Line: 10, Column: 1},
		}

		_, err := visitor.processPartialLinking(context.Background(), pInfo, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil invoker context")
	})
}

func TestLinkingVisitor_HandlePartialInvocation(t *testing.T) {
	t.Parallel()

	t.Run("returns error when partial component not in virtual module", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitor()

		node := &ast_domain.TemplateNode{
			TagName: "Card",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				PartialInfo: &ast_domain.PartialInvocationInfo{
					PartialPackageName: "nonexistent_hash",
					PartialAlias:       "Card",
					PassedProps:        make(map[string]ast_domain.PropValue),
				},
			},
			Location: ast_domain.Location{Line: 5, Column: 1},
		}

		_, _, err := visitor.handlePartialInvocation(context.Background(), node, visitor.ctx, "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "nonexistent_hash")
	})

	t.Run("succeeds for valid partial invocation and creates context for children", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitorForHandlePartial()

		node := &ast_domain.TemplateNode{
			TagName: "Card",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				PartialInfo: &ast_domain.PartialInvocationInfo{
					PartialPackageName:  "partial_full",
					PartialAlias:        "Card",
					InvokerPackageAlias: "page_hash",
					PassedProps:         make(map[string]ast_domain.PropValue),
				},
				OriginalPackageAlias: new("partial_full"),
				OriginalSourcePath:   new("/test/partial.piko"),
			},
			Location: ast_domain.Location{Line: 5, Column: 1},
		}

		ctxForChildren, invocationKey, err := visitor.handlePartialInvocation(context.Background(), node, visitor.ctx, "  ")

		require.NoError(t, err)
		assert.NotNil(t, ctxForChildren)
		assert.NotEmpty(t, invocationKey)
	})

	t.Run("falls back to parent context when node lacks origin annotations", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitorForPartialLinking()

		node := &ast_domain.TemplateNode{
			TagName: "Card",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				PartialInfo: &ast_domain.PartialInvocationInfo{
					PartialPackageName:  "partial_abc",
					PartialAlias:        "Card",
					InvokerPackageAlias: "page_hash",
					PassedProps:         make(map[string]ast_domain.PropValue),
				},
			},
			Location: ast_domain.Location{Line: 5, Column: 1},
		}

		ctxForChildren, invocationKey, err := visitor.handlePartialInvocation(context.Background(), node, visitor.ctx, "  ")

		require.NoError(t, err)
		assert.NotNil(t, ctxForChildren)
		assert.NotEmpty(t, invocationKey)
	})

	t.Run("sets InvokerInvocationKey on pInfo from current invocation key", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitorForPartialLinking()
		visitor.currentInvocationKey = "parent_inv_key_123"

		pInfo := &ast_domain.PartialInvocationInfo{
			PartialPackageName:  "partial_abc",
			PartialAlias:        "Card",
			InvokerPackageAlias: "page_hash",
			PassedProps:         make(map[string]ast_domain.PropValue),
		}
		node := &ast_domain.TemplateNode{
			TagName: "Card",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				PartialInfo:          pInfo,
				OriginalPackageAlias: new("partial_abc"),
				OriginalSourcePath:   new("/test/card.piko"),
			},
			Location: ast_domain.Location{Line: 5, Column: 1},
		}

		_, _, err := visitor.handlePartialInvocation(context.Background(), node, visitor.ctx, "  ")

		require.NoError(t, err)
		assert.Equal(t, "parent_inv_key_123", pInfo.InvokerInvocationKey)
	})

	t.Run("stores unique invocation in shared state", func(t *testing.T) {
		t.Parallel()

		visitor := createTestLinkingVisitorForHandlePartial()

		node := &ast_domain.TemplateNode{
			TagName: "Card",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				PartialInfo: &ast_domain.PartialInvocationInfo{
					PartialPackageName:  "partial_full",
					PartialAlias:        "Card",
					InvokerPackageAlias: "page_hash",
					PassedProps:         make(map[string]ast_domain.PropValue),
				},
				OriginalPackageAlias: new("partial_full"),
				OriginalSourcePath:   new("/test/partial.piko"),
			},
			Location: ast_domain.Location{Line: 5, Column: 1},
		}

		_, _, err := visitor.handlePartialInvocation(context.Background(), node, visitor.ctx, "  ")

		require.NoError(t, err)
		assert.Len(t, visitor.state.uniqueInvocations, 1)
		assert.Len(t, visitor.state.invocationOrder, 1)
	})
}

func createTestLinkingVisitorForPartialLinking() *linkingVisitor {
	diagnostics := make([]*ast_domain.Diagnostic, 0)
	ctx := &AnalysisContext{
		Symbols:                  NewSymbolTable(nil),
		Diagnostics:              &diagnostics,
		CurrentGoFullPackagePath: "test/page",
		CurrentGoPackageName:     "page",
		CurrentGoSourcePath:      "/test/page.go",
		SFCSourcePath:            "/test/page.piko",
		Logger:                   logger_domain.GetLogger("test"),
	}

	partialComp := &annotator_dto.VirtualComponent{
		HashedName:             "partial_abc",
		CanonicalGoPackagePath: "test/partial",
		VirtualGoFilePath:      "/virtual/partial.go",
		RewrittenScriptAST: &goast.File{
			Name: goast.NewIdent("partial_pkg"),
		},
		Source: &annotator_dto.ParsedComponent{
			SourcePath: "/test/card.piko",
		},
	}

	vm := &annotator_dto.VirtualModule{
		ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
			"partial_abc": partialComp,
		},
		ComponentsByGoPath: map[string]*annotator_dto.VirtualComponent{
			"test/page": {
				HashedName:             "page_hash",
				CanonicalGoPackagePath: "test/page",
				VirtualGoFilePath:      "/test/page.go",
				RewrittenScriptAST: &goast.File{
					Name: goast.NewIdent("page"),
				},
				Source: &annotator_dto.ParsedComponent{
					SourcePath: "/test/page.piko",
				},
			},
		},
	}

	state := &linkingSharedState{
		uniqueInvocations: make(map[string]*annotator_dto.PartialInvocation),
		invocationOrder:   make([]string, 0),
	}

	mockInspector := &inspector_domain.MockTypeQuerier{
		GetImportsForFileFunc: func(_, _ string) map[string]string {
			return map[string]string{}
		},
		GetAllPackagesFunc: func() map[string]*inspector_dto.Package {
			return map[string]*inspector_dto.Package{}
		},
	}

	return &linkingVisitor{

		typeResolver:  &TypeResolver{inspector: mockInspector},
		virtualModule: vm,
		diagnostics:   &diagnostics,
		ctx:           ctx,
		depth:         0,
		state:         state,
	}
}

func createTestLinkingVisitorForHandlePartial() *linkingVisitor {
	diagnostics := make([]*ast_domain.Diagnostic, 0)
	ctx := &AnalysisContext{
		Symbols:                  NewSymbolTable(nil),
		Diagnostics:              &diagnostics,
		CurrentGoFullPackagePath: "test/page",
		CurrentGoPackageName:     "page",
		CurrentGoSourcePath:      "/test/page.go",
		SFCSourcePath:            "/test/page.piko",
		Logger:                   logger_domain.GetLogger("test"),
	}

	partialComp := &annotator_dto.VirtualComponent{
		HashedName:             "partial_full",
		CanonicalGoPackagePath: "test/partial",
		VirtualGoFilePath:      "/virtual/partial.go",
		RewrittenScriptAST: &goast.File{
			Name: goast.NewIdent("partial_pkg"),
		},
		Source: &annotator_dto.ParsedComponent{
			SourcePath: "/test/partial.piko",
		},
	}

	pageComp := &annotator_dto.VirtualComponent{
		HashedName:             "page_hash",
		CanonicalGoPackagePath: "test/page",
		VirtualGoFilePath:      "/test/page.go",
		RewrittenScriptAST: &goast.File{
			Name: goast.NewIdent("page"),
		},
		Source: &annotator_dto.ParsedComponent{
			SourcePath: "/test/page.piko",
		},
	}

	vm := &annotator_dto.VirtualModule{
		ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
			"partial_full": partialComp,
			"page_hash":    pageComp,
		},
		ComponentsByGoPath: map[string]*annotator_dto.VirtualComponent{
			"test/page":    pageComp,
			"test/partial": partialComp,
		},
	}

	state := &linkingSharedState{
		uniqueInvocations: make(map[string]*annotator_dto.PartialInvocation),
		invocationOrder:   make([]string, 0),
	}

	mockInspector := &inspector_domain.MockTypeQuerier{
		GetImportsForFileFunc: func(_, _ string) map[string]string {
			return map[string]string{}
		},
		GetAllPackagesFunc: func() map[string]*inspector_dto.Package {
			return map[string]*inspector_dto.Package{}
		},
	}

	return &linkingVisitor{

		typeResolver:  &TypeResolver{inspector: mockInspector},
		virtualModule: vm,
		diagnostics:   &diagnostics,
		ctx:           ctx,
		depth:         0,
		state:         state,
	}
}
