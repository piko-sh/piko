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
	"reflect"
	"strconv"

	"piko.sh/piko/internal/vectormaths"
)

// handleFlatSubOpMathSin sets floats[B] = math.Sin(floats[C]).
//
// Takes registers (*Registers) which holds the typed register banks.
// Takes instr (instruction) which provides operand indices.
//
// Returns opResult which signals continued dispatch.
func handleFlatSubOpMathSin(_ *VM, _ *callFrame, registers *Registers, instr instruction) opResult {
	registers.floats[instr.b] = math.Sin(registers.floats[instr.c])
	return opContinue
}

// handleFlatSubOpMathCos sets floats[B] = math.Cos(floats[C]).
//
// Takes registers (*Registers) which holds the typed register banks.
// Takes instr (instruction) which provides operand indices.
//
// Returns opResult which signals continued dispatch.
func handleFlatSubOpMathCos(_ *VM, _ *callFrame, registers *Registers, instr instruction) opResult {
	registers.floats[instr.b] = math.Cos(registers.floats[instr.c])
	return opContinue
}

// handleFlatSubOpMathExp sets floats[B] = math.Exp(floats[C]).
//
// Takes registers (*Registers) which holds the typed register banks.
// Takes instr (instruction) which provides operand indices.
//
// Returns opResult which signals continued dispatch.
func handleFlatSubOpMathExp(_ *VM, _ *callFrame, registers *Registers, instr instruction) opResult {
	registers.floats[instr.b] = math.Exp(registers.floats[instr.c])
	return opContinue
}

// handleFlatSubOpMathTan sets floats[B] = math.Tan(floats[C]).
//
// Takes registers (*Registers) which holds the typed register banks.
// Takes instr (instruction) which provides operand indices.
//
// Returns opResult which signals continued dispatch.
func handleFlatSubOpMathTan(_ *VM, _ *callFrame, registers *Registers, instr instruction) opResult {
	registers.floats[instr.b] = math.Tan(registers.floats[instr.c])
	return opContinue
}

// handleFlatSubOpMathMod sets floats[B] = math.Mod(floats[C], floats[ext.a]).
//
// Takes frame (*callFrame) which supplies the extension word.
// Takes registers (*Registers) which holds the typed register banks.
// Takes instr (instruction) which provides operand indices.
//
// Returns opResult which signals continued dispatch.
func handleFlatSubOpMathMod(_ *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	extension := frame.function.body[frame.programCounter]
	frame.programCounter++
	registers.floats[instr.b] = math.Mod(registers.floats[instr.c], registers.floats[extension.a])
	return opContinue
}

// handleFlatSubOpStrconvFormatBool sets strings[B] = strconv.FormatBool(bools[C]).
//
// Takes registers (*Registers) which holds the typed register banks.
// Takes instr (instruction) which provides operand indices.
//
// Returns opResult which signals continued dispatch.
func handleFlatSubOpStrconvFormatBool(_ *VM, _ *callFrame, registers *Registers, instr instruction) opResult {
	registers.strings[instr.b] = strconv.FormatBool(registers.bools[instr.c])
	return opContinue
}

// handleFlatSubOpStrconvFormatInt formats ints[C] with the supplied base.
//
// Takes vm (*VM) which owns the arena and panic state.
// Takes frame (*callFrame) which supplies the extension word.
// Takes registers (*Registers) which holds the typed register banks.
// Takes instr (instruction) which provides operand indices.
//
// Returns opResult which signals continued dispatch or a panic.
func handleFlatSubOpStrconvFormatInt(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	extension := frame.function.body[frame.programCounter]
	frame.programCounter++
	base := int(registers.ints[extension.a])
	if base < minStrconvBase || base > maxStrconvBase {
		vm.evalError = newRuntimePanicError("strconv: illegal AppendInt/FormatInt base %d", base)
		return opPanicError
	}
	registers.strings[instr.b] = arenaFormatIntString(vm.arena, registers.ints[instr.c], base)
	return opContinue
}

