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
	"context"
	"errors"
)

// ModuleProvider is the port a host implements to translate a ModuleRef into a loadable
// ModuleBundle. The provider abstracts where bundles come from - local filesystem,
// in-memory map, GOPROXY fetch, OCI registry, signed asset store - so the rest of piko
// stays agnostic.
//
// Resolve is consulted by the piko loader before any bytecode is dispatched. Hosts
// implement gating, signature verification, approval workflow, and offline-mode policy
// here.
//
// Implementations MUST be safe for concurrent use by multiple goroutines: piko spawns
// goroutines for parallel module-graph resolution.
//
// Resolve returns ErrModuleNotFound for unknown refs, ErrIntegrityMismatch when a found
// bundle fails the host's own integrity check, ErrCapabilityDenied when host policy
// rejects the bundle outright, or any other error for transport failures.
type ModuleProvider interface {
	// Resolve fetches the bundle for ref via the provider's backend.
	//
	// Takes ctx (context.Context) which carries deadlines and cancel.
	// Takes ref (ModuleRef) which identifies the requested module.
	//
	// Returns *ModuleBundle which is the resolved artefact.
	// Returns error when lookup, integrity, or policy checks fail.
	Resolve(ctx context.Context, ref ModuleRef) (*ModuleBundle, error)
}

// ProviderFunc adapts an ordinary function to the ModuleProvider interface. Useful for
// tests and for ad-hoc providers that don't need any state.
type ProviderFunc func(ctx context.Context, ref ModuleRef) (*ModuleBundle, error)

// Resolve forwards to the wrapped function.
//
// Takes ctx (context.Context) which carries deadlines and cancel.
// Takes ref (ModuleRef) which identifies the requested module.
//
// Returns *ModuleBundle which is the resolved artefact.
// Returns error when the wrapped function reports a failure.
func (f ProviderFunc) Resolve(ctx context.Context, ref ModuleRef) (*ModuleBundle, error) {
	return f(ctx, ref)
}

// LoadedModule is the host's handle to a successfully-loaded module. Returned by piko's
// LoadModule after the bundle has been verified, its symbols registered, and its
// capability set bound to the active CapabilityHook.
//
// Carries the descriptor (for inspection - module listings, audit, debugging) and a
// Fingerprint that hosts use as a cache key. Unloading is via
// Service.UnloadModule(handle).
type LoadedModule struct {
	// Descriptor is the canonical descriptor used at load time. Held by reference; callers
	// must not mutate.
	Descriptor *ModuleDescriptor

	// Fingerprint is the bundle fingerprint captured at load time. Stable across invocations
	// of the same bundle, used as the identity for cache lookups and for cross-machine
	// deduplication.
	Fingerprint string
}

var (
	// ErrModuleNotFound indicates the provider could not resolve the requested ref.
	//
	// Used by chained providers as a fall-through signal: "not in this layer, try the next
	// one."
	ErrModuleNotFound = errors.New("modules_domain: module not found")

	// ErrIntegrityMismatch indicates the resolved bundle's content hash does not match the
	// expected pin.
	//
	// Distinct from ErrModuleNotFound so callers can distinguish "missing" from "tampered."
	ErrIntegrityMismatch = errors.New("modules_domain: integrity mismatch")

	// ErrCapabilityDenied indicates the host's CapabilityHook refused to load the module.
	// Distinct from ErrIntegrityMismatch so audit logs can route policy denials separately
	// from tampering events.
	ErrCapabilityDenied = errors.New("modules_domain: capability denied")

	// ErrCapabilityExceedsPolicy indicates the module's declared capability set is not a
	// subset of the host policy's grants: the validator-time equivalent of CapabilityDenied.
	// Surfaced at LoadModule before the bundle is registered.
	ErrCapabilityExceedsPolicy = errors.New("modules_domain: module capabilities exceed policy grants")

	// ErrStdlibIncompatible indicates the descriptor's pinned stdlib version is not
	// satisfied by the running piko.
	ErrStdlibIncompatible = errors.New("modules_domain: stdlib version incompatible")

	// ErrFrozen indicates a provider operating in frozen mode (no fetch, lockfile-only) was
	// asked to resolve a ref absent from its lockfile.
	ErrFrozen = errors.New("modules_domain: frozen provider rejects unpinned ref")
)
