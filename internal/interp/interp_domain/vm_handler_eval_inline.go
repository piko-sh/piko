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
	"sync/atomic"
	"unsafe"
)

const (
	// inlineEvalMethodName is the hard-coded method name targeted by the binop inline shape.
	// The inline-eval node family is the sole consumer, so the name is a package constant
	// rather than a per-inlineDescriptor field.
	inlineEvalMethodName = "Eval"

	// inlineBinopUintMaxMaskWidth caps accepted mask widths at 64.
	//
	// The piko compiler emits TRUNCATE_NARROW only for widths 8, 16, 32; a width of 64 would
	// be a no-op and is never emitted, but we cap at 64 defensively so the (1<<width)-1 mask
	// never overflows.
	inlineBinopUintMaxMaskWidth = 64
)

// handleCallMethodInlineable dispatches opCallMethodInlineable.
//
// Behaves identically to handleCallMethod at the bytecode level (same operand shape, same
// call-site lookup) but the runtime path consults a per-callSite per-receiver-type
// inlineDescriptor cache and, on a known fused shape (binary-op Eval today, extensible to
// other shapes), runs the body inline in the caller's frame. On a non-inlineable shape it
// falls back to the standard dispatch via dispatchMethodCallSite.
//
// Takes vm (*VM) which is the virtual machine.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which encodes the call-site index in wideIndex().
//
// Returns opResult indicating the next execution step.
func handleCallMethodInlineable(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	siteIndex := instr.wideIndex()
	if int(siteIndex) >= len(frame.function.callSites) {
		vmBoundsError(vm, frame, boundsTableCallSite, int(siteIndex), len(frame.function.callSites))
		return opPanicError
	}
	site := &frame.function.callSites[siteIndex]
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++

	if len(site.arguments) == 0 {
		vm.evalError = newRuntimePanicError("interp: method call site has no receiver argument")
		return opPanicError
	}
	receiver := registers.general[site.arguments[0].register]
	recvType := receiver.Type()
	if recvType.Kind() == reflect.Pointer {
		recvType = recvType.Elem()
	}

	desc := lookupInlineDescriptor(site, recvType)
	if desc != nil {
		if desc.shape == inlineShapeBinopUint {
			return runInlineBinopUint(vm, frame, registers, site, desc)
		}
		if desc.callee.tinyLeafShape != tinyLeafNone {
			return runTinyLeafInline(vm, registers, site, desc.callee)
		}
		return pushCompiledFrame(vm, registers, site, desc.callee)
	}

	result := dispatchMethodCallSite(vm, frame, registers, site, extensionWord)
	classifyAndCacheFromMethodIC(site, recvType)
	return result
}

// lookupInlineDescriptor walks the call site's inlineDescriptorSlots looking for a cached
// classification of the given receiver type.
// Returns nil on miss.
//
// Takes site (*callSite) which holds the cache slots.
// Takes recvType (reflect.Type) which is the receiver concrete type.
//
// Returns the cached descriptor on hit, nil on miss.
func lookupInlineDescriptor(site *callSite, recvType reflect.Type) *inlineDescriptor {
	for slotIndex := range site.inlineDescriptorSlots {
		pointer := atomic.LoadPointer(&site.inlineDescriptorSlots[slotIndex])
		if pointer == nil {
			continue
		}
		desc := (*inlineDescriptor)(pointer)
		if desc.recvType == recvType {
			return desc
		}
	}
	return nil
}

