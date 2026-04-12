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
	"context"
	"fmt"
)

const (
	// maxCseFieldReadScanWindow caps the forward-scan distance from a candidate first read
	// to its potential redundant successor. Most real-world redundant reads sit within a
	// handful of instructions inside the same loop body or conditional arm; widening the
	// window only increases pass cost without finding more matches.
	maxCseFieldReadScanWindow = 16

	// cseAliasSetCapacity caps the alias set size. Beyond this the CSE pass stops adding new
	// aliases (but keeps the existing entries), trading completeness for bounded scan cost.
	cseAliasSetCapacity = 4
)

// elideRedundantStructFieldRead replaces a redundant struct-field reload with a move.
//
// The pass detects a struct-field read that re-loads the same (receiver register, layout
// index) into a different destination register and rewrites it as a same-bank move from
// the prior destination, provided every intervening instruction is known not to mutate
// the receiver, the prior destination, or any struct field reachable from the receiver.
// The second read becomes a MOVE_GENERAL in place of a slice-header snapshot plus
// reflect.Value materialisation.
//
// Coverage includes tier-0 general bank reads (opGetStructFieldGeneral,
// opGetStructFieldRawPointerT0), tier-0 scalar reads (opGetStructFieldIntT0,
// opGetStructFieldUint, opGetStructFieldFloat, opGetStructFieldBool), and tier-1 reads
// with an EXT word (subOpGetStructFieldInt, subOpGetStructFieldUint,
// subOpGetStructFieldFloat, subOpGetStructFieldBool, subOpGetStructFieldString).
//
// The scan terminates at jump targets (an arrival from elsewhere may have left the prior
// destination holding a different value), at any write to the receiver register, prior
// destination register, or any register we cannot prove distinct from those two, and at
// any instruction that may mutate a struct field, the heap, or invoke user code. The
// conservative block list covers calls (opCall, opTailCall, opCallMethod, opCallNative,
// opCallIIFE), explicit struct-field writes (opSetField, the opSetStructField* family,
// opSwapStructFieldsGeneralT0), the tier-1 struct-field write sub-ops, opMapSet,
// opIndexSet, opAddr, and the closure / upvalue mutators.
//
// Takes ctx (context.Context) which cancels the rewrite walk.
// Takes body ([]instruction) which is the function's instruction stream rewritten in
// place.
//
// Returns error when context cancellation fires mid-rewrite.
func (cf *CompiledFunction) elideRedundantStructFieldRead(ctx context.Context, body []instruction) error {
	if !peepholeCseEnabled {
		return nil
	}
	if len(body) < 2 {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("elideRedundantStructFieldRead cancelled: %w", err)
	}
	jumpTargets := buildAllJumpTargets(body)
	for i := range body {
		if i&optimisationLoopCheckMask == 0 {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("elideRedundantStructFieldRead cancelled: %w", err)
			}
		}
		if isTier0StructFieldRead(body[i].op) {
			cf.elideTier0Read(body, i, jumpTargets)
			continue
		}
		if isTier1StructFieldReadAt(body, i) {
			cf.elideTier1Read(body, i, jumpTargets)
			continue
		}
		if isGeneralBankFieldWrite(body[i]) {
			cf.elidePostSetRead(body, i, jumpTargets)
		}
	}
	return nil
}

// elideTier0Read rewrites a redundant tier-0 struct-field read at firstIdx as a MOVE from
// the prior live alias, recording the rewrite.
//
// Takes body ([]instruction) which is the function's instruction stream rewritten in
// place.
// Takes firstIdx (int) which is the index of the candidate first read in body.
// Takes jumpTargets (map[int]bool) which is the set of indices that are branch
// destinations.
func (cf *CompiledFunction) elideTier0Read(body []instruction, firstIdx int, jumpTargets map[int]bool) {
	matchIdx, srcReg, found := findRedundantTier0StructFieldRead(cf, body, firstIdx, jumpTargets)
	if !found {
		return
	}

	if tier0ReadDestinationFedToFieldWrite(body, matchIdx, jumpTargets) {
		return
	}
	body[matchIdx] = emitMoveForTier0ReadOp(body[firstIdx].op, body[matchIdx].a, srcReg)
	cf.recordPeepholeRewrite(matchIdx, peepholeRewriteCseTier0, firstIdx)
}

// elideTier1Read rewrites a redundant tier-1 struct-field read at firstIdx as a same-bank
// tier-1 MOVE, nopping the extension word.
//
// Takes body ([]instruction) which is the function's instruction stream rewritten in
// place.
// Takes firstIdx (int) which is the umbrella-word index of the candidate first read in
// body.
// Takes jumpTargets (map[int]bool) which is the set of indices that are branch
// destinations.
func (cf *CompiledFunction) elideTier1Read(body []instruction, firstIdx int, jumpTargets map[int]bool) {
	matchIdx, srcReg, found := findRedundantTier1StructFieldRead(cf, body, firstIdx, jumpTargets)
	if !found {
		return
	}

	if tier0ReadDestinationFedToFieldWrite(body, matchIdx, jumpTargets) {
		return
	}
	rewriteTier1ReadAsMove(body, matchIdx, srcReg)
	cf.recordPeepholeRewrite(matchIdx, peepholeRewriteCseTier1Umbrella, firstIdx)
	cf.recordPeepholeRewrite(matchIdx+1, peepholeRewriteCseTier1Ext, firstIdx)
}

// elidePostSetRead rewrites a struct-field read following a matching general-bank field
// write as a MOVE_GENERAL from the setter's source.
//
// Takes body ([]instruction) which is the function's instruction stream rewritten in
// place.
// Takes firstIdx (int) which is the index of the field-write instruction in body.
// Takes jumpTargets (map[int]bool) which is the set of indices that are branch
// destinations.
func (cf *CompiledFunction) elidePostSetRead(body []instruction, firstIdx int, jumpTargets map[int]bool) {
	matchIdx, srcReg, found := findGetAfterGeneralFieldSet(cf, body, firstIdx, jumpTargets)
	if !found {
		return
	}
	body[matchIdx] = makeInstruction(opMoveGeneral, body[matchIdx].a, srcReg, 0)
	cf.recordPeepholeRewrite(matchIdx, peepholeRewriteCsePostSet, firstIdx)
}

