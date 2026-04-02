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
)

func TestOtter_ComputeWithTTL_SetsValueWithCustomTTL(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cache := createOtterCache(t, cache_dto.Options[string, int64]{
		MaximumSize: 100,
	})

	value, ok, err := cache.ComputeWithTTL(ctx, "counter", func(oldValue int64, found bool) cache_dto.ComputeResult[int64] {
		if found {
			t.Error("expected key to be absent initially")
		}
		return cache_dto.ComputeResult[int64]{
			Value:  1,
			Action: cache_dto.ComputeActionSet,
			TTL:    100 * time.Millisecond,
		}
	})
	if err != nil {
		t.Fatalf("unexpected error from ComputeWithTTL: %v", err)
	}

	if !ok {
		t.Fatal("expected operation to succeed")
	}
	Equal(t, value, int64(1), "value after first compute")

	retrieved, found, _ := cache.GetIfPresent(ctx, "counter")
	if !found {
		t.Fatal("expected key to be present after compute")
	}
	Equal(t, retrieved, int64(1), "retrieved value")
}

func TestOtter_ComputeWithTTL_ZeroTTLUsesDefault(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cache := createOtterCache(t, cache_dto.Options[string, string]{
		MaximumSize: 100,
	})

	value, ok, err := cache.ComputeWithTTL(ctx, "key", func(oldValue string, found bool) cache_dto.ComputeResult[string] {
		return cache_dto.ComputeResult[string]{
			Value:  "test-value",
			Action: cache_dto.ComputeActionSet,
			TTL:    0,
		}
	})
	if err != nil {
		t.Fatalf("unexpected error from ComputeWithTTL: %v", err)
	}

	if !ok {
		t.Fatal("expected operation to succeed")
	}
	Equal(t, value, "test-value", "value after compute")

	retrieved, found, _ := cache.GetIfPresent(ctx, "key")
	if !found {
		t.Fatal("expected key to be present")
	}
	Equal(t, retrieved, "test-value", "retrieved value")
}

func TestOtter_ComputeWithTTL_IncrementExistingWithoutChangingTTL(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cache := createOtterCache(t, cache_dto.Options[string, int64]{
		MaximumSize: 100,
	})

	_, ok, err := cache.ComputeWithTTL(ctx, "counter", func(oldValue int64, found bool) cache_dto.ComputeResult[int64] {
		return cache_dto.ComputeResult[int64]{
			Value:  1,
			Action: cache_dto.ComputeActionSet,
			TTL:    500 * time.Millisecond,
		}
	})
	if err != nil {
		t.Fatalf("unexpected error from first ComputeWithTTL: %v", err)
	}
	if !ok {
		t.Fatal("first compute failed")
	}

	value, ok, err := cache.ComputeWithTTL(ctx, "counter", func(oldValue int64, found bool) cache_dto.ComputeResult[int64] {
		if !found {
			t.Error("expected key to be present")
		}
		return cache_dto.ComputeResult[int64]{
			Value:  oldValue + 1,
			Action: cache_dto.ComputeActionSet,
			TTL:    0,
		}
	})
	if err != nil {
		t.Fatalf("unexpected error from second ComputeWithTTL: %v", err)
	}

	if !ok {
		t.Fatal("second compute failed")
	}
	Equal(t, value, int64(2), "incremented value")
}

func TestOtter_ComputeWithTTL_DeleteAction(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cache := createOtterCache(t, cache_dto.Options[string, string]{
		MaximumSize: 100,
	})

	_ = cache.Set(ctx, "key", "value")

	_, ok, err := cache.ComputeWithTTL(ctx, "key", func(oldValue string, found bool) cache_dto.ComputeResult[string] {
		if !found {
			t.Error("expected key to be present")
		}
		Equal(t, oldValue, "value", "old value")
		return cache_dto.ComputeResult[string]{
			Action: cache_dto.ComputeActionDelete,
		}
	})
	if err != nil {
		t.Fatalf("unexpected error from ComputeWithTTL: %v", err)
	}

	if ok {
		t.Error("expected ok to be false after delete")
	}

	_, found, _ := cache.GetIfPresent(ctx, "key")
	if found {
		t.Error("expected key to be deleted")
	}
}

func TestOtter_ComputeWithTTL_NoopAction(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cache := createOtterCache(t, cache_dto.Options[string, string]{
		MaximumSize: 100,
	})

	_ = cache.Set(ctx, "key", "original")

	value, ok, err := cache.ComputeWithTTL(ctx, "key", func(oldValue string, found bool) cache_dto.ComputeResult[string] {
		if !found {
			t.Error("expected key to be present")
		}
		return cache_dto.ComputeResult[string]{
			Value:  "ignored",
			Action: cache_dto.ComputeActionNoop,
		}
	})
	if err != nil {
		t.Fatalf("unexpected error from ComputeWithTTL: %v", err)
	}

	if !ok {
		t.Fatal("expected ok to be true (key still exists)")
	}
	Equal(t, value, "original", "value should be unchanged")
}

func TestOtter_ComputeWithTTL_ConcurrentOperations(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cache := createOtterCache(t, cache_dto.Options[string, int64]{
		MaximumSize: 100,
	})

	const numGoroutines = 50
	const incrementsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for range numGoroutines {
		go func() {
			defer wg.Done()
			for range incrementsPerGoroutine {
				_, _, _ = cache.ComputeWithTTL(ctx, "counter", func(oldValue int64, found bool) cache_dto.ComputeResult[int64] {
					var ttl time.Duration
					if !found {
						ttl = time.Hour
					}
					return cache_dto.ComputeResult[int64]{
						Value:  oldValue + 1,
						Action: cache_dto.ComputeActionSet,
						TTL:    ttl,
					}
				})
			}
		}()
	}

	wg.Wait()

	finalValue, found, _ := cache.GetIfPresent(ctx, "counter")
	if !found {
		t.Fatal("expected counter to be present")
	}

	expectedTotal := int64(numGoroutines * incrementsPerGoroutine)
	Equal(t, finalValue, expectedTotal, "final counter value")
}

func TestMock_ComputeWithTTL_SetsValueWithCustomTTL(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cache := createMockCache[string, int64](t)

	value, ok, err := cache.ComputeWithTTL(ctx, "counter", func(oldValue int64, found bool) cache_dto.ComputeResult[int64] {
		if found {
			t.Error("expected key to be absent initially")
		}
		return cache_dto.ComputeResult[int64]{
			Value:  42,
			Action: cache_dto.ComputeActionSet,
			TTL:    time.Hour,
		}
	})
	if err != nil {
		t.Fatalf("unexpected error from ComputeWithTTL: %v", err)
	}

	if !ok {
		t.Fatal("expected operation to succeed")
	}
	Equal(t, value, int64(42), "value after compute")

	retrieved, found, _ := cache.GetIfPresent(ctx, "counter")
	if !found {
		t.Fatal("expected key to be present")
	}
	Equal(t, retrieved, int64(42), "retrieved value")
}

func createMockCache[K comparable, V any](t *testing.T) *provider_otter.OtterAdapter[K, V] {
	t.Helper()

	cache, err := provider_otter.OtterProviderFactory(cache_dto.Options[K, V]{
		MaximumSize: 1000,
	})
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	t.Cleanup(func() {
		_ = cache.Close(context.Background())
	})

	otterCache, ok := cache.(*provider_otter.OtterAdapter[K, V])
	if !ok {
		t.Fatalf("unexpected cache type")
	}

	return otterCache
}
