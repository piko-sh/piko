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
	"go/ast"
	"go/token"
	"testing"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/collection/collection_dto"
)

func newTestCollectionService(t *testing.T) *collectionService {
	t.Helper()
	registry := newTestProviderRegistry()
	return mustCastToCollectionService(t, NewCollectionService(context.Background(), registry))
}

func TestCapitalize(t *testing.T) {
	testCases := []struct {
		name   string
		input  string
		expect string
	}{
		{name: "Empty", input: "", expect: ""},
		{name: "SingleChar", input: "a", expect: "A"},
		{name: "NormalWord", input: "title", expect: "Title"},
		{name: "AlreadyCapitalised", input: "Title", expect: "Title"},
		{name: "AllUppercase", input: "URL", expect: "URL"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := capitalise(tc.input)
			if result != tc.expect {
				t.Errorf("capitalise(%q) = %q, want %q", tc.input, result, tc.expect)
			}
		})
	}
}

func TestValueToASTExpr(t *testing.T) {
	service := newTestCollectionService(t)

	t.Run("Nil", func(t *testing.T) {
		expression, err := service.valueToASTExpr(context.Background(), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		identifier, ok := expression.(*ast.Ident)
		if !ok {
			t.Fatalf("expected *ast.Ident, got %T", expression)
		}
		if identifier.Name != "nil" {
			t.Errorf("expected 'nil', got %q", identifier.Name)
		}
	})

	t.Run("String", func(t *testing.T) {
		expression, err := service.valueToASTExpr(context.Background(), "hello")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		lit, ok := expression.(*ast.BasicLit)
		if !ok {
			t.Fatalf("expected *ast.BasicLit, got %T", expression)
		}
		if lit.Kind != token.STRING {
			t.Errorf("expected STRING token, got %v", lit.Kind)
		}
		if lit.Value != `"hello"` {
			t.Errorf("expected %q, got %q", `"hello"`, lit.Value)
		}
	})

	t.Run("Int", func(t *testing.T) {
		expression, err := service.valueToASTExpr(context.Background(), 42)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		lit, ok := expression.(*ast.BasicLit)
		if !ok {
			t.Fatalf("expected *ast.BasicLit, got %T", expression)
		}
		if lit.Kind != token.INT {
			t.Errorf("expected INT token, got %v", lit.Kind)
		}
		if lit.Value != "42" {
			t.Errorf("expected '42', got %q", lit.Value)
		}
	})

	t.Run("Int64", func(t *testing.T) {
		expression, err := service.valueToASTExpr(context.Background(), int64(100))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		lit, ok := expression.(*ast.BasicLit)
		require.True(t, ok, "expected expression to be *ast.BasicLit")
		if lit.Kind != token.INT {
			t.Errorf("expected INT token, got %v", lit.Kind)
		}
	})

	t.Run("Uint", func(t *testing.T) {
		expression, err := service.valueToASTExpr(context.Background(), uint(10))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		lit, ok := expression.(*ast.BasicLit)
		require.True(t, ok, "expected expression to be *ast.BasicLit")
		if lit.Kind != token.INT {
			t.Errorf("expected INT token, got %v", lit.Kind)
		}
	})

	t.Run("Float64", func(t *testing.T) {
		expression, err := service.valueToASTExpr(context.Background(), 3.14)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		lit, ok := expression.(*ast.BasicLit)
		require.True(t, ok, "expected expression to be *ast.BasicLit")
		if lit.Kind != token.FLOAT {
			t.Errorf("expected FLOAT token, got %v", lit.Kind)
		}
	})

	t.Run("Float32", func(t *testing.T) {
		expression, err := service.valueToASTExpr(context.Background(), float32(2.5))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		lit, ok := expression.(*ast.BasicLit)
		require.True(t, ok, "expected expression to be *ast.BasicLit")
		if lit.Kind != token.FLOAT {
			t.Errorf("expected FLOAT token, got %v", lit.Kind)
		}
	})

	t.Run("BoolTrue", func(t *testing.T) {
		expression, err := service.valueToASTExpr(context.Background(), true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		identifier, ok := expression.(*ast.Ident)
		require.True(t, ok, "expected expression to be *ast.Ident")
		if identifier.Name != "true" {
			t.Errorf("expected 'true', got %q", identifier.Name)
		}
	})

	t.Run("BoolFalse", func(t *testing.T) {
		expression, err := service.valueToASTExpr(context.Background(), false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		identifier, ok := expression.(*ast.Ident)
		require.True(t, ok, "expected expression to be *ast.Ident")
		if identifier.Name != "false" {
			t.Errorf("expected 'false', got %q", identifier.Name)
		}
	})

	t.Run("SliceOfAny", func(t *testing.T) {
		expression, err := service.valueToASTExpr(context.Background(), []any{"a", "b"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		comp, ok := expression.(*ast.CompositeLit)
		if !ok {
			t.Fatalf("expected *ast.CompositeLit, got %T", expression)
		}
		if len(comp.Elts) != 2 {
			t.Errorf("expected 2 elements, got %d", len(comp.Elts))
		}
	})

	t.Run("MapStringAny", func(t *testing.T) {
		expression, err := service.valueToASTExpr(context.Background(), map[string]any{"key": "value"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		comp, ok := expression.(*ast.CompositeLit)
		if !ok {
			t.Fatalf("expected *ast.CompositeLit, got %T", expression)
		}
		if len(comp.Elts) != 1 {
			t.Errorf("expected 1 key-value pair, got %d", len(comp.Elts))
		}
	})

	t.Run("UnsupportedTypeFallback", func(t *testing.T) {
		type custom struct{ X int }
		expression, err := service.valueToASTExpr(context.Background(), custom{X: 5})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		lit, ok := expression.(*ast.BasicLit)
		if !ok {
			t.Fatalf("expected *ast.BasicLit (fallback string), got %T", expression)
		}
		if lit.Kind != token.STRING {
			t.Errorf("expected STRING token, got %v", lit.Kind)
		}
	})
}

func TestBuildItemLiteral(t *testing.T) {
	service := newTestCollectionService(t)

	t.Run("SimpleMetadata", func(t *testing.T) {
		item := &collection_dto.ContentItem{
			Metadata: map[string]any{"title": "Hello"},
		}
		expression, err := service.buildItemLiteral(context.Background(), item)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		comp, ok := expression.(*ast.CompositeLit)
		if !ok {
			t.Fatalf("expected *ast.CompositeLit, got %T", expression)
		}
		if len(comp.Elts) != 1 {
			t.Errorf("expected 1 field (Title), got %d", len(comp.Elts))
		}
	})

	t.Run("MultipleFieldsSorted", func(t *testing.T) {
		item := &collection_dto.ContentItem{
			Metadata: map[string]any{"zebra": "z", "alpha": "a"},
		}
		expression, err := service.buildItemLiteral(context.Background(), item)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		comp, ok := expression.(*ast.CompositeLit)
		require.True(t, ok, "expected expression to be *ast.CompositeLit")
		if len(comp.Elts) < 2 {
			t.Fatalf("expected at least 2 fields, got %d", len(comp.Elts))
		}
		firstKV, ok := comp.Elts[0].(*ast.KeyValueExpr)
		require.True(t, ok, "expected comp.Elts[0] to be *ast.KeyValueExpr")
		firstIdent, ok := firstKV.Key.(*ast.Ident)
		require.True(t, ok, "expected firstKV.Key to be *ast.Ident")
		secondKV, ok := comp.Elts[1].(*ast.KeyValueExpr)
		require.True(t, ok, "expected comp.Elts[1] to be *ast.KeyValueExpr")
		secondIdent, ok := secondKV.Key.(*ast.Ident)
		require.True(t, ok, "expected secondKV.Key to be *ast.Ident")
		first := firstIdent.Name
		second := secondIdent.Name
		if first != "Alpha" || second != "Zebra" {
			t.Errorf("expected sorted keys [Alpha, Zebra], got [%s, %s]", first, second)
		}
	})

	t.Run("InternalKeysFiltered", func(t *testing.T) {
		item := &collection_dto.ContentItem{
			Metadata: map[string]any{
				"title":                          "Test",
				collection_dto.MetaKeyDraft:      true,
				collection_dto.MetaKeyWordCount:  100,
				collection_dto.MetaKeySections:   nil,
				collection_dto.MetaKeyNavigation: nil,
			},
		}
		expression, err := service.buildItemLiteral(context.Background(), item)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		comp, ok := expression.(*ast.CompositeLit)
		require.True(t, ok, "expected expression to be *ast.CompositeLit")
		for _, elt := range comp.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			require.True(t, ok, "expected elt to be *ast.KeyValueExpr")
			identifier, ok := kv.Key.(*ast.Ident)
			require.True(t, ok, "expected kv.Key to be *ast.Ident")
			name := identifier.Name
			for internalKey := range internalMetadataKeys {
				if name == capitalise(internalKey) {
					t.Errorf("internal key %q should be filtered out", name)
				}
			}
		}
	})

	t.Run("SlugAddedIfMissing", func(t *testing.T) {
		item := &collection_dto.ContentItem{
			Slug:     "test-slug",
			Metadata: map[string]any{"title": "Test"},
		}
		expression, err := service.buildItemLiteral(context.Background(), item)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		comp, ok := expression.(*ast.CompositeLit)
		require.True(t, ok, "expected expression to be *ast.CompositeLit")
		found := false
		for _, elt := range comp.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			require.True(t, ok, "expected elt to be *ast.KeyValueExpr")
			identifier, ok := kv.Key.(*ast.Ident)
			require.True(t, ok, "expected kv.Key to be *ast.Ident")
			if identifier.Name == "Slug" {
				found = true
			}
		}
		if !found {
			t.Error("expected Slug field to be added")
		}
	})

	t.Run("URLAddedIfMissing", func(t *testing.T) {
		item := &collection_dto.ContentItem{
			URL:      "/blog/test",
			Metadata: map[string]any{"title": "Test"},
		}
		expression, err := service.buildItemLiteral(context.Background(), item)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		comp, ok := expression.(*ast.CompositeLit)
		require.True(t, ok, "expected expression to be *ast.CompositeLit")
		found := false
		for _, elt := range comp.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			require.True(t, ok, "expected elt to be *ast.KeyValueExpr")
			identifier, ok := kv.Key.(*ast.Ident)
			require.True(t, ok, "expected kv.Key to be *ast.Ident")
			if identifier.Name == "URL" {
				found = true
			}
		}
		if !found {
			t.Error("expected URL field to be added")
		}
	})

	t.Run("SlugNotDuplicatedIfInMetadata", func(t *testing.T) {
		item := &collection_dto.ContentItem{
			Slug:     "from-field",
			Metadata: map[string]any{"slug": "from-metadata"},
		}
		expression, err := service.buildItemLiteral(context.Background(), item)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		comp, ok := expression.(*ast.CompositeLit)
		require.True(t, ok, "expected expression to be *ast.CompositeLit")
		slugCount := 0
		for _, elt := range comp.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			require.True(t, ok, "expected elt to be *ast.KeyValueExpr")
			identifier, ok := kv.Key.(*ast.Ident)
			require.True(t, ok, "expected kv.Key to be *ast.Ident")
			if identifier.Name == "Slug" {
				slugCount++
			}
		}
		if slugCount != 1 {
			t.Errorf("expected exactly 1 Slug field, got %d", slugCount)
		}
	})
}

func TestBuildSliceLiteral(t *testing.T) {
	service := newTestCollectionService(t)
	targetType := &ast.Ident{Name: "Post"}

	t.Run("Empty", func(t *testing.T) {
		expression, err := service.buildSliceLiteral(context.Background(), targetType, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		comp, ok := expression.(*ast.CompositeLit)
		require.True(t, ok, "expected expression to be *ast.CompositeLit")
		if len(comp.Elts) != 0 {
			t.Errorf("expected 0 elements, got %d", len(comp.Elts))
		}
	})

	t.Run("SingleItem", func(t *testing.T) {
		items := []collection_dto.ContentItem{
			{Metadata: map[string]any{"title": "First"}},
		}
		expression, err := service.buildSliceLiteral(context.Background(), targetType, items)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		comp, ok := expression.(*ast.CompositeLit)
		require.True(t, ok, "expected expression to be *ast.CompositeLit")
		if len(comp.Elts) != 1 {
			t.Errorf("expected 1 element, got %d", len(comp.Elts))
		}
	})

	t.Run("MultipleItems", func(t *testing.T) {
		items := []collection_dto.ContentItem{
			{Metadata: map[string]any{"title": "First"}},
			{Metadata: map[string]any{"title": "Second"}},
			{Metadata: map[string]any{"title": "Third"}},
		}
		expression, err := service.buildSliceLiteral(context.Background(), targetType, items)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		comp, ok := expression.(*ast.CompositeLit)
		require.True(t, ok, "expected expression to be *ast.CompositeLit")
		if len(comp.Elts) != 3 {
			t.Errorf("expected 3 elements, got %d", len(comp.Elts))
		}
		arrType, ok := comp.Type.(*ast.ArrayType)
		require.True(t, ok, "expected comp.Type to be *ast.ArrayType")
		arrEltIdent, ok := arrType.Elt.(*ast.Ident)
		require.True(t, ok, "expected arrType.Elt to be *ast.Ident")
		if arrEltIdent.Name != "Post" {
			t.Errorf("expected element type 'Post', got %q", arrEltIdent.Name)
		}
	})
}

func TestCreateSliceLit(t *testing.T) {
	service := newTestCollectionService(t)

	t.Run("Empty", func(t *testing.T) {
		expression, err := service.createSliceLit(context.Background(), []any{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		comp, ok := expression.(*ast.CompositeLit)
		require.True(t, ok, "expected expression to be *ast.CompositeLit")
		if len(comp.Elts) != 0 {
			t.Errorf("expected 0 elements, got %d", len(comp.Elts))
		}
	})

	t.Run("Nested", func(t *testing.T) {
		expression, err := service.createSliceLit(context.Background(), []any{"a", 1, true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		comp, ok := expression.(*ast.CompositeLit)
		require.True(t, ok, "expected expression to be *ast.CompositeLit")
		if len(comp.Elts) != 3 {
			t.Errorf("expected 3 elements, got %d", len(comp.Elts))
		}
	})
}

func TestCreateMapLit(t *testing.T) {
	service := newTestCollectionService(t)

	t.Run("Empty", func(t *testing.T) {
		expression, err := service.createMapLit(context.Background(), map[string]any{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		comp, ok := expression.(*ast.CompositeLit)
		require.True(t, ok, "expected expression to be *ast.CompositeLit")
		if len(comp.Elts) != 0 {
			t.Errorf("expected 0 pairs, got %d", len(comp.Elts))
		}
	})

	t.Run("SortedKeys", func(t *testing.T) {
		expression, err := service.createMapLit(context.Background(), map[string]any{"z": 1, "a": 2, "m": 3})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		comp, ok := expression.(*ast.CompositeLit)
		require.True(t, ok, "expected expression to be *ast.CompositeLit")
		if len(comp.Elts) != 3 {
			t.Fatalf("expected 3 pairs, got %d", len(comp.Elts))
		}
		firstKV, ok := comp.Elts[0].(*ast.KeyValueExpr)
		require.True(t, ok, "expected comp.Elts[0] to be *ast.KeyValueExpr")
		firstKeyLit, ok := firstKV.Key.(*ast.BasicLit)
		require.True(t, ok, "expected firstKV.Key to be *ast.BasicLit")
		if firstKeyLit.Value != `"a"` {
			t.Errorf("expected first key '\"a\"', got %s", firstKeyLit.Value)
		}
	})

	t.Run("NestedValues", func(t *testing.T) {
		expression, err := service.createMapLit(context.Background(), map[string]any{
			"nested": map[string]any{"inner": "value"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		comp, ok := expression.(*ast.CompositeLit)
		require.True(t, ok, "expected expression to be *ast.CompositeLit")
		if len(comp.Elts) != 1 {
			t.Errorf("expected 1 pair, got %d", len(comp.Elts))
		}
	})
}

func TestLiteralConstructors(t *testing.T) {
	t.Run("CreateNilIdent", func(t *testing.T) {
		identifier := createNilIdent()
		if identifier.Name != "nil" {
			t.Errorf("expected 'nil', got %q", identifier.Name)
		}
	})

	t.Run("CreateStringLit", func(t *testing.T) {
		lit := createStringLit("test")
		if lit.Kind != token.STRING {
			t.Errorf("expected STRING, got %v", lit.Kind)
		}
		if lit.Value != `"test"` {
			t.Errorf("expected '\"test\"', got %s", lit.Value)
		}
	})

	t.Run("CreateIntLit", func(t *testing.T) {
		lit := createIntLit(42)
		if lit.Kind != token.INT {
			t.Errorf("expected INT, got %v", lit.Kind)
		}
		if lit.Value != "42" {
			t.Errorf("expected '42', got %s", lit.Value)
		}
	})

	t.Run("CreateFloatLit", func(t *testing.T) {
		lit := createFloatLit(3.14)
		if lit.Kind != token.FLOAT {
			t.Errorf("expected FLOAT, got %v", lit.Kind)
		}
	})

	t.Run("CreateBoolIdent_True", func(t *testing.T) {
		identifier := createBoolIdent(true)
		if identifier.Name != "true" {
			t.Errorf("expected 'true', got %q", identifier.Name)
		}
	})

	t.Run("CreateBoolIdent_False", func(t *testing.T) {
		identifier := createBoolIdent(false)
		if identifier.Name != "false" {
			t.Errorf("expected 'false', got %q", identifier.Name)
		}
	})
}

func TestCreateKeyValuePair(t *testing.T) {
	kv := createKeyValuePair("Title", createStringLit("Hello"))
	keyIdent, ok := kv.Key.(*ast.Ident)
	require.True(t, ok, "expected kv.Key to be *ast.Ident")
	if keyIdent.Name != "Title" {
		t.Errorf("expected key 'Title', got %q", keyIdent.Name)
	}
	lit, ok := kv.Value.(*ast.BasicLit)
	require.True(t, ok, "expected kv.Value to be *ast.BasicLit")
	if lit.Value != `"Hello"` {
		t.Errorf("expected value '\"Hello\"', got %s", lit.Value)
	}
}

func TestCreateFallbackStringLit(t *testing.T) {
	service := newTestCollectionService(t)
	type custom struct{ X int }
	lit := service.createFallbackStringLit(context.Background(), custom{X: 5})
	if lit.Kind != token.STRING {
		t.Errorf("expected STRING, got %v", lit.Kind)
	}
}