// cseAliasSet tracks registers known to hold the cached struct-field value.
//
// The first element is the original read destination; later entries are added when a
// same-bank move copies the value into a new register. The set has a small fixed capacity
// because real-world move chains rarely exceed three or four hops.
type cseAliasSet struct {
	// entries holds the alias register indices; valid up to length.
	entries [cseAliasSetCapacity]uint8

	// length is the number of valid entries currently in the set.
	length uint8
}

// add inserts reg into the alias set when capacity remains and reg is not already
// present. Silently no-ops when the set is full.
//
// Takes reg (uint8) which is the register index to add to the set.
func (set *cseAliasSet) add(reg uint8) {
	if set.contains(reg) {
		return
	}
	if int(set.length) >= len(set.entries) {
		return
	}
	set.entries[set.length] = reg
	set.length++
}

// contains reports whether reg is currently in the alias set.
//
// Takes reg (uint8) which is the register index to test for membership.
//
// Returns true when reg is present, false otherwise.
func (set *cseAliasSet) contains(reg uint8) bool {
	for i := uint8(0); i < set.length; i++ {
		if set.entries[i] == reg {
			return true
		}
	}
	return false
}

// empty reports whether all aliases have been killed.
//
// Returns true when the set contains no live aliases.
func (set *cseAliasSet) empty() bool {
	return set.length == 0
}

// priorDest returns the original first-read destination.
//
// Useful for continuing to detect receiver-register writes regardless of which alias the
// rewrite ultimately selects.
//
// Returns the original read's destination register, or 0 when the set is empty.
func (set *cseAliasSet) priorDest() uint8 {
	if set.length == 0 {
		return 0
	}
	return set.entries[0]
}

// preferredSource returns the smallest-index alias from the set. Picking the smallest
// register makes the downstream MOVE-elimination pass more likely to coalesce the
// inserted move into existing register state.
//
// Returns the smallest-index register in the set, or 0 when the set is empty.
func (set *cseAliasSet) preferredSource() uint8 {
	if set.length == 0 {
		return 0
	}
	best := set.entries[0]
	for i := uint8(1); i < set.length; i++ {
		if set.entries[i] < best {
			best = set.entries[i]
		}
	}
	return best
}

// dropRegistersWrittenBy removes any alias whose register the instruction writes in the
// matching bank.
//
// Takes inst (instruction) whose writes are checked against the set.
// Takes bank (operandRole) which is the destination bank role to test for; roleNone skips
// the scan.
func (set *cseAliasSet) dropRegistersWrittenBy(inst instruction, bank operandRole) {
	if bank == roleNone {
		return
	}
	write := 0
	for read := uint8(0); read < set.length; read++ {
		reg := set.entries[read]
		if instructionWritesRegisterInBank(inst, bank, reg) {
			continue
		}
		set.entries[write] = reg
		write++
	}
	set.length = uint8(write)
}

// tier0ReadDestinationFedToFieldWrite reports whether the destination register of the
// tier-0 struct-field read at matchIdx is read by any later struct-field-write opcode as
// its receiver before that destination is overwritten or the scan window expires.
//
// Used by elideTier0Read to refuse a MOVE-replacement that would sever addressability
// when the second read's result is the target of a subsequent write. The original
// GET_FIELD returns an addressable reflect.Value that aliases the parent struct's memory;
// the MOVE goes through valueCopyForBoundary which deep-copies Struct/Array kinds,
// breaking the alias. Without this guard, a sequence like `c.value.A = 1; c.value.B = 2`
// would compile into two GET_FIELDs of c.value (the second one elided via MOVE) and two
// SET_FIELDs whose receivers point at independent struct copies - the second SET would be
// lost.
//
// Takes body ([]instruction) which is the function's instruction stream.
// Takes matchIdx (int) which is the index of the candidate redundant read.
// Takes jumpTargets (map[int]bool) which is the set of indices that are branch
// destinations.
//
// Returns true when matchIdx's destination is observed as a struct-field-write receiver
// before being overwritten; false otherwise.
func tier0ReadDestinationFedToFieldWrite(body []instruction, matchIdx int, jumpTargets map[int]bool) bool {
	if matchIdx < 0 || matchIdx >= len(body) {
		return false
	}
	destReg := body[matchIdx].a
	limit := min(matchIdx+maxCseFieldReadScanWindow, len(body))
	for j := matchIdx + 1; j < limit; j++ {
		if jumpTargets[j] {
			return false
		}
		inst := body[j]
		if receiver, ok := structFieldWriteReceiver(inst); ok && receiver == destReg {
			return true
		}
		if instructionWritesRegisterInBank(inst, roleRegGeneral, destReg) {
			return false
		}
	}
	return false
}

// isGeneralBankFieldWrite reports whether inst writes a general-bank struct field whose
// post-write value lives in a register we can reference from a MOVE instead of a
// follow-up reflect-walk read.
//
// Limited to opSetStructFieldGeneral: operand A is the general-bank receiver, operand B
// is the source general register carrying the value being stored, operand C is the
// layoutTable index. Scalar setters (opSetStructFieldIntT0 et al.) may narrow the source
// value to the field's storage width and so a follow-up GET would re-widen with
// potentially different upper bits; they are excluded from this fast path because a
// width-preservation analysis is required to admit them.
//
// Takes inst (instruction) which is the candidate instruction to classify.
//
// Returns true when inst is opSetStructFieldGeneral, false otherwise.
func isGeneralBankFieldWrite(inst instruction) bool {
	return inst.op == opSetStructFieldGeneral
}

