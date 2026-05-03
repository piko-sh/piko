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
	"fmt"
	"sync"
	"time"

	"piko.sh/piko/internal/crypto/crypto_dto"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
)

// EncryptBatch encrypts multiple values using either envelope encryption or
// direct KMS mode.
//
// When EnableEnvelopeEncryption is true (default):
//   - 1 KMS call (GenerateDataKey) + fast local encryption
//   - Data key is briefly in memory (in SecureBytes with mlock)
//   - Best for: High throughput, cost-sensitive scenarios
//
// When EnableEnvelopeEncryption is false:
//   - N parallel KMS calls (one per item, limited by DirectModeMaxConcurrency)
//   - Encryption keys NEVER enter application memory
//   - Best for: Maximum security, compliance-heavy scenarios (HIPAA, PCI-DSS,
//     SOC2)
//
// Takes plaintexts ([]string) which contains the values to encrypt.
//
// Returns []string which contains the encrypted ciphertexts in the same order.
// Returns error when encryption fails or KMS is unavailable.
func (s *cryptoService) EncryptBatch(ctx context.Context, plaintexts []string) ([]string, error) {
	if len(plaintexts) == 0 {
		return []string{}, nil
	}

	if !s.enableEnvelopeEncryption {
		return s.encryptBatchDirect(ctx, plaintexts)
	}

	return s.encryptBatchEnvelope(ctx, plaintexts)
}

// encryptBatchEnvelope encrypts multiple items using envelope encryption with
// a single data key.
//
// Takes plaintexts ([]string) which contains the items to encrypt.
//
// Returns []string which contains the encrypted ciphertexts.
// Returns error when encryption setup fails or any item fails to encrypt.
func (s *cryptoService) encryptBatchEnvelope(ctx context.Context, plaintexts []string) ([]string, error) {
	ctx, l := logger_domain.From(ctx, log)
	startTime := time.Now()

	provider, err := s.getProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting provider for envelope batch encryption: %w", err)
	}

	ephemeralProvider, dataKey, err := s.setupBatchEncryption(ctx)
	if err != nil {
		s.recordOperationError(ctx, provider, opEncryptBatch)
		return nil, fmt.Errorf("setting up batch encryption: %w", err)
	}
	defer func() { _ = dataKey.Close() }()

	ciphertexts, encryptionErrors := s.encryptBatchItems(ctx, ephemeralProvider, plaintexts, dataKey)
	if len(encryptionErrors) > 0 {
		s.recordBatchMetrics(ctx, provider, opEncryptBatch, statusPartialFailure, len(plaintexts))
		return ciphertexts, fmt.Errorf("batch encryption had %d failures: %w", len(encryptionErrors), encryptionErrors[0])
	}

	duration := time.Since(startTime).Milliseconds()
	l.Trace("Batch encryption completed",
		logger_domain.Int(attributeKeyCount, len(plaintexts)),
		logger_domain.Int64(attributeKeyDurationMS, duration),
		logger_domain.String(attributeKeyProvider, string(provider.Type())),
		logger_domain.String("method", "envelope_encryption"),
	)
	s.recordBatchMetrics(ctx, provider, opEncryptBatch, statusSuccess, len(plaintexts))

	return ciphertexts, nil
}

// encryptBatchDirect calls KMS directly for each item without keys in memory.
//
// Takes plaintexts ([]string) which contains the data to encrypt.
//
// Returns []string which contains the encrypted ciphertext values.
// Returns error when the provider cannot be retrieved or encryption fails.
func (s *cryptoService) encryptBatchDirect(ctx context.Context, plaintexts []string) ([]string, error) {
	provider, err := s.getProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting provider for direct batch encryption: %w", err)
	}
	return s.runBatchDirect(ctx, plaintexts, directBatchConfig{
		operation:    s.Encrypt,
		opMetric:     opEncryptBatch,
		opName:       "encryption",
		providerType: provider.Type(),
	})
}

// directBatchConfig holds settings for running batch operations directly.
type directBatchConfig struct {
	// operation is the function to run for each input in the batch.
	operation func(context.Context, string) (string, error)

	// opMetric is the metric name used when recording batch operation results.
	opMetric string

	// opName identifies the batch operation for error messages and logs.
	opName string

	// providerType identifies the encryption provider for logging.
	providerType crypto_dto.ProviderType
}

