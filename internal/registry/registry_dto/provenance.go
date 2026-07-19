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
	"errors"
	"slices"
	"strconv"
	"strings"
)

// VariantProducer records who produced a variant's bytes.
//
// It is one of the two orthogonal provenance axes, separate from VariantKind. A single
// four-valued enum would force every "who made it" test to enumerate two of four
// constants and vice versa; keeping the axes apart lets each be tested for what it means.
type VariantProducer uint8

const (
	// ProducerUnknown is the zero value and is never valid on a persisted variant.
	ProducerUnknown VariantProducer = 0

	// ProducerBuild marks bytes produced by generation (CI or `piko build`).
	ProducerBuild VariantProducer = 1

	// ProducerRuntime marks bytes produced by the running server (an upload, an on-demand
	// variant, or the page cache).
	ProducerRuntime VariantProducer = 2
)

// VariantKind records whether a variant has a parent, and therefore whether it is
// regenerable cache or irreplaceable data.
type VariantKind uint8

const (
	// KindUnknown is the zero value and is never valid on a persisted variant.
	KindUnknown VariantKind = 0

	// KindSource marks a variant with no parent (an upload or a build input). It is data.
	KindSource VariantKind = 1

	// KindDerived marks a variant produced by a transform from a parent variant. It is
	// cache.
	KindDerived VariantKind = 2
)

const (
	// fingerprintFormat tags the fingerprint preimage layout. Bumping it invalidates every
	// derived variant, so change it only when the preimage encoding itself changes.
	fingerprintFormat = "pikofp/2"
)

var (
	// ErrFingerprintNoParent is returned when a transform has no parent content hash.
	ErrFingerprintNoParent = errors.New("variant fingerprint: parent content hash is empty")

	// ErrFingerprintNoCapability is returned when a transform names no capability.
	ErrFingerprintNoCapability = errors.New("variant fingerprint: capability name is empty")

	// ErrFingerprintNoVersion is returned when a transform carries no capability version.
	ErrFingerprintNoVersion = errors.New("variant fingerprint: capability version is zero")
)

// VariantTransform describes exactly how a derived variant was produced. It is the
// preimage of a derived variant's input fingerprint, and it is zero for a source variant.
//
// The fields are ordered for struct-field alignment (CapabilityVersion trails the pointer
// fields), not for readability; the JSON tags fix the wire order, so reordering is safe.
type VariantTransform struct {
	// ParentVariantID is the variant this one was derived from. It is the profile's
	// DependsOn target, which is not necessarily "source": a compression profile depends on
	// another derived variant.
	ParentVariantID string `json:"parentVariantId"`

	// ParentContentHash is the parent's content hash captured at derivation time. A mismatch
	// against the parent's current hash is what makes this variant stale.
	ParentContentHash string `json:"parentContentHash"`

	// CapabilityName is the transform that ran (for example "image-transform").
	CapabilityName string `json:"capabilityName"`

	// Params are the parameters actually passed to the capability, not the declared profile
	// parameters.
	Params ProfileParams `json:"params"`

	// CapabilityVersion is the output version of that transform.
	CapabilityVersion uint32 `json:"capabilityVersion"`
}

// IsZero reports whether the transform carries no derivation information, which is the
// case for a source variant.
//
// Returns bool which is true when the transform is empty.
//
//nolint:gocritic // IsZero must keep a value receiver so encoding/json omitzero finds it.
func (t VariantTransform) IsZero() bool {
	return t.ParentVariantID == "" && t.ParentContentHash == "" && t.CapabilityName == "" &&
		t.CapabilityVersion == 0 && t.Params.Len() == 0
}

// Fingerprint hashes everything that determines a derived variant's validity: the
// parent's content, the transform, the transform's code version, and the parameters
// actually applied.
//
// Returns string which is the hex-encoded SHA-256 of the transform preimage.
// Returns error when a parent content hash, capability name or version is missing.
func (t *VariantTransform) Fingerprint() (string, error) {
	if t.ParentContentHash == "" {
		return "", ErrFingerprintNoParent
	}
	if t.CapabilityName == "" {
		return "", ErrFingerprintNoCapability
	}
	if t.CapabilityVersion == 0 {
		return "", ErrFingerprintNoVersion
	}

	type parameter struct {
		key   string
		value string
	}
	parameters := make([]parameter, 0, t.Params.Len())
	for key, value := range t.Params.All() {
		parameters = append(parameters, parameter{key: key, value: value})
	}
	slices.SortFunc(parameters, func(a, b parameter) int { return strings.Compare(a.key, b.key) })

	var builder strings.Builder
	builder.WriteString(fingerprintFormat)
	builder.WriteByte(0)
	builder.WriteString(t.ParentContentHash)
	builder.WriteByte(0)
	builder.WriteString(t.CapabilityName)
	builder.WriteByte(0)
	builder.WriteString(strconv.FormatUint(uint64(t.CapabilityVersion), 10))
	builder.WriteByte(0)
	for _, p := range parameters {
		builder.WriteString(p.key)
		builder.WriteByte('=')
		builder.WriteString(p.value)
		builder.WriteByte(0)
	}

	sum := sha256.Sum256([]byte(builder.String()))
	return hex.EncodeToString(sum[:]), nil
}

// SourceFingerprint is the fingerprint of a source variant: its own content hash. A
// source has no inputs beyond itself and is never stale.
//
// Takes contentHash (string) which is the source variant's content hash.
//
// Returns string which is the source fingerprint.
func SourceFingerprint(contentHash string) string {
	return contentHash
}
