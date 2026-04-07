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
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/collection/collection_adapters"
	"piko.sh/piko/internal/collection/collection_dto"
)

func TestRegisterStaticCollectionBlob(t *testing.T) {
	ResetStaticCollectionRegistry()

	blob := []byte("test collection blob")
	RegisterStaticCollectionBlob(context.Background(), "test-collection", blob)

	if !HasStaticCollection("test-collection") {
		t.Error("Expected collection to be registered")
	}
}

func TestRegisterStaticCollectionBlob_Overwrite(t *testing.T) {
	ResetStaticCollectionRegistry()

	blob1 := []byte("first blob")
	blob2 := []byte("second blob")

	RegisterStaticCollectionBlob(context.Background(), "overwrite-test", blob1)
	RegisterStaticCollectionBlob(context.Background(), "overwrite-test", blob2)

	if !HasStaticCollection("overwrite-test") {
		t.Error("Expected collection to exist after overwrite")
	}
}

func TestRegisterStaticCollectionBlob_Multiple(t *testing.T) {
	ResetStaticCollectionRegistry()

	collections := []string{"coll-a", "coll-b", "coll-c"}
	for _, name := range collections {
		RegisterStaticCollectionBlob(context.Background(), name, []byte(name+"-data"))
	}

	for _, name := range collections {
		if !HasStaticCollection(name) {
			t.Errorf("Expected collection %q to be registered", name)
		}
	}
}

func TestHasStaticCollection_NotFound(t *testing.T) {
	ResetStaticCollectionRegistry()

	if HasStaticCollection("nonexistent") {
		t.Error("Expected HasStaticCollection to return false for nonexistent collection")
	}
}

func TestHasStaticCollection_Found(t *testing.T) {
	ResetStaticCollectionRegistry()

	RegisterStaticCollectionBlob(context.Background(), "exists", []byte("data"))

	if !HasStaticCollection("exists") {
		t.Error("Expected HasStaticCollection to return true for existing collection")
	}
}

func TestNewDefaultStaticCollectionRegistry(t *testing.T) {
	registry := NewDefaultStaticCollectionRegistry()
	require.NotNil(t, registry, "NewDefaultStaticCollectionRegistry() returned nil")
}

func TestDefaultStaticCollectionRegistry_Register(t *testing.T) {
	ResetStaticCollectionRegistry()

	registry := NewDefaultStaticCollectionRegistry()
	registry.Register(context.Background(), "registry-test", []byte("data"))

	if !HasStaticCollection("registry-test") {
		t.Error("Expected collection to be registered via interface")
	}
}

func TestDefaultStaticCollectionRegistry_Has(t *testing.T) {
	ResetStaticCollectionRegistry()

	registry := NewDefaultStaticCollectionRegistry()

	if registry.Has("has-test") {
		t.Error("Expected Has to return false for unregistered collection")
	}

	RegisterStaticCollectionBlob(context.Background(), "has-test", []byte("data"))

	if !registry.Has("has-test") {
		t.Error("Expected Has to return true for registered collection")
	}
}

func TestDefaultStaticCollectionRegistry_List(t *testing.T) {
	ResetStaticCollectionRegistry()

	collections := []string{"list-x", "list-y", "list-z"}
	for _, name := range collections {
		RegisterStaticCollectionBlob(context.Background(), name, []byte("data"))
	}

	registry := NewDefaultStaticCollectionRegistry()
	names := registry.List()

	if len(names) != len(collections) {
		t.Errorf("Expected %d collections, got %d", len(collections), len(names))
	}

	nameMap := make(map[string]bool)
	for _, n := range names {
		nameMap[n] = true
	}

	for _, expected := range collections {
		if !nameMap[expected] {
			t.Errorf("Expected collection %q in list", expected)
		}
	}
}

func TestGetStaticCollectionItem_CollectionNotFound(t *testing.T) {
	ResetStaticCollectionRegistry()

	_, _, _, err := GetStaticCollectionItem(context.Background(), "nonexistent", "/some/route")
	if err == nil {
		t.Error("Expected error for nonexistent collection")
	}
}

