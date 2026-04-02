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
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/internal/collection/collection_dto"
)

func TestMockHybridRegistry_Register(t *testing.T) {
	t.Parallel()

	t.Run("nil RegisterFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockHybridRegistry{}

		m.Register(context.Background(), "md", "blog", []byte("data"), "etag1", collection_dto.HybridConfig{})

		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RegisterCallCount))
	})

	t.Run("delegates to RegisterFunc", func(t *testing.T) {
		t.Parallel()
		var capturedProvider, capturedCollection string
		var capturedBlob []byte
		var capturedETag string
		m := &MockHybridRegistry{
			RegisterFunc: func(_ context.Context, providerName, collectionName string, blob []byte, etag string, _ collection_dto.HybridConfig) {
				capturedProvider = providerName
				capturedCollection = collectionName
				capturedBlob = blob
				capturedETag = etag
			},
		}

		m.Register(context.Background(), "md", "blog", []byte("payload"), "etag-abc", collection_dto.HybridConfig{})

		assert.Equal(t, "md", capturedProvider)
		assert.Equal(t, "blog", capturedCollection)
		assert.Equal(t, []byte("payload"), capturedBlob)
		assert.Equal(t, "etag-abc", capturedETag)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RegisterCallCount))
	})
}

func TestMockHybridRegistry_GetBlob(t *testing.T) {
	t.Parallel()

	t.Run("nil GetBlobFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockHybridRegistry{}

		blob, needsRevalidation := m.GetBlob(context.Background(), "md", "blog")

		assert.Nil(t, blob)
		assert.False(t, needsRevalidation)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetBlobCallCount))
	})

	t.Run("delegates to GetBlobFunc", func(t *testing.T) {
		t.Parallel()
		m := &MockHybridRegistry{
			GetBlobFunc: func(_ context.Context, providerName, collectionName string) ([]byte, bool) {
				return []byte("cached-" + providerName + "-" + collectionName), true
			},
		}

		blob, needsRevalidation := m.GetBlob(context.Background(), "md", "blog")

		assert.Equal(t, []byte("cached-md-blog"), blob)
		assert.True(t, needsRevalidation)
	})
}

func TestMockHybridRegistry_GetETag(t *testing.T) {
	t.Parallel()

	t.Run("nil GetETagFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockHybridRegistry{}

		got := m.GetETag("md", "blog")

		assert.Equal(t, "", got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetETagCallCount))
	})

	t.Run("delegates to GetETagFunc", func(t *testing.T) {
		t.Parallel()
		m := &MockHybridRegistry{
			GetETagFunc: func(_, _ string) string { return "etag-123" },
		}

		got := m.GetETag("md", "blog")

		assert.Equal(t, "etag-123", got)
	})
}

func TestMockHybridRegistry_Has(t *testing.T) {
	t.Parallel()

	t.Run("nil HasFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockHybridRegistry{}

		got := m.Has("md", "blog")

		assert.False(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.HasCallCount))
	})

	t.Run("delegates to HasFunc", func(t *testing.T) {
		t.Parallel()
		m := &MockHybridRegistry{
			HasFunc: func(_, _ string) bool { return true },
		}

		got := m.Has("md", "blog")

		assert.True(t, got)
	})
}

func TestMockHybridRegistry_List(t *testing.T) {
	t.Parallel()

	t.Run("nil ListFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockHybridRegistry{}

		got := m.List()

		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ListCallCount))
	})

	t.Run("delegates to ListFunc", func(t *testing.T) {
		t.Parallel()
		m := &MockHybridRegistry{
			ListFunc: func() []string { return []string{"md:blog", "md:docs"} },
		}

		got := m.List()

		assert.Equal(t, []string{"md:blog", "md:docs"}, got)
	})
}

func TestMockHybridRegistry_TriggerRevalidation(t *testing.T) {
	t.Parallel()

	t.Run("nil TriggerRevalidationFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockHybridRegistry{}

		m.TriggerRevalidation(context.Background(), "md", "blog")

		assert.Equal(t, int64(1), atomic.LoadInt64(&m.TriggerRevalidationCallCount))
	})

	t.Run("delegates to TriggerRevalidationFunc", func(t *testing.T) {
		t.Parallel()
		var capturedProvider, capturedCollection string
		m := &MockHybridRegistry{
			TriggerRevalidationFunc: func(_ context.Context, providerName, collectionName string) {
				capturedProvider = providerName
				capturedCollection = collectionName
			},
		}

		m.TriggerRevalidation(context.Background(), "md", "blog")

		assert.Equal(t, "md", capturedProvider)
		assert.Equal(t, "blog", capturedCollection)
	})
}

func TestMockHybridRegistry_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	m := &MockHybridRegistry{}

	m.Register(context.Background(), "md", "blog", nil, "", collection_dto.HybridConfig{})

	blob, needsRevalidation := m.GetBlob(context.Background(), "md", "blog")
	assert.Nil(t, blob)
	assert.False(t, needsRevalidation)

	assert.Equal(t, "", m.GetETag("md", "blog"))
	assert.False(t, m.Has("md", "blog"))
	assert.Nil(t, m.List())

	m.TriggerRevalidation(context.Background(), "md", "blog")
}

func TestMockHybridRegistry_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	m := &MockHybridRegistry{
		RegisterFunc:            func(_ context.Context, _, _ string, _ []byte, _ string, _ collection_dto.HybridConfig) {},
		GetBlobFunc:             func(_ context.Context, _, _ string) ([]byte, bool) { return nil, false },
		GetETagFunc:             func(_, _ string) string { return "" },
		HasFunc:                 func(_, _ string) bool { return false },
		ListFunc:                func() []string { return nil },
		TriggerRevalidationFunc: func(_ context.Context, _, _ string) {},
	}

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			m.Register(context.Background(), "md", "blog", nil, "", collection_dto.HybridConfig{})
			_, _ = m.GetBlob(context.Background(), "md", "blog")
			_ = m.GetETag("md", "blog")
			_ = m.Has("md", "blog")
			_ = m.List()
			m.TriggerRevalidation(context.Background(), "md", "blog")
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.RegisterCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetBlobCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetETagCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.HasCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.ListCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.TriggerRevalidationCallCount))
}
