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

package bootstrap

// deref returns the value pointed to by ptr, or fallback if ptr is nil.
// This is used by the config conversion layer to safely extract values from
// pointer-based ServerConfig fields into value-typed domain config structs.
//
// Takes ptr (*T) which is the pointer to dereference.
// Takes fallback (T) which is the default value returned when ptr is nil.
//
// Returns T which is the dereferenced value or the fallback.
func deref[T any](ptr *T, fallback T) T {
	if ptr == nil {
		return fallback
	}
	return *ptr
}
