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
	"errors"
	"testing"

	"piko.sh/piko/internal/cache/cache_dto"
)

// runCoreOpsTests runs the core cache operation tests using the given config.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which specifies the cache configuration to test.
func runCoreOpsTests(t *testing.T, config StringConfig) {
	t.Helper()

	t.Run("GetIfPresent_Missing", func(t *testing.T) {
		t.Parallel()
		testGetIfPresentMissing(t, config)
	})

	t.Run("GetIfPresent_Found", func(t *testing.T) {
		t.Parallel()
		testGetIfPresentFound(t, config)
	})

	t.Run("Set_Basic", func(t *testing.T) {
		t.Parallel()
		testSetBasic(t, config)
	})

	t.Run("Set_Overwrite", func(t *testing.T) {
		t.Parallel()
		testSetOverwrite(t, config)
	})

	t.Run("Set_WithTags", func(t *testing.T) {
		t.Parallel()
		testSetWithTags(t, config)
	})

	t.Run("Get_WithLoader", func(t *testing.T) {
		t.Parallel()
		testGetWithLoader(t, config)
	})

	t.Run("Get_WithLoader_Cached", func(t *testing.T) {
		t.Parallel()
		testGetWithLoaderCached(t, config)
	})

	t.Run("Get_WithLoader_Error", func(t *testing.T) {
		t.Parallel()
		testGetWithLoaderError(t, config)
	})

	t.Run("Invalidate_Existing", func(t *testing.T) {
		t.Parallel()
		testInvalidateExisting(t, config)
	})

	t.Run("Invalidate_Missing", func(t *testing.T) {
		t.Parallel()
		testInvalidateMissing(t, config)
	})

	t.Run("GetEntry_Found", func(t *testing.T) {
		t.Parallel()
		testGetEntryFound(t, config)
	})

	t.Run("GetEntry_Missing", func(t *testing.T) {
		t.Parallel()
		testGetEntryMissing(t, config)
	})

	t.Run("ProbeEntry_Found", func(t *testing.T) {
		t.Parallel()
		testProbeEntryFound(t, config)
	})

	t.Run("EstimatedSize", func(t *testing.T) {
		t.Parallel()
		testEstimatedSize(t, config)
	})

	t.Run("Stats", func(t *testing.T) {
		t.Parallel()
		testStats(t, config)
	})

	t.Run("Close", func(t *testing.T) {
		t.Parallel()
		testClose(t, config)
	})

	if config.SupportsMaximum {
		t.Run("GetMaximum", func(t *testing.T) {
			t.Parallel()
			testGetMaximum(t, config)
		})

		t.Run("SetMaximum", func(t *testing.T) {
			t.Parallel()
			testSetMaximum(t, config)
		})
	}

	if config.SupportsWeightedSize {
		t.Run("WeightedSize", func(t *testing.T) {
			t.Parallel()
			testWeightedSize(t, config)
		})
	}
}

// testGetIfPresentMissing verifies that GetIfPresent returns false for a
// non-existent key.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache factory for the test.
func testGetIfPresentMissing(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	_, found, err := cache.GetIfPresent(ctx, "missing-key")
	if err != nil {
		t.Fatalf("GetIfPresent failed: %v", err)
	}
	if found {
		t.Error("GetIfPresent should return false for missing key")
	}
}

