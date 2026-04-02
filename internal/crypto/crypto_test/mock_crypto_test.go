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

package crypto_test_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/crypto/crypto_domain"
	"piko.sh/piko/internal/crypto/crypto_test"
	"piko.sh/piko/internal/provider/provider_domain"
)

func TestNewMockCryptoService(t *testing.T) {
	t.Parallel()

	service := crypto_test.NewMockCryptoService()

	require.NotNil(t, service)
	_, ok := service.(*crypto_test.MockCryptoService)
	assert.True(t, ok, "NewMockCryptoService should return *MockCryptoService")
}

func TestMockCryptoService_Encrypt(t *testing.T) {
	t.Parallel()

	t.Run("nil EncryptFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &crypto_test.MockCryptoService{}

		result, err := mock.Encrypt(context.Background(), "hello")

		assert.Empty(t, result)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.EncryptCallCount))
	})

	t.Run("delegates to EncryptFunc", func(t *testing.T) {
		t.Parallel()

		var capturedPlaintext string

		mock := &crypto_test.MockCryptoService{
			EncryptFunc: func(_ context.Context, plaintext string) (string, error) {
				capturedPlaintext = plaintext
				return "encrypted:" + plaintext, nil
			},
		}

		result, err := mock.Encrypt(context.Background(), "sensitive-data")

		require.NoError(t, err)
		assert.Equal(t, "encrypted:sensitive-data", result)
		assert.Equal(t, "sensitive-data", capturedPlaintext)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.EncryptCallCount))
	})

	t.Run("propagates error from EncryptFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("encryption failed")

		mock := &crypto_test.MockCryptoService{
			EncryptFunc: func(_ context.Context, _ string) (string, error) {
				return "", expected
			},
		}

		result, err := mock.Encrypt(context.Background(), "data")

		assert.Empty(t, result)
		assert.ErrorIs(t, err, expected)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.EncryptCallCount))
	})
}

func TestMockCryptoService_Decrypt(t *testing.T) {
	t.Parallel()

	t.Run("nil DecryptFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &crypto_test.MockCryptoService{}

		result, err := mock.Decrypt(context.Background(), "ciphertext")

		assert.Empty(t, result)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.DecryptCallCount))
	})

	t.Run("delegates to DecryptFunc", func(t *testing.T) {
		t.Parallel()

		var capturedCiphertext string

		mock := &crypto_test.MockCryptoService{
			DecryptFunc: func(_ context.Context, ciphertext string) (string, error) {
				capturedCiphertext = ciphertext
				return "decrypted:" + ciphertext, nil
			},
		}

		result, err := mock.Decrypt(context.Background(), "enc-data")

		require.NoError(t, err)
		assert.Equal(t, "decrypted:enc-data", result)
		assert.Equal(t, "enc-data", capturedCiphertext)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.DecryptCallCount))
	})

	t.Run("propagates error from DecryptFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("decryption failed")

		mock := &crypto_test.MockCryptoService{
			DecryptFunc: func(_ context.Context, _ string) (string, error) {
				return "", expected
			},
		}

		result, err := mock.Decrypt(context.Background(), "bad-data")

		assert.Empty(t, result)
		assert.ErrorIs(t, err, expected)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.DecryptCallCount))
	})
}

func TestMockCryptoService_EncryptWithKey(t *testing.T) {
	t.Parallel()

	t.Run("nil EncryptWithKeyFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &crypto_test.MockCryptoService{}

		result, err := mock.EncryptWithKey(context.Background(), "hello", "key-1")

		assert.Empty(t, result)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.EncryptWithKeyCallCount))
	})

	t.Run("delegates to EncryptWithKeyFunc", func(t *testing.T) {
		t.Parallel()

		var capturedPlaintext, capturedKeyID string

		mock := &crypto_test.MockCryptoService{
			EncryptWithKeyFunc: func(_ context.Context, plaintext string, keyID string) (string, error) {
				capturedPlaintext = plaintext
				capturedKeyID = keyID
				return "enc:" + keyID + ":" + plaintext, nil
			},
		}

		result, err := mock.EncryptWithKey(context.Background(), "data", "key-42")

		require.NoError(t, err)
		assert.Equal(t, "enc:key-42:data", result)
		assert.Equal(t, "data", capturedPlaintext)
		assert.Equal(t, "key-42", capturedKeyID)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.EncryptWithKeyCallCount))
	})

	t.Run("propagates error from EncryptWithKeyFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("key not found")

		mock := &crypto_test.MockCryptoService{
			EncryptWithKeyFunc: func(_ context.Context, _ string, _ string) (string, error) {
				return "", expected
			},
		}

		result, err := mock.EncryptWithKey(context.Background(), "data", "bad-key")

		assert.Empty(t, result)
		assert.ErrorIs(t, err, expected)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.EncryptWithKeyCallCount))
	})
}

