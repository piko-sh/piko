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
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/sony/gobreaker/v2"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/crypto/crypto_dto"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/provider/provider_domain"
)

const (
	// defaultDirectModeConcurrency is the default number of parallel KMS calls
	// when direct KMS mode is used without envelope encryption.
	defaultDirectModeConcurrency = 50

	// circuitBreakerTimeout is how long the circuit stays open before letting a
	// test request through.
	circuitBreakerTimeout = 30 * time.Second

	// circuitBreakerBucketPeriod is the duration of each measurement bucket
	// for tracking failure counts.
	circuitBreakerBucketPeriod = 10 * time.Second

	// circuitBreakerConsecutiveFailures is the number of consecutive failures
	// required to trip the circuit breaker.
	circuitBreakerConsecutiveFailures = 5

	// safeCallCryptoType is the goroutine.SafeCallValue label for
	// provider type lookups.
	safeCallCryptoType = "crypto.Type"
)

// cryptoService is the concrete implementation of CryptoServicePort.
type cryptoService struct {
	// localProviderFactory creates local encryption providers for envelope
	// encryption.
	localProviderFactory LocalProviderFactory

	// dataKeyCache stores decrypted data keys for envelope encryption.
	dataKeyCache cache_domain.Cache[string, *crypto_dto.SecureBytes]

	// registry stores and looks up encryption providers by name.
	registry *provider_domain.StandardRegistry[EncryptionProvider]

	// breaker guards against failures in the encryption provider service.
	breaker *gobreaker.CircuitBreaker[any]

	// activeKeyID is the key identifier used for encryption operations.
	activeKeyID string

	// deprecatedKeyIDs holds the list of key IDs that have been rotated out
	// and should no longer be used for encryption.
	deprecatedKeyIDs []string

	// directModeMaxConcurrency limits parallel KMS calls when envelope encryption
	// is disabled.
	directModeMaxConcurrency int

	// mu guards localProviderFactory and dataKeyCache for thread-safe access.
	mu sync.RWMutex

	// enableAutoReEncrypt controls whether data encrypted with a deprecated key
	// is automatically re-encrypted with the current key.
	enableAutoReEncrypt bool

	// enableEnvelopeEncryption controls batch operation mode.
	//
	// When true: uses envelope encryption (data key in memory
	// briefly, fast).
	// When false: calls KMS directly for each item (no keys in
	// memory, slower).
	enableEnvelopeEncryption bool
}

// ServiceOption configures the crypto service.
type ServiceOption func(*cryptoService)

// Encrypt encrypts plaintext using the active key.
//
// Takes plaintext (string) which is the text to encrypt.
//
// Returns string which is the encrypted ciphertext.
// Returns error when encryption fails.
func (s *cryptoService) Encrypt(ctx context.Context, plaintext string) (string, error) {
	return s.EncryptWithKey(ctx, plaintext, s.activeKeyID)
}

// EncryptWithKey encrypts plaintext using a specific key.
//
// Takes plaintext (string) which is the data to encrypt.
// Takes keyID (string) which identifies the encryption key to use.
//
// Returns string which is the enveloped ciphertext.
// Returns error when encryption fails or the envelope cannot be created.
func (s *cryptoService) EncryptWithKey(ctx context.Context, plaintext string, keyID string) (string, error) {
	ctx, l := logger_domain.From(ctx, log)
	startTime := time.Now()

	provider, err := s.getProvider(ctx)
	if err != nil {
		return "", fmt.Errorf("getting encryption provider: %w", err)
	}

	request := &crypto_dto.EncryptRequest{
		Plaintext: plaintext,
		KeyID:     keyID,
		Context:   nil,
	}

	result, err := goroutine.SafeCall1(ctx, "crypto.Encrypt", func() (any, error) {
		return s.breaker.Execute(func() (any, error) {
			return provider.Encrypt(ctx, request)
		})
	})
	if err != nil {
		s.recordOperationError(ctx, provider, opEncrypt)
		provType := string(goroutine.SafeCallValue(ctx, safeCallCryptoType, func() crypto_dto.ProviderType { return provider.Type() }))
		return "", crypto_dto.NewEncryptionError("Encrypt", provType, keyID, err)
	}
	response, ok := result.(*crypto_dto.EncryptResponse)
	if !ok {
		s.recordOperationError(ctx, provider, opEncrypt)
		provType := string(goroutine.SafeCallValue(ctx, safeCallCryptoType, func() crypto_dto.ProviderType { return provider.Type() }))
		return "", crypto_dto.NewEncryptionError("Encrypt", provType, keyID, errors.New("unexpected response type from provider"))
	}

	envelopedCiphertext, err := createEnvelopedCiphertext(response.KeyID, string(response.Provider), response.Ciphertext, "")
	if err != nil {
		return "", fmt.Errorf("creating enveloped ciphertext: %w", err)
	}

	duration := time.Since(startTime).Milliseconds()
	s.recordOperationMetrics(ctx, provider, opEncrypt, statusSuccess, duration)
	l.Trace("Encryption completed",
		logger_domain.Int64(attributeKeyDurationMS, duration),
		logger_domain.String("key_id", keyID),
		logger_domain.String(attributeKeyProvider, string(goroutine.SafeCallValue(ctx, safeCallCryptoType, func() crypto_dto.ProviderType { return provider.Type() }))),
	)

	return envelopedCiphertext, nil
}