func TestGetStaticCollectionItem_WithRealEncoder(t *testing.T) {
	ResetStaticCollectionRegistry()

	items := []collection_dto.ContentItem{
		{
			ID:   "item-1",
			Slug: "test-item",
			URL:  "/test/item",
			Metadata: map[string]any{
				"title": "Test Item",
			},
		},
	}

	encoder := collection_adapters.NewFlatBufferEncoder()
	blob, err := encoder.EncodeCollection(items)
	if err != nil {
		t.Fatalf("Failed to encode items: %v", err)
	}

	RegisterStaticCollectionBlob(context.Background(), "real-test", blob)

	metadata, contentAST, excerptAST, err := GetStaticCollectionItem(context.Background(), "real-test", "/test/item")
	if err != nil {
		t.Fatalf("GetStaticCollectionItem(context.Background(),) failed: %v", err)
	}

	if metadata["title"] != "Test Item" {
		t.Errorf("Expected title 'Test Item', got %v", metadata["title"])
	}

	_ = contentAST
	_ = excerptAST
}

func TestGetStaticCollectionItem_RouteNotFound(t *testing.T) {
	ResetStaticCollectionRegistry()

	items := []collection_dto.ContentItem{
		{
			ID:   "item-1",
			URL:  "/existing/route",
			Slug: "existing",
		},
	}

	encoder := collection_adapters.NewFlatBufferEncoder()
	blob, err := encoder.EncodeCollection(items)
	if err != nil {
		t.Fatalf("Failed to encode items: %v", err)
	}

	RegisterStaticCollectionBlob(context.Background(), "route-test", blob)

	_, _, _, err = GetStaticCollectionItem(context.Background(), "route-test", "/nonexistent/route")
	if err == nil {
		t.Error("Expected error for nonexistent route")
	}
}

func TestGetStaticCollectionItem_IndexFallback(t *testing.T) {
	ResetStaticCollectionRegistry()

	items := []collection_dto.ContentItem{
		{
			ID:   "index-item",
			URL:  "/docs/index",
			Slug: "index",
			Metadata: map[string]any{
				"title": "Index Page",
			},
		},
	}

	encoder := collection_adapters.NewFlatBufferEncoder()
	blob, err := encoder.EncodeCollection(items)
	if err != nil {
		t.Fatalf("Failed to encode items: %v", err)
	}

	RegisterStaticCollectionBlob(context.Background(), "index-test", blob)

	metadata, _, _, err := GetStaticCollectionItem(context.Background(), "index-test", "/docs/")
	if err != nil {
		t.Fatalf("GetStaticCollectionItem(context.Background(),) failed: %v", err)
	}

	if metadata["title"] != "Index Page" {
		t.Errorf("Expected title 'Index Page', got %v", metadata["title"])
	}
}

func TestGetStaticCollectionItems_CollectionNotFound(t *testing.T) {
	ResetStaticCollectionRegistry()

	_, err := GetStaticCollectionItems("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent collection")
	}
}

func TestGetStaticCollectionItems_ReturnsAllItems(t *testing.T) {
	ResetStaticCollectionRegistry()

	items := []collection_dto.ContentItem{
		{
			ID:   "item-1",
			URL:  "/blog/post-1",
			Slug: "post-1",
			Metadata: map[string]any{
				"title": "Post 1",
			},
		},
		{
			ID:   "item-2",
			URL:  "/blog/post-2",
			Slug: "post-2",
			Metadata: map[string]any{
				"title": "Post 2",
			},
		},
		{
			ID:   "item-3",
			URL:  "/blog/post-3",
			Slug: "post-3",
			Metadata: map[string]any{
				"title": "Post 3",
			},
		},
	}

	encoder := collection_adapters.NewFlatBufferEncoder()
	blob, err := encoder.EncodeCollection(items)
	if err != nil {
		t.Fatalf("Failed to encode items: %v", err)
	}

	RegisterStaticCollectionBlob(context.Background(), "all-items-test", blob)

	allItems, err := GetStaticCollectionItems("all-items-test")
	if err != nil {
		t.Fatalf("GetStaticCollectionItems() failed: %v", err)
	}

	if len(allItems) != 3 {
		t.Errorf("Expected 3 items, got %d", len(allItems))
	}

	for i, item := range allItems {
		title, ok := item["title"].(string)
		if !ok {
			t.Errorf("Item %d: expected title to be string", i)
			continue
		}
		if title == "" {
			t.Errorf("Item %d: expected non-empty title", i)
		}
	}
}

