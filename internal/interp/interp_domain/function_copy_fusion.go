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

// Copy-fusion peephole: recognises the
//
// 	GET_STRUCT_FIELD_GENERAL_T0 destination=T, sourceReceiver=A, sourceField=Fa
// 	SET_STRUCT_FIELD_GENERAL_T0 destinationReceiver=B, value=T, destinationField=Fb
//
// pair (with an optional intermediate CSE-emitted MOVE between the GET and the SET) and
// rewrites it into a single opCopyStructFieldGeneralT0 + opExt pair. The fuser owns the
// supporting liveness scan and a classifier that reports per-instruction reads/writes of
// general-bank registers; both ride along here so function.go stays under the file-length
// limit.

const (
	// copyFusionLookaheadLimit bounds the forward scan inside the liveness check used by
	// fuseCopyStructFieldGeneralT0, keeping compile-time bounded on pathologically large
	// functions.
	copyFusionLookaheadLimit = 24
)

// fuseCopyStructFieldGeneralT0 fuses an adjacent get/set struct-field pair into a single
// copy when the temp register is dead and the layouts are byte-compatible.
//
// Rewrites
//
//	GET_STRUCT_FIELD_GENERAL_T0 destination=T, sourceReceiver=A, sourceField=Fa
//	SET_STRUCT_FIELD_GENERAL_T0 destinationReceiver=B, value=T, destinationField=Fb
//
// into
//
//	COPY_STRUCT_FIELD_GENERAL_T0 sourceReceiver=A, destinationReceiver=B, sourceField=Fa
//	  opExt a=Fb
//
// A gap variant tolerates one neutral intermediate instruction between the GET and the
// SET (typically a CSE-emitted opMoveGeneral preparing the SET's receiver).
//
// Takes body ([]instruction) which is the function's instruction stream.
// Takes i (int) which is the candidate window start (GET position).
// Takes n (int) which is the length of body.
// Takes jumpTargets (map[int]bool) which marks branch destinations.
//
// Returns true on a successful rewrite.
func (cf *CompiledFunction) fuseCopyStructFieldGeneralT0(
	body []instruction,
	i, n int,
	jumpTargets map[int]bool,
) bool {
	if i+1 >= n {
		return false
	}

	getOp := body[i].op
	if getOp != opGetStructFieldGeneral && getOp != opGetStructFieldRawPointerT0 {
		return false
	}
	if cf.tryFuseCopyAdjacent(body, i, n, jumpTargets) {
		return true
	}
	return cf.tryFuseCopyWithGap(body, i, n, jumpTargets)
}

// tryFuseCopyAdjacent handles the strict GET; SET adjacency case.
//
// Takes body ([]instruction) which is the function's instruction stream.
// Takes i (int) which is the candidate window start (GET position).
// Takes n (int) which is the length of body.
// Takes jumpTargets (map[int]bool) which marks branch destinations.
//
// Returns true on a successful rewrite.
func (cf *CompiledFunction) tryFuseCopyAdjacent(
	body []instruction,
	i, n int,
	jumpTargets map[int]bool,
) bool {
	if i+1 >= n {
		return false
	}
	if body[i+1].op != opSetStructFieldGeneral {
		return false
	}

	if jumpTargets[i+1] {
		return false
	}
	tempReg := body[i].a
	if body[i+1].b != tempReg {
		return false
	}
	sourceLayoutIndex := body[i].c
	destinationLayoutIndex := body[i+1].c
	if !cf.copyFusionLayoutsCompatible(sourceLayoutIndex, destinationLayoutIndex) {
		return false
	}

	if !generalRegisterIsDeadAfter(body, n, i+2, tempReg, jumpTargets) {
		return false
	}
	sourceReceiver := body[i].b
	destinationReceiver := body[i+1].a
	body[i] = makeInstruction(opCopyStructFieldGeneralT0, sourceReceiver, destinationReceiver, sourceLayoutIndex)
	body[i+1] = makeInstruction(opExt, destinationLayoutIndex, 0, 0)
	return true
}

