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

package storage_integration_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/klauspost/compress/gzip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/storage/storage_domain"
)

func TestGzip_RoundTrip(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)

	testCases := []struct {
		name string
		size int
	}{
		{name: "small/256B", size: 256},
		{name: "medium/1MB", size: 1024 * 1024},
		{name: "large/10MB", size: 10 * 1024 * 1024},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original := generateRepeatableText(tc.size)
			key := uniqueKey(t, "gzip-roundtrip")
			transformer := newGzipTransformer(t, gzip.DefaultCompression)
			wrapper := newTransformerWrapper(t, provider, singleTransformer(transformer), []string{"gzip"})

			putObject(ctx, t, wrapper, key, original)

			raw := getRawBytesFromS3(ctx, t, testBucketPrimary, key)
			assert.True(t, isValidGzip(raw), "raw bytes should have gzip magic header")
			assert.NotEqual(t, original, raw, "raw bytes should differ from original")

			retrieved := getObject(ctx, t, wrapper, key)
			assert.Equal(t, original, retrieved, "round-trip content should match original")
		})
	}
}

func TestGzip_RawIsActuallyGzip(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	original := generateRepeatableText(10 * 1024)
	key := uniqueKey(t, "gzip-raw-verify")

	transformer := newGzipTransformer(t, gzip.DefaultCompression)
	wrapper := newTransformerWrapper(t, provider, singleTransformer(transformer), []string{"gzip"})

	putObject(ctx, t, wrapper, key, original)

	raw := getRawBytesFromS3(ctx, t, testBucketPrimary, key)
	reader, err := gzip.NewReader(bytes.NewReader(raw))
	require.NoError(t, err, "raw bytes should be valid gzip")
	defer func() { _ = reader.Close() }()

	decompressed, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, original, decompressed, "decompressed raw bytes should match original")
}

func TestGzip_CompressionLevels(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	original := generateRepeatableText(100 * 1024)

	testCases := []struct {
		name  string
		level int
	}{
		{name: "NoCompression", level: gzip.NoCompression},
		{name: "BestSpeed", level: gzip.BestSpeed},
		{name: "DefaultCompression", level: gzip.DefaultCompression},
		{name: "BestCompression", level: gzip.BestCompression},
	}

	sizes := make(map[string]int)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			key := uniqueKey(t, "gzip-levels")
			transformer := newGzipTransformer(t, tc.level)
			wrapper := newTransformerWrapper(t, provider, singleTransformer(transformer), []string{"gzip"})

			putObject(ctx, t, wrapper, key, original)

			raw := getRawBytesFromS3(ctx, t, testBucketPrimary, key)
			assert.True(t, isValidGzip(raw), "raw bytes should have gzip magic header")
			sizes[tc.name] = len(raw)

			retrieved := getObject(ctx, t, wrapper, key)
			assert.Equal(t, original, retrieved)
		})
	}

	if bestComp, ok := sizes["BestCompression"]; ok {
		if bestSpeed, ok := sizes["BestSpeed"]; ok {
			assert.LessOrEqual(t, bestComp, bestSpeed,
				"BestCompression (%d) should not exceed BestSpeed (%d)", bestComp, bestSpeed)
		}
	}
}

func TestGzip_BinaryData(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	original := generateRandomData(t, 64*1024)
	key := uniqueKey(t, "gzip-binary")

	transformer := newGzipTransformer(t, gzip.DefaultCompression)
	wrapper := newTransformerWrapper(t, provider, singleTransformer(transformer), []string{"gzip"})

	putObject(ctx, t, wrapper, key, original)

	raw := getRawBytesFromS3(ctx, t, testBucketPrimary, key)
	assert.True(t, isValidGzip(raw), "binary data should produce valid gzip")

	retrieved := getObject(ctx, t, wrapper, key)
	assert.Equal(t, original, retrieved, "binary data round-trip should be exact")
}

func TestGzip_CompressesWell(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	original := generateRepeatableText(1024 * 1024)
	key := uniqueKey(t, "gzip-ratio")

	transformer := newGzipTransformer(t, gzip.DefaultCompression)
	wrapper := newTransformerWrapper(t, provider, singleTransformer(transformer), []string{"gzip"})

	putObject(ctx, t, wrapper, key, original)

	raw := getRawBytesFromS3(ctx, t, testBucketPrimary, key)
	ratio := float64(len(raw)) / float64(len(original))
	assert.Less(t, ratio, 0.5,
		"repeating text should compress to less than 50%% of original (got %.1f%%)", ratio*100)
}

func singleTransformer(t storage_domain.StreamTransformerPort) []storage_domain.StreamTransformerPort {
	return []storage_domain.StreamTransformerPort{t}
}
