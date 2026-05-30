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

const (
	// minTruncateAfterBitAndWindow is the minimum body length for which
	// elideRedundantTruncateAfterBitAnd has a complete match window (BIT_AND_UINT_CONST +
	// EXT + TRUNCATE_NARROW).
	minTruncateAfterBitAndWindow = 3

	// minBitAndAfterTruncateWindow is the minimum body length for which
	// elideRedundantBitAndAfterTruncate has a complete match window (TRUNCATE_NARROW +
	// BIT_AND_UINT_CONST + EXT + MOVE_UINT).
	minBitAndAfterTruncateWindow = 4

	// truncateNarrowWidth32 is the bit width passed to opTruncateNarrow to denote "truncate
	// to 32 bits". The fast-mask peepholes only fire for this width.
	truncateNarrowWidth32 = 32

	// uint32Mask is the 32-bit all-ones constant the fast-mask peepholes expect to find in
	// the uintConstants pool.
	uint32Mask = 0xFFFFFFFF
)

// uintRegReadDecision is the per-instruction verdict from classifyUintRegRead.
type uintRegReadDecision uint8

const (
	// uintRegReadContinue means the instruction does not read reg and the scan should
	// advance.
	uintRegReadContinue uintRegReadDecision = iota

	// uintRegReadSafe means the instruction is an end-of-scan boundary (function return) and
	// the elision is safe.
	uintRegReadSafe

	// uintRegReadEscapes means the instruction may read reg and the elision must be refused.
	uintRegReadEscapes
)

// elideRedundantTruncateAfterBitAnd nops a TRUNCATE_NARROW that follows a BIT_AND with a
// 32-bit mask.
//
// The target pattern is BIT_AND_UINT_CONST destReg, srcReg followed by EXT constantIndex
// (with uintConstants[constantIndex] == 0xFFFFFFFF) and TRUNCATE_NARROW destReg, 32.
// Because the BIT_AND already clears every bit above 32 the truncate is a guaranteed
// no-op. Polyast Eval bodies emit this shape for every `& resultMask` so the elision pays
// back per recursive descent. Width 32 only; other widths legitimately narrow further.
// The destReg match guards against eliding a truncate that targets a different register
// than the BIT_AND wrote. The pattern spans indices i, i+1, i+2 and is skipped on
// jump-target collisions to keep branch destinations stable.
//
// Takes body ([]instruction) which is the instruction sequence to scan and rewrite in
// place.
func (cf *CompiledFunction) elideRedundantTruncateAfterBitAnd(body []instruction) {
	if len(body) < minTruncateAfterBitAndWindow {
		return
	}
	jumpTargets := cf.buildJumpTargets(body)
	for i := 0; i+2 < len(body); i++ {
		if !cf.matchesTruncateAfterBitAnd(body, i, jumpTargets) {
			continue
		}
		body[i+2] = makeInstruction(opNop, 0, 0, 0)
	}
}

// matchesTruncateAfterBitAnd reports whether the three-instruction window starting at i
// is the redundant BIT_AND_UINT_CONST(0xFFFFFFFF) + EXT + TRUNCATE_NARROW(32) pattern
// safe to collapse.
//
// Takes body ([]instruction) which is the instruction sequence.
// Takes i (int) which is the candidate start index.
// Takes jumpTargets (map[int]bool) which marks branch destinations whose PCs must not be
// rewritten.
//
// Returns true when every match condition holds for this window.
func (cf *CompiledFunction) matchesTruncateAfterBitAnd(body []instruction, i int, jumpTargets map[int]bool) bool {
	if body[i].op != opDrillTier1 || subOpcode(body[i].a) != subOpBitAndUintConst {
		return false
	}
	if body[i+1].op != opExt {
		return false
	}
	if body[i+2].op != opTruncateNarrow {
		return false
	}
	if body[i+2].a != body[i].b {
		return false
	}
	if body[i+2].b != truncateNarrowWidth32 {
		return false
	}
	constantIndex := int(body[i+1].a)
	if constantIndex >= len(cf.uintConstants) || cf.uintConstants[constantIndex] != uint32Mask {
		return false
	}
	return !jumpTargets[i+2]
}

