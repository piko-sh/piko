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
	"reflect"
	"strings"
	"sync"
)

const (
	// initialIntSlabs is the starting capacity for the int64 slab.
	// Typical: 16 registers x 32 call depth = 512.
	initialIntSlabs = 512

	// initialFloatSlabs is the starting capacity for the float64 slab.
	initialFloatSlabs = 128

	// initialStringSlabs is the starting capacity for the string slab.
	// Kept small because strings are GC-visible; sizeArenaFromFunctions
	// grows the slab to match the compiled function's actual needs.
	initialStringSlabs = 32

	// initialGeneralSlabs is the starting capacity for the reflect.Value slab.
	// Kept small because reflect.Value is pointer-rich and scanned on every
	// GC cycle; sizeArenaFromFunctions grows as needed.
	initialGeneralSlabs = 32

	// initialBoolSlabs is the starting capacity for the bool slab.
	initialBoolSlabs = 128

	// initialUintSlabs is the starting capacity for the uint64 slab.
	initialUintSlabs = 128

	// initialComplexSlabs is the starting capacity for the complex128 slab.
	initialComplexSlabs = 64

	// initialUpvalueCellSlabs is the starting capacity for the upvalueCell slab.
	// Typical: 1-3 upvalues per closure x ~32 call depth.
	initialUpvalueCellSlabs = 64

	// initialUpvalueRefSlabs is the starting capacity for the upvalue ref slab.
	initialUpvalueRefSlabs = 64

	// initialFrameSlabs is the starting capacity for the callFrame slab.
	// Matches initialCallStackSize in vm.go.
	initialFrameSlabs = 64

	// initialByteSlabSize is the starting capacity for the byte slab
	// used to intern string character data. Strings created during VM
	// execution have their bytes bump-allocated here instead of on the
	// Go heap, eliminating per-concat GC pressure.
	initialByteSlabSize = 4096

	// maxArenaMultiplier is the DoS protection threshold that caps slab
	// growth. If a slab grows beyond initialSize * maxArenaMultiplier,
	// it is shrunk on reset, preventing memory bloat from pathological
	// inputs while allowing legitimate growth to be retained across requests.
	maxArenaMultiplier = 8
)

// ArenaSavePoint records the arena allocation indices at a point in time,
// enabling call frames to restore the arena when popped. This turns the
// bump allocator into a stack allocator: a recursive function only needs
// max-depth x registers-per-frame arena slots, not total-calls x
// registers-per-frame.
type ArenaSavePoint struct {
	// intIndex stores the saved allocation index for the int64 slab.
	intIndex int

	// floatIndex stores the saved allocation index for the float64 slab.
	floatIndex int

	// stringIndex stores the saved allocation index for the string slab.
	stringIndex int

	// generalIndex stores the saved allocation index for the reflect.Value slab.
	generalIndex int

	// boolIndex stores the saved allocation index for the bool slab.
	boolIndex int

	// uintIndex stores the saved allocation index for the uint64 slab.
	uintIndex int

	// complexIndex stores the saved allocation index for the complex128 slab.
	complexIndex int

	// upvalueCellIndex stores the saved allocation index for the upvalueCell slab.
	upvalueCellIndex int

	// upvalueReferenceIndex stores the saved allocation index for the
	// upvalue reference slab.
	upvalueReferenceIndex int
}

