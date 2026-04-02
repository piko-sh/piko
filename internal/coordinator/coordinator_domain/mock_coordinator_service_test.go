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
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/annotator/annotator_dto"
)

func TestMockCoordinatorService_Subscribe(t *testing.T) {
	t.Parallel()

	t.Run("nil SubscribeFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockCoordinatorService{}
		notificationChannel, unsub := m.Subscribe("watcher")

		require.NotNil(t, notificationChannel, "channel must not be nil even when SubscribeFunc is nil")
		require.NotNil(t, unsub, "unsubscribe function must not be nil")

		_, open := <-notificationChannel
		assert.False(t, open, "default channel should be closed")

		assert.NotPanics(t, func() { unsub() })

		assert.Equal(t, int64(1), atomic.LoadInt64(&m.SubscribeCallCount))
	})

	t.Run("delegates to SubscribeFunc", func(t *testing.T) {
		t.Parallel()

		expected := make(chan BuildNotification, 1)
		var calledWith string

		m := &MockCoordinatorService{
			SubscribeFunc: func(name string) (<-chan BuildNotification, UnsubscribeFunc) {
				calledWith = name
				return expected, func() {}
			},
		}

		notificationChannel, unsub := m.Subscribe("my-subscriber")

		assert.Equal(t, "my-subscriber", calledWith)
		assert.Equal(t, (<-chan BuildNotification)(expected), notificationChannel)
		require.NotNil(t, unsub)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.SubscribeCallCount))
	})
}

func TestMockCoordinatorService_RequestRebuild(t *testing.T) {
	t.Parallel()

	t.Run("nil RequestRebuildFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockCoordinatorService{}

		assert.NotPanics(t, func() {
			m.RequestRebuild(context.Background(), nil)
		})

		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RequestRebuildCallCount))
	})

	t.Run("delegates to RequestRebuildFunc", func(t *testing.T) {
		t.Parallel()

		var (
			capturedCtx  context.Context
			capturedEPs  []annotator_dto.EntryPoint
			capturedOpts []BuildOption
		)

		eps := []annotator_dto.EntryPoint{{Path: "page.pk"}}
		opt := WithCausationID("rebuild-1")

		m := &MockCoordinatorService{
			RequestRebuildFunc: func(ctx context.Context, entryPoints []annotator_dto.EntryPoint, opts ...BuildOption) {
				capturedCtx = ctx
				capturedEPs = entryPoints
				capturedOpts = opts
			},
		}

		ctx := context.Background()
		m.RequestRebuild(ctx, eps, opt)

		assert.Equal(t, ctx, capturedCtx)
		assert.Equal(t, eps, capturedEPs)
		require.Len(t, capturedOpts, 1)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RequestRebuildCallCount))
	})
}

func TestMockCoordinatorService_GetResult(t *testing.T) {
	t.Parallel()

	t.Run("nil GetResultFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockCoordinatorService{}
		result, err := m.GetResult(context.Background(), nil)

		assert.Nil(t, result)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetResultCallCount))
	})

	t.Run("delegates to GetResultFunc", func(t *testing.T) {
		t.Parallel()

		want := &annotator_dto.ProjectAnnotationResult{}
		eps := []annotator_dto.EntryPoint{{Path: "index.pk"}}

		m := &MockCoordinatorService{
			GetResultFunc: func(_ context.Context, entryPoints []annotator_dto.EntryPoint, _ ...BuildOption) (*annotator_dto.ProjectAnnotationResult, error) {
				assert.Equal(t, eps, entryPoints)
				return want, nil
			},
		}

		got, err := m.GetResult(context.Background(), eps)

		assert.NoError(t, err)
		assert.Same(t, want, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetResultCallCount))
	})

	t.Run("propagates error from GetResultFunc", func(t *testing.T) {
		t.Parallel()

		sentinel := errors.New("build failed")

		m := &MockCoordinatorService{
			GetResultFunc: func(_ context.Context, _ []annotator_dto.EntryPoint, _ ...BuildOption) (*annotator_dto.ProjectAnnotationResult, error) {
				return nil, sentinel
			},
		}

		result, err := m.GetResult(context.Background(), nil)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, sentinel)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetResultCallCount))
	})
}

func TestMockCoordinatorService_GetOrBuildProject(t *testing.T) {
	t.Parallel()

	t.Run("nil GetOrBuildProjectFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockCoordinatorService{}
		result, err := m.GetOrBuildProject(context.Background(), nil)

		assert.Nil(t, result)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetOrBuildProjectCallCount))
	})

	t.Run("delegates to GetOrBuildProjectFunc", func(t *testing.T) {
		t.Parallel()

		want := &annotator_dto.ProjectAnnotationResult{}
		eps := []annotator_dto.EntryPoint{{Path: "app.pk"}}

		m := &MockCoordinatorService{
			GetOrBuildProjectFunc: func(_ context.Context, entryPoints []annotator_dto.EntryPoint, _ ...BuildOption) (*annotator_dto.ProjectAnnotationResult, error) {
				assert.Equal(t, eps, entryPoints)
				return want, nil
			},
		}

		got, err := m.GetOrBuildProject(context.Background(), eps, WithFaultTolerance())

		assert.NoError(t, err)
		assert.Same(t, want, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetOrBuildProjectCallCount))
	})

	t.Run("propagates error from GetOrBuildProjectFunc", func(t *testing.T) {
		t.Parallel()

		sentinel := errors.New("context cancelled")

		m := &MockCoordinatorService{
			GetOrBuildProjectFunc: func(_ context.Context, _ []annotator_dto.EntryPoint, _ ...BuildOption) (*annotator_dto.ProjectAnnotationResult, error) {
				return nil, sentinel
			},
		}

		result, err := m.GetOrBuildProject(context.Background(), nil)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, sentinel)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetOrBuildProjectCallCount))
	})
}

