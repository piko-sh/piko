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
	"math"

	"piko.sh/piko/wdk/safeconv"
)

const (
	// byteEncodingLimit is the exclusive upper bound for indices encoded in a single byte of
	// an opcode operand. Pool merges that exceed it under encodingByte must refuse the
	// splice.
	byteEncodingLimit = 256

	// uint16EncodingLimit is the exclusive upper bound for indices encoded in a uint16 pool
	// slot. Pool merges that would push the caller past it refuse the splice.
	uint16EncodingLimit = math.MaxUint16
)

// indexFits returns true when newIndex satisfies the encoding-byte constraint (always
// true when encodingByte is false).
//
// Takes newIndex (uint16) which is the candidate caller-side pool index.
// Takes encodingByte (bool) which is true when the consuming opcode encodes the index in
// a single byte.
//
// Returns true when the index fits the requested encoding.
func indexFits(newIndex uint16, encodingByte bool) bool {
	return !encodingByte || newIndex < byteEncodingLimit
}

// mergePoolIndex dispatches to the per-pool merge helper and returns the new caller-side
// index.
//
// Used by the inliner's operand walker when remapping any per-function table reference.
// The poolCallSites case is delegated to mergeCallSiteForCtx (which needs the
// inlineContext); calling mergePoolIndex with poolCallSites returns (0, false).
//
// On any kind of overflow (e.g. caller's structLayoutTable already holds
// byteEncodingLimit entries when the opcode encodes the index as uint8), returns (0,
// false) so the caller can refuse the splice.
//
// Takes caller (*CompiledFunction) which is the inlining caller whose pool receives the
// merged entry.
// Takes callee (*CompiledFunction) which is the function being spliced.
// Takes pool (inlinePool) which selects which parallel pool is being merged.
// Takes oldIndex (uint16) which is the entry's index in the callee pool.
// Takes encodingByte (bool) which is true when the consuming opcode encodes the result in
// a single byte and the index must fit in 8 bits.
//
// Returns the new caller-side index for the merged entry.
// Returns true when the merge fits; false on overflow or when encodingByte is set and the
// result exceeds the byte limit.
func mergePoolIndex(caller, callee *CompiledFunction, pool inlinePool, oldIndex uint16, encodingByte bool) (uint16, bool) {
	switch pool {
	case poolNone:
		return oldIndex, true
	case poolIntConsts:
		return mergeIntConstIndex(caller, callee, oldIndex, encodingByte)
	case poolFloatConsts:
		return mergeFloatConstIndex(caller, callee, oldIndex, encodingByte)
	case poolStringConsts:
		return mergeStringConstIndex(caller, callee, oldIndex, encodingByte)
	case poolBoolConsts:
		return mergeBoolConstIndex(caller, callee, oldIndex, encodingByte)
	case poolUintConsts:
		return mergeUintConstIndex(caller, callee, oldIndex, encodingByte)
	case poolComplexConsts:
		return mergeComplexConstIndex(caller, callee, oldIndex, encodingByte)
	case poolGeneralConsts:
		return mergeGeneralConstIndex(caller, callee, oldIndex, encodingByte)
	case poolTypeTable:
		return mergeTypeTableIndex(caller, callee, oldIndex, encodingByte)
	case poolStructLayoutTable:
		return mergeStructLayoutIndex(caller, callee, oldIndex, encodingByte)
	case poolCallSites:
		return 0, false
	case poolFunctions:
		return mergeFuncIndex(caller, callee, oldIndex, encodingByte)
	}
	return 0, false
}

// mergeIntConstIndex merges a callee intConstants entry into the caller's pool. See
// mergePoolIndex for the broader contract.
//
// Takes caller (*CompiledFunction) which is the inlining caller whose pool receives the
// merged entry.
// Takes callee (*CompiledFunction) which is the function being spliced.
// Takes oldIndex (uint16) which is the entry's index in the callee pool.
// Takes encodingByte (bool) which is true when the result must fit in a single byte.
//
// Returns the new caller-side index for the merged entry and true when the merge fits the
// requested encoding.
func mergeIntConstIndex(caller, callee *CompiledFunction, oldIndex uint16, encodingByte bool) (uint16, bool) {
	if int(oldIndex) >= len(callee.intConstants) {
		return 0, false
	}
	newIndex, err := caller.addIntConstant(callee.intConstants[oldIndex])
	if err != nil {
		return 0, false
	}
	return newIndex, indexFits(newIndex, encodingByte)
}

