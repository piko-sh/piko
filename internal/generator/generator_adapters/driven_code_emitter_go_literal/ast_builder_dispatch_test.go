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

package driven_code_emitter_go_literal

import (
	"context"
	"testing"

	goast "go/ast"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/internal/ast/ast_domain"
)

func TestAstBuilder_EmitNode_Dispatch(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		node             *ast_domain.TemplateNode
		validateFunc     func(t *testing.T, statements []goast.Stmt, consumed int, diagnostics []*ast_domain.Diagnostic)
		name             string
		expectedConsumed int
	}{
		{
			name: "p-if directive dispatches to IfEmitter",
			node: &ast_domain.TemplateNode{
				TagName: "div",
				DirIf: &ast_domain.Directive{
					Expression: &ast_domain.BooleanLiteral{
						Value: true,
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							ResolvedType: &ast_domain.ResolvedTypeInfo{
								TypeExpression: cachedIdent("bool"),
							},
						},
					},
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{},
				},
			},
			expectedConsumed: 1,
			validateFunc: func(t *testing.T, statements []goast.Stmt, consumed int, diagnostics []*ast_domain.Diagnostic) {
				assert.Empty(t, diagnostics)
				assert.NotEmpty(t, statements, "IfEmitter should generate statements")

				assert.GreaterOrEqual(t, consumed, 1)
			},
		},
		{
			name: "p-else node is skipped",
			node: &ast_domain.TemplateNode{
				TagName: "div",
				DirElse: &ast_domain.Directive{
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{},
				},
			},
			expectedConsumed: 1,
			validateFunc: func(t *testing.T, statements []goast.Stmt, consumed int, diagnostics []*ast_domain.Diagnostic) {
				assert.Empty(t, diagnostics)
				assert.Empty(t, statements, "Else nodes should be skipped (handled by IfEmitter)")
				assert.Equal(t, 1, consumed, "Should still consume the node")
			},
		},
		{
			name: "static node dispatches to staticEmitter",
			node: &ast_domain.TemplateNode{
				TagName:  "div",
				NodeType: ast_domain.NodeElement,
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					IsStructurallyStatic: true,
				},
			},
			expectedConsumed: 1,
			validateFunc: func(t *testing.T, statements []goast.Stmt, consumed int, diagnostics []*ast_domain.Diagnostic) {

				assert.Equal(t, 1, consumed)
			},
		},
		{
			name: "fragment node with no dynamic features emits children",
			node: &ast_domain.TemplateNode{
				TagName:  "",
				NodeType: ast_domain.NodeFragment,
				Children: []*ast_domain.TemplateNode{
					{
						TagName:  "span",
						NodeType: ast_domain.NodeElement,
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							IsStructurallyStatic: true,
						},
					},
				},
			},
			expectedConsumed: 1,
			validateFunc: func(t *testing.T, statements []goast.Stmt, consumed int, diagnostics []*ast_domain.Diagnostic) {
				assert.Equal(t, 1, consumed)

			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			em := requireEmitter(t)
			em.resetState(context.Background())
			em.ctx = NewEmitterContext()

			ctx := context.Background()
			parentSliceExpr := &goast.SelectorExpr{
				X:   cachedIdent("parent"),
				Sel: cachedIdent("Children"),
			}

			emitCtx := newNodeEmissionContext(ctx, nodeEmissionParams{
				Node:                  tc.node,
				ParentSliceExpression: parentSliceExpr,
				Index:                 0,
				Siblings:              []*ast_domain.TemplateNode{tc.node},
				IsRootNode:            false,
				PartialScopeID:        "",
				MainComponentScope:    "",
			})

			statements, consumed, diagnostics := em.astBuilder.emitNode(emitCtx)

			tc.validateFunc(t, statements, consumed, diagnostics)
			assert.Equal(t, tc.expectedConsumed, consumed, "Should consume expected number of nodes")
		})
	}
}

func TestAstBuilder_PrepareNodeForEmission(t *testing.T) {
	t.Parallel()

	t.Run("returns original node when not a partial", func(t *testing.T) {
		t.Parallel()

		em := requireEmitter(t)
		em.resetState(context.Background())
		em.config = EmitterConfig{IsPage: true}

		node := &ast_domain.TemplateNode{
			TagName: "div",
		}

		ctx := context.Background()
		emitCtx := newNodeEmissionContext(ctx, nodeEmissionParams{
			Node:                  node,
			ParentSliceExpression: nil,
			Index:                 0,
			Siblings:              []*ast_domain.TemplateNode{node},
			IsRootNode:            false,
			PartialScopeID:        "",
			MainComponentScope:    "",
		})

		preparedNode := em.astBuilder.prepareNodeForEmission(emitCtx)

		assert.Same(t, node, preparedNode, "Should return original node for pages")
	})

}

func TestAstBuilder_EmitContentTag(t *testing.T) {
	t.Parallel()

	em := requireEmitter(t)
	em.resetState(context.Background())
	em.ctx = NewEmitterContext()
	em.config = EmitterConfig{}

	contentNode := &ast_domain.TemplateNode{
		TagName:  "piko:content",
		NodeType: ast_domain.NodeElement,
	}

	ctx := context.Background()
	parentSliceExpr := &goast.SelectorExpr{
		X:   cachedIdent("parent"),
		Sel: cachedIdent("Children"),
	}

	emitCtx := newNodeEmissionContext(ctx, nodeEmissionParams{
		Node:                  contentNode,
		ParentSliceExpression: parentSliceExpr,
		Index:                 0,
		Siblings:              []*ast_domain.TemplateNode{contentNode},
		IsRootNode:            false,
		PartialScopeID:        "",
		MainComponentScope:    "",
	})

	statements, consumed, diagnostics := em.astBuilder.emitNode(emitCtx)

	assert.Empty(t, diagnostics)
	assert.NotEmpty(t, statements, "Should emit content nodes")
	assert.Equal(t, 1, consumed, "Should consume the content tag")
}