func TestGetStaticCollectionItems_EncoderRejectsEmptyCollection(t *testing.T) {
	ResetStaticCollectionRegistry()

	items := []collection_dto.ContentItem{}

	encoder := collection_adapters.NewFlatBufferEncoder()
	_, err := encoder.EncodeCollection(items)
	if err == nil {
		t.Error("Expected encoder to reject empty collection, but it succeeded")
	}

	if err != nil && !containsString(err.Error(), "empty") {
		t.Errorf("Expected error to mention 'empty', got: %v", err)
	}
}

func TestGetStaticCollectionItems_UnregisteredCollection(t *testing.T) {
	ResetStaticCollectionRegistry()

	_, err := GetStaticCollectionItems("nonexistent-collection")
	if err == nil {
		t.Error("Expected error for unregistered collection, got nil")
	}
}

func TestDefaultStaticCollectionRegistry_GetItem(t *testing.T) {
	ResetStaticCollectionRegistry()

	items := []collection_dto.ContentItem{
		{
			ID:   "registry-item",
			URL:  "/registry/item",
			Slug: "item",
			Metadata: map[string]any{
				"title": "Registry Item",
			},
		},
	}

	encoder := collection_adapters.NewFlatBufferEncoder()
	blob, err := encoder.EncodeCollection(items)
	if err != nil {
		t.Fatalf("Failed to encode items: %v", err)
	}

	RegisterStaticCollectionBlob(context.Background(), "get-item-test", blob)

	registry := NewDefaultStaticCollectionRegistry()
	result, err := registry.GetItem(context.Background(), "get-item-test", "/registry/item")
	if err != nil {
		t.Fatalf("GetItem() failed: %v", err)
	}

	if result.Metadata["title"] != "Registry Item" {
		t.Errorf("Expected title 'Registry Item', got %v", result.Metadata["title"])
	}
}

func TestDefaultStaticCollectionRegistry_GetItem_NotFound(t *testing.T) {
	ResetStaticCollectionRegistry()

	registry := NewDefaultStaticCollectionRegistry()
	_, err := registry.GetItem(context.Background(), "nonexistent", "/some/route")
	if err == nil {
		t.Error("Expected error for nonexistent collection")
	}
}

func TestDefaultStaticCollectionRegistry_GetAllItems(t *testing.T) {
	ResetStaticCollectionRegistry()

	items := []collection_dto.ContentItem{
		{
			ID:   "all-1",
			URL:  "/all/1",
			Slug: "1",
			Metadata: map[string]any{
				"title": "Item 1",
			},
		},
		{
			ID:   "all-2",
			URL:  "/all/2",
			Slug: "2",
			Metadata: map[string]any{
				"title": "Item 2",
			},
		},
	}

	encoder := collection_adapters.NewFlatBufferEncoder()
	blob, err := encoder.EncodeCollection(items)
	if err != nil {
		t.Fatalf("Failed to encode items: %v", err)
	}

	RegisterStaticCollectionBlob(context.Background(), "get-all-test", blob)

	registry := NewDefaultStaticCollectionRegistry()
	allItems, err := registry.GetAllItems("get-all-test")
	if err != nil {
		t.Fatalf("GetAllItems() failed: %v", err)
	}

	if len(allItems) != 2 {
		t.Errorf("Expected 2 items, got %d", len(allItems))
	}
}

func TestStaticCollectionRegistry_ConcurrentRegister(t *testing.T) {
	ResetStaticCollectionRegistry()

	var wg sync.WaitGroup
	numGoroutines := 100

	for i := range numGoroutines {
		index := i
		wg.Go(func() {
			name := "concurrent-" + string(rune('A'+index%26))
			RegisterStaticCollectionBlob(context.Background(), name, []byte("data"))
		})
	}

	wg.Wait()

	registry := NewDefaultStaticCollectionRegistry()
	names := registry.List()
	if len(names) == 0 {
		t.Error("Expected at least one collection to be registered")
	}
}

