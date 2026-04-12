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

// inlinePool identifies which per-function table an opcode operand references.
type inlinePool uint8

const (
	// poolNone marks an opcode operand that does not reference any per-function pool.
	poolNone inlinePool = iota

	// poolIntConsts references the function's integer constants pool.
	poolIntConsts

	// poolFloatConsts references the function's float constants pool.
	poolFloatConsts

	// poolStringConsts references the function's string constants pool.
	poolStringConsts

	// poolBoolConsts references the function's bool constants pool.
	poolBoolConsts

	// poolUintConsts references the function's uint constants pool.
	poolUintConsts

	// poolComplexConsts references the function's complex constants pool.
	poolComplexConsts

	// poolGeneralConsts references the function's general (reflect.Value) constants pool.
	poolGeneralConsts

	// poolTypeTable references the function's type table (for MakeSlice, Convert,
	// TypeAssert, AllocIndirect).
	poolTypeTable

	// poolStructLayoutTable references the function's struct-layout table (used by the
	// tier-0 struct-field ops).
	poolStructLayoutTable

	// poolCallSites references the function's call-site table.
	poolCallSites

	// poolFunctions references the program-wide function table.
	poolFunctions
)

// inlinePoolShape records the per-opcode pool descriptor used by the inliner.
//
// Encodes which operand byte (or pair of bytes) carries an index into a per-function
// table, which table that is, whether the opcode consumes a trailing opExt extension
// word, and (if so) whether THAT word carries a table index or a register reference that
// needs bank-specific remapping.
//
// Only opcodes with at least one non-register, non-jump-offset operand need an entry; for
// everything else, operandShapes is already sufficient and the inliner just walks
// register operands.
type inlinePoolShape struct {
	// bKindByte names the pool whose uint8 index lives in operand B alone when nonzero (e.g.
	// opLoadBoolConst).
	bKindByte inlinePool

	// cKindByte names the pool whose uint8 index lives in operand C of the main instruction
	// when nonzero (e.g. tier-0 struct field T0 ops).
	cKindByte inlinePool

	// bcWide16 names the pool whose uint16 little-endian index is formed by operands B and C
	// together (b|c<<8) when nonzero (e.g. opLoadIntConst).
	bcWide16 inlinePool

	// extAWide16 names the pool whose uint16 index lives in operands A and B of the
	// following opExt word when nonzero. Used by ops like opTypeAssert that follow with
	// `opExt typeIndexLow typeIndexHigh _`.
	extAWide16 inlinePool

	// hasExtensionWord is true when this opcode consumes one opExt instruction following it
	// in the bytecode stream. The inliner must walk and (when applicable) remap that
	// extension when iterating the body.
	hasExtensionWord bool

	// extARegBank names the bank for an ext-word operand A register reference (used by
	// opMapIndexOk* variants whose extension A is `okRegister`, an int register; registerInt
	// = 0 conflicts with the zero default so extARegSet acts as the discriminator).
	extARegBank registerKind

	// extARegSet is true when extARegBank carries a meaningful value (needed because
	// registerInt = 0).
	extARegSet bool
}

var (
	// inlinePoolShapes maps each opcode to its pool descriptor. Sparse: most opcodes have a
	// zero-value entry meaning "no pool indices".
	//
	//nolint:gochecknoglobals // immutable lookup table populated at init
	inlinePoolShapes [opcodeCount]inlinePoolShape
)

func init() {
	registerInlineLoadConstShapes()
	registerInlineConstFusedArithShapes()
	registerInlineStructFieldShapes()
	registerInlineExtensionTypeTableShapes()
	registerInlineMapIndexOkShapes()
	registerInlineExtensionPassthroughShapes()
	registerInlineCallShape()
}

// registerInlineLoadConstShapes populates wide-index constant load shapes (b|c<<8 ->
// respective constant pool) and the single-byte bool-constant pool index. The bool pool
// has at most 2 entries (true/false) but the callee's index can differ from the caller's
// after merge so the byte must be remapped via bKindByte.
func registerInlineLoadConstShapes() {
	inlinePoolShapes[opLoadIntConst] = inlinePoolShape{bcWide16: poolIntConsts}
	inlinePoolShapes[opLoadFloatConst] = inlinePoolShape{bcWide16: poolFloatConsts}
	inlinePoolShapes[opLoadStringConst] = inlinePoolShape{bcWide16: poolStringConsts}
	inlinePoolShapes[opLoadUintConst] = inlinePoolShape{bcWide16: poolUintConsts}
	inlinePoolShapes[opLoadComplexConst] = inlinePoolShape{bcWide16: poolComplexConsts}
	inlinePoolShapes[opLoadGeneralConst] = inlinePoolShape{bcWide16: poolGeneralConsts}
	inlinePoolShapes[opLoadBoolConst] = inlinePoolShape{bKindByte: poolBoolConsts}
}

// registerInlineConstFusedArithShapes populates the const-fused arithmetic ops with the
// int-const pool index in operand C. The const-fused compare-and-jump ops are
// intentionally omitted: their extension-word encoding varies per opcode and is refused
// by the inliner.
func registerInlineConstFusedArithShapes() {
	inlinePoolShapes[opAddIntConst] = inlinePoolShape{cKindByte: poolIntConsts}
	inlinePoolShapes[opSubIntConst] = inlinePoolShape{cKindByte: poolIntConsts}
	inlinePoolShapes[opMulIntConst] = inlinePoolShape{cKindByte: poolIntConsts}
}

