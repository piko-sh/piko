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

package coordinator_domain

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/coordinator/coordinator_dto"
	"piko.sh/piko/internal/generator/generator_dto"
)

type mockDiagnosticOutput struct {
	Diagnostics []*ast_domain.Diagnostic
	Calls       int
	IsError     bool
}

func (m *mockDiagnosticOutput) OutputDiagnostics(diagnostics []*ast_domain.Diagnostic, _ map[string][]byte, isError bool) {
	m.Calls++
	m.Diagnostics = diagnostics
	m.IsError = isError
}

type mockCodeEmitter struct {
	Error       error
	Result      []byte
	Diagnostics []*ast_domain.Diagnostic
	Calls       int
}

func (m *mockCodeEmitter) EmitCode(
	_ context.Context,
	_ *annotator_dto.AnnotationResult,
	_ generator_dto.GenerateRequest,
) ([]byte, []*ast_domain.Diagnostic, error) {
	m.Calls++
	return m.Result, m.Diagnostics, m.Error
}

func TestOutputDiagnosticsIfPresent(t *testing.T) {
	t.Run("returns false when build result is nil", func(t *testing.T) {
		s := &coordinatorService{}
		result := s.outputDiagnosticsIfPresent(context.Background(), nil, nil, nil)
		assert.False(t, result)
	})

	t.Run("returns false when diagnostics are empty", func(t *testing.T) {
		s := &coordinatorService{}
		buildResult := &annotator_dto.ProjectAnnotationResult{
			AllDiagnostics: nil,
		}
		result := s.outputDiagnosticsIfPresent(context.Background(), buildResult, nil, nil)
		assert.False(t, result)
	})

	t.Run("outputs diagnostics as errors when build failed", func(t *testing.T) {
		diagOutput := &mockDiagnosticOutput{}
		s := &coordinatorService{
			diagnosticOutput: diagOutput,
		}
		buildResult := &annotator_dto.ProjectAnnotationResult{
			AllDiagnostics: []*ast_domain.Diagnostic{
				{Message: "test error"},
			},
		}
		buildErr := errors.New("build failed")

		result := s.outputDiagnosticsIfPresent(context.Background(), buildResult, buildErr, nil)

		assert.True(t, result)
		assert.Equal(t, 1, diagOutput.Calls)
		assert.True(t, diagOutput.IsError)
		assert.Len(t, diagOutput.Diagnostics, 1)
	})

	t.Run("outputs diagnostics as warnings when build succeeded", func(t *testing.T) {
		diagOutput := &mockDiagnosticOutput{}
		s := &coordinatorService{
			diagnosticOutput: diagOutput,
		}
		buildResult := &annotator_dto.ProjectAnnotationResult{
			AllDiagnostics: []*ast_domain.Diagnostic{
				{Message: "test warning"},
			},
		}

		result := s.outputDiagnosticsIfPresent(context.Background(), buildResult, nil, nil)

		assert.True(t, result)
		assert.Equal(t, 1, diagOutput.Calls)
		assert.False(t, diagOutput.IsError)
	})

	t.Run("handles nil diagnostic output gracefully", func(t *testing.T) {
		s := &coordinatorService{
			diagnosticOutput: nil,
		}
		buildResult := &annotator_dto.ProjectAnnotationResult{
			AllDiagnostics: []*ast_domain.Diagnostic{
				{Message: "test diagnostic"},
			},
		}

		result := s.outputDiagnosticsIfPresent(context.Background(), buildResult, nil, nil)
		assert.True(t, result)
	})
}

