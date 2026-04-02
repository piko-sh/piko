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

package crypto

import (
	"context"
	"fmt"

	"piko.sh/piko/internal/bootstrap"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/crypto/crypto_domain"
	"piko.sh/piko/internal/crypto/crypto_dto"
)

const (
	// ProviderTypeLocalAESGCM is the provider type for local AES-GCM encryption.
	ProviderTypeLocalAESGCM = crypto_dto.ProviderTypeLocalAESGCM

	// ProviderTypeAWSKMS is the provider type for AWS Key Management Service.
	ProviderTypeAWSKMS = crypto_dto.ProviderTypeAWSKMS

	// ProviderTypeGCPKMS is the provider type for Google Cloud KMS.
	ProviderTypeGCPKMS = crypto_dto.ProviderTypeGCPKMS

	// ProviderTypeAzureKeyVault is the provider type for Azure Key Vault.
	ProviderTypeAzureKeyVault = crypto_dto.ProviderTypeAzureKeyVault

	// ProviderTypeHashiCorpVault is the provider type for HashiCorp Vault.
	ProviderTypeHashiCorpVault = crypto_dto.ProviderTypeHashiCorpVault

	// KeyStatusActive is the status for keys that are in use.
	KeyStatusActive = crypto_dto.KeyStatusActive

	// KeyStatusDeprecated indicates that a key has been marked as deprecated.
	KeyStatusDeprecated = crypto_dto.KeyStatusDeprecated

	// KeyStatusDisabled is a disabled key status value from the crypto DTO.
	KeyStatusDisabled = crypto_dto.KeyStatusDisabled

	// KeyStatusDestroyed indicates that the key has been permanently destroyed.
	KeyStatusDestroyed = crypto_dto.KeyStatusDestroyed

	// errFmtGettingDefaultService is the format string for wrapping errors when
	// retrieving the default crypto service.
	errFmtGettingDefaultService = "getting default crypto service: %w"
)

var (
	// ErrKeyNotFound is returned when a requested key does not exist.
	ErrKeyNotFound = crypto_dto.ErrKeyNotFound

	// ErrInvalidKey is returned when the provided key is invalid or malformed.
	ErrInvalidKey = crypto_dto.ErrInvalidKey

	// ErrInvalidCiphertext is returned when the ciphertext cannot be decrypted.
	ErrInvalidCiphertext = crypto_dto.ErrInvalidCiphertext

	// ErrDecryptionFailed is returned when decryption of data fails.
	ErrDecryptionFailed = crypto_dto.ErrDecryptionFailed

	// ErrEncryptionFailed is returned when data encryption fails.
	ErrEncryptionFailed = crypto_dto.ErrEncryptionFailed

	// ErrProviderUnavailable is returned when the requested cryptographic
	// provider is not available or cannot be initialised.
	ErrProviderUnavailable = crypto_dto.ErrProviderUnavailable

	// ErrInvalidProvider is returned when an unrecognised or unsupported
	// cryptographic provider is specified.
	ErrInvalidProvider = crypto_dto.ErrInvalidProvider

	// ErrEmptyPlaintext is returned when encryption is attempted with empty input.
	ErrEmptyPlaintext = crypto_dto.ErrEmptyPlaintext

	// ErrEmptyCiphertext is returned when the ciphertext to decrypt is empty.
	ErrEmptyCiphertext = crypto_dto.ErrEmptyCiphertext

	// ErrContextMismatch is returned when the context does not match the expected
	// encryption context.
	ErrContextMismatch = crypto_dto.ErrContextMismatch

	// ErrKeyRotationInProgress is returned when an operation cannot proceed
	// because a key rotation is already in progress.
	ErrKeyRotationInProgress = crypto_dto.ErrKeyRotationInProgress
)

// ServicePort is the main interface for encryption operations.
// Application code should depend on this rather than concrete types.
//
// The service provides a high-level API that hides provider details and
// includes automatic key selection, envelope encryption for batch operations,
// key rotation support, and observability integration.
type ServicePort = crypto_domain.CryptoServicePort

