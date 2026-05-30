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
	// gvnRegisterBankCount is the number of typed register banks the GVN state tracks (int,
	// float, string, general, bool, uint, complex).
	gvnRegisterBankCount = 7

	// gvnRegisterBankSize is the per-bank register-index space, matching the VM's
	// 256-register banks.
	gvnRegisterBankSize = 256

	// gvnRegisterTrackedBanks aliases gvnRegisterBankCount for readability at iteration
	// sites.
	gvnRegisterTrackedBanks = gvnRegisterBankCount

	// gvnNoDefiningPC marks a register whose last-defining PC is unknown or has been
	// invalidated.
	gvnNoDefiningPC = -1
)

const (
	// gvnBankIndexInt is the bank discriminator for int-typed registers in gvnState's
	// regLastDef array.
	gvnBankIndexInt = iota

	// gvnBankIndexFloat is the bank discriminator for float-typed registers.
	gvnBankIndexFloat

	// gvnBankIndexString is the bank discriminator for string-typed registers.
	gvnBankIndexString

	// gvnBankIndexGeneral is the bank discriminator for general-purpose registers.
	gvnBankIndexGeneral

	// gvnBankIndexBool is the bank discriminator for bool-typed registers.
	gvnBankIndexBool

	// gvnBankIndexUint is the bank discriminator for uint-typed registers.
	gvnBankIndexUint

	// gvnBankIndexComplex is the bank discriminator for complex-typed registers.
	gvnBankIndexComplex
)

const (
	// gvnBankIndexInvalid is the sentinel returned by gvnBankForArithmetic when the opcode
	// does not classify into one of the tracked banks.
	gvnBankIndexInvalid = -1
)

// Global value numbering extends the per-block CSE (function_cse_field_read.go, scoped to
// struct-field reads) to pure arithmetic redundancy across the full function. Strictly
// less ambitious than a textbook GVN: only the operand-canonical form and the dominator
// check go further than CSE.
//
// What GVN catches that CSE does not: commutative operands, where `opAddInt a b` and
// `opAddInt b a` get the same value key; and cross-block redundancy, where a candidate
// definition in an earlier dominator can rewrite a later use even when they sit on
// different control-flow paths joining at the use's PC.
//
// Scope limitations: no memory-version tracking for struct-field reads (the alias-aware
// CSE in function_cse_field_read.go handles that within a block); no cross-call value
// numbering (calls invalidate every value number conservatively, the same convention the
// alias analysis uses for register classes); no type-folded equalities (e.g. recognising
// that opUintToInt of a non-negative int produces the same value); no constant equality
// (two opLoadIntConst with the same constant index, since the compiler's dedup at emit
// time covers most cases and the rest fall through to the value table whose key already
// includes the constant index).

// runFunctionGvn rewrites later equal-value computations as MOVE.
//
// Each MOVE reads from an earlier dominator-validated definition. Bounded: skips
// functions over 2000 instructions and bounded internal worklist; cost is proportional to
// body length and stops well before the verifier's superlinear cost regime. Sound by
// construction: every rewrite must satisfy the operand value-numbers matching the
// candidate definer's, the candidate definer dominating the use's PC, and the candidate
// definer's destination register still holding the cached value (no intervening write to
// that register).
//
// Takes ctx (context.Context) which carries cancellation.
//
// Returns error when context cancellation fires.
func (cf *CompiledFunction) runFunctionGvn(ctx context.Context) error {
	if !peepholeGvnEnabled {
		return nil
	}
	body := cf.body
	if len(body) < 2 {
		return nil
	}
	const maxBodyForGvn = 2000
	if len(body) > maxBodyForGvn {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("runFunctionGvn cancelled: %w", err)
	}
	dom := computeFunctionDominators(body)
	if dom == nil {
		return nil
	}
	state := newGvnState(len(body))
	for pc := range body {
		if pc&optimisationLoopCheckMask == 0 {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("runFunctionGvn cancelled: %w", err)
			}
		}
		cf.applyGvnAtPC(body, pc, dom, state)
	}
	return nil
}

