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

import "fmt"

// maxRegisterIndex is the maximum index a register can have, constrained
// by the uint8 encoding in the instruction format.
const maxRegisterIndex = 255

// spillThreshold is the register index at which new variable
// declarations are spilled instead of receiving a real register.
// Registers 0 through spillThreshold-1 are available for normal
// allocation; spillThreshold through maxRegisterIndex are reserved
// as headroom for temporary registers and scratch.
const spillThreshold uint32 = 246

// spillAreaOffset is the starting index in the register file where
// spill slots begin. Spill slot N maps to register file index
// spillAreaOffset + N.
const spillAreaOffset = 256

// varLocation records where a variable's value lives at runtime.
//
// IMPORTANT: The field layout of this struct is accessed at runtime by
// ASM dispatch code via hardcoded offsets in dispatch_offsets.h. The
// fields upvalueIndex, register, kind, isUpvalue, isIndirect, and
// originalKind MUST remain at their current offsets. New fields must
// be appended after isCaptured.
type varLocation struct {
	// upvalueIndex is the index into the upvalue table when isUpvalue
	// is true.
	upvalueIndex int

	// register is the register index within the bank.
	register uint8

	// kind is the register bank (int, float, string, general, bool,
	// uint, complex).
	kind registerKind

	// isUpvalue is true when the variable has been captured by a
	// closure and lives on the heap rather than in a register.
	isUpvalue bool

	// isIndirect is true when the variable's address has been taken
	// (&x). The general register holds a pointer (reflect.Value of
	// *T) and reads/writes must go through the pointee.
	isIndirect bool

	// originalKind stores the register bank the variable had before
	// it was heap-escaped by &x. Used to unpack indirect reads back
	// into the correct typed register.
	originalKind registerKind

	// isCaptured is true when this variable has been captured by a
	// closure. Writes to captured variables must emit opWriteSharedCell
	// to keep the upvalue cell in sync with the register.
	isCaptured bool

	// isSpilled is true when this variable lives in the spill area
	// (register file index >= 256) rather than a directly-addressable
	// register. Must be materialised via opReload before use in
	// instructions and stored back via opSpill after writes.
	isSpilled bool

	// spillSlot is the spill slot index (0-based) when isSpilled is
	// true. The actual register file index is 256 + spillSlot.
	spillSlot uint16
}

// registerAllocator manages register allocation within a function,
// allocating sequentially per bank and freeing when scopes exit.
// Each function has its own allocator.
//
// nextRegister and peakRegister use uint32 to avoid silent overflow
// when a function needs exactly 256 registers in a bank (indices
// 0-255). With uint8, incrementing past 255 wraps to 0 and
// peakRegister under-reports, causing runtime panics.
//
// recycledRegisters holds registers returned by dead-variable
// analysis. declareVar uses allocOrRecycle to prefer these over
// fresh sequential allocation, keeping peak register counts low
// for functions with many short-lived variables.
type registerAllocator struct {
	// overflowErr is set when a register bank exceeds the 256-register
	// limit imposed by the uint8 instruction encoding. Checked by the
	// compiler after function compilation completes.
	overflowErr error

	// functionName identifies the function being compiled, used in
	// diagnostic messages when register overflow is detected.
	functionName string

	// recycledRegisters holds register indices returned to the
	// allocator by dead-variable analysis.
	//
	// Entries are consumed LIFO by allocOrRecycle. Only entries at
	// or above the current scope's base index may be consumed,
	// preventing inner scopes from stealing outer-scope recycled
	// registers.
	recycledRegisters [NumRegisterKinds][]uint32

	// nextRegister tracks the next free register per bank.
	nextRegister [NumRegisterKinds]uint32

	// peakRegister tracks the highest register ever allocated per bank.
	// This determines the register file size for the function.
	peakRegister [NumRegisterKinds]uint32

	// nextSpillSlot tracks the next free spill slot per bank.
	// Spill slots are indexed from 0; the actual register file
	// index is 256 + spillSlot.
	nextSpillSlot [NumRegisterKinds]uint32

	// peakSpillSlot tracks the highest spill slot ever allocated
	// per bank. Used together with peakRegister to size the
	// register file: max(peakRegister, 256 + peakSpillSlot).
	peakSpillSlot [NumRegisterKinds]uint32
}

