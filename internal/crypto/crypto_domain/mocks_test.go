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
	"context"
	"encoding/base64"
	"errors"
	"io"
	"iter"
	"sync"
	"time"

	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/crypto/crypto_dto"
)

type mockEncryptionProvider struct {
	encryptFunc          func(ctx context.Context, request *crypto_dto.EncryptRequest) (*crypto_dto.EncryptResponse, error)
	decryptFunc          func(ctx context.Context, request *crypto_dto.DecryptRequest) (*crypto_dto.DecryptResponse, error)
	generateDataKey      func(ctx context.Context, request *crypto_dto.GenerateDataKeyRequest) (*crypto_dto.DataKey, error)
	getKeyInfoFunc       func(ctx context.Context, keyID string) (*crypto_dto.KeyInfo, error)
	healthCheckFunc      func(ctx context.Context) error
	encryptStreamFunc    func(ctx context.Context, output io.Writer, request *crypto_dto.EncryptRequest) (io.WriteCloser, error)
	decryptStreamFunc    func(ctx context.Context, input io.Reader) (io.ReadCloser, error)
	providerType         crypto_dto.ProviderType
	encryptCalls         []encryptCall
	decryptCalls         []decryptCall
	generateDataKeyCalls int
	mu                   sync.Mutex
}

type encryptCall struct {
	Plaintext string
	KeyID     string
}

type decryptCall struct {
	Ciphertext string
	KeyID      string
}

func newMockProvider() *mockEncryptionProvider {
	return &mockEncryptionProvider{
		providerType: crypto_dto.ProviderTypeLocalAESGCM,
	}
}

func (m *mockEncryptionProvider) Type() crypto_dto.ProviderType {
	return m.providerType
}

func (m *mockEncryptionProvider) Encrypt(ctx context.Context, request *crypto_dto.EncryptRequest) (*crypto_dto.EncryptResponse, error) {
	m.mu.Lock()
	m.encryptCalls = append(m.encryptCalls, encryptCall{Plaintext: request.Plaintext, KeyID: request.KeyID})
	m.mu.Unlock()

	if m.encryptFunc != nil {
		return m.encryptFunc(ctx, request)
	}

	return &crypto_dto.EncryptResponse{
		Ciphertext: base64.StdEncoding.EncodeToString([]byte(request.Plaintext)),
		KeyID:      request.KeyID,
		Provider:   m.providerType,
	}, nil
}

func (m *mockEncryptionProvider) Decrypt(ctx context.Context, request *crypto_dto.DecryptRequest) (*crypto_dto.DecryptResponse, error) {
	m.mu.Lock()
	m.decryptCalls = append(m.decryptCalls, decryptCall{Ciphertext: request.Ciphertext, KeyID: request.KeyID})
	m.mu.Unlock()

	if m.decryptFunc != nil {
		return m.decryptFunc(ctx, request)
	}

	decoded, err := base64.StdEncoding.DecodeString(request.Ciphertext)
	if err != nil {
		return nil, err
	}
	return &crypto_dto.DecryptResponse{
		Plaintext: string(decoded),
	}, nil
}

func (m *mockEncryptionProvider) GenerateDataKey(ctx context.Context, request *crypto_dto.GenerateDataKeyRequest) (*crypto_dto.DataKey, error) {
	m.mu.Lock()
	m.generateDataKeyCalls++
	m.mu.Unlock()

	if m.generateDataKey != nil {
		return m.generateDataKey(ctx, request)
	}

	plaintextKey := []byte("01234567890123456789012345678901")
	secureKey, err := crypto_dto.NewSecureBytesFromSlice(plaintextKey, crypto_dto.WithID("mock-datakey"))
	if err != nil {
		return nil, err
	}
	return &crypto_dto.DataKey{
		PlaintextKey: secureKey,
		EncryptedKey: base64.StdEncoding.EncodeToString(plaintextKey),
		KeyID:        request.KeyID,
		Provider:     m.providerType,
	}, nil
}

