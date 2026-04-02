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
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestBuildActionsFromManifest(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		manifest      *annotator_dto.ActionManifest
		name          string
		expectedCount int
		expectedNil   bool
	}{
		{
			name:        "nil manifest returns nil",
			manifest:    nil,
			expectedNil: true,
		},
		{
			name: "manifest with no actions returns nil",
			manifest: &annotator_dto.ActionManifest{
				Actions: []annotator_dto.ActionDefinition{},
				ByName:  map[string]*annotator_dto.ActionDefinition{},
			},
			expectedNil: true,
		},
		{
			name: "manifest with one action returns single entry",
			manifest: &annotator_dto.ActionManifest{
				Actions: []annotator_dto.ActionDefinition{
					{
						Name:       "email.contact",
						HTTPMethod: "POST",
					},
				},
				ByName: map[string]*annotator_dto.ActionDefinition{},
			},
			expectedNil:   false,
			expectedCount: 1,
		},
		{
			name: "manifest with multiple actions returns all entries",
			manifest: &annotator_dto.ActionManifest{
				Actions: []annotator_dto.ActionDefinition{
					{
						Name:       "email.contact",
						HTTPMethod: "POST",
					},
					{
						Name:       "user.create",
						HTTPMethod: "POST",
					},
					{
						Name:       "data.fetch",
						HTTPMethod: "GET",
					},
				},
				ByName: map[string]*annotator_dto.ActionDefinition{},
			},
			expectedNil:   false,
			expectedCount: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := buildActionsFromManifest(tc.manifest)

			if tc.expectedNil {
				assert.Nil(t, result)
				return
			}

			require.NotNil(t, result)
			assert.Len(t, result, tc.expectedCount)
		})
	}
}

func TestBuildActionsFromManifest_ActionInfoProvider(t *testing.T) {
	t.Parallel()

	manifest := &annotator_dto.ActionManifest{
		Actions: []annotator_dto.ActionDefinition{
			{
				Name:       "email.contact",
				HTTPMethod: "POST",
			},
			{
				Name:       "data.fetch",
				HTTPMethod: "GET",
			},
			{
				Name:       "default.action",
				HTTPMethod: "",
			},
		},
		ByName: map[string]*annotator_dto.ActionDefinition{},
	}

	result := buildActionsFromManifest(manifest)
	require.NotNil(t, result)

	postAction, ok := result["email.contact"]
	require.True(t, ok, "expected email.contact to be present in the map")
	assert.Equal(t, "POST", postAction.Method())

	getAction, ok := result["data.fetch"]
	require.True(t, ok, "expected data.fetch to be present in the map")
	assert.Equal(t, "GET", getAction.Method())

	defaultAction, ok := result["default.action"]
	require.True(t, ok, "expected default.action to be present in the map")
	assert.Equal(t, "POST", defaultAction.Method())
}

func TestBuildActionsFromManifest_KeysMatchActionNames(t *testing.T) {
	t.Parallel()

	manifest := &annotator_dto.ActionManifest{
		Actions: []annotator_dto.ActionDefinition{
			{Name: "alpha"},
			{Name: "beta"},
			{Name: "gamma"},
		},
		ByName: map[string]*annotator_dto.ActionDefinition{},
	}

	result := buildActionsFromManifest(manifest)
	require.NotNil(t, result)

	for _, action := range manifest.Actions {
		_, ok := result[action.Name]
		assert.True(t, ok, "expected key %q to be present in the map", action.Name)
	}
}

func TestCalculateWorkerCount(t *testing.T) {
	t.Parallel()

	cpuCount := runtime.NumCPU()

	testCases := []struct {
		name     string
		jobCount int
		wantMin  int
		wantMax  int
	}{
		{
			name:     "zero jobs returns at least 1",
			jobCount: 0,
			wantMin:  1,
			wantMax:  1,
		},
		{
			name:     "one job returns 1",
			jobCount: 1,
			wantMin:  1,
			wantMax:  1,
		},
		{
			name:     "jobs equal to CPU count returns CPU count",
			jobCount: cpuCount,
			wantMin:  cpuCount,
			wantMax:  cpuCount,
		},
		{
			name:     "jobs exceeding CPU count caps at CPU count",
			jobCount: cpuCount + 100,
			wantMin:  cpuCount,
			wantMax:  cpuCount,
		},
		{
			name:     "two jobs returns min of 2 and CPU count",
			jobCount: 2,
			wantMin:  min(2, cpuCount),
			wantMax:  min(2, cpuCount),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := calculateWorkerCount(tc.jobCount)

			assert.GreaterOrEqual(t, result, tc.wantMin,
				"worker count should be at least %d", tc.wantMin)
			assert.LessOrEqual(t, result, tc.wantMax,
				"worker count should be at most %d", tc.wantMax)
		})
	}
}

