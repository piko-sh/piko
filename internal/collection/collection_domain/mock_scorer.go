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
	"sync/atomic"

	"piko.sh/piko/internal/search/search_domain"
	"piko.sh/piko/internal/search/search_dto"
)

// MockScorer is a test double for search_domain.ScorerPort that returns zero
// values from nil function fields and tracks call counts atomically.
type MockScorer struct {
	// ScoreFunc is the function called by Score.
	ScoreFunc func(ctx context.Context, queryTerms []string, documentID uint32, reader search_domain.IndexReaderPort, config search_dto.SearchConfig) (search_domain.ScoreResult, error)

	// ScoreCallCount tracks how many times Score was
	// called.
	ScoreCallCount int64
}

var _ search_domain.ScorerPort = (*MockScorer)(nil)

// Score delegates to ScoreFunc if set.
//
// Takes ctx (context.Context) which carries deadlines and cancellation
// signals.
// Takes queryTerms ([]string) which is the list of query terms to
// score against.
// Takes documentID (uint32) which identifies the document to score.
// Takes reader (search_domain.IndexReaderPort) which provides access
// to the search index.
// Takes config (search_dto.SearchConfig) which provides search configuration.
//
// Returns (ScoreResult{}, nil) if ScoreFunc is nil.
func (m *MockScorer) Score(ctx context.Context, queryTerms []string, documentID uint32, reader search_domain.IndexReaderPort, config search_dto.SearchConfig) (search_domain.ScoreResult, error) {
	atomic.AddInt64(&m.ScoreCallCount, 1)
	if m.ScoreFunc != nil {
		return m.ScoreFunc(ctx, queryTerms, documentID, reader, config)
	}
	return search_domain.ScoreResult{}, nil
}