// RegisterArena provides arena-based allocation for register banks.
// Instead of calling make() per call frame (one sync.Pool.Get per function
// call), the arena pre-allocates contiguous slabs and hands out sub-slices
// via simple index bumping - zero sync overhead in the hot path.
//
// Frames record a save point before allocating. When a frame is popped,
// the save point is restored, reclaiming the arena space. This makes the
// arena a stack allocator - ideal for a call stack where allocation and
// deallocation follow LIFO order.
//
// The arena also holds VM-parallel arrays (callInfoBasesSlab, dispatchSavesSlab)
// that shadow the call stack. These are pooled alongside the frame slab
// to avoid per-Execute() allocations.
//
// One arena is obtained per Eval() call (or per goroutine for opGo).
// The arena itself is pooled via sync.Pool.
type RegisterArena struct {
	// intSlab holds the contiguous int64 register memory.
	intSlab []int64

	// floatSlab holds the contiguous float64 register memory.
	floatSlab []float64

	// stringSlab holds the contiguous string register memory.
	stringSlab []string

	// generalSlab holds the contiguous reflect.Value register memory.
	generalSlab []reflect.Value

	// boolSlab holds the contiguous bool register memory.
	boolSlab []bool

	// uintSlab holds the contiguous uint64 register memory.
	uintSlab []uint64

	// complexSlab holds the contiguous complex128 register memory.
	complexSlab []complex128

	// frameSlab holds the pre-allocated call frame storage.
	frameSlab []callFrame

	// callInfoBasesSlab holds ASM call-info base pointers parallel to frameSlab.
	callInfoBasesSlab []uintptr

	// dispatchSavesSlab holds ASM dispatch register saves parallel to frameSlab.
	dispatchSavesSlab []asmDispatchSave

	// byteSlab holds bump-allocated byte storage for interned strings.
	byteSlab []byte

	// oldByteSlabs retains previous byte slabs so existing strings remain valid.
	oldByteSlabs [][]byte

	// upvalueCellSlab holds the contiguous upvalueCell storage.
	upvalueCellSlab []upvalueCell

	// upvalueReferenceSlab holds the contiguous upvalue reference storage.
	upvalueReferenceSlab []upvalue

	// intIndex tracks the current allocation position in intSlab.
	intIndex int

	// floatIndex tracks the current allocation position in floatSlab.
	floatIndex int

	// stringIndex tracks the current allocation position in stringSlab.
	stringIndex int

	// generalIndex tracks the current allocation position in generalSlab.
	generalIndex int

	// boolIndex tracks the current allocation position in boolSlab.
	boolIndex int

	// uintIndex tracks the current allocation position in uintSlab.
	uintIndex int

	// complexIndex tracks the current allocation position in complexSlab.
	complexIndex int

	// byteIndex tracks the current allocation position in byteSlab.
	byteIndex int

	// upvalueCellIndex tracks the current allocation position in upvalueCellSlab.
	upvalueCellIndex int

	// upvalueReferenceIndex tracks the current allocation position in
	// upvalueReferenceSlab.
	upvalueReferenceIndex int
}

// registerArenaPool is the single pool for arena instances.
var registerArenaPool = sync.Pool{
	New: func() any {
		return newRegisterArena()
	},
}

// CallInfoBases returns the arena's pre-allocated slab for ASM call-info
// base pointers, parallel to the frame stack.
//
// Returns the callInfoBasesSlab slice.
func (a *RegisterArena) CallInfoBases() []uintptr {
	return a.callInfoBasesSlab
}

// AllocStringBytes bump-allocates n bytes from the arena's byte slab
// and returns a sub-slice the caller must fully write before use.
//
// The returned slice shares memory with the arena; strings created via
// unsafe.String point into this slab rather than the Go heap.
//
// Takes n (int) which is the number of bytes to allocate.
//
// Returns a byte slice of length n backed by the arena's byte slab.
func (a *RegisterArena) AllocStringBytes(n int) []byte {
	if a.byteIndex+n > len(a.byteSlab) {
		a.growByteSlab(n)
	}
	buffer := a.byteSlab[a.byteIndex : a.byteIndex+n]
	a.byteIndex += n
	return buffer
}

// EnsureCapacity pre-sizes the arena slabs so that AllocRegisters
// never triggers a grow during execution.
//
// Called once after compilation with hints derived from the compiled
// function table.
//
// Takes ints, floats, stringSlice, generals, bools, uints, and complexes
// (int) which are the minimum capacities for each respective slab.
func (a *RegisterArena) EnsureCapacity(ints, floats, stringSlice, generals, bools, uints, complexes int) {
	if ints > len(a.intSlab) {
		a.intSlab = make([]int64, ints)
	}
	if floats > len(a.floatSlab) {
		a.floatSlab = make([]float64, floats)
	}
	if stringSlice > len(a.stringSlab) {
		a.stringSlab = make([]string, stringSlice)
	}
	if generals > len(a.generalSlab) {
		a.generalSlab = make([]reflect.Value, generals)
	}
	if bools > len(a.boolSlab) {
		a.boolSlab = make([]bool, bools)
	}
	if uints > len(a.uintSlab) {
		a.uintSlab = make([]uint64, uints)
	}
	if complexes > len(a.complexSlab) {
		a.complexSlab = make([]complex128, complexes)
	}
}