// mergeFloatConstIndex merges a callee floatConstants entry into the caller's pool. See
// mergePoolIndex for the broader contract.
//
// Takes caller (*CompiledFunction) which is the inlining caller whose pool receives the
// merged entry.
// Takes callee (*CompiledFunction) which is the function being spliced.
// Takes oldIndex (uint16) which is the entry's index in the callee pool.
// Takes encodingByte (bool) which is true when the result must fit in a single byte.
//
// Returns the new caller-side index for the merged entry and true when the merge fits the
// requested encoding.
func mergeFloatConstIndex(caller, callee *CompiledFunction, oldIndex uint16, encodingByte bool) (uint16, bool) {
	if int(oldIndex) >= len(callee.floatConstants) {
		return 0, false
	}
	newIndex, err := caller.addFloatConstant(callee.floatConstants[oldIndex])
	if err != nil {
		return 0, false
	}
	return newIndex, indexFits(newIndex, encodingByte)
}

// mergeStringConstIndex merges a callee stringConstants entry into the caller's pool. See
// mergePoolIndex for the broader contract.
//
// Takes caller (*CompiledFunction) which is the inlining caller whose pool receives the
// merged entry.
// Takes callee (*CompiledFunction) which is the function being spliced.
// Takes oldIndex (uint16) which is the entry's index in the callee pool.
// Takes encodingByte (bool) which is true when the result must fit in a single byte.
//
// Returns the new caller-side index for the merged entry and true when the merge fits the
// requested encoding.
func mergeStringConstIndex(caller, callee *CompiledFunction, oldIndex uint16, encodingByte bool) (uint16, bool) {
	if int(oldIndex) >= len(callee.stringConstants) {
		return 0, false
	}
	newIndex, err := caller.addStringConstant(callee.stringConstants[oldIndex])
	if err != nil {
		return 0, false
	}
	return newIndex, indexFits(newIndex, encodingByte)
}

// mergeBoolConstIndex merges a callee boolConstants entry into the caller's pool. See
// mergePoolIndex for the broader contract.
//
// Takes caller (*CompiledFunction) which is the inlining caller whose pool receives the
// merged entry.
// Takes callee (*CompiledFunction) which is the function being spliced.
// Takes oldIndex (uint16) which is the entry's index in the callee pool.
// Takes encodingByte (bool) which is true when the result must fit in a single byte.
//
// Returns the new caller-side index for the merged entry and true when the merge fits the
// requested encoding.
func mergeBoolConstIndex(caller, callee *CompiledFunction, oldIndex uint16, encodingByte bool) (uint16, bool) {
	if int(oldIndex) >= len(callee.boolConstants) {
		return 0, false
	}
	newIndex, err := caller.addBoolConstant(callee.boolConstants[oldIndex])
	if err != nil {
		return 0, false
	}
	return newIndex, indexFits(newIndex, encodingByte)
}

// mergeUintConstIndex merges a callee uintConstants entry into the caller's pool. See
// mergePoolIndex for the broader contract.
//
// Takes caller (*CompiledFunction) which is the inlining caller whose pool receives the
// merged entry.
// Takes callee (*CompiledFunction) which is the function being spliced.
// Takes oldIndex (uint16) which is the entry's index in the callee pool.
// Takes encodingByte (bool) which is true when the result must fit in a single byte.
//
// Returns the new caller-side index for the merged entry and true when the merge fits the
// requested encoding.
func mergeUintConstIndex(caller, callee *CompiledFunction, oldIndex uint16, encodingByte bool) (uint16, bool) {
	if int(oldIndex) >= len(callee.uintConstants) {
		return 0, false
	}
	newIndex, err := caller.addUintConstant(callee.uintConstants[oldIndex])
	if err != nil {
		return 0, false
	}
	return newIndex, indexFits(newIndex, encodingByte)
}