// copyFusionLayoutsCompatible reports whether two struct-field layouts have identical
// storage shapes for a direct typedmemmove.
//
// A direct typedmemmove between source and destination is only sound when both share the
// same reflect.Kind (so byte size matches; for example an Interface field is 16 bytes and
// a Pointer field is 8 bytes, and copying between them would either truncate or read past
// the source storage) and the same FieldTypeIndex (so the destination GC pointer map
// matches the source content; typedmemmove emits write barriers based on the type's
// pointer map, and a mismatch would either skip barriers we need or fire barriers on
// non-pointer bits).
//
// When the layouts disagree the original GET+SET sequence is the correct path: the SET
// handler routes through coerceValue and reflect.Value.Set, which performs the kind-aware
// conversion (such as boxing a Pointer into an Interface) that the fast copy cannot
// reproduce.
//
// Takes sourceLayoutIndex (uint8) which is the source GET op's layout table index.
// Takes destinationLayoutIndex (uint8) which is the destination SET op's layout table
// index.
//
// Returns true when the two layouts are byte-for-byte interchangeable.
func (cf *CompiledFunction) copyFusionLayoutsCompatible(sourceLayoutIndex, destinationLayoutIndex uint8) bool {
	if int(sourceLayoutIndex) >= len(cf.structLayoutTable) {
		return false
	}
	if int(destinationLayoutIndex) >= len(cf.structLayoutTable) {
		return false
	}
	sourceLayout := cf.structLayoutTable[sourceLayoutIndex]
	destinationLayout := cf.structLayoutTable[destinationLayoutIndex]
	if sourceLayout.Kind != destinationLayout.Kind {
		return false
	}
	if sourceLayout.FieldTypeIndex != destinationLayout.FieldTypeIndex {
		return false
	}
	return true
}

// tryFuseCopyWithGap handles the GET; X; SET case where a single neutral instruction X
// sits between the GET and the SET.
//
// The intermediate is typically a CSE-emitted opMoveGeneral preparing the SET's receiver.
// X is moved BEFORE the GET so the fused layout
//
//	X
//	COPY sourceReceiver=A, destinationReceiver=B, sourceField=Fa
//	EXT  destinationLayoutIndex=Fb
//
// preserves observable semantics. Safety requires that X neither reads nor writes the
// GET's destination register T (reads would change T's observed value once X runs before
// GET; writes are also caught by the post-SET liveness scan), X does not write the GET's
// source receiver register A (the COPY needs A intact), and PC i+1 and i+2 are not jump
// targets.
//
// Takes body ([]instruction) which is the function's instruction stream.
// Takes i (int) which is the candidate window start (GET position).
// Takes n (int) which is the length of body.
// Takes jumpTargets (map[int]bool) which marks branch destinations.
//
// Returns true on a successful rewrite.
func (cf *CompiledFunction) tryFuseCopyWithGap(
	body []instruction,
	i, n int,
	jumpTargets map[int]bool,
) bool {
	if i+2 >= n {
		return false
	}
	if body[i+2].op != opSetStructFieldGeneral {
		return false
	}
	if jumpTargets[i+1] || jumpTargets[i+2] {
		return false
	}
	tempReg := body[i].a
	sourceReceiver := body[i].b
	if body[i+2].b != tempReg {
		return false
	}
	sourceLayoutIndex := body[i].c
	destinationLayoutIndex := body[i+2].c
	if !cf.copyFusionLayoutsCompatible(sourceLayoutIndex, destinationLayoutIndex) {
		return false
	}
	intermediate := body[i+1]
	if intermediate.op == opNop || intermediate.op == opExt {
		return false
	}
	intermediateReads, intermediateWrites := classifyGeneralRegisterUse(intermediate, tempReg)
	if intermediateReads || intermediateWrites {
		return false
	}

	_, writesSource := classifyGeneralRegisterUse(intermediate, sourceReceiver)
	if writesSource {
		return false
	}

	if !generalRegisterIsDeadAfter(body, n, i+3, tempReg, jumpTargets) {
		return false
	}
	destinationReceiver := body[i+2].a

	body[i] = intermediate
	body[i+1] = makeInstruction(opCopyStructFieldGeneralT0, sourceReceiver, destinationReceiver, sourceLayoutIndex)
	body[i+2] = makeInstruction(opExt, destinationLayoutIndex, 0, 0)
	return true
}

