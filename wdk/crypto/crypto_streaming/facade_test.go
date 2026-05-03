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

package crypto_streaming_test

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/crypto/crypto_streaming"
)

func newAEAD(t *testing.T) cipher.AEAD {
	t.Helper()

	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)

	block, err := aes.NewCipher(key)
	require.NoError(t, err)

	aead, err := cipher.NewGCM(block)
	require.NoError(t, err)

	return aead
}

func TestGenerateIV_ReturnsCorrectSize(t *testing.T) {
	t.Parallel()

	iv, err := crypto_streaming.GenerateIV()

	require.NoError(t, err)
	require.Len(t, iv, crypto_streaming.IVSize)
}

func TestGenerateIV_ReturnsUniqueValues(t *testing.T) {
	t.Parallel()

	first, err := crypto_streaming.GenerateIV()
	require.NoError(t, err)

	second, err := crypto_streaming.GenerateIV()
	require.NoError(t, err)

	require.NotEqual(t, first, second)
}

func TestStreamingRoundTrip_RecoversPlaintextFromCiphertext(t *testing.T) {
	t.Parallel()

	aead := newAEAD(t)

	baseIV, err := crypto_streaming.GenerateIV()
	require.NoError(t, err)

	plaintext := []byte("the quick brown fox jumps over the lazy dog. " +
		"the quick brown fox jumps over the lazy dog. " +
		"the quick brown fox jumps over the lazy dog.")

	var ciphertext bytes.Buffer
	writer := crypto_streaming.NewEncryptingWriter(&ciphertext, aead, baseIV, crypto_streaming.DefaultChunkSize)
	written, err := writer.Write(plaintext)
	require.NoError(t, err)
	require.Equal(t, len(plaintext), written)
	require.NoError(t, writer.Close())
	require.Greater(t, ciphertext.Len(), len(plaintext))

	reader := crypto_streaming.NewDecryptingReader(&ciphertext, aead, baseIV)
	got, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	require.Equal(t, plaintext, got)
}

func TestStreamingRoundTrip_HandlesSmallChunks(t *testing.T) {
	t.Parallel()

	aead := newAEAD(t)

	baseIV, err := crypto_streaming.GenerateIV()
	require.NoError(t, err)

	plaintext := []byte("hello world from a tiny chunk size")

	var ciphertext bytes.Buffer
	writer := crypto_streaming.NewEncryptingWriter(&ciphertext, aead, baseIV, 4)
	_, err = writer.Write(plaintext)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	reader := crypto_streaming.NewDecryptingReader(&ciphertext, aead, baseIV)
	got, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	require.Equal(t, plaintext, got)
}

func TestStreamingHeader_RoundTrip(t *testing.T) {
	t.Parallel()

	original := &crypto_streaming.StreamingHeader{
		KeyID:            "test-key",
		Provider:         "local_aes_gcm",
		IV:               "abcd1234",
		EncryptedDataKey: "edk-value",
		Algorithm:        "AES-256-GCM",
		Version:          2,
	}

	var buf bytes.Buffer
	require.NoError(t, crypto_streaming.WriteStreamingHeader(&buf, original))
	require.Greater(t, buf.Len(), 0)

	parsed, err := crypto_streaming.ReadStreamingHeader(&buf)
	require.NoError(t, err)
	require.Equal(t, original.KeyID, parsed.KeyID)
	require.Equal(t, original.Provider, parsed.Provider)
	require.Equal(t, original.IV, parsed.IV)
	require.Equal(t, original.EncryptedDataKey, parsed.EncryptedDataKey)
	require.Equal(t, original.Algorithm, parsed.Algorithm)
	require.Equal(t, original.Version, parsed.Version)
}

func TestReadStreamingHeader_ReturnsErrorOnTruncatedInput(t *testing.T) {
	t.Parallel()

	_, err := crypto_streaming.ReadStreamingHeader(bytes.NewReader(nil))
	require.Error(t, err)
}

func TestNewDecryptingReader_FailsOnTamperedCiphertext(t *testing.T) {
	t.Parallel()

	aead := newAEAD(t)

	baseIV, err := crypto_streaming.GenerateIV()
	require.NoError(t, err)

	plaintext := bytes.Repeat([]byte("AB"), 256)

	var ciphertext bytes.Buffer
	writer := crypto_streaming.NewEncryptingWriter(&ciphertext, aead, baseIV, 16)
	_, err = writer.Write(plaintext)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	tampered := ciphertext.Bytes()
	tampered[len(tampered)-1] ^= 0xFF

	reader := crypto_streaming.NewDecryptingReader(bytes.NewReader(tampered), aead, baseIV)
	_, err = io.ReadAll(reader)
	require.Error(t, err)
	_ = reader.Close()
}
