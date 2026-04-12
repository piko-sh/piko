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
)

// rootVisitor collects callbacks invoked on every arena-relevant GC root.
//
// Callbacks may be nil to ignore a category. Visitor implementations are responsible for
// distinguishing arena-backed from heap-backed values using the arena's ownsX predicates.
// Visit order is documented and stable across runs of the same VM state, which the tests
// depend on. The mark phase itself processes roots via a worklist and does not depend on
// the order.
type rootVisitor struct {
	// visitString receives string-bank entries (frame registers, globals) whose backing
	// bytes may live in arena byteSlab.
	visitString func(s string)

	// visitReflectValue receives general-bank entries and other reflect.Value roots
	// (closureCache, deferred-call native functions). The visitor inspects v.Kind() to
	// decide what further pointer-following to do.
	visitReflectValue func(v reflect.Value)

	// visitSliceInt receives []int64 register/global slice headers whose .Data pointer may
	// live in arena intBackingSlab or its oldIntBackings retention list.
	visitSliceInt func(s []int64)

	// visitSliceFloat is the float64 sibling of visitSliceInt.
	visitSliceFloat func(s []float64)

	// visitSliceString receives []string register/global slice headers whose .Data may live
	// in stringBackingSlab. Each element string is also a root that may point into byteSlab.
	visitSliceString func(s []string)

	// visitSliceBool is the bool sibling of visitSliceInt.
	visitSliceBool func(s []bool)

	// visitSliceUint is the uint64 sibling of visitSliceInt.
	visitSliceUint func(s []uint64)

	// visitSliceByte receives []byte register slice headers whose .Data may live in byteSlab
	// (via AllocByteBacking).
	visitSliceByte func(s []byte)

	// visitAny receives values held in any-typed fields: panicValue, evalResult,
	// evalAllResults entries, defer-call arguments after extraction. The visitor wraps in
	// reflect.ValueOf if needed.
	visitAny func(v any)

	// visitUpvalueCell receives each heap-allocated *upvalueCell reachable from
	// frame.upvalues or frame.sharedCells. The visitor then inspects cell.generalValue and
	// cell.stringValue for arena-backed contents.
	visitUpvalueCell func(c *upvalueCell)

	// visitRuntimeClosure receives each *runtimeClosure reachable from the dispatch state
	// (closureCache, defer records). The visitor inspects its upvalues / inlineCells.
	visitRuntimeClosure func(c *runtimeClosure)
}

// walkRoots invokes visitor callbacks on every arena-relevant root.
//
// Called by the GC mark phase before tracing, covering every root reachable from this
// VM's current state. The root set covers active call frames
// (vm.callStack[0:vm.framePointer+1]), the global store, the defer stack and frame-local
// simpleDefer records, the panic value, the eval result and all-results slots, and the
// closure cache.
//
// walkRoots holds no locks of its own: the dispatch loop pauses execution before
// invoking, and per-arena single-owner invariants guarantee no concurrent mutator. Global
// store access is unlocked; when shared, this VM owns its arena exclusively and other VMs
// do not point into this arena (materialise barrier), so concurrent reads from other VMs
// through globals only see materialised heap values that are not GC targets.
//
// Takes visitor (rootVisitor) which carries the per-category callbacks.
func (vm *VM) walkRoots(visitor rootVisitor) {
	vm.walkFrameRoots(visitor)
	vm.walkGlobalRoots(visitor)
	vm.walkDeferStackRoots(visitor)
	vm.walkVMStateRoots(visitor)
	vm.walkClosureCacheRoots(visitor)
}

// walkFrameRoots visits every register and frame-metadata pointer in the active call
// stack.
//
// Takes visitor (rootVisitor) which carries the per-category callbacks.
func (vm *VM) walkFrameRoots(visitor rootVisitor) {
	for i := 0; i <= vm.framePointer && i < len(vm.callStack); i++ {
		frame := &vm.callStack[i]
		walkRegisterRoots(&frame.registers, visitor)
		walkFrameMetadataRoots(frame, visitor)
	}
}