// generalRegisterIsDeadAfter reports whether a general-bank register is provably dead at
// startPC.
//
// Scans forward at most copyFusionLookaheadLimit instructions. Jump-target merges are
// tolerated when the merging instruction unconditionally writes reg, because the merged
// predecessor values do not matter once the write clobbers them. The classifier is
// conservative on unknown opcodes, so any operand of an unrecognised instruction is
// treated as a potential read; the COPY fuser is correctness-critical and leans towards
// refusing fusion when it cannot prove safety.
//
// Takes body ([]instruction) which is the function's instruction stream.
// Takes n (int) which is the length of body.
// Takes startPC (int) which is the first PC the scan inspects.
// Takes reg (uint8) which is the general-bank register whose liveness is checked.
// Takes jumpTargets (map[int]bool) which marks branch destinations.
//
// Returns true on a clean write-before-read, false on a read-before-write or budget
// exhaustion.
func generalRegisterIsDeadAfter(body []instruction, n, startPC int, reg uint8, jumpTargets map[int]bool) bool {
	return generalRegisterIsDeadFrom(body, n, startPC, reg, jumpTargets, copyFusionLookaheadLimit)
}

// generalRegisterIsDeadFrom is the bounded forward scan implementation behind
// generalRegisterIsDeadAfter.
//
// The budget caps the total instructions inspected across any jumps the scanner follows,
// so runaway scans on adversarial input cost at most copyFusionLookaheadLimit
// instructions.
//
// Takes body ([]instruction) which is the function's instruction stream.
// Takes n (int) which is the length of body.
// Takes startPC (int) which is the first PC the scan inspects.
// Takes reg (uint8) which is the general-bank register whose liveness is checked.
// Takes jumpTargets (map[int]bool) which marks branch destinations.
// Takes budget (int) which is the remaining instruction inspection budget.
//
// Returns true when reg is provably dead within the budget, false otherwise.
func generalRegisterIsDeadFrom(body []instruction, n, startPC int, reg uint8, jumpTargets map[int]bool, budget int) bool {
	pc := startPC
	for budget > 0 && pc < n {
		instr := body[pc]
		readsReg, writesReg := classifyGeneralRegisterUse(instr, reg)
		if readsReg {
			return false
		}
		if writesReg {
			return true
		}
		if isConditionalJump(instr) {
			return scanBothConditionalSuccessors(body, n, pc, instr, reg, jumpTargets, budget)
		}
		if isUnconditionalJump(instr) {
			targetPC, ok := unconditionalJumpTarget(pc, instr)
			if !ok || targetPC < 0 || targetPC >= n {
				return false
			}
			budget--
			pc = targetPC
			continue
		}
		budget--
		pc++
	}
	return false
}

// scanBothConditionalSuccessors recurses into the fall-through and taken edges of a
// conditional jump with a halved budget each.
//
// Both branches must independently confirm reg is dead; either reading it forces the
// fuser to refuse.
//
// Takes body ([]instruction) which is the function's instruction stream.
// Takes n (int) which is the length of body.
// Takes pc (int) which is the conditional jump's PC.
// Takes instr (instruction) which is the conditional jump instruction.
// Takes reg (uint8) which is the general-bank register whose liveness is checked.
// Takes jumpTargets (map[int]bool) which marks branch destinations.
// Takes budget (int) which is the remaining instruction inspection budget.
//
// Returns true when both successors confirm reg is dead.
func scanBothConditionalSuccessors(body []instruction, n, pc int, instr instruction, reg uint8, jumpTargets map[int]bool, budget int) bool {
	targetPC, ok := conditionalJumpTarget(pc, instr, body)
	if !ok {
		return false
	}
	halfBudget := budget / 2
	if halfBudget < 1 {
		return false
	}
	if !generalRegisterIsDeadFrom(body, n, pc+1, reg, jumpTargets, halfBudget) {
		return false
	}
	return generalRegisterIsDeadFrom(body, n, targetPC, reg, jumpTargets, halfBudget)
}

// conditionalJumpTarget decodes the absolute jump target for a conditional branch.
//
// Supports the variants the dead-check might encounter: opJumpIfFalse, opJumpIfTrue,
// opTestNilJump*, and the tier-1 conditional sub-ops listed in isConditionalJump.
//
// Takes pc (int) which is the PC of the conditional jump.
// Takes instr (instruction) which is the conditional jump instruction.
// Takes body ([]instruction) which is the function's instruction stream.
//
// Returns the absolute target PC and true on success, or 0 and false when the encoding is
// not recognised.
func conditionalJumpTarget(pc int, instr instruction, body []instruction) (int, bool) {
	switch instr.op {
	case opJumpIfFalse, opJumpIfTrue, opTestNilJumpFalse, opTestNilJumpTrue:
		offset := int(instr.signedOffset())
		return pc + 1 + offset, true
	default:
	}
	if instr.op == opDrillTier1 {
		switch subOpcode(instr.a) {
		case subOpEqUintConstJumpFalse, subOpRangeCheckUintJumpFalse:

			_ = body
			return 0, false
		default:
		}
	}
	return 0, false
}

