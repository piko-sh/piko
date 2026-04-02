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
	"testing"

	"piko.sh/piko/internal/cache/cache_dto"
)

func TestFieldExtractor_ExtractTextFields(t *testing.T) {
	schema := cache_dto.NewSearchSchema(
		cache_dto.TextField("Name"),
		cache_dto.TextField("Description"),
	)

	extractor := NewFieldExtractor[Product](schema)

	product := Product{
		ID:          "1",
		Name:        "Premium Widget",
		Description: "High quality product",
		Category:    "electronics",
	}

	texts := extractor.ExtractTextFields(product)

	if len(texts) != 2 {
		t.Fatalf("expected 2 text fields, got %d", len(texts))
	}

	expectedTexts := map[string]bool{
		"Premium Widget":       true,
		"High quality product": true,
	}

	for _, text := range texts {
		if !expectedTexts[text] {
			t.Errorf("unexpected text field: %s", text)
		}
	}
}

func TestFieldExtractor_ExtractTagValue(t *testing.T) {
	schema := cache_dto.NewSearchSchema(
		cache_dto.TagField("Category"),
	)

	extractor := NewFieldExtractor[Product](schema)

	product := Product{
		ID:       "1",
		Category: "electronics",
	}

	tag := extractor.ExtractTagValue(product, "Category")

	if tag != "electronics" {
		t.Errorf("expected 'electronics', got '%s'", tag)
	}
}

func TestFieldExtractor_ExtractNumericValue(t *testing.T) {
	schema := cache_dto.NewSearchSchema(
		cache_dto.NumericField("Price"),
		cache_dto.NumericField("Stock"),
	)

	extractor := NewFieldExtractor[Product](schema)

	product := Product{
		ID:    "1",
		Price: 99.99,
		Stock: 42,
	}

	price, ok := extractor.ExtractNumericValue(product, "Price")
	if !ok {
		t.Error("failed to extract Price")
	}
	if price != 99.99 {
		t.Errorf("expected price 99.99, got %f", price)
	}

	stock, ok := extractor.ExtractNumericValue(product, "Stock")
	if !ok {
		t.Error("failed to extract Stock")
	}
	if stock != 42.0 {
		t.Errorf("expected stock 42.0, got %f", stock)
	}
}

func TestFieldExtractor_ExtractSortableValue(t *testing.T) {
	schema := cache_dto.NewSearchSchema(
		cache_dto.SortableNumericField("Price"),
		cache_dto.NumericField("Stock"),
	)

	extractor := NewFieldExtractor[Product](schema)

	product := Product{
		ID:    "1",
		Price: 99.99,
		Stock: 42,
	}

	price, ok := extractor.ExtractSortableValue(product, "Price")
	if !ok {
		t.Error("failed to extract sortable Price")
	}
	if priceFloat, ok := price.(float64); !ok || priceFloat != 99.99 {
		t.Errorf("expected price 99.99, got %v", price)
	}

	_, ok = extractor.ExtractSortableValue(product, "Stock")
	if ok {
		t.Error("Stock should not be extractable as sortable")
	}
}

func TestFieldExtractor_IsSortable(t *testing.T) {
	schema := cache_dto.NewSearchSchema(
		cache_dto.SortableNumericField("Price"),
		cache_dto.NumericField("Stock"),
	)

	extractor := NewFieldExtractor[Product](schema)

	if !extractor.IsSortable("Price") {
		t.Error("Price should be sortable")
	}

	if extractor.IsSortable("Stock") {
		t.Error("Stock should not be sortable")
	}

	if extractor.IsSortable("NonExistent") {
		t.Error("NonExistent field should not be sortable")
	}
}

func TestFieldExtractor_GetSortableFields(t *testing.T) {
	schema := cache_dto.NewSearchSchema(
		cache_dto.SortableNumericField("Price"),
		cache_dto.SortableNumericField("Rating"),
		cache_dto.NumericField("Stock"),
	)

	extractor := NewFieldExtractor[Product](schema)

	fields := extractor.GetSortableFields()

	if len(fields) != 2 {
		t.Fatalf("expected 2 sortable fields, got %d", len(fields))
	}

	fieldSet := make(map[string]bool)
	for _, field := range fields {
		fieldSet[field] = true
	}

	if !fieldSet["Price"] {
		t.Error("Price should be in sortable fields")
	}
	if !fieldSet["Rating"] {
		t.Error("Rating should be in sortable fields")
	}
	if fieldSet["Stock"] {
		t.Error("Stock should not be in sortable fields")
	}
}

