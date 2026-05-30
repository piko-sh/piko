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

//go:build !go1.26

package interp_domain

// This file deliberately fails to compile on Go releases older than 1.26. The
// interp_domain package relies on reflect.PointerTo, reflect.TypeFor[T], unsafe.Slice and
// unsafe.String (which require Go 1.18+, well below the floor anyway); the reflect.Value
// internal layout {typ_, ptr, flag} introduced by the Go 1.21 rename of `typ` to `typ_`
// (the unsafeNewAt helper in reflect_value_unsafe.go reads/writes this layout directly
// via unsafe.Pointer, so we need a Go release where the layout matches); and build-tag
// predicates of the form "go1.26" used elsewhere in this package and in
// cmd/asmgen-generated .s files.

var (
	_ = piko_requires_go_1_26_or_newer //nolint:typecheck // intentional compile-time gate
)