// Decrypt decrypts ciphertext, automatically detecting the key used.
//
// Takes ciphertext (string) which is the encrypted text to decrypt.
//
// Returns string which is the decrypted plaintext.
// Returns error when the ciphertext format is invalid or decryption fails.
func (s *cryptoService) Decrypt(ctx context.Context, ciphertext string) (string, error) {
	ctx, l := logger_domain.From(ctx, log)
	startTime := time.Now()

	provider, err := s.getProvider(ctx)
	if err != nil {
		return "", fmt.Errorf("getting decryption provider: %w", err)
	}

	metadata, err := extractCiphertextMetadata(ciphertext)
	if err != nil {
		s.recordOperationError(ctx, provider, opDecrypt)
		return "", fmt.Errorf("invalid ciphertext format: %w", err)
	}

	request := &crypto_dto.DecryptRequest{
		Ciphertext: metadata.Ciphertext,
		KeyID:      metadata.KeyID,
		Context:    nil,
	}

	result, err := goroutine.SafeCall1(ctx, "crypto.Decrypt", func() (any, error) {
		return s.breaker.Execute(func() (any, error) {
			return provider.Decrypt(ctx, request)
		})
	})
	if err != nil {
		s.recordOperationError(ctx, provider, opDecrypt)
		provType := string(goroutine.SafeCallValue(ctx, safeCallCryptoType, func() crypto_dto.ProviderType { return provider.Type() }))
		return "", crypto_dto.NewEncryptionError("Decrypt", provType, metadata.KeyID, err)
	}
	response, ok := result.(*crypto_dto.DecryptResponse)
	if !ok {
		s.recordOperationError(ctx, provider, opDecrypt)
		provType := string(goroutine.SafeCallValue(ctx, safeCallCryptoType, func() crypto_dto.ProviderType { return provider.Type() }))
		return "", crypto_dto.NewEncryptionError("Decrypt", provType, metadata.KeyID, errors.New("unexpected response type from provider"))
	}

	duration := time.Since(startTime).Milliseconds()
	s.recordOperationMetrics(ctx, provider, opDecrypt, statusSuccess, duration)
	l.Trace("Decryption completed",
		logger_domain.Int64(attributeKeyDurationMS, duration),
		logger_domain.String("key_id", metadata.KeyID),
		logger_domain.String(attributeKeyProvider, metadata.Provider),
	)

	return response.Plaintext, nil
}

// GetActiveKeyID returns the identifier of the active encryption key.
//
// Returns string which is the active key identifier.
// Returns error when the key identifier cannot be retrieved.
func (s *cryptoService) GetActiveKeyID(_ context.Context) (string, error) {
	return s.activeKeyID, nil
}

// NewEncrypt creates a new encryption builder.
//
// Returns *EncryptBuilder which provides a fluent interface for encrypting
// data.
func (s *cryptoService) NewEncrypt() *EncryptBuilder {
	return &EncryptBuilder{
		service: s,
	}
}

// NewDecrypt creates a new decryption builder.
//
// Returns *DecryptBuilder which provides a fluent interface for decrypting
// data.
func (s *cryptoService) NewDecrypt() *DecryptBuilder {
	return &DecryptBuilder{
		service: s,
	}
}

// NewBatchEncrypt creates a new batch encryption builder.
//
// Returns *BatchEncryptBuilder which provides a fluent interface for batch
// encryption.
func (s *cryptoService) NewBatchEncrypt() *BatchEncryptBuilder {
	return &BatchEncryptBuilder{
		service: s,
	}
}

// NewBatchDecrypt creates a new batch decryption builder.
//
// Returns *BatchDecryptBuilder which provides a fluent interface for batch
// decryption.
func (s *cryptoService) NewBatchDecrypt() *BatchDecryptBuilder {
	return &BatchDecryptBuilder{
		service: s,
	}
}

// NewStreamEncrypt creates a new streaming encryption builder.
//
// Returns *StreamEncryptBuilder which provides a fluent interface for streaming
// encryption.
func (s *cryptoService) NewStreamEncrypt() *StreamEncryptBuilder {
	return &StreamEncryptBuilder{
		service: s,
	}
}

// NewStreamDecrypt creates a new streaming decryption builder.
//
// Returns *StreamDecryptBuilder which provides a fluent interface for streaming
// decryption.
func (s *cryptoService) NewStreamDecrypt() *StreamDecryptBuilder {
	return &StreamDecryptBuilder{
		service: s,
	}
}

