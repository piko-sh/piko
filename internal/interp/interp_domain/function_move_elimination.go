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
	// moveElimScanWindow bounds how far the pass walks forward looking for the unique
	// downstream reader of a MOVE's destination. Matches the convention of existing
	// pattern-fusers (8-slot lookahead) with extra headroom for chained MOVE patterns that
	// arise from tuple assigns.
	moveElimScanWindow = 16

	// tier1ScanOperandPositionC is the operand index of `c` (the source operand of a tier-1
	// same-bank MOVE) within an instruction.
	tier1ScanOperandPositionC = 2
)

// moveScanDecision describes the outcome of inspecting a single instruction during
// MOVE-elimination scanning.
type moveScanDecision uint8

const (
	// moveScanContinue keeps scanning forward.
	moveScanContinue moveScanDecision = iota

	// moveScanBail abandons elimination (destination may have additional readers beyond what
	// the scan can prove safe).
	moveScanBail

	// moveScanEliminate concludes that destination is killed and the MOVE can be replaced
	// with opNop.
	moveScanEliminate
)

// moveCandidate carries the recognised MOVE's bank kind, destination register, and source
// register from the entry-condition classifier down to the scanner. Bundling the trio
// reduces argument count on every helper that consumes them.
type moveCandidate struct {
	// kind holds the register bank in which the candidate MOVE operates.
	kind registerKind

	// destination holds the destination register index of the candidate MOVE.
	destination uint8

	// source holds the source register index of the candidate MOVE.
	source uint8
}

// fuseMoveElimination implements a conservative single-use forward- propagation pass for
// redundant same-bank MOVE instructions.
//
// The Go-to-piko compiler emits redundant MOVEs from two sources: tuple-assign lowering
// creates per-RHS temporaries that flow through MOVE into the user-visible destination,
// and comma-ok result routing emits MOVEs from typed scratch registers into the target
// variable slot. Both temp sources are read exactly once and are prime candidates for
// elimination.
//
// The pass recognises body[i] as a same-bank MOVE (tier-1 subOpMove* or opMoveGeneral;
// cross-bank boxes excluded), scans forward from i+1 within the current basic block
// tracking reads and writes of source and destination via operandShapes, and if
// destination is read exactly once before either side is overwritten (and the scan never
// crosses an opcode with shapeFlagFollowsExtension whose ext word may encode unknowable
// register reads) it rewrites that unique reader to read source directly and replaces the
// MOVE with opNop. If destination is written before being read, the MOVE is dead and is
// replaced with opNop without any rewrite. Conservative safety predicates ensure no
// consumer is silently dropped; the snippet and app test suites verify that every elision
// preserves observable behaviour.
//
// Takes body ([]instruction) which specifies the instruction sequence being optimised.
// Takes i (int) which specifies the candidate slot index.
// Takes n (int) which specifies the body length.
// Takes jumpTargets (map[int]bool) which specifies the set of instruction indices that
// are jump destinations (basic-block boundary proxy).
//
// Returns true if a MOVE was eliminated at this slot, false otherwise.
func (cf *CompiledFunction) fuseMoveElimination(
	body []instruction, i, n int,
	jumpTargets map[int]bool,
) bool {
	candidate, eligible := cf.classifyMoveEliminationCandidate(body, i, n, jumpTargets)
	if !eligible {
		return false
	}
	if candidate.destination == candidate.source {
		body[i] = makeInstruction(opNop, 0, 0, 0)
		return true
	}
	if int(candidate.destination) < cf.returnSlotCount(candidate.kind) || int(candidate.source) < cf.returnSlotCount(candidate.kind) {
		return false
	}
	return scanAndEliminateMove(body, i, n, jumpTargets, candidate)
}

