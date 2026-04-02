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

package cache_transformer_crypto

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/crypto/crypto_domain"
	"piko.sh/piko/internal/provider/provider_domain"
)

type mockCryptoService struct {
	encryptFunc func(ctx context.Context, plaintext string) (string, error)
	decryptFunc func(ctx context.Context, ciphertext string) (string, error)
}

func (m *mockCryptoService) Encrypt(ctx context.Context, plaintext string) (string, error) {
	if m.encryptFunc != nil {
		return m.encryptFunc(ctx, plaintext)
	}
	return base64.StdEncoding.EncodeToString([]byte(plaintext)), nil
}

func (m *mockCryptoService) Decrypt(ctx context.Context, ciphertext string) (string, error) {
	if m.decryptFunc != nil {
		return m.decryptFunc(ctx, ciphertext)
	}
	decoded, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

func (*mockCryptoService) EncryptWithKey(context.Context, string, string) (string, error) {
	panic("not implemented")
}

func (*mockCryptoService) EncryptBatch(context.Context, []string) ([]string, error) {
	panic("not implemented")
}

func (*mockCryptoService) DecryptBatch(context.Context, []string) ([]string, error) {
	panic("not implemented")
}

func (*mockCryptoService) RotateKey(context.Context, string, string) error {
	panic("not implemented")
}

func (*mockCryptoService) GetActiveKeyID(context.Context) (string, error) {
	panic("not implemented")
}

func (*mockCryptoService) DecryptAndReEncrypt(context.Context, string) (string, string, bool, error) {
	panic("not implemented")
}

func (*mockCryptoService) HealthCheck(context.Context) error {
	panic("not implemented")
}

func (*mockCryptoService) EncryptStream(context.Context, io.Writer, string) (io.WriteCloser, error) {
	panic("not implemented")
}

func (*mockCryptoService) DecryptStream(context.Context, io.Reader) (io.ReadCloser, error) {
	panic("not implemented")
}

func (*mockCryptoService) NewEncrypt() *crypto_domain.EncryptBuilder {
	panic("not implemented")
}

func (*mockCryptoService) NewDecrypt() *crypto_domain.DecryptBuilder {
	panic("not implemented")
}

func (*mockCryptoService) NewBatchEncrypt() *crypto_domain.BatchEncryptBuilder {
	panic("not implemented")
}

func (*mockCryptoService) NewBatchDecrypt() *crypto_domain.BatchDecryptBuilder {
	panic("not implemented")
}

func (*mockCryptoService) NewStreamEncrypt() *crypto_domain.StreamEncryptBuilder {
	panic("not implemented")
}

func (*mockCryptoService) NewStreamDecrypt() *crypto_domain.StreamDecryptBuilder {
	panic("not implemented")
}

func (*mockCryptoService) RegisterProvider(context.Context, string, crypto_domain.EncryptionProvider) error {
	panic("not implemented")
}

func (*mockCryptoService) SetDefaultProvider(string) error {
	panic("not implemented")
}

func (*mockCryptoService) GetProviders(context.Context) []string {
	panic("not implemented")
}

func (*mockCryptoService) HasProvider(string) bool {
	panic("not implemented")
}

func (*mockCryptoService) ListProviders(context.Context) []provider_domain.ProviderInfo {
	panic("not implemented")
}

func (*mockCryptoService) Close(context.Context) error {
	panic("not implemented")
}

var _ crypto_domain.CryptoServicePort = (*mockCryptoService)(nil)

func TestNew_DefaultValues(t *testing.T) {
	t.Parallel()

	transformer := New(&mockCryptoService{}, "", 0)

	assert.Equal(t, "crypto-service", transformer.Name())
	assert.Equal(t, 250, transformer.Priority())
}

func TestNew_CustomValues(t *testing.T) {
	t.Parallel()

	transformer := New(&mockCryptoService{}, "my-crypto", 300)

	assert.Equal(t, "my-crypto", transformer.Name())
	assert.Equal(t, 300, transformer.Priority())
}

func TestName(t *testing.T) {
	t.Parallel()

	transformer := New(&mockCryptoService{}, "test-name", 0)

	assert.Equal(t, "test-name", transformer.Name())
}

func TestType(t *testing.T) {
	t.Parallel()

	transformer := New(&mockCryptoService{}, "", 0)

	assert.Equal(t, cache_dto.TransformerEncryption, transformer.Type())
}

func TestPriority(t *testing.T) {
	t.Parallel()

	transformer := New(&mockCryptoService{}, "", 42)

	assert.Equal(t, 42, transformer.Priority())
}

func TestTransformReverse_RoundTrip(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input []byte
	}{
		{name: "small text", input: []byte("hello world")},
		{name: "large text", input: []byte(strings.Repeat("the quick brown fox jumps over the lazy dog. ", 25))},
		{name: "binary data", input: func() []byte {
			data := make([]byte, 256)
			for i := range data {
				data[i] = byte(i)
			}
			return data
		}()},
		{name: "UTF-8 text with special characters", input: []byte("héllo wörld 日本語 🌍 café résumé naïve")},
	}

	transformer := New(&mockCryptoService{}, "", 0)
	ctx := context.Background()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			encrypted, err := transformer.Transform(ctx, testCase.input, nil)
			require.NoError(t, err)

			decrypted, err := transformer.Reverse(ctx, encrypted, nil)
			require.NoError(t, err)

			assert.Equal(t, testCase.input, decrypted)
		})
	}
}

