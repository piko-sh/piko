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

package local_aes_gcm

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/crypto/crypto_dto"
)

func TestStreamingRoundtrip(t *testing.T) {
	ctx := context.Background()

	key := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, key)
	require.NoError(t, err)

	provider, err := NewProvider(Config{Key: key, KeyID: "test-key"})
	require.NoError(t, err)

	testCases := []struct {
		name     string
		dataSize int
	}{
		{name: "empty", dataSize: 0},
		{name: "small (10 bytes)", dataSize: 10},
		{name: "one chunk (64KB)", dataSize: 64 * 1024},
		{name: "two chunks (128KB)", dataSize: 128 * 1024},
		{name: "large (1MB)", dataSize: 1024 * 1024},
		{name: "slightly over chunk boundary", dataSize: 64*1024 + 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			plaintext := make([]byte, tc.dataSize)
			if tc.dataSize > 0 {
				_, err := io.ReadFull(rand.Reader, plaintext)
				require.NoError(t, err)
			}

			var ciphertextBuf bytes.Buffer
			encryptingWriter, err := provider.EncryptStream(ctx, &ciphertextBuf, &crypto_dto.EncryptRequest{})
			require.NoError(t, err)

			n, err := encryptingWriter.Write(plaintext)
			require.NoError(t, err)
			assert.Equal(t, len(plaintext), n)

			err = encryptingWriter.Close()
			require.NoError(t, err)

			if tc.dataSize > 0 {
				assert.NotEqual(t, plaintext, ciphertextBuf.Bytes(), "ciphertext should differ from plaintext")
			}

			decryptingReader, err := provider.DecryptStream(ctx, &ciphertextBuf)
			require.NoError(t, err)
			defer func() { _ = decryptingReader.Close() }()

			decrypted, err := io.ReadAll(decryptingReader)
			require.NoError(t, err)

			assert.Equal(t, plaintext, decrypted, "decrypted plaintext should match original")
		})
	}
}

func TestStreamingMultipleWrites(t *testing.T) {
	ctx := context.Background()

	key := make([]byte, 32)
	provider, _ := NewProvider(Config{Key: key, KeyID: "test-key"})

	chunk1 := []byte("Hello, ")
	chunk2 := []byte("streaming ")
	chunk3 := []byte("world!")
	expectedPlaintext := append(append(chunk1, chunk2...), chunk3...)

	var ciphertextBuf bytes.Buffer
	encryptingWriter, err := provider.EncryptStream(ctx, &ciphertextBuf, &crypto_dto.EncryptRequest{})
	require.NoError(t, err)

	_, err = encryptingWriter.Write(chunk1)
	require.NoError(t, err)
	_, err = encryptingWriter.Write(chunk2)
	require.NoError(t, err)
	_, err = encryptingWriter.Write(chunk3)
	require.NoError(t, err)

	err = encryptingWriter.Close()
	require.NoError(t, err)

	decryptingReader, err := provider.DecryptStream(ctx, &ciphertextBuf)
	require.NoError(t, err)
	defer func() { _ = decryptingReader.Close() }()

	decrypted, err := io.ReadAll(decryptingReader)
	require.NoError(t, err)

	assert.Equal(t, expectedPlaintext, decrypted)
}

func TestStreamingMultipleReads(t *testing.T) {
	ctx := context.Background()

	key := make([]byte, 32)
	provider, _ := NewProvider(Config{Key: key, KeyID: "test-key"})

	plaintext := []byte("This is a test message that will be read in small chunks")

	var ciphertextBuf bytes.Buffer
	encryptingWriter, _ := provider.EncryptStream(ctx, &ciphertextBuf, &crypto_dto.EncryptRequest{})
	_, _ = encryptingWriter.Write(plaintext)
	_ = encryptingWriter.Close()

	decryptingReader, _ := provider.DecryptStream(ctx, &ciphertextBuf)
	defer func() { _ = decryptingReader.Close() }()

	var decrypted []byte
	smallBuf := make([]byte, 5)

	for {
		n, err := decryptingReader.Read(smallBuf)
		if n > 0 {
			decrypted = append(decrypted, smallBuf[:n]...)
		}
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}

	assert.Equal(t, plaintext, decrypted)
}

