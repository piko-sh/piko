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
	"fmt"
)

// BatchEncryptBuilder provides a fluent interface for batch encryption.
// It uses envelope encryption to reduce provider calls (for example, one KMS
// call instead of many).
type BatchEncryptBuilder struct {
	// service provides encryption operations for processing batch requests.
	service CryptoServicePort

	// plaintexts holds the strings to be encrypted in the batch operation.
	plaintexts []string
}

// Items sets the plaintexts to encrypt.
//
// Takes plaintexts ([]string) which contains the values to encrypt.
//
// Returns *BatchEncryptBuilder for method chaining.
func (b *BatchEncryptBuilder) Items(plaintexts []string) *BatchEncryptBuilder {
	b.plaintexts = plaintexts
	return b
}

// Do executes the batch encryption operation.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Returns []string which contains the encrypted values.
// Returns error when encryption fails.
func (b *BatchEncryptBuilder) Do(ctx context.Context) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("batch encryption context cancelled before execution: %w", err)
	}
	return b.service.EncryptBatch(ctx, b.plaintexts)
}
