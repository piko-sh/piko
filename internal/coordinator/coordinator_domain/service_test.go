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
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/wdk/safedisk"
)

type mockAnnotator struct {
	ResultToReturn *annotator_dto.ProjectAnnotationResult
	ErrorToReturn  error
	BuildDelay     time.Duration
	Calls          int
	mu             sync.Mutex
}

func (m *mockAnnotator) GetCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Calls
}

func (m *mockAnnotator) AnnotateProject(ctx context.Context, entryPoints []annotator_dto.EntryPoint, scriptHashes map[string]string, _ ...annotator_domain.AnnotationOption) (*annotator_dto.ProjectAnnotationResult, *annotator_domain.CompilationLogStore, error) {
	m.mu.Lock()
	m.Calls++
	m.mu.Unlock()

	if m.BuildDelay > 0 {
		time.Sleep(m.BuildDelay)
	}

	if ctx.Err() != nil {
		return nil, nil, ctx.Err()
	}

	logStore, _ := annotator_domain.NewCompilationLogStore(context.Background(), false, "", slog.LevelDebug)
	return m.ResultToReturn, logStore, m.ErrorToReturn
}

func (m *mockAnnotator) Annotate(_ context.Context, _ string, _ bool) (*annotator_dto.AnnotationResult, *annotator_domain.CompilationLogStore, error) {
	return nil, nil, errors.New("annotate not implemented in mock")
}