func TestCalculateWorkerCount_AlwaysAtLeastOneForNonNegativeInputs(t *testing.T) {
	t.Parallel()

	for _, n := range []int{0, 1, 2, 100, 10000} {
		result := calculateWorkerCount(n)
		assert.GreaterOrEqual(t, result, 1,
			"calculateWorkerCount(%d) must be >= 1", n)
	}
}

func TestCreateErrorJobResult(t *testing.T) {
	t.Parallel()

	service := &AnnotatorService{}
	vc := &annotator_dto.VirtualComponent{
		Source: &annotator_dto.ParsedComponent{
			SourcePath: "/project/pages/index.pk",
		},
		HashedName: "index_abc123",
	}
	testErr := errors.New("something went wrong")

	result := service.createErrorJobResult(context.Background(), vc, testErr)

	require.NotNil(t, result)
	assert.Nil(t, result.result, "the annotation result should be nil for an error job")
	require.Len(t, result.diagnostics, 1)

	diagnostic := result.diagnostics[0]
	assert.Equal(t, ast_domain.Error, diagnostic.Severity)
	assert.Contains(t, diagnostic.Message, "Fatal error during annotation")
	assert.Contains(t, diagnostic.Message, "something went wrong")
	assert.Equal(t, "/project/pages/index.pk", diagnostic.SourcePath)
	assert.Equal(t, 1, diagnostic.Location.Line)
	assert.Equal(t, 1, diagnostic.Location.Column)
	assert.Equal(t, 0, diagnostic.Location.Offset)
}

func TestCreateErrorJobResult_PreservesSourcePath(t *testing.T) {
	t.Parallel()

	service := &AnnotatorService{}

	testCases := []struct {
		name       string
		sourcePath string
	}{
		{
			name:       "simple path",
			sourcePath: "/app/pages/home.pk",
		},
		{
			name:       "nested path",
			sourcePath: "/app/components/deeply/nested/card.pk",
		},
		{
			name:       "path with spaces",
			sourcePath: "/my project/pages/about page.pk",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			vc := &annotator_dto.VirtualComponent{
				Source: &annotator_dto.ParsedComponent{
					SourcePath: tc.sourcePath,
				},
			}

			result := service.createErrorJobResult(context.Background(), vc, errors.New("test error"))

			require.Len(t, result.diagnostics, 1)
			assert.Equal(t, tc.sourcePath, result.diagnostics[0].SourcePath)
		})
	}
}

func TestAggregateAnnotationResults_EmptyChannel(t *testing.T) {
	t.Parallel()

	resultsChan := make(chan *annotationJobResult)
	close(resultsChan)

	finalResult := &annotator_dto.ProjectAnnotationResult{
		ComponentResults: make(map[string]*annotator_dto.AnnotationResult),
		AllDiagnostics:   make([]*ast_domain.Diagnostic, 0),
	}

	severeErrors := aggregateAnnotationResults(resultsChan, finalResult)

	assert.Empty(t, severeErrors)
	assert.Empty(t, finalResult.ComponentResults)
	assert.Empty(t, finalResult.AllDiagnostics)
}