// runBatchDirect executes a batch operation directly via KMS with parallel
// goroutines.
//
// Takes inputs ([]string) which contains the data to process.
// Takes config (directBatchConfig) which specifies the batch operation settings.
//
// Returns []string which contains the processed outputs for each input.
// Returns error when any operation in the batch fails.
func (s *cryptoService) runBatchDirect(ctx context.Context, inputs []string, config directBatchConfig) ([]string, error) {
	ctx, l := logger_domain.From(ctx, log)
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("direct batch %s context cancelled before execution: %w", config.opName, err)
	}

	startTime := time.Now()

	provider, err := s.getProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting provider for direct batch %s: %w", config.opName, err)
	}

	outputs := make([]string, len(inputs))
	errorResults := make([]error, len(inputs))

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, s.directModeMaxConcurrency)

	for i, input := range inputs {
		wg.Go(func() {
			output, err := s.runWithSemaphore(ctx, input, semaphore, config.operation)
			if err != nil {
				errorResults[i] = err
				return
			}
			outputs[i] = output
		})
	}

	wg.Wait()

	errorCount, firstErr := countErrors(errorResults)
	duration := time.Since(startTime).Milliseconds()

	if errorCount > 0 {
		s.recordBatchMetrics(ctx, provider, config.opMetric, statusPartialFailure, len(inputs))
		return outputs, fmt.Errorf("direct batch %s had %d failures: %w", config.opName, errorCount, firstErr)
	}

	l.Trace("Direct batch "+config.opName+" completed",
		logger_domain.Int(attributeKeyCount, len(inputs)),
		logger_domain.Int64(attributeKeyDurationMS, duration),
		logger_domain.String(attributeKeyProvider, string(config.providerType)),
		logger_domain.String("method", "direct_kms"),
		logger_domain.Int("max_concurrency", s.directModeMaxConcurrency),
	)
	s.recordBatchMetrics(ctx, provider, config.opMetric, statusSuccess, len(inputs))

	return outputs, nil
}

// runWithSemaphore acquires a semaphore slot and runs the operation.
//
// Takes input (string) which is the value to pass to the operation.
// Takes semaphore (chan struct{}) which limits concurrent operations.
// Takes op (func) which performs the actual work with the input.
//
// Returns string which is the result from the operation.
// Returns error when the context is cancelled or the operation fails.
func (*cryptoService) runWithSemaphore(ctx context.Context, input string, semaphore chan struct{}, op func(context.Context, string) (string, error)) (string, error) {
	select {
	case semaphore <- struct{}{}:
		defer func() { <-semaphore }()
	case <-ctx.Done():
		return "", ctx.Err()
	}
	return op(ctx, input)
}

