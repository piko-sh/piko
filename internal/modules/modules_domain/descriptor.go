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

package modules_domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
)

const (
	// DescriptorVersion is the current ModuleDescriptor schema version.
	//
	// Bumped on any breaking change to field shape. Hosts that load a descriptor with an
	// unrecognised version surface a clear migration error rather than silently
	// misinterpreting fields.
	DescriptorVersion = 1
)

// ModuleDescriptor declares what a module IS: identity, declared capabilities, pinned
// stdlib version, exported symbol allowlist, and named entrypoints.
//
// Descriptors are part of a ModuleBundle and are content-hashed to derive bundle
// identity. Two descriptors that normalise to the same canonical JSON produce the same
// hash; hosts use this to detect tampering and to deduplicate equivalent bundles.
type ModuleDescriptor struct {
	// Entrypoints names well-known entry functions by role.
	//
	// Common keys: "main" (when the module is itself an executable), "init" (initialisation
	// hook). Other keys are host-defined. Values are the qualified function names within the
	// module.
	Entrypoints map[string]string `json:"entrypoints,omitempty"`

	// Annotations carry host-specific metadata content-hashed with the descriptor.
	Annotations map[string]string `json:"annotations,omitempty"`

	// Ref is the canonical reference for this module: the (path, version, pin) identity. Pin
	// in a descriptor is the integrity hash of the bundle itself once finalised; left empty
	// during descriptor construction and filled in by the bundler.
	Ref ModuleRef `json:"ref"`

	// StdlibVersion pins which piko stdlib symbol surface the module was compiled against.
	// Hosts with a newer stdlib accept it when symbols remain compatible; mismatch surfaces
	// a clear load-time error.
	StdlibVersion string `json:"stdlib_version,omitempty"`

	// Capabilities declares the operations the module performs that require host approval.
	// Subset semantics: a hosting policy must declare a superset of this set or refuse to
	// load the module.
	Capabilities CapabilitySet `json:"capabilities,omitempty"`

	// SymbolAllowlist enumerates the package-qualified symbols.
	//
	// Empty means "all top-level exported identifiers": the default Go visibility rule.
	// Hosts can enforce stricter visibility by passing a non-empty list at packaging time.
	SymbolAllowlist []string `json:"symbol_allowlist,omitempty"`

	// SchemaVersion identifies the descriptor's wire-format version. Always
	// DescriptorVersion for descriptors produced by this piko version; older hosts reject
	// bundles with newer versions.
	SchemaVersion int `json:"schema_version"`
}

// Validate reports schema and consistency errors that piko detects without access to the
// bundle's bytecode. Used by the bundler before signing, and by the loader before
// piko.LoadModule dispatches.
//
// Returns nil when the descriptor is well-formed.
func (d *ModuleDescriptor) Validate() error {
	if d == nil {
		return errors.New("modules_domain: nil descriptor")
	}
	if d.SchemaVersion == 0 {
		return errors.New("modules_domain: descriptor missing schema_version")
	}
	if d.SchemaVersion > DescriptorVersion {
		return fmt.Errorf("modules_domain: descriptor schema_version %d newer than supported %d", d.SchemaVersion, DescriptorVersion)
	}
	if d.Ref.Path == "" {
		return errors.New("modules_domain: descriptor missing ref.path")
	}
	for _, capability := range d.Capabilities {
		if capability.Axis == "" {
			return errors.New("modules_domain: capability with empty axis")
		}
	}
	for key, value := range d.Entrypoints {
		if key == "" {
			return fmt.Errorf("modules_domain: entrypoint with empty key (value=%q)", value)
		}
		if value == "" {
			return fmt.Errorf("modules_domain: entrypoint %q maps to empty function name", key)
		}
	}
	return nil
}

// Canonicalise returns a copy with deterministic ordering.
//
// Sorted capabilities, sorted symbol allowlist, and sorted annotation keys are produced
// so two equivalent descriptors hash to the same bytes regardless of construction order.
//
// ModuleRef.Pin is zeroed in the canonical form: the pin is the fingerprint of the bundle
// (descriptor + bytecode + types-export) and so cannot contribute to its own input
// without creating a self-referential hash. Producers compute the fingerprint with Pin
// empty, then store it back on the descriptor; verifiers recompute the same way.
//
// The receiver is left unchanged.
//
// Returns *ModuleDescriptor which is the canonical copy.
func (d *ModuleDescriptor) Canonicalise() *ModuleDescriptor {
	out := *d
	out.Ref.Pin = ""
	out.Capabilities = d.Capabilities.Normalise()
	if len(d.SymbolAllowlist) > 0 {
		symbols := make([]string, len(d.SymbolAllowlist))
		copy(symbols, d.SymbolAllowlist)
		slices.Sort(symbols)
		out.SymbolAllowlist = symbols
	}
	return &out
}

// MarshalCanonicalJSON serialises the descriptor to deterministic JSON: canonicalised,
// key-sorted, indent-free, no trailing newline. The output is the wire form used for
// content hashing and for embedding in ModuleBundle.
//
// Returns the canonical bytes or an encoding error.
func (d *ModuleDescriptor) MarshalCanonicalJSON() ([]byte, error) {
	canonical := d.Canonicalise()
	return json.Marshal(canonical)
}

// ContentDigest returns the SHA-256 digest of the descriptor's canonical JSON form, in
// "sha256:<64-hex>" format. Used as the Pin component of ModuleRef in lockfiles and audit
// logs.
//
// Returns the digest string and any marshalling error.
func (d *ModuleDescriptor) ContentDigest() (string, error) {
	bytes, err := d.MarshalCanonicalJSON()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(bytes)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// UnmarshalDescriptor decodes a canonical-JSON descriptor and validates its schema. Used
// at module-load time by every ModuleProvider implementation.
//
// Takes data ([]byte) which is the canonical JSON.
//
// Returns the decoded descriptor or a decode/validation error.
func UnmarshalDescriptor(data []byte) (*ModuleDescriptor, error) {
	var descriptor ModuleDescriptor
	if err := json.Unmarshal(data, &descriptor); err != nil {
		return nil, fmt.Errorf("modules_domain: decoding descriptor: %w", err)
	}
	if err := descriptor.Validate(); err != nil {
		return nil, err
	}
	return &descriptor, nil
}
