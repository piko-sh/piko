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

package search_domain_test

import (
	"context"
	"fmt"
	"testing"

	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/search/search_adapters"
	"piko.sh/piko/internal/search/search_domain"
	"piko.sh/piko/internal/search/search_schema/search_schema_gen"
)

func buildTestIndex(
	t *testing.T,
	docs []collection_dto.ContentItem,
	mode search_schema_gen.SearchMode,
	config search_domain.IndexBuildConfig,
) search_domain.IndexReaderPort {
	t.Helper()

	builder := search_domain.NewIndexBuilder()

	indexData, err := builder.BuildIndex(
		context.Background(),
		"test",
		docs,
		mode,
		config,
	)
	if err != nil {
		t.Fatalf("Failed to build index: %v", err)
	}

	reader := search_adapters.NewFlatBufferIndexReader()
	if err := reader.LoadIndex(indexData); err != nil {
		t.Fatalf("Failed to load index: %v", err)
	}

	return reader
}

func createTestDocuments(count int, prefix string) []collection_dto.ContentItem {
	docs := make([]collection_dto.ContentItem, count)
	for i := range count {
		docs[i] = collection_dto.ContentItem{
			URL: fmt.Sprintf("/doc%d", i),
			Metadata: map[string]any{
				"title": fmt.Sprintf("%s Document %d", prefix, i),
			},
			RawContent: fmt.Sprintf("Content for document %d with %s keywords", i, prefix),
		}
	}
	return docs
}

func createTestDocWithMetadata(url string, metadata map[string]any, rawContent string) collection_dto.ContentItem {
	return collection_dto.ContentItem{
		URL:        url,
		Metadata:   metadata,
		RawContent: rawContent,
	}
}
