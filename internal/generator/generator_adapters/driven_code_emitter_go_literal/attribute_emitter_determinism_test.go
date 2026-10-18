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
	"slices"
	"testing"

	goast "go/ast"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

const (
	bindDeterminismRuns = 32
)

var (
	bindDeterminismNames = []string{
		"aria-label",
		"data-alpha",
		"data-beta",
		"data-gamma",
		"href",
		"id",
		"rel",
		"title",
	}
)

func newBindDeterminismNode() *ast_domain.TemplateNode {
	node := createMockTemplateNode(ast_domain.NodeElement, "a", "")
	node.Binds = make(map[string]*ast_domain.Directive, len(bindDeterminismNames))
	for _, name := range slices.Backward(bindDeterminismNames) {
		node.Binds[name] = &ast_domain.Directive{
			Type: ast_domain.DirectiveBind,
			Arg:  name,
			Expression: &ast_domain.Identifier{
				Name:          name,
				GoAnnotations: createMockAnnotation("string", inspector_dto.StringablePrimitive),
			},
			GoAnnotations: createMockAnnotation("string", inspector_dto.StringablePrimitive),
		}
	}
	return node
}

func newRecordingAttributeEmitter(visited *[]string) *attributeEmitter {
	mockExpr := &mockExpressionEmitter{
		emitFunc: func(expression ast_domain.Expression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
			if identifier, ok := expression.(*ast_domain.Identifier); ok {
				*visited = append(*visited, identifier.Name)
			}
			return cachedIdent("emittedValue"), nil, nil
		},
	}
	return newAttributeEmitter(&emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}, mockExpr)
}

func TestEmitDynamicAttributes_BindsEmittedInNameOrder(t *testing.T) {
	t.Parallel()

	for range bindDeterminismRuns {
		var visited []string
		ae := newRecordingAttributeEmitter(&visited)
		attributeSliceExpression := &goast.SelectorExpr{X: cachedIdent("node"), Sel: cachedIdent("Attributes")}

		_, diagnostics := ae.emitDynamicAttributes(cachedIdent("node"), attributeSliceExpression, newBindDeterminismNode())

		require.Empty(t, diagnostics)
		require.Equal(t, bindDeterminismNames, visited, "p-bind attributes must be emitted in name order on every run")
	}
}

func TestEmitBindAttributes_BindsEmittedInNameOrder(t *testing.T) {
	t.Parallel()

	for range bindDeterminismRuns {
		var visited []string
		ae := newRecordingAttributeEmitter(&visited)

		_, diagnostics := ae.emitBindAttributes(cachedIdent("node"), newBindDeterminismNode())

		require.Empty(t, diagnostics)
		require.Equal(t, bindDeterminismNames, visited, "p-bind attribute writers must be emitted in name order on every run")
	}
}
