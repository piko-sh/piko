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
	"testing"

	"piko.sh/piko/internal/cache/cache_dto"
)

// runContextCancellationTests runs the context cancellation test suite.
//
// For in-memory providers, all operations should succeed even with a cancelled
// context because they are non-blocking. For distributed providers, operations
// should return context errors.
//
// Takes t (*testing.T) which is the test context.
// Takes config (StringConfig) which specifies the cache configuration to test.
func runContextCancellationTests(t *testing.T, config StringConfig) {
	t.Helper()

	if config.HonoursContextCancellation {
		t.Run("Set_CancelledContext_ReturnsError", func(t *testing.T) {
			t.Parallel()
			testSetCancelledContextReturnsError(t, config)
		})

		t.Run("GetIfPresent_CancelledContext_ReturnsError", func(t *testing.T) {
			t.Parallel()
			testGetIfPresentCancelledContextReturnsError(t, config)
		})

		t.Run("Get_CancelledContext_ReturnsError", func(t *testing.T) {
			t.Parallel()
			testGetCancelledContextReturnsError(t, config)
		})

		t.Run("Invalidate_CancelledContext_ReturnsError", func(t *testing.T) {
			t.Parallel()
			testInvalidateCancelledContextReturnsError(t, config)
		})

		t.Run("InvalidateAll_CancelledContext_ReturnsError", func(t *testing.T) {
			t.Parallel()
			testInvalidateAllCancelledContextReturnsError(t, config)
		})

		t.Run("InvalidateByTags_CancelledContext_ReturnsError", func(t *testing.T) {
			t.Parallel()
			testInvalidateByTagsCancelledContextReturnsError(t, config)
		})

		t.Run("GetEntry_CancelledContext_ReturnsError", func(t *testing.T) {
			t.Parallel()
			testGetEntryCancelledContextReturnsError(t, config)
		})

		t.Run("Close_CancelledContext_ReturnsError", func(t *testing.T) {
			t.Parallel()
			testCloseCancelledContextReturnsError(t, config)
		})

		t.Run("Set_DeadlineExceeded_ReturnsError", func(t *testing.T) {
			t.Parallel()
			testSetDeadlineExceededReturnsError(t, config)
		})

		t.Run("Get_DeadlineExceeded_ReturnsError", func(t *testing.T) {
			t.Parallel()
			testGetDeadlineExceededReturnsError(t, config)
		})
	} else {
		t.Run("Set_CancelledContext_StillSucceeds", func(t *testing.T) {
			t.Parallel()
			testSetCancelledContextStillSucceeds(t, config)
		})

		t.Run("GetIfPresent_CancelledContext_StillSucceeds", func(t *testing.T) {
			t.Parallel()
			testGetIfPresentCancelledContextStillSucceeds(t, config)
		})

		t.Run("Get_CancelledContext_StillSucceeds", func(t *testing.T) {
			t.Parallel()
			testGetCancelledContextStillSucceeds(t, config)
		})

		t.Run("Invalidate_CancelledContext_StillSucceeds", func(t *testing.T) {
			t.Parallel()
			testInvalidateCancelledContextStillSucceeds(t, config)
		})

		t.Run("InvalidateAll_CancelledContext_StillSucceeds", func(t *testing.T) {
			t.Parallel()
			testInvalidateAllCancelledContextStillSucceeds(t, config)
		})

		t.Run("GetEntry_CancelledContext_StillSucceeds", func(t *testing.T) {
			t.Parallel()
			testGetEntryCancelledContextStillSucceeds(t, config)
		})

		t.Run("Close_CancelledContext_StillSucceeds", func(t *testing.T) {
			t.Parallel()
			testCloseCancelledContextStillSucceeds(t, config)
		})
	}
}

func testSetCancelledContextStillSucceeds(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	err := cache.Set(ctx, "key", "value")
	if err != nil {
		t.Fatalf("in-memory Set should succeed with cancelled context: %v", err)
	}

	value, found, err := cache.GetIfPresent(ctx, "key")
	if err != nil {
		t.Fatalf("in-memory GetIfPresent should succeed with cancelled context: %v", err)
	}
	if !found {
		t.Error("value should be retrievable after Set with cancelled context")
	}
	if value != "value" {
		t.Errorf("wrong value: got %q, want %q", value, "value")
	}
}

