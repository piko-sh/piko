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
	"encoding/binary"
	"testing"

	"github.com/klauspost/compress/gzip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/storage/storage_domain"
)

func envelopeBodyOffset(raw []byte) int {
	if len(raw) < 5 {
		return len(raw)
	}
	headerLen := int(binary.BigEndian.Uint32(raw[1:5]))
	return 5 + headerLen
}

func TestCorruption_CorruptedCiphertext(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	crypto := newCryptoSetup(t)

	original := generateRepeatableText(10 * 1024)
	key := uniqueKey(t, "corrupt-ciphertext")
	wrapper := newTransformerWrapper(t, provider,
		singleTransformer(crypto.transformer),
		[]string{"crypto-service"})

	putObject(ctx, t, wrapper, key, original)

	raw := getRawBytesFromS3(ctx, t, testBucketPrimary, key)
	metadata := getObjectMetadataFromS3(ctx, t, testBucketPrimary, key)

	corrupted := make([]byte, len(raw))
	copy(corrupted, raw)
	bodyStart := envelopeBodyOffset(corrupted)

	corruptOffset := bodyStart + 4 + 20
	for i := range 10 {
		if corruptOffset+i < len(corrupted) {
			corrupted[corruptOffset+i] ^= 0xFF
		}
	}

	corruptedKey := uniqueKey(t, "corrupt-ciphertext-tampered")
	putRawBytesToS3(ctx, t, testBucketPrimary, corruptedKey, corrupted, metadata)

	err := getObjectExpectError(ctx, t, wrapper, corruptedKey)
	assert.Error(t, err, "corrupted ciphertext should cause decryption failure")
}

func TestCorruption_TruncatedCiphertext(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	crypto := newCryptoSetup(t)

	plaintext := generateRepeatableText(10 * 1024)
	encrypted := encryptBytesDirectly(ctx, t, crypto, plaintext)

	bodyStart := envelopeBodyOffset(encrypted)
	truncateAt := bodyStart + 50
	if truncateAt > len(encrypted) {
		truncateAt = len(encrypted) - 10
	}
	truncated := encrypted[:truncateAt]

	key := uniqueKey(t, "corrupt-truncated")
	metadata := map[string]string{"x-piko-transformers": `["crypto-service"]`}
	putRawBytesToS3(ctx, t, testBucketPrimary, key, truncated, metadata)

	wrapper := newTransformerWrapper(t, provider,
		singleTransformer(crypto.transformer),
		[]string{"crypto-service"})

	err := getObjectExpectError(ctx, t, wrapper, key)
	assert.Error(t, err, "truncated ciphertext should cause decryption failure")
}

func TestCorruption_InvalidEnvelopeVersion(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	crypto := newCryptoSetup(t)

	plaintext := generateRepeatableText(1024)
	encrypted := encryptBytesDirectly(ctx, t, crypto, plaintext)
	require.True(t, len(encrypted) > 1)

	modified := make([]byte, len(encrypted))
	copy(modified, encrypted)
	modified[0] = 0xFF

	key := uniqueKey(t, "corrupt-bad-version")
	metadata := map[string]string{"x-piko-transformers": `["crypto-service"]`}
	putRawBytesToS3(ctx, t, testBucketPrimary, key, modified, metadata)

	wrapper := newTransformerWrapper(t, provider,
		singleTransformer(crypto.transformer),
		[]string{"crypto-service"})

	err := getObjectExpectError(ctx, t, wrapper, key)
	assert.Error(t, err, "invalid envelope version should cause decryption failure")
}

func TestCorruption_BitFlipInCiphertext(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	crypto := newCryptoSetup(t)

	original := generateRepeatableText(10 * 1024)
	key := uniqueKey(t, "corrupt-bitflip")
	wrapper := newTransformerWrapper(t, provider,
		singleTransformer(crypto.transformer),
		[]string{"crypto-service"})

	putObject(ctx, t, wrapper, key, original)

	raw := getRawBytesFromS3(ctx, t, testBucketPrimary, key)
	metadata := getObjectMetadataFromS3(ctx, t, testBucketPrimary, key)

	flipped := make([]byte, len(raw))
	copy(flipped, raw)
	bodyStart := envelopeBodyOffset(flipped)
	flipOffset := bodyStart + 4 + 32
	if flipOffset < len(flipped) {
		flipped[flipOffset] ^= 0x01
	}

	flippedKey := uniqueKey(t, "corrupt-bitflip-tampered")
	putRawBytesToS3(ctx, t, testBucketPrimary, flippedKey, flipped, metadata)

	err := getObjectExpectError(ctx, t, wrapper, flippedKey)
	assert.Error(t, err, "single bit flip should cause AES-GCM authentication failure")
}

func TestCorruption_WrongKeyDecryption(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)

	cryptoA := newCryptoSetupWithKey(t, "key-A-material-for-testing!!!!!", "key-a")
	cryptoB := newCryptoSetupWithKey(t, "key-B-material-for-testing!!!!!", "key-b")

	original := generateRepeatableText(10 * 1024)
	key := uniqueKey(t, "corrupt-wrong-key")

	wrapperA := newTransformerWrapper(t, provider,
		singleTransformer(cryptoA.transformer),
		[]string{"crypto-service"})

	putObject(ctx, t, wrapperA, key, original)

	wrapperB := newTransformerWrapper(t, provider,
		singleTransformer(cryptoB.transformer),
		[]string{"crypto-service"})

	err := getObjectExpectError(ctx, t, wrapperB, key)
	assert.Error(t, err, "decryption with wrong key should fail")
}

