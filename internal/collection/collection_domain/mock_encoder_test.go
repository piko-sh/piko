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

	"piko.sh/piko/internal/collection/collection_dto"
)

func TestMockEncoder_EncodeCollection(t *testing.T) {
	t.Parallel()

	t.Run("nil EncodeCollectionFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockEncoder{}

		got, err := m.EncodeCollection(nil)

		assert.NoError(t, err)
		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.EncodeCollectionCallCount))
	})

	t.Run("delegates to EncodeCollectionFunc", func(t *testing.T) {
		t.Parallel()
		items := []collection_dto.ContentItem{{Metadata: map[string]any{"title": "A"}}}
		expected := []byte("encoded-blob")
		m := &MockEncoder{
			EncodeCollectionFunc: func(input []collection_dto.ContentItem) ([]byte, error) {
				assert.Equal(t, items, input)
				return expected, nil
			},
		}

		got, err := m.EncodeCollection(items)

		require.NoError(t, err)
		assert.Equal(t, expected, got)
	})

	t.Run("propagates error from EncodeCollectionFunc", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("encoding failed")
		m := &MockEncoder{
			EncodeCollectionFunc: func(_ []collection_dto.ContentItem) ([]byte, error) {
				return nil, wantErr
			},
		}

		_, err := m.EncodeCollection(nil)

		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockEncoder_DecodeCollectionItem(t *testing.T) {
	t.Parallel()

	t.Run("nil DecodeCollectionItemFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockEncoder{}

		meta, content, excerpt, err := m.DecodeCollectionItem([]byte("blob"), "/blog/hello")

		assert.NoError(t, err)
		assert.Nil(t, meta)
		assert.Nil(t, content)
		assert.Nil(t, excerpt)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.DecodeCollectionItemCallCount))
	})

	t.Run("delegates to DecodeCollectionItemFunc", func(t *testing.T) {
		t.Parallel()
		wantMeta := []byte(`{"title":"Hello"}`)
		wantContent := []byte(`<p>Hello</p>`)
		wantExcerpt := []byte(`<p>Hello...</p>`)
		m := &MockEncoder{
			DecodeCollectionItemFunc: func(blob []byte, route string) ([]byte, []byte, []byte, error) {
				assert.Equal(t, []byte("blob"), blob)
				assert.Equal(t, "/blog/hello", route)
				return wantMeta, wantContent, wantExcerpt, nil
			},
		}

		meta, content, excerpt, err := m.DecodeCollectionItem([]byte("blob"), "/blog/hello")

		require.NoError(t, err)
		assert.Equal(t, wantMeta, meta)
		assert.Equal(t, wantContent, content)
		assert.Equal(t, wantExcerpt, excerpt)
	})

	t.Run("propagates error from DecodeCollectionItemFunc", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("decoding failed")
		m := &MockEncoder{
			DecodeCollectionItemFunc: func(_ []byte, _ string) ([]byte, []byte, []byte, error) {
				return nil, nil, nil, wantErr
			},
		}

		_, _, _, err := m.DecodeCollectionItem(nil, "")

		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockEncoder_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	m := &MockEncoder{}

	blob, err := m.EncodeCollection(nil)
	assert.NoError(t, err)
	assert.Nil(t, blob)

	meta, content, excerpt, err := m.DecodeCollectionItem(nil, "")
	assert.NoError(t, err)
	assert.Nil(t, meta)
	assert.Nil(t, content)
	assert.Nil(t, excerpt)
}

func TestMockEncoder_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	m := &MockEncoder{
		EncodeCollectionFunc:     func(_ []collection_dto.ContentItem) ([]byte, error) { return nil, nil },
		DecodeCollectionItemFunc: func(_ []byte, _ string) ([]byte, []byte, []byte, error) { return nil, nil, nil, nil },
	}

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			_, _ = m.EncodeCollection(nil)
			_, _, _, _ = m.DecodeCollectionItem(nil, "")
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.EncodeCollectionCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.DecodeCollectionItemCallCount))
}
