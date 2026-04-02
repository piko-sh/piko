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
	"encoding/base64"
	"fmt"

	"piko.sh/piko/internal/json"
)

// envelopeData holds the envelope format for encrypted data.
// It provides the details needed for decryption and key rotation.
type envelopeData struct {
	// KeyID is the identifier of the encryption key used for this envelope.
	KeyID string `json:"key_id"`

	// Provider is the encryption provider that sealed this envelope.
	Provider string `json:"provider"`

	// EncryptedDataKey holds the wrapped data key used for envelope encryption.
	EncryptedDataKey string `json:"edk,omitempty"`

	// Ciphertext is the encrypted data payload.
	Ciphertext string `json:"ct"`

	// Version is the envelope format version; currently only version 1 is supported.
	Version int `json:"v"`
}

// ciphertextMetadata holds the parts extracted from a ciphertext envelope.
type ciphertextMetadata struct {
	// KeyID is the identifier of the encryption key used to encrypt this data.
	KeyID string

	// Provider identifies which encryption service was used.
	Provider string

	// Ciphertext is the encrypted data to be decrypted.
	Ciphertext string

	// EncryptedDataKey is the encrypted data key used for envelope encryption.
	EncryptedDataKey string
}

// createEnvelopedCiphertext wraps ciphertext with metadata in a base64-encoded
// envelope.
//
// Takes keyID (string) which identifies the encryption key used.
// Takes provider (string) which specifies the key provider name.
// Takes ciphertext (string) which contains the encrypted data.
// Takes encryptedDataKey (string) which holds the wrapped data key.
//
// Returns string which is the base64-encoded JSON envelope.
// Returns error when the envelope cannot be marshalled to JSON.
func createEnvelopedCiphertext(keyID, provider, ciphertext, encryptedDataKey string) (string, error) {
	envelope := &envelopeData{
		KeyID:            keyID,
		Provider:         provider,
		EncryptedDataKey: encryptedDataKey,
		Ciphertext:       ciphertext,
		Version:          1,
	}

	envelopeJSON, err := json.Marshal(envelope)
	if err != nil {
		return "", fmt.Errorf("failed to marshal envelope: %w", err)
	}

	return base64.StdEncoding.EncodeToString(envelopeJSON), nil
}

// extractCiphertextMetadata extracts metadata from the ciphertext envelope.
//
// Takes envelopedCiphertext (string) which is the base64-encoded envelope.
//
// Returns *ciphertextMetadata which contains the extracted key ID, provider,
// ciphertext, and encrypted data key.
// Returns error when the envelope cannot be decoded, parsed, or has an
// unsupported version.
func extractCiphertextMetadata(envelopedCiphertext string) (*ciphertextMetadata, error) {
	envelopeJSON, err := base64.StdEncoding.DecodeString(envelopedCiphertext)
	if err != nil {
		return nil, fmt.Errorf("failed to decode envelope: %w", err)
	}

	var env envelopeData
	if err := json.Unmarshal(envelopeJSON, &env); err != nil {
		return nil, fmt.Errorf("failed to parse envelope: %w", err)
	}

	if env.Version != 1 {
		return nil, fmt.Errorf("unsupported envelope version: %d", env.Version)
	}

	return &ciphertextMetadata{
		KeyID:            env.KeyID,
		Provider:         env.Provider,
		Ciphertext:       env.Ciphertext,
		EncryptedDataKey: env.EncryptedDataKey,
	}, nil
}