func TestAggregateAnnotationResults_NilResultSkipped(t *testing.T) {
	t.Parallel()

	resultsChan := make(chan *annotationJobResult, 2)

	diagnostic := ast_domain.NewDiagnostic(
		ast_domain.Error,
		"test error",
		"",
		ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		"/test.pk",
	)
	resultsChan <- &annotationJobResult{
		result:      nil,
		diagnostics: []*ast_domain.Diagnostic{diagnostic},
	}
	close(resultsChan)

	finalResult := &annotator_dto.ProjectAnnotationResult{
		ComponentResults: make(map[string]*annotator_dto.AnnotationResult),
		AllDiagnostics:   make([]*ast_domain.Diagnostic, 0),
	}

	severeErrors := aggregateAnnotationResults(resultsChan, finalResult)

	assert.Empty(t, severeErrors)
	assert.Empty(t, finalResult.ComponentResults, "nil results should not be added to ComponentResults")
	require.Len(t, finalResult.AllDiagnostics, 1,
		"diagnostics from nil results should still be collected")
	assert.Equal(t, "test error", finalResult.AllDiagnostics[0].Message)
}

func TestAggregateAnnotationResults_ValidResult(t *testing.T) {
	t.Parallel()

	sourcePath := "/project/pages/index.pk"
	hashedName := "index_abc123"

	vm := &annotator_dto.VirtualModule{
		Graph: &annotator_dto.ComponentGraph{
			PathToHashedName: map[string]string{
				sourcePath: hashedName,
			},
		},
		ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
			hashedName: {
				Source: &annotator_dto.ParsedComponent{
					SourcePath: sourcePath,
				},
				HashedName: hashedName,
			},
		},
	}

	annotationResult := &annotator_dto.AnnotationResult{
		AnnotatedAST: &ast_domain.TemplateAST{
			SourcePath: &sourcePath,
		},
		VirtualModule: vm,
	}

	resultsChan := make(chan *annotationJobResult, 1)
	resultsChan <- &annotationJobResult{
		result:      annotationResult,
		diagnostics: nil,
	}
	close(resultsChan)

	finalResult := &annotator_dto.ProjectAnnotationResult{
		ComponentResults: make(map[string]*annotator_dto.AnnotationResult),
		AllDiagnostics:   make([]*ast_domain.Diagnostic, 0),
	}

	severeErrors := aggregateAnnotationResults(resultsChan, finalResult)

	assert.Empty(t, severeErrors)
	require.Len(t, finalResult.ComponentResults, 1)
	assert.Equal(t, annotationResult, finalResult.ComponentResults[hashedName])
}

func TestAggregateAnnotationResults_MissingASTReturnsError(t *testing.T) {
	t.Parallel()

	resultsChan := make(chan *annotationJobResult, 1)
	resultsChan <- &annotationJobResult{
		result: &annotator_dto.AnnotationResult{
			AnnotatedAST:  nil,
			VirtualModule: &annotator_dto.VirtualModule{},
		},
		diagnostics: nil,
	}
	close(resultsChan)

	finalResult := &annotator_dto.ProjectAnnotationResult{
		ComponentResults: make(map[string]*annotator_dto.AnnotationResult),
		AllDiagnostics:   make([]*ast_domain.Diagnostic, 0),
	}

	severeErrors := aggregateAnnotationResults(resultsChan, finalResult)

	require.Len(t, severeErrors, 1)
	assert.Contains(t, severeErrors[0].Error(), "internal error")
	assert.Empty(t, finalResult.ComponentResults)
}

func TestAggregateAnnotationResults_DiagnosticsAccumulated(t *testing.T) {
	t.Parallel()

	resultsChan := make(chan *annotationJobResult, 3)

	for i := range 3 {
		diagnostic := ast_domain.NewDiagnostic(
			ast_domain.Warning,
			"warning message",
			"",
			ast_domain.Location{Line: i + 1, Column: 1, Offset: 0},
			"/test.pk",
		)
		resultsChan <- &annotationJobResult{
			result:      nil,
			diagnostics: []*ast_domain.Diagnostic{diagnostic},
		}
	}
	close(resultsChan)

	finalResult := &annotator_dto.ProjectAnnotationResult{
		ComponentResults: make(map[string]*annotator_dto.AnnotationResult),
		AllDiagnostics:   make([]*ast_domain.Diagnostic, 0),
	}

	severeErrors := aggregateAnnotationResults(resultsChan, finalResult)

	assert.Empty(t, severeErrors)
	assert.Len(t, finalResult.AllDiagnostics, 3,
		"all diagnostics from all job results should be accumulated")
}