// EncryptionProvider is the interface that all encryption adapters must
// implement. Implementations include local_aes_gcm.Provider (always available),
// aws_kms.Provider, and gcp_kms.Provider (both require explicit registration).
type EncryptionProvider = crypto_domain.EncryptionProvider

// LocalProviderFactory creates short-lived encryption providers for envelope
// encryption. The crypto service uses this to create temporary providers
// set up with data keys.
type LocalProviderFactory = crypto_domain.LocalProviderFactory

// EncryptBuilder provides a fluent interface for encrypting data.
type EncryptBuilder = crypto_domain.EncryptBuilder

// DecryptBuilder provides a fluent interface for decrypting data.
type DecryptBuilder = crypto_domain.DecryptBuilder

// BatchEncryptBuilder provides a builder for encrypting multiple values.
type BatchEncryptBuilder = crypto_domain.BatchEncryptBuilder

// BatchDecryptBuilder provides a builder for decrypting multiple items at once.
type BatchDecryptBuilder = crypto_domain.BatchDecryptBuilder

// StreamEncryptBuilder provides a builder for streaming encryption.
type StreamEncryptBuilder = crypto_domain.StreamEncryptBuilder

// StreamDecryptBuilder provides a builder for setting up streaming decryption.
type StreamDecryptBuilder = crypto_domain.StreamDecryptBuilder

// ServiceConfig holds settings for the crypto service.
type ServiceConfig = crypto_dto.ServiceConfig

// ProviderType identifies which encryption provider to use.
type ProviderType = crypto_dto.ProviderType

// EncryptRequest is an alias for crypto_dto.EncryptRequest.
type EncryptRequest = crypto_dto.EncryptRequest

// DecryptRequest represents a request to decrypt ciphertext.
type DecryptRequest = crypto_dto.DecryptRequest

// GenerateDataKeyRequest is a request to create a new data encryption key.
type GenerateDataKeyRequest = crypto_dto.GenerateDataKeyRequest

// EncryptResponse represents the result of an encryption operation.
type EncryptResponse = crypto_dto.EncryptResponse

// DecryptResponse holds the result of a decryption operation.
type DecryptResponse = crypto_dto.DecryptResponse

// DataKey represents a key used to encrypt data in envelope encryption.
type DataKey = crypto_dto.DataKey

// KeyInfo represents metadata about an encryption key.
type KeyInfo = crypto_dto.KeyInfo

// KeyStatus represents the state of an encryption key in its lifecycle.
type KeyStatus = crypto_dto.KeyStatus

// KeyRotationPolicy defines when and how keys should be rotated.
type KeyRotationPolicy = crypto_dto.KeyRotationPolicy

// EncryptionError wraps an encryption error with added context.
type EncryptionError = crypto_dto.EncryptionError

// ServiceOption is a function that sets up options for the crypto service.
type ServiceOption = crypto_domain.ServiceOption

// DefaultServiceConfig returns a service config with sensible default values.
//
// Returns *ServiceConfig which contains the default settings.
func DefaultServiceConfig() *ServiceConfig {
	return crypto_dto.DefaultServiceConfig()
}

// DefaultKeyRotationPolicy returns a rotation policy with sensible defaults.
//
// Returns *KeyRotationPolicy which contains pre-set rotation settings.
func DefaultKeyRotationPolicy() *KeyRotationPolicy {
	return crypto_dto.DefaultKeyRotationPolicy()
}

// NewEncryptionError creates a new EncryptionError with context about the
// failed operation.
//
// Takes op (string) which specifies the operation that failed.
// Takes provider (string) which identifies the encryption provider.
// Takes keyID (string) which identifies the key involved.
// Takes err (error) which is the cause of the failure.
//
// Returns *EncryptionError which wraps the error with encryption context.
func NewEncryptionError(op, provider, keyID string, err error) *EncryptionError {
	return crypto_dto.NewEncryptionError(op, provider, keyID, err)
}

