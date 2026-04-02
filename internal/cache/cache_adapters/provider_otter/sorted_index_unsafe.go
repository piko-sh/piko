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

//go:build !safe && !(js && wasm)

package provider_otter

import "unsafe"

// keyTiebreak provides a deterministic ordering for B-tree items when values
// and typed keys compare equal. It uses pointer addresses as a stable
// tiebreaker within a single GC cycle.
//
// Takes a (*K) which is the first key to compare.
// Takes b (*K) which is the second key to compare.
//
// Returns bool which is true if a should come before b.
func keyTiebreak[K comparable](a, b *K) bool {
	return uintptr(unsafe.Pointer(a)) < uintptr(unsafe.Pointer(b))
}
