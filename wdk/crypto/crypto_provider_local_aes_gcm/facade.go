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

package crypto_provider_local_aes_gcm

import (
	"piko.sh/piko/internal/crypto/crypto_adapters/local_aes_gcm"
	"piko.sh/piko/internal/crypto/crypto_domain"
)

const (
	// KeySize is the required key size for AES-256 encryption (32 bytes).
	KeySize = local_aes_gcm.KeySize

	// IVSize is the size of the GCM initialisation vector in bytes.
	IVSize = local_aes_gcm.IVSize
)

var (
	// ErrInvalidKeySize is returned when the provided encryption key has an
	// incorrect length for AES-GCM operations.
	ErrInvalidKeySize = local_aes_gcm.ErrInvalidKeySize

	// ErrEmptyPlaintext is returned when encryption is attempted with empty input.
	ErrEmptyPlaintext = local_aes_gcm.ErrEmptyPlaintext

	// ErrEmptyCiphertext is returned when decryption is attempted with empty input.
	ErrEmptyCiphertext = local_aes_gcm.ErrEmptyCiphertext

	// ErrCiphertextTooShort is returned when the ciphertext is too short to be valid.
	ErrCiphertextTooShort = local_aes_gcm.ErrCiphertextTooShort

	// ErrInvalidBase64 is returned when the input is not valid base64 encoding.
	ErrInvalidBase64 = local_aes_gcm.ErrInvalidBase64
)

// Config holds configuration for the local AES-GCM encryption provider.
type Config = local_aes_gcm.Config

// NewProvider creates a new local AES-GCM encryption provider.
//
// The key must be exactly 32 bytes (256 bits) for AES-256.
//
// Takes config (Config) which specifies the encryption key and key identifier.
//
// Returns crypto_domain.EncryptionProvider which provides AES-GCM encryption.
// Returns error when the key is not exactly 32 bytes.
//
// Example:
//
//	key := make([]byte, 32)
//	_, _ = rand.Read(key)
//
//	provider, err := NewProvider(Config{
//		Key:   key,
//		KeyID: "production-key-v1",
//	})
//	if err != nil {
//		panic(err)
//	}
func NewProvider(config Config) (crypto_domain.EncryptionProvider, error) {
	return local_aes_gcm.NewProvider(config)
}