// WithLocalProviderFactory sets a factory for creating short-lived local
// providers. This is needed for envelope encryption in batch operations.
//
// If not set when using batch operations, the service will return an error.
// The bootstrap layer adds a default factory on its own.
//
// Takes factory (crypto_domain.LocalProviderFactory) which creates short-lived
// local providers for batch encryption operations.
//
// Returns ServiceOption which sets up the service with the given factory.
func WithLocalProviderFactory(factory crypto_domain.LocalProviderFactory) ServiceOption {
	return crypto_domain.WithLocalProviderFactory(factory)
}

// NewService creates a new crypto service with the given settings.
//
// This is the low-level constructor. Most applications should use
// GetDefaultService or set options via piko.New instead.
//
// The cacheService parameter may be nil to turn off data key caching (e.g., for
// testing or when using the local provider). For production KMS providers,
// providing a cache service greatly improves performance by caching decrypted
// data keys for a short TTL.
//
// For batch operations (EncryptBatch/DecryptBatch), you must provide a local
// provider factory via WithLocalProviderFactory option.
//
// After creating the service, you must register at least one provider using
// RegisterProvider() and set a default provider using SetDefaultProvider().
//
// Takes cacheService (cache_domain.Service) which caches decrypted data keys, or
// nil to turn off caching.
// Takes config (*ServiceConfig) which specifies service settings including the
// active key ID.
// Takes opts (...ServiceOption) which provides optional behaviour controls
// such as WithLocalProviderFactory.
//
// Returns ServicePort which is the crypto service ready for provider
// registration.
// Returns error when the configuration is not valid.
//
// Example:
//
//	provider := local_aes_gcm.NewProvider(...)
//	factory := local_aes_gcm.NewFactory()
//	cacheService := cache.GetDefaultService() // or nil to turn off caching
//	config := crypto.DefaultServiceConfig()
//	config.ActiveKeyID = "my-key-id"
//	service, err := crypto.NewService(cacheService, config,
//		crypto.WithLocalProviderFactory(factory),
//	)
//	if err != nil {
//		return nil, err
//	}
//
//	// Register the provider
//	err = service.RegisterProvider("local-aes-gcm", provider)
//	if err != nil {
//		return nil, err
//	}
//
//	// Set it as the default
//	err = service.SetDefaultProvider("local-aes-gcm")
//	if err != nil {
//		return nil, err
//	}
func NewService(ctx context.Context, cacheService cache_domain.Service, config *ServiceConfig, opts ...ServiceOption) (ServicePort, error) {
	return crypto_domain.NewCryptoService(ctx, cacheService, config, opts...)
}

// GetDefaultService returns the default crypto service configured during
// bootstrap.
//
// This is the recommended way to access the crypto service in most application
// code. The service is configured via:
//   - piko.New() options (recommended)
//   - config/server_config.yaml
//   - Environment variables
//
// Example:
//
//	cryptoService, err := crypto.GetDefaultService()
//	if err != nil {
//	    return err
//	}
//	encrypted, err := cryptoService.Encrypt(ctx, "sensitive-data")
//
// Returns ServicePort which is the configured crypto service.
// Returns error when the framework is not initialised or the service cannot be
// created.
func GetDefaultService() (ServicePort, error) {
	service, err := bootstrap.GetCryptoService()
	if err != nil {
		return nil, fmt.Errorf("crypto: get default service: %w", err)
	}
	return service, nil
}

// Encrypt encrypts plaintext using the default crypto service set up during
// bootstrap.
//
// This is a convenience wrapper around GetCryptoService().Encrypt().
//
// Takes plaintext (string) which is the data to encrypt.
//
// Returns string which is the encrypted data.
// Returns error when encryption fails or the service is not set up.
func Encrypt(ctx context.Context, plaintext string) (string, error) {
	service, err := bootstrap.GetCryptoService()
	if err != nil {
		return "", fmt.Errorf("crypto: encrypt: %w", err)
	}
	return service.Encrypt(ctx, plaintext)
}