func TestMockCryptoService_EncryptBatch(t *testing.T) {
	t.Parallel()

	t.Run("nil EncryptBatchFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &crypto_test.MockCryptoService{}

		result, err := mock.EncryptBatch(context.Background(), []string{"a", "b"})

		assert.Nil(t, result)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.EncryptBatchCallCount))
	})

	t.Run("delegates to EncryptBatchFunc", func(t *testing.T) {
		t.Parallel()

		var capturedPlaintexts []string

		mock := &crypto_test.MockCryptoService{
			EncryptBatchFunc: func(_ context.Context, plaintexts []string) ([]string, error) {
				capturedPlaintexts = plaintexts
				result := make([]string, len(plaintexts))
				for i, pt := range plaintexts {
					result[i] = "enc:" + pt
				}
				return result, nil
			},
		}

		result, err := mock.EncryptBatch(context.Background(), []string{"token1", "token2"})

		require.NoError(t, err)
		assert.Equal(t, []string{"enc:token1", "enc:token2"}, result)
		assert.Equal(t, []string{"token1", "token2"}, capturedPlaintexts)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.EncryptBatchCallCount))
	})

	t.Run("propagates error from EncryptBatchFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("batch encryption failed")

		mock := &crypto_test.MockCryptoService{
			EncryptBatchFunc: func(_ context.Context, _ []string) ([]string, error) {
				return nil, expected
			},
		}

		result, err := mock.EncryptBatch(context.Background(), []string{"a"})

		assert.Nil(t, result)
		assert.ErrorIs(t, err, expected)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.EncryptBatchCallCount))
	})
}

func TestMockCryptoService_DecryptBatch(t *testing.T) {
	t.Parallel()

	t.Run("nil DecryptBatchFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &crypto_test.MockCryptoService{}

		result, err := mock.DecryptBatch(context.Background(), []string{"a", "b"})

		assert.Nil(t, result)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.DecryptBatchCallCount))
	})

	t.Run("delegates to DecryptBatchFunc", func(t *testing.T) {
		t.Parallel()

		var capturedCiphertexts []string

		mock := &crypto_test.MockCryptoService{
			DecryptBatchFunc: func(_ context.Context, ciphertexts []string) ([]string, error) {
				capturedCiphertexts = ciphertexts
				result := make([]string, len(ciphertexts))
				for i, ct := range ciphertexts {
					result[i] = "dec:" + ct
				}
				return result, nil
			},
		}

		result, err := mock.DecryptBatch(context.Background(), []string{"enc1", "enc2"})

		require.NoError(t, err)
		assert.Equal(t, []string{"dec:enc1", "dec:enc2"}, result)
		assert.Equal(t, []string{"enc1", "enc2"}, capturedCiphertexts)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.DecryptBatchCallCount))
	})

	t.Run("propagates error from DecryptBatchFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("batch decryption failed")

		mock := &crypto_test.MockCryptoService{
			DecryptBatchFunc: func(_ context.Context, _ []string) ([]string, error) {
				return nil, expected
			},
		}

		result, err := mock.DecryptBatch(context.Background(), []string{"bad"})

		assert.Nil(t, result)
		assert.ErrorIs(t, err, expected)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.DecryptBatchCallCount))
	})
}

