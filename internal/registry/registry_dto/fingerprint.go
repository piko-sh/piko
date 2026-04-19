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

package registry_dto

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"
)

// ComputeVariantFingerprint returns a stable hash of the inputs that determine a derived
// variant's validity.
//
// The hashed inputs are the source content hash, the transform capability, and the
// capability parameters (sorted so ordering is irrelevant; the parameters carry quality
// and the like). Two builds that produce a variant from identical inputs yield the same
// fingerprint, so a re-seed keeps the existing variant instead of regenerating it, and
// two releases whose inputs differ produce distinct fingerprints that coexist during a
// canary.
//
// It deliberately does NOT hash the capability's code version.
//
// Takes sourceContentHash (string) which is the content hash of the source asset.
// Takes profile (*DesiredProfile) which supplies the capability and its parameters.
//
// Returns string which is the hex-encoded SHA256 fingerprint of the inputs.
func ComputeVariantFingerprint(sourceContentHash string, profile *DesiredProfile) string {
	type parameter struct {
		key   string
		value string
	}
	parameters := make([]parameter, 0, profile.Params.Len())
	for key, value := range profile.Params.All() {
		parameters = append(parameters, parameter{key: key, value: value})
	}
	slices.SortFunc(parameters, func(first, second parameter) int {
		return strings.Compare(first.key, second.key)
	})

	var builder strings.Builder
	builder.WriteString(sourceContentHash)
	builder.WriteByte(0)
	builder.WriteString(profile.CapabilityName)
	builder.WriteByte(0)
	for _, parameter := range parameters {
		builder.WriteString(parameter.key)
		builder.WriteByte('=')
		builder.WriteString(parameter.value)
		builder.WriteByte(0)
	}

	sum := sha256.Sum256([]byte(builder.String()))
	return hex.EncodeToString(sum[:])
}