func TestTryGenerateArtefacts(t *testing.T) {
	t.Run("returns nil when code emitter is nil", func(t *testing.T) {
		s := &coordinatorService{
			codeEmitter: nil,
		}

		err := s.tryGenerateArtefacts(
			context.Background(),
			nil,
			&annotator_dto.ProjectAnnotationResult{},
			&coordinator_dto.BuildRequest{},
			"",
		)

		assert.NoError(t, err)
	})

	t.Run("generates artefacts when emitter is present", func(t *testing.T) {
		emitter := &mockCodeEmitter{
			Result: []byte("generated code"),
		}
		s := &coordinatorService{
			codeEmitter: emitter,
			status:      buildStatus{State: stateBuilding},
		}

		buildResult := &annotator_dto.ProjectAnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
					"hash1": {
						HashedName:        "hash1",
						VirtualGoFilePath: "/virtual/hash1.go",
						Source:            &annotator_dto.ParsedComponent{SourcePath: "/src/comp1.pk"},
					},
				},
			},
			ComponentResults: map[string]*annotator_dto.AnnotationResult{
				"hash1": {},
			},
		}

		err := s.tryGenerateArtefacts(
			context.Background(),
			nil,
			buildResult,
			&coordinator_dto.BuildRequest{},
			"",
		)

		assert.NoError(t, err)
		assert.Equal(t, 1, emitter.Calls)
		assert.NotNil(t, buildResult.FinalGeneratedArtefacts)
	})

	t.Run("returns error when emitter fails", func(t *testing.T) {
		emitter := &mockCodeEmitter{
			Error: errors.New("emitter failed"),
		}
		s := &coordinatorService{
			codeEmitter: emitter,
			status:      buildStatus{State: stateBuilding},
		}

		buildResult := &annotator_dto.ProjectAnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
					"hash1": {
						HashedName:        "hash1",
						VirtualGoFilePath: "/virtual/hash1.go",
						Source:            &annotator_dto.ParsedComponent{SourcePath: "/src/comp1.pk"},
					},
				},
			},
			ComponentResults: map[string]*annotator_dto.AnnotationResult{
				"hash1": {},
			},
		}

		err := s.tryGenerateArtefacts(
			context.Background(),
			nil,
			buildResult,
			&coordinator_dto.BuildRequest{},
			"",
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "emitter failed")
	})
}

func TestTier1CacheResult(t *testing.T) {
	t.Run("correctly represents cache miss", func(t *testing.T) {
		result := tier1CacheResult{
			scriptHashes:      map[string]string{"file.pk": "hash1"},
			entry:             nil,
			introspectionHash: "abc123",
			useFastPath:       false,
		}

		assert.False(t, result.useFastPath)
		assert.Nil(t, result.entry)
		assert.NotEmpty(t, result.introspectionHash)
	})

	t.Run("correctly represents cache hit", func(t *testing.T) {
		entry := &IntrospectionCacheEntry{
			Version:   1,
			Timestamp: time.Now(),
		}
		result := tier1CacheResult{
			scriptHashes:      map[string]string{"file.pk": "hash1"},
			entry:             entry,
			introspectionHash: "abc123",
			useFastPath:       true,
		}

		assert.True(t, result.useFastPath)
		assert.NotNil(t, result.entry)
	})
}