// AllocRegisters returns a registers struct backed by sub-slices of the
// arena's contiguous slabs via O(1) index bumping.
//
// The compiler emits explicit opLoadZero instructions for any variables
// that require zero-initialisation (named returns, uninitialised var
// declarations), so the arena does not zero int/float registers here.
//
// Takes numRegs ([NumRegisterKinds]uint32) which is the number of
// registers to allocate for each register kind.
//
// Returns a registers struct with each bank pointing into the arena slabs.
func (a *RegisterArena) AllocRegisters(numRegs [NumRegisterKinds]uint32) Registers {
	numInts := int(numRegs[registerInt])
	numFloats := int(numRegs[registerFloat])
	numStrings := int(numRegs[registerString])
	numGenerals := int(numRegs[registerGeneral])
	numBools := int(numRegs[registerBool])
	numUints := int(numRegs[registerUint])
	numComplexes := int(numRegs[registerComplex])

	if a.intIndex+numInts > len(a.intSlab) || a.floatIndex+numFloats > len(a.floatSlab) ||
		a.stringIndex+numStrings > len(a.stringSlab) || a.generalIndex+numGenerals > len(a.generalSlab) ||
		a.boolIndex+numBools > len(a.boolSlab) || a.uintIndex+numUints > len(a.uintSlab) ||
		a.complexIndex+numComplexes > len(a.complexSlab) {
		a.growSlabs(numInts, numFloats, numStrings, numGenerals, numBools, numUints, numComplexes)
	}

	r := Registers{
		ints:    a.intSlab[a.intIndex : a.intIndex+numInts],
		floats:  a.floatSlab[a.floatIndex : a.floatIndex+numFloats],
		strings: a.stringSlab[a.stringIndex : a.stringIndex+numStrings],
		general: a.generalSlab[a.generalIndex : a.generalIndex+numGenerals],
		bools:   a.boolSlab[a.boolIndex : a.boolIndex+numBools],
		uints:   a.uintSlab[a.uintIndex : a.uintIndex+numUints],
		complex: a.complexSlab[a.complexIndex : a.complexIndex+numComplexes],
	}

	a.intIndex += numInts
	a.floatIndex += numFloats
	a.stringIndex += numStrings
	a.generalIndex += numGenerals
	a.boolIndex += numBools
	a.uintIndex += numUints
	a.complexIndex += numComplexes

	return r
}

// AllocRegistersInto writes arena-backed sub-slices directly into the
// target registers pointer, avoiding the 168-byte by-value copy that
// AllocRegisters incurs.
//
// Used in the hot call path where the target is a callFrame.registers field
// already at its final address.
//
// Takes r (*Registers) which is the target registers to populate in
// place.
// Takes numRegs ([NumRegisterKinds]uint32) which is the number of
// registers to allocate for each register kind.
func (a *RegisterArena) AllocRegistersInto(r *Registers, numRegs [NumRegisterKinds]uint32) {
	numInts := int(numRegs[registerInt])
	numFloats := int(numRegs[registerFloat])
	numStrings := int(numRegs[registerString])
	numGenerals := int(numRegs[registerGeneral])
	numBools := int(numRegs[registerBool])
	numUints := int(numRegs[registerUint])
	numComplexes := int(numRegs[registerComplex])

	if a.intIndex+numInts > len(a.intSlab) || a.floatIndex+numFloats > len(a.floatSlab) ||
		a.stringIndex+numStrings > len(a.stringSlab) || a.generalIndex+numGenerals > len(a.generalSlab) ||
		a.boolIndex+numBools > len(a.boolSlab) || a.uintIndex+numUints > len(a.uintSlab) ||
		a.complexIndex+numComplexes > len(a.complexSlab) {
		a.growSlabs(numInts, numFloats, numStrings, numGenerals, numBools, numUints, numComplexes)
	}

	r.ints = a.intSlab[a.intIndex : a.intIndex+numInts]
	r.floats = a.floatSlab[a.floatIndex : a.floatIndex+numFloats]
	r.strings = a.stringSlab[a.stringIndex : a.stringIndex+numStrings]
	r.general = a.generalSlab[a.generalIndex : a.generalIndex+numGenerals]
	r.bools = a.boolSlab[a.boolIndex : a.boolIndex+numBools]
	r.uints = a.uintSlab[a.uintIndex : a.uintIndex+numUints]
	r.complex = a.complexSlab[a.complexIndex : a.complexIndex+numComplexes]

	a.intIndex += numInts
	a.floatIndex += numFloats
	a.stringIndex += numStrings
	a.generalIndex += numGenerals
	a.boolIndex += numBools
	a.uintIndex += numUints
	a.complexIndex += numComplexes
}