// setupBatchEncryption generates a data key and creates an ephemeral provider
// for batch encryption.
//
// Returns EncryptionProvider which is the ephemeral provider for encrypting
// batch items.
// Returns *crypto_dto.DataKey which contains the encrypted data key for
// storage.
// Returns error when key generation fails or the local provider factory is
// not configured.
//
// Safe for concurrent use. Uses a read lock to access the local provider
// factory.
func (s *cryptoService) setupBatchEncryption(ctx context.Context) (EncryptionProvider, *crypto_dto.DataKey, error) {
	provider, err := s.getProvider(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("getting provider for batch encryption setup: %w", err)
	}

	result, err := s.breaker.Execute(func() (any, error) {
		return provider.GenerateDataKey(ctx, &crypto_dto.GenerateDataKeyRequest{
			KeyID:   s.activeKeyID,
			KeySpec: "AES_256",
		})
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate data key for batch: %w", err)
	}
	dataKey, ok := result.(*crypto_dto.DataKey)
	if !ok {
		return nil, nil, errors.New("unexpected response type from GenerateDataKey")
	}

	s.mu.RLock()
	factory := s.localProviderFactory
	s.mu.RUnlock()

	if factory == nil {
		_ = dataKey.Close()
		return nil, nil, errors.New("local provider factory is required for envelope encryption")
	}

	ephemeralProvider, err := factory.CreateWithKey(dataKey.PlaintextKey, "ephemeral-data-key")
	if err != nil {
		_ = dataKey.Close()
		return nil, nil, fmt.Errorf("failed to create ephemeral encryption provider: %w", err)
	}

	return ephemeralProvider, dataKey, nil
}

// encryptBatchItems encrypts each plaintext using the ephemeral provider.
//
// Takes provider (EncryptionProvider) which performs the encryption.
// Takes plaintexts ([]string) which contains the values to encrypt.
// Takes dataKey (*crypto_dto.DataKey) which provides the encryption key.
//
// Returns []string which contains the encrypted ciphertexts.
// Returns []error which contains any errors encountered during encryption.
func (*cryptoService) encryptBatchItems(ctx context.Context, provider EncryptionProvider, plaintexts []string, dataKey *crypto_dto.DataKey) ([]string, []error) {
	ciphertexts := make([]string, len(plaintexts))
	var encryptionErrors []error

	for i, plaintext := range plaintexts {
		ciphertext, err := encryptBatchItem(ctx, provider, plaintext, dataKey)
		if err != nil {
			encryptionErrors = append(encryptionErrors, fmt.Errorf("batch encryption failed at index %d: %w", i, err))
			continue
		}
		ciphertexts[i] = ciphertext
	}

	return ciphertexts, encryptionErrors
}

// DecryptBatch decrypts multiple values using either envelope decryption or
// direct KMS mode.
//
// When EnableEnvelopeEncryption is true (default):
//   - Detects envelope format and uses single data key decryption
//   - Falls back to direct decryption for non-envelope ciphertexts
//
// When EnableEnvelopeEncryption is false:
//   - Uses parallel direct KMS decryption for all items
//   - No keys enter application memory
//
// Takes ciphertexts ([]string) which contains the encrypted values to decrypt.
//
// Returns []string which contains the decrypted plaintext values.
// Returns error when decryption fails for any value.
func (s *cryptoService) DecryptBatch(ctx context.Context, ciphertexts []string) ([]string, error) {
	if len(ciphertexts) == 0 {
		return []string{}, nil
	}

	if !s.enableEnvelopeEncryption {
		return s.decryptBatchDirect(ctx, ciphertexts)
	}

	return s.decryptBatchEnvelope(ctx, ciphertexts)
}

// decryptBatchEnvelope tries envelope decryption first, then falls back to
// direct decryption for non-envelope ciphertexts.
//
// Takes ciphertexts ([]string) which contains the encrypted data to decrypt.
//
// Returns []string which contains the decrypted plain text values.
// Returns error when envelope setup fails or batch decryption has failures.
func (s *cryptoService) decryptBatchEnvelope(ctx context.Context, ciphertexts []string) ([]string, error) {
	ctx, l := logger_domain.From(ctx, log)
	startTime := time.Now()

	provider, err := s.getProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting provider for envelope batch decryption: %w", err)
	}

	ephemeralProvider, cleanup, err := s.setupEnvelopeDecryption(ctx, ciphertexts[0])
	if err != nil {
		s.recordOperationError(ctx, provider, opDecryptBatch)
		return nil, fmt.Errorf("setting up envelope decryption: %w", err)
	}
	if cleanup != nil {
		defer cleanup()
	}

	plaintexts, decryptionErrors := s.decryptBatchItems(ctx, ephemeralProvider, ciphertexts)
	if len(decryptionErrors) > 0 {
		s.recordBatchMetrics(ctx, provider, opDecryptBatch, statusPartialFailure, len(ciphertexts))
		return plaintexts, fmt.Errorf("batch decryption had %d failures: %w", len(decryptionErrors), decryptionErrors[0])
	}

	duration := time.Since(startTime).Milliseconds()
	l.Trace("Batch decryption completed",
		logger_domain.Int(attributeKeyCount, len(ciphertexts)),
		logger_domain.Int64(attributeKeyDurationMS, duration),
	)
	s.recordBatchMetrics(ctx, provider, opDecryptBatch, statusSuccess, len(ciphertexts))

	return plaintexts, nil
}

// decryptBatchDirect calls KMS directly for each item without storing keys
// in memory. Uses parallel goroutines with a semaphore to limit how many run
// at once.
//
// Takes ciphertexts ([]string) which contains the encrypted values to decrypt.
//
// Returns []string which contains the decrypted plaintext values.
// Returns error when any decryption operation fails.
func (s *cryptoService) decryptBatchDirect(ctx context.Context, ciphertexts []string) ([]string, error) {
	provider, err := s.getProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting provider for direct batch decryption: %w", err)
	}
	return s.runBatchDirect(ctx, ciphertexts, directBatchConfig{
		operation:    s.Decrypt,
		opMetric:     opDecryptBatch,
		opName:       "decryption",
		providerType: provider.Type(),
	})
}