// classifyAndCacheFromMethodIC publishes an inline descriptor.
//
// Walks methodICSlots looking for an entry that matches recvType (just populated by the
// dispatch we returned from), classifies the resolved callee's body for known fused
// inline shapes, and publishes the result into the inlineDescriptorSlots cache.
// Subsequent calls with the same (callSite, recvType) hit the cache and take the inline
// path.
//
// Takes site (*callSite) which holds both the method IC and the inline-descriptor cache.
// Takes recvType (reflect.Type) which is the receiver type we just dispatched for.
func classifyAndCacheFromMethodIC(site *callSite, recvType reflect.Type) {
	var callee *CompiledFunction
	for slotIndex := range site.methodICSlots {
		pointer := atomic.LoadPointer(&site.methodICSlots[slotIndex])
		if pointer == nil {
			continue
		}
		entry := (*monomorphicCacheEntry)(pointer)
		if entry.receiverType == recvType {
			callee = entry.callee
			break
		}
	}
	if callee == nil {
		return
	}
	desc := &inlineDescriptor{
		recvType: recvType,
		callee:   callee,
		shape:    classifyInlineShape(callee),
	}
	if desc.shape == inlineShapeBinopUint {
		fillBinopUintDescriptor(callee, desc)
	}
	victim := site.inlineDescriptorVictim & inlineDescriptorVictimMask
	site.inlineDescriptorVictim++
	atomic.StorePointer(&site.inlineDescriptorSlots[victim], unsafe.Pointer(desc))
}

// classifyInlineShape walks callee.body for known fused shapes.
//
// The binop matcher accepts:
//
//	GET_STRUCT_FIELD_GENERAL leftDest recv=0 leftLayoutIdx
//	CALL_METHOD[_INLINEABLE] + ext     (left.Eval(env))
//	GET_STRUCT_FIELD_GENERAL rightDest recv=0 rightLayoutIdx
//	CALL_METHOD[_INLINEABLE] + ext     (right.Eval(env))
//	{ADD|SUB|MUL}_UINT result leftReg rightReg
//	(optional) LOAD_UINT_CONST or {opDrillTier1,subOpLoadUintConstSmall}
//	(optional) BIT_AND_UINT result result maskReg
//	TIER2_RETURN
//
// NOPs between ops are skipped. Any unrecognised op aborts the match.
//
// Takes callee (*CompiledFunction) which is the function whose body is scanned.
//
// Returns inlineShape which is the recognised shape, or inlineShapeNone when no pattern
// matches.
func classifyInlineShape(callee *CompiledFunction) inlineShape {
	if callee == nil || len(callee.body) == 0 {
		return inlineShapeNone
	}
	if matchBinopUintShape(callee, nil) {
		return inlineShapeBinopUint
	}
	return inlineShapeNone
}

// fillBinopUintDescriptor populates desc's binop-shape metadata (layouts, binop opcode,
// mask). Called only after classifyInlineShape already returned inlineShapeBinopUint, so
// re-runs the matcher with desc != nil to extract the metadata into the descriptor.
//
// Takes callee (*CompiledFunction) which is the matched function.
// Takes desc (*inlineDescriptor) which receives the metadata.
func fillBinopUintDescriptor(callee *CompiledFunction, desc *inlineDescriptor) {
	matchBinopUintShape(callee, desc)
}

