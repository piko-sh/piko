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

// buildJumpTargets pre-computes the set of instruction indices that are jump
// destinations, enabling O(1) lookup during pattern matching.
//
// Takes body ([]instruction) which specifies the instruction sequence to scan.
//
// Returns a map of instruction indices that are jump targets.
func (*CompiledFunction) buildJumpTargets(body []instruction) map[int]bool {
	jumpTargets := make(map[int]bool, len(body)/4)
	for i, instr := range body {
		switch instr.op {
		case opJumpIfTrue, opJumpIfFalse:
			offset := instr.signedOffset()
			jumpTargets[i+1+int(offset)] = true
		case opDrillTier1:
			if subOpcode(instr.a) == subOpJump {
				offset := instr.signedOffset()
				jumpTargets[i+1+int(offset)] = true
			}
		default:
		}
	}
	return jumpTargets
}

// fuseLongPatterns dispatches all multi-instruction fusion patterns that span more than
// three opcodes. Runs before fuseThreeInstrPatterns so longer patterns get first dibs and
// are not partially consumed by the shorter compare-jump or compare-const-jump fusers.
//
// Takes body ([]instruction) which specifies the instruction sequence.
// Takes i (int) which specifies the current index.
// Takes n (int) which specifies the length.
// Takes jumpTargets (map[int]bool) which specifies the set of protected jump
// destinations.
//
// Returns true if a pattern was matched and applied.
func (cf *CompiledFunction) fuseLongPatterns(
	body []instruction, i, n int,
	jumpTargets map[int]bool,
) bool {
	return cf.fuseRangeCheckUintJumpFalse(body, i, n, jumpTargets)
}

// fuseRangeCheckUintJumpFalse fuses the canonical byte classifier pattern `value >= lo &&
// value <= hi` into a single tier-1 sub-op (subOpRangeCheckUintJumpFalse) plus two
// extension words.
//
// Pre-fusion the compiler emits an 8-instruction window for isLetterByte-style
// predicates: LoadUintConstSmall(vRegLo, loImm), GeUint(cmpLo, valueReg, vRegLo),
// MoveInt(condReg, cmpLo), then JumpIfFalse(condReg, +3) which chains to the second
// JumpIfFalse, then the matching high-bound sequence ending in JumpIfFalse(condReg,
// off2). Both failure paths land on the same instruction the second JumpIfFalse targets,
// so one fused jump suffices.
//
// Post-fusion the window becomes the umbrella op at [i+0], two opExt words carrying
// {loImm, hiImm} and {off2Lo, off2Hi}, then five opNops padding out the original 8-slot
// window. The handler reads both ext words, advances past the NOPs, and performs the
// range check. The trailing NOPs preserve body length and any unrelated jump targets that
// may land on indexes after this block.
//
// Takes body ([]instruction) which specifies the instruction sequence.
// Takes i (int) which specifies the current index.
// Takes n (int) which specifies the length.
// Takes jumpTargets (map[int]bool) which specifies the set of protected jump
// destinations.
//
// Returns true if the pattern matched and was applied.
func (cf *CompiledFunction) fuseRangeCheckUintJumpFalse(
	body []instruction, i, n int,
	jumpTargets map[int]bool,
) bool {
	if i+rangeCheckWindowSize > n {
		return false
	}
	shape, ok := cf.matchRangeCheckShape(body, i)
	if !ok {
		return false
	}
	if !cf.rangeCheckJumpTargetsClear(body, i, jumpTargets) {
		return false
	}
	valueReg, _, ok := matchRangeCheckRegisters(body, i, shape.vRegLo, shape.vRegHi)
	if !ok {
		return false
	}
	off2, ok := validateRangeCheckOffsets(body, i)
	if !ok {
		return false
	}
	offLo, offHi := splitOffset(off2)
	body[i] = makeInstruction(opDrillTier1, uint8(subOpRangeCheckUintJumpFalse), valueReg, 0)
	body[i+1] = makeInstruction(opExt, shape.loImm, shape.hiImm, 0)
	body[i+2] = makeInstruction(opExt, offLo, offHi, 0)
	for slot := i + rangeCheckFirstNopOffset; slot < i+rangeCheckWindowSize; slot++ {
		body[slot] = makeInstruction(opNop, 0, 0, 0)
	}
	return true
}

// rangeCheckShape carries the per-bound immediate and destination register decoded from
// the fuseRangeCheckUintJumpFalse window.
type rangeCheckShape struct {
	// loImm is the low-bound immediate decoded from the first LoadUintConstSmall.
	loImm uint8

	// vRegLo is the destination register of the low-bound load.
	vRegLo uint8

	// hiImm is the high-bound immediate decoded from the second LoadUintConstSmall.
	hiImm uint8

	// vRegHi is the destination register of the high-bound load.
	vRegHi uint8
}

// matchRangeCheckShape verifies the opcode shape of the 8-instruction range-check window
// and returns the small-uint immediates plus their destination registers from the two
// LoadUintConstSmall positions.
//
// Takes body ([]instruction) which is the instruction sequence.
// Takes i (int) which is the candidate window start.
//
// Returns the decoded shape (immediates plus destination registers).
// Returns true when every opcode at the expected positions matches.
func (cf *CompiledFunction) matchRangeCheckShape(body []instruction, i int) (rangeCheckShape, bool) {
	var shape rangeCheckShape
	var ok bool
	shape.loImm, shape.vRegLo, ok = cf.matchSmallUintConstLoad(body[i])
	if !ok {
		return rangeCheckShape{}, false
	}
	if body[i+1].op != opGeUint ||
		!instrIsTier1SubOp(body[i+2], subOpMoveInt) ||
		body[i+rangeCheckFirstJumpOffset].op != opJumpIfFalse {
		return rangeCheckShape{}, false
	}
	shape.hiImm, shape.vRegHi, ok = cf.matchSmallUintConstLoad(body[i+rangeCheckSecondLoadOffset])
	if !ok {
		return rangeCheckShape{}, false
	}
	if body[i+rangeCheckSecondLoadOffset+1].op != opLeUint ||
		!instrIsTier1SubOp(body[i+rangeCheckSecondMoveOffset], subOpMoveInt) ||
		body[i+rangeCheckSecondJumpOffset].op != opJumpIfFalse {
		return rangeCheckShape{}, false
	}
	return shape, true
}