// WithLocalProviderFactory sets a factory for creating local providers used in
// envelope encryption.
//
// The factory creates providers for the local encryption part of envelope
// encryption:
//   - During EncryptBatch: creates a provider with the KMS data key and
//     encrypts each item.
//   - During DecryptBatch: creates a provider with the decrypted data key and
//     decrypts each item.
//
// If not set, the bootstrap layer injects a default local AES-GCM factory.
//
// This follows the Dependency Inversion Principle: the domain depends on the
// LocalProviderFactory interface, not on any specific implementation.
//
// Takes factory (LocalProviderFactory) which creates local providers for
// envelope encryption operations.
//
// Returns ServiceOption which sets the local provider factory.
func WithLocalProviderFactory(factory LocalProviderFactory) ServiceOption {
	return func(s *cryptoService) {
		s.localProviderFactory = factory
	}
}

// NewCryptoService creates a new crypto service with the given configuration.
//
// The cacheService parameter can be nil to disable data key caching (e.g., for
// testing or local providers).
//
// Takes ctx (context.Context) which carries cancellation and tracing.
// Takes cacheService (cache_domain.Service) which provides caching for data keys,
// or nil to disable caching.
// Takes config (*crypto_dto.ServiceConfig) which specifies encryption settings
// and cache parameters.
// Takes opts (...ServiceOption) which provides optional behaviour controls.
//
// Returns CryptoServicePort which is the configured crypto service ready for
// use.
// Returns error when the data key cache cannot be created.
func NewCryptoService(ctx context.Context, cacheService cache_domain.Service, config *crypto_dto.ServiceConfig, opts ...ServiceOption) (CryptoServicePort, error) {
	ctx, l := logger_domain.From(ctx, log)
	directModeConcurrency := config.DirectModeMaxConcurrency
	if directModeConcurrency <= 0 {
		directModeConcurrency = defaultDirectModeConcurrency
	}

	s := &cryptoService{
		registry:                 provider_domain.NewStandardRegistry[EncryptionProvider]("crypto"),
		localProviderFactory:     nil,
		dataKeyCache:             nil,
		breaker:                  newCryptoCircuitBreaker(),
		activeKeyID:              config.ActiveKeyID,
		deprecatedKeyIDs:         config.DeprecatedKeyIDs,
		enableAutoReEncrypt:      config.EnableAutoReEncrypt,
		enableEnvelopeEncryption: config.EnableEnvelopeEncryption,
		directModeMaxConcurrency: directModeConcurrency,
	}

	for _, opt := range opts {
		opt(s)
	}

	if s.enableEnvelopeEncryption && s.dataKeyCache == nil && cacheService != nil && config.DataKeyCacheTTL > 0 && config.DataKeyCacheMaxSize > 0 {
		dkCache, err := cache_domain.NewCacheBuilder[string, *crypto_dto.SecureBytes](cacheService).
			FactoryBlueprint("crypto-secure-bytes").
			Namespace("crypto-datakeys").
			MaximumSize(config.DataKeyCacheMaxSize).
			Expiration(config.DataKeyCacheTTL).
			OnDeletion(func(e cache_dto.DeletionEvent[string, *crypto_dto.SecureBytes]) {
				if e.Value != nil {
					_ = e.Value.Close()
				}
			}).
			Build(ctx)

		if err != nil {
			return nil, fmt.Errorf("failed to create data key cache: %w", err)
		}

		s.dataKeyCache = dkCache
		l.Internal("Data key cache enabled for crypto service (SecureBytes)",
			logger_domain.String("factory", "crypto-secure-bytes"),
			logger_domain.Int("max_size", config.DataKeyCacheMaxSize),
			logger_domain.Duration("ttl", config.DataKeyCacheTTL))
	}

	if s.enableEnvelopeEncryption {
		l.Internal("Crypto service using envelope encryption mode (data key briefly in memory)")
	} else {
		l.Notice("Crypto service using DIRECT KMS mode (no keys in memory, slower)",
			logger_domain.Int("max_concurrency", s.directModeMaxConcurrency))
	}

	return s, nil
}

// newCryptoCircuitBreaker creates a circuit breaker for crypto provider
// operations.
//
// Returns *gobreaker.CircuitBreaker[any] configured with standard settings
// for crypto operations.
func newCryptoCircuitBreaker() *gobreaker.CircuitBreaker[any] {
	settings := gobreaker.Settings{
		Name:         "crypto-service",
		MaxRequests:  1,
		Interval:     0,
		Timeout:      circuitBreakerTimeout,
		BucketPeriod: circuitBreakerBucketPeriod,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= circuitBreakerConsecutiveFailures
		},
		IsExcluded: func(err error) bool {
			return errors.Is(err, context.Canceled) ||
				errors.Is(err, context.DeadlineExceeded)
		},
	}
	return gobreaker.NewCircuitBreaker[any](settings)
}

// withDataKeyCache sets a pre-built data key cache for the service.
//
// Use it for testing with mock caches or specific cache setups. When set,
// the constructor will skip building a cache from the cache service. The
// cache stores SecureBytes (locked memory) to stop data keys from being swapped
// to disk or copied by the garbage collector.
//
// Takes cache (Cache[string, *SecureBytes]) which provides the data key cache.
//
// Returns ServiceOption which sets the service to use the given cache.
func withDataKeyCache(cache cache_domain.Cache[string, *crypto_dto.SecureBytes]) ServiceOption {
	return func(s *cryptoService) {
		s.dataKeyCache = cache
	}
}
