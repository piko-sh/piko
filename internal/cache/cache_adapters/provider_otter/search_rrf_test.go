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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/cache/cache_dto"
)

func newRRFTestAdapter(t *testing.T) *OtterAdapter[string, Product] {
	t.Helper()

	schema := cache_dto.NewSearchSchema(
		cache_dto.TextField("Name"),
		cache_dto.TextField("Description"),
		cache_dto.VectorField("Tags", 3),
	)

	options := cache_dto.Options[string, Product]{
		MaximumSize:  100,
		SearchSchema: schema,
	}

	adapter, err := OtterProviderFactory[string, Product](options)
	require.NoError(t, err, "failed to create adapter")
	t.Cleanup(func() { _ = adapter.Close(context.Background()) })

	otter := requireOtterAdapter(t, adapter)
	return otter
}

func TestRRFFusion_BothListsCombine(t *testing.T) {
	adapter := newRRFTestAdapter(t)
	ctx := context.Background()

	_ = adapter.Set(ctx, "doc1", Product{ID: "1", Name: "Premium Widget", Description: "High quality"})
	_ = adapter.Set(ctx, "doc2", Product{ID: "2", Name: "Basic Gadget", Description: "Simple gadget"})
	_ = adapter.Set(ctx, "doc3", Product{ID: "3", Name: "Premium Device", Description: "Professional"})

	vectorHits := []VectorHit[string]{
		{Key: "doc1", Score: 0.95},
		{Key: "doc2", Score: 0.85},
	}
	textScored := []ScoredResult[string]{
		{Key: "doc1", Score: 2.5},
		{Key: "doc3", Score: 1.8},
	}

	result, err := adapter.rrfFusion(vectorHits, textScored, nil, 0, 10)
	require.NoError(t, err)

	assert.Equal(t, int64(3), result.Total, "should have 3 unique keys from union")

	require.True(t, len(result.Items) >= 1)
	assert.Equal(t, "doc1", result.Items[0].Key, "doc1 should be ranked first (in both lists)")

	if len(result.Items) > 1 {
		assert.Greater(t, result.Items[0].Score, result.Items[1].Score,
			"doc1 score should be higher than items in only one list")
	}
}

func TestRRFFusion_ItemInBothListsScoresHigher(t *testing.T) {
	adapter := newRRFTestAdapter(t)
	ctx := context.Background()

	_ = adapter.Set(ctx, "both", Product{ID: "both", Name: "both item"})
	_ = adapter.Set(ctx, "vec_only", Product{ID: "vec_only", Name: "vector only"})
	_ = adapter.Set(ctx, "text_only", Product{ID: "text_only", Name: "text only"})

	vectorHits := []VectorHit[string]{
		{Key: "both", Score: 0.9},
		{Key: "vec_only", Score: 0.8},
	}
	textScored := []ScoredResult[string]{
		{Key: "both", Score: 2.0},
		{Key: "text_only", Score: 1.5},
	}

	result, err := adapter.rrfFusion(vectorHits, textScored, nil, 0, 10)
	require.NoError(t, err)

	assert.Equal(t, int64(3), result.Total)

	require.True(t, len(result.Items) >= 1)
	assert.Equal(t, "both", result.Items[0].Key)

	keys := make(map[string]bool)
	for _, item := range result.Items {
		keys[item.Key] = true
	}
	assert.True(t, keys["vec_only"], "vec_only should be in results")
	assert.True(t, keys["text_only"], "text_only should be in results")
}

func TestRRFFusion_PureVectorNoText(t *testing.T) {
	adapter := newRRFTestAdapter(t)
	ctx := context.Background()

	_ = adapter.Set(ctx, "doc1", Product{ID: "1", Name: "Widget"})
	_ = adapter.Set(ctx, "doc2", Product{ID: "2", Name: "Gadget"})

	vectorHits := []VectorHit[string]{
		{Key: "doc1", Score: 0.95},
		{Key: "doc2", Score: 0.85},
	}
	var textScored []ScoredResult[string]

	result, err := adapter.rrfFusion(vectorHits, textScored, nil, 0, 10)
	require.NoError(t, err)

	assert.Equal(t, int64(2), result.Total)
	assert.Len(t, result.Items, 2)
}

