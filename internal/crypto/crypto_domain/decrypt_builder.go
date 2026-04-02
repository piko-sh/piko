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

// DecryptBuilder provides a fluent interface for decrypting data.
type DecryptBuilder struct {
	// service provides cryptographic operations for decryption.
	service CryptoServicePort

	// ciphertext is the encrypted data to decrypt.
	ciphertext string
}

// Data sets the ciphertext to decrypt.
//
// Takes ciphertext (string) which is the encrypted data to decrypt.
//
// Returns *DecryptBuilder for method chaining.
func (b *DecryptBuilder) Data(ciphertext string) *DecryptBuilder {
	b.ciphertext = ciphertext
	return b
}

// Do executes the decryption operation.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Returns string which is the decrypted plaintext.
// Returns error when decryption fails.
func (b *DecryptBuilder) Do(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return b.service.Decrypt(ctx, b.ciphertext)
}
