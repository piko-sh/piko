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

func TestMockSearchIndexLoader_GetIndex(t *testing.T) {
	t.Parallel()

	t.Run("nil GetIndexFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockSearchIndexLoader{}

		got, err := m.GetIndex("blog", "fulltext")

		assert.NoError(t, err)
		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetIndexCallCount))
	})

	t.Run("delegates to GetIndexFunc", func(t *testing.T) {
		t.Parallel()
		sentinel := struct{ name string }{name: "test-index"}
		m := &MockSearchIndexLoader{
			GetIndexFunc: func(collectionName, searchMode string) (any, error) {
				assert.Equal(t, "blog", collectionName)
				assert.Equal(t, "fulltext", searchMode)
				return sentinel, nil
			},
		}

		got, err := m.GetIndex("blog", "fulltext")

		require.NoError(t, err)
		assert.Equal(t, sentinel, got)
	})

	t.Run("propagates error from GetIndexFunc", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("index not found")
		m := &MockSearchIndexLoader{
			GetIndexFunc: func(_, _ string) (any, error) { return nil, wantErr },
		}

		_, err := m.GetIndex("blog", "fulltext")

		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockSearchIndexLoader_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	m := &MockSearchIndexLoader{}

	got, err := m.GetIndex("c", "m")

	assert.NoError(t, err)
	assert.Nil(t, got)
}

func TestMockSearchIndexLoader_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	m := &MockSearchIndexLoader{
		GetIndexFunc: func(_, _ string) (any, error) { return nil, nil },
	}

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			_, _ = m.GetIndex("blog", "fulltext")
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetIndexCallCount))
}
