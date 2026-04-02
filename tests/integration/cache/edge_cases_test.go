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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEdge_EmptyStringValue(t *testing.T) {
	t.Parallel()

	c := createRedisStringCache(t, uniqueKey(t, "edge-empty")+":")

	ctx := context.Background()
	key := uniqueKey(t, "empty")
	require.NoError(t, c.Set(ctx, key, ""))

	got, ok, err := c.GetIfPresent(ctx, key)
	require.NoError(t, err)
	require.True(t, ok, "key should be present")
	assert.Equal(t, "", got, "empty string should round-trip")
}

func TestEdge_SingleByteValue(t *testing.T) {
	t.Parallel()

	c := createRedisStringCache(t, uniqueKey(t, "edge-single")+":")

	ctx := context.Background()
	key := uniqueKey(t, "single")
	require.NoError(t, c.Set(ctx, key, "x"))

	got, ok, err := c.GetIfPresent(ctx, key)
	require.NoError(t, err)
	require.True(t, ok, "key should be present")
	assert.Equal(t, "x", got, "single byte value should round-trip")
}

func TestEdge_VeryLargeValue(t *testing.T) {
	t.Parallel()

	c := createRedisStringCache(t, uniqueKey(t, "edge-large")+":")

	ctx := context.Background()
	largeValue := generateRepeatableText(2 * 1024 * 1024)

	key := uniqueKey(t, "large")
	require.NoError(t, c.Set(ctx, key, largeValue))

	got, ok, err := c.GetIfPresent(ctx, key)
	require.NoError(t, err)
	require.True(t, ok, "key should be present")
	assert.Equal(t, largeValue, got, "2MB value should round-trip")
}

func TestEdge_SpecialCharactersInKeys(t *testing.T) {
	t.Parallel()

	c := createRedisStringCache(t, uniqueKey(t, "edge-keys")+":")

	testCases := []struct {
		name string
		key  string
	}{
		{name: "spaces", key: "key with spaces"},
		{name: "unicode", key: "key-\u4F60\u597D-\U0001F600"},
		{name: "colons", key: "a:b:c:d"},
		{name: "slashes", key: "path/to/resource"},
		{name: "dots", key: "config.setting.value"},
		{name: "brackets", key: "items[0].name"},
	}

	ctx := context.Background()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			value := "value-for-" + tc.name
			require.NoError(t, c.Set(ctx, tc.key, value))

			got, ok, err := c.GetIfPresent(ctx, tc.key)
			require.NoError(t, err)
			require.True(t, ok, "key %q should be present", tc.key)
			assert.Equal(t, value, got, "value mismatch for key %q", tc.key)
		})
	}
}

func TestEdge_UnicodeValues(t *testing.T) {
	t.Parallel()

	c := createRedisStringCache(t, uniqueKey(t, "edge-unicode")+":")

	testCases := []struct {
		name  string
		value string
	}{
		{name: "emoji", value: "\U0001F600\U0001F680\U0001F4A5\U0001F31F\U0001F308"},
		{name: "cjk", value: "\u4F60\u597D\u4E16\u754C\uFF01"},
		{name: "cyrillic", value: "\u041F\u0440\u0438\u0432\u0435\u0442 \u043C\u0438\u0440!"},
		{name: "accented", value: "caf\u00E9 na\u00EFve \u00FCber stra\u00DFe"},
		{name: "arabic", value: "\u0645\u0631\u062D\u0628\u0627"},
		{name: "mixed-scripts", value: "Hello \u4E16\u754C \U0001F600 caf\u00E9 \u041F\u0440\u0438\u0432\u0435\u0442"},
	}

	ctx := context.Background()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			key := uniqueKey(t, tc.name)
			require.NoError(t, c.Set(ctx, key, tc.value))

			got, ok, err := c.GetIfPresent(ctx, key)
			require.NoError(t, err)
			require.True(t, ok, "key %q should be present", key)
			assert.Equal(t, tc.value, got, "unicode value mismatch for %s", tc.name)
		})
	}
}

func TestEdge_NullBytesInValues(t *testing.T) {
	t.Parallel()

	c := createRedisStringCache(t, uniqueKey(t, "edge-null")+":")

	ctx := context.Background()
	value := "before\x00middle\x00after"

	key := uniqueKey(t, "null-bytes")
	require.NoError(t, c.Set(ctx, key, value))

	got, ok, err := c.GetIfPresent(ctx, key)
	require.NoError(t, err)
	require.True(t, ok, "key should be present")
	assert.Equal(t, value, got, "value with null bytes should round-trip")
}

func TestEdge_TTL_Expiry(t *testing.T) {
	t.Parallel()

	c := createRedisStringCache(t, uniqueKey(t, "edge-ttl")+":")
	ctx := context.Background()

	key := uniqueKey(t, "ttl")
	err := c.SetWithTTL(ctx, key, "temporary", 1*time.Second)
	require.NoError(t, err, "setting value with TTL")

	got, ok, err := c.GetIfPresent(ctx, key)
	require.NoError(t, err)
	require.True(t, ok, "key should be present immediately after set")
	assert.Equal(t, "temporary", got)

	time.Sleep(2 * time.Second)

	_, ok, err = c.GetIfPresent(ctx, key)
	require.NoError(t, err)
	assert.False(t, ok, "key should have expired after TTL")
}

