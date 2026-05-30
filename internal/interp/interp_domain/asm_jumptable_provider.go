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
	"piko.sh/piko/internal/interp/interp_domain/asm"
)

const (
	// bytesPerJumpTableSlot is the stride (in bytes) between consecutive uintptr entries in
	// the ASM dispatch jump tables. Each slot stores a handler address (uintptr = 8 bytes on
	// every supported target).
	bytesPerJumpTableSlot = 8

	// tier1JumpTableSymbol is the ASM symbol name for the tier-1 jump table.
	tier1JumpTableSymbol = "tier1JumpTable"

	// tier2JumpTableSymbol is the ASM symbol name for the tier-2 jump table.
	tier2JumpTableSymbol = "tier2JumpTable"

	// staticJumpTableEntriesCapacity is the initial capacity hint for the concatenated
	// static-entry slice. Sized to hold every existing entry without reallocation.
	staticJumpTableEntriesCapacity = 160

	// tier0EntriesCapacity is the initial capacity hint for the tier-0 entry slice. Sized to
	// comfortably hold every existing entry without reallocation; over-counting wastes a few
	// entry slots, under-counting causes one append-grow on initialisation.
	tier0EntriesCapacity = 80
)

// ProvideAsmHandlerJumpTableEntries returns the list of (handler, offset) pairs for every
// ASM handler that needs a jump-table installation. Asmgen iterates this list to emit
// initJumpTable's body.
//
// The Offset is computed at call time from the corresponding opcode's current iota value
// (multiplied by 8 bytes per slot). When the main opcode enum is reorganised, the offsets
// recompute automatically; there is no parallel jtOffsetXxx constant to maintain.
//
// Two categories appear: tier-0 handlers with one entry per opcode that has an ASM body
// (offset is int(opXxx) * bytesPerJumpTableSlot), and super-instruction handlers
// (handlerCallInline, handlerReturnInline, handlerTailCallExit) which reuse a main-enum
// opcode's slot pinned at the iota of opCall, opReturn and opTailCall respectively. Order
// within the slice does not matter for correctness; asmgen emits LEAQ/MOVQ pairs in the
// listed order, but every pair writes a different slot.
//
// Returns []asm.AsmHandlerJumpTableEntry which lists every handler installation entry,
// with tier-2 shim entries appended after the static set.
func ProvideAsmHandlerJumpTableEntries() []asm.AsmHandlerJumpTableEntry {
	staticEntries := buildStaticJumpTableEntries()
	shims := asm.Tier2HandlerShims()
	entries := make([]asm.AsmHandlerJumpTableEntry, len(staticEntries), len(staticEntries)+len(shims))
	copy(entries, staticEntries)
	return appendTier2ShimEntries(entries, shims)
}

// appendTier2ShimEntries appends tier-2 ASM-call shim install entries to entries.
//
// Each shim wraps a Go tier-2 handler so the common opContinue path resumes DISPATCH_NEXT
// in ASM (see asm/handlers_tier2_lift.go and tier2_shim_registry.go). The registry is
// populated at package init from tier2ShimRegistrations; iterating it here means adding a
// new shim only requires editing tier2ShimRegistrations.
//
// Takes entries ([]asm.AsmHandlerJumpTableEntry) which are the pre-existing static
// jump-table entries to extend.
// Takes shims ([]asm.Tier2HandlerShimSpec) which are the tier-2 shim specifications to
// append.
//
// Returns the entries slice with one appended item per shim specification.
func appendTier2ShimEntries(entries []asm.AsmHandlerJumpTableEntry, shims []asm.Tier2HandlerShimSpec) []asm.AsmHandlerJumpTableEntry {
	for _, spec := range shims {
		entries = append(entries, asm.AsmHandlerJumpTableEntry{
			Name:        spec.ShimSymbol,
			TableSymbol: spec.JumpTableSymbol,
			Offset:      spec.JumpTableOffset,
		})
	}
	return entries
}