func TestRRFFusion_PureTextNoVector(t *testing.T) {
	adapter := newRRFTestAdapter(t)
	ctx := context.Background()

	_ = adapter.Set(ctx, "doc1", Product{ID: "1", Name: "Widget"})
	_ = adapter.Set(ctx, "doc2", Product{ID: "2", Name: "Gadget"})

	var vectorHits []VectorHit[string]
	textScored := []ScoredResult[string]{
		{Key: "doc1", Score: 2.5},
		{Key: "doc2", Score: 1.8},
	}

	result, err := adapter.rrfFusion(vectorHits, textScored, nil, 0, 10)
	require.NoError(t, err)

	assert.Equal(t, int64(2), result.Total)
	assert.Len(t, result.Items, 2)
}

func TestRRFFusion_Pagination(t *testing.T) {
	adapter := newRRFTestAdapter(t)
	ctx := context.Background()

	for i := range 5 {
		key := string(rune('a' + i))
		_ = adapter.Set(ctx, key, Product{ID: key, Name: "Item " + key})
	}

	vectorHits := []VectorHit[string]{
		{Key: "a", Score: 0.9},
		{Key: "b", Score: 0.8},
		{Key: "c", Score: 0.7},
		{Key: "d", Score: 0.6},
		{Key: "e", Score: 0.5},
	}

	result, err := adapter.rrfFusion(vectorHits, nil, nil, 1, 2)
	require.NoError(t, err)

	assert.Equal(t, int64(5), result.Total, "total should reflect all items before pagination")
	assert.Len(t, result.Items, 2, "should return 2 items after offset 1, limit 2")
}

func TestRRFFusion_WithFilters(t *testing.T) {
	schema := cache_dto.NewSearchSchema(
		cache_dto.TextField("Name"),
		cache_dto.TagField("Category"),
	)

	options := cache_dto.Options[string, Product]{
		MaximumSize:  100,
		SearchSchema: schema,
	}

	adapter, err := OtterProviderFactory[string, Product](options)
	require.NoError(t, err)
	ctx := context.Background()
	t.Cleanup(func() { _ = adapter.Close(ctx) })
	otter := requireOtterAdapter(t, adapter)

	_ = otter.Set(ctx, "doc1", Product{ID: "1", Name: "Widget", Category: "electronics"})
	_ = otter.Set(ctx, "doc2", Product{ID: "2", Name: "Gadget", Category: "toys"})
	_ = otter.Set(ctx, "doc3", Product{ID: "3", Name: "Device", Category: "electronics"})

	vectorHits := []VectorHit[string]{
		{Key: "doc1", Score: 0.9},
		{Key: "doc2", Score: 0.8},
		{Key: "doc3", Score: 0.7},
	}

	filters := []cache_dto.Filter{
		cache_dto.Eq("Category", "electronics"),
	}

	result, err := otter.rrfFusion(vectorHits, nil, filters, 0, 10)
	require.NoError(t, err)

	assert.Equal(t, int64(2), result.Total, "only electronics items should pass filter")
	for _, item := range result.Items {
		assert.Equal(t, "electronics", item.Value.Category,
			"all results should be in electronics category")
	}
}

func TestRRFFusion_ScoreCalculation(t *testing.T) {
	adapter := newRRFTestAdapter(t)
	ctx := context.Background()

	_ = adapter.Set(ctx, "doc1", Product{ID: "1", Name: "First"})

	vectorHits := []VectorHit[string]{
		{Key: "doc1", Score: 0.9},
	}
	textScored := []ScoredResult[string]{
		{Key: "doc1", Score: 2.0},
	}

	result, err := adapter.rrfFusion(vectorHits, textScored, nil, 0, 10)
	require.NoError(t, err)

	require.Len(t, result.Items, 1)

	expectedScore := 2.0 / float64(rrfK+1)
	assert.InDelta(t, expectedScore, result.Items[0].Score, 0.0001,
		"RRF score should be the sum of reciprocal ranks from both lists")
}

