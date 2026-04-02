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

//go:build bench

package coordinator_test_bench

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/coordinator/coordinator_adapters"
	"piko.sh/piko/internal/coordinator/coordinator_domain"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

func newTestCacheService() cache_domain.Service {
	return cache_domain.NewService("")
}

type mockAnnotator struct {
	ResultToReturn *annotator_dto.ProjectAnnotationResult
	ErrorToReturn  error
	BuildDelay     time.Duration
	Calls          atomic.Int64
}

func (m *mockAnnotator) AnnotateProject(ctx context.Context, entryPoints []annotator_dto.EntryPoint, scriptHashes map[string]string, opts ...annotator_domain.AnnotationOption) (*annotator_dto.ProjectAnnotationResult, *annotator_domain.CompilationLogStore, error) {
	m.Calls.Add(1)

	if m.BuildDelay > 0 {
		time.Sleep(m.BuildDelay)
	}
	if ctx.Err() != nil {
		return nil, nil, ctx.Err()
	}
	return m.ResultToReturn, nil, m.ErrorToReturn
}

func (m *mockAnnotator) Annotate(_ context.Context, _ string, _ bool) (*annotator_dto.AnnotationResult, *annotator_domain.CompilationLogStore, error) {
	return nil, nil, errors.New("Annotate not implemented in mock")
}

func (m *mockAnnotator) RunPhase1IntrospectionAndAnnotate(
	_ context.Context,
	_ []annotator_dto.EntryPoint,
	_ map[string]string,
	_ ...annotator_domain.AnnotationOption,
) (*annotator_domain.Phase1Result, error) {
	return nil, errors.New("RunPhase1IntrospectionAndAnnotate not implemented in mock")
}

func (m *mockAnnotator) AnnotateProjectWithCachedIntrospection(
	_ context.Context,
	_ *annotator_dto.ComponentGraph,
	_ *annotator_dto.VirtualModule,
	_ *annotator_domain.TypeResolver,
	_ ...annotator_domain.AnnotationOption,
) (*annotator_dto.ProjectAnnotationResult, *annotator_domain.CompilationLogStore, error) {
	return nil, nil, errors.New("AnnotateProjectWithCachedIntrospection not implemented in mock")
}

type mockCache struct {
	store      map[string]*annotator_dto.ProjectAnnotationResult
	mu         sync.RWMutex
	getCalls   atomic.Int64
	setCalls   atomic.Int64
	clearCalls atomic.Int64
}

func newMockCache() *mockCache {
	return &mockCache{store: make(map[string]*annotator_dto.ProjectAnnotationResult)}
}
func (m *mockCache) Get(ctx context.Context, key string) (*annotator_dto.ProjectAnnotationResult, error) {
	m.getCalls.Add(1)
	m.mu.RLock()
	defer m.mu.RUnlock()
	cached, ok := m.store[key]
	if !ok {
		return nil, coordinator_domain.ErrCacheMiss
	}
	return cached, nil
}
func (m *mockCache) Set(ctx context.Context, key string, result *annotator_dto.ProjectAnnotationResult) error {
	m.setCalls.Add(1)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[key] = result
	return nil
}
func (m *mockCache) Clear(ctx context.Context) error {
	m.clearCalls.Add(1)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store = make(map[string]*annotator_dto.ProjectAnnotationResult)
	return nil
}

type mockFSReader struct {
	Files map[string][]byte
}

func (m *mockFSReader) ReadFile(ctx context.Context, path string) ([]byte, error) {
	content, ok := m.Files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return content, nil
}

type benchmarkHarness struct {
	service     coordinator_domain.CoordinatorService
	annotator   *mockAnnotator
	cache       *mockCache
	fsReader    *mockFSReader
	resolver    *resolver_domain.MockResolver
	entryPoints []annotator_dto.EntryPoint
}

