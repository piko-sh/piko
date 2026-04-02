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

package crypto_dto

import (
	"errors"
	"fmt"
)

var (
	// ErrKeyNotFound indicates the specified key does not exist.
	ErrKeyNotFound = errors.New("encryption key not found")

	// ErrInvalidKey is returned when an encryption key is malformed or invalid.
	ErrInvalidKey = errors.New("invalid encryption key")

	// ErrInvalidCiphertext indicates the ciphertext is malformed or corrupted.
	ErrInvalidCiphertext = errors.New("invalid ciphertext format")

	// ErrDecryptionFailed indicates that decryption failed due to a wrong key,
	// tampered data, or other cryptographic error.
	ErrDecryptionFailed = errors.New("decryption failed")

	// ErrEncryptionFailed indicates that an encryption operation failed.
	ErrEncryptionFailed = errors.New("encryption failed")

	// ErrProviderUnavailable indicates the encryption provider is not available.
	ErrProviderUnavailable = errors.New("encryption provider unavailable")

	// ErrInvalidProvider is returned when the provider type is not supported.
	ErrInvalidProvider = errors.New("invalid or unsupported provider type")

	// ErrEmptyPlaintext indicates an attempt to encrypt empty data.
	ErrEmptyPlaintext = errors.New("plaintext cannot be empty")

	// ErrEmptyCiphertext indicates an attempt to decrypt empty data.
	ErrEmptyCiphertext = errors.New("ciphertext cannot be empty")

	// ErrContextMismatch indicates the decryption context does not match the
	// encryption context.
	ErrContextMismatch = errors.New("decryption context does not match encryption context")

	// ErrKeyRotationInProgress is returned when a key rotation is already running.
	ErrKeyRotationInProgress = errors.New("key rotation is in progress")

	// ErrCryptoDisabled indicates encryption is not configured.
	ErrCryptoDisabled = errors.New("crypto: encryption not configured - set security.encryptionKey or use piko.WithCryptoProvider()")
)

// EncryptionError wraps an encryption error with additional context.
type EncryptionError struct {
	// Err is the underlying error that caused the failure.
	Err error

	// Operation is the operation that failed, such as "Encrypt" or "Decrypt".
	Operation string

	// Provider is the name of the encryption provider that encountered the error.
	Provider string

	// KeyID is the identifier for the key used in the operation; empty if not used.
	KeyID string
}

// NewEncryptionError creates a new EncryptionError with the given details.
//
// Takes op (string) which specifies the operation that failed.
// Takes provider (string) which identifies the encryption provider.
// Takes keyID (string) which identifies the key involved in the error.
// Takes err (error) which is the underlying error that caused the failure.
//
// Returns *EncryptionError which wraps the error with encryption context.
func NewEncryptionError(op, provider, keyID string, err error) *EncryptionError {
	return &EncryptionError{
		Operation: op,
		Provider:  provider,
		KeyID:     keyID,
		Err:       err,
	}
}

// Error implements the error interface for EncryptionError.
//
// Returns string which describes the failed crypto operation, including the
// provider and key ID when available.
func (e *EncryptionError) Error() string {
	if e.KeyID != "" {
		return fmt.Sprintf("crypto %s failed with provider %s and key %s: %v", e.Operation, e.Provider, e.KeyID, e.Err)
	}
	return fmt.Sprintf("crypto %s failed with provider %s: %v", e.Operation, e.Provider, e.Err)
}

// Unwrap returns the underlying error.
//
// Returns error which is the wrapped error, or nil if none exists.
func (e *EncryptionError) Unwrap() error {
	return e.Err
}
