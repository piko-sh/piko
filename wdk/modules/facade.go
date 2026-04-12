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

package modules

import (
	"piko.sh/piko/internal/modules/modules_domain"
)

// Re-exports for the public surface. All names below are type aliases (not new types) so
// external code can pass values to and from piko's internal layers without conversion.
// See the package doc for the contract overview.

// ModuleRef identifies a module dependency. See [modules_domain.ModuleRef] for field
// semantics.
type ModuleRef = modules_domain.ModuleRef

// ModuleDescriptor declares what a module is. See [modules_domain.ModuleDescriptor].
type ModuleDescriptor = modules_domain.ModuleDescriptor

// ModuleBundle pairs descriptor with compiled bytecode. See
// [modules_domain.ModuleBundle].
type ModuleBundle = modules_domain.ModuleBundle

// Capability is a single capability claim. See [modules_domain.Capability].
type Capability = modules_domain.Capability

// CapabilitySet is an ordered collection of Capability claims.
type CapabilitySet = modules_domain.CapabilitySet

// ModuleProvider is the host's resolution port.
type ModuleProvider = modules_domain.ModuleProvider

// ProviderFunc adapts a function to the ModuleProvider interface.
type ProviderFunc = modules_domain.ProviderFunc

// LoadedModule is piko's handle to a loaded module.
type LoadedModule = modules_domain.LoadedModule

// TypesExportEntry is one (importPath, gcexportdata-bytes) pair from
// ModuleBundle.TypesExport's TLV stream.
type TypesExportEntry = modules_domain.TypesExportEntry

const (
	// DescriptorVersion is the current ModuleDescriptor schema version.
	DescriptorVersion = modules_domain.DescriptorVersion
)

// EncodeTypesExport serialises entries to the TLV bytes stored in
// ModuleBundle.TypesExport.
//
// Takes entries ([]TypesExportEntry) which is the import-path keyed stream to encode.
//
// Returns []byte which holds the TLV-encoded payload.
// Returns error when an entry fails to encode.
func EncodeTypesExport(entries []TypesExportEntry) ([]byte, error) {
	return modules_domain.EncodeTypesExport(entries)
}

// DecodeTypesExport reverses EncodeTypesExport.
//
// Empty input yields a nil slice for legacy bundles.
//
// Takes data ([]byte) which is the raw TLV-encoded payload.
//
// Returns []TypesExportEntry which holds the decoded entries.
// Returns error when the payload is malformed.
func DecodeTypesExport(data []byte) ([]TypesExportEntry, error) {
	return modules_domain.DecodeTypesExport(data)
}

var (
	// ErrModuleNotFound is returned when a provider cannot resolve the ref. Use errors.Is to
	// route on it.
	ErrModuleNotFound = modules_domain.ErrModuleNotFound

	// ErrIntegrityMismatch is returned when a bundle content hash does not match the
	// expected pin.
	ErrIntegrityMismatch = modules_domain.ErrIntegrityMismatch

	// ErrCapabilityDenied is returned when the host CapabilityHook refuses the load.
	ErrCapabilityDenied = modules_domain.ErrCapabilityDenied

	// ErrCapabilityExceedsPolicy is returned when module capabilities exceed policy grants.
	ErrCapabilityExceedsPolicy = modules_domain.ErrCapabilityExceedsPolicy

	// ErrStdlibIncompatible is returned when the descriptor stdlib pin is not satisfied.
	ErrStdlibIncompatible = modules_domain.ErrStdlibIncompatible

	// ErrFrozen is returned when a frozen provider rejects an unpinned ref.
	ErrFrozen = modules_domain.ErrFrozen
)

// ParseModuleRef parses a "path[@version][#pin]" string.
//
// Takes input (string) which is the textual module reference.
//
// Returns ModuleRef which is the parsed reference.
// Returns error when the input cannot be parsed.
//
// SeeAlso: [modules_domain.ParseModuleRef].
func ParseModuleRef(input string) (ModuleRef, error) {
	return modules_domain.ParseModuleRef(input)
}

// ParseCapability parses the "axis[(scope)]" form.
//
// Takes input (string) which is the textual capability claim.
//
// Returns Capability which is the parsed claim.
//
// SeeAlso: [modules_domain.ParseCapability].
func ParseCapability(input string) Capability {
	return modules_domain.ParseCapability(input)
}

// UnmarshalDescriptor decodes canonical JSON.
//
// Takes data ([]byte) which is the canonical JSON payload.
//
// Returns *ModuleDescriptor which is the decoded descriptor.
// Returns error when the payload is malformed or fails validation.
//
// SeeAlso: [modules_domain.UnmarshalDescriptor].
func UnmarshalDescriptor(data []byte) (*ModuleDescriptor, error) {
	return modules_domain.UnmarshalDescriptor(data)
}