func TestMockCryptoService_RotateKey(t *testing.T) {
	t.Parallel()

	t.Run("nil RotateKeyFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &crypto_test.MockCryptoService{}

		err := mock.RotateKey(context.Background(), "old-key", "new-key")

		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.RotateKeyCallCount))
	})

	t.Run("delegates to RotateKeyFunc", func(t *testing.T) {
		t.Parallel()

		var capturedOld, capturedNew string

		mock := &crypto_test.MockCryptoService{
			RotateKeyFunc: func(_ context.Context, oldKeyID, newKeyID string) error {
				capturedOld = oldKeyID
				capturedNew = newKeyID
				return nil
			},
		}

		err := mock.RotateKey(context.Background(), "key-v1", "key-v2")

		assert.NoError(t, err)
		assert.Equal(t, "key-v1", capturedOld)
		assert.Equal(t, "key-v2", capturedNew)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.RotateKeyCallCount))
	})

	t.Run("propagates error from RotateKeyFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("rotation failed")

		mock := &crypto_test.MockCryptoService{
			RotateKeyFunc: func(_ context.Context, _, _ string) error {
				return expected
			},
		}

		err := mock.RotateKey(context.Background(), "old", "new")

		assert.ErrorIs(t, err, expected)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.RotateKeyCallCount))
	})
}

func TestMockCryptoService_GetActiveKeyID(t *testing.T) {
	t.Parallel()

	t.Run("nil GetActiveKeyIDFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &crypto_test.MockCryptoService{}

		result, err := mock.GetActiveKeyID(context.Background())

		assert.Empty(t, result)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GetActiveKeyIDCallCount))
	})

	t.Run("delegates to GetActiveKeyIDFunc", func(t *testing.T) {
		t.Parallel()

		mock := &crypto_test.MockCryptoService{
			GetActiveKeyIDFunc: func(_ context.Context) (string, error) {
				return "active-key-42", nil
			},
		}

		result, err := mock.GetActiveKeyID(context.Background())

		require.NoError(t, err)
		assert.Equal(t, "active-key-42", result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GetActiveKeyIDCallCount))
	})

	t.Run("propagates error from GetActiveKeyIDFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("no active key")

		mock := &crypto_test.MockCryptoService{
			GetActiveKeyIDFunc: func(_ context.Context) (string, error) {
				return "", expected
			},
		}

		result, err := mock.GetActiveKeyID(context.Background())

		assert.Empty(t, result)
		assert.ErrorIs(t, err, expected)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GetActiveKeyIDCallCount))
	})
}

func TestMockCryptoService_DecryptAndReEncrypt(t *testing.T) {
	t.Parallel()

	t.Run("nil DecryptAndReEncryptFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &crypto_test.MockCryptoService{}

		plaintext, newCiphertext, wasReEncrypted, err := mock.DecryptAndReEncrypt(context.Background(), "cipher")

		assert.Empty(t, plaintext)
		assert.Empty(t, newCiphertext)
		assert.False(t, wasReEncrypted)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.DecryptAndReEncryptCallCount))
	})

	t.Run("delegates to DecryptAndReEncryptFunc", func(t *testing.T) {
		t.Parallel()

		var capturedCiphertext string

		mock := &crypto_test.MockCryptoService{
			DecryptAndReEncryptFunc: func(_ context.Context, ciphertext string) (string, string, bool, error) {
				capturedCiphertext = ciphertext
				return "plain", "new-cipher", true, nil
			},
		}

		plaintext, newCiphertext, wasReEncrypted, err := mock.DecryptAndReEncrypt(context.Background(), "old-cipher")

		require.NoError(t, err)
		assert.Equal(t, "plain", plaintext)
		assert.Equal(t, "new-cipher", newCiphertext)
		assert.True(t, wasReEncrypted)
		assert.Equal(t, "old-cipher", capturedCiphertext)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.DecryptAndReEncryptCallCount))
	})

	t.Run("propagates error from DecryptAndReEncryptFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("re-encryption failed")

		mock := &crypto_test.MockCryptoService{
			DecryptAndReEncryptFunc: func(_ context.Context, _ string) (string, string, bool, error) {
				return "", "", false, expected
			},
		}

		plaintext, newCiphertext, wasReEncrypted, err := mock.DecryptAndReEncrypt(context.Background(), "data")

		assert.Empty(t, plaintext)
		assert.Empty(t, newCiphertext)
		assert.False(t, wasReEncrypted)
		assert.ErrorIs(t, err, expected)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.DecryptAndReEncryptCallCount))
	})
}

