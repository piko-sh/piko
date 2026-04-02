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

package transformer_crypto

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/crypto/crypto_domain"
	"piko.sh/piko/internal/provider/provider_domain"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
)

var _ storage_domain.StreamTransformerPort = (*CryptoTransformer)(nil)

type stubCryptoService struct {
	encryptStreamFunc func(ctx context.Context, output io.Writer, keyID string) (io.WriteCloser, error)
	decryptStreamFunc func(ctx context.Context, input io.Reader) (io.ReadCloser, error)
}

func (s *stubCryptoService) Encrypt(context.Context, string) (string, error) {
	return "", nil
}

func (s *stubCryptoService) Decrypt(context.Context, string) (string, error) {
	return "", nil
}

func (s *stubCryptoService) EncryptWithKey(context.Context, string, string) (string, error) {
	return "", nil
}

func (s *stubCryptoService) EncryptBatch(context.Context, []string) ([]string, error) {
	return nil, nil
}

func (s *stubCryptoService) DecryptBatch(context.Context, []string) ([]string, error) {
	return nil, nil
}

func (s *stubCryptoService) RotateKey(context.Context, string, string) error {
	return nil
}

func (s *stubCryptoService) GetActiveKeyID(context.Context) (string, error) {
	return "", nil
}

func (s *stubCryptoService) DecryptAndReEncrypt(context.Context, string) (string, string, bool, error) {
	return "", "", false, nil
}

func (s *stubCryptoService) HealthCheck(context.Context) error {
	return nil
}

func (s *stubCryptoService) EncryptStream(ctx context.Context, output io.Writer, keyID string) (io.WriteCloser, error) {
	if s.encryptStreamFunc != nil {
		return s.encryptStreamFunc(ctx, output, keyID)
	}
	return nil, errors.New("not implemented")
}

func (s *stubCryptoService) DecryptStream(ctx context.Context, input io.Reader) (io.ReadCloser, error) {
	if s.decryptStreamFunc != nil {
		return s.decryptStreamFunc(ctx, input)
	}
	return nil, errors.New("not implemented")
}

func (s *stubCryptoService) NewEncrypt() *crypto_domain.EncryptBuilder { return nil }
func (s *stubCryptoService) NewDecrypt() *crypto_domain.DecryptBuilder { return nil }
func (s *stubCryptoService) NewBatchEncrypt() *crypto_domain.BatchEncryptBuilder {
	return nil
}
func (s *stubCryptoService) NewBatchDecrypt() *crypto_domain.BatchDecryptBuilder {
	return nil
}
func (s *stubCryptoService) NewStreamEncrypt() *crypto_domain.StreamEncryptBuilder {
	return nil
}
func (s *stubCryptoService) NewStreamDecrypt() *crypto_domain.StreamDecryptBuilder {
	return nil
}

func (s *stubCryptoService) RegisterProvider(context.Context, string, crypto_domain.EncryptionProvider) error {
	return nil
}

func (s *stubCryptoService) SetDefaultProvider(string) error { return nil }

func (s *stubCryptoService) GetProviders(context.Context) []string { return nil }

func (s *stubCryptoService) HasProvider(string) bool { return false }

func (s *stubCryptoService) ListProviders(context.Context) []provider_domain.ProviderInfo {
	return nil
}

func (s *stubCryptoService) Close(context.Context) error { return nil }

var _ crypto_domain.CryptoServicePort = (*stubCryptoService)(nil)

type passthroughWriteCloser struct {
	writer io.Writer
}

func (p *passthroughWriteCloser) Write(data []byte) (int, error) {
	return p.writer.Write(data)
}

func (p *passthroughWriteCloser) Close() error {
	return nil
}

type failingWriteCloser struct {
	writer   io.Writer
	closeErr error
}

func (f *failingWriteCloser) Write(data []byte) (int, error) {
	return f.writer.Write(data)
}

func (f *failingWriteCloser) Close() error {
	return f.closeErr
}

func TestCryptoTransformer_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	var transformer storage_domain.StreamTransformerPort = New(nil, "", 0)
	require.NotNil(t, transformer)
}

func TestCryptoTransformer_Name(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		inputName    string
		expectedName string
	}{
		{name: "default name", inputName: "", expectedName: "crypto-service"},
		{name: "custom name", inputName: "my-encryption", expectedName: "my-encryption"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			transformer := New(nil, testCase.inputName, 0)
			assert.Equal(t, testCase.expectedName, transformer.Name())
		})
	}
}

