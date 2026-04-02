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

package local_aes_gcm

import (
	"cmp"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"piko.sh/piko/internal/crypto/crypto_domain"
	"piko.sh/piko/internal/crypto/crypto_dto"
)

const (
	// IVSize is the size of the initialisation vector for AES-GCM in bytes.
	IVSize = 12

	// KeySize is the required key length in bytes for AES-256 encryption.
	KeySize = 32
)

var (
	// ErrInvalidKeySize is returned when the key is not exactly 32 bytes.
	ErrInvalidKeySize = errors.New("invalid key size: must be exactly 32 bytes for AES-256")

	// ErrEmptyPlaintext is returned when attempting to encrypt empty plaintext.
	ErrEmptyPlaintext = errors.New("plaintext cannot be empty")

	// ErrEmptyCiphertext is returned when attempting to decrypt empty ciphertext.
	ErrEmptyCiphertext = errors.New("ciphertext cannot be empty")

	// ErrCiphertextTooShort is returned when ciphertext is shorter than the
	// minimum length required for decryption.
	ErrCiphertextTooShort = errors.New("ciphertext too short: must be at least IV + tag length")

	// ErrInvalidBase64 is returned when ciphertext is not valid base64 encoding.
	ErrInvalidBase64 = errors.New("ciphertext is not valid base64 encoding")
)

// provider implements local AES-256-GCM encryption without external dependencies.
// It is suitable for development, testing, and single-server deployments.
type provider struct {
	// aead provides authenticated encryption for sealing and opening data.
	aead cipher.AEAD

	// keyID is the identifier returned in encryption responses.
	keyID string
}

var _ crypto_domain.EncryptionProvider = (*provider)(nil)

// Type returns the provider type.
//
// Returns crypto_dto.ProviderType which identifies this as a local AES-GCM
// provider.
func (*provider) Type() crypto_dto.ProviderType {
	return crypto_dto.ProviderTypeLocalAESGCM
}

// Encrypt encrypts plaintext using AES-256-GCM.
//
// The encryption process:
//  1. Generates a random 12-byte IV (must be unique for each encryption)
//  2. Encrypts plaintext with AEAD.Seal (produces ciphertext and 16-byte tag)
//  3. Prepends IV to ciphertext: [iv][ciphertext][tag]
//  4. Base64-encodes the result for storage or sending
//
// Output format: base64([12-byte iv][ciphertext][16-byte tag])
//
// Takes request (*crypto_dto.EncryptRequest) which contains the plaintext to
// encrypt.
//
// Returns *crypto_dto.EncryptResponse which contains the base64-encoded
// ciphertext, key ID, and provider type.
// Returns error when the plaintext is empty or IV generation fails.
func (p *provider) Encrypt(_ context.Context, request *crypto_dto.EncryptRequest) (*crypto_dto.EncryptResponse, error) {
	if request.Plaintext == "" {
		return nil, crypto_dto.ErrEmptyPlaintext
	}

	iv := make([]byte, IVSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, fmt.Errorf("failed to generate IV: %w", err)
	}

	plaintextBytes := []byte(request.Plaintext)
	//nolint:gosec // IV from crypto/rand
	sealed := p.aead.Seal(iv, iv, plaintextBytes, nil)

	encoded := base64.StdEncoding.EncodeToString(sealed)

	return &crypto_dto.EncryptResponse{
		Ciphertext: encoded,
		KeyID:      p.keyID,
		Provider:   crypto_dto.ProviderTypeLocalAESGCM,
	}, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM.
//
// The algorithm base64-decodes the input, extracts the IV (first 12 bytes)
// and ciphertext with tag (remaining bytes), then verifies the authentication
// tag and decrypts using AEAD.Open.
//
// Takes request (*crypto_dto.DecryptRequest) which contains the base64-encoded
// ciphertext to decrypt.
//
// Returns *crypto_dto.DecryptResponse which contains the decrypted plaintext.
// Returns error when the ciphertext is empty, not valid base64, too short,
// or when the authentication tag is invalid (data tampered).
func (p *provider) Decrypt(_ context.Context, request *crypto_dto.DecryptRequest) (*crypto_dto.DecryptResponse, error) {
	if request.Ciphertext == "" {
		return nil, crypto_dto.ErrEmptyCiphertext
	}

	sealed, err := base64.StdEncoding.DecodeString(request.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidBase64, err)
	}

	minLength := IVSize + p.aead.Overhead()
	if len(sealed) < minLength {
		return nil, fmt.Errorf("%w: got %d bytes, need at least %d", ErrCiphertextTooShort, len(sealed), minLength)
	}

	iv := sealed[:IVSize]
	ciphertextWithTag := sealed[IVSize:]

	plaintext, err := p.aead.Open(nil, iv, ciphertextWithTag, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed (invalid ciphertext or tampered data): %w", err)
	}

	return &crypto_dto.DecryptResponse{
		Plaintext: string(plaintext),
	}, nil
}

