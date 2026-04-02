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
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/coordinator/coordinator_dto"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/safedisk"
)

func noopSpan() trace.Span {
	_, span := noop.NewTracerProvider().Tracer("test").Start(context.Background(), "test")
	return span
}

func TestWithMaxBuildWaitDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		duration time.Duration
		want     time.Duration
	}{
		{
			name:     "positive duration is applied",
			duration: 60 * time.Second,
			want:     60 * time.Second,
		},
		{
			name:     "zero duration is ignored",
			duration: 0,
			want:     defaultMaxBuildWaitDuration,
		},
		{
			name:     "negative duration is ignored",
			duration: -5 * time.Second,
			want:     defaultMaxBuildWaitDuration,
		},
		{
			name:     "very small positive duration is applied",
			duration: 1 * time.Millisecond,
			want:     1 * time.Millisecond,
		},
		{
			name:     "very large duration is applied",
			duration: 24 * time.Hour,
			want:     24 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts := applyCoordinatorOptions(WithMaxBuildWaitDuration(tt.duration))
			assert.Equal(t, tt.want, opts.maxBuildWaitDuration)
		})
	}
}

func TestWithMaxBuildWaitDuration_PropagatedToService(t *testing.T) {
	t.Parallel()

	service := newCoordinatorService(
		&mockAnnotator{},
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(WithMaxBuildWaitDuration(45*time.Second)),
	)

	assert.Equal(t, 45*time.Second, service.maxBuildWaitDuration)
}

func TestNewService_ReturnsCoordinatorService(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)

	service := NewService(
		context.Background(),
		&mockAnnotator{},
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		WithBaseDirSandbox(sandbox),
		WithDebounceDuration(100*time.Millisecond),
		WithMaxBuildWaitDuration(10*time.Second),
	)

	require.NotNil(t, service)
	concrete, ok := service.(*coordinatorService)
	require.True(t, ok, "NewService should return *coordinatorService")
	assert.Equal(t, 100*time.Millisecond, concrete.debounceDuration)
	assert.Equal(t, 10*time.Second, concrete.maxBuildWaitDuration)

	service.Shutdown(context.Background())
}

func TestNewService_BuildLoopIsRunning(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)

	service := NewService(
		context.Background(),
		&mockAnnotator{},
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		WithBaseDirSandbox(sandbox),
	)

	concrete, ok := service.(*coordinatorService)
	if !ok {
		t.Fatal("expected *coordinatorService")
	}

	concrete.Shutdown(context.Background())
}

func TestNewService_WithAllOptions(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	fhCache := newMockFileHashCache()
	emitter := &mockCodeEmitter{}
	diagOut := &mockDiagnosticOutput{}
	mockClock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	service := NewService(
		context.Background(),
		&mockAnnotator{},
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		WithBaseDirSandbox(sandbox),
		WithFileHashCache(fhCache),
		WithCodeEmitter(emitter),
		WithDiagnosticOutput(diagOut),
		WithDebounceDuration(200*time.Millisecond),
		WithMaxBuildWaitDuration(5*time.Second),
		withClock(mockClock),
	)

	concrete, ok := service.(*coordinatorService)
	if !ok {
		t.Fatal("expected *coordinatorService")
	}
	assert.Same(t, fhCache, concrete.fileHashCache)
	assert.Same(t, emitter, concrete.codeEmitter)
	assert.Same(t, diagOut, concrete.diagnosticOutput)
	assert.Same(t, sandbox, concrete.baseDirSandbox)
	assert.Equal(t, 200*time.Millisecond, concrete.debounceDuration)
	assert.Equal(t, 5*time.Second, concrete.maxBuildWaitDuration)

	service.Shutdown(context.Background())
}

func TestCheckCacheHit_Hit(t *testing.T) {
	t.Parallel()

	expected := &annotator_dto.ProjectAnnotationResult{}
	cache := newMockCache()
	cache.setArgs["test-hash"] = expected

	service := newCoordinatorService(
		&mockAnnotator{},
		cache,
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(),
	)

	buildOpts := &buildOptions{
		InspectionCacheHints: nil,
		CausationID:          "cause-1",
		ChangedFiles:         nil,
		Resolver:             nil,
		SkipInspection:       false,
		FaultTolerant:        false,
	}

	result := service.checkCacheHit(context.Background(), noopSpan(), "test-hash", buildOpts)

	require.NotNil(t, result)
	assert.Same(t, expected, result)
	assert.Equal(t, stateReady, service.GetStatus().State)
}

func TestCheckCacheHit_Miss(t *testing.T) {
	t.Parallel()

	cache := newMockCache()

	service := newCoordinatorService(
		&mockAnnotator{},
		cache,
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(),
	)

	buildOpts := &buildOptions{
		InspectionCacheHints: nil,
		CausationID:          "",
		ChangedFiles:         nil,
		Resolver:             nil,
		SkipInspection:       false,
		FaultTolerant:        false,
	}

	result := service.checkCacheHit(context.Background(), noopSpan(), "nonexistent-hash", buildOpts)

	assert.Nil(t, result)
}

func TestCheckCacheHit_CacheError(t *testing.T) {
	t.Parallel()

	cache := newMockCache()
	cache.getError = errors.New("cache unavailable")

	service := newCoordinatorService(
		&mockAnnotator{},
		cache,
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(),
	)

	buildOpts := &buildOptions{
		InspectionCacheHints: nil,
		CausationID:          "",
		ChangedFiles:         nil,
		Resolver:             nil,
		SkipInspection:       false,
		FaultTolerant:        false,
	}

	result := service.checkCacheHit(context.Background(), noopSpan(), "test-hash", buildOpts)

	assert.Nil(t, result)
}

func TestCheckTier1Cache_CacheMiss(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	sandbox.AddFile("pages/index.pk", []byte(`<script lang="go">
package pages
</script>`))

	fsReader := &mockFSReader{
		Files: map[string][]byte{
			"/project/pages/index.pk": []byte(`<script lang="go">
package pages
</script>`),
		},
	}

	introCache := newMockIntrospectionCache()

	service := newCoordinatorService(
		&mockAnnotator{},
		newMockCache(),
		introCache,
		fsReader,
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(WithBaseDirSandbox(sandbox)),
	)

	request := &coordinator_dto.BuildRequest{
		CausationID:   "cause-1",
		EntryPoints:   []annotator_dto.EntryPoint{{Path: "test-module/pages/index.pk", IsPage: true}},
		Resolver:      nil,
		FaultTolerant: false,
	}

	result := service.checkTier1Cache(context.Background(), noopSpan(), request)

	assert.False(t, result.useFastPath)
	assert.Nil(t, result.entry)
	assert.NotEmpty(t, result.introspectionHash)
	assert.NotNil(t, result.scriptHashes)
}