// mergeComplexConstIndex merges a callee complexConstants entry into the caller's pool.
// See mergePoolIndex for the broader contract.
//
// Takes caller (*CompiledFunction) which is the inlining caller whose pool receives the
// merged entry.
// Takes callee (*CompiledFunction) which is the function being spliced.
// Takes oldIndex (uint16) which is the entry's index in the callee pool.
// Takes encodingByte (bool) which is true when the result must fit in a single byte.
//
// Returns the new caller-side index for the merged entry and true when the merge fits the
// requested encoding.
func mergeComplexConstIndex(caller, callee *CompiledFunction, oldIndex uint16, encodingByte bool) (uint16, bool) {
	if int(oldIndex) >= len(callee.complexConstants) {
		return 0, false
	}
	newIndex, err := caller.addComplexConstant(callee.complexConstants[oldIndex])
	if err != nil {
		return 0, false
	}
	return newIndex, indexFits(newIndex, encodingByte)
}

// mergeGeneralConstIndex merges a callee generalConstants entry into the caller's pool.
// addGeneralConstant takes (v, descriptor); the descriptor is the parallel slice that
// this helper copies alongside the value.
//
// Takes caller (*CompiledFunction) which is the inlining caller whose pool receives the
// merged entry.
// Takes callee (*CompiledFunction) which is the function being spliced.
// Takes oldIndex (uint16) which is the entry's index in the callee pool.
// Takes encodingByte (bool) which is true when the result must fit in a single byte.
//
// Returns the new caller-side index for the merged entry and true when the merge fits the
// requested encoding.
//
// See mergePoolIndex for the broader contract.
func mergeGeneralConstIndex(caller, callee *CompiledFunction, oldIndex uint16, encodingByte bool) (uint16, bool) {
	if int(oldIndex) >= len(callee.generalConstants) {
		return 0, false
	}
	value := callee.generalConstants[oldIndex]
	var descriptor generalConstantDescriptor
	if int(oldIndex) < len(callee.generalConstantDescriptors) {
		descriptor = callee.generalConstantDescriptors[oldIndex]
	}
	newIndex, err := caller.addGeneralConstant(value, descriptor)
	if err != nil {
		return 0, false
	}
	return newIndex, indexFits(newIndex, encodingByte)
}

// mergeTypeTableIndex merges a callee typeTable entry into the caller's pool. See
// mergePoolIndex for the broader contract.
//
// Takes caller (*CompiledFunction) which is the inlining caller whose pool receives the
// merged entry.
// Takes callee (*CompiledFunction) which is the function being spliced.
// Takes oldIndex (uint16) which is the entry's index in the callee pool.
// Takes encodingByte (bool) which is true when the result must fit in a single byte.
//
// Returns the new caller-side index for the merged entry and true when the merge fits the
// requested encoding.
func mergeTypeTableIndex(caller, callee *CompiledFunction, oldIndex uint16, encodingByte bool) (uint16, bool) {
	if int(oldIndex) >= len(callee.typeTable) {
		return 0, false
	}
	newIndex, err := caller.addTypeRef(callee.typeTable[oldIndex])
	if err != nil {
		return 0, false
	}
	return newIndex, indexFits(newIndex, encodingByte)
}

// mergeCallSiteForCtx deep-copies the callee's callSites[oldIndex] entry, remaps
// argument/return register operands via ctx.remap, rebuilds the argCopyProgram against
// the callee's parameterKinds, and appends the result to caller.callSites.
//
// Multi-level inlining path: when the callee's body contains an opCall, the splice copies
// the corresponding callSite into the caller so the runtime can dispatch through it.
//
// Takes ctx (*inlineContext) which carries caller, callee, and the register remap table.
// Takes oldIndex (uint16) which is the callee callSite index to merge.
// Takes encodingByte (bool) which is true when the result must fit in a single byte.
//
// Returns the new caller-side callSite index.
// Returns true when the merge succeeded; false on overflow or when any register failed to
// map.
func mergeCallSiteForCtx(ctx *inlineContext, oldIndex uint16, encodingByte bool) (uint16, bool) {
	caller := ctx.caller
	callee := ctx.callee
	if int(oldIndex) >= len(callee.callSites) {
		return 0, false
	}
	source := callee.callSites[oldIndex]

	arguments, ok := remapVarLocations(ctx, source.arguments)
	if !ok {
		return 0, false
	}
	returns, ok := remapVarLocations(ctx, source.returns)
	if !ok {
		return 0, false
	}

	newSite := buildRemappedCallSite(&source, arguments, returns)
	if !remapSpecialRegisters(ctx, &source, &newSite) {
		return 0, false
	}

	if newSite.cachedCallee != nil && len(newSite.cachedCallee.parameterKinds) == len(arguments) {
		newSite.argCopyProgram = buildCallArgCopyProgram(arguments, newSite.cachedCallee.parameterKinds, newSite.cachedCallee.parameterRegisters)
	}

	if len(caller.callSites) >= uint16EncodingLimit {
		return 0, false
	}
	newIndex := safeconv.IntToUint16(len(caller.callSites))
	caller.callSites = append(caller.callSites, newSite)
	return newIndex, indexFits(newIndex, encodingByte)
}

