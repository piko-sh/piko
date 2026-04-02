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

package crypto_test

import (
	"context"
	"io"
	"sync/atomic"

	"piko.sh/piko/internal/crypto/crypto_domain"
	"piko.sh/piko/internal/provider/provider_domain"
)

// MockCryptoService is a test double for crypto_domain.CryptoServicePort.
// Nil function fields return zero values. Call counts are tracked atomically.
type MockCryptoService struct {
	EncryptFunc func(ctx context.Context, plaintext string) (string, error)

	DecryptFunc func(ctx context.Context, ciphertext string) (string, error)

	EncryptWithKeyFunc func(ctx context.Context, plaintext string, keyID string) (string, error)

	EncryptBatchFunc func(ctx context.Context, plaintexts []string) ([]string, error)

	DecryptBatchFunc func(ctx context.Context, ciphertexts []string) ([]string, error)

	RotateKeyFunc func(ctx context.Context, oldKeyID, newKeyID string) error

	GetActiveKeyIDFunc func(ctx context.Context) (string, error)

	DecryptAndReEncryptFunc func(ctx context.Context, ciphertext string) (string, string, bool, error)

	HealthCheckFunc func(ctx context.Context) error

	EncryptStreamFunc func(ctx context.Context, output io.Writer, keyID string) (io.WriteCloser, error)

	DecryptStreamFunc func(ctx context.Context, input io.Reader) (io.ReadCloser, error)

	NewEncryptFunc func() *crypto_domain.EncryptBuilder

	NewDecryptFunc func() *crypto_domain.DecryptBuilder

	NewBatchEncryptFunc func() *crypto_domain.BatchEncryptBuilder

	NewBatchDecryptFunc func() *crypto_domain.BatchDecryptBuilder

	NewStreamEncryptFunc func() *crypto_domain.StreamEncryptBuilder

	NewStreamDecryptFunc func() *crypto_domain.StreamDecryptBuilder

	RegisterProviderFunc func(ctx context.Context, name string, provider crypto_domain.EncryptionProvider) error

	SetDefaultProviderFunc func(name string) error

	GetProvidersFunc func(ctx context.Context) []string

	HasProviderFunc func(name string) bool

	ListProvidersFunc func(ctx context.Context) []provider_domain.ProviderInfo

	CloseFunc func(ctx context.Context) error

	EncryptCallCount int64

	DecryptCallCount int64

	EncryptWithKeyCallCount int64

	EncryptBatchCallCount int64

	DecryptBatchCallCount int64

	RotateKeyCallCount int64

	GetActiveKeyIDCallCount int64

	DecryptAndReEncryptCallCount int64

	HealthCheckCallCount int64

	EncryptStreamCallCount int64

	DecryptStreamCallCount int64

	NewEncryptCallCount int64

	NewDecryptCallCount int64

	NewBatchEncryptCallCount int64

	NewBatchDecryptCallCount int64

	NewStreamEncryptCallCount int64

	NewStreamDecryptCallCount int64

	RegisterProviderCallCount int64

	SetDefaultProviderCallCount int64

	GetProvidersCallCount int64

	HasProviderCallCount int64

	ListProvidersCallCount int64

	CloseCallCount int64
}

var _ crypto_domain.CryptoServicePort = (*MockCryptoService)(nil)

// Encrypt delegates to EncryptFunc if set.
//
// Returns zero values if EncryptFunc is nil.
func (m *MockCryptoService) Encrypt(ctx context.Context, plaintext string) (string, error) {
	atomic.AddInt64(&m.EncryptCallCount, 1)
	if m.EncryptFunc != nil {
		return m.EncryptFunc(ctx, plaintext)
	}
	return "", nil
}

// Decrypt delegates to DecryptFunc if set.
//
// Returns zero values if DecryptFunc is nil.
func (m *MockCryptoService) Decrypt(ctx context.Context, ciphertext string) (string, error) {
	atomic.AddInt64(&m.DecryptCallCount, 1)
	if m.DecryptFunc != nil {
		return m.DecryptFunc(ctx, ciphertext)
	}
	return "", nil
}

// EncryptWithKey delegates to EncryptWithKeyFunc if set.
//
// Returns zero values if EncryptWithKeyFunc is nil.
func (m *MockCryptoService) EncryptWithKey(ctx context.Context, plaintext string, keyID string) (string, error) {
	atomic.AddInt64(&m.EncryptWithKeyCallCount, 1)
	if m.EncryptWithKeyFunc != nil {
		return m.EncryptWithKeyFunc(ctx, plaintext, keyID)
	}
	return "", nil
}

