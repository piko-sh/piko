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

package collection_domain

import "unsafe"

// sliceDataPointer returns the address of the backing array for the given
// slice. This is used to identify whether a []map[string]any slice is the
// same instance returned by GetStaticCollectionItems.
//
// Takes items ([]map[string]any) which is the slice to fingerprint.
//
// Returns uintptr which is the address of the backing array, or 0 if empty.
func sliceDataPointer(items []map[string]any) uintptr {
	if len(items) == 0 {
		return 0
	}
	return uintptr(unsafe.Pointer(&items[0]))
}