func TestStaticCollectionRegistry_ConcurrentRead(t *testing.T) {
	ResetStaticCollectionRegistry()

	items := []collection_dto.ContentItem{
		{
			ID:   "concurrent-item",
			URL:  "/concurrent/item",
			Slug: "item",
			Metadata: map[string]any{
				"title": "Concurrent Item",
			},
		},
	}

	encoder := collection_adapters.NewFlatBufferEncoder()
	blob, err := encoder.EncodeCollection(items)
	if err != nil {
		t.Fatalf("Failed to encode items: %v", err)
	}

	RegisterStaticCollectionBlob(context.Background(), "concurrent-read", blob)

	var wg sync.WaitGroup
	numGoroutines := 100

	for range numGoroutines {
		wg.Go(func() {
			_, _, _, err := GetStaticCollectionItem(context.Background(), "concurrent-read", "/concurrent/item")
			if err != nil {
				t.Errorf("GetStaticCollectionItem(context.Background(),) failed: %v", err)
			}
		})
	}

	wg.Wait()
}

func TestStaticCollectionRegistry_ConcurrentHas(t *testing.T) {
	ResetStaticCollectionRegistry()

	RegisterStaticCollectionBlob(context.Background(), "has-concurrent", []byte("data"))

	var wg sync.WaitGroup
	numGoroutines := 100

	for range numGoroutines {
		wg.Go(func() {
			if !HasStaticCollection("has-concurrent") {
				t.Error("HasStaticCollection returned false")
			}
		})
	}

	wg.Wait()
}

func TestResetStaticCollectionRegistry(t *testing.T) {
	ResetStaticCollectionRegistry()

	for _, name := range []string{"reset-a", "reset-b"} {
		RegisterStaticCollectionBlob(context.Background(), name, []byte("data"))
	}

	registry := NewDefaultStaticCollectionRegistry()
	if len(registry.List()) != 2 {
		t.Fatal("Expected 2 collections before reset")
	}

	ResetStaticCollectionRegistry()

	if len(registry.List()) != 0 {
		t.Error("Expected 0 collections after reset")
	}
}

func TestNewDefaultASTDecoder(t *testing.T) {
	decoder := NewDefaultASTDecoder()
	require.NotNil(t, decoder, "NewDefaultASTDecoder() returned nil")
}

func TestGetStaticCollectionNavigation_NotFound(t *testing.T) {
	ResetStaticCollectionRegistry()

	_, err := GetStaticCollectionNavigation(
		context.Background(),
		"nonexistent-nav",
		collection_dto.DefaultNavigationConfig(),
	)
	if err == nil {
		t.Error("Expected error for nonexistent collection")
	}
}

func TestGetStaticCollectionNavigation_BuildsTree(t *testing.T) {
	ResetStaticCollectionRegistry()

	items := []collection_dto.ContentItem{
		{
			ID:     "intro",
			Slug:   "intro",
			URL:    "/docs/intro",
			Locale: "en",
			Metadata: map[string]any{
				"Title": "Introduction",
				"Navigation": map[string]any{
					"Groups": map[string]any{
						"sidebar": map[string]any{
							"Section": "guides",
							"Order":   float64(1),
						},
					},
				},
			},
		},
		{
			ID:     "install",
			Slug:   "install",
			URL:    "/docs/install",
			Locale: "en",
			Metadata: map[string]any{
				"Title": "Installation",
				"Navigation": map[string]any{
					"Groups": map[string]any{
						"sidebar": map[string]any{
							"Section": "guides",
							"Order":   float64(2),
						},
					},
				},
			},
		},
	}

	encoder := collection_adapters.NewFlatBufferEncoder()
	blob, err := encoder.EncodeCollection(items)
	if err != nil {
		t.Fatalf("Failed to encode items: %v", err)
	}

	RegisterStaticCollectionBlob(context.Background(), "nav-test", blob)

	groups, err := GetStaticCollectionNavigation(
		context.Background(),
		"nav-test",
		collection_dto.NavigationConfig{
			IncludeHidden:  false,
			DefaultOrder:   999,
			GroupBySection: true,
		},
	)
	if err != nil {
		t.Fatalf("GetStaticCollectionNavigation() failed: %v", err)
	}

	require.NotNil(t, groups, "Expected non-nil navigation groups")

	if _, ok := groups.Groups["sidebar"]; !ok {
		t.Error("Expected navigation groups to contain 'sidebar'")
	}
}