// applyGvnAtPC runs the GVN per-instruction logic at pc against state, either clearing on
// calls, rewriting redundant computations, or updating the regLastDef table for unhandled
// writes.
//
// Takes body ([]instruction) which is the function body being analysed.
// Takes pc (int) which is the current program counter.
// Takes dom (*functionDominators) which is the dominator information used to decide
// whether a rewrite is sound.
// Takes state (*gvnState) which tracks per-PC value numbers and the regLastDef table.
func (cf *CompiledFunction) applyGvnAtPC(body []instruction, pc int, dom *functionDominators, state *gvnState) {
	inst := body[pc]
	if isCallOpcode(inst.op) {
		state.clearOnCall(pc)
		return
	}
	key, ok := computeGvnValueKey(inst, state)
	if ok {
		candidatePC, found := state.valueTable[key]
		if found && cf.gvnRewriteIfSafe(body, pc, candidatePC, dom, state) {
			state.recordRewrite(inst, pc, candidatePC)
			return
		}
		state.valueTable[key] = pc
	}
	state.updateRegLastDef(inst, pc)
}

// updateRegLastDef refreshes the regLastDef table for the destination of inst.
//
// When gvnInstructionDest reports a tracked-bank destination, the dest's last-defining PC
// becomes this PC. For other ops, every tracked register that
// instructionWritesRegisterInBank reports as written has its last-defining PC cleared to
// gvnNoDefiningPC, preventing stale value numbers from leaking past spills, reloads, or
// unmodelled writes.
//
// Takes inst (instruction) which is the instruction whose destination effects update the
// table.
// Takes pc (int) which is the program counter assigned as the new last-defining PC.
func (state *gvnState) updateRegLastDef(inst instruction, pc int) {
	if bank, reg, ok := gvnInstructionDest(inst); ok {
		state.regLastDef[bank][reg] = pc
		if pc >= 0 && pc < len(state.perPCValueNum) {
			state.perPCValueNum[pc] = pc
		}
		return
	}
	for bank := range gvnRegisterTrackedBanks {
		role, ok := gvnBankToRole(bank)
		if !ok {
			continue
		}
		for reg := range gvnRegisterBankSize {
			if state.regLastDef[bank][reg] == gvnNoDefiningPC {
				continue
			}
			if instructionWritesRegisterInBank(inst, role, uint8(reg)) {
				state.regLastDef[bank][reg] = gvnNoDefiningPC
			}
		}
	}
}

// gvnState carries the per-function GVN tables: the value-key lookup, the per-register
// last-defining PC, and the per-PC value number used by computeGvnValueKey to recurse
// into operand history.
//
// Memory-version tracking for struct-field / map / slice reads is out of scope:
// linear-walk memory versioning under a dominator check degenerates to "match only when
// no writes seen between candidate and use in PC-numeric order", which is strictly weaker
// than the alias-aware intra-block CSE provided by function_cse_field_read.go. A sound
// cross-block memory CSE requires full memory SSA with phi-of-memory at every join.
type gvnState struct {
	// valueTable maps each computed value key to the PC that produced it.
	valueTable map[gvnValueKey]int

	// perPCValueNum records the canonical value number assigned to each PC.
	perPCValueNum []int

	// regLastDef holds the last-defining PC for every tracked register slot, indexed by bank
	// then register.
	regLastDef [gvnRegisterTrackedBanks][gvnRegisterBankSize]int
}

// clearOnCall wipes every value-table entry and resets every register's last-defining PC.
// Called when the analysis encounters any call opcode, mirroring the alias analysis's
// "calls go wild" rule.
func (state *gvnState) clearOnCall(_ int) {
	for k := range state.valueTable {
		delete(state.valueTable, k)
	}
	for bank := range gvnRegisterTrackedBanks {
		for reg := range gvnRegisterBankSize {
			state.regLastDef[bank][reg] = gvnNoDefiningPC
		}
	}
}

