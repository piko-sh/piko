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
	"time"

	"piko.sh/piko/internal/crypto/crypto_dto"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
)

// EncryptStream encrypts a data stream using the specified key, or the active
// key if keyID is empty. The caller writes plaintext to the returned
// WriteCloser, and encrypted data is written to the provided output Writer.
//
// Designed for encrypting large files without loading them entirely into memory.
// Memory usage remains constant (O(chunk_size) ~64KB) regardless of stream size.
//
// The encrypted output uses the v2 streaming envelope format:
// [Version 0x02][Header Length][JSON Header][Chunked Encrypted Data]
// This format is distinct from the v1 string-based envelope format and is
// optimised for streaming large data.
//
// Takes output (io.Writer) which receives the encrypted data.
// Takes keyID (string) which specifies the encryption key, or empty for the
// active key.
//
// Returns io.WriteCloser which accepts plaintext data to be encrypted.
// Returns error when the encryption stream cannot be initialised.
func (s *cryptoService) EncryptStream(ctx context.Context, output io.Writer, keyID string) (io.WriteCloser, error) {
	ctx, l := logger_domain.From(ctx, log)
	startTime := time.Now()

	provider, err := s.getProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting provider for stream encryption: %w", err)
	}

	if keyID == "" {
		keyID = s.activeKeyID
	}

	request := &crypto_dto.EncryptRequest{
		Plaintext: "",
		KeyID:     keyID,
		Context:   nil,
	}

	writer, err := goroutine.SafeCall1(ctx, "crypto.EncryptStream", func() (io.WriteCloser, error) { return provider.EncryptStream(ctx, output, request) })
	if err != nil {
		cryptoOperationCount.Add(ctx, 1,
			metricAttributes(
				attributeKeyOperation, opEncryptStream,
				attributeKeyProvider, string(provider.Type()),
				attributeKeyStatus, statusError,
			),
		)
		return nil, crypto_dto.NewEncryptionError("EncryptStream", string(provider.Type()), keyID, err)
	}

	duration := time.Since(startTime).Milliseconds()
	l.Trace("Stream encryption initiated",
		logger_domain.Int64("init_duration_ms", duration),
		logger_domain.String("key_id", keyID),
		logger_domain.String(attributeKeyProvider, string(provider.Type())),
	)

	cryptoOperationCount.Add(ctx, 1,
		metricAttributes(
			attributeKeyOperation, opEncryptStream,
			attributeKeyProvider, string(provider.Type()),
			attributeKeyStatus, statusInitiated,
		),
	)

	return writer, nil
}

// DecryptStream decrypts a data stream.
//
// The caller reads plaintext from the returned ReadCloser. Automatically detects
// and parses the v2 streaming envelope format, extracting metadata and setting up
// the decryption pipeline. Suitable for decrypting large files with constant
// memory usage (O(chunk_size) ~64KB).
//
// Takes input (io.Reader) which provides the encrypted data stream.
//
// Returns io.ReadCloser which yields decrypted plaintext when read.
// Returns error when decryption fails or the stream format is invalid.
func (s *cryptoService) DecryptStream(ctx context.Context, input io.Reader) (io.ReadCloser, error) {
	ctx, l := logger_domain.From(ctx, log)
	startTime := time.Now()

	provider, err := s.getProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting provider for stream decryption: %w", err)
	}

	reader, err := goroutine.SafeCall1(ctx, "crypto.DecryptStream", func() (io.ReadCloser, error) { return provider.DecryptStream(ctx, input) })
	if err != nil {
		cryptoOperationCount.Add(ctx, 1,
			metricAttributes(
				attributeKeyOperation, opDecryptStream,
				attributeKeyProvider, string(provider.Type()),
				attributeKeyStatus, statusError,
			),
		)
		return nil, crypto_dto.NewEncryptionError("DecryptStream", string(provider.Type()), "", err)
	}

	duration := time.Since(startTime).Milliseconds()
	l.Trace("Stream decryption initiated",
		logger_domain.Int64("init_duration_ms", duration),
		logger_domain.String(attributeKeyProvider, string(provider.Type())),
	)

	cryptoOperationCount.Add(ctx, 1,
		metricAttributes(
			attributeKeyOperation, opDecryptStream,
			attributeKeyProvider, string(provider.Type()),
			attributeKeyStatus, statusInitiated,
		),
	)

	return reader, nil
}