func TestGetStaticCollectionNavigation_CachesResult(t *testing.T) {
	ResetStaticCollectionRegistry()

	items := []collection_dto.ContentItem{
		{
			ID:     "cached-item",
			Slug:   "cached",
			URL:    "/docs/cached",
			Locale: "en",
			Metadata: map[string]any{
				"Title": "Cached",
				"Navigation": map[string]any{
					"Groups": map[string]any{
						"sidebar": map[string]any{
							"Section": "guides",
							"Order":   float64(1),
						},
					},
				},
			},
		},
	}

	encoder := collection_adapters.NewFlatBufferEncoder()
	blob, err := encoder.EncodeCollection(items)
	if err != nil {
		t.Fatalf("Failed to encode items: %v", err)
	}

	RegisterStaticCollectionBlob(context.Background(), "cache-test", blob)

	config := collection_dto.NavigationConfig{
		IncludeHidden:  false,
		DefaultOrder:   999,
		GroupBySection: true,
	}

	first, err := GetStaticCollectionNavigation(context.Background(), "cache-test", config)
	if err != nil {
		t.Fatalf("First call failed: %v", err)
	}

	second, err := GetStaticCollectionNavigation(context.Background(), "cache-test", config)
	if err != nil {
		t.Fatalf("Second call failed: %v", err)
	}

	if first != second {
		t.Error("Expected same pointer from cached navigation")
	}
}

func TestTryGetCachedNavigation_CacheMiss(t *testing.T) {
	ResetStaticCollectionRegistry()

	freshItems := []map[string]any{
		{"Title": "Not from registry"},
	}

	_, ok := TryGetCachedNavigation(
		context.Background(),
		freshItems,
		collection_dto.DefaultNavigationConfig(),
	)
	if ok {
		t.Error("Expected cache miss for items not from the registry")
	}
}

func TestTryGetCachedNavigation_EmptySlice(t *testing.T) {
	ResetStaticCollectionRegistry()

	_, ok := TryGetCachedNavigation(
		context.Background(),
		[]map[string]any{},
		collection_dto.DefaultNavigationConfig(),
	)
	if ok {
		t.Error("Expected cache miss for empty slice")
	}
}

func TestMetadataToContentItems(t *testing.T) {
	t.Run("EmptyInput", func(t *testing.T) {
		result := metadataToContentItems(nil)
		if len(result) != 0 {
			t.Errorf("Expected 0 items, got %d", len(result))
		}
	})

	t.Run("PopulatesFieldsFromMetadata", func(t *testing.T) {
		items := []map[string]any{
			{
				"ID":     "item-1",
				"Slug":   "intro",
				"Locale": "en",
				"URL":    "/docs/intro",
				"Title":  "Introduction",
			},
		}

		result := metadataToContentItems(items)
		if len(result) != 1 {
			t.Fatalf("Expected 1 item, got %d", len(result))
		}

		item := result[0]
		if item.ID != "item-1" {
			t.Errorf("ID = %q, want %q", item.ID, "item-1")
		}
		if item.Slug != "intro" {
			t.Errorf("Slug = %q, want %q", item.Slug, "intro")
		}
		if item.Locale != "en" {
			t.Errorf("Locale = %q, want %q", item.Locale, "en")
		}
		if item.URL != "/docs/intro" {
			t.Errorf("URL = %q, want %q", item.URL, "/docs/intro")
		}
	})

	t.Run("MissingKeysUseZeroValues", func(t *testing.T) {
		items := []map[string]any{
			{"Title": "No ID or URL"},
		}

		result := metadataToContentItems(items)
		if len(result) != 1 {
			t.Fatalf("Expected 1 item, got %d", len(result))
		}

		item := result[0]
		if item.ID != "" {
			t.Errorf("ID = %q, want empty", item.ID)
		}
		if item.URL != "" {
			t.Errorf("URL = %q, want empty", item.URL)
		}
	})
}