func TestTransform_EmptyInput(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input []byte
	}{
		{name: "nil input", input: nil},
		{name: "empty slice", input: []byte{}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var encryptCalled atomic.Bool
			mock := &mockCryptoService{
				encryptFunc: func(_ context.Context, plaintext string) (string, error) {
					encryptCalled.Store(true)
					return base64.StdEncoding.EncodeToString([]byte(plaintext)), nil
				},
			}
			transformer := New(mock, "", 0)
			ctx := context.Background()

			result, err := transformer.Transform(ctx, testCase.input, nil)
			require.NoError(t, err)

			assert.Equal(t, testCase.input, result)
			assert.False(t, encryptCalled.Load(), "Encrypt should not have been called for empty input")
		})
	}
}

func TestReverse_EmptyInput(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input []byte
	}{
		{name: "nil input", input: nil},
		{name: "empty slice", input: []byte{}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var decryptCalled atomic.Bool
			mock := &mockCryptoService{
				decryptFunc: func(_ context.Context, ciphertext string) (string, error) {
					decryptCalled.Store(true)
					decoded, err := base64.StdEncoding.DecodeString(ciphertext)
					if err != nil {
						return "", err
					}
					return string(decoded), nil
				},
			}
			transformer := New(mock, "", 0)
			ctx := context.Background()

			result, err := transformer.Reverse(ctx, testCase.input, nil)
			require.NoError(t, err)

			assert.Equal(t, testCase.input, result)
			assert.False(t, decryptCalled.Load(), "Decrypt should not have been called for empty input")
		})
	}
}

func TestTransform_EncryptError(t *testing.T) {
	t.Parallel()

	expectedError := errors.New("encryption provider unavailable")
	mock := &mockCryptoService{
		encryptFunc: func(context.Context, string) (string, error) {
			return "", expectedError
		},
	}
	transformer := New(mock, "", 0)
	ctx := context.Background()

	result, err := transformer.Transform(ctx, []byte("sensitive data"), nil)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "crypto service encryption failed")
	assert.ErrorIs(t, err, expectedError)
}

func TestReverse_DecryptError(t *testing.T) {
	t.Parallel()

	expectedError := errors.New("decryption key expired")
	mock := &mockCryptoService{
		decryptFunc: func(context.Context, string) (string, error) {
			return "", expectedError
		},
	}
	transformer := New(mock, "", 0)
	ctx := context.Background()

	result, err := transformer.Reverse(ctx, []byte("encrypted-data"), nil)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "crypto service decryption failed")
	assert.ErrorIs(t, err, expectedError)
}

func TestParseConfigValues(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		config           any
		expectedService  crypto_domain.CryptoServicePort
		expectedName     string
		expectedPriority int
	}{
		{
			name:             "nil config",
			config:           nil,
			expectedService:  nil,
			expectedName:     "",
			expectedPriority: 0,
		},
		{
			name: "Config struct",
			config: Config{
				CryptoService: &mockCryptoService{},
				Name:          "from-struct",
				Priority:      150,
			},
			expectedService:  &mockCryptoService{},
			expectedName:     "from-struct",
			expectedPriority: 150,
		},
		{
			name: "map with all keys",
			config: map[string]any{
				"cryptoService": &mockCryptoService{},
				"name":          "from-map",
				"priority":      200,
			},
			expectedService:  &mockCryptoService{},
			expectedName:     "from-map",
			expectedPriority: 200,
		},
		{
			name:             "unknown type",
			config:           12345,
			expectedService:  nil,
			expectedName:     "",
			expectedPriority: 0,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service, name, priority := parseConfigValues(testCase.config)

			if testCase.expectedService == nil {
				assert.Nil(t, service)
			} else {
				assert.NotNil(t, service)
			}
			assert.Equal(t, testCase.expectedName, name)
			assert.Equal(t, testCase.expectedPriority, priority)
		})
	}
}

func TestParseMapConfig(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		config           map[string]any
		expectedService  crypto_domain.CryptoServicePort
		expectedName     string
		expectedPriority int
	}{
		{
			name: "all keys present",
			config: map[string]any{
				"cryptoService": &mockCryptoService{},
				"name":          "full-map",
				"priority":      175,
			},
			expectedService:  &mockCryptoService{},
			expectedName:     "full-map",
			expectedPriority: 175,
		},
		{
			name:             "missing keys",
			config:           map[string]any{},
			expectedService:  nil,
			expectedName:     "",
			expectedPriority: 0,
		},
		{
			name: "wrong value types",
			config: map[string]any{
				"cryptoService": "not-a-service",
				"name":          12345,
				"priority":      "not-an-int",
			},
			expectedService:  nil,
			expectedName:     "",
			expectedPriority: 0,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service, name, priority := parseMapConfig(testCase.config)

			if testCase.expectedService == nil {
				assert.Nil(t, service)
			} else {
				assert.NotNil(t, service)
			}
			assert.Equal(t, testCase.expectedName, name)
			assert.Equal(t, testCase.expectedPriority, priority)
		})
	}
}

func TestCreateTransformerFromConfig(t *testing.T) {
	t.Parallel()

	mock := &mockCryptoService{}
	config := Config{
		CryptoService: mock,
		Name:          "config-transformer",
		Priority:      300,
	}

	transformer, err := createTransformerFromConfig(config)

	require.NoError(t, err)
	assert.Equal(t, "config-transformer", transformer.Name())
	assert.Equal(t, 300, transformer.Priority())
	assert.Equal(t, cache_dto.TransformerEncryption, transformer.Type())
}