// setupEnvelopeDecryption prepares the ephemeral provider for batch
// decryption. The provider is nil when the ciphertexts do not use
// envelope encryption.
//
// Takes firstCiphertext (string) which is the first ciphertext in
// the batch, used to detect envelope format and extract the
// encrypted data key.
//
// Returns EncryptionProvider which is the ephemeral provider for
// decrypting envelope-encrypted items, or nil when the ciphertexts
// do not use envelope encryption.
// Returns func() which is a cleanup callback that releases the data key
// resources; always non-nil on success.
// Returns error when data key decryption or provider creation fails.
func (s *cryptoService) setupEnvelopeDecryption(ctx context.Context, firstCiphertext string) (EncryptionProvider, func(), error) {
	metadata, err := extractCiphertextMetadata(firstCiphertext)
	if err != nil || metadata.EncryptedDataKey == "" {
		return nil, func() {}, nil
	}

	secureDataKey, err := s.decryptDataKeyWithCache(ctx, metadata.EncryptedDataKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decrypt data key: %w", err)
	}

	if s.localProviderFactory == nil {
		_ = secureDataKey.Close()
		return nil, nil, errors.New("local provider factory is required for envelope decryption")
	}

	ephemeralProvider, err := s.localProviderFactory.CreateWithKey(secureDataKey, "ephemeral-data-key")
	if err != nil {
		_ = secureDataKey.Close()
		return nil, nil, fmt.Errorf("failed to create ephemeral provider: %w", err)
	}

	cleanup := func() { _ = secureDataKey.Close() }
	return ephemeralProvider, cleanup, nil
}

// decryptBatchItems decrypts each ciphertext using either envelope or
// standard decryption.
//
// Takes ephemeralProvider (EncryptionProvider) which provides the key for
// envelope-encrypted items.
// Takes ciphertexts ([]string) which contains the encrypted values to decrypt.
//
// Returns []string which contains the decrypted plaintext values in the same
// order as the input.
// Returns []error which contains any decryption errors with their indices.
func (s *cryptoService) decryptBatchItems(ctx context.Context, ephemeralProvider EncryptionProvider, ciphertexts []string) ([]string, []error) {
	plaintexts := make([]string, len(ciphertexts))
	var decryptionErrors []error

	for i, ciphertext := range ciphertexts {
		plaintext, err := s.decryptSingleBatchItem(ctx, ephemeralProvider, ciphertext)
		if err != nil {
			decryptionErrors = append(decryptionErrors, fmt.Errorf("batch decryption failed at index %d: %w", i, err))
			continue
		}
		plaintexts[i] = plaintext
	}

	return plaintexts, decryptionErrors
}

// decryptSingleBatchItem decrypts a single item using envelope or standard
// decryption.
//
// Takes ephemeralProvider (EncryptionProvider) which handles envelope
// decryption when set. Falls back to standard decryption when nil.
// Takes ciphertext (string) which is the encrypted data to decrypt.
//
// Returns string which is the decrypted plaintext.
// Returns error when decryption fails.
func (s *cryptoService) decryptSingleBatchItem(ctx context.Context, ephemeralProvider EncryptionProvider, ciphertext string) (string, error) {
	if ephemeralProvider != nil {
		return decryptBatchItemWithEnvelope(ctx, ephemeralProvider, ciphertext)
	}
	return s.Decrypt(ctx, ciphertext)
}