func (m *mockAnnotator) RunPhase1IntrospectionAndAnnotate(
	ctx context.Context,
	_ []annotator_dto.EntryPoint,
	_ map[string]string,
	_ ...annotator_domain.AnnotationOption,
) (*annotator_domain.Phase1Result, error) {
	m.mu.Lock()
	m.Calls++
	m.mu.Unlock()

	if m.BuildDelay > 0 {
		select {
		case <-time.After(m.BuildDelay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if m.ErrorToReturn != nil {
		return nil, m.ErrorToReturn
	}

	logStore, _ := annotator_domain.NewCompilationLogStore(context.Background(), false, "", slog.LevelDebug)
	return &annotator_domain.Phase1Result{
		Annotations: m.ResultToReturn,
		Logs:        logStore,
	}, nil
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
	getError   error
	getResult  *annotator_dto.ProjectAnnotationResult
	setArgs    map[string]*annotator_dto.ProjectAnnotationResult
	getCalls   int
	setCalls   int
	clearCalls int
	mu         sync.Mutex
}

type mockIntrospectionCache struct {
	entries    map[string]*IntrospectionCacheEntry
	mu         sync.Mutex
	getCalls   int
	setCalls   int
	clearCalls int
}

func newMockIntrospectionCache() *mockIntrospectionCache {
	return &mockIntrospectionCache{entries: make(map[string]*IntrospectionCacheEntry)}
}

func (m *mockIntrospectionCache) Get(_ context.Context, key string) (*IntrospectionCacheEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getCalls++
	if entry, ok := m.entries[key]; ok {
		return entry, nil
	}
	return nil, ErrCacheMiss
}

func (m *mockIntrospectionCache) Set(_ context.Context, key string, entry *IntrospectionCacheEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setCalls++
	m.entries[key] = entry
	return nil
}

func (m *mockIntrospectionCache) Clear(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clearCalls++
	m.entries = make(map[string]*IntrospectionCacheEntry)
	return nil
}

func newMockCache() *mockCache {
	return &mockCache{setArgs: make(map[string]*annotator_dto.ProjectAnnotationResult)}
}
func (m *mockCache) Get(ctx context.Context, key string) (*annotator_dto.ProjectAnnotationResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getCalls++
	if m.getError != nil {
		return nil, m.getError
	}
	cached, ok := m.setArgs[key]
	if ok {
		return cached, nil
	}
	return m.getResult, ErrCacheMiss
}
func (m *mockCache) Set(ctx context.Context, key string, result *annotator_dto.ProjectAnnotationResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setCalls++
	m.setArgs[key] = result
	return nil
}
func (m *mockCache) Clear(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clearCalls++
	m.setArgs = make(map[string]*annotator_dto.ProjectAnnotationResult)
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

type testHarness struct {
	service            *coordinatorService
	annotator          *mockAnnotator
	cache              *mockCache
	introspectionCache *mockIntrospectionCache
	fsReader           *mockFSReader
	resolver           *resolver_domain.MockResolver
	sandbox            *safedisk.MockSandbox
	entryPoints        []annotator_dto.EntryPoint
}

func newTestHarness(t *testing.T) *testHarness {
	t.Helper()
	fsReader := &mockFSReader{Files: make(map[string][]byte)}
	baseDir := "/project"
	resolver := &resolver_domain.MockResolver{
		GetBaseDirFunc:    func() string { return baseDir },
		GetModuleNameFunc: func() string { return "test-module" },
		ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
			const moduleName = "test-module"
			if after, ok := strings.CutPrefix(importPath, moduleName+"/"); ok {
				relPath := after
				return filepath.Join(baseDir, relPath), nil
			}
			return importPath, nil
		},
		ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
	}
	mockSandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)

	h := &testHarness{
		annotator: &mockAnnotator{
			ResultToReturn: &annotator_dto.ProjectAnnotationResult{},
		},
		cache:              newMockCache(),
		introspectionCache: newMockIntrospectionCache(),
		fsReader:           fsReader,
		resolver:           resolver,
		sandbox:            mockSandbox,
		entryPoints: []annotator_dto.EntryPoint{
			{Path: "test-module/pages/index.pk", IsPage: true},
		},
	}

	h.fsReader.Files["/project/pages/index.pk"] = []byte("content")
	h.sandbox.AddFile("pages/index.pk", []byte("content"))

	service := NewService(
		context.Background(),
		h.annotator,
		h.cache,
		h.introspectionCache,
		h.fsReader,
		h.resolver,
		WithBaseDirSandbox(mockSandbox),
	)
	var ok bool
	h.service, ok = service.(*coordinatorService)
	if !ok {
		t.Fatal("expected NewService to return *coordinatorService")
	}

	return h
}

func TestCoordinatorService(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping coordinator integration tests in short mode (requires real filesystem access)")
	}

	t.Run("A: Synchronous Build Logic", func(t *testing.T) {
		t.Run("A.1: Cold Start (First Ever Build)", func(t *testing.T) {
			h := newTestHarness(t)
			defer h.service.Shutdown(context.Background())

			h.cache.getError = ErrCacheMiss

			result, err := h.service.GetOrBuildProject(context.Background(), h.entryPoints)

			require.NoError(t, err)
			assert.Same(t, h.annotator.ResultToReturn, result, "Should return the result from the annotator")
			assert.Equal(t, 1, h.annotator.GetCallCount(), "Annotator should have been called once")
			assert.Equal(t, 1, h.cache.setCalls, "Cache.Set should have been called once")
			assert.Equal(t, stateReady, h.service.GetStatus().State)
		})
	})

	t.Run("B: Caching Behaviour", func(t *testing.T) {
		t.Run("B.1: Cache Hit", func(t *testing.T) {
			h := newTestHarness(t)
			defer h.service.Shutdown(context.Background())

			inputHash, _, _ := h.service.calculateInputHash(context.Background(), h.entryPoints, &buildOptions{})
			h.cache.setArgs[inputHash] = h.annotator.ResultToReturn

			result, err := h.service.GetOrBuildProject(context.Background(), h.entryPoints)

			require.NoError(t, err)
			assert.Same(t, h.annotator.ResultToReturn, result, "Should return the cached result")
			assert.Equal(t, 0, h.annotator.GetCallCount(), "Annotator should NOT have been called on a cache hit")
			assert.Equal(t, 1, h.cache.getCalls, "Cache.Get should have been called once")
		})

		t.Run("B.2: Stale Cache (Input Hash Changed)", func(t *testing.T) {
			h := newTestHarness(t)
			defer h.service.Shutdown(context.Background())
			h.cache.getError = ErrCacheMiss

			_, err := h.service.GetOrBuildProject(context.Background(), h.entryPoints)
			require.NoError(t, err)

			h.fsReader.Files["/project/components/card.pk"] = []byte("new content")
			h.sandbox.AddFile("components/card.pk", []byte("new content"))
			newEntryPoints := append(h.entryPoints, annotator_dto.EntryPoint{Path: "test-module/components/card.pk"})

			_, err = h.service.GetOrBuildProject(context.Background(), newEntryPoints)
			require.NoError(t, err)

			assert.Equal(t, 2, h.annotator.GetCallCount(), "A new build should be triggered when inputs change")
			assert.Equal(t, 2, h.cache.setCalls, "Cache should be set for each unique build")
		})

		t.Run("B.3: Undo Scenario (Cache Hit on Previous State)", func(t *testing.T) {
			h := newTestHarness(t)
			defer h.service.Shutdown(context.Background())

			contentA := "original content"
			h.fsReader.Files["/project/pages/index.pk"] = []byte(contentA)
			resultA := &annotator_dto.ProjectAnnotationResult{}
			h.annotator.ResultToReturn = resultA

			res1, err1 := h.service.GetOrBuildProject(context.Background(), h.entryPoints)
			require.NoError(t, err1)
			assert.Same(t, resultA, res1)
			assert.Equal(t, 1, h.annotator.GetCallCount(), "Annotator should be called for the first build")
			assert.Equal(t, 1, h.cache.setCalls, "Cache should be set after the first build")

			contentB := "changed content"
			h.fsReader.Files["/project/pages/index.pk"] = []byte(contentB)
			resultB := &annotator_dto.ProjectAnnotationResult{}
			h.annotator.ResultToReturn = resultB

			res2, err2 := h.service.GetOrBuildProject(context.Background(), h.entryPoints)
			require.NoError(t, err2)
			assert.Same(t, resultB, res2)
			assert.Equal(t, 2, h.annotator.GetCallCount(), "Annotator should be called again for the changed content")
			assert.Equal(t, 2, h.cache.setCalls, "Cache should be set again for the new state")

			h.fsReader.Files["/project/pages/index.pk"] = []byte(contentA)
			h.annotator.ResultToReturn = resultA

			res3, err3 := h.service.GetOrBuildProject(context.Background(), h.entryPoints)
			require.NoError(t, err3)
			assert.Same(t, resultA, res3, "Should return the original cached result for State A")
			assert.Equal(t, 2, h.annotator.GetCallCount(), "Annotator should NOT be called on the undo, it should be a cache hit")
			assert.Equal(t, 2, h.cache.setCalls, "Cache.Set should not be called on a cache hit")

			lastBuild, ok := h.service.GetLastSuccessfulBuild()
			require.True(t, ok)
			assert.Same(t, resultA, lastBuild)
		})
	})

	t.Run("C: Concurrency and Stampede Protection", func(t *testing.T) {
		t.Run("C.1: Simultaneous Identical Requests (singleflight)", func(t *testing.T) {
			h := newTestHarness(t)
			defer h.service.Shutdown(context.Background())

			h.cache.getError = ErrCacheMiss
			h.annotator.BuildDelay = 100 * time.Millisecond

			var wg sync.WaitGroup
			numGoroutines := 10
			results := make(chan *annotator_dto.ProjectAnnotationResult, numGoroutines)

			wg.Add(numGoroutines)
			for range numGoroutines {
				go func() {
					defer wg.Done()
					buildResult, err := h.service.GetOrBuildProject(context.Background(), h.entryPoints)
					require.NoError(t, err)
					results <- buildResult
				}()
			}
			wg.Wait()
			close(results)

			assert.Equal(t, 1, h.annotator.GetCallCount(), "Annotator should only be called ONCE for all concurrent requests")
			assert.Equal(t, 1, h.cache.setCalls)

			firstResult := h.annotator.ResultToReturn
			for buildResult := range results {
				assert.Same(t, firstResult, buildResult, "All goroutines should receive the identical result pointer")
			}
		})
	})

	t.Run("D: Asynchronous Rebuild and Debouncing", func(t *testing.T) {
		t.Run("D.1: Single Request (Leading-Edge Trigger)", func(t *testing.T) {
			h := newTestHarness(t)
			defer h.service.Shutdown(context.Background())

			h.service.debounceDuration = 50 * time.Millisecond
			h.annotator.BuildDelay = 20 * time.Millisecond

			h.service.RequestRebuild(context.Background(), h.entryPoints)

			time.Sleep(100 * time.Millisecond)

			assert.Equal(t, 1, h.annotator.GetCallCount(), "An immediate build should be triggered")
		})

		t.Run("D.2: Multiple Requests (Trailing-Edge Trigger)", func(t *testing.T) {
			h := newTestHarness(t)
			defer h.service.Shutdown(context.Background())
			h.service.debounceDuration = 200 * time.Millisecond

			for range 5 {
				h.service.RequestRebuild(context.Background(), h.entryPoints)
				time.Sleep(20 * time.Millisecond)
			}

			time.Sleep(250 * time.Millisecond)

			assert.LessOrEqual(t, h.annotator.GetCallCount(), 1, "Rapid-fire requests should be coalesced, not trigger 5 separate builds")
			assert.GreaterOrEqual(t, h.annotator.GetCallCount(), 1, "At least one build should have occurred (leading-edge)")
		})
	})

	t.Run("E: Unified Build Trigger (Race Condition Fix)", func(t *testing.T) {
		t.Run("E.1: Sync request waits for async-triggered build", func(t *testing.T) {
			h := newTestHarness(t)
			defer h.service.Shutdown(context.Background())

			h.annotator.BuildDelay = 200 * time.Millisecond
			h.cache.getError = ErrCacheMiss
			var wg sync.WaitGroup

			h.service.RequestRebuild(context.Background(), h.entryPoints)

			wg.Go(func() {
				_, err := h.service.GetOrBuildProject(context.Background(), h.entryPoints)
				require.NoError(t, err)
			})

			wg.Wait()

			assert.Equal(t, 1, h.annotator.GetCallCount(), "Only one build should have occurred for both requests")
		})
	})

	t.Run("F: State Management and Error Handling", func(t *testing.T) {
		t.Run("F.1: Status transitions on success", func(t *testing.T) {
			h := newTestHarness(t)
			defer h.service.Shutdown(context.Background())

			assert.Equal(t, stateIdle, h.service.GetStatus().State, "Initial state should be Idle")
			_, ok := h.service.GetLastSuccessfulBuild()
			assert.False(t, ok, "Should be no initial successful build")

			h.service.debounceDuration = 50 * time.Millisecond
			h.annotator.BuildDelay = 100 * time.Millisecond
			h.service.RequestRebuild(context.Background(), h.entryPoints)
			time.Sleep(10 * time.Millisecond)

			assert.Equal(t, stateBuilding, h.service.GetStatus().State, "State should be Building after trigger")

			time.Sleep(200 * time.Millisecond)
			status := h.service.GetStatus()
			assert.Equal(t, stateReady, status.State, "State should be Ready after success")
			assert.NotNil(t, status.Result)

			lastBuild, ok := h.service.GetLastSuccessfulBuild()
			assert.True(t, ok)
			assert.NotNil(t, lastBuild)
		})

		t.Run("F.2: Status transitions on failure", func(t *testing.T) {
			h := newTestHarness(t)
			defer h.service.Shutdown(context.Background())
			buildErr := errors.New("annotator failed")
			h.annotator.ErrorToReturn = buildErr

			_, err := h.service.GetOrBuildProject(context.Background(), h.entryPoints)
			require.Error(t, err)

			status := h.service.GetStatus()
			assert.Equal(t, stateFailed, status.State, "State should be Failed")
			assert.Equal(t, buildErr, status.LastBuildError)
		})
	})

	t.Run("G: Shutdown and Lifecycle", func(t *testing.T) {
		t.Run("G.1: Shutdown cancels pending debounced build", func(t *testing.T) {
			h := newTestHarness(t)
			h.service.debounceDuration = 500 * time.Millisecond

			h.service.lastTriggerTime = time.Now()

			h.service.RequestRebuild(context.Background(), h.entryPoints)
			h.service.Shutdown(context.Background())

			time.Sleep(600 * time.Millisecond)

			assert.Equal(t, 0, h.annotator.GetCallCount(), "The pending build should have been cancelled by shutdown")
		})
	})

	t.Run("H: Input Hashing", func(t *testing.T) {
		t.Run("H.1: Hash is deterministic", func(t *testing.T) {
			h := newTestHarness(t)
			defer h.service.Shutdown(context.Background())

			hash1, _, _ := h.service.calculateInputHash(context.Background(), h.entryPoints, &buildOptions{})
			hash2, _, _ := h.service.calculateInputHash(context.Background(), h.entryPoints, &buildOptions{})
			assert.Equal(t, hash1, hash2)
		})

		t.Run("H.2: Hash changes with file content", func(t *testing.T) {
			h := newTestHarness(t)
			defer h.service.Shutdown(context.Background())

			hash1, _, _ := h.service.calculateInputHash(context.Background(), h.entryPoints, &buildOptions{})
			h.fsReader.Files["/project/pages/index.pk"] = []byte("new different content")
			hash2, _, _ := h.service.calculateInputHash(context.Background(), h.entryPoints, &buildOptions{})
			assert.NotEqual(t, hash1, hash2)
		})

		t.Run("H.3: Hash changes with new file", func(t *testing.T) {
			h := newTestHarness(t)
			defer h.service.Shutdown(context.Background())

			hash1, _, _ := h.service.calculateInputHash(context.Background(), h.entryPoints, &buildOptions{})
			newEntryPoints := append(h.entryPoints, annotator_dto.EntryPoint{Path: "test-module/pages/about.pk"})
			h.fsReader.Files["/project/pages/about.pk"] = []byte("about page")
			h.sandbox.AddFile("pages/about.pk", []byte("about page"))
			hash2, _, _ := h.service.calculateInputHash(context.Background(), newEntryPoints, &buildOptions{})
			assert.NotEqual(t, hash1, hash2)
		})

	})

	t.Run("I: Pub/Sub Notifications", func(t *testing.T) {
		t.Run("I.1: Notification is sent with correct CausationID on build", func(t *testing.T) {
			h := newTestHarness(t)
			defer h.service.Shutdown(context.Background())
			h.cache.getError = ErrCacheMiss

			subChan, unsubscribe := h.service.Subscribe("test-subscriber")
			defer unsubscribe()

			causationID := "file-save-event-123"
			_, err := h.service.GetOrBuildProject(context.Background(), h.entryPoints, WithCausationID(causationID))
			require.NoError(t, err)

			select {
			case notification := <-subChan:
				assert.Equal(t, causationID, notification.CausationID, "Notification should have the correct CausationID")
				assert.Same(t, h.annotator.ResultToReturn, notification.Result)
			case <-time.After(1 * time.Second):
				t.Fatal("timed out waiting for build notification")
			}
		})

		t.Run("I.2: Notification is sent with correct CausationID on cache hit", func(t *testing.T) {
			h := newTestHarness(t)
			defer h.service.Shutdown(context.Background())

			_, err := h.service.GetOrBuildProject(context.Background(), h.entryPoints, WithCausationID("initial-build"))
			require.NoError(t, err)
			assert.Equal(t, 1, h.annotator.GetCallCount())

			subChan, unsubscribe := h.service.Subscribe("test-subscriber")
			defer unsubscribe()

			causationID := "browser-refresh-456"
			_, err = h.service.GetOrBuildProject(context.Background(), h.entryPoints, WithCausationID(causationID))
			require.NoError(t, err)
			assert.Equal(t, 1, h.annotator.GetCallCount(), "Annotator should not be called again on cache hit")

			select {
			case notification := <-subChan:
				assert.Equal(t, causationID, notification.CausationID, "Notification from cache hit should have the new CausationID")
			case <-time.After(1 * time.Second):
				t.Fatal("timed out waiting for build notification")
			}
		})
	})

	t.Run("J: Waiter Cleanup", func(t *testing.T) {
		t.Run("J.1: Context cancellation cleans up waiter from map", func(t *testing.T) {
			h := newTestHarness(t)
			defer h.service.Shutdown(context.Background())

			h.annotator.BuildDelay = 500 * time.Millisecond

			ctx, cancel := context.WithCancelCause(context.Background())

			var wg sync.WaitGroup
			var buildErr error

			wg.Go(func() {
				_, buildErr = h.service.GetOrBuildProject(ctx, h.entryPoints)
			})

			time.Sleep(50 * time.Millisecond)

			cancel(fmt.Errorf("test: simulating cancelled context"))

			wg.Wait()

			assert.ErrorIs(t, buildErr, context.Canceled)

			time.Sleep(10 * time.Millisecond)

			h.annotator.BuildDelay = 0
			result, err := h.service.GetOrBuildProject(context.Background(), h.entryPoints)
			require.NoError(t, err)
			assert.NotNil(t, result)
		})

		t.Run("J.2: Cancelled goroutine does not block subsequent builds", func(t *testing.T) {
			h := newTestHarness(t)
			defer h.service.Shutdown(context.Background())

			h.annotator.BuildDelay = 200 * time.Millisecond
			ctx1, cancel1 := context.WithCancelCause(context.Background())

			var wg sync.WaitGroup

			wg.Go(func() {
				_, _ = h.service.GetOrBuildProject(ctx1, h.entryPoints)
			})

			time.Sleep(50 * time.Millisecond)
			cancel1(fmt.Errorf("test: simulating cancelled context"))
			wg.Wait()

			h.fsReader.Files["/project/pages/index.pk"] = []byte("new content")
			h.annotator.BuildDelay = 0

			result, err := h.service.GetOrBuildProject(context.Background(), h.entryPoints)
			require.NoError(t, err)
			assert.NotNil(t, result)
		})
	})
}