func TestMockCryptoService_HealthCheck(t *testing.T) {
	t.Parallel()

	t.Run("nil HealthCheckFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &crypto_test.MockCryptoService{}

		err := mock.HealthCheck(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.HealthCheckCallCount))
	})

	t.Run("delegates to HealthCheckFunc", func(t *testing.T) {
		t.Parallel()

		var called bool

		mock := &crypto_test.MockCryptoService{
			HealthCheckFunc: func(_ context.Context) error {
				called = true
				return nil
			},
		}

		err := mock.HealthCheck(context.Background())

		assert.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.HealthCheckCallCount))
	})

	t.Run("propagates error from HealthCheckFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("service unhealthy")

		mock := &crypto_test.MockCryptoService{
			HealthCheckFunc: func(_ context.Context) error {
				return expected
			},
		}

		err := mock.HealthCheck(context.Background())

		assert.ErrorIs(t, err, expected)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.HealthCheckCallCount))
	})
}

func TestMockCryptoService_EncryptStream(t *testing.T) {
	t.Parallel()

	t.Run("nil EncryptStreamFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &crypto_test.MockCryptoService{}

		result, err := mock.EncryptStream(context.Background(), &bytes.Buffer{}, "key-1")

		assert.Nil(t, result)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.EncryptStreamCallCount))
	})

	t.Run("delegates to EncryptStreamFunc", func(t *testing.T) {
		t.Parallel()

		var capturedKeyID string
		expectedWriter := io.NopCloser(strings.NewReader(""))

		mock := &crypto_test.MockCryptoService{
			EncryptStreamFunc: func(_ context.Context, _ io.Writer, keyID string) (io.WriteCloser, error) {
				capturedKeyID = keyID
				return nopWriteCloser{}, nil
			},
		}

		result, err := mock.EncryptStream(context.Background(), &bytes.Buffer{}, "stream-key")

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "stream-key", capturedKeyID)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.EncryptStreamCallCount))

		_ = expectedWriter
	})

	t.Run("propagates error from EncryptStreamFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("stream encryption not supported")

		mock := &crypto_test.MockCryptoService{
			EncryptStreamFunc: func(_ context.Context, _ io.Writer, _ string) (io.WriteCloser, error) {
				return nil, expected
			},
		}

		result, err := mock.EncryptStream(context.Background(), &bytes.Buffer{}, "key")

		assert.Nil(t, result)
		assert.ErrorIs(t, err, expected)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.EncryptStreamCallCount))
	})
}

func TestMockCryptoService_DecryptStream(t *testing.T) {
	t.Parallel()

	t.Run("nil DecryptStreamFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &crypto_test.MockCryptoService{}

		result, err := mock.DecryptStream(context.Background(), strings.NewReader("data"))

		assert.Nil(t, result)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.DecryptStreamCallCount))
	})

	t.Run("delegates to DecryptStreamFunc", func(t *testing.T) {
		t.Parallel()

		expectedReader := io.NopCloser(strings.NewReader("plaintext"))

		mock := &crypto_test.MockCryptoService{
			DecryptStreamFunc: func(_ context.Context, _ io.Reader) (io.ReadCloser, error) {
				return expectedReader, nil
			},
		}

		result, err := mock.DecryptStream(context.Background(), strings.NewReader("encrypted"))

		require.NoError(t, err)
		assert.Equal(t, expectedReader, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.DecryptStreamCallCount))
	})

	t.Run("propagates error from DecryptStreamFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("stream decryption not supported")

		mock := &crypto_test.MockCryptoService{
			DecryptStreamFunc: func(_ context.Context, _ io.Reader) (io.ReadCloser, error) {
				return nil, expected
			},
		}

		result, err := mock.DecryptStream(context.Background(), strings.NewReader("bad"))

		assert.Nil(t, result)
		assert.ErrorIs(t, err, expected)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.DecryptStreamCallCount))
	})
}

func TestMockCryptoService_NewEncrypt(t *testing.T) {
	t.Parallel()

	t.Run("nil NewEncryptFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &crypto_test.MockCryptoService{}

		result := mock.NewEncrypt()

		assert.Nil(t, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.NewEncryptCallCount))
	})

	t.Run("delegates to NewEncryptFunc", func(t *testing.T) {
		t.Parallel()

		mock := &crypto_test.MockCryptoService{
			NewEncryptFunc: func() *crypto_domain.EncryptBuilder {

				return nil
			},
		}

		result := mock.NewEncrypt()

		assert.Nil(t, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.NewEncryptCallCount))
	})
}

