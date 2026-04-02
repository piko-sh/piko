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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/cache/cache_domain"
)

func TestZstd_RoundTrip(t *testing.T) {
	t.Parallel()

	zstd := newZstdTransformer(t)
	ctx := testContext(t)

	testCases := []struct {
		name string
		size int
	}{
		{name: "256B", size: 256},
		{name: "1KB", size: 1024},
		{name: "1MB", size: 1024 * 1024},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			original := []byte(generateRepeatableText(tc.size))

			compressed, err := zstd.Transform(ctx, original, nil)
			require.NoError(t, err, "compressing %s", tc.name)

			decompressed, err := zstd.Reverse(ctx, compressed, nil)
			require.NoError(t, err, "decompressing %s", tc.name)

			assert.Equal(t, original, decompressed, "round-trip mismatch for %s", tc.name)
		})
	}
}

func TestZstd_RawIsCompressed(t *testing.T) {
	t.Parallel()

	zstd := newZstdTransformer(t)
	namespace := uniqueKey(t, "zstd-raw") + ":"

	original := []byte(generateRepeatableText(1024))

	wrapped := transformAndWrap(t, original, []cache_domain.CacheTransformerPort{zstd})

	redisKey := rawRedisKeyForNamespace(namespace, "test-key")
	putRawBytesToRedis(t, redisKey, wrapped, 1*time.Hour)

	raw := getRawBytesFromRedis(t, redisKey)
	assert.Equal(t, wrapped, raw, "raw bytes should match what we stored")

	assert.NotEqual(t, original, raw, "raw bytes should differ from original")

	recovered := reverseAndUnwrap(t, raw, []cache_domain.CacheTransformerPort{zstd})
	assert.Equal(t, original, recovered, "recovered data should match original")
}

func TestZstd_CompressesWell(t *testing.T) {
	t.Parallel()

	zstd := newZstdTransformer(t)
	ctx := testContext(t)

	original := []byte(generateRepeatableText(1024 * 1024))

	compressed, err := zstd.Transform(ctx, original, nil)
	require.NoError(t, err, "compressing 1MB")

	ratio := float64(len(compressed)) / float64(len(original)) * 100
	t.Logf("Compression ratio: %.2f%% (original=%d, compressed=%d)", ratio, len(original), len(compressed))

	assert.Less(t, ratio, 50.0, "repeating text should compress to less than 50%% of original")
}

func TestZstd_BinaryData(t *testing.T) {
	t.Parallel()

	zstd := newZstdTransformer(t)
	ctx := testContext(t)

	original := generateRandomData(t, 4096)

	compressed, err := zstd.Transform(ctx, original, nil)
	require.NoError(t, err, "compressing binary data")

	decompressed, err := zstd.Reverse(ctx, compressed, nil)
	require.NoError(t, err, "decompressing binary data")

	assert.Equal(t, original, decompressed, "binary data round-trip mismatch")
}

func TestZstd_HighlyCompressible(t *testing.T) {
	t.Parallel()

	zstd := newZstdTransformer(t)
	ctx := testContext(t)

	original := make([]byte, 1024*1024)

	compressed, err := zstd.Transform(ctx, original, nil)
	require.NoError(t, err, "compressing zeros")

	ratio := float64(len(compressed)) / float64(len(original)) * 100
	t.Logf("Zeros compression ratio: %.2f%% (original=%d, compressed=%d)", ratio, len(original), len(compressed))

	assert.Less(t, ratio, 5.0, "all-zeros should compress to less than 5%% of original")

	decompressed, err := zstd.Reverse(ctx, compressed, nil)
	require.NoError(t, err, "decompressing zeros")

	assert.Equal(t, original, decompressed, "zeros round-trip mismatch")
}

func TestZstd_Incompressible(t *testing.T) {
	t.Parallel()

	zstd := newZstdTransformer(t)
	ctx := testContext(t)

	original := generateRandomData(t, 64*1024)

	compressed, err := zstd.Transform(ctx, original, nil)
	require.NoError(t, err, "compressing random data")

	decompressed, err := zstd.Reverse(ctx, compressed, nil)
	require.NoError(t, err, "decompressing random data")

	assert.Equal(t, original, decompressed, "random data round-trip mismatch")
}

func TestZstd_RoundTripViaRedis(t *testing.T) {
	t.Parallel()

	zstd := newZstdTransformer(t)
	namespace := uniqueKey(t, "zstd-redis") + ":"

	testCases := []struct {
		name string
		size int
	}{
		{name: "256B", size: 256},
		{name: "1KB", size: 1024},
		{name: "64KB", size: 64 * 1024},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			original := []byte(generateRepeatableText(tc.size))
			transformers := []cache_domain.CacheTransformerPort{zstd}

			wrapped := transformAndWrap(t, original, transformers)

			redisKey := rawRedisKeyForNamespace(namespace, tc.name)
			putRawBytesToRedis(t, redisKey, wrapped, 1*time.Hour)

			raw := getRawBytesFromRedis(t, redisKey)
			recovered := reverseAndUnwrap(t, raw, transformers)

			assert.Equal(t, original, recovered, "Redis round-trip mismatch for %s", tc.name)
		})
	}
}