func TestCryptoTransformer_Priority(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		inputPriority    int
		expectedPriority int
	}{
		{name: "default priority", inputPriority: 0, expectedPriority: 250},
		{name: "custom priority", inputPriority: 100, expectedPriority: 100},
		{name: "high priority", inputPriority: 500, expectedPriority: 500},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			transformer := New(nil, "", testCase.inputPriority)
			assert.Equal(t, testCase.expectedPriority, transformer.Priority())
		})
	}
}

func TestCryptoTransformer_Type(t *testing.T) {
	t.Parallel()

	transformer := New(nil, "", 0)
	assert.Equal(t, storage_dto.TransformerEncryption, transformer.Type())
}

func TestCryptoTransformer_Transform_HappyPath(t *testing.T) {
	t.Parallel()

	plaintext := "hello, encrypted world"
	service := &stubCryptoService{
		encryptStreamFunc: func(_ context.Context, output io.Writer, _ string) (io.WriteCloser, error) {
			return &passthroughWriteCloser{writer: output}, nil
		},
	}

	transformer := New(service, "test-crypto", 250)
	reader, err := transformer.Transform(context.Background(), strings.NewReader(plaintext), nil)
	require.NoError(t, err)
	require.NotNil(t, reader)

	result, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, plaintext, string(result))
}

func TestCryptoTransformer_Transform_EncryptStreamError(t *testing.T) {
	t.Parallel()

	encryptionError := errors.New("key not found")
	service := &stubCryptoService{
		encryptStreamFunc: func(context.Context, io.Writer, string) (io.WriteCloser, error) {
			return nil, encryptionError
		},
	}

	transformer := New(service, "test-crypto", 250)
	reader, err := transformer.Transform(context.Background(), strings.NewReader("data"), nil)
	require.NoError(t, err)
	require.NotNil(t, reader)

	_, readErr := io.ReadAll(reader)
	require.Error(t, readErr)
	assert.Contains(t, readErr.Error(), "failed to create encrypting stream")
}

func TestCryptoTransformer_Transform_CopyError(t *testing.T) {
	t.Parallel()

	copyError := errors.New("read failure")
	failingReader := &errorReader{err: copyError}
	service := &stubCryptoService{
		encryptStreamFunc: func(_ context.Context, output io.Writer, _ string) (io.WriteCloser, error) {
			return &passthroughWriteCloser{writer: output}, nil
		},
	}

	transformer := New(service, "test-crypto", 250)
	reader, err := transformer.Transform(context.Background(), failingReader, nil)
	require.NoError(t, err)
	require.NotNil(t, reader)

	_, readErr := io.ReadAll(reader)
	require.Error(t, readErr)
	assert.Contains(t, readErr.Error(), "encryption copy failed")
}

func TestCryptoTransformer_Transform_CloseError(t *testing.T) {
	t.Parallel()

	closeError := errors.New("flush failed")
	service := &stubCryptoService{
		encryptStreamFunc: func(_ context.Context, output io.Writer, _ string) (io.WriteCloser, error) {
			return &failingWriteCloser{writer: output, closeErr: closeError}, nil
		},
	}

	transformer := New(service, "test-crypto", 250)
	reader, err := transformer.Transform(context.Background(), strings.NewReader("data"), nil)
	require.NoError(t, err)
	require.NotNil(t, reader)

	_, readErr := io.ReadAll(reader)
	require.Error(t, readErr)
	assert.Contains(t, readErr.Error(), "failed to close encrypting writer")
}

func TestCryptoTransformer_Transform_LargePayload(t *testing.T) {
	t.Parallel()

	largePayload := bytes.Repeat([]byte("A"), 256*1024)
	service := &stubCryptoService{
		encryptStreamFunc: func(_ context.Context, output io.Writer, _ string) (io.WriteCloser, error) {
			return &passthroughWriteCloser{writer: output}, nil
		},
	}

	transformer := New(service, "test-crypto", 250)
	reader, err := transformer.Transform(context.Background(), bytes.NewReader(largePayload), nil)
	require.NoError(t, err)

	result, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, largePayload, result)
}

