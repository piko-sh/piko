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

// EncryptRequest represents a request to encrypt plaintext data.
// It is used by encryption adapters including local AES-GCM and cloud KMS.
type EncryptRequest struct {
	// Context provides additional authenticated data (AAD) for encryption.
	// This data is authenticated but not encrypted and must be provided
	// during decryption; optional, used for extra security context.
	Context map[string]string

	// Plaintext is the data to encrypt; must not be empty.
	Plaintext string

	// KeyID optionally specifies which key to use for encryption; if empty, the
	// service's active key is used.
	KeyID string
}

// DecryptRequest represents a request to decrypt ciphertext.
type DecryptRequest struct {
	// Context provides optional extra data that was used during encryption.
	// Must match the value given when encrypting; leave empty if none was used.
	Context map[string]string

	// Ciphertext is the encrypted data to decrypt, encoded as base64.
	Ciphertext string

	// KeyID optionally hints which key was used for encryption; if empty, the
	// service extracts the key ID from the ciphertext envelope.
	KeyID string
}

// GenerateDataKeyRequest specifies parameters for generating a new data
// encryption key. Used for envelope encryption patterns where a master key
// encrypts many data keys.
type GenerateDataKeyRequest struct {
	// KeyID specifies the master key to use for encrypting the data key. If empty,
	// the provider's default key is used.
	KeyID string

	// KeySpec specifies the type of data key to generate.
	// Common values: "AES_256" (default), "AES_128".
	KeySpec string
}