// buildStaticJumpTableEntries concatenates every category of static jump-table entry
// (tier-0 ASM bodies, tier-1 sub-op shims, and tier-2 sub-op shims) into one slice in the
// order expected by the asmgen emitter.
//
// Returns the combined slice of static jump-table entries.
func buildStaticJumpTableEntries() []asm.AsmHandlerJumpTableEntry {
	entries := make([]asm.AsmHandlerJumpTableEntry, 0, staticJumpTableEntriesCapacity)
	entries = append(entries, tier0AsmEntries()...)
	entries = append(entries, tier1MoveAndControlEntries()...)
	entries = append(entries, tier1StructFieldEntries()...)
	entries = append(entries, tier2IncDecEntries()...)
	entries = append(entries, tier1SliceTypedEntries()...)
	entries = append(entries, tier1ComplexEntries()...)
	entries = append(entries, tier1MathTrigEntries()...)
	entries = append(entries, tier1StrconvEntries()...)
	entries = append(entries, tier1CapAndBoxEntries()...)
	entries = append(entries, tier1MakeSliceEntries()...)
	entries = append(entries, tier1ArithUnaryEntries()...)
	entries = append(entries, tier1ConversionEntries()...)
	entries = append(entries, tier1MathScalarEntries()...)
	entries = append(entries, tier1MiscEntries()...)
	return entries
}

// tier0AsmEntries returns one entry per opcode that still has a tier-0 ASM body. The
// offset is int(opXxx) * bytesPerJumpTableSlot.
//
// Returns the concatenated tier-0 jump-table entries.
func tier0AsmEntries() []asm.AsmHandlerJumpTableEntry {
	entries := make([]asm.AsmHandlerJumpTableEntry, 0, tier0EntriesCapacity)
	entries = append(entries, tier0LoadConstEntries()...)
	entries = append(entries, tier0IntArithEntries()...)
	entries = append(entries, tier0FloatArithEntries()...)
	entries = append(entries, tier0UintArithEntries()...)
	entries = append(entries, tier0ComparisonEntries()...)
	entries = append(entries, tier0StringEntries()...)
	entries = append(entries, tier0ControlFlowEntries()...)
	return entries
}

// tier0LoadConstEntries returns the tier-0 constant load entries.
//
// Returns the tier-0 load-constant entries.
func tier0LoadConstEntries() []asm.AsmHandlerJumpTableEntry {
	return []asm.AsmHandlerJumpTableEntry{
		{Name: "handlerLoadIntConst", Offset: int(opLoadIntConst) * bytesPerJumpTableSlot},
		{Name: "handlerLoadFloatConst", Offset: int(opLoadFloatConst) * bytesPerJumpTableSlot},
		{Name: "handlerLoadStringConst", Offset: int(opLoadStringConst) * bytesPerJumpTableSlot},
		{Name: "handlerLoadBoolConst", Offset: int(opLoadBoolConst) * bytesPerJumpTableSlot},
	}
}

// tier0IntArithEntries returns the int arithmetic and bitwise tier-0 entries.
//
// Returns the tier-0 int arithmetic and bitwise entries.
func tier0IntArithEntries() []asm.AsmHandlerJumpTableEntry {
	return []asm.AsmHandlerJumpTableEntry{
		{Name: "handlerAddInt", Offset: int(opAddInt) * bytesPerJumpTableSlot},
		{Name: "handlerSubInt", Offset: int(opSubInt) * bytesPerJumpTableSlot},
		{Name: "handlerMulInt", Offset: int(opMulInt) * bytesPerJumpTableSlot},
		{Name: "handlerDivInt", Offset: int(opDivInt) * bytesPerJumpTableSlot},
		{Name: "handlerRemInt", Offset: int(opRemInt) * bytesPerJumpTableSlot},
		{Name: "handlerBitAnd", Offset: int(opBitAnd) * bytesPerJumpTableSlot},
		{Name: "handlerBitOr", Offset: int(opBitOr) * bytesPerJumpTableSlot},
		{Name: "handlerBitXor", Offset: int(opBitXor) * bytesPerJumpTableSlot},
		{Name: "handlerBitAndNot", Offset: int(opBitAndNot) * bytesPerJumpTableSlot},
		{Name: "handlerShiftLeft", Offset: int(opShiftLeft) * bytesPerJumpTableSlot},
		{Name: "handlerShiftRight", Offset: int(opShiftRight) * bytesPerJumpTableSlot},
	}
}

