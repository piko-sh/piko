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
	"fmt"

	"piko.sh/piko/internal/crypto/crypto_domain"
	"piko.sh/piko/internal/crypto/crypto_dto"
)

// Factory implements the LocalProviderFactory interface to create AES-GCM
// encryption providers with a given key for envelope encryption.
type Factory struct{}

// NewFactory creates a new factory for local AES-GCM providers.
//
// Returns *Factory which is the configured factory ready for use.
func NewFactory() *Factory {
	return &Factory{}
}

// CreateWithKey creates a new local AES-GCM provider configured with the given
// key. The key material is accessed within a scoped callback to minimise
// exposure.
//
// This implements crypto_domain.LocalProviderFactory.CreateWithKey.
//
// Takes key (*crypto_dto.SecureBytes) which contains the encryption key
// material.
// Takes keyID (string) which identifies the key for later reference.
//
// Returns crypto_domain.EncryptionProvider which is the configured provider.
// Returns error when the provider cannot be created with the given key.
func (*Factory) CreateWithKey(key *crypto_dto.SecureBytes, keyID string) (crypto_domain.EncryptionProvider, error) {
	var provider crypto_domain.EncryptionProvider

	err := key.WithAccess(func(keyBytes []byte) error {
		var createErr error
		provider, createErr = NewProvider(Config{
			Key:   keyBytes,
			KeyID: keyID,
		})
		return createErr
	})

	if err != nil {
		return nil, fmt.Errorf("creating AES-GCM provider with key %q: %w", keyID, err)
	}

	return provider, nil
}
