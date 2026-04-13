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

package cache_provider_firestore

import (
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/cache/cache_search"
	"piko.sh/piko/wdk/cache"
)

type testProduct struct {
	Name     string
	Category string
	Price    float64
	Summary  string
}

func newSearchSchema() *cache.SearchSchema {
	return cache.NewSearchSchema(
		cache.TagField("Category"),
		cache.SortableNumericField("Price"),
		cache.TextField("Summary"),
	)
}

func newSearchAdapter() *FirestoreAdapter[string, testProduct] {
	schema := newSearchSchema()
	adapter := &FirestoreAdapter[string, testProduct]{
		schema: schema,
	}
	configureSearchSchema(adapter, schema)
	return adapter
}

func TestSupportsSearch_WithSchema(t *testing.T) {
	t.Parallel()

	adapter := newSearchAdapter()
	assert.True(t, adapter.SupportsSearch(), "SupportsSearch should return true when schema is set")
}

func TestSupportsSearch_WithoutSchema(t *testing.T) {
	t.Parallel()

	adapter := &FirestoreAdapter[string, testProduct]{}
	assert.False(t, adapter.SupportsSearch(), "SupportsSearch should return false when schema is nil")
}

func TestGetSchema_ReturnsSchema(t *testing.T) {
	t.Parallel()

	adapter := newSearchAdapter()
	schema := adapter.GetSchema()
	require.NotNil(t, schema, "GetSchema should return non-nil schema")
	assert.Len(t, schema.Fields, 3, "Schema should have 3 fields")
}

func TestGetSchema_ReturnsNil(t *testing.T) {
	t.Parallel()

	adapter := &FirestoreAdapter[string, testProduct]{}
	assert.Nil(t, adapter.GetSchema(), "GetSchema should return nil when no schema is set")
}

func TestAddSearchFields_Tag(t *testing.T) {
	t.Parallel()

	adapter := newSearchAdapter()
	product := testProduct{
		Name:     "Widget",
		Category: "electronics",
		Price:    29.99,
		Summary:  "A fine widget",
	}

	data := make(map[string]any)
	adapter.addSearchFields(data, product)

	tagValue, ok := data["sf_Category"]
	require.True(t, ok, "sf_Category should be present")
	assert.Equal(t, "electronics", tagValue, "Tag field should contain the extracted value")
}

func TestAddSearchFields_Numeric(t *testing.T) {
	t.Parallel()

	adapter := newSearchAdapter()
	product := testProduct{
		Name:     "Widget",
		Category: "electronics",
		Price:    42.5,
		Summary:  "A fine widget",
	}

	data := make(map[string]any)
	adapter.addSearchFields(data, product)

	numericValue, ok := data["sf_Price"]
	require.True(t, ok, "sf_Price should be present")
	assert.Equal(t, 42.5, numericValue, "Numeric field should contain the extracted float64 value")
}

func TestAddSearchFields_Text(t *testing.T) {
	t.Parallel()

	adapter := newSearchAdapter()
	product := testProduct{
		Name:     "Widget",
		Category: "electronics",
		Price:    10.0,
		Summary:  "A brilliant test product",
	}

	data := make(map[string]any)
	adapter.addSearchFields(data, product)

	textValue, ok := data["sf_Summary"]
	require.True(t, ok, "sf_Summary should be present")
	assert.Equal(t, "A brilliant test product", textValue, "Text field should contain the joined text")
}

func TestApplyFilterToQuery_AllOperations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		filter    cache.Filter
		wantField string
	}{
		{"Eq", cache.Eq("Status", "active"), "sf_Status"},
		{"Ne", cache.Ne("Status", "deleted"), "sf_Status"},
		{"Gt", cache.Gt("Price", 100), "sf_Price"},
		{"Ge", cache.Ge("Price", 50), "sf_Price"},
		{"Lt", cache.Lt("Age", 18), "sf_Age"},
		{"Le", cache.Le("Score", 99), "sf_Score"},
		{"In", cache.In("Colour", "red", "blue"), "sf_Colour"},
		{"Between", cache.Between("Price", 10, 100), "sf_Price"},
		{"Prefix", cache.Prefix("Name", "pro"), "sf_Name"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.NotPanics(t, func() {
				query := firestore.Query{}
				applyFilterToQuery(&query, testCase.wantField, testCase.filter)
			}, "applyFilterToQuery should not panic for %s operation", testCase.name)
		})
	}
}

func TestResolveLimit(t *testing.T) {
	t.Parallel()

	assert.Equal(t, cache_search.DefaultSearchLimit, resolveLimit(0), "Zero limit should resolve to default")
	assert.Equal(t, cache_search.DefaultSearchLimit, resolveLimit(-1), "Negative limit should resolve to default")
	assert.Equal(t, 25, resolveLimit(25), "Positive limit should be returned as-is")
}

func TestSearch_ReturnsErrSearchNotSupported(t *testing.T) {
	t.Parallel()

	adapter := newSearchAdapter()
	_, err := adapter.Search(t.Context(), "test query", &cache.SearchOptions{})

	require.Error(t, err)
	assert.ErrorIs(t, err, cache.ErrSearchNotSupported,
		"Search should always return ErrSearchNotSupported")
}

func TestSearch_WithoutSchema_ReturnsErrSearchNotSupported(t *testing.T) {
	t.Parallel()

	adapter := &FirestoreAdapter[string, testProduct]{}
	_, err := adapter.Search(t.Context(), "test query", &cache.SearchOptions{})

	require.Error(t, err)
	assert.ErrorIs(t, err, cache.ErrSearchNotSupported,
		"Search without schema should return ErrSearchNotSupported")
}

func TestQuery_WithoutSchema_ReturnsErrSearchNotSupported(t *testing.T) {
	t.Parallel()

	adapter := &FirestoreAdapter[string, testProduct]{}
	_, err := adapter.Query(t.Context(), &cache.QueryOptions{})

	require.Error(t, err)
	assert.ErrorIs(t, err, cache.ErrSearchNotSupported,
		"Query without schema should return ErrSearchNotSupported")
}

func TestQuery_WithVector_ReturnsErrSearchNotSupported(t *testing.T) {
	t.Parallel()

	adapter := newSearchAdapter()
	_, err := adapter.Query(t.Context(), &cache.QueryOptions{
		Vector: []float32{0.1, 0.2, 0.3},
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, cache.ErrSearchNotSupported,
		"Query with vector should return ErrSearchNotSupported")
}