// recordRewrite updates the per-PC value number to point at the candidate so subsequent
// ops that read this PC's destination see it as the same value as candidatePC.
//
// Takes inst (instruction) which is the rewritten instruction whose destination register
// is being tracked.
// Takes pc (int) which is the PC of the rewritten instruction.
// Takes candidatePC (int) which is the PC whose value number now applies to pc.
func (state *gvnState) recordRewrite(inst instruction, pc, candidatePC int) {
	bankIndex, reg, ok := gvnInstructionDest(inst)
	if !ok {
		return
	}
	state.regLastDef[bankIndex][reg] = pc
	if pc >= 0 && pc < len(state.perPCValueNum) {
		state.perPCValueNum[pc] = candidatePC
	}
}

// gvnValueKey is the hash-table key uniquely naming the computation at one PC. Two PCs
// hash to the same key iff their opcodes match and their operands' value-numbers match
// (after commutative canonicalisation).
type gvnValueKey struct {
	// op identifies the opcode that produces the value.
	op opcode

	// operand0 is the canonical value number of the first operand.
	operand0 int

	// operand1 is the canonical value number of the second operand.
	operand1 int
}

// gvnRewriteIfSafe attempts to rewrite the instruction at pc as a MOVE from the
// candidate's destination register.
//
// Safety checks include the dominator condition (candidatePC must dominate pc) and a
// liveness check (the candidate's destination register must still hold the candidate's
// value at pc, meaning the last-write-PC equals candidatePC).
//
// Takes body ([]instruction) which is the function body whose instruction at pc may be
// rewritten.
// Takes pc (int) which is the program counter of the redundant computation.
// Takes candidatePC (int) which is the earlier PC whose value can be reused.
// Takes dom (*functionDominators) which is the dominator information used for the
// dominance check.
// Takes state (*gvnState) which is the per-function GVN state holding register liveness.
//
// Returns true when the rewrite was applied; false otherwise.
func (cf *CompiledFunction) gvnRewriteIfSafe(body []instruction, pc, candidatePC int, dom *functionDominators, state *gvnState) bool {
	if !dom.dominates(candidatePC, pc) {
		return false
	}
	candidateInst := body[candidatePC]
	bankIndex, candidateDest, ok := gvnInstructionDest(candidateInst)
	if !ok {
		return false
	}
	if state.regLastDef[bankIndex][candidateDest] != candidatePC {
		return false
	}
	currentInst := body[pc]
	_, currentDest, ok := gvnInstructionDest(currentInst)
	if !ok {
		return false
	}
	moveInst, ok := gvnEmitMoveForBank(bankIndex, currentDest, candidateDest)
	if !ok {
		return false
	}
	body[pc] = moveInst
	cf.recordPeepholeRewrite(pc, peepholeRewriteGvn, candidatePC)
	return true
}

// gvnBankToRole returns the operandRole that classifies a register in the given GVN-bank
// index.
//
// Used by updateRegLastDef to drive instructionWritesRegisterInBank for kill-on-write.
//
// Takes bank (int) which is the GVN bank index whose role mapping is required.
//
// Returns the operandRole corresponding to bank, and true when bank maps to a tracked
// role; false otherwise.
func gvnBankToRole(bank int) (operandRole, bool) {
	switch bank {
	case gvnBankIndexInt:
		return roleRegInt, true
	case gvnBankIndexFloat:
		return roleRegFloat, true
	case gvnBankIndexString:
		return roleRegString, true
	case gvnBankIndexGeneral:
		return roleRegGeneral, true
	case gvnBankIndexBool:
		return roleRegBool, true
	case gvnBankIndexUint:
		return roleRegUint, true
	case gvnBankIndexComplex:
		return roleRegComplex, true
	}
	return roleNone, false
}

