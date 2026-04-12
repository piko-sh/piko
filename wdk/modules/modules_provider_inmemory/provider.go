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

package modules_provider_inmemory

import (
	"context"
	"sync"

	"piko.sh/piko/wdk/modules"
)

// Provider serves the ModuleProvider port from an in-memory map.
//
// Holds a map of modules.ModuleRef to modules.ModuleBundle and looks up exact matches.
// The map is keyed by (Path, Version); Pin is checked separately by the bundle's
// VerifyAgainstRef. Hosts may Inject multiple versions of the same module path; lookups
// match on Version equality.
//
// Safe for concurrent Inject and Resolve calls.
type Provider struct {
	// bundles maps the (path, version) lookup key to the stored bundle pointer.
	bundles map[providerKey]*modules.ModuleBundle

	// mu guards bundles against concurrent Inject, Remove, and Resolve calls.
	mu sync.RWMutex
}

// providerKey is the internal (Path, Version) lookup key.
//
// Pin is not part of the key because two refs with the same (Path, Version) must resolve
// to the same bundle by construction.
type providerKey struct {
	// path is the module's canonical import path.
	path string

	// version is the module's resolved version string.
	version string
}

// New constructs an empty Provider ready for Inject.
//
// Returns a *Provider with an initialised internal map.
func New() *Provider {
	return &Provider{
		bundles: make(map[providerKey]*modules.ModuleBundle),
	}
}

// Inject stores bundle under the given ref's (Path, Version) key.
//
// Pin in the ref is ignored at injection time; the caller is responsible for ensuring the
// bundle's fingerprint matches the ref's Pin if integrity verification is enabled.
//
// Replaces any existing bundle at the same (Path, Version), which is useful for
// hot-reload patterns.
//
// Takes ref (modules.ModuleRef) which is the lookup ref.
// Takes bundle (*modules.ModuleBundle) which is the bundle to associate.
//
// Concurrency: acquires p.mu in write mode for the duration of the store.
func (p *Provider) Inject(ref modules.ModuleRef, bundle *modules.ModuleBundle) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.bundles[providerKey{path: ref.Path, version: ref.Version}] = bundle
}

// Remove deletes any bundle stored under (ref.Path, ref.Version).
//
// A subsequent Resolve for the same ref returns modules.ErrModuleNotFound.
//
// Takes ref (modules.ModuleRef) which identifies the entry to drop.
//
// Concurrency: acquires p.mu in write mode for the duration of the delete.
func (p *Provider) Remove(ref modules.ModuleRef) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.bundles, providerKey{path: ref.Path, version: ref.Version})
}

// Resolve implements modules.ModuleProvider.
//
// Looks up the bundle stored for the ref's (Path, Version) tuple, yielding
// modules.ErrModuleNotFound when no entry matches. The bundle's Pin is not verified here;
// the loader does that via modules.ModuleBundle.VerifyAgainstRef.
//
// Takes ctx (context.Context) which is observed for cancellation only; in-memory
// resolution does no I/O.
// Takes ref (modules.ModuleRef) which is the requested module.
//
// Returns *modules.ModuleBundle which is the stored bundle.
// Returns error when the ref does not match any stored entry or when ctx is cancelled.
//
// Concurrency: acquires p.mu in read mode for the lookup.
func (p *Provider) Resolve(ctx context.Context, ref modules.ModuleRef) (*modules.ModuleBundle, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p.mu.RLock()
	bundle, ok := p.bundles[providerKey{path: ref.Path, version: ref.Version}]
	p.mu.RUnlock()
	if !ok {
		return nil, modules.ErrModuleNotFound
	}
	return bundle, nil
}

// Len reports the number of bundles currently stored.
//
// Useful for tests that need to assert provider state.
//
// Returns int which is the number of (Path, Version) keys.
//
// Concurrency: acquires p.mu in read mode for the count.
func (p *Provider) Len() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.bundles)
}