// isConditionalJump reports whether instr is any conditional branch shape.
//
// Recognises opJumpIfFalse, opJumpIfTrue, opTestNilJump*, and the tier-1 conditional-jump
// sub-ops. Conditional branches have two reachable successors; the dead-check abandons
// analysis at these because following both paths would require a per-PC liveness lattice
// we do not compute.
//
// Takes instr (instruction) which is the instruction to classify.
//
// Returns true when instr is a conditional branch.
func isConditionalJump(instr instruction) bool {
	switch instr.op {
	case opJumpIfFalse, opJumpIfTrue, opTestNilJumpFalse, opTestNilJumpTrue:
		return true
	default:
	}
	if instr.op == opDrillTier1 {
		switch subOpcode(instr.a) {
		case subOpEqUintConstJumpFalse, subOpRangeCheckUintJumpFalse:
			return true
		default:
		}
	}
	return false
}

// isUnconditionalJump reports whether instr is an unconditional forward or backward jump.
//
// Recognises the tier-1 subOpJump shape the compiler emits. Conditional jumps such as
// opJumpIfFalse are excluded because their fall-through could still reach a read of reg.
//
// Takes instr (instruction) which is the instruction to classify.
//
// Returns true when instr is an unconditional jump.
func isUnconditionalJump(instr instruction) bool {
	if instr.op != opDrillTier1 {
		return false
	}
	return subOpcode(instr.a) == subOpJump
}

// unconditionalJumpTarget decodes the signed 16-bit offset packed into operands B|C of a
// subOpJump and returns the absolute target PC. The offset is relative to the PC
// immediately following the jump instruction.
//
// Takes jumpPC (int) which is the PC of the jump instruction.
// Takes instr (instruction) which is the subOpJump instruction whose offset is decoded.
//
// Returns the absolute target PC and true on success, or 0 and false when instr is not an
// unconditional jump.
func unconditionalJumpTarget(jumpPC int, instr instruction) (int, bool) {
	if !isUnconditionalJump(instr) {
		return 0, false
	}
	target := jumpPC + 1 + int(instr.signedOffset())
	return target, true
}

// classifyGeneralRegisterUse reports whether instr reads and writes general-bank register
// reg.
//
// Cases are grouped by the (reads, writes) signature so the table is dense and the switch
// has no identical branches. Anything not enumerated conservatively reports a read
// whenever reg appears as an operand, which is safe (fusion refused) but
// over-restrictive.
//
// Takes instr (instruction) which is the instruction to classify.
// Takes reg (uint8) which is the general-bank register being queried.
//
// Returns reads (bool) which is true when instr reads reg.
// Returns writes (bool) which is true when instr writes reg.
func classifyGeneralRegisterUse(instr instruction, reg uint8) (reads, writes bool) {
	switch instr.op {
	case opJumpIfFalse, opJumpIfTrue, opDrillTier1, opExt:

		return false, false
	case opCall, opTailCall, opCallMethod, opCallMethodInlineable, opCallNative:

		return true, false
	case opLoadGeneralConst:

		return false, instr.a == reg
	case opGetStructFieldGeneral, opGetStructFieldRawPointerT0, opMoveGeneral:

		return instr.b == reg, instr.a == reg
	case opSetStructFieldGeneral, opCopyStructFieldGeneralT0:

		return instr.a == reg || instr.b == reg, false
	case opGetStructFieldIntT0, opGetStructFieldUint, opGetStructFieldFloat, opGetStructFieldBool,
		opEqInterfaceNil, opNeInterfaceNil:

		return instr.b == reg, false
	case opSetStructFieldIntT0, opSetStructFieldUint, opSetStructFieldFloat, opSetStructFieldBool,
		opTestNilJumpFalse, opTestNilJumpTrue:

		return instr.a == reg, false
	case opEqGeneral, opNeGeneral, opLtGeneral, opLeGeneral, opGtGeneral, opGeGeneral:

		return instr.b == reg || instr.c == reg, false
	default:

		return instr.a == reg || instr.b == reg || instr.c == reg, false
	}
}
