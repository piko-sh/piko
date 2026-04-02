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
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func TestMockTypeInspectorBuilder_SetConfig(t *testing.T) {
	t.Parallel()

	t.Run("nil SetConfigFunc does nothing", func(t *testing.T) {
		t.Parallel()

		mock := &MockTypeInspectorBuilder{}

		mock.SetConfig(inspector_dto.Config{BaseDir: "/project"})

		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.SetConfigCallCount))
	})

	t.Run("delegates to SetConfigFunc", func(t *testing.T) {
		t.Parallel()

		var captured inspector_dto.Config

		mock := &MockTypeInspectorBuilder{
			SetConfigFunc: func(config inspector_dto.Config) {
				captured = config
			},
		}

		input := inspector_dto.Config{
			BaseDir:    "/my/project",
			ModuleName: "piko.sh/example",
			GOOS:       "linux",
			GOARCH:     "amd64",
		}

		mock.SetConfig(input)

		assert.Equal(t, input, captured)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.SetConfigCallCount))
	})
}

func TestMockTypeInspectorBuilder_Build(t *testing.T) {
	t.Parallel()

	t.Run("nil BuildFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &MockTypeInspectorBuilder{}

		err := mock.Build(
			context.Background(),
			map[string][]byte{"main.go": []byte("package main")},
			map[string]string{"script.js": "abc123"},
		)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.BuildCallCount))
	})

	t.Run("delegates to BuildFunc", func(t *testing.T) {
		t.Parallel()

		wantOverlay := map[string][]byte{"gen.go": []byte("package gen")}
		wantHashes := map[string]string{"app.js": "hash1"}

		mock := &MockTypeInspectorBuilder{
			BuildFunc: func(
				ctx context.Context,
				sourceOverlay map[string][]byte,
				scriptHashes map[string]string,
			) error {
				assert.Equal(t, wantOverlay, sourceOverlay)
				assert.Equal(t, wantHashes, scriptHashes)
				return nil
			},
		}

		err := mock.Build(context.Background(), wantOverlay, wantHashes)

		require.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.BuildCallCount))
	})

	t.Run("propagates error from BuildFunc", func(t *testing.T) {
		t.Parallel()

		wantErr := errors.New("build failed: missing module")

		mock := &MockTypeInspectorBuilder{
			BuildFunc: func(_ context.Context, _ map[string][]byte, _ map[string]string) error {
				return wantErr
			},
		}

		err := mock.Build(context.Background(), nil, nil)

		assert.ErrorIs(t, err, wantErr)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.BuildCallCount))
	})
}

func TestMockTypeInspectorBuilder_GetQuerier(t *testing.T) {
	t.Parallel()

	t.Run("nil GetQuerierFunc returns default mock querier and true", func(t *testing.T) {
		t.Parallel()

		mock := &MockTypeInspectorBuilder{}

		querier, ok := mock.GetQuerier()

		require.True(t, ok)
		assert.IsType(t, &inspector_domain.MockTypeQuerier{}, querier)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GetQuerierCallCount))
	})

	t.Run("delegates to GetQuerierFunc", func(t *testing.T) {
		t.Parallel()

		wantQuerier := &inspector_domain.MockTypeQuerier{}

		mock := &MockTypeInspectorBuilder{
			GetQuerierFunc: func() (TypeInspectorPort, bool) {
				return wantQuerier, true
			},
		}

		querier, ok := mock.GetQuerier()

		require.True(t, ok)
		assert.Same(t, wantQuerier, querier)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GetQuerierCallCount))
	})

	t.Run("GetQuerierFunc can return false", func(t *testing.T) {
		t.Parallel()

		mock := &MockTypeInspectorBuilder{
			GetQuerierFunc: func() (TypeInspectorPort, bool) {
				return nil, false
			},
		}

		querier, ok := mock.GetQuerier()

		assert.False(t, ok)
		assert.Nil(t, querier)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GetQuerierCallCount))
	})
}

func TestMockTypeInspectorBuilder_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var mock MockTypeInspectorBuilder

	mock.SetConfig(inspector_dto.Config{})
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.SetConfigCallCount))

	err := mock.Build(context.Background(), nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.BuildCallCount))

	querier, ok := mock.GetQuerier()
	assert.True(t, ok)
	assert.NotNil(t, querier)
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GetQuerierCallCount))
}

func TestMockTypeInspectorBuilder_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	const goroutines = 50

	mock := &MockTypeInspectorBuilder{
		SetConfigFunc: func(_ inspector_dto.Config) {},
		BuildFunc: func(_ context.Context, _ map[string][]byte, _ map[string]string) error {
			return nil
		},
		GetQuerierFunc: func() (TypeInspectorPort, bool) {
			return &inspector_domain.MockTypeQuerier{}, true
		},
	}

	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	for range goroutines {
		go func() {
			defer wg.Done()
			mock.SetConfig(inspector_dto.Config{BaseDir: "/test"})
		}()
		go func() {
			defer wg.Done()
			_ = mock.Build(context.Background(), nil, nil)
		}()
		go func() {
			defer wg.Done()
			_, _ = mock.GetQuerier()
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.SetConfigCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.BuildCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.GetQuerierCallCount))
}