func TestCheckTier1Cache_CacheHit_MatchingHashes(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	sandbox.AddFile("pages/index.pk", []byte(`<script lang="go">
package pages
</script>`))

	fsReader := &mockFSReader{
		Files: map[string][]byte{
			"/project/pages/index.pk": []byte(`<script lang="go">
package pages
</script>`),
		},
	}

	introCache := newMockIntrospectionCache()

	service := newCoordinatorService(
		&mockAnnotator{},
		newMockCache(),
		introCache,
		fsReader,
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(WithBaseDirSandbox(sandbox)),
	)

	request := &coordinator_dto.BuildRequest{
		CausationID:   "",
		EntryPoints:   []annotator_dto.EntryPoint{{Path: "test-module/pages/index.pk", IsPage: true}},
		Resolver:      nil,
		FaultTolerant: false,
	}

	firstResult := service.checkTier1Cache(context.Background(), noopSpan(), request)
	require.False(t, firstResult.useFastPath)

	entry := &IntrospectionCacheEntry{
		VirtualModule:  &annotator_dto.VirtualModule{},
		TypeResolver:   &annotator_domain.TypeResolver{},
		ComponentGraph: &annotator_dto.ComponentGraph{},
		ScriptHashes:   firstResult.scriptHashes,
		Timestamp:      time.Now(),
		Version:        CurrentIntrospectionCacheVersion,
	}
	err := introCache.Set(context.Background(), firstResult.introspectionHash, entry)
	require.NoError(t, err)

	secondResult := service.checkTier1Cache(context.Background(), noopSpan(), request)

	assert.True(t, secondResult.useFastPath)
	assert.NotNil(t, secondResult.entry)
	assert.Same(t, entry, secondResult.entry)
}

func TestCheckTier1Cache_CacheHit_StaleHashes(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	sandbox.AddFile("pages/index.pk", []byte(`<script lang="go">
package pages
</script>`))

	fsReader := &mockFSReader{
		Files: map[string][]byte{
			"/project/pages/index.pk": []byte(`<script lang="go">
package pages
</script>`),
		},
	}

	introCache := newMockIntrospectionCache()

	service := newCoordinatorService(
		&mockAnnotator{},
		newMockCache(),
		introCache,
		fsReader,
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(WithBaseDirSandbox(sandbox)),
	)

	request := &coordinator_dto.BuildRequest{
		CausationID:   "",
		EntryPoints:   []annotator_dto.EntryPoint{{Path: "test-module/pages/index.pk", IsPage: true}},
		Resolver:      nil,
		FaultTolerant: false,
	}

	firstResult := service.checkTier1Cache(context.Background(), noopSpan(), request)

	entry := &IntrospectionCacheEntry{
		VirtualModule:  &annotator_dto.VirtualModule{},
		TypeResolver:   &annotator_domain.TypeResolver{},
		ComponentGraph: &annotator_dto.ComponentGraph{},
		ScriptHashes:   map[string]string{"totally-wrong-path": "wrong-hash"},
		Timestamp:      time.Now(),
		Version:        CurrentIntrospectionCacheVersion,
	}
	err := introCache.Set(context.Background(), firstResult.introspectionHash, entry)
	require.NoError(t, err)

	secondResult := service.checkTier1Cache(context.Background(), noopSpan(), request)

	assert.False(t, secondResult.useFastPath)
	assert.Nil(t, secondResult.entry)
	assert.NotEmpty(t, secondResult.introspectionHash)
}

func TestCheckTier1Cache_IntrospectionCacheError(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	sandbox.AddFile("pages/index.pk", []byte(`<script lang="go">
package pages
</script>`))

	fsReader := &mockFSReader{
		Files: map[string][]byte{
			"/project/pages/index.pk": []byte(`<script lang="go">
package pages
</script>`),
		},
	}

	failCache := &failingIntrospectionCache{setErr: errors.New("broken")}

	service := newCoordinatorService(
		&mockAnnotator{},
		newMockCache(),
		failCache,
		fsReader,
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(WithBaseDirSandbox(sandbox)),
	)

	request := &coordinator_dto.BuildRequest{
		CausationID:   "",
		EntryPoints:   []annotator_dto.EntryPoint{{Path: "test-module/pages/index.pk", IsPage: true}},
		Resolver:      nil,
		FaultTolerant: false,
	}

	result := service.checkTier1Cache(context.Background(), noopSpan(), request)

	assert.False(t, result.useFastPath)
	assert.Nil(t, result.entry)
}

func TestCalculateHashForBuild_Success(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	sandbox.AddFile("pages/index.pk", []byte("content"))

	fsReader := &mockFSReader{
		Files: map[string][]byte{
			"/project/pages/index.pk": []byte("content"),
		},
	}

	service := newCoordinatorService(
		&mockAnnotator{},
		newMockCache(),
		newMockIntrospectionCache(),
		fsReader,
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(WithBaseDirSandbox(sandbox)),
	)

	entryPoints := []annotator_dto.EntryPoint{
		{Path: "test-module/pages/index.pk", IsPage: true},
	}
	buildOpts := &buildOptions{
		InspectionCacheHints: nil,
		CausationID:          "",
		ChangedFiles:         nil,
		Resolver:             nil,
		SkipInspection:       false,
		FaultTolerant:        false,
	}

	hash, err := service.calculateHashForBuild(context.Background(), noopSpan(), entryPoints, buildOpts)

	require.NoError(t, err)
	assert.NotEmpty(t, hash)
}

func TestCalculateHashForBuild_ContextCancelled(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)

	service := newCoordinatorService(
		&mockAnnotator{},
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(WithBaseDirSandbox(sandbox)),
	)

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	entryPoints := []annotator_dto.EntryPoint{
		{Path: "test-module/pages/missing.pk", IsPage: true},
	}
	buildOpts := &buildOptions{
		InspectionCacheHints: nil,
		CausationID:          "",
		ChangedFiles:         nil,
		Resolver:             nil,
		SkipInspection:       false,
		FaultTolerant:        false,
	}

	_, err := service.calculateHashForBuild(ctx, noopSpan(), entryPoints, buildOpts)

	assert.Error(t, err)
}

func TestCalculateInputHash_Success(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	sandbox.AddFile("pages/index.pk", []byte("content"))

	fsReader := &mockFSReader{
		Files: map[string][]byte{
			"/project/pages/index.pk": []byte("content"),
		},
	}

	service := newCoordinatorService(
		&mockAnnotator{},
		newMockCache(),
		newMockIntrospectionCache(),
		fsReader,
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(WithBaseDirSandbox(sandbox)),
	)

	entryPoints := []annotator_dto.EntryPoint{
		{Path: "test-module/pages/index.pk", IsPage: true},
	}
	buildOpts := &buildOptions{
		InspectionCacheHints: nil,
		CausationID:          "",
		ChangedFiles:         nil,
		Resolver:             nil,
		SkipInspection:       false,
		FaultTolerant:        false,
	}

	hash, contents, err := service.calculateInputHash(context.Background(), entryPoints, buildOpts)

	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEmpty(t, contents)
	assert.Contains(t, contents, "/project/pages/index.pk")
}

