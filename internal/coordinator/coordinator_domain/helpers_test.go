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
	"io/fs"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/coordinator/coordinator_dto"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/safedisk"
)

func TestSentinelErrors(t *testing.T) {
	t.Parallel()

	t.Run("ErrCacheMiss is a non-nil sentinel", func(t *testing.T) {
		t.Parallel()
		require.NotNil(t, ErrCacheMiss)
		assert.Contains(t, ErrCacheMiss.Error(), "not found")
	})

	t.Run("ErrInvalidCacheEntry is a non-nil sentinel", func(t *testing.T) {
		t.Parallel()
		require.NotNil(t, ErrInvalidCacheEntry)
		assert.Contains(t, ErrInvalidCacheEntry.Error(), "invalid cache entry")
	})

	t.Run("errBuildInProgress is a non-nil sentinel", func(t *testing.T) {
		t.Parallel()
		require.NotNil(t, errBuildInProgress)
		assert.Contains(t, errBuildInProgress.Error(), "build is in progress")
	})

	t.Run("errNoBuildAvailable is a non-nil sentinel", func(t *testing.T) {
		t.Parallel()
		require.NotNil(t, errNoBuildAvailable)
		assert.Contains(t, errNoBuildAvailable.Error(), "no build result")
	})

	t.Run("sentinel errors are distinct", func(t *testing.T) {
		t.Parallel()
		assert.NotEqual(t, ErrCacheMiss, ErrInvalidCacheEntry)
		assert.NotEqual(t, errBuildInProgress, errNoBuildAvailable)
		assert.False(t, errors.Is(ErrCacheMiss, ErrInvalidCacheEntry))
	})
}

func TestConstants(t *testing.T) {
	t.Parallel()

	t.Run("defaultDebounceDuration is 750ms", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 750*time.Millisecond, defaultDebounceDuration)
	})

	t.Run("defaultCacheWriteTimeout is 10 seconds", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 10*time.Second, defaultCacheWriteTimeout)
	})

	t.Run("defaultMaxBuildWaitDuration is 30 seconds", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 30*time.Second, defaultMaxBuildWaitDuration)
	})

	t.Run("hashingFileReadWorkers is 16", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 16, hashingFileReadWorkers)
	})

	t.Run("logKeyHashedName value", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "hashed_name", logKeyHashedName)
	})
}

func TestWithFileHashCache(t *testing.T) {
	t.Parallel()

	t.Run("sets file hash cache", func(t *testing.T) {
		t.Parallel()
		cache := newMockFileHashCache()
		opts := applyCoordinatorOptions(WithFileHashCache(cache))
		assert.Same(t, cache, opts.fileHashCache)
	})

	t.Run("nil cache is accepted", func(t *testing.T) {
		t.Parallel()
		opts := applyCoordinatorOptions(WithFileHashCache(nil))
		assert.Nil(t, opts.fileHashCache)
	})
}

func TestWithCodeEmitter(t *testing.T) {
	t.Parallel()

	t.Run("sets code emitter", func(t *testing.T) {
		t.Parallel()
		emitter := &mockCodeEmitter{}
		opts := applyCoordinatorOptions(WithCodeEmitter(emitter))
		assert.Same(t, emitter, opts.codeEmitter)
	})

	t.Run("nil emitter is accepted", func(t *testing.T) {
		t.Parallel()
		opts := applyCoordinatorOptions(WithCodeEmitter(nil))
		assert.Nil(t, opts.codeEmitter)
	})
}

func TestWithDiagnosticOutput(t *testing.T) {
	t.Parallel()

	t.Run("sets diagnostic output", func(t *testing.T) {
		t.Parallel()
		output := &mockDiagnosticOutput{}
		opts := applyCoordinatorOptions(WithDiagnosticOutput(output))
		assert.Same(t, output, opts.diagnosticOutput)
	})

	t.Run("nil output is accepted", func(t *testing.T) {
		t.Parallel()
		opts := applyCoordinatorOptions(WithDiagnosticOutput(nil))
		assert.Nil(t, opts.diagnosticOutput)
	})
}

func TestWithClock(t *testing.T) {
	t.Parallel()

	t.Run("sets custom clock", func(t *testing.T) {
		t.Parallel()
		c := clock.RealClock()
		opts := applyCoordinatorOptions(withClock(c))
		assert.Equal(t, c, opts.clock)
	})

	t.Run("nil clock gets replaced by default", func(t *testing.T) {
		t.Parallel()
		opts := applyCoordinatorOptions(withClock(nil))

		assert.NotNil(t, opts.clock)
	})
}

func TestWithBaseDirSandbox(t *testing.T) {
	t.Parallel()

	t.Run("sets sandbox", func(t *testing.T) {
		t.Parallel()
		builder := safedisk.NewMockSandbox("/tmp", safedisk.ModeReadOnly)
		opts := applyCoordinatorOptions(WithBaseDirSandbox(builder))
		assert.Same(t, builder, opts.baseDirSandbox)
	})

	t.Run("nil sandbox is accepted", func(t *testing.T) {
		t.Parallel()
		opts := applyCoordinatorOptions(WithBaseDirSandbox(nil))
		assert.Nil(t, opts.baseDirSandbox)
	})
}

func TestApplyCoordinatorOptions_Multiple(t *testing.T) {
	t.Parallel()

	cache := newMockFileHashCache()
	emitter := &mockCodeEmitter{}
	diagOut := &mockDiagnosticOutput{}
	builder := safedisk.NewMockSandbox("/tmp", safedisk.ModeReadOnly)

	opts := applyCoordinatorOptions(
		WithDebounceDuration(2*time.Second),
		WithFileHashCache(cache),
		WithCodeEmitter(emitter),
		WithDiagnosticOutput(diagOut),
		WithBaseDirSandbox(builder),
	)

	assert.Equal(t, 2*time.Second, opts.debounceDuration)
	assert.Same(t, cache, opts.fileHashCache)
	assert.Same(t, emitter, opts.codeEmitter)
	assert.Same(t, diagOut, opts.diagnosticOutput)
	assert.Same(t, builder, opts.baseDirSandbox)
	assert.NotNil(t, opts.clock, "clock should be set by default")
}

