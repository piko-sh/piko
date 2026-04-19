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

package embedregistry

import (
	"context"
	"io/fs"
	"slices"
	"sync"

	"piko.sh/piko/internal/logger/logger_domain"
)

// payload couples the embedded filesystem with the embedded manifest bytes. They register
// together because the container rejects an embedded .piko filesystem without an embedded
// manifest, so a half-registered pair must be impossible.
type payload struct {
	// fsys is the embedded runtime tree, rooted where a .piko directory would be.
	fsys fs.FS

	// manifest is the compiled manifest (dist/manifest.bin) bytes.
	manifest []byte
}

var (
	// registryMu guards registered.
	registryMu sync.RWMutex

	// registered holds the embedded runtime payload, or nil when nothing registered.
	registered *payload
)

// Register stores the embedded runtime payload for pickup at boot. Generated code
// (dist/embed_gen.go) calls it during package initialisation; the last registration wins,
// which is logged because two embed packages in one binary is almost certainly a mistake.
//
// A call with a nil filesystem or an empty manifest is logged and ignored: the pair is
// only usable together, so a partial registration must never become the boot default.
//
// Takes fsys (fs.FS) which is the embedded runtime tree.
// Takes manifest ([]byte) which is the compiled manifest bytes.
//
// Concurrency: acquires registryMu while storing the payload.
func Register(ctx context.Context, fsys fs.FS, manifest []byte) {
	_, l := logger_domain.From(ctx, log)
	if fsys == nil || len(manifest) == 0 {
		l.Error("Ignoring embedded runtime registration with a missing filesystem or manifest; the pair must register together")
		return
	}

	registryMu.Lock()
	defer registryMu.Unlock()
	if registered != nil {
		l.Warn("Replacing a previously registered embedded runtime payload; two embed packages in one binary is almost certainly a mistake")
	}
	registered = &payload{fsys: fsys, manifest: manifest}
}

// Payload returns the registered embedded filesystem and manifest bytes. The manifest is
// returned as a defensive copy, so a caller cannot mutate the registry's shared state.
//
// Returns fsys (fs.FS) which is the embedded runtime tree, or nil when none registered.
// Returns manifest ([]byte) which is a copy of the compiled manifest bytes.
// Returns ok (bool) which is false when no payload has been registered.
//
// Concurrency: acquires registryMu (read lock) while copying the registered payload.
func Payload() (fsys fs.FS, manifest []byte, ok bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	if registered == nil {
		return nil, nil, false
	}
	return registered.fsys, slices.Clone(registered.manifest), true
}

// Reset clears the registry. It exists for tests, which share the process-global state.
//
// Concurrency: acquires registryMu while clearing the registry.
func Reset() {
	registryMu.Lock()
	defer registryMu.Unlock()
	registered = nil
}
