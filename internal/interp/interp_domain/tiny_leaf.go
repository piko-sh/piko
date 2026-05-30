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
	"unsafe"
)

// tinyLeafShape classifies the body-pattern of trivial method callees that
// handleCallMethod can execute inline in the caller's frame, bypassing the full
// pushCompiledFrame machinery.
//
// Each shape matches a precise bytecode pattern recognised by classifyTinyLeaf during
// optimise(); runTinyLeafInline switches on the shape and hand-codes the inline read.
//
// Inline-leaf execution avoids the per-call frame-setup overhead and reduces the body to
// a single struct-field read followed by a register write. Trivial accessor methods like
// `func (n constNode) Eval() T { return n.value }` (2 ops) and `func (n varNode) Eval(env
// []T) T { return env[n.slot] }` (3 ops) match the recognised shapes.
type tinyLeafShape uint8

const (
	// tinyLeafNone marks a body that does not match any recognised shape; dispatch falls
	// back to pushCompiledFrame. Default for all functions.
	tinyLeafNone tinyLeafShape = iota

	// tinyLeafReturnUintField marks a `return recv.fieldX` body for a uint field.
	//
	// fieldX is a uint8/16/32/64 field on the receiver struct. The body is emitted as a
	// single GET_STRUCT_FIELD_UINT_T0 reading layoutIdx from the receiver followed by a
	// TIER2_RETURN of the single result. tinyLeafLayout holds the field's structFieldLayout
	// (offset and kind; FieldTypeIndex is unused). Matches polyast's constNode.Eval.
	tinyLeafReturnUintField

	// tinyLeafReturnIntField marks a `return recv.fieldX` body where fieldX is an int-kind
	// field. Symmetric to tinyLeafReturnUintField.
	tinyLeafReturnIntField

	// tinyLeafReturnEnvUintAtIntFieldSlot marks a `return env[recv.slot]` body.
	//
	// slot is an int-kind field on recv and env is a uint slice parameter. The body reads
	// recv.slot via GET_STRUCT_FIELD_INT_T0 into int[0], indexes env via SLICE_GET_UINT to
	// load uint[0], and returns the single uint result. tinyLeafLayout holds the slot-field
	// layout; tinyLeafEnvArgIdx names the parameter index whose general register holds the
	// env slice. Matches polyast's varNode.Eval.
	tinyLeafReturnEnvUintAtIntFieldSlot
)

// classifyTinyLeaf walks cf.body, recognises one of the supported shapes and populates
// cf.tinyLeafShape + cf.tinyLeafLayout + cf.tinyLeafEnvArgIdx accordingly. Leaves
// cf.tinyLeafShape = tinyLeafNone when no shape matches.
//
// Bails without setting a shape when the body length sits outside [2, 3] effective ops
// (NOPs skipped), when the first op is not a recognised GET_STRUCT_FIELD_*_T0 opener,
// when subsequent ops do not match a known pattern, or when the structLayoutTable index
// is out of bounds. Runs once per function at the end of optimise(); the allowlist is
// strict so anything not provably safe stays unclassified.
func (cf *CompiledFunction) classifyTinyLeaf() {
	const minTinyLeafBodyLength = 2
	const maxTinyLeafBodyLength = 6
	if len(cf.body) < minTinyLeafBodyLength || len(cf.body) > maxTinyLeafBodyLength {
		return
	}
	body := cf.body
	first := findNextNonNop(body, 0)
	if first < 0 {
		return
	}
	op0 := body[first]
	switch op0.op {
	case opGetStructFieldUint:
		cf.tryClassifyReturnUintField(body, first)
	case opGetStructFieldIntT0:
		cf.tryClassifyEnvAtIntFieldSlotOrReturnIntField(body, first)
	default:
	}
}

// tryClassifyReturnUintField matches `GET_STRUCT_FIELD_UINT_T0 + TIER2_RETURN`.
//
// Requires GET to write uint[0] with a==0 and b==0 (receiver is param 0) and the next
// non-NOP op to be opDrillTier1 carrying the subOpDrillTier2 + subOpTier2Return decoder
// bytes. On match, stores the layout and sets the shape.
//
// Takes body ([]instruction) which is the function body slice.
// Takes first (int) which is the index of the opening GET op.
func (cf *CompiledFunction) tryClassifyReturnUintField(body []instruction, first int) {
	op0 := body[first]
	if op0.a != 0 || op0.b != 0 {
		return
	}
	layoutIndex := int(op0.c)
	if layoutIndex >= len(cf.structLayoutTable) {
		return
	}
	next := findNextNonNop(body, first+1)
	if next < 0 || !isTier2Return(body[next]) {
		return
	}
	if hasNonNopAfter(body, next+1) {
		return
	}
	cf.tinyLeafLayout = cf.structLayoutTable[layoutIndex]
	cf.tinyLeafShape = tinyLeafReturnUintField
}