// Save returns a save point capturing the current allocation indices.
//
// Called before AllocRegisters to record where the arena was before
// allocating a frame's registers.
//
// Returns an ArenaSavePoint recording all current slab positions.
func (a *RegisterArena) Save() ArenaSavePoint {
	return ArenaSavePoint{
		intIndex:              a.intIndex,
		floatIndex:            a.floatIndex,
		stringIndex:           a.stringIndex,
		generalIndex:          a.generalIndex,
		boolIndex:             a.boolIndex,
		uintIndex:             a.uintIndex,
		complexIndex:          a.complexIndex,
		upvalueCellIndex:      a.upvalueCellIndex,
		upvalueReferenceIndex: a.upvalueReferenceIndex,
	}
}

// Restore rolls the arena back to a previous save point, reclaiming
// all register slots allocated since the save point was taken.
//
// Only string and general (reflect.Value) slabs are zeroed because they hold
// GC-visible references. Int and float slabs are left dirty because the compiler
// guarantees registers are written before read, so stale numeric data is
// harmless and skipping the clear saves significant time in call-heavy
// workloads.
//
// Takes sp (ArenaSavePoint) which is the save point to restore to.
func (a *RegisterArena) Restore(sp ArenaSavePoint) {
	clear(a.stringSlab[sp.stringIndex:a.stringIndex])
	clear(a.generalSlab[sp.generalIndex:a.generalIndex])

	a.intIndex = sp.intIndex
	a.floatIndex = sp.floatIndex
	a.stringIndex = sp.stringIndex
	a.generalIndex = sp.generalIndex
	a.boolIndex = sp.boolIndex
	a.uintIndex = sp.uintIndex
	a.complexIndex = sp.complexIndex
	a.upvalueCellIndex = sp.upvalueCellIndex
	a.upvalueReferenceIndex = sp.upvalueReferenceIndex
}

// Reset zeroes the used portions of each slab, resets indices, and shrinks
// any slabs exceeding the DoS protection threshold.
//
// Slabs are not shrunk unless they exceed the threshold, allowing
// naturally-grown slabs to be reused without reallocation.
func (a *RegisterArena) Reset() {
	clear(a.intSlab[:a.intIndex])
	clear(a.floatSlab[:a.floatIndex])
	clear(a.stringSlab[:a.stringIndex])
	clear(a.generalSlab[:a.generalIndex])
	clear(a.boolSlab[:a.boolIndex])
	clear(a.uintSlab[:a.uintIndex])
	clear(a.complexSlab[:a.complexIndex])
	clear(a.upvalueCellSlab[:a.upvalueCellIndex])
	clear(a.upvalueReferenceSlab[:a.upvalueReferenceIndex])

	for i := range a.frameSlab {
		f := &a.frameSlab[i]
		f.registers.strings = nil
		f.registers.general = nil
		f.function = nil
		f.sharedCells = nil
		f.upvalues = nil
		f.returnDestination = nil
	}

	a.intIndex = 0
	a.floatIndex = 0
	a.stringIndex = 0
	a.generalIndex = 0
	a.boolIndex = 0
	a.uintIndex = 0
	a.complexIndex = 0
	a.byteIndex = 0
	a.upvalueCellIndex = 0
	a.upvalueReferenceIndex = 0
	a.oldByteSlabs = a.oldByteSlabs[:0]

	a.shrinkOvergrownSlabs()
}