// tier0FloatArithEntries returns the float arithmetic tier-0 entries.
//
// Returns the tier-0 float arithmetic entries.
func tier0FloatArithEntries() []asm.AsmHandlerJumpTableEntry {
	return []asm.AsmHandlerJumpTableEntry{
		{Name: "handlerAddFloat", Offset: int(opAddFloat) * bytesPerJumpTableSlot},
		{Name: "handlerSubFloat", Offset: int(opSubFloat) * bytesPerJumpTableSlot},
		{Name: "handlerMulFloat", Offset: int(opMulFloat) * bytesPerJumpTableSlot},
		{Name: "handlerDivFloat", Offset: int(opDivFloat) * bytesPerJumpTableSlot},
	}
}

// tier0UintArithEntries returns the uint arithmetic and bitwise tier-0 entries.
//
// Returns the tier-0 uint arithmetic and bitwise entries.
func tier0UintArithEntries() []asm.AsmHandlerJumpTableEntry {
	return []asm.AsmHandlerJumpTableEntry{
		{Name: "handlerAddUint", Offset: int(opAddUint) * bytesPerJumpTableSlot},
		{Name: "handlerSubUint", Offset: int(opSubUint) * bytesPerJumpTableSlot},
		{Name: "handlerMulUint", Offset: int(opMulUint) * bytesPerJumpTableSlot},
		{Name: "handlerBitAndUint", Offset: int(opBitAndUint) * bytesPerJumpTableSlot},
		{Name: "handlerBitOrUint", Offset: int(opBitOrUint) * bytesPerJumpTableSlot},
		{Name: "handlerBitXorUint", Offset: int(opBitXorUint) * bytesPerJumpTableSlot},
		{Name: "handlerBitAndNotUint", Offset: int(opBitAndNotUint) * bytesPerJumpTableSlot},
		{Name: "handlerShiftLeftUint", Offset: int(opShiftLeftUint) * bytesPerJumpTableSlot},
		{Name: "handlerShiftRightUint", Offset: int(opShiftRightUint) * bytesPerJumpTableSlot},
	}
}

// tier0ComparisonEntries returns the tier-0 comparison entries for uint, int and float.
//
// Returns the tier-0 comparison entries.
func tier0ComparisonEntries() []asm.AsmHandlerJumpTableEntry {
	return []asm.AsmHandlerJumpTableEntry{
		{Name: "handlerEqUint", Offset: int(opEqUint) * bytesPerJumpTableSlot},
		{Name: "handlerNeUint", Offset: int(opNeUint) * bytesPerJumpTableSlot},
		{Name: "handlerLtUint", Offset: int(opLtUint) * bytesPerJumpTableSlot},
		{Name: "handlerLeUint", Offset: int(opLeUint) * bytesPerJumpTableSlot},
		{Name: "handlerGtUint", Offset: int(opGtUint) * bytesPerJumpTableSlot},
		{Name: "handlerGeUint", Offset: int(opGeUint) * bytesPerJumpTableSlot},
		{Name: "handlerEqInt", Offset: int(opEqInt) * bytesPerJumpTableSlot},
		{Name: "handlerNeInt", Offset: int(opNeInt) * bytesPerJumpTableSlot},
		{Name: "handlerLtInt", Offset: int(opLtInt) * bytesPerJumpTableSlot},
		{Name: "handlerLeInt", Offset: int(opLeInt) * bytesPerJumpTableSlot},
		{Name: "handlerGtInt", Offset: int(opGtInt) * bytesPerJumpTableSlot},
		{Name: "handlerGeInt", Offset: int(opGeInt) * bytesPerJumpTableSlot},
		{Name: "handlerEqFloat", Offset: int(opEqFloat) * bytesPerJumpTableSlot},
		{Name: "handlerNeFloat", Offset: int(opNeFloat) * bytesPerJumpTableSlot},
		{Name: "handlerLtFloat", Offset: int(opLtFloat) * bytesPerJumpTableSlot},
		{Name: "handlerLeFloat", Offset: int(opLeFloat) * bytesPerJumpTableSlot},
		{Name: "handlerGtFloat", Offset: int(opGtFloat) * bytesPerJumpTableSlot},
		{Name: "handlerGeFloat", Offset: int(opGeFloat) * bytesPerJumpTableSlot},
	}
}

