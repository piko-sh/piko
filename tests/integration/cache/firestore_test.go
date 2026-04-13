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
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/cache"
)

func TestFirestore_BasicSetGet(t *testing.T) {
	skipIfNoFirestore(t)
	t.Parallel()

	namespace := uniqueKey(t, "basic") + ":"
	c := createFirestoreStringCache(t, namespace)
	ctx := testContext(t)

	err := c.Set(ctx, "hello", "world")
	require.NoError(t, err)

	val, ok, err := c.GetIfPresent(ctx, "hello")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "world", val)
}

func TestFirestore_EmptyStringValue(t *testing.T) {
	skipIfNoFirestore(t)
	t.Parallel()

	namespace := uniqueKey(t, "empty") + ":"
	c := createFirestoreStringCache(t, namespace)
	ctx := testContext(t)

	err := c.Set(ctx, "empty-key", "")
	require.NoError(t, err)

	val, ok, err := c.GetIfPresent(ctx, "empty-key")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "", val)
}

func TestFirestore_UnicodeValues(t *testing.T) {
	skipIfNoFirestore(t)
	t.Parallel()

	namespace := uniqueKey(t, "unicode") + ":"
	c := createFirestoreStringCache(t, namespace)
	ctx := testContext(t)

	testCases := []struct {
		key   string
		value string
	}{
		{"emoji", "Hello 🌍🎉"},
		{"cjk", "你好世界"},
		{"cyrillic", "Привет мир"},
		{"mixed", "Hello 世界 🌍 Мир"},
	}

	for _, tc := range testCases {
		err := c.Set(ctx, tc.key, tc.value)
		require.NoError(t, err, "setting %s", tc.key)

		val, ok, err := c.GetIfPresent(ctx, tc.key)
		require.NoError(t, err, "getting %s", tc.key)
		assert.True(t, ok, "key %s should be present", tc.key)
		assert.Equal(t, tc.value, val, "value mismatch for key %s", tc.key)
	}
}

func TestFirestore_Invalidate(t *testing.T) {
	skipIfNoFirestore(t)
	t.Parallel()

	namespace := uniqueKey(t, "invalidate") + ":"
	c := createFirestoreStringCache(t, namespace)
	ctx := testContext(t)

	err := c.Set(ctx, "to-delete", "value")
	require.NoError(t, err)

	err = c.Invalidate(ctx, "to-delete")
	require.NoError(t, err)

	_, ok, err := c.GetIfPresent(ctx, "to-delete")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestFirestore_TTL_Expiry(t *testing.T) {
	skipIfNoFirestore(t)
	t.Parallel()

	namespace := uniqueKey(t, "ttl") + ":"
	c := createFirestoreStringCache(t, namespace)
	ctx := testContext(t)

	err := c.SetWithTTL(ctx, "short-lived", "value", 1*time.Second)
	require.NoError(t, err)

	val, ok, err := c.GetIfPresent(ctx, "short-lived")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "value", val)

	time.Sleep(2 * time.Second)

	_, ok, err = c.GetIfPresent(ctx, "short-lived")
	require.NoError(t, err)
	assert.False(t, ok, "expired item should not be returned")
}

func TestFirestore_Compute(t *testing.T) {
	skipIfNoFirestore(t)
	t.Parallel()

	namespace := uniqueKey(t, "compute") + ":"
	c := createFirestoreStringCache(t, namespace)
	ctx := testContext(t)

	val, ok, err := c.Compute(ctx, "counter", func(oldValue string, found bool) (string, cache.ComputeAction) {
		if !found {
			return "1", cache.ComputeActionSet
		}
		return oldValue + "+1", cache.ComputeActionSet
	})
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "1", val)

	val, ok, err = c.Compute(ctx, "counter", func(oldValue string, found bool) (string, cache.ComputeAction) {
		if !found {
			return "1", cache.ComputeActionSet
		}
		return oldValue + "+1", cache.ComputeActionSet
	})
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "1+1", val)
}