func TestAggregateAnnotationResults_MultipleValidResults(t *testing.T) {
	t.Parallel()

	sourcePathA := "/project/pages/home.pk"
	hashedNameA := "home_aaa"
	sourcePathB := "/project/pages/about.pk"
	hashedNameB := "about_bbb"

	vm := &annotator_dto.VirtualModule{
		Graph: &annotator_dto.ComponentGraph{
			PathToHashedName: map[string]string{
				sourcePathA: hashedNameA,
				sourcePathB: hashedNameB,
			},
		},
		ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
			hashedNameA: {
				Source:     &annotator_dto.ParsedComponent{SourcePath: sourcePathA},
				HashedName: hashedNameA,
			},
			hashedNameB: {
				Source:     &annotator_dto.ParsedComponent{SourcePath: sourcePathB},
				HashedName: hashedNameB,
			},
		},
	}

	resultA := &annotator_dto.AnnotationResult{
		AnnotatedAST:  &ast_domain.TemplateAST{SourcePath: &sourcePathA},
		VirtualModule: vm,
	}
	resultB := &annotator_dto.AnnotationResult{
		AnnotatedAST:  &ast_domain.TemplateAST{SourcePath: &sourcePathB},
		VirtualModule: vm,
	}

	resultsChan := make(chan *annotationJobResult, 2)
	resultsChan <- &annotationJobResult{result: resultA, diagnostics: nil}
	resultsChan <- &annotationJobResult{result: resultB, diagnostics: nil}
	close(resultsChan)

	finalResult := &annotator_dto.ProjectAnnotationResult{
		ComponentResults: make(map[string]*annotator_dto.AnnotationResult),
		AllDiagnostics:   make([]*ast_domain.Diagnostic, 0),
	}

	severeErrors := aggregateAnnotationResults(resultsChan, finalResult)

	assert.Empty(t, severeErrors)
	require.Len(t, finalResult.ComponentResults, 2)
	assert.Same(t, resultA, finalResult.ComponentResults[hashedNameA])
	assert.Same(t, resultB, finalResult.ComponentResults[hashedNameB])
}

func TestHandlePhase2Completion_NoDiagnostics(t *testing.T) {
	t.Parallel()

	finalResult := &annotator_dto.ProjectAnnotationResult{
		ComponentResults: make(map[string]*annotator_dto.AnnotationResult),
		AllDiagnostics:   []*ast_domain.Diagnostic{},
	}
	logStore := newTestLogStore(t)
	options := &annotationOptions{faultTolerant: false}

	result, returnedLogStore, err := handlePhase2Completion(context.Background(), finalResult, logStore, options)

	require.NoError(t, err)
	assert.Same(t, finalResult, result)
	assert.Same(t, logStore, returnedLogStore)
}

func TestHandlePhase2Completion_WarningsOnly(t *testing.T) {
	t.Parallel()

	finalResult := &annotator_dto.ProjectAnnotationResult{
		ComponentResults: make(map[string]*annotator_dto.AnnotationResult),
		AllDiagnostics: []*ast_domain.Diagnostic{
			ast_domain.NewDiagnostic(
				ast_domain.Warning,
				"some warning",
				"",
				ast_domain.Location{Line: 1, Column: 1, Offset: 0},
				"/test.pk",
			),
		},
	}
	logStore := newTestLogStore(t)
	options := &annotationOptions{faultTolerant: false}

	result, returnedLogStore, err := handlePhase2Completion(context.Background(), finalResult, logStore, options)

	require.NoError(t, err, "warnings alone should not produce an error")
	assert.Same(t, finalResult, result)
	assert.Same(t, logStore, returnedLogStore)
}

func TestHandlePhase2Completion_ErrorsWithFaultTolerance(t *testing.T) {
	t.Parallel()

	finalResult := &annotator_dto.ProjectAnnotationResult{
		ComponentResults: make(map[string]*annotator_dto.AnnotationResult),
		AllDiagnostics: []*ast_domain.Diagnostic{
			ast_domain.NewDiagnostic(
				ast_domain.Error,
				"critical error",
				"",
				ast_domain.Location{Line: 1, Column: 1, Offset: 0},
				"/test.pk",
			),
		},
	}
	logStore := newTestLogStore(t)
	options := &annotationOptions{faultTolerant: true}

	result, returnedLogStore, err := handlePhase2Completion(context.Background(), finalResult, logStore, options)

	require.NoError(t, err, "fault-tolerant mode should not return an error")
	assert.Same(t, finalResult, result)
	assert.Same(t, logStore, returnedLogStore)
}