// rangeCheckJumpTargetsClear returns true when no jump destination lands inside the
// window's interior. The second jump may land on itself when it is the target of the
// first jump (the chained design); that case is allowed as long as no other incoming jump
// reaches the second-jump position.
//
// Takes body ([]instruction) which is the instruction sequence.
// Takes i (int) which is the window start.
// Takes jumpTargets (map[int]bool) which marks branch destinations.
//
// Returns true when the rewrite is safe with respect to jump targets.
func (cf *CompiledFunction) rangeCheckJumpTargetsClear(body []instruction, i int, jumpTargets map[int]bool) bool {
	for k := i + 1; k <= i+rangeCheckSecondMoveOffset; k++ {
		if jumpTargets[k] {
			return false
		}
	}
	if jumpTargets[i+rangeCheckSecondJumpOffset] && cf.countIncomingJumps(body, i+rangeCheckSecondJumpOffset, i+rangeCheckFirstJumpOffset) > 0 {
		return false
	}
	return true
}

// fuseEqUintConstJumpFalse fuses the 3-op byte-switch dispatch pattern
// `LoadUintConstSmall + EqUint + JumpIfFalse` into a single tier-1 sub-op
// (subOpEqUintConstJumpFalse) plus one extension word and one trailing opNop. Hits the
// canonical `switch byteValue` chain (brainfuck dispatch, tokenisers, hand-rolled lexers)
// where each case becomes one of these triples.
//
// Pre-fusion the window holds LoadUintConstSmall(constReg, immVal), EqUint(condReg,
// valueReg, constReg), then JumpIfFalse(condReg, off). Post-fusion the window becomes the
// umbrella op carrying valueReg+immVal, an opExt word holding the split offset, and a
// trailing opNop that keeps the body length stable so unrelated jump targets land at
// unchanged indices. The handler reads uints[valueReg] and advances programCounter by the
// encoded offset when the value does not match immVal.
//
// Takes body ([]instruction) which specifies the instruction sequence.
// Takes i (int) which specifies the current index.
// Takes n (int) which specifies the length.
// Takes jumpTargets (map[int]bool) which specifies the set of protected jump
// destinations.
//
// Returns true if the pattern matched and was applied.
func (cf *CompiledFunction) fuseEqUintConstJumpFalse(
	body []instruction, i, n int,
	jumpTargets map[int]bool,
) bool {
	if i+threeInstructionWindow > n {
		return false
	}
	immVal, constReg, ok := cf.matchSmallUintConstLoad(body[i])
	if !ok {
		return false
	}
	if body[i+1].op != opEqUint || body[i+2].op != opJumpIfFalse {
		return false
	}
	if jumpTargets[i+1] || jumpTargets[i+2] {
		return false
	}
	var valueReg uint8
	switch {
	case body[i+1].c == constReg:
		valueReg = body[i+1].b
	case body[i+1].b == constReg:
		valueReg = body[i+1].c
	default:
		return false
	}
	condReg := body[i+1].a
	if body[i+2].a != condReg {
		return false
	}
	rawOffset := body[i+2].signedOffset()
	offLo, offHi := splitOffset(rawOffset)
	body[i] = makeInstruction(opDrillTier1, uint8(subOpEqUintConstJumpFalse), valueReg, immVal)
	body[i+1] = makeInstruction(opExt, offLo, offHi, 0)
	body[i+2] = makeInstruction(opNop, 0, 0, 0)
	return true
}

// fuseMapIndexOkJumpFalse fuses a typed `opMapIndexOkXXX + ext + opJumpIfFalse` triple (3
// body slots, strict adjacency) into a single fused dispatch.
//
// Strict adjacency is required because the fuser cannot guarantee correctness when
// arbitrary ops sit between the mapIndexOk and the JumpIfFalse. The compiler routinely
// emits MOVE_INT chains there (copying the value to a final register, then copying the ok
// flag, then jumping); a walk-and-chain-extend variant would break code that reads the
// moved value on the jump-taken branch, e.g. `if ok { return v } else { return v - 1 }`.
//
// Pre-fusion the window holds {sourceOpcode, dest, map, key}, an opExt word carrying the
// okReg, and the JumpIfFalse on okReg. Post-fusion the source opcode becomes
// targetOpcode, the ext word folds the jump offset alongside okReg, and the third slot
// becomes opNop.
//
// Takes body ([]instruction) which specifies the instruction sequence.
// Takes i (int) which specifies the current index.
// Takes n (int) which specifies the length.
// Takes jumpTargets (map[int]bool) which specifies the set of protected jump
// destinations.
// Takes sourceOpcode (opcode) which is the unfused typed mapIndexOk opcode to recognise
// (e.g. opMapIndexOkIntGeneral).
// Takes targetOpcode (opcode) which is the fused replacement opcode (e.g.
// opMapIndexOkJumpIfFalseIntGeneral).
//
// Returns true if the pattern matched and was applied.
func (*CompiledFunction) fuseMapIndexOkJumpFalse(
	body []instruction, i, n int,
	jumpTargets map[int]bool,
	sourceOpcode opcode, targetOpcode opcode,
) bool {
	if i+threeInstructionWindow > n {
		return false
	}
	if body[i].op != sourceOpcode {
		return false
	}
	if body[i+2].op != opJumpIfFalse {
		return false
	}
	if jumpTargets[i+1] || jumpTargets[i+2] {
		return false
	}
	okReg := body[i+1].a
	if body[i+2].a != okReg {
		return false
	}
	origOffset := body[i+2].signedOffset()
	packedOffset := int32(origOffset) + 1
	if packedOffset > math.MaxInt16 || packedOffset < math.MinInt16 {
		return false
	}
	offLo, offHi := splitOffset(int16(packedOffset))
	body[i] = makeInstruction(targetOpcode, body[i].a, body[i].b, body[i].c)
	body[i+1] = makeInstruction(opExt, okReg, offLo, offHi)
	body[i+2] = makeInstruction(opNop, 0, 0, 0)
	return true
}

