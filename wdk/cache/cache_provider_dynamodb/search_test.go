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

package cache_provider_dynamodb

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/cache"
)

type testProduct struct {
	Name     string
	Category string
	Price    float64
	Count    int
}

func newSearchTestAdapter() *DynamoDBAdapter[string, testProduct] {
	schema := &cache.SearchSchema{
		Fields: []cache.FieldSchema{
			{Name: "Name", Type: cache.FieldTypeText},
			{Name: "Category", Type: cache.FieldTypeTag, Sortable: true},
			{Name: "Price", Type: cache.FieldTypeNumeric, Sortable: true},
			{Name: "Count", Type: cache.FieldTypeNumeric},
		},
	}

	adapter := &DynamoDBAdapter[string, testProduct]{
		schema: schema,
	}
	configureSearchSchema(adapter, schema)
	return adapter
}

func newNoSchemaAdapter() *DynamoDBAdapter[string, testProduct] {
	return &DynamoDBAdapter[string, testProduct]{}
}

func TestBuildFilterExpression_Eq(t *testing.T) {
	filters := []cache.Filter{
		{Field: "Category", Operation: cache.FilterOpEq, Value: "electronics"},
	}

	expression, names, values := buildFilterExpression(filters)

	assert.Equal(t, "#sf_0 = :sf_0_0", expression)
	assert.Equal(t, "sf_Category", names["#sf_0"])

	valAttr, ok := values[":sf_0_0"].(*types.AttributeValueMemberS)
	require.True(t, ok)
	assert.Equal(t, "electronics", valAttr.Value)
}

func TestBuildFilterExpression_Ne(t *testing.T) {
	filters := []cache.Filter{
		{Field: "Category", Operation: cache.FilterOpNe, Value: "books"},
	}

	expression, names, values := buildFilterExpression(filters)

	assert.Equal(t, "#sf_0 <> :sf_0_0", expression)
	assert.Equal(t, "sf_Category", names["#sf_0"])

	valAttr, ok := values[":sf_0_0"].(*types.AttributeValueMemberS)
	require.True(t, ok)
	assert.Equal(t, "books", valAttr.Value)
}

func TestBuildFilterExpression_Gt(t *testing.T) {
	filters := []cache.Filter{
		{Field: "Price", Operation: cache.FilterOpGt, Value: 10.0},
	}

	expression, names, values := buildFilterExpression(filters)

	assert.Equal(t, "#sf_0 > :sf_0_0", expression)
	assert.Equal(t, "sf_Price", names["#sf_0"])

	valAttr, ok := values[":sf_0_0"].(*types.AttributeValueMemberN)
	require.True(t, ok)
	assert.Equal(t, "10", valAttr.Value)
}

func TestBuildFilterExpression_Ge(t *testing.T) {
	filters := []cache.Filter{
		{Field: "Price", Operation: cache.FilterOpGe, Value: 5},
	}

	expression, _, _ := buildFilterExpression(filters)
	assert.Equal(t, "#sf_0 >= :sf_0_0", expression)
}

func TestBuildFilterExpression_Lt(t *testing.T) {
	filters := []cache.Filter{
		{Field: "Price", Operation: cache.FilterOpLt, Value: 100},
	}

	expression, _, _ := buildFilterExpression(filters)
	assert.Equal(t, "#sf_0 < :sf_0_0", expression)
}

func TestBuildFilterExpression_Le(t *testing.T) {
	filters := []cache.Filter{
		{Field: "Count", Operation: cache.FilterOpLe, Value: 50},
	}

	expression, _, _ := buildFilterExpression(filters)
	assert.Equal(t, "#sf_0 <= :sf_0_0", expression)
}

func TestBuildFilterExpression_In(t *testing.T) {
	filters := []cache.Filter{
		{Field: "Category", Operation: cache.FilterOpIn, Values: []any{"a", "b", "c"}},
	}

	expression, names, values := buildFilterExpression(filters)

	assert.Equal(t, "#sf_0 IN (:sf_0_0, :sf_0_1, :sf_0_2)", expression)
	assert.Equal(t, "sf_Category", names["#sf_0"])
	assert.Len(t, values, 3)

	for _, alias := range []string{":sf_0_0", ":sf_0_1", ":sf_0_2"} {
		_, ok := values[alias].(*types.AttributeValueMemberS)
		assert.True(t, ok, "expected string attribute for %s", alias)
	}
}

func TestBuildFilterExpression_Between(t *testing.T) {
	filters := []cache.Filter{
		{Field: "Price", Operation: cache.FilterOpBetween, Values: []any{10, 50}},
	}

	expression, names, values := buildFilterExpression(filters)

	assert.Equal(t, "#sf_0 BETWEEN :sf_0_lo AND :sf_0_hi", expression)
	assert.Equal(t, "sf_Price", names["#sf_0"])

	loAttr, ok := values[":sf_0_lo"].(*types.AttributeValueMemberN)
	require.True(t, ok)
	assert.Equal(t, "10", loAttr.Value)

	hiAttr, ok := values[":sf_0_hi"].(*types.AttributeValueMemberN)
	require.True(t, ok)
	assert.Equal(t, "50", hiAttr.Value)
}

