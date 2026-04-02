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
	"testing"
	"time"

	"piko.sh/piko/internal/cache/cache_dto"
)

// runComputeOpsTests runs all compute operation tests for the string cache.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache configuration to test.
func runComputeOpsTests(t *testing.T, config StringConfig) {
	t.Helper()

	t.Run("Compute_CreateNew", func(t *testing.T) {
		t.Parallel()
		testComputeCreateNew(t, config)
	})

	t.Run("Compute_UpdateExisting", func(t *testing.T) {
		t.Parallel()
		testComputeUpdateExisting(t, config)
	})

	t.Run("Compute_Delete", func(t *testing.T) {
		t.Parallel()
		testComputeDelete(t, config)
	})

	t.Run("Compute_Noop", func(t *testing.T) {
		t.Parallel()
		testComputeNoop(t, config)
	})

	t.Run("ComputeIfAbsent_Absent", func(t *testing.T) {
		t.Parallel()
		testComputeIfAbsentAbsent(t, config)
	})

	t.Run("ComputeIfAbsent_Present", func(t *testing.T) {
		t.Parallel()
		testComputeIfAbsentPresent(t, config)
	})

	t.Run("ComputeIfPresent_Absent", func(t *testing.T) {
		t.Parallel()
		testComputeIfPresentAbsent(t, config)
	})

	t.Run("ComputeIfPresent_Present", func(t *testing.T) {
		t.Parallel()
		testComputeIfPresentPresent(t, config)
	})

	t.Run("ComputeWithTTL", func(t *testing.T) {
		t.Parallel()
		testComputeWithTTL(t, config)
	})
}

// testComputeCreateNew verifies that Compute creates a new cache entry when
// the key does not exist.
//
// Takes t (*testing.T) which provides test control and error reporting.
// Takes config (StringConfig) which supplies the cache provider factory.
func testComputeCreateNew(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	value, present, err := cache.Compute(ctx, "key", func(oldValue string, found bool) (string, cache_dto.ComputeAction) {
		if found {
			t.Error("Key should not exist yet")
		}
		return "new-value", cache_dto.ComputeActionSet
	})
	if err != nil {
		t.Fatalf("Compute failed: %v", err)
	}

	if !present {
		t.Error("Compute with Set should return present=true")
	}
	if value != "new-value" {
		t.Errorf("Compute returned wrong value: got %q, want %q", value, "new-value")
	}

	got, found, err := cache.GetIfPresent(ctx, "key")
	if err != nil {
		t.Fatalf("GetIfPresent failed: %v", err)
	}
	if !found {
		t.Error("Key should be present after Compute with Set")
	}
	if got != "new-value" {
		t.Errorf("Stored value mismatch: got %q, want %q", got, "new-value")
	}
}