func TestEdge_Invalidate(t *testing.T) {
	t.Parallel()

	c := createRedisStringCache(t, uniqueKey(t, "edge-invalidate")+":")

	ctx := context.Background()
	key := uniqueKey(t, "inv")
	require.NoError(t, c.Set(ctx, key, "to-be-invalidated"))

	got, ok, err := c.GetIfPresent(ctx, key)
	require.NoError(t, err)
	require.True(t, ok, "key should be present before invalidation")
	assert.Equal(t, "to-be-invalidated", got)

	require.NoError(t, c.Invalidate(ctx, key))

	_, ok, err = c.GetIfPresent(ctx, key)
	require.NoError(t, err)
	assert.False(t, ok, "key should be absent after invalidation")
}

func TestEdge_BulkSet(t *testing.T) {
	t.Parallel()

	c := createRedisStringCache(t, uniqueKey(t, "edge-bulk")+":")
	ctx := context.Background()

	items := map[string]string{
		uniqueKey(t, "bulk-1"): "value-1",
		uniqueKey(t, "bulk-2"): "value-2",
		uniqueKey(t, "bulk-3"): "value-3",
	}

	err := c.BulkSet(ctx, items)
	require.NoError(t, err, "bulk setting values")

	for key, expected := range items {
		got, ok, err := c.GetIfPresent(ctx, key)
		require.NoError(t, err)
		require.True(t, ok, "key %q should be present", key)
		assert.Equal(t, expected, got, "value mismatch for key %q", key)
	}
}

func TestEdge_Overwrite(t *testing.T) {
	t.Parallel()

	c := createRedisStringCache(t, uniqueKey(t, "edge-overwrite")+":")

	ctx := context.Background()
	key := uniqueKey(t, "overwrite")
	require.NoError(t, c.Set(ctx, key, "original"))

	got, ok, err := c.GetIfPresent(ctx, key)
	require.NoError(t, err)
	require.True(t, ok, "key should be present")
	assert.Equal(t, "original", got)

	require.NoError(t, c.Set(ctx, key, "updated"))

	got, ok, err = c.GetIfPresent(ctx, key)
	require.NoError(t, err)
	require.True(t, ok, "key should still be present")
	assert.Equal(t, "updated", got, "value should be updated after overwrite")
}

func TestEdge_Compute(t *testing.T) {
	t.Parallel()

	c := createRedisStringCache(t, uniqueKey(t, "edge-compute")+":")

	ctx := context.Background()
	key := uniqueKey(t, "compute")

	got, computed, err := c.ComputeIfAbsent(ctx, key, func() string {
		return "computed-value"
	})
	require.NoError(t, err)
	assert.Equal(t, "computed-value", got)
	assert.True(t, computed, "computation should have occurred for absent key")

	got, computed, err = c.ComputeIfAbsent(ctx, key, func() string {
		return "should-not-be-used"
	})
	require.NoError(t, err)
	assert.Equal(t, "computed-value", got)
	assert.False(t, computed, "computation should not occur for present key")
}

func TestEdge_TransformerPipeline_EmptyData(t *testing.T) {
	t.Parallel()

	setups := buildTransformerSetups(t)
	ctx := testContext(t)

	for _, setup := range setups {
		t.Run(setup.name, func(t *testing.T) {
			t.Parallel()

			input := []byte{}

			current := input
			for _, tr := range setup.transformers {
				var err error
				current, err = tr.Transform(ctx, current, nil)
				require.NoError(t, err, "transforming empty data with %s", tr.Name())
			}

			for i := len(setup.transformers) - 1; i >= 0; i-- {
				var err error
				current, err = setup.transformers[i].Reverse(ctx, current, nil)
				require.NoError(t, err, "reversing empty data with %s", setup.transformers[i].Name())
			}

			assert.Len(t, current, 0, "empty data round-trip should produce empty output for %s", setup.name)
		})
	}
}

func TestEdge_TransformerPipeline_SingleByte(t *testing.T) {
	t.Parallel()

	setups := buildTransformerSetups(t)

	for _, setup := range setups {
		t.Run(setup.name, func(t *testing.T) {
			t.Parallel()

			original := []byte("X")

			wrapped := transformAndWrap(t, original, setup.transformers)
			recovered := reverseAndUnwrap(t, wrapped, setup.transformers)

			assert.Equal(t, original, recovered, "single byte round-trip mismatch for %s", setup.name)
		})
	}
}

func TestEdge_TransformerPipeline_LargeData(t *testing.T) {
	t.Parallel()

	setups := buildTransformerSetups(t)

	for _, setup := range setups {
		t.Run(setup.name, func(t *testing.T) {
			t.Parallel()

			original := []byte(generateRepeatableText(2 * 1024 * 1024))

			wrapped := transformAndWrap(t, original, setup.transformers)
			recovered := reverseAndUnwrap(t, wrapped, setup.transformers)

			assert.Equal(t, original, recovered, "large data round-trip mismatch for %s", setup.name)
		})
	}
}