func TestNewCoordinatorService(t *testing.T) {
	t.Parallel()

	annotator := &mockAnnotator{ResultToReturn: &annotator_dto.ProjectAnnotationResult{}}
	cache := newMockCache()
	introspectionCache := newMockIntrospectionCache()
	fsReader := &mockFSReader{Files: make(map[string][]byte)}
	resolver := &resolver_domain.MockResolver{
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
	}

	options := applyCoordinatorOptions()

	service := newCoordinatorService(annotator, cache, introspectionCache, fsReader, resolver, options)

	require.NotNil(t, service)
	assert.Same(t, annotator, service.annotator)
	assert.Same(t, cache, service.cache)
	assert.Same(t, introspectionCache, service.introspectionCache)
	assert.Same(t, fsReader, service.fsReader)
	assert.Same(t, resolver, service.resolver)
	assert.NotNil(t, service.clock)
	assert.NotNil(t, service.shutdown)
	assert.NotNil(t, service.subscribers)
	assert.NotNil(t, service.rebuildTrigger)
	assert.Equal(t, stateIdle, service.status.State)
	assert.Nil(t, service.status.Result)
	assert.Nil(t, service.status.LastBuildError)
	assert.True(t, service.status.LastBuildTime.IsZero())
	assert.Equal(t, uint64(0), service.nextSubID)
	assert.Nil(t, service.debounceTimer)
	assert.Nil(t, service.lastBuildRequest)
	assert.True(t, service.lastTriggerTime.IsZero())
	assert.Equal(t, defaultDebounceDuration, service.debounceDuration)
	assert.Nil(t, service.codeEmitter)
	assert.Nil(t, service.diagnosticOutput)
	assert.Nil(t, service.fileHashCache)
	assert.Nil(t, service.baseDirSandbox)
	assert.Empty(t, service.baseDirSandboxPath)
}

func TestNewCoordinatorService_WithOptions(t *testing.T) {
	t.Parallel()

	emitter := &mockCodeEmitter{}
	diagOut := &mockDiagnosticOutput{}
	fhCache := newMockFileHashCache()
	builder := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)

	options := applyCoordinatorOptions(
		WithCodeEmitter(emitter),
		WithDiagnosticOutput(diagOut),
		WithFileHashCache(fhCache),
		WithBaseDirSandbox(builder),
		WithDebounceDuration(5*time.Second),
	)

	service := newCoordinatorService(
		&mockAnnotator{ResultToReturn: &annotator_dto.ProjectAnnotationResult{}},
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
		options,
	)

	require.NotNil(t, service)
	assert.Same(t, emitter, service.codeEmitter)
	assert.Same(t, diagOut, service.diagnosticOutput)
	assert.Same(t, fhCache, service.fileHashCache)
	assert.Same(t, builder, service.baseDirSandbox)
	assert.Equal(t, 5*time.Second, service.debounceDuration)
}