//nolint:dupl // different methods, similar structure
func testGetIfPresentFound(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	if err := cache.Set(ctx, "test-key", "test-value"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	got, found, err := cache.GetIfPresent(ctx, "test-key")
	if err != nil {
		t.Fatalf("GetIfPresent failed: %v", err)
	}
	if !found {
		t.Error("GetIfPresent should return true for present key")
	}
	if got != "test-value" {
		t.Errorf("GetIfPresent returned wrong value: got %q, want %q", got, "test-value")
	}
}

//nolint:dupl // different methods, similar structure
func testSetBasic(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	if err := cache.Set(ctx, "key", "value"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	got, found, err := cache.GetIfPresent(ctx, "key")
	if err != nil {
		t.Fatalf("GetIfPresent failed: %v", err)
	}
	if !found {
		t.Error("Set value should be retrievable")
	}
	if got != "value" {
		t.Errorf("Set value mismatch: got %q, want %q", got, "value")
	}
}

// testSetOverwrite verifies that setting a key twice overwrites the first value.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache factory and settings.
func testSetOverwrite(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	if err := cache.Set(ctx, "key", "value1"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	got, found, err := cache.GetIfPresent(ctx, "key")
	if err != nil {
		t.Fatalf("GetIfPresent failed: %v", err)
	}
	if !found {
		t.Error("First value should be present")
	}
	if got != "value1" {
		t.Errorf("First value mismatch: got %q, want %q", got, "value1")
	}

	if err := cache.Set(ctx, "key", "value2"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	got, found, err = cache.GetIfPresent(ctx, "key")
	if err != nil {
		t.Fatalf("GetIfPresent failed: %v", err)
	}
	if !found {
		t.Error("Second value should be present")
	}
	if got != "value2" {
		t.Errorf("Second value mismatch: got %q, want %q", got, "value2")
	}
}

// testSetWithTags verifies that values can be stored with associated tags.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache factory and settings.
func testSetWithTags(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	if err := cache.Set(ctx, "key", "value", "tag1", "tag2"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	got, found, err := cache.GetIfPresent(ctx, "key")
	if err != nil {
		t.Fatalf("GetIfPresent failed: %v", err)
	}
	if !found {
		t.Error("Tagged value should be present")
	}
	if got != "value" {
		t.Errorf("Tagged value mismatch: got %q, want %q", got, "value")
	}
}

// testGetWithLoader tests that the cache correctly invokes a loader function
// when retrieving a value.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache factory and settings.
func testGetWithLoader(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	loadCount := 0
	loader := cache_dto.LoaderFunc[string, string](func(ctx context.Context, key string) (string, error) {
		loadCount++
		return "loaded-" + key, nil
	})

	got, err := cache.Get(ctx, "key", loader)
	if err != nil {
		t.Errorf("Get with loader failed: %v", err)
	}
	if got != "loaded-key" {
		t.Errorf("Get with loader returned wrong value: got %q, want %q", got, "loaded-key")
	}
	if loadCount != 1 {
		t.Errorf("Loader should be called once: got %d", loadCount)
	}
}

// testGetWithLoaderCached verifies that a cached value is returned without
// calling the loader again.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache factory and settings.
func testGetWithLoaderCached(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	loadCount := 0
	loader := cache_dto.LoaderFunc[string, string](func(ctx context.Context, key string) (string, error) {
		loadCount++
		return "loaded-" + key, nil
	})

	_, err := cache.Get(ctx, "key", loader)
	if err != nil {
		t.Errorf("First Get failed: %v", err)
	}

	got, err := cache.Get(ctx, "key", loader)
	if err != nil {
		t.Errorf("Second Get failed: %v", err)
	}
	if got != "loaded-key" {
		t.Errorf("Second Get returned wrong value: got %q, want %q", got, "loaded-key")
	}
	if loadCount != 1 {
		t.Errorf("Loader should only be called once: got %d", loadCount)
	}
}

// testGetWithLoaderError verifies that Get returns errors from the loader.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache configuration for testing.
func testGetWithLoaderError(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	expectedErr := errors.New("loader error")
	loader := cache_dto.LoaderFunc[string, string](func(ctx context.Context, key string) (string, error) {
		return "", expectedErr
	})

	_, err := cache.Get(ctx, "key", loader)
	if err == nil {
		t.Error("Get should return error from loader")
	}
}

// testInvalidateExisting verifies that invalidating an existing key removes it
// from the cache.
//
// Takes t (*testing.T) which provides testing utilities.
// Takes config (StringConfig) which provides the cache factory and settings.
func testInvalidateExisting(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	if err := cache.Set(ctx, "key", "value"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	_, found, err := cache.GetIfPresent(ctx, "key")
	if err != nil {
		t.Fatalf("GetIfPresent failed: %v", err)
	}
	if !found {
		t.Error("Value should be present before invalidation")
	}

	if err := cache.Invalidate(ctx, "key"); err != nil {
		t.Fatalf("Invalidate failed: %v", err)
	}

	_, found, err = cache.GetIfPresent(ctx, "key")
	if err != nil {
		t.Fatalf("GetIfPresent failed: %v", err)
	}
	if found {
		t.Error("Value should be absent after invalidation")
	}
}

// testInvalidateMissing verifies that invalidating a non-existent key does not
// cause an error.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache factory and settings.
func testInvalidateMissing(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	if err := cache.Invalidate(ctx, "missing-key"); err != nil {
		t.Fatalf("Invalidate failed: %v", err)
	}
}

//nolint:dupl // different methods, similar structure
func testGetEntryFound(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	if err := cache.Set(ctx, "key", "value"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	entry, found, err := cache.GetEntry(ctx, "key")
	if err != nil {
		t.Fatalf("GetEntry failed: %v", err)
	}
	if !found {
		t.Error("GetEntry should find existing key")
	}
	if entry.Key != "key" {
		t.Errorf("Entry key mismatch: got %q, want %q", entry.Key, "key")
	}
	if entry.Value != "value" {
		t.Errorf("Entry value mismatch: got %q, want %q", entry.Value, "value")
	}
}

// testGetEntryMissing verifies that GetEntry returns not found for a missing key.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache factory to test.
func testGetEntryMissing(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	_, found, err := cache.GetEntry(ctx, "missing-key")
	if err != nil {
		t.Fatalf("GetEntry failed: %v", err)
	}
	if found {
		t.Error("GetEntry should not find missing key")
	}
}

//nolint:dupl // different methods, similar structure
func testProbeEntryFound(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	if err := cache.Set(ctx, "key", "value"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	entry, found, err := cache.ProbeEntry(ctx, "key")
	if err != nil {
		t.Fatalf("ProbeEntry failed: %v", err)
	}
	if !found {
		t.Error("ProbeEntry should find existing key")
	}
	if entry.Key != "key" {
		t.Errorf("Entry key mismatch: got %q, want %q", entry.Key, "key")
	}
	if entry.Value != "value" {
		t.Errorf("Entry value mismatch: got %q, want %q", entry.Value, "value")
	}
}

// testEstimatedSize verifies that the cache reports its estimated size correctly.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache factory and test settings.
func testEstimatedSize(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	initialSize := cache.EstimatedSize()
	if initialSize != 0 {
		t.Errorf("Initial size should be 0: got %d", initialSize)
	}

	if err := cache.Set(ctx, "key", "value"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	size := cache.EstimatedSize()
	if size < 1 {
		t.Errorf("Size should be at least 1 after Set: got %d", size)
	}
}

// testStats verifies that cache statistics return valid values.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache factory to test.
func testStats(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())

	stats := cache.Stats()

	ratio := stats.HitRatio()
	if ratio < 0 || ratio > 1 {
		t.Errorf("Invalid hit ratio: %f", ratio)
	}
}

// testClose tests that a cache can be closed after storing a value.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache factory and settings.
func testClose(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	if err := cache.Set(ctx, "key", "value"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	if err := cache.Close(ctx); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

// testGetMaximum verifies that a cache returns its configured maximum size.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache factory to test.
func testGetMaximum(t *testing.T, config StringConfig) {
	t.Helper()

	opts := defaultStringOptions()
	opts.MaximumSize = 100
	cache := config.ProviderFactory(t, opts)

	maxSize := cache.GetMaximum()
	if maxSize != 100 {
		t.Errorf("GetMaximum should return configured value: got %d, want 100", maxSize)
	}
}

// testSetMaximum verifies that SetMaximum updates the cache size limit.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache factory for testing.
func testSetMaximum(t *testing.T, config StringConfig) {
	t.Helper()

	opts := defaultStringOptions()
	opts.MaximumSize = 100
	cache := config.ProviderFactory(t, opts)

	cache.SetMaximum(200)

	maxSize := cache.GetMaximum()
	if maxSize != 200 {
		t.Errorf("SetMaximum should change value: got %d, want 200", maxSize)
	}
}

// testWeightedSize verifies that a new cache has a weighted size of zero.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache factory to test.
func testWeightedSize(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())

	size := cache.WeightedSize()

	if size != 0 {
		t.Errorf("Initial weighted size should be 0: got %d", size)
	}
}
