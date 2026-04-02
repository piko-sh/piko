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

func TestMockCollectionItemsLoader_GetAllItems(t *testing.T) {
	t.Parallel()

	t.Run("nil GetAllItemsFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockCollectionItemsLoader{}

		got, err := m.GetAllItems("blog")

		assert.NoError(t, err)
		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetAllItemsCallCount))
	})

	t.Run("delegates to GetAllItemsFunc", func(t *testing.T) {
		t.Parallel()
		expected := []map[string]any{
			{"title": "Hello", "slug": "hello"},
			{"title": "World", "slug": "world"},
		}
		m := &MockCollectionItemsLoader{
			GetAllItemsFunc: func(collectionName string) ([]map[string]any, error) {
				assert.Equal(t, "blog", collectionName)
				return expected, nil
			},
		}

		got, err := m.GetAllItems("blog")

		require.NoError(t, err)
		assert.Equal(t, expected, got)
	})

	t.Run("propagates error from GetAllItemsFunc", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("collection not found")
		m := &MockCollectionItemsLoader{
			GetAllItemsFunc: func(_ string) ([]map[string]any, error) { return nil, wantErr },
		}

		_, err := m.GetAllItems("missing")

		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockCollectionItemsLoader_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	m := &MockCollectionItemsLoader{}

	got, err := m.GetAllItems("blog")

	assert.NoError(t, err)
	assert.Nil(t, got)
}

func TestMockCollectionItemsLoader_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	m := &MockCollectionItemsLoader{
		GetAllItemsFunc: func(_ string) ([]map[string]any, error) { return nil, nil },
	}

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			_, _ = m.GetAllItems("blog")
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetAllItemsCallCount))
}
