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
	"slices"

	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	// errKeyIDsEmpty is returned when either the old or new key ID is empty
	// during key rotation.
	errKeyIDsEmpty = errors.New("both oldKeyID and newKeyID must be non-empty")

	// errKeyIDsSame is returned when the old and new key IDs are identical
	// during key rotation.
	errKeyIDsSame = errors.New("old and new key IDs cannot be the same")
)

// RotateKey initiates key rotation by marking the old key as deprecated and
// activating the new key.
//
// This is a graceful rotation strategy that allows existing encrypted data to
// remain encrypted with the old key while all new encryption operations use
// the new key.
//
// Takes oldKeyID (string) which identifies the currently active key to
// deprecate.
// Takes newKeyID (string) which identifies the key to activate.
//
// Returns error when either key ID is empty, both IDs are the same, the old
// key is not currently active, or the old key is already deprecated.
//
// Steps:
//  1. Validate the old and new key IDs
//  2. Mark the old key as deprecated (add to deprecatedKeyIDs list)
//  3. Set the new key as active
//  4. Log the rotation event
//
// This does NOT automatically re-encrypt existing data. For automatic
// re-encryption, use the Decrypt-and-re-encrypt pattern or trigger a
// background migration job.
func (s *cryptoService) RotateKey(ctx context.Context, oldKeyID, newKeyID string) error {
	ctx, l := logger_domain.From(ctx, log)
	if oldKeyID == "" || newKeyID == "" {
		return errKeyIDsEmpty
	}

	if oldKeyID == newKeyID {
		return errKeyIDsSame
	}

	if s.activeKeyID != oldKeyID {
		return fmt.Errorf("oldKeyID %q is not the active key (current active: %q)", oldKeyID, s.activeKeyID)
	}

	if slices.Contains(s.deprecatedKeyIDs, oldKeyID) {
		return fmt.Errorf("oldKeyID %q is already deprecated", oldKeyID)
	}

	s.deprecatedKeyIDs = append(s.deprecatedKeyIDs, oldKeyID)

	oldActive := s.activeKeyID
	s.activeKeyID = newKeyID

	provider, err := s.getProvider(ctx)
	providerType := "unknown"
	if err == nil {
		providerType = string(provider.Type())
	}

	l.Internal("Key rotation completed",
		logger_domain.String("old_key_id", oldActive),
		logger_domain.String("new_key_id", newKeyID),
		logger_domain.Int("deprecated_keys_count", len(s.deprecatedKeyIDs)),
	)

	cryptoOperationCount.Add(ctx, 1,
		metricAttributes(
			attributeKeyOperation, opKeyRotation,
			attributeKeyProvider, providerType,
			attributeKeyStatus, statusSuccess,
		),
	)

	return nil
}

// isDeprecatedKey checks if a given key ID is in the deprecated keys list.
//
// Takes keyID (string) which is the identifier to check.
//
// Returns bool which is true if the key is deprecated.
func (s *cryptoService) isDeprecatedKey(keyID string) bool {
	return slices.Contains(s.deprecatedKeyIDs, keyID)
}

// DecryptAndReEncrypt decrypts ciphertext and optionally re-encrypts it if
// using a deprecated key, supporting gradual key rotation without explicit
// migration.
//
// Takes ciphertext (string) which is the encrypted value to decrypt and
// potentially re-encrypt with the active key.
//
// Returns plaintext (string) which is the decrypted value.
// Returns newCiphertext (string) which is the re-encrypted value using
// the active key, or empty if re-encryption did not occur.
// Returns wasReEncrypted (bool) which is true when the value was
// re-encrypted with the active key.
// Returns err (error) when decryption fails.
func (s *cryptoService) DecryptAndReEncrypt(ctx context.Context, ciphertext string) (plaintext, newCiphertext string, wasReEncrypted bool, err error) {
	ctx, l := logger_domain.From(ctx, log)
	plaintext, err = s.Decrypt(ctx, ciphertext)
	if err != nil {
		return "", "", false, err
	}

	metadata, metaErr := extractCiphertextMetadata(ciphertext)
	if metaErr != nil {
		return plaintext, "", false, nil
	}

	if !s.isDeprecatedKey(metadata.KeyID) {
		return plaintext, "", false, nil
	}

	if !s.enableAutoReEncrypt {
		l.Trace("Data encrypted with deprecated key (auto re-encrypt disabled)",
			logger_domain.String("deprecated_key_id", metadata.KeyID),
			logger_domain.String("active_key_id", s.activeKeyID),
		)
		return plaintext, "", false, nil
	}

	newCiphertext, err = s.Encrypt(ctx, plaintext)
	if err != nil {
		l.Warn("Failed to re-encrypt data with active key",
			logger_domain.String("deprecated_key_id", metadata.KeyID),
			logger_domain.String("active_key_id", s.activeKeyID),
			logger_domain.Error(err),
		)
		return plaintext, "", false, nil
	}

	l.Trace("Data re-encrypted with active key",
		logger_domain.String("old_key_id", metadata.KeyID),
		logger_domain.String("new_key_id", s.activeKeyID),
	)

	provider, err2 := s.getProvider(ctx)
	providerType := "unknown"
	if err2 == nil {
		providerType = string(provider.Type())
	}

	cryptoOperationCount.Add(ctx, 1,
		metricAttributes(
			attributeKeyOperation, opAutoReencrypt,
			attributeKeyProvider, providerType,
			attributeKeyStatus, statusSuccess,
		),
	)

	return plaintext, newCiphertext, true, nil
}