func TestDeriveIV(t *testing.T) {
	baseIV := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}

	testCases := []struct {
		name     string
		expected []byte
		chunkNum uint32
	}{
		{name: "chunk 0", chunkNum: 0, expected: []byte{1, 2, 3, 4, 5, 6, 7, 8, 0, 0, 0, 0}},
		{name: "chunk 1", chunkNum: 1, expected: []byte{1, 2, 3, 4, 5, 6, 7, 8, 0, 0, 0, 1}},
		{name: "chunk 255", chunkNum: 255, expected: []byte{1, 2, 3, 4, 5, 6, 7, 8, 0, 0, 0, 255}},
		{name: "chunk 256", chunkNum: 256, expected: []byte{1, 2, 3, 4, 5, 6, 7, 8, 0, 0, 1, 0}},
		{name: "chunk 65536", chunkNum: 65536, expected: []byte{1, 2, 3, 4, 5, 6, 7, 8, 0, 1, 0, 0}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			iv := deriveIV(baseIV, tc.chunkNum)
			assert.Equal(t, tc.expected, iv)
			assert.Len(t, iv, IVSize)
		})
	}
}

func TestDeriveIVUniqueness(t *testing.T) {
	baseIV := make([]byte, IVSize)
	_, _ = io.ReadFull(rand.Reader, baseIV)

	seen := make(map[string]bool)

	for i := range uint32(10000) {
		iv := deriveIV(baseIV, i)
		ivString := string(iv)

		assert.False(t, seen[ivString], "IV for chunk %d collides with previous IV", i)
		seen[ivString] = true
	}
}

func TestStreamingHeaderRoundtrip(t *testing.T) {
	header := &crypto_dto.StreamingHeader{
		Version:   2,
		KeyID:     "test-key-123",
		Provider:  "local_aes_gcm",
		IV:        "dGVzdC1pdi0xMjM0NTY=",
		Algorithm: "AES-256-GCM",
	}

	var buffer bytes.Buffer
	err := WriteStreamingHeader(&buffer, header)
	require.NoError(t, err)

	data := buffer.Bytes()
	assert.Equal(t, crypto_dto.StreamingEnvelopeVersion, data[0])

	parsedHeader, err := ReadStreamingHeader(&buffer)
	require.NoError(t, err)

	assert.Equal(t, header.Version, parsedHeader.Version)
	assert.Equal(t, header.KeyID, parsedHeader.KeyID)
	assert.Equal(t, header.Provider, parsedHeader.Provider)
	assert.Equal(t, header.IV, parsedHeader.IV)
	assert.Equal(t, header.Algorithm, parsedHeader.Algorithm)
}

func TestStreamingHeaderInvalidVersion(t *testing.T) {
	var buffer bytes.Buffer
	buffer.WriteByte(0x99)
	buffer.Write([]byte{0, 0, 0, 10})
	buffer.WriteString(`{"v":99}`)

	_, err := ReadStreamingHeader(&buffer)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported ciphertext version")
}

func TestGenerateIV(t *testing.T) {
	ivs := make(map[string]bool)

	for range 1000 {
		iv, err := GenerateIV()
		require.NoError(t, err)
		assert.Len(t, iv, IVSize)

		ivString := string(iv)
		assert.False(t, ivs[ivString], "generated duplicate IV")
		ivs[ivString] = true
	}
}

