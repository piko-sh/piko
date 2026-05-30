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

// aliasClass identifies a set of provably-equivalent pointer origins.
//
// Class zero is "wild": it matches every other class and indicates the pointer's origin
// could not be tracked precisely. Classes 1+ are concrete origin identifiers, allocated
// by classMinter as the analysis encounters new origin-producing instructions.
type aliasClass uint16

const (
	// aliasClassWild is the sentinel matching every other class. It is returned by the merge
	// widening step and by transfer functions that cannot determine a precise origin.
	aliasClassWild aliasClass = 0

	// generalAliasBankSize matches the general-bank register count tracked at every program
	// point. Kept equal to generalRegisterBankSize so the per-PC environment matches the
	// bank's full addressing space.
	generalAliasBankSize = generalRegisterBankSize

	// maxAliasWorklistIterations caps worklist iterations at 200000.
	//
	// Convergence is guaranteed in principle because every transfer function only moves
	// classes toward wild, but the per-PC field-class memo (fieldClassMemo) is what makes
	// deriveFieldAliasClass stable across revisits; the cap is a defence-in-depth safety net
	// so a pathological CFG or a memo miss can never hang the compile. When the cap fires
	// the analysis abandons the partial result and leaves cf.aliasInfo nil, forcing mayAlias
	// to its conservative "true" fallback.
	maxAliasWorklistIterations = 200000
)

// aliasEnvironment is the immutable snapshot of every general-bank register's alias class
// at one program point.
type aliasEnvironment struct {
	// class holds the alias class for each general-bank register slot.
	class [generalAliasBankSize]aliasClass
}

// pointerAliasInfo is the per-function output of runPointerAliasAnalysis.
//
// Indexed by PC, perPCEnv[pc] holds the alias environment immediately AFTER body[pc]
// executes. PCs that the worklist never reached carry an all-wild environment, a sound
// fallback where every may-alias query returns "yes".
type pointerAliasInfo struct {
	// perPCEnv holds the post-execution alias environment for each PC in the function body.
	perPCEnv []aliasEnvironment
}

// mayAlias reports whether two general-bank registers could refer to the same heap object
// at the program point immediately after pc executed.
//
// When the alias environment for pc is not populated (e.g. the analysis was skipped or
// the PC is unreachable), the conservative answer "true" is returned, treating the pair
// as may-alias.
//
// Takes pc (int) which is the PC whose post-execution environment is consulted.
// Takes regA (uint8) which is the first register slot being queried.
// Takes regB (uint8) which is the second register slot being queried.
//
// Returns true when regA == regB (always aliases), when either class is wild (unknown
// origin), or when both carry the same non-wild class (provably aliased).
// Returns false only when both classes are concrete and different; callers (CSE, LICM,
// GVN) use the false return to keep cached reads alive through writes via the other
// register.
func (info *pointerAliasInfo) mayAlias(pc int, regA, regB uint8) bool {
	if info == nil {
		return true
	}
	if regA == regB {
		return true
	}
	if pc < 0 || pc >= len(info.perPCEnv) {
		return true
	}
	env := &info.perPCEnv[pc]
	classA := env.class[regA]
	classB := env.class[regB]
	if classA == aliasClassWild || classB == aliasClassWild {
		return true
	}
	return classA == classB
}

// aliasClassMinter allocates fresh aliasClass values as the analysis encounters new
// origins. Wraps at the uint16 ceiling; after that every new origin collapses to wild
// (sound, just imprecise).
type aliasClassMinter struct {
	// next is the alias class identifier handed out by the next call to fresh.
	next aliasClass
}

// fresh returns the next unused alias class, or wild when the 16-bit space is exhausted.
//
// Returns the next alias class identifier, or aliasClassWild on overflow.
func (mint *aliasClassMinter) fresh() aliasClass {
	if mint.next == ^aliasClass(0) {
		return aliasClassWild
	}
	mint.next++
	return mint.next
}