func TestMockCryptoService_NewDecrypt(t *testing.T) {
	t.Parallel()

	t.Run("nil NewDecryptFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &crypto_test.MockCryptoService{}

		result := mock.NewDecrypt()

		assert.Nil(t, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.NewDecryptCallCount))
	})

	t.Run("delegates to NewDecryptFunc", func(t *testing.T) {
		t.Parallel()

		mock := &crypto_test.MockCryptoService{
			NewDecryptFunc: func() *crypto_domain.DecryptBuilder {
				return nil
			},
		}

		result := mock.NewDecrypt()

		assert.Nil(t, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.NewDecryptCallCount))
	})
}

func TestMockCryptoService_NewBatchEncrypt(t *testing.T) {
	t.Parallel()

	t.Run("nil NewBatchEncryptFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &crypto_test.MockCryptoService{}

		result := mock.NewBatchEncrypt()

		assert.Nil(t, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.NewBatchEncryptCallCount))
	})

	t.Run("delegates to NewBatchEncryptFunc", func(t *testing.T) {
		t.Parallel()

		mock := &crypto_test.MockCryptoService{
			NewBatchEncryptFunc: func() *crypto_domain.BatchEncryptBuilder {
				return nil
			},
		}

		result := mock.NewBatchEncrypt()

		assert.Nil(t, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.NewBatchEncryptCallCount))
	})
}

func TestMockCryptoService_NewBatchDecrypt(t *testing.T) {
	t.Parallel()

	t.Run("nil NewBatchDecryptFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &crypto_test.MockCryptoService{}

		result := mock.NewBatchDecrypt()

		assert.Nil(t, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.NewBatchDecryptCallCount))
	})

	t.Run("delegates to NewBatchDecryptFunc", func(t *testing.T) {
		t.Parallel()

		mock := &crypto_test.MockCryptoService{
			NewBatchDecryptFunc: func() *crypto_domain.BatchDecryptBuilder {
				return nil
			},
		}

		result := mock.NewBatchDecrypt()

		assert.Nil(t, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.NewBatchDecryptCallCount))
	})
}

func TestMockCryptoService_NewStreamEncrypt(t *testing.T) {
	t.Parallel()

	t.Run("nil NewStreamEncryptFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &crypto_test.MockCryptoService{}

		result := mock.NewStreamEncrypt()

		assert.Nil(t, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.NewStreamEncryptCallCount))
	})

	t.Run("delegates to NewStreamEncryptFunc", func(t *testing.T) {
		t.Parallel()

		mock := &crypto_test.MockCryptoService{
			NewStreamEncryptFunc: func() *crypto_domain.StreamEncryptBuilder {
				return nil
			},
		}

		result := mock.NewStreamEncrypt()

		assert.Nil(t, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.NewStreamEncryptCallCount))
	})
}

func TestMockCryptoService_NewStreamDecrypt(t *testing.T) {
	t.Parallel()

	t.Run("nil NewStreamDecryptFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &crypto_test.MockCryptoService{}

		result := mock.NewStreamDecrypt()

		assert.Nil(t, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.NewStreamDecryptCallCount))
	})

	t.Run("delegates to NewStreamDecryptFunc", func(t *testing.T) {
		t.Parallel()

		mock := &crypto_test.MockCryptoService{
			NewStreamDecryptFunc: func() *crypto_domain.StreamDecryptBuilder {
				return nil
			},
		}

		result := mock.NewStreamDecrypt()

		assert.Nil(t, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.NewStreamDecryptCallCount))
	})
}

func TestMockCryptoService_RegisterProvider(t *testing.T) {
	t.Parallel()

	t.Run("nil RegisterProviderFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &crypto_test.MockCryptoService{}

		err := mock.RegisterProvider(context.Background(), "local-aes", nil)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.RegisterProviderCallCount))
	})

	t.Run("delegates to RegisterProviderFunc", func(t *testing.T) {
		t.Parallel()

		var capturedName string

		mock := &crypto_test.MockCryptoService{
			RegisterProviderFunc: func(_ context.Context, name string, _ crypto_domain.EncryptionProvider) error {
				capturedName = name
				return nil
			},
		}

		err := mock.RegisterProvider(context.Background(), "aws-kms", nil)

		assert.NoError(t, err)
		assert.Equal(t, "aws-kms", capturedName)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.RegisterProviderCallCount))
	})

	t.Run("propagates error from RegisterProviderFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("duplicate provider")

		mock := &crypto_test.MockCryptoService{
			RegisterProviderFunc: func(_ context.Context, _ string, _ crypto_domain.EncryptionProvider) error {
				return expected
			},
		}

		err := mock.RegisterProvider(context.Background(), "dupe", nil)

		assert.ErrorIs(t, err, expected)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.RegisterProviderCallCount))
	})
}