// shrinkOvergrownSlabs replaces any slab that has grown beyond
// the DoS protection threshold with a fresh default-sized allocation.
func (a *RegisterArena) shrinkOvergrownSlabs() {
	if len(a.intSlab) > initialIntSlabs*maxArenaMultiplier {
		a.intSlab = make([]int64, initialIntSlabs)
	}
	if len(a.floatSlab) > initialFloatSlabs*maxArenaMultiplier {
		a.floatSlab = make([]float64, initialFloatSlabs)
	}
	if len(a.stringSlab) > initialStringSlabs*maxArenaMultiplier {
		a.stringSlab = make([]string, initialStringSlabs)
	}
	if len(a.generalSlab) > initialGeneralSlabs*maxArenaMultiplier {
		a.generalSlab = make([]reflect.Value, initialGeneralSlabs)
	}
	if len(a.boolSlab) > initialBoolSlabs*maxArenaMultiplier {
		a.boolSlab = make([]bool, initialBoolSlabs)
	}
	if len(a.uintSlab) > initialUintSlabs*maxArenaMultiplier {
		a.uintSlab = make([]uint64, initialUintSlabs)
	}
	if len(a.complexSlab) > initialComplexSlabs*maxArenaMultiplier {
		a.complexSlab = make([]complex128, initialComplexSlabs)
	}
	if len(a.frameSlab) > initialFrameSlabs*maxArenaMultiplier {
		a.frameSlab = make([]callFrame, initialFrameSlabs)
	}
	if len(a.callInfoBasesSlab) > initialFrameSlabs*maxArenaMultiplier {
		a.callInfoBasesSlab = make([]uintptr, initialFrameSlabs)
	}
	if len(a.dispatchSavesSlab) > initialFrameSlabs*maxArenaMultiplier {
		a.dispatchSavesSlab = make([]asmDispatchSave, initialFrameSlabs)
	}
	if len(a.upvalueCellSlab) > initialUpvalueCellSlabs*maxArenaMultiplier {
		a.upvalueCellSlab = make([]upvalueCell, initialUpvalueCellSlabs)
	}
	if len(a.upvalueReferenceSlab) > initialUpvalueRefSlabs*maxArenaMultiplier {
		a.upvalueReferenceSlab = make([]upvalue, initialUpvalueRefSlabs)
	}
	if len(a.byteSlab) > initialByteSlabSize*maxArenaMultiplier {
		a.byteSlab = make([]byte, initialByteSlabSize)
	}
}

// frameStack returns the arena's pre-allocated call frame slab.
//
// Returns the frameSlab slice.
func (a *RegisterArena) frameStack() []callFrame {
	return a.frameSlab
}

// dispatchSaves returns the arena's pre-allocated slab for ASM dispatch
// register saves, parallel to the frame stack.
//
// Returns the dispatchSavesSlab slice.
func (a *RegisterArena) dispatchSaves() []asmDispatchSave {
	return a.dispatchSavesSlab
}

// growFrameStack grows the frameSlab, callInfoBasesSlab, and dispatchSavesSlab
// together to at least minCap.
//
// Takes minCap (int) which is the minimum required capacity for the slabs.
//
// Returns the new frame, call-info base, and dispatch save slabs.
//
//go:noinline
func (a *RegisterArena) growFrameStack(minCap int) ([]callFrame, []uintptr, []asmDispatchSave) {
	newCap := max(len(a.frameSlab)*2, minCap)

	newFrames := make([]callFrame, newCap)
	copy(newFrames, a.frameSlab)
	a.frameSlab = newFrames

	newCI := make([]uintptr, newCap)
	copy(newCI, a.callInfoBasesSlab)
	a.callInfoBasesSlab = newCI

	newDisp := make([]asmDispatchSave, newCap)
	copy(newDisp, a.dispatchSavesSlab)
	a.dispatchSavesSlab = newDisp

	return newFrames, newCI, newDisp
}

// allocUpvalueCells bump-allocates n upvalueCell slots from the arena.
//
// Takes n (int) which is the number of upvalue cells to allocate.
//
// Returns a zeroed slice of upvalueCell of length n.
func (a *RegisterArena) allocUpvalueCells(n int) []upvalueCell {
	if a.upvalueCellIndex+n > len(a.upvalueCellSlab) {
		a.growUpvalueCellSlab(a.upvalueCellIndex + n)
	}
	start := a.upvalueCellIndex
	a.upvalueCellIndex += n
	cells := a.upvalueCellSlab[start:a.upvalueCellIndex]
	clear(cells)
	return cells
}

