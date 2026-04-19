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

package crypto_adapters

import (
	"cmp"
	"encoding/base64"
	"fmt"

	"piko.sh/piko/internal/crypto/crypto_adapters/local_aes_gcm"
	"piko.sh/piko/internal/crypto/crypto_domain"
)

// CreateProviderFromBase64Key creates a local AES-GCM provider from a
// base64-encoded key. This is a convenience function for loading keys from
// environment variables.
//
// Used only internally by the bootstrap container for config-based initialisation.
// For modern applications, prefer using the option-based approach:
// import "piko.sh/piko/wdk/crypto/crypto_provider_local_aes_gcm"
// keyBytes, _ := base64.StdEncoding.DecodeString(base64Key)
// provider, _ := crypto_provider_local_aes_gcm.NewProvider(
//
//	crypto_provider_local_aes_gcm.Config{
//		Key:   keyBytes,
//		KeyID: keyID,
//	},
//
// )
//
// Takes base64Key (string) which is the base64-encoded 32-byte
// AES key.
// Takes keyID (string) which identifies the key; defaults to
// "local-default" if empty.
//
// Returns crypto_domain.EncryptionProvider which is the configured AES-GCM
// encryption provider.
// Returns error when the base64 decoding fails or the decoded key is not
// exactly 32 bytes.
func CreateProviderFromBase64Key(base64Key string, keyID string) (crypto_domain.EncryptionProvider, error) {
	key, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 key: %w", err)
	}

	if len(key) != local_aes_gcm.KeySize {
		return nil, fmt.Errorf("decoded key must be exactly 32 bytes, got %d bytes", len(key))
	}

	keyID = cmp.Or(keyID, "local-default")

	return local_aes_gcm.NewProvider(local_aes_gcm.Config{
		Key:   key,
		KeyID: keyID,
	})
}
