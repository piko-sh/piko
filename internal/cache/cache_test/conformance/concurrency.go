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
	"fmt"
	"sync"
	"testing"
	"time"

	"piko.sh/piko/internal/cache/cache_dto"
)

// runConcurrencyTests runs the concurrency test suite for cache behaviour.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache configuration to test.
func runConcurrencyTests(t *testing.T, config StringConfig) {
	t.Helper()

	t.Run("ConcurrentReads", func(t *testing.T) {
		t.Parallel()
		testConcurrentReads(t, config)
	})

	t.Run("ConcurrentWrites", func(t *testing.T) {
		t.Parallel()
		testConcurrentWrites(t, config)
	})

	t.Run("ConcurrentReadWrite", func(t *testing.T) {
		t.Parallel()
		testConcurrentReadWrite(t, config)
	})

	t.Run("ConcurrentHotKey", func(t *testing.T) {
		t.Parallel()
		testConcurrentHotKey(t, config)
	})

	t.Run("ConcurrentInvalidation", func(t *testing.T) {
		t.Parallel()
		testConcurrentInvalidation(t, config)
	})

	t.Run("ThunderingHerd", func(t *testing.T) {
		t.Parallel()
		testThunderingHerd(t, config)
	})
}

func testConcurrentReads(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	for i := range 100 {
		if err := cache.Set(ctx, fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i)); err != nil {
			t.Fatalf("Set key-%d failed: %v", i, err)
		}
	}

	const numGoroutines = 50
	const opsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for range numGoroutines {
		go func() {
			defer wg.Done()
			for i := range opsPerGoroutine {
				key := fmt.Sprintf("key-%d", i%100)
				_, _, _ = cache.GetIfPresent(ctx, key)
			}
		}()
	}

	wg.Wait()
}

func testConcurrentWrites(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	const numGoroutines = 50
	const opsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for g := range numGoroutines {
		go func(goroutineID int) {
			defer wg.Done()
			for i := range opsPerGoroutine {
				key := fmt.Sprintf("key-%d-%d", goroutineID, i)
				_ = cache.Set(ctx, key, fmt.Sprintf("value-%d", i))
			}
		}(g)
	}

	wg.Wait()
}

func testConcurrentReadWrite(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	const numGoroutines = 50
	const opsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2)

	for g := range numGoroutines {
		go func(goroutineID int) {
			defer wg.Done()
			for i := range opsPerGoroutine {
				key := fmt.Sprintf("key-%d", i%50)
				_ = cache.Set(ctx, key, fmt.Sprintf("value-%d-%d", goroutineID, i))
			}
		}(g)
	}

	for range numGoroutines {
		go func() {
			defer wg.Done()
			for i := range opsPerGoroutine {
				key := fmt.Sprintf("key-%d", i%50)
				_, _, _ = cache.GetIfPresent(ctx, key)
			}
		}()
	}

	wg.Wait()
}

func testConcurrentHotKey(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	const hotKey = "hot-key"
	if err := cache.Set(ctx, hotKey, "initial"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	const numGoroutines = 100
	const opsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for g := range numGoroutines {
		go func(goroutineID int) {
			defer wg.Done()
			for i := range opsPerGoroutine {
				if i%2 == 0 {
					_, _, _ = cache.GetIfPresent(ctx, hotKey)
				} else {
					_ = cache.Set(ctx, hotKey, fmt.Sprintf("value-%d", goroutineID))
				}
			}
		}(g)
	}

	wg.Wait()

	_, found, err := cache.GetIfPresent(ctx, hotKey)
	if err != nil {
		t.Fatalf("GetIfPresent failed: %v", err)
	}
	if !found {
		t.Error("Hot key should still be present after concurrent access")
	}
}

func testConcurrentInvalidation(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	const numGoroutines = 20

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3)

	for g := range numGoroutines {
		go func(goroutineID int) {
			defer wg.Done()
			for i := range 50 {
				_ = cache.Set(ctx, fmt.Sprintf("key-%d-%d", goroutineID, i), "value", "tag")
			}
		}(g)
	}

	for g := range numGoroutines {
		go func(goroutineID int) {
			defer wg.Done()
			for i := range 50 {
				_, _, _ = cache.GetIfPresent(ctx, fmt.Sprintf("key-%d-%d", goroutineID, i))
			}
		}(g)
	}

	for range numGoroutines {
		go func() {
			defer wg.Done()
			for range 10 {
				_, _ = cache.InvalidateByTags(ctx, "tag")
			}
		}()
	}

	wg.Wait()
}

// testThunderingHerd verifies that the cache prevents thundering herd problems.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache provider factory.
//
// Concurrency: Spawns 50 goroutines that simultaneously request the same key
// and verifies the loader is called at most 5 times.
func testThunderingHerd(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	const numGoroutines = 50

	var loadCount int
	var loadMu sync.Mutex

	start := make(chan struct{})

	loader := cache_dto.LoaderFunc[string, string](func(ctx context.Context, key string) (string, error) {
		loadMu.Lock()
		loadCount++
		loadMu.Unlock()

		time.Sleep(10 * time.Millisecond)

		return "loaded-value", nil
	})

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for range numGoroutines {
		go func() {
			defer wg.Done()
			<-start
			value, err := cache.Get(ctx, "shared-key", loader)
			if err != nil {
				t.Errorf("Get failed: %v", err)
			}
			if value != "loaded-value" {
				t.Errorf("Got wrong value: %q", value)
			}
		}()
	}

	close(start)
	wg.Wait()

	loadMu.Lock()
	count := loadCount
	loadMu.Unlock()

	if count > 5 {
		t.Errorf("Thundering herd protection failed: loader called %d times (expected <= 5)", count)
	}
}