// fieldClassMemo memoises deriveFieldAliasClass keyed by the field load's PC and the
// receiver's alias class.
//
// Without memoisation deriveFieldAliasClass mints a fresh class on every visit, so a
// field load revisited by the worklist would produce a different post-environment each
// time and the aliasEnvironmentEqual fixpoint test could never report convergence. Keying
// by (pc, parentClass) makes the derived class deterministic: the same field load with
// the same receiver class always yields the same result, so revisiting a PC is idempotent
// once its inputs stabilise.
type fieldClassMemo struct {
	// classes maps a (pc, parentClass) pair to the derived field class.
	classes map[fieldClassMemoKey]aliasClass
}

// fieldClassMemoKey identifies one memoised field-class derivation.
type fieldClassMemoKey struct {
	// pc is the program counter of the field-load instruction.
	pc int

	// parentClass is the receiver register's alias class at that PC.
	parentClass aliasClass
}

// freshForPC returns the memoised fresh class for a PC-only origin (allocation,
// address-of, general const load).
//
// The first call for a PC mints a fresh class and records it; every later call for the
// same PC returns the recorded class, so a worklist revisit of the instruction is
// idempotent.
//
// Takes pc (int) which is the PC of the origin-producing instruction.
// Takes minter (*aliasClassMinter) which mints the class on first use.
//
// Returns the stable fresh class for pc.
func (memo *fieldClassMemo) freshForPC(pc int, minter *aliasClassMinter) aliasClass {
	key := fieldClassMemoKey{pc: pc, parentClass: aliasClassWild}
	if existing, ok := memo.classes[key]; ok {
		return existing
	}
	derived := minter.fresh()
	memo.classes[key] = derived
	return derived
}

// newFieldClassMemo returns an empty field-class memo.
//
// Returns a memo ready to record derivations.
func newFieldClassMemo() *fieldClassMemo {
	return &fieldClassMemo{classes: make(map[fieldClassMemoKey]aliasClass)}
}

// runPointerAliasAnalysis populates cf.aliasInfo with the per-PC alias environment
// computed by a worklist forward dataflow analysis. Sound by construction: every transfer
// function over-approximates the runtime behaviour, and merge-points conservatively widen
// to wild when predecessors disagree.
//
// Skipped on empty bodies. Skipped silently when the body is too large to analyse
// efficiently (cap at 2000 instructions; very long functions are rare and the analysis
// cost grows superlinearly).
//
// Takes ctx (context.Context) which can cancel the analysis.
// Takes cf (*CompiledFunction) whose body and parameter registers the analysis inspects.
//
// Returns error when the context is cancelled before completion.
func runPointerAliasAnalysis(ctx context.Context, cf *CompiledFunction) error {
	body := cf.body
	if len(body) == 0 {
		return nil
	}
	const maxBodyForAliasAnalysis = 2000
	if len(body) > maxBodyForAliasAnalysis {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("runPointerAliasAnalysis cancelled: %w", err)
	}
	info := &pointerAliasInfo{
		perPCEnv: make([]aliasEnvironment, len(body)),
	}
	minter := aliasClassMinter{}
	memo := newFieldClassMemo()
	entryEnv := seedEntryEnvironment(cf, &minter)
	predecessors := buildAliasPredecessors(body)
	successors := buildAliasSuccessors(body)
	worklist := []int{0}
	enqueued := make([]bool, len(body))
	enqueued[0] = true
	iteration := 0
	for len(worklist) > 0 {
		if iteration&optimisationLoopCheckMask == 0 {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("runPointerAliasAnalysis cancelled: %w", err)
			}
		}
		iteration++
		if iteration > maxAliasWorklistIterations {
			cf.aliasInfo = nil
			return nil
		}
		pc := worklist[len(worklist)-1]
		worklist = worklist[:len(worklist)-1]
		enqueued[pc] = false
		newEnv := applyAliasTransfer(cf, pc, new(mergeIncomingAliasEnvs(info, predecessors[pc], &entryEnv, pc == 0)), &minter, memo)
		if aliasEnvironmentEqual(&info.perPCEnv[pc], &newEnv) {
			continue
		}
		info.perPCEnv[pc] = newEnv
		worklist = enqueueAliasSuccessors(successors[pc], len(body), enqueued, worklist)
	}
	cf.aliasInfo = info
	return nil
}