// findGetAfterGeneralFieldSet locates a redundant read after a matching write.
//
// When a general-bank struct-field write is followed within the scan window by a matching
// general-bank read of the same (receiver, layout) pair with no intervening invalidator,
// the read can be rewritten as a MOVE_GENERAL from the setter's source register (operand
// B), saving the reflect-walk that the read handler would otherwise perform.
//
// Move propagation is reused via cseAliasSet: the setter's value register is the initial
// alias, and subsequent same-bank moves extend the set so the matching read can rewrite
// against any register that still holds the stored value.
//
// Takes cf (*CompiledFunction) which is the function being analysed, consulted for alias
// info on calls.
// Takes body ([]instruction) which is the function's instruction stream.
// Takes firstIdx (int) which is the index of the candidate general-bank field write.
// Takes jumpTargets (map[int]bool) which is the set of indices that are branch
// destinations.
//
// Returns the read instruction index, the source register for the rewrite, and true on
// success; (0, 0, false) otherwise.
func findGetAfterGeneralFieldSet(cf *CompiledFunction, body []instruction, firstIdx int, jumpTargets map[int]bool) (int, uint8, bool) {
	first := body[firstIdx]
	receiverReg := first.a
	layoutIdx := first.c
	valueReg := first.b
	destBank := roleRegGeneral
	aliases := newCseAliasSet(valueReg)
	limit := min(firstIdx+maxCseFieldReadScanWindow, len(body))
	for j := firstIdx + 1; j < limit; j++ {
		if jumpTargets[j] {
			return 0, 0, false
		}
		inst := body[j]
		if isGeneralBankMatchingRead(inst.op) && inst.b == receiverReg && inst.c == layoutIdx {
			return j, aliases.preferredSource(), true
		}
		if newAlias, srcReg, ok := detectAliasingMove(inst, destBank); ok && aliases.contains(srcReg) {
			aliases.add(newAlias)
			continue
		}
		if structFieldReadScanBlocksAt(cf, j, inst, receiverReg, aliases.priorDest(), destBank) {
			return 0, 0, false
		}
		aliases.dropRegistersWrittenBy(inst, destBank)
		if aliases.empty() {
			return 0, 0, false
		}
	}
	return 0, 0, false
}

// isGeneralBankMatchingRead reports whether op reads the value last written by
// opSetStructFieldGeneral.
//
// opGetStructFieldGeneral matches directly; the cycle-broken raw-pointer specialisation
// also reads the eface header that opSetStructFieldGeneral writes, so it matches too.
//
// Takes op (opcode) which is the candidate read opcode to classify.
//
// Returns true for opGetStructFieldGeneral or opGetStructFieldRawPointerT0.
func isGeneralBankMatchingRead(op opcode) bool {
	return op == opGetStructFieldGeneral || op == opGetStructFieldRawPointerT0
}

// isTier0StructFieldRead reports whether op is one of the single-word struct-field read
// opcodes the CSE pass recognises.
//
// Covers two encoding groups, both consisting of one instruction word with operands
// `(dest=A, receiver=B, layoutOrFieldIndex=C)`. Layout-driven tier-0 reads
// (opGetStructFieldXxxT0 family) carry C as a `structLayoutTable` index in [0, 255],
// resolved via the pre-baked layout entry at runtime. Generic reads (opGetField,
// opGetFieldInt) carry C as the direct reflect.Type field index; the dispatch handler
// walks via reflect.Field at runtime with no layout table lookup.
//
// Both groups share an identical CSE key: matching `(op, B, C)` across two instances
// proves the same field is being read from the same receiver. The rewrite emits the
// same-bank move appropriate to the destination bank (general for opGetField / general /
// RawPointerT0; scalar tier-1 move otherwise).
//
// Takes op (opcode) which is the candidate opcode to classify.
//
// Returns true for any recognised single-word struct-field read opcode.
func isTier0StructFieldRead(op opcode) bool {
	switch op {
	case opGetStructFieldGeneral, opGetStructFieldRawPointerT0,
		opGetStructFieldIntT0, opGetStructFieldUint,
		opGetStructFieldFloat, opGetStructFieldBool,
		opGetField, opGetFieldInt:
		return true
	default:
	}
	return false
}

// tier0FieldReadDestRole returns the bank role of a tier-0 struct-field read's
// destination register. Used by the scan to detect writes to the prior destination in the
// matching bank.
//
// Takes op (opcode) which is a tier-0 struct-field read opcode.
//
// Returns the destination bank role; roleNone for unrecognised opcodes.
func tier0FieldReadDestRole(op opcode) operandRole {
	switch op {
	case opGetStructFieldIntT0, opGetFieldInt:
		return roleRegInt
	case opGetStructFieldUint:
		return roleRegUint
	case opGetStructFieldFloat:
		return roleRegFloat
	case opGetStructFieldBool:
		return roleRegBool
	case opGetStructFieldGeneral, opGetStructFieldRawPointerT0, opGetField:
		return roleRegGeneral
	default:
	}
	return roleNone
}