// classifyMoveEliminationCandidate checks the entry conditions for MOVE elimination at
// body[i]: the slot must hold a recognised MOVE, must not itself be a jump target, and
// the function must not capture shared cells (opMakeClosure).
//
// Takes body ([]instruction) which is the instruction stream under analysis.
// Takes i (int) which is the candidate slot index.
// Takes n (int) which is the body length used to bound lookahead checks.
// Takes jumpTargets (map[int]bool) which marks slots that are jump destinations.
//
// Returns the parsed move candidate and true when the slot qualifies for elimination; the
// zero value and false otherwise.
func (cf *CompiledFunction) classifyMoveEliminationCandidate(body []instruction, i, n int, jumpTargets map[int]bool) (moveCandidate, bool) {
	if i+1 >= n {
		return moveCandidate{}, false
	}
	kind, ok := moveOpcodeKind(body[i])
	if !ok {
		return moveCandidate{}, false
	}
	if jumpTargets[i] {
		return moveCandidate{}, false
	}
	if cf.mayCreateSharedCells() {
		return moveCandidate{}, false
	}
	destination, source := moveOperands(body[i])
	return moveCandidate{kind: kind, destination: destination, source: source}, true
}

// returnSlotCount counts bank-counted result slots for the given kind.
//
// These slots are read by handleReturn / handleTailCall via resultKinds metadata rather
// than via instruction operand bytes, so the MOVE-elimination pass treats them as live
// for the entire function body and refuses to eliminate MOVEs that read or write them.
//
// Takes kind (registerKind) which selects which result-bank to count.
//
// Returns the number of return slots in that bank.
func (cf *CompiledFunction) returnSlotCount(kind registerKind) int {
	count := 0
	for _, k := range cf.resultKinds {
		if k == kind {
			count++
		}
	}
	return count
}

// scanAndEliminateMove walks the basic block forward from i+1 to rewrite or remove a
// same-bank MOVE.
//
// When a definitive kill of destination is found (an overriding write or a terminator),
// the MOVE at i is replaced with opNop and any single intermediate read of destination is
// rewritten to use source.
//
// Takes body ([]instruction) which is the instruction stream under analysis.
// Takes i (int) which is the index of the MOVE under consideration.
// Takes n (int) which is the body length bounding the forward walk.
// Takes jumpTargets (map[int]bool) which marks slots that are jump destinations.
// Takes candidate (moveCandidate) which carries the MOVE's bank, destination, and source.
//
// Returns true when the MOVE was eliminated; false otherwise.
func scanAndEliminateMove(body []instruction, i, n int, jumpTargets map[int]bool, candidate moveCandidate) bool {
	readPos := -1
	readJ := -1
	limit := min(i+1+moveElimScanWindow, n)
	for j := i + 1; j < limit; j++ {
		decision := classifyMoveScanStep(body, j, jumpTargets, candidate, &readJ, &readPos)
		switch decision {
		case moveScanBail:
			return false
		case moveScanEliminate:
			if readJ != -1 {
				rewriteOperand(body, readJ, readPos, candidate.source)
			}
			body[i] = makeInstruction(opNop, 0, 0, 0)
			return true
		case moveScanContinue:
		}
	}
	return false
}