func TestCorruption_TruncatedGzipData(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)

	truncatedGzip := []byte{0x1f, 0x8b, 0x08, 0x00}

	key := uniqueKey(t, "corrupt-truncated-gzip")
	metadata := map[string]string{"x-piko-transformers": `["gzip"]`}
	putRawBytesToS3(ctx, t, testBucketPrimary, key, truncatedGzip, metadata)

	gzipT := newGzipTransformer(t, gzip.DefaultCompression)
	wrapper := newTransformerWrapper(t, provider,
		singleTransformer(gzipT),
		[]string{"gzip"})

	err := getObjectExpectError(ctx, t, wrapper, key)
	assert.Error(t, err, "truncated gzip data should cause decompression failure")
}

func TestCorruption_TruncatedZstdData(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)

	truncatedZstd := []byte{0x28, 0xB5, 0x2F, 0xFD}

	key := uniqueKey(t, "corrupt-truncated-zstd")
	metadata := map[string]string{"x-piko-transformers": `["zstd"]`}
	putRawBytesToS3(ctx, t, testBucketPrimary, key, truncatedZstd, metadata)

	zstdT := newZstdTransformer(t)
	wrapper := newTransformerWrapper(t, provider,
		singleTransformer(zstdT),
		[]string{"zstd"})

	err := getObjectExpectError(ctx, t, wrapper, key)
	assert.Error(t, err, "truncated zstd data should cause decompression failure")
}

func TestCorruption_CorruptedGzipBody(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)

	original := generateRepeatableText(10 * 1024)
	key := uniqueKey(t, "corrupt-gzip-body")

	gzipT := newGzipTransformer(t, gzip.DefaultCompression)
	wrapper := newTransformerWrapper(t, provider,
		singleTransformer(gzipT),
		[]string{"gzip"})

	putObject(ctx, t, wrapper, key, original)

	raw := getRawBytesFromS3(ctx, t, testBucketPrimary, key)
	metadata := getObjectMetadataFromS3(ctx, t, testBucketPrimary, key)
	require.True(t, isValidGzip(raw), "raw data should be valid gzip before corruption")

	corrupted := make([]byte, len(raw))
	copy(corrupted, raw)
	for i := 10; i < len(corrupted) && i < 50; i++ {
		corrupted[i] ^= 0xFF
	}

	corruptedKey := uniqueKey(t, "corrupt-gzip-body-tampered")
	putRawBytesToS3(ctx, t, testBucketPrimary, corruptedKey, corrupted, metadata)

	err := getObjectExpectError(ctx, t, wrapper, corruptedKey)
	assert.Error(t, err, "corrupted gzip body should cause decompression failure")
}

func TestCorruption_CorruptedChainedData(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	crypto := newCryptoSetup(t)
	gzipT := newGzipTransformer(t, gzip.DefaultCompression)

	original := generateRepeatableText(10 * 1024)
	key := uniqueKey(t, "corrupt-chained")
	wrapper := newTransformerWrapper(t, provider,
		[]storage_domain.StreamTransformerPort{gzipT, crypto.transformer},
		[]string{"gzip", "crypto-service"})

	putObject(ctx, t, wrapper, key, original)

	raw := getRawBytesFromS3(ctx, t, testBucketPrimary, key)
	metadata := getObjectMetadataFromS3(ctx, t, testBucketPrimary, key)
	require.True(t, isStreamingEnvelope(raw), "outermost layer should be crypto envelope")

	corrupted := make([]byte, len(raw))
	copy(corrupted, raw)
	bodyStart := envelopeBodyOffset(corrupted)
	corruptOffset := bodyStart + 4 + 20
	for i := range 10 {
		if corruptOffset+i < len(corrupted) {
			corrupted[corruptOffset+i] ^= 0xFF
		}
	}

	corruptedKey := uniqueKey(t, "corrupt-chained-tampered")
	putRawBytesToS3(ctx, t, testBucketPrimary, corruptedKey, corrupted, metadata)

	err := getObjectExpectError(ctx, t, wrapper, corruptedKey)
	assert.Error(t, err, "corrupted chained data should fail at crypto layer")
}

func TestCorruption_EmptyInputThroughCrypto(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	crypto := newCryptoSetup(t)

	original := []byte{}
	key := uniqueKey(t, "corrupt-empty-crypto")
	wrapper := newTransformerWrapper(t, provider,
		singleTransformer(crypto.transformer),
		[]string{"crypto-service"})

	putObject(ctx, t, wrapper, key, original)

	retrieved := getObject(ctx, t, wrapper, key)
	assert.Empty(t, retrieved, "empty file through crypto should round-trip correctly")
}

func TestCorruption_ExactAESBlockSize(t *testing.T) {
	ctx := t.Context()
	provider := newS3Provider(ctx, t, globalEnv)
	crypto := newCryptoSetup(t)

	original := []byte("0123456789ABCDEF")
	key := uniqueKey(t, "corrupt-aes-block-size")
	wrapper := newTransformerWrapper(t, provider,
		singleTransformer(crypto.transformer),
		[]string{"crypto-service"})

	putObject(ctx, t, wrapper, key, original)

	retrieved := getObject(ctx, t, wrapper, key)
	assert.Equal(t, original, retrieved,
		"data exactly matching AES block size should round-trip correctly")
}
