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

package i18n_domain

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockService_GetStore(t *testing.T) {
	t.Parallel()

	t.Run("nil GetStoreFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &MockService{
			GetStoreFunc:           nil,
			GetStrBufPoolFunc:      nil,
			DefaultLocaleFunc:      nil,
			GetStoreCallCount:      0,
			GetStrBufPoolCallCount: 0,
			DefaultLocaleCallCount: 0,
		}

		result := mock.GetStore()

		assert.Nil(t, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GetStoreCallCount))
	})

	t.Run("delegates to GetStoreFunc", func(t *testing.T) {
		t.Parallel()

		expectedStore := &Store{}

		mock := &MockService{
			GetStoreFunc: func() *Store {
				return expectedStore
			},
			GetStrBufPoolFunc:      nil,
			DefaultLocaleFunc:      nil,
			GetStoreCallCount:      0,
			GetStrBufPoolCallCount: 0,
			DefaultLocaleCallCount: 0,
		}

		result := mock.GetStore()

		require.NotNil(t, result)
		assert.Same(t, expectedStore, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GetStoreCallCount))
	})
}

func TestMockService_GetStrBufPool(t *testing.T) {
	t.Parallel()

	t.Run("nil GetStrBufPoolFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &MockService{
			GetStoreFunc:           nil,
			GetStrBufPoolFunc:      nil,
			DefaultLocaleFunc:      nil,
			GetStoreCallCount:      0,
			GetStrBufPoolCallCount: 0,
			DefaultLocaleCallCount: 0,
		}

		result := mock.GetStrBufPool()

		assert.Nil(t, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GetStrBufPoolCallCount))
	})

	t.Run("delegates to GetStrBufPoolFunc", func(t *testing.T) {
		t.Parallel()

		expectedPool := NewStrBufPool(64)

		mock := &MockService{
			GetStoreFunc: nil,
			GetStrBufPoolFunc: func() *StrBufPool {
				return expectedPool
			},
			DefaultLocaleFunc:      nil,
			GetStoreCallCount:      0,
			GetStrBufPoolCallCount: 0,
			DefaultLocaleCallCount: 0,
		}

		result := mock.GetStrBufPool()

		require.NotNil(t, result)
		assert.Same(t, expectedPool, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GetStrBufPoolCallCount))
	})
}

func TestMockService_DefaultLocale(t *testing.T) {
	t.Parallel()

	t.Run("nil DefaultLocaleFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &MockService{
			GetStoreFunc:           nil,
			GetStrBufPoolFunc:      nil,
			DefaultLocaleFunc:      nil,
			GetStoreCallCount:      0,
			GetStrBufPoolCallCount: 0,
			DefaultLocaleCallCount: 0,
		}

		result := mock.DefaultLocale()

		assert.Equal(t, "", result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.DefaultLocaleCallCount))
	})

	t.Run("delegates to DefaultLocaleFunc", func(t *testing.T) {
		t.Parallel()

		mock := &MockService{
			GetStoreFunc:      nil,
			GetStrBufPoolFunc: nil,
			DefaultLocaleFunc: func() string {
				return "en-GB"
			},
			GetStoreCallCount:      0,
			GetStrBufPoolCallCount: 0,
			DefaultLocaleCallCount: 0,
		}

		result := mock.DefaultLocale()

		assert.Equal(t, "en-GB", result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.DefaultLocaleCallCount))
	})
}

func TestMockService_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var mock MockService

	store := mock.GetStore()
	assert.Nil(t, store)

	pool := mock.GetStrBufPool()
	assert.Nil(t, pool)

	locale := mock.DefaultLocale()
	assert.Equal(t, "", locale)

	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GetStoreCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GetStrBufPoolCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.DefaultLocaleCallCount))
}

func TestMockService_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	mock := &MockService{
		GetStoreFunc:           nil,
		GetStrBufPoolFunc:      nil,
		DefaultLocaleFunc:      nil,
		GetStoreCallCount:      0,
		GetStrBufPoolCallCount: 0,
		DefaultLocaleCallCount: 0,
	}

	const goroutines = 50

	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	for range goroutines {
		go func() {
			defer wg.Done()
			_ = mock.GetStore()
		}()
		go func() {
			defer wg.Done()
			_ = mock.GetStrBufPool()
		}()
		go func() {
			defer wg.Done()
			_ = mock.DefaultLocale()
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.GetStoreCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.GetStrBufPoolCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.DefaultLocaleCallCount))
}