// tier0StringEntries returns the tier-0 string-handling entries.
//
// Returns the tier-0 string-handling entries.
func tier0StringEntries() []asm.AsmHandlerJumpTableEntry {
	return []asm.AsmHandlerJumpTableEntry{
		{Name: "handlerEqString", Offset: int(opEqString) * bytesPerJumpTableSlot},
		{Name: "handlerNeString", Offset: int(opNeString) * bytesPerJumpTableSlot},
		{Name: "handlerStringIndex", Offset: int(opStringIndex) * bytesPerJumpTableSlot},
		{Name: "handlerSliceString", Offset: int(opSliceString) * bytesPerJumpTableSlot},
		{Name: "handlerStringIndexToInt", Offset: int(opStringIndexToInt) * bytesPerJumpTableSlot},
	}
}

// tier0ControlFlowEntries returns the tier-0 control-flow entries (jumps, const-fused
// compares, calls and the narrow truncation).
//
// Returns the tier-0 control-flow entries.
func tier0ControlFlowEntries() []asm.AsmHandlerJumpTableEntry {
	return []asm.AsmHandlerJumpTableEntry{
		{Name: "handlerJumpIfTrue", Offset: int(opJumpIfTrue) * bytesPerJumpTableSlot},
		{Name: "handlerJumpIfFalse", Offset: int(opJumpIfFalse) * bytesPerJumpTableSlot},
		{Name: "handlerSubIntConst", Offset: int(opSubIntConst) * bytesPerJumpTableSlot},
		{Name: "handlerAddIntConst", Offset: int(opAddIntConst) * bytesPerJumpTableSlot},
		{Name: "handlerLeIntConstJumpFalse", Offset: int(opLeIntConstJumpFalse) * bytesPerJumpTableSlot},
		{Name: "handlerLtIntConstJumpFalse", Offset: int(opLtIntConstJumpFalse) * bytesPerJumpTableSlot},
		{Name: "handlerEqIntConstJumpFalse", Offset: int(opEqIntConstJumpFalse) * bytesPerJumpTableSlot},
		{Name: "handlerEqIntConstJumpTrue", Offset: int(opEqIntConstJumpTrue) * bytesPerJumpTableSlot},
		{Name: "handlerGeIntConstJumpFalse", Offset: int(opGeIntConstJumpFalse) * bytesPerJumpTableSlot},
		{Name: "handlerGtIntConstJumpFalse", Offset: int(opGtIntConstJumpFalse) * bytesPerJumpTableSlot},
		{Name: "handlerMulIntConst", Offset: int(opMulIntConst) * bytesPerJumpTableSlot},
		{Name: "handlerAddIntJump", Offset: int(opAddIntJump) * bytesPerJumpTableSlot},
		{Name: "handlerCallInline", Offset: int(opCall) * bytesPerJumpTableSlot},
		{Name: "handlerCallInlineScalar", Offset: int(opCallScalar) * bytesPerJumpTableSlot},
		{Name: "handlerTailCallInline", Offset: int(opTailCall) * bytesPerJumpTableSlot},
		{Name: "handlerTruncateNarrow", Offset: int(opTruncateNarrow) * bytesPerJumpTableSlot},
	}
}