// handleFlatSubOpStrconvItoa sets strings[B] = strconv.Itoa(ints[C]).
//
// Takes vm (*VM) which owns the arena allocator.
// Takes registers (*Registers) which holds the typed register banks.
// Takes instr (instruction) which provides operand indices.
//
// Returns opResult which signals continued dispatch.
func handleFlatSubOpStrconvItoa(vm *VM, _ *callFrame, registers *Registers, instr instruction) opResult {
	registers.strings[instr.b] = arenaItoaString(vm.arena, registers.ints[instr.c])
	return opContinue
}

// handleFlatSubOpRealComplex sets floats[B] = real(complex[C]).
//
// Takes registers (*Registers) which holds the typed register banks.
// Takes instr (instruction) which provides operand indices.
//
// Returns opResult which signals continued dispatch.
func handleFlatSubOpRealComplex(_ *VM, _ *callFrame, registers *Registers, instr instruction) opResult {
	registers.floats[instr.b] = real(registers.complex[instr.c])
	return opContinue
}

// handleFlatSubOpImagComplex sets floats[B] = imag(complex[C]).
//
// Takes registers (*Registers) which holds the typed register banks.
// Takes instr (instruction) which provides operand indices.
//
// Returns opResult which signals continued dispatch.
func handleFlatSubOpImagComplex(_ *VM, _ *callFrame, registers *Registers, instr instruction) opResult {
	registers.floats[instr.b] = imag(registers.complex[instr.c])
	return opContinue
}

// handleFlatSubOpBytesToString sets strings[B] = string(general[C].Bytes()).
//
// Takes vm (*VM) which owns the arena allocator.
// Takes registers (*Registers) which holds the typed register banks.
// Takes instr (instruction) which provides operand indices.
//
// Returns opResult which signals continued dispatch.
func handleFlatSubOpBytesToString(vm *VM, _ *callFrame, registers *Registers, instr instruction) opResult {
	registers.strings[instr.b] = arenaBytesToString(vm.arena, registers.general[instr.c].Bytes())
	return opContinue
}

// handleFlatSubOpCap sets ints[B] = cap(general[C]).
//
// Takes registers (*Registers) which holds the typed register banks.
// Takes instr (instruction) which provides operand indices.
//
// Returns opResult which signals continued dispatch.
func handleFlatSubOpCap(_ *VM, _ *callFrame, registers *Registers, instr instruction) opResult {
	registers.ints[instr.b] = collectionLengthOrCap(registers.general[instr.c], reflect.Value.Cap)
	return opContinue
}

// handleFlatSubOpNegComplex sets complex[B] = -complex[C].
//
// Takes registers (*Registers) which holds the typed register banks.
// Takes instr (instruction) which provides operand indices.
//
// Returns opResult which signals continued dispatch.
func handleFlatSubOpNegComplex(_ *VM, _ *callFrame, registers *Registers, instr instruction) opResult {
	registers.complex[instr.b] = -registers.complex[instr.c]
	return opContinue
}

// handleFlatSubOpMoveComplex sets complex[B] = complex[C].
//
// Takes registers (*Registers) which holds the typed register banks.
// Takes instr (instruction) which provides operand indices.
//
// Returns opResult which signals continued dispatch.
func handleFlatSubOpMoveComplex(_ *VM, _ *callFrame, registers *Registers, instr instruction) opResult {
	registers.complex[instr.b] = registers.complex[instr.c]
	return opContinue
}

// handleFlatSubOpLoadIntConstSmall sets ints[B] = int64(C).
//
// Shared with subOpLoadBool which also writes a 0/1 int.
//
// Takes registers (*Registers) which holds the typed register banks.
// Takes instr (instruction) which provides operand indices.
//
// Returns opResult which signals continued dispatch.
func handleFlatSubOpLoadIntConstSmall(_ *VM, _ *callFrame, registers *Registers, instr instruction) opResult {
	registers.ints[instr.b] = int64(instr.c)
	return opContinue
}

