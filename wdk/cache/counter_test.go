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

package cache_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"piko.sh/piko/internal/cache/cache_adapters/provider_otter"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/wdk/cache"
)

func createTestCache(t *testing.T) cache.Cache[string, int64] {
	t.Helper()

	cacheInstance, err := provider_otter.OtterProviderFactory(cache_dto.Options[string, int64]{
		MaximumSize: 1000,
	})
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	t.Cleanup(func() {
		_ = cacheInstance.Close(context.Background())
	})

	return cacheInstance
}

func TestIncrementWithExpiry_CreatesCounterWithTTL(t *testing.T) {
	t.Parallel()

	c := createTestCache(t)
	ctx := context.Background()

	value, ok, err := cache.IncrementWithExpiry(ctx, c, "counter", 1, time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected increment to succeed")
	}

	if value != 1 {
		t.Errorf("expected value 1, got %d", value)
	}

	retrieved, found, err := cache.GetCounter(ctx, c, "counter")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Fatal("expected counter to be present")
	}
	if retrieved != 1 {
		t.Errorf("expected retrieved value 1, got %d", retrieved)
	}
}

func TestIncrementWithExpiry_IncrementsExistingCounter(t *testing.T) {
	t.Parallel()

	c := createTestCache(t)
	ctx := context.Background()

	_, ok, err := cache.IncrementWithExpiry(ctx, c, "counter", 1, time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("first increment failed")
	}

	value, ok, err := cache.IncrementWithExpiry(ctx, c, "counter", 1, time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("second increment failed")
	}

	if value != 2 {
		t.Errorf("expected value 2, got %d", value)
	}

	value, ok, err = cache.IncrementWithExpiry(ctx, c, "counter", 5, time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("third increment failed")
	}

	if value != 7 {
		t.Errorf("expected value 7, got %d", value)
	}
}

func TestGetCounter_ReturnsCorrectValue(t *testing.T) {
	t.Parallel()

	c := createTestCache(t)
	ctx := context.Background()

	_, found, _ := cache.GetCounter(ctx, c, "missing")
	if found {
		t.Error("expected counter to be absent")
	}

	_, _, _ = cache.IncrementWithExpiry(ctx, c, "counter", 42, time.Hour)

	value, found, err := cache.GetCounter(ctx, c, "counter")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Fatal("expected counter to be present")
	}
	if value != 42 {
		t.Errorf("expected value 42, got %d", value)
	}
}

func TestResetCounter_RemovesCounter(t *testing.T) {
	t.Parallel()

	c := createTestCache(t)
	ctx := context.Background()

	_, _, _ = cache.IncrementWithExpiry(ctx, c, "counter", 10, time.Hour)

	_, found, _ := cache.GetCounter(ctx, c, "counter")
	if !found {
		t.Fatal("expected counter to exist before reset")
	}

	_ = cache.ResetCounter(ctx, c, "counter")

	_, found, _ = cache.GetCounter(ctx, c, "counter")
	if found {
		t.Error("expected counter to be removed after reset")
	}
}

func TestIncrementWithExpiry_ConcurrentIncrements(t *testing.T) {
	t.Parallel()

	c := createTestCache(t)
	ctx := context.Background()

	const numGoroutines = 100
	const incrementsPerGoroutine = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for range numGoroutines {
		go func() {
			defer wg.Done()
			for range incrementsPerGoroutine {
				_, _, _ = cache.IncrementWithExpiry(ctx, c, "counter", 1, time.Hour)
			}
		}()
	}

	wg.Wait()

	finalValue, found, err := cache.GetCounter(ctx, c, "counter")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Fatal("expected counter to exist")
	}

	expected := int64(numGoroutines * incrementsPerGoroutine)
	if finalValue != expected {
		t.Errorf("expected %d, got %d", expected, finalValue)
	}
}

func TestIncrementWithExpiry_MultipleCounters(t *testing.T) {
	t.Parallel()

	c := createTestCache(t)
	ctx := context.Background()

	_, _, _ = cache.IncrementWithExpiry(ctx, c, "counter-a", 1, time.Hour)
	_, _, _ = cache.IncrementWithExpiry(ctx, c, "counter-b", 10, time.Hour)
	_, _, _ = cache.IncrementWithExpiry(ctx, c, "counter-c", 100, time.Hour)

	_, _, _ = cache.IncrementWithExpiry(ctx, c, "counter-a", 1, time.Hour)
	_, _, _ = cache.IncrementWithExpiry(ctx, c, "counter-b", 5, time.Hour)
	_, _, _ = cache.IncrementWithExpiry(ctx, c, "counter-c", 25, time.Hour)

	valueA, _, _ := cache.GetCounter(ctx, c, "counter-a")
	valueB, _, _ := cache.GetCounter(ctx, c, "counter-b")
	valueC, _, _ := cache.GetCounter(ctx, c, "counter-c")

	if valueA != 2 {
		t.Errorf("counter-a: expected 2, got %d", valueA)
	}
	if valueB != 15 {
		t.Errorf("counter-b: expected 15, got %d", valueB)
	}
	if valueC != 125 {
		t.Errorf("counter-c: expected 125, got %d", valueC)
	}
}
