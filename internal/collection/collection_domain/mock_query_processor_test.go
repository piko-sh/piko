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

	"piko.sh/piko/internal/search/search_domain"
	"piko.sh/piko/internal/search/search_dto"
)

func TestMockQueryProcessor_Search(t *testing.T) {
	t.Parallel()

	t.Run("nil SearchFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockQueryProcessor{}

		got, err := m.Search(context.Background(), "hello", nil, nil, search_dto.SearchConfig{})

		assert.NoError(t, err)
		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.SearchCallCount))
	})

	t.Run("delegates to SearchFunc", func(t *testing.T) {
		t.Parallel()
		expected := []search_domain.QueryResult{
			{DocumentID: 1, Score: 2.5, FieldScores: map[string]float64{"title": 2.5}},
		}
		m := &MockQueryProcessor{
			SearchFunc: func(_ context.Context, query string, _ search_domain.IndexReaderPort, _ search_domain.ScorerPort, _ search_dto.SearchConfig) ([]search_domain.QueryResult, error) {
				assert.Equal(t, "hello world", query)
				return expected, nil
			},
		}

		got, err := m.Search(context.Background(), "hello world", nil, nil, search_dto.SearchConfig{})

		require.NoError(t, err)
		assert.Equal(t, expected, got)
	})

	t.Run("propagates error from SearchFunc", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("search failed")
		m := &MockQueryProcessor{
			SearchFunc: func(_ context.Context, _ string, _ search_domain.IndexReaderPort, _ search_domain.ScorerPort, _ search_dto.SearchConfig) ([]search_domain.QueryResult, error) {
				return nil, wantErr
			},
		}

		_, err := m.Search(context.Background(), "q", nil, nil, search_dto.SearchConfig{})

		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockQueryProcessor_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	m := &MockQueryProcessor{}

	got, err := m.Search(context.Background(), "", nil, nil, search_dto.SearchConfig{})

	assert.NoError(t, err)
	assert.Nil(t, got)
}

func TestMockQueryProcessor_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	m := &MockQueryProcessor{
		SearchFunc: func(_ context.Context, _ string, _ search_domain.IndexReaderPort, _ search_domain.ScorerPort, _ search_dto.SearchConfig) ([]search_domain.QueryResult, error) {
			return nil, nil
		},
	}

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			_, _ = m.Search(context.Background(), "q", nil, nil, search_dto.SearchConfig{})
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.SearchCallCount))
}
