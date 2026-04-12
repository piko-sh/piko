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
	"piko.sh/piko/wdk/safeconv"
)

const (
	// kindByteDestShift is the bit shift placing the destination kind in the upper nibble of
	// the packed kindByte field. The source kind occupies the lower nibble; both kinds fit
	// in 4 bits because there are fewer than 16 register banks.
	kindByteDestShift = 4
)

// callArgCopyOp encodes the exact register-bank operation needed for one argument copy.
// Same-kind copies (the overwhelmingly common case) are seven distinct ops;
// box/unbox/cross-kind cases fall through to a generic runtime handler.
type callArgCopyOp uint8

const (
	// copyIntToInt copies ints[srcReg] -> ints[dstReg].
	copyIntToInt callArgCopyOp = iota

	// copyFloatToFloat copies floats[srcReg] -> floats[dstReg].
	copyFloatToFloat

	// copyStringToString copies strings[srcReg] -> strings[dstReg].
	copyStringToString

	// copyGeneralToGeneral copies general[srcReg] -> general[dstReg] via
	// valueCopyForBoundary (preserves boundary semantics).
	copyGeneralToGeneral

	// copyBoolToBool copies bools[srcReg] -> bools[dstReg].
	copyBoolToBool

	// copyUintToUint copies uints[srcReg] -> uints[dstReg].
	copyUintToUint

	// copyComplexToComplex copies complex[srcReg] -> complex[dstReg].
	copyComplexToComplex

	// copySliceIntToSliceInt copies slicesInt[srcReg] -> slicesInt[dstReg].
	copySliceIntToSliceInt

	// copySliceFloatToSliceFloat copies slicesFloat[srcReg] -> slicesFloat[dstReg].
	copySliceFloatToSliceFloat

	// copySliceStringToSliceString copies slicesString[srcReg] -> slicesString[dstReg].
	copySliceStringToSliceString

	// copySliceBoolToSliceBool copies slicesBool[srcReg] -> slicesBool[dstReg].
	copySliceBoolToSliceBool

	// copySliceUintToSliceUint copies slicesUint[srcReg] -> slicesUint[dstReg].
	copySliceUintToSliceUint

	// copySliceByteToSliceByte copies slicesByte[srcReg] -> slicesByte[dstReg].
	copySliceByteToSliceByte

	// copyBoxOrUnbox handles every cross-kind copy.
	//
	// Covers boxing scalar into general, unboxing general into scalar, and kind-mismatched
	// fallbacks. Routed to the runtime copyOneCallArgument shim. The compiler stores the
	// original srcKind in srcKindByte and the dstKind in dstKindByte for the runtime to
	// dispatch on.
	copyBoxOrUnbox
)

// callArgCopy is one entry of a per-callsite copy program.
//
// The compiler builds a []callArgCopy at emit time; runtime copyCallArgs walks the slice
// with a single switch and no helper function calls in the same-kind hot path. Sized to 4
// bytes so a 16-arg call site costs 64 B of program memory, small enough to fit alongside
// the per-callsite metadata.
type callArgCopy struct {
	// op is the encoded register-bank operation for this argument.
	op callArgCopyOp

	// sourceRegister is the index of the value in the source bank.
	sourceRegister uint8

	// destinationRegister is the index of the value in the destination bank.
	destinationRegister uint8

	// kindByte packs sourceKind (low nibble) and destinationKind (high nibble) for the
	// copyBoxOrUnbox slow-path dispatch. Unused for same-kind ops (op = copy*To* with
	// matching banks).
	kindByte uint8
}

// buildCallArgCopyProgram lays out a per-arg copy plan at compile time. Each entry
// resolves to a single bank-to-bank move at runtime with no kind-switching, kind-index
// tracking, or function calls in the same-kind common case.
//
// Takes arguments ([]varLocation) which are the caller-side argument locations from the
// call site's site.arguments.
// Takes parameterKinds ([]registerKind) which is the callee's expected per-parameter
// destination banks.
// Takes parameterRegisters ([]uint8) which are the callee's pre-allocated per-parameter
// destination slots; a short or empty slice falls back to sequential per-bank allocation.
//
// Returns a []callArgCopy of length min(len(arguments), len(parameterKinds)). Variadic
// and mismatched-arity sites trim to the callee's arity.
func buildCallArgCopyProgram(arguments []varLocation, parameterKinds []registerKind, parameterRegisters []uint8) []callArgCopy {
	n := min(len(arguments), len(parameterKinds))
	if n == 0 {
		return nil
	}

	if len(parameterRegisters) != len(parameterKinds) {
		return nil
	}
	program := make([]callArgCopy, n)
	var kindIndex [NumRegisterKinds]int
	for i := range n {
		argumentLocation := arguments[i]
		destinationKind := parameterKinds[i]

		var destinationRegister int
		if i < len(parameterRegisters) {
			destinationRegister = int(parameterRegisters[i])
			kindIndex[destinationKind] = destinationRegister + 1
		} else {
			destinationRegister = kindIndex[destinationKind]
			kindIndex[destinationKind]++
		}
		program[i] = buildOneArgCopyEntry(argumentLocation, destinationKind, safeconv.MustIntToUint8(destinationRegister))
	}
	return program
}