// Decrypt decrypts ciphertext using the default crypto service set up during
// bootstrap.
//
// This is a wrapper around GetDefaultService().Decrypt().
//
// Takes ciphertext (string) which is the encrypted data to decrypt.
//
// Returns string which is the decrypted plaintext.
// Returns error when decryption fails or the service is not set up.
func Decrypt(ctx context.Context, ciphertext string) (string, error) {
	service, err := bootstrap.GetCryptoService()
	if err != nil {
		return "", fmt.Errorf("crypto: decrypt: %w", err)
	}
	return service.Decrypt(ctx, ciphertext)
}

// EncryptBatch encrypts multiple plaintexts using envelope encryption with the
// default service.
//
// This is a convenience wrapper around GetDefaultService().EncryptBatch().
//
// Example:
//
//	tokens := []string{"token1", "token2", "token3"}
//	encrypted, err := crypto.EncryptBatch(ctx, tokens)
//
// Takes plaintexts ([]string) which contains the values to encrypt.
//
// Returns []string which contains the encrypted strings. On error, the
// returned slice is nil; partial results are never returned.
// Returns error when encryption fails or the service is not set up.
func EncryptBatch(ctx context.Context, plaintexts []string) ([]string, error) {
	service, err := bootstrap.GetCryptoService()
	if err != nil {
		return nil, fmt.Errorf("crypto: encrypt batch: %w", err)
	}
	return service.EncryptBatch(ctx, plaintexts)
}

// DecryptBatch decrypts multiple ciphertexts using the default service.
//
// This is a convenience wrapper around GetDefaultService().DecryptBatch().
//
// Takes ciphertexts ([]string) which contains the encrypted strings to decrypt.
//
// Returns []string which contains the decrypted plaintext strings. On error,
// the returned slice is nil; partial results are never returned.
// Returns error when decryption fails or the service is not set up.
func DecryptBatch(ctx context.Context, ciphertexts []string) ([]string, error) {
	service, err := bootstrap.GetCryptoService()
	if err != nil {
		return nil, fmt.Errorf("crypto: decrypt batch: %w", err)
	}
	return service.DecryptBatch(ctx, ciphertexts)
}

// NewEncryptBuilder creates an encryption builder using the given service.
//
// Takes service (ServicePort) which is the crypto service to use.
//
// Returns *EncryptBuilder which provides a fluent interface for encrypting
// data.
func NewEncryptBuilder(service ServicePort) *EncryptBuilder {
	return service.NewEncrypt()
}

// NewEncryptBuilderFromDefault creates a new encryption builder using the
// framework's bootstrapped service.
//
// Returns *EncryptBuilder which is the configured builder ready for use.
// Returns error when the framework has not been bootstrapped.
//
// Example:
//
//	builder, err := crypto.NewEncryptBuilderFromDefault()
//	if err != nil {
//	    return err
//	}
//	encrypted, err := builder.Data("sensitive-value").Do(ctx)
func NewEncryptBuilderFromDefault() (*EncryptBuilder, error) {
	service, err := GetDefaultService()
	if err != nil {
		return nil, fmt.Errorf(errFmtGettingDefaultService, err)
	}
	return NewEncryptBuilder(service), nil
}

// NewDecryptBuilder creates a new decryption builder with a given service.
//
// Takes service (ServicePort) which is the crypto service to use.
//
// Returns *DecryptBuilder which provides a fluent interface for decrypting
// data.
func NewDecryptBuilder(service ServicePort) *DecryptBuilder {
	return service.NewDecrypt()
}