// tier1MoveAndControlEntries returns tier-1 sub-op handlers for move, jump,
// load-const-small, load-bool, fused increment/loop, and length-string-lt-jump-false.
//
// Returns the tier-1 move and control-flow entries.
func tier1MoveAndControlEntries() []asm.AsmHandlerJumpTableEntry {
	return []asm.AsmHandlerJumpTableEntry{
		{Name: "handlerSubOpMoveInt", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpMoveInt) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpMoveFloat", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpMoveFloat) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpMoveBool", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpMoveBool) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpMoveUint", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpMoveUint) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpMoveString", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpMoveString) * bytesPerJumpTableSlot},
		{Name: "handlerJump", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpJump) * bytesPerJumpTableSlot},
		{Name: "handlerLoadIntConstSmall", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpLoadIntConstSmall) * bytesPerJumpTableSlot},
		{Name: "handlerLoadBool", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpLoadBool) * bytesPerJumpTableSlot},
		{Name: "handlerIncIntJumpLt", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpIncIntJumpLt) * bytesPerJumpTableSlot},
		{Name: "handlerLenStringLtJumpFalse", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpLenStringLtJumpFalse) * bytesPerJumpTableSlot},
	}
}

// tier2IncDecEntries returns tier-2 sub-op handlers for return and increment/decrement on
// int/uint.
//
// Returns the tier-2 return and inc/dec entries.
func tier2IncDecEntries() []asm.AsmHandlerJumpTableEntry {
	return []asm.AsmHandlerJumpTableEntry{
		{Name: "handlerReturnInline", TableSymbol: tier2JumpTableSymbol, Offset: int(subOpTier2Return) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpTier2IncInt", TableSymbol: tier2JumpTableSymbol, Offset: int(subOpTier2IncInt) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpTier2DecInt", TableSymbol: tier2JumpTableSymbol, Offset: int(subOpTier2DecInt) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpTier2IncUint", TableSymbol: tier2JumpTableSymbol, Offset: int(subOpTier2IncUint) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpTier2DecUint", TableSymbol: tier2JumpTableSymbol, Offset: int(subOpTier2DecUint) * bytesPerJumpTableSlot},
	}
}

// tier1StructFieldEntries returns tier-1 sub-op shims for fused inc/dec struct-field
// super-instructions. Only the shim is installed; the real handler is called from the
// shim.
//
// Returns the tier-1 struct-field inc/dec entries.
func tier1StructFieldEntries() []asm.AsmHandlerJumpTableEntry {
	return []asm.AsmHandlerJumpTableEntry{
		{Name: "handlerSubOpIncStructFieldInt", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpIncStructFieldInt) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpDecStructFieldInt", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpDecStructFieldInt) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpIncStructFieldUint", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpIncStructFieldUint) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpDecStructFieldUint", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpDecStructFieldUint) * bytesPerJumpTableSlot},
	}
}

// tier1SliceTypedEntries returns tier-1 sub-op handlers for typed slice length, get/set,
// byte-slice slicing, range-next byte, and range-check uint.
//
// Returns the tier-1 typed-slice entries.
func tier1SliceTypedEntries() []asm.AsmHandlerJumpTableEntry {
	return []asm.AsmHandlerJumpTableEntry{
		{Name: "handlerSubOpLenSliceIntDirect", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpLenSliceIntDirect) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpLenSliceFloatDirect", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpLenSliceFloatDirect) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpLenSliceStringDirect", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpLenSliceStringDirect) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpLenSliceBoolDirect", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpLenSliceBoolDirect) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpLenSliceUintDirect", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpLenSliceUintDirect) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpSliceGetFloatDirect", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpSliceGetFloatDirect) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpSliceSetFloatDirect", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpSliceSetFloatDirect) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpSliceGetUintDirect", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpSliceGetUintDirect) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpSliceSetUintDirect", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpSliceSetUintDirect) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpSliceGetBoolDirect", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpSliceGetBoolDirect) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpSliceSetBoolDirect", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpSliceSetBoolDirect) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpSliceGetStringDirect", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpSliceGetStringDirect) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpSliceSetStringDirect", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpSliceSetStringDirect) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpLenSliceByteDirect", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpLenSliceByteDirect) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpSliceGetByteDirect", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpSliceGetByteDirect) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpSliceSetByteDirect", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpSliceSetByteDirect) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpSliceByteSlice", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpSliceByteSlice) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpRangeNextSliceByte", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpRangeNextSliceByte) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpRangeCheckUintJumpFalse", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpRangeCheckUintJumpFalse) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpMoveSliceInt", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpMoveSliceInt) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpMoveSliceFloat", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpMoveSliceFloat) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpMoveSliceString", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpMoveSliceString) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpMoveSliceBool", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpMoveSliceBool) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpMoveSliceUint", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpMoveSliceUint) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpMoveSliceByte", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpMoveSliceByte) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpSliceSliceIntDirect", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpSliceSliceIntDirect) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpSliceSliceFloatDirect", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpSliceSliceFloatDirect) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpSliceSliceStringDirect", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpSliceSliceStringDirect) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpSliceSliceBoolDirect", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpSliceSliceBoolDirect) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpSliceSliceUintDirect", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpSliceSliceUintDirect) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpAppendSliceIntDirect", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpAppendSliceIntDirect) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpAppendSliceFloatDirect", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpAppendSliceFloatDirect) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpAppendSliceStringDirect", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpAppendSliceStringDirect) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpAppendSliceBoolDirect", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpAppendSliceBoolDirect) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpAppendSliceUintDirect", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpAppendSliceUintDirect) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpAppendSliceByteDirect", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpAppendSliceByteDirect) * bytesPerJumpTableSlot},
	}
}