func TestGetStatus_InitialState(t *testing.T) {
	t.Parallel()

	service := newCoordinatorService(
		&mockAnnotator{ResultToReturn: &annotator_dto.ProjectAnnotationResult{}},
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

	status := service.GetStatus()
	assert.Equal(t, stateIdle, status.State)
	assert.Nil(t, status.Result)
	assert.Nil(t, status.LastBuildError)
	assert.True(t, status.LastBuildTime.IsZero())
}

func TestGetLastSuccessfulBuild_NoResult(t *testing.T) {
	t.Parallel()

	service := newCoordinatorService(
		&mockAnnotator{ResultToReturn: &annotator_dto.ProjectAnnotationResult{}},
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

	result, ok := service.GetLastSuccessfulBuild()
	assert.False(t, ok)
	assert.Nil(t, result)
}

func TestGetLastSuccessfulBuild_WithResult(t *testing.T) {
	t.Parallel()

	expected := &annotator_dto.ProjectAnnotationResult{}
	service := newCoordinatorService(
		&mockAnnotator{ResultToReturn: expected},
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

	service.status.Result = expected

	result, ok := service.GetLastSuccessfulBuild()
	assert.True(t, ok)
	assert.Same(t, expected, result)
}

func TestUpdateStatus(t *testing.T) {
	t.Parallel()

	t.Run("sets state to building", func(t *testing.T) {
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

		service.updateStatus(context.Background(), stateBuilding, nil, nil, "cause-1")

		status := service.GetStatus()
		assert.Equal(t, stateBuilding, status.State)
		assert.Nil(t, status.Result)
		assert.Nil(t, status.LastBuildError)
		assert.False(t, status.LastBuildTime.IsZero())
	})

	t.Run("sets state to failed with error", func(t *testing.T) {
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

		buildErr := errors.New("build failed")
		service.updateStatus(context.Background(), stateFailed, nil, buildErr, "cause-2")

		status := service.GetStatus()
		assert.Equal(t, stateFailed, status.State)
		assert.ErrorIs(t, status.LastBuildError, buildErr)
	})

	t.Run("publishes notification on stateReady with result", func(t *testing.T) {
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

		notificationChannel, unsub := service.Subscribe("test")
		defer unsub()

		expected := &annotator_dto.ProjectAnnotationResult{}
		service.updateStatus(context.Background(), stateReady, expected, nil, "cause-3")

		select {
		case notification := <-notificationChannel:
			assert.Same(t, expected, notification.Result)
			assert.Equal(t, "cause-3", notification.CausationID)
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for notification")
		}
	})

	t.Run("does not publish on stateReady with nil result", func(t *testing.T) {
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

		notificationChannel, unsub := service.Subscribe("test")
		defer unsub()

		service.updateStatus(context.Background(), stateReady, nil, nil, "cause-4")

		select {
		case <-notificationChannel:
			t.Fatal("should not have received notification for nil result")
		case <-time.After(50 * time.Millisecond):

		}
	})

	t.Run("does not publish on non-ready state", func(t *testing.T) {
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

		notificationChannel, unsub := service.Subscribe("test")
		defer unsub()

		result := &annotator_dto.ProjectAnnotationResult{}
		service.updateStatus(context.Background(), stateBuilding, result, nil, "cause-5")

		select {
		case <-notificationChannel:
			t.Fatal("should not have received notification for stateBuilding")
		case <-time.After(50 * time.Millisecond):

		}
	})
}

func TestSubscribe(t *testing.T) {
	t.Parallel()

	t.Run("returns channel and unsubscribe func", func(t *testing.T) {
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

		notificationChannel, unsub := service.Subscribe("sub-1")
		require.NotNil(t, notificationChannel)
		require.NotNil(t, unsub)

		assert.Equal(t, 1, len(service.subscribers))
		unsub()
		assert.Equal(t, 0, len(service.subscribers))
	})

	t.Run("double unsubscribe is safe", func(t *testing.T) {
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

		_, unsub := service.Subscribe("sub-2")
		unsub()
		assert.NotPanics(t, func() { unsub() })
	})

	t.Run("multiple subscribers get unique IDs", func(t *testing.T) {
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

		_, unsub1 := service.Subscribe("sub-a")
		_, unsub2 := service.Subscribe("sub-b")
		_, unsub3 := service.Subscribe("sub-c")
		defer unsub1()
		defer unsub2()
		defer unsub3()

		assert.Equal(t, 3, len(service.subscribers))
	})
}

func TestPublish(t *testing.T) {
	t.Parallel()

	t.Run("no subscribers is a no-op", func(t *testing.T) {
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

		assert.NotPanics(t, func() {
			service.publish(context.Background(), BuildNotification{})
		})
	})

	t.Run("notification is sent to all subscribers", func(t *testing.T) {
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

		ch1, unsub1 := service.Subscribe("sub-1")
		ch2, unsub2 := service.Subscribe("sub-2")
		defer unsub1()
		defer unsub2()

		expected := &annotator_dto.ProjectAnnotationResult{}
		notification := BuildNotification{
			Result:      expected,
			CausationID: "test-cause",
		}
		service.publish(context.Background(), notification)

		select {
		case n := <-ch1:
			assert.Same(t, expected, n.Result)
			assert.Equal(t, "test-cause", n.CausationID)
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for notification on ch1")
		}

		select {
		case n := <-ch2:
			assert.Same(t, expected, n.Result)
			assert.Equal(t, "test-cause", n.CausationID)
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for notification on ch2")
		}
	})

	t.Run("full channel does not block publisher", func(t *testing.T) {
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

		notificationChannel, unsub := service.Subscribe("slow-sub")
		defer unsub()

		ch2 := notificationChannel
		_ = ch2
		service.publish(context.Background(), BuildNotification{CausationID: "first"})

		done := make(chan struct{})
		go func() {
			service.publish(context.Background(), BuildNotification{CausationID: "second"})
			close(done)
		}()

		select {
		case <-done:

		case <-time.After(time.Second):
			t.Fatal("publish blocked on full channel")
		}
	})
}

func TestTriggerBuild(t *testing.T) {
	t.Parallel()

	t.Run("sends request to rebuild trigger channel", func(t *testing.T) {
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

		request := &coordinator_dto.BuildRequest{
			CausationID: "test-trigger",
		}

		service.triggerBuild(context.Background(), request)

		select {
		case received := <-service.rebuildTrigger:
			assert.Same(t, request, received)
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for request in rebuild trigger channel")
		}
	})

	t.Run("does not block when channel is full", func(t *testing.T) {
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

		service.rebuildTrigger <- &coordinator_dto.BuildRequest{CausationID: "first"}

		done := make(chan struct{})
		go func() {
			service.triggerBuild(context.Background(), &coordinator_dto.BuildRequest{CausationID: "second"})
			close(done)
		}()

		select {
		case <-done:

		case <-time.After(time.Second):
			t.Fatal("triggerBuild blocked on full channel")
		}
	})
}

func TestNotifyWaiters(t *testing.T) {
	t.Parallel()

	t.Run("notifies waiter with result", func(t *testing.T) {
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

		waiter := &buildWaiter{done: make(chan struct{})}
		service.waiters.Store("hash-1", waiter)

		expected := &annotator_dto.ProjectAnnotationResult{}
		service.notifyWaiters(context.Background(), "hash-1", expected, nil)

		select {
		case <-waiter.done:
			assert.Same(t, expected, waiter.result)
			assert.NoError(t, waiter.err)
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for waiter notification")
		}
	})

	t.Run("notifies waiter with error", func(t *testing.T) {
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

		waiter := &buildWaiter{done: make(chan struct{})}
		service.waiters.Store("hash-2", waiter)

		expectedErr := errors.New("build failed")
		service.notifyWaiters(context.Background(), "hash-2", nil, expectedErr)

		select {
		case <-waiter.done:
			assert.Nil(t, waiter.result)
			assert.ErrorIs(t, waiter.err, expectedErr)
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for waiter notification")
		}
	})

	t.Run("no-op when hash not found", func(t *testing.T) {
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

		assert.NotPanics(t, func() {
			service.notifyWaiters(context.Background(), "nonexistent", nil, nil)
		})
	})

	t.Run("removes waiter from map after notification", func(t *testing.T) {
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

		waiter := &buildWaiter{done: make(chan struct{})}
		service.waiters.Store("hash-3", waiter)

		service.notifyWaiters(context.Background(), "hash-3", &annotator_dto.ProjectAnnotationResult{}, nil)

		_, loaded := service.waiters.Load("hash-3")
		assert.False(t, loaded, "waiter should be removed after notification")
	})

	t.Run("handles non-ProjectAnnotationResult gracefully", func(t *testing.T) {
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

		waiter := &buildWaiter{done: make(chan struct{})}
		service.waiters.Store("hash-4", waiter)

		service.notifyWaiters(context.Background(), "hash-4", "not a result", nil)

		select {
		case <-waiter.done:
			assert.Nil(t, waiter.result, "result should be nil when wrong type")
		case <-time.After(time.Second):
			t.Fatal("timed out")
		}
	})

	t.Run("handles wrong type in waiters map", func(t *testing.T) {
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

		service.waiters.Store("hash-5", "wrong type")

		assert.NotPanics(t, func() {
			service.notifyWaiters(context.Background(), "hash-5", nil, nil)
		})
	})
}

func TestInvalidate(t *testing.T) {
	t.Parallel()

	t.Run("clears cache and status result", func(t *testing.T) {
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

		service.status.Result = &annotator_dto.ProjectAnnotationResult{}

		err := service.Invalidate(context.Background())

		require.NoError(t, err)
		assert.Equal(t, 1, cache.clearCalls)
		assert.Nil(t, service.status.Result)
	})

	t.Run("returns error when cache clear fails", func(t *testing.T) {
		t.Parallel()

		failCache := &mockCacheWithClearError{clearErr: errors.New("clear failed")}
		service := newCoordinatorService(
			&mockAnnotator{},
			failCache,
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

		err := service.Invalidate(context.Background())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "clearing annotation cache")
	})
}

type mockCacheWithClearError struct {
	clearErr error
}

func (m *mockCacheWithClearError) Get(_ context.Context, _ string) (*annotator_dto.ProjectAnnotationResult, error) {
	return nil, ErrCacheMiss
}

func (m *mockCacheWithClearError) Set(_ context.Context, _ string, _ *annotator_dto.ProjectAnnotationResult) error {
	return nil
}

func (m *mockCacheWithClearError) Clear(_ context.Context) error {
	return m.clearErr
}

func TestHandleSemanticError(t *testing.T) {
	t.Parallel()

	t.Run("returns partial result with error", func(t *testing.T) {
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

		partialResult := &annotator_dto.ProjectAnnotationResult{
			ComponentResults: map[string]*annotator_dto.AnnotationResult{"a": {}},
		}

		semErr := annotator_domain.NewSemanticError([]*ast_domain.Diagnostic{
			{Message: "type error", SourcePath: "/test.pk"},
		})

		logStore, _ := annotator_domain.NewCompilationLogStore(context.Background(), false, "", slog.LevelDebug)
		request := &coordinator_dto.BuildRequest{CausationID: "cause-1"}
		allSourceContents := map[string][]byte{"/test.pk": []byte("content")}

		result, err := service.handleSemanticError(
			context.Background(),
			semErr,
			partialResult,
			allSourceContents,
			logStore,
			request,
		)

		assert.Error(t, err)
		assert.Same(t, semErr, err)
		assert.Same(t, partialResult, result)
		assert.Equal(t, allSourceContents, partialResult.AllSourceContents)
		assert.Equal(t, stateFailed, service.GetStatus().State)
	})

	t.Run("returns nil result when build result is nil", func(t *testing.T) {
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

		semErr := annotator_domain.NewSemanticError(nil)
		logStore, _ := annotator_domain.NewCompilationLogStore(context.Background(), false, "", slog.LevelDebug)
		request := &coordinator_dto.BuildRequest{CausationID: "cause-2"}

		result, err := service.handleSemanticError(
			context.Background(),
			semErr,
			nil,
			nil,
			logStore,
			request,
		)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("calls diagnostic output when set and diagnostics present", func(t *testing.T) {
		t.Parallel()
		diagOut := &mockDiagnosticOutput{}
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
			applyCoordinatorOptions(WithDiagnosticOutput(diagOut)),
		)

		semErr := annotator_domain.NewSemanticError([]*ast_domain.Diagnostic{
			{Message: "error 1", SourcePath: "/a.pk"},
		})
		logStore, _ := annotator_domain.NewCompilationLogStore(context.Background(), false, "", slog.LevelDebug)
		request := &coordinator_dto.BuildRequest{CausationID: "cause-3"}

		_, _ = service.handleSemanticError(
			context.Background(),
			semErr,
			&annotator_dto.ProjectAnnotationResult{},
			map[string][]byte{"/a.pk": []byte("content")},
			logStore,
			request,
		)

		assert.Equal(t, 1, diagOut.Calls)
		assert.True(t, diagOut.IsError)
	})
}

func TestBuildStatus(t *testing.T) {
	t.Parallel()

	t.Run("zero value has expected defaults", func(t *testing.T) {
		t.Parallel()
		var bs buildStatus
		assert.Equal(t, stateIdle, bs.State)
		assert.Nil(t, bs.Result)
		assert.Nil(t, bs.LastBuildError)
		assert.True(t, bs.LastBuildTime.IsZero())
	})

	t.Run("can be constructed with all fields", func(t *testing.T) {
		t.Parallel()
		now := time.Now()
		result := &annotator_dto.ProjectAnnotationResult{}
		err := errors.New("some error")

		bs := buildStatus{
			State:          stateFailed,
			Result:         result,
			LastBuildError: err,
			LastBuildTime:  now,
		}

		assert.Equal(t, stateFailed, bs.State)
		assert.Same(t, result, bs.Result)
		assert.ErrorIs(t, bs.LastBuildError, err)
		assert.Equal(t, now, bs.LastBuildTime)
	})
}

func TestBuildNotification(t *testing.T) {
	t.Parallel()

	t.Run("zero value", func(t *testing.T) {
		t.Parallel()
		var bn BuildNotification
		assert.Nil(t, bn.Result)
		assert.Empty(t, bn.CausationID)
	})

	t.Run("with values", func(t *testing.T) {
		t.Parallel()
		result := &annotator_dto.ProjectAnnotationResult{}
		bn := BuildNotification{
			Result:      result,
			CausationID: "test-id",
		}
		assert.Same(t, result, bn.Result)
		assert.Equal(t, "test-id", bn.CausationID)
	})
}

func TestSubscriberStruct(t *testing.T) {
	t.Parallel()

	t.Run("fields are accessible", func(t *testing.T) {
		t.Parallel()
		notificationChannel := make(chan BuildNotification, 1)
		sub := subscriber{
			notificationChannel: notificationChannel,
			name:                "test-subscriber",
		}

		assert.Equal(t, "test-subscriber", sub.name)
		assert.NotNil(t, sub.notificationChannel)
	})
}

func TestFileHashResult(t *testing.T) {
	t.Parallel()

	t.Run("fields are accessible", func(t *testing.T) {
		t.Parallel()
		fhr := fileHashResult{
			path:    "/test/file.go",
			hash:    "abc123",
			content: []byte("package main"),
		}

		assert.Equal(t, "/test/file.go", fhr.path)
		assert.Equal(t, "abc123", fhr.hash)
		assert.Equal(t, []byte("package main"), fhr.content)
	})

	t.Run("zero value", func(t *testing.T) {
		t.Parallel()
		var fhr fileHashResult
		assert.Empty(t, fhr.path)
		assert.Empty(t, fhr.hash)
		assert.Nil(t, fhr.content)
	})
}

func TestTier1CacheResultStruct(t *testing.T) {
	t.Parallel()

	t.Run("zero value defaults to not using fast path", func(t *testing.T) {
		t.Parallel()
		var r tier1CacheResult
		assert.False(t, r.useFastPath)
		assert.Nil(t, r.entry)
		assert.Nil(t, r.scriptHashes)
		assert.Empty(t, r.introspectionHash)
	})

	t.Run("fields with complete cache miss info", func(t *testing.T) {
		t.Parallel()
		r := tier1CacheResult{
			scriptHashes:      map[string]string{"a.pk": "hash1"},
			entry:             nil,
			introspectionHash: "hash-abc",
			useFastPath:       false,
		}
		assert.False(t, r.useFastPath)
		assert.Nil(t, r.entry)
		assert.Equal(t, "hash-abc", r.introspectionHash)
		assert.Equal(t, "hash1", r.scriptHashes["a.pk"])
	})
}

func TestBuildWaiterStruct(t *testing.T) {
	t.Parallel()

	t.Run("signalling completion works", func(t *testing.T) {
		t.Parallel()
		waiter := &buildWaiter{
			done: make(chan struct{}),
		}

		result := &annotator_dto.ProjectAnnotationResult{}
		go func() {
			waiter.result = result
			waiter.err = nil
			close(waiter.done)
		}()

		select {
		case <-waiter.done:
			assert.Same(t, result, waiter.result)
			assert.NoError(t, waiter.err)
		case <-time.After(time.Second):
			t.Fatal("timed out")
		}
	})

	t.Run("error signalling works", func(t *testing.T) {
		t.Parallel()
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
			assert.Nil(t, waiter.result)
		case <-time.After(time.Second):
			t.Fatal("timed out")
		}
	})
}

func TestBuildOptionsStruct(t *testing.T) {
	t.Parallel()

	t.Run("zero value", func(t *testing.T) {
		t.Parallel()
		var bo buildOptions
		assert.Nil(t, bo.Resolver)
		assert.Nil(t, bo.InspectionCacheHints)
		assert.Empty(t, bo.CausationID)
		assert.Nil(t, bo.ChangedFiles)
		assert.False(t, bo.SkipInspection)
		assert.False(t, bo.FaultTolerant)
	})
}

func TestCoordinatorOptionsStruct(t *testing.T) {
	t.Parallel()

	t.Run("zero value", func(t *testing.T) {
		t.Parallel()
		var co coordinatorOptions
		assert.Nil(t, co.fileHashCache)
		assert.Nil(t, co.codeEmitter)
		assert.Nil(t, co.diagnosticOutput)
		assert.Nil(t, co.clock)
		assert.Nil(t, co.baseDirSandbox)
		assert.Equal(t, time.Duration(0), co.debounceDuration)
	})
}

func TestSetLastBuildRequest(t *testing.T) {
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

	entryPoints := []annotator_dto.EntryPoint{
		{Path: "/project/pages/index.pk", IsPage: true},
	}
	buildOpts := &buildOptions{
		CausationID:   "test-cause",
		FaultTolerant: true,
	}

	service.setLastBuildRequest(context.Background(), entryPoints, buildOpts)

	service.mu.RLock()
	defer service.mu.RUnlock()

	require.NotNil(t, service.lastBuildRequest)
	assert.Equal(t, "test-cause", service.lastBuildRequest.CausationID)
	assert.True(t, service.lastBuildRequest.FaultTolerant)
	assert.Equal(t, entryPoints, service.lastBuildRequest.EntryPoints)
}

func TestGetStatusConcurrent(t *testing.T) {
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

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Go(func() {
			result := &annotator_dto.ProjectAnnotationResult{}
			if i%2 == 0 {
				service.updateStatus(context.Background(), stateReady, result, nil, "cause")
			} else {
				service.updateStatus(context.Background(), stateBuilding, nil, nil, "cause")
			}
		})

		wg.Go(func() {
			_ = service.GetStatus()
			_, _ = service.GetLastSuccessfulBuild()
		})
	}

	wg.Wait()
}

func TestOutputInternalCompilerLogs(t *testing.T) {
	t.Parallel()

	t.Run("no-op when diagnostics are empty", func(t *testing.T) {
		t.Parallel()
		logStore, _ := annotator_domain.NewCompilationLogStore(context.Background(), false, "", slog.LevelDebug)
		assert.NotPanics(t, func() {
			outputInternalCompilerLogs(nil, logStore)
		})
	})

	t.Run("no-op when diagnostics have empty slice", func(t *testing.T) {
		t.Parallel()
		logStore, _ := annotator_domain.NewCompilationLogStore(context.Background(), false, "", slog.LevelDebug)
		assert.NotPanics(t, func() {
			outputInternalCompilerLogs([]*ast_domain.Diagnostic{}, logStore)
		})
	})

	t.Run("handles diagnostics with no matching log file", func(t *testing.T) {
		t.Parallel()
		logStore, _ := annotator_domain.NewCompilationLogStore(context.Background(), false, "", slog.LevelDebug)
		diagnostics := []*ast_domain.Diagnostic{
			{SourcePath: "/nonexistent.pk", Message: "error"},
		}
		assert.NotPanics(t, func() {
			outputInternalCompilerLogs(diagnostics, logStore)
		})
	})
}

func TestCacheIntrospectionResults_VirtualModuleNil(t *testing.T) {
	t.Parallel()

	cache := newMockIntrospectionCache()
	service := &coordinatorService{introspectionCache: cache}

	tier1Result := tier1CacheResult{
		introspectionHash: "hash123",
		scriptHashes:      map[string]string{"file.pk": "hash"},
	}

	service.cacheIntrospectionResults(
		context.Background(),
		tier1Result,
		&annotator_dto.ComponentGraph{},
		nil,
		&annotator_domain.TypeResolver{},
	)

	assert.Equal(t, 0, cache.setCalls)
}

func TestCacheIntrospectionResults_TypeResolverNil(t *testing.T) {
	t.Parallel()

	cache := newMockIntrospectionCache()
	service := &coordinatorService{introspectionCache: cache}

	tier1Result := tier1CacheResult{
		introspectionHash: "hash123",
		scriptHashes:      map[string]string{"file.pk": "hash"},
	}

	service.cacheIntrospectionResults(
		context.Background(),
		tier1Result,
		&annotator_dto.ComponentGraph{},
		&annotator_dto.VirtualModule{},
		nil,
	)

	assert.Equal(t, 0, cache.setCalls)
}

func TestCacheIntrospectionResults_CacheSetError(t *testing.T) {
	t.Parallel()

	cache := &failingIntrospectionCache{setErr: errors.New("cache write failed")}
	service := &coordinatorService{introspectionCache: cache}

	tier1Result := tier1CacheResult{
		introspectionHash: "hash123",
		scriptHashes:      map[string]string{"file.pk": "hash"},
	}

	assert.NotPanics(t, func() {
		service.cacheIntrospectionResults(
			context.Background(),
			tier1Result,
			&annotator_dto.ComponentGraph{},
			&annotator_dto.VirtualModule{},
			&annotator_domain.TypeResolver{},
		)
	})
}

type failingIntrospectionCache struct {
	setErr error
}

func (m *failingIntrospectionCache) Get(_ context.Context, _ string) (*IntrospectionCacheEntry, error) {
	return nil, ErrCacheMiss
}

func (m *failingIntrospectionCache) Set(_ context.Context, _ string, _ *IntrospectionCacheEntry) error {
	return m.setErr
}

func (m *failingIntrospectionCache) Clear(_ context.Context) error {
	return nil
}

func TestResolveEntryPoint(t *testing.T) {
	t.Parallel()

	t.Run("resolves path with module prefix already present", func(t *testing.T) {
		t.Parallel()
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
		discoverer := &hasherDependencyDiscoverer{
			resolver: resolver,
			fsReader: &mockFSReader{Files: make(map[string][]byte)},
		}

		result, err := discoverer.resolveEntryPoint(context.Background(), "test-module/pages/index.pk", "test-module")
		require.NoError(t, err)
		assert.Equal(t, "/project/pages/index.pk", result)
	})

	t.Run("resolves path without module prefix", func(t *testing.T) {
		t.Parallel()
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
		discoverer := &hasherDependencyDiscoverer{
			resolver: resolver,
			fsReader: &mockFSReader{Files: make(map[string][]byte)},
		}

		result, err := discoverer.resolveEntryPoint(context.Background(), "pages/index.pk", "test-module")
		require.NoError(t, err)
		assert.Equal(t, "/project/pages/index.pk", result)
	})

	t.Run("absolute paths are returned directly", func(t *testing.T) {
		t.Parallel()
		resolver := &resolver_domain.MockResolver{
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
		}
		discoverer := &hasherDependencyDiscoverer{
			resolver: resolver,
			fsReader: &mockFSReader{Files: make(map[string][]byte)},
		}

		result, err := discoverer.resolveEntryPoint(context.Background(), "/absolute/path.pk", "")
		require.NoError(t, err)
		assert.Equal(t, "/absolute/path.pk", result)
	})
}

func TestEnqueueEntryPoints(t *testing.T) {
	t.Parallel()

	t.Run("adds resolved paths to queue", func(t *testing.T) {
		t.Parallel()
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
		discoverer := &hasherDependencyDiscoverer{
			resolver: resolver,
			fsReader: &mockFSReader{Files: make(map[string][]byte)},
		}

		var queue []string
		visited := make(map[string]bool)

		err := discoverer.enqueueEntryPoints(context.Background(),
			[]string{"test-module/pages/index.pk"},
			&queue,
			visited,
		)

		require.NoError(t, err)
		assert.Len(t, queue, 1)
		assert.True(t, visited[queue[0]])
	})

	t.Run("deduplicates paths", func(t *testing.T) {
		t.Parallel()
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
		discoverer := &hasherDependencyDiscoverer{
			resolver: resolver,
			fsReader: &mockFSReader{Files: make(map[string][]byte)},
		}

		var queue []string
		visited := make(map[string]bool)

		err := discoverer.enqueueEntryPoints(context.Background(),
			[]string{
				"test-module/pages/index.pk",
				"test-module/pages/index.pk",
			},
			&queue,
			visited,
		)

		require.NoError(t, err)
		assert.Len(t, queue, 1)
	})

	t.Run("empty paths", func(t *testing.T) {
		t.Parallel()
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
		discoverer := &hasherDependencyDiscoverer{
			resolver: resolver,
			fsReader: &mockFSReader{Files: make(map[string][]byte)},
		}

		var queue []string
		visited := make(map[string]bool)

		err := discoverer.enqueueEntryPoints(context.Background(), nil, &queue, visited)

		require.NoError(t, err)
		assert.Empty(t, queue)
	})
}

func TestProcessImportQueue(t *testing.T) {
	t.Parallel()

	t.Run("processes queue of pk files", func(t *testing.T) {
		t.Parallel()
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
		fsReader := &mockFSReader{
			Files: map[string][]byte{
				"/project/pages/index.pk": []byte(`<script lang="go">
package pages

import "test-module/components/header.pk"
</script>`),
				"/project/components/header.pk": []byte(`<script lang="go">
package components
</script>`),
			},
		}
		discoverer := &hasherDependencyDiscoverer{
			resolver: resolver,
			fsReader: fsReader,
		}

		queue := []string{"/project/pages/index.pk"}
		visited := map[string]bool{"/project/pages/index.pk": true}

		err := discoverer.processImportQueue(context.Background(), &queue, visited)

		require.NoError(t, err)
		assert.Len(t, queue, 2, "should have discovered the header.pk import")
		assert.True(t, visited["/project/components/header.pk"])
	})

	t.Run("empty queue is a no-op", func(t *testing.T) {
		t.Parallel()
		discoverer := &hasherDependencyDiscoverer{
			resolver: &resolver_domain.MockResolver{
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
			fsReader: &mockFSReader{Files: make(map[string][]byte)},
		}

		queue := []string{}
		visited := make(map[string]bool)

		err := discoverer.processImportQueue(context.Background(), &queue, visited)
		require.NoError(t, err)
		assert.Empty(t, queue)
	})
}

func TestExtractPKImports_EmptyScript(t *testing.T) {
	t.Parallel()

	fsReader := &mockFSReader{
		Files: map[string][]byte{
			"/test.pk": []byte(`<script lang="go">
</script>`),
		},
	}

	discoverer := &hasherDependencyDiscoverer{
		fsReader: fsReader,
	}

	imports, err := discoverer.extractPKImports(context.Background(), "/test.pk")
	require.NoError(t, err)
	assert.Nil(t, imports, "empty script block should return nil imports")
}

func TestExtractPKImports_ReadError(t *testing.T) {
	t.Parallel()

	fsReader := &mockFSReader{
		Files: map[string][]byte{},
	}

	discoverer := &hasherDependencyDiscoverer{
		fsReader: fsReader,
	}

	imports, err := discoverer.extractPKImports(context.Background(), "/nonexistent.pk")
	assert.Error(t, err)
	assert.Nil(t, imports)
}

func TestExtractPKImports_InvalidSFC(t *testing.T) {
	t.Parallel()

	fsReader := &mockFSReader{
		Files: map[string][]byte{
			"/bad.pk": []byte(`<script lang="go">
package test

import "some/path.pk"
`),
		},
	}

	discoverer := &hasherDependencyDiscoverer{
		fsReader: fsReader,
	}

	imports, err := discoverer.extractPKImports(context.Background(), "/bad.pk")

	_ = err
	_ = imports
}

func TestExtractScriptBlockContent_InvalidContent(t *testing.T) {
	t.Parallel()

	t.Run("empty content", func(t *testing.T) {
		t.Parallel()
		scriptContent, scriptHash, err := extractScriptBlockContent("/test.pk", []byte(""))

		_ = err
		_ = scriptContent
		_ = scriptHash
	})

	t.Run("content with only template", func(t *testing.T) {
		t.Parallel()
		content := []byte(`<template><div>Hello</div></template>`)
		scriptContent, scriptHash, err := extractScriptBlockContent("/test.pk", content)
		require.NoError(t, err)
		assert.Empty(t, scriptContent)
		assert.Empty(t, scriptHash)
	})
}

func TestHashIntrospectionContent_NonPKNonGo(t *testing.T) {
	t.Parallel()

	hasher := xxhash.New()
	paths := []string{"/readme.md", "/config.json"}
	contents := map[string][]byte{
		"/readme.md":   []byte("# README"),
		"/config.json": []byte(`{"key":"value"}`),
	}

	scriptHashes, err := hashIntrospectionContent(hasher, paths, contents)
	require.NoError(t, err)
	assert.Empty(t, scriptHashes, "non-pk, non-go files should produce no script hashes")
}

func TestHashIntrospectionContent_EmptyPaths(t *testing.T) {
	t.Parallel()

	hasher := xxhash.New()
	scriptHashes, err := hashIntrospectionContent(hasher, nil, nil)
	require.NoError(t, err)
	assert.Empty(t, scriptHashes)
}

func TestHashIntrospectionContent_EmptyPathsSlice(t *testing.T) {
	t.Parallel()

	hasher := xxhash.New()
	scriptHashes, err := hashIntrospectionContent(hasher, []string{}, map[string][]byte{})
	require.NoError(t, err)
	assert.Empty(t, scriptHashes)
}

func TestCollectHashResults_LargeCapacity(t *testing.T) {
	t.Parallel()

	results := make(chan fileHashResult, 2)
	results <- fileHashResult{path: "/a.go", hash: "h1", content: []byte("a")}
	results <- fileHashResult{path: "/b.go", hash: "h2", content: []byte("b")}
	close(results)

	hashes, contents, err := collectHashResults(results, 100)
	require.NoError(t, err)
	assert.Len(t, hashes, 2)
	assert.Len(t, contents, 2)
}

func TestCollectHashResults_DuplicatePaths(t *testing.T) {
	t.Parallel()

	results := make(chan fileHashResult, 2)
	results <- fileHashResult{path: "/a.go", hash: "h1", content: []byte("first")}
	results <- fileHashResult{path: "/a.go", hash: "h2", content: []byte("second")}
	close(results)

	hashes, contents, err := collectHashResults(results, 2)
	require.NoError(t, err)

	assert.Len(t, hashes, 1)
	assert.Equal(t, "h2", hashes["/a.go"])
	assert.Equal(t, []byte("second"), contents["/a.go"])
}

func TestIsRelevantGoFile_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{name: "just .go extension", filename: ".go", expected: true},
		{name: "_test.go only", filename: "_test.go", expected: false},
		{name: "double .go.go", filename: "file.go.go", expected: true},
		{name: "with directory separator", filename: "pkg/file.go", expected: true},
		{name: "uppercase extension", filename: "file.GO", expected: false},
		{name: "mixed case", filename: "file.Go", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, isRelevantGoFile(tt.filename))
		})
	}
}