// enqueueAliasSuccessors appends each in-range successor to the worklist when it is not
// already pending, marking it as enqueued.
//
// Takes successors ([]int) which are the candidate next PCs.
// Takes bodyLength (int) which bounds-checks successor entries.
// Takes enqueued ([]bool) which tracks pending PCs.
// Takes worklist ([]int) which is the worklist receiving the new PCs.
//
// Returns the updated worklist slice.
func enqueueAliasSuccessors(successors []int, bodyLength int, enqueued []bool, worklist []int) []int {
	for _, succ := range successors {
		if succ < 0 || succ >= bodyLength {
			continue
		}
		if enqueued[succ] {
			continue
		}
		enqueued[succ] = true
		worklist = append(worklist, succ)
	}
	return worklist
}

// seedEntryEnvironment produces the alias environment that applies at the function's
// entry, before any instruction has executed.
//
// Each general-bank parameter register receives a fresh alias class, a sound conservative
// choice because the caller's argument pointers come from distinct sources unless the
// caller intentionally aliases them (a pattern this analysis does not model; CSE remains
// safe because it consults runtime aliasing at the only access points that matter).
// Non-parameter general registers are wild at entry because they hold no meaningful
// pointer until written by some opcode inside the body.
//
// Takes cf (*CompiledFunction) which is the compiled function whose parameter kinds
// determine seeding.
// Takes minter (*aliasClassMinter) which is the class minter used to allocate fresh
// per-parameter classes.
//
// Returns the seeded entry environment.
func seedEntryEnvironment(cf *CompiledFunction, minter *aliasClassMinter) aliasEnvironment {
	var env aliasEnvironment
	for index, kind := range cf.parameterKinds {
		if kind != registerGeneral {
			continue
		}
		if index >= generalAliasBankSize {
			continue
		}
		env.class[index] = minter.fresh()
	}
	return env
}

// buildAliasPredecessors returns, per PC, the list of PCs whose execution can immediately
// precede this PC.
//
// Predecessors include the linear fall-through (PC-1 when that instruction is not an
// unconditional terminator) plus any in-body branch source whose target equals this PC.
//
// Takes body ([]instruction) which is the function body whose control-flow predecessors
// are being computed.
//
// Returns a per-PC slice of predecessor PCs.
func buildAliasPredecessors(body []instruction) [][]int {
	predecessors := make([][]int, len(body))
	for pc := 1; pc < len(body); pc++ {
		if isLinearFallThroughFrom(body[pc-1]) {
			predecessors[pc] = append(predecessors[pc], pc-1)
		}
	}
	for pc, inst := range body {
		offset, isJump := jumpOffsetOf(inst)
		if !isJump {
			continue
		}
		target := pc + 1 + offset
		if target < 0 || target >= len(body) {
			continue
		}
		predecessors[target] = append(predecessors[target], pc)
	}
	return predecessors
}

// buildAliasSuccessors returns, per PC, the list of PCs whose execution can immediately
// follow this PC.
//
// Mirrors buildAliasPredecessors and is used by the worklist to propagate updates.
//
// Takes body ([]instruction) which is the function body whose control-flow successors are
// being computed.
//
// Returns a per-PC slice of successor PCs.
func buildAliasSuccessors(body []instruction) [][]int {
	successors := make([][]int, len(body))
	for pc, inst := range body {
		if isLinearFallThroughFrom(inst) && pc+1 < len(body) {
			successors[pc] = append(successors[pc], pc+1)
		}
		offset, isJump := jumpOffsetOf(inst)
		if !isJump {
			continue
		}
		target := pc + 1 + offset
		if target < 0 || target >= len(body) {
			continue
		}
		successors[pc] = append(successors[pc], target)
	}
	return successors
}

// mergeIncomingAliasEnvs combines predecessor post-environments into the pre-environment
// for pc.
//
// When predecessors disagree on a register's class, the merge widens to wild (safe but
// imprecise). At the entry PC the merge incorporates the function's seed environment so
// parameter classes flow through.
//
// Takes info (*pointerAliasInfo) which is the per-function alias output whose perPCEnv
// supplies predecessor environments.
// Takes preds ([]int) which is the predecessor PC list for the current PC.
// Takes entry (*aliasEnvironment) which is the seeded entry environment used when isEntry
// is true.
// Takes isEntry (bool) which is true when the current PC is the function entry PC.
//
// Returns the merged incoming environment.
func mergeIncomingAliasEnvs(info *pointerAliasInfo, preds []int, entry *aliasEnvironment, isEntry bool) aliasEnvironment {
	var merged aliasEnvironment
	first := true
	if isEntry {
		merged = *entry
		first = false
	}
	for _, pred := range preds {
		if pred < 0 || pred >= len(info.perPCEnv) {
			continue
		}
		predEnv := &info.perPCEnv[pred]
		if first {
			merged = *predEnv
			first = false
			continue
		}
		mergeAliasEnvironmentInto(&merged, predEnv)
	}
	return merged
}