func TestCryptoTransformer_Reverse_HappyPath(t *testing.T) {
	t.Parallel()

	ciphertext := "encrypted-data-here"
	service := &stubCryptoService{
		decryptStreamFunc: func(_ context.Context, input io.Reader) (io.ReadCloser, error) {
			return io.NopCloser(input), nil
		},
	}

	transformer := New(service, "test-crypto", 250)
	reader, err := transformer.Reverse(context.Background(), strings.NewReader(ciphertext), nil)
	require.NoError(t, err)
	require.NotNil(t, reader)

	result, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, ciphertext, string(result))
}

func TestCryptoTransformer_Reverse_DecryptStreamError(t *testing.T) {
	t.Parallel()

	decryptionError := errors.New("corrupted envelope")
	service := &stubCryptoService{
		decryptStreamFunc: func(context.Context, io.Reader) (io.ReadCloser, error) {
			return nil, decryptionError
		},
	}

	transformer := New(service, "test-crypto", 250)
	reader, err := transformer.Reverse(context.Background(), strings.NewReader("bad-data"), nil)
	require.Error(t, err)
	assert.Nil(t, reader)
	assert.Contains(t, err.Error(), "failed to create decrypting stream")
	assert.ErrorIs(t, err, decryptionError)
}

func TestCryptoTransformer_Reverse_LargePayload(t *testing.T) {
	t.Parallel()

	largePayload := bytes.Repeat([]byte("B"), 256*1024)
	service := &stubCryptoService{
		decryptStreamFunc: func(_ context.Context, input io.Reader) (io.ReadCloser, error) {
			return io.NopCloser(input), nil
		},
	}

	transformer := New(service, "test-crypto", 250)
	reader, err := transformer.Reverse(context.Background(), bytes.NewReader(largePayload), nil)
	require.NoError(t, err)

	result, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, largePayload, result)
}

func TestCryptoTransformer_TransformThenReverse_Roundtrip(t *testing.T) {
	t.Parallel()

	originalText := "roundtrip test data with some content"
	var encrypted bytes.Buffer

	service := &stubCryptoService{
		encryptStreamFunc: func(_ context.Context, output io.Writer, _ string) (io.WriteCloser, error) {
			return &passthroughWriteCloser{writer: output}, nil
		},
		decryptStreamFunc: func(_ context.Context, input io.Reader) (io.ReadCloser, error) {
			return io.NopCloser(input), nil
		},
	}

	transformer := New(service, "test-crypto", 250)

	encryptedReader, err := transformer.Transform(context.Background(), strings.NewReader(originalText), nil)
	require.NoError(t, err)
	_, err = io.Copy(&encrypted, encryptedReader)
	require.NoError(t, err)

	decryptedReader, err := transformer.Reverse(context.Background(), &encrypted, nil)
	require.NoError(t, err)

	result, err := io.ReadAll(decryptedReader)
	require.NoError(t, err)
	assert.Equal(t, originalText, string(result))
}

func TestNew_Config(t *testing.T) {
	t.Parallel()

	service := &stubCryptoService{}
	config := Config{
		CryptoService: service,
		Name:          "from-config",
		Priority:      42,
	}

	transformer := New(config.CryptoService, config.Name, config.Priority)
	require.NotNil(t, transformer)
	assert.Equal(t, "from-config", transformer.Name())
	assert.Equal(t, 42, transformer.Priority())
	assert.Equal(t, storage_dto.TransformerEncryption, transformer.Type())
}

func TestCryptoTransformer_Transform_EmptyInput(t *testing.T) {
	t.Parallel()

	service := &stubCryptoService{
		encryptStreamFunc: func(_ context.Context, output io.Writer, _ string) (io.WriteCloser, error) {
			return &passthroughWriteCloser{writer: output}, nil
		},
	}

	transformer := New(service, "test-crypto", 250)
	reader, err := transformer.Transform(context.Background(), strings.NewReader(""), nil)
	require.NoError(t, err)

	result, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestCryptoTransformer_Reverse_EmptyInput(t *testing.T) {
	t.Parallel()

	service := &stubCryptoService{
		decryptStreamFunc: func(_ context.Context, input io.Reader) (io.ReadCloser, error) {
			return io.NopCloser(input), nil
		},
	}

	transformer := New(service, "test-crypto", 250)
	reader, err := transformer.Reverse(context.Background(), strings.NewReader(""), nil)
	require.NoError(t, err)

	result, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Empty(t, result)
}

type errorReader struct {
	err error
}

func (r *errorReader) Read([]byte) (int, error) {
	return 0, r.err
}

var _ io.Reader = (*errorReader)(nil)