func TestMockCryptoService_SetDefaultProvider(t *testing.T) {
	t.Parallel()

	t.Run("nil SetDefaultProviderFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &crypto_test.MockCryptoService{}

		err := mock.SetDefaultProvider("local")

		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.SetDefaultProviderCallCount))
	})

	t.Run("delegates to SetDefaultProviderFunc", func(t *testing.T) {
		t.Parallel()

		var capturedName string

		mock := &crypto_test.MockCryptoService{
			SetDefaultProviderFunc: func(name string) error {
				capturedName = name
				return nil
			},
		}

		err := mock.SetDefaultProvider("aws-kms")

		assert.NoError(t, err)
		assert.Equal(t, "aws-kms", capturedName)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.SetDefaultProviderCallCount))
	})

	t.Run("propagates error from SetDefaultProviderFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("provider not found")

		mock := &crypto_test.MockCryptoService{
			SetDefaultProviderFunc: func(_ string) error {
				return expected
			},
		}

		err := mock.SetDefaultProvider("nonexistent")

		assert.ErrorIs(t, err, expected)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.SetDefaultProviderCallCount))
	})
}

func TestMockCryptoService_GetProviders(t *testing.T) {
	t.Parallel()

	t.Run("nil GetProvidersFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &crypto_test.MockCryptoService{}

		result := mock.GetProviders(context.Background())

		assert.Nil(t, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GetProvidersCallCount))
	})

	t.Run("delegates to GetProvidersFunc", func(t *testing.T) {
		t.Parallel()

		expected := []string{"aws-kms", "local-aes"}

		mock := &crypto_test.MockCryptoService{
			GetProvidersFunc: func(_ context.Context) []string {
				return expected
			},
		}

		result := mock.GetProviders(context.Background())

		assert.Equal(t, expected, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GetProvidersCallCount))
	})
}

func TestMockCryptoService_HasProvider(t *testing.T) {
	t.Parallel()

	t.Run("nil HasProviderFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &crypto_test.MockCryptoService{}

		result := mock.HasProvider("any")

		assert.False(t, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.HasProviderCallCount))
	})

	t.Run("delegates to HasProviderFunc", func(t *testing.T) {
		t.Parallel()

		var capturedName string

		mock := &crypto_test.MockCryptoService{
			HasProviderFunc: func(name string) bool {
				capturedName = name
				return name == "mock"
			},
		}

		assert.True(t, mock.HasProvider("mock"))
		assert.Equal(t, "mock", capturedName)
		assert.False(t, mock.HasProvider("nonexistent"))
		assert.Equal(t, int64(2), atomic.LoadInt64(&mock.HasProviderCallCount))
	})
}

func TestMockCryptoService_ListProviders(t *testing.T) {
	t.Parallel()

	t.Run("nil ListProvidersFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &crypto_test.MockCryptoService{}

		result := mock.ListProviders(context.Background())

		assert.Nil(t, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ListProvidersCallCount))
	})

	t.Run("delegates to ListProvidersFunc", func(t *testing.T) {
		t.Parallel()

		expected := []provider_domain.ProviderInfo{
			{
				Name:         "mock",
				ProviderType: "mock",
				IsDefault:    true,
			},
		}

		mock := &crypto_test.MockCryptoService{
			ListProvidersFunc: func(_ context.Context) []provider_domain.ProviderInfo {
				return expected
			},
		}

		result := mock.ListProviders(context.Background())

		require.Len(t, result, 1)
		assert.Equal(t, expected, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ListProvidersCallCount))
	})
}

func TestMockCryptoService_Close(t *testing.T) {
	t.Parallel()

	t.Run("nil CloseFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &crypto_test.MockCryptoService{}

		err := mock.Close(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.CloseCallCount))
	})

	t.Run("delegates to CloseFunc", func(t *testing.T) {
		t.Parallel()

		var called bool

		mock := &crypto_test.MockCryptoService{
			CloseFunc: func(_ context.Context) error {
				called = true
				return nil
			},
		}

		err := mock.Close(context.Background())

		assert.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.CloseCallCount))
	})

	t.Run("propagates error from CloseFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("close failed")

		mock := &crypto_test.MockCryptoService{
			CloseFunc: func(_ context.Context) error {
				return expected
			},
		}

		err := mock.Close(context.Background())

		assert.ErrorIs(t, err, expected)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.CloseCallCount))
	})
}

