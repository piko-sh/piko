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

package crypto_domain

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/crypto/crypto_dto"
)

func TestEncryptStream(t *testing.T) {
	t.Parallel()

	t.Run("returns writer for encryption", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createTestConfig("test-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		var output bytes.Buffer
		writer, err := service.EncryptStream(context.Background(), &output, "test-key")

		require.NoError(t, err)
		assert.NotNil(t, writer)

		err = writer.Close()
		require.NoError(t, err)
	})

	t.Run("uses active key when keyID is empty", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		var capturedKeyID string
		provider.encryptStreamFunc = func(_ context.Context, output io.Writer, request *crypto_dto.EncryptRequest) (io.WriteCloser, error) {
			capturedKeyID = request.KeyID
			return &mockWriteCloser{output: output}, nil
		}

		config := createTestConfig("my-active-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		var output bytes.Buffer
		writer, err := service.EncryptStream(context.Background(), &output, "")

		require.NoError(t, err)
		assert.NotNil(t, writer)
		assert.Equal(t, "my-active-key", capturedKeyID)

		err = writer.Close()
		require.NoError(t, err)
	})

	t.Run("uses specified key when provided", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		var capturedKeyID string
		provider.encryptStreamFunc = func(_ context.Context, output io.Writer, request *crypto_dto.EncryptRequest) (io.WriteCloser, error) {
			capturedKeyID = request.KeyID
			return &mockWriteCloser{output: output}, nil
		}

		config := createTestConfig("active-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		var output bytes.Buffer
		writer, err := service.EncryptStream(context.Background(), &output, "specific-key")

		require.NoError(t, err)
		assert.NotNil(t, writer)
		assert.Equal(t, "specific-key", capturedKeyID)

		err = writer.Close()
		require.NoError(t, err)
	})

	t.Run("returns error when provider fails", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		provider.encryptStreamFunc = func(_ context.Context, _ io.Writer, _ *crypto_dto.EncryptRequest) (io.WriteCloser, error) {
			return nil, errors.New("stream encryption failed")
		}

		config := createTestConfig("test-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		var output bytes.Buffer
		_, err = service.EncryptStream(context.Background(), &output, "test-key")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "EncryptStream")
	})

	t.Run("writes data through encryption writer", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createTestConfig("test-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		var output bytes.Buffer
		writer, err := service.EncryptStream(context.Background(), &output, "test-key")
		require.NoError(t, err)

		data := []byte("hello world")
		n, err := writer.Write(data)

		require.NoError(t, err)
		assert.Equal(t, len(data), n)

		err = writer.Close()
		require.NoError(t, err)
	})
}

func TestDecryptStream(t *testing.T) {
	t.Parallel()

	t.Run("returns reader for decryption", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createTestConfig("test-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		input := bytes.NewReader([]byte("encrypted-data"))
		reader, err := service.DecryptStream(context.Background(), input)

		require.NoError(t, err)
		assert.NotNil(t, reader)

		err = reader.Close()
		require.NoError(t, err)
	})

	t.Run("returns error when provider fails", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		provider.decryptStreamFunc = func(_ context.Context, _ io.Reader) (io.ReadCloser, error) {
			return nil, errors.New("stream decryption failed")
		}

		config := createTestConfig("test-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		input := bytes.NewReader([]byte("encrypted-data"))
		_, err = service.DecryptStream(context.Background(), input)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "DecryptStream")
	})

	t.Run("reads data through decryption reader", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createTestConfig("test-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		expectedData := []byte("plaintext data")
		input := bytes.NewReader(expectedData)
		reader, err := service.DecryptStream(context.Background(), input)
		require.NoError(t, err)

		result, err := io.ReadAll(reader)

		require.NoError(t, err)
		assert.Equal(t, expectedData, result)

		err = reader.Close()
		require.NoError(t, err)
	})
}

func TestStreamRoundtrip(t *testing.T) {
	t.Parallel()

	t.Run("encrypt then decrypt stream preserves data", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createTestConfig("test-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		originalData := []byte("This is a test message for streaming encryption.")

		var encrypted bytes.Buffer
		encWriter, err := service.EncryptStream(context.Background(), &encrypted, "test-key")
		require.NoError(t, err)

		_, err = encWriter.Write(originalData)
		require.NoError(t, err)

		err = encWriter.Close()
		require.NoError(t, err)

		decReader, err := service.DecryptStream(context.Background(), &encrypted)
		require.NoError(t, err)

		decrypted, err := io.ReadAll(decReader)
		require.NoError(t, err)

		err = decReader.Close()
		require.NoError(t, err)

		assert.Equal(t, originalData, decrypted)
	})

	t.Run("handles large data streams", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createTestConfig("test-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		largeData := make([]byte, 1024*1024)
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}

		var encrypted bytes.Buffer
		encWriter, err := service.EncryptStream(context.Background(), &encrypted, "test-key")
		require.NoError(t, err)

		_, err = encWriter.Write(largeData)
		require.NoError(t, err)

		err = encWriter.Close()
		require.NoError(t, err)

		decReader, err := service.DecryptStream(context.Background(), &encrypted)
		require.NoError(t, err)

		decrypted, err := io.ReadAll(decReader)
		require.NoError(t, err)

		err = decReader.Close()
		require.NoError(t, err)

		assert.Equal(t, largeData, decrypted)
	})
}