// remapVarLocations deep-copies a varLocation slice while remapping each register through
// ctx.lookupRegister.
//
// Takes ctx (*inlineContext) which carries the register remap table.
// Takes source ([]varLocation) which is the callee-side slice.
//
// Returns the remapped slice and true on success; nil and false when any register fails
// to map.
func remapVarLocations(ctx *inlineContext, source []varLocation) ([]varLocation, bool) {
	out := make([]varLocation, len(source))
	for i, location := range source {
		newReg, ok := ctx.lookupRegister(location.kind, location.register)
		if !ok {
			return nil, false
		}
		out[i] = varLocation{
			register:  newReg,
			kind:      location.kind,
			isSpilled: location.isSpilled,
			spillSlot: location.spillSlot,
		}
	}
	return out, true
}

// buildRemappedCallSite constructs a fresh callSite from source whose register fields are
// zeroed and whose argument/return slices have already been remapped. The argCopyProgram
// is left nil so the caller can rebuild it after the special-register fields are
// populated.
//
// funcIndex stays as-is. It refers to the program-wide rootFunction.functions slice,
// which is shared across all compiled functions, so the index remains valid even after
// merging the call-site into a different caller. We do NOT remap via callee.functions
// (which is the nested-closures slice and typically empty).
//
// Takes source (*callSite) which is the original callee-side site.
// Takes arguments ([]varLocation) which is the remapped argument slice.
// Takes returns ([]varLocation) which is the remapped return slice.
//
// Returns the constructed callSite ready for special-register remapping.
func buildRemappedCallSite(source *callSite, arguments, returns []varLocation) callSite {
	return callSite{
		arguments:                 arguments,
		returns:                   returns,
		funcIndex:                 source.funcIndex,
		cachedCallee:              source.cachedCallee,
		isClosure:                 source.isClosure,
		isNative:                  source.isNative,
		isMethod:                  source.isMethod,
		closureRegister:           0,
		nativeRegister:            0,
		methodReceiverRegister:    0,
		isEllipsisSpread:          source.isEllipsisSpread,
		argumentStaticTypeNames:   source.argumentStaticTypeNames,
		argumentStaticTypeStrings: source.argumentStaticTypeStrings,
		parameterInterfaceFlags:   source.parameterInterfaceFlags,
		runtimeVariadicSliceType:  source.runtimeVariadicSliceType,
		runtimeVariadicNumFixed:   source.runtimeVariadicNumFixed,
		linkedTypeArgs:            source.linkedTypeArgs,
	}
}

// remapSpecialRegisters maps the closure/native/method-receiver registers from source
// onto destination via ctx. Each special register belongs to the general bank.
//
// Takes ctx (*inlineContext) which carries the register remap table.
// Takes source (*callSite) which is the original callee-side site.
// Takes destination (*callSite) which is the freshly-built caller-side site.
//
// Returns true when every special register present in source maps successfully; false
// when any lookup fails.
func remapSpecialRegisters(ctx *inlineContext, source, destination *callSite) bool {
	if source.isClosure {
		newReg, ok := ctx.lookupRegister(registerGeneral, source.closureRegister)
		if !ok {
			return false
		}
		destination.closureRegister = newReg
	}
	if source.isNative {
		newReg, ok := ctx.lookupRegister(registerGeneral, source.nativeRegister)
		if !ok {
			return false
		}
		destination.nativeRegister = newReg
	}
	if source.isMethod {
		newReg, ok := ctx.lookupRegister(registerGeneral, source.methodReceiverRegister)
		if !ok {
			return false
		}
		destination.methodReceiverRegister = newReg
	}
	return true
}

