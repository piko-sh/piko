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
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockProviderRegistry_Register(t *testing.T) {
	t.Parallel()

	t.Run("nil RegisterFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockProviderRegistry{}

		err := m.Register(&MockCollectionProvider{})

		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RegisterCallCount))
	})

	t.Run("delegates to RegisterFunc", func(t *testing.T) {
		t.Parallel()
		var captured CollectionProvider
		m := &MockProviderRegistry{
			RegisterFunc: func(provider CollectionProvider) error {
				captured = provider
				return nil
			},
		}
		provider := &MockCollectionProvider{}

		err := m.Register(provider)

		require.NoError(t, err)
		assert.Equal(t, provider, captured)
	})

	t.Run("propagates error from RegisterFunc", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("duplicate registration")
		m := &MockProviderRegistry{
			RegisterFunc: func(_ CollectionProvider) error { return wantErr },
		}

		err := m.Register(&MockCollectionProvider{})

		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockProviderRegistry_Get(t *testing.T) {
	t.Parallel()

	t.Run("nil GetFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockProviderRegistry{}

		got, ok := m.Get("md")

		assert.Nil(t, got)
		assert.False(t, ok)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetCallCount))
	})

	t.Run("delegates to GetFunc", func(t *testing.T) {
		t.Parallel()
		expected := &MockCollectionProvider{}
		m := &MockProviderRegistry{
			GetFunc: func(name string) (CollectionProvider, bool) {
				if name == "md" {
					return expected, true
				}
				return nil, false
			},
		}

		got, ok := m.Get("md")

		assert.True(t, ok)
		assert.Equal(t, expected, got)
	})
}

func TestMockProviderRegistry_List(t *testing.T) {
	t.Parallel()

	t.Run("nil ListFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockProviderRegistry{}

		got := m.List()

		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ListCallCount))
	})

	t.Run("delegates to ListFunc", func(t *testing.T) {
		t.Parallel()
		m := &MockProviderRegistry{
			ListFunc: func() []string { return []string{"md", "api"} },
		}

		got := m.List()

		assert.Equal(t, []string{"md", "api"}, got)
	})
}

func TestMockProviderRegistry_Has(t *testing.T) {
	t.Parallel()

	t.Run("nil HasFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockProviderRegistry{}

		got := m.Has("md")

		assert.False(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.HasCallCount))
	})

	t.Run("delegates to HasFunc", func(t *testing.T) {
		t.Parallel()
		m := &MockProviderRegistry{
			HasFunc: func(name string) bool { return name == "md" },
		}

		assert.True(t, m.Has("md"))
		assert.False(t, m.Has("api"))
	})
}

func TestMockProviderRegistry_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	m := &MockProviderRegistry{}

	assert.NoError(t, m.Register(&MockCollectionProvider{}))

	got, ok := m.Get("md")
	assert.Nil(t, got)
	assert.False(t, ok)

	assert.Nil(t, m.List())
	assert.False(t, m.Has("md"))
}

func TestMockProviderRegistry_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	m := &MockProviderRegistry{
		RegisterFunc: func(_ CollectionProvider) error { return nil },
		GetFunc:      func(_ string) (CollectionProvider, bool) { return nil, false },
		ListFunc:     func() []string { return nil },
		HasFunc:      func(_ string) bool { return false },
	}

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			_ = m.Register(&MockCollectionProvider{})
			_, _ = m.Get("md")
			_ = m.List()
			_ = m.Has("md")
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.RegisterCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.ListCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.HasCallCount))
}