func TestFieldExtractor_NestedFields(t *testing.T) {
	type Address struct {
		City     string
		Country  string
		Postcode int
	}

	type User struct {
		Name    string
		Address Address
		Age     int
	}

	schema := cache_dto.NewSearchSchema(
		cache_dto.TextField("Name"),
		cache_dto.TextField("Address.City"),
		cache_dto.NumericField("Address.Postcode"),
	)

	extractor := NewFieldExtractor[User](schema)

	user := User{
		Name: "John Doe",
		Age:  30,
		Address: Address{
			City:     "London",
			Country:  "UK",
			Postcode: 12345,
		},
	}

	texts := extractor.ExtractTextFields(user)
	if len(texts) != 2 {
		t.Fatalf("expected 2 text fields, got %d", len(texts))
	}

	expectedTexts := map[string]bool{
		"John Doe": true,
		"London":   true,
	}
	for _, text := range texts {
		if !expectedTexts[text] {
			t.Errorf("unexpected text: %s", text)
		}
	}

	postcode, ok := extractor.ExtractNumericValue(user, "Address.Postcode")
	if !ok {
		t.Error("failed to extract Address.Postcode")
	}
	if postcode != 12345.0 {
		t.Errorf("expected postcode 12345.0, got %f", postcode)
	}
}

func TestFieldExtractor_PointerFields(t *testing.T) {
	type Item struct {
		Name  string
		Price float64
	}

	schema := cache_dto.NewSearchSchema(
		cache_dto.TextField("Name"),
		cache_dto.NumericField("Price"),
	)

	extractor := NewFieldExtractor[*Item](schema)

	item := &Item{
		Name:  "Widget",
		Price: 99.99,
	}

	texts := extractor.ExtractTextFields(item)
	if len(texts) != 1 {
		t.Fatalf("expected 1 text field, got %d", len(texts))
	}
	if texts[0] != "Widget" {
		t.Errorf("expected 'Widget', got '%s'", texts[0])
	}

	price, ok := extractor.ExtractNumericValue(item, "Price")
	if !ok {
		t.Error("failed to extract Price from pointer struct")
	}
	if price != 99.99 {
		t.Errorf("expected price 99.99, got %f", price)
	}
}

func TestFieldExtractor_CaseInsensitiveFields(t *testing.T) {
	schema := cache_dto.NewSearchSchema(
		cache_dto.TextField("name"),
		cache_dto.NumericField("PRICE"),
	)

	extractor := NewFieldExtractor[Product](schema)

	product := Product{
		Name:  "Widget",
		Price: 99.99,
	}

	texts := extractor.ExtractTextFields(product)
	if len(texts) != 1 {
		t.Fatalf("expected 1 text field, got %d", len(texts))
	}
	if texts[0] != "Widget" {
		t.Errorf("expected 'Widget', got '%s'", texts[0])
	}

	price, ok := extractor.ExtractNumericValue(product, "PRICE")
	if !ok {
		t.Error("failed to extract PRICE (case insensitive)")
	}
	if price != 99.99 {
		t.Errorf("expected price 99.99, got %f", price)
	}
}

func TestFieldExtractor_EmptyFields(t *testing.T) {
	schema := cache_dto.NewSearchSchema(
		cache_dto.TextField("Name"),
		cache_dto.NumericField("Price"),
	)

	extractor := NewFieldExtractor[Product](schema)

	product := Product{
		Name:  "",
		Price: 0.0,
	}

	texts := extractor.ExtractTextFields(product)
	if len(texts) != 0 {
		t.Errorf("expected 0 text fields for empty string, got %d", len(texts))
	}

	price, ok := extractor.ExtractNumericValue(product, "Price")
	if !ok {
		t.Error("failed to extract zero Price")
	}
	if price != 0.0 {
		t.Errorf("expected price 0.0, got %f", price)
	}
}