// mergeFuncIndex appends callee.functions[oldIndex] to caller.functions and returns the
// new index.
//
// Dedup uses pointer identity since *CompiledFunction is the key. When oldIndex is out of
// range (sites reference funcIndex = 0 even when no nested function is referenced, such
// as closures or native calls), returns (0, true) so the runtime passes through without
// reading.
//
// Linear-scan dedup: functions slices are small, so the scan is cheap compared to a hash
// map.
//
// Takes caller (*CompiledFunction) which receives the merged entry.
// Takes callee (*CompiledFunction) which supplies the source function.
// Takes oldIndex (uint16) which is the source functions slice index.
// Takes encodingByte (bool) which is true when the result must fit in a single byte.
//
// Returns the caller-side functions index.
// Returns true on success; false on overflow or when encodingByte is set and the result
// exceeds the byte limit.
func mergeFuncIndex(caller, callee *CompiledFunction, oldIndex uint16, encodingByte bool) (uint16, bool) {
	if int(oldIndex) >= len(callee.functions) {
		return 0, true
	}
	target := callee.functions[oldIndex]
	for i, existing := range caller.functions {
		if existing == target {
			index := safeconv.IntToUint16(i)
			return index, indexFits(index, encodingByte)
		}
	}
	if len(caller.functions) >= uint16EncodingLimit {
		return 0, false
	}
	index := safeconv.IntToUint16(len(caller.functions))
	caller.functions = append(caller.functions, target)
	return index, indexFits(index, encodingByte)
}

// mergeStructLayoutIndex merges a callee's layoutTable entry into the caller's table.
//
// The layout's TypeIndex (and FieldTypeIndex, when populated) reference the callee's
// typeTable, so those are remapped first before searching for an existing equivalent
// entry in the caller. Dedup uses a linear scan: callee tables are small, so the scan is
// amortised O(n) per splice.
//
// Takes caller (*CompiledFunction) which receives the merged layout.
// Takes callee (*CompiledFunction) which supplies the source layout.
// Takes oldIndex (uint16) which is the source structLayoutTable index.
// Takes encodingByte (bool) which is true when the result must fit in a single byte.
//
// Returns the caller-side structLayoutTable index.
// Returns true on success; false on overflow or unresolved type-table remap.
func mergeStructLayoutIndex(caller, callee *CompiledFunction, oldIndex uint16, encodingByte bool) (uint16, bool) {
	if int(oldIndex) >= len(callee.structLayoutTable) {
		return 0, false
	}
	source := callee.structLayoutTable[oldIndex]
	newTypeIndex, ok := mergePoolIndex(caller, callee, poolTypeTable, source.TypeIndex, false)
	if !ok {
		return 0, false
	}
	source.TypeIndex = newTypeIndex
	if source.FieldTypeIndex != 0 || (source.RegisterKind == uint8(registerGeneral)) {
		newFieldTypeIndex, ok := mergePoolIndex(caller, callee, poolTypeTable, source.FieldTypeIndex, false)
		if !ok {
			return 0, false
		}
		source.FieldTypeIndex = newFieldTypeIndex
	}
	for i := range caller.structLayoutTable {
		if structLayoutEqual(caller.structLayoutTable[i], source) {
			index := safeconv.IntToUint16(i)
			return index, indexFits(index, encodingByte)
		}
	}
	index := safeconv.IntToUint16(len(caller.structLayoutTable))
	caller.structLayoutTable = append(caller.structLayoutTable, source)
	return index, indexFits(index, encodingByte)
}

// structLayoutEqual compares two structFieldLayout entries field by field. Used by
// mergeStructLayoutIndex's dedup search.
//
// Takes a (structFieldLayout) which is the first layout.
// Takes b (structFieldLayout) which is the second layout.
//
// Returns true when every scalar field and every path entry matches.
func structLayoutEqual(a, b structFieldLayout) bool {
	if a.Offset != b.Offset ||
		a.TypeIndex != b.TypeIndex ||
		a.PathLength != b.PathLength ||
		a.Kind != b.Kind ||
		a.RegisterKind != b.RegisterKind ||
		a.Flags != b.Flags ||
		a.FieldTypeIndex != b.FieldTypeIndex {
		return false
	}
	for i := range a.Path {
		if a.Path[i] != b.Path[i] {
			return false
		}
	}
	return true
}