// handleFlatSubOpLoadUintConstSmall sets uints[B] = uint64(C).
//
// Takes registers (*Registers) which holds the typed register banks.
// Takes instr (instruction) which provides operand indices.
//
// Returns opResult which signals continued dispatch.
func handleFlatSubOpLoadUintConstSmall(_ *VM, _ *callFrame, registers *Registers, instr instruction) opResult {
	registers.uints[instr.b] = uint64(instr.c)
	return opContinue
}

// SIMD sub-op handlers run their kernels and return opResult directly; reaching the
// handler implies the dispatch matched.

// handleFlatSubOpSimdDotProductFloat64 accumulates the bounded dot product.
//
// floats[B] += dotProduct(slicesFloat[C], slicesFloat[ext.a])[:ints[ext.b]].
//
// Takes vm (*VM) which owns panic state.
// Takes frame (*callFrame) which supplies the extension word.
// Takes registers (*Registers) which holds the typed register banks.
// Takes instr (instruction) which provides operand indices.
//
// Returns opResult which signals continued dispatch or a panic.
func handleFlatSubOpSimdDotProductFloat64(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	extension := frame.function.body[frame.programCounter]
	frame.programCounter++
	a, b, count, ok := simdReadDualSliceWithCount(vm, registers, instr.c, extension.a, extension.b)
	if !ok {
		return opPanicError
	}
	registers.floats[instr.b] += simdDotProductFloat64(a[:count], b[:count])
	return opContinue
}

// handleFlatSubOpSimdSumSliceFloat64 accumulates the bounded slice sum.
//
// floats[B] += sum(slicesFloat[C])[:ints[ext.a]].
//
// Takes vm (*VM) which owns panic state.
// Takes frame (*callFrame) which supplies the extension word.
// Takes registers (*Registers) which holds the typed register banks.
// Takes instr (instruction) which provides operand indices.
//
// Returns opResult which signals continued dispatch or a panic.
func handleFlatSubOpSimdSumSliceFloat64(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	extension := frame.function.body[frame.programCounter]
	frame.programCounter++
	slice, count, ok := simdReadSliceWithCount(vm, registers, instr.c, extension.a)
	if !ok {
		return opPanicError
	}
	registers.floats[instr.b] += simdSumFloat64(slice[:count])
	return opContinue
}

// handleFlatSubOpSimdNormSquaredFloat64 accumulates the squared norm.
//
// floats[B] += dot(slice, slice) over the bounded slice.
//
// Takes vm (*VM) which owns panic state.
// Takes frame (*callFrame) which supplies the extension word.
// Takes registers (*Registers) which holds the typed register banks.
// Takes instr (instruction) which provides operand indices.
//
// Returns opResult which signals continued dispatch or a panic.
func handleFlatSubOpSimdNormSquaredFloat64(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	extension := frame.function.body[frame.programCounter]
	frame.programCounter++
	slice, count, ok := simdReadSliceWithCount(vm, registers, instr.c, extension.a)
	if !ok {
		return opPanicError
	}
	bounded := slice[:count]
	registers.floats[instr.b] += simdDotProductFloat64(bounded, bounded)
	return opContinue
}

// handleFlatSubOpSimdEuclideanDistanceSquaredFloat64 accumulates the squared distance.
//
// floats[B] += euclideanDistanceSquared(slicesFloat[C], slicesFloat[ext.a]).
//
// Takes vm (*VM) which owns panic state.
// Takes frame (*callFrame) which supplies the extension word.
// Takes registers (*Registers) which holds the typed register banks.
// Takes instr (instruction) which provides operand indices.
//
// Returns opResult which signals continued dispatch or a panic.
func handleFlatSubOpSimdEuclideanDistanceSquaredFloat64(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	extension := frame.function.body[frame.programCounter]
	frame.programCounter++
	a, b, count, ok := simdReadDualSliceWithCount(vm, registers, instr.c, extension.a, extension.b)
	if !ok {
		return opPanicError
	}
	registers.floats[instr.b] += simdEuclideanDistanceSquaredFloat64(a[:count], b[:count])
	return opContinue
}

