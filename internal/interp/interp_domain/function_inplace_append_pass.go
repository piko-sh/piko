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

// verifyInPlaceAppendSafety is the post-bytecode safety gate for the in-place append
// opcodes the AST-level fast paths emit (opAppendByteFastInPlace, opAppendInPlace,
// opAppendSpreadInPlace, and the subOpAppendUintInPlace / subOpAppendByteSpreadInPlace
// sub-ops). It runs AFTER runPointerAliasAnalysis has built cf.aliasInfo so it can
// consult the per-PC alias classes the AST pre-pass had no access to.
//
// For each in-place site, the pass verifies that the source register is provably
// non-aliased to every other live general-bank register at that program point. If alias
// analysis disagrees with the AST-level safety predicate, the opcode is DEMOTED back to
// its allocate-fresh-slot sibling (opAppendByteFast, opAppend, opAppendSpread,
// subOpAppendUint, subOpStarAppendByteSpread for the sub-op spread). The fresh-slot path
// is always safe - the worst case for a demotion is no perf win, never a correctness
// regression.
//
// When cf.aliasInfo is nil (analysis skipped due to function-size cap), the pass leaves
// every site untouched: the AST predicate's conservative emission is the best available
// safety bound. Same when the function body is empty.
//
// Called from runPostPurityPeepholePass alongside the other post-purity peephole passes.
func (cf *CompiledFunction) verifyInPlaceAppendSafety() {
	if cf == nil || len(cf.body) == 0 {
		return
	}
	if cf.aliasInfo == nil {
		return
	}
	for index := range cf.body {
		instruction := cf.body[index]
		switch instruction.op {
		case opAppendByteFastInPlace:
			if !cf.inPlaceAppendSiteIsSafe(index, instruction.b) {
				cf.body[index] = makeInstruction(opAppendByteFast, instruction.a, instruction.b, instruction.c)
			}
		case opAppendInPlace:
			if !cf.inPlaceAppendSiteIsSafe(index, instruction.b) {
				cf.body[index] = makeInstruction(opAppend, instruction.a, instruction.b, instruction.c)
			}
		case opAppendSpreadInPlace:
			if !cf.inPlaceAppendSiteIsSafe(index, instruction.b) {
				cf.body[index] = makeInstruction(opAppendSpread, instruction.a, instruction.b, instruction.c)
			}
		case opDrillTier1:
			cf.maybeDemoteTier1InPlaceAppend(index, instruction)
		default:
		}
	}
}

// maybeDemoteTier1InPlaceAppend demotes unsafe tier-1 in-place sub-ops.
//
// Demotes to allocate-fresh-slot siblings when alias analysis cannot confirm safety. Only
// demotes sub-ops whose fresh-slot sibling has the SAME operand encoding (so the runtime
// dispatcher reads the same operand fields). subOpAppendByteSpreadInPlace has no
// same-encoding fresh-slot sibling (subOpStarAppendByteSpread expects a pointer receiver,
// not a slice) so its safety is left to the handler's runtime arena-ownership check.
//
// Takes index (int) which is the PC of the candidate instruction.
// Takes instr (instruction) which is the instruction at that PC.
func (cf *CompiledFunction) maybeDemoteTier1InPlaceAppend(index int, instr instruction) {
	if subOpcode(instr.a) == subOpAppendUintInPlace && !cf.inPlaceAppendSiteIsSafe(index, instr.c) {
		cf.body[index] = makeInstruction(opDrillTier1, uint8(subOpAppendUint), instr.b, instr.c)
	}
}

// inPlaceAppendSiteIsSafe reports whether the source register is unaliased.
//
// Uses cf.aliasInfo to query per-PC alias classes and generalRegisterIsDeadAfter to skip
// dead registers (their stale alias class no longer matters). Returns false on the first
// observed aliasing register - the source is potentially aliased and in-place mutation
// would break Go slice semantics.
//
// Takes pc (int) which is the PC of the candidate in-place instruction.
// Takes sourceRegister (uint8) which is the register whose alias class the predicate
// checks.
//
// Returns bool which is true when no live register shares the source's alias class.
func (cf *CompiledFunction) inPlaceAppendSiteIsSafe(pc int, sourceRegister uint8) bool {
	if cf.aliasInfo == nil {
		return false
	}
	peak := cf.numRegisters[registerGeneral]
	if peak == 0 {
		return true
	}
	jumpTargets := buildAllJumpTargets(cf.body)
	for register := range peak {
		other := uint8(register)
		if other == sourceRegister {
			continue
		}
		if generalRegisterIsDeadAfter(cf.body, len(cf.body), pc, other, jumpTargets) {
			continue
		}
		if cf.aliasInfo.mayAlias(pc, sourceRegister, other) {
			return false
		}
	}
	return true
}