func TestHandlePhase2Completion_ErrorsWithoutFaultTolerance(t *testing.T) {
	t.Parallel()

	finalResult := &annotator_dto.ProjectAnnotationResult{
		ComponentResults: make(map[string]*annotator_dto.AnnotationResult),
		AllDiagnostics: []*ast_domain.Diagnostic{
			ast_domain.NewDiagnostic(
				ast_domain.Error,
				"critical error",
				"",
				ast_domain.Location{Line: 1, Column: 1, Offset: 0},
				"/test.pk",
			),
		},
	}
	logStore := newTestLogStore(t)
	options := &annotationOptions{faultTolerant: false}

	result, returnedLogStore, err := handlePhase2Completion(context.Background(), finalResult, logStore, options)

	require.Error(t, err)
	_, ok := errors.AsType[*SemanticError](err)
	require.True(t, ok,
		"error should be a *SemanticError")
	assert.Same(t, finalResult, result)
	assert.Same(t, logStore, returnedLogStore)
}

func TestHandlePhase2Completion_DeduplicatesDiagnostics(t *testing.T) {
	t.Parallel()

	diagnostic := ast_domain.NewDiagnostic(
		ast_domain.Warning,
		"duplicate warning",
		"expr",
		ast_domain.Location{Line: 5, Column: 10, Offset: 0},
		"/test.pk",
	)
	finalResult := &annotator_dto.ProjectAnnotationResult{
		ComponentResults: make(map[string]*annotator_dto.AnnotationResult),
		AllDiagnostics:   []*ast_domain.Diagnostic{diagnostic, diagnostic, diagnostic},
	}
	logStore := newTestLogStore(t)
	options := &annotationOptions{faultTolerant: false}

	result, _, err := handlePhase2Completion(context.Background(), finalResult, logStore, options)

	require.NoError(t, err)
	assert.Len(t, result.AllDiagnostics, 1,
		"duplicate diagnostics should be removed")
}

func TestHandlePhase2Completion_SemanticErrorContainsDiagnostics(t *testing.T) {
	t.Parallel()

	errorDiag := ast_domain.NewDiagnostic(
		ast_domain.Error,
		"undefined variable 'x'",
		"x",
		ast_domain.Location{Line: 10, Column: 5, Offset: 0},
		"/project/pages/broken.pk",
	)

	finalResult := &annotator_dto.ProjectAnnotationResult{
		ComponentResults: make(map[string]*annotator_dto.AnnotationResult),
		AllDiagnostics:   []*ast_domain.Diagnostic{errorDiag},
	}
	logStore := newTestLogStore(t)
	options := &annotationOptions{faultTolerant: false}

	_, _, err := handlePhase2Completion(context.Background(), finalResult, logStore, options)

	require.Error(t, err)
	semanticErr, ok := errors.AsType[*SemanticError](err)
	require.True(t, ok)
	require.Len(t, semanticErr.Diagnostics, 1)
	assert.Equal(t, "undefined variable 'x'", semanticErr.Diagnostics[0].Message)
	assert.True(t, strings.Contains(semanticErr.Error(), "semantic validation error"),
		"SemanticError.Error() should mention semantic validation")
}