// handleFlatSubOpSimdMaxSliceFloat64 sets floats[B] to the running maximum.
//
// floats[B] = max(floats[B], max(slicesFloat[C])).
//
// Takes vm (*VM) which owns panic state.
// Takes frame (*callFrame) which supplies the extension word.
// Takes registers (*Registers) which holds the typed register banks.
// Takes instr (instruction) which provides operand indices.
//
// Returns opResult which signals continued dispatch or a panic.
func handleFlatSubOpSimdMaxSliceFloat64(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	extension := frame.function.body[frame.programCounter]
	frame.programCounter++
	slice, count, ok := simdReadSliceWithCount(vm, registers, instr.c, extension.a)
	if !ok {
		return opPanicError
	}
	registers.floats[instr.b] = math.Max(registers.floats[instr.b], simdMaxFloat64(slice[:count]))
	return opContinue
}

// handleFlatSubOpSimdMinSliceFloat64 sets floats[B] to the running minimum.
//
// floats[B] = min(floats[B], min(slicesFloat[C])).
//
// Takes vm (*VM) which owns panic state.
// Takes frame (*callFrame) which supplies the extension word.
// Takes registers (*Registers) which holds the typed register banks.
// Takes instr (instruction) which provides operand indices.
//
// Returns opResult which signals continued dispatch or a panic.
func handleFlatSubOpSimdMinSliceFloat64(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	extension := frame.function.body[frame.programCounter]
	frame.programCounter++
	slice, count, ok := simdReadSliceWithCount(vm, registers, instr.c, extension.a)
	if !ok {
		return opPanicError
	}
	registers.floats[instr.b] = math.Min(registers.floats[instr.b], simdMinFloat64(slice[:count]))
	return opContinue
}

// handleFlatSubOpSimdAddSliceFloat64 vectorises dst[i] = a[i] + b[i].
//
// Takes vm (*VM) which owns panic state.
// Takes frame (*callFrame) which supplies the extension word.
// Takes registers (*Registers) which holds the typed register banks.
// Takes instr (instruction) which provides operand indices.
//
// Returns opResult which signals continued dispatch or a panic.
func handleFlatSubOpSimdAddSliceFloat64(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	extension := frame.function.body[frame.programCounter]
	frame.programCounter++
	plan := simdReadElementwiseFloat64(vm, registers, instr.b, instr.c, extension.a, extension.b)
	if !plan.ok {
		return opPanicError
	}
	vectormaths.AddF64(plan.dst[:plan.count], plan.a[:plan.count], plan.b[:plan.count])
	return opContinue
}

// handleFlatSubOpSimdSubSliceFloat64 vectorises dst[i] = a[i] - b[i].
//
// Takes vm (*VM) which owns panic state.
// Takes frame (*callFrame) which supplies the extension word.
// Takes registers (*Registers) which holds the typed register banks.
// Takes instr (instruction) which provides operand indices.
//
// Returns opResult which signals continued dispatch or a panic.
func handleFlatSubOpSimdSubSliceFloat64(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	extension := frame.function.body[frame.programCounter]
	frame.programCounter++
	ok := simdElementwiseFloat64(vm, registers, instr.b, instr.c, extension.a, extension.b, simdSubFloat64)
	if !ok {
		return opPanicError
	}
	return opContinue
}

// handleFlatSubOpSimdMulSliceFloat64 vectorises dst[i] = a[i] * b[i].
//
// Takes vm (*VM) which owns panic state.
// Takes frame (*callFrame) which supplies the extension word.
// Takes registers (*Registers) which holds the typed register banks.
// Takes instr (instruction) which provides operand indices.
//
// Returns opResult which signals continued dispatch or a panic.
func handleFlatSubOpSimdMulSliceFloat64(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	extension := frame.function.body[frame.programCounter]
	frame.programCounter++
	ok := simdElementwiseFloat64(vm, registers, instr.b, instr.c, extension.a, extension.b, simdMulFloat64)
	if !ok {
		return opPanicError
	}
	return opContinue
}