// alloc allocates a register in the given bank and returns its index.
//
// Takes kind (registerKind) which specifies the register bank to
// allocate from.
//
// Returns the index of the newly allocated register.
func (r *registerAllocator) alloc(kind registerKind) uint8 {
	register := r.nextRegister[kind]
	r.nextRegister[kind]++
	if r.nextRegister[kind] > r.peakRegister[kind] {
		r.peakRegister[kind] = r.nextRegister[kind]
	}
	if register > maxRegisterIndex && r.overflowErr == nil {
		r.overflowErr = fmt.Errorf(
			"register overflow in function %q: %s bank requires more than %d registers "+
				"(instruction encoding is limited to uint8 indices); "+
				"peak registers: int=%d float=%d string=%d general=%d bool=%d uint=%d complex=%d; "+
				"recycled pool lengths: int=%d float=%d string=%d general=%d bool=%d uint=%d complex=%d",
			r.functionName, kind, maxRegisterIndex+1,
			r.peakRegister[registerInt], r.peakRegister[registerFloat], r.peakRegister[registerString],
			r.peakRegister[registerGeneral], r.peakRegister[registerBool], r.peakRegister[registerUint],
			r.peakRegister[registerComplex],
			len(r.recycledRegisters[registerInt]), len(r.recycledRegisters[registerFloat]),
			len(r.recycledRegisters[registerString]), len(r.recycledRegisters[registerGeneral]),
			len(r.recycledRegisters[registerBool]), len(r.recycledRegisters[registerUint]),
			len(r.recycledRegisters[registerComplex]),
		)
	}
	return uint8(register) //nolint:gosec // overflow guarded
}

// allocTemp allocates a temporary register. The caller must call
// freeTemp when the temporary is no longer needed.
//
// Takes kind (registerKind) which specifies the register bank to
// allocate from.
//
// Returns the index of the newly allocated temporary register.
func (r *registerAllocator) allocTemp(kind registerKind) uint8 {
	return r.alloc(kind)
}

// freeTemp releases a temporary register. Temporaries must be freed
// in reverse allocation order (LIFO).
//
// Takes kind (registerKind) which specifies the register bank to
// free from.
func (r *registerAllocator) freeTemp(kind registerKind, _ uint8) {
	r.nextRegister[kind]--
}

// allocOrRecycle returns a recycled register if one is available
// above baseFreeIndex, otherwise allocates a fresh register
// sequentially. Used by declareVar for named variable allocation
// so that dead-variable registers are reused.
//
// Takes kind (registerKind) which specifies the register bank to
// allocate from.
// Takes baseFreeIndex (int) which is the free-list length when the
// current scope was entered. Only entries at or above this index
// may be consumed, preventing inner scopes from stealing outer
// scope recycled registers.
//
// Returns the index of the allocated or recycled register.
func (r *registerAllocator) allocOrRecycle(kind registerKind, baseFreeIndex int) uint8 {
	if length := len(r.recycledRegisters[kind]); length > baseFreeIndex {
		register := r.recycledRegisters[kind][length-1]
		r.recycledRegisters[kind] = r.recycledRegisters[kind][:length-1]
		return uint8(register) //nolint:gosec // overflow guarded
	}
	return r.alloc(kind)
}

// recycleRegister returns a register to the recycled pool for
// reuse by future allocOrRecycle calls. Called when dead-variable
// analysis determines a declared variable will never be referenced
// again.
//
// Takes kind (registerKind) which specifies the register bank.
// Takes register (uint8) which is the register index to recycle.
func (r *registerAllocator) recycleRegister(kind registerKind, register uint8) {
	r.recycledRegisters[kind] = append(r.recycledRegisters[kind], uint32(register))
}