func TestCalculateInputHash_Deterministic(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	sandbox.AddFile("pages/index.pk", []byte("deterministic content"))

	fsReader := &mockFSReader{
		Files: map[string][]byte{
			"/project/pages/index.pk": []byte("deterministic content"),
		},
	}

	service := newCoordinatorService(
		&mockAnnotator{},
		newMockCache(),
		newMockIntrospectionCache(),
		fsReader,
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(WithBaseDirSandbox(sandbox)),
	)

	entryPoints := []annotator_dto.EntryPoint{
		{Path: "test-module/pages/index.pk", IsPage: true},
	}
	buildOpts := &buildOptions{
		InspectionCacheHints: nil,
		CausationID:          "",
		ChangedFiles:         nil,
		Resolver:             nil,
		SkipInspection:       false,
		FaultTolerant:        false,
	}

	hash1, _, err1 := service.calculateInputHash(context.Background(), entryPoints, buildOpts)
	hash2, _, err2 := service.calculateInputHash(context.Background(), entryPoints, buildOpts)

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Equal(t, hash1, hash2)
}

func TestCalculateInputHash_ChangedContent(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	sandbox.AddFile("pages/index.pk", []byte("original"))

	fsReader := &mockFSReader{
		Files: map[string][]byte{
			"/project/pages/index.pk": []byte("original"),
		},
	}

	service := newCoordinatorService(
		&mockAnnotator{},
		newMockCache(),
		newMockIntrospectionCache(),
		fsReader,
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(WithBaseDirSandbox(sandbox)),
	)

	entryPoints := []annotator_dto.EntryPoint{
		{Path: "test-module/pages/index.pk", IsPage: true},
	}
	buildOpts := &buildOptions{
		InspectionCacheHints: nil,
		CausationID:          "",
		ChangedFiles:         nil,
		Resolver:             nil,
		SkipInspection:       false,
		FaultTolerant:        false,
	}

	hash1, _, err1 := service.calculateInputHash(context.Background(), entryPoints, buildOpts)
	require.NoError(t, err1)

	fsReader.Files["/project/pages/index.pk"] = []byte("modified content")

	hash2, _, err2 := service.calculateInputHash(context.Background(), entryPoints, buildOpts)
	require.NoError(t, err2)

	assert.NotEqual(t, hash1, hash2)
}

func TestCalculateIntrospectionHash_Success(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	rootDir := sandbox.AddFile(".", nil)
	rootDir.StatInfo = mockDirStatInfo{}
	sandbox.AddFile("pages/index.pk", []byte(`<script lang="go">
package pages
</script>
<template><div>hello</div></template>`))

	fsReader := &mockFSReader{
		Files: map[string][]byte{
			"/project/pages/index.pk": []byte(`<script lang="go">
package pages
</script>
<template><div>hello</div></template>`),
		},
	}

	service := newCoordinatorService(
		&mockAnnotator{},
		newMockCache(),
		newMockIntrospectionCache(),
		fsReader,
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(WithBaseDirSandbox(sandbox)),
	)
	service.baseDirSandboxPath = "/project"

	entryPoints := []annotator_dto.EntryPoint{
		{Path: "test-module/pages/index.pk", IsPage: true},
	}
	buildOpts := &buildOptions{
		InspectionCacheHints: nil,
		CausationID:          "",
		ChangedFiles:         nil,
		Resolver:             nil,
		SkipInspection:       false,
		FaultTolerant:        false,
	}

	hash, scriptHashes, err := service.calculateIntrospectionHash(context.Background(), entryPoints, buildOpts)

	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEmpty(t, scriptHashes)
}

func TestCalculateIntrospectionHash_TemplateChangeDoesNotAffectHash(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	sandbox.AddFile("pages/index.pk", []byte(`<script lang="go">
package pages
</script>
<template><div>version 1</div></template>`))

	fsReader := &mockFSReader{
		Files: map[string][]byte{
			"/project/pages/index.pk": []byte(`<script lang="go">
package pages
</script>
<template><div>version 1</div></template>`),
		},
	}

	service := newCoordinatorService(
		&mockAnnotator{},
		newMockCache(),
		newMockIntrospectionCache(),
		fsReader,
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(WithBaseDirSandbox(sandbox)),
	)

	entryPoints := []annotator_dto.EntryPoint{
		{Path: "test-module/pages/index.pk", IsPage: true},
	}
	buildOpts := &buildOptions{
		InspectionCacheHints: nil,
		CausationID:          "",
		ChangedFiles:         nil,
		Resolver:             nil,
		SkipInspection:       false,
		FaultTolerant:        false,
	}

	hash1, _, err1 := service.calculateIntrospectionHash(context.Background(), entryPoints, buildOpts)
	require.NoError(t, err1)

	fsReader.Files["/project/pages/index.pk"] = []byte(`<script lang="go">
package pages
</script>
<template><div>version 2 changed</div></template>`)

	hash2, _, err2 := service.calculateIntrospectionHash(context.Background(), entryPoints, buildOpts)
	require.NoError(t, err2)

	assert.Equal(t, hash1, hash2, "introspection hash should not change when only template changes")
}

func TestWaitForBuildResult_ContextCancelled(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	sandbox.AddFile("pages/index.pk", []byte("content"))

	service := newCoordinatorService(
		&mockAnnotator{BuildDelay: 5 * time.Second},
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{
			Files: map[string][]byte{
				"/project/pages/index.pk": []byte("content"),
			},
		},
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(
			WithBaseDirSandbox(sandbox),
			WithMaxBuildWaitDuration(10*time.Second),
		),
	)

	service.wg.Add(1)
	go service.buildLoop(context.Background())
	defer service.Shutdown(context.Background())

	ctx, cancel := context.WithCancelCause(context.Background())

	entryPoints := []annotator_dto.EntryPoint{
		{Path: "test-module/pages/index.pk", IsPage: true},
	}

	var wg sync.WaitGroup
	var result *annotator_dto.ProjectAnnotationResult
	var waitErr error

	wg.Go(func() {
		result, waitErr = service.waitForBuildResult(ctx, "test-hash", entryPoints, nil)
	})

	time.Sleep(50 * time.Millisecond)
	cancel(fmt.Errorf("test: simulating cancelled context"))
	wg.Wait()

	assert.Error(t, waitErr)
	assert.ErrorIs(t, waitErr, context.Canceled)
	assert.Nil(t, result)
}