// tier1ComplexEntries returns tier-1 sub-op handlers for complex number operations.
//
// Returns the tier-1 complex-number entries.
func tier1ComplexEntries() []asm.AsmHandlerJumpTableEntry {
	return []asm.AsmHandlerJumpTableEntry{
		{Name: "handlerSubOpRealComplex", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpRealComplex) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpImagComplex", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpImagComplex) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpMoveComplex", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpMoveComplex) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpNegComplex", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpNegComplex) * bytesPerJumpTableSlot},
	}
}

// tier1MathTrigEntries returns tier-1 sub-op handlers for the math-package trigonometric
// and exponential functions.
//
// Returns the tier-1 trig and exponential entries.
func tier1MathTrigEntries() []asm.AsmHandlerJumpTableEntry {
	return []asm.AsmHandlerJumpTableEntry{
		{Name: "handlerSubOpMathSin", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpMathSin) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpMathCos", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpMathCos) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpMathExp", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpMathExp) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpMathTan", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpMathTan) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpMathMod", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpMathMod) * bytesPerJumpTableSlot},
	}
}

// tier1StrconvEntries returns tier-1 sub-op handlers for the strconv package conversion
// intrinsics.
//
// Returns the tier-1 strconv conversion entries.
func tier1StrconvEntries() []asm.AsmHandlerJumpTableEntry {
	return []asm.AsmHandlerJumpTableEntry{
		{Name: "handlerSubOpStrconvFormatBool", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpStrconvFormatBool) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpStrconvItoa", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpStrconvItoa) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpStrconvFormatInt", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpStrconvFormatInt) * bytesPerJumpTableSlot},
	}
}

// tier1CapAndBoxEntries returns tier-1 sub-op handlers for cap, bytes-to-string
// conversion, and typed slice box/unbox.
//
// Returns the tier-1 cap and box/unbox entries.
func tier1CapAndBoxEntries() []asm.AsmHandlerJumpTableEntry {
	return []asm.AsmHandlerJumpTableEntry{
		{Name: "handlerSubOpCap", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpCap) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpBytesToString", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpBytesToString) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpBoxSliceInt", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpBoxSliceInt) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpUnboxSliceInt", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpUnboxSliceInt) * bytesPerJumpTableSlot},
	}
}

// tier1MakeSliceEntries returns tier-1 sub-op handlers for the typed make-slice
// intrinsics.
//
// Returns the tier-1 typed make-slice entries.
func tier1MakeSliceEntries() []asm.AsmHandlerJumpTableEntry {
	return tier1EntriesFromPairs([]tier1HandlerPair{
		{Name: "handlerSubOpMakeSliceInt", SubOp: subOpMakeSliceInt},
		{Name: "handlerSubOpMakeSliceFloat", SubOp: subOpMakeSliceFloat},
		{Name: "handlerSubOpMakeSliceString", SubOp: subOpMakeSliceString},
		{Name: "handlerSubOpMakeSliceBool", SubOp: subOpMakeSliceBool},
		{Name: "handlerSubOpMakeSliceUint", SubOp: subOpMakeSliceUint},
		{Name: "handlerSubOpMakeSliceByte", SubOp: subOpMakeSliceByte},
	})
}