// countIncomingJumps counts branches targeting targetPC, excluding the instruction at
// sourcePC.
//
// Used by fuseRangeCheckUintJumpFalse to verify that the chained jump from the inner
// JumpIfFalse to the outer JumpIfFalse is the ONLY entry into the outer JumpIfFalse, so
// collapsing both into a single fused op is safe even when the outer position appears in
// jumpTargets.
//
// Takes body ([]instruction) which specifies the instruction sequence.
// Takes targetPC (int) which is the destination index to count entries for.
// Takes sourcePC (int) which is the index of the instruction whose jump should NOT be
// counted (the legitimate intra-pattern chain).
//
// Returns the count of jumps from any other instruction targeting targetPC.
func (*CompiledFunction) countIncomingJumps(body []instruction, targetPC, sourcePC int) int {
	count := 0
	for j, instr := range body {
		if j == sourcePC {
			continue
		}
		switch instr.op {
		case opJumpIfTrue, opJumpIfFalse:
			if j+1+int(instr.signedOffset()) == targetPC {
				count++
			}
		case opDrillTier1:
			if subOpcode(instr.a) == subOpJump {
				if j+1+int(instr.signedOffset()) == targetPC {
					count++
				}
			}
		default:
		}
	}
	return count
}

// matchSmallUintConstLoad recognises both forms of a uint constant load that fits in a
// byte immediate.
//
// Recognises the tier-0 opLoadUintConst with a pool reference whose value is in
// 0..math.MaxUint8, and the post-rewrite tier-1 subOpLoadUintConstSmall with the
// immediate already inlined. Callers use this to fuse patterns that read a small uint
// constant regardless of whether optimiseLoadUintConst has already rewritten the load
// instruction.
//
// Takes instr (instruction) which is the candidate load instruction to inspect.
//
// Returns the 8-bit immediate value when matched, the load destination when matched, and
// true when instr is a recognised small-uint-constant load.
func (cf *CompiledFunction) matchSmallUintConstLoad(instr instruction) (immediate, destinationRegister uint8, ok bool) {
	if instrIsTier1SubOp(instr, subOpLoadUintConstSmall) {
		return instr.c, instr.b, true
	}
	if instr.op != opLoadUintConst || instr.c != 0 {
		return 0, 0, false
	}
	if int(instr.b) >= len(cf.uintConstants) {
		return 0, 0, false
	}
	value := cf.uintConstants[instr.b]
	if value > math.MaxUint8 {
		return 0, 0, false
	}
	return uint8(value), instr.a, true
}

// fuseThreeInstrPatterns dispatches all 3-instruction fusion patterns. Must run before
// 2-instruction patterns to avoid partial consumption.
//
// Takes body ([]instruction) which specifies the instruction sequence.
// Takes i (int) which specifies the current index.
// Takes n (int) which specifies the length.
// Takes jumpTargets (map[int]bool) which specifies the set of protected jump
// destinations.
//
// Returns true if a pattern was matched and applied.
func (cf *CompiledFunction) fuseThreeInstrPatterns(
	body []instruction, i, n int,
	jumpTargets map[int]bool,
) bool {
	return cf.fuseCompareConstJump(body, i, n, jumpTargets, opLeInt, opJumpIfFalse, opLeIntConstJumpFalse) ||
		cf.fuseCompareConstJump(body, i, n, jumpTargets, opLtInt, opJumpIfFalse, opLtIntConstJumpFalse) ||
		cf.fuseCompareConstJump(body, i, n, jumpTargets, opEqInt, opJumpIfFalse, opEqIntConstJumpFalse) ||
		cf.fuseCompareConstJump(body, i, n, jumpTargets, opEqInt, opJumpIfTrue, opEqIntConstJumpTrue) ||
		cf.fuseCompareConstJump(body, i, n, jumpTargets, opGeInt, opJumpIfFalse, opGeIntConstJumpFalse) ||
		cf.fuseCompareConstJump(body, i, n, jumpTargets, opGtInt, opJumpIfFalse, opGtIntConstJumpFalse) ||
		cf.fuseStringConstJump(body, i, n, jumpTargets) ||
		cf.fuseNilTestJump(body, i, n, jumpTargets) ||
		cf.fuseIncIntJumpLt(body, i, n, jumpTargets) ||
		cf.fuseLenStringLtJump(body, i, n, jumpTargets) ||
		cf.fuseEqUintConstJumpFalse(body, i, n, jumpTargets) ||
		cf.fuseMapIndexOkJumpFalse(body, i, n, jumpTargets, opMapIndexOkIntInt, opMapIndexOkJumpIfFalseIntInt) ||
		cf.fuseMapIndexOkJumpFalse(body, i, n, jumpTargets, opMapIndexOkStringInt, opMapIndexOkJumpIfFalseStringInt) ||
		cf.fuseMapIndexOkJumpFalse(body, i, n, jumpTargets, opMapIndexOkStringString, opMapIndexOkJumpIfFalseStringString) ||
		cf.fuseMapIndexOkJumpFalse(body, i, n, jumpTargets, opMapIndexOkIntString, opMapIndexOkJumpIfFalseIntString) ||
		cf.fuseMapIndexOkJumpFalse(body, i, n, jumpTargets, opMapIndexOkIntGeneral, opMapIndexOkJumpIfFalseIntGeneral) ||
		cf.fuseMapIndexOkJumpFalse(body, i, n, jumpTargets, opMapIndexOkStringGeneral, opMapIndexOkJumpIfFalseStringGeneral)
}