// registerInlineStructFieldShapes populates tier-0 struct field access ops; operand C is
// a uint8 layoutTable index for each.
func registerInlineStructFieldShapes() {
	inlinePoolShapes[opGetStructFieldIntT0] = inlinePoolShape{cKindByte: poolStructLayoutTable}
	inlinePoolShapes[opSetStructFieldIntT0] = inlinePoolShape{cKindByte: poolStructLayoutTable}
	inlinePoolShapes[opGetStructFieldUint] = inlinePoolShape{cKindByte: poolStructLayoutTable}
	inlinePoolShapes[opSetStructFieldUint] = inlinePoolShape{cKindByte: poolStructLayoutTable}
	inlinePoolShapes[opGetStructFieldFloat] = inlinePoolShape{cKindByte: poolStructLayoutTable}
	inlinePoolShapes[opSetStructFieldFloat] = inlinePoolShape{cKindByte: poolStructLayoutTable}
	inlinePoolShapes[opGetStructFieldBool] = inlinePoolShape{cKindByte: poolStructLayoutTable}
	inlinePoolShapes[opSetStructFieldBool] = inlinePoolShape{cKindByte: poolStructLayoutTable}
	inlinePoolShapes[opGetStructFieldGeneral] = inlinePoolShape{cKindByte: poolStructLayoutTable}
	inlinePoolShapes[opSetStructFieldGeneral] = inlinePoolShape{cKindByte: poolStructLayoutTable}

	inlinePoolShapes[opCopyStructFieldGeneralT0] = inlinePoolShape{
		cKindByte:        poolStructLayoutTable,
		extAWide16:       poolStructLayoutTable,
		hasExtensionWord: true,
	}
}

// registerInlineExtensionTypeTableShapes populates the ops whose extension word carries a
// typeTable index in ext A|B<<8. Each emits emit(op, ...) then emitExtension(typeIndex,
// hint).
func registerInlineExtensionTypeTableShapes() {
	typeTableShape := inlinePoolShape{
		extAWide16:       poolTypeTable,
		hasExtensionWord: true,
	}
	inlinePoolShapes[opMakeSlice] = typeTableShape
	inlinePoolShapes[opConvert] = typeTableShape
	inlinePoolShapes[opTypeAssert] = typeTableShape
	inlinePoolShapes[opAllocIndirect] = typeTableShape
	inlinePoolShapes[opPackTyped] = typeTableShape
}

// registerInlineMapIndexOkShapes populates the opMapIndexOk variants. The main word is
// pure-register; the extension holds the okRegister (registerInt) in operand A.
func registerInlineMapIndexOkShapes() {
	mapIndexShape := inlinePoolShape{
		hasExtensionWord: true,
		extARegBank:      registerInt,
		extARegSet:       true,
	}
	inlinePoolShapes[opMapIndexOk] = mapIndexShape
	inlinePoolShapes[opMapIndexOkIntInt] = mapIndexShape
	inlinePoolShapes[opMapIndexOkIntString] = mapIndexShape
	inlinePoolShapes[opMapIndexOkIntGeneral] = mapIndexShape
	inlinePoolShapes[opMapIndexOkStringInt] = mapIndexShape
	inlinePoolShapes[opMapIndexOkStringString] = mapIndexShape
}

// registerInlineExtensionPassthroughShapes populates ops whose extension word carries
// data that needs no remapping (append immediate length, global-wide high bits).
// hasExtensionWord is set so the body walker advances past the ext without
// misinterpreting it.
func registerInlineExtensionPassthroughShapes() {
	passthrough := inlinePoolShape{hasExtensionWord: true}
	inlinePoolShapes[opAppend] = passthrough
	inlinePoolShapes[opAppendSpread] = passthrough
	inlinePoolShapes[opGetGlobalWide] = passthrough
	inlinePoolShapes[opSetGlobalWide] = passthrough
}

// registerInlineCallShape populates opCall whose site index lives in operand B|C<<8.
// mergeCallSite deep-copies the callSite entry, remaps its register operands, rebuilds
// argCopyProgram, and appends to caller.callSites.
//
// opTailCall shares the encoding (site index in B|C<<8). The inliner converts opTailCall
// to opCall at splice time, so giving them the same pool shape lets the same remap path
// service both.
func registerInlineCallShape() {
	callSiteShape := inlinePoolShape{bcWide16: poolCallSites}
	inlinePoolShapes[opCall] = callSiteShape
	inlinePoolShapes[opTailCall] = callSiteShape
}

// inlinePoolShapeFor returns the inlinePoolShape entry for op.
//
// Sparse table; zero-value (poolNone everywhere) means the opcode has no pool-index
// operands and the inliner walks only its register operands. Out-of-range opcodes return
// the zero value.
//
// Takes op (opcode) which is the opcode whose shape to fetch.
//
// Returns the inlinePoolShape entry for op, zero-value if op is out of range.
func inlinePoolShapeFor(op opcode) inlinePoolShape {
	if int(op) >= len(inlinePoolShapes) {
		return inlinePoolShape{}
	}
	return inlinePoolShapes[op]
}
