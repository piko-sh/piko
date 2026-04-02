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

// runDispatched executes bytecode starting from baseFramePointer using
// the standard switch dispatch loop. Used when the safe build tag is
// active or when targeting WebAssembly.
//
// Takes baseFramePointer (int) which specifies the call stack frame to
// return from when execution completes.
//
// Returns the execution result and any error encountered.
func (vm *VM) runDispatched(baseFramePointer int) (any, error) {
	return vm.run(baseFramePointer)
}
