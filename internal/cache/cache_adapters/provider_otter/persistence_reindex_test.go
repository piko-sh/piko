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

package provider_otter

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/wal/wal_domain"
)

type testVecDocument struct {
	Title     string    `json:"title"`
	Category  string    `json:"category"`
	Embedding []float32 `json:"embedding"`
	Rating    float64   `json:"rating"`
}

type jsonVecDocCodec struct{}

func (jsonVecDocCodec) EncodeValue(value testVecDocument) ([]byte, error) {
	return json.Marshal(value)
}

func (jsonVecDocCodec) DecodeValue(data []byte) (testVecDocument, error) {
	var v testVecDocument
	err := json.Unmarshal(data, &v)
	return v, err
}

func newVecPersistenceConfig(directory string) PersistenceConfig[string, testVecDocument] {
	walConfig := wal_domain.DefaultConfig(directory)
	walConfig.SyncMode = wal_domain.SyncModeEveryWrite
	walConfig.EnableCompression = false

	return PersistenceConfig[string, testVecDocument]{
		Enabled:    true,
		WALConfig:  walConfig,
		KeyCodec:   stringKeyCodec{},
		ValueCodec: jsonVecDocCodec{},
	}
}

func TestReindex_TextSearch(t *testing.T) {
	directory := t.TempDir()

	schema := cache_dto.NewSearchSchema(
		cache_dto.TextField("Title"),
		cache_dto.TextField("Content"),
	)

	opts := cache_dto.Options[string, testArticle]{
		MaximumSize:      100,
		SearchSchema:     schema,
		ProviderSpecific: newPersistenceConfig(directory),
	}

	adapter, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	ctx := context.Background()

	_ = adapter.Set(ctx, "a1", testArticle{Title: "Premium Widget", Content: "High quality widget", Views: 100})
	_ = adapter.Set(ctx, "a2", testArticle{Title: "Basic Gadget", Content: "Simple gadget tool", Views: 200})
	_ = adapter.Set(ctx, "a3", testArticle{Title: "Premium Device", Content: "Professional premium device", Views: 50})

	_ = adapter.Close(ctx)

	adapter2, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	defer func() { _ = adapter2.Close(ctx) }()

	tests := []struct {
		name        string
		query       string
		expectedIDs []string
	}{
		{
			name:        "single term match",
			query:       "premium",
			expectedIDs: []string{"a1", "a3"},
		},
		{
			name:        "multi term match",
			query:       "premium widget",
			expectedIDs: []string{"a1"},
		},
		{
			name:        "match in content only",
			query:       "professional",
			expectedIDs: []string{"a3"},
		},
		{
			name:        "no match",
			query:       "nonexistent",
			expectedIDs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := adapter2.Search(ctx, tt.query, &cache_dto.SearchOptions{Limit: 10})
			require.NoError(t, err)

			assert.Equal(t, int64(len(tt.expectedIDs)), result.Total)

			foundIDs := make(map[string]bool, len(result.Items))
			for _, item := range result.Items {
				foundIDs[item.Key] = true
			}
			for _, id := range tt.expectedIDs {
				assert.True(t, foundIDs[id], "expected to find %s in results", id)
			}
		})
	}
}

func TestReindex_BM25ScoreOrdering(t *testing.T) {
	directory := t.TempDir()

	analyser := func(text string) []string {
		return splitWords(text)
	}

	schema := cache_dto.NewSearchSchemaWithAnalyser(
		analyser,
		cache_dto.TextField("Title"),
	)

	opts := cache_dto.Options[string, testArticle]{
		MaximumSize:      100,
		SearchSchema:     schema,
		ProviderSpecific: newPersistenceConfig(directory),
	}

	adapter, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	ctx := context.Background()

	_ = adapter.Set(ctx, "short", testArticle{Title: "widget"})
	_ = adapter.Set(ctx, "long", testArticle{Title: "widget gadget device tool item product accessory kit"})

	_ = adapter.Close(ctx)

	adapter2, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	defer func() { _ = adapter2.Close(ctx) }()

	result, err := adapter2.Search(ctx, "widget", &cache_dto.SearchOptions{Limit: 10})
	require.NoError(t, err)
	require.Equal(t, int64(2), result.Total)
	require.Len(t, result.Items, 2)

	assert.Equal(t, "short", result.Items[0].Key)
	assert.Greater(t, result.Items[0].Score, result.Items[1].Score)
}