func TestFieldExtractor_NonExistentField(t *testing.T) {
	schema := cache_dto.NewSearchSchema(
		cache_dto.TextField("Name"),
	)

	extractor := NewFieldExtractor[Product](schema)

	product := Product{
		Name: "Widget",
	}

	value := extractor.ExtractTagValue(product, "NonExistent")
	if value != "" {
		t.Errorf("expected empty string for non-existent field, got '%s'", value)
	}

	_, ok := extractor.ExtractNumericValue(product, "NonExistent")
	if ok {
		t.Error("expected false when extracting non-existent numeric field")
	}
}

func TestFieldExtractor_NilSchema(t *testing.T) {
	extractor := NewFieldExtractor[Product](nil)

	if extractor != nil {
		t.Error("expected nil extractor for nil schema")
	}
}

func TestFieldExtractor_NumericTypes(t *testing.T) {
	type NumericTypes struct {
		Int8Val    int8
		Int16Val   int16
		Int32Val   int32
		Int64Val   int64
		Uint8Val   uint8
		Uint16Val  uint16
		Uint32Val  uint32
		Uint64Val  uint64
		Float32Val float32
		Float64Val float64
	}

	schema := cache_dto.NewSearchSchema(
		cache_dto.NumericField("Int8Val"),
		cache_dto.NumericField("Int16Val"),
		cache_dto.NumericField("Int32Val"),
		cache_dto.NumericField("Int64Val"),
		cache_dto.NumericField("Uint8Val"),
		cache_dto.NumericField("Uint16Val"),
		cache_dto.NumericField("Uint32Val"),
		cache_dto.NumericField("Uint64Val"),
		cache_dto.NumericField("Float32Val"),
		cache_dto.NumericField("Float64Val"),
	)

	extractor := NewFieldExtractor[NumericTypes](schema)

	value := NumericTypes{
		Int8Val:    127,
		Int16Val:   32767,
		Int32Val:   2147483647,
		Int64Val:   9223372036854775807,
		Uint8Val:   255,
		Uint16Val:  65535,
		Uint32Val:  4294967295,
		Uint64Val:  18446744073709551615,
		Float32Val: 3.14,
		Float64Val: 2.718281828,
	}

	tests := []struct {
		name     string
		field    string
		expected float64
	}{
		{name: "int8", field: "Int8Val", expected: 127},
		{name: "int16", field: "Int16Val", expected: 32767},
		{name: "int32", field: "Int32Val", expected: 2147483647},
		{name: "int64", field: "Int64Val", expected: 9223372036854775807},
		{name: "uint8", field: "Uint8Val", expected: 255},
		{name: "uint16", field: "Uint16Val", expected: 65535},
		{name: "uint32", field: "Uint32Val", expected: 4294967295},
		{name: "uint64", field: "Uint64Val", expected: 18446744073709551615},
		{name: "float32", field: "Float32Val", expected: 3.14},
		{name: "float64", field: "Float64Val", expected: 2.718281828},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := extractor.ExtractNumericValue(value, tt.field)
			if !ok {
				t.Fatalf("failed to extract %s", tt.field)
			}

			delta := result - tt.expected
			if delta < 0 {
				delta = -delta
			}
			if delta > 0.0001 {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}

type CustomID struct {
	Value string
}

func (c CustomID) String() string {
	return c.Value
}

type ItemWithCustomID struct {
	ID CustomID
}

func TestFieldExtractor_StringerInterface(t *testing.T) {
	schema := cache_dto.NewSearchSchema(
		cache_dto.TextField("ID"),
	)

	extractor := NewFieldExtractor[ItemWithCustomID](schema)

	item := ItemWithCustomID{
		ID: CustomID{Value: "custom-123"},
	}

	texts := extractor.ExtractTextFields(item)
	if len(texts) != 1 {
		t.Fatalf("expected 1 text field, got %d", len(texts))
	}
	if texts[0] != "custom-123" {
		t.Errorf("expected 'custom-123', got '%s'", texts[0])
	}
}
