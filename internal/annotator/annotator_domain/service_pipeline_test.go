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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestHandleStageError(t *testing.T) {
	t.Parallel()

	t.Run("returns error when not fault tolerant", func(t *testing.T) {
		t.Parallel()

		pipeline := &componentAnnotationPipeline{
			vc: &annotator_dto.VirtualComponent{
				Source: &annotator_dto.ParsedComponent{
					SourcePath: "/test/component.piko",
				},
			},
			options:     &annotationOptions{faultTolerant: false},
			diagnostics: make([]*ast_domain.Diagnostic, 0),
		}

		err := pipeline.handleStageError(context.Background(), errors.New("test error"), "expansion", nil)

		assert.Error(t, err)
		assert.Equal(t, `annotator stage "expansion" failed: test error`, err.Error())
		assert.Empty(t, pipeline.diagnostics)
	})

	t.Run("returns nil and adds diagnostic when fault tolerant", func(t *testing.T) {
		t.Parallel()

		pipeline := &componentAnnotationPipeline{
			vc: &annotator_dto.VirtualComponent{
				Source: &annotator_dto.ParsedComponent{
					SourcePath: "/test/component.piko",
				},
			},
			options:     &annotationOptions{faultTolerant: true},
			diagnostics: make([]*ast_domain.Diagnostic, 0),
		}

		err := pipeline.handleStageError(context.Background(), errors.New("stage failed"), "linking", nil)

		assert.NoError(t, err)
		require.Len(t, pipeline.diagnostics, 1)
		assert.Contains(t, pipeline.diagnostics[0].Message, "Fatal linking error")
		assert.Contains(t, pipeline.diagnostics[0].Message, "stage failed")
		assert.Equal(t, ast_domain.Error, pipeline.diagnostics[0].Severity)
		assert.Equal(t, "/test/component.piko", pipeline.diagnostics[0].SourcePath)
		assert.Equal(t, 1, pipeline.diagnostics[0].Location.Line)
	})

	t.Run("includes stage name in diagnostic message", func(t *testing.T) {
		t.Parallel()

		pipeline := &componentAnnotationPipeline{
			vc: &annotator_dto.VirtualComponent{
				Source: &annotator_dto.ParsedComponent{
					SourcePath: "/test/file.piko",
				},
			},
			options:     &annotationOptions{faultTolerant: true},
			diagnostics: make([]*ast_domain.Diagnostic, 0),
		}

		_ = pipeline.handleStageError(context.Background(), errors.New("boom"), "annotation", nil)

		require.Len(t, pipeline.diagnostics, 1)
		assert.Contains(t, pipeline.diagnostics[0].Message, "annotation")
	})
}

func TestEarlyResult(t *testing.T) {
	t.Parallel()

	t.Run("returns result with AST from expansion result", func(t *testing.T) {
		t.Parallel()

		vm := &annotator_dto.VirtualModule{}
		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{TagName: "div", NodeType: ast_domain.NodeElement},
			},
		}

		pipeline := &componentAnnotationPipeline{
			virtualModule: vm,
		}

		result := pipeline.earlyResult(&annotator_dto.ExpansionResult{
			FlattenedAST: ast,
		})

		require.NotNil(t, result)
		assert.Same(t, ast, result.AnnotatedAST)
		assert.Same(t, vm, result.VirtualModule)
		assert.Nil(t, result.AnalysisMap)
		assert.Empty(t, result.StyleBlock)
		assert.Empty(t, result.ClientScript)
		assert.Nil(t, result.UniqueInvocations)
		assert.Nil(t, result.AssetDependencies)
	})

	t.Run("returns result with nil AST when expansion result is nil", func(t *testing.T) {
		t.Parallel()

		pipeline := &componentAnnotationPipeline{
			virtualModule: &annotator_dto.VirtualModule{},
		}

		result := pipeline.earlyResult(nil)

		require.NotNil(t, result)
		assert.Nil(t, result.AnnotatedAST)
	})
}