func TestEncryptingWriterCloseFlushes(t *testing.T) {
	ctx := context.Background()

	key := make([]byte, 32)
	provider, _ := NewProvider(Config{Key: key})

	plaintext := []byte("small data")

	var ciphertextBuf bytes.Buffer
	encryptingWriter, _ := provider.EncryptStream(ctx, &ciphertextBuf, &crypto_dto.EncryptRequest{})

	_, _ = encryptingWriter.Write(plaintext)

	ciphertextBeforeClose := len(ciphertextBuf.Bytes())

	_ = encryptingWriter.Close()

	ciphertextAfterClose := len(ciphertextBuf.Bytes())
	assert.Greater(t, ciphertextAfterClose, ciphertextBeforeClose, "Close() should flush remaining data")

	decryptingReader, _ := provider.DecryptStream(ctx, &ciphertextBuf)
	defer func() { _ = decryptingReader.Close() }()

	decrypted, err := io.ReadAll(decryptingReader)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryptingWriterErrorAfterClose(t *testing.T) {
	ctx := context.Background()

	key := make([]byte, 32)
	provider, _ := NewProvider(Config{Key: key})

	var ciphertextBuf bytes.Buffer
	encryptingWriter, _ := provider.EncryptStream(ctx, &ciphertextBuf, &crypto_dto.EncryptRequest{})

	_ = encryptingWriter.Close()

	_, err := encryptingWriter.Write([]byte("data after close"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "closed")
}

func TestEncryptingWriterCloseIsIdempotent(t *testing.T) {
	ctx := context.Background()

	key := make([]byte, 32)
	provider, _ := NewProvider(Config{Key: key})

	var ciphertextBuf bytes.Buffer
	encryptingWriter, _ := provider.EncryptStream(ctx, &ciphertextBuf, &crypto_dto.EncryptRequest{})

	_, _ = encryptingWriter.Write([]byte("data"))
	err := encryptingWriter.Close()
	require.NoError(t, err)

	err = encryptingWriter.Close()
	assert.NoError(t, err)
}

func TestDecryptingReaderErrorAfterClose(t *testing.T) {
	ctx := context.Background()

	key := make([]byte, 32)
	provider, _ := NewProvider(Config{Key: key})

	plaintext := []byte("test data")

	var ciphertextBuf bytes.Buffer
	encryptingWriter, _ := provider.EncryptStream(ctx, &ciphertextBuf, &crypto_dto.EncryptRequest{})
	_, _ = encryptingWriter.Write(plaintext)
	_ = encryptingWriter.Close()

	decryptingReader, _ := provider.DecryptStream(ctx, &ciphertextBuf)
	_ = decryptingReader.Close()

	buffer := make([]byte, 10)
	_, err := decryptingReader.Read(buffer)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "closed")
}

type errorWriter struct {
	err               error
	writesBeforeError int
	writeCount        int
}

func (w *errorWriter) Write(p []byte) (int, error) {
	if w.writeCount >= w.writesBeforeError {
		return 0, w.err
	}
	w.writeCount++
	return len(p), nil
}

func TestNewEncryptingWriter_Success(t *testing.T) {
	key := generateTestKey()

	iv := make([]byte, IVSize)
	_, err := io.ReadFull(rand.Reader, iv)
	require.NoError(t, err)

	block, err := aes.NewCipher(key)
	require.NoError(t, err)
	aead, err := cipher.NewGCM(block)
	require.NoError(t, err)

	var buffer bytes.Buffer
	writer := NewEncryptingWriter(&buffer, aead, iv, 1024)
	require.NotNil(t, writer)

	_, err = writer.Write([]byte("hello"))
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)
	assert.Greater(t, buffer.Len(), 0)
}

func TestNewEncryptingWriter_PanicsOnInvalidIV(t *testing.T) {
	key := generateTestKey()
	block, _ := aes.NewCipher(key)
	aead, _ := cipher.NewGCM(block)

	assert.Panics(t, func() {
		NewEncryptingWriter(&bytes.Buffer{}, aead, []byte{1, 2, 3}, 1024)
	})
}