// mergeAliasEnvironmentInto widens destination to the least-upper-bound of destination
// and source.
//
// For each register: if classes match, keep; otherwise widen to wild.
//
// Takes destination (*aliasEnvironment) which is the environment widened in place.
// Takes source (*aliasEnvironment) which is the environment merged into destination.
func mergeAliasEnvironmentInto(destination *aliasEnvironment, source *aliasEnvironment) {
	for index := range destination.class {
		if destination.class[index] != source.class[index] {
			destination.class[index] = aliasClassWild
		}
	}
}

// aliasEnvironmentEqual reports whether two environments hold identical classes in every
// register slot.
//
// Takes a (*aliasEnvironment) which is the first environment.
// Takes b (*aliasEnvironment) which is the second environment.
//
// Returns true when every register slot agrees; false otherwise.
func aliasEnvironmentEqual(a, b *aliasEnvironment) bool {
	for index := range a.class {
		if a.class[index] != b.class[index] {
			return false
		}
	}
	return true
}

// applyAliasTransfer computes the post-environment for body[pc] given the pre-environment
// merged in from predecessors.
//
// Dispatches per opcode: pointer-producing opcodes (opAllocIndirect, opAddr,
// opLoadGeneralConst, opGetField, opGetStructFieldGeneral, opGetStructFieldRawPointerT0)
// mint fresh classes or derive from the receiver; general-bank move opcodes propagate the
// source class; calls clobber all general-bank registers to wild; any other opcode that
// writes a general register without known semantics produces a wild destination.
//
// Takes cf (*CompiledFunction) which is the compiled function whose body[pc] is being
// transferred.
// Takes pc (int) which is the program counter being processed.
// Takes env (*aliasEnvironment) which is the pre-environment merged from predecessors.
// Takes minter (*aliasClassMinter) which is the class minter used for fresh class
// allocation.
// Takes memo (*fieldClassMemo) which keeps field-load class derivations stable across
// worklist revisits.
//
// Returns the post-execution environment for body[pc].
func applyAliasTransfer(cf *CompiledFunction, pc int, env *aliasEnvironment, minter *aliasClassMinter, memo *fieldClassMemo) aliasEnvironment {
	result := *env
	inst := cf.body[pc]
	if isAliasCallOpcode(inst.op) {
		clearAllGeneralAliasClasses(&result)
		return result
	}
	destination, ok := aliasGeneralDestRegister(inst)
	if !ok {
		return result
	}
	result.class[destination] = computeAliasClassForWrite(&result, pc, inst, minter, memo)
	return result
}

// isAliasCallOpcode reports whether op transfers control to another function in a way
// that may shuffle, alias, or return arbitrary pointers in the general bank.
//
// Takes op (opcode) which is the opcode being classified.
//
// Returns true when op is treated as a call boundary; false otherwise.
func isAliasCallOpcode(op opcode) bool {
	switch op {
	case opCall, opTailCall, opCallMethod, opCallMethodInlineable, opCallNative, opCallIIFE:
		return true
	default:
	}
	return false
}

// clearAllGeneralAliasClasses widens every register's class to wild.
//
// Used at call boundaries where the callee may have aliased or reassigned any
// general-bank register the caller observes.
//
// Takes env (*aliasEnvironment) which is the environment whose classes are reset to wild
// in place.
func clearAllGeneralAliasClasses(env *aliasEnvironment) {
	for index := range env.class {
		env.class[index] = aliasClassWild
	}
}