// tryClassifyEnvAtIntFieldSlotOrReturnIntField classifies bodies that begin with
// GET_STRUCT_FIELD_INT_T0. The body is either the 2-op `return recv.intField`
// (tinyLeafReturnIntField) or the 3-op `return env[recv.slot]`
// (tinyLeafReturnEnvUintAtIntFieldSlot); disambiguates by inspecting the next non-NOP op.
//
// Takes body ([]instruction) which is the function body slice.
// Takes first (int) which is the index of the opening GET op.
func (cf *CompiledFunction) tryClassifyEnvAtIntFieldSlotOrReturnIntField(body []instruction, first int) {
	op0 := body[first]
	if op0.a != 0 || op0.b != 0 {
		return
	}
	layoutIndex := int(op0.c)
	if layoutIndex >= len(cf.structLayoutTable) {
		return
	}
	next := findNextNonNop(body, first+1)
	if next < 0 {
		return
	}
	op1 := body[next]
	switch {
	case isTier2Return(op1):
		cf.classifyReturnIntFieldShape(body, next, layoutIndex)
	case op1.op == opSliceGetUint:
		cf.classifyEnvUintAtSliceGetUint(body, next, layoutIndex, op1)
	case op1.op == opDrillTier1 && subOpcode(op1.a) == subOpSliceGetUintDirect:
		cf.classifyEnvUintAtSliceGetUintDirect(body, next, layoutIndex, op1)
	}
}

// classifyReturnIntFieldShape finalises tinyLeafReturnIntField when the GET op is
// immediately followed by a tier-2 return with no other non-NOP ops trailing.
//
// Takes body ([]instruction) which is the function body.
// Takes retPos (int) which is the position of the return op.
// Takes layoutIndex (int) which indexes into structLayoutTable.
func (cf *CompiledFunction) classifyReturnIntFieldShape(body []instruction, retPos, layoutIndex int) {
	if hasNonNopAfter(body, retPos+1) {
		return
	}
	cf.tinyLeafLayout = cf.structLayoutTable[layoutIndex]
	cf.tinyLeafShape = tinyLeafReturnIntField
}

// classifyEnvUintAtSliceGetUint finalises tinyLeafReturnEnvUintAtIntFieldSlot when the
// GET op is followed by opSliceGetUint env[recv.field] and a tier-2 return.
//
// Takes body ([]instruction) which is the function body.
// Takes next (int) which is the position immediately after the GET.
// Takes layoutIndex (int) which indexes into structLayoutTable.
// Takes op1 (instruction) which is the candidate slice-get instruction.
func (cf *CompiledFunction) classifyEnvUintAtSliceGetUint(body []instruction, next, layoutIndex int, op1 instruction) {
	if op1.a != 0 || op1.c != 0 {
		return
	}
	envRegister := op1.b
	ret := findNextNonNop(body, next+1)
	if ret < 0 || !isTier2Return(body[ret]) {
		return
	}
	if hasNonNopAfter(body, ret+1) {
		return
	}
	cf.tinyLeafLayout = cf.structLayoutTable[layoutIndex]
	cf.tinyLeafEnvArgIdx = envRegister
	cf.tinyLeafShape = tinyLeafReturnEnvUintAtIntFieldSlot
}

