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

package crypto_streaming

import (
	"crypto/cipher"
	"io"

	"piko.sh/piko/internal/crypto/crypto_adapters/local_aes_gcm"
	"piko.sh/piko/internal/crypto/crypto_dto"
)

const (
	// IVSize is the size of the GCM initialisation vector in bytes (12 bytes).
	// This is the standard size for AES-GCM as per NIST guidance.
	IVSize = local_aes_gcm.IVSize

	// DefaultChunkSize is the default chunk size for streaming encryption (64KB).
	// This value balances memory use with encryption overhead.
	DefaultChunkSize = crypto_dto.DefaultChunkSize
)

// StreamingHeader contains metadata for streaming encryption envelopes.
// It is written at the start of encrypted streams to allow decryption
// without out-of-band metadata.
type StreamingHeader = crypto_dto.StreamingHeader

// GenerateIV creates a random initialisation vector for AES-GCM encryption.
//
// Returns []byte which contains the generated IV of IVSize bytes.
// Returns error when the random source cannot be read.
func GenerateIV() ([]byte, error) {
	return local_aes_gcm.GenerateIV()
}

// WriteStreamingHeader writes the v2 streaming envelope header to the output.
// Format: [Version (1 byte)] [Header Length (4 bytes)] [JSON Header].
//
// Used by cloud KMS providers to write consistent envelope headers that
// include the encrypted data key and other metadata.
//
// Takes output (io.Writer) which receives the encoded header bytes.
// Takes header (*StreamingHeader) which contains the envelope metadata to
// encode.
//
// Returns error when the header cannot be marshalled to JSON or when writing
// to the output fails.
func WriteStreamingHeader(output io.Writer, header *StreamingHeader) error {
	return local_aes_gcm.WriteStreamingHeader(output, header)
}

// ReadStreamingHeader reads and parses the v2 streaming envelope header.
//
// Cloud KMS providers use it to read envelope headers and extract the
// encrypted data key and other metadata.
//
// Takes input (io.Reader) which provides the encrypted stream to read from.
//
// Returns *StreamingHeader which contains the parsed header data.
// Returns error when the format is invalid, the version is unsupported, or
// the header cannot be read.
func ReadStreamingHeader(input io.Reader) (*StreamingHeader, error) {
	return local_aes_gcm.ReadStreamingHeader(input)
}

// NewEncryptingWriter creates a new encrypting writer for streaming encryption.
//
// This is used by cloud KMS providers to perform local AES-GCM streaming
// encryption with their own envelope encryption (where the DEK is encrypted by
// the cloud KMS).
//
// Takes destination (io.Writer) which receives the encrypted output.
// Takes aead (cipher.AEAD) which is the AES-GCM cipher instance initialised
// with the data encryption key.
// Takes baseIV ([]byte) which is the base IV for the stream (must be IVSize
// bytes).
// Takes chunkSize (int) which is the size of each plaintext chunk in bytes.
//
// Returns io.WriteCloser which encrypts data written to it.
//
// Example:
//
//	// After generating/decrypting a DEK from KMS
//	block, _ := aes.NewCipher(plaintextDEK)
//	aead, _ := cipher.NewGCM(block)
//	baseIV, _ := crypto_streaming.GenerateIV()
//
//	writer := crypto_streaming.NewEncryptingWriter(output, aead, baseIV, crypto_streaming.DefaultChunkSize)
//	defer writer.Close()
//	io.Copy(writer, plaintext)
func NewEncryptingWriter(destination io.Writer, aead cipher.AEAD, baseIV []byte, chunkSize int) io.WriteCloser {
	return local_aes_gcm.NewEncryptingWriter(destination, aead, baseIV, chunkSize)
}

// NewDecryptingReader creates a new decrypting reader for streaming
// decryption.
//
// This is used by cloud KMS providers to perform local AES-GCM streaming
// decryption with their own envelope encryption (where the DEK is decrypted
// by the cloud KMS).
//
// Takes source (io.Reader) which provides the encrypted input stream (after the
// header has been read).
// Takes aead (cipher.AEAD) which is the AES-GCM cipher instance initialised
// with the data encryption key.
// Takes baseIV ([]byte) which is the base IV for the stream (must be IVSize
// bytes).
//
// Returns io.ReadCloser which decrypts data as it is read.
//
// Example:
//
//	// After reading header and decrypting DEK from KMS
//	block, _ := aes.NewCipher(plaintextDEK)
//	aead, _ := cipher.NewGCM(block)
//	baseIV, _ := base64.StdEncoding.DecodeString(header.IV)
//
//	reader := crypto_streaming.NewDecryptingReader(input, aead, baseIV)
//	defer reader.Close()
//	plaintext, _ := io.ReadAll(reader)
func NewDecryptingReader(source io.Reader, aead cipher.AEAD, baseIV []byte) io.ReadCloser {
	return local_aes_gcm.NewDecryptingReader(source, aead, baseIV)
}