// EncryptBatch delegates to EncryptBatchFunc if set.
//
// Returns zero values if EncryptBatchFunc is nil.
func (m *MockCryptoService) EncryptBatch(ctx context.Context, plaintexts []string) ([]string, error) {
	atomic.AddInt64(&m.EncryptBatchCallCount, 1)
	if m.EncryptBatchFunc != nil {
		return m.EncryptBatchFunc(ctx, plaintexts)
	}
	return nil, nil
}

// DecryptBatch delegates to DecryptBatchFunc if set.
//
// Returns zero values if DecryptBatchFunc is nil.
func (m *MockCryptoService) DecryptBatch(ctx context.Context, ciphertexts []string) ([]string, error) {
	atomic.AddInt64(&m.DecryptBatchCallCount, 1)
	if m.DecryptBatchFunc != nil {
		return m.DecryptBatchFunc(ctx, ciphertexts)
	}
	return nil, nil
}

// RotateKey delegates to RotateKeyFunc if set.
//
// Returns nil if RotateKeyFunc is nil.
func (m *MockCryptoService) RotateKey(ctx context.Context, oldKeyID, newKeyID string) error {
	atomic.AddInt64(&m.RotateKeyCallCount, 1)
	if m.RotateKeyFunc != nil {
		return m.RotateKeyFunc(ctx, oldKeyID, newKeyID)
	}
	return nil
}

// GetActiveKeyID delegates to GetActiveKeyIDFunc if set.
//
// Returns zero values if GetActiveKeyIDFunc is nil.
func (m *MockCryptoService) GetActiveKeyID(ctx context.Context) (string, error) {
	atomic.AddInt64(&m.GetActiveKeyIDCallCount, 1)
	if m.GetActiveKeyIDFunc != nil {
		return m.GetActiveKeyIDFunc(ctx)
	}
	return "", nil
}

// DecryptAndReEncrypt delegates to DecryptAndReEncryptFunc if set.
//
// Returns zero values if DecryptAndReEncryptFunc is nil.
func (m *MockCryptoService) DecryptAndReEncrypt(ctx context.Context, ciphertext string) (plaintext, newCiphertext string, wasReEncrypted bool, err error) {
	atomic.AddInt64(&m.DecryptAndReEncryptCallCount, 1)
	if m.DecryptAndReEncryptFunc != nil {
		return m.DecryptAndReEncryptFunc(ctx, ciphertext)
	}
	return "", "", false, nil
}

// HealthCheck delegates to HealthCheckFunc if set.
//
// Returns nil if HealthCheckFunc is nil.
func (m *MockCryptoService) HealthCheck(ctx context.Context) error {
	atomic.AddInt64(&m.HealthCheckCallCount, 1)
	if m.HealthCheckFunc != nil {
		return m.HealthCheckFunc(ctx)
	}
	return nil
}

// EncryptStream delegates to EncryptStreamFunc if set.
//
// Returns zero values if EncryptStreamFunc is nil.
func (m *MockCryptoService) EncryptStream(ctx context.Context, output io.Writer, keyID string) (io.WriteCloser, error) {
	atomic.AddInt64(&m.EncryptStreamCallCount, 1)
	if m.EncryptStreamFunc != nil {
		return m.EncryptStreamFunc(ctx, output, keyID)
	}
	return nil, nil
}

// DecryptStream delegates to DecryptStreamFunc if set.
//
// Returns zero values if DecryptStreamFunc is nil.
func (m *MockCryptoService) DecryptStream(ctx context.Context, input io.Reader) (io.ReadCloser, error) {
	atomic.AddInt64(&m.DecryptStreamCallCount, 1)
	if m.DecryptStreamFunc != nil {
		return m.DecryptStreamFunc(ctx, input)
	}
	return nil, nil
}

// NewEncrypt delegates to NewEncryptFunc if set.
//
// Returns nil if NewEncryptFunc is nil.
func (m *MockCryptoService) NewEncrypt() *crypto_domain.EncryptBuilder {
	atomic.AddInt64(&m.NewEncryptCallCount, 1)
	if m.NewEncryptFunc != nil {
		return m.NewEncryptFunc()
	}
	return nil
}

// NewDecrypt delegates to NewDecryptFunc if set.
//
// Returns nil if NewDecryptFunc is nil.
func (m *MockCryptoService) NewDecrypt() *crypto_domain.DecryptBuilder {
	atomic.AddInt64(&m.NewDecryptCallCount, 1)
	if m.NewDecryptFunc != nil {
		return m.NewDecryptFunc()
	}
	return nil
}