// matchBinopUintShape walks the body once.
//
// When desc is non-nil and the body matches, also writes layout, binop, and mask metadata
// into desc.
//
// Takes callee (*CompiledFunction) which is the function to scan.
// Takes desc (*inlineDescriptor) which is optional metadata sink.
//
// Returns bool which is true when the body matches.
//
//nolint:revive // single-purpose pattern matcher
func matchBinopUintShape(callee *CompiledFunction, desc *inlineDescriptor) bool {
	body := callee.body
	pos := findNextNonNop(body, 0)
	if pos < 0 {
		return false
	}

	if pos >= len(body) || body[pos].op != opGetStructFieldGeneral || body[pos].b != 0 {
		return false
	}
	leftLayoutIdx := body[pos].c
	if int(leftLayoutIdx) >= len(callee.structLayoutTable) {
		return false
	}
	pos++

	pos = findNextNonNop(body, pos)
	if pos < 0 || pos+1 >= len(body) {
		return false
	}
	if body[pos].op != opCallMethod && body[pos].op != opCallMethodInlineable {
		return false
	}
	leftSiteIndex := body[pos].wideIndex()
	if int(leftSiteIndex) >= len(callee.callSites) {
		return false
	}
	leftSite := &callee.callSites[leftSiteIndex]
	if !isInlineableShape(leftSite) {
		return false
	}
	leftReturnReg := leftSite.returns[0].register
	pos += 2

	pos = findNextNonNop(body, pos)
	if pos < 0 || pos >= len(body) || body[pos].op != opGetStructFieldGeneral || body[pos].b != 0 {
		return false
	}
	rightLayoutIdx := body[pos].c
	if int(rightLayoutIdx) >= len(callee.structLayoutTable) {
		return false
	}
	pos++

	pos = findNextNonNop(body, pos)
	if pos < 0 || pos+1 >= len(body) {
		return false
	}
	if body[pos].op != opCallMethod && body[pos].op != opCallMethodInlineable {
		return false
	}
	rightSiteIndex := body[pos].wideIndex()
	if int(rightSiteIndex) >= len(callee.callSites) {
		return false
	}
	rightSite := &callee.callSites[rightSiteIndex]
	if !isInlineableShape(rightSite) {
		return false
	}
	rightReturnReg := rightSite.returns[0].register
	pos += 2

	pos = findNextNonNop(body, pos)
	if pos < 0 || pos >= len(body) {
		return false
	}
	binopInstr := body[pos]
	switch binopInstr.op {
	case opAddUint, opSubUint, opMulUint:
	default:
		return false
	}

	if binopInstr.b != leftReturnReg || binopInstr.c != rightReturnReg {
		return false
	}
	binopResultReg := binopInstr.a
	pos++

	maskApplies := false
	var maskValue uint64
	pos = findNextNonNop(body, pos)
	if pos > 0 && pos < len(body) && body[pos].op == opTruncateNarrow {
		truncInstr := body[pos]
		if truncInstr.a == binopResultReg {
			width := int(truncInstr.b)
			if width > 0 && width < inlineBinopUintMaxMaskWidth {
				maskApplies = true
				maskValue = (uint64(1) << width) - 1
				pos++
			}
		}
	}

	pos = findNextNonNop(body, pos)
	if pos > 0 && pos < len(body) {
		moveInstr := body[pos]
		if moveInstr.op == opDrillTier1 && subOpcode(moveInstr.a) == subOpMoveUint && moveInstr.c == binopResultReg {
			pos++
		}
	}

	pos = findNextNonNop(body, pos)
	if pos < 0 || pos >= len(body) {
		return false
	}
	if !isTier2Return(body[pos]) {
		return false
	}

	if hasNonNopAfter(body, pos+1) {
		return false
	}

	if desc != nil {
		desc.leftLayout = callee.structLayoutTable[leftLayoutIdx]
		desc.rightLayout = callee.structLayoutTable[rightLayoutIdx]
		desc.binopOpcode = binopInstr.op
		desc.maskApplies = maskApplies
		desc.maskValue = maskValue
	}
	return true
}