func TestHandlePhase2Completion_MixedDiagnosticSeverities(t *testing.T) {
	t.Parallel()

	finalResult := &annotator_dto.ProjectAnnotationResult{
		ComponentResults: make(map[string]*annotator_dto.AnnotationResult),
		AllDiagnostics: []*ast_domain.Diagnostic{
			ast_domain.NewDiagnostic(
				ast_domain.Warning,
				"a warning",
				"",
				ast_domain.Location{Line: 1, Column: 1, Offset: 0},
				"/test.pk",
			),
			ast_domain.NewDiagnostic(
				ast_domain.Error,
				"an error",
				"",
				ast_domain.Location{Line: 2, Column: 1, Offset: 0},
				"/test.pk",
			),
			ast_domain.NewDiagnostic(
				ast_domain.Info,
				"some info",
				"",
				ast_domain.Location{Line: 3, Column: 1, Offset: 0},
				"/test.pk",
			),
		},
	}
	logStore := newTestLogStore(t)

	options := &annotationOptions{faultTolerant: false}
	_, _, err := handlePhase2Completion(context.Background(), finalResult, logStore, options)
	require.Error(t, err)

	finalResult.AllDiagnostics = []*ast_domain.Diagnostic{
		ast_domain.NewDiagnostic(
			ast_domain.Warning,
			"a warning",
			"",
			ast_domain.Location{Line: 1, Column: 1, Offset: 0},
			"/test.pk",
		),
		ast_domain.NewDiagnostic(
			ast_domain.Error,
			"an error",
			"",
			ast_domain.Location{Line: 2, Column: 1, Offset: 0},
			"/test.pk",
		),
	}
	optionsTolerant := &annotationOptions{faultTolerant: true}
	_, _, err = handlePhase2Completion(context.Background(), finalResult, logStore, optionsTolerant)
	require.NoError(t, err)
}

func TestRunAssetAggregation_EmptyResults(t *testing.T) {
	t.Parallel()

	finalResult := &annotator_dto.ProjectAnnotationResult{
		ComponentResults:   make(map[string]*annotator_dto.AnnotationResult),
		FinalAssetManifest: nil,
	}

	runAssetAggregation(context.Background(), finalResult)

	assert.NotNil(t, finalResult.FinalAssetManifest,
		"FinalAssetManifest should be set even when empty")
	assert.Empty(t, finalResult.FinalAssetManifest)
}

func TestRunAssetAggregation_WithResults(t *testing.T) {
	t.Parallel()

	finalResult := &annotator_dto.ProjectAnnotationResult{
		ComponentResults: map[string]*annotator_dto.AnnotationResult{
			"comp_a": {
				AssetDependencies: []*annotator_dto.StaticAssetDependency{
					{
						SourcePath:           "img/hero.jpg",
						AssetType:            "image",
						TransformationParams: map[string]string{},
					},
				},
			},
			"comp_b": {
				AssetDependencies: []*annotator_dto.StaticAssetDependency{
					{
						SourcePath:           "img/logo.png",
						AssetType:            "image",
						TransformationParams: map[string]string{},
					},
				},
			},
		},
		FinalAssetManifest: nil,
	}

	runAssetAggregation(context.Background(), finalResult)

	require.NotNil(t, finalResult.FinalAssetManifest)
	assert.Len(t, finalResult.FinalAssetManifest, 2)
}

func TestRunAssetAggregation_DeduplicatesAcrossComponents(t *testing.T) {
	t.Parallel()

	finalResult := &annotator_dto.ProjectAnnotationResult{
		ComponentResults: map[string]*annotator_dto.AnnotationResult{
			"comp_a": {
				AssetDependencies: []*annotator_dto.StaticAssetDependency{
					{
						SourcePath:           "img/shared.jpg",
						AssetType:            "image",
						TransformationParams: map[string]string{"width": "400"},
					},
				},
			},
			"comp_b": {
				AssetDependencies: []*annotator_dto.StaticAssetDependency{
					{
						SourcePath:           "img/shared.jpg",
						AssetType:            "image",
						TransformationParams: map[string]string{"width": "800"},
					},
				},
			},
		},
		FinalAssetManifest: nil,
	}

	runAssetAggregation(context.Background(), finalResult)

	require.NotNil(t, finalResult.FinalAssetManifest)
	assert.Len(t, finalResult.FinalAssetManifest, 1,
		"same asset from different components should be merged")
}

func TestFeedAnnotationJobs_EmptyComponents(t *testing.T) {
	t.Parallel()

	jobs := make(chan *annotationJob, 1)
	components := map[string]*annotator_dto.VirtualComponent{}
	prepareJob := func(_ context.Context, _ *annotator_dto.VirtualComponent) *annotationJob {
		return &annotationJob{}
	}

	feedAnnotationJobs(context.Background(), jobs, prepareJob, components)

	_, open := <-jobs
	assert.False(t, open, "jobs channel should be closed")
}