// emitMoveForTier0ReadOp returns the instruction that copies the prior destination
// register of a tier-0 struct-field read into a fresh destination, preserving the read
// op's destination bank.
//
// For general-bank reads this is `opMoveGeneral destination, source, 0` (a tier-0 move).
// For scalar reads this is `opDrillTier1, subOpMoveX, destination, source`, a tier-1
// same-bank move whose handler copies one scalar register to another without entering the
// general bank.
//
// Takes readOp (opcode) which is the original read opcode at the matched site.
// Takes destination (uint8) which is the destination register of the matched read
// (preserved on rewrite).
// Takes source (uint8) which is the destination register of the earlier matching read
// (becomes the move source).
//
// Returns the rewritten instruction. The fallback opNop is unreachable for any callers
// obeying the isTier0StructFieldRead gate.
func emitMoveForTier0ReadOp(readOp opcode, destination, source uint8) instruction {
	switch readOp {
	case opGetStructFieldGeneral, opGetStructFieldRawPointerT0, opGetField:
		return makeInstruction(opMoveGeneral, destination, source, 0)
	case opGetStructFieldIntT0, opGetFieldInt:
		return makeInstruction(opDrillTier1, uint8(subOpMoveInt), destination, source)
	case opGetStructFieldUint:
		return makeInstruction(opDrillTier1, uint8(subOpMoveUint), destination, source)
	case opGetStructFieldFloat:
		return makeInstruction(opDrillTier1, uint8(subOpMoveFloat), destination, source)
	case opGetStructFieldBool:
		return makeInstruction(opDrillTier1, uint8(subOpMoveBool), destination, source)
	default:
	}
	return makeInstruction(opNop, 0, 0, 0)
}

// findRedundantTier0StructFieldRead scans forward from a tier-0 read at firstIdx looking
// for a matching read with identical opcode, receiver register, and layout index. Tracks
// alias registers introduced by same-bank moves so the rewrite can reference any register
// that still holds the cached value.
//
// Takes cf (*CompiledFunction) which is the function being analysed, consulted for alias
// info on calls.
// Takes body ([]instruction) which is the function's instruction stream.
// Takes firstIdx (int) which is the index of the candidate first tier-0 read in body.
// Takes jumpTargets (map[int]bool) which is the set of indices that are branch
// destinations.
//
// Returns the matching instruction index, the source register for the rewrite (the
// smallest-index alias still live), and true when the scan terminates at a same-shape
// read with no intervening invalidator.
// Returns (0, 0, false) when the scan encounters any invalidator before finding a match
// or reaches the scan window limit.
func findRedundantTier0StructFieldRead(cf *CompiledFunction, body []instruction, firstIdx int, jumpTargets map[int]bool) (int, uint8, bool) {
	first := body[firstIdx]
	receiverReg := first.b
	layoutIdx := first.c
	destBank := tier0FieldReadDestRole(first.op)
	aliases := newCseAliasSet(first.a)
	limit := min(firstIdx+maxCseFieldReadScanWindow, len(body))
	for j := firstIdx + 1; j < limit; j++ {
		if jumpTargets[j] {
			return 0, 0, false
		}
		inst := body[j]
		if inst.op == first.op && inst.b == receiverReg && inst.c == layoutIdx {
			return j, aliases.preferredSource(), true
		}
		if newAlias, srcReg, ok := detectAliasingMove(inst, destBank); ok && aliases.contains(srcReg) {
			aliases.add(newAlias)
			continue
		}
		if structFieldReadScanBlocksAt(cf, j, inst, receiverReg, aliases.priorDest(), destBank) {
			return 0, 0, false
		}
		aliases.dropRegistersWrittenBy(inst, destBank)
		if aliases.empty() {
			return 0, 0, false
		}
	}
	return 0, 0, false
}

// newCseAliasSet seeds the set with the original read destination as the only live alias.
//
// Takes priorDest (uint8) which is the original read's destination register index.
//
// Returns a cseAliasSet containing priorDest as its sole entry.
func newCseAliasSet(priorDest uint8) cseAliasSet {
	var set cseAliasSet
	set.entries[0] = priorDest
	set.length = 1
	return set
}

// detectAliasingMove reports whether inst is a same-bank move that would copy the value
// in srcReg into a new register (newAlias) of the same bank.
//
// For general-bank scans this recognises the tier-0 `opMoveGeneral` regardless of its
// snapshot-mode operand: snapshot and alias modes both leave the destination register
// holding the same observable value as the source until either is mutated, and the CSE
// scan already terminates on struct mutators.
//
// For scalar-bank scans this recognises the tier-1 `opDrillTier1 + subOpMoveX` form,
// matching only the move whose bank equals the read's destination bank.
//
// Takes inst (instruction) which is the candidate instruction.
// Takes bank (operandRole) which is the destination bank of the active read being
// tracked.
//
// Returns (newAlias=B, srcReg=C, true) when inst is a same-bank move; (0, 0, false)
// otherwise.
func detectAliasingMove(inst instruction, bank operandRole) (newAlias, srcReg uint8, ok bool) {
	if inst.op == opMoveGeneral && bank == roleRegGeneral {
		return inst.a, inst.b, true
	}
	if inst.op != opDrillTier1 {
		return 0, 0, false
	}
	if subOpMatchesBankMove(subOpcode(inst.a), bank) {
		return inst.b, inst.c, true
	}
	return 0, 0, false
}

// subOpMatchesBankMove reports whether sub is a same-bank tier-1 move targeting bank.
//
// Takes sub (subOpcode) which is the candidate tier-1 sub-opcode.
// Takes bank (operandRole) which is the bank role the caller wants the move to target.
//
// Returns true when sub is the same-bank move sub-op for bank.
func subOpMatchesBankMove(sub subOpcode, bank operandRole) bool {
	switch sub {
	case subOpMoveInt:
		return bank == roleRegInt
	case subOpMoveUint:
		return bank == roleRegUint
	case subOpMoveFloat:
		return bank == roleRegFloat
	case subOpMoveBool:
		return bank == roleRegBool
	case subOpMoveString:
		return bank == roleRegString
	case subOpMoveComplex:
		return bank == roleRegComplex
	default:
	}
	return false
}