// runInlineBinopUint executes a classified binop-shape inline.
//
// Skips the outer pushCompiledFrame plus copyCallArgs for the binop method itself by
// reading left and right field pointers from the receiver via tinyLeafReflectField,
// dispatching the two inner Eval calls via invokeInlineEvalInner (each resolves the
// callee per-receiver-type via the outer site's methodICSlots, then pushes a frame
// manually with direct register placement to avoid the site-mismatch panic copyCallArgs
// would raise), applying the binop and optional mask inline, then writing the final
// result into site.returns[0].register.
//
// The manual placement bypassing copyCallArgs is required because the outer site's
// argCopyProgram is nil (method-call sites are runtime-resolved) and the fallback path's
// per-bank kindIndex reads source registers from positions encoded for the outer caller,
// not the inner callee's parameter-bank layout.
//
// Takes vm (*VM) which drives the synchronous sub-dispatch loop.
// Takes registers (*Registers) which holds the caller's banks.
// Takes site (*callSite) which describes the outer call.
// Takes desc (*inlineDescriptor) which carries the binop metadata.
//
// Returns opResult indicating the next execution step.
func runInlineBinopUint(vm *VM, _ *callFrame, registers *Registers, site *callSite, desc *inlineDescriptor) opResult {
	if len(site.arguments) < 2 || len(site.returns) < 1 {
		return pushCompiledFrame(vm, registers, site, desc.callee)
	}
	if vm.framePointer >= vm.callDepthLimit() {
		return opStackOverflow
	}

	recvReg := site.arguments[0].register
	returnReg := site.returns[0].register
	recv := registers.general[recvReg]
	envValue, envOk := readInlineEnvAsReflect(registers, site.arguments[1])
	if !envOk {
		return pushCompiledFrame(vm, registers, site, desc.callee)
	}

	leftField, leftFieldOk := resolveInlineBinopOperand(recv, desc.leftLayout)
	rightField, rightFieldOk := resolveInlineBinopOperand(recv, desc.rightLayout)
	if !leftFieldOk || !rightFieldOk {
		return pushCompiledFrame(vm, registers, site, desc.callee)
	}

	leftResult, leftErr := invokeInlineEvalInner(vm, registers, site, desc, leftField, envValue, returnReg)
	if leftErr != opContinue {
		return leftErr
	}
	rightResult, rightErr := invokeInlineEvalInner(vm, registers, site, desc, rightField, envValue, returnReg)
	if rightErr != opContinue {
		return rightErr
	}

	var result uint64
	switch desc.binopOpcode {
	case opAddUint:
		result = leftResult + rightResult
	case opSubUint:
		result = leftResult - rightResult
	case opMulUint:
		result = leftResult * rightResult
	default:
		return pushCompiledFrame(vm, registers, site, desc.callee)
	}
	if desc.maskApplies {
		result &= desc.maskValue
	}
	registers.uints[returnReg] = result
	return opContinue
}

// resolveInlineBinopOperand walks the receiver through the layout to the field, unwraps
// any interface, and validates the result.
// Returns (field, true) on success; (zero, false) when the field can't be resolved or is
// invalid.
//
// Takes recv (reflect.Value) which is the receiver value.
// Takes layout (structFieldLayout) which selects the field.
//
// Returns reflect.Value which is the resolved field, valid on success.
// Returns bool which is true when resolution succeeded.
func resolveInlineBinopOperand(recv reflect.Value, layout structFieldLayout) (reflect.Value, bool) {
	field, ok := tinyLeafReflectField(recv, layout)
	if !ok || !field.IsValid() {
		return reflect.Value{}, false
	}
	if field.Kind() == reflect.Interface {
		field = field.Elem()
	}
	if !field.IsValid() {
		return reflect.Value{}, false
	}
	return field, true
}

// invokeInlineEvalInner dispatches a single inner Eval call from the inlined binop body.
// Resolves the callee for receiver's concrete type via the outer site's methodICSlots (or
// methodTable on miss), then routes to either the tinyLeaf inline path or the manual
// frame-push path based on the callee's classification.
//
// Takes vm (*VM) which drives the re-entrant dispatch loop.
// Takes registers (*Registers) which holds the outer caller's banks.
// Takes site (*callSite) which holds methodICSlots for type resolution and site.returns
// for the synthetic returnDestination.
// Takes receiver (reflect.Value) which is the leaf or binop pointer passed as recv to the
// inner Eval.
// Takes envValue (reflect.Value) which is the env []uint32 carried from the outer caller.
// Takes returnReg (uint8) which is the caller's uint slot where the inner Eval's result
// lands and from which we read it.
//
// Returns the inner Eval's uint64 result and opContinue on success.
// Returns 0 and a failing opResult on panic, stack overflow, or callee resolution
// failure.
func invokeInlineEvalInner(vm *VM, registers *Registers, site *callSite, desc *inlineDescriptor, receiver, envValue reflect.Value, returnReg uint8) (uint64, opResult) {
	if !receiver.IsValid() {
		return 0, opPanicError
	}
	recvType := receiver.Type()
	if recvType.Kind() == reflect.Pointer {
		recvType = recvType.Elem()
	}

	callee := lookupInnerCallee(desc, recvType)
	if callee == nil {
		callee = lookupCalleeForType(site, recvType)
		if callee == nil {
			var ok bool
			callee, ok = resolveInlineEvalCallee(vm, recvType)
			if !ok || callee == nil {
				return 0, opPanicError
			}
		}
		publishInnerCallee(desc, recvType, callee)
	}

	if callee.tinyLeafShape != tinyLeafNone {
		return invokeInlineEvalTinyLeaf(vm, registers, site, receiver, returnReg, callee)
	}
	return invokeInlineEvalFullFrame(vm, registers, site, receiver, envValue, returnReg, callee)
}

