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

//go:build integration

package cache_integration_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/cache/cache_domain"
)

func TestConcurrency_ParallelSetGet(t *testing.T) {
	t.Parallel()

	c := createRedisStringCache(t, uniqueKey(t, "conc-setget")+":")

	const goroutines = 10
	const opsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := range goroutines {
		go func(id int) {
			defer wg.Done()

			for i := range opsPerGoroutine {
				key := fmt.Sprintf("g%d-k%d", id, i)
				value := fmt.Sprintf("v%d-%d", id, i)

				_ = c.Set(context.Background(), key, value)

				got, ok, _ := c.GetIfPresent(context.Background(), key)
				assert.True(t, ok, "key %q should be present", key)
				assert.Equal(t, value, got, "value mismatch for key %q", key)
			}
		}(g)
	}

	wg.Wait()
}

func TestConcurrency_HotKey(t *testing.T) {
	t.Parallel()

	c := createRedisStringCache(t, uniqueKey(t, "conc-hotkey")+":")

	const goroutines = 50
	hotKey := uniqueKey(t, "hot")

	_ = c.Set(context.Background(), hotKey, "initial-value")

	var wg sync.WaitGroup
	wg.Add(goroutines)

	errors := make(chan error, goroutines)

	for g := range goroutines {
		go func(id int) {
			defer wg.Done()

			value := fmt.Sprintf("writer-%d", id)
			_ = c.Set(context.Background(), hotKey, value)

			got, ok, _ := c.GetIfPresent(context.Background(), hotKey)
			if !ok {
				errors <- fmt.Errorf("goroutine %d: key not found after set", id)
				return
			}

			if len(got) == 0 {
				errors <- fmt.Errorf("goroutine %d: got empty value", id)
				return
			}
			_ = got
		}(g)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}

func TestConcurrency_ParallelSetGet_WithTransformers(t *testing.T) {
	t.Parallel()

	zstd := newZstdTransformer(t)
	crypto := newCryptoSetup(t)
	transformers := []cache_domain.CacheTransformerPort{zstd, crypto.transformer}
	namespace := uniqueKey(t, "conc-transformers") + ":"

	const goroutines = 10
	const opsPerGoroutine = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)

	errors := make(chan error, goroutines*opsPerGoroutine)

	for g := range goroutines {
		go func(id int) {
			defer wg.Done()

			for i := range opsPerGoroutine {
				key := fmt.Sprintf("g%d-k%d", id, i)
				original := fmt.Appendf(nil, "value-%d-%d-padding-%s",
					id, i, generateRepeatableText(128))

				wrapped := transformAndWrap(t, original, transformers)

				redisKey := rawRedisKeyForNamespace(namespace, key)
				putRawBytesToRedis(t, redisKey, wrapped, 1*time.Hour)

				raw := getRawBytesFromRedis(t, redisKey)
				recovered := reverseAndUnwrap(t, raw, transformers)

				if string(original) != string(recovered) {
					errors <- fmt.Errorf("goroutine %d, op %d: round-trip mismatch", id, i)
				}
			}
		}(g)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}

func TestConcurrency_ParallelSetGet_Cluster(t *testing.T) {
	skipIfNoCluster(t)
	t.Parallel()

	const goroutines = 10
	const opsPerGoroutine = 50
	namespace := uniqueKey(t, "conc-cluster") + ":"

	var wg sync.WaitGroup
	wg.Add(goroutines)

	errors := make(chan error, goroutines*opsPerGoroutine)

	for g := range goroutines {
		go func(id int) {
			defer wg.Done()

			ctx := t.Context()
			for i := range opsPerGoroutine {
				key := fmt.Sprintf("%sg%d-k%d", namespace, id, i)
				value := fmt.Sprintf("cluster-value-%d-%d", id, i)

				err := globalEnv.rawClusterClient.Set(ctx, key, value, 1*time.Hour).Err()
				if err != nil {
					errors <- fmt.Errorf("goroutine %d, op %d: set failed: %w", id, i, err)
					continue
				}

				got, err := globalEnv.rawClusterClient.Get(ctx, key).Result()
				if err != nil {
					errors <- fmt.Errorf("goroutine %d, op %d: get failed: %w", id, i, err)
					continue
				}

				if got != value {
					errors <- fmt.Errorf("goroutine %d, op %d: value mismatch: got %q, want %q", id, i, got, value)
				}
			}
		}(g)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}

func TestConcurrency_Compute_Contention(t *testing.T) {
	t.Parallel()

	c := createRedisStringCache(t, uniqueKey(t, "conc-compute")+":")

	const goroutines = 20
	key := uniqueKey(t, "compute-contention")

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := range goroutines {
		go func(id int) {
			defer wg.Done()

			value := fmt.Sprintf("writer-%d", id)
			_, _, _ = c.ComputeIfAbsent(context.Background(), key, func() string {
				return value
			})
		}(g)
	}

	wg.Wait()

	got, ok, _ := c.GetIfPresent(context.Background(), key)
	require.True(t, ok, "key should be present after compute contention")
	assert.NotEmpty(t, got, "value should not be empty")
}