func TestNewDecryptingReader_Success(t *testing.T) {
	key := generateTestKey()
	block, _ := aes.NewCipher(key)
	aead, _ := cipher.NewGCM(block)

	iv := make([]byte, IVSize)
	_, _ = io.ReadFull(rand.Reader, iv)

	var encBuf bytes.Buffer
	writer := NewEncryptingWriter(&encBuf, aead, iv, 1024)
	_, _ = writer.Write([]byte("test data"))
	_ = writer.Close()

	reader := NewDecryptingReader(&encBuf, aead, iv)
	require.NotNil(t, reader)

	decrypted, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, []byte("test data"), decrypted)

	_ = reader.Close()
}

func TestNewDecryptingReader_PanicsOnInvalidIV(t *testing.T) {
	key := generateTestKey()
	block, _ := aes.NewCipher(key)
	aead, _ := cipher.NewGCM(block)

	assert.Panics(t, func() {
		NewDecryptingReader(&bytes.Buffer{}, aead, []byte{1, 2})
	})
}

func TestDeriveIV_PanicsOnInvalidBaseIV(t *testing.T) {
	testCases := []struct {
		name string
		iv   []byte
	}{
		{name: "too short", iv: []byte{1, 2, 3}},
		{name: "too long", iv: make([]byte, 16)},
		{name: "empty", iv: []byte{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Panics(t, func() {
				deriveIV(tc.iv, 0)
			})
		})
	}
}

func TestWriteStreamingHeader_WriteErrors(t *testing.T) {
	header := &crypto_dto.StreamingHeader{
		Version:   2,
		KeyID:     "test",
		Provider:  "local_aes_gcm",
		IV:        "dGVzdA==",
		Algorithm: "AES-256-GCM",
	}

	t.Run("version byte write fails", func(t *testing.T) {
		w := &errorWriter{
			writesBeforeError: 0,
			err:               errors.New("disk full"),
		}
		err := WriteStreamingHeader(w, header)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write version byte")
	})

	t.Run("header length write fails", func(t *testing.T) {
		w := &errorWriter{
			writesBeforeError: 1,
			err:               errors.New("disk full"),
		}
		err := WriteStreamingHeader(w, header)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write header length")
	})

	t.Run("header body write fails", func(t *testing.T) {
		w := &errorWriter{
			writesBeforeError: 2,
			err:               errors.New("disk full"),
		}
		err := WriteStreamingHeader(w, header)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write header")
	})
}