func testGetIfPresentCancelledContextStillSucceeds(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	liveCtx := context.Background()

	if err := cache.Set(liveCtx, "key", "value"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	value, found, err := cache.GetIfPresent(ctx, "key")
	if err != nil {
		t.Fatalf("in-memory GetIfPresent should succeed with cancelled context: %v", err)
	}
	if !found {
		t.Error("value should be found")
	}
	if value != "value" {
		t.Errorf("wrong value: got %q, want %q", value, "value")
	}
}

func testGetCancelledContextStillSucceeds(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	loader := cache_dto.LoaderFunc[string, string](func(_ context.Context, key string) (string, error) {
		return "loaded-" + key, nil
	})

	value, err := cache.Get(ctx, "key", loader)
	if err != nil {
		t.Fatalf("in-memory Get should succeed with cancelled context: %v", err)
	}
	if value != "loaded-key" {
		t.Errorf("wrong value: got %q, want %q", value, "loaded-key")
	}
}

func testInvalidateCancelledContextStillSucceeds(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	liveCtx := context.Background()

	if err := cache.Set(liveCtx, "key", "value"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	if err := cache.Invalidate(ctx, "key"); err != nil {
		t.Fatalf("in-memory Invalidate should succeed with cancelled context: %v", err)
	}

	_, found, _ := cache.GetIfPresent(liveCtx, "key")
	if found {
		t.Error("key should be absent after invalidation")
	}
}

func testInvalidateAllCancelledContextStillSucceeds(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	liveCtx := context.Background()

	if err := cache.Set(liveCtx, "key1", "value1"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	if err := cache.Set(liveCtx, "key2", "value2"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	if err := cache.InvalidateAll(ctx); err != nil {
		t.Fatalf("in-memory InvalidateAll should succeed with cancelled context: %v", err)
	}

	size := cache.EstimatedSize()
	if size != 0 {
		t.Errorf("cache should be empty after InvalidateAll: got size %d", size)
	}
}

func testGetEntryCancelledContextStillSucceeds(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())
	liveCtx := context.Background()

	if err := cache.Set(liveCtx, "key", "value"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	entry, found, err := cache.GetEntry(ctx, "key")
	if err != nil {
		t.Fatalf("in-memory GetEntry should succeed with cancelled context: %v", err)
	}
	if !found {
		t.Error("entry should be found")
	}
	if entry.Key != "key" || entry.Value != "value" {
		t.Errorf("wrong entry: got key=%q value=%q", entry.Key, entry.Value)
	}
}

func testCloseCancelledContextStillSucceeds(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	if err := cache.Close(ctx); err != nil {
		t.Fatalf("in-memory Close should succeed with cancelled context: %v", err)
	}
}

func testSetCancelledContextReturnsError(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	err := cache.Set(ctx, "key", "value")
	if err == nil {
		t.Error("distributed Set should return error with cancelled context")
	}
}

func testGetIfPresentCancelledContextReturnsError(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	_, _, err := cache.GetIfPresent(ctx, "key")
	if err == nil {
		t.Error("distributed GetIfPresent should return error with cancelled context")
	}
}

func testGetCancelledContextReturnsError(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	loader := cache_dto.LoaderFunc[string, string](func(_ context.Context, key string) (string, error) {
		return "loaded-" + key, nil
	})

	_, err := cache.Get(ctx, "key", loader)
	if err == nil {
		t.Error("distributed Get should return error with cancelled context")
	}
}

func testInvalidateCancelledContextReturnsError(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	err := cache.Invalidate(ctx, "key")
	if err == nil {
		t.Error("distributed Invalidate should return error with cancelled context")
	}
}

func testInvalidateAllCancelledContextReturnsError(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	err := cache.InvalidateAll(ctx)
	if err == nil {
		t.Error("distributed InvalidateAll should return error with cancelled context")
	}
}

func testInvalidateByTagsCancelledContextReturnsError(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	_, err := cache.InvalidateByTags(ctx, "tag1")
	if err == nil {
		t.Error("distributed InvalidateByTags should return error with cancelled context")
	}
}

func testGetEntryCancelledContextReturnsError(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	_, _, err := cache.GetEntry(ctx, "key")
	if err == nil {
		t.Error("distributed GetEntry should return error with cancelled context")
	}
}

func testCloseCancelledContextReturnsError(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	err := cache.Close(ctx)
	if err == nil {
		t.Error("distributed Close should return error with cancelled context")
	}
}

func testSetDeadlineExceededReturnsError(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())

	ctx, cancel := context.WithTimeoutCause(context.Background(), 0, fmt.Errorf("test: simulating expired deadline"))
	defer cancel()

	err := cache.Set(ctx, "key", "value")
	if err == nil {
		t.Error("distributed Set should return error with expired context")
	}
}

func testGetDeadlineExceededReturnsError(t *testing.T, config StringConfig) {
	t.Helper()

	cache := config.ProviderFactory(t, defaultStringOptions())

	ctx, cancel := context.WithTimeoutCause(context.Background(), 0, fmt.Errorf("test: simulating expired deadline"))
	defer cancel()

	loader := cache_dto.LoaderFunc[string, string](func(_ context.Context, key string) (string, error) {
		return "loaded-" + key, nil
	})

	_, err := cache.Get(ctx, "key", loader)
	if err == nil {
		t.Error("distributed Get should return error with expired context")
	}
}