// classifyEnvUintAtSliceGetUintDirect finalises typed-direct shape.
//
// Finalises tinyLeafReturnEnvUintAtIntFieldSlot for the typed-direct variant emitted when
// narrow-uint widths are promoted to registerSliceUint at the call-slot. Encoding:
//
//	DRILL_TIER1 a=subOpSliceGetUintDirect b=0 c=envSliceReg
//	EXT         a=indexReg                b=0 c=0
//	TIER2_RETURN (1 value)
//
// b=0 because the typed-direct op writes the uint result into uints[0] (the leaf's return
// slot); c is the slicesUint register holding env.
//
// Takes body ([]instruction) which is the function body.
// Takes next (int) which is the position immediately after the GET.
// Takes layoutIndex (int) which indexes into structLayoutTable.
// Takes op1 (instruction) which is the candidate direct slice-get instruction.
func (cf *CompiledFunction) classifyEnvUintAtSliceGetUintDirect(body []instruction, next, layoutIndex int, op1 instruction) {
	if op1.b != 0 {
		return
	}
	envRegister := op1.c
	extPos := next + 1
	if extPos >= len(body) || body[extPos].op != opExt {
		return
	}
	ret := findNextNonNop(body, extPos+1)
	if ret < 0 || !isTier2Return(body[ret]) {
		return
	}
	if hasNonNopAfter(body, ret+1) {
		return
	}
	cf.tinyLeafLayout = cf.structLayoutTable[layoutIndex]
	cf.tinyLeafEnvArgIdx = envRegister
	cf.tinyLeafShape = tinyLeafReturnEnvUintAtIntFieldSlot
}

// findNextNonNop returns the index of the first non-NOP instruction at or after start, or
// -1 if none. NOPs in piko are encoded as {opDrillTier1, 0, 0, 0} (the all-zero word that
// the dispatch fast-path treats as a no-op).
//
// Takes body ([]instruction) which is the function body slice.
// Takes start (int) which is the starting index.
//
// Returns the index of the first non-NOP at or after start, or -1 when only NOPs remain.
func findNextNonNop(body []instruction, start int) int {
	for i := start; i < len(body); i++ {
		instr := body[i]
		if instr.op == opDrillTier1 && instr.a == 0 && instr.b == 0 && instr.c == 0 {
			continue
		}
		return i
	}
	return -1
}

// hasNonNopAfter reports whether any instruction at or after start is not a NOP. Used to
// guard against accidental "match" on bodies that have trailing real ops the classifier
// does not recognise.
//
// Takes body ([]instruction) which is the function body slice.
// Takes start (int) which is the starting index.
//
// Returns true when at least one non-NOP exists at or after start.
func hasNonNopAfter(body []instruction, start int) bool {
	return findNextNonNop(body, start) >= 0
}

// isTier2Return reports whether instr is the tier-2 TIER2_RETURN op. The encoding is
// {opDrillTier1, subOpDrillTier2, subOpTier2Return, count}.
//
// Takes instr (instruction) which is the candidate instruction.
//
// Returns true when instr matches the TIER2_RETURN encoding.
func isTier2Return(instr instruction) bool {
	return instr.op == opDrillTier1 &&
		subOpcode(instr.a) == subOpDrillTier2 &&
		subOpcodeTier2(instr.b) == subOpTier2Return
}

// runTinyLeafInline executes the recognised tiny-leaf body of callee directly in the
// caller's frame, writing the result into the caller's return slot defined by
// site.returns[0]. The path bypasses pushCompiledFrame entirely.
//
// Callers in handleCallMethod must ensure callee.tinyLeafShape is not tinyLeafNone, that
// frame.programCounter has already been advanced past the CALL_METHOD opcode and its
// extension word (matching the standard handleCallMethod path), and that site.returns has
// at least one entry whose kind matches the shape's expected return kind. Never panics on
// a successfully-classified leaf because the classifier guarantees operand bounds and
// shape match.
//
// Takes registers (*Registers) which is the caller's register file.
// Takes site (*callSite) which carries argument and return slot metadata for the inlined
// call.
// Takes callee (*CompiledFunction) which carries the tinyLeafShape, tinyLeafLayout, and
// tinyLeafEnvArgIdx selected by the classifier.
//
// Returns opContinue.
func runTinyLeafInline(_ *VM, registers *Registers, site *callSite, callee *CompiledFunction) opResult {
	switch callee.tinyLeafShape {
	case tinyLeafReturnUintField:
		return runTinyLeafReturnUintField(registers, site, callee)
	case tinyLeafReturnIntField:
		return runTinyLeafReturnIntField(registers, site, callee)
	case tinyLeafReturnEnvUintAtIntFieldSlot:
		return runTinyLeafReturnEnvUintAtIntFieldSlot(registers, site, callee)
	default:
	}
	return opContinue
}