// isTier1StructFieldReadAt reports whether body[i] is the umbrella word of a tier-1
// struct-field read with a following opExt layout extension. The CSE pass only considers
// tier-1 reads when both words are present and the next word is an opExt.
//
// Takes body ([]instruction) which is the function's instruction stream.
// Takes i (int) which is the candidate umbrella-word index.
//
// Returns true when body[i] and body[i+1] form a tier-1 struct-field read.
func isTier1StructFieldReadAt(body []instruction, i int) bool {
	if i+1 >= len(body) {
		return false
	}
	if body[i].op != opDrillTier1 {
		return false
	}
	if body[i+1].op != opExt {
		return false
	}
	return isTier1StructFieldReadSubOp(subOpcode(body[i].a))
}

// isTier1StructFieldReadSubOp reports whether a tier-1 sub-op is one of the
// layout-table-driven struct-field reads.
//
// Takes sub (subOpcode) which is the candidate tier-1 sub-opcode.
//
// Returns true for any subOpGetStructFieldX read sub-op.
func isTier1StructFieldReadSubOp(sub subOpcode) bool {
	switch sub {
	case subOpGetStructFieldInt, subOpGetStructFieldUint,
		subOpGetStructFieldFloat, subOpGetStructFieldBool,
		subOpGetStructFieldString:
		return true
	default:
	}
	return false
}

// tier1FieldReadDestRole returns the bank role of a tier-1 struct-field read sub-op's
// destination register.
//
// Takes sub (subOpcode) which is a tier-1 struct-field read sub-opcode.
//
// Returns the destination bank role; roleNone for unrecognised sub-ops.
func tier1FieldReadDestRole(sub subOpcode) operandRole {
	switch sub {
	case subOpGetStructFieldInt:
		return roleRegInt
	case subOpGetStructFieldUint:
		return roleRegUint
	case subOpGetStructFieldFloat:
		return roleRegFloat
	case subOpGetStructFieldBool:
		return roleRegBool
	case subOpGetStructFieldString:
		return roleRegString
	default:
	}
	return roleNone
}

// tier1ReadToMoveSubOp maps a tier-1 struct-field read sub-op to the matching same-bank
// move sub-op used as the CSE rewrite target.
//
// Takes sub (subOpcode) which is a tier-1 struct-field read sub-opcode.
//
// Returns the matching move sub-op and true on success; (0, false) for unrecognised
// sub-ops.
func tier1ReadToMoveSubOp(sub subOpcode) (subOpcode, bool) {
	switch sub {
	case subOpGetStructFieldInt:
		return subOpMoveInt, true
	case subOpGetStructFieldUint:
		return subOpMoveUint, true
	case subOpGetStructFieldFloat:
		return subOpMoveFloat, true
	case subOpGetStructFieldBool:
		return subOpMoveBool, true
	case subOpGetStructFieldString:
		return subOpMoveString, true
	default:
	}
	return 0, false
}

// findRedundantTier1StructFieldRead scans forward from a tier-1 read at firstIdx looking
// for a matching tier-1 read with identical sub-opcode, receiver register, and 16-bit EXT
// layout index. Tracks alias registers introduced by same-bank moves so the rewrite can
// reference any register that still holds the cached value.
//
// The tier-1 read spans two instruction words: the umbrella opDrillTier1 followed by
// opExt carrying the layout index as (low byte, high byte). Both words must match for the
// rewrite to fire.
//
// Takes cf (*CompiledFunction) which is the function being analysed, consulted for alias
// info on calls.
// Takes body ([]instruction) which is the function's instruction stream.
// Takes firstIdx (int) which is the umbrella-word index of the candidate first tier-1
// read.
// Takes jumpTargets (map[int]bool) which is the set of indices that are branch
// destinations.
//
// Returns the umbrella-word index of the redundant read, the source register for the
// rewrite (the smallest-index alias still live), and true on success.
// Returns (0, 0, false) otherwise.
func findRedundantTier1StructFieldRead(cf *CompiledFunction, body []instruction, firstIdx int, jumpTargets map[int]bool) (int, uint8, bool) {
	first := body[firstIdx]
	firstExt := body[firstIdx+1]
	sub := subOpcode(first.a)
	receiverReg := first.c
	destBank := tier1FieldReadDestRole(sub)
	aliases := newCseAliasSet(first.b)
	layoutLow := firstExt.a
	layoutHigh := firstExt.b
	limit := min(firstIdx+maxCseFieldReadScanWindow, len(body))
	for j := firstIdx + 2; j < limit; j++ {
		if jumpTargets[j] {
			return 0, 0, false
		}
		if matchesTier1Read(body, j, sub, receiverReg, layoutLow, layoutHigh) {
			return j, aliases.preferredSource(), true
		}
		inst := body[j]
		if newAlias, srcReg, ok := detectAliasingMove(inst, destBank); ok && aliases.contains(srcReg) {
			aliases.add(newAlias)
			continue
		}
		if structFieldReadScanBlocksAt(cf, j, inst, receiverReg, aliases.priorDest(), destBank) {
			return 0, 0, false
		}
		aliases.dropRegistersWrittenBy(inst, destBank)
		if aliases.empty() {
			return 0, 0, false
		}
	}
	return 0, 0, false
}

// matchesTier1Read reports whether the tier-1 read at PC j has the same sub-opcode,
// receiver register, and extension-word layout (split across low/high bytes) as the
// cached read being tracked.
//
// Takes body ([]instruction) which is the function's instruction stream.
// Takes j (int) which is the umbrella-word index of the candidate read.
// Takes sub (subOpcode) which is the cached read's sub-opcode to match against.
// Takes receiverReg (uint8) which is the cached read's receiver register to match
// against.
// Takes layoutLow (uint8) which is the low byte of the cached read's layout index.
// Takes layoutHigh (uint8) which is the high byte of the cached read's layout index.
//
// Returns true when body[j..j+1] match the cached read on all keys.
func matchesTier1Read(body []instruction, j int, sub subOpcode, receiverReg, layoutLow, layoutHigh uint8) bool {
	if !isTier1StructFieldReadAt(body, j) {
		return false
	}
	inst := body[j]
	if subOpcode(inst.a) != sub || inst.c != receiverReg {
		return false
	}
	ext := body[j+1]
	return ext.a == layoutLow && ext.b == layoutHigh
}