// newGvnState builds an empty GVN state sized for a function body of bodyLen
// instructions.
//
// Every regLastDef and perPCValueNum entry is seeded to gvnNoDefiningPC so the first read
// of any register before any write returns "no value number".
//
// Takes bodyLen (int) which is the instruction count used to size the value tables.
//
// Returns the freshly initialised *gvnState.
func newGvnState(bodyLen int) *gvnState {
	state := &gvnState{
		valueTable:    make(map[gvnValueKey]int, bodyLen),
		perPCValueNum: make([]int, bodyLen),
	}
	for bank := range gvnRegisterTrackedBanks {
		for reg := range gvnRegisterBankSize {
			state.regLastDef[bank][reg] = gvnNoDefiningPC
		}
	}
	for pc := range bodyLen {
		state.perPCValueNum[pc] = gvnNoDefiningPC
	}
	return state
}

// computeGvnValueKey returns the value-key for inst when inst is a pure value-producing
// opcode whose result depends only on the observed values of its operand registers.
//
// Commutative ops (opAddInt, opMulInt, opAddUint, opMulUint, opAddFloat, opMulFloat) sort
// their operand value-numbers so the key is invariant under operand-order swaps.
//
// Takes inst (instruction) which is the instruction whose value key is required.
// Takes state (*gvnState) which is the per-function GVN state used to resolve operand
// value numbers.
//
// Returns the computed gvnValueKey when inst is modelled precisely, and a boolean true
// when a key was produced; false for opcodes outside the model.
func computeGvnValueKey(inst instruction, state *gvnState) (gvnValueKey, bool) {
	switch inst.op {
	case opAddInt, opMulInt, opAddUint, opMulUint, opAddFloat, opMulFloat:
		left, leftOK := gvnLookupValueNumber(state, inst.b, gvnBankForArithmetic(inst.op))
		right, rightOK := gvnLookupValueNumber(state, inst.c, gvnBankForArithmetic(inst.op))
		if !leftOK || !rightOK {
			return gvnValueKey{}, false
		}
		if left > right {
			left, right = right, left
		}
		return gvnValueKey{op: inst.op, operand0: left, operand1: right}, true
	case opSubInt, opSubUint, opSubFloat, opDivInt, opDivUint, opDivFloat:
		left, leftOK := gvnLookupValueNumber(state, inst.b, gvnBankForArithmetic(inst.op))
		right, rightOK := gvnLookupValueNumber(state, inst.c, gvnBankForArithmetic(inst.op))
		if !leftOK || !rightOK {
			return gvnValueKey{}, false
		}
		return gvnValueKey{op: inst.op, operand0: left, operand1: right}, true
	case opAddIntConst, opSubIntConst, opMulIntConst:
		src, ok := gvnLookupValueNumber(state, inst.b, gvnBankIndexInt)
		if !ok {
			return gvnValueKey{}, false
		}
		return gvnValueKey{op: inst.op, operand0: src, operand1: int(inst.c)}, true
	default:
	}
	return gvnValueKey{}, false
}

// gvnLookupValueNumber returns the canonical value number of the value currently held in
// register reg of bank.
//
// When the register's last-defining PC has a recorded value number in perPCValueNum, that
// number is returned, handling transitive equivalences where a MOVE from a
// previously-numbered register propagates the number. When no value number is recorded,
// the defining PC itself is returned so subsequent same-PC lookups produce the same key.
//
// Takes state (*gvnState) which is the per-function GVN state holding the register
// tables.
// Takes reg (uint8) which is the register slot whose value number is required.
// Takes bank (int) which is the bank index identifying the register's type family.
//
// Returns the canonical value number (zero when none is known) and a boolean false when
// the register has not been written within the visible scope (i.e. before any call
// boundary or function entry); true otherwise.
func gvnLookupValueNumber(state *gvnState, reg uint8, bank int) (int, bool) {
	if bank < 0 || bank >= gvnRegisterTrackedBanks {
		return 0, false
	}
	def := state.regLastDef[bank][reg]
	if def == gvnNoDefiningPC {
		return 0, false
	}
	if def < 0 || def >= len(state.perPCValueNum) {
		return def, true
	}
	if state.perPCValueNum[def] != gvnNoDefiningPC {
		return state.perPCValueNum[def], true
	}
	return def, true
}