// fuseArithConst fuses LoadIntConst or LoadUintConst with an arithmetic op into a
// constant-operand superinstruction.
//
// Handles int variants (AddInt, SubInt, MulInt) directly and the uint variants (AddUint,
// SubUint, BitAndUint) via tier-1 subops with the constant index carried in a following
// opExt word (the uint constant pool is up to 16 bits wide, which doesn't fit in the
// tier-1 operand budget alone).
//
// Takes body ([]instruction) which specifies the instruction sequence.
// Takes i (int) which specifies the current index.
// Takes n (int) which specifies the length.
// Takes jumpTargets (map[int]bool) which specifies the set of protected jump
// destinations.
//
// Returns true if a pattern was matched and applied.
func (cf *CompiledFunction) fuseArithConst(
	body []instruction, i, n int,
	jumpTargets map[int]bool,
) bool {
	if cf.fuseIntArithConst(body, i, n, jumpTargets) {
		return true
	}
	return cf.fuseUintArithConst(body, i, n, jumpTargets)
}

// fuseIntArithConst fuses a LoadIntConst followed by AddInt, SubInt, or MulInt into the
// corresponding *IntConst superinstruction.
//
// The load must write the same register the arith op reads as its right operand, and the
// slot immediately after the load must not be a jump target.
//
// Takes body ([]instruction) which is the instruction sequence.
// Takes i (int) which is the current index.
// Takes n (int) which is the length of body.
// Takes jumpTargets (map[int]bool) which marks protected jump destinations.
//
// Returns true when the pattern matched and was rewritten in place.
func (*CompiledFunction) fuseIntArithConst(
	body []instruction, i, n int,
	jumpTargets map[int]bool,
) bool {
	if i+1 >= n ||
		body[i].op != opLoadIntConst || body[i].c != 0 ||
		jumpTargets[i+1] {
		return false
	}
	next := body[i+1]
	if body[i].a != next.c {
		return false
	}
	var fusedOp opcode
	switch next.op {
	case opSubInt:
		fusedOp = opSubIntConst
	case opAddInt:
		fusedOp = opAddIntConst
	case opMulInt:
		fusedOp = opMulIntConst
	default:
		return false
	}
	body[i] = makeInstruction(fusedOp, next.a, next.b, body[i].b)
	body[i+1] = makeInstruction(opNop, 0, 0, 0)
	return true
}

// fuseUintArithConst fuses a LoadUintConst followed by AddUint, SubUint, or BitAndUint
// into a tier-1 subop carrying the constant via a trailing opExt word.
//
// The uint constant pool index can be up to 16 bits, so it does not fit in the tier-1
// operand budget alone; the opExt word holds the full index. The load's destination must
// match the arith op's right operand register.
//
// Takes body ([]instruction) which is the instruction sequence.
// Takes i (int) which is the current index.
// Takes n (int) which is the length of body.
// Takes jumpTargets (map[int]bool) which marks protected jump destinations.
//
// Returns true when the pattern matched and was rewritten in place.
func (*CompiledFunction) fuseUintArithConst(
	body []instruction, i, n int,
	jumpTargets map[int]bool,
) bool {
	if i+1 >= n ||
		body[i].op != opLoadUintConst ||
		jumpTargets[i+1] {
		return false
	}
	load := body[i]
	next := body[i+1]
	if load.a != next.c {
		return false
	}
	var fusedSubOp subOpcode
	switch next.op {
	case opAddUint:
		fusedSubOp = subOpAddUintConst
	case opSubUint:
		fusedSubOp = subOpSubUintConst
	case opBitAndUint:
		fusedSubOp = subOpBitAndUintConst
	default:
		return false
	}
	body[i] = makeInstruction(opDrillTier1, uint8(fusedSubOp), next.a, next.b)
	body[i+1] = makeInstruction(opExt, load.b, load.c, 0)
	return true
}

// fuseAddIntJump fuses AddIntConst + Jump into opAddIntJump + opExt.
//
// Takes body ([]instruction) which specifies the instruction sequence.
// Takes i (int) which specifies the current index.
// Takes n (int) which specifies the length.
// Takes jumpTargets (map[int]bool) which specifies the set of protected jump
// destinations.
//
// Returns true if a pattern was matched and applied.
func (*CompiledFunction) fuseAddIntJump(
	body []instruction, i, n int,
	jumpTargets map[int]bool,
) bool {
	if i+1 >= n ||
		body[i].op != opAddIntConst || !instrIsTier1SubOp(body[i+1], subOpJump) ||
		jumpTargets[i+1] {
		return false
	}
	raw := body[i+1].signedOffset()
	adj := raw + 1
	lo, hi := splitOffset(adj)
	body[i] = makeInstruction(opAddIntJump, body[i].a, body[i].b, body[i].c)
	body[i+1] = makeInstruction(opExt, lo, hi, 0)
	return true
}

