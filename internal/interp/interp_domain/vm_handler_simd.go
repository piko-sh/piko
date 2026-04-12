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

	"piko.sh/piko/internal/vectormaths"
)

// simdReadSliceWithCount resolves a typed-bank slice and a count.
//
// Applies the bounds check the scalar loop would have done element-by-element. count >
// len(slice) or count < 0 raises an interpreted panic matching the scalar out-of-range
// semantics. count is clamped to a non-negative integer before being returned so the
// caller can `slice[:count]` without risking a host panic.
//
// Takes vm (*VM).
// Takes registers (*Registers).
// Takes sliceFloatRegister (uint8) which is the typed slicesFloat index.
// Takes countIntRegister (uint8) which is the int-bank register holding the loop's
// iteration count (from len(slice) or a const).
//
// Returns the slice, the validated count, and ok=true; nil/0/false after raising an
// interpreted panic.
func simdReadSliceWithCount(vm *VM, registers *Registers, sliceFloatRegister, countIntRegister uint8) ([]float64, int, bool) {
	slice := registers.slicesFloat[sliceFloatRegister]
	count := int(registers.ints[countIntRegister])
	if count < 0 || count > len(slice) {
		raiseNativePanicAsInterpreted(vm, newRuntimePanicError("simd kernel: count %d out of range [0, %d]", count, len(slice)))
		return nil, 0, false
	}
	return slice, count, true
}

// simdReadDualSliceWithCount resolves two typed-bank float64 slices and validates a count
// against both. Bounds-checks mirror what the original scalar loop would have done at
// index `count-1`.
//
// Takes vm (*VM).
// Takes registers (*Registers).
// Takes firstRegister (uint8) which is the first slicesFloat index.
// Takes secondRegister (uint8) which is the second slicesFloat index.
// Takes countIntRegister (uint8) which is the int-bank count.
//
// Returns both slices, the validated count, and ok=true; nil/nil/0/ false after raising
// an interpreted panic.
func simdReadDualSliceWithCount(vm *VM, registers *Registers, firstRegister, secondRegister, countIntRegister uint8) (firstSlice, secondSlice []float64, count int, ok bool) {
	firstSlice = registers.slicesFloat[firstRegister]
	secondSlice = registers.slicesFloat[secondRegister]
	count = int(registers.ints[countIntRegister])
	if count < 0 || count > len(firstSlice) || count > len(secondSlice) {
		raiseNativePanicAsInterpreted(vm, newRuntimePanicError("simd kernel: count %d out of range (len(a)=%d, len(b)=%d)", count, len(firstSlice), len(secondSlice)))
		return nil, nil, 0, false
	}
	return firstSlice, secondSlice, count, true
}

// simdElementwiseFloat64 applies binaryOp pairwise to a[:count] and b[:count], writing
// the result into dst[:count]. Used by the elementwise sub/mul opcodes whose binaryOp is
// not (yet) a vectormaths kernel.
//
// Takes vm (*VM) which receives the interpreted panic on failure.
// Takes registers (*Registers) which is the active register file.
// Takes destinationRegister (uint8) which is the slicesFloat index of the destination
// slice.
// Takes firstRegister, secondRegister (uint8) which are the operand slicesFloat indices.
// Takes countIntRegister (uint8) which is the int-bank register holding the loop's
// iteration count.
// Takes binaryOp (func(float64, float64) float64) which is the element-wise operation to
// apply.
//
// Returns ok=true on success; ok=false after raising an interpreted panic on any failure.
func simdElementwiseFloat64(vm *VM, registers *Registers, destinationRegister, firstRegister, secondRegister, countIntRegister uint8, binaryOp func(float64, float64) float64) bool {
	plan := simdReadElementwiseFloat64(vm, registers, destinationRegister, firstRegister, secondRegister, countIntRegister)
	if !plan.ok {
		return false
	}
	bdst := plan.dst[:plan.count]
	ba := plan.a[:plan.count]
	bb := plan.b[:plan.count]
	for i := range bdst {
		bdst[i] = binaryOp(ba[i], bb[i])
	}
	return true
}

// simdReadElementwiseFloat64 resolves the destination + two operand typed-bank float64
// slices and validates a shared count against all three lengths. Shared between the
// elementwise dispatchers (add via vectormaths, sub/mul via simdElementwiseFloat64) so
// the bounds-checking logic lives in one place.
//
// Takes vm (*VM).
// Takes registers (*Registers).
// Takes destinationRegister (uint8) which is the slicesFloat index of the destination
// slice.
// Takes firstRegister, secondRegister (uint8) which are the operand slicesFloat indices.
// Takes countIntRegister (uint8) which is the int-bank count.
//
// Returns an elementwiseFloat64Plan; the ok field is false after raising an interpreted
// panic for an out-of-range count.
func simdReadElementwiseFloat64(vm *VM, registers *Registers, destinationRegister, firstRegister, secondRegister, countIntRegister uint8) elementwiseFloat64Plan {
	dst := registers.slicesFloat[destinationRegister]
	a := registers.slicesFloat[firstRegister]
	b := registers.slicesFloat[secondRegister]
	count := int(registers.ints[countIntRegister])
	if count < 0 || count > len(dst) || count > len(a) || count > len(b) {
		raiseNativePanicAsInterpreted(vm, newRuntimePanicError("simd kernel: count %d out of range (len(dst)=%d, len(a)=%d, len(b)=%d)", count, len(dst), len(a), len(b)))
		return elementwiseFloat64Plan{}
	}
	return elementwiseFloat64Plan{dst: dst, count: count, a: a, b: b, ok: true}
}

