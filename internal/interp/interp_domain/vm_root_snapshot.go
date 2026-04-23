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

// frameRootSnapshot captures the VM dispatch state saved across a
// cross-compile closure call. popFrame restores these values so the
// caller sees the same functions slice and rootFunction it had
// before the swap.
type frameRootSnapshot struct {
	// rootFunction is the caller's rootFunction prior to the swap.
	rootFunction *CompiledFunction

	// functions is the caller's dispatch table prior to the swap.
	functions []*CompiledFunction
}

// swapToClosureRoot points the VM's dispatch tables at the closure's root.
//
// The frame-stack caller stores the returned snapshot in
// vm.rootSnapshots[framePointer] via recordFrameSnapshot so popFrame
// can restore it. A nil root, or a root that matches the current
// VM's, is a no-op and returns a nil snapshot.
//
// Takes closureRoot (*CompiledFunction) which is the closure's own
// rootFunction; may be nil when the closure was authored inside the
// current compile or lacks provenance.
//
// Returns *frameRootSnapshot containing the prior dispatch state, or
// nil when no swap occurred.
func (vm *VM) swapToClosureRoot(closureRoot *CompiledFunction) *frameRootSnapshot {
	if closureRoot == nil || closureRoot == vm.rootFunction {
		return nil
	}
	snapshot := &frameRootSnapshot{
		functions:    vm.functions,
		rootFunction: vm.rootFunction,
	}
	vm.functions = closureRoot.functions
	vm.rootFunction = closureRoot
	return snapshot
}

// recordFrameSnapshot stores the given snapshot for the current frame
// index, growing the parallel rootSnapshots slice when it is shorter
// than the call stack. A nil snapshot is a no-op when the slot is
// absent, since the zero value of an absent index is already nil.
//
// Takes framePointer (int) which is the index of the frame whose
// snapshot is being recorded.
// Takes snapshot (*frameRootSnapshot) which holds the prior dispatch
// state; nil when no swap occurred.
func (vm *VM) recordFrameSnapshot(framePointer int, snapshot *frameRootSnapshot) {
	if snapshot == nil && framePointer >= len(vm.rootSnapshots) {
		return
	}
	if framePointer >= len(vm.rootSnapshots) {
		required := framePointer + 1
		if cap(vm.rootSnapshots) < required {
			grown := make([]*frameRootSnapshot, required, max(required, 2*cap(vm.rootSnapshots)))
			copy(grown, vm.rootSnapshots)
			vm.rootSnapshots = grown
		} else {
			vm.rootSnapshots = vm.rootSnapshots[:required]
		}
	}
	vm.rootSnapshots[framePointer] = snapshot
}
