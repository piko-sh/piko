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

package collection_domain

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/collection/collection_dto"
)

func TestMockRuntimeProviderRegistry_Register(t *testing.T) {
	t.Parallel()

	t.Run("nil RegisterFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockRuntimeProviderRegistry{}

		err := m.Register(&MockRuntimeProvider{})

		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RegisterCallCount))
	})

	t.Run("delegates to RegisterFunc", func(t *testing.T) {
		t.Parallel()
		var captured RuntimeProvider
		m := &MockRuntimeProviderRegistry{
			RegisterFunc: func(provider RuntimeProvider) error {
				captured = provider
				return nil
			},
		}
		provider := &MockRuntimeProvider{}

		err := m.Register(provider)

		require.NoError(t, err)
		assert.Equal(t, provider, captured)
	})

	t.Run("propagates error from RegisterFunc", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("duplicate provider")
		m := &MockRuntimeProviderRegistry{
			RegisterFunc: func(_ RuntimeProvider) error { return wantErr },
		}

		err := m.Register(&MockRuntimeProvider{})

		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockRuntimeProviderRegistry_Get(t *testing.T) {
	t.Parallel()

	t.Run("nil GetFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockRuntimeProviderRegistry{}

		got, err := m.Get("md")

		assert.NoError(t, err)
		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetCallCount))
	})

	t.Run("delegates to GetFunc", func(t *testing.T) {
		t.Parallel()
		expected := &MockRuntimeProvider{NameFunc: func() string { return "md" }}
		m := &MockRuntimeProviderRegistry{
			GetFunc: func(name string) (RuntimeProvider, error) {
				assert.Equal(t, "md", name)
				return expected, nil
			},
		}

		got, err := m.Get("md")

		require.NoError(t, err)
		assert.Equal(t, expected, got)
	})

	t.Run("propagates error from GetFunc", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("not found")
		m := &MockRuntimeProviderRegistry{
			GetFunc: func(_ string) (RuntimeProvider, error) { return nil, wantErr },
		}

		_, err := m.Get("missing")

		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockRuntimeProviderRegistry_List(t *testing.T) {
	t.Parallel()

	t.Run("nil ListFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockRuntimeProviderRegistry{}

		got := m.List()

		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ListCallCount))
	})

	t.Run("delegates to ListFunc", func(t *testing.T) {
		t.Parallel()
		m := &MockRuntimeProviderRegistry{
			ListFunc: func() []string { return []string{"md", "api"} },
		}

		got := m.List()

		assert.Equal(t, []string{"md", "api"}, got)
	})
}

func TestMockRuntimeProviderRegistry_Has(t *testing.T) {
	t.Parallel()

	t.Run("nil HasFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockRuntimeProviderRegistry{}

		got := m.Has("md")

		assert.False(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.HasCallCount))
	})

	t.Run("delegates to HasFunc", func(t *testing.T) {
		t.Parallel()
		m := &MockRuntimeProviderRegistry{
			HasFunc: func(name string) bool { return name == "md" },
		}

		assert.True(t, m.Has("md"))
		assert.False(t, m.Has("api"))
	})
}

func TestMockRuntimeProviderRegistry_Fetch(t *testing.T) {
	t.Parallel()

	t.Run("nil FetchFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockRuntimeProviderRegistry{}

		err := m.Fetch(context.Background(), "md", "blog", nil, nil)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.FetchCallCount))
	})

	t.Run("delegates to FetchFunc", func(t *testing.T) {
		t.Parallel()
		var capturedProvider, capturedCollection string
		m := &MockRuntimeProviderRegistry{
			FetchFunc: func(_ context.Context, providerName, collectionName string, _ *collection_dto.FetchOptions, _ any) error {
				capturedProvider = providerName
				capturedCollection = collectionName
				return nil
			},
		}

		err := m.Fetch(context.Background(), "md", "blog", nil, nil)

		require.NoError(t, err)
		assert.Equal(t, "md", capturedProvider)
		assert.Equal(t, "blog", capturedCollection)
	})

	t.Run("propagates error from FetchFunc", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("fetch failed")
		m := &MockRuntimeProviderRegistry{
			FetchFunc: func(_ context.Context, _, _ string, _ *collection_dto.FetchOptions, _ any) error {
				return wantErr
			},
		}

		err := m.Fetch(context.Background(), "md", "blog", nil, nil)

		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockRuntimeProviderRegistry_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	m := &MockRuntimeProviderRegistry{}

	assert.NoError(t, m.Register(&MockRuntimeProvider{}))

	got, err := m.Get("md")
	assert.NoError(t, err)
	assert.Nil(t, got)

	assert.Nil(t, m.List())
	assert.False(t, m.Has("md"))
	assert.NoError(t, m.Fetch(context.Background(), "md", "blog", nil, nil))
}

func TestMockRuntimeProviderRegistry_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	m := &MockRuntimeProviderRegistry{
		RegisterFunc: func(_ RuntimeProvider) error { return nil },
		GetFunc:      func(_ string) (RuntimeProvider, error) { return nil, nil },
		ListFunc:     func() []string { return nil },
		HasFunc:      func(_ string) bool { return false },
		FetchFunc:    func(_ context.Context, _, _ string, _ *collection_dto.FetchOptions, _ any) error { return nil },
	}

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			_ = m.Register(&MockRuntimeProvider{})
			_, _ = m.Get("md")
			_ = m.List()
			_ = m.Has("md")
			_ = m.Fetch(context.Background(), "md", "blog", nil, nil)
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.RegisterCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.ListCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.HasCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.FetchCallCount))
}