// tier1HandlerPair binds a handler symbol name to its tier-1 sub-opcode for one
// jump-table slot. Used by helpers that share a dense list of (name, subOp) pairs so the
// asm.AsmHandlerJumpTableEntry construction is centralised in tier1EntriesFromPairs.
type tier1HandlerPair struct {
	// Name holds the handler symbol name installed in the jump table.
	Name string

	// SubOp holds the tier-1 sub-opcode whose iota provides the slot offset.
	SubOp subOpcode
}

// tier1EntriesFromPairs converts a list of tier-1 handler pairs into the full
// asm.AsmHandlerJumpTableEntry form, populating the table symbol and computing each
// offset from the sub-opcode iota.
//
// Takes pairs ([]tier1HandlerPair) which are the tier-1 handler name and sub-opcode pairs
// to convert.
//
// Returns one jump-table entry per input pair, in the same order.
func tier1EntriesFromPairs(pairs []tier1HandlerPair) []asm.AsmHandlerJumpTableEntry {
	entries := make([]asm.AsmHandlerJumpTableEntry, len(pairs))
	for index, pair := range pairs {
		entries[index] = asm.AsmHandlerJumpTableEntry{
			Name:        pair.Name,
			TableSymbol: tier1JumpTableSymbol,
			Offset:      int(pair.SubOp) * bytesPerJumpTableSlot,
		}
	}
	return entries
}

// tier1ArithUnaryEntries returns tier-1 sub-op handlers for unary negation and bit-not.
//
// Returns the tier-1 unary arithmetic entries.
func tier1ArithUnaryEntries() []asm.AsmHandlerJumpTableEntry {
	return []asm.AsmHandlerJumpTableEntry{
		{Name: "handlerSubOpNegInt", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpNegInt) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpNegFloat", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpNegFloat) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpBitNot", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpBitNot) * bytesPerJumpTableSlot},
	}
}

// tier1ConversionEntries returns tier-1 sub-op handlers for numeric conversions between
// int/uint and float.
//
// Returns the tier-1 numeric conversion entries.
func tier1ConversionEntries() []asm.AsmHandlerJumpTableEntry {
	return []asm.AsmHandlerJumpTableEntry{
		{Name: "handlerSubOpIntToFloat", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpIntToFloat) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpFloatToInt", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpFloatToInt) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpUintToFloat", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpUintToFloat) * bytesPerJumpTableSlot},
		{Name: "handlerSubOpFloatToUint", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpFloatToUint) * bytesPerJumpTableSlot},
	}
}

// tier1MathScalarEntries returns tier-1 sub-op handlers for the math-package scalar
// functions (sqrt, abs, floor, ceil, trunc, round).
//
// Returns the tier-1 math scalar entries.
func tier1MathScalarEntries() []asm.AsmHandlerJumpTableEntry {
	return tier1EntriesFromPairs([]tier1HandlerPair{
		{Name: "handlerSubOpMathSqrt", SubOp: subOpMathSqrt},
		{Name: "handlerSubOpMathAbs", SubOp: subOpMathAbs},
		{Name: "handlerSubOpMathFloor", SubOp: subOpMathFloor},
		{Name: "handlerSubOpMathCeil", SubOp: subOpMathCeil},
		{Name: "handlerSubOpMathTrunc", SubOp: subOpMathTrunc},
		{Name: "handlerSubOpMathRound", SubOp: subOpMathRound},
	})
}

// tier1MiscEntries returns the remaining tier-1 sub-op handlers that do not belong to any
// of the focused categories.
//
// Returns the remaining miscellaneous tier-1 entries.
func tier1MiscEntries() []asm.AsmHandlerJumpTableEntry {
	return []asm.AsmHandlerJumpTableEntry{
		{Name: "handlerSubOpLenString", TableSymbol: tier1JumpTableSymbol, Offset: int(subOpLenString) * bytesPerJumpTableSlot},
	}
}