// classifyMoveScanStep evaluates one instruction at body[j] and returns the decision the
// outer scanner should make. Updates *readJ and *readPos when a single qualifying read of
// destination is observed.
//
// Takes body ([]instruction) which is the instruction stream under analysis.
// Takes j (int) which is the current slot being inspected.
// Takes jumpTargets (map[int]bool) which marks slots that are jump destinations.
// Takes candidate (moveCandidate) which carries the MOVE's bank, destination, and source.
// Takes readJ (*int) which receives the slot index of a single observed read of
// destination.
// Takes readPos (*int) which receives the operand position of that read.
//
// Returns the moveScanDecision selecting the next scanner action.
func classifyMoveScanStep(body []instruction, j int, jumpTargets map[int]bool, candidate moveCandidate, readJ, readPos *int) moveScanDecision {
	if jumpTargets[j] {
		return moveScanBail
	}
	if isControlFlowJump(body[j]) {
		return moveScanBail
	}
	if body[j].op == opDrillTier1 {
		return classifyTier1ScanStep(body[j], j, candidate, readJ, readPos)
	}
	shape := operandShapes[body[j].op]
	if shape.flags&shapeFlagDescribed == 0 {
		return moveScanBail
	}
	if shape.flags&shapeFlagFollowsExtension != 0 {
		return moveScanBail
	}
	readCount, writesSource, writesDestination, pos := classifyOperands(body[j], shape, candidate.kind, candidate.destination, candidate.source)
	if readCount > 1 {
		return moveScanBail
	}
	if readCount == 1 {
		if *readJ != -1 || writesSource {
			return moveScanBail
		}
		*readJ = j
		*readPos = pos
	}
	if writesSource && *readJ == -1 {
		return moveScanBail
	}
	if writesDestination {
		return moveScanEliminate
	}
	if shape.flags&shapeFlagTerminator != 0 {
		return moveScanEliminate
	}
	return moveScanContinue
}

// classifyOperands examines body[j] for reads/writes of destination and source in the
// given register bank.
//
// Cross-bank operands are ignored - only operands typed in `kind` participate in the
// analysis.
//
// Takes instr (instruction) which is the instruction word to classify.
// Takes shape (operandShape) which records each operand's role/flags.
// Takes kind (registerKind) which selects the bank under analysis.
// Takes destination (uint8) which is the destination register of the MOVE.
// Takes source (uint8) which is the source register of the MOVE.
//
// Returns readCount, the number of operand positions that read destination, writesSource
// (true when the instruction writes source), writesDestination (true when the instruction
// writes destination), and readPos (the operand position 0/1/2 of the sole destination
// read when readCount == 1; -1 otherwise).
func classifyOperands(
	instr instruction, shape operandShape,
	kind registerKind, destination, source uint8,
) (readCount int, writesSource, writesDestination bool, readPos int) {
	readPos = -1
	operands := [numInstructionOperands]uint8{instr.a, instr.b, instr.c}
	roles := [numInstructionOperands]operandRole{shape.a, shape.b, shape.c}
	for p := range numInstructionOperands {
		opKind, isReg := kindForRole(roles[p])
		if !isReg || opKind != kind {
			continue
		}
		if shape.reads[p] && operands[p] == destination {
			readCount++
			readPos = p
		}
		if shape.writes[p] {
			switch operands[p] {
			case source:
				writesSource = true
			case destination:
				writesDestination = true
			}
		}
	}
	return readCount, writesSource, writesDestination, readPos
}

// rewriteOperand replaces one of body[j]'s operand bytes (position 0=a, 1=b, 2=c) with
// newReg, leaving the other operands and opcode untouched. Implemented as full
// reconstruction via makeInstruction to match the existing fuser style.
//
// Takes body ([]instruction) which is the in-place instruction stream.
// Takes j (int) which is the index of the instruction to rewrite.
// Takes position (int) which selects which operand byte to overwrite.
// Takes newReg (uint8) which is the replacement register index.
func rewriteOperand(body []instruction, j, position int, newReg uint8) {
	cur := body[j]
	switch position {
	case 0:
		body[j] = makeInstruction(cur.op, newReg, cur.b, cur.c)
	case 1:
		body[j] = makeInstruction(cur.op, cur.a, newReg, cur.c)
	case 2:
		body[j] = makeInstruction(cur.op, cur.a, cur.b, newReg)
	}
}