// rewriteTier1ReadAsMove replaces the tier-1 read at matchIdx with the corresponding
// same-bank tier-1 move, and nops out the trailing opExt word that carries the read's
// layout index.
//
// Takes body ([]instruction) which is the function's instruction stream.
// Takes matchIdx (int) which is the umbrella-word index of the matched read.
// Takes priorReadDest (uint8) which is the destination register of the earlier matching
// read (becomes the move's source).
func rewriteTier1ReadAsMove(body []instruction, matchIdx int, priorReadDest uint8) {
	matchedReadDest := body[matchIdx].b
	moveSub, ok := tier1ReadToMoveSubOp(subOpcode(body[matchIdx].a))
	if !ok {
		return
	}
	body[matchIdx] = makeInstruction(opDrillTier1, uint8(moveSub), matchedReadDest, priorReadDest)
	body[matchIdx+1] = makeInstruction(opNop, 0, 0, 0)
}

// structFieldReadScanBlocksAt is the alias-aware scan-blocker check.
//
// When pc is non-negative and the function has populated alias info, struct-field-write
// instructions whose receiver provably does not alias the cached-read's receiver no
// longer invalidate the scan. Calls and undescribed opcodes remain blanket blockers
// because those routes can mutate any heap state.
//
// cf is consulted at call instructions: a call to a callee classified as heapPureCallee
// by runHeapPurityAnalysis does not bail the scan. nil cf falls back to the
// call-blanket-blocker behaviour.
//
// Takes cf (*CompiledFunction) which is the function being analysed; nil disables alias
// refinement.
// Takes pc (int) which is the program counter of inst within cf.body; pass -1 when no PC
// context is available (the fallback path skips the alias refinement).
// Takes inst (instruction) which is the candidate instruction to classify.
// Takes receiverReg (uint8) which is the cached read's receiver register.
// Takes priorDest (uint8) which is the cached read's prior destination register.
// Takes destBank (operandRole) which is the cached read's destination bank role.
//
// Returns true when inst must terminate the scan, false when it is safe to skip.
func structFieldReadScanBlocksAt(cf *CompiledFunction, pc int, inst instruction, receiverReg, priorDest uint8, destBank operandRole) bool {
	if structFieldWriteWithKnownReceiverDistinct(cf, pc, inst, receiverReg) {
		return false
	}
	if invalidatesCachedFieldReads(cf, inst) {
		return true
	}
	if !instructionShapeAllowsCseScan(inst) {
		return true
	}
	if instructionWritesRegisterInBank(inst, roleRegGeneral, receiverReg) {
		return true
	}
	if destBank != roleNone && instructionWritesRegisterInBank(inst, destBank, priorDest) {
		return true
	}
	return false
}

// structFieldWriteWithKnownReceiverDistinct reports whether inst is a struct-field write
// whose receiver register provably does not alias the cached-read's receiver per the
// function's alias info.
//
// The check uses the alias environment from pc-1, the state going INTO the candidate
// instruction, before its effects apply. When pc == 0 the entry environment applies; the
// seedEntryEnvironment initialiser gives parameter registers distinct classes.
//
// Takes cf (*CompiledFunction) which is the function being analysed; nil returns false.
// Takes pc (int) which is the program counter of inst within cf.body.
// Takes inst (instruction) which is the candidate instruction to classify.
// Takes cachedReceiverReg (uint8) which is the cached read's receiver register.
//
// Returns true only when inst is one of the recognised struct-field-write opcodes, cf
// carries populated alias info, pc is a valid program counter, and the alias info reports
// the write's receiver and the cached receiver are definitely-not-aliasing at this
// program point.
func structFieldWriteWithKnownReceiverDistinct(cf *CompiledFunction, pc int, inst instruction, cachedReceiverReg uint8) bool {
	if cf == nil || cf.aliasInfo == nil {
		return false
	}
	if pc <= 0 || pc >= len(cf.body) {
		return false
	}
	writeReceiverReg, ok := structFieldWriteReceiver(inst)
	if !ok {
		return false
	}
	queryPC := pc - 1
	return !cf.aliasInfo.mayAlias(queryPC, writeReceiverReg, cachedReceiverReg)
}

// structFieldWriteReceiver returns the general-bank receiver register of a
// struct-field-write opcode, or (0, false) when inst is not a recognised struct-field
// write.
//
// Takes inst (instruction) which is the candidate instruction to classify.
//
// Returns the receiver register and true on success; (0, false) otherwise.
//
// Recognised forms place the receiver in operand A for opSetStructFieldGeneral /
// opSetStructFieldIntT0 / opSetStructFieldUint / opSetStructFieldFloat /
// opSetStructFieldBool / opSwapStructFieldsGeneralT0 / opSetField, and in operand B for
// opDrillTier1 + (subOpSetStructFieldInt / Uint / Float / Bool / String /
// subOpIncStructFieldInt / Uint / subOpDecStructFieldInt / Uint) where operand A holds
// the sub-op.
func structFieldWriteReceiver(inst instruction) (uint8, bool) {
	switch inst.op {
	case opSetStructFieldGeneral, opSetStructFieldIntT0, opSetStructFieldUint,
		opSetStructFieldFloat, opSetStructFieldBool,
		opSwapStructFieldsGeneralT0, opSetField:
		return inst.a, true
	case opDrillTier1:
		switch subOpcode(inst.a) {
		case subOpSetStructFieldInt, subOpSetStructFieldUint,
			subOpSetStructFieldFloat, subOpSetStructFieldBool,
			subOpSetStructFieldString,
			subOpIncStructFieldInt, subOpDecStructFieldInt,
			subOpIncStructFieldUint, subOpDecStructFieldUint:
			return inst.b, true
		default:
		}
	default:
	}
	return 0, false
}

