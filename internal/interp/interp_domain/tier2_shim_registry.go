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
	// expectedTier2ShimCount is the pre-allocation hint for the shim spec slice; sized
	// comfortably above the current entry count so the underlying array does not grow when
	// categories are added.
	expectedTier2ShimCount = 96
)

// makeShimSpec builds a Tier2HandlerShimSpec using the standard naming convention.
//
// ShimSymbol is "handlerTier2Shim<suffix>", TrampolineSymbol is the Plan-9 form prefixed
// with a package separator before "asmCall<handlerName>(SB)", and JumpTableOffset is the
// opcode's position scaled by bytesPerJumpTableSlot. All registered entries are
// non-frame-changing so NeedsFrameRebuild stays false.
//
// Takes registration (shimRegistration) which supplies the opcode, handler name, shim
// suffix, and narrow flag.
//
// Returns asm.Tier2HandlerShimSpec ready to append to the registry.
func makeShimSpec(registration shimRegistration) asm.Tier2HandlerShimSpec {
	return asm.Tier2HandlerShimSpec{
		ShimSymbol:        "handlerTier2Shim" + registration.ShimSuffix,
		TrampolineSymbol:  "·asmCall" + registration.HandlerName + "(SB)",
		NeedsFrameRebuild: false,
		IsNarrow:          registration.Narrow,
		JumpTableSymbol:   "",
		JumpTableOffset:   int(registration.Op) * bytesPerJumpTableSlot,
	}
}

// shimRegistration pairs an opcode with the matching handler, shim suffix, and narrow
// flag used by makeShimSpec. Declaring every shim as one dense table (tier2ShimRegistry)
// keeps the registration order explicit and avoids the per-category boilerplate.
type shimRegistration struct {
	// HandlerName is the trampoline base name; the full Plan-9 trampoline symbol prefixes a
	// package separator before "asmCall<HandlerName>(SB)".
	HandlerName string

	// ShimSuffix is the suffix appended to "handlerTier2Shim".
	ShimSuffix string

	// Op is the opcode whose tier-0 jump-table slot the shim populates.
	Op opcode

	// Narrow tags handlers that do not read or write frame.programCounter, selecting the
	// slim shim body + Go trampoline pair (skipping the pre-CALL CTX_PC / CTX_SAVED_PC
	// writes and the frame.programCounter sync/writeback). The narrow path roughly halves
	// the per-call trampoline self-cost on the opContinue branch.
	Narrow bool
}