// ensureMin guarantees that the peak register count for the given
// bank is at least count.
//
// Used by return compilation to ensure the frame has registers for
// cross-bank return-position writes.
//
// Takes kind (registerKind) which specifies the register bank to
// check.
// Takes count (uint32) which is the minimum peak count to enforce.
func (r *registerAllocator) ensureMin(kind registerKind, count uint32) {
	if count > r.peakRegister[kind] {
		r.peakRegister[kind] = count
	}
}

// snapshot returns the current allocation state for all banks.
//
// Used when entering a new scope so registers can be restored on exit.
//
// Returns a copy of the current register allocation counters.
func (r *registerAllocator) snapshot() [NumRegisterKinds]uint32 {
	return r.nextRegister
}

// restore resets the allocation state to a previously saved
// snapshot.
//
// Used when exiting a scope to free all registers allocated
// within it.
//
// Takes saved ([NumRegisterKinds]uint32) which is the snapshot to
// restore from.
//
// Panics if any saved counter exceeds the current nextRegister
// value for that bank.
func (r *registerAllocator) restore(saved [NumRegisterKinds]uint32) {
	for i := range NumRegisterKinds {
		if saved[i] > r.nextRegister[i] {
			panic(fmt.Sprintf(
				"registerAllocator.restore: saved[%s]=%d exceeds nextRegister=%d in function %q",
				registerKind(i), saved[i], r.nextRegister[i], r.functionName,
			))
		}
	}
	r.nextRegister = saved
}

// allocSpill allocates a spill slot in the given bank and returns
// its 0-based index. The actual register file index at runtime is
// 256 + the returned value.
//
// Takes kind (registerKind) which specifies the register bank to
// allocate a spill slot in.
//
// Returns the spill slot index.
func (r *registerAllocator) allocSpill(kind registerKind) uint16 {
	slot := r.nextSpillSlot[kind]
	r.nextSpillSlot[kind]++
	if r.nextSpillSlot[kind] > r.peakSpillSlot[kind] {
		r.peakSpillSlot[kind] = r.nextSpillSlot[kind]
	}
	return uint16(slot) //nolint:gosec // slot count bounded
}

// spillSnapshot returns the current spill allocation state for all
// banks.
//
// Used when entering a new scope so spill slots can be restored
// on exit.
//
// Returns [NumRegisterKinds]uint32 which is a copy of the current
// spill slot allocation counters.
func (r *registerAllocator) spillSnapshot() [NumRegisterKinds]uint32 {
	return r.nextSpillSlot
}

// spillRestore resets the spill allocation state to a previously
// saved snapshot.
//
// Used when exiting a scope to free spill slots allocated within
// it.
//
// Takes saved ([NumRegisterKinds]uint32) which is the snapshot to
// restore from.
func (r *registerAllocator) spillRestore(saved [NumRegisterKinds]uint32) {
	r.nextSpillSlot = saved
}

// needsSpill reports whether the next sequential allocation in
// the given bank would exceed the spill threshold.
//
// When true, the caller should use allocSpill instead of alloc
// for variable declarations.
//
// Takes kind (registerKind) which specifies the register bank to
// check.
//
// Returns bool which is true when the next allocation would
// exceed the spill threshold.
func (r *registerAllocator) needsSpill(kind registerKind) bool {
	return r.nextRegister[kind] >= spillThreshold
}