// lookupCalleeForType walks methodICSlots and returns the cached callee for the given
// concrete receiver type, or nil on miss.
//
// Takes site (*callSite) which holds the methodIC.
// Takes recvType (reflect.Type) which is the receiver's concrete dereferenced type.
//
// Returns the cached *CompiledFunction or nil.
func lookupCalleeForType(site *callSite, recvType reflect.Type) *CompiledFunction {
	for slotIndex := range site.methodICSlots {
		pointer := atomic.LoadPointer(&site.methodICSlots[slotIndex])
		if pointer == nil {
			continue
		}
		entry := (*monomorphicCacheEntry)(pointer)
		if entry.receiverType == recvType {
			return entry.callee
		}
	}
	return nil
}

// lookupInnerCallee walks the descriptor's innerCalleeSlots looking for a cached
// (recvType -> callee) entry. Pattern mirrors lookupCalleeForType /
// lookupInlineDescriptor: each slot is read with atomic.LoadPointer so concurrent
// publishers cannot tear the entry's (receiverType, callee) pair.
//
// Takes desc (*inlineDescriptor) whose innerCalleeSlots carries the memoised inner-Eval
// callee cache.
// Takes recvType (reflect.Type) which is the dereferenced concrete receiver type.
//
// Returns the cached *CompiledFunction on hit, nil on miss.
func lookupInnerCallee(desc *inlineDescriptor, recvType reflect.Type) *CompiledFunction {
	for slotIndex := range desc.innerCalleeSlots {
		pointer := atomic.LoadPointer(&desc.innerCalleeSlots[slotIndex])
		if pointer == nil {
			continue
		}
		entry := (*innerCalleeCacheEntry)(pointer)
		if entry.receiverType == recvType {
			return entry.callee
		}
	}
	return nil
}

// publishInnerCallee installs an inner-callee cache entry.
//
// Stores a fresh (recvType, callee) entry into the descriptor's innerCalleeSlots via
// atomic.StorePointer at the current round- robin victim slot. Pattern mirrors
// classifyAndCacheFromMethodIC's publish step. Concurrent writers may race on
// innerCalleeVictim but the tear only costs a sub-optimal eviction; correctness is
// preserved because each entry is immutable after publish and readers atomically load
// slot pointers.
//
// Takes desc (*inlineDescriptor) whose innerCalleeSlots receives the freshly resolved
// entry.
// Takes recvType (reflect.Type) which is the dereferenced concrete receiver type being
// cached.
// Takes callee (*CompiledFunction) which is the resolved inner-Eval target for recvType.
func publishInnerCallee(desc *inlineDescriptor, recvType reflect.Type, callee *CompiledFunction) {
	entry := &innerCalleeCacheEntry{receiverType: recvType, callee: callee}
	victim := desc.innerCalleeVictim & innerCalleeVictimMask
	desc.innerCalleeVictim++
	atomic.StorePointer(&desc.innerCalleeSlots[victim], unsafe.Pointer(entry))
}

