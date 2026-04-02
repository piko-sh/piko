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
	"testing"
	"time"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
)

// StringConfig holds configuration for cache conformance test suites.
// It specifies how to create cache instances and which features to test.
type StringConfig struct {
	// ProviderFactory creates a cache instance for testing.
	ProviderFactory func(t *testing.T, opts cache_dto.Options[string, string]) cache_domain.Cache[string, string]

	// AdvanceTime is a function to advance time during tests; nil uses real time.
	AdvanceTime func(d time.Duration)

	// Cleanup is called after all tests complete; nil means no cleanup is performed.
	Cleanup func()

	// SupportsSearch indicates whether this string option supports search filtering.
	SupportsSearch bool

	// SupportsTTL indicates whether the storage supports time-to-live expiry.
	SupportsTTL bool

	// SupportsIteration indicates whether the string type supports iteration.
	SupportsIteration bool

	// SupportsCompute indicates whether compute operations should be tested.
	SupportsCompute bool

	// SupportsMaximum indicates whether the string type supports a maximum length.
	SupportsMaximum bool

	// SupportsWeightedSize indicates whether the cache supports weighted item sizes.
	SupportsWeightedSize bool

	// SupportsRefresh indicates whether the store supports TTL refresh operations.
	SupportsRefresh bool

	// HonoursContextCancellation indicates whether the provider respects
	// context cancellation and deadlines. Distributed providers (Redis, Valkey)
	// set this to true. In-memory providers (Otter) leave it false because
	// their operations are non-blocking and succeed regardless of context state.
	HonoursContextCancellation bool
}

// ProductConfig defines the test configuration for cache conformance tests.
type ProductConfig struct {
	// ProviderFactory creates a cache instance for testing.
	ProviderFactory func(t *testing.T, opts cache_dto.Options[string, Product]) cache_domain.Cache[string, Product]

	// AdvanceTime advances the simulated clock by the given duration.
	AdvanceTime func(d time.Duration)

	// Cleanup is called after all tests complete; nil means no cleanup is
	// performed.
	Cleanup func()
}

// Product holds the details of a purchasable item in the catalogue.
type Product struct {
	// ID is the unique product identifier.
	ID string `json:"id"`

	// Name is the product's display name.
	Name string `json:"name"`

	// Description is the product's detailed description text.
	Description string `json:"description"`

	// Category is the product classification group.
	Category string `json:"category"`

	// Price is the item cost.
	Price float64 `json:"price"`

	// InStock indicates whether the product is available for purchase.
	InStock bool `json:"in_stock"`
}

// variableExpiryCalculator computes expiry times that vary per cache entry.
type variableExpiryCalculator[K comparable, V any] struct{}

// ExpireAfterCreate returns the duration until the entry should expire.
//
// Takes entry (cache_dto.Entry) which contains the expiry time in nanoseconds.
//
// Returns time.Duration which is the remaining time until expiry. Returns zero
// if the entry has no expiry set, or one nanosecond if already expired.
func (c *variableExpiryCalculator[K, V]) ExpireAfterCreate(entry cache_dto.Entry[K, V]) time.Duration {
	if entry.ExpiresAtNano <= 0 {
		return 0
	}
	remaining := entry.ExpiresAtNano - time.Now().UnixNano()
	if remaining <= 0 {
		return time.Nanosecond
	}
	return time.Duration(remaining)
}

func (c *variableExpiryCalculator[K, V]) ExpireAfterUpdate(entry cache_dto.Entry[K, V], _ V) time.Duration {
	return c.ExpireAfterCreate(entry)
}

// ExpireAfterRead returns the time until expiry after a cache entry is read.
//
// Takes entry (cache_dto.Entry[K, V]) which is the cache entry being read.
//
// Returns time.Duration which is the time until the entry expires.
func (c *variableExpiryCalculator[K, V]) ExpireAfterRead(entry cache_dto.Entry[K, V]) time.Duration {
	return c.ExpireAfterCreate(entry)
}

// RunStringSuite runs the complete string operations test suite.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which specifies which tests to run and provides
// the test fixtures.
func RunStringSuite(t *testing.T, config StringConfig) {
	t.Helper()

	if config.Cleanup != nil {
		t.Cleanup(config.Cleanup)
	}

	t.Run("CoreOps", func(t *testing.T) {
		runCoreOpsTests(t, config)
	})

	t.Run("BulkOps", func(t *testing.T) {
		runBulkOpsTests(t, config)
	})

	if config.SupportsCompute {
		t.Run("ComputeOps", func(t *testing.T) {
			runComputeOpsTests(t, config)
		})
	}

	if config.SupportsIteration {
		t.Run("Iteration", func(t *testing.T) {
			runIterationTests(t, config)
		})
	}

	if config.SupportsTTL {
		t.Run("TTL", func(t *testing.T) {
			runTTLTests(t, config)
		})
	}

	t.Run("Concurrency", func(t *testing.T) {
		runConcurrencyTests(t, config)
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		runContextCancellationTests(t, config)
	})
}

// RunSearchSuite runs the search test suite for a product.
//
// Takes t (*testing.T) which is the test context.
// Takes config (ProductConfig) which specifies the product to test and cleanup.
func RunSearchSuite(t *testing.T, config ProductConfig) {
	t.Helper()

	if config.Cleanup != nil {
		t.Cleanup(config.Cleanup)
	}

	t.Run("Search", func(t *testing.T) {
		runSearchTests(t, config)
	})
}

func defaultStringOptions() cache_dto.Options[string, string] {
	return cache_dto.Options[string, string]{
		MaximumSize:      1000,
		ExpiryCalculator: &variableExpiryCalculator[string, string]{},
	}
}

func defaultProductOptions() cache_dto.Options[string, Product] {
	return cache_dto.Options[string, Product]{
		MaximumSize:      1000,
		ExpiryCalculator: &variableExpiryCalculator[string, Product]{},
		SearchSchema: cache_dto.NewSearchSchema(
			cache_dto.TextField("name"),
			cache_dto.TextField("description"),
			cache_dto.SortableNumericField("price"),
			cache_dto.TagField("category"),
		),
	}
}

// waitForExpiry waits for a cache entry to expire.
//
// Takes config (StringConfig) which provides time control for testing.
// Takes d (time.Duration) which specifies the expiry duration to wait.
func waitForExpiry(config StringConfig, d time.Duration) {
	if config.AdvanceTime != nil {
		config.AdvanceTime(d)
	} else {
		time.Sleep(d + 50*time.Millisecond)
	}
}