// allocUpvalueRefs bump-allocates n upvalue slots from the arena.
//
// Takes n (int) which is the number of upvalue references to allocate.
//
// Returns a zeroed slice of upvalue of length n.
func (a *RegisterArena) allocUpvalueRefs(n int) []upvalue {
	if a.upvalueReferenceIndex+n > len(a.upvalueReferenceSlab) {
		a.growUpvalueRefSlab(a.upvalueReferenceIndex + n)
	}
	start := a.upvalueReferenceIndex
	a.upvalueReferenceIndex += n
	refs := a.upvalueReferenceSlab[start:a.upvalueReferenceIndex]
	clear(refs)
	return refs
}

// growUpvalueCellSlab grows the upvalueCell slab to at least minCap.
//
// Takes minCap (int) which is the minimum required capacity.
//
//go:noinline
func (a *RegisterArena) growUpvalueCellSlab(minCap int) {
	newCap := max(len(a.upvalueCellSlab)*2, minCap)
	newSlab := make([]upvalueCell, newCap)
	copy(newSlab, a.upvalueCellSlab)
	a.upvalueCellSlab = newSlab
}

// growUpvalueRefSlab grows the upvalue ref slab to at least minCap.
//
// Takes minCap (int) which is the minimum required capacity.
//
//go:noinline
func (a *RegisterArena) growUpvalueRefSlab(minCap int) {
	newCap := max(len(a.upvalueReferenceSlab)*2, minCap)
	newSlab := make([]upvalue, newCap)
	copy(newSlab, a.upvalueReferenceSlab)
	a.upvalueReferenceSlab = newSlab
}

// growByteSlab allocates a new byte slab, preserving the old one in
// oldByteSlabs so that existing strings pointing into it remain valid.
//
// Takes minExtra (int) which is the minimum number of bytes the new
// slab must hold.
//
//go:noinline
func (a *RegisterArena) growByteSlab(minExtra int) {
	a.oldByteSlabs = append(a.oldByteSlabs, a.byteSlab)
	newSize := max(len(a.byteSlab)*2, minExtra)
	a.byteSlab = make([]byte, newSize)
	a.byteIndex = 0
}

// growSlabs handles the rare case where arena slabs need to grow.
//
// Takes numInts, numFloats, numStrings, numGenerals, numBools, numUints,
// and numComplexes (int) which are the number of additional elements
// required for each respective slab.
//
//go:noinline
func (a *RegisterArena) growSlabs(numInts, numFloats, numStrings, numGenerals, numBools, numUints, numComplexes int) {
	if a.intIndex+numInts > len(a.intSlab) {
		a.growIntSlab(a.intIndex + numInts)
	}
	if a.floatIndex+numFloats > len(a.floatSlab) {
		a.growFloatSlab(a.floatIndex + numFloats)
	}
	if a.stringIndex+numStrings > len(a.stringSlab) {
		a.growStringSlab(a.stringIndex + numStrings)
	}
	if a.generalIndex+numGenerals > len(a.generalSlab) {
		a.growGeneralSlab(a.generalIndex + numGenerals)
	}
	if a.boolIndex+numBools > len(a.boolSlab) {
		a.growBoolSlab(a.boolIndex + numBools)
	}
	if a.uintIndex+numUints > len(a.uintSlab) {
		a.growUintSlab(a.uintIndex + numUints)
	}
	if a.complexIndex+numComplexes > len(a.complexSlab) {
		a.growComplexSlab(a.complexIndex + numComplexes)
	}
}

// growIntSlab grows the int slab to at least minCap.
//
// Takes minCap (int) which is the minimum required capacity.
func (a *RegisterArena) growIntSlab(minCap int) {
	newCap := max(len(a.intSlab)*2, minCap)
	newSlab := make([]int64, newCap)
	copy(newSlab, a.intSlab)
	a.intSlab = newSlab
}

// growFloatSlab grows the float slab to at least minCap.
//
// Takes minCap (int) which is the minimum required capacity.
func (a *RegisterArena) growFloatSlab(minCap int) {
	newCap := max(len(a.floatSlab)*2, minCap)
	newSlab := make([]float64, newCap)
	copy(newSlab, a.floatSlab)
	a.floatSlab = newSlab
}