func TestWaitForBuildResult_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow build-timeout test in short mode")
	}
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	sandbox.AddFile("pages/index.pk", []byte("content"))

	service := newCoordinatorService(
		&mockAnnotator{BuildDelay: 2 * time.Second},
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{
			Files: map[string][]byte{
				"/project/pages/index.pk": []byte("content"),
			},
		},
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(
			WithBaseDirSandbox(sandbox),
			WithMaxBuildWaitDuration(100*time.Millisecond),
			withClock(mockClock),
		),
	)

	service.wg.Add(1)
	go service.buildLoop(context.Background())
	defer service.Shutdown(context.Background())

	entryPoints := []annotator_dto.EntryPoint{
		{Path: "test-module/pages/index.pk", IsPage: true},
	}

	var wg sync.WaitGroup
	var waitErr error

	baseline := mockClock.TimerCount()
	wg.Go(func() {
		_, waitErr = service.waitForBuildResult(context.Background(), "timeout-hash", entryPoints, nil)
	})

	mockClock.AwaitTimerSetup(baseline, 2*time.Second)
	mockClock.AwaitTimerSetup(baseline+1, 2*time.Second)
	mockClock.Advance(200 * time.Millisecond)

	wg.Wait()

	assert.Error(t, waitErr)
	assert.ErrorIs(t, waitErr, context.DeadlineExceeded)
}

func TestExecuteBuild_Tier2CacheHit(t *testing.T) {
	t.Parallel()

	expected := &annotator_dto.ProjectAnnotationResult{}
	cache := newMockCache()
	cache.setArgs["input-hash"] = expected

	service := newCoordinatorService(
		&mockAnnotator{},
		cache,
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(),
	)

	request := &coordinator_dto.BuildRequest{
		CausationID:   "test-cause",
		EntryPoints:   nil,
		Resolver:      nil,
		FaultTolerant: false,
	}

	result, err := service.executeBuild(context.Background(), "input-hash", request, nil)

	require.NoError(t, err)
	assert.Same(t, expected, result)
	assert.Equal(t, stateReady, service.GetStatus().State)
}

func TestExecuteBuild_Tier2CacheError(t *testing.T) {
	t.Parallel()

	cache := newMockCache()
	cache.getError = errors.New("cache backend error")

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	sandbox.AddFile("pages/index.pk", []byte(`<script lang="go">
package pages
</script>`))

	fsReader := &mockFSReader{
		Files: map[string][]byte{
			"/project/pages/index.pk": []byte(`<script lang="go">
package pages
</script>`),
		},
	}

	annotator := &mockAnnotator{
		ResultToReturn: &annotator_dto.ProjectAnnotationResult{},
	}

	service := newCoordinatorService(
		annotator,
		cache,
		newMockIntrospectionCache(),
		fsReader,
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(WithBaseDirSandbox(sandbox)),
	)

	request := &coordinator_dto.BuildRequest{
		CausationID:   "test",
		EntryPoints:   []annotator_dto.EntryPoint{{Path: "test-module/pages/index.pk", IsPage: true}},
		Resolver:      nil,
		FaultTolerant: false,
	}

	allSourceContents := map[string][]byte{
		"/project/pages/index.pk": []byte(`<script lang="go">
package pages
</script>`),
	}

	result, err := service.executeBuild(context.Background(), "missing-hash", request, allSourceContents)

	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestExecuteSlowPathBuild_Success(t *testing.T) {
	t.Parallel()

	expected := &annotator_dto.ProjectAnnotationResult{}
	annotator := &mockAnnotator{
		ResultToReturn: expected,
	}

	cache := newMockCache()
	introCache := newMockIntrospectionCache()

	service := newCoordinatorService(
		annotator,
		cache,
		introCache,
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(),
	)

	request := &coordinator_dto.BuildRequest{
		CausationID:   "slow-path-test",
		EntryPoints:   nil,
		Resolver:      nil,
		FaultTolerant: false,
	}

	tier1Result := tier1CacheResult{
		scriptHashes:      nil,
		entry:             nil,
		introspectionHash: "",
		useFastPath:       false,
	}

	allSourceContents := map[string][]byte{
		"/project/pages/index.pk": []byte("content"),
	}

	result, err := service.executeSlowPathBuild(
		context.Background(),
		noopSpan(),
		request,
		allSourceContents,
		"test-input-hash",
		tier1Result, 0,
	)

	require.NoError(t, err)
	assert.Same(t, expected, result)
	assert.Equal(t, stateReady, service.GetStatus().State)
	assert.Equal(t, 1, cache.setCalls, "result should be cached in Tier 2")
}

func TestExecuteSlowPathBuild_AnnotatorError(t *testing.T) {
	t.Parallel()

	annotator := &mockAnnotator{
		ErrorToReturn: errors.New("annotator failed"),
	}

	service := newCoordinatorService(
		annotator,
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(),
	)

	request := &coordinator_dto.BuildRequest{
		CausationID:   "fail-test",
		EntryPoints:   nil,
		Resolver:      nil,
		FaultTolerant: false,
	}

	tier1Result := tier1CacheResult{
		scriptHashes:      nil,
		entry:             nil,
		introspectionHash: "",
		useFastPath:       false,
	}

	result, err := service.executeSlowPathBuild(
		context.Background(),
		noopSpan(),
		request,
		nil,
		"hash",
		tier1Result, 0,
	)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, stateFailed, service.GetStatus().State)
}

func TestExecuteSlowPathBuild_SemanticError(t *testing.T) {
	t.Parallel()

	semErr := annotator_domain.NewSemanticError([]*ast_domain.Diagnostic{
		{Message: "type error", SourcePath: "/test.pk"},
	})
	logStore, _ := annotator_domain.NewCompilationLogStore(context.Background(), false, "", slog.LevelDebug)

	annotator := &mockAnnotatorWithSemanticError{
		partialResult: &annotator_dto.ProjectAnnotationResult{
			ComponentResults: map[string]*annotator_dto.AnnotationResult{},
		},
		semanticErr: semErr,
		logs:        logStore,
	}

	service := newCoordinatorService(
		annotator,
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(),
	)

	request := &coordinator_dto.BuildRequest{
		CausationID:   "sem-err-test",
		EntryPoints:   nil,
		Resolver:      nil,
		FaultTolerant: true,
	}

	tier1Result := tier1CacheResult{
		scriptHashes:      nil,
		entry:             nil,
		introspectionHash: "",
		useFastPath:       false,
	}

	allSourceContents := map[string][]byte{
		"/test.pk": []byte("content"),
	}

	result, err := service.executeSlowPathBuild(
		context.Background(),
		noopSpan(),
		request,
		allSourceContents,
		"hash",
		tier1Result, 0,
	)

	assert.Error(t, err)
	assert.NotNil(t, result, "should return partial result on semantic error")
	assert.Equal(t, stateFailed, service.GetStatus().State)
}