// lexicalScope represents a lexical scope during compilation.
// Variables declared in this scope are freed when the scope exits.
type lexicalScope struct {
	// vars maps variable names to their register locations.
	vars map[string]varLocation

	// savedRegisters records the register allocation state when this scope
	// was entered, allowing restoration on exit.
	savedRegisters [NumRegisterKinds]uint32

	// savedSpillSlots records the spill slot allocation state when this
	// scope was entered, allowing restoration on exit.
	savedSpillSlots [NumRegisterKinds]uint32

	// savedRecycledLengths records the length of each recycled-register
	// free list when this scope was entered. On scope exit, the free
	// lists are truncated back to these lengths, discarding any entries
	// added by the inner scope (those registers are freed by the
	// nextRegister restore) and preserving outer-scope entries.
	savedRecycledLengths [NumRegisterKinds]int
}

// scopeStack tracks nested lexical scopes during compilation.
type scopeStack struct {
	// alloc is the register allocator shared by all scopes.
	alloc *registerAllocator

	// debugVarTable is the variable debug table being built. Nil
	// when debug info is disabled.
	debugVarTable *debugVarTable

	// debugBodyLenFunc returns len(function.body) for recording
	// the current PC in var entries. Nil when debug is disabled.
	debugBodyLenFunc func() int

	// scopes is the stack of active lexical scopes.
	scopes []lexicalScope
}

// pushScope enters a new lexical scope.
func (s *scopeStack) pushScope() {
	var recycledLengths [NumRegisterKinds]int
	for i := range NumRegisterKinds {
		recycledLengths[i] = len(s.alloc.recycledRegisters[i])
	}
	s.scopes = append(s.scopes, lexicalScope{
		vars:                 make(map[string]varLocation),
		savedRegisters:       s.alloc.snapshot(),
		savedSpillSlots:      s.alloc.spillSnapshot(),
		savedRecycledLengths: recycledLengths,
	})
}

// popScope exits the current lexical scope, freeing all registers
// allocated within it and restoring the recycled-register free
// lists to their state when the scope was entered.
func (s *scopeStack) popScope() {
	if len(s.scopes) == 0 {
		return
	}
	top := s.scopes[len(s.scopes)-1]

	if s.debugVarTable != nil && s.debugBodyLenFunc != nil {
		endPC := s.debugBodyLenFunc()
		for name := range top.vars {
			for i := len(s.debugVarTable.entries) - 1; i >= 0; i-- {
				entry := &s.debugVarTable.entries[i]
				if entry.name == name && entry.endPC == 0 {
					entry.endPC = endPC
					break
				}
			}
		}
	}

	s.alloc.restore(top.savedRegisters)
	s.alloc.spillRestore(top.savedSpillSlots)
	for i := range NumRegisterKinds {
		s.alloc.recycledRegisters[i] = s.alloc.recycledRegisters[i][:top.savedRecycledLengths[i]]
	}
	s.scopes = s.scopes[:len(s.scopes)-1]
}

// declareVar declares a variable in the current scope and allocates
// a register for it. Prefers recycled registers from dead-variable
// analysis over fresh sequential allocation.
//
// Takes name (string) which is the variable name.
// Takes kind (registerKind) which specifies the register bank to
// allocate from.
//
// Returns the register location for the new variable.
func (s *scopeStack) declareVar(name string, kind registerKind) varLocation {
	baseFreeIndex := 0
	if len(s.scopes) > 0 {
		baseFreeIndex = s.scopes[len(s.scopes)-1].savedRecycledLengths[kind]
	}

	var location varLocation

	hasRecycled := len(s.alloc.recycledRegisters[kind]) > baseFreeIndex
	if hasRecycled || !s.alloc.needsSpill(kind) {
		register := s.alloc.allocOrRecycle(kind, baseFreeIndex)
		location = varLocation{register: register, kind: kind}
	} else {
		slot := s.alloc.allocSpill(kind)
		location = varLocation{kind: kind, isSpilled: true, spillSlot: slot}
	}

	if len(s.scopes) > 0 {
		s.scopes[len(s.scopes)-1].vars[name] = location
	}
	if s.debugVarTable != nil && s.debugBodyLenFunc != nil && name != "_" {
		s.debugVarTable.entries = append(s.debugVarTable.entries, debugVarEntry{
			name:     name,
			location: location,
			startPC:  s.debugBodyLenFunc(),
		})
	}
	return location
}