// buildOneArgCopyEntry produces a single argCopy entry.
//
// Same-kind copies route through the per-bank opcode (one bytecode load avoids the boxing
// dispatch); mismatched-kind copies fall through to the generic box-or-unbox path that
// records both kinds in kindByte.
//
// Takes argumentLocation (varLocation) which holds the source register and kind.
// Takes destinationKind (registerKind) which is the callee parameter bank.
// Takes destinationRegister (uint8) which is the callee-side slot.
//
// Returns callArgCopy which describes the copy plan.
func buildOneArgCopyEntry(argumentLocation varLocation, destinationKind registerKind, destinationRegister uint8) callArgCopy {
	entry := callArgCopy{
		sourceRegister:      argumentLocation.register,
		destinationRegister: destinationRegister,
	}
	if argumentLocation.kind != destinationKind {
		entry.op = copyBoxOrUnbox
		entry.kindByte = uint8(argumentLocation.kind) | (uint8(destinationKind) << kindByteDestShift)
		return entry
	}
	op, ok := sameKindCopyOp(destinationKind)
	if !ok {
		entry.op = copyBoxOrUnbox
		entry.kindByte = uint8(argumentLocation.kind) | (uint8(destinationKind) << kindByteDestShift)
		return entry
	}
	entry.op = op
	return entry
}

// sameKindCopyOp returns the same-kind copy opcode for kind.
//
// Takes kind (registerKind) which selects the bank's copy opcode.
//
// Returns callArgCopyOp which is the matching per-bank opcode.
// Returns bool which is false when no per-bank opcode exists (e.g. upvalueKindAsPointer).
func sameKindCopyOp(kind registerKind) (callArgCopyOp, bool) {
	switch kind {
	case registerInt:
		return copyIntToInt, true
	case registerFloat:
		return copyFloatToFloat, true
	case registerString:
		return copyStringToString, true
	case registerGeneral:
		return copyGeneralToGeneral, true
	case registerBool:
		return copyBoolToBool, true
	case registerUint:
		return copyUintToUint, true
	case registerComplex:
		return copyComplexToComplex, true
	case registerSliceInt:
		return copySliceIntToSliceInt, true
	case registerSliceFloat:
		return copySliceFloatToSliceFloat, true
	case registerSliceString:
		return copySliceStringToSliceString, true
	case registerSliceBool:
		return copySliceBoolToSliceBool, true
	case registerSliceUint:
		return copySliceUintToSliceUint, true
	case registerSliceByte:
		return copySliceByteToSliceByte, true
	case upvalueKindAsPointer:
		return 0, false
	}
	return 0, false
}

// detectTailCallArgsAlias reports whether a tail call's argument copy program would
// suffer from source-destination aliasing if executed in sequential order against a
// single shared register file (the tail-reuse case, where the caller's register windows
// ARE the callee's register windows).
//
// Aliasing occurs when arg i's destination position equals some later arg j's source
// position in the same bank. Writing dst_i first overwrites src_j before arg j is read,
// so the copy of arg j would observe the freshly-written value rather than the original.
//
// Takes arguments ([]varLocation) which are the caller-side argument locations from the
// call site's site.arguments.
// Takes parameterKinds ([]registerKind) which is the callee's expected per-parameter
// destination banks.
//
// Returns true when at least one alias exists, requiring the runtime to fall back to the
// snapshot-then-place pattern.
// Returns false when the sequential copy can run directly against the shared register
// file (the common case for accumulator-passing tail calls such as fib_tail(n-1, acc+1),
// and any tail call whose destination positions sit at the low end of the parameter banks
// while sources sit at compiler-allocated temporary positions further up).
func detectTailCallArgsAlias(arguments []varLocation, parameterKinds []registerKind) bool {
	n := min(len(arguments), len(parameterKinds))
	var kindIndex [NumRegisterKinds]int
	destinationPositions := make([]struct {
		kind registerKind
		reg  int
	}, n)
	for i := range n {
		destinationKind := parameterKinds[i]
		destinationPositions[i].kind = destinationKind
		destinationPositions[i].reg = kindIndex[destinationKind]
		kindIndex[destinationKind]++
	}
	for i := range n {
		destination := destinationPositions[i]
		for j := i + 1; j < n; j++ {
			sourceLocation := arguments[j]
			if sourceLocation.kind == destination.kind && int(sourceLocation.register) == destination.reg {
				return true
			}
		}
	}
	return false
}
