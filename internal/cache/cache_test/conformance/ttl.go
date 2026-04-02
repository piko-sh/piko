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
)

// runTTLTests runs the TTL-related test suite for the string cache.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which specifies the cache configuration to test.
func runTTLTests(t *testing.T, config StringConfig) {
	t.Helper()

	if config.AdvanceTime != nil {
		t.Run("SetWithTTL_Expiry", func(t *testing.T) {
			t.Parallel()
			testSetWithTTLExpiry(t, config)
		})

		t.Run("SetExpiresAfter", func(t *testing.T) {
			t.Parallel()
			testSetExpiresAfter(t, config)
		})
	} else {
		t.Run("SetWithTTL_Expiry_LazyExpiration", func(t *testing.T) {
			t.Parallel()
			testSetWithTTLExpiryLazy(t, config)
		})

		t.Run("SetExpiresAfter_LazyExpiration", func(t *testing.T) {
			t.Parallel()
			testSetExpiresAfterLazy(t, config)
		})
	}

	t.Run("SetWithTTL_BeforeExpiry", func(t *testing.T) {
		t.Parallel()
		testSetWithTTLBeforeExpiry(t, config)
	})

	if config.SupportsRefresh {
		t.Run("SetRefreshableAfter", func(t *testing.T) {
			t.Parallel()
			testSetRefreshableAfter(t, config)
		})
	}
}

// testSetWithTTLExpiry verifies that a cache entry expires after its TTL.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache factory and settings.
func testSetWithTTLExpiry(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	ttl := 100 * time.Millisecond

	err := cache.SetWithTTL(ctx, "key", "value", ttl)
	if err != nil {
		t.Errorf("SetWithTTL failed: %v", err)
	}

	_, found, err := cache.GetIfPresent(ctx, "key")
	if err != nil {
		t.Fatalf("GetIfPresent failed: %v", err)
	}
	if !found {
		t.Error("Key should be present immediately after SetWithTTL")
	}

	waitForExpiry(config, ttl+50*time.Millisecond)

	_, found, err = cache.GetIfPresent(ctx, "key")
	if err != nil {
		t.Fatalf("GetIfPresent failed: %v", err)
	}
	if found {
		t.Error("Key should be absent after TTL expires")
	}
}

// testSetWithTTLExpiryLazy tests that a cached entry expires after its TTL.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache factory for testing.
func testSetWithTTLExpiryLazy(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	ttl := 200 * time.Millisecond

	err := cache.SetWithTTL(ctx, "key", "value", ttl)
	if err != nil {
		t.Errorf("SetWithTTL failed: %v", err)
	}

	_, found, err := cache.GetIfPresent(ctx, "key")
	if err != nil {
		t.Fatalf("GetIfPresent failed: %v", err)
	}
	if !found {
		t.Error("Key should be present immediately after SetWithTTL")
	}

	time.Sleep(ttl + 100*time.Millisecond)

	_, found, err = cache.GetIfPresent(ctx, "key")
	if err != nil {
		t.Fatalf("GetIfPresent failed: %v", err)
	}
	if found {
		t.Error("Key should be absent after TTL expires")
	}
}

// testSetWithTTLBeforeExpiry verifies that a value set with a TTL can be
// retrieved before it expires.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache factory and test settings.
func testSetWithTTLBeforeExpiry(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	ttl := 1 * time.Hour

	err := cache.SetWithTTL(ctx, "key", "value", ttl)
	if err != nil {
		t.Errorf("SetWithTTL failed: %v", err)
	}

	got, found, err := cache.GetIfPresent(ctx, "key")
	if err != nil {
		t.Fatalf("GetIfPresent failed: %v", err)
	}
	if !found {
		t.Error("Key should be present")
	}
	if got != "value" {
		t.Errorf("Value mismatch: got %q, want %q", got, "value")
	}
}

// testSetExpiresAfter verifies that SetExpiresAfter causes a key to expire
// after the specified duration.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache configuration for testing.
func testSetExpiresAfter(t *testing.T, config StringConfig) {
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
		t.Error("Key should be present after Set")
	}

	ttl := 100 * time.Millisecond
	if err := cache.SetExpiresAfter(ctx, "key", ttl); err != nil {
		t.Fatalf("SetExpiresAfter failed: %v", err)
	}

	waitForExpiry(config, ttl+50*time.Millisecond)

	_, found, err = cache.GetIfPresent(ctx, "key")
	if err != nil {
		t.Fatalf("GetIfPresent failed: %v", err)
	}
	if found {
		t.Error("Key should be absent after SetExpiresAfter expires")
	}
}

// testSetExpiresAfterLazy verifies that SetExpiresAfter correctly expires a
// key after the specified duration when using lazy expiration.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache configuration to test.
func testSetExpiresAfterLazy(t *testing.T, config StringConfig) {
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
		t.Error("Key should be present after Set")
	}

	ttl := 200 * time.Millisecond
	if err := cache.SetExpiresAfter(ctx, "key", ttl); err != nil {
		t.Fatalf("SetExpiresAfter failed: %v", err)
	}

	time.Sleep(ttl + 100*time.Millisecond)

	_, found, err = cache.GetIfPresent(ctx, "key")
	if err != nil {
		t.Fatalf("GetIfPresent failed: %v", err)
	}
	if found {
		t.Error("Key should be absent after SetExpiresAfter expires")
	}
}

// testSetRefreshableAfter verifies that SetRefreshableAfter updates the
// refresh time for an existing key without affecting its value.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which provides the cache factory and settings.
func testSetRefreshableAfter(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	ctx := context.Background()

	if err := cache.Set(ctx, "key", "value"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	if err := cache.SetRefreshableAfter(ctx, "key", 100*time.Millisecond); err != nil {
		t.Fatalf("SetRefreshableAfter failed: %v", err)
	}

	got, found, err := cache.GetIfPresent(ctx, "key")
	if err != nil {
		t.Fatalf("GetIfPresent failed: %v", err)
	}
	if !found {
		t.Error("Key should be present after SetRefreshableAfter")
	}
	if got != "value" {
		t.Errorf("Value mismatch: got %q, want %q", got, "value")
	}
}
