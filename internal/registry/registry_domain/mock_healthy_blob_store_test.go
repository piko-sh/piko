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
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

func TestMockHealthyBlobStore_Name(t *testing.T) {
	t.Parallel()

	t.Run("nil NameFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockHealthyBlobStore{}

		got := m.Name()

		assert.Equal(t, "", got)
		assert.Equal(t, int64(1), m.NameCallCount.Load())
	})

	t.Run("delegates to NameFunc", func(t *testing.T) {
		t.Parallel()
		m := &MockHealthyBlobStore{
			NameFunc: func() string {
				return "s3-backend"
			},
		}

		got := m.Name()

		assert.Equal(t, "s3-backend", got)
		assert.Equal(t, int64(1), m.NameCallCount.Load())
	})
}

func TestMockHealthyBlobStore_Check(t *testing.T) {
	t.Parallel()

	t.Run("nil CheckFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockHealthyBlobStore{}

		got := m.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

		assert.Equal(t, healthprobe_dto.Status{}, got)
		assert.Equal(t, int64(1), m.CheckCallCount.Load())
	})

	t.Run("delegates to CheckFunc", func(t *testing.T) {
		t.Parallel()
		want := healthprobe_dto.Status{Name: "s3-backend", State: healthprobe_dto.StateHealthy}
		m := &MockHealthyBlobStore{
			CheckFunc: func(_ context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status {
				assert.Equal(t, healthprobe_dto.CheckTypeLiveness, checkType)
				return want
			},
		}

		got := m.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

		assert.Equal(t, want, got)
		assert.Equal(t, int64(1), m.CheckCallCount.Load())
	})
}

func TestMockHealthyBlobStore_EmbeddedPut(t *testing.T) {
	t.Parallel()

	t.Run("nil PutFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockHealthyBlobStore{}

		err := m.Put(context.Background(), "key-1", strings.NewReader("data"))

		assert.NoError(t, err)
		assert.Equal(t, int64(1), m.PutCallCount.Load())
	})

	t.Run("delegates to PutFunc", func(t *testing.T) {
		t.Parallel()
		m := &MockHealthyBlobStore{}
		m.PutFunc = func(_ context.Context, key string, data io.Reader) error {
			assert.Equal(t, "key-1", key)
			return nil
		}

		err := m.Put(context.Background(), "key-1", strings.NewReader("data"))

		assert.NoError(t, err)
		assert.Equal(t, int64(1), m.PutCallCount.Load())
	})
}

func TestMockHealthyBlobStore_EmbeddedGet(t *testing.T) {
	t.Parallel()

	t.Run("nil GetFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockHealthyBlobStore{}

		got, err := m.Get(context.Background(), "key-1")

		assert.Nil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), m.GetCallCount.Load())
	})

	t.Run("delegates to GetFunc", func(t *testing.T) {
		t.Parallel()
		want := io.NopCloser(strings.NewReader("blob"))
		m := &MockHealthyBlobStore{}
		m.GetFunc = func(_ context.Context, key string) (io.ReadCloser, error) {
			assert.Equal(t, "key-1", key)
			return want, nil
		}

		got, err := m.Get(context.Background(), "key-1")

		require.NoError(t, err)
		assert.Equal(t, want, got)
	})
}

func TestMockHealthyBlobStore_EmbeddedRangeGet(t *testing.T) {
	t.Parallel()

	t.Run("nil RangeGetFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockHealthyBlobStore{}

		got, err := m.RangeGet(context.Background(), "key-1", 0, 100)

		assert.Nil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), m.RangeGetCallCount.Load())
	})

	t.Run("delegates to RangeGetFunc", func(t *testing.T) {
		t.Parallel()
		want := io.NopCloser(strings.NewReader("range"))
		m := &MockHealthyBlobStore{}
		m.RangeGetFunc = func(_ context.Context, key string, offset int64, length int64) (io.ReadCloser, error) {
			assert.Equal(t, "key-1", key)
			assert.Equal(t, int64(10), offset)
			assert.Equal(t, int64(50), length)
			return want, nil
		}

		got, err := m.RangeGet(context.Background(), "key-1", 10, 50)

		require.NoError(t, err)
		assert.Equal(t, want, got)
	})
}

