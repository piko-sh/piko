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

package conformance

import (
	"context"
	"maps"
	"slices"
	"testing"
)

// runIterationTests runs the cache iteration test suite.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache configuration to test.
func runIterationTests(t *testing.T, config StringConfig) {
	t.Helper()

	t.Run("All_Empty", func(t *testing.T) {
		t.Parallel()
		testAllEmpty(t, config)
	})

	t.Run("All_WithItems", func(t *testing.T) {
		t.Parallel()
		testAllWithItems(t, config)
	})

	t.Run("Keys_Empty", func(t *testing.T) {
		t.Parallel()
		testKeysEmpty(t, config)
	})

	t.Run("Keys_WithItems", func(t *testing.T) {
		t.Parallel()
		testKeysWithItems(t, config)
	})

	t.Run("Values_Empty", func(t *testing.T) {
		t.Parallel()
		testValuesEmpty(t, config)
	})

	t.Run("Values_WithItems", func(t *testing.T) {
		t.Parallel()
		testValuesWithItems(t, config)
	})

	t.Run("All_EarlyBreak", func(t *testing.T) {
		t.Parallel()
		testAllEarlyBreak(t, config)
	})
}

// testAllEmpty verifies that All returns no items for an empty cache.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache factory to test.
func testAllEmpty(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())

	count := 0
	for range cache.All() {
		count++
	}

	if count != 0 {
		t.Errorf("All() on empty cache should yield no items: got %d", count)
	}
}

// testAllWithItems verifies that All returns all items stored in the cache.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache factory and settings.
func testAllWithItems(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	expected := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	for k, v := range expected {
		if err := cache.Set(ctx, k, v); err != nil {
			t.Fatalf("Set %q failed: %v", k, err)
		}
	}

	found := maps.Collect(cache.All())

	if len(found) != len(expected) {
		t.Errorf("All() should yield %d items: got %d", len(expected), len(found))
	}

	for k, expectedV := range expected {
		if foundV, ok := found[k]; !ok {
			t.Errorf("Key %q not found in iteration", k)
		} else if foundV != expectedV {
			t.Errorf("Key %q: got %q, want %q", k, foundV, expectedV)
		}
	}
}

// testKeysEmpty verifies that Keys returns no items for an empty cache.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache factory to test.
func testKeysEmpty(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())

	count := 0
	for range cache.Keys() {
		count++
	}

	if count != 0 {
		t.Errorf("Keys() on empty cache should yield no items: got %d", count)
	}
}

// testKeysWithItems tests that Keys returns all stored keys from the cache.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache factory and settings.
func testKeysWithItems(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	expectedKeys := []string{"key1", "key2", "key3"}
	for _, k := range expectedKeys {
		if err := cache.Set(ctx, k, "value"); err != nil {
			t.Fatalf("Set %q failed: %v", k, err)
		}
	}

	foundKeys := slices.Collect(cache.Keys())

	if len(foundKeys) != len(expectedKeys) {
		t.Errorf("Keys() should yield %d items: got %d", len(expectedKeys), len(foundKeys))
	}

	for _, expected := range expectedKeys {
		if !slices.Contains(foundKeys, expected) {
			t.Errorf("Expected key %q not found in iteration", expected)
		}
	}
}

// testValuesEmpty verifies that Values returns no items for an empty cache.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache factory to test.
func testValuesEmpty(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())

	count := 0
	for range cache.Values() {
		count++
	}

	if count != 0 {
		t.Errorf("Values() on empty cache should yield no items: got %d", count)
	}
}

// testValuesWithItems verifies that Values returns all stored values.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache factory and settings.
func testValuesWithItems(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	if err := cache.Set(ctx, "key1", "value1"); err != nil {
		t.Fatalf("Set key1 failed: %v", err)
	}
	if err := cache.Set(ctx, "key2", "value2"); err != nil {
		t.Fatalf("Set key2 failed: %v", err)
	}
	if err := cache.Set(ctx, "key3", "value3"); err != nil {
		t.Fatalf("Set key3 failed: %v", err)
	}

	foundValues := slices.Collect(cache.Values())

	if len(foundValues) != 3 {
		t.Errorf("Values() should yield 3 items: got %d", len(foundValues))
	}

	expectedValues := []string{"value1", "value2", "value3"}
	for _, expected := range expectedValues {
		if !slices.Contains(foundValues, expected) {
			t.Errorf("Expected value %q not found in iteration", expected)
		}
	}
}

// testAllEarlyBreak verifies that breaking early from a cache All iteration
// stops at the expected count.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache factory and settings.
func testAllEarlyBreak(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	for i := range 10 {
		if err := cache.Set(ctx, string(rune('a'+i)), "value"); err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	}

	count := 0
	for range cache.All() {
		count++
		if count >= 3 {
			break
		}
	}

	if count != 3 {
		t.Errorf("Early break should stop at 3: got %d", count)
	}
}