// resolveInlineEvalCallee resolves the Eval method for recvType.
//
// Looks up the method through rootFunction's methodTable. Used when the outer site's
// methodICSlots misses on a concrete type (which happens regularly for the inner-Eval
// targets because the outer site only ever sees the tree root's type). Goes through
// resolveReceiverTypeName to handle piko's synthetic _pikoID_*-named struct types.
//
// Takes vm (*VM) which carries the rootFunction and functions table.
// Takes recvType (reflect.Type) which is the receiver type.
//
// Returns the resolved callee and ok=true, or nil and false.
func resolveInlineEvalCallee(vm *VM, recvType reflect.Type) (*CompiledFunction, bool) {
	if vm == nil || vm.rootFunction == nil || vm.rootFunction.methodTable == nil {
		return nil, false
	}
	typeName := resolveReceiverTypeName(vm, nil, recvType)
	if typeName == "" {
		return nil, false
	}
	funcIndex, ok := vm.rootFunction.methodTable[typeName+"."+inlineEvalMethodName]
	if !ok || int(funcIndex) >= len(vm.functions) {
		return nil, false
	}
	return vm.functions[funcIndex], true
}

// invokeInlineEvalTinyLeaf dispatches an inner Eval to a tinyLeaf.
//
// tinyLeaf bypasses pushCompiledFrame entirely, running the hand- coded inline body in
// the caller's frame and writing the result into registers.uints[returnReg], which is
// read back and returned.
//
// Temporarily swaps registers.general[site.arguments[0].register] to point at the inner
// receiver while runTinyLeafInline reads it, then restores. The env parameter is
// implicitly read by varNode- shape leaves from
// registers.general[site.arguments[1].register] which is NOT touched - the outer caller
// already placed env there when calling the binop method.
//
// Takes vm (*VM) which drives the tiny-leaf dispatch.
// Takes registers (*Registers) which holds the caller's banks.
// Takes site (*callSite) which describes the outer call.
// Takes receiver (reflect.Value) which is the inner receiver.
// Takes returnReg (uint8) which receives the result in uints.
// Takes callee (*CompiledFunction) which is the tinyLeaf target.
//
// Returns uint64 which is the inner uint result.
// Returns opResult which signals continuation or a panic.
func invokeInlineEvalTinyLeaf(vm *VM, registers *Registers, site *callSite, receiver reflect.Value, returnReg uint8, callee *CompiledFunction) (uint64, opResult) {
	savedRecv := registers.general[site.arguments[0].register]
	registers.general[site.arguments[0].register] = receiver
	res := runTinyLeafInline(vm, registers, site, callee)
	registers.general[site.arguments[0].register] = savedRecv
	if res != opContinue {
		return 0, res
	}
	return registers.uints[returnReg], opContinue
}

