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

// Package wasmrecover holds the panic-recovery primitives shared by the
// Piko WASM command's synchronous JS handlers. It lives in its own
// platform-agnostic package so the recovery semantics can be unit tested
// outside the js/wasm build constraint.
package wasmrecover

import "fmt"

// Sync runs operation under a deferred recover and reports whether it
// panicked. Callers in the wasm command wrap the result into a JS
// errorResult; tests can verify the recovery contract directly.
//
// Takes component (string) which identifies the JS handler for diagnostic
// logging when a panic is recovered.
// Takes operation (func()) which is the synchronous body to run.
//
// Returns string which is the formatted panic message ("panic in <component>:
// <value>") when operation panicked. Empty when operation returned cleanly.
// Returns bool which is true when a panic was recovered, false otherwise.
func Sync(component string, operation func()) (panicMessage string, panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[Piko WASM] panic in %s: %v\n", component, r)
			panicMessage = fmt.Sprintf("panic in %s: %v", component, r)
			panicked = true
		}
	}()
	operation()
	return "", false
}