func TestFirestore_BulkSetAndGet(t *testing.T) {
	skipIfNoFirestore(t)
	t.Parallel()

	namespace := uniqueKey(t, "bulk") + ":"
	c := createFirestoreStringCache(t, namespace)
	ctx := testContext(t)

	items := make(map[string]string, 50)
	for i := range 50 {
		items[fmt.Sprintf("bulk-key-%d", i)] = fmt.Sprintf("bulk-value-%d", i)
	}

	err := c.BulkSet(ctx, items)
	require.NoError(t, err)

	for k, expected := range items {
		got, ok, err := c.GetIfPresent(ctx, k)
		require.NoError(t, err)
		require.True(t, ok, "key %q should be present", k)
		assert.Equal(t, expected, got)
	}
}

func TestFirestore_InvalidateByTags(t *testing.T) {
	skipIfNoFirestore(t)
	t.Parallel()

	namespace := uniqueKey(t, "tags") + ":"
	c := createFirestoreStringCache(t, namespace)
	ctx := testContext(t)

	for i := range 5 {
		err := c.Set(ctx, fmt.Sprintf("group-a-%d", i), "value-a", "group-a")
		require.NoError(t, err)
	}
	for i := range 5 {
		err := c.Set(ctx, fmt.Sprintf("group-b-%d", i), "value-b", "group-b")
		require.NoError(t, err)
	}

	count, err := c.InvalidateByTags(ctx, "group-a")
	require.NoError(t, err)
	assert.Equal(t, 5, count)

	for i := range 5 {
		_, ok, err := c.GetIfPresent(ctx, fmt.Sprintf("group-a-%d", i))
		require.NoError(t, err)
		assert.False(t, ok, "group-a item should be invalidated")
	}

	for i := range 5 {
		val, ok, err := c.GetIfPresent(ctx, fmt.Sprintf("group-b-%d", i))
		require.NoError(t, err)
		assert.True(t, ok, "group-b item should still exist")
		assert.Equal(t, "value-b", val)
	}
}

func TestFirestore_Concurrency_ParallelSetGet(t *testing.T) {
	skipIfNoFirestore(t)
	t.Parallel()

	namespace := uniqueKey(t, "concurrent") + ":"
	c := createFirestoreStringCache(t, namespace)
	ctx := testContext(t)

	var wg sync.WaitGroup
	errCh := make(chan error, 1000)

	for g := range 10 {
		wg.Go(func() {
			goroutineID := g
			for i := range 50 {
				key := fmt.Sprintf("g%d-k%d", goroutineID, i)
				value := fmt.Sprintf("v%d-%d", goroutineID, i)

				if err := c.Set(ctx, key, value); err != nil {
					errCh <- err
					return
				}

				got, ok, err := c.GetIfPresent(ctx, key)
				if err != nil {
					errCh <- err
					return
				}
				if !ok {
					errCh <- fmt.Errorf("key %q missing after set", key)
					return
				}
				if got != value {
					errCh <- fmt.Errorf("key %q: expected %q, got %q", key, value, got)
					return
				}
			}
		})
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Error(err)
	}
}

func TestFirestore_Overwrite(t *testing.T) {
	skipIfNoFirestore(t)
	t.Parallel()

	namespace := uniqueKey(t, "overwrite") + ":"
	c := createFirestoreStringCache(t, namespace)
	ctx := testContext(t)

	err := c.Set(ctx, "key", "first")
	require.NoError(t, err)

	err = c.Set(ctx, "key", "second")
	require.NoError(t, err)

	val, ok, err := c.GetIfPresent(ctx, "key")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "second", val)
}

func TestFirestore_InvalidateAll(t *testing.T) {
	skipIfNoFirestore(t)
	t.Parallel()

	namespace := uniqueKey(t, "flush") + ":"
	c := createFirestoreStringCache(t, namespace)
	ctx := testContext(t)

	for i := range 10 {
		err := c.Set(ctx, fmt.Sprintf("flush-key-%d", i), "value")
		require.NoError(t, err)
	}

	err := c.InvalidateAll(ctx)
	require.NoError(t, err)

	for i := range 10 {
		_, ok, err := c.GetIfPresent(ctx, fmt.Sprintf("flush-key-%d", i))
		require.NoError(t, err)
		assert.False(t, ok, "all keys should be invalidated")
	}
}
