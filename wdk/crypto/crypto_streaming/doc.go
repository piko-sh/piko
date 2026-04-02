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

// Package crypto_streaming provides streaming encryption primitives
// for envelope encryption with constant memory usage.
//
// This package exposes the low-level AES-GCM streaming readers and
// writers used by cloud KMS providers (AWS KMS, GCP KMS) to implement
// envelope encryption. Most applications should use the high-level
// streaming methods on the crypto service (EncryptStream,
// DecryptStream) rather than these primitives directly.
//
// # Envelope encryption pattern
//
// Cloud KMS providers use envelope encryption to minimise API calls:
//
//  1. Generate or retrieve a data encryption key (DEK) from the KMS
//  2. Use the DEK with local AES-GCM streaming for encryption
//  3. Store the encrypted DEK alongside the encrypted data
//
// This reduces KMS calls from O(chunks) to O(1), keeping costs
// low and throughput high for large files.
//
// # Usage
//
// Writing an encrypted stream:
//
//	block, _ := aes.NewCipher(plaintextDEK)
//	aead, _ := cipher.NewGCM(block)
//	baseIV, _ := crypto_streaming.GenerateIV()
//
//	writer := crypto_streaming.NewEncryptingWriter(
//	    output, aead, baseIV, crypto_streaming.DefaultChunkSize,
//	)
//	defer writer.Close()
//	io.Copy(writer, plaintext)
//
// Reading a decrypted stream:
//
//	header, _ := crypto_streaming.ReadStreamingHeader(input)
//	baseIV, _ := base64.StdEncoding.DecodeString(header.IV)
//
//	reader := crypto_streaming.NewDecryptingReader(
//	    input, aead, baseIV,
//	)
//	defer reader.Close()
//	plaintext, _ := io.ReadAll(reader)
//
// # Memory usage
//
// The streaming readers and writers use O(chunk_size) memory (~64KB
// by default), regardless of the total data size. This makes them
// suitable for multi-GB files.
//
// # Thread safety
//
// The streaming readers and writers are NOT safe for concurrent use.
// Each stream should be used from a single goroutine.
package crypto_streaming