func TestExecuteSlowPathBuild_WithFaultTolerance(t *testing.T) {
	t.Parallel()

	expected := &annotator_dto.ProjectAnnotationResult{}
	annotator := &mockAnnotator{
		ResultToReturn: expected,
	}

	service := newCoordinatorService(
		annotator,
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(),
	)

	request := &coordinator_dto.BuildRequest{
		CausationID:   "ft-test",
		EntryPoints:   nil,
		Resolver:      nil,
		FaultTolerant: true,
	}

	tier1Result := tier1CacheResult{
		scriptHashes:      nil,
		entry:             nil,
		introspectionHash: "",
		useFastPath:       false,
	}

	result, err := service.executeSlowPathBuild(
		context.Background(),
		noopSpan(),
		request,
		nil,
		"hash",
		tier1Result, 0,
	)

	require.NoError(t, err)
	assert.Same(t, expected, result)
}

func TestExecuteSlowPathBuild_WithResolverOverride(t *testing.T) {
	t.Parallel()

	expected := &annotator_dto.ProjectAnnotationResult{}
	annotator := &mockAnnotator{
		ResultToReturn: expected,
	}

	overrideResolver := &resolver_domain.MockResolver{GetBaseDirFunc: func() string { return "/other" }}

	service := newCoordinatorService(
		annotator,
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(),
	)

	request := &coordinator_dto.BuildRequest{
		CausationID:   "resolver-test",
		EntryPoints:   nil,
		Resolver:      overrideResolver,
		FaultTolerant: false,
	}

	tier1Result := tier1CacheResult{
		scriptHashes:      nil,
		entry:             nil,
		introspectionHash: "",
		useFastPath:       false,
	}

	result, err := service.executeSlowPathBuild(
		context.Background(),
		noopSpan(),
		request,
		nil,
		"hash",
		tier1Result, 0,
	)

	require.NoError(t, err)
	assert.Same(t, expected, result)
}

func TestExecuteSlowPathBuild_CachesIntrospectionResults(t *testing.T) {
	t.Parallel()

	expected := &annotator_dto.ProjectAnnotationResult{}
	annotator := &mockAnnotator{
		ResultToReturn: expected,
	}

	introCache := newMockIntrospectionCache()

	service := newCoordinatorService(
		annotator,
		newMockCache(),
		introCache,
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(),
	)

	request := &coordinator_dto.BuildRequest{
		CausationID:   "cache-intro-test",
		EntryPoints:   nil,
		Resolver:      nil,
		FaultTolerant: false,
	}

	tier1Result := tier1CacheResult{
		scriptHashes:      map[string]string{"a.pk": "hash1"},
		entry:             nil,
		introspectionHash: "intro-hash-123",
		useFastPath:       false,
	}

	result, err := service.executeSlowPathBuild(
		context.Background(),
		noopSpan(),
		request,
		nil,
		"input-hash",
		tier1Result, 0,
	)

	require.NoError(t, err)
	assert.Same(t, expected, result)

}

func TestExecutePartialBuild_Success(t *testing.T) {
	t.Parallel()

	expected := &annotator_dto.ProjectAnnotationResult{}
	cache := newMockCache()

	annotator := &mockAnnotatorWithCachedIntrospection{
		resultToReturn: expected,
	}

	service := newCoordinatorService(
		annotator,
		cache,
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(),
	)

	entry := &IntrospectionCacheEntry{
		VirtualModule:  &annotator_dto.VirtualModule{},
		TypeResolver:   &annotator_domain.TypeResolver{},
		ComponentGraph: &annotator_dto.ComponentGraph{},
		ScriptHashes:   map[string]string{},
		Timestamp:      time.Now(),
		Version:        CurrentIntrospectionCacheVersion,
	}

	request := &coordinator_dto.BuildRequest{
		CausationID:   "partial-build",
		EntryPoints:   nil,
		Resolver:      nil,
		FaultTolerant: false,
	}

	allSourceContents := map[string][]byte{
		"/project/pages/index.pk": []byte("content"),
	}

	result, err := service.executePartialBuild(
		context.Background(),
		entry,
		request,
		allSourceContents,
		"full-hash-123", 0,
	)

	require.NoError(t, err)
	assert.Same(t, expected, result)
	assert.Equal(t, stateReady, service.GetStatus().State)
	assert.Equal(t, 1, cache.setCalls, "result should be cached")
}

func TestExecutePartialBuild_AnnotatorError(t *testing.T) {
	t.Parallel()

	annotator := &mockAnnotatorWithCachedIntrospection{
		errorToReturn: errors.New("partial build failed"),
	}

	service := newCoordinatorService(
		annotator,
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(),
	)

	entry := &IntrospectionCacheEntry{
		VirtualModule:  &annotator_dto.VirtualModule{},
		TypeResolver:   &annotator_domain.TypeResolver{},
		ComponentGraph: &annotator_dto.ComponentGraph{},
		ScriptHashes:   map[string]string{},
		Timestamp:      time.Now(),
		Version:        CurrentIntrospectionCacheVersion,
	}

	request := &coordinator_dto.BuildRequest{
		CausationID:   "fail",
		EntryPoints:   nil,
		Resolver:      nil,
		FaultTolerant: false,
	}

	result, err := service.executePartialBuild(
		context.Background(),
		entry,
		request,
		nil,
		"hash", 0,
	)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, stateFailed, service.GetStatus().State)
}

func TestExecutePartialBuild_SemanticError(t *testing.T) {
	t.Parallel()

	semErr := annotator_domain.NewSemanticError([]*ast_domain.Diagnostic{
		{Message: "type mismatch", SourcePath: "/comp.pk"},
	})
	logStore, _ := annotator_domain.NewCompilationLogStore(context.Background(), false, "", slog.LevelDebug)

	partialResult := &annotator_dto.ProjectAnnotationResult{
		ComponentResults: map[string]*annotator_dto.AnnotationResult{},
	}

	annotator := &mockAnnotatorWithCachedIntrospection{
		resultToReturn: partialResult,
		errorToReturn:  semErr,
		logsToReturn:   logStore,
	}

	service := newCoordinatorService(
		annotator,
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(),
	)

	entry := &IntrospectionCacheEntry{
		VirtualModule:  &annotator_dto.VirtualModule{},
		TypeResolver:   &annotator_domain.TypeResolver{},
		ComponentGraph: &annotator_dto.ComponentGraph{},
		ScriptHashes:   map[string]string{},
		Timestamp:      time.Now(),
		Version:        CurrentIntrospectionCacheVersion,
	}

	request := &coordinator_dto.BuildRequest{
		CausationID:   "sem-err",
		EntryPoints:   nil,
		Resolver:      nil,
		FaultTolerant: true,
	}

	result, err := service.executePartialBuild(
		context.Background(),
		entry,
		request,
		map[string][]byte{"/comp.pk": []byte("content")},
		"hash", 0,
	)

	assert.Error(t, err)
	assert.NotNil(t, result, "should return partial result on semantic error")
}