func (m *mockEncryptionProvider) encryptCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.encryptCalls)
}

func (m *mockEncryptionProvider) generateDataKeyCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.generateDataKeyCalls
}

func (m *mockEncryptionProvider) GetKeyInfo(ctx context.Context, keyID string) (*crypto_dto.KeyInfo, error) {
	if m.getKeyInfoFunc != nil {
		return m.getKeyInfoFunc(ctx, keyID)
	}
	return &crypto_dto.KeyInfo{
		KeyID:    keyID,
		Provider: m.providerType,
	}, nil
}

func (m *mockEncryptionProvider) HealthCheck(ctx context.Context) error {
	if m.healthCheckFunc != nil {
		return m.healthCheckFunc(ctx)
	}
	return nil
}

func (m *mockEncryptionProvider) EncryptStream(ctx context.Context, output io.Writer, request *crypto_dto.EncryptRequest) (io.WriteCloser, error) {
	if m.encryptStreamFunc != nil {
		return m.encryptStreamFunc(ctx, output, request)
	}
	return &mockWriteCloser{output: output}, nil
}

func (m *mockEncryptionProvider) DecryptStream(ctx context.Context, input io.Reader) (io.ReadCloser, error) {
	if m.decryptStreamFunc != nil {
		return m.decryptStreamFunc(ctx, input)
	}
	return &mockReadCloser{input: input}, nil
}

type mockLocalProviderFactory struct {
	createWithKeyFunc    func(key *crypto_dto.SecureBytes, keyID string) (EncryptionProvider, error)
	shouldReturnProvider *mockEncryptionProvider
	createdProviders     []*mockEncryptionProvider
	lastKeyUsed          []byte
	createWithKeyCalls   int
	mu                   sync.Mutex
}

func newMockLocalProviderFactory() *mockLocalProviderFactory {
	return &mockLocalProviderFactory{}
}

func (f *mockLocalProviderFactory) CreateWithKey(key *crypto_dto.SecureBytes, keyID string) (EncryptionProvider, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.createWithKeyCalls++
	_ = key.WithAccess(func(keyBytes []byte) error {
		f.lastKeyUsed = make([]byte, len(keyBytes))
		copy(f.lastKeyUsed, keyBytes)
		return nil
	})

	if f.createWithKeyFunc != nil {
		return f.createWithKeyFunc(key, keyID)
	}

	if f.shouldReturnProvider != nil {
		f.createdProviders = append(f.createdProviders, f.shouldReturnProvider)
		return f.shouldReturnProvider, nil
	}

	provider := newMockProvider()
	f.createdProviders = append(f.createdProviders, provider)
	return provider, nil
}

func (f *mockLocalProviderFactory) getCreateWithKeyCallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.createWithKeyCalls
}

type mockCache struct {
	getFunc           func(key string) (*crypto_dto.SecureBytes, bool, error)
	setFunc           func(key string, value *crypto_dto.SecureBytes) error
	data              map[string]*crypto_dto.SecureBytes
	mu                sync.RWMutex
	getIfPresentCalls int
	setCalls          int
}

func newMockCache() *mockCache {
	return &mockCache{
		data: make(map[string]*crypto_dto.SecureBytes),
	}
}