func TestMockHealthyBlobStore_EmbeddedDelete(t *testing.T) {
	t.Parallel()

	t.Run("nil DeleteFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockHealthyBlobStore{}

		err := m.Delete(context.Background(), "key-1")

		assert.NoError(t, err)
		assert.Equal(t, int64(1), m.DeleteCallCount.Load())
	})

	t.Run("delegates to DeleteFunc", func(t *testing.T) {
		t.Parallel()
		m := &MockHealthyBlobStore{}
		m.DeleteFunc = func(_ context.Context, key string) error {
			assert.Equal(t, "key-1", key)
			return nil
		}

		err := m.Delete(context.Background(), "key-1")

		assert.NoError(t, err)
	})
}

func TestMockHealthyBlobStore_EmbeddedRename(t *testing.T) {
	t.Parallel()

	t.Run("nil RenameFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockHealthyBlobStore{}

		err := m.Rename(context.Background(), "temp", "final")

		assert.NoError(t, err)
		assert.Equal(t, int64(1), m.RenameCallCount.Load())
	})

	t.Run("delegates to RenameFunc", func(t *testing.T) {
		t.Parallel()
		m := &MockHealthyBlobStore{}
		m.RenameFunc = func(_ context.Context, tempKey string, key string) error {
			assert.Equal(t, "temp", tempKey)
			assert.Equal(t, "final", key)
			return nil
		}

		err := m.Rename(context.Background(), "temp", "final")

		assert.NoError(t, err)
	})
}

func TestMockHealthyBlobStore_EmbeddedExists(t *testing.T) {
	t.Parallel()

	t.Run("nil ExistsFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockHealthyBlobStore{}

		got, err := m.Exists(context.Background(), "key-1")

		assert.False(t, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), m.ExistsCallCount.Load())
	})

	t.Run("delegates to ExistsFunc", func(t *testing.T) {
		t.Parallel()
		m := &MockHealthyBlobStore{}
		m.ExistsFunc = func(_ context.Context, key string) (bool, error) {
			assert.Equal(t, "key-1", key)
			return true, nil
		}

		got, err := m.Exists(context.Background(), "key-1")

		require.NoError(t, err)
		assert.True(t, got)
	})
}

func TestMockHealthyBlobStore_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var m MockHealthyBlobStore
	ctx := context.Background()

	assert.Equal(t, "", m.Name())
	assert.Equal(t, healthprobe_dto.Status{}, m.Check(ctx, healthprobe_dto.CheckTypeLiveness))

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

func TestMockHealthyBlobStore_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	const goroutines = 50

	m := &MockHealthyBlobStore{
		NameFunc: func() string { return "backend" },
		CheckFunc: func(context.Context, healthprobe_dto.CheckType) healthprobe_dto.Status {
			return healthprobe_dto.Status{}
		},
	}
	m.PutFunc = func(context.Context, string, io.Reader) error { return nil }
	m.GetFunc = func(context.Context, string) (io.ReadCloser, error) { return nil, nil }
	m.RangeGetFunc = func(context.Context, string, int64, int64) (io.ReadCloser, error) { return nil, nil }
	m.DeleteFunc = func(context.Context, string) error { return nil }
	m.RenameFunc = func(context.Context, string, string) error { return nil }
	m.ExistsFunc = func(context.Context, string) (bool, error) { return false, nil }

	ctx := context.Background()
	var wg sync.WaitGroup

	for range goroutines {
		wg.Go(func() {
			_ = m.Name()
			_ = m.Check(ctx, healthprobe_dto.CheckTypeLiveness)
			_ = m.Put(ctx, "", nil)
			_, _ = m.Get(ctx, "")
			_, _ = m.RangeGet(ctx, "", 0, 0)
			_ = m.Delete(ctx, "")
			_ = m.Rename(ctx, "", "")
			_, _ = m.Exists(ctx, "")
		})
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), m.NameCallCount.Load())
	assert.Equal(t, int64(goroutines), m.CheckCallCount.Load())
	assert.Equal(t, int64(goroutines), m.PutCallCount.Load())
	assert.Equal(t, int64(goroutines), m.GetCallCount.Load())
	assert.Equal(t, int64(goroutines), m.RangeGetCallCount.Load())
	assert.Equal(t, int64(goroutines), m.DeleteCallCount.Load())
	assert.Equal(t, int64(goroutines), m.RenameCallCount.Load())
	assert.Equal(t, int64(goroutines), m.ExistsCallCount.Load())
}