// fuseConcatRune fuses RuneToString + ConcatString into opConcatRuneString.
//
// Operand layout: subOpRuneToString lives in tier-1 form {opDrillTier1,
// subOpRuneToString, dst=B, src=C}. The fusion reads dst from body[i].b and src from
// body[i].c.
//
// Takes body ([]instruction) which specifies the instruction sequence.
// Takes i (int) which specifies the current index.
// Takes n (int) which specifies the length.
// Takes jumpTargets (map[int]bool) which specifies the set of protected jump
// destinations.
//
// Returns true if a pattern was matched and applied.
func (*CompiledFunction) fuseConcatRune(
	body []instruction, i, n int,
	jumpTargets map[int]bool,
) bool {
	if i+1 >= n ||
		!instrIsTier1SubOp(body[i], subOpRuneToString) || body[i+1].op != opConcatString ||
		body[i].b != body[i+1].c ||
		jumpTargets[i+1] {
		return false
	}
	body[i] = makeInstruction(opConcatRuneString, body[i+1].a, body[i+1].b, body[i].c)
	body[i+1] = makeInstruction(opNop, 0, 0, 0)
	return true
}

// fuseAppendMove fuses APPEND_xxx + MOVE_GENERAL into an in-place APPEND_xxx where the
// new-slice destination is rewritten to equal the source-slice register. The downstream
// handler then takes the in-place fast path (Grow/SetLen/Index.Set on the addressable
// slice) instead of allocating a fresh reflect.Value to wrap the result on every append.
//
// Two encodings are recognised. The tier-0 generic form rewrites `APPEND R_new, R_src,
// R_elem; MOVE_GEN R_src, R_new` into `APPEND R_src, R_src, R_elem; NOP`. The tier-1
// typed form (subOpAppend{Int,String,Float,Bool,Uint}) rewrites the equivalent
// DRILL_TIER1 + EXT(R_elem) + MOVE_GEN window so the drill instruction targets R_src
// directly and the trailing move becomes a NOP; the ext word carrying R_elem is left
// untouched.
//
// Takes body ([]instruction) which specifies the instruction sequence.
// Takes i (int) which specifies the current index.
// Takes n (int) which specifies the length.
// Takes jumpTargets (map[int]bool) which specifies the set of protected jump
// destinations.
//
// Returns true if a pattern was matched and applied.
func (*CompiledFunction) fuseAppendMove(
	body []instruction, i, n int,
	jumpTargets map[int]bool,
) bool {
	if i+1 >= n || jumpTargets[i+1] {
		return false
	}
	appendInstr := body[i]
	if appendInstr.op == opAppend {
		return tryFuseTier0AppendMove(body, i, appendInstr)
	}
	return tryFuseTier1AppendMove(body, i, n, jumpTargets, appendInstr)
}

// optimiseLoadIntConst rewrites LoadIntConst to LoadIntConstSmall when the constant value
// fits in [0, 255].
//
// Takes body ([]instruction) which specifies the instruction sequence.
// Takes i (int) which specifies the current instruction index.
func (cf *CompiledFunction) optimiseLoadIntConst(body []instruction, i int) {
	if body[i].op != opLoadIntConst || body[i].c != 0 {
		return
	}
	if int(body[i].b) >= len(cf.intConstants) {
		return
	}
	value := cf.intConstants[body[i].b]
	if value >= 0 && value <= maxSmallConstant {
		body[i] = makeInstruction(opDrillTier1, uint8(subOpLoadIntConstSmall), body[i].a, safeconv.MustIntToUint8(int(value)))
	}
}

// optimiseLoadUintConst rewrites LoadUintConst to LoadUintConstSmall when the constant
// fits in the 8-bit inline immediate slot.
//
// Mirror of optimiseLoadIntConst for the uint bank.
//
// Takes body ([]instruction) which is the instruction sequence to scan and rewrite in
// place.
// Takes i (int) which is the index of the candidate instruction.
func (cf *CompiledFunction) optimiseLoadUintConst(body []instruction, i int) {
	if body[i].op != opLoadUintConst || body[i].c != 0 {
		return
	}
	if int(body[i].b) >= len(cf.uintConstants) {
		return
	}
	value := cf.uintConstants[body[i].b]
	if value <= uint64(maxSmallConstant) {
		body[i] = makeInstruction(opDrillTier1, uint8(subOpLoadUintConstSmall), body[i].a, safeconv.MustIntToUint8(int(value)))
	}
}

// fuseCompareConstJump attempts to fuse a LoadIntConst + cmpOp + jumpOp sequence into a
// single superinstruction.
//
// Takes body ([]instruction) which specifies the instruction sequence.
// Takes i (int) which specifies the current index.
// Takes n (int) which specifies the length.
// Takes jumpTargets (map[int]bool) which specifies the set of protected jump
// destinations.
// Takes cmpOp (opcode) which specifies the comparison opcode.
// Takes jumpOp (opcode) which specifies the jump opcode.
// Takes fusedOp (opcode) which specifies the fused opcode.
//
// Returns true if the pattern matched and was applied.
func (*CompiledFunction) fuseCompareConstJump(
	body []instruction, i, n int,
	jumpTargets map[int]bool,
	cmpOp, jumpOp, fusedOp opcode,
) bool {
	if i+2 >= n ||
		body[i].op != opLoadIntConst || body[i+1].op != cmpOp ||
		body[i+2].op != jumpOp ||
		body[i].a != body[i+1].c ||
		body[i+1].a != body[i+2].a ||
		body[i].c != 0 ||
		jumpTargets[i+1] || jumpTargets[i+2] {
		return false
	}
	raw := body[i+2].signedOffset()
	adj := raw + 1
	lo, hi := splitOffset(adj)
	body[i] = makeInstruction(fusedOp, body[i+1].b, body[i].b, 0)
	body[i+1] = makeInstruction(opExt, lo, hi, 0)
	body[i+2] = makeInstruction(opNop, 0, 0, 0)
	return true
}