// invokeInlineEvalFullFrame dispatches an inner Eval via a full frame.
//
// Manually pushes a frame and places args directly into callee's general[0] and env's
// typed slot (bypassing copyCallArgs to avoid the outer-site-mismatch panic). Routes the
// inner return value to the outer caller's returnReg via the synthetic returnDestination
// (= site.returns), drives vm.run to completion, and reads the result back from
// registers.uints[returnReg].
//
// All inline-eval Eval methods share the (recv, env) -> uint signature with parameter[0]
// = general (interface receiver) and result = uint. Parameter[1] may be either
// registerGeneral (when the body taking env was demoted by the survivor walk - e.g.
// addNode.Eval passes env through an interface call that can't be resolved at compile
// time) or registerSliceUint (when the body passed env only to typed-bank-accepting
// callees, as in the constNode/varNode leaves). The manual placement handles both kinds
// by writing into the matching bank of the callee's frame. Surprise signatures fall
// through to the standard-path fallback.
//
// Takes vm (*VM) which drives the synchronous sub-dispatch loop.
// Takes registers (*Registers) which holds the caller's banks.
// Takes site (*callSite) which describes the outer call.
// Takes receiver (reflect.Value) which is the inner receiver.
// Takes envValue (reflect.Value) which is the env argument.
// Takes callee (*CompiledFunction) which is the inner Eval target.
//
// Returns uint64 which is the inner uint result.
// Returns opResult which signals continuation or a panic.
func invokeInlineEvalFullFrame(vm *VM, registers *Registers, site *callSite, receiver, envValue reflect.Value, _ uint8, callee *CompiledFunction) (uint64, opResult) {
	if vm.framePointer >= vm.callDepthLimit() {
		return 0, opStackOverflow
	}
	if !inlineEvalFullFrameCalleeAcceptable(callee) {
		return invokeInlineEvalViaStandardPath(vm, registers, site, callee)
	}

	vm.framePointer++
	if vm.framePointer >= len(vm.callStack) {
		vm.growCallStack()
	}
	pushedFrameIndex := vm.framePointer
	f := &vm.callStack[pushedFrameIndex]
	if vm.arena != nil {
		vm.arena.SaveInto(&f.arenaSave)
		callee.ensurePrecomputedAllocCounts()
		vm.arena.AllocRegistersIntoCached(&f.registers, callee.precomputedAllocCounts, callee.nonZeroBankMask)
	} else {
		f.registers = newRegisters(callee.numRegisters)
	}
	f.function = callee
	f.programCounter = 0
	f.returnDestination = site.returns
	f.deferBase = len(vm.deferStack)
	f.upvalues = nil
	f.hasGeneralAlloc = callee.numRegisters[registerGeneral] > 0
	f.registers.general[0] = receiver
	if !placeInlineEvalEnv(&f.registers, callee.parameterKinds[1], envValue, vm.arena) {
		vm.framePointer--
		return invokeInlineEvalViaStandardPath(vm, registers, site, callee)
	}

	previousFlag := vm.inlineDispatchExpectUintResult
	vm.inlineDispatchExpectUintResult = true
	_, err := vm.run(pushedFrameIndex)
	uintResult := vm.inlineDispatchUintResult
	vm.inlineDispatchExpectUintResult = previousFlag
	if err != nil {
		return 0, opPanicError
	}
	return uintResult, opContinue
}

// inlineEvalFullFrameCalleeAcceptable reports whether callee's signature and per-bank
// register budget match what invokeInlineEvalFullFrame can manually drive: 2-param
// (general receiver + general/slicesUint env), single uint return, with enough
// general/uint registers reserved and (for slicesUint env) a typed-slice slot available.
//
// Takes callee (*CompiledFunction) which is the candidate Eval.
//
// Returns bool which is true when the manual placement is safe.
func inlineEvalFullFrameCalleeAcceptable(callee *CompiledFunction) bool {
	if len(callee.parameterKinds) < 2 ||
		callee.parameterKinds[0] != registerGeneral ||
		!isInlineEvalEnvKind(callee.parameterKinds[1]) ||
		len(callee.resultKinds) < 1 ||
		callee.resultKinds[0] != registerUint {
		return false
	}
	if callee.numRegisters[registerGeneral] < 1 || callee.numRegisters[registerUint] < 1 {
		return false
	}
	if callee.parameterKinds[1] == registerGeneral && callee.numRegisters[registerGeneral] < 2 {
		return false
	}
	if callee.parameterKinds[1] == registerSliceUint && callee.numRegisters[registerSliceUint] < 1 {
		return false
	}
	return true
}

// isInlineEvalEnvKind reports whether kind is supported for env.
//
// The inline-eval family (constNode/varNode/addNode/...) declares env as []uint32, which
// the kind-classifier routes to registerSliceUint when the body's usage survives the
// typed-slice disqualifier walk and to registerGeneral when the body forces a demotion
// (e.g. passing env to an interface method whose callee cannot be statically resolved).
// Both kinds round-trip through the inline path; other kinds fall back to the standard
// path.
//
// Takes kind (registerKind) which is the callee's env parameter kind.
//
// Returns true when the inline path can place envValue into a slot of the given kind
// without crossing the reflect boundary twice.
func isInlineEvalEnvKind(kind registerKind) bool {
	return kind == registerGeneral || kind == registerSliceUint
}