// invalidatesCachedFieldReads reports whether inst, interpreted in the context of caller
// cf, may mutate any heap-resident state that a cached struct-field read would observe.
//
// Direct mutators (struct-field writes, map sets, etc.) always return true. Call
// instructions defer to the static callee's heapMutationClass when one is resolvable;
// when the callee is unresolvable or unclassified, the call conservatively returns true.
//
// Falling back from cf-aware to call-blanket-blocker behaviour when cf is nil keeps the
// CSE pass usable from unit tests that build instruction streams directly.
//
// Takes cf (*CompiledFunction) which is the function being analysed; nil triggers the
// conservative call-blanket-blocker fallback.
// Takes inst (instruction) which is the candidate instruction to classify.
//
// Returns true when inst may invalidate a cached field read, false otherwise.
func invalidatesCachedFieldReads(cf *CompiledFunction, inst instruction) bool {
	if instructionDirectlyMutatesHeap(inst) {
		return true
	}
	if !isCallOpcode(inst.op) {
		return false
	}
	if cf == nil {
		return true
	}
	return callInvalidatesPurity(cf, inst)
}

// instructionShapeAllowsCseScan reports whether the operand-shape entry for inst's opcode
// is populated enough that the scan can rely on the described reads/writes. Returns false
// when the opcode has no described shape, in which case the caller treats this as an
// opaque side-effect and bails.
//
// shapeFlagFollowsExtension is permitted: the CSE pass either knows the exact extension
// semantics (tier-1 struct-field reads, the generic opGetField) and matches both words
// explicitly, or refuses the scan via the heap-mutation allow-list for unknown
// EXT-bearing opcodes.
//
// Takes inst (instruction) which is the candidate instruction to classify.
//
// Returns true when the operand-shape table describes inst's opcode.
func instructionShapeAllowsCseScan(inst instruction) bool {
	if int(inst.op) >= len(operandShapes) {
		return false
	}
	return operandShapes[inst.op].flags&shapeFlagDescribed != 0
}

// subOpDrillTier1MutatesHeap reports whether a tier-1 sub-op writes a struct field,
// mutates a slice or map, or transfers control to a tier-2 / tier-3 op that may do so.
//
// Takes inst (instruction) which is the candidate tier-1 instruction to classify.
//
// Returns true for the conservative block list. Sub-ops not listed here are presumed safe
// with respect to heap state (pure arithmetic, comparisons, loads, moves, jumps that read
// but do not write the heap). subOpDrillTier2 delegates classification to
// subOpDrillTier2MutatesHeap on the secondary discriminator.
func subOpDrillTier1MutatesHeap(inst instruction) bool {
	switch subOpcode(inst.a) {
	case subOpSetStructFieldInt, subOpSetStructFieldUint,
		subOpSetStructFieldFloat, subOpSetStructFieldBool,
		subOpSetStructFieldString,
		subOpIncStructFieldInt, subOpDecStructFieldInt,
		subOpIncStructFieldUint, subOpDecStructFieldUint,
		subOpMapDelete, subOpChannelReceive,
		subOpAppendUint, subOpAppendInt, subOpAppendString,
		subOpAppendFloat, subOpAppendBool,
		subOpStarAppendByteFast, subOpStarAppendByteSpread,
		subOpSpill, subOpReload:
		return true
	case subOpDrillTier2:
		return subOpDrillTier2MutatesHeap(inst)
	default:
	}
	return false
}

// subOpDrillTier2MutatesHeap classifies tier-2 sub-ops dispatched through opDrillTier1.
//
// The tier-2 discriminator lives in operand B; channel-close is the only mutator
// recognised. Tier-2 inc/dec ops, set-zero, load-nil, return, panic, and recover are pure
// with respect to heap state (they mutate scalar register banks only).
// subOpTier2DrillTier3 delegates further to subOpDrillTier3MutatesHeap.
//
// Takes inst (instruction) which is the candidate tier-2 instruction to classify.
//
// Returns true when the tier-2 sub-op may mutate heap state.
func subOpDrillTier2MutatesHeap(inst instruction) bool {
	tier2 := subOpcodeTier2(inst.b)
	switch tier2 {
	case subOpTier2ChannelClose:
		return true
	case subOpTier2DrillTier3:
		return subOpDrillTier3MutatesHeap(inst)
	default:
	}
	return false
}

// subOpDrillTier3MutatesHeap classifies tier-3 sub-ops dispatched through the tier-2
// drill.
//
// The tier-3 sub-ops in use are subOpTier3Nop and subOpTier3ReturnVoid; neither mutates
// heap. Any unknown tier-3 sub-op falls through to the conservative "true" return to
// preserve safety as the dispatch table grows.
//
// Takes inst (instruction) which is the candidate tier-3 instruction to classify.
//
// Returns true when the tier-3 sub-op may mutate heap state.
func subOpDrillTier3MutatesHeap(inst instruction) bool {
	switch subOpcodeTier3(inst.c) {
	case subOpTier3Nop, subOpTier3ReturnVoid:
		return false
	}
	return true
}

