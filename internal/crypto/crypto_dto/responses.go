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

// EncryptResponse represents the result of an encryption operation.
type EncryptResponse struct {
	// Ciphertext is the encrypted data as a base64-encoded string.
	Ciphertext string

	// KeyID identifies which key was used for encryption.
	KeyID string

	// Provider identifies which encryption provider was used.
	Provider ProviderType
}

// DecryptResponse represents the result of a decryption operation.
type DecryptResponse struct {
	// Plaintext is the decrypted data returned from the decryption operation.
	Plaintext string
}

// DataKey represents a data encryption key for envelope encryption.
type DataKey struct {
	// PlaintextKey is the unencrypted data encryption key held in secure
	// memory for local encryption and decryption. The caller must call
	// PlaintextKey.Close() when done.
	//
	// Uses secure memory (mmap+mlock on Unix, VirtualAlloc+VirtualLock on
	// Windows) to prevent garbage collector copying and disk swapping.
	PlaintextKey *SecureBytes

	// EncryptedKey is the data key encrypted with the master key.
	// It is stored alongside the encrypted data.
	EncryptedKey string

	// KeyID identifies the master key that encrypted this data key.
	KeyID string

	// Provider identifies which encryption provider created this key.
	Provider ProviderType
}

// Close releases the secure memory holding the plaintext key.
// This is a convenience method that calls PlaintextKey.Close if it is not nil.
//
// Returns error when the underlying PlaintextKey fails to close.
func (dk *DataKey) Close() error {
	if dk.PlaintextKey != nil {
		return dk.PlaintextKey.Close()
	}
	return nil
}