// runTinyLeafReturnUintField implements the inlined body for the `return recv.fieldX`
// shape where the field is uint-kind.
//
// Takes registers (*Registers) which is the caller's register file.
// Takes site (*callSite) which carries argument and return slot metadata.
// Takes callee (*CompiledFunction) which carries the layout selected by the classifier.
//
// Returns opContinue once the result has been written into the caller's return register.
//
//nolint:dupl // uint/int twin: generic helper would cost an indirection.
func runTinyLeafReturnUintField(registers *Registers, site *callSite, callee *CompiledFunction) opResult {
	recv := registers.general[site.arguments[0].register]
	if base, ok := structFieldUnsafeBaseFromValue(recv); ok {
		fieldPointer := unsafe.Add(base, uintptr(callee.tinyLeafLayout.Offset))
		registers.uints[site.returns[0].register] = readUintAt(fieldPointer, reflect.Kind(callee.tinyLeafLayout.Kind))
		return opContinue
	}
	field, fieldOK := tinyLeafReflectField(recv, callee.tinyLeafLayout)
	if !fieldOK {
		return opContinue
	}
	registers.uints[site.returns[0].register] = field.Uint()
	return opContinue
}

// runTinyLeafReturnIntField implements the inlined body for the `return recv.fieldX`
// shape where the field is int-kind.
//
// Takes registers (*Registers) which is the caller's register file.
// Takes site (*callSite) which carries argument and return slot metadata.
// Takes callee (*CompiledFunction) which carries the layout selected by the classifier.
//
// Returns opContinue once the result has been written into the caller's return register.
//
//nolint:dupl // uint/int twin: generic helper would cost an indirection.
func runTinyLeafReturnIntField(registers *Registers, site *callSite, callee *CompiledFunction) opResult {
	recv := registers.general[site.arguments[0].register]
	if base, ok := structFieldUnsafeBaseFromValue(recv); ok {
		fieldPointer := unsafe.Add(base, uintptr(callee.tinyLeafLayout.Offset))
		registers.ints[site.returns[0].register] = readIntAt(fieldPointer, reflect.Kind(callee.tinyLeafLayout.Kind))
		return opContinue
	}
	field, fieldOK := tinyLeafReflectField(recv, callee.tinyLeafLayout)
	if !fieldOK {
		return opContinue
	}
	registers.ints[site.returns[0].register] = field.Int()
	return opContinue
}

// runTinyLeafReturnEnvUintAtIntFieldSlot implements the inlined body for `return
// env[recv.slot]`: read the int slot field from the receiver, then index into the uint
// slice held in the env arg (arguments[1] for the recognised shape: varNode.Eval style).
//
// Takes registers (*Registers) which is the caller's register file.
// Takes site (*callSite) which carries argument and return slot metadata.
// Takes callee (*CompiledFunction) which carries the layout selected by the classifier.
//
// Returns opContinue once the result has been written into the caller's return register.
func runTinyLeafReturnEnvUintAtIntFieldSlot(registers *Registers, site *callSite, callee *CompiledFunction) opResult {
	if len(site.arguments) < 2 {
		return opContinue
	}
	recv := registers.general[site.arguments[0].register]
	base, ok := structFieldUnsafeBaseFromValue(recv)
	if !ok {
		return opContinue
	}
	fieldPointer := unsafe.Add(base, uintptr(callee.tinyLeafLayout.Offset))
	slot := readIntAt(fieldPointer, reflect.Kind(callee.tinyLeafLayout.Kind))

	switch envArg := site.arguments[1]; envArg.kind {
	case registerSliceUint:
		envSlice := registers.slicesUint[envArg.register]
		if int(slot) < 0 || int(slot) >= len(envSlice) {
			return opContinue
		}
		registers.uints[site.returns[0].register] = envSlice[int(slot)]
		return opContinue
	case registerSliceInt:
		envSlice := registers.slicesInt[envArg.register]
		if int(slot) < 0 || int(slot) >= len(envSlice) {
			return opContinue
		}
		//nolint:gosec // two's-complement reinterpret matching reflect.Value.Uint wrap semantics
		registers.uints[site.returns[0].register] = uint64(envSlice[int(slot)])
		return opContinue
	default:
		envValue := registers.general[envArg.register]
		if !envValue.IsValid() || envValue.Kind() != reflect.Slice {
			return opContinue
		}
		registers.uints[site.returns[0].register] = envValue.Index(int(slot)).Uint()
		return opContinue
	}
}