func TestReindex_CustomTextAnalyser(t *testing.T) {
	directory := t.TempDir()

	mockAnalyser := func(text string) []string {
		words := splitWords(text)
		var result []string
		seen := make(map[string]struct{})
		for _, w := range words {
			if len(w) > 4 && w[len(w)-3:] == "ing" {
				w = w[:len(w)-3]
			}
			if _, ok := seen[w]; !ok {
				seen[w] = struct{}{}
				result = append(result, w)
			}
		}
		return result
	}

	schema := cache_dto.NewSearchSchemaWithAnalyser(
		mockAnalyser,
		cache_dto.TextField("Title"),
		cache_dto.TextField("Content"),
	)

	opts := cache_dto.Options[string, testArticle]{
		MaximumSize:      100,
		SearchSchema:     schema,
		ProviderSpecific: newPersistenceConfig(directory),
	}

	adapter, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	ctx := context.Background()

	_ = adapter.Set(ctx, "a1", testArticle{Title: "Running Shoes", Content: "Fast running shoes"})
	_ = adapter.Set(ctx, "a2", testArticle{Title: "Walking Boots", Content: "Comfortable walking boots"})
	_ = adapter.Set(ctx, "a3", testArticle{Title: "Swim Goggles", Content: "Professional swim equipment"})

	_ = adapter.Close(ctx)

	adapter2, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	defer func() { _ = adapter2.Close(ctx) }()

	result, err := adapter2.Search(ctx, "running", &cache_dto.SearchOptions{Limit: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), result.Total)
	assert.Equal(t, "a1", result.Items[0].Key)
	assert.Greater(t, result.Items[0].Score, 0.0, "expected positive BM25 score")
}

func TestReindex_SortedIndex(t *testing.T) {
	directory := t.TempDir()

	schema := cache_dto.NewSearchSchema(
		cache_dto.TextField("Title"),
		cache_dto.SortableNumericField("Views"),
	)

	opts := cache_dto.Options[string, testArticle]{
		MaximumSize:      100,
		SearchSchema:     schema,
		ProviderSpecific: newPersistenceConfig(directory),
	}

	adapter, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	ctx := context.Background()

	_ = adapter.Set(ctx, "a1", testArticle{Title: "Widget A", Views: 300})
	_ = adapter.Set(ctx, "a2", testArticle{Title: "Widget B", Views: 100})
	_ = adapter.Set(ctx, "a3", testArticle{Title: "Widget C", Views: 200})

	_ = adapter.Close(ctx)

	adapter2, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	defer func() { _ = adapter2.Close(ctx) }()

	result, err := adapter2.Search(ctx, "widget", &cache_dto.SearchOptions{
		Limit:     10,
		SortBy:    "Views",
		SortOrder: cache_dto.SortAsc,
	})
	require.NoError(t, err)
	require.Len(t, result.Items, 3)

	assert.Equal(t, "a2", result.Items[0].Key)
	assert.Equal(t, "a3", result.Items[1].Key)
	assert.Equal(t, "a1", result.Items[2].Key)

	result, err = adapter2.Search(ctx, "widget", &cache_dto.SearchOptions{
		Limit:     10,
		SortBy:    "Views",
		SortOrder: cache_dto.SortDesc,
	})
	require.NoError(t, err)
	require.Len(t, result.Items, 3)

	assert.Equal(t, "a1", result.Items[0].Key)
	assert.Equal(t, "a3", result.Items[1].Key)
	assert.Equal(t, "a2", result.Items[2].Key)
}

