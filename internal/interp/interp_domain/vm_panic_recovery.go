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

// deliverRecoveredReturn copies the current frame's named return values to the caller's
// return destination registers when a panic was caught by a deferred recover(). Mirrors
// the value-passing handleReturn does for normal returns; without it, callers see
// whatever stale values were in their destination registers when the panic happened.
//
// Takes frame (*callFrame) which is the recovered frame whose named return values must be
// delivered back to the caller.
func (vm *VM) deliverRecoveredReturn(frame *callFrame) {
	namedLocations := frame.function.namedResultLocations
	if len(namedLocations) == 0 {
		return
	}
	returnDestination := frame.returnDestination
	if len(returnDestination) == 0 {
		vm.evalResult, _ = vm.extractResult(frame)
		vm.evalAllResults = vm.extractAllResults(frame)
		return
	}
	var bankCounters [NumRegisterKinds]uint8
	for i := range returnDestination {
		dest := returnDestination[i]
		kind := frame.function.resultKinds[i]
		sourceRegister := bankCounters[kind]
		bankCounters[kind]++
		vm.copyReturnValueAt(frame, kind, sourceRegister, dest)
	}
}
