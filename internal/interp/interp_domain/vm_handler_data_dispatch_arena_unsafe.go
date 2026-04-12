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

//go:build !safe

package interp_domain

import (
	"reflect"
)

// ensureSliceAddressableForHeader is a no-op in the unsafe build.
//
// reflectValuePtr reads the internal .ptr field directly via the reflect.Value layout
// cast and is never nil for a valid slice (addressable or not), so
// snapshotSliceHeaderArena always receives a valid header pointer. The safe-build sibling
// promotes non-addressable slices into a heap-backed holder so its reflectValuePtr (which
// only returns a pointer for addressable Values or direct-iface kinds) has somewhere to
// point.
//
// Takes v (reflect.Value) which is the slice value to pass through.
//
// Returns reflect.Value which is v unchanged.
func ensureSliceAddressableForHeader(v reflect.Value) reflect.Value {
	return v
}