func newBenchmarkHarness(b *testing.B, numFiles int, fileSize int) *benchmarkHarness {
	b.Helper()

	fileContent := make([]byte, fileSize)
	_, err := rand.Read(fileContent)
	if err != nil {
		b.Fatalf("Failed to generate random file content: %v", err)
	}

	fsReader := &mockFSReader{Files: make(map[string][]byte, numFiles)}
	entryPoints := make([]annotator_dto.EntryPoint, numFiles)
	baseDir := "/project"

	for i := range numFiles {

		var path string
		if i%10 == 0 {
			path = fmt.Sprintf("%s/pages/page_%d.pk", baseDir, i)
		} else {
			path = fmt.Sprintf("%s/components/comp_%d.pk", baseDir, i)
		}
		fsReader.Files[path] = fileContent
		entryPoints[i] = annotator_dto.EntryPoint{Path: path, IsPage: true}
	}

	h := &benchmarkHarness{
		annotator: &mockAnnotator{
			BuildDelay:     10 * time.Millisecond,
			ResultToReturn: &annotator_dto.ProjectAnnotationResult{},
		},
		cache:    newMockCache(),
		fsReader: fsReader,
		resolver: &resolver_domain.MockResolver{
			GetBaseDirFunc:    func() string { return baseDir },
			GetModuleNameFunc: func() string { return "benchmark-module" },
			ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				return importPath, nil
			},
			ResolveAssetPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
				return importPath, nil
			},
			ConvertEntryPointPathToManifestKeyFunc: func(entryPointPath string) string {
				return entryPointPath
			},
		},
		entryPoints: entryPoints,
	}

	cacheService := newTestCacheService()
	introspectionCache, err := coordinator_adapters.NewIntrospectionCache(context.Background(), cacheService)
	if err != nil {
		b.Fatalf("failed to create introspection cache: %v", err)
	}
	h.service = coordinator_domain.NewService(
		context.Background(), h.annotator, h.cache, introspectionCache, h.fsReader, h.resolver,
		coordinator_domain.WithDiagnosticOutput(coordinator_adapters.NewSilentDiagnosticOutput()),
	)

	b.Cleanup(func() {
		h.service.Shutdown(context.Background())
	})

	return h
}

func BenchmarkGetOrBuild_CacheMiss(b *testing.B) {
	h := newBenchmarkHarness(b, 100, 4096)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {

		b.StopTimer()
		h.cache.Clear(ctx)
		b.StartTimer()

		_, err := h.service.GetOrBuildProject(ctx, h.entryPoints)
		if err != nil {
			b.Fatalf("GetOrBuildProject failed: %v", err)
		}
	}
}

func BenchmarkGetOrBuild_CacheHit(b *testing.B) {
	h := newBenchmarkHarness(b, 100, 4096)
	ctx := context.Background()

	_, err := h.service.GetOrBuildProject(ctx, h.entryPoints)
	if err != nil {
		b.Fatalf("Initial build to populate cache failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, err := h.service.GetOrBuildProject(ctx, h.entryPoints)
		if err != nil {
			b.Fatalf("GetOrBuildProject failed: %v", err)
		}
	}

	if h.annotator.Calls.Load() != 1 {
		b.Errorf("Expected annotator to be called only once, but got %d", h.annotator.Calls.Load())
	}
}

func BenchmarkGetOrBuild_CacheMiss_Concurrent(b *testing.B) {
	h := newBenchmarkHarness(b, 100, 4096)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := h.service.GetOrBuildProject(ctx, h.entryPoints)
			if err != nil {

				b.Errorf("GetOrBuildProject failed: %v", err)
			}
		}
	})

	if h.annotator.Calls.Load() != 1 {
		b.Errorf("Expected singleflight to allow only 1 build, but %d occurred", h.annotator.Calls.Load())
	}
}

func BenchmarkTimeToReady_AsyncRebuild(b *testing.B) {
	h := newBenchmarkHarness(b, 50, 4096)
	ctx := context.Background()

	introspectionCacheDebounce, err := coordinator_adapters.NewIntrospectionCache(context.Background(), newTestCacheService())
	if err != nil {
		b.Fatalf("failed to create introspection cache for debounce: %v", err)
	}
	serviceWithDebounce := coordinator_domain.NewService(
		context.Background(), h.annotator, h.cache, introspectionCacheDebounce, h.fsReader, h.resolver,
		coordinator_domain.WithDiagnosticOutput(coordinator_adapters.NewSilentDiagnosticOutput()),
		coordinator_domain.WithDebounceDuration(25*time.Millisecond),
	)
	b.Cleanup(func() { serviceWithDebounce.Shutdown(context.Background()) })

	subChan, unsubscribe := serviceWithDebounce.Subscribe("benchmark-subscriber")
	defer unsubscribe()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {

		b.StopTimer()

		h.cache.Clear(ctx)

		select {
		case <-subChan:
		default:
		}

		b.StartTimer()

		serviceWithDebounce.RequestRebuild(ctx, h.entryPoints)

		select {
		case <-subChan:

		case <-time.After(5 * time.Second):
			b.Fatal("Timed out waiting for build notification")
		}
	}
}
