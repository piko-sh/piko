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

package crypto_dto

const (
	// StreamingEnvelopeVersion is the version byte for the v2 streaming format.
	StreamingEnvelopeVersion byte = 0x02

	// DefaultChunkSize is the size of each plaintext chunk for streaming
	// encryption (64KB).
	//
	// It provides a good balance between memory usage and performance.
	// Each chunk will be slightly larger when encrypted due to the GCM
	// authentication tag.
	DefaultChunkSize = 64 * 1024
)

// StreamingHeader represents the metadata at the beginning of a v2
// streaming ciphertext.
//
// The header is written as:
// [Version (1 byte)] [Header Length (4 bytes)] [JSON-encoded StreamingHeader]
// The JSON encoding allows for extensibility and human-readability
// during debugging.
type StreamingHeader struct {
	// KeyID is the identifier of the master encryption key used.
	KeyID string `json:"key_id"`

	// Provider identifies which encryption provider was used (e.g.,
	// "local_aes_gcm", "aws_kms").
	Provider string `json:"provider"`

	// IV is the base64-encoded initialisation vector for the stream. Per-chunk
	// IVs are derived from this base IV during AES-GCM streaming decryption.
	IV string `json:"iv"`

	// EncryptedDataKey is the encrypted data key, encoded in Base64. Only present
	// when using envelope encryption (AWS KMS, GCP KMS); empty for local_aes_gcm.
	EncryptedDataKey string `json:"edk,omitempty"`

	// Algorithm specifies the encryption method used (e.g., "AES-256-GCM").
	Algorithm string `json:"algorithm,omitempty"`

	// Version is the format version number, always 2 for streaming format.
	// This is also stored in the version byte but is included here for ease of use.
	Version int `json:"version"`
}