func TestHybridSearch_VectorPlusText(t *testing.T) {
	schema := cache_dto.NewSearchSchema(
		cache_dto.TextField("Name"),
		cache_dto.VectorField("Tags", 3),
	)

	options := cache_dto.Options[string, Product]{
		MaximumSize:  100,
		SearchSchema: schema,
	}

	adapter, err := OtterProviderFactory[string, Product](options)
	require.NoError(t, err)
	ctx := context.Background()
	t.Cleanup(func() { _ = adapter.Close(ctx) })
	otter := requireOtterAdapter(t, adapter)

	products := []struct {
		key  string
		prod Product
	}{
		{key: "doc1", prod: Product{ID: "1", Name: "Premium Widget"}},
		{key: "doc2", prod: Product{ID: "2", Name: "Basic Gadget"}},
		{key: "doc3", prod: Product{ID: "3", Name: "Premium Device"}},
	}

	for _, p := range products {
		_ = otter.Set(ctx, p.key, p.prod)
	}

	if vIndex, ok := otter.vectorIndexes["Tags"]; ok {
		vIndex.Add("doc1", []float32{1, 0, 0})
		vIndex.Add("doc2", []float32{0, 1, 0})
		vIndex.Add("doc3", []float32{0.9, 0.1, 0})
	}

	result, err := otter.Search(ctx, "premium", &cache_dto.SearchOptions{
		Vector: []float32{1, 0, 0},
		Limit:  10,
	})
	require.NoError(t, err)

	assert.True(t, len(result.Items) > 0, "should have results from hybrid search")

	if len(result.Items) > 0 {
		foundDoc1 := false
		for _, item := range result.Items {
			if item.Key == "doc1" {
				foundDoc1 = true
				break
			}
		}
		assert.True(t, foundDoc1, "doc1 should be in hybrid results (matches both vector and text)")
	}
}

func TestHybridSearch_PureVectorFallback(t *testing.T) {
	schema := cache_dto.NewSearchSchema(
		cache_dto.TextField("Name"),
		cache_dto.VectorField("Tags", 3),
	)

	options := cache_dto.Options[string, Product]{
		MaximumSize:  100,
		SearchSchema: schema,
	}

	adapter, err := OtterProviderFactory[string, Product](options)
	require.NoError(t, err)
	ctx := context.Background()
	t.Cleanup(func() { _ = adapter.Close(ctx) })
	otter := requireOtterAdapter(t, adapter)

	_ = otter.Set(ctx, "doc1", Product{ID: "1", Name: "Widget"})
	_ = otter.Set(ctx, "doc2", Product{ID: "2", Name: "Gadget"})

	if vIndex, ok := otter.vectorIndexes["Tags"]; ok {
		vIndex.Add("doc1", []float32{1, 0, 0})
		vIndex.Add("doc2", []float32{0, 1, 0})
	}

	result, err := otter.Search(ctx, "", &cache_dto.SearchOptions{
		Vector: []float32{1, 0, 0},
		Limit:  10,
	})
	require.NoError(t, err)

	assert.True(t, len(result.Items) > 0, "should have results from pure vector search")
}

func TestHybridSearch_TextNoVectorMatch(t *testing.T) {
	schema := cache_dto.NewSearchSchema(
		cache_dto.TextField("Name"),
		cache_dto.VectorField("Tags", 3),
	)

	options := cache_dto.Options[string, Product]{
		MaximumSize:  100,
		SearchSchema: schema,
	}

	adapter, err := OtterProviderFactory[string, Product](options)
	require.NoError(t, err)
	ctx := context.Background()
	t.Cleanup(func() { _ = adapter.Close(ctx) })
	otter := requireOtterAdapter(t, adapter)

	_ = otter.Set(ctx, "doc1", Product{ID: "1", Name: "Widget"})

	if vIndex, ok := otter.vectorIndexes["Tags"]; ok {
		vIndex.Add("doc1", []float32{1, 0, 0})
	}

	result, err := otter.Search(ctx, "nonexistent", &cache_dto.SearchOptions{
		Vector: []float32{1, 0, 0},
		Limit:  10,
	})
	require.NoError(t, err)

	assert.True(t, len(result.Items) > 0, "should still have vector results even without text matches")
}
