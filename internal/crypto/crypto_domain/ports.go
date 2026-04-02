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
	"io"

	"piko.sh/piko/internal/crypto/crypto_dto"
	"piko.sh/piko/internal/provider/provider_domain"
)

// LocalProviderFactory creates ephemeral encryption providers for envelope
// encryption. This is used by the crypto service to create temporary providers
// configured with data keys.
//
// The factory pattern allows the domain to request providers with specific key
// material without knowing how those providers are implemented. This maintains
// the Dependency Inversion Principle - the domain depends on this interface,
// not on concrete implementations.
type LocalProviderFactory interface {
	// CreateWithKey creates a new encryption provider configured with the given
	// key material. This is used for envelope encryption where each batch
	// operation needs a provider configured with a unique data key from the KMS.
	//
	// The factory accesses the key material via SecureBytes.WithAccess and does
	// NOT take ownership. The caller remains responsible for calling Close() on
	// the SecureBytes when done.
	//
	// Takes key (*crypto_dto.SecureBytes) which provides the encryption key
	// material in secure memory (e.g., 32 bytes for AES-256).
	// Takes keyID (string) which identifies this ephemeral key (usually
	// "ephemeral-data-key").
	//
	// Returns EncryptionProvider which is configured with the provided key.
	// Returns error when the provider cannot be created.
	CreateWithKey(key *crypto_dto.SecureBytes, keyID string) (EncryptionProvider, error)
}

// EncryptionProvider defines the interface for encryption adapters.
//
// Implementations include:
//   - local_aes_gcm.Provider: Local AES-256-GCM encryption
//   - aws_kms.Provider: AWS Key Management Service
//   - gcp_kms.Provider: Google Cloud KMS
//   - azure_keyvault.Provider: Azure Key Vault
//   - hashicorp_vault.Provider: HashiCorp Vault
type EncryptionProvider interface {
	// Type returns the provider type for identification and logging.
	Type() crypto_dto.ProviderType

	// Encrypt encrypts plaintext and returns authenticated
	// ciphertext. The returned ciphertext format is
	// provider-specific but must be self-describing, containing
	// enough metadata to decrypt without additional context.
	//
	// Takes request (*crypto_dto.EncryptRequest) which contains
	// the plaintext to encrypt.
	//
	// Returns *crypto_dto.EncryptResponse which contains the encrypted ciphertext.
	// Returns error when encryption fails.
	Encrypt(ctx context.Context, request *crypto_dto.EncryptRequest) (*crypto_dto.EncryptResponse, error)

	// Decrypt decrypts authenticated ciphertext and returns plaintext. It verifies
	// authenticity before returning plaintext as required by AEAD.
	//
	// Takes request (*crypto_dto.DecryptRequest) which contains the ciphertext to
	// decrypt.
	//
	// Returns *crypto_dto.DecryptResponse which contains the decrypted plaintext.
	// Returns error when decryption or authentication fails.
	Decrypt(ctx context.Context, request *crypto_dto.DecryptRequest) (*crypto_dto.DecryptResponse, error)

	// GenerateDataKey creates a new data encryption key for envelope encryption.
	//
	// Takes request (*crypto_dto.GenerateDataKeyRequest) which specifies the key
	// generation parameters.
	//
	// Returns *crypto_dto.DataKey which contains both the plaintext key for
	// immediate use and the encrypted key for storage.
	// Returns error when key generation fails.
	//
	// Optional: not all providers need to implement this method. The local
	// provider uses the master key directly.
	GenerateDataKey(ctx context.Context, request *crypto_dto.GenerateDataKeyRequest) (*crypto_dto.DataKey, error)

	// GetKeyInfo returns metadata about the encryption key.
	//
	// Takes keyID (string) which identifies the key to retrieve information for.
	//
	// Returns *crypto_dto.KeyInfo which contains key metadata for auditing, key
	// rotation planning, and compliance reporting.
	// Returns error when the key cannot be found or retrieval fails.
	GetKeyInfo(ctx context.Context, keyID string) (*crypto_dto.KeyInfo, error)

	// HealthCheck verifies provider connectivity and configuration.
	//
	// Returns error when the health check fails.
	HealthCheck(ctx context.Context) error

	// EncryptStream encrypts a data stream.
	//
	// The caller writes plaintext to the returned WriteCloser, and encrypted
	// data is written to the provided output Writer.
	//
	// The encrypted output uses a streaming envelope format (version 2) that
	// includes:
	//   - A header with metadata (key ID, provider, IV, etc.)
	//   - The encrypted data stream
	//
	// Memory usage is constant regardless of stream size, making this suitable
	// for encrypting large files (e.g., >1GB).
	//
	// Takes output (io.Writer) which receives the encrypted data.
	// Takes request (*crypto_dto.EncryptRequest) which specifies encryption options.
	//
	// Returns plaintextWriter (io.WriteCloser) which accepts plaintext to encrypt.
	// Returns error when encryption setup fails.
	EncryptStream(ctx context.Context, output io.Writer, request *crypto_dto.EncryptRequest) (plaintextWriter io.WriteCloser, err error)

	// DecryptStream decrypts a data stream.
	// The caller reads plaintext from the returned ReadCloser.
	//
	// The method automatically detects and parses the streaming envelope format,
	// extracting metadata and setting up the decryption pipeline.
	//
	// Memory usage is constant regardless of stream size.
	DecryptStream(ctx context.Context, input io.Reader) (plaintextReader io.ReadCloser, err error)
}

