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

func TestMockScorer_Score(t *testing.T) {
	t.Parallel()

	t.Run("nil ScoreFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockScorer{}

		got, err := m.Score(context.Background(), []string{"hello"}, 1, nil, search_dto.SearchConfig{})

		assert.NoError(t, err)
		assert.Equal(t, search_domain.ScoreResult{}, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ScoreCallCount))
	})

	t.Run("delegates to ScoreFunc", func(t *testing.T) {
		t.Parallel()
		expected := search_domain.ScoreResult{
			FieldScores: map[string]float64{"title": 1.5},
		}
		m := &MockScorer{
			ScoreFunc: func(_ context.Context, queryTerms []string, documentID uint32, _ search_domain.IndexReaderPort, _ search_dto.SearchConfig) (search_domain.ScoreResult, error) {
				assert.Equal(t, []string{"hello", "world"}, queryTerms)
				assert.Equal(t, uint32(42), documentID)
				return expected, nil
			},
		}

		got, err := m.Score(context.Background(), []string{"hello", "world"}, 42, nil, search_dto.SearchConfig{})

		require.NoError(t, err)
		assert.Equal(t, expected, got)
	})

	t.Run("propagates error from ScoreFunc", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("scoring failed")
		m := &MockScorer{
			ScoreFunc: func(_ context.Context, _ []string, _ uint32, _ search_domain.IndexReaderPort, _ search_dto.SearchConfig) (search_domain.ScoreResult, error) {
				return search_domain.ScoreResult{}, wantErr
			},
		}

		_, err := m.Score(context.Background(), []string{"q"}, 1, nil, search_dto.SearchConfig{})

		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockScorer_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	m := &MockScorer{}

	got, err := m.Score(context.Background(), nil, 0, nil, search_dto.SearchConfig{})

	assert.NoError(t, err)
	assert.Equal(t, search_domain.ScoreResult{}, got)
}

func TestMockScorer_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	m := &MockScorer{
		ScoreFunc: func(_ context.Context, _ []string, _ uint32, _ search_domain.IndexReaderPort, _ search_dto.SearchConfig) (search_domain.ScoreResult, error) {
			return search_domain.ScoreResult{}, nil
		},
	}

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			_, _ = m.Score(context.Background(), []string{"q"}, 1, nil, search_dto.SearchConfig{})
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.ScoreCallCount))
}