// instructionWritesRegisterInBank reports whether inst writes the given register index in
// the given bank.
//
// Reads the operand-shape table to determine which operand byte (if any) writes a
// register matching the requested bank. For ops whose shape lists a `roleRegDynamic`
// write operand, the runtime bank is resolved from a side channel (a `roleKindMarker`
// operand or a tier-1 sub-op discriminator) via resolveDynamicWriteBank. When the
// resolver returns a concrete bank, the comparison is exact; otherwise the role is
// treated as "any bank" and the caller over-invalidates rather than under-invalidates.
//
// Opcodes whose shape is not populated return false; callers must rely on the
// heap-mutation allow-list to bail on opaque instructions.
//
// Takes inst (instruction) which is the candidate instruction.
// Takes bank (operandRole) which is the destination bank to test for (roleRegInt,
// roleRegUint, roleRegFloat, roleRegBool, roleRegGeneral, roleRegString, etc.).
// Takes reg (uint8) which is the register index to test for.
//
// Returns true when inst writes register reg in bank.
func instructionWritesRegisterInBank(inst instruction, bank operandRole, reg uint8) bool {
	if bank == roleNone {
		return false
	}
	if int(inst.op) >= len(operandShapes) {
		return false
	}
	shape := operandShapes[inst.op]
	if shape.writes[0] && operandWriteMatchesBank(inst, 0, shape.a, bank) && inst.a == reg {
		return true
	}
	if shape.writes[1] && operandWriteMatchesBank(inst, 1, shape.b, bank) && inst.b == reg {
		return true
	}
	if shape.writes[2] && operandWriteMatchesBank(inst, 2, shape.c, bank) && inst.c == reg {
		return true
	}
	return false
}

// operandWriteMatchesBank reports whether the write at operand position targets the
// requested bank.
//
// Exact role matches always succeed; roleRegDynamic resolves through
// resolveDynamicWriteBank to recover the actual bank when the opcode carries it in a side
// operand. When the resolver cannot determine the bank, the comparison conservatively
// succeeds so callers over-invalidate rather than under-invalidate.
//
// Takes inst (instruction) which is the candidate instruction.
// Takes pos (int) which is the operand position (0, 1, or 2) within inst.
// Takes observed (operandRole) which is the role recorded in the operand-shape table for
// pos.
// Takes bank (operandRole) which is the destination bank to test for.
//
// Returns true when the write at pos matches bank (exactly or conservatively).
func operandWriteMatchesBank(inst instruction, pos int, observed, bank operandRole) bool {
	if observed == bank {
		return true
	}
	if observed != roleRegDynamic {
		return false
	}
	resolved, ok := resolveDynamicWriteBank(inst, pos)
	if !ok {
		return true
	}
	return resolved == bank
}

// resolveDynamicWriteBank attempts to decode the concrete bank a dynamic-bank write
// targets, when the opcode carries the bank in a side channel.
//
// For opTruncateNarrow the write targets operand A in the bank named by the
// `roleKindMarker` operand C; operand C carries a registerKind value whose roleForKind
// lookup gives the operand role. For opDrillTier1 the write targets operand B in the bank
// named by the tier-1 sub-op discriminator (operand A), with resolveTier1WriteBank
// handling the move / read / inc / dec sub-ops that target specific banks.
//
// Takes inst (instruction) which is the candidate instruction.
// Takes pos (int) which is the operand position (0, 1, or 2) within inst.
//
// Returns the resolved bank and true on a recognised case; returns (roleNone, false) when
// the opcode is unrecognised or operand pos is not a known dynamic-write position.
// Callers treat the false return as "bank unknown" and conservatively over-invalidate.
func resolveDynamicWriteBank(inst instruction, pos int) (operandRole, bool) {
	switch inst.op {
	case opTruncateNarrow:
		if pos != 0 {
			return roleNone, false
		}
		kind := registerKind(inst.c)
		return roleForKind(kind), true
	case opDrillTier1:
		if pos != 1 {
			return roleNone, false
		}
		return resolveTier1WriteBank(subOpcode(inst.a))
	default:
	}
	return roleNone, false
}

// resolveTier1WriteBank reports the destination bank of a tier-1 sub-op that writes
// operand B. Covers same-bank moves, cross-bank moves, struct-field reads, struct-field
// inc/dec ops, append helpers, and a handful of other arithmetic / load sub-ops whose
// target bank is statically determined by the sub-opcode.
//
// Takes sub (subOpcode) which is the tier-1 sub-opcode to classify.
//
// Returns (bank, true) on a recognised sub-op; (roleNone, false) otherwise. The CSE /
// LICM passes treat the false return as "bank unknown" and conservatively
// over-invalidate.
//
//nolint:revive,gocognit,cyclop // dispatch table: one case per sub-op
func resolveTier1WriteBank(sub subOpcode) (operandRole, bool) {
	switch sub {
	case subOpMoveInt, subOpMoveGeneralToInt:
		return roleRegInt, true
	case subOpMoveUint:
		return roleRegUint, true
	case subOpMoveFloat, subOpMoveGeneralToFloat:
		return roleRegFloat, true
	case subOpMoveBool:
		return roleRegBool, true
	case subOpMoveString, subOpMoveGeneralToString:
		return roleRegString, true
	case subOpMoveComplex:
		return roleRegComplex, true
	case subOpMoveIntToGeneral, subOpMoveFloatToGeneral,
		subOpMoveStringToGeneral:
		return roleRegGeneral, true
	case subOpGetStructFieldInt:
		return roleRegInt, true
	case subOpGetStructFieldUint:
		return roleRegUint, true
	case subOpGetStructFieldFloat:
		return roleRegFloat, true
	case subOpGetStructFieldBool:
		return roleRegBool, true
	case subOpGetStructFieldString:
		return roleRegString, true
	case subOpIncStructFieldInt, subOpDecStructFieldInt:
		return roleRegInt, true
	case subOpIncStructFieldUint, subOpDecStructFieldUint:
		return roleRegUint, true
	case subOpLen, subOpLenString,
		subOpLenSliceIntDirect, subOpLenSliceFloatDirect,
		subOpLenSliceStringDirect, subOpLenSliceBoolDirect,
		subOpLenSliceUintDirect, subOpLenSliceByteDirect,
		subOpCapSliceIntDirect, subOpCapSliceFloatDirect,
		subOpCapSliceStringDirect, subOpCapSliceBoolDirect,
		subOpCapSliceUintDirect, subOpCapSliceByteDirect:
		return roleRegInt, true
	default:
	}
	return roleNone, false
}