// fuseIncIntJumpLt fuses IncInt(R) + LtInt(cmp, R, B) + JumpIfTrue(cmp, off) into
// opIncIntJumpLt(R, B, 0) + opExt(lo, hi, 0) + opNop.
//
// Operand layout: subOpTier2IncInt lives in tier-2 form {opDrillTier1, subOpDrillTier2,
// subOpTier2IncInt, R=C}.
//
// Takes body ([]instruction) which specifies the instruction sequence.
// Takes i (int) which specifies the current index.
// Takes n (int) which specifies the length.
// Takes jumpTargets (map[int]bool) which specifies the set of protected jump
// destinations.
//
// Returns true if a pattern was matched and applied.
func (*CompiledFunction) fuseIncIntJumpLt(
	body []instruction, i, n int,
	jumpTargets map[int]bool,
) bool {
	if i+2 >= n ||
		!instrIsTier2SubOp(body[i], subOpTier2IncInt) || body[i+1].op != opLtInt ||
		body[i+2].op != opJumpIfTrue ||
		body[i+1].b != body[i].c ||
		body[i+2].a != body[i+1].a ||
		jumpTargets[i+1] || jumpTargets[i+2] {
		return false
	}
	raw := body[i+2].signedOffset()
	adj := raw + 1
	lo, hi := splitOffset(adj)
	body[i] = makeInstruction(opDrillTier1, uint8(subOpIncIntJumpLt), body[i].c, body[i+1].c)
	body[i+1] = makeInstruction(opExt, lo, hi, 0)
	body[i+2] = makeInstruction(opNop, 0, 0, 0)
	return true
}

// fuseStringIndexToInt fuses opStringIndex + opUintToInt into opStringIndexToInt,
// avoiding the intermediate uint register and one tier-2 trampoline per string byte
// access converted to int.
//
// The fuser matches STRING_INDEX(R_uint, R_str, R_idx) followed by {opDrillTier1,
// subOpUintToInt, R_int=B, R_uint=C} and rewrites the pair to STRING_INDEX_TO_INT(R_int,
// R_str, R_idx) + NOP. subOpUintToInt lives in tier-1 form {opDrillTier1, subOpUintToInt,
// dst=B, src=C}.
//
// Takes body ([]instruction) which specifies the instruction sequence.
// Takes i (int) which specifies the current index.
// Takes n (int) which specifies the length.
// Takes jumpTargets (map[int]bool) which specifies the set of protected jump
// destinations.
//
// Returns true if a pattern was matched and applied.
func (*CompiledFunction) fuseStringIndexToInt(
	body []instruction, i, n int,
	jumpTargets map[int]bool,
) bool {
	if i+1 >= n ||
		body[i].op != opStringIndex || !instrIsTier1SubOp(body[i+1], subOpUintToInt) ||
		body[i].a != body[i+1].c ||
		jumpTargets[i+1] {
		return false
	}
	body[i] = makeInstruction(opStringIndexToInt, body[i+1].b, body[i].b, body[i].c)
	body[i+1] = makeInstruction(opNop, 0, 0, 0)
	return true
}

// fuseLenStringLtJump fuses opLenString + opLtInt + opJumpIfFalse into
// opLenStringLtJumpFalse + opExt, collapsing the entire for-loop condition `i < len(s)`
// into a single fused instruction.
//
// The fuser matches {opDrillTier1, subOpLenString, R_len, R_str} + LT_INT(R_bool, R_i,
// R_len) + JUMP_IF_FALSE(R_bool, lo, hi) and rewrites them to
// LEN_STRING_LT_JUMP_FALSE(R_i, R_str, 0) + EXT(lo, hi, 0) + NOP.
//
// Takes body ([]instruction) which specifies the instruction sequence.
// Takes i (int) which specifies the current index.
// Takes n (int) which specifies the length.
// Takes jumpTargets (map[int]bool) which specifies the set of protected jump
// destinations.
//
// Returns true if a pattern was matched and applied.
func (*CompiledFunction) fuseLenStringLtJump(
	body []instruction, i, n int,
	jumpTargets map[int]bool,
) bool {
	if i+2 >= n ||
		!instrIsTier1SubOp(body[i], subOpLenString) || body[i+1].op != opLtInt ||
		body[i+2].op != opJumpIfFalse ||
		body[i].b != body[i+1].c ||
		body[i+1].a != body[i+2].a ||
		jumpTargets[i+1] || jumpTargets[i+2] {
		return false
	}
	raw := body[i+2].signedOffset()
	adj := raw + 1
	lo, hi := splitOffset(adj)
	body[i] = makeInstruction(opDrillTier1, uint8(subOpLenStringLtJumpFalse), body[i+1].b, body[i].c)
	body[i+1] = makeInstruction(opExt, lo, hi, 0)
	body[i+2] = makeInstruction(opNop, 0, 0, 0)
	return true
}