func TestReadStreamingHeader_EmptyInput(t *testing.T) {
	_, err := ReadStreamingHeader(&bytes.Buffer{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read version byte")
}

func TestReadStreamingHeader_InvalidHeaderLength(t *testing.T) {
	t.Run("header length zero", func(t *testing.T) {
		var buffer bytes.Buffer
		buffer.WriteByte(crypto_dto.StreamingEnvelopeVersion)
		buffer.Write([]byte{0, 0, 0, 0})

		_, err := ReadStreamingHeader(&buffer)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid header length")
	})

	t.Run("header length too large", func(t *testing.T) {
		var buffer bytes.Buffer
		buffer.WriteByte(crypto_dto.StreamingEnvelopeVersion)

		buffer.Write([]byte{0x00, 0x10, 0x00, 0x01})

		_, err := ReadStreamingHeader(&buffer)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid header length")
	})
}

func TestReadStreamingHeader_TruncatedHeader(t *testing.T) {
	var buffer bytes.Buffer
	buffer.WriteByte(crypto_dto.StreamingEnvelopeVersion)
	buffer.Write([]byte{0, 0, 0, 100})
	buffer.WriteString("short")

	_, err := ReadStreamingHeader(&buffer)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read header")
}

func TestReadStreamingHeader_InvalidJSON(t *testing.T) {
	var buffer bytes.Buffer
	buffer.WriteByte(crypto_dto.StreamingEnvelopeVersion)
	invalidJSON := []byte("not valid json!")
	headerLen := make([]byte, 4)
	binary.BigEndian.PutUint32(headerLen, uint32(len(invalidJSON)))
	buffer.Write(headerLen)
	buffer.Write(invalidJSON)

	_, err := ReadStreamingHeader(&buffer)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse streaming header")
}

func TestReadStreamingHeader_HeaderLengthReadFails(t *testing.T) {

	var buffer bytes.Buffer
	buffer.WriteByte(crypto_dto.StreamingEnvelopeVersion)
	buffer.Write([]byte{0, 0})

	_, err := ReadStreamingHeader(&buffer)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read header length")
}

func TestDecryptStream_InvalidIVBase64(t *testing.T) {
	key := generateTestKey()
	p, err := NewProvider(Config{Key: key, KeyID: "test"})
	require.NoError(t, err)

	header := &crypto_dto.StreamingHeader{
		Version:   2,
		KeyID:     "test",
		Provider:  "local_aes_gcm",
		IV:        "!!!not-base64!!!",
		Algorithm: "AES-256-GCM",
	}

	var buffer bytes.Buffer
	err = WriteStreamingHeader(&buffer, header)
	require.NoError(t, err)

	_, err = p.DecryptStream(context.Background(), &buffer)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode IV")
}

func TestDecryptStream_WrongIVLength(t *testing.T) {
	key := generateTestKey()
	p, err := NewProvider(Config{Key: key, KeyID: "test"})
	require.NoError(t, err)

	wrongLenIV := base64.StdEncoding.EncodeToString([]byte{1, 2, 3, 4, 5})

	header := &crypto_dto.StreamingHeader{
		Version:   2,
		KeyID:     "test",
		Provider:  "local_aes_gcm",
		IV:        wrongLenIV,
		Algorithm: "AES-256-GCM",
	}

	var buffer bytes.Buffer
	err = WriteStreamingHeader(&buffer, header)
	require.NoError(t, err)

	_, err = p.DecryptStream(context.Background(), &buffer)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid IV length")
}

func TestEncryptStream_WritesHeader(t *testing.T) {
	ctx := context.Background()
	key := generateTestKey()
	p, err := NewProvider(Config{Key: key, KeyID: "stream-key"})
	require.NoError(t, err)

	var buffer bytes.Buffer
	writer, err := p.EncryptStream(ctx, &buffer, &crypto_dto.EncryptRequest{})
	require.NoError(t, err)

	_, err = writer.Write([]byte("hello streaming"))
	require.NoError(t, err)
	err = writer.Close()
	require.NoError(t, err)

	data := buffer.Bytes()
	assert.Equal(t, crypto_dto.StreamingEnvelopeVersion, data[0])
}

func TestEncryptingWriter_FlushChunkWriteError(t *testing.T) {
	key := generateTestKey()
	block, _ := aes.NewCipher(key)
	aead, _ := cipher.NewGCM(block)

	iv := make([]byte, IVSize)
	_, _ = io.ReadFull(rand.Reader, iv)

	w := &errorWriter{
		writesBeforeError: 0,
		err:               errors.New("write failed"),
	}

	writer := NewEncryptingWriter(w, aead, iv, 8)

	_, err := writer.Write([]byte("12345678"))
	assert.Error(t, err)
}

func TestDecryptingReader_ReadChunk_InvalidChunkLength(t *testing.T) {
	key := generateTestKey()
	block, _ := aes.NewCipher(key)
	aead, _ := cipher.NewGCM(block)

	iv := make([]byte, IVSize)
	_, _ = io.ReadFull(rand.Reader, iv)

	var buffer bytes.Buffer
	buffer.Write([]byte{0, 0, 0, 0})

	reader := NewDecryptingReader(&buffer, aead, iv)
	p := make([]byte, 10)
	_, err := reader.Read(p)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid chunk length")
}

func TestDecryptingReader_ReadChunk_TruncatedCiphertext(t *testing.T) {
	key := generateTestKey()
	block, _ := aes.NewCipher(key)
	aead, _ := cipher.NewGCM(block)

	iv := make([]byte, IVSize)
	_, _ = io.ReadFull(rand.Reader, iv)

	var buffer bytes.Buffer
	buffer.Write([]byte{0, 0, 0, 100})
	buffer.Write([]byte{1, 2, 3, 4, 5})

	reader := NewDecryptingReader(&buffer, aead, iv)
	p := make([]byte, 10)
	_, err := reader.Read(p)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read encrypted chunk")
}

func TestDecryptingReader_ReadChunk_TamperedChunk(t *testing.T) {
	key := generateTestKey()
	block, _ := aes.NewCipher(key)
	aead, _ := cipher.NewGCM(block)

	iv := make([]byte, IVSize)
	_, _ = io.ReadFull(rand.Reader, iv)

	var encBuf bytes.Buffer
	writer := NewEncryptingWriter(&encBuf, aead, iv, 1024)
	_, _ = writer.Write([]byte("valid data"))
	_ = writer.Close()

	data := encBuf.Bytes()
	if len(data) > chunkLengthBytes+2 {
		data[chunkLengthBytes+2] ^= 0xFF
	}

	reader := NewDecryptingReader(bytes.NewReader(data), aead, iv)
	p := make([]byte, 100)
	_, err := reader.Read(p)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decrypt chunk")
}

func BenchmarkStreamingEncryptionSmall(b *testing.B) {
	ctx := context.Background()
	key := make([]byte, 32)
	provider, _ := NewProvider(Config{Key: key})

	plaintext := make([]byte, 1024)
	_, _ = io.ReadFull(rand.Reader, plaintext)

	b.ResetTimer()

	for b.Loop() {
		var buffer bytes.Buffer
		w, _ := provider.EncryptStream(ctx, &buffer, &crypto_dto.EncryptRequest{})
		_, _ = w.Write(plaintext)
		_ = w.Close()
	}
}

func BenchmarkStreamingEncryptionLarge(b *testing.B) {
	ctx := context.Background()
	key := make([]byte, 32)
	provider, _ := NewProvider(Config{Key: key})

	plaintext := make([]byte, 10*1024*1024)
	_, _ = io.ReadFull(rand.Reader, plaintext)

	b.ResetTimer()

	for b.Loop() {
		var buffer bytes.Buffer
		w, _ := provider.EncryptStream(ctx, &buffer, &crypto_dto.EncryptRequest{})
		_, _ = io.Copy(w, bytes.NewReader(plaintext))
		_ = w.Close()
	}
}

func BenchmarkStreamingDecryptionSmall(b *testing.B) {
	ctx := context.Background()
	key := make([]byte, 32)
	provider, _ := NewProvider(Config{Key: key})

	plaintext := make([]byte, 1024)
	_, _ = io.ReadFull(rand.Reader, plaintext)

	var ciphertextBuf bytes.Buffer
	w, _ := provider.EncryptStream(ctx, &ciphertextBuf, &crypto_dto.EncryptRequest{})
	_, _ = w.Write(plaintext)
	_ = w.Close()

	ciphertext := ciphertextBuf.Bytes()

	b.ResetTimer()

	for b.Loop() {
		r, _ := provider.DecryptStream(ctx, bytes.NewReader(ciphertext))
		_, _ = io.ReadAll(r)
		_ = r.Close()
	}
}

func BenchmarkStreamingDecryptionLarge(b *testing.B) {
	ctx := context.Background()
	key := make([]byte, 32)
	provider, _ := NewProvider(Config{Key: key})

	plaintext := make([]byte, 10*1024*1024)
	_, _ = io.ReadFull(rand.Reader, plaintext)

	var ciphertextBuf bytes.Buffer
	w, _ := provider.EncryptStream(ctx, &ciphertextBuf, &crypto_dto.EncryptRequest{})
	_, _ = io.Copy(w, bytes.NewReader(plaintext))
	_ = w.Close()

	ciphertext := ciphertextBuf.Bytes()

	b.ResetTimer()

	for b.Loop() {
		r, _ := provider.DecryptStream(ctx, bytes.NewReader(ciphertext))
		_, _ = io.ReadAll(r)
		_ = r.Close()
	}
}