// NewDecryptBuilderFromDefault creates a new decryption builder using the
// framework's bootstrapped service.
//
// Returns *DecryptBuilder which is the configured builder ready for use.
// Returns error when the framework has not been bootstrapped.
//
// Example:
//
//	builder, err := crypto.NewDecryptBuilderFromDefault()
//	if err != nil {
//	    return err
//	}
//	plaintext, err := builder.Data(ciphertext).Do(ctx)
func NewDecryptBuilderFromDefault() (*DecryptBuilder, error) {
	service, err := GetDefaultService()
	if err != nil {
		return nil, fmt.Errorf(errFmtGettingDefaultService, err)
	}
	return NewDecryptBuilder(service), nil
}

// NewBatchEncryptBuilder creates a new batch encryption builder with an
// explicit service.
//
// Takes service (ServicePort) which is the crypto service to use.
//
// Returns *BatchEncryptBuilder which provides a fluent interface for batch
// encryption.
//
// Example:
//
//	service := crypto.NewService(provider, cacheService, config)
//	encrypted, err := crypto.NewBatchEncryptBuilder(service).
//	    Items(tokens).
//	    Do(ctx)
func NewBatchEncryptBuilder(service ServicePort) *BatchEncryptBuilder {
	return service.NewBatchEncrypt()
}

// NewBatchEncryptBuilderFromDefault creates a new batch encryption builder
// using the framework's bootstrapped service.
//
// Returns *BatchEncryptBuilder which is the configured builder ready for use.
// Returns error when the framework has not been bootstrapped.
//
// Example:
//
//	builder, err := crypto.NewBatchEncryptBuilderFromDefault()
//	if err != nil {
//	    return err
//	}
//	encrypted, err := builder.Items(tokens).Do(ctx)
func NewBatchEncryptBuilderFromDefault() (*BatchEncryptBuilder, error) {
	service, err := GetDefaultService()
	if err != nil {
		return nil, fmt.Errorf(errFmtGettingDefaultService, err)
	}
	return NewBatchEncryptBuilder(service), nil
}

// NewBatchDecryptBuilder creates a new batch decryption builder with an
// explicit service.
//
// Takes service (ServicePort) which is the crypto service to use.
//
// Returns *BatchDecryptBuilder which provides a fluent interface for batch
// decryption.
//
// Example:
//
//	service := crypto.NewService(provider, cacheService, config)
//	plaintexts, err := crypto.NewBatchDecryptBuilder(service).
//	    Items(encryptedTokens).
//	    Do(ctx)
func NewBatchDecryptBuilder(service ServicePort) *BatchDecryptBuilder {
	return service.NewBatchDecrypt()
}

// NewBatchDecryptBuilderFromDefault creates a new batch decryption builder
// using the framework's bootstrapped service.
//
// Returns *BatchDecryptBuilder which is the configured builder ready for use.
// Returns error when the framework has not been bootstrapped.
//
// Example:
//
//	builder, err := crypto.NewBatchDecryptBuilderFromDefault()
//	if err != nil {
//	    return err
//	}
//	plaintexts, err := builder.Items(encryptedTokens).Do(ctx)
func NewBatchDecryptBuilderFromDefault() (*BatchDecryptBuilder, error) {
	service, err := GetDefaultService()
	if err != nil {
		return nil, fmt.Errorf(errFmtGettingDefaultService, err)
	}
	return NewBatchDecryptBuilder(service), nil
}

// NewStreamEncryptBuilder creates a streaming encryption builder.
//
// Takes service (ServicePort) which is the crypto service to use.
//
// Returns *StreamEncryptBuilder which provides a fluent interface for
// streaming encryption.
//
// Example:
//
//	service := crypto.NewService(provider, cacheService, config)
//	writer, err := crypto.NewStreamEncryptBuilder(service).
//	    Output(outputFile).
//	    KeyID("key-id").
//	    Stream(ctx)
func NewStreamEncryptBuilder(service ServicePort) *StreamEncryptBuilder {
	return service.NewStreamEncrypt()
}

