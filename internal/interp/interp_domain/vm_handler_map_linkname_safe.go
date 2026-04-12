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

//go:build safe || (js && wasm)

package interp_domain

import (
	"reflect"
	"unsafe"
)

// runtimeMapassignFast64 is the safe-build stub for the linkname trampoline; it always
// panics with ErrMapLinknameSafeStubInvoked.
//
// The go:linkname signature is fixed by the runtime contract on the unsafe build so this
// stub cannot return a structured error directly. It surfaces
// ErrMapLinknameSafeStubInvoked as the panic value so a recover() upstream can detect the
// build-tag misconfiguration via errors.Is without parsing message strings.
//
// Callers must gate the fast path through useMapFastLinkname so this stub stays
// unreachable.
//
// Returns nil; the panic interrupts before any value would be returned.
//
// Panics with ErrMapLinknameSafeStubInvoked on every call.
func runtimeMapassignFast64(_, _ unsafe.Pointer, _ uint64) unsafe.Pointer {
	panic(ErrMapLinknameSafeStubInvoked)
}

// runtimeMapassignFaststr is the safe-build stub for the linkname trampoline; it always
// panics with ErrMapLinknameSafeStubInvoked.
//
// Callers must gate the fast path through useMapFastLinkname so this stub stays
// unreachable.
//
// Returns nil; the panic interrupts before any value would be returned.
//
// Panics with ErrMapLinknameSafeStubInvoked on every call.
//
// See runtimeMapassignFast64 for the rationale on using the sentinel as the panic value.
func runtimeMapassignFaststr(_, _ unsafe.Pointer, _ string) unsafe.Pointer {
	panic(ErrMapLinknameSafeStubInvoked)
}

// runtimeTypedmemmove is the safe-build stub for the linkname trampoline; it always
// panics with ErrMapLinknameSafeStubInvoked.
//
// Callers must gate the fast path through useMapFastLinkname so this stub stays
// unreachable.
//
// Panics with ErrMapLinknameSafeStubInvoked on every call.
//
// See runtimeMapassignFast64 for the rationale on using the sentinel as the panic value.
func runtimeTypedmemmove(_, _, _ unsafe.Pointer) {
	panic(ErrMapLinknameSafeStubInvoked)
}

// mapAccessFast64ToGeneral returns (nil, false) so callers fall back to the reflect path.
//
// Returns nil and false unconditionally on the safe build.
func mapAccessFast64ToGeneral(_ reflect.Value, _ int64) (unsafe.Pointer, bool) {
	return nil, false
}

// mapAccessFastStrToGeneral returns (nil, false) so callers fall back to the reflect
// path.
//
// Returns nil and false unconditionally on the safe build.
func mapAccessFastStrToGeneral(_ reflect.Value, _ string) (unsafe.Pointer, bool) {
	return nil, false
}

// mapAssignFastStr returns nil so callers fall back to the reflect path.
//
// Returns nil unconditionally on the safe build.
func mapAssignFastStr(_ reflect.Value, _ string) unsafe.Pointer {
	return nil
}

// useMapFastLinkname reports whether the linkname-backed map fast paths are available.
// Safe / wasm builds return false so callers take the reflect fallback rather than
// reaching the panicking stubs.
//
// Returns false on safe / wasm builds; the unsafe build returns true.
func useMapFastLinkname() bool {
	return false
}
