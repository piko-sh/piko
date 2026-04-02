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

package lifecycle_domain

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/lifecycle/lifecycle_dto"
)

func TestMockFileSystemWatcher_Watch(t *testing.T) {
	t.Parallel()

	t.Run("nil WatchFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockFileSystemWatcher{}

		events, err := m.Watch(context.Background(), []string{"/src"}, []string{"/assets"})
		assert.NoError(t, err)
		assert.Nil(t, events)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.WatchCallCount))
	})

	t.Run("delegates to WatchFunc", func(t *testing.T) {
		t.Parallel()

		fileEventChannel := make(chan lifecycle_dto.FileEvent, 1)
		m := &MockFileSystemWatcher{
			WatchFunc: func(_ context.Context, recursive []string, nonRecursive []string) (<-chan lifecycle_dto.FileEvent, error) {
				assert.Equal(t, []string{"/src"}, recursive)
				assert.Equal(t, []string{"/assets"}, nonRecursive)
				return fileEventChannel, nil
			},
		}

		events, err := m.Watch(context.Background(), []string{"/src"}, []string{"/assets"})
		require.NoError(t, err)
		assert.Equal(t, (<-chan lifecycle_dto.FileEvent)(fileEventChannel), events)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.WatchCallCount))
	})

	t.Run("propagates error from WatchFunc", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("watcher initialisation failed")
		m := &MockFileSystemWatcher{
			WatchFunc: func(_ context.Context, _ []string, _ []string) (<-chan lifecycle_dto.FileEvent, error) {
				return nil, expectedErr
			},
		}

		events, err := m.Watch(context.Background(), nil, nil)
		assert.Nil(t, events)
		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.WatchCallCount))
	})
}

func TestMockFileSystemWatcher_UpdateWatchedFiles(t *testing.T) {
	t.Parallel()

	t.Run("nil UpdateWatchedFilesFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockFileSystemWatcher{}

		err := m.UpdateWatchedFiles(context.Background(), []string{"/a.txt", "/b.txt"})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.UpdateWatchedFilesCallCount))
	})

	t.Run("delegates to UpdateWatchedFilesFunc", func(t *testing.T) {
		t.Parallel()

		m := &MockFileSystemWatcher{
			UpdateWatchedFilesFunc: func(_ context.Context, files []string) error {
				assert.Equal(t, []string{"/a.txt", "/b.txt"}, files)
				return nil
			},
		}

		err := m.UpdateWatchedFiles(context.Background(), []string{"/a.txt", "/b.txt"})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.UpdateWatchedFilesCallCount))
	})

	t.Run("propagates error from UpdateWatchedFilesFunc", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("update failed")
		m := &MockFileSystemWatcher{
			UpdateWatchedFilesFunc: func(_ context.Context, _ []string) error {
				return expectedErr
			},
		}

		err := m.UpdateWatchedFiles(context.Background(), nil)
		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.UpdateWatchedFilesCallCount))
	})
}

func TestMockFileSystemWatcher_Close(t *testing.T) {
	t.Parallel()

	t.Run("nil CloseFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockFileSystemWatcher{}

		err := m.Close()
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.CloseCallCount))
	})

	t.Run("delegates to CloseFunc", func(t *testing.T) {
		t.Parallel()

		called := false
		m := &MockFileSystemWatcher{
			CloseFunc: func() error {
				called = true
				return nil
			},
		}

		err := m.Close()
		assert.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.CloseCallCount))
	})

	t.Run("propagates error from CloseFunc", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("close failed")
		m := &MockFileSystemWatcher{
			CloseFunc: func() error {
				return expectedErr
			},
		}

		err := m.Close()
		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.CloseCallCount))
	})
}

func TestMockFileSystemWatcher_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	m := &MockFileSystemWatcher{}

	events, err := m.Watch(context.Background(), nil, nil)
	assert.NoError(t, err)
	assert.Nil(t, events)

	err = m.UpdateWatchedFiles(context.Background(), nil)
	assert.NoError(t, err)

	err = m.Close()
	assert.NoError(t, err)

	assert.Equal(t, int64(1), atomic.LoadInt64(&m.WatchCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&m.UpdateWatchedFilesCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&m.CloseCallCount))
}

func TestMockFileSystemWatcher_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	const goroutines = 50

	m := &MockFileSystemWatcher{
		WatchFunc: func(_ context.Context, _ []string, _ []string) (<-chan lifecycle_dto.FileEvent, error) {
			return nil, nil
		},
		UpdateWatchedFilesFunc: func(_ context.Context, _ []string) error {
			return nil
		},
		CloseFunc: func() error {
			return nil
		},
	}

	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	for range goroutines {
		go func() {
			defer wg.Done()
			_, _ = m.Watch(context.Background(), nil, nil)
		}()
		go func() {
			defer wg.Done()
			_ = m.UpdateWatchedFiles(context.Background(), nil)
		}()
		go func() {
			defer wg.Done()
			_ = m.Close()
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.WatchCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.UpdateWatchedFilesCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.CloseCallCount))
}