// NewStreamEncryptBuilderFromDefault creates a new streaming encryption
// builder using the framework's bootstrapped service.
//
// Returns *StreamEncryptBuilder which is the configured builder.
// Returns error when the framework has not been bootstrapped.
func NewStreamEncryptBuilderFromDefault() (*StreamEncryptBuilder, error) {
	service, err := GetDefaultService()
	if err != nil {
		return nil, fmt.Errorf(errFmtGettingDefaultService, err)
	}
	return NewStreamEncryptBuilder(service), nil
}

// NewStreamDecryptBuilder creates a streaming decryption builder with the
// provided service.
//
// Takes service (ServicePort) which is the crypto service to use.
//
// Returns *StreamDecryptBuilder which provides a fluent interface for
// streaming decryption.
//
// Example:
//
//	service := crypto.NewService(provider, cacheService, config)
//	reader, err := crypto.NewStreamDecryptBuilder(service).
//	    Input(encryptedFile).
//	    Stream(ctx)
func NewStreamDecryptBuilder(service ServicePort) *StreamDecryptBuilder {
	return service.NewStreamDecrypt()
}

// NewStreamDecryptBuilderFromDefault creates a new streaming decryption
// builder using the framework's bootstrapped service.
//
// Returns *StreamDecryptBuilder which is the configured builder.
// Returns error when the framework has not been bootstrapped.
func NewStreamDecryptBuilderFromDefault() (*StreamDecryptBuilder, error) {
	service, err := GetDefaultService()
	if err != nil {
		return nil, fmt.Errorf(errFmtGettingDefaultService, err)
	}
	return NewStreamDecryptBuilder(service), nil
}

// IsValidProviderType checks if the provider type is recognised.
//
// Takes pt (ProviderType) which is the provider type to validate.
//
// Returns bool which is true if the provider type is valid.
func IsValidProviderType(pt ProviderType) bool {
	return pt.IsValid()
}

// ExampleBasicEncryption shows how to encrypt and decrypt a string.
//
// Panics if the crypto service cannot be set up, or if encryption or
// decryption fails.
func ExampleBasicEncryption() {
	ctx := context.Background()
	cryptoService, err := GetDefaultService()
	if err != nil {
		panic(err)
	}

	encrypted, err := cryptoService.Encrypt(ctx, "my-secret-token")
	if err != nil {
		panic(err)
	}

	_ = encrypted

	plaintext, err := cryptoService.Decrypt(ctx, encrypted)
	if err != nil {
		panic(err)
	}

	_ = plaintext
}

// ExampleBatchOperations shows how to encrypt and decrypt many tokens at once.
//
// Panics if the crypto service cannot be set up or if encryption or decryption
// fails.
func ExampleBatchOperations() {
	const exampleBatchSize = 1000

	ctx := context.Background()
	cryptoService, err := GetDefaultService()
	if err != nil {
		panic(err)
	}

	tokens := make([]string, exampleBatchSize)
	for i := range tokens {
		tokens[i] = "user-token-" + string(rune(i))
	}

	encrypted, err := cryptoService.EncryptBatch(ctx, tokens)
	if err != nil {
		panic(err)
	}

	plaintext, err := cryptoService.DecryptBatch(ctx, encrypted)
	if err != nil {
		panic(err)
	}

	_ = plaintext
}

// ExampleKeyRotation shows how to rotate keys without downtime.
//
// Panics if GetDefaultService, RotateKey, or Encrypt fails.
func ExampleKeyRotation() {
	ctx := context.Background()
	cryptoService, err := GetDefaultService()
	if err != nil {
		panic(err)
	}

	err = cryptoService.RotateKey(ctx, "old-key-id", "new-key-id")
	if err != nil {
		panic(err)
	}

	newEncrypted, err := cryptoService.Encrypt(ctx, "new-data")
	if err != nil {
		panic(err)
	}

	_ = newEncrypted
}

// ExampleStreamingEncryption shows how to use streaming encryption for large
// files.
//
// Panics if the crypto service cannot be set up.
func ExampleStreamingEncryption() {
	_ = context.Background()
	cryptoService, err := GetDefaultService()
	if err != nil {
		panic(err)
	}
	_ = cryptoService
}