// lookupVar searches for a variable in the scope stack, starting from
// the innermost scope.
//
// Takes name (string) which is the variable name to look up.
//
// Returns the location and true if found.
func (s *scopeStack) lookupVar(name string) (varLocation, bool) {
	for i := len(s.scopes) - 1; i >= 0; i-- {
		if location, ok := s.scopes[i].vars[name]; ok {
			return location, true
		}
	}
	return varLocation{}, false
}

// markCaptured marks a variable as captured by a closure, so that
// subsequent writes to it will emit opWriteSharedCell.
//
// Takes name (string) which is the variable name to mark.
func (s *scopeStack) markCaptured(name string) {
	for i := len(s.scopes) - 1; i >= 0; i-- {
		if loc, ok := s.scopes[i].vars[name]; ok {
			loc.isCaptured = true
			s.scopes[i].vars[name] = loc
			return
		}
	}
}

// updateVar updates a variable's location in the scope stack.
//
// Takes name (string) which is the variable name to update.
// Takes location (varLocation) which is the new register location.
//
// Returns true if the variable was found and updated.
func (s *scopeStack) updateVar(name string, location varLocation) bool {
	for i := len(s.scopes) - 1; i >= 0; i-- {
		if _, ok := s.scopes[i].vars[name]; ok {
			s.scopes[i].vars[name] = location
			return true
		}
	}
	return false
}

// restoreWatermark restores register allocation to a saved
// snapshot while preserving registers referenced by scope
// variables.
//
// This prevents watermark restores from freeing registers
// allocated by variable promotion (e.g. compileAddressOfIdent
// promoting an int-bank variable to a general-bank indirect
// pointer).
//
// Takes saved ([NumRegisterKinds]uint32) which is the snapshot to
// restore from.
func (s *scopeStack) restoreWatermark(saved [NumRegisterKinds]uint32) {
	s.alloc.restore(saved)
	for _, scope := range s.scopes {
		for _, location := range scope.vars {
			if location.isSpilled {
				continue
			}
			needed := uint32(location.register) + 1
			if needed > s.alloc.nextRegister[location.kind] {
				s.alloc.nextRegister[location.kind] = needed
				if s.alloc.peakRegister[location.kind] < needed {
					s.alloc.peakRegister[location.kind] = needed
				}
			}
		}
	}
}

// peakRegisters returns the peak register usage per bank, which
// determines the register file size for the compiled function.
// When spilling is active for a bank, the register file must be
// large enough to hold both the directly-addressable registers
// (0-255) and the spill area (256+).
//
// Returns the peak register count array indexed by register kind.
func (s *scopeStack) peakRegisters() [NumRegisterKinds]uint32 {
	var result [NumRegisterKinds]uint32
	for i := range NumRegisterKinds {
		result[i] = s.alloc.peakRegister[i]
		if s.alloc.peakSpillSlot[i] > 0 {
			spillTotal := spillAreaOffset + s.alloc.peakSpillSlot[i]
			if spillTotal > result[i] {
				result[i] = spillTotal
			}
		}
	}
	return result
}

// overflowError returns the register overflow error if any bank
// exceeded the 256-register limit during compilation.
//
// Returns error which is the overflow error, or nil when all
// banks are within limits.
func (s *scopeStack) overflowError() error {
	return s.alloc.overflowErr
}

// newScopeStack creates a scope stack with a fresh register allocator.
//
// Takes functionName (string) which identifies the function being
// compiled, used in diagnostic messages.
//
// Returns a new scope stack ready for use.
func newScopeStack(functionName string) *scopeStack {
	return &scopeStack{
		alloc: &registerAllocator{functionName: functionName},
	}
}
