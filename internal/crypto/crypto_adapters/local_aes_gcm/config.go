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

package local_aes_gcm

// Config holds settings for the local AES-GCM encryption provider.
type Config struct {
	// KeyID is an optional name for this key, used for key rotation when multiple
	// keys exist. Defaults to "local-default" if empty.
	KeyID string

	// Key is the 32-byte encryption key for AES-256-GCM. Load this from a safe
	// source such as an environment variable or secrets manager.
	Key []byte
}

// validate checks that the configuration is valid.
//
// Returns error when the key size is incorrect.
func (c *Config) validate() error {
	if len(c.Key) != KeySize {
		return ErrInvalidKeySize
	}
	return nil
}