// walkGlobalRoots visits every pointer-typed global slot in the global store. Skips
// primitive globals (ints/floats/bools/uints/ complexes) - their slot values cannot
// reference arena memory.
//
// Takes visitor (rootVisitor) which carries the per-category callbacks.
func (vm *VM) walkGlobalRoots(visitor rootVisitor) {
	if vm.globals == nil {
		return
	}
	if visitor.visitString != nil {
		for _, s := range vm.globals.strings {
			visitor.visitString(s)
		}
	}
	if visitor.visitReflectValue != nil {
		for _, v := range vm.globals.general {
			visitor.visitReflectValue(v)
		}
	}
}

// walkDeferStackRoots visits every deferred call captured on the VM's defer stack. Each
// record holds a target closure, an optional native function reflect.Value, and an
// arguments slice.
//
// Takes visitor (rootVisitor) which carries the per-category callbacks.
func (vm *VM) walkDeferStackRoots(visitor rootVisitor) {
	for i := range vm.deferStack {
		record := &vm.deferStack[i]
		if record.function != nil && visitor.visitRuntimeClosure != nil {
			visitor.visitRuntimeClosure(record.function)
		}
		if record.nativeFunction.IsValid() && visitor.visitReflectValue != nil {
			visitor.visitReflectValue(record.nativeFunction)
		}
		if visitor.visitReflectValue != nil {
			for _, argument := range record.arguments {
				visitor.visitReflectValue(argument)
			}
		}
	}
}

// walkVMStateRoots visits VM-level any-typed and reflect.Value-typed fields: panicValue,
// evalResult, evalAllResults.
//
// Takes visitor (rootVisitor) which carries the per-category callbacks.
func (vm *VM) walkVMStateRoots(visitor rootVisitor) {
	if visitor.visitAny != nil {
		if vm.panicValue != nil {
			visitor.visitAny(vm.panicValue)
		}
		if vm.evalResult != nil {
			visitor.visitAny(vm.evalResult)
		}
		for _, r := range vm.evalAllResults {
			if r != nil {
				visitor.visitAny(r)
			}
		}
	}
}

// walkClosureCacheRoots visits every cached runtime closure. The closure cache holds
// reflect.Value wrappers for zero-upvalue closure instances that can be reused across
// calls.
//
// Takes visitor (rootVisitor) which carries the per-category callbacks.
func (vm *VM) walkClosureCacheRoots(visitor rootVisitor) {
	if visitor.visitReflectValue != nil {
		for _, c := range vm.closureCache {
			if c.IsValid() {
				visitor.visitReflectValue(c)
			}
		}
	}
}

// walkRegisterRoots visits every pointer-bearing register in a single frame's register
// file. Primitive banks (ints/floats/bools/uints/ complex) are skipped - their slot
// values cannot reference arena memory.
//
// Takes r (*Registers) which is the frame's register file.
// Takes visitor (rootVisitor) which carries the per-category callbacks.
func walkRegisterRoots(r *Registers, visitor rootVisitor) {
	walkRegisterScalarRoots(r, visitor)
	walkRegisterTypedSliceRoots(r, visitor)
}

// walkRegisterScalarRoots visits the per-register scalar banks that can hold arena
// pointers (strings, general reflect values).
//
// Takes r (*Registers) which is the register file whose scalar banks are walked.
// Takes visitor (rootVisitor) which provides callbacks invoked for each live scalar root.
func walkRegisterScalarRoots(r *Registers, visitor rootVisitor) {
	if visitor.visitString != nil {
		for _, s := range r.strings {
			visitor.visitString(s)
		}
	}
	if visitor.visitReflectValue != nil {
		for _, v := range r.general {
			visitor.visitReflectValue(v)
		}
	}
}

// walkRegisterTypedSliceRoots visits the per-register typed slice banks (int, float,
// string, bool, uint, byte).
//
// Takes r (*Registers) which is the register file whose typed-slice banks are walked.
// Takes visitor (rootVisitor) which provides callbacks invoked for each live typed-slice
// root.
func walkRegisterTypedSliceRoots(r *Registers, visitor rootVisitor) {
	walkRegisterNumericSliceRoots(r, visitor)
	walkRegisterCompositeSliceRoots(r, visitor)
}