// elideRedundantBitAndAfterTruncate folds out a BIT_AND that follows a TRUNCATE_NARROW.
//
// Targets TRUNCATE_NARROW X, 32 followed by BIT_AND_UINT_CONST Y, X (with mask 0xFFFFFFFF
// carried in the EXT word) and a trailing MOVE_UINT Z, Y. The pass folds the MOVE source
// from Y to X and replaces the BIT_AND + EXT pair with two opNops, so the redundant mask
// disappears. Polyast Eval bodies emit this shape for every `return (a + b) & resultMask`
// statement. Safe only when Y is dead after the MOVE; the dead-Y check is a conservative
// forward scan from the MOVE position requiring Y not to appear as any operand in
// subsequent uint-bank reads before control reaches a return.
//
// Takes body ([]instruction) which is the instruction sequence to scan and rewrite in
// place.
func (cf *CompiledFunction) elideRedundantBitAndAfterTruncate(body []instruction) {
	if len(body) < minBitAndAfterTruncateWindow {
		return
	}
	jumpTargets := cf.buildJumpTargets(body)
	for i := 0; i+minTruncateAfterBitAndWindow < len(body); i++ {
		x, _, moveIndex, ok := cf.matchBitAndAfterTruncate(body, i, jumpTargets)
		if !ok {
			continue
		}
		body[moveIndex].c = x
		body[i+1] = makeInstruction(opNop, 0, 0, 0)
		body[i+2] = makeInstruction(opNop, 0, 0, 0)
	}
}

// matchBitAndAfterTruncate reports whether the window starting at i matches the
// TRUNCATE_NARROW(X, 32) + BIT_AND_UINT_CONST(Y, X, mask) + EXT + (opNop)* + MOVE_UINT(Z,
// Y) collapse pattern.
//
// Takes body ([]instruction) which is the instruction sequence.
// Takes i (int) which is the candidate start index.
// Takes jumpTargets (map[int]bool) which marks branch destinations whose PCs must not be
// rewritten.
//
// Returns x (the truncate destination retained as the MOVE source).
// Returns y (the BIT_AND destination scheduled for elision).
// Returns moveIndex (the index of the trailing MOVE_UINT to rewrite).
// Returns ok (true when every match condition holds).
func (cf *CompiledFunction) matchBitAndAfterTruncate(body []instruction, i int, jumpTargets map[int]bool) (x, y uint8, moveIndex int, ok bool) {
	if body[i].op != opTruncateNarrow || body[i].b != truncateNarrowWidth32 {
		return 0, 0, 0, false
	}
	x = body[i].a
	if body[i+1].op != opDrillTier1 || subOpcode(body[i+1].a) != subOpBitAndUintConst {
		return 0, 0, 0, false
	}
	if body[i+1].c != x {
		return 0, 0, 0, false
	}
	y = body[i+1].b
	if body[i+2].op != opExt {
		return 0, 0, 0, false
	}
	constantIndex := int(body[i+2].a)
	if constantIndex >= len(cf.uintConstants) || cf.uintConstants[constantIndex] != uint32Mask {
		return 0, 0, 0, false
	}
	moveIndex = skipTrailingNops(body, i+minTruncateAfterBitAndWindow)
	if moveIndex >= len(body) {
		return 0, 0, 0, false
	}
	if body[moveIndex].op != opDrillTier1 || subOpcode(body[moveIndex].a) != subOpMoveUint {
		return 0, 0, 0, false
	}
	if body[moveIndex].c != y {
		return 0, 0, 0, false
	}
	if jumpTargets[i+1] || jumpTargets[i+2] || jumpTargets[moveIndex] {
		return 0, 0, 0, false
	}
	if cf.uintRegReadAfter(body, moveIndex+1, y, jumpTargets) {
		return 0, 0, 0, false
	}
	return x, y, moveIndex, true
}