// readInlineEnvAsReflect extracts the outer caller's env argument as a reflect.Value
// regardless of the bank it currently lives on. runInlineBinopUint and the tinyLeaf path
// treat envValue as a reflect.Value throughout, so the inline driver can carry env across
// recursive Eval calls without re-checking the bank at each hop.
//
// Takes registers (*Registers) which is the outer caller's frame.
// Takes argLocation (varLocation) which is site.arguments[1] for the outer Eval call.
//
// Returns the env value boxed as reflect.Value and true on supported banks;
// reflect.Value{} and false on unsupported banks (the inline driver then falls back to
// pushCompiledFrame).
func readInlineEnvAsReflect(registers *Registers, argLocation varLocation) (reflect.Value, bool) {
	switch argLocation.kind {
	case registerGeneral:
		return registers.general[argLocation.register], true
	case registerSliceUint:
		slice := registers.slicesUint[argLocation.register]
		if slice == nil {
			return reflect.Value{}, true
		}
		return reflect.ValueOf(slice), true
	case registerSliceInt:
		slice := registers.slicesInt[argLocation.register]
		if slice == nil {
			return reflect.Value{}, true
		}
		return reflect.ValueOf(slice), true
	default:
		return reflect.Value{}, false
	}
}

// placeInlineEvalEnv writes envValue into the callee's env slot using the bank dictated
// by the callee's parameterKinds[1]. The inline driver bypasses copyCallArgs, so this
// helper is the only place where the env-value -> callee-slot transfer happens.
//
// For the general case the receiver is written as the reflect.Value directly. For
// typed-slice banks the helper attempts a same-storage type assertion ([]uint64 /
// []int64) first; if that fails, it widens narrower element kinds via the
// unboxToTypedXxxSlice helpers so e.g. a []uint32 env arriving from the outer caller is
// preserved element-wise into the callee's uint64-backed typed bank.
//
// Takes registers (*Registers) which is the callee's frame.
// Takes envKind (registerKind) which selects the env slot bank.
// Takes envValue (reflect.Value) which carries the env value.
// Takes arena (*RegisterArena) which provides the widen backing.
//
// Returns true on success; false when the value cannot be placed (the caller must then
// unwind framePointer and route through the standard path).
func placeInlineEvalEnv(registers *Registers, envKind registerKind, envValue reflect.Value, arena *RegisterArena) bool {
	switch envKind {
	case registerGeneral:
		registers.general[1] = envValue
		return true
	case registerSliceUint:
		if !envValue.IsValid() {
			registers.slicesUint[0] = nil
			return true
		}
		typed := unboxToTypedUintSlice(envValue, arena)
		if typed == nil && envValue.Len() != 0 {
			return false
		}
		registers.slicesUint[0] = typed
		return true
	default:
		return false
	}
}

// invokeInlineEvalViaStandardPath is the safety fallback path.
//
// Used when the callee's signature does not match the expected (recv, env) -> uint shape,
// or the callee's pre-allocated bank cannot accommodate the manual placement. Pays the
// full pushCompiledFrame + copyCallArgs cost but stays correct in unusual edge cases
// (which the inline-eval classifier should never produce in practice).
//
// Takes vm (*VM) which drives the synchronous sub-dispatch loop.
// Takes registers (*Registers) which holds the caller's banks.
// Takes site (*callSite) which describes the outer call.
// Takes callee (*CompiledFunction) which is the inner Eval target.
//
// Returns uint64 which is the inner uint result.
// Returns opResult which signals continuation or a panic.
func invokeInlineEvalViaStandardPath(vm *VM, registers *Registers, site *callSite, callee *CompiledFunction) (uint64, opResult) {
	fpBefore := vm.framePointer
	res := pushCompiledFrame(vm, registers, site, callee)
	if res == opPanicError || res == opStackOverflow {
		return 0, res
	}
	if vm.framePointer > fpBefore {
		if _, err := vm.run(fpBefore); err != nil {
			return 0, opPanicError
		}
	}
	return registers.uints[site.returns[0].register], opContinue
}