func TestCacheIntrospectionResults(t *testing.T) {
	t.Run("does nothing when introspection hash is empty", func(t *testing.T) {
		cache := newMockIntrospectionCache()
		s := &coordinatorService{
			introspectionCache: cache,
		}

		tier1Result := tier1CacheResult{
			introspectionHash: "",
			scriptHashes:      nil,
		}

		s.cacheIntrospectionResults(
			context.Background(),
			tier1Result,
			&annotator_dto.ComponentGraph{},
			&annotator_dto.VirtualModule{},
			&annotator_domain.TypeResolver{},
		)

		assert.Equal(t, 0, cache.setCalls)
	})

	t.Run("does nothing when script hashes are nil", func(t *testing.T) {
		cache := newMockIntrospectionCache()
		s := &coordinatorService{
			introspectionCache: cache,
		}

		tier1Result := tier1CacheResult{
			introspectionHash: "hash123",
			scriptHashes:      nil,
		}

		s.cacheIntrospectionResults(
			context.Background(),
			tier1Result,
			&annotator_dto.ComponentGraph{},
			&annotator_dto.VirtualModule{},
			&annotator_domain.TypeResolver{},
		)

		assert.Equal(t, 0, cache.setCalls)
	})

	t.Run("does nothing when component graph is nil", func(t *testing.T) {
		cache := newMockIntrospectionCache()
		s := &coordinatorService{
			introspectionCache: cache,
		}

		tier1Result := tier1CacheResult{
			introspectionHash: "hash123",
			scriptHashes:      map[string]string{"file.pk": "hash"},
		}

		s.cacheIntrospectionResults(
			context.Background(),
			tier1Result,
			nil,
			&annotator_dto.VirtualModule{},
			&annotator_domain.TypeResolver{},
		)

		assert.Equal(t, 0, cache.setCalls)
	})

	t.Run("caches results when all parameters are valid", func(t *testing.T) {
		cache := newMockIntrospectionCache()
		s := &coordinatorService{
			introspectionCache: cache,
		}

		tier1Result := tier1CacheResult{
			introspectionHash: "hash123",
			scriptHashes:      map[string]string{"file.pk": "scripthash"},
		}

		s.cacheIntrospectionResults(
			context.Background(),
			tier1Result,
			&annotator_dto.ComponentGraph{},
			&annotator_dto.VirtualModule{},
			&annotator_domain.TypeResolver{},
		)

		assert.Equal(t, 1, cache.setCalls)
	})
}

func TestGenerateArtefacts(t *testing.T) {
	t.Run("returns error when virtual module is nil", func(t *testing.T) {
		s := &coordinatorService{
			codeEmitter: &mockCodeEmitter{Result: []byte("code")},
		}

		buildResult := &annotator_dto.ProjectAnnotationResult{
			VirtualModule: nil,
		}

		artefacts, err := s.generateArtefacts(context.Background(), buildResult)

		assert.Error(t, err)
		assert.Nil(t, artefacts)
		assert.Contains(t, err.Error(), "no virtual module")
	})

	t.Run("generates artefacts for all components", func(t *testing.T) {
		emitter := &mockCodeEmitter{
			Result: []byte("generated code"),
		}
		s := &coordinatorService{
			codeEmitter: emitter,
		}

		buildResult := &annotator_dto.ProjectAnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
					"hash1": {
						HashedName:        "hash1",
						VirtualGoFilePath: "/virtual/hash1.go",
						Source:            &annotator_dto.ParsedComponent{SourcePath: "/src/comp1.pk"},
					},
					"hash2": {
						HashedName:        "hash2",
						VirtualGoFilePath: "/virtual/hash2.go",
						Source:            &annotator_dto.ParsedComponent{SourcePath: "/src/comp2.pk"},
					},
				},
			},
			ComponentResults: map[string]*annotator_dto.AnnotationResult{
				"hash1": {},
				"hash2": {},
			},
		}

		artefacts, err := s.generateArtefacts(context.Background(), buildResult)

		require.NoError(t, err)
		assert.Len(t, artefacts, 2)
		assert.Equal(t, 2, emitter.Calls)
	})

	t.Run("skips components without annotation results", func(t *testing.T) {
		emitter := &mockCodeEmitter{
			Result: []byte("generated code"),
		}
		s := &coordinatorService{
			codeEmitter: emitter,
		}

		buildResult := &annotator_dto.ProjectAnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
					"hash1": {
						HashedName:        "hash1",
						VirtualGoFilePath: "/virtual/hash1.go",
						Source:            &annotator_dto.ParsedComponent{SourcePath: "/src/comp1.pk"},
					},
					"hash2": {
						HashedName:        "hash2",
						VirtualGoFilePath: "/virtual/hash2.go",
						Source:            &annotator_dto.ParsedComponent{SourcePath: "/src/comp2.pk"},
					},
				},
			},
			ComponentResults: map[string]*annotator_dto.AnnotationResult{
				"hash1": {},
			},
		}

		artefacts, err := s.generateArtefacts(context.Background(), buildResult)

		require.NoError(t, err)
		assert.Len(t, artefacts, 1)
		assert.Equal(t, 1, emitter.Calls)
	})

	t.Run("returns error when emitter fails", func(t *testing.T) {
		emitter := &mockCodeEmitter{
			Error: errors.New("emit failed"),
		}
		s := &coordinatorService{
			codeEmitter: emitter,
		}

		buildResult := &annotator_dto.ProjectAnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
					"hash1": {
						HashedName:        "hash1",
						VirtualGoFilePath: "/virtual/hash1.go",
						Source:            &annotator_dto.ParsedComponent{SourcePath: "/src/comp1.pk"},
					},
				},
			},
			ComponentResults: map[string]*annotator_dto.AnnotationResult{
				"hash1": {},
			},
		}

		artefacts, err := s.generateArtefacts(context.Background(), buildResult)

		assert.Error(t, err)
		assert.Nil(t, artefacts)
	})
}