var (
	// tier2ShimRegistry enumerates every tier-2 ASM-call shim in registration order.
	//
	// Each entry feeds into both:
	//
	//  1. asm.tier2HandlerShims, consumed by asmgen at code-generation time to emit the
	//     per-shim ASM body via the BytecodeArchitecturePort.EmitTier2CallShim primitive.
	//  2. The runtime jump-table install path (ProvideAsmHandlerJumpTableEntries iterates
	//     the registry to append shim install entries after the static set).
	//
	// The registry uses opcode iota values directly so the byte offset updates automatically
	// if the opcode order changes in opcode.go. Ordering note: opAppend / opAppendByteFast
	// keep their direct-exit handlers (preserve batching); opTruncateNarrow is ASM-lifted;
	// opMakeMap stays a tier-1 sub-op; none are registered here.
	tier2ShimRegistry = []shimRegistration{
		// Data access: indexing, map access, type assertion.
		{HandlerName: "HandleTypeAssert", ShimSuffix: "TypeAssert", Op: opTypeAssert},
		{HandlerName: "HandleIndex", ShimSuffix: "Index", Op: opIndex},
		{HandlerName: "HandleIndexSet", ShimSuffix: "IndexSet", Op: opIndexSet},
		{HandlerName: "HandleMapIndexOk", ShimSuffix: "MapIndexOk", Op: opMapIndexOk},
		{HandlerName: "HandleMapSet", ShimSuffix: "MapSet", Op: opMapSet},

		// Slice / append.
		{HandlerName: "HandleAppendSpread", ShimSuffix: "AppendSpread", Op: opAppendSpread},
		{HandlerName: "HandleCopy", ShimSuffix: "Copy", Op: opCopy},
		{HandlerName: "HandleSliceOp", ShimSuffix: "SliceOp", Op: opSliceOp},
		{HandlerName: "HandleSliceGetInt", ShimSuffix: "SliceGetInt", Op: opSliceGetInt},
		{HandlerName: "HandleSliceSetInt", ShimSuffix: "SliceSetInt", Op: opSliceSetInt},
		{HandlerName: "HandleSliceGetFloat", ShimSuffix: "SliceGetFloat", Op: opSliceGetFloat},
		{HandlerName: "HandleSliceSetFloat", ShimSuffix: "SliceSetFloat", Op: opSliceSetFloat},
		{HandlerName: "HandleSliceGetString", ShimSuffix: "SliceGetString", Op: opSliceGetString},
		{HandlerName: "HandleSliceSetString", ShimSuffix: "SliceSetString", Op: opSliceSetString},
		{HandlerName: "HandleSliceGetBool", ShimSuffix: "SliceGetBool", Op: opSliceGetBool},
		{HandlerName: "HandleSliceSetBool", ShimSuffix: "SliceSetBool", Op: opSliceSetBool},
		{HandlerName: "HandleSliceGetUint", ShimSuffix: "SliceGetUint", Op: opSliceGetUint},
		{HandlerName: "HandleSliceSetUint", ShimSuffix: "SliceSetUint", Op: opSliceSetUint},

		// Value conversion.
		{HandlerName: "HandleAddr", ShimSuffix: "Addr", Op: opAddr},
		{HandlerName: "HandleDeref", ShimSuffix: "Deref", Op: opDeref},
		{HandlerName: "HandleConvert", ShimSuffix: "Convert", Op: opConvert},
		{HandlerName: "HandlePackInterface", ShimSuffix: "PackInterface", Op: opPackInterface},
		{HandlerName: "HandleUnpackInterface", ShimSuffix: "UnpackInterface", Op: opUnpackInterface},

		// Value constructors.
		{HandlerName: "HandleMakeSlice", ShimSuffix: "MakeSlice", Op: opMakeSlice},
		{HandlerName: "HandleMakeClosure", ShimSuffix: "MakeClosure", Op: opMakeClosure},
		{HandlerName: "HandleAllocIndirect", ShimSuffix: "AllocIndirect", Op: opAllocIndirect},

		// Globals, upvalues, shared cells.
		{HandlerName: "HandleGetGlobal", ShimSuffix: "GetGlobal", Op: opGetGlobal},
		{HandlerName: "HandleSetGlobal", ShimSuffix: "SetGlobal", Op: opSetGlobal},
		{HandlerName: "HandleGetGlobalWide", ShimSuffix: "GetGlobalWide", Op: opGetGlobalWide},
		{HandlerName: "HandleSetGlobalWide", ShimSuffix: "SetGlobalWide", Op: opSetGlobalWide},
		{HandlerName: "HandleGetUpvalue", ShimSuffix: "GetUpvalue", Op: opGetUpvalue},
		{HandlerName: "HandleSetUpvalue", ShimSuffix: "SetUpvalue", Op: opSetUpvalue},
		{HandlerName: "HandleSyncClosureUpvalues", ShimSuffix: "SyncClosureUpvalues", Op: opSyncClosureUpvalues},
		{HandlerName: "HandleResetSharedCell", ShimSuffix: "ResetSharedCell", Op: opResetSharedCell},
		{HandlerName: "HandleWriteSharedCell", ShimSuffix: "WriteSharedCell", Op: opWriteSharedCell},
		{HandlerName: "HandleBindMethod", ShimSuffix: "BindMethod", Op: opBindMethod},

		// Range / iteration.
		{HandlerName: "HandleRangeInit", ShimSuffix: "RangeInit", Op: opRangeInit},
		{HandlerName: "HandleRangeNext", ShimSuffix: "RangeNext", Op: opRangeNext},

		// General-bank moves and comparisons.
		{HandlerName: "HandleMoveGeneral", ShimSuffix: "MoveGeneral", Op: opMoveGeneral},
		{HandlerName: "HandleLoadGeneralConst", ShimSuffix: "LoadGeneralConst", Op: opLoadGeneralConst},
		{HandlerName: "HandleEqGeneral", ShimSuffix: "EqGeneral", Op: opEqGeneral},
		{HandlerName: "HandleNeGeneral", ShimSuffix: "NeGeneral", Op: opNeGeneral},
		{HandlerName: "HandleLtGeneral", ShimSuffix: "LtGeneral", Op: opLtGeneral},
		{HandlerName: "HandleLeGeneral", ShimSuffix: "LeGeneral", Op: opLeGeneral},
		{HandlerName: "HandleGtGeneral", ShimSuffix: "GtGeneral", Op: opGtGeneral},
		{HandlerName: "HandleGeGeneral", ShimSuffix: "GeGeneral", Op: opGeGeneral},
		{HandlerName: "HandleEqInterfaceNil", ShimSuffix: "EqInterfaceNil", Op: opEqInterfaceNil},
		{HandlerName: "HandleNeInterfaceNil", ShimSuffix: "NeInterfaceNil", Op: opNeInterfaceNil},

		// Allocation-heavy string builtins.
		{HandlerName: "HandleConcatString", ShimSuffix: "ConcatString", Op: opConcatString},
		{HandlerName: "HandleConcatRuneString", ShimSuffix: "ConcatRuneString", Op: opConcatRuneString},
		{HandlerName: "HandleStrContains", ShimSuffix: "StrContains", Op: opStrContains},
		{HandlerName: "HandleStrContainsRune", ShimSuffix: "StrContainsRune", Op: opStrContainsRune},
		{HandlerName: "HandleStrHasPrefix", ShimSuffix: "StrHasPrefix", Op: opStrHasPrefix},
		{HandlerName: "HandleStrHasSuffix", ShimSuffix: "StrHasSuffix", Op: opStrHasSuffix},
		{HandlerName: "HandleStrEqualFold", ShimSuffix: "StrEqualFold", Op: opStrEqualFold},
		{HandlerName: "HandleStrIndex", ShimSuffix: "StrIndex", Op: opStrIndex},
		{HandlerName: "HandleStrIndexRune", ShimSuffix: "StrIndexRune", Op: opStrIndexRune},
		{HandlerName: "HandleStrLastIndex", ShimSuffix: "StrLastIndex", Op: opStrLastIndex},
		{HandlerName: "HandleStrCount", ShimSuffix: "StrCount", Op: opStrCount},
		{HandlerName: "HandleStrTrim", ShimSuffix: "StrTrim", Op: opStrTrim},
		{HandlerName: "HandleStrTrimPrefix", ShimSuffix: "StrTrimPrefix", Op: opStrTrimPrefix},
		{HandlerName: "HandleStrTrimSuffix", ShimSuffix: "StrTrimSuffix", Op: opStrTrimSuffix},
		{HandlerName: "HandleStrRepeat", ShimSuffix: "StrRepeat", Op: opStrRepeat},
		{HandlerName: "HandleStrJoin", ShimSuffix: "StrJoin", Op: opStrJoin},
		{HandlerName: "HandleStrSplit", ShimSuffix: "StrSplit", Op: opStrSplit},
		{HandlerName: "HandleStrReplaceAll", ShimSuffix: "StrReplaceAll", Op: opStrReplaceAll},

		// Math, complex numbers, unsafe pointers.
		{HandlerName: "HandleMathPow", ShimSuffix: "MathPow", Op: opMathPow},
		{HandlerName: "HandleBuildComplex", ShimSuffix: "BuildComplex", Op: opBuildComplex},
		{HandlerName: "HandleAddComplex", ShimSuffix: "AddComplex", Op: opAddComplex},
		{HandlerName: "HandleSubComplex", ShimSuffix: "SubComplex", Op: opSubComplex},
		{HandlerName: "HandleMulComplex", ShimSuffix: "MulComplex", Op: opMulComplex},
		{HandlerName: "HandleDivComplex", ShimSuffix: "DivComplex", Op: opDivComplex},
		{HandlerName: "HandleEqComplex", ShimSuffix: "EqComplex", Op: opEqComplex},
		{HandlerName: "HandleNeComplex", ShimSuffix: "NeComplex", Op: opNeComplex},
		{HandlerName: "HandleLoadComplexConst", ShimSuffix: "LoadComplexConst", Op: opLoadComplexConst},
		{HandlerName: "HandleUnsafeString", ShimSuffix: "UnsafeString", Op: opUnsafeString},
		{HandlerName: "HandleUnsafeSlice", ShimSuffix: "UnsafeSlice", Op: opUnsafeSlice},
		{HandlerName: "HandleUnsafeAdd", ShimSuffix: "UnsafeAdd", Op: opUnsafeAdd},

		// Typed maps (int/string keys, int/string/general values) plus the fused
		// MapIndexOk+JumpIfFalse super-ops. The fused handler updates frame.programCounter on
		// !ok; the shim writes it back to CTX_PC.
		{HandlerName: "HandleMapGetIntInt", ShimSuffix: "MapGetIntInt", Op: opMapGetIntInt},
		{HandlerName: "HandleMapSetIntInt", ShimSuffix: "MapSetIntInt", Op: opMapSetIntInt},
		{HandlerName: "HandleMapGetStringInt", ShimSuffix: "MapGetStringInt", Op: opMapGetStringInt},
		{HandlerName: "HandleMapSetStringInt", ShimSuffix: "MapSetStringInt", Op: opMapSetStringInt},
		{HandlerName: "HandleMapGetStringString", ShimSuffix: "MapGetStringString", Op: opMapGetStringString},
		{HandlerName: "HandleMapSetStringString", ShimSuffix: "MapSetStringString", Op: opMapSetStringString},
		{HandlerName: "HandleMapGetIntString", ShimSuffix: "MapGetIntString", Op: opMapGetIntString},
		{HandlerName: "HandleMapSetIntString", ShimSuffix: "MapSetIntString", Op: opMapSetIntString},
		{HandlerName: "HandleMapIndexOkIntInt", ShimSuffix: "MapIndexOkIntInt", Op: opMapIndexOkIntInt},
		{HandlerName: "HandleMapIndexOkStringInt", ShimSuffix: "MapIndexOkStringInt", Op: opMapIndexOkStringInt},
		{HandlerName: "HandleMapIndexOkStringString", ShimSuffix: "MapIndexOkStringString", Op: opMapIndexOkStringString},
		{HandlerName: "HandleMapIndexOkIntString", ShimSuffix: "MapIndexOkIntString", Op: opMapIndexOkIntString},
		{HandlerName: "HandleMapGetIntGeneral", ShimSuffix: "MapGetIntGeneral", Op: opMapGetIntGeneral},
		{HandlerName: "HandleMapIndexOkIntGeneral", ShimSuffix: "MapIndexOkIntGeneral", Op: opMapIndexOkIntGeneral},
		{HandlerName: "HandleMapGetStringGeneral", ShimSuffix: "MapGetStringGeneral", Op: opMapGetStringGeneral},
		{HandlerName: "HandleMapIndexOkStringGeneral", ShimSuffix: "MapIndexOkStringGeneral", Op: opMapIndexOkStringGeneral},
		{HandlerName: "HandleMapSetStringGeneral", ShimSuffix: "MapSetStringGeneral", Op: opMapSetStringGeneral},
		{HandlerName: "HandleMapAddIntInt", ShimSuffix: "MapAddIntInt", Op: opMapAddIntInt},
		{HandlerName: "HandleMapAddStringInt", ShimSuffix: "MapAddStringInt", Op: opMapAddStringInt},
		{HandlerName: "HandleMapIndexOkJumpIfFalseIntInt", ShimSuffix: "MapIndexOkJumpIfFalseIntInt", Op: opMapIndexOkJumpIfFalseIntInt},
		{HandlerName: "HandleMapIndexOkJumpIfFalseStringInt", ShimSuffix: "MapIndexOkJumpIfFalseStringInt", Op: opMapIndexOkJumpIfFalseStringInt},
		{HandlerName: "HandleMapIndexOkJumpIfFalseStringString", ShimSuffix: "MapIndexOkJumpIfFalseStringString", Op: opMapIndexOkJumpIfFalseStringString},
		{HandlerName: "HandleMapIndexOkJumpIfFalseIntString", ShimSuffix: "MapIndexOkJumpIfFalseIntString", Op: opMapIndexOkJumpIfFalseIntString},
		{HandlerName: "HandleMapIndexOkJumpIfFalseIntGeneral", ShimSuffix: "MapIndexOkJumpIfFalseIntGeneral", Op: opMapIndexOkJumpIfFalseIntGeneral},
		{HandlerName: "HandleMapIndexOkJumpIfFalseStringGeneral", ShimSuffix: "MapIndexOkJumpIfFalseStringGeneral", Op: opMapIndexOkJumpIfFalseStringGeneral},

		// Narrow: general-bank struct-field reader/writer/copy/swap and the cycle-broken
		// raw-pointer reader. Handlers write reflect.Value or call typedmemmove from Go, so GC
		// write barriers are preserved.
		{HandlerName: "HandleGetStructFieldGeneralT0", ShimSuffix: "GetStructFieldGeneralT0", Op: opGetStructFieldGeneral, Narrow: true},
		{HandlerName: "HandleSetStructFieldGeneralT0", ShimSuffix: "SetStructFieldGeneralT0", Op: opSetStructFieldGeneral, Narrow: true},
		{HandlerName: "HandleCopyStructFieldGeneralT0", ShimSuffix: "CopyStructFieldGeneralT0", Op: opCopyStructFieldGeneralT0, Narrow: true},
		{HandlerName: "HandleSwapStructFieldsGeneralT0", ShimSuffix: "SwapStructFieldsGeneralT0", Op: opSwapStructFieldsGeneralT0, Narrow: true},
		{HandlerName: "HandleGetStructFieldRawPointerT0", ShimSuffix: "GetStructFieldRawPointerT0", Op: opGetStructFieldRawPointerT0, Narrow: true},

		// Narrow: primitive struct-field readers/writers (Int/Uint/Float/Bool).
		{HandlerName: "HandleGetStructFieldIntT0", ShimSuffix: "GetStructFieldIntT0", Op: opGetStructFieldIntT0, Narrow: true},
		{HandlerName: "HandleSetStructFieldIntT0", ShimSuffix: "SetStructFieldIntT0", Op: opSetStructFieldIntT0, Narrow: true},
		{HandlerName: "HandleGetStructFieldUintT0", ShimSuffix: "GetStructFieldUintT0", Op: opGetStructFieldUint, Narrow: true},
		{HandlerName: "HandleSetStructFieldUintT0", ShimSuffix: "SetStructFieldUintT0", Op: opSetStructFieldUint, Narrow: true},
		{HandlerName: "HandleGetStructFieldFloatT0", ShimSuffix: "GetStructFieldFloatT0", Op: opGetStructFieldFloat, Narrow: true},
		{HandlerName: "HandleSetStructFieldFloatT0", ShimSuffix: "SetStructFieldFloatT0", Op: opSetStructFieldFloat, Narrow: true},
		{HandlerName: "HandleGetStructFieldBoolT0", ShimSuffix: "GetStructFieldBoolT0", Op: opGetStructFieldBool, Narrow: true},
		{HandlerName: "HandleSetStructFieldBoolT0", ShimSuffix: "SetStructFieldBoolT0", Op: opSetStructFieldBool, Narrow: true},

		// Test-nil-jump pair (hit per linked-list iteration).
		{HandlerName: "HandleTestNilJumpFalse", ShimSuffix: "TestNilJumpFalse", Op: opTestNilJumpFalse},
		{HandlerName: "HandleTestNilJumpTrue", ShimSuffix: "TestNilJumpTrue", Op: opTestNilJumpTrue},
	}
)

// tier2ShimRegistrations converts the registry table into the canonical
// []asm.Tier2HandlerShimSpec, preserving order.
//
// Returns []asm.Tier2HandlerShimSpec built from the registry entries.
func tier2ShimRegistrations() []asm.Tier2HandlerShimSpec {
	specs := make([]asm.Tier2HandlerShimSpec, 0, expectedTier2ShimCount)
	for _, registration := range tier2ShimRegistry {
		specs = append(specs, makeShimSpec(registration))
	}
	return specs
}

func init() { //nolint:gochecknoinits // one-shot registry registration; mirrors handler-table init pattern.
	for _, spec := range tier2ShimRegistrations() {
		asm.RegisterTier2Shim(spec)
	}
}