// moveOpcodeKind classifies an instruction as a same-bank MOVE that the elimination pass
// may consider, returning the register bank the MOVE operates within. Cross-bank moves
// (subOpMoveIntToGeneral, subOpMoveGeneralToInt, etc.) are deliberately excluded because
// they cross allocation boundaries and are not no-ops.
//
// opMoveGeneral is also excluded from this pass: its C operand encodes a snapshot mode
// (moveGeneralModeDynamic / Alias / Snapshot) that may invoke valueCopyForBoundary to
// deep-copy struct or array values. Eliminating such a MOVE and rewriting the downstream
// consumer to read the source directly would lose the snapshot semantics, so downstream
// code that expected the captured value at MOVE time would instead see whatever the live
// source holds at read time.
//
// Takes instr (instruction) which is the candidate slot.
//
// Returns the register kind the MOVE targets and true when the instruction is a
// recognised same-bank MOVE; zero/false otherwise.
func moveOpcodeKind(instr instruction) (registerKind, bool) {
	if instr.op != opDrillTier1 {
		return 0, false
	}
	switch subOpcode(instr.a) {
	case subOpMoveInt:
		return registerInt, true
	case subOpMoveFloat:
		return registerFloat, true
	case subOpMoveString:
		return registerString, true
	case subOpMoveBool:
		return registerBool, true
	case subOpMoveUint:
		return registerUint, true
	case subOpMoveComplex:
		return registerComplex, true
	default:
	}
	return 0, false
}

// isControlFlowJump reports whether instr is a jump that could skip past the candidate
// kill point.
//
// Bailing on these is mandatory: if a conditional jump branches around the
// writesDestination we observed, the branch target reaches an unset destination register,
// breaking the rewrite's invariant that the read can be safely retargeted to source.
//
// Takes instr (instruction) which is the slot under inspection.
//
// Returns true when instr is a recognised control-flow jump; false otherwise.
func isControlFlowJump(instr instruction) bool {
	switch instr.op {
	case opJumpIfTrue, opJumpIfFalse:
		return true
	default:
	}
	if instr.op == opDrillTier1 && subOpcode(instr.a) == subOpJump {
		return true
	}
	return false
}

// classifyTier1ScanStep handles the opDrillTier1 case during MOVE-elimination scanning.
//
// The generic classifyOperands cannot see register accesses through tier-1 sub-ops
// because the shared operand shape uses roleRegDynamic; kindForRole returns false for
// that role, hiding the read/write from the analysis. This helper special-cases the
// tier-1 same-bank MOVE encoding (`opDrillTier1, subOpMove*, b=destination, c=source`) so
// reads of and writes to the candidate's destination and source are observed. Tier-1
// sub-ops other than the recognised same-bank MOVEs (arithmetic, jumps, loads, etc.)
// cannot be safely classified here and must bail.
//
// Takes instr (instruction) which is the tier-1 instruction at body[j].
// Takes j (int) which is the instruction index used to record readJ.
// Takes candidate (moveCandidate) which carries the original MOVE's kind, destination,
// and source.
// Takes readJ, readPos (*int) which are updated when a single qualifying read of
// destination is observed.
//
// Returns the decision for the outer scanner.
func classifyTier1ScanStep(instr instruction, j int, candidate moveCandidate, readJ, readPos *int) moveScanDecision {
	subKind, isMove := moveOpcodeKind(instr)
	if !isMove {
		return moveScanBail
	}
	if subKind != candidate.kind {
		return moveScanContinue
	}
	destination, source := moveOperands(instr)
	if source == candidate.destination {
		if *readJ != -1 {
			return moveScanBail
		}
		*readJ = j
		*readPos = tier1ScanOperandPositionC
	}
	if destination == candidate.destination {
		return moveScanEliminate
	}
	if destination == candidate.source && *readJ == -1 {
		return moveScanBail
	}
	return moveScanContinue
}

// moveOperands extracts destination and source register indices from a same-bank tier-1
// MOVE.
//
// The compiler's emitMoveTyped elides self-MOVEs so callers can assume destination !=
// source. Tier-1 subOpMove* encoding: {opDrillTier1, subOpMoveX, b=destination,
// c=source}.
//
// Takes instr (instruction) which is the candidate slot (callers must verify with
// moveOpcodeKind first).
//
// Returns the destination and source register indices.
func moveOperands(instr instruction) (destination, source uint8) {
	return instr.b, instr.c
}
