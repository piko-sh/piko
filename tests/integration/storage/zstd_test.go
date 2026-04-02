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

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/storage/storage_domain"
)

func TestZstd_RoundTrip(t *testing.T) {
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
			key := uniqueKey(t, "zstd-roundtrip")
			transformer := newZstdTransformer(t)
			wrapper := newTransformerWrapper(t, provider,
				[]storage_domain.StreamTransformerPort{transformer}, []string{"zstd"})

			putObject(ctx, t, wrapper, key, original)

			raw := getRawBytesFromS3(ctx, t, testBucketPrimary, key)
			assert.True(t, isValidZstd(raw), "raw bytes should have zstd magic number")
			assert.NotEqual(t, original, raw, "raw bytes should differ from original")

			retrieved := getObject(ctx, t, wrapper, key)
			assert.Equal(t, original, retrieved, "round-trip content should match original")
		})
	}
}

func TestZstd_RawIsActuallyZstd(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	original := generateRepeatableText(10 * 1024)
	key := uniqueKey(t, "zstd-raw-verify")

	transformer := newZstdTransformer(t)
	wrapper := newTransformerWrapper(t, provider,
		[]storage_domain.StreamTransformerPort{transformer}, []string{"zstd"})

	putObject(ctx, t, wrapper, key, original)

	raw := getRawBytesFromS3(ctx, t, testBucketPrimary, key)
	decoder, err := zstd.NewReader(bytes.NewReader(raw))
	require.NoError(t, err, "raw bytes should be valid zstd")
	defer decoder.Close()

	decompressed, err := io.ReadAll(decoder)
	require.NoError(t, err)
	assert.Equal(t, original, decompressed, "decompressed raw bytes should match original")
}

func TestZstd_BinaryData(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	original := generateRandomData(t, 64*1024)
	key := uniqueKey(t, "zstd-binary")

	transformer := newZstdTransformer(t)
	wrapper := newTransformerWrapper(t, provider,
		[]storage_domain.StreamTransformerPort{transformer}, []string{"zstd"})

	putObject(ctx, t, wrapper, key, original)

	raw := getRawBytesFromS3(ctx, t, testBucketPrimary, key)
	assert.True(t, isValidZstd(raw), "binary data should produce valid zstd")

	retrieved := getObject(ctx, t, wrapper, key)
	assert.Equal(t, original, retrieved, "binary data round-trip should be exact")
}

func TestZstd_CompressesWell(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	original := generateRepeatableText(1024 * 1024)
	key := uniqueKey(t, "zstd-ratio")

	transformer := newZstdTransformer(t)
	wrapper := newTransformerWrapper(t, provider,
		[]storage_domain.StreamTransformerPort{transformer}, []string{"zstd"})

	putObject(ctx, t, wrapper, key, original)

	raw := getRawBytesFromS3(ctx, t, testBucketPrimary, key)
	ratio := float64(len(raw)) / float64(len(original))
	assert.Less(t, ratio, 0.5,
		"repeating text should compress to less than 50%% of original (got %.1f%%)", ratio*100)
}