// fuseNilTestJump fuses LoadNil(R) + EqGeneral/NeGeneral(C, X, R) + Jump into
// opTestNilJumpTrue/False(X, lo, hi) + opNop + opNop.
//
// Operand layout: subOpTier2LoadNil lives in tier-2 form {opDrillTier1, subOpDrillTier2,
// subOpTier2LoadNil, R=C}.
//
// Takes body ([]instruction) which specifies the instruction sequence.
// Takes i (int) which specifies the current index.
// Takes n (int) which specifies the length.
// Takes jumpTargets (map[int]bool) which specifies the set of protected jump
// destinations.
//
// Returns true if a pattern was matched and applied.
func (*CompiledFunction) fuseNilTestJump(
	body []instruction, i, n int,
	jumpTargets map[int]bool,
) bool {
	if i+2 >= n ||
		!instrIsTier2SubOp(body[i], subOpTier2LoadNil) ||
		(body[i+1].op != opEqGeneral && body[i+1].op != opNeGeneral) ||
		(body[i+2].op != opJumpIfTrue && body[i+2].op != opJumpIfFalse) ||
		body[i+1].a != body[i+2].a ||
		jumpTargets[i+1] || jumpTargets[i+2] {
		return false
	}
	nilRegister := body[i].c
	var testRegister uint8
	if body[i+1].b == nilRegister {
		testRegister = body[i+1].c
	} else if body[i+1].c == nilRegister {
		testRegister = body[i+1].b
	} else {
		return false
	}
	wantNilJump := (body[i+1].op == opEqGeneral && body[i+2].op == opJumpIfTrue) ||
		(body[i+1].op == opNeGeneral && body[i+2].op == opJumpIfFalse)
	raw := body[i+2].signedOffset()
	adj := raw + 2
	lo, hi := splitOffset(adj)
	var fusedOp opcode
	if wantNilJump {
		fusedOp = opTestNilJumpTrue
	} else {
		fusedOp = opTestNilJumpFalse
	}
	body[i] = makeInstruction(fusedOp, testRegister, lo, hi)
	body[i+1] = makeInstruction(opNop, 0, 0, 0)
	body[i+2] = makeInstruction(opNop, 0, 0, 0)
	return true
}

// fuseStringConstJump fuses LoadStringConst(R, index) + EqString(C, X, R) +
// JumpIfFalse(C, off) into opEqStringConstJumpFalse(X, index, 0) + opExt(lo, hi, 0) +
// opNop.
//
// Takes body ([]instruction) which specifies the instruction sequence.
// Takes i (int) which specifies the current index.
// Takes n (int) which specifies the length.
// Takes jumpTargets (map[int]bool) which specifies the set of protected jump
// destinations.
//
// Returns true if a pattern was matched and applied.
func (*CompiledFunction) fuseStringConstJump(
	body []instruction, i, n int,
	jumpTargets map[int]bool,
) bool {
	if i+2 >= n ||
		body[i].op != opLoadStringConst || body[i+1].op != opEqString ||
		body[i+2].op != opJumpIfFalse ||
		body[i].c != 0 ||
		body[i].a != body[i+1].c ||
		body[i+1].a != body[i+2].a ||
		jumpTargets[i+1] || jumpTargets[i+2] {
		return false
	}
	raw := body[i+2].signedOffset()
	adj := raw + 1
	lo, hi := splitOffset(adj)
	body[i] = makeInstruction(opEqStringConstJumpFalse, body[i+1].b, body[i].b, 0)
	body[i+1] = makeInstruction(opExt, lo, hi, 0)
	body[i+2] = makeInstruction(opNop, 0, 0, 0)
	return true
}

// syncSourceMapAfterOptimise zeroes source positions for instructions that were replaced
// with NOPs during optimisation, keeping the source map consistent with the instruction
// body.
//
// Takes body ([]instruction) which specifies the instruction sequence to synchronise.
func (cf *CompiledFunction) syncSourceMapAfterOptimise(body []instruction) {
	if cf.debugSourceMap == nil {
		return
	}
	for i, instr := range body {
		if isCanonicalNop(instr) && i < len(cf.debugSourceMap.positions) {
			cf.debugSourceMap.positions[i] = sourcePosition{}
		}
	}
}

// matchRangeCheckRegisters verifies the per-instruction register references inside the
// window agree with the canonical pattern and returns the inferred valueReg and condReg.
//
// Takes body ([]instruction) which is the instruction sequence.
// Takes i (int) which is the window start.
// Takes vRegLo (uint8) which is the low-bound load destination.
// Takes vRegHi (uint8) which is the high-bound load destination.
//
// Returns valueReg (uint8) which is the uint register holding the value being classified.
// Returns condReg (uint8) which is the int register propagated through the move and jump
// pair.
// Returns ok (bool) which is true when every register reference matches.
func matchRangeCheckRegisters(body []instruction, i int, vRegLo, vRegHi uint8) (valueReg, condReg uint8, ok bool) {
	if body[i+1].c != vRegLo {
		return 0, 0, false
	}
	cmpLo := body[i+1].a
	valueReg = body[i+1].b
	if body[i+2].c != cmpLo {
		return 0, 0, false
	}
	condReg = body[i+2].b
	if body[i+rangeCheckFirstJumpOffset].a != condReg {
		return 0, 0, false
	}
	if body[i+rangeCheckSecondLoadOffset+1].b != valueReg || body[i+rangeCheckSecondLoadOffset+1].c != vRegHi {
		return 0, 0, false
	}
	cmpHi := body[i+rangeCheckSecondLoadOffset+1].a
	if body[i+rangeCheckSecondMoveOffset].c != cmpHi || body[i+rangeCheckSecondMoveOffset].b != condReg {
		return 0, 0, false
	}
	if body[i+rangeCheckSecondJumpOffset].a != condReg {
		return 0, 0, false
	}
	return valueReg, condReg, true
}