// GenerateDataKey generates a random data encryption key for envelope
// encryption patterns. For the local provider, this generates a random
// 32-byte key and encrypts it with the master key.
//
// Takes request (*crypto_dto.GenerateDataKeyRequest) which specifies the key ID
// to use for encrypting the generated data key.
//
// Returns *crypto_dto.DataKey which contains the plaintext key in secure
// memory (mmap+mlock) and the encrypted key. The PlaintextKey MUST be closed
// by the caller when no longer needed.
// Returns error when key generation, encryption, or secure memory allocation
// fails.
func (p *provider) GenerateDataKey(ctx context.Context, request *crypto_dto.GenerateDataKeyRequest) (*crypto_dto.DataKey, error) {
	dataKey := make([]byte, KeySize)
	if _, err := io.ReadFull(rand.Reader, dataKey); err != nil {
		return nil, fmt.Errorf("failed to generate data key: %w", err)
	}

	encryptResp, err := p.Encrypt(ctx, &crypto_dto.EncryptRequest{
		Plaintext: base64.StdEncoding.EncodeToString(dataKey),
		KeyID:     request.KeyID,
	})
	if err != nil {
		zeroBytes(dataKey)
		return nil, fmt.Errorf("failed to encrypt data key: %w", err)
	}

	secureKey, err := crypto_dto.NewSecureBytesFromSlice(dataKey, crypto_dto.WithID("local-datakey-"+p.keyID))
	if err != nil {
		zeroBytes(dataKey)
		return nil, fmt.Errorf("failed to create secure bytes for data key: %w", err)
	}

	zeroBytes(dataKey)

	return &crypto_dto.DataKey{
		PlaintextKey: secureKey,
		EncryptedKey: encryptResp.Ciphertext,
		KeyID:        p.keyID,
		Provider:     crypto_dto.ProviderTypeLocalAESGCM,
	}, nil
}

// GetKeyInfo returns metadata about the specified key.
// For a local provider, this returns mostly fixed information.
//
// Takes keyID (string) which identifies the key to look up.
//
// Returns *crypto_dto.KeyInfo which contains the key metadata.
// Returns error when the keyID does not match the provider's key.
func (p *provider) GetKeyInfo(_ context.Context, keyID string) (*crypto_dto.KeyInfo, error) {
	if keyID != "" && keyID != p.keyID {
		return nil, crypto_dto.ErrKeyNotFound
	}

	return &crypto_dto.KeyInfo{
		KeyID:     p.keyID,
		Provider:  crypto_dto.ProviderTypeLocalAESGCM,
		Algorithm: "AES-256-GCM",
		CreatedAt: time.Now(),
		Status:    crypto_dto.KeyStatusActive,
		Origin:    "LOCAL",
		Metadata: map[string]string{
			"iv_size":  fmt.Sprintf("%d", IVSize),
			"tag_size": fmt.Sprintf("%d", p.aead.Overhead()),
		},
	}, nil
}

// HealthCheck checks that the provider is working by running a test
// encryption and decryption cycle.
//
// Returns error when encryption fails, decryption fails, or the decrypted
// result does not match the original plaintext.
func (p *provider) HealthCheck(ctx context.Context) error {
	testPlaintext := "health-check-test"

	encrypted, err := p.Encrypt(ctx, &crypto_dto.EncryptRequest{
		Plaintext: testPlaintext,
	})
	if err != nil {
		return fmt.Errorf("health check encryption failed: %w", err)
	}

	decrypted, err := p.Decrypt(ctx, &crypto_dto.DecryptRequest{
		Ciphertext: encrypted.Ciphertext,
	})
	if err != nil {
		return fmt.Errorf("health check decryption failed: %w", err)
	}

	if decrypted.Plaintext != testPlaintext {
		return fmt.Errorf("health check validation failed: expected %q, got %q", testPlaintext, decrypted.Plaintext)
	}

	return nil
}

// NewProvider creates a local AES-GCM encryption provider.
//
// Takes config (Config) which specifies the encryption key and key ID.
//
// Returns crypto_domain.EncryptionProvider which provides AES-GCM encryption.
// Returns error when the configuration is invalid or cipher creation fails.
func NewProvider(config Config) (crypto_domain.EncryptionProvider, error) {
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("validating AES-GCM provider config: %w", err)
	}

	block, err := aes.NewCipher(config.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	keyID := cmp.Or(config.KeyID, "local-default")

	return &provider{
		aead:  aead,
		keyID: keyID,
	}, nil
}

// zeroBytes clears the given byte slice in a way that the compiler cannot
// remove during optimisation.
//
// Takes data ([]byte) which is the memory to clear.
func zeroBytes(data []byte) {
	for i := range data {
		data[i] = 0
	}
	atomic.StoreInt32(new(int32), 0)
}
