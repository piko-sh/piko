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
	"testing"

	"github.com/klauspost/compress/gzip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/storage/storage_domain"
)

func TestMetadataEdge_MalformedJSON(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)

	original := generateRepeatableText(256)
	key := uniqueKey(t, "meta-edge-malformed-json")

	metadata := map[string]string{"x-piko-transformers": "not-json"}
	putRawBytesToS3(ctx, t, testBucketPrimary, key, original, metadata)

	gzipT := newGzipTransformer(t, gzip.DefaultCompression)
	registry := storage_domain.NewTransformerRegistry()
	require.NoError(t, registry.Register(gzipT))
	wrapper := storage_domain.NewTransformerWrapper(provider, registry, nil, "s3-test")

	retrieved := getObject(ctx, t, wrapper, key)
	assert.Equal(t, original, retrieved,
		"malformed JSON metadata should be ignored; raw data returned as-is")
}

func TestMetadataEdge_UnknownTransformerInMetadata(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)

	original := generateRepeatableText(1024)
	uploadKey := uniqueKey(t, "meta-edge-unknown-upload")

	gzipT := newGzipTransformer(t, gzip.DefaultCompression)
	uploadWrapper := newTransformerWrapper(t, provider,
		singleTransformer(gzipT),
		[]string{"gzip"})
	putObject(ctx, t, uploadWrapper, uploadKey, original)

	raw := getRawBytesFromS3(ctx, t, testBucketPrimary, uploadKey)
	corruptedKey := uniqueKey(t, "meta-edge-unknown-reupload")
	metadata := map[string]string{"x-piko-transformers": `["gzip","nonexistent"]`}
	putRawBytesToS3(ctx, t, testBucketPrimary, corruptedKey, raw, metadata)

	registry := storage_domain.NewTransformerRegistry()
	require.NoError(t, registry.Register(gzipT))
	wrapper := storage_domain.NewTransformerWrapper(provider, registry, nil, "s3-test")

	err := getObjectExpectError(ctx, t, wrapper, corruptedKey)
	assert.Error(t, err, "unknown transformer name in metadata should cause chain creation failure")
	assert.Contains(t, err.Error(), "nonexistent",
		"error message should reference the unknown transformer name")
}

func TestMetadataEdge_EmptyArray(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)

	original := generateRepeatableText(256)
	key := uniqueKey(t, "meta-edge-empty-array")

	metadata := map[string]string{"x-piko-transformers": "[]"}
	putRawBytesToS3(ctx, t, testBucketPrimary, key, original, metadata)

	gzipT := newGzipTransformer(t, gzip.DefaultCompression)
	registry := storage_domain.NewTransformerRegistry()
	require.NoError(t, registry.Register(gzipT))
	wrapper := storage_domain.NewTransformerWrapper(provider, registry, nil, "s3-test")

	retrieved := getObject(ctx, t, wrapper, key)
	assert.Equal(t, original, retrieved,
		"empty transformer array metadata should result in raw passthrough")
}

func TestMetadataEdge_SwappedOrderStillWorks(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	crypto := newCryptoSetup(t)
	gzipT := newGzipTransformer(t, gzip.DefaultCompression)

	original := generateRepeatableText(10 * 1024)
	uploadKey := uniqueKey(t, "meta-edge-ordering-upload")

	uploadWrapper := newTransformerWrapper(t, provider,
		[]storage_domain.StreamTransformerPort{gzipT, crypto.transformer},
		[]string{"gzip", "crypto-service"})
	putObject(ctx, t, uploadWrapper, uploadKey, original)

	raw := getRawBytesFromS3(ctx, t, testBucketPrimary, uploadKey)
	swappedKey := uniqueKey(t, "meta-edge-ordering-swapped")
	metadata := map[string]string{"x-piko-transformers": `["crypto-service","gzip"]`}
	putRawBytesToS3(ctx, t, testBucketPrimary, swappedKey, raw, metadata)

	registry := storage_domain.NewTransformerRegistry()
	require.NoError(t, registry.Register(gzipT))
	require.NoError(t, registry.Register(crypto.transformer))
	wrapper := storage_domain.NewTransformerWrapper(provider, registry, nil, "s3-test")

	retrieved := getObject(ctx, t, wrapper, swappedKey)
	assert.Equal(t, original, retrieved,
		"swapped metadata ordering should still work because chain sorts by priority")
}

func TestMetadataEdge_PlainDataWithConfiguredWrapper(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	crypto := newCryptoSetup(t)
	gzipT := newGzipTransformer(t, gzip.DefaultCompression)

	original := generateRepeatableText(256)
	key := uniqueKey(t, "meta-edge-plain-with-config")

	putRawBytesToS3(ctx, t, testBucketPrimary, key, original, nil)

	wrapper := newTransformerWrapper(t, provider,
		[]storage_domain.StreamTransformerPort{gzipT, crypto.transformer},
		[]string{"gzip", "crypto-service"})

	err := getObjectExpectError(ctx, t, wrapper, key)
	assert.Error(t, err,
		"plain data downloaded through configured wrapper should fail (cannot decrypt/decompress plaintext)")
}