// elementwiseFloat64Plan bundles the resolved destination plus two operand slices and the
// validated element count for a SIMD elementwise float64 kernel. The ok field is false
// when the count validation failed and an interpreted panic was raised.
type elementwiseFloat64Plan struct {
	// dst is the destination typed-slice float64 bank.
	dst []float64

	// a is the first operand typed-slice float64 bank.
	a []float64

	// b is the second operand typed-slice float64 bank.
	b []float64

	// count is the validated element count to operate on.
	count int

	// ok reports whether the plan is valid; false when validation raised an interpreted
	// panic.
	ok bool
}

// simdDotProductFloat64 computes the scalar dot product of two equal-length []float64
// slices, dispatching to the architecture-tuned kernel in internal/vectormaths (SSE2 /
// AVX2 on amd64, scalar fallback elsewhere). The Go-side helper gives SIMD dispatch a
// single named entry point regardless of how vectormaths picks its implementation.
//
// Takes a, b ([]float64) which are the equal-length operand slices.
//
// Returns the scalar sum sum(a[i]*b[i]).
func simdDotProductFloat64(a, b []float64) float64 {
	return vectormaths.DotF64(a, b)
}

// simdSumFloat64 computes the scalar sum of a []float64, dispatching to the
// architecture-tuned kernel in internal/vectormaths.
//
// Takes a ([]float64) which is the operand slice.
//
// Returns sum(a[i]).
func simdSumFloat64(a []float64) float64 {
	return vectormaths.SumF64(a)
}

// simdEuclideanDistanceSquaredFloat64 computes sum((a[i]-b[i])^2) for two equal-length
// []float64 slices.
//
// Takes a, b ([]float64) which are the equal-length operand slices.
//
// Returns the scalar squared Euclidean distance.
func simdEuclideanDistanceSquaredFloat64(a, b []float64) float64 {
	sum := 0.0
	for i := range a {
		difference := a[i] - b[i]
		sum += difference * difference
	}
	return sum
}

// simdMaxFloat64 reduces a []float64 to its maximum element. An empty slice produces
// math.Inf(-1) so chained reductions (max-of-maxes across partitions) remain associative.
//
// Takes a ([]float64) which is the operand slice.
//
// Returns the maximum element or math.Inf(-1) when empty.
func simdMaxFloat64(a []float64) float64 {
	if len(a) == 0 {
		return math.Inf(-1)
	}
	maximum := a[0]
	for _, value := range a[1:] {
		if value > maximum {
			maximum = value
		}
	}
	return maximum
}

// simdMinFloat64 reduces a []float64 to its minimum element. An empty slice produces
// math.Inf(+1) so chained reductions remain associative.
//
// Takes a ([]float64) which is the operand slice.
//
// Returns the minimum element or math.Inf(+1) when empty.
func simdMinFloat64(a []float64) float64 {
	if len(a) == 0 {
		return math.Inf(1)
	}
	minimum := a[0]
	for _, value := range a[1:] {
		if value < minimum {
			minimum = value
		}
	}
	return minimum
}

// simdAxpyFloat64 computes y[i] += alpha * x[i] in place for two equal-length []float64
// slices and a scalar alpha (BLAS axpy).
//
// Takes y ([]float64) which is the accumulator slice, mutated in place.
// Takes alpha (float64) which is the scalar coefficient.
// Takes x ([]float64) which is the operand slice.
func simdAxpyFloat64(y []float64, alpha float64, x []float64) {
	for i := range y {
		y[i] += alpha * x[i]
	}
}

// simdScaleFloat64 multiplies every element of a []float64 by a scalar k in place,
// dispatching to the architecture-tuned kernel in internal/vectormaths.
//
// Takes slice ([]float64) which is mutated in place.
// Takes k (float64) which is the scalar coefficient.
func simdScaleFloat64(slice []float64, k float64) {
	vectormaths.ScaleF64(slice, k)
}

// simdFillFloat64 sets every element of a []float64 to value.
//
// Takes slice ([]float64) which is mutated in place.
// Takes value (float64) which is the fill value.
func simdFillFloat64(slice []float64, value float64) {
	for i := range slice {
		slice[i] = value
	}
}

// simdSubFloat64 and simdMulFloat64 are the binary operations passed to
// simdElementwiseFloat64. Kept as standalone named functions so the Go compiler can
// inline them into the element-wise loop body without indirect-call overhead.

// simdSubFloat64 returns a - b.
//
// Takes a, b (float64) which are the operands.
//
// Returns a - b.
func simdSubFloat64(a, b float64) float64 { return a - b }

// simdMulFloat64 returns a * b.
//
// Takes a, b (float64) which are the operands.
//
// Returns a * b.
func simdMulFloat64(a, b float64) float64 { return a * b }