func TestBuildFilterExpression_Prefix(t *testing.T) {
	filters := []cache.Filter{
		{Field: "Name", Operation: cache.FilterOpPrefix, Value: "wire"},
	}

	expression, names, values := buildFilterExpression(filters)

	assert.Equal(t, "begins_with(#sf_0, :sf_0_0)", expression)
	assert.Equal(t, "sf_Name", names["#sf_0"])

	valAttr, ok := values[":sf_0_0"].(*types.AttributeValueMemberS)
	require.True(t, ok)
	assert.Equal(t, "wire", valAttr.Value)
}

func TestBuildFilterExpression_MultipleFilters(t *testing.T) {
	filters := []cache.Filter{
		{Field: "Category", Operation: cache.FilterOpEq, Value: "electronics"},
		{Field: "Price", Operation: cache.FilterOpGt, Value: 10},
	}

	expression, names, values := buildFilterExpression(filters)

	assert.Equal(t, "#sf_0 = :sf_0_0 AND #sf_1 > :sf_1_0", expression)
	assert.Equal(t, "sf_Category", names["#sf_0"])
	assert.Equal(t, "sf_Price", names["#sf_1"])
	assert.Len(t, values, 2)
}

func TestBuildFilterExpression_Empty(t *testing.T) {
	expression, names, values := buildFilterExpression(nil)

	assert.Empty(t, expression)
	assert.Nil(t, names)
	assert.Nil(t, values)
}

func TestExtractSearchAttributes_Tag(t *testing.T) {
	adapter := newSearchTestAdapter()
	product := testProduct{
		Name:     "Laptop",
		Category: "electronics",
		Price:    999.99,
		Count:    5,
	}

	attributes := adapter.extractSearchAttributes(product)

	require.NotNil(t, attributes)
	catAttr, ok := attributes["sf_Category"]
	require.True(t, ok)
	strAttr, ok := catAttr.(*types.AttributeValueMemberS)
	require.True(t, ok)
	assert.Equal(t, "electronics", strAttr.Value)
}

func TestExtractSearchAttributes_Numeric(t *testing.T) {
	adapter := newSearchTestAdapter()
	product := testProduct{
		Name:     "Phone",
		Category: "electronics",
		Price:    499.99,
		Count:    10,
	}

	attributes := adapter.extractSearchAttributes(product)

	require.NotNil(t, attributes)
	priceAttr, ok := attributes["sf_Price"]
	require.True(t, ok)
	numAttr, ok := priceAttr.(*types.AttributeValueMemberN)
	require.True(t, ok)
	assert.Equal(t, "499.99", numAttr.Value)
}

func TestExtractSearchAttributes_Text(t *testing.T) {
	adapter := newSearchTestAdapter()
	product := testProduct{
		Name:     "Wireless Headphones",
		Category: "audio",
		Price:    79.99,
		Count:    100,
	}

	attributes := adapter.extractSearchAttributes(product)

	require.NotNil(t, attributes)
	nameAttr, ok := attributes["sf_Name"]
	require.True(t, ok)
	strAttr, ok := nameAttr.(*types.AttributeValueMemberS)
	require.True(t, ok)
	assert.Equal(t, "Wireless Headphones", strAttr.Value)
}

func TestSupportsSearch_WithSchema(t *testing.T) {
	adapter := newSearchTestAdapter()
	assert.True(t, adapter.SupportsSearch())
}

func TestSupportsSearch_WithoutSchema(t *testing.T) {
	adapter := newNoSchemaAdapter()
	assert.False(t, adapter.SupportsSearch())
}

func TestToAttributeValue_String(t *testing.T) {
	result := toAttributeValue("hello")
	strAttr, ok := result.(*types.AttributeValueMemberS)
	require.True(t, ok)
	assert.Equal(t, "hello", strAttr.Value)
}

func TestToAttributeValue_Int(t *testing.T) {
	result := toAttributeValue(42)
	numAttr, ok := result.(*types.AttributeValueMemberN)
	require.True(t, ok)
	assert.Equal(t, "42", numAttr.Value)
}

func TestToAttributeValue_Float64(t *testing.T) {
	result := toAttributeValue(3.14)
	numAttr, ok := result.(*types.AttributeValueMemberN)
	require.True(t, ok)
	assert.Equal(t, "3.14", numAttr.Value)
}

func TestToAttributeValue_Bool(t *testing.T) {
	result := toAttributeValue(true)

	strAttr, ok := result.(*types.AttributeValueMemberS)
	require.True(t, ok)
	assert.Equal(t, "true", strAttr.Value)
}

func TestGetSchema_WithSchema(t *testing.T) {
	adapter := newSearchTestAdapter()
	schema := adapter.GetSchema()
	require.NotNil(t, schema)
	assert.Len(t, schema.Fields, 4)
}

func TestGetSchema_WithoutSchema(t *testing.T) {
	adapter := newNoSchemaAdapter()
	assert.Nil(t, adapter.GetSchema())
}

func TestSortKeysByField_Empty(t *testing.T) {
	adapter := newSearchTestAdapter()
	result := adapter.sortKeys(nil, "Price", cache.SortAsc)
	assert.Nil(t, result)
}

func TestSortKeysByField_NoSortBy(t *testing.T) {
	adapter := newSearchTestAdapter()
	keys := []string{"a", "b", "c"}
	result := adapter.sortKeys(keys, "", cache.SortAsc)
	assert.Equal(t, keys, result)
}