func TestGenerateSingleArtefact(t *testing.T) {
	t.Run("generates artefact successfully", func(t *testing.T) {
		emitter := &mockCodeEmitter{
			Result: []byte("generated code content"),
		}
		s := &coordinatorService{
			codeEmitter: emitter,
		}

		buildResult := &annotator_dto.ProjectAnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{},
		}
		vc := &annotator_dto.VirtualComponent{
			HashedName:             "test_hash",
			VirtualGoFilePath:      "/virtual/test.go",
			CanonicalGoPackagePath: "example.com/pkg",
			Source:                 &annotator_dto.ParsedComponent{SourcePath: "/src/test.pk"},
		}
		annotationResult := &annotator_dto.AnnotationResult{}

		artefact, err := s.generateSingleArtefact(
			context.Background(),
			buildResult,
			"test_hash",
			vc,
			annotationResult,
		)

		require.NoError(t, err)
		require.NotNil(t, artefact)
		assert.Equal(t, []byte("generated code content"), artefact.Content)
		assert.Equal(t, "/virtual/test.go", artefact.SuggestedPath)
		assert.Same(t, vc, artefact.Component)
	})

	t.Run("sets virtual module on annotation result if missing", func(t *testing.T) {
		emitter := &mockCodeEmitter{
			Result: []byte("code"),
		}
		s := &coordinatorService{
			codeEmitter: emitter,
		}

		virtualModule := &annotator_dto.VirtualModule{}
		buildResult := &annotator_dto.ProjectAnnotationResult{
			VirtualModule: virtualModule,
		}
		vc := &annotator_dto.VirtualComponent{
			HashedName:        "test_hash",
			VirtualGoFilePath: "/virtual/test.go",
			Source:            &annotator_dto.ParsedComponent{SourcePath: "/src/test.pk"},
		}
		annotationResult := &annotator_dto.AnnotationResult{
			VirtualModule: nil,
		}

		_, err := s.generateSingleArtefact(
			context.Background(),
			buildResult,
			"test_hash",
			vc,
			annotationResult,
		)

		require.NoError(t, err)
		assert.Same(t, virtualModule, annotationResult.VirtualModule)
	})

	t.Run("returns error when emitter fails", func(t *testing.T) {
		emitter := &mockCodeEmitter{
			Error: errors.New("code generation failed"),
		}
		s := &coordinatorService{
			codeEmitter: emitter,
		}

		buildResult := &annotator_dto.ProjectAnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{},
		}
		vc := &annotator_dto.VirtualComponent{
			HashedName:        "test_hash",
			VirtualGoFilePath: "/virtual/test.go",
			Source:            &annotator_dto.ParsedComponent{SourcePath: "/src/test.pk"},
		}
		annotationResult := &annotator_dto.AnnotationResult{}

		artefact, err := s.generateSingleArtefact(
			context.Background(),
			buildResult,
			"test_hash",
			vc,
			annotationResult,
		)

		assert.Error(t, err)
		assert.Nil(t, artefact)
		assert.Contains(t, err.Error(), "code generation failed")
	})

	t.Run("handles emitter returning diagnostics", func(t *testing.T) {
		emitter := &mockCodeEmitter{
			Result: []byte("code with warnings"),
			Diagnostics: []*ast_domain.Diagnostic{
				{Message: "warning: deprecated function"},
			},
		}
		s := &coordinatorService{
			codeEmitter: emitter,
		}

		buildResult := &annotator_dto.ProjectAnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{},
		}
		vc := &annotator_dto.VirtualComponent{
			HashedName:        "test_hash",
			VirtualGoFilePath: "/virtual/test.go",
			Source:            &annotator_dto.ParsedComponent{SourcePath: "/src/test.pk"},
		}
		annotationResult := &annotator_dto.AnnotationResult{}

		artefact, err := s.generateSingleArtefact(
			context.Background(),
			buildResult,
			"test_hash",
			vc,
			annotationResult,
		)

		require.NoError(t, err)
		require.NotNil(t, artefact)
	})
}