func TestReindex_FilterQueries(t *testing.T) {
	directory := t.TempDir()

	schema := cache_dto.NewSearchSchema(
		cache_dto.TextField("Title"),
		cache_dto.TextField("Content"),
		cache_dto.SortableNumericField("Views"),
	)

	opts := cache_dto.Options[string, testArticle]{
		MaximumSize:      100,
		SearchSchema:     schema,
		ProviderSpecific: newPersistenceConfig(directory),
	}

	adapter, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	ctx := context.Background()

	_ = adapter.Set(ctx, "a1", testArticle{Title: "Widget A", Content: "Cheap widget", Views: 50})
	_ = adapter.Set(ctx, "a2", testArticle{Title: "Widget B", Content: "Mid widget", Views: 150})
	_ = adapter.Set(ctx, "a3", testArticle{Title: "Widget C", Content: "Expensive widget", Views: 300})
	_ = adapter.Set(ctx, "a4", testArticle{Title: "Widget D", Content: "Budget widget", Views: 80})

	_ = adapter.Close(ctx)

	adapter2, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	defer func() { _ = adapter2.Close(ctx) }()

	result, err := adapter2.Search(ctx, "widget", &cache_dto.SearchOptions{
		Limit:   10,
		Filters: []cache_dto.Filter{cache_dto.Gt("Views", 100)},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), result.Total)

	foundIDs := make(map[string]bool)
	for _, item := range result.Items {
		foundIDs[item.Key] = true
	}
	assert.True(t, foundIDs["a2"])
	assert.True(t, foundIDs["a3"])

	result, err = adapter2.Search(ctx, "widget", &cache_dto.SearchOptions{
		Limit:   10,
		Filters: []cache_dto.Filter{cache_dto.Between("Views", 50, 150)},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(3), result.Total)
}

func TestReindex_InvalidatedEntryNotSearchable(t *testing.T) {
	directory := t.TempDir()

	schema := cache_dto.NewSearchSchema(
		cache_dto.TextField("Title"),
	)

	opts := cache_dto.Options[string, testArticle]{
		MaximumSize:      100,
		SearchSchema:     schema,
		ProviderSpecific: newPersistenceConfig(directory),
	}

	adapter, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	ctx := context.Background()

	_ = adapter.Set(ctx, "a1", testArticle{Title: "Widget Alpha"})
	_ = adapter.Set(ctx, "a2", testArticle{Title: "Widget Beta"})
	_ = adapter.Invalidate(ctx, "a1")

	_ = adapter.Close(ctx)

	adapter2, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	defer func() { _ = adapter2.Close(ctx) }()

	result, err := adapter2.Search(ctx, "widget", &cache_dto.SearchOptions{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
	require.Len(t, result.Items, 1)
	assert.Equal(t, "a2", result.Items[0].Key)
}

func TestReindex_UpdatedEntryReflectsNewTerms(t *testing.T) {
	directory := t.TempDir()

	schema := cache_dto.NewSearchSchema(
		cache_dto.TextField("Title"),
	)

	opts := cache_dto.Options[string, testArticle]{
		MaximumSize:      100,
		SearchSchema:     schema,
		ProviderSpecific: newPersistenceConfig(directory),
	}

	adapter, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	ctx := context.Background()

	_ = adapter.Set(ctx, "a1", testArticle{Title: "Widget Alpha"})

	_ = adapter.Set(ctx, "a1", testArticle{Title: "Gadget Alpha"})

	_ = adapter.Close(ctx)

	adapter2, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	defer func() { _ = adapter2.Close(ctx) }()

	result, err := adapter2.Search(ctx, "widget", &cache_dto.SearchOptions{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(0), result.Total, "old term 'widget' should not match after update")

	result, err = adapter2.Search(ctx, "gadget", &cache_dto.SearchOptions{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total, "new term 'gadget' should match after update")
	require.Len(t, result.Items, 1)
	assert.Equal(t, "a1", result.Items[0].Key)
}

func TestReindex_ClearBeforeRestartEmptiesSearch(t *testing.T) {
	directory := t.TempDir()

	schema := cache_dto.NewSearchSchema(
		cache_dto.TextField("Title"),
	)

	opts := cache_dto.Options[string, testArticle]{
		MaximumSize:      100,
		SearchSchema:     schema,
		ProviderSpecific: newPersistenceConfig(directory),
	}

	adapter, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	ctx := context.Background()

	_ = adapter.Set(ctx, "a1", testArticle{Title: "Widget Alpha"})
	_ = adapter.Set(ctx, "a2", testArticle{Title: "Widget Beta"})
	_ = adapter.InvalidateAll(ctx)

	_ = adapter.Close(ctx)

	adapter2, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	defer func() { _ = adapter2.Close(ctx) }()

	result, err := adapter2.Search(ctx, "widget", &cache_dto.SearchOptions{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(0), result.Total)
	assert.Empty(t, result.Items)
}

func TestReindex_ClearThenAddBeforeRestart(t *testing.T) {
	directory := t.TempDir()

	schema := cache_dto.NewSearchSchema(
		cache_dto.TextField("Title"),
	)

	opts := cache_dto.Options[string, testArticle]{
		MaximumSize:      100,
		SearchSchema:     schema,
		ProviderSpecific: newPersistenceConfig(directory),
	}

	adapter, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	ctx := context.Background()

	_ = adapter.Set(ctx, "a1", testArticle{Title: "Widget Alpha"})
	_ = adapter.Set(ctx, "a2", testArticle{Title: "Widget Beta"})
	_ = adapter.InvalidateAll(ctx)
	_ = adapter.Set(ctx, "a3", testArticle{Title: "Gadget Gamma"})

	_ = adapter.Close(ctx)

	adapter2, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	defer func() { _ = adapter2.Close(ctx) }()

	result, err := adapter2.Search(ctx, "widget", &cache_dto.SearchOptions{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(0), result.Total, "cleared entries should not be searchable")

	result, err = adapter2.Search(ctx, "gadget", &cache_dto.SearchOptions{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
	require.Len(t, result.Items, 1)
	assert.Equal(t, "a3", result.Items[0].Key)
}

func TestReindex_VectorIndex(t *testing.T) {
	directory := t.TempDir()

	schema := cache_dto.NewSearchSchema(
		cache_dto.TextField("Title"),
		cache_dto.VectorField("Embedding", 3),
	)

	opts := cache_dto.Options[string, testVecDocument]{
		MaximumSize:      100,
		SearchSchema:     schema,
		ProviderSpecific: newVecPersistenceConfig(directory),
	}

	adapter, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	ctx := context.Background()

	_ = adapter.Set(ctx, "v1", testVecDocument{Title: "Doc A", Embedding: []float32{1, 0, 0}})
	_ = adapter.Set(ctx, "v2", testVecDocument{Title: "Doc B", Embedding: []float32{0, 1, 0}})
	_ = adapter.Set(ctx, "v3", testVecDocument{Title: "Doc C", Embedding: []float32{0.9, 0.1, 0}})

	_ = adapter.Close(ctx)

	adapter2, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	defer func() { _ = adapter2.Close(ctx) }()

	result, err := adapter2.Search(ctx, "", &cache_dto.SearchOptions{
		Vector: []float32{1, 0, 0},
		Limit:  10,
		TopK:   3,
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(result.Items), 2)

	assert.Equal(t, "v1", result.Items[0].Key, "exact vector match should rank first")
	assert.Equal(t, "v3", result.Items[1].Key, "close vector should rank second")
}

func TestReindex_QueryWithSortedIndex(t *testing.T) {
	directory := t.TempDir()

	schema := cache_dto.NewSearchSchema(
		cache_dto.TagField("Content"),
		cache_dto.SortableNumericField("Views"),
	)

	opts := cache_dto.Options[string, testArticle]{
		MaximumSize:      100,
		SearchSchema:     schema,
		ProviderSpecific: newPersistenceConfig(directory),
	}

	adapter, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	ctx := context.Background()

	_ = adapter.Set(ctx, "a1", testArticle{Content: "tech", Views: 300})
	_ = adapter.Set(ctx, "a2", testArticle{Content: "tech", Views: 100})
	_ = adapter.Set(ctx, "a3", testArticle{Content: "science", Views: 200})
	_ = adapter.Set(ctx, "a4", testArticle{Content: "tech", Views: 150})

	_ = adapter.Close(ctx)

	adapter2, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	defer func() { _ = adapter2.Close(ctx) }()

	result, err := adapter2.Query(ctx, &cache_dto.QueryOptions{
		Filters:   []cache_dto.Filter{cache_dto.Eq("Content", "tech")},
		SortBy:    "Views",
		SortOrder: cache_dto.SortAsc,
		Limit:     10,
	})
	require.NoError(t, err)
	require.Len(t, result.Items, 3)

	assert.Equal(t, "a2", result.Items[0].Key)
	assert.Equal(t, "a4", result.Items[1].Key)
	assert.Equal(t, "a1", result.Items[2].Key)
}

func TestReindex_MultipleRestarts(t *testing.T) {
	directory := t.TempDir()

	schema := cache_dto.NewSearchSchema(
		cache_dto.TextField("Title"),
	)

	opts := cache_dto.Options[string, testArticle]{
		MaximumSize:      100,
		SearchSchema:     schema,
		ProviderSpecific: newPersistenceConfig(directory),
	}

	ctx := context.Background()

	a1, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	_ = a1.Set(ctx, "a1", testArticle{Title: "Widget Alpha"})
	_ = a1.Set(ctx, "a2", testArticle{Title: "Gadget Beta"})
	_ = a1.Close(ctx)

	a2, err := OtterProviderFactory(opts)
	require.NoError(t, err)

	result, err := a2.Search(ctx, "widget", &cache_dto.SearchOptions{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)

	_ = a2.Set(ctx, "a3", testArticle{Title: "Widget Gamma"})
	_ = a2.Close(ctx)

	a3, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	defer func() { _ = a3.Close(ctx) }()

	result, err = a3.Search(ctx, "widget", &cache_dto.SearchOptions{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(2), result.Total)

	foundIDs := make(map[string]bool)
	for _, item := range result.Items {
		foundIDs[item.Key] = true
	}
	assert.True(t, foundIDs["a1"])
	assert.True(t, foundIDs["a3"])
}