func TestEarlyResultFromLink(t *testing.T) {
	t.Parallel()

	t.Run("uses linked AST when available", func(t *testing.T) {
		t.Parallel()

		expandedAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{{TagName: "expanded"}},
		}
		linkedAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{{TagName: "linked"}},
		}

		pipeline := &componentAnnotationPipeline{
			virtualModule: &annotator_dto.VirtualModule{},
		}

		result := pipeline.earlyResultFromLink(
			&annotator_dto.ExpansionResult{FlattenedAST: expandedAST},
			&annotator_dto.LinkingResult{LinkedAST: linkedAST},
		)

		require.NotNil(t, result)
		assert.Same(t, linkedAST, result.AnnotatedAST)
	})

	t.Run("falls back to expanded AST when linking result is nil", func(t *testing.T) {
		t.Parallel()

		expandedAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{{TagName: "expanded"}},
		}

		pipeline := &componentAnnotationPipeline{
			virtualModule: &annotator_dto.VirtualModule{},
		}

		result := pipeline.earlyResultFromLink(
			&annotator_dto.ExpansionResult{FlattenedAST: expandedAST},
			nil,
		)

		require.NotNil(t, result)
		assert.Same(t, expandedAST, result.AnnotatedAST)
	})

	t.Run("falls back to expanded AST when linked AST is nil", func(t *testing.T) {
		t.Parallel()

		expandedAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{{TagName: "expanded"}},
		}

		pipeline := &componentAnnotationPipeline{
			virtualModule: &annotator_dto.VirtualModule{},
		}

		result := pipeline.earlyResultFromLink(
			&annotator_dto.ExpansionResult{FlattenedAST: expandedAST},
			&annotator_dto.LinkingResult{LinkedAST: nil},
		)

		require.NotNil(t, result)
		assert.Same(t, expandedAST, result.AnnotatedAST)
	})
}

func TestEarlyResultFromAnnotation(t *testing.T) {
	t.Parallel()

	t.Run("returns analysis result when not nil", func(t *testing.T) {
		t.Parallel()

		analysisResult := &annotator_dto.AnnotationResult{
			StyleBlock: "test-css",
		}
		linkedAST := &ast_domain.TemplateAST{}

		pipeline := &componentAnnotationPipeline{
			virtualModule: &annotator_dto.VirtualModule{},
		}

		result := pipeline.earlyResultFromAnnotation(
			&annotator_dto.LinkingResult{LinkedAST: linkedAST},
			analysisResult,
		)

		assert.Same(t, analysisResult, result)
		assert.Equal(t, "test-css", result.StyleBlock)
	})

	t.Run("builds default result from linking result when analysis is nil", func(t *testing.T) {
		t.Parallel()

		linkedAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{{TagName: "linked"}},
		}
		vm := &annotator_dto.VirtualModule{}

		pipeline := &componentAnnotationPipeline{
			virtualModule: vm,
		}

		result := pipeline.earlyResultFromAnnotation(
			&annotator_dto.LinkingResult{LinkedAST: linkedAST},
			nil,
		)

		require.NotNil(t, result)
		assert.Same(t, linkedAST, result.AnnotatedAST)
		assert.Same(t, vm, result.VirtualModule)
		assert.Empty(t, result.StyleBlock)
		assert.Empty(t, result.ClientScript)
	})
}

func TestHandleStageError_FaultTolerantAccumulatesDiagnostics(t *testing.T) {
	t.Parallel()

	pipeline := &componentAnnotationPipeline{
		vc: &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/test/comp.piko",
			},
		},
		options:     &annotationOptions{faultTolerant: true},
		diagnostics: make([]*ast_domain.Diagnostic, 0),
	}

	err1 := pipeline.handleStageError(context.Background(), errors.New("first error"), "expansion", nil)
	err2 := pipeline.handleStageError(context.Background(), errors.New("second error"), "linking", nil)

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	require.Len(t, pipeline.diagnostics, 2)
	assert.Contains(t, pipeline.diagnostics[0].Message, "expansion")
	assert.Contains(t, pipeline.diagnostics[1].Message, "linking")
}