// handleFlatSubOpSimdAxpyFloat64 vectorises y[i] += alpha*x[i].
//
// Takes vm (*VM) which owns panic state.
// Takes frame (*callFrame) which supplies the extension word.
// Takes registers (*Registers) which holds the typed register banks.
// Takes instr (instruction) which provides operand indices.
//
// Returns opResult which signals continued dispatch or a panic.
func handleFlatSubOpSimdAxpyFloat64(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	extension := frame.function.body[frame.programCounter]
	frame.programCounter++
	y, x, count, ok := simdReadDualSliceWithCount(vm, registers, instr.b, instr.c, extension.b)
	if !ok {
		return opPanicError
	}
	simdAxpyFloat64(y[:count], registers.floats[extension.a], x[:count])
	return opContinue
}

// handleFlatSubOpSimdScaleSliceFloat64 vectorises s[i] *= alpha.
//
// Takes vm (*VM) which owns panic state.
// Takes frame (*callFrame) which supplies the extension word.
// Takes registers (*Registers) which holds the typed register banks.
// Takes instr (instruction) which provides operand indices.
//
// Returns opResult which signals continued dispatch or a panic.
func handleFlatSubOpSimdScaleSliceFloat64(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	extension := frame.function.body[frame.programCounter]
	frame.programCounter++
	slice, count, ok := simdReadSliceWithCount(vm, registers, instr.b, extension.a)
	if !ok {
		return opPanicError
	}
	simdScaleFloat64(slice[:count], registers.floats[instr.c])
	return opContinue
}

// handleFlatSubOpSimdClearSliceFloat64 zeros the bounded slice.
//
// Takes vm (*VM) which owns panic state.
// Takes frame (*callFrame) which supplies the extension word.
// Takes registers (*Registers) which holds the typed register banks.
// Takes instr (instruction) which provides operand indices.
//
// Returns opResult which signals continued dispatch or a panic.
func handleFlatSubOpSimdClearSliceFloat64(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	extension := frame.function.body[frame.programCounter]
	frame.programCounter++
	slice, count, ok := simdReadSliceWithCount(vm, registers, instr.b, extension.a)
	if !ok {
		return opPanicError
	}
	clear(slice[:count])
	return opContinue
}

// handleFlatSubOpSimdFillSliceFloat64 fills the bounded slice with floats[C].
//
// Takes vm (*VM) which owns panic state.
// Takes frame (*callFrame) which supplies the extension word.
// Takes registers (*Registers) which holds the typed register banks.
// Takes instr (instruction) which provides operand indices.
//
// Returns opResult which signals continued dispatch or a panic.
func handleFlatSubOpSimdFillSliceFloat64(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	extension := frame.function.body[frame.programCounter]
	frame.programCounter++
	slice, count, ok := simdReadSliceWithCount(vm, registers, instr.b, extension.a)
	if !ok {
		return opPanicError
	}
	simdFillFloat64(slice[:count], registers.floats[instr.c])
	return opContinue
}

// handleFlatSubOpAdoptGeneralToSlicesFloat unboxes a []float64 into the typed bank.
//
// Takes vm (*VM) which owns panic state.
// Takes registers (*Registers) which holds the typed register banks.
// Takes instr (instruction) which provides operand indices.
//
// Returns opResult which signals continued dispatch or a panic.
func handleFlatSubOpAdoptGeneralToSlicesFloat(vm *VM, _ *callFrame, registers *Registers, instr instruction) opResult {
	value := registers.general[instr.c]
	if !value.IsValid() {
		raiseNativePanicAsInterpreted(vm, newRuntimePanicError("simd adopt: source general register holds an invalid reflect.Value"))
		return opPanicError
	}
	slice, ok := reflect.TypeAssert[[]float64](value)
	if !ok {
		raiseNativePanicAsInterpreted(vm, newRuntimePanicError("simd adopt: expected []float64, got %s", value.Type()))
		return opPanicError
	}
	registers.slicesFloat[instr.b] = slice
	return opContinue
}