// CryptoServicePort defines the public interface for encryption operations.
// This is what other hexagons (identity, content, tenancy, etc.) depend on.
//
// The service layer abstracts provider-specific details and provides:
//   - Simplified API for common use cases
//   - Automatic key selection (active vs deprecated keys)
//   - Ciphertext envelope management (metadata extraction)
//   - Observability integration (metrics, tracing, logging)
//   - Batch optimisation (envelope encryption)
type CryptoServicePort interface {
	// Encrypt encrypts a string value using the active key.
	//
	// Takes plaintext (string) which is the sensitive data to encrypt.
	//
	// Returns string which is the encrypted ciphertext.
	// Returns error when encryption fails.
	Encrypt(ctx context.Context, plaintext string) (string, error)

	// Decrypt decrypts a string value and returns the original plaintext.
	// It detects which key was used for encryption.
	//
	// Takes ciphertext (string) which is the encrypted value to decrypt.
	//
	// Returns string which is the decrypted plaintext.
	// Returns error when decryption fails.
	Decrypt(ctx context.Context, ciphertext string) (string, error)

	// EncryptWithKey encrypts plaintext using a specific key ID.
	//
	// Used for key rotation scenarios where you need to re-encrypt with a new key.
	//
	// Takes plaintext (string) which is the data to encrypt.
	// Takes keyID (string) which identifies the encryption key to use.
	//
	// Returns string which is the encrypted ciphertext.
	// Returns error when encryption fails.
	EncryptWithKey(ctx context.Context, plaintext string, keyID string) (string, error)

	// EncryptBatch encrypts multiple values efficiently.
	//
	// Uses envelope encryption to minimise provider calls (e.g., 1 KMS call
	// instead of 1000).
	//
	// Takes plaintexts ([]string) which contains the values to encrypt.
	//
	// Returns []string which contains the encrypted values.
	// Returns error when encryption fails.
	EncryptBatch(ctx context.Context, plaintexts []string) ([]string, error)

	// DecryptBatch decrypts multiple ciphertext values in a single operation.
	//
	// Takes ciphertexts ([]string) which contains the encrypted values to decrypt.
	//
	// Returns []string which contains the decrypted plaintext values.
	// Returns error when decryption fails for any value.
	DecryptBatch(ctx context.Context, ciphertexts []string) ([]string, error)

	// RotateKey initiates key rotation from oldKeyID to newKeyID.
	//
	// This marks oldKeyID as deprecated and newKeyID as active. Future versions
	// may trigger a background re-encryption job.
	//
	// Takes oldKeyID (string) which identifies the key to deprecate.
	// Takes newKeyID (string) which identifies the key to make active.
	//
	// Returns error when the rotation fails.
	RotateKey(ctx context.Context, oldKeyID, newKeyID string) error

	// GetActiveKeyID returns the currently active key ID. Useful for diagnostics
	// and key rotation planning.
	//
	// Returns string which is the active key identifier.
	// Returns error when the active key cannot be determined.
	GetActiveKeyID(ctx context.Context) (string, error)

	// DecryptAndReEncrypt decrypts ciphertext and optionally re-encrypts it if
	// using a deprecated key. This enables gradual key rotation without explicit
	// migration.
	//
	// Takes ciphertext (string) which is the encrypted data to decrypt.
	//
	// Returns plaintext (string) which is the decrypted content.
	// Returns newCiphertext (string) which is the re-encrypted ciphertext, only
	// set if the key was deprecated and EnableAutoReEncrypt is true.
	// Returns wasReEncrypted (bool) which indicates if re-encryption occurred.
	// Returns err (error) when decryption or re-encryption fails.
	DecryptAndReEncrypt(ctx context.Context, ciphertext string) (plaintext, newCiphertext string, wasReEncrypted bool, err error)

	// HealthCheck verifies the crypto service is operational by performing a full
	// encrypt/decrypt roundtrip with the active provider.
	//
	// Returns error when the health check fails.
	HealthCheck(ctx context.Context) error

	// EncryptStream encrypts a data stream using the specified key, or the active
	// key if keyID is empty. The caller writes plaintext to the returned
	// WriteCloser, and encrypted data is written to the provided output Writer.
	//
	// This method is designed for encrypting large files without loading them
	// entirely into memory. Memory usage remains constant regardless of stream
	// size.
	//
	// Takes output (io.Writer) which receives the encrypted data.
	// Takes keyID (string) which specifies the encryption key, or empty for the
	// active key.
	//
	// Returns io.WriteCloser which accepts plaintext to be encrypted.
	// Returns error when the key cannot be found or encryption setup fails.
	EncryptStream(ctx context.Context, output io.Writer, keyID string) (io.WriteCloser, error)

	// DecryptStream decrypts a data stream.
	// The caller reads plaintext from the returned ReadCloser.
	//
	// This method automatically detects the envelope format (v1 or v2) and sets up
	// the appropriate decryption pipeline. Suitable for decrypting large files.
	DecryptStream(ctx context.Context, input io.Reader) (io.ReadCloser, error)

	// NewEncrypt creates a new encryption builder.
	//
	// Returns *EncryptBuilder which is used to configure and build encryption.
	NewEncrypt() *EncryptBuilder

	// NewDecrypt creates a new decryption builder.
	//
	// Returns *DecryptBuilder which is used to set up a decryption operation.
	NewDecrypt() *DecryptBuilder

	// NewBatchEncrypt creates a new batch encryption builder.
	//
	// Returns *BatchEncryptBuilder which is used to build batch encryption tasks.
	NewBatchEncrypt() *BatchEncryptBuilder

	// NewBatchDecrypt creates a new batch decryption builder.
	//
	// Returns *BatchDecryptBuilder which is used to build batch decryption
	// operations.
	NewBatchDecrypt() *BatchDecryptBuilder

	// NewStreamEncrypt creates a new builder for streaming encryption.
	//
	// Returns *StreamEncryptBuilder which configures the encryption stream.
	NewStreamEncrypt() *StreamEncryptBuilder

	// NewStreamDecrypt creates a new builder for streaming decryption.
	//
	// Returns *StreamDecryptBuilder which configures the decryption stream.
	NewStreamDecrypt() *StreamDecryptBuilder

	// RegisterProvider adds a new encryption provider with the given name.
	//
	// Takes ctx (context.Context) which carries cancellation and tracing.
	// Takes name (string) which identifies the provider.
	// Takes provider (EncryptionProvider) which handles encryption operations.
	//
	// Returns error when the provider cannot be registered.
	RegisterProvider(ctx context.Context, name string, provider EncryptionProvider) error

	// SetDefaultProvider sets the provider to use when no specific provider is
	// named.
	//
	// Takes name (string) which is the name of the provider to use as default.
	//
	// Returns error when the named provider does not exist.
	SetDefaultProvider(name string) error

	// GetProviders returns a sorted list of all registered provider names.
	//
	// Takes ctx (context.Context) which carries cancellation and tracing.
	GetProviders(ctx context.Context) []string

	// HasProvider checks if a provider with the given name has been registered.
	//
	// Takes name (string) which is the provider name to look for.
	//
	// Returns bool which is true if the provider exists, false otherwise.
	HasProvider(name string) bool

	// ListProviders returns details about all registered providers.
	//
	// Returns []provider_domain.ProviderInfo which contains information about each
	// provider.
	ListProviders(ctx context.Context) []provider_domain.ProviderInfo

	// Close shuts down all providers in a controlled manner.
	//
	// Returns error when shutdown fails or the context is cancelled.
	Close(ctx context.Context) error
}