func TestHandleStageError_StrictMode_ReturnsOriginalError(t *testing.T) {
	t.Parallel()

	originalErr := errors.New("original error with details")
	pipeline := &componentAnnotationPipeline{
		vc: &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/test/comp.piko",
			},
		},
		options:     &annotationOptions{faultTolerant: false},
		diagnostics: make([]*ast_domain.Diagnostic, 0),
	}

	err := pipeline.handleStageError(context.Background(), originalErr, "annotation", nil)

	require.ErrorIs(t, err, originalErr)
	assert.Equal(t, `annotator stage "annotation" failed: original error with details`, err.Error())
	assert.True(t, errors.Is(err, originalErr), "wrapped error should unwrap to the original")
	assert.Empty(t, pipeline.diagnostics, "strict mode should not add diagnostics")
}

func TestHandleStageError_FaultTolerant_DiagnosticLocation(t *testing.T) {
	t.Parallel()

	pipeline := &componentAnnotationPipeline{
		vc: &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/my/component.piko",
			},
		},
		options:     &annotationOptions{faultTolerant: true},
		diagnostics: make([]*ast_domain.Diagnostic, 0),
	}

	_ = pipeline.handleStageError(context.Background(), errors.New("something broke"), "expansion", nil)

	require.Len(t, pipeline.diagnostics, 1)
	diagnostic := pipeline.diagnostics[0]
	assert.Equal(t, 1, diagnostic.Location.Line)
	assert.Equal(t, 1, diagnostic.Location.Column)
	assert.Equal(t, 0, diagnostic.Location.Offset)
	assert.Equal(t, "/my/component.piko", diagnostic.SourcePath)
	assert.Equal(t, ast_domain.Error, diagnostic.Severity)
}

func TestEarlyResult_PreservesVirtualModule(t *testing.T) {
	t.Parallel()

	vm := &annotator_dto.VirtualModule{
		ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
			"hash1": {},
		},
	}

	pipeline := &componentAnnotationPipeline{
		virtualModule: vm,
	}

	result := pipeline.earlyResult(&annotator_dto.ExpansionResult{
		FlattenedAST: &ast_domain.TemplateAST{},
	})

	require.NotNil(t, result)
	assert.Same(t, vm, result.VirtualModule)
	assert.Nil(t, result.AnalysisMap)
	assert.Nil(t, result.EntryPointStyleBlocks)
	assert.Nil(t, result.AssetRefs)
	assert.Nil(t, result.CustomTags)
	assert.Nil(t, result.UniqueInvocations)
	assert.Nil(t, result.AssetDependencies)
}

func TestEarlyResult_WithNilExpansionAndFlattenedAST(t *testing.T) {
	t.Parallel()

	pipeline := &componentAnnotationPipeline{
		virtualModule: &annotator_dto.VirtualModule{},
	}

	result := pipeline.earlyResult(&annotator_dto.ExpansionResult{
		FlattenedAST: nil,
	})

	require.NotNil(t, result)
	assert.Nil(t, result.AnnotatedAST)
}

func TestEarlyResultFromLink_AllFieldsAreNil(t *testing.T) {
	t.Parallel()

	vm := &annotator_dto.VirtualModule{}
	pipeline := &componentAnnotationPipeline{
		virtualModule: vm,
	}

	expandedAST := &ast_domain.TemplateAST{}
	result := pipeline.earlyResultFromLink(
		&annotator_dto.ExpansionResult{FlattenedAST: expandedAST},
		nil,
	)

	require.NotNil(t, result)
	assert.Same(t, expandedAST, result.AnnotatedAST)
	assert.Same(t, vm, result.VirtualModule)
	assert.Empty(t, result.StyleBlock)
	assert.Empty(t, result.ClientScript)
	assert.Nil(t, result.AnalysisMap)
	assert.Nil(t, result.EntryPointStyleBlocks)
	assert.Nil(t, result.AssetRefs)
	assert.Nil(t, result.CustomTags)
	assert.Nil(t, result.UniqueInvocations)
	assert.Nil(t, result.AssetDependencies)
}

