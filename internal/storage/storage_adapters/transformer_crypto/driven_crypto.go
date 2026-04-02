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

package transformer_crypto

import (
	"context"
	"fmt"
	"io"

	"piko.sh/piko/internal/contextaware"
	"piko.sh/piko/internal/crypto/crypto_domain"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
)

const (
	// defaultPriorityEncryption is the default priority for encryption
	// transformers. Recommended: 250 for encryption transformers (after
	// compression at 100).
	defaultPriorityEncryption = 250
)

// CryptoTransformer implements StreamTransformerPort by delegating
// encryption and decryption to the centralised crypto service.
//
// This implementation uses streaming encryption with constant memory usage
// (O(chunk_size) ~64KB) regardless of file size, so it handles
// multi-GB files efficiently.
//
// Features:
//   - Constant memory footprint regardless of file size
//   - Centralised key management and key rotation support
//   - Chunked AES-256-GCM encryption (64KB chunks)
//   - Automatic envelope encryption for cloud KMS providers
//   - v2 streaming envelope format with metadata
type CryptoTransformer struct {
	// cryptoService handles stream encryption and decryption.
	cryptoService crypto_domain.CryptoServicePort

	// name is the identifier for this transformer.
	name string

	// priority is the execution order; lower values run first.
	priority int
}

var _ storage_domain.StreamTransformerPort = (*CryptoTransformer)(nil)

// Config holds configuration for the crypto-service storage transformer.
type Config struct {
	// CryptoService is the cryptographic service to use. Required.
	CryptoService crypto_domain.CryptoServicePort

	// Name is the unique identifier for this transformer instance.
	// Default: "crypto-service".
	Name string

	// Priority determines execution order where lower values run first.
	// Default is 250; recommended for encryption transformers (after compression
	// at 100).
	Priority int
}

// Name returns the transformer's name.
//
// Returns string which is the name that identifies this transformer.
func (t *CryptoTransformer) Name() string {
	return t.name
}

// Type returns the transformer type (encryption).
//
// Returns storage_dto.TransformerType which identifies this as an encryption
// transformer.
func (*CryptoTransformer) Type() storage_dto.TransformerType {
	return storage_dto.TransformerEncryption
}

// Priority returns the execution priority.
//
// Returns int which is the transformer's position in the processing order.
func (t *CryptoTransformer) Priority() int {
	return t.priority
}

// Transform encrypts the input stream using the crypto service. This is called
// during upload operations.
//
// Implementation: Uses streaming encryption with constant memory usage
// (O(chunk_size) ~64KB) regardless of file size. This enables efficient
// encryption of multi-GB files.
//
// The method uses an io.Pipe to bridge the WriteCloser streaming API to the
// io.Reader interface required by the transformer port. Encryption happens
// asynchronously in a goroutine as data is read from the returned reader.
//
// Takes input (io.Reader) which provides the plaintext data to encrypt.
//
// Returns io.Reader which provides the encrypted data stream.
// Returns error when the encryption stream cannot be initialised.
//
// Safe for concurrent use. The spawned goroutine runs until all
// input data has been encrypted or an error occurs.
func (t *CryptoTransformer) Transform(ctx context.Context, input io.Reader, _ any) (io.Reader, error) {
	pipeReader, pipeWriter := io.Pipe()

	go func() {
		encryptingWriter, err := t.cryptoService.EncryptStream(ctx, pipeWriter, "")
		if err != nil {
			_ = pipeWriter.CloseWithError(fmt.Errorf("failed to create encrypting stream: %w", err))
			return
		}

		_, copyErr := io.Copy(encryptingWriter, contextaware.NewReader(ctx, input))
		if copyErr != nil {
			_ = encryptingWriter.Close()
			_ = pipeWriter.CloseWithError(fmt.Errorf("encryption copy failed: %w", copyErr))
			return
		}

		if closeErr := encryptingWriter.Close(); closeErr != nil {
			_ = pipeWriter.CloseWithError(fmt.Errorf("failed to close encrypting writer: %w", closeErr))
			return
		}

		_ = pipeWriter.Close()
	}()

	return pipeReader, nil
}

// Reverse decrypts the input stream using the crypto service. This is called
// during download/read operations.
//
// Implementation: Uses streaming decryption with constant memory usage
// (O(chunk_size) ~64KB) regardless of file size. This enables efficient
// decryption of multi-GB files.
//
// The returned reader is actually an io.ReadCloser. Callers should close it
// when done to release resources, though this is optional for read operations.
//
// Takes input (io.Reader) which provides the encrypted data to decrypt.
//
// Returns io.Reader which provides the decrypted data stream.
// Returns error when the decryption stream cannot be created.
func (t *CryptoTransformer) Reverse(ctx context.Context, input io.Reader, _ any) (io.Reader, error) {
	decryptingReader, err := t.cryptoService.DecryptStream(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create decrypting stream: %w", err)
	}

	return decryptingReader, nil
}

// New creates a new crypto-service storage transformer.
//
// Takes cryptoService (CryptoServicePort) which provides encryption and
// decryption operations. This parameter is required.
// Takes name (string) which identifies the transformer. Defaults to
// "crypto-service" if empty.
// Takes priority (int) which sets the transformer priority. Defaults to 250
// if zero.
//
// Returns *CryptoTransformer which is ready for use with storage operations.
func New(cryptoService crypto_domain.CryptoServicePort, name string, priority int) *CryptoTransformer {
	if name == "" {
		name = "crypto-service"
	}
	if priority == 0 {
		priority = defaultPriorityEncryption
	}

	return &CryptoTransformer{
		cryptoService: cryptoService,
		name:          name,
		priority:      priority,
	}
}