func TestFeedAnnotationJobs_AllComponentsSent(t *testing.T) {
	t.Parallel()

	components := map[string]*annotator_dto.VirtualComponent{
		"comp_a": {HashedName: "comp_a"},
		"comp_b": {HashedName: "comp_b"},
		"comp_c": {HashedName: "comp_c"},
	}

	jobs := make(chan *annotationJob, len(components))
	prepareJob := func(_ context.Context, vc *annotator_dto.VirtualComponent) *annotationJob {
		return &annotationJob{vc: vc}
	}

	feedAnnotationJobs(context.Background(), jobs, prepareJob, components)

	receivedNames := make(map[string]bool)
	for job := range jobs {
		receivedNames[job.vc.HashedName] = true
	}

	assert.Len(t, receivedNames, 3)
	assert.True(t, receivedNames["comp_a"])
	assert.True(t, receivedNames["comp_b"])
	assert.True(t, receivedNames["comp_c"])
}

func TestFeedAnnotationJobs_CancelledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	components := map[string]*annotator_dto.VirtualComponent{
		"comp_a": {HashedName: "comp_a"},
		"comp_b": {HashedName: "comp_b"},
	}

	jobs := make(chan *annotationJob)
	prepareJob := func(_ context.Context, vc *annotator_dto.VirtualComponent) *annotationJob {
		return &annotationJob{vc: vc}
	}

	feedAnnotationJobs(ctx, jobs, prepareJob, components)

	_, open := <-jobs
	assert.False(t, open, "jobs channel should be closed after cancellation")
}

func newTestLogStore(t *testing.T) *CompilationLogStore {
	t.Helper()
	store, err := NewCompilationLogStore(context.Background(), false, "", slog.LevelInfo)
	require.NoError(t, err)
	return store
}

func TestRunSrcsetAnnotation_EmptyResults(t *testing.T) {
	t.Parallel()

	service := &AnnotatorService{}
	finalResult := &annotator_dto.ProjectAnnotationResult{
		ComponentResults: make(map[string]*annotator_dto.AnnotationResult),
	}

	assert.NotPanics(t, func() {
		service.runSrcsetAnnotation(context.Background(), finalResult)
	})
}

func TestRunSrcsetAnnotation_WithComponentsNoAssets(t *testing.T) {
	t.Parallel()

	service := &AnnotatorService{}
	finalResult := &annotator_dto.ProjectAnnotationResult{
		ComponentResults: map[string]*annotator_dto.AnnotationResult{
			"comp_a": {
				AnnotatedAST: &ast_domain.TemplateAST{
					RootNodes: []*ast_domain.TemplateNode{
						{TagName: "div", NodeType: ast_domain.NodeElement},
					},
				},
				AssetDependencies: nil,
			},
		},
	}

	assert.NotPanics(t, func() {
		service.runSrcsetAnnotation(context.Background(), finalResult)
	})
}

func TestRunSrcsetAnnotation_WithAssetDependencies(t *testing.T) {
	t.Parallel()

	service := &AnnotatorService{}
	finalResult := &annotator_dto.ProjectAnnotationResult{
		ComponentResults: map[string]*annotator_dto.AnnotationResult{
			"comp_a": {
				AnnotatedAST: &ast_domain.TemplateAST{
					RootNodes: []*ast_domain.TemplateNode{
						{TagName: "img", NodeType: ast_domain.NodeElement},
					},
				},
				AssetDependencies: []*annotator_dto.StaticAssetDependency{
					{
						SourcePath:           "img/hero.jpg",
						AssetType:            "image",
						TransformationParams: map[string]string{},
					},
				},
			},
		},
	}

	assert.NotPanics(t, func() {
		service.runSrcsetAnnotation(context.Background(), finalResult)
	})
}