func (c *mockCache) GetIfPresent(_ context.Context, key string) (*crypto_dto.SecureBytes, bool, error) {
	c.mu.Lock()
	c.getIfPresentCalls++
	c.mu.Unlock()

	if c.getFunc != nil {
		return c.getFunc(key)
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	value, ok := c.data[key]
	if ok {
		clone, err := value.Clone()
		if err != nil {
			return nil, false, nil
		}
		return clone, true, nil
	}
	return nil, false, nil
}

func (c *mockCache) Set(_ context.Context, key string, value *crypto_dto.SecureBytes, _ ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.setCalls++

	if c.setFunc != nil {
		return c.setFunc(key, value)
	}

	clone, err := value.Clone()
	if err != nil {
		return nil
	}
	c.data[key] = clone
	return nil
}

func (c *mockCache) Get(_ context.Context, _ string, _ cache_dto.Loader[string, *crypto_dto.SecureBytes]) (*crypto_dto.SecureBytes, error) {
	return nil, errors.New("not implemented")
}

func (c *mockCache) SetWithTTL(_ context.Context, _ string, _ *crypto_dto.SecureBytes, _ time.Duration, _ ...string) error {
	return nil
}

func (c *mockCache) Invalidate(_ context.Context, _ string) error { return nil }

func (c *mockCache) InvalidateAll(_ context.Context) error { return nil }

func (c *mockCache) InvalidateByTags(_ context.Context, _ ...string) (int, error) { return 0, nil }

func (c *mockCache) Compute(_ context.Context, _ string, _ func(*crypto_dto.SecureBytes, bool) (*crypto_dto.SecureBytes, cache_dto.ComputeAction)) (*crypto_dto.SecureBytes, bool, error) {
	return nil, false, nil
}

func (c *mockCache) ComputeIfAbsent(_ context.Context, _ string, _ func() *crypto_dto.SecureBytes) (*crypto_dto.SecureBytes, bool, error) {
	return nil, false, nil
}

func (c *mockCache) ComputeIfPresent(_ context.Context, _ string, _ func(*crypto_dto.SecureBytes) (*crypto_dto.SecureBytes, cache_dto.ComputeAction)) (*crypto_dto.SecureBytes, bool, error) {
	return nil, false, nil
}

func (c *mockCache) ComputeWithTTL(_ context.Context, _ string, _ func(*crypto_dto.SecureBytes, bool) cache_dto.ComputeResult[*crypto_dto.SecureBytes]) (*crypto_dto.SecureBytes, bool, error) {
	return nil, false, nil
}

func (c *mockCache) BulkGet(_ context.Context, _ []string, _ cache_dto.BulkLoader[string, *crypto_dto.SecureBytes]) (map[string]*crypto_dto.SecureBytes, error) {
	return nil, errors.New("not implemented")
}

func (c *mockCache) BulkSet(_ context.Context, _ map[string]*crypto_dto.SecureBytes, _ ...string) error {
	return nil
}

func (c *mockCache) BulkRefresh(_ context.Context, _ []string, _ cache_dto.BulkLoader[string, *crypto_dto.SecureBytes]) {
}

func (c *mockCache) Refresh(_ context.Context, _ string, _ cache_dto.Loader[string, *crypto_dto.SecureBytes]) <-chan cache_dto.LoadResult[*crypto_dto.SecureBytes] {
	return nil
}

func (c *mockCache) All() iter.Seq2[string, *crypto_dto.SecureBytes] {
	return func(yield func(string, *crypto_dto.SecureBytes) bool) {
		c.mu.RLock()
		defer c.mu.RUnlock()
		for k, v := range c.data {
			if !yield(k, v) {
				return
			}
		}
	}
}

func (c *mockCache) Keys() iter.Seq[string] {
	return func(yield func(string) bool) {
		c.mu.RLock()
		defer c.mu.RUnlock()
		for k := range c.data {
			if !yield(k) {
				return
			}
		}
	}
}

func (c *mockCache) Values() iter.Seq[*crypto_dto.SecureBytes] {
	return func(yield func(*crypto_dto.SecureBytes) bool) {
		c.mu.RLock()
		defer c.mu.RUnlock()
		for _, v := range c.data {
			if !yield(v) {
				return
			}
		}
	}
}

func (c *mockCache) GetEntry(_ context.Context, _ string) (cache_dto.Entry[string, *crypto_dto.SecureBytes], bool, error) {
	return cache_dto.Entry[string, *crypto_dto.SecureBytes]{}, false, nil
}

func (c *mockCache) ProbeEntry(_ context.Context, _ string) (cache_dto.Entry[string, *crypto_dto.SecureBytes], bool, error) {
	return cache_dto.Entry[string, *crypto_dto.SecureBytes]{}, false, nil
}

func (c *mockCache) EstimatedSize() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.data)
}