func TestRequestRebuild_ImmediateTrigger(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	service := newCoordinatorService(
		&mockAnnotator{},
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(
			WithDebounceDuration(100*time.Millisecond),
			withClock(mockClock),
		),
	)

	service.lastTriggerTime = mockClock.Now().Add(-200 * time.Millisecond)

	entryPoints := []annotator_dto.EntryPoint{
		{Path: "test-module/pages/index.pk", IsPage: true},
	}

	service.RequestRebuild(context.Background(), entryPoints, WithCausationID("immediate"))

	select {
	case request := <-service.rebuildTrigger:
		require.NotNil(t, request)
		assert.Equal(t, "immediate", request.CausationID)
	case <-time.After(time.Second):
		t.Fatal("expected build request in channel")
	}
}

func TestRequestRebuild_DebouncedTrigger(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	service := newCoordinatorService(
		&mockAnnotator{},
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(
			WithDebounceDuration(500*time.Millisecond),
			withClock(mockClock),
		),
	)

	service.lastTriggerTime = mockClock.Now().Add(-100 * time.Millisecond)

	entryPoints := []annotator_dto.EntryPoint{
		{Path: "test-module/pages/index.pk", IsPage: true},
	}

	baseline := mockClock.TimerCount()
	service.RequestRebuild(context.Background(), entryPoints, WithCausationID("debounced"))

	mockClock.AwaitTimerSetup(baseline, 2*time.Second)

	assert.Equal(t, stateBuilding, service.GetStatus().State)

	mockClock.Advance(600 * time.Millisecond)

	select {
	case request := <-service.rebuildTrigger:
		require.NotNil(t, request)
	case <-time.After(2 * time.Second):
		t.Fatal("expected build request after debounce")
	}
}

func TestRequestRebuild_ReplacesDebounceTimer(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	service := newCoordinatorService(
		&mockAnnotator{},
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(
			WithDebounceDuration(500*time.Millisecond),
			withClock(mockClock),
		),
	)

	entryPoints := []annotator_dto.EntryPoint{
		{Path: "test-module/pages/index.pk", IsPage: true},
	}

	service.lastTriggerTime = mockClock.Now().Add(-100 * time.Millisecond)

	baseline := mockClock.TimerCount()

	service.RequestRebuild(context.Background(), entryPoints, WithCausationID("first"))
	mockClock.AwaitTimerSetup(baseline, 2*time.Second)

	baseline = mockClock.TimerCount()

	service.RequestRebuild(context.Background(), entryPoints, WithCausationID("second"))
	mockClock.AwaitTimerSetup(baseline, 2*time.Second)

	service.mu.RLock()
	request := service.lastBuildRequest
	service.mu.RUnlock()
	assert.Equal(t, "second", request.CausationID)
}

func TestRequestRebuild_WithFaultTolerance(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	service := newCoordinatorService(
		&mockAnnotator{},
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(
			WithDebounceDuration(100*time.Millisecond),
			withClock(mockClock),
		),
	)

	service.lastTriggerTime = mockClock.Now().Add(-200 * time.Millisecond)

	service.RequestRebuild(
		context.Background(),
		[]annotator_dto.EntryPoint{{Path: "test-module/pages/index.pk", IsPage: true}},
		WithFaultTolerance(),
		WithCausationID("ft-rebuild"),
	)

	service.mu.RLock()
	request := service.lastBuildRequest
	service.mu.RUnlock()
	assert.True(t, request.FaultTolerant)
	assert.Equal(t, "ft-rebuild", request.CausationID)
}

func TestBuildLoop_NilRequest(t *testing.T) {
	t.Parallel()

	service := newCoordinatorService(
		&mockAnnotator{},
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(),
	)

	service.wg.Add(1)
	go service.buildLoop(context.Background())

	service.rebuildTrigger <- nil

	time.Sleep(50 * time.Millisecond)

	close(service.shutdown)
	service.wg.Wait()
}

func TestBuildLoop_ProcessesBuildRequest(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	sandbox.AddFile("pages/index.pk", []byte("content"))

	expected := &annotator_dto.ProjectAnnotationResult{}
	annotator := &mockAnnotator{ResultToReturn: expected}

	service := newCoordinatorService(
		annotator,
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{
			Files: map[string][]byte{
				"/project/pages/index.pk": []byte("content"),
			},
		},
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(WithBaseDirSandbox(sandbox)),
	)

	service.wg.Add(1)
	go service.buildLoop(context.Background())

	request := &coordinator_dto.BuildRequest{
		EntryPoints: []annotator_dto.EntryPoint{
			{Path: "test-module/pages/index.pk", IsPage: true},
		},
		CausationID:   "loop-test",
		Resolver:      nil,
		FaultTolerant: false,
	}

	service.rebuildTrigger <- request

	time.Sleep(200 * time.Millisecond)

	assert.True(t, annotator.GetCallCount() >= 1)

	close(service.shutdown)
	service.wg.Wait()
}

func TestBuildLoop_BuildError(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	sandbox.AddFile("pages/index.pk", []byte("content"))

	annotator := &mockAnnotator{
		ErrorToReturn: errors.New("build failed"),
	}

	service := newCoordinatorService(
		annotator,
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{
			Files: map[string][]byte{
				"/project/pages/index.pk": []byte("content"),
			},
		},
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(WithBaseDirSandbox(sandbox)),
	)

	service.wg.Add(1)
	go service.buildLoop(context.Background())

	request := &coordinator_dto.BuildRequest{
		EntryPoints: []annotator_dto.EntryPoint{
			{Path: "test-module/pages/index.pk", IsPage: true},
		},
		CausationID:   "err-test",
		Resolver:      nil,
		FaultTolerant: false,
	}

	service.rebuildTrigger <- request

	time.Sleep(200 * time.Millisecond)

	close(service.shutdown)
	service.wg.Wait()

}

func TestGetOrBuildProject_CacheHitPath(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	sandbox.AddFile("pages/index.pk", []byte("content"))

	expected := &annotator_dto.ProjectAnnotationResult{}
	cache := newMockCache()

	fsReader := &mockFSReader{
		Files: map[string][]byte{
			"/project/pages/index.pk": []byte("content"),
		},
	}

	service := newCoordinatorService(
		&mockAnnotator{},
		cache,
		newMockIntrospectionCache(),
		fsReader,
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(WithBaseDirSandbox(sandbox)),
	)

	entryPoints := []annotator_dto.EntryPoint{
		{Path: "test-module/pages/index.pk", IsPage: true},
	}
	buildOpts := &buildOptions{
		InspectionCacheHints: nil,
		CausationID:          "",
		ChangedFiles:         nil,
		Resolver:             nil,
		SkipInspection:       false,
		FaultTolerant:        false,
	}
	hash, _, err := service.calculateInputHash(context.Background(), entryPoints, buildOpts)
	require.NoError(t, err)

	cache.setArgs[hash] = expected

	service.wg.Add(1)
	go service.buildLoop(context.Background())
	defer service.Shutdown(context.Background())

	result, err := service.GetOrBuildProject(context.Background(), entryPoints)

	require.NoError(t, err)
	assert.Same(t, expected, result)
	assert.Equal(t, stateReady, service.GetStatus().State)
}