func TestHandleDirEntry_EdgeCases(t *testing.T) {
	t.Parallel()

	s := &coordinatorService{}

	tests := []struct {
		name       string
		dirname    string
		expectSkip bool
	}{
		{name: "single dot", dirname: ".", expectSkip: false},
		{name: "double dot", dirname: "..", expectSkip: true},
		{name: "actions directory (not skipped)", dirname: "actions", expectSkip: false},
		{name: "internal directory (not skipped)", dirname: "internal", expectSkip: false},
		{name: "Vendor with capital (not skipped)", dirname: "Vendor", expectSkip: false},
		{name: "DIST uppercase (not skipped)", dirname: "DIST", expectSkip: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := s.handleDirEntry(tt.dirname)
			if tt.expectSkip {
				assert.ErrorIs(t, err, fs.SkipDir)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetEntrypointPaths_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("preserves IsPage information in paths only", func(t *testing.T) {
		t.Parallel()
		entryPoints := []annotator_dto.EntryPoint{
			{Path: "/page.pk", IsPage: true},
			{Path: "/component.pk", IsPage: false},
		}
		paths := getEntrypointPaths(entryPoints)
		assert.Equal(t, []string{"/page.pk", "/component.pk"}, paths)
	})

	t.Run("preserves order", func(t *testing.T) {
		t.Parallel()
		entryPoints := []annotator_dto.EntryPoint{
			{Path: "/z.pk"},
			{Path: "/a.pk"},
			{Path: "/m.pk"},
		}
		paths := getEntrypointPaths(entryPoints)
		assert.Equal(t, []string{"/z.pk", "/a.pk", "/m.pk"}, paths)
	})
}

func TestGetEffectiveResolver_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("returns default when buildOpts resolver is same as default", func(t *testing.T) {
		t.Parallel()
		resolver := &resolver_domain.MockResolver{GetBaseDirFunc: func() string { return "/default" }}
		service := &coordinatorService{resolver: resolver}

		result := service.getEffectiveResolver(&buildOptions{Resolver: resolver})
		assert.Same(t, resolver, result)
	})
}

func TestMockFileHashCache_Load(t *testing.T) {
	t.Parallel()
	cache := newMockFileHashCache()
	err := cache.Load(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 1, cache.loadCalls)
}

func TestMockFileHashCache_Persist(t *testing.T) {
	t.Parallel()

	t.Run("no error", func(t *testing.T) {
		t.Parallel()
		cache := newMockFileHashCache()
		err := cache.Persist(context.Background())
		assert.NoError(t, err)
	})

	t.Run("with error", func(t *testing.T) {
		t.Parallel()
		cache := newMockFileHashCache()
		cache.persistErr = errors.New("persist failed")
		err := cache.Persist(context.Background())
		assert.Error(t, err)
	})
}

func TestInvalidate_DoesNotClearIntrospectionCache(t *testing.T) {
	t.Parallel()

	cache := newMockCache()
	introCache := newMockIntrospectionCache()

	service := newCoordinatorService(
		&mockAnnotator{},
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

	err := service.Invalidate(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 1, cache.clearCalls, "Tier 2 cache should be cleared")
	assert.Equal(t, 0, introCache.clearCalls, "Tier 1 introspection cache should NOT be cleared")
}

func TestOutputDiagnosticsIfPresent_EmptyDiagnosticSlice(t *testing.T) {
	t.Parallel()

	service := &coordinatorService{}
	buildResult := &annotator_dto.ProjectAnnotationResult{
		AllDiagnostics: []*ast_domain.Diagnostic{},
	}
	result := service.outputDiagnosticsIfPresent(context.Background(), buildResult, nil, nil)
	assert.False(t, result)
}

func TestStateString_AllValues(t *testing.T) {
	t.Parallel()

	allStates := []struct {
		want string
		s    state
	}{
		{want: "Idle", s: stateIdle},
		{want: "Building", s: stateBuilding},
		{want: "Ready", s: stateReady},
		{want: "Failed", s: stateFailed},
		{want: "Unknown", s: state(-1)},
		{want: "Unknown", s: state(100)},
		{want: "Unknown", s: state(4)},
	}

	for _, tc := range allStates {
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, tc.s.String())
		})
	}
}
