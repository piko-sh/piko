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

import (
	"runtime"
)

// lockInterpreter acquires the family interpreter lock when this VM is the host-goroutine
// boundary that should own it under safe mode.
//
// It is a no-op (returning false) outside safe mode, when this VM already holds the lock,
// or for a re-entrant VM (a closure/method invoked by native code), which runs under the
// caller's held lock or is the documented native-goroutine residual.
//
// Returns bool reporting whether this call acquired the lock (so the caller knows to pair
// it with unlockInterpreter, typically via defer).
//
// Concurrency: locks the per-family interpreter mutex (the GIL) shared across every VM in
// the family.
func (vm *VM) lockInterpreter() bool {
	if !vm.limits.safeMode || vm.holdsInterpreterLock || vm.reentrantInterpreterVM {
		return false
	}
	vm.globals.interpreterLock.Lock()
	vm.holdsInterpreterLock = true
	return true
}

// unlockInterpreter releases the family interpreter lock if this VM holds it.
//
// Concurrency: unlocks the per-family interpreter mutex (the GIL) shared across every VM
// in the family.
func (vm *VM) unlockInterpreter() {
	if !vm.holdsInterpreterLock {
		return
	}
	vm.holdsInterpreterLock = false
	vm.globals.interpreterLock.Unlock()
}

// yieldInterpreterLock briefly releases and re-acquires the interpreter lock so a sibling
// interpreted goroutine can run.
//
// Called at the periodic instruction checkpoint so a CPU-bound safe-mode goroutine cannot
// starve its siblings. A no-op unless this VM currently holds the lock.
//
// Concurrency: unlocks then re-locks the per-family interpreter mutex (the GIL), with a
// runtime.Gosched between, so a waiting sibling can acquire it.
func (vm *VM) yieldInterpreterLock() {
	if !vm.holdsInterpreterLock {
		return
	}
	vm.globals.interpreterLock.Unlock()
	runtime.Gosched()
	vm.globals.interpreterLock.Lock()
}

// releaseAroundBlock releases the interpreter lock before this VM parks on a blocking
// channel or select operation, so a sibling can run the matching operation that unblocks
// it.
//
// Without the release the family would deadlock. The caller pairs it with
// reacquireAfterBlock once the blocking call returns.
//
// Returns bool reporting whether the lock was released and must be re-acquired.
//
// Concurrency: unlocks the per-family interpreter mutex (the GIL) when held, leaving it
// released for the duration of the blocking call.
func (vm *VM) releaseAroundBlock() bool {
	if !vm.holdsInterpreterLock {
		return false
	}
	vm.globals.interpreterLock.Unlock()
	return true
}

// reacquireAfterBlock re-acquires the interpreter lock after a blocking operation
// returns, but only when releaseAroundBlock actually released it.
//
// Takes released (bool) which is the value returned by the paired releaseAroundBlock.
//
// Concurrency: re-locks the per-family interpreter mutex (the GIL) when the paired
// releaseAroundBlock reported it released.
func (vm *VM) reacquireAfterBlock(released bool) {
	if !released {
		return
	}
	vm.globals.interpreterLock.Lock()
}