// NewBatchEncrypt delegates to NewBatchEncryptFunc if set.
//
// Returns nil if NewBatchEncryptFunc is nil.
func (m *MockCryptoService) NewBatchEncrypt() *crypto_domain.BatchEncryptBuilder {
	atomic.AddInt64(&m.NewBatchEncryptCallCount, 1)
	if m.NewBatchEncryptFunc != nil {
		return m.NewBatchEncryptFunc()
	}
	return nil
}

// NewBatchDecrypt delegates to NewBatchDecryptFunc if set.
//
// Returns nil if NewBatchDecryptFunc is nil.
func (m *MockCryptoService) NewBatchDecrypt() *crypto_domain.BatchDecryptBuilder {
	atomic.AddInt64(&m.NewBatchDecryptCallCount, 1)
	if m.NewBatchDecryptFunc != nil {
		return m.NewBatchDecryptFunc()
	}
	return nil
}

// NewStreamEncrypt delegates to NewStreamEncryptFunc if set.
//
// Returns nil if NewStreamEncryptFunc is nil.
func (m *MockCryptoService) NewStreamEncrypt() *crypto_domain.StreamEncryptBuilder {
	atomic.AddInt64(&m.NewStreamEncryptCallCount, 1)
	if m.NewStreamEncryptFunc != nil {
		return m.NewStreamEncryptFunc()
	}
	return nil
}

// NewStreamDecrypt delegates to NewStreamDecryptFunc if set.
//
// Returns nil if NewStreamDecryptFunc is nil.
func (m *MockCryptoService) NewStreamDecrypt() *crypto_domain.StreamDecryptBuilder {
	atomic.AddInt64(&m.NewStreamDecryptCallCount, 1)
	if m.NewStreamDecryptFunc != nil {
		return m.NewStreamDecryptFunc()
	}
	return nil
}

// RegisterProvider delegates to RegisterProviderFunc if set.
//
// Returns nil if RegisterProviderFunc is nil.
func (m *MockCryptoService) RegisterProvider(ctx context.Context, name string, provider crypto_domain.EncryptionProvider) error {
	atomic.AddInt64(&m.RegisterProviderCallCount, 1)
	if m.RegisterProviderFunc != nil {
		return m.RegisterProviderFunc(ctx, name, provider)
	}
	return nil
}

// SetDefaultProvider delegates to SetDefaultProviderFunc if set.
//
// Returns nil if SetDefaultProviderFunc is nil.
func (m *MockCryptoService) SetDefaultProvider(name string) error {
	atomic.AddInt64(&m.SetDefaultProviderCallCount, 1)
	if m.SetDefaultProviderFunc != nil {
		return m.SetDefaultProviderFunc(name)
	}
	return nil
}

// GetProviders delegates to GetProvidersFunc if set.
//
// Returns nil if GetProvidersFunc is nil.
func (m *MockCryptoService) GetProviders(ctx context.Context) []string {
	atomic.AddInt64(&m.GetProvidersCallCount, 1)
	if m.GetProvidersFunc != nil {
		return m.GetProvidersFunc(ctx)
	}
	return nil
}

// HasProvider delegates to HasProviderFunc if set.
//
// Returns false if HasProviderFunc is nil.
func (m *MockCryptoService) HasProvider(name string) bool {
	atomic.AddInt64(&m.HasProviderCallCount, 1)
	if m.HasProviderFunc != nil {
		return m.HasProviderFunc(name)
	}
	return false
}

// ListProviders delegates to ListProvidersFunc if set.
//
// Returns nil if ListProvidersFunc is nil.
func (m *MockCryptoService) ListProviders(ctx context.Context) []provider_domain.ProviderInfo {
	atomic.AddInt64(&m.ListProvidersCallCount, 1)
	if m.ListProvidersFunc != nil {
		return m.ListProvidersFunc(ctx)
	}
	return nil
}

// Close delegates to CloseFunc if set.
//
// Returns nil if CloseFunc is nil.
func (m *MockCryptoService) Close(ctx context.Context) error {
	atomic.AddInt64(&m.CloseCallCount, 1)
	if m.CloseFunc != nil {
		return m.CloseFunc(ctx)
	}
	return nil
}

// NewMockCryptoService creates a new mock crypto service for testing.
//
// Returns crypto_domain.CryptoServicePort which is the mock implementation.
func NewMockCryptoService() crypto_domain.CryptoServicePort {
	return &MockCryptoService{}
}