func TestMockCryptoService_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var mock crypto_test.MockCryptoService
	ctx := context.Background()

	encResult, encErr := mock.Encrypt(ctx, "plaintext")
	assert.Empty(t, encResult)
	assert.NoError(t, encErr)

	decResult, decErr := mock.Decrypt(ctx, "ciphertext")
	assert.Empty(t, decResult)
	assert.NoError(t, decErr)

	ewkResult, ewkErr := mock.EncryptWithKey(ctx, "data", "key")
	assert.Empty(t, ewkResult)
	assert.NoError(t, ewkErr)

	ebResult, ebErr := mock.EncryptBatch(ctx, []string{"a"})
	assert.Nil(t, ebResult)
	assert.NoError(t, ebErr)

	dbResult, dbErr := mock.DecryptBatch(ctx, []string{"b"})
	assert.Nil(t, dbResult)
	assert.NoError(t, dbErr)

	rkErr := mock.RotateKey(ctx, "old", "new")
	assert.NoError(t, rkErr)

	akResult, akErr := mock.GetActiveKeyID(ctx)
	assert.Empty(t, akResult)
	assert.NoError(t, akErr)

	drPlain, drCipher, drReEncrypted, drErr := mock.DecryptAndReEncrypt(ctx, "cipher")
	assert.Empty(t, drPlain)
	assert.Empty(t, drCipher)
	assert.False(t, drReEncrypted)
	assert.NoError(t, drErr)

	hcErr := mock.HealthCheck(ctx)
	assert.NoError(t, hcErr)

	esResult, esErr := mock.EncryptStream(ctx, &bytes.Buffer{}, "key")
	assert.Nil(t, esResult)
	assert.NoError(t, esErr)

	dsResult, dsErr := mock.DecryptStream(ctx, strings.NewReader("data"))
	assert.Nil(t, dsResult)
	assert.NoError(t, dsErr)

	assert.Nil(t, mock.NewEncrypt())
	assert.Nil(t, mock.NewDecrypt())
	assert.Nil(t, mock.NewBatchEncrypt())
	assert.Nil(t, mock.NewBatchDecrypt())
	assert.Nil(t, mock.NewStreamEncrypt())
	assert.Nil(t, mock.NewStreamDecrypt())

	rpErr := mock.RegisterProvider(ctx, "test", nil)
	assert.NoError(t, rpErr)

	sdErr := mock.SetDefaultProvider("test")
	assert.NoError(t, sdErr)

	assert.Nil(t, mock.GetProviders(ctx))
	assert.False(t, mock.HasProvider("any"))
	assert.Nil(t, mock.ListProviders(ctx))

	clErr := mock.Close(ctx)
	assert.NoError(t, clErr)

	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.EncryptCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.DecryptCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.EncryptWithKeyCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.EncryptBatchCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.DecryptBatchCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.RotateKeyCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GetActiveKeyIDCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.DecryptAndReEncryptCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.HealthCheckCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.EncryptStreamCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.DecryptStreamCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.NewEncryptCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.NewDecryptCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.NewBatchEncryptCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.NewBatchDecryptCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.NewStreamEncryptCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.NewStreamDecryptCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.RegisterProviderCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.SetDefaultProviderCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GetProvidersCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.HasProviderCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ListProvidersCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.CloseCallCount))
}