// growStringSlab grows the string slab to at least minCap.
//
// Takes minCap (int) which is the minimum required capacity.
func (a *RegisterArena) growStringSlab(minCap int) {
	newCap := max(len(a.stringSlab)*2, minCap)
	newSlab := make([]string, newCap)
	copy(newSlab, a.stringSlab)
	a.stringSlab = newSlab
}

// growGeneralSlab grows the general slab to at least minCap.
//
// Takes minCap (int) which is the minimum required capacity.
func (a *RegisterArena) growGeneralSlab(minCap int) {
	newCap := max(len(a.generalSlab)*2, minCap)
	newSlab := make([]reflect.Value, newCap)
	copy(newSlab, a.generalSlab)
	a.generalSlab = newSlab
}

// growBoolSlab grows the bool slab to at least minCap.
//
// Takes minCap (int) which is the minimum required capacity.
func (a *RegisterArena) growBoolSlab(minCap int) {
	newCap := max(len(a.boolSlab)*2, minCap)
	newSlab := make([]bool, newCap)
	copy(newSlab, a.boolSlab)
	a.boolSlab = newSlab
}

// growUintSlab grows the uint slab to at least minCap.
//
// Takes minCap (int) which is the minimum required capacity.
func (a *RegisterArena) growUintSlab(minCap int) {
	newCap := max(len(a.uintSlab)*2, minCap)
	newSlab := make([]uint64, newCap)
	copy(newSlab, a.uintSlab)
	a.uintSlab = newSlab
}

// growComplexSlab grows the complex slab to at least minCap.
//
// Takes minCap (int) which is the minimum required capacity.
func (a *RegisterArena) growComplexSlab(minCap int) {
	newCap := max(len(a.complexSlab)*2, minCap)
	newSlab := make([]complex128, newCap)
	copy(newSlab, a.complexSlab)
	a.complexSlab = newSlab
}

// GetRegisterArena retrieves a RegisterArena from the pool.
//
// Returns a reset RegisterArena ready for use.
func GetRegisterArena() *RegisterArena {
	a, ok := registerArenaPool.Get().(*RegisterArena)
	if !ok {
		return newRegisterArena()
	}
	return a
}

// PutRegisterArena returns a RegisterArena to the pool after resetting.
//
// Takes a (*RegisterArena) which is the arena to return to the pool.
func PutRegisterArena(a *RegisterArena) {
	if a == nil {
		return
	}
	a.Reset()
	registerArenaPool.Put(a)
}

// materialiseString returns a heap-backed copy of s if it points into
// the arena's byte slabs.
//
// Takes arena (*RegisterArena) which is the arena whose byte slabs are
// checked for ownership.
// Takes s (string) which is the string to materialise.
//
// Returns a cloned string if s points into the arena, or s unchanged if
// it is already heap-backed.
func materialiseString(arena *RegisterArena, s string) string {
	if arena.ownsString(s) {
		return strings.Clone(s)
	}
	return s
}

// newRegisterArena creates a fresh arena with default slab sizes.
//
// Returns a newly allocated RegisterArena with all slabs at initial capacity.
func newRegisterArena() *RegisterArena {
	return &RegisterArena{
		intSlab:              make([]int64, initialIntSlabs),
		floatSlab:            make([]float64, initialFloatSlabs),
		stringSlab:           make([]string, initialStringSlabs),
		generalSlab:          make([]reflect.Value, initialGeneralSlabs),
		boolSlab:             make([]bool, initialBoolSlabs),
		uintSlab:             make([]uint64, initialUintSlabs),
		complexSlab:          make([]complex128, initialComplexSlabs),
		frameSlab:            make([]callFrame, initialFrameSlabs),
		callInfoBasesSlab:    make([]uintptr, initialFrameSlabs),
		dispatchSavesSlab:    make([]asmDispatchSave, initialFrameSlabs),
		byteSlab:             make([]byte, initialByteSlabSize),
		upvalueCellSlab:      make([]upvalueCell, initialUpvalueCellSlabs),
		upvalueReferenceSlab: make([]upvalue, initialUpvalueRefSlabs),
	}
}
