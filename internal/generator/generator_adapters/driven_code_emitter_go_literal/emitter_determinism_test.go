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

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/generator/generator_dto"
)

const (
	emitterDeterminismRuns = 16
	emitterDeterminismHash = "determinism_hash"
)

func newEmitterDeterminismResult() *annotator_dto.AnnotationResult {
	result := createMinimalAnnotationResult(emitterDeterminismHash)
	sourcePath := "/test/component.pk"

	element := createMockTemplateNode(ast_domain.NodeElement, "a", "")
	element.Location = ast_domain.Location{Line: 1, Column: 1}
	element.Attributes = []ast_domain.HTMLAttribute{
		{Name: "class", Value: "link"},
		{Name: "data-static", Value: "fixed"},
	}
	element.Binds = make(map[string]*ast_domain.Directive)
	for _, name := range []string{"title", "href", "id", "rel", "data-alpha", "data-beta", "data-gamma", "aria-label"} {
		identifier := createTypedIdentifier(name, "string")
		element.Binds[name] = &ast_domain.Directive{
			Type:          ast_domain.DirectiveBind,
			Arg:           name,
			RawExpression: name,
			Expression:    identifier,
			GoAnnotations: identifier.GoAnnotations,
		}
	}
	for _, name := range []string{"data-first", "data-second"} {
		identifier := createTypedIdentifier(name, "string")
		element.DynamicAttributes = append(element.DynamicAttributes, ast_domain.DynamicAttribute{
			Name:          name,
			RawExpression: name,
			Expression:    identifier,
			GoAnnotations: identifier.GoAnnotations,
		})
	}

	result.AnnotatedAST = &ast_domain.TemplateAST{
		SourcePath: &sourcePath,
		RootNodes:  []*ast_domain.TemplateNode{element},
	}
	return result
}

func TestEmitCode_IsDeterministic(t *testing.T) {
	t.Parallel()

	request := generator_dto.GenerateRequest{
		SourcePath:             "/test/component.pk",
		PackageName:            "testpkg",
		HashedName:             emitterDeterminismHash,
		CanonicalGoPackagePath: "test.com/pkg",
		BaseDir:                "/test",
		IsPage:                 true,
	}

	var first []byte
	for run := range emitterDeterminismRuns {
		code, _, err := NewEmitter().EmitCode(context.Background(), newEmitterDeterminismResult(), request)
		require.NoError(t, err)
		require.NotEmpty(t, code)

		if run == 0 {
			first = code
			continue
		}
		require.Equal(t, string(first), string(code), "EmitCode output differed on run %d", run+1)
	}
}

func TestEmitMiscDirectives_ModelBeforeScaffold(t *testing.T) {
	t.Parallel()

	for range emitterDeterminismRuns {
		var visited []string
		ae := newRecordingAttributeEmitter(&visited)
		ne := &nodeEmitter{emitter: ae.emitter, expressionEmitter: ae.expressionEmitter, attributeEmitter: ae}
		node := createMockTemplateNode(ast_domain.NodeElement, "input", "")
		node.DirModel = &ast_domain.Directive{Type: ast_domain.DirectiveModel, Expression: createTypedIdentifier("model", "string")}
		node.DirScaffold = &ast_domain.Directive{Type: ast_domain.DirectiveScaffold, Expression: createTypedIdentifier("scaffold", "string")}

		statements, diagnostics := ne.emitMiscDirectives(cachedIdent("node"), node)

		require.Empty(t, diagnostics)
		require.Len(t, statements, 2)
		require.Equal(t, []string{"model", "scaffold"}, visited, "p-model must be emitted before p-scaffold on every run")
	}
}

func TestEmitDynamicCollectionFetcher_ImportAliasesAssignedInPathOrder(t *testing.T) {
	t.Parallel()

	for range emitterDeterminismRuns {
		em := requireEmitter(t)
		em.resetState()
		em.ctx = NewEmitterContext()
		info := &collection_dto.DynamicCollectionInfo{
			FetcherCode: &collection_dto.RuntimeFetcherCode{
				FetcherFunc: &goast.FuncDecl{Name: cachedIdent("Fetch"), Type: &goast.FuncType{}, Body: &goast.BlockStmt{}},
				RequiredImports: map[string]string{
					"example.com/d/dto": "dto",
					"example.com/b/dto": "dto",
					"example.com/a/dto": "dto",
					"example.com/c/dto": "dto",
				},
			},
		}

		_, _, diagnostics := requireExpressionEmitter(t, em).emitDynamicCollectionFetcher(info)

		require.Empty(t, diagnostics)
		require.Equal(t, "dto", em.ctx.requiredImports["example.com/a/dto"], "the first import path in sorted order must keep its alias on every run")
		require.Equal(t, "example.com/a/dto", em.ctx.usedAliases["dto"])
	}
}