// testComputeUpdateExisting verifies that Compute correctly updates an
// existing cache entry.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache configuration to test.
func testComputeUpdateExisting(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	if err := cache.Set(ctx, "key", "original"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	value, present, err := cache.Compute(ctx, "key", func(oldValue string, found bool) (string, cache_dto.ComputeAction) {
		if !found {
			t.Error("Key should exist")
		}
		if oldValue != "original" {
			t.Errorf("Old value mismatch: got %q, want %q", oldValue, "original")
		}
		return "updated", cache_dto.ComputeActionSet
	})
	if err != nil {
		t.Fatalf("Compute failed: %v", err)
	}

	if !present {
		t.Error("Compute with Set should return present=true")
	}
	if value != "updated" {
		t.Errorf("Compute returned wrong value: got %q, want %q", value, "updated")
	}
}

// testComputeDelete verifies that Compute with Delete action removes a key.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache provider factory.
func testComputeDelete(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	if err := cache.Set(ctx, "key", "value"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	_, present, err := cache.Compute(ctx, "key", func(oldValue string, found bool) (string, cache_dto.ComputeAction) {
		return "", cache_dto.ComputeActionDelete
	})
	if err != nil {
		t.Fatalf("Compute failed: %v", err)
	}

	if present {
		t.Error("Compute with Delete should return present=false")
	}

	_, found, err := cache.GetIfPresent(ctx, "key")
	if err != nil {
		t.Fatalf("GetIfPresent failed: %v", err)
	}
	if found {
		t.Error("Key should be absent after Compute with Delete")
	}
}

// testComputeNoop verifies that Compute with Noop action leaves the value
// unchanged.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache factory and settings.
func testComputeNoop(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	if err := cache.Set(ctx, "key", "original"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	value, present, err := cache.Compute(ctx, "key", func(oldValue string, found bool) (string, cache_dto.ComputeAction) {
		return "ignored", cache_dto.ComputeActionNoop
	})
	if err != nil {
		t.Fatalf("Compute failed: %v", err)
	}

	if !present {
		t.Error("Compute with Noop should return present=true for existing key")
	}
	if value != "original" {
		t.Errorf("Compute with Noop should return original value: got %q, want %q", value, "original")
	}
}

// testComputeIfAbsentAbsent verifies that ComputeIfAbsent calls the compute
// function and returns the computed value when the key is absent.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache configuration to test.
func testComputeIfAbsentAbsent(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	computeCount := 0
	value, computed, err := cache.ComputeIfAbsent(ctx, "key", func() string {
		computeCount++
		return "computed"
	})
	if err != nil {
		t.Fatalf("ComputeIfAbsent failed: %v", err)
	}

	if !computed {
		t.Error("ComputeIfAbsent should return computed=true for absent key")
	}
	if value != "computed" {
		t.Errorf("ComputeIfAbsent returned wrong value: got %q, want %q", value, "computed")
	}
	if computeCount != 1 {
		t.Errorf("Compute function should be called once: got %d", computeCount)
	}
}

// testComputeIfAbsentPresent tests that ComputeIfAbsent returns the existing
// value without calling the compute function when the key is already present.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache factory and settings.
func testComputeIfAbsentPresent(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	if err := cache.Set(ctx, "key", "existing"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	computeCount := 0
	value, _, err := cache.ComputeIfAbsent(ctx, "key", func() string {
		computeCount++
		return "computed"
	})
	if err != nil {
		t.Fatalf("ComputeIfAbsent failed: %v", err)
	}

	if value != "existing" {
		t.Errorf("ComputeIfAbsent should return existing value: got %q, want %q", value, "existing")
	}
	if computeCount != 0 {
		t.Errorf("Compute function should not be called for present key: got %d", computeCount)
	}
}

// testComputeIfPresentAbsent verifies that ComputeIfPresent does not call the
// compute function when the key is absent from the cache.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache factory and test settings.
func testComputeIfPresentAbsent(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	computeCount := 0
	_, present, err := cache.ComputeIfPresent(ctx, "key", func(oldValue string) (string, cache_dto.ComputeAction) {
		computeCount++
		return "computed", cache_dto.ComputeActionSet
	})
	if err != nil {
		t.Fatalf("ComputeIfPresent failed: %v", err)
	}

	if present {
		t.Error("ComputeIfPresent should return present=false for absent key")
	}
	if computeCount != 0 {
		t.Errorf("Compute function should not be called: got %d", computeCount)
	}
}

// testComputeIfPresentPresent verifies that ComputeIfPresent calls the
// compute function and updates the value when the key exists.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache factory and settings.
func testComputeIfPresentPresent(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	if err := cache.Set(ctx, "key", "existing"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	computeCount := 0
	value, present, err := cache.ComputeIfPresent(ctx, "key", func(oldValue string) (string, cache_dto.ComputeAction) {
		computeCount++
		if oldValue != "existing" {
			t.Errorf("Old value mismatch: got %q, want %q", oldValue, "existing")
		}
		return "updated", cache_dto.ComputeActionSet
	})
	if err != nil {
		t.Fatalf("ComputeIfPresent failed: %v", err)
	}

	if !present {
		t.Error("ComputeIfPresent should return present=true for present key")
	}
	if value != "updated" {
		t.Errorf("ComputeIfPresent returned wrong value: got %q, want %q", value, "updated")
	}
	if computeCount != 1 {
		t.Errorf("Compute function should be called once: got %d", computeCount)
	}
}

// testComputeWithTTL verifies that ComputeWithTTL correctly stores a value
// with a custom TTL.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache configuration to test.
func testComputeWithTTL(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	value, present, err := cache.ComputeWithTTL(ctx, "key", func(oldValue string, found bool) cache_dto.ComputeResult[string] {
		return cache_dto.ComputeResult[string]{
			Value:  "computed",
			Action: cache_dto.ComputeActionSet,
			TTL:    5 * time.Second,
		}
	})
	if err != nil {
		t.Fatalf("ComputeWithTTL failed: %v", err)
	}

	if !present {
		t.Error("ComputeWithTTL with Set should return present=true")
	}
	if value != "computed" {
		t.Errorf("ComputeWithTTL returned wrong value: got %q, want %q", value, "computed")
	}

	got, found, err := cache.GetIfPresent(ctx, "key")
	if err != nil {
		t.Fatalf("GetIfPresent failed: %v", err)
	}
	if !found {
		t.Error("Key should be present after ComputeWithTTL")
	}
	if got != "computed" {
		t.Errorf("Stored value mismatch: got %q, want %q", got, "computed")
	}
}