// uintRegReadAfter reports whether any instruction at or after startIndex reads the given
// uint-bank register.
//
// Conservative: returns true for any op that could plausibly read a uint register
// operand. The forward scan is a straight-line liveness walk that does not model the CFG,
// so it returns true the moment it crosses any jump target inside the window; at such a
// PC a predecessor outside the linear range could observe reg, making the "dead"
// conclusion unsound. The restriction confines the elision to a single straight-line
// region with no incoming edges. Treats opTruncateNarrow on the same reg as a read (reg
// &= mask reads the operand); opNop / opTier3-Nop are guaranteed no-ops and the scan
// skips them.
//
// Takes body ([]instruction) which is the instruction sequence to scan.
// Takes startIndex (int) which is the first index to inspect.
// Takes reg (uint8) which is the uint-bank register to check for reads.
// Takes jumpTargets (map[int]bool) which marks every branch destination in the function;
// a target inside the window forces a conservative "may read" verdict.
//
// Returns true when any subsequent instruction may read reg.
func (*CompiledFunction) uintRegReadAfter(body []instruction, startIndex int, reg uint8, jumpTargets map[int]bool) bool {
	for j := startIndex; j < len(body); j++ {
		if jumpTargets[j] {
			return true
		}
		decision := classifyUintRegRead(body[j], reg)
		if decision == uintRegReadContinue {
			continue
		}
		return decision == uintRegReadEscapes
	}
	return false
}

// skipTrailingNops advances past any opNop placeholders (encoded as opDrillTier1 with
// all-zero operands) starting at startIndex.
//
// The MOVE following the BIT_AND elision may be separated from the EXT by intervening
// opNop slots: the earlier elideRedundantTruncateAfterBitAnd converted a trailing
// TRUNCATE into NOP, and that slot now sits between EXT and the MOVE.
//
// Takes body ([]instruction) which is the instruction sequence.
// Takes startIndex (int) which is the first index to inspect.
//
// Returns the first index at or after startIndex that is not opNop.
func skipTrailingNops(body []instruction, startIndex int) int {
	moveIndex := startIndex
	for moveIndex < len(body) && body[moveIndex].op == opDrillTier1 &&
		body[moveIndex].a == 0 && body[moveIndex].b == 0 && body[moveIndex].c == 0 {
		moveIndex++
	}
	return moveIndex
}

// classifyUintRegRead returns the per-instruction decision for the "is reg read after
// this point?" scan.
//
// Conservative: returns uintRegReadEscapes for any op that could plausibly read uint
// register operands. The goal is correctness - false positives only block the peephole;
// false negatives would corrupt the rewrite.
//
// Treats opTruncateNarrow on the same reg as a read (reg &= mask reads the operand) and
// returns the escapes verdict; opNop / opTier3-Nop are guaranteed no-ops and continue the
// scan.
//
// Takes instr (instruction) which is the candidate instruction.
// Takes reg (uint8) which is the uint-bank register to check for reads.
//
// Returns the decision verdict.
func classifyUintRegRead(instr instruction, reg uint8) uintRegReadDecision {
	if instr.op == opDrillTier1 && instr.a == 0 && instr.b == 0 && instr.c == 0 {
		return uintRegReadContinue
	}
	switch instr.op {
	case opAddUint, opSubUint, opMulUint, opEqUint, opGeUint, opLeUint:
		if instr.b == reg || instr.c == reg {
			return uintRegReadEscapes
		}
		return uintRegReadContinue
	case opTruncateNarrow:
		if instr.a == reg {
			return uintRegReadEscapes
		}
		return uintRegReadContinue
	case opDrillTier1:
		return classifyUintRegReadTier1(instr, reg)
	case opExt:
		return uintRegReadEscapes
	default:
	}
	return uintRegReadEscapes
}

// classifyUintRegReadTier1 specialises classifyUintRegRead for opDrillTier1 instructions
// whose sub-opcode lives in instr.a.
//
// Takes instr (instruction) which is the opDrillTier1 wrapper.
// Takes reg (uint8) which is the uint-bank register to check.
//
// Returns the decision verdict for this sub-op.
func classifyUintRegReadTier1(instr instruction, reg uint8) uintRegReadDecision {
	switch subOpcode(instr.a) {
	case subOpMoveUint, subOpBitAndUintConst, subOpAddUintConst, subOpSubUintConst:
		if instr.c == reg {
			return uintRegReadEscapes
		}
		return uintRegReadContinue
	case subOpLoadUintConstSmall:
		return uintRegReadContinue
	case subOpDrillTier2:
		if subOpcodeTier2(instr.b) == subOpTier2Return {
			return uintRegReadSafe
		}
		return uintRegReadEscapes
	default:
	}
	return uintRegReadEscapes
}
