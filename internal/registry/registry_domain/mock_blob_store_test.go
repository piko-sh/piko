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

package registry_domain

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockBlobStore_Put(t *testing.T) {
	t.Parallel()

	t.Run("nil PutFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockBlobStore{}

		err := m.Put(context.Background(), "key-1", strings.NewReader("data"))

		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.PutCallCount))
	})

	t.Run("delegates to PutFunc", func(t *testing.T) {
		t.Parallel()
		m := &MockBlobStore{
			PutFunc: func(_ context.Context, key string, data io.Reader) error {
				assert.Equal(t, "key-1", key)
				assert.NotNil(t, data)
				return nil
			},
		}

		err := m.Put(context.Background(), "key-1", strings.NewReader("data"))

		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.PutCallCount))
	})

	t.Run("propagates error from PutFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("put failed")
		m := &MockBlobStore{
			PutFunc: func(context.Context, string, io.Reader) error {
				return expectedErr
			},
		}

		err := m.Put(context.Background(), "key-1", strings.NewReader("data"))

		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockBlobStore_Get(t *testing.T) {
	t.Parallel()

	t.Run("nil GetFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockBlobStore{}

		got, err := m.Get(context.Background(), "key-1")

		assert.Nil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetCallCount))
	})

	t.Run("delegates to GetFunc", func(t *testing.T) {
		t.Parallel()
		want := io.NopCloser(strings.NewReader("blob-data"))
		m := &MockBlobStore{
			GetFunc: func(_ context.Context, key string) (io.ReadCloser, error) {
				assert.Equal(t, "key-1", key)
				return want, nil
			},
		}

		got, err := m.Get(context.Background(), "key-1")

		require.NoError(t, err)
		assert.Equal(t, want, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetCallCount))
	})

	t.Run("propagates error from GetFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("get failed")
		m := &MockBlobStore{
			GetFunc: func(context.Context, string) (io.ReadCloser, error) {
				return nil, expectedErr
			},
		}

		got, err := m.Get(context.Background(), "key-1")

		assert.Nil(t, got)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockBlobStore_RangeGet(t *testing.T) {
	t.Parallel()

	t.Run("nil RangeGetFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockBlobStore{}

		got, err := m.RangeGet(context.Background(), "key-1", 0, 100)

		assert.Nil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RangeGetCallCount))
	})

	t.Run("delegates to RangeGetFunc", func(t *testing.T) {
		t.Parallel()
		want := io.NopCloser(strings.NewReader("range-data"))
		m := &MockBlobStore{
			RangeGetFunc: func(_ context.Context, key string, offset int64, length int64) (io.ReadCloser, error) {
				assert.Equal(t, "key-1", key)
				assert.Equal(t, int64(10), offset)
				assert.Equal(t, int64(50), length)
				return want, nil
			},
		}

		got, err := m.RangeGet(context.Background(), "key-1", 10, 50)

		require.NoError(t, err)
		assert.Equal(t, want, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RangeGetCallCount))
	})

	t.Run("propagates error from RangeGetFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("range get failed")
		m := &MockBlobStore{
			RangeGetFunc: func(context.Context, string, int64, int64) (io.ReadCloser, error) {
				return nil, expectedErr
			},
		}

		got, err := m.RangeGet(context.Background(), "key-1", 0, 100)

		assert.Nil(t, got)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockBlobStore_Delete(t *testing.T) {
	t.Parallel()

	t.Run("nil DeleteFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockBlobStore{}

		err := m.Delete(context.Background(), "key-1")

		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.DeleteCallCount))
	})

	t.Run("delegates to DeleteFunc", func(t *testing.T) {
		t.Parallel()
		m := &MockBlobStore{
			DeleteFunc: func(_ context.Context, key string) error {
				assert.Equal(t, "key-1", key)
				return nil
			},
		}

		err := m.Delete(context.Background(), "key-1")

		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.DeleteCallCount))
	})

	t.Run("propagates error from DeleteFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("delete failed")
		m := &MockBlobStore{
			DeleteFunc: func(context.Context, string) error {
				return expectedErr
			},
		}

		err := m.Delete(context.Background(), "key-1")

		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockBlobStore_Rename(t *testing.T) {
	t.Parallel()

	t.Run("nil RenameFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockBlobStore{}

		err := m.Rename(context.Background(), "temp-key", "final-key")

		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RenameCallCount))
	})

	t.Run("delegates to RenameFunc", func(t *testing.T) {
		t.Parallel()
		m := &MockBlobStore{
			RenameFunc: func(_ context.Context, tempKey string, key string) error {
				assert.Equal(t, "temp-key", tempKey)
				assert.Equal(t, "final-key", key)
				return nil
			},
		}

		err := m.Rename(context.Background(), "temp-key", "final-key")

		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RenameCallCount))
	})

	t.Run("propagates error from RenameFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("rename failed")
		m := &MockBlobStore{
			RenameFunc: func(context.Context, string, string) error {
				return expectedErr
			},
		}

		err := m.Rename(context.Background(), "temp-key", "final-key")

		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockBlobStore_Exists(t *testing.T) {
	t.Parallel()

	t.Run("nil ExistsFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockBlobStore{}

		got, err := m.Exists(context.Background(), "key-1")

		assert.False(t, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ExistsCallCount))
	})

	t.Run("delegates to ExistsFunc", func(t *testing.T) {
		t.Parallel()
		m := &MockBlobStore{
			ExistsFunc: func(_ context.Context, key string) (bool, error) {
				assert.Equal(t, "key-1", key)
				return true, nil
			},
		}

		got, err := m.Exists(context.Background(), "key-1")

		require.NoError(t, err)
		assert.True(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ExistsCallCount))
	})

	t.Run("propagates error from ExistsFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("exists failed")
		m := &MockBlobStore{
			ExistsFunc: func(context.Context, string) (bool, error) {
				return false, expectedErr
			},
		}

		got, err := m.Exists(context.Background(), "key-1")

		assert.False(t, got)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockBlobStore_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var m MockBlobStore
	ctx := context.Background()

	assert.NoError(t, m.Put(ctx, "", nil))

	got1, err := m.Get(ctx, "")
	assert.Nil(t, got1)
	assert.NoError(t, err)

	got2, err := m.RangeGet(ctx, "", 0, 0)
	assert.Nil(t, got2)
	assert.NoError(t, err)

	assert.NoError(t, m.Delete(ctx, ""))
	assert.NoError(t, m.Rename(ctx, "", ""))

	got3, err := m.Exists(ctx, "")
	assert.False(t, got3)
	assert.NoError(t, err)
}

func TestMockBlobStore_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	const goroutines = 50

	m := &MockBlobStore{
		PutFunc:      func(context.Context, string, io.Reader) error { return nil },
		GetFunc:      func(context.Context, string) (io.ReadCloser, error) { return nil, nil },
		RangeGetFunc: func(context.Context, string, int64, int64) (io.ReadCloser, error) { return nil, nil },
		DeleteFunc:   func(context.Context, string) error { return nil },
		RenameFunc:   func(context.Context, string, string) error { return nil },
		ExistsFunc:   func(context.Context, string) (bool, error) { return false, nil },
	}

	ctx := context.Background()
	var wg sync.WaitGroup

	for range goroutines {
		wg.Go(func() {
			_ = m.Put(ctx, "", nil)
			_, _ = m.Get(ctx, "")
			_, _ = m.RangeGet(ctx, "", 0, 0)
			_ = m.Delete(ctx, "")
			_ = m.Rename(ctx, "", "")
			_, _ = m.Exists(ctx, "")
		})
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.PutCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.RangeGetCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.DeleteCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.RenameCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.ExistsCallCount))
}