// structFieldUnsafeBaseFromValue extracts the receiver's struct base pointer from a
// reflect.Value held in the caller's general bank. Mirrors structFieldUnsafeBase but
// takes the Value directly because the leaf-inline path does not have a register index
// handy.
//
// Takes v (reflect.Value) which is the receiver value.
//
// Returns the base pointer and true when the Value is a Pointer to a struct (or an
// addressable struct directly).
// Returns (nil, false) for invalid Values, interfaces over non-pointer types, and any
// case the existing helper would refuse.
func structFieldUnsafeBaseFromValue(v reflect.Value) (unsafe.Pointer, bool) {
	if !v.IsValid() {
		return nil, false
	}
	rv := (*unsafeReflectValue)(unsafe.Pointer(&v))
	switch reflect.Kind(rv.flag & flagKindMask) {
	case reflect.Pointer:
		if rv.typ == nil {
			return nil, false
		}
		var base unsafe.Pointer
		if rv.flag&flagIndir != 0 {
			base = *(*unsafe.Pointer)(rv.ptr)
		} else {
			base = rv.ptr
		}
		if base == nil {
			return nil, false
		}
		return base, true
	case reflect.Struct:
		if rv.flag&flagAddr == 0 {
			return nil, false
		}
		return rv.ptr, true
	default:
	}
	return nil, false
}

// tinyLeafReflectField walks the layout's path via reflect to produce the leaf field when
// the unsafe base extraction failed. Conservative slow-path used by runTinyLeafInline for
// receivers the unsafe helper cannot crack (e.g., interface-typed receivers holding
// non-pointer concrete values).
//
// Takes recv (reflect.Value) which is the receiver value.
// Takes layout (structFieldLayout) which carries the field path.
//
// Returns the leaf field and true on success.
// Returns a zero reflect.Value and false when the path cannot be walked through reflect.
func tinyLeafReflectField(recv reflect.Value, layout structFieldLayout) (reflect.Value, bool) {
	if !recv.IsValid() {
		return reflect.Value{}, false
	}
	if recv.Kind() == reflect.Pointer || recv.Kind() == reflect.Interface {
		recv = recv.Elem()
	}
	if recv.Kind() != reflect.Struct {
		return reflect.Value{}, false
	}
	current := recv
	for i := uint8(0); i < layout.PathLength; i++ {
		if current.Kind() != reflect.Struct {
			return reflect.Value{}, false
		}
		fieldIndex := int(layout.Path[i])
		if fieldIndex < 0 || fieldIndex >= current.NumField() {
			return reflect.Value{}, false
		}
		current = current.Field(fieldIndex)
		if i+1 < layout.PathLength && (current.Kind() == reflect.Pointer || current.Kind() == reflect.Interface) {
			current = current.Elem()
		}
	}
	return current, true
}

// readUintAt loads a uint8/16/32/64-kind field from fieldPtr. Mirrors the switch in
// handleGetStructFieldUintT0.
//
// Takes p (unsafe.Pointer) which addresses the field bytes.
// Takes kind (reflect.Kind) which selects the load width.
//
// Returns the loaded value widened to uint64, or 0 when kind is not a recognised uint
// kind.
func readUintAt(p unsafe.Pointer, kind reflect.Kind) uint64 {
	switch kind {
	case reflect.Uint:
		return uint64(*(*uint)(p))
	case reflect.Uint8:
		return uint64(*(*uint8)(p))
	case reflect.Uint16:
		return uint64(*(*uint16)(p))
	case reflect.Uint32:
		return uint64(*(*uint32)(p))
	case reflect.Uint64:
		return *(*uint64)(p)
	case reflect.Uintptr:
		return uint64(*(*uintptr)(p))
	default:
	}
	return 0
}

// readIntAt loads an int8/16/32/64-kind field from fieldPtr. Mirrors the switch in
// handleGetStructFieldIntT0.
//
// Takes p (unsafe.Pointer) which addresses the field bytes.
// Takes kind (reflect.Kind) which selects the load width.
//
// Returns the loaded value widened to int64, or 0 when kind is not a recognised int kind.
func readIntAt(p unsafe.Pointer, kind reflect.Kind) int64 {
	switch kind {
	case reflect.Int:
		return int64(*(*int)(p))
	case reflect.Int8:
		return int64(*(*int8)(p))
	case reflect.Int16:
		return int64(*(*int16)(p))
	case reflect.Int32:
		return int64(*(*int32)(p))
	case reflect.Int64:
		return *(*int64)(p)
	default:
	}
	return 0
}
