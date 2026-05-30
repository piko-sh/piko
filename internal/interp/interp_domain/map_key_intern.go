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

package interp_domain

import (
	"strings"
)

// internMapKey returns a stable heap-clone of the supplied string.
//
// Reuses the cached copy whenever the same content has been observed before, via a map
// lookup; the cloned string is allocated once per unique content value and shared across
// all subsequent map inserts.
//
// Routing: when the VM has a globalStore (the standard case for Execute-driven programs),
// the intern lives on the globalStore so the cache is shared across every goroutine
// spawned by the same Execute. Parallel workloads spawn multiple worker goroutines, each
// receiving a fresh per-goroutine VM from runCompiledGoroutine. The shared globalStore
// intern means only the first observer across all workers pays the strings.Clone for a
// given key; subsequent observers reuse the cached copy.
//
// Fallback: when vm.globals is nil (test scaffolds, ad-hoc Eval calls), the per-VM
// internedMapKeys map is used, giving single-VM interning with no concurrency.
//
// Takes vm (*VM) which owns the per-VM fallback intern map and the globalStore pointer.
// Takes s (string) which is the key content to intern.
//
// Returns the canonical cloned copy.
func internMapKey(vm *VM, s string) string {
	if vm.arena == nil || !vm.arena.ownsString(s) {
		return s
	}
	if vm.globals != nil {
		return vm.globals.internMapKey(s)
	}
	if vm.internedMapKeys == nil {
		vm.internedMapKeys = make(map[string]string)
	}
	if cached, ok := vm.internedMapKeys[s]; ok {
		return cached
	}
	cloned := strings.Clone(s)
	vm.internedMapKeys[cloned] = cloned
	return cloned
}