func (c *mockCache) Stats() cache_dto.Stats { return cache_dto.Stats{} }

func (c *mockCache) Close(_ context.Context) error { return nil }

func (c *mockCache) GetMaximum() uint64 { return 0 }

func (c *mockCache) SetMaximum(_ uint64) {}

func (c *mockCache) WeightedSize() uint64 { return 0 }

func (c *mockCache) SetExpiresAfter(_ context.Context, _ string, _ time.Duration) error { return nil }

func (c *mockCache) SetRefreshableAfter(_ context.Context, _ string, _ time.Duration) error {
	return nil
}

func (c *mockCache) Search(_ context.Context, _ string, _ *cache_dto.SearchOptions) (cache_dto.SearchResult[string, *crypto_dto.SecureBytes], error) {
	return cache_dto.SearchResult[string, *crypto_dto.SecureBytes]{}, nil
}

func (c *mockCache) Query(_ context.Context, _ *cache_dto.QueryOptions) (cache_dto.SearchResult[string, *crypto_dto.SecureBytes], error) {
	return cache_dto.SearchResult[string, *crypto_dto.SecureBytes]{}, nil
}

func (c *mockCache) SupportsSearch() bool { return false }

func (c *mockCache) GetSchema() *cache_dto.SearchSchema { return nil }

func (c *mockCache) getGetIfPresentCallCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.getIfPresentCalls
}

func (c *mockCache) getSetCallCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.setCalls
}

type mockWriteCloser struct {
	output io.Writer
	closed bool
}

func (m *mockWriteCloser) Write(p []byte) (n int, err error) {
	if m.closed {
		return 0, errors.New("writer closed")
	}
	return m.output.Write(p)
}

func (m *mockWriteCloser) Close() error {
	m.closed = true
	return nil
}

type mockReadCloser struct {
	input  io.Reader
	closed bool
}

func (m *mockReadCloser) Read(p []byte) (n int, err error) {
	if m.closed {
		return 0, errors.New("reader closed")
	}
	return m.input.Read(p)
}

func (m *mockReadCloser) Close() error {
	m.closed = true
	return nil
}

func createTestConfig(activeKeyID string) *crypto_dto.ServiceConfig {
	return &crypto_dto.ServiceConfig{
		ActiveKeyID:              activeKeyID,
		ProviderType:             crypto_dto.ProviderTypeLocalAESGCM,
		DeprecatedKeyIDs:         []string{},
		EnableAutoReEncrypt:      false,
		EnableEnvelopeEncryption: true,
		DirectModeMaxConcurrency: 10,
		DataKeyCacheTTL:          0,
		DataKeyCacheMaxSize:      0,
	}
}

func createDirectModeTestConfig(activeKeyID string) *crypto_dto.ServiceConfig {
	return &crypto_dto.ServiceConfig{
		ActiveKeyID:              activeKeyID,
		ProviderType:             crypto_dto.ProviderTypeLocalAESGCM,
		DeprecatedKeyIDs:         []string{},
		EnableAutoReEncrypt:      false,
		EnableEnvelopeEncryption: false,
		DirectModeMaxConcurrency: 5,
		DataKeyCacheTTL:          0,
		DataKeyCacheMaxSize:      0,
	}
}

func createTestService(provider *mockEncryptionProvider, config *crypto_dto.ServiceConfig, opts ...ServiceOption) (CryptoServicePort, error) {
	service, err := NewCryptoService(context.Background(), nil, config, opts...)
	if err != nil {
		return nil, err
	}

	if err := service.RegisterProvider(context.Background(), "test", provider); err != nil {
		return nil, err
	}

	if err := service.SetDefaultProvider("test"); err != nil {
		return nil, err
	}

	return service, nil
}

func createTestEnvelope(keyID, provider, ciphertext, encryptedDataKey string) string {
	envelope, _ := createEnvelopedCiphertext(keyID, provider, ciphertext, encryptedDataKey)
	return envelope
}