// decryptDataKeyWithCache decrypts an encrypted data key using the provider,
// with caching to reduce KMS API calls for frequently-used keys.
//
// Takes encryptedKey (string) which is the encrypted data key to decrypt.
//
// Returns *crypto_dto.SecureBytes which is a clone with independent lifecycle.
// Returns error when decryption fails or secure bytes cannot be created.
//
// Safe for concurrent use. Uses a read lock when accessing the cache.
//
// Critical for performance in high-throughput scenarios where many small
// operations share the same encrypted data key.
//
// Security considerations:
//   - Cache stores SecureBytes (locked memory, prevents swap)
//   - Short TTL (default 5 minutes) to limit exposure window
//   - Returns a clone to ensure caller has independent lifecycle
//   - Caller is responsible for calling Close() on the returned SecureBytes
func (s *cryptoService) decryptDataKeyWithCache(ctx context.Context, encryptedKey string) (*crypto_dto.SecureBytes, error) {
	s.mu.RLock()
	cache := s.dataKeyCache
	s.mu.RUnlock()

	if cache != nil {
		if cachedKey, found, _ := cache.GetIfPresent(ctx, encryptedKey); found {
			return cachedKey.Clone()
		}
	}

	provider, err := s.getProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting provider for data key decryption: %w", err)
	}

	result, err := goroutine.SafeCall1(ctx, "crypto.Decrypt", func() (any, error) {
		return s.breaker.Execute(func() (any, error) {
			return provider.Decrypt(ctx, &crypto_dto.DecryptRequest{
				Ciphertext: encryptedKey,
				KeyID:      "",
				Context:    nil,
			})
		})
	})
	if err != nil {
		return nil, fmt.Errorf("decrypting data key via circuit breaker: %w", err)
	}
	decryptResp, ok := result.(*crypto_dto.DecryptResponse)
	if !ok {
		return nil, errors.New("unexpected response type from Decrypt")
	}

	plaintextKey, err := base64.StdEncoding.DecodeString(decryptResp.Plaintext)
	if err != nil {
		plaintextKey = []byte(decryptResp.Plaintext)
	}

	secureKey, err := crypto_dto.NewSecureBytesFromSlice(plaintextKey, crypto_dto.WithID("data-key-"+encryptedKey[:8]))
	if err != nil {
		return nil, fmt.Errorf("failed to create secure bytes for data key: %w", err)
	}

	zeroBytes(plaintextKey)

	if s.dataKeyCache != nil {
		cacheKey, cloneErr := secureKey.Clone()
		if cloneErr == nil {
			_ = s.dataKeyCache.Set(ctx, encryptedKey, cacheKey)
		}
	}

	return secureKey, nil
}

// encryptBatchItem encrypts a single item using the given provider and wraps
// it in an envelope.
//
// Takes provider (EncryptionProvider) which handles the encryption.
// Takes plaintext (string) which is the data to encrypt.
// Takes dataKey (*crypto_dto.DataKey) which contains the key details.
//
// Returns string which is the enveloped ciphertext.
// Returns error when encryption fails.
func encryptBatchItem(ctx context.Context, provider EncryptionProvider, plaintext string, dataKey *crypto_dto.DataKey) (string, error) {
	encryptResp, err := goroutine.SafeCall1(ctx, "crypto.Encrypt", func() (*crypto_dto.EncryptResponse, error) {
		return provider.Encrypt(ctx, &crypto_dto.EncryptRequest{
			Plaintext: plaintext,
			KeyID:     "ephemeral-data-key",
			Context:   nil,
		})
	})
	if err != nil {
		return "", fmt.Errorf("encrypting batch item: %w", err)
	}

	return createEnvelopedCiphertext(dataKey.KeyID, string(dataKey.Provider), encryptResp.Ciphertext, dataKey.EncryptedKey)
}

// countErrors counts the errors in a slice and returns the total with the
// first error found.
//
// Takes errs ([]error) which contains the errors to count.
//
// Returns int which is the number of non-nil errors in the slice.
// Returns error which is the first non-nil error found, or nil if there are
// none.
func countErrors(errs []error) (int, error) {
	var firstErr error
	errorCount := 0
	for _, err := range errs {
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			errorCount++
		}
	}
	return errorCount, firstErr
}

// decryptBatchItemWithEnvelope decrypts a single item using envelope decryption.
//
// Takes provider (EncryptionProvider) which handles the decryption.
// Takes ciphertext (string) which is the encrypted data to decrypt.
//
// Returns string which is the decrypted plaintext.
// Returns error when metadata extraction or decryption fails.
func decryptBatchItemWithEnvelope(ctx context.Context, provider EncryptionProvider, ciphertext string) (string, error) {
	metadata, err := extractCiphertextMetadata(ciphertext)
	if err != nil {
		return "", fmt.Errorf("extracting ciphertext metadata for envelope decryption: %w", err)
	}

	decryptResp, err := goroutine.SafeCall1(ctx, "crypto.Decrypt", func() (*crypto_dto.DecryptResponse, error) {
		return provider.Decrypt(ctx, &crypto_dto.DecryptRequest{
			Ciphertext: metadata.Ciphertext,
			KeyID:      "",
			Context:    nil,
		})
	})
	if err != nil {
		return "", fmt.Errorf("decrypting batch item with envelope: %w", err)
	}

	return decryptResp.Plaintext, nil
}
