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

// Package crypto provides the public API for encryption and key
// management in Piko.
//
// This is a facade that re-exports types from internal packages,
// providing a stable import path for application developers.
//
// # Usage
//
//	encrypted, err := crypto.Encrypt(ctx, "sensitive-data")
//	plaintext, err := crypto.Decrypt(ctx, encrypted)
//
// # Providers
//
// A local AES-GCM provider is included by default. Cloud KMS
// providers are available in the crypto_provider_* sub-packages.
//
// # Design rationale
//
// The service supports envelope encryption for batch operations,
// graceful key rotation with zero downtime, and automatic
// re-encryption of deprecated keys. Ciphertext envelopes are
// self-describing, carrying the metadata needed for decryption.
//
// # Batch operations
//
// For encrypting large datasets efficiently:
//
//	tokens := []string{"token1", "token2", /* ... */ "token1000"}
//	encrypted, err := cryptoService.EncryptBatch(ctx, tokens)
//
// # Streaming encryption
//
// For encrypting large files without loading them into memory:
//
//	writer, err := cryptoService.EncryptStream(
//		ctx, outputFile, "key-id",
//	)
//	if err != nil { return err }
//	defer writer.Close()
//
//	_, err = io.Copy(writer, largeInputFile)
//	// Memory usage: O(64KB) regardless of file size
//
// Decrypting streams:
//
//	reader, err := cryptoService.DecryptStream(ctx, encryptedFile)
//	if err != nil { return err }
//	defer reader.Close()
//
//	plaintext, err := io.ReadAll(reader)
//
// # Key rotation
//
// Zero-downtime key rotation:
//
//	err := cryptoService.RotateKey(ctx, "old-key-id", "new-key-id")
//	// All new encryptions use new-key-id
//	// Old data can still be decrypted
//
// # Thread safety
//
// The crypto service and its methods are safe for concurrent use.
// Streaming readers and writers are NOT safe for concurrent use;
// each stream should be used from a single goroutine.
package crypto