func TestBuildWaiter(t *testing.T) {
	t.Run("done channel signals completion", func(t *testing.T) {
		waiter := &buildWaiter{
			done: make(chan struct{}),
		}

		go func() {
			waiter.result = &annotator_dto.ProjectAnnotationResult{}
			close(waiter.done)
		}()

		select {
		case <-waiter.done:
			assert.NotNil(t, waiter.result)
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for completion")
		}
	})

	t.Run("captures error on failure", func(t *testing.T) {
		waiter := &buildWaiter{
			done: make(chan struct{}),
		}

		expectedErr := errors.New("build failed")

		go func() {
			waiter.err = expectedErr
			close(waiter.done)
		}()

		select {
		case <-waiter.done:
			assert.ErrorIs(t, waiter.err, expectedErr)
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for completion")
		}
	})
}

func TestIntrospectionCacheEntry_MatchesScriptHashes(t *testing.T) {
	t.Run("returns true for matching hashes", func(t *testing.T) {
		entry := &IntrospectionCacheEntry{
			ScriptHashes: map[string]string{
				"file1.pk": "hash1",
				"file2.pk": "hash2",
			},
		}

		currentHashes := map[string]string{
			"file1.pk": "hash1",
			"file2.pk": "hash2",
		}

		assert.True(t, entry.MatchesScriptHashes(currentHashes))
	})

	t.Run("returns false when hashes differ", func(t *testing.T) {
		entry := &IntrospectionCacheEntry{
			ScriptHashes: map[string]string{
				"file1.pk": "hash1",
				"file2.pk": "hash2",
			},
		}

		currentHashes := map[string]string{
			"file1.pk": "hash1",
			"file2.pk": "different_hash",
		}

		assert.False(t, entry.MatchesScriptHashes(currentHashes))
	})

	t.Run("returns false when file count differs", func(t *testing.T) {
		entry := &IntrospectionCacheEntry{
			ScriptHashes: map[string]string{
				"file1.pk": "hash1",
				"file2.pk": "hash2",
			},
		}

		currentHashes := map[string]string{
			"file1.pk": "hash1",
		}

		assert.False(t, entry.MatchesScriptHashes(currentHashes))
	})

	t.Run("returns false when file is missing from current", func(t *testing.T) {
		entry := &IntrospectionCacheEntry{
			ScriptHashes: map[string]string{
				"file1.pk": "hash1",
				"file2.pk": "hash2",
			},
		}

		currentHashes := map[string]string{
			"file1.pk": "hash1",
			"file3.pk": "hash3",
		}

		assert.False(t, entry.MatchesScriptHashes(currentHashes))
	})

	t.Run("returns true for both empty", func(t *testing.T) {
		entry := &IntrospectionCacheEntry{
			ScriptHashes: map[string]string{},
		}

		currentHashes := map[string]string{}

		assert.True(t, entry.MatchesScriptHashes(currentHashes))
	})

	t.Run("returns false when entry script hashes are nil", func(t *testing.T) {
		entry := &IntrospectionCacheEntry{
			ScriptHashes: nil,
		}

		assert.False(t, entry.MatchesScriptHashes(nil))
	})
}