// validateRangeCheckOffsets ensures the first jump chains to the second jump's position
// and the second jump's offset is non-negative (forward only).
//
// Takes body ([]instruction) which is the instruction sequence, and i (int) which is the
// window start.
//
// Returns int16 which is the second jump's signed offset that the rewrite encodes into
// the fused extension word.
// Returns bool which is true when both offsets are well-formed.
func validateRangeCheckOffsets(body []instruction, i int) (int16, bool) {
	if body[i+rangeCheckFirstJumpOffset].signedOffset() != rangeCheckFirstJumpDelta {
		return 0, false
	}
	off2 := body[i+rangeCheckSecondJumpOffset].signedOffset()
	if off2 < 0 {
		return 0, false
	}
	return off2, true
}

// instrIsTier1SubOp reports whether instr is the tier-1 form {opDrillTier1, subOp, *, *}.
//
// Peephole match sites that target tier-1 opcodes consult this helper instead of the bare
// `instr.op == opXxx` check.
//
// Takes instr (instruction) which is the instruction to classify.
// Takes subOp (subOpcode) which is the tier-1 sub-opcode to compare against.
//
// Returns true when instr carries the requested tier-1 sub-op.
func instrIsTier1SubOp(instr instruction, subOp subOpcode) bool {
	return instr.op == opDrillTier1 && subOpcode(instr.a) == subOp
}

// instrIsTier2SubOp reports whether instr is the tier-2 form {opDrillTier1,
// subOpDrillTier2, subOp, *}.
//
// Mirror of instrIsTier1SubOp for the deeper drill chain.
//
// Takes instr (instruction) which is the instruction to classify.
// Takes subOp (subOpcodeTier2) which is the tier-2 sub-opcode to compare against.
//
// Returns true when instr carries the requested tier-2 sub-op.
func instrIsTier2SubOp(instr instruction, subOp subOpcodeTier2) bool {
	return instr.op == opDrillTier1 &&
		subOpcode(instr.a) == subOpDrillTier2 &&
		subOpcodeTier2(instr.b) == subOp
}

// tryFuseTier0AppendMove handles the contiguous opAppend + opMoveGeneral pattern. Returns
// true when the fusion fires.
//
// Takes body ([]instruction) which is the instruction sequence to rewrite.
// Takes i (int) which is the opAppend index.
// Takes appendInstr (instruction) which is body[i].
//
// Returns true when the fusion fires.
func tryFuseTier0AppendMove(body []instruction, i int, appendInstr instruction) bool {
	moveInstr := body[i+1]
	if moveInstr.op != opMoveGeneral {
		return false
	}
	if moveInstr.a != appendInstr.b || moveInstr.b != appendInstr.a {
		return false
	}
	body[i] = makeInstruction(appendInstr.op, appendInstr.b, appendInstr.b, appendInstr.c)
	body[i+1] = makeInstruction(opNop, 0, 0, 0)
	return true
}

// tryFuseTier1AppendMove handles the DRILL_TIER1(subOpAppend*) + EXT + MOVE_GEN pattern.
// The MOVE_GEN lives at body[i+2] (after the extension word) and the rewrite rebrands the
// append destination so the runtime adapter takes the in-place helper path.
//
// Takes body ([]instruction) which is the instruction sequence to rewrite.
// Takes i (int) which is the opDrillTier1 index.
// Takes n (int) which is the body length.
// Takes jumpTargets (map[int]bool) which marks branch destinations.
// Takes appendInstr (instruction) which is body[i].
//
// Returns true when the fusion fires.
func tryFuseTier1AppendMove(body []instruction, i, n int, jumpTargets map[int]bool, appendInstr instruction) bool {
	if appendInstr.op != opDrillTier1 {
		return false
	}
	if !isTier1AppendSubOp(subOpcode(appendInstr.a)) {
		return false
	}
	if i+2 >= n || jumpTargets[i+2] {
		return false
	}
	if body[i+1].op != opExt {
		return false
	}
	moveInstr := body[i+2]
	if moveInstr.op != opMoveGeneral {
		return false
	}
	if moveInstr.a != appendInstr.c || moveInstr.b != appendInstr.b {
		return false
	}
	body[i] = makeInstruction(opDrillTier1, appendInstr.a, appendInstr.c, appendInstr.c)
	body[i+2] = makeInstruction(opNop, 0, 0, 0)
	return true
}

// isTier1AppendSubOp reports whether sub is one of the typed subOpAppend* sub-opcodes
// that this fusion recognises.
//
// Takes sub (subOpcode) which is the candidate sub-opcode.
//
// Returns true when sub is in the supported append family.
func isTier1AppendSubOp(sub subOpcode) bool {
	switch sub {
	case subOpAppendInt, subOpAppendString, subOpAppendFloat, subOpAppendBool, subOpAppendUint:
		return true
	default:
	}
	return false
}

// isCanonicalNop reports whether instr is the tier-3 canonical NOP encoding emitted by
// the peephole passes.
//
// The canonical NOP is the tier-3 form {opDrillTier1, subOpDrillTier2,
// subOpTier2DrillTier3, subOpTier3Nop} rather than an all-zero word so it dispatches
// deterministically through the drill cascade.
//
// Takes instr (instruction) which is the instruction to test.
//
// Returns true when instr is the canonical NOP encoding.
func isCanonicalNop(instr instruction) bool {
	return instr.op == opDrillTier1 &&
		subOpcode(instr.a) == subOpDrillTier2 &&
		subOpcodeTier2(instr.b) == subOpTier2DrillTier3 &&
		subOpcodeTier3(instr.c) == subOpTier3Nop
}