func TestRunSrcsetAnnotation_NilAnnotatedAST(t *testing.T) {
	t.Parallel()

	service := &AnnotatorService{}
	finalResult := &annotator_dto.ProjectAnnotationResult{
		ComponentResults: map[string]*annotator_dto.AnnotationResult{
			"comp_a": {
				AnnotatedAST: nil,
				AssetDependencies: []*annotator_dto.StaticAssetDependency{
					{
						SourcePath:           "img/hero.jpg",
						AssetType:            "image",
						TransformationParams: map[string]string{},
					},
				},
			},
		},
	}

	assert.NotPanics(t, func() {
		service.runSrcsetAnnotation(context.Background(), finalResult)
	})
}

func TestFeedAnnotationJobs_PrepareJobIsCalled(t *testing.T) {
	t.Parallel()

	callCount := 0
	components := map[string]*annotator_dto.VirtualComponent{
		"comp_a": {HashedName: "comp_a"},
	}

	jobs := make(chan *annotationJob, len(components))
	prepareJob := func(_ context.Context, vc *annotator_dto.VirtualComponent) *annotationJob {
		callCount++
		return &annotationJob{vc: vc}
	}

	feedAnnotationJobs(context.Background(), jobs, prepareJob, components)

	receivedJobs := 0
	for range jobs {
		receivedJobs++
	}

	assert.Equal(t, 1, receivedJobs)
	assert.Equal(t, 1, callCount, "prepareJob should be called once per component")
}

func TestCreateAnnotationWorker_ProcessesJobs(t *testing.T) {
	t.Parallel()

	service := &AnnotatorService{}
	jobs := make(chan *annotationJob, 1)
	results := make(chan *annotationJobResult, 1)

	vm := &annotator_dto.VirtualModule{
		Graph: &annotator_dto.ComponentGraph{
			PathToHashedName: make(map[string]string),
		},
		ComponentsByHash:   make(map[string]*annotator_dto.VirtualComponent),
		ComponentsByGoPath: make(map[string]*annotator_dto.VirtualComponent),
	}

	workerConfig := &annotationWorkerConfig{
		componentGraph: &annotator_dto.ComponentGraph{
			PathToHashedName: make(map[string]string),
		},
		virtualModule: vm,
		typeResolver:  nil,
		actions:       nil,
		options:       &annotationOptions{faultTolerant: true},
	}

	workerFunction := service.createAnnotationWorker(context.Background(), jobs, results, workerConfig)

	close(jobs)

	err := workerFunction()
	assert.NoError(t, err, "worker should return nil when no jobs are received")
}

func TestAggregateAnnotationResults_MixedValidAndNilResults(t *testing.T) {
	t.Parallel()

	sourcePath := "/project/pages/home.pk"
	hashedName := "home_abc"

	vm := &annotator_dto.VirtualModule{
		Graph: &annotator_dto.ComponentGraph{
			PathToHashedName: map[string]string{
				sourcePath: hashedName,
			},
		},
		ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
			hashedName: {
				Source:     &annotator_dto.ParsedComponent{SourcePath: sourcePath},
				HashedName: hashedName,
			},
		},
	}

	validResult := &annotator_dto.AnnotationResult{
		AnnotatedAST:  &ast_domain.TemplateAST{SourcePath: &sourcePath},
		VirtualModule: vm,
	}

	errorDiag := ast_domain.NewDiagnostic(
		ast_domain.Error,
		"component failed",
		"",
		ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		"/project/pages/broken.pk",
	)

	resultsChan := make(chan *annotationJobResult, 2)
	resultsChan <- &annotationJobResult{result: validResult, diagnostics: nil}
	resultsChan <- &annotationJobResult{result: nil, diagnostics: []*ast_domain.Diagnostic{errorDiag}}
	close(resultsChan)

	finalResult := &annotator_dto.ProjectAnnotationResult{
		ComponentResults: make(map[string]*annotator_dto.AnnotationResult),
		AllDiagnostics:   make([]*ast_domain.Diagnostic, 0),
	}

	severeErrors := aggregateAnnotationResults(resultsChan, finalResult)

	assert.Empty(t, severeErrors)
	require.Len(t, finalResult.ComponentResults, 1, "only the valid result should be added")
	assert.Same(t, validResult, finalResult.ComponentResults[hashedName])
	require.Len(t, finalResult.AllDiagnostics, 1, "diagnostic from nil result should be collected")
	assert.Equal(t, "component failed", finalResult.AllDiagnostics[0].Message)
}