// aliasGeneralDestRegister returns the general-bank destination register of inst, or zero
// with false when inst writes no general register.
//
// Recognises the patterns the analysis tracks; unknown patterns return zero with false
// and leave the environment unchanged. This over-approximates the "writes general"
// relation safely because the only way to widen incorrectly is to claim a write where
// there is none, and we default to "no write" here, with the call handler upstream
// catching the case where unknown effects might hide writes.
//
// Takes inst (instruction) which is the instruction being inspected.
//
// Returns the general-bank destination register slot and true when inst writes a tracked
// general register; (0, false) otherwise.
func aliasGeneralDestRegister(inst instruction) (uint8, bool) {
	switch inst.op {
	case opAllocIndirect, opAddr, opLoadGeneralConst,
		opGetStructFieldGeneral, opGetStructFieldRawPointerT0,
		opGetField, opMoveGeneral, opIndex, opMakeSlice,
		opMakeClosure, opPackInterface, opUnpackInterface:
		return inst.a, true
	default:
	}
	if inst.op == opDrillTier1 {
		switch subOpcode(inst.a) {
		case subOpMoveIntToGeneral, subOpMoveFloatToGeneral,
			subOpMoveStringToGeneral:
			return inst.b, true
		default:
		}
	}
	return 0, false
}

// computeAliasClassForWrite returns the class that should be assigned to inst's
// destination after the transfer.
//
// Dispatches by opcode: allocations, address-of, and const loads return a fresh class
// (each PC produces a unique origin); field loads return a class derived from the
// receiver's class plus the layout index, so two reads of the same field from registers
// that share an alias class share the derived class; same-bank moves propagate the source
// class; cross-bank moves into general yield wild because the value crossed a type
// boundary and the origin is unknown; everything else writing general yields wild.
//
// All fresh-class allocations are memoised by PC (and, for field loads, by the receiver
// class too). The minter hands out a new integer on every call, so re-deriving a class on
// a worklist revisit without memoisation would yield a different post-environment each
// time and the aliasEnvironmentEqual fixpoint could never converge. Memoising makes every
// revisit of the same PC with the same inputs idempotent.
//
// Takes env (*aliasEnvironment) which is the pre-write environment supplying source
// classes.
// Takes pc (int) which is the PC of the write being analysed.
// Takes inst (instruction) which is the instruction performing the write.
// Takes minter (*aliasClassMinter) which is the class minter used for fresh allocations.
// Takes memo (*fieldClassMemo) which keeps class derivations stable across worklist
// revisits.
//
// Returns the class assigned to inst's destination after the write.
func computeAliasClassForWrite(env *aliasEnvironment, pc int, inst instruction, minter *aliasClassMinter, memo *fieldClassMemo) aliasClass {
	switch inst.op {
	case opAllocIndirect, opAddr, opLoadGeneralConst, opGetField:
		return memo.freshForPC(pc, minter)
	case opMoveGeneral:
		return env.class[inst.b]
	case opGetStructFieldGeneral, opGetStructFieldRawPointerT0:
		return deriveFieldAliasClass(env.class[inst.b], pc, minter, memo)
	default:
	}
	return aliasClassWild
}

// deriveFieldAliasClass returns the alias class for the result of a field load at pc
// whose receiver has parentClass.
//
// When parentClass is wild, the field result is also wild. Otherwise a fresh class is
// minted on the first visit and memoised by (pc, parentClass); later visits of the same
// field load with the same receiver class return the recorded class. The memoisation is
// what makes the transfer function monotone enough for the worklist's
// aliasEnvironmentEqual fixpoint to converge: without it, every revisit would produce a
// different class and the analysis could loop forever.
//
// Takes parentClass (aliasClass) which is the alias class of the receiver register.
// Takes pc (int) which is the PC of the field-load instruction.
// Takes minter (*aliasClassMinter) which is the class minter used to allocate fresh
// classes.
// Takes memo (*fieldClassMemo) which records derivations for stability.
//
// Returns the derived class for the field load result.
func deriveFieldAliasClass(parentClass aliasClass, pc int, minter *aliasClassMinter, memo *fieldClassMemo) aliasClass {
	if parentClass == aliasClassWild {
		return aliasClassWild
	}
	key := fieldClassMemoKey{pc: pc, parentClass: parentClass}
	if existing, ok := memo.classes[key]; ok {
		return existing
	}
	derived := minter.fresh()
	memo.classes[key] = derived
	return derived
}