func TestGetResult_SlowPath_FallsBackToGetOrBuildProject(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	sandbox.AddFile("pages/index.pk", []byte("content"))

	expected := &annotator_dto.ProjectAnnotationResult{}
	annotator := &mockAnnotator{ResultToReturn: expected}
	cache := newMockCache()

	fsReader := &mockFSReader{
		Files: map[string][]byte{
			"/project/pages/index.pk": []byte("content"),
		},
	}

	service := newCoordinatorService(
		annotator,
		cache,
		newMockIntrospectionCache(),
		fsReader,
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(WithBaseDirSandbox(sandbox)),
	)

	service.wg.Add(1)
	go service.buildLoop(context.Background())
	defer service.Shutdown(context.Background())

	entryPoints := []annotator_dto.EntryPoint{
		{Path: "test-module/pages/index.pk", IsPage: true},
	}

	result, err := service.GetResult(context.Background(), entryPoints)

	require.NoError(t, err)
	assert.Same(t, expected, result)
}

type mockAnnotatorWithSemanticError struct {
	partialResult *annotator_dto.ProjectAnnotationResult
	semanticErr   *annotator_domain.SemanticError
	logs          *annotator_domain.CompilationLogStore
}

func (m *mockAnnotatorWithSemanticError) AnnotateProject(
	_ context.Context,
	_ []annotator_dto.EntryPoint,
	_ map[string]string,
	_ ...annotator_domain.AnnotationOption,
) (*annotator_dto.ProjectAnnotationResult, *annotator_domain.CompilationLogStore, error) {
	return m.partialResult, m.logs, m.semanticErr
}

func (m *mockAnnotatorWithSemanticError) Annotate(
	_ context.Context,
	_ string,
	_ bool,
) (*annotator_dto.AnnotationResult, *annotator_domain.CompilationLogStore, error) {
	return nil, nil, errors.New("not implemented")
}

func (m *mockAnnotatorWithSemanticError) RunPhase1IntrospectionAndAnnotate(
	_ context.Context,
	_ []annotator_dto.EntryPoint,
	_ map[string]string,
	_ ...annotator_domain.AnnotationOption,
) (*annotator_domain.Phase1Result, error) {
	if m.semanticErr != nil {
		return &annotator_domain.Phase1Result{
			Annotations: m.partialResult,
			Logs:        m.logs,
		}, m.semanticErr
	}
	return &annotator_domain.Phase1Result{
		Annotations: m.partialResult,
		Logs:        m.logs,
	}, nil
}

func (m *mockAnnotatorWithSemanticError) AnnotateProjectWithCachedIntrospection(
	_ context.Context,
	_ *annotator_dto.ComponentGraph,
	_ *annotator_dto.VirtualModule,
	_ *annotator_domain.TypeResolver,
	_ ...annotator_domain.AnnotationOption,
) (*annotator_dto.ProjectAnnotationResult, *annotator_domain.CompilationLogStore, error) {
	return nil, nil, errors.New("not implemented")
}

type mockAnnotatorWithCachedIntrospection struct {
	resultToReturn *annotator_dto.ProjectAnnotationResult
	errorToReturn  error
	logsToReturn   *annotator_domain.CompilationLogStore
}

func (m *mockAnnotatorWithCachedIntrospection) AnnotateProject(
	_ context.Context,
	_ []annotator_dto.EntryPoint,
	_ map[string]string,
	_ ...annotator_domain.AnnotationOption,
) (*annotator_dto.ProjectAnnotationResult, *annotator_domain.CompilationLogStore, error) {
	return nil, nil, errors.New("not implemented")
}

func (m *mockAnnotatorWithCachedIntrospection) Annotate(
	_ context.Context,
	_ string,
	_ bool,
) (*annotator_dto.AnnotationResult, *annotator_domain.CompilationLogStore, error) {
	return nil, nil, errors.New("not implemented")
}

func (m *mockAnnotatorWithCachedIntrospection) RunPhase1IntrospectionAndAnnotate(
	_ context.Context,
	_ []annotator_dto.EntryPoint,
	_ map[string]string,
	_ ...annotator_domain.AnnotationOption,
) (*annotator_domain.Phase1Result, error) {
	return nil, errors.New("not implemented")
}

func (m *mockAnnotatorWithCachedIntrospection) AnnotateProjectWithCachedIntrospection(
	_ context.Context,
	_ *annotator_dto.ComponentGraph,
	_ *annotator_dto.VirtualModule,
	_ *annotator_domain.TypeResolver,
	_ ...annotator_domain.AnnotationOption,
) (*annotator_dto.ProjectAnnotationResult, *annotator_domain.CompilationLogStore, error) {
	logs := m.logsToReturn
	if logs == nil {
		logs, _ = annotator_domain.NewCompilationLogStore(context.Background(), false, "", slog.LevelDebug)
	}
	return m.resultToReturn, logs, m.errorToReturn
}