// gvnBankForArithmetic returns the int-encoded bank index for the arithmetic opcode's
// operand registers.
//
// The op family determines the bank (Int arithmetic reads ints, Float reads floats,
// etc.).
//
// Takes op (opcode) which is the arithmetic opcode whose operand bank is required.
//
// Returns the bank index, or gvnBankIndexInvalid when op is not arithmetic.
func gvnBankForArithmetic(op opcode) int {
	switch op {
	case opAddInt, opSubInt, opMulInt, opDivInt:
		return gvnBankIndexInt
	case opAddUint, opSubUint, opMulUint, opDivUint:
		return gvnBankIndexUint
	case opAddFloat, opSubFloat, opMulFloat, opDivFloat:
		return gvnBankIndexFloat
	default:
	}
	return gvnBankIndexInvalid
}

// gvnInstructionDest returns the destination bank index and register of inst, or zero
// values when inst does not write a register in a tracked bank.
//
// Mirrors the GVN-relevant subset of operandShapes so that only opcodes whose
// destinations participate in the value key are tracked. Opcodes outside this set still
// feed the regLastDef update via updateRegLastDef which clears regLastDef so stale values
// do not bleed into future keys.
//
// Takes inst (instruction) which is the instruction whose destination is being
// classified.
//
// Returns the destination bank index, the destination register slot and a boolean true
// when inst writes a tracked register; false otherwise.
func gvnInstructionDest(inst instruction) (int, uint8, bool) {
	switch inst.op {
	case opAddInt, opSubInt, opMulInt, opDivInt, opRemInt,
		opAddIntConst, opSubIntConst, opMulIntConst,
		opLoadIntConst:
		return gvnBankIndexInt, inst.a, true
	case opAddUint, opSubUint, opMulUint, opDivUint, opRemUint,
		opLoadUintConst:
		return gvnBankIndexUint, inst.a, true
	case opAddFloat, opSubFloat, opMulFloat, opDivFloat,
		opLoadFloatConst:
		return gvnBankIndexFloat, inst.a, true
	default:
	}
	return 0, 0, false
}

// gvnEmitMoveForBank returns the bytecode instruction that copies a register from source
// to destination within the named bank.
//
// The general bank uses tier-0 opMoveGeneral; scalar banks use the tier-1 same-bank move
// sub-ops. Banks that have no GVN-supported MOVE in this VM (such as complex or
// dynamic-bank) yield a zero instruction with false.
//
// Takes bank (int) which is the GVN bank index that determines the MOVE encoding.
// Takes destination (uint8) which is the destination register slot.
// Takes source (uint8) which is the source register slot.
//
// Returns the encoded MOVE instruction when supported and true when bank has a supported
// MOVE encoding; false otherwise.
func gvnEmitMoveForBank(bank int, destination, source uint8) (instruction, bool) {
	switch bank {
	case gvnBankIndexInt:
		return makeInstruction(opDrillTier1, uint8(subOpMoveInt), destination, source), true
	case gvnBankIndexUint:
		return makeInstruction(opDrillTier1, uint8(subOpMoveUint), destination, source), true
	case gvnBankIndexFloat:
		return makeInstruction(opDrillTier1, uint8(subOpMoveFloat), destination, source), true
	case gvnBankIndexGeneral:
		return makeInstruction(opMoveGeneral, destination, source, 0), true
	}
	return instruction{}, false
}
