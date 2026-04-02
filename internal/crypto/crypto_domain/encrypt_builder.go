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
)

// EncryptBuilder provides a fluent interface for encrypting data.
type EncryptBuilder struct {
	// service provides encryption operations.
	service CryptoServicePort

	// plaintext is the data to encrypt.
	plaintext string

	// keyID is the optional encryption key identifier; empty uses the default key.
	keyID string
}

// Data sets the plaintext to encrypt.
//
// Takes plaintext (string) which is the data to encrypt.
//
// Returns *EncryptBuilder for method chaining.
func (b *EncryptBuilder) Data(plaintext string) *EncryptBuilder {
	b.plaintext = plaintext
	return b
}

// KeyID sets the specific key ID to use for encryption.
// If not set, the service's active key will be used.
//
// Takes keyID (string) which identifies the encryption key to use.
//
// Returns *EncryptBuilder for method chaining.
func (b *EncryptBuilder) KeyID(keyID string) *EncryptBuilder {
	b.keyID = keyID
	return b
}

// Do executes the encryption operation.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Returns string which is the encrypted ciphertext.
// Returns error when encryption fails.
func (b *EncryptBuilder) Do(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if b.keyID != "" {
		return b.service.EncryptWithKey(ctx, b.plaintext, b.keyID)
	}
	return b.service.Encrypt(ctx, b.plaintext)
}
