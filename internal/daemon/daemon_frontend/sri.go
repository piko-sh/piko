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

package daemon_frontend

import (
	"crypto/sha512"
	"encoding/base64"
	"sync"
)

var (
	sriEnabled bool

	sriHashes sync.Map
)

// SetSRIEnabled controls whether SRI integrity attributes are included in
// generated HTML. Enabled by default in production mode, disabled in
// development mode where assets change on every hot reload.
//
// Takes enabled (bool) which activates SRI when true.
func SetSRIEnabled(enabled bool) {
	sriEnabled = enabled
}

// IsSRIEnabled reports whether SRI integrity attributes should be emitted.
//
// Returns bool which is true when SRI is active.
func IsSRIEnabled() bool {
	return sriEnabled
}

// ComputeSRIHash computes a SHA-384 Subresource Integrity hash for the given
// content bytes. The returned string is in the format "sha384-<base64>" as
// specified by the W3C SRI specification.
//
// The hash must be computed on uncompressed content, since browsers decompress
// before verifying integrity.
//
// Takes content ([]byte) which is the uncompressed asset bytes to hash.
//
// Returns string which is the SRI hash in "sha384-<base64>" format.
func ComputeSRIHash(content []byte) string {
	h := sha512.Sum384(content)
	return "sha384-" + base64.StdEncoding.EncodeToString(h[:])
}

// SetSRIHash stores an SRI hash for the given asset path. Call this during
// asset initialisation or registration.
//
// Takes assetPath (string) which identifies the asset.
// Takes hash (string) which is the SRI hash to store.
func SetSRIHash(assetPath, hash string) {
	sriHashes.Store(assetPath, hash)
}

// GetSRIHash returns the SRI hash for the given asset path.
//
// Takes assetPath (string) which identifies the asset to look up.
//
// Returns string which is the SRI hash, or empty when SRI is disabled or no
// hash exists for the path.
func GetSRIHash(assetPath string) string {
	if !sriEnabled {
		return ""
	}
	if v, ok := sriHashes.Load(assetPath); ok {
		s, isString := v.(string)
		if !isString {
			return ""
		}
		return s
	}
	return ""
}

// FilterSRIHash returns the given hash when SRI is enabled, or an empty
// string when disabled.
//
// Takes hash (string) which is the SRI hash to filter.
//
// Returns string which is the hash unchanged when SRI is enabled, or empty
// when disabled.
func FilterSRIHash(hash string) string {
	if !sriEnabled {
		return ""
	}
	return hash
}

// ResetSRIState clears all cached SRI hashes and resets the enabled flag.
// This is intended for use in tests only.
func ResetSRIState() {
	sriEnabled = false
	sriHashes = sync.Map{}
}