func TestEarlyResultFromAnnotation_ReturnsAnalysisResultWithAllFields(t *testing.T) {
	t.Parallel()

	analysisResult := &annotator_dto.AnnotationResult{
		StyleBlock:   "body { color: blue; }",
		ClientScript: "export function init() {}",
		AnnotatedAST: &ast_domain.TemplateAST{},
		AnalysisMap:  make(map[*ast_domain.TemplateNode]*AnalysisContext),
	}
	linkedAST := &ast_domain.TemplateAST{}

	pipeline := &componentAnnotationPipeline{
		virtualModule: &annotator_dto.VirtualModule{},
	}

	result := pipeline.earlyResultFromAnnotation(
		&annotator_dto.LinkingResult{LinkedAST: linkedAST},
		analysisResult,
	)

	assert.Same(t, analysisResult, result)
	assert.Equal(t, "body { color: blue; }", result.StyleBlock)
	assert.Equal(t, "export function init() {}", result.ClientScript)
	assert.NotNil(t, result.AnalysisMap)
}

func TestEarlyResultFromAnnotation_NilAnalysisResult_AllFieldsPopulated(t *testing.T) {
	t.Parallel()

	linkedAST := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{{TagName: "p"}},
	}
	vm := &annotator_dto.VirtualModule{}

	pipeline := &componentAnnotationPipeline{
		virtualModule: vm,
	}

	result := pipeline.earlyResultFromAnnotation(
		&annotator_dto.LinkingResult{LinkedAST: linkedAST},
		nil,
	)

	require.NotNil(t, result)
	assert.Same(t, linkedAST, result.AnnotatedAST)
	assert.Same(t, vm, result.VirtualModule)
	assert.Empty(t, result.StyleBlock)
	assert.Empty(t, result.ClientScript)
	assert.Nil(t, result.AnalysisMap)
	assert.Nil(t, result.EntryPointStyleBlocks)
	assert.Nil(t, result.AssetRefs)
	assert.Nil(t, result.CustomTags)
	assert.Nil(t, result.UniqueInvocations)
	assert.Nil(t, result.AssetDependencies)
}

func TestComponentAnnotationPipeline_RunPropDataSourceLinking(t *testing.T) {
	t.Parallel()

	pipeline := &componentAnnotationPipeline{
		diagnostics: make([]*ast_domain.Diagnostic, 0),
		virtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: make(map[string]*annotator_dto.VirtualComponent),
		},
	}

	analysisResult := &annotator_dto.AnnotationResult{
		AnnotatedAST: &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{TagName: "div", NodeType: ast_domain.NodeElement},
			},
		},
	}

	assert.NotPanics(t, func() {
		pipeline.runPropDataSourceLinking(context.Background(), analysisResult)
	})
	assert.Nil(t, analysisResult.AnnotatedAST.Diagnostics)
}

func TestComponentAnnotationPipeline_RunPropDataSourceLinking_NilAST(t *testing.T) {
	t.Parallel()

	pipeline := &componentAnnotationPipeline{
		diagnostics: make([]*ast_domain.Diagnostic, 0),
		virtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: make(map[string]*annotator_dto.VirtualComponent),
		},
	}

	analysisResult := &annotator_dto.AnnotationResult{
		AnnotatedAST: nil,
	}

	assert.NotPanics(t, func() {
		pipeline.runPropDataSourceLinking(context.Background(), analysisResult)
	})
}