func TestExecuteSlowPathBuild_WithCodeEmitter(t *testing.T) {
	t.Parallel()

	expected := &annotator_dto.ProjectAnnotationResult{
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

	emitter := &mockCodeEmitter{
		Result: []byte("generated code"),
	}

	annotator := &mockAnnotator{
		ResultToReturn: expected,
	}

	service := newCoordinatorService(
		annotator,
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(WithCodeEmitter(emitter)),
	)

	request := &coordinator_dto.BuildRequest{
		CausationID:   "emitter-test",
		EntryPoints:   nil,
		Resolver:      nil,
		FaultTolerant: false,
	}

	tier1Result := tier1CacheResult{
		scriptHashes:      nil,
		entry:             nil,
		introspectionHash: "",
		useFastPath:       false,
	}

	result, err := service.executeSlowPathBuild(
		context.Background(),
		noopSpan(),
		request,
		nil,
		"hash",
		tier1Result, 0,
	)

	require.NoError(t, err)
	assert.Same(t, expected, result)
	assert.Equal(t, 1, emitter.Calls)
	assert.NotNil(t, result.FinalGeneratedArtefacts)
}

func TestExecutePartialBuild_WithCodeEmitter(t *testing.T) {
	t.Parallel()

	expected := &annotator_dto.ProjectAnnotationResult{
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

	emitter := &mockCodeEmitter{
		Result: []byte("fast path code"),
	}

	annotator := &mockAnnotatorWithCachedIntrospection{
		resultToReturn: expected,
	}

	service := newCoordinatorService(
		annotator,
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(WithCodeEmitter(emitter)),
	)

	entry := &IntrospectionCacheEntry{
		VirtualModule:  &annotator_dto.VirtualModule{},
		TypeResolver:   &annotator_domain.TypeResolver{},
		ComponentGraph: &annotator_dto.ComponentGraph{},
		ScriptHashes:   map[string]string{},
		Timestamp:      time.Now(),
		Version:        CurrentIntrospectionCacheVersion,
	}

	request := &coordinator_dto.BuildRequest{
		CausationID:   "emitter-fast",
		EntryPoints:   nil,
		Resolver:      nil,
		FaultTolerant: false,
	}

	result, err := service.executePartialBuild(
		context.Background(),
		entry,
		request,
		nil,
		"hash", 0,
	)

	require.NoError(t, err)
	assert.Same(t, expected, result)
	assert.Equal(t, 1, emitter.Calls)
	assert.NotNil(t, result.FinalGeneratedArtefacts)
}

func TestExecuteBuild_SingleflightWithErrorAndPartialResult(t *testing.T) {
	t.Parallel()

	semErr := annotator_domain.NewSemanticError([]*ast_domain.Diagnostic{
		{Message: "type error", SourcePath: "/test.pk"},
	})
	logStore, _ := annotator_domain.NewCompilationLogStore(context.Background(), false, "", slog.LevelDebug)

	partialResult := &annotator_dto.ProjectAnnotationResult{
		ComponentResults: map[string]*annotator_dto.AnnotationResult{},
	}

	annotator := &mockAnnotatorWithSemanticError{
		partialResult: partialResult,
		semanticErr:   semErr,
		logs:          logStore,
	}

	cache := newMockCache()
	cache.getError = ErrCacheMiss

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	sandbox.AddFile("pages/index.pk", []byte(`<script lang="go">
package pages
</script>`))

	service := newCoordinatorService(
		annotator,
		cache,
		newMockIntrospectionCache(),
		&mockFSReader{
			Files: map[string][]byte{
				"/project/pages/index.pk": []byte(`<script lang="go">
package pages
</script>`),
			},
		},
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(WithBaseDirSandbox(sandbox)),
	)

	request := &coordinator_dto.BuildRequest{
		EntryPoints: []annotator_dto.EntryPoint{
			{Path: "test-module/pages/index.pk", IsPage: true},
		},
		CausationID:   "sf-err",
		Resolver:      nil,
		FaultTolerant: true,
	}

	allSourceContents := map[string][]byte{
		"/test.pk": []byte("content"),
	}

	result, err := service.executeBuild(context.Background(), "sf-hash", request, allSourceContents)

	assert.Error(t, err)
	assert.NotNil(t, result)
}

func TestExecutePartialBuild_WithFaultToleranceAndResolver(t *testing.T) {
	t.Parallel()

	expected := &annotator_dto.ProjectAnnotationResult{}
	overrideResolver := &resolver_domain.MockResolver{GetBaseDirFunc: func() string { return "/other" }}

	annotator := &mockAnnotatorWithCachedIntrospection{
		resultToReturn: expected,
	}

	service := newCoordinatorService(
		annotator,
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(),
	)

	entry := &IntrospectionCacheEntry{
		VirtualModule:  &annotator_dto.VirtualModule{},
		TypeResolver:   &annotator_domain.TypeResolver{},
		ComponentGraph: &annotator_dto.ComponentGraph{},
		ScriptHashes:   map[string]string{},
		Timestamp:      time.Now(),
		Version:        CurrentIntrospectionCacheVersion,
	}

	request := &coordinator_dto.BuildRequest{
		CausationID:   "ft-resolver",
		EntryPoints:   nil,
		Resolver:      overrideResolver,
		FaultTolerant: true,
	}

	result, err := service.executePartialBuild(
		context.Background(),
		entry,
		request,
		nil,
		"hash", 0,
	)

	require.NoError(t, err)
	assert.Same(t, expected, result)
}

func TestBuildLoop_HashCalculationError(t *testing.T) {
	t.Parallel()

	service := newCoordinatorService(
		&mockAnnotator{},
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(),
	)

	service.wg.Add(1)
	go service.buildLoop(context.Background())

	request := &coordinator_dto.BuildRequest{
		EntryPoints: []annotator_dto.EntryPoint{
			{Path: "test-module/nonexistent.pk", IsPage: true},
		},
		CausationID:   "hash-fail",
		Resolver:      nil,
		FaultTolerant: false,
	}

	service.rebuildTrigger <- request

	time.Sleep(100 * time.Millisecond)

	close(service.shutdown)
	service.wg.Wait()
}

func TestBuildLoop_NotifiesWaiters(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
	sandbox.AddFile("pages/index.pk", []byte("content"))

	expected := &annotator_dto.ProjectAnnotationResult{}
	annotator := &mockAnnotator{ResultToReturn: expected}

	fsReader := &mockFSReader{
		Files: map[string][]byte{
			"/project/pages/index.pk": []byte("content"),
		},
	}

	service := newCoordinatorService(
		annotator,
		newMockCache(),
		newMockIntrospectionCache(),
		fsReader,
		&resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return "/project" },
			GetModuleNameFunc: func() string { return "test-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				const moduleName = "test-module"
				if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
					relPath := after
					return filepath.Join("/project", relPath), nil
				}
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(WithBaseDirSandbox(sandbox)),
	)

	service.wg.Add(1)
	go service.buildLoop(context.Background())

	entryPoints := []annotator_dto.EntryPoint{
		{Path: "test-module/pages/index.pk", IsPage: true},
	}
	buildOpts := &buildOptions{
		InspectionCacheHints: nil,
		CausationID:          "",
		ChangedFiles:         nil,
		Resolver:             nil,
		SkipInspection:       false,
		FaultTolerant:        false,
	}
	inputHash, _, err := service.calculateInputHash(context.Background(), entryPoints, buildOpts)
	require.NoError(t, err)

	waiter := &buildWaiter{result: nil, err: nil, done: make(chan struct{})}
	service.waiters.Store(inputHash, waiter)

	request := &coordinator_dto.BuildRequest{
		EntryPoints:   entryPoints,
		CausationID:   "notify-test",
		Resolver:      nil,
		FaultTolerant: false,
	}

	service.rebuildTrigger <- request

	select {
	case <-waiter.done:
		assert.NotNil(t, waiter.result)
		assert.NoError(t, waiter.err)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for build notification")
	}

	close(service.shutdown)
	service.wg.Wait()
}

type mockDirStatInfo struct{}

func (mockDirStatInfo) Name() string       { return "." }
func (mockDirStatInfo) Size() int64        { return 0 }
func (mockDirStatInfo) Mode() fs.FileMode  { return fs.ModeDir | 0o755 }
func (mockDirStatInfo) ModTime() time.Time { return time.Time{} }
func (mockDirStatInfo) IsDir() bool        { return true }
func (mockDirStatInfo) Sys() any           { return nil }