func TestMockCryptoService_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	mock := &crypto_test.MockCryptoService{
		EncryptFunc: func(_ context.Context, plaintext string) (string, error) {
			return "enc:" + plaintext, nil
		},
		DecryptFunc: func(_ context.Context, ciphertext string) (string, error) {
			return "dec:" + ciphertext, nil
		},
		EncryptWithKeyFunc: func(_ context.Context, plaintext, keyID string) (string, error) {
			return "ewk:" + plaintext, nil
		},
		EncryptBatchFunc: func(_ context.Context, plaintexts []string) ([]string, error) {
			return plaintexts, nil
		},
		DecryptBatchFunc: func(_ context.Context, ciphertexts []string) ([]string, error) {
			return ciphertexts, nil
		},
		RotateKeyFunc: func(_ context.Context, _, _ string) error {
			return nil
		},
		GetActiveKeyIDFunc: func(_ context.Context) (string, error) {
			return "key-1", nil
		},
		DecryptAndReEncryptFunc: func(_ context.Context, ct string) (string, string, bool, error) {
			return ct, "", false, nil
		},
		HealthCheckFunc: func(_ context.Context) error {
			return nil
		},
		EncryptStreamFunc: func(_ context.Context, _ io.Writer, _ string) (io.WriteCloser, error) {
			return nopWriteCloser{}, nil
		},
		DecryptStreamFunc: func(_ context.Context, _ io.Reader) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader("")), nil
		},
		NewEncryptFunc: func() *crypto_domain.EncryptBuilder {
			return nil
		},
		NewDecryptFunc: func() *crypto_domain.DecryptBuilder {
			return nil
		},
		NewBatchEncryptFunc: func() *crypto_domain.BatchEncryptBuilder {
			return nil
		},
		NewBatchDecryptFunc: func() *crypto_domain.BatchDecryptBuilder {
			return nil
		},
		NewStreamEncryptFunc: func() *crypto_domain.StreamEncryptBuilder {
			return nil
		},
		NewStreamDecryptFunc: func() *crypto_domain.StreamDecryptBuilder {
			return nil
		},
		RegisterProviderFunc: func(_ context.Context, _ string, _ crypto_domain.EncryptionProvider) error {
			return nil
		},
		SetDefaultProviderFunc: func(_ string) error {
			return nil
		},
		GetProvidersFunc: func(_ context.Context) []string {
			return []string{"mock"}
		},
		HasProviderFunc: func(_ string) bool {
			return true
		},
		ListProvidersFunc: func(_ context.Context) []provider_domain.ProviderInfo {
			return nil
		},
		CloseFunc: func(_ context.Context) error {
			return nil
		},
	}

	const goroutines = 50
	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			_, _ = mock.Encrypt(ctx, "data")
			_, _ = mock.Decrypt(ctx, "data")
			_, _ = mock.EncryptWithKey(ctx, "data", "key")
			_, _ = mock.EncryptBatch(ctx, []string{"a"})
			_, _ = mock.DecryptBatch(ctx, []string{"b"})
			_ = mock.RotateKey(ctx, "old", "new")
			_, _ = mock.GetActiveKeyID(ctx)
			_, _, _, _ = mock.DecryptAndReEncrypt(ctx, "ct")
			_ = mock.HealthCheck(ctx)
			_, _ = mock.EncryptStream(ctx, &bytes.Buffer{}, "key")
			_, _ = mock.DecryptStream(ctx, strings.NewReader("in"))
			_ = mock.NewEncrypt()
			_ = mock.NewDecrypt()
			_ = mock.NewBatchEncrypt()
			_ = mock.NewBatchDecrypt()
			_ = mock.NewStreamEncrypt()
			_ = mock.NewStreamDecrypt()
			_ = mock.RegisterProvider(ctx, "p", nil)
			_ = mock.SetDefaultProvider("p")
			_ = mock.GetProviders(ctx)
			_ = mock.HasProvider("p")
			_ = mock.ListProviders(ctx)
			_ = mock.Close(ctx)
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.EncryptCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.DecryptCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.EncryptWithKeyCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.EncryptBatchCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.DecryptBatchCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.RotateKeyCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.GetActiveKeyIDCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.DecryptAndReEncryptCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.HealthCheckCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.EncryptStreamCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.DecryptStreamCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.NewEncryptCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.NewDecryptCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.NewBatchEncryptCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.NewBatchDecryptCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.NewStreamEncryptCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.NewStreamDecryptCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.RegisterProviderCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.SetDefaultProviderCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.GetProvidersCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.HasProviderCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.ListProvidersCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.CloseCallCount))
}

func TestMockCryptoService_ImplementsInterface(t *testing.T) {
	t.Parallel()

	var _ crypto_domain.CryptoServicePort = (*crypto_test.MockCryptoService)(nil)
}

type nopWriteCloser struct{}

func (nopWriteCloser) Write(p []byte) (int, error) { return len(p), nil }
func (nopWriteCloser) Close() error                { return nil }
