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

// MockQueryProcessor is a test double for search_domain.QueryProcessorPort
// that returns zero values from nil function fields and tracks call counts
// atomically.
type MockQueryProcessor struct {
	// SearchFunc is the function called by Search.
	SearchFunc func(ctx context.Context, query string, reader search_domain.IndexReaderPort, scorer search_domain.ScorerPort, config search_dto.SearchConfig) ([]search_domain.QueryResult, error)

	// SearchCallCount tracks how many times Search was
	// called.
	SearchCallCount int64
}

var _ search_domain.QueryProcessorPort = (*MockQueryProcessor)(nil)

// Search delegates to SearchFunc if set.
//
// Takes ctx (context.Context) which carries deadlines and cancellation
// signals.
// Takes query (string) which is the search query string.
// Takes reader (search_domain.IndexReaderPort) which provides access
// to the search index.
// Takes scorer (search_domain.ScorerPort) which computes relevance scores.
// Takes config (search_dto.SearchConfig) which provides search configuration.
//
// Returns (nil, nil) if SearchFunc is nil.
func (m *MockQueryProcessor) Search(
	ctx context.Context, query string, reader search_domain.IndexReaderPort,
	scorer search_domain.ScorerPort, config search_dto.SearchConfig,
) ([]search_domain.QueryResult, error) {
	atomic.AddInt64(&m.SearchCallCount, 1)
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, query, reader, scorer, config)
	}
	return nil, nil
}
