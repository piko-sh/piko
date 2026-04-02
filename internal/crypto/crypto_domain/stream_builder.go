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
	"io"
)

// StreamEncryptBuilder provides a fluent interface for streaming encryption.
// It is designed for encrypting large files without loading them entirely into
// memory, keeping memory usage constant regardless of stream size.
type StreamEncryptBuilder struct {
	// service provides encryption operations for building streams.
	service CryptoServicePort

	// output is the destination writer for encrypted data.
	output io.Writer

	// keyID is the identifier for the encryption key to use.
	keyID string
}

// Output sets the writer that will receive the encrypted data.
//
// Takes output (io.Writer) which receives the encrypted data.
//
// Returns *StreamEncryptBuilder for method chaining.
func (b *StreamEncryptBuilder) Output(output io.Writer) *StreamEncryptBuilder {
	b.output = output
	return b
}

// KeyID sets the specific key ID to use for encryption.
// If not set, the service's active key will be used.
//
// Takes keyID (string) which identifies the encryption key to use.
//
// Returns *StreamEncryptBuilder for method chaining.
func (b *StreamEncryptBuilder) KeyID(keyID string) *StreamEncryptBuilder {
	b.keyID = keyID
	return b
}

// Stream executes the streaming encryption operation.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Takes ctx (context.Context) which controls the operation lifecycle.
//
// Returns io.WriteCloser which accepts plaintext to be encrypted.
// Returns error when the key cannot be found or encryption setup fails.
func (b *StreamEncryptBuilder) Stream(ctx context.Context) (io.WriteCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("stream encrypt context cancelled: %w", err)
	}
	return b.service.EncryptStream(ctx, b.output, b.keyID)
}

// StreamDecryptBuilder provides a fluent interface for streaming decryption.
// It decrypts large files without loading them fully into memory.
type StreamDecryptBuilder struct {
	// service provides the cryptographic operations for stream decryption.
	service CryptoServicePort

	// input is the encrypted data source to decrypt.
	input io.Reader
}

// Input sets the reader that provides the encrypted data.
//
// Takes input (io.Reader) which provides the encrypted data.
//
// Returns *StreamDecryptBuilder for method chaining.
func (b *StreamDecryptBuilder) Input(input io.Reader) *StreamDecryptBuilder {
	b.input = input
	return b
}

// Stream executes the streaming decryption operation.
//
// When the context is already cancelled or has exceeded its deadline, returns
// the context's error without performing any work.
//
// Returns io.ReadCloser which provides plaintext data as it is decrypted.
// Returns error when decryption setup fails.
func (b *StreamDecryptBuilder) Stream(ctx context.Context) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("stream decrypt context cancelled: %w", err)
	}
	return b.service.DecryptStream(ctx, b.input)
}