func TestMockCoordinatorService_GetLastSuccessfulBuild(t *testing.T) {
	t.Parallel()

	t.Run("nil GetLastSuccessfulBuildFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockCoordinatorService{}
		result, ok := m.GetLastSuccessfulBuild()

		assert.Nil(t, result)
		assert.False(t, ok)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetLastSuccessfulBuildCallCount))
	})

	t.Run("delegates to GetLastSuccessfulBuildFunc", func(t *testing.T) {
		t.Parallel()

		want := &annotator_dto.ProjectAnnotationResult{}

		m := &MockCoordinatorService{
			GetLastSuccessfulBuildFunc: func() (*annotator_dto.ProjectAnnotationResult, bool) {
				return want, true
			},
		}

		got, ok := m.GetLastSuccessfulBuild()

		assert.True(t, ok)
		assert.Same(t, want, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetLastSuccessfulBuildCallCount))
	})
}

func TestMockCoordinatorService_Invalidate(t *testing.T) {
	t.Parallel()

	t.Run("nil InvalidateFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockCoordinatorService{}
		err := m.Invalidate(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.InvalidateCallCount))
	})

	t.Run("delegates to InvalidateFunc", func(t *testing.T) {
		t.Parallel()

		var capturedCtx context.Context

		m := &MockCoordinatorService{
			InvalidateFunc: func(ctx context.Context) error {
				capturedCtx = ctx
				return nil
			},
		}

		ctx := context.Background()
		err := m.Invalidate(ctx)

		assert.NoError(t, err)
		assert.Equal(t, ctx, capturedCtx)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.InvalidateCallCount))
	})

	t.Run("propagates error from InvalidateFunc", func(t *testing.T) {
		t.Parallel()

		sentinel := errors.New("cache clear failed")

		m := &MockCoordinatorService{
			InvalidateFunc: func(_ context.Context) error {
				return sentinel
			},
		}

		err := m.Invalidate(context.Background())

		assert.ErrorIs(t, err, sentinel)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.InvalidateCallCount))
	})
}

func TestMockCoordinatorService_Shutdown(t *testing.T) {
	t.Parallel()

	t.Run("nil ShutdownFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockCoordinatorService{}

		assert.NotPanics(t, func() { m.Shutdown(context.Background()) })
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ShutdownCallCount))
	})

	t.Run("delegates to ShutdownFunc", func(t *testing.T) {
		t.Parallel()

		var called bool

		m := &MockCoordinatorService{
			ShutdownFunc: func(_ context.Context) {
				called = true
			},
		}

		m.Shutdown(context.Background())

		assert.True(t, called)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ShutdownCallCount))
	})
}

func TestMockCoordinatorService_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var m MockCoordinatorService

	var _ CoordinatorService = &m

	ctx := context.Background()
	eps := []annotator_dto.EntryPoint{{Path: "index.pk"}}

	notificationChannel, unsub := m.Subscribe("test")
	require.NotNil(t, notificationChannel)
	require.NotNil(t, unsub)
	unsub()

	m.RequestRebuild(ctx, eps)

	result, err := m.GetResult(ctx, eps)
	assert.Nil(t, result)
	assert.NoError(t, err)

	result, err = m.GetOrBuildProject(ctx, eps, WithCausationID("zero"))
	assert.Nil(t, result)
	assert.NoError(t, err)

	lastResult, ok := m.GetLastSuccessfulBuild()
	assert.Nil(t, lastResult)
	assert.False(t, ok)

	assert.NoError(t, m.Invalidate(ctx))

	assert.NotPanics(t, func() { m.Shutdown(context.Background()) })

	assert.Equal(t, int64(1), atomic.LoadInt64(&m.SubscribeCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&m.RequestRebuildCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetResultCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetOrBuildProjectCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetLastSuccessfulBuildCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&m.InvalidateCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&m.ShutdownCallCount))
}

func TestMockCoordinatorService_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	const goroutines = 50

	m := &MockCoordinatorService{
		SubscribeFunc: func(_ string) (<-chan BuildNotification, UnsubscribeFunc) {
			notificationChannel := make(chan BuildNotification)
			close(notificationChannel)
			return notificationChannel, func() {}
		},
		RequestRebuildFunc: func(_ context.Context, _ []annotator_dto.EntryPoint, _ ...BuildOption) {},
		GetResultFunc: func(_ context.Context, _ []annotator_dto.EntryPoint, _ ...BuildOption) (*annotator_dto.ProjectAnnotationResult, error) {
			return nil, nil
		},
		GetOrBuildProjectFunc: func(_ context.Context, _ []annotator_dto.EntryPoint, _ ...BuildOption) (*annotator_dto.ProjectAnnotationResult, error) {
			return nil, nil
		},
		GetLastSuccessfulBuildFunc: func() (*annotator_dto.ProjectAnnotationResult, bool) {
			return nil, false
		},
		InvalidateFunc: func(_ context.Context) error { return nil },
		ShutdownFunc:   func(_ context.Context) {},
	}

	ctx := context.Background()
	eps := []annotator_dto.EntryPoint{{Path: "concurrent.pk"}}

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			notificationChannel, unsub := m.Subscribe("concurrent")
			_ = notificationChannel
			unsub()

			m.RequestRebuild(ctx, eps)

			_, _ = m.GetResult(ctx, eps)
			_, _ = m.GetOrBuildProject(ctx, eps)
			_, _ = m.GetLastSuccessfulBuild()
			_ = m.Invalidate(ctx)
			m.Shutdown(context.Background())
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.SubscribeCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.RequestRebuildCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetResultCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetOrBuildProjectCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetLastSuccessfulBuildCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.InvalidateCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.ShutdownCallCount))
}