// walkRegisterNumericSliceRoots visits the typed slice register banks for numeric element
// types (int, float, uint, byte).
//
// Takes r (*Registers) which is the register file whose numeric-slice banks are walked.
// Takes visitor (rootVisitor) which provides callbacks invoked for each live
// numeric-slice root.
func walkRegisterNumericSliceRoots(r *Registers, visitor rootVisitor) {
	if visitor.visitSliceInt != nil {
		for _, s := range r.slicesInt {
			visitor.visitSliceInt(s)
		}
	}
	if visitor.visitSliceFloat != nil {
		for _, s := range r.slicesFloat {
			visitor.visitSliceFloat(s)
		}
	}
	if visitor.visitSliceUint != nil {
		for _, s := range r.slicesUint {
			visitor.visitSliceUint(s)
		}
	}
	if visitor.visitSliceByte != nil {
		for _, s := range r.slicesByte {
			visitor.visitSliceByte(s)
		}
	}
}

// walkRegisterCompositeSliceRoots visits the typed slice register banks whose elements
// are not pure numerics (string, bool).
//
// Takes r (*Registers) which is the register file whose composite-slice banks are walked.
// Takes visitor (rootVisitor) which provides callbacks invoked for each live
// composite-slice root.
func walkRegisterCompositeSliceRoots(r *Registers, visitor rootVisitor) {
	if visitor.visitSliceString != nil {
		for _, s := range r.slicesString {
			visitor.visitSliceString(s)
		}
	}
	if visitor.visitSliceBool != nil {
		for _, s := range r.slicesBool {
			visitor.visitSliceBool(s)
		}
	}
}

// walkFrameMetadataRoots visits a single frame's metadata pointers: upvalues,
// sharedCells, simpleDefer record.
//
// Takes frame (*callFrame) which holds the metadata pointers.
// Takes visitor (rootVisitor) which carries the per-category callbacks.
func walkFrameMetadataRoots(frame *callFrame, visitor rootVisitor) {
	if visitor.visitUpvalueCell != nil {
		for _, u := range frame.upvalues {
			if u.value != nil {
				visitor.visitUpvalueCell(u.value)
			}
		}
	}
	if visitor.visitUpvalueCell != nil && frame.sharedCells != nil {
		walkSharedCellsRoots(frame.sharedCells, visitor)
	}
	if frame.simpleDefer != nil && frame.simpleDefer.active {
		walkSimpleDeferRoots(frame.simpleDefer, visitor)
	}
}

// walkSharedCellsRoots visits inline + overflow cells in a sharedCellMap. Each cell may
// carry arena-backed data in generalValue / stringValue.
//
// Takes m (*sharedCellMap) which holds the inline and overflow cells.
// Takes visitor (rootVisitor) which carries the per-category callbacks.
func walkSharedCellsRoots(m *sharedCellMap, visitor rootVisitor) {
	for i := 0; i < m.inlineLen && i < len(m.inlineCells); i++ {
		if m.inlineCells[i] != nil {
			visitor.visitUpvalueCell(m.inlineCells[i])
		}
	}
	for _, entry := range m.overflow {
		if entry.cell != nil {
			visitor.visitUpvalueCell(entry.cell)
		}
	}
}

// walkSimpleDeferRoots visits the target closure, native function, and argument list
// captured in a simpleDeferRecord.
//
// Takes record (*simpleDeferRecord) which holds the deferred call.
// Takes visitor (rootVisitor) which carries the per-category callbacks.
func walkSimpleDeferRoots(record *simpleDeferRecord, visitor rootVisitor) {
	if record.target != nil && visitor.visitRuntimeClosure != nil {
		visitor.visitRuntimeClosure(record.target)
	}
	if record.nativeFunction.IsValid() && visitor.visitReflectValue != nil {
		visitor.visitReflectValue(record.nativeFunction)
	}
	if visitor.visitReflectValue != nil {
		for _, argument := range record.arguments {
			visitor.visitReflectValue(argument)
		}
	}
}
