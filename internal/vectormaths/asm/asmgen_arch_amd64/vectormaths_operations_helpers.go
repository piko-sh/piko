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

package asmgen_arch_amd64

import (
	"fmt"

	"piko.sh/piko/wdk/asmgen"
	"piko.sh/piko/wdk/asmgen/asmamd64"
)

// addF64SSEPrologue emits the argument-load and unroll-width guard for the SSE float64
// pointwise add kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes unrollWidth (int) which is the float64 elements processed per main-loop
// iteration.
// Takes firstTail (string) which is the label jumped to when input is short.
func addF64SSEPrologue(e *asmgen.Emitter, unrollWidth int, firstTail string) {
	inst(e, asmamd64.OperationMove64Bits, "dst_base+0(FP), DI")
	inst(e, asmamd64.OperationMove64Bits, "dst_len+8(FP), CX")
	inst(e, asmamd64.OperationMove64Bits, "a_base+24(FP), SI")
	inst(e, asmamd64.OperationMove64Bits, "b_base+48(FP), BX")
	e.Blank()
	inst(e, asmamd64.OperationCompare64Bits, fmt.Sprintf("CX, $%d", unrollWidth))
	inst(e, asmamd64.OperationJumpIfLessSigned, firstTail)
}

// addF64SSESlotBody emits one body slot of the SSE float64 pointwise add main loop. Slot
// 0 uses X0/X1 and slot 1 uses X2/X3 so the two slots have independent register
// dependencies and can issue in parallel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes off (int) which is the slot's byte offset and selects the register pair.
func addF64SSESlotBody(e *asmgen.Emitter, off int) {
	var lhs, rhs string
	if off == 0 {
		lhs, rhs = "X0", "X1"
	} else {
		lhs, rhs = "X2", "X3"
	}
	inst(e, asmamd64.OperationMoveUnalignedPackedDoubles, fmt.Sprintf("%d(SI), %s", off, lhs))
	inst(e, asmamd64.OperationMoveUnalignedPackedDoubles, fmt.Sprintf("%d(BX), %s", off, rhs))
	inst(e, asmamd64.OperationAddPackedDoubles, fmt.Sprintf("%s, %s", rhs, lhs))
	inst(e, asmamd64.OperationMoveUnalignedPackedDoubles, fmt.Sprintf("%s, %d(DI)", lhs, off))
}

// addF64SSELoopFooter emits the pointer-advance, counter decrement, and back branch for
// the SSE float64 pointwise add main loop.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes byteStride (int) which is the bytes per unrolled iteration.
func addF64SSELoopFooter(e *asmgen.Emitter, byteStride int) {
	unrollWidth := byteStride / 8
	inst(e, asmamd64.OperationAdd64Bits, fmt.Sprintf("$%d, SI", byteStride))
	inst(e, asmamd64.OperationAdd64Bits, fmt.Sprintf("$%d, BX", byteStride))
	inst(e, asmamd64.OperationAdd64Bits, fmt.Sprintf("$%d, DI", byteStride))
	inst(e, asmamd64.OperationSubtract64Bits, fmt.Sprintf("$%d, CX", unrollWidth))
	inst(e, asmamd64.OperationCompare64Bits, fmt.Sprintf("CX, $%d", unrollWidth))
	inst(e, asmamd64.OperationJumpIfGreaterOrEqualSigned, fmt.Sprintf("addf64sse_loop%d", unrollWidth))
}

// addF64SSETail2 emits the 2-wide SSE straight-line tail for the SSE float64 pointwise
// add kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func addF64SSETail2(e *asmgen.Emitter) {
	e.Label("addf64sse_tail2")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $2")
	inst(e, asmamd64.OperationJumpIfLessSigned, "addf64sse_tail")
	e.Blank()
	inst(e, asmamd64.OperationMoveUnalignedPackedDoubles, "(SI), X0")
	inst(e, asmamd64.OperationMoveUnalignedPackedDoubles, "(BX), X1")
	inst(e, asmamd64.OperationAddPackedDoubles, "X1, X0")
	inst(e, asmamd64.OperationMoveUnalignedPackedDoubles, "X0, (DI)")
	inst(e, asmamd64.OperationAdd64Bits, "$16, SI")
	inst(e, asmamd64.OperationAdd64Bits, "$16, BX")
	inst(e, asmamd64.OperationAdd64Bits, "$16, DI")
	inst(e, asmamd64.OperationSubtract64Bits, "$2, CX")
	e.Blank()
}

// addF64SSEScalarTail emits the scalar drain for the SSE float64 pointwise add kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func addF64SSEScalarTail(e *asmgen.Emitter) {
	e.Label("addf64sse_tail")
	inst(e, asmamd64.OperationTest64Bits, "CX, CX")
	inst(e, asmamd64.OperationJumpIfZero, "addf64sse_done")
	e.Blank()
	e.Label("addf64sse_tail_loop")
	inst(e, asmamd64.OperationMoveScalarDouble, "(SI), X0")
	inst(e, asmamd64.OperationMoveScalarDouble, "(BX), X1")
	inst(e, asmamd64.OperationAddScalarDouble, "X1, X0")
	inst(e, asmamd64.OperationMoveScalarDouble, "X0, (DI)")
	inst(e, asmamd64.OperationAdd64Bits, "$8, SI")
	inst(e, asmamd64.OperationAdd64Bits, "$8, BX")
	inst(e, asmamd64.OperationAdd64Bits, "$8, DI")
	inst(e, asmamd64.OperationDecrement64Bits, "CX")
	inst(e, asmamd64.OperationJumpIfNotZero, "addf64sse_tail_loop")
	e.Blank()
}

// addF64SSEReturn emits the done label and final RET for the SSE float64 pointwise add
// kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func addF64SSEReturn(e *asmgen.Emitter) {
	e.Label("addf64sse_done")
	e.Instruction(asmamd64.OperationReturn)
}

// addF64AVX2Prologue emits the argument-load and unroll-width guard for the AVX2 float64
// pointwise add kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes unrollWidth (int) which is the float64 elements processed per main-loop
// iteration.
// Takes firstTail (string) which is the label jumped to when input is short.
func addF64AVX2Prologue(e *asmgen.Emitter, unrollWidth int, firstTail string) {
	inst(e, asmamd64.OperationMove64Bits, "dst_base+0(FP), DI")
	inst(e, asmamd64.OperationMove64Bits, "a_base+24(FP), SI")
	inst(e, asmamd64.OperationMove64Bits, "a_len+32(FP), CX")
	inst(e, asmamd64.OperationMove64Bits, "b_base+48(FP), BX")
	e.Blank()
	inst(e, asmamd64.OperationCompare64Bits, fmt.Sprintf("CX, $%d", unrollWidth))
	inst(e, asmamd64.OperationJumpIfLessSigned, firstTail)
}

// addF64AVX2SlotBody emits one body slot of the AVX2 float64 pointwise add main loop
// using VADDPD's memory-operand form.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes off (int) which is the byte offset of this slot within the unrolled iteration.
func addF64AVX2SlotBody(e *asmgen.Emitter, off int) {
	inst(e, asmamd64.OperationVexMoveUnalignedPackedDoubles, fmt.Sprintf("%d(SI), Y0", off))
	inst(e, asmamd64.OperationVexAddPackedDoubles, fmt.Sprintf("%d(BX), Y0, Y0", off))
	inst(e, asmamd64.OperationVexMoveUnalignedPackedDoubles, fmt.Sprintf("Y0, %d(DI)", off))
}

// addF64AVX2LoopFooter emits the pointer-advance, counter decrement, and back branch for
// the AVX2 float64 pointwise add main loop.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes byteStride (int) which is the bytes per unrolled iteration.
func addF64AVX2LoopFooter(e *asmgen.Emitter, byteStride int) {
	unrollWidth := byteStride / 8
	inst(e, asmamd64.OperationAdd64Bits, fmt.Sprintf("$%d, SI", byteStride))
	inst(e, asmamd64.OperationAdd64Bits, fmt.Sprintf("$%d, BX", byteStride))
	inst(e, asmamd64.OperationAdd64Bits, fmt.Sprintf("$%d, DI", byteStride))
	inst(e, asmamd64.OperationSubtract64Bits, fmt.Sprintf("$%d, CX", unrollWidth))
	inst(e, asmamd64.OperationCompare64Bits, fmt.Sprintf("CX, $%d", unrollWidth))
	inst(e, asmamd64.OperationJumpIfGreaterOrEqualSigned, fmt.Sprintf("addf64avx_loop%d", unrollWidth))
}

// addF64AVX2Tail4 emits the 4-wide AVX2 loop tail for the AVX2 float64 pointwise add
// kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func addF64AVX2Tail4(e *asmgen.Emitter) {
	e.Label("addf64avx_tail4")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $4")
	inst(e, asmamd64.OperationJumpIfLessSigned, "addf64avx_tail2")
	e.Blank()
	e.Label("addf64avx_loop4")
	inst(e, asmamd64.OperationVexMoveUnalignedPackedDoubles, "(SI), Y0")
	inst(e, asmamd64.OperationVexAddPackedDoubles, "(BX), Y0, Y0")
	inst(e, asmamd64.OperationVexMoveUnalignedPackedDoubles, "Y0, (DI)")
	inst(e, asmamd64.OperationAdd64Bits, "$32, SI")
	inst(e, asmamd64.OperationAdd64Bits, "$32, BX")
	inst(e, asmamd64.OperationAdd64Bits, "$32, DI")
	inst(e, asmamd64.OperationSubtract64Bits, "$4, CX")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $4")
	inst(e, asmamd64.OperationJumpIfGreaterOrEqualSigned, "addf64avx_loop4")
	e.Blank()
}

// addF64AVX2Tail2 emits the 2-wide SSE straight-line tail (no loop) for the AVX2 float64
// pointwise add kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func addF64AVX2Tail2(e *asmgen.Emitter) {
	e.Label("addf64avx_tail2")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $2")
	inst(e, asmamd64.OperationJumpIfLessSigned, "addf64avx_tail1")
	e.Blank()
	inst(e, asmamd64.OperationMoveUnalignedPackedDoubles, "(SI), X0")
	inst(e, asmamd64.OperationMoveUnalignedPackedDoubles, "(BX), X1")
	inst(e, asmamd64.OperationAddPackedDoubles, "X1, X0")
	inst(e, asmamd64.OperationMoveUnalignedPackedDoubles, "X0, (DI)")
	inst(e, asmamd64.OperationAdd64Bits, "$16, SI")
	inst(e, asmamd64.OperationAdd64Bits, "$16, BX")
	inst(e, asmamd64.OperationAdd64Bits, "$16, DI")
	inst(e, asmamd64.OperationSubtract64Bits, "$2, CX")
	e.Blank()
}

// addF64AVX2ScalarTail emits the scalar drain for the AVX2 float64 pointwise add kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func addF64AVX2ScalarTail(e *asmgen.Emitter) {
	e.Label("addf64avx_tail1")
	inst(e, asmamd64.OperationTest64Bits, "CX, CX")
	inst(e, asmamd64.OperationJumpIfZero, "addf64avx_done")
	e.Blank()
	e.Label("addf64avx_tail_loop")
	inst(e, asmamd64.OperationVexMoveScalarDouble, "(SI), X0")
	inst(e, asmamd64.OperationVexAddScalarDouble, "(BX), X0, X0")
	inst(e, asmamd64.OperationVexMoveScalarDouble, "X0, (DI)")
	inst(e, asmamd64.OperationAdd64Bits, "$8, SI")
	inst(e, asmamd64.OperationAdd64Bits, "$8, BX")
	inst(e, asmamd64.OperationAdd64Bits, "$8, DI")
	inst(e, asmamd64.OperationDecrement64Bits, "CX")
	inst(e, asmamd64.OperationJumpIfNotZero, "addf64avx_tail_loop")
	e.Blank()
}

// addF64AVX2Return emits the done label, VZEROUPPER, and final RET for the AVX2 float64
// pointwise add kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func addF64AVX2Return(e *asmgen.Emitter) {
	e.Label("addf64avx_done")
	e.Instruction(asmamd64.OperationVexZeroUpper)
	e.Instruction(asmamd64.OperationReturn)
}

// sumF64SSEPrologue emits the argument-load, accumulator zeroing, and unroll-width guard
// for the SSE float64 sum kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes accumulators ([]string) which names the XMM registers zeroed as packed
// accumulators.
// Takes unrollWidth (int) which is the float64 elements processed per main-loop
// iteration.
// Takes narrowerTailFullLabel (string) which is the label jumped to when input is short.
func sumF64SSEPrologue(e *asmgen.Emitter, accumulators []string, unrollWidth int, narrowerTailFullLabel string) {
	inst(e, asmamd64.OperationMove64Bits, "a_base+0(FP), SI")
	inst(e, asmamd64.OperationMove64Bits, "a_len+8(FP), CX")
	e.Blank()
	for _, acc := range accumulators {
		inst(e, asmamd64.OperationXorPackedDoubles, fmt.Sprintf("%s, %s", acc, acc))
	}
	inst(e, asmamd64.OperationXorPackedDoubles, "X3, X3")
	e.Blank()
	inst(e, asmamd64.OperationCompare64Bits, fmt.Sprintf("CX, $%d", unrollWidth))
	inst(e, asmamd64.OperationJumpIfLessSigned, narrowerTailFullLabel)
}

// sumF64SSEAccumBody emits one slot of the SSE float64 sum main loop.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes acc (string) which is the XMM accumulator register this slot adds into.
// Takes off (int) which is the slot's byte offset in the unrolled iteration.
func sumF64SSEAccumBody(e *asmgen.Emitter, acc string, off int) {
	inst(e, asmamd64.OperationMoveUnalignedPackedDoubles, fmt.Sprintf("%d(SI), X1", off))
	inst(e, asmamd64.OperationAddPackedDoubles, fmt.Sprintf("X1, %s", acc))
}

// sumF64SSELoopFooter emits the pointer-advance, counter decrement, and back branch for
// the SSE float64 sum main loop.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes byteStride (int) which is the bytes per unrolled iteration.
func sumF64SSELoopFooter(e *asmgen.Emitter, byteStride int) {
	unrollWidth := byteStride / 8
	inst(e, asmamd64.OperationAdd64Bits, fmt.Sprintf("$%d, SI", byteStride))
	inst(e, asmamd64.OperationSubtract64Bits, fmt.Sprintf("$%d, CX", unrollWidth))
	inst(e, asmamd64.OperationCompare64Bits, fmt.Sprintf("CX, $%d", unrollWidth))
	inst(e, asmamd64.OperationJumpIfGreaterOrEqualSigned, fmt.Sprintf("sumf64sse_loop%d", unrollWidth))
}

// sumF64SSEFold collapses accumulators[1:] into accumulators[0] for the SSE float64 sum
// kernel using ADDPD.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes accs ([]string) which is the ordered XMM accumulators; tail accs fold into head.
func sumF64SSEFold(e *asmgen.Emitter, accs []string) {
	for _, acc := range accs[1:] {
		inst(e, asmamd64.OperationAddPackedDoubles, fmt.Sprintf("%s, %s", acc, accs[0]))
	}
}

// sumF64SSENarrowerLoop emits the 2-wide narrower tail for the SSE float64 sum kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func sumF64SSENarrowerLoop(e *asmgen.Emitter) {
	e.Label("sumf64sse_tail2")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $2")
	inst(e, asmamd64.OperationJumpIfLessSigned, "sumf64sse_tail")
	e.Blank()
	e.Label("sumf64sse_loop2")
	inst(e, asmamd64.OperationMoveUnalignedPackedDoubles, "(SI), X1")
	inst(e, asmamd64.OperationAddPackedDoubles, "X1, X0")
	inst(e, asmamd64.OperationAdd64Bits, "$16, SI")
	inst(e, asmamd64.OperationSubtract64Bits, "$2, CX")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $2")
	inst(e, asmamd64.OperationJumpIfGreaterOrEqualSigned, "sumf64sse_loop2")
	e.Blank()
}

// sumF64SSEScalarTail emits the scalar drain for the SSE float64 sum kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func sumF64SSEScalarTail(e *asmgen.Emitter) {
	e.Label("sumf64sse_tail")
	inst(e, asmamd64.OperationTest64Bits, "CX, CX")
	inst(e, asmamd64.OperationJumpIfZero, "sumf64sse_reduce")
	e.Blank()
	e.Label("sumf64sse_tail_loop")
	inst(e, asmamd64.OperationMoveScalarDouble, "(SI), X1")
	inst(e, asmamd64.OperationAddScalarDouble, "X1, X3")
	inst(e, asmamd64.OperationAdd64Bits, "$8, SI")
	inst(e, asmamd64.OperationDecrement64Bits, "CX")
	inst(e, asmamd64.OperationJumpIfNotZero, "sumf64sse_tail_loop")
	e.Blank()
}

// sumF64SSEReduceAndReturn emits the horizontal reduce, scalar tail merge, return-slot
// store, and final RET for the SSE float64 sum kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes v (*amd64VectormathsOps) which provides the SSE horizontal reduce helper.
func sumF64SSEReduceAndReturn(e *asmgen.Emitter, v *amd64VectormathsOps) {
	e.Label("sumf64sse_reduce")
	v.emitSSEDoubleHorizontalReduce(e)
	inst(e, asmamd64.OperationAddScalarDouble, "X3, X0")
	inst(e, asmamd64.OperationMoveScalarDouble, "X0, ret+24(FP)")
	e.Instruction(asmamd64.OperationReturn)
}

// sumF64AVX2Prologue emits the argument-load, accumulator zeroing, and unroll-width guard
// for the AVX2 float64 sum kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes accumulators ([]string) which names the YMM registers zeroed as packed
// accumulators.
// Takes unrollWidth (int) which is the float64 elements processed per main-loop
// iteration.
// Takes narrowerTailFullLabel (string) which is the label jumped to when input is short.
func sumF64AVX2Prologue(e *asmgen.Emitter, accumulators []string, unrollWidth int, narrowerTailFullLabel string) {
	inst(e, asmamd64.OperationMove64Bits, "a_base+0(FP), SI")
	inst(e, asmamd64.OperationMove64Bits, "a_len+8(FP), CX")
	e.Blank()
	for _, acc := range accumulators {
		inst(e, asmamd64.OperationVexXorPackedDoubles, fmt.Sprintf("%s, %s, %s", acc, acc, acc))
	}
	inst(e, asmamd64.OperationVexXorPackedDoubles, "X3, X3, X3")
	e.Blank()
	inst(e, asmamd64.OperationCompare64Bits, fmt.Sprintf("CX, $%d", unrollWidth))
	inst(e, asmamd64.OperationJumpIfLessSigned, narrowerTailFullLabel)
}

// sumF64AVX2AccumBody emits one slot of the AVX2 float64 sum main loop.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes acc (string) which is the YMM accumulator register this slot adds into.
// Takes off (int) which is the slot's byte offset in the unrolled iteration.
func sumF64AVX2AccumBody(e *asmgen.Emitter, acc string, off int) {
	inst(e, asmamd64.OperationVexMoveUnalignedPackedDoubles, fmt.Sprintf("%d(SI), Y1", off))
	inst(e, asmamd64.OperationVexAddPackedDoubles, fmt.Sprintf("Y1, %s, %s", acc, acc))
}

// sumF64AVX2LoopFooter emits the pointer-advance, counter decrement, and back branch for
// the AVX2 float64 sum main loop.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes byteStride (int) which is the bytes per unrolled iteration.
func sumF64AVX2LoopFooter(e *asmgen.Emitter, byteStride int) {
	unrollWidth := byteStride / 8
	inst(e, asmamd64.OperationAdd64Bits, fmt.Sprintf("$%d, SI", byteStride))
	inst(e, asmamd64.OperationSubtract64Bits, fmt.Sprintf("$%d, CX", unrollWidth))
	inst(e, asmamd64.OperationCompare64Bits, fmt.Sprintf("CX, $%d", unrollWidth))
	inst(e, asmamd64.OperationJumpIfGreaterOrEqualSigned, fmt.Sprintf("sumf64avx_loop%d", unrollWidth))
}

// sumF64AVX2Fold collapses accumulators[1:] into accumulators[0] for the AVX2 float64 sum
// kernel using VADDPD.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes accs ([]string) which is the ordered YMM accumulators; tail accs fold into head.
func sumF64AVX2Fold(e *asmgen.Emitter, accs []string) {
	for _, acc := range accs[1:] {
		inst(e, asmamd64.OperationVexAddPackedDoubles, fmt.Sprintf("%s, %s, %s", acc, accs[0], accs[0]))
	}
}

// sumF64AVX2NarrowerLoop emits the 4-wide narrower tail for the AVX2 float64 sum kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func sumF64AVX2NarrowerLoop(e *asmgen.Emitter) {
	e.Label("sumf64avx_tail4")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $4")
	inst(e, asmamd64.OperationJumpIfLessSigned, "sumf64avx_tail")
	e.Blank()
	e.Label("sumf64avx_loop4")
	inst(e, asmamd64.OperationVexMoveUnalignedPackedDoubles, "(SI), Y1")
	inst(e, asmamd64.OperationVexAddPackedDoubles, "Y1, Y0, Y0")
	inst(e, asmamd64.OperationAdd64Bits, "$32, SI")
	inst(e, asmamd64.OperationSubtract64Bits, "$4, CX")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $4")
	inst(e, asmamd64.OperationJumpIfGreaterOrEqualSigned, "sumf64avx_loop4")
	e.Blank()
}

// sumF64AVX2ScalarTail emits the scalar drain for the AVX2 float64 sum kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func sumF64AVX2ScalarTail(e *asmgen.Emitter) {
	e.Label("sumf64avx_tail")
	inst(e, asmamd64.OperationTest64Bits, "CX, CX")
	inst(e, asmamd64.OperationJumpIfZero, "sumf64avx_reduce")
	e.Blank()
	e.Label("sumf64avx_tail_loop")
	inst(e, asmamd64.OperationVexMoveScalarDouble, "(SI), X1")
	inst(e, asmamd64.OperationVexAddScalarDouble, "X1, X3, X3")
	inst(e, asmamd64.OperationAdd64Bits, "$8, SI")
	inst(e, asmamd64.OperationDecrement64Bits, "CX")
	inst(e, asmamd64.OperationJumpIfNotZero, "sumf64avx_tail_loop")
	e.Blank()
}

// sumF64AVX2ReduceAndReturn emits the horizontal reduce, scalar tail merge, VZEROUPPER,
// return-slot store, and final RET for the AVX2 float64 sum kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes v (*amd64VectormathsOps) which provides the AVX2 horizontal reduce helper.
func sumF64AVX2ReduceAndReturn(e *asmgen.Emitter, v *amd64VectormathsOps) {
	e.Label("sumf64avx_reduce")
	v.emitAVX2DoubleHorizontalReduce(e)
	inst(e, asmamd64.OperationVexAddScalarDouble, "X3, X0, X0")
	e.Instruction(asmamd64.OperationVexZeroUpper)
	inst(e, asmamd64.OperationMoveScalarDouble, "X0, ret+24(FP)")
	e.Instruction(asmamd64.OperationReturn)
}

// dotProductSSEPrologue emits the prologue for the SSE dot product kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes accumulators ([]string) which names the XMM registers zeroed as packed
// accumulators.
// Takes unrollWidth (int) which is the float32 elements processed per main-loop
// iteration.
// Takes narrowerTailFullLabel (string) which is the label jumped to when input is short.
func dotProductSSEPrologue(e *asmgen.Emitter, accumulators []string, unrollWidth int, narrowerTailFullLabel string) {
	inst(e, asmamd64.OperationMove64Bits, "a_base+0(FP), SI")
	inst(e, asmamd64.OperationMove64Bits, "a_len+8(FP), CX")
	inst(e, asmamd64.OperationMove64Bits, "b_base+24(FP), DI")
	e.Blank()
	for _, acc := range accumulators {
		inst(e, asmamd64.OperationXorPackedSingles, fmt.Sprintf("%s, %s", acc, acc))
	}
	inst(e, asmamd64.OperationXorPackedSingles, "X3, X3")
	e.Blank()
	inst(e, asmamd64.OperationCompare64Bits, fmt.Sprintf("CX, $%d", unrollWidth))
	inst(e, asmamd64.OperationJumpIfLessSigned, narrowerTailFullLabel)
}

// dotProductSSEAccumBody emits one slot of the SSE dot product main loop.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes acc (string) which is the XMM accumulator register this slot multiplies into.
// Takes off (int) which is the slot's byte offset in the unrolled iteration.
func dotProductSSEAccumBody(e *asmgen.Emitter, acc string, off int) {
	inst(e, asmamd64.OperationMoveUnalignedPackedSingles, fmt.Sprintf("%d(SI), X1", off))
	inst(e, asmamd64.OperationMoveUnalignedPackedSingles, fmt.Sprintf("%d(DI), X2", off))
	inst(e, asmamd64.OperationMultiplyPackedSingles, "X2, X1")
	inst(e, asmamd64.OperationAddPackedSingles, fmt.Sprintf("X1, %s", acc))
}

// dotProductSSELoopFooter emits the pointer-advance, counter decrement, and back branch
// for the SSE dot product main loop.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes byteStride (int) which is the bytes per unrolled iteration.
func dotProductSSELoopFooter(e *asmgen.Emitter, byteStride int) {
	unrollWidth := byteStride / 4
	inst(e, asmamd64.OperationAdd64Bits, fmt.Sprintf("$%d, SI", byteStride))
	inst(e, asmamd64.OperationAdd64Bits, fmt.Sprintf("$%d, DI", byteStride))
	inst(e, asmamd64.OperationSubtract64Bits, fmt.Sprintf("$%d, CX", unrollWidth))
	inst(e, asmamd64.OperationCompare64Bits, fmt.Sprintf("CX, $%d", unrollWidth))
	inst(e, asmamd64.OperationJumpIfGreaterOrEqualSigned, fmt.Sprintf("dotsse_loop%d", unrollWidth))
}

// dotProductSSEFold collapses accumulators[1:] into accumulators[0] for the SSE dot
// product kernel using ADDPS.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes accs ([]string) which is the ordered XMM accumulators; tail accs fold into head.
func dotProductSSEFold(e *asmgen.Emitter, accs []string) {
	for _, acc := range accs[1:] {
		inst(e, asmamd64.OperationAddPackedSingles, fmt.Sprintf("%s, %s", acc, accs[0]))
	}
}

// dotProductSSENarrowerLoop emits the 4-wide narrower tail for the SSE dot product
// kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func dotProductSSENarrowerLoop(e *asmgen.Emitter) {
	e.Label("dotsse_tail4")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $4")
	inst(e, asmamd64.OperationJumpIfLessSigned, "dotsse_tail")
	e.Blank()
	e.Label("dotsse_loop4")
	inst(e, asmamd64.OperationMoveUnalignedPackedSingles, "(SI), X1")
	inst(e, asmamd64.OperationMoveUnalignedPackedSingles, "(DI), X2")
	inst(e, asmamd64.OperationMultiplyPackedSingles, "X2, X1")
	inst(e, asmamd64.OperationAddPackedSingles, "X1, X0")
	inst(e, asmamd64.OperationAdd64Bits, "$16, SI")
	inst(e, asmamd64.OperationAdd64Bits, "$16, DI")
	inst(e, asmamd64.OperationSubtract64Bits, "$4, CX")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $4")
	inst(e, asmamd64.OperationJumpIfGreaterOrEqualSigned, "dotsse_loop4")
	e.Blank()
}

// dotProductSSEScalarTail emits the scalar drain for the SSE dot product kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func dotProductSSEScalarTail(e *asmgen.Emitter) {
	e.Label("dotsse_tail")
	inst(e, asmamd64.OperationTest64Bits, "CX, CX")
	inst(e, asmamd64.OperationJumpIfZero, "dotsse_reduce")
	e.Blank()
	e.Label("dotsse_tail_loop")
	inst(e, asmamd64.OperationMoveScalarSingle, "(SI), X1")
	inst(e, asmamd64.OperationMoveScalarSingle, "(DI), X2")
	inst(e, asmamd64.OperationMultiplyScalarSingle, "X2, X1")
	inst(e, asmamd64.OperationAddScalarSingle, "X1, X3")
	inst(e, asmamd64.OperationAdd64Bits, "$4, SI")
	inst(e, asmamd64.OperationAdd64Bits, "$4, DI")
	inst(e, asmamd64.OperationDecrement64Bits, "CX")
	inst(e, asmamd64.OperationJumpIfNotZero, "dotsse_tail_loop")
	e.Blank()
}

// dotProductSSEReduceAndReturn emits the horizontal reduce, scalar tail merge,
// return-slot store, and final RET for the SSE dot product kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes v (*amd64VectormathsOps) which provides the SSE horizontal reduce helper.
func dotProductSSEReduceAndReturn(e *asmgen.Emitter, v *amd64VectormathsOps) {
	e.Label("dotsse_reduce")
	v.emitSSEHorizontalReduce(e)
	e.Blank()
	inst(e, asmamd64.OperationAddScalarSingle, "X3, X0")
	e.Blank()
	inst(e, asmamd64.OperationMoveScalarSingle, "X0, ret+48(FP)")
	e.Instruction(asmamd64.OperationReturn)
}

// dotProductAVX2Prologue emits the prologue for the AVX2 dot product kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes accumulators ([]string) which names the YMM registers zeroed as packed
// accumulators.
// Takes unrollWidth (int) which is the float32 elements processed per main-loop
// iteration.
// Takes narrowerTailFullLabel (string) which is the label jumped to when input is short.
func dotProductAVX2Prologue(e *asmgen.Emitter, accumulators []string, unrollWidth int, narrowerTailFullLabel string) {
	inst(e, asmamd64.OperationMove64Bits, "a_base+0(FP), SI")
	inst(e, asmamd64.OperationMove64Bits, "a_len+8(FP), CX")
	inst(e, asmamd64.OperationMove64Bits, "b_base+24(FP), DI")
	e.Blank()
	for _, acc := range accumulators {
		inst(e, asmamd64.OperationVexXorPackedSingles, fmt.Sprintf("%s, %s, %s", acc, acc, acc))
	}
	inst(e, asmamd64.OperationVexXorPackedSingles, "X3, X3, X3")
	e.Blank()
	inst(e, asmamd64.OperationCompare64Bits, fmt.Sprintf("CX, $%d", unrollWidth))
	inst(e, asmamd64.OperationJumpIfLessSigned, narrowerTailFullLabel)
}

// dotProductAVX2AccumBody emits one slot of the AVX2 dot product main loop.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes acc (string) which is the YMM accumulator register this slot multiplies into.
// Takes off (int) which is the slot's byte offset in the unrolled iteration.
func dotProductAVX2AccumBody(e *asmgen.Emitter, acc string, off int) {
	inst(e, asmamd64.OperationVexMoveUnalignedPackedSingles, fmt.Sprintf("%d(SI), Y1", off))
	inst(e, asmamd64.OperationVexMoveUnalignedPackedSingles, fmt.Sprintf("%d(DI), Y2", off))
	inst(e, asmamd64.OperationVexMultiplyPackedSingles, "Y2, Y1, Y1")
	inst(e, asmamd64.OperationVexAddPackedSingles, fmt.Sprintf("Y1, %s, %s", acc, acc))
}

// dotProductAVX2LoopFooter emits the pointer-advance, counter decrement, and back branch
// for the AVX2 dot product main loop.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes byteStride (int) which is the bytes per unrolled iteration.
func dotProductAVX2LoopFooter(e *asmgen.Emitter, byteStride int) {
	unrollWidth := byteStride / 4
	inst(e, asmamd64.OperationAdd64Bits, fmt.Sprintf("$%d, SI", byteStride))
	inst(e, asmamd64.OperationAdd64Bits, fmt.Sprintf("$%d, DI", byteStride))
	inst(e, asmamd64.OperationSubtract64Bits, fmt.Sprintf("$%d, CX", unrollWidth))
	inst(e, asmamd64.OperationCompare64Bits, fmt.Sprintf("CX, $%d", unrollWidth))
	inst(e, asmamd64.OperationJumpIfGreaterOrEqualSigned, fmt.Sprintf("dotavx_loop%d", unrollWidth))
}

// dotProductAVX2Fold collapses accumulators[1:] into accumulators[0] for the AVX2 dot
// product kernel using VADDPS.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes accs ([]string) which is the ordered YMM accumulators; tail accs fold into head.
func dotProductAVX2Fold(e *asmgen.Emitter, accs []string) {
	for _, acc := range accs[1:] {
		inst(e, asmamd64.OperationVexAddPackedSingles, fmt.Sprintf("%s, %s, %s", acc, accs[0], accs[0]))
	}
}

// dotProductAVX2NarrowerLoop emits the 8-wide narrower tail for the AVX2 dot product
// kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func dotProductAVX2NarrowerLoop(e *asmgen.Emitter) {
	e.Label("dotavx_tail8")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $8")
	inst(e, asmamd64.OperationJumpIfLessSigned, "dotavx_tail")
	e.Blank()
	e.Label("dotavx_loop8")
	inst(e, asmamd64.OperationVexMoveUnalignedPackedSingles, "(SI), Y1")
	inst(e, asmamd64.OperationVexMoveUnalignedPackedSingles, "(DI), Y2")
	inst(e, asmamd64.OperationVexMultiplyPackedSingles, "Y2, Y1, Y1")
	inst(e, asmamd64.OperationVexAddPackedSingles, "Y1, Y0, Y0")
	inst(e, asmamd64.OperationAdd64Bits, "$32, SI")
	inst(e, asmamd64.OperationAdd64Bits, "$32, DI")
	inst(e, asmamd64.OperationSubtract64Bits, "$8, CX")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $8")
	inst(e, asmamd64.OperationJumpIfGreaterOrEqualSigned, "dotavx_loop8")
	e.Blank()
}

// dotProductAVX2ScalarTail emits the scalar drain for the AVX2 dot product kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func dotProductAVX2ScalarTail(e *asmgen.Emitter) {
	e.Label("dotavx_tail")
	inst(e, asmamd64.OperationTest64Bits, "CX, CX")
	inst(e, asmamd64.OperationJumpIfZero, "dotavx_reduce")
	e.Blank()
	e.Label("dotavx_tail_loop")
	inst(e, asmamd64.OperationVexMoveScalarSingle, "(SI), X1")
	inst(e, asmamd64.OperationVexMoveScalarSingle, "(DI), X2")
	inst(e, asmamd64.OperationVexMultiplyScalarSingle, "X2, X1, X1")
	inst(e, asmamd64.OperationVexAddScalarSingle, "X1, X3, X3")
	inst(e, asmamd64.OperationAdd64Bits, "$4, SI")
	inst(e, asmamd64.OperationAdd64Bits, "$4, DI")
	inst(e, asmamd64.OperationDecrement64Bits, "CX")
	inst(e, asmamd64.OperationJumpIfNotZero, "dotavx_tail_loop")
	e.Blank()
}

// dotProductAVX2ReduceAndReturn emits the horizontal reduce, scalar tail merge,
// VZEROUPPER, return-slot store, and final RET for the AVX2 dot product kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes v (*amd64VectormathsOps) which provides the AVX2 horizontal reduce helper.
func dotProductAVX2ReduceAndReturn(e *asmgen.Emitter, v *amd64VectormathsOps) {
	e.Label("dotavx_reduce")
	v.emitAVX2HorizontalReduce(e)
	e.Blank()
	inst(e, asmamd64.OperationVexAddScalarSingle, "X3, X0, X0")
	e.Blank()
	e.Instruction(asmamd64.OperationVexZeroUpper)
	inst(e, asmamd64.OperationMoveScalarSingle, "X0, ret+48(FP)")
	e.Instruction(asmamd64.OperationReturn)
}

// euclidSSEPrologue emits the prologue for the SSE squared Euclidean distance kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes accumulators ([]string) which names the XMM registers zeroed as packed
// accumulators.
// Takes unrollWidth (int) which is the float32 elements processed per main-loop
// iteration.
// Takes narrowerTailFullLabel (string) which is the label jumped to when input is short.
func euclidSSEPrologue(e *asmgen.Emitter, accumulators []string, unrollWidth int, narrowerTailFullLabel string) {
	inst(e, asmamd64.OperationMove64Bits, "a_base+0(FP), SI")
	inst(e, asmamd64.OperationMove64Bits, "a_len+8(FP), CX")
	inst(e, asmamd64.OperationMove64Bits, "b_base+24(FP), DI")
	e.Blank()
	for _, acc := range accumulators {
		inst(e, asmamd64.OperationXorPackedSingles, fmt.Sprintf("%s, %s", acc, acc))
	}
	inst(e, asmamd64.OperationXorPackedSingles, "X3, X3")
	e.Blank()
	inst(e, asmamd64.OperationCompare64Bits, fmt.Sprintf("CX, $%d", unrollWidth))
	inst(e, asmamd64.OperationJumpIfLessSigned, narrowerTailFullLabel)
}

// euclidSSEAccumBody emits one slot of the SSE squared Euclidean distance main loop.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes acc (string) which is the XMM accumulator for the squared difference.
// Takes off (int) which is the slot's byte offset in the unrolled iteration.
func euclidSSEAccumBody(e *asmgen.Emitter, acc string, off int) {
	inst(e, asmamd64.OperationMoveUnalignedPackedSingles, fmt.Sprintf("%d(SI), X1", off))
	inst(e, asmamd64.OperationMoveUnalignedPackedSingles, fmt.Sprintf("%d(DI), X2", off))
	inst(e, asmamd64.OperationSubtractPackedSingles, "X2, X1")
	inst(e, asmamd64.OperationMultiplyPackedSingles, "X1, X1")
	inst(e, asmamd64.OperationAddPackedSingles, fmt.Sprintf("X1, %s", acc))
}

// euclidSSELoopFooter emits the pointer-advance, counter decrement, and back branch for
// the SSE squared Euclidean distance main loop.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes byteStride (int) which is the bytes per unrolled iteration.
func euclidSSELoopFooter(e *asmgen.Emitter, byteStride int) {
	unrollWidth := byteStride / 4
	inst(e, asmamd64.OperationAdd64Bits, fmt.Sprintf("$%d, SI", byteStride))
	inst(e, asmamd64.OperationAdd64Bits, fmt.Sprintf("$%d, DI", byteStride))
	inst(e, asmamd64.OperationSubtract64Bits, fmt.Sprintf("$%d, CX", unrollWidth))
	inst(e, asmamd64.OperationCompare64Bits, fmt.Sprintf("CX, $%d", unrollWidth))
	inst(e, asmamd64.OperationJumpIfGreaterOrEqualSigned, fmt.Sprintf("euclidsse_loop%d", unrollWidth))
}

// euclidSSEFold collapses accumulators[1:] into accumulators[0] for the SSE squared
// Euclidean distance kernel using ADDPS.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes accs ([]string) which is the ordered XMM accumulators; tail accs fold into head.
func euclidSSEFold(e *asmgen.Emitter, accs []string) {
	for _, acc := range accs[1:] {
		inst(e, asmamd64.OperationAddPackedSingles, fmt.Sprintf("%s, %s", acc, accs[0]))
	}
}

// euclidSSENarrowerLoop emits the 4-wide narrower tail for the SSE squared Euclidean
// distance kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func euclidSSENarrowerLoop(e *asmgen.Emitter) {
	e.Label("euclidsse_tail4")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $4")
	inst(e, asmamd64.OperationJumpIfLessSigned, "euclidsse_tail")
	e.Blank()
	e.Label("euclidsse_loop4")
	inst(e, asmamd64.OperationMoveUnalignedPackedSingles, "(SI), X1")
	inst(e, asmamd64.OperationMoveUnalignedPackedSingles, "(DI), X2")
	inst(e, asmamd64.OperationSubtractPackedSingles, "X2, X1")
	inst(e, asmamd64.OperationMultiplyPackedSingles, "X1, X1")
	inst(e, asmamd64.OperationAddPackedSingles, "X1, X0")
	inst(e, asmamd64.OperationAdd64Bits, "$16, SI")
	inst(e, asmamd64.OperationAdd64Bits, "$16, DI")
	inst(e, asmamd64.OperationSubtract64Bits, "$4, CX")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $4")
	inst(e, asmamd64.OperationJumpIfGreaterOrEqualSigned, "euclidsse_loop4")
	e.Blank()
}

// euclidSSEScalarTail emits the scalar drain for the SSE squared Euclidean distance
// kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func euclidSSEScalarTail(e *asmgen.Emitter) {
	e.Label("euclidsse_tail")
	inst(e, asmamd64.OperationTest64Bits, "CX, CX")
	inst(e, asmamd64.OperationJumpIfZero, "euclidsse_reduce")
	e.Blank()
	e.Label("euclidsse_tail_loop")
	inst(e, asmamd64.OperationMoveScalarSingle, "(SI), X1")
	inst(e, asmamd64.OperationMoveScalarSingle, "(DI), X2")
	inst(e, asmamd64.OperationSubtractScalarSingle, "X2, X1")
	inst(e, asmamd64.OperationMultiplyScalarSingle, "X1, X1")
	inst(e, asmamd64.OperationAddScalarSingle, "X1, X3")
	inst(e, asmamd64.OperationAdd64Bits, "$4, SI")
	inst(e, asmamd64.OperationAdd64Bits, "$4, DI")
	inst(e, asmamd64.OperationDecrement64Bits, "CX")
	inst(e, asmamd64.OperationJumpIfNotZero, "euclidsse_tail_loop")
	e.Blank()
}

// euclidSSEReduceAndReturn emits the horizontal reduce, scalar tail merge, return-slot
// store, and final RET for the SSE squared Euclidean distance kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes v (*amd64VectormathsOps) which provides the SSE horizontal reduce helper.
func euclidSSEReduceAndReturn(e *asmgen.Emitter, v *amd64VectormathsOps) {
	e.Label("euclidsse_reduce")
	v.emitSSEHorizontalReduce(e)
	e.Blank()
	inst(e, asmamd64.OperationAddScalarSingle, "X3, X0")
	e.Blank()
	inst(e, asmamd64.OperationMoveScalarSingle, "X0, ret+48(FP)")
	e.Instruction(asmamd64.OperationReturn)
}

// euclidAVX2Prologue emits the prologue for the AVX2 squared Euclidean distance kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes accumulators ([]string) which names the YMM registers zeroed as packed
// accumulators.
// Takes unrollWidth (int) which is the float32 elements processed per main-loop
// iteration.
// Takes narrowerTailFullLabel (string) which is the label jumped to when input is short.
func euclidAVX2Prologue(e *asmgen.Emitter, accumulators []string, unrollWidth int, narrowerTailFullLabel string) {
	inst(e, asmamd64.OperationMove64Bits, "a_base+0(FP), SI")
	inst(e, asmamd64.OperationMove64Bits, "a_len+8(FP), CX")
	inst(e, asmamd64.OperationMove64Bits, "b_base+24(FP), DI")
	e.Blank()
	for _, acc := range accumulators {
		inst(e, asmamd64.OperationVexXorPackedSingles, fmt.Sprintf("%s, %s, %s", acc, acc, acc))
	}
	inst(e, asmamd64.OperationVexXorPackedSingles, "X3, X3, X3")
	e.Blank()
	inst(e, asmamd64.OperationCompare64Bits, fmt.Sprintf("CX, $%d", unrollWidth))
	inst(e, asmamd64.OperationJumpIfLessSigned, narrowerTailFullLabel)
}

// euclidAVX2AccumBody emits one slot of the AVX2 squared Euclidean distance main loop
// using VFMADD231PS to fuse the square and accumulate after VSUBPS.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes acc (string) which is the YMM accumulator for the fused squared difference.
// Takes off (int) which is the slot's byte offset in the unrolled iteration.
func euclidAVX2AccumBody(e *asmgen.Emitter, acc string, off int) {
	inst(e, asmamd64.OperationVexMoveUnalignedPackedSingles, fmt.Sprintf("%d(SI), Y1", off))
	inst(e, asmamd64.OperationVexSubtractPackedSingles, fmt.Sprintf("%d(DI), Y1, Y1", off))
	inst(e, asmamd64.OperationVexFusedMultiplyAdd231PackedSingles, fmt.Sprintf("Y1, Y1, %s", acc))
}

// euclidAVX2LoopFooter emits the pointer-advance, counter decrement, and back branch for
// the AVX2 squared Euclidean distance main loop.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes byteStride (int) which is the bytes per unrolled iteration.
func euclidAVX2LoopFooter(e *asmgen.Emitter, byteStride int) {
	unrollWidth := byteStride / 4
	inst(e, asmamd64.OperationAdd64Bits, fmt.Sprintf("$%d, SI", byteStride))
	inst(e, asmamd64.OperationAdd64Bits, fmt.Sprintf("$%d, DI", byteStride))
	inst(e, asmamd64.OperationSubtract64Bits, fmt.Sprintf("$%d, CX", unrollWidth))
	inst(e, asmamd64.OperationCompare64Bits, fmt.Sprintf("CX, $%d", unrollWidth))
	inst(e, asmamd64.OperationJumpIfGreaterOrEqualSigned, fmt.Sprintf("euclidavx_loop%d", unrollWidth))
}

// euclidAVX2Fold collapses accumulators[1:] into accumulators[0] for the AVX2 squared
// Euclidean distance kernel using VADDPS.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes accs ([]string) which is the ordered YMM accumulators; tail accs fold into head.
func euclidAVX2Fold(e *asmgen.Emitter, accs []string) {
	for _, acc := range accs[1:] {
		inst(e, asmamd64.OperationVexAddPackedSingles, fmt.Sprintf("%s, %s, %s", acc, accs[0], accs[0]))
	}
}

// euclidAVX2NarrowerLoop emits the 8-wide narrower tail for the AVX2 squared Euclidean
// distance kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func euclidAVX2NarrowerLoop(e *asmgen.Emitter) {
	e.Label("euclidavx_tail8")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $8")
	inst(e, asmamd64.OperationJumpIfLessSigned, "euclidavx_tail")
	e.Blank()
	e.Label("euclidavx_loop8")
	inst(e, asmamd64.OperationVexMoveUnalignedPackedSingles, "(SI), Y1")
	inst(e, asmamd64.OperationVexSubtractPackedSingles, "(DI), Y1, Y1")
	inst(e, asmamd64.OperationVexFusedMultiplyAdd231PackedSingles, "Y1, Y1, Y0")
	inst(e, asmamd64.OperationAdd64Bits, "$32, SI")
	inst(e, asmamd64.OperationAdd64Bits, "$32, DI")
	inst(e, asmamd64.OperationSubtract64Bits, "$8, CX")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $8")
	inst(e, asmamd64.OperationJumpIfGreaterOrEqualSigned, "euclidavx_loop8")
	e.Blank()
}

// euclidAVX2ScalarTail emits the scalar drain for the AVX2 squared Euclidean distance
// kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func euclidAVX2ScalarTail(e *asmgen.Emitter) {
	e.Label("euclidavx_tail")
	inst(e, asmamd64.OperationTest64Bits, "CX, CX")
	inst(e, asmamd64.OperationJumpIfZero, "euclidavx_reduce")
	e.Blank()
	e.Label("euclidavx_tail_loop")
	inst(e, asmamd64.OperationVexMoveScalarSingle, "(SI), X1")
	inst(e, asmamd64.OperationVexSubtractScalarSingle, "(DI), X1, X1")
	inst(e, asmamd64.OperationVexFusedMultiplyAdd231ScalarSingle, "X1, X1, X3")
	inst(e, asmamd64.OperationAdd64Bits, "$4, SI")
	inst(e, asmamd64.OperationAdd64Bits, "$4, DI")
	inst(e, asmamd64.OperationDecrement64Bits, "CX")
	inst(e, asmamd64.OperationJumpIfNotZero, "euclidavx_tail_loop")
	e.Blank()
}

// euclidAVX2ReduceAndReturn emits the horizontal reduce, scalar tail merge, VZEROUPPER,
// return-slot store, and final RET for the AVX2 squared Euclidean distance kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes v (*amd64VectormathsOps) which provides the AVX2 horizontal reduce helper.
func euclidAVX2ReduceAndReturn(e *asmgen.Emitter, v *amd64VectormathsOps) {
	e.Label("euclidavx_reduce")
	v.emitAVX2HorizontalReduce(e)
	inst(e, asmamd64.OperationVexAddScalarSingle, "X3, X0, X0")
	e.Instruction(asmamd64.OperationVexZeroUpper)
	inst(e, asmamd64.OperationMoveScalarSingle, "X0, ret+48(FP)")
	e.Instruction(asmamd64.OperationReturn)
}

// dotF64SSEPrologue emits the prologue for the SSE float64 dot product kernel: argument
// loads, accumulator zeroing, and the unroll-width guard.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes accumulators ([]string) which names the XMM registers zeroed as packed
// accumulators.
// Takes unrollWidth (int) which is the float64 elements processed per main-loop
// iteration.
// Takes narrowerTailFullLabel (string) which is the label jumped to when input is short.
func dotF64SSEPrologue(e *asmgen.Emitter, accumulators []string, unrollWidth int, narrowerTailFullLabel string) {
	inst(e, asmamd64.OperationMove64Bits, "a_base+0(FP), SI")
	inst(e, asmamd64.OperationMove64Bits, "a_len+8(FP), CX")
	inst(e, asmamd64.OperationMove64Bits, "b_base+24(FP), DI")
	e.Blank()
	for _, acc := range accumulators {
		inst(e, asmamd64.OperationXorPackedDoubles, fmt.Sprintf("%s, %s", acc, acc))
	}
	inst(e, asmamd64.OperationXorPackedDoubles, "X3, X3")
	e.Blank()
	inst(e, asmamd64.OperationCompare64Bits, fmt.Sprintf("CX, $%d", unrollWidth))
	inst(e, asmamd64.OperationJumpIfLessSigned, narrowerTailFullLabel)
}

// dotF64SSEAccumBody emits one slot of the SSE float64 dot product main loop.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes acc (string) which is the XMM accumulator register this slot adds into.
// Takes off (int) which is the slot's byte offset in the unrolled iteration.
func dotF64SSEAccumBody(e *asmgen.Emitter, acc string, off int) {
	inst(e, asmamd64.OperationMoveUnalignedPackedDoubles, fmt.Sprintf("%d(SI), X1", off))
	inst(e, asmamd64.OperationMoveUnalignedPackedDoubles, fmt.Sprintf("%d(DI), X2", off))
	inst(e, asmamd64.OperationMultiplyPackedDoubles, "X2, X1")
	inst(e, asmamd64.OperationAddPackedDoubles, fmt.Sprintf("X1, %s", acc))
}

// dotF64SSELoopFooter emits the pointer-advance, counter decrement, and back branch for
// the SSE float64 dot product main loop.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes byteStride (int) which is the bytes per unrolled iteration.
func dotF64SSELoopFooter(e *asmgen.Emitter, byteStride int) {
	unrollWidth := byteStride / 8
	inst(e, asmamd64.OperationAdd64Bits, fmt.Sprintf("$%d, SI", byteStride))
	inst(e, asmamd64.OperationAdd64Bits, fmt.Sprintf("$%d, DI", byteStride))
	inst(e, asmamd64.OperationSubtract64Bits, fmt.Sprintf("$%d, CX", unrollWidth))
	inst(e, asmamd64.OperationCompare64Bits, fmt.Sprintf("CX, $%d", unrollWidth))
	inst(e, asmamd64.OperationJumpIfGreaterOrEqualSigned, fmt.Sprintf("dotf64sse_loop%d", unrollWidth))
}

// dotF64SSEFold collapses accumulators[1:] into accumulators[0] for the SSE float64 dot
// product kernel using ADDPD.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes accs ([]string) which is the ordered XMM accumulators; tail accs fold into head.
func dotF64SSEFold(e *asmgen.Emitter, accs []string) {
	for _, acc := range accs[1:] {
		inst(e, asmamd64.OperationAddPackedDoubles, fmt.Sprintf("%s, %s", acc, accs[0]))
	}
}

// dotF64SSENarrowerLoop emits the 2-wide narrower tail for the SSE float64 dot product
// kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func dotF64SSENarrowerLoop(e *asmgen.Emitter) {
	e.Label("dotf64sse_tail2")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $2")
	inst(e, asmamd64.OperationJumpIfLessSigned, "dotf64sse_tail")
	e.Blank()
	e.Label("dotf64sse_loop2")
	inst(e, asmamd64.OperationMoveUnalignedPackedDoubles, "(SI), X1")
	inst(e, asmamd64.OperationMoveUnalignedPackedDoubles, "(DI), X2")
	inst(e, asmamd64.OperationMultiplyPackedDoubles, "X2, X1")
	inst(e, asmamd64.OperationAddPackedDoubles, "X1, X0")
	inst(e, asmamd64.OperationAdd64Bits, "$16, SI")
	inst(e, asmamd64.OperationAdd64Bits, "$16, DI")
	inst(e, asmamd64.OperationSubtract64Bits, "$2, CX")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $2")
	inst(e, asmamd64.OperationJumpIfGreaterOrEqualSigned, "dotf64sse_loop2")
	e.Blank()
}

// dotF64SSEScalarTail emits the scalar drain for the SSE float64 dot product kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func dotF64SSEScalarTail(e *asmgen.Emitter) {
	e.Label("dotf64sse_tail")
	inst(e, asmamd64.OperationTest64Bits, "CX, CX")
	inst(e, asmamd64.OperationJumpIfZero, "dotf64sse_reduce")
	e.Blank()
	e.Label("dotf64sse_tail_loop")
	inst(e, asmamd64.OperationMoveScalarDouble, "(SI), X1")
	inst(e, asmamd64.OperationMoveScalarDouble, "(DI), X2")
	inst(e, asmamd64.OperationMultiplyScalarDouble, "X2, X1")
	inst(e, asmamd64.OperationAddScalarDouble, "X1, X3")
	inst(e, asmamd64.OperationAdd64Bits, "$8, SI")
	inst(e, asmamd64.OperationAdd64Bits, "$8, DI")
	inst(e, asmamd64.OperationDecrement64Bits, "CX")
	inst(e, asmamd64.OperationJumpIfNotZero, "dotf64sse_tail_loop")
	e.Blank()
}

// dotF64SSEReduceAndReturn emits the horizontal reduce, scalar tail merge, return-slot
// store, and final RET for the SSE float64 dot product kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes v (*amd64VectormathsOps) which provides the SSE horizontal reduce helper.
func dotF64SSEReduceAndReturn(e *asmgen.Emitter, v *amd64VectormathsOps) {
	e.Label("dotf64sse_reduce")
	v.emitSSEDoubleHorizontalReduce(e)
	inst(e, asmamd64.OperationAddScalarDouble, "X3, X0")
	inst(e, asmamd64.OperationMoveScalarDouble, "X0, ret+48(FP)")
	e.Instruction(asmamd64.OperationReturn)
}

// dotF64AVX2Prologue emits the prologue for the AVX2 float64 dot product kernel: argument
// loads, YMM accumulator zeroing, and the unroll-width guard.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes accumulators ([]string) which names the YMM registers zeroed as packed
// accumulators.
// Takes unrollWidth (int) which is the float64 elements processed per main-loop
// iteration.
// Takes narrowerTailFullLabel (string) which is the label jumped to when input is short.
func dotF64AVX2Prologue(e *asmgen.Emitter, accumulators []string, unrollWidth int, narrowerTailFullLabel string) {
	inst(e, asmamd64.OperationMove64Bits, "a_base+0(FP), SI")
	inst(e, asmamd64.OperationMove64Bits, "a_len+8(FP), CX")
	inst(e, asmamd64.OperationMove64Bits, "b_base+24(FP), DI")
	e.Blank()
	for _, acc := range accumulators {
		inst(e, asmamd64.OperationVexXorPackedDoubles, fmt.Sprintf("%s, %s, %s", acc, acc, acc))
	}
	inst(e, asmamd64.OperationVexXorPackedDoubles, "X3, X3, X3")
	e.Blank()
	inst(e, asmamd64.OperationCompare64Bits, fmt.Sprintf("CX, $%d", unrollWidth))
	inst(e, asmamd64.OperationJumpIfLessSigned, narrowerTailFullLabel)
}

// dotF64AVX2AccumBody emits one slot of the AVX2 float64 dot product main loop using
// VFMADD231PD to fuse the multiply and accumulate.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes acc (string) which is the YMM accumulator register this slot adds into.
// Takes off (int) which is the slot's byte offset in the unrolled iteration.
func dotF64AVX2AccumBody(e *asmgen.Emitter, acc string, off int) {
	inst(e, asmamd64.OperationVexMoveUnalignedPackedDoubles, fmt.Sprintf("%d(SI), Y1", off))
	inst(e, asmamd64.OperationVexMoveUnalignedPackedDoubles, fmt.Sprintf("%d(DI), Y2", off))
	inst(e, asmamd64.OperationVexFusedMultiplyAdd231PackedDoubles, fmt.Sprintf("Y1, Y2, %s", acc))
}

// dotF64AVX2LoopFooter emits the pointer-advance, counter decrement, and back branch for
// the AVX2 float64 dot product main loop.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes byteStride (int) which is the bytes per unrolled iteration.
func dotF64AVX2LoopFooter(e *asmgen.Emitter, byteStride int) {
	unrollWidth := byteStride / 8
	inst(e, asmamd64.OperationAdd64Bits, fmt.Sprintf("$%d, SI", byteStride))
	inst(e, asmamd64.OperationAdd64Bits, fmt.Sprintf("$%d, DI", byteStride))
	inst(e, asmamd64.OperationSubtract64Bits, fmt.Sprintf("$%d, CX", unrollWidth))
	inst(e, asmamd64.OperationCompare64Bits, fmt.Sprintf("CX, $%d", unrollWidth))
	inst(e, asmamd64.OperationJumpIfGreaterOrEqualSigned, fmt.Sprintf("dotf64avx_loop%d", unrollWidth))
}

// dotF64AVX2Fold collapses accumulators[1:] into accumulators[0] for the AVX2 float64 dot
// product kernel using VADDPD.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes accs ([]string) which is the ordered YMM accumulators; tail accs fold into head.
func dotF64AVX2Fold(e *asmgen.Emitter, accs []string) {
	for _, acc := range accs[1:] {
		inst(e, asmamd64.OperationVexAddPackedDoubles, fmt.Sprintf("%s, %s, %s", acc, accs[0], accs[0]))
	}
}

// dotF64AVX2NarrowerLoop emits the 4-wide narrower tail for the AVX2 float64 dot product
// kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func dotF64AVX2NarrowerLoop(e *asmgen.Emitter) {
	e.Label("dotf64avx_tail4")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $4")
	inst(e, asmamd64.OperationJumpIfLessSigned, "dotf64avx_tail")
	e.Blank()
	e.Label("dotf64avx_loop4")
	inst(e, asmamd64.OperationVexMoveUnalignedPackedDoubles, "(SI), Y1")
	inst(e, asmamd64.OperationVexMoveUnalignedPackedDoubles, "(DI), Y2")
	inst(e, asmamd64.OperationVexFusedMultiplyAdd231PackedDoubles, "Y1, Y2, Y0")
	inst(e, asmamd64.OperationAdd64Bits, "$32, SI")
	inst(e, asmamd64.OperationAdd64Bits, "$32, DI")
	inst(e, asmamd64.OperationSubtract64Bits, "$4, CX")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $4")
	inst(e, asmamd64.OperationJumpIfGreaterOrEqualSigned, "dotf64avx_loop4")
	e.Blank()
}

// dotF64AVX2ScalarTail emits the scalar drain for the AVX2 float64 dot product kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func dotF64AVX2ScalarTail(e *asmgen.Emitter) {
	e.Label("dotf64avx_tail")
	inst(e, asmamd64.OperationTest64Bits, "CX, CX")
	inst(e, asmamd64.OperationJumpIfZero, "dotf64avx_reduce")
	e.Blank()
	e.Label("dotf64avx_tail_loop")
	inst(e, asmamd64.OperationVexMoveScalarDouble, "(SI), X1")
	inst(e, asmamd64.OperationVexMoveScalarDouble, "(DI), X2")
	inst(e, asmamd64.OperationVexMultiplyScalarDouble, "X2, X1, X1")
	inst(e, asmamd64.OperationVexAddScalarDouble, "X1, X3, X3")
	inst(e, asmamd64.OperationAdd64Bits, "$8, SI")
	inst(e, asmamd64.OperationAdd64Bits, "$8, DI")
	inst(e, asmamd64.OperationDecrement64Bits, "CX")
	inst(e, asmamd64.OperationJumpIfNotZero, "dotf64avx_tail_loop")
	e.Blank()
}

// dotF64AVX2ReduceAndReturn emits the horizontal reduce, scalar tail merge, VZEROUPPER,
// return-slot store, and final RET for the AVX2 float64 dot product kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes v (*amd64VectormathsOps) which provides the AVX2 horizontal reduce helper.
func dotF64AVX2ReduceAndReturn(e *asmgen.Emitter, v *amd64VectormathsOps) {
	e.Label("dotf64avx_reduce")
	v.emitAVX2DoubleHorizontalReduce(e)
	inst(e, asmamd64.OperationVexAddScalarDouble, "X3, X0, X0")
	e.Instruction(asmamd64.OperationVexZeroUpper)
	inst(e, asmamd64.OperationMoveScalarDouble, "X0, ret+48(FP)")
	e.Instruction(asmamd64.OperationReturn)
}

// scaleF64SSEPrologue emits the prologue for the SSE in-place float64 scale kernel:
// argument loads, broadcast of the scalar coefficient to both lanes of X7, and the
// unroll-width guard.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes unrollWidth (int) which is the float64 elements processed per main-loop
// iteration.
// Takes firstTail (string) which is the label jumped to when input is short.
func scaleF64SSEPrologue(e *asmgen.Emitter, unrollWidth int, firstTail string) {
	inst(e, asmamd64.OperationMove64Bits, "a_base+0(FP), SI")
	inst(e, asmamd64.OperationMove64Bits, "a_len+8(FP), CX")
	inst(e, asmamd64.OperationMoveScalarDouble, "k+24(FP), X7")
	inst(e, asmamd64.OperationShufflePackedDoubles, "$0, X7, X7")
	e.Blank()
	inst(e, asmamd64.OperationCompare64Bits, fmt.Sprintf("CX, $%d", unrollWidth))
	inst(e, asmamd64.OperationJumpIfLessSigned, firstTail)
}

// scaleF64SSESlotBody emits one body slot of the SSE in-place float64 scale main loop.
// Slot 0 uses X0 and slot 1 uses X2 so the two slots have independent load-multiply-store
// dependencies.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes off (int) which is the slot's byte offset and selects the working register.
func scaleF64SSESlotBody(e *asmgen.Emitter, off int) {
	reg := "X0"
	if off != 0 {
		reg = "X2"
	}
	inst(e, asmamd64.OperationMoveUnalignedPackedDoubles, fmt.Sprintf("%d(SI), %s", off, reg))
	inst(e, asmamd64.OperationMultiplyPackedDoubles, fmt.Sprintf("X7, %s", reg))
	inst(e, asmamd64.OperationMoveUnalignedPackedDoubles, fmt.Sprintf("%s, %d(SI)", reg, off))
}

// scaleF64SSELoopFooter emits the pointer-advance, counter decrement, and back branch for
// the SSE in-place float64 scale main loop.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes byteStride (int) which is the bytes per unrolled iteration.
func scaleF64SSELoopFooter(e *asmgen.Emitter, byteStride int) {
	unrollWidth := byteStride / 8
	inst(e, asmamd64.OperationAdd64Bits, fmt.Sprintf("$%d, SI", byteStride))
	inst(e, asmamd64.OperationSubtract64Bits, fmt.Sprintf("$%d, CX", unrollWidth))
	inst(e, asmamd64.OperationCompare64Bits, fmt.Sprintf("CX, $%d", unrollWidth))
	inst(e, asmamd64.OperationJumpIfGreaterOrEqualSigned, fmt.Sprintf("scalef64sse_loop%d", unrollWidth))
}

// scaleF64SSETail2 emits the 2-wide SSE straight-line tail for the SSE in-place float64
// scale kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func scaleF64SSETail2(e *asmgen.Emitter) {
	e.Label("scalef64sse_tail2")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $2")
	inst(e, asmamd64.OperationJumpIfLessSigned, "scalef64sse_tail")
	e.Blank()
	inst(e, asmamd64.OperationMoveUnalignedPackedDoubles, "(SI), X0")
	inst(e, asmamd64.OperationMultiplyPackedDoubles, "X7, X0")
	inst(e, asmamd64.OperationMoveUnalignedPackedDoubles, "X0, (SI)")
	inst(e, asmamd64.OperationAdd64Bits, "$16, SI")
	inst(e, asmamd64.OperationSubtract64Bits, "$2, CX")
	e.Blank()
}

// scaleF64SSEScalarTail emits the scalar drain for the SSE in-place float64 scale kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func scaleF64SSEScalarTail(e *asmgen.Emitter) {
	e.Label("scalef64sse_tail")
	inst(e, asmamd64.OperationTest64Bits, "CX, CX")
	inst(e, asmamd64.OperationJumpIfZero, "scalef64sse_done")
	e.Blank()
	e.Label("scalef64sse_tail_loop")
	inst(e, asmamd64.OperationMoveScalarDouble, "(SI), X0")
	inst(e, asmamd64.OperationMultiplyScalarDouble, "X7, X0")
	inst(e, asmamd64.OperationMoveScalarDouble, "X0, (SI)")
	inst(e, asmamd64.OperationAdd64Bits, "$8, SI")
	inst(e, asmamd64.OperationDecrement64Bits, "CX")
	inst(e, asmamd64.OperationJumpIfNotZero, "scalef64sse_tail_loop")
	e.Blank()
}

// scaleF64SSEReturn emits the done label and final RET for the SSE in-place float64 scale
// kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func scaleF64SSEReturn(e *asmgen.Emitter) {
	e.Label("scalef64sse_done")
	e.Instruction(asmamd64.OperationReturn)
}

// scaleF64AVX2Prologue emits the prologue for the AVX2 in-place float64 scale kernel:
// argument loads, VBROADCASTSD of k to all four lanes of Y7, and the unroll-width guard.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes unrollWidth (int) which is the float64 elements processed per main-loop
// iteration.
// Takes firstTail (string) which is the label jumped to when input is short.
func scaleF64AVX2Prologue(e *asmgen.Emitter, unrollWidth int, firstTail string) {
	inst(e, asmamd64.OperationMove64Bits, "a_base+0(FP), SI")
	inst(e, asmamd64.OperationMove64Bits, "a_len+8(FP), CX")
	inst(e, asmamd64.OperationVexBroadcastScalarDouble, "k+24(FP), Y7")
	e.Blank()
	inst(e, asmamd64.OperationCompare64Bits, fmt.Sprintf("CX, $%d", unrollWidth))
	inst(e, asmamd64.OperationJumpIfLessSigned, firstTail)
}

// scaleF64AVX2SlotBody emits one body slot of the AVX2 in-place float64 scale main loop.
// Each slot uses an independent YMM working register so the four slots can issue
// load-multiply-store streams in parallel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes off (int) which is the slot's byte offset in the unrolled iteration.
func scaleF64AVX2SlotBody(e *asmgen.Emitter, off int) {
	registers := []string{"Y0", "Y1", "Y2", "Y3"}
	reg := registers[off/32]
	inst(e, asmamd64.OperationVexMoveUnalignedPackedDoubles, fmt.Sprintf("%d(SI), %s", off, reg))
	inst(e, asmamd64.OperationVexMultiplyPackedDoubles, fmt.Sprintf("Y7, %s, %s", reg, reg))
	inst(e, asmamd64.OperationVexMoveUnalignedPackedDoubles, fmt.Sprintf("%s, %d(SI)", reg, off))
}

// scaleF64AVX2LoopFooter emits the pointer-advance, counter decrement, and back branch
// for the AVX2 in-place float64 scale main loop.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes byteStride (int) which is the bytes per unrolled iteration.
func scaleF64AVX2LoopFooter(e *asmgen.Emitter, byteStride int) {
	unrollWidth := byteStride / 8
	inst(e, asmamd64.OperationAdd64Bits, fmt.Sprintf("$%d, SI", byteStride))
	inst(e, asmamd64.OperationSubtract64Bits, fmt.Sprintf("$%d, CX", unrollWidth))
	inst(e, asmamd64.OperationCompare64Bits, fmt.Sprintf("CX, $%d", unrollWidth))
	inst(e, asmamd64.OperationJumpIfGreaterOrEqualSigned, fmt.Sprintf("scalef64avx_loop%d", unrollWidth))
}

// scaleF64AVX2Tail4 emits the 4-wide AVX2 loop tail for the AVX2 in-place float64 scale
// kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func scaleF64AVX2Tail4(e *asmgen.Emitter) {
	e.Label("scalef64avx_tail4")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $4")
	inst(e, asmamd64.OperationJumpIfLessSigned, "scalef64avx_tail2")
	e.Blank()
	e.Label("scalef64avx_loop4")
	inst(e, asmamd64.OperationVexMoveUnalignedPackedDoubles, "(SI), Y0")
	inst(e, asmamd64.OperationVexMultiplyPackedDoubles, "Y7, Y0, Y0")
	inst(e, asmamd64.OperationVexMoveUnalignedPackedDoubles, "Y0, (SI)")
	inst(e, asmamd64.OperationAdd64Bits, "$32, SI")
	inst(e, asmamd64.OperationSubtract64Bits, "$4, CX")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $4")
	inst(e, asmamd64.OperationJumpIfGreaterOrEqualSigned, "scalef64avx_loop4")
	e.Blank()
}

// scaleF64AVX2Tail2 emits the 2-wide SSE straight-line tail (no loop) for the AVX2
// in-place float64 scale kernel. The low half of Y7 already holds k in both lanes after
// VBROADCASTSD, so MULPD with X7 is well-defined.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func scaleF64AVX2Tail2(e *asmgen.Emitter) {
	e.Label("scalef64avx_tail2")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $2")
	inst(e, asmamd64.OperationJumpIfLessSigned, "scalef64avx_tail1")
	e.Blank()
	inst(e, asmamd64.OperationMoveUnalignedPackedDoubles, "(SI), X0")
	inst(e, asmamd64.OperationMultiplyPackedDoubles, "X7, X0")
	inst(e, asmamd64.OperationMoveUnalignedPackedDoubles, "X0, (SI)")
	inst(e, asmamd64.OperationAdd64Bits, "$16, SI")
	inst(e, asmamd64.OperationSubtract64Bits, "$2, CX")
	e.Blank()
}

// scaleF64AVX2ScalarTail emits the scalar drain for the AVX2 in-place float64 scale
// kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func scaleF64AVX2ScalarTail(e *asmgen.Emitter) {
	e.Label("scalef64avx_tail1")
	inst(e, asmamd64.OperationTest64Bits, "CX, CX")
	inst(e, asmamd64.OperationJumpIfZero, "scalef64avx_done")
	e.Blank()
	e.Label("scalef64avx_tail_loop")
	inst(e, asmamd64.OperationMoveScalarDouble, "(SI), X0")
	inst(e, asmamd64.OperationMultiplyScalarDouble, "X7, X0")
	inst(e, asmamd64.OperationMoveScalarDouble, "X0, (SI)")
	inst(e, asmamd64.OperationAdd64Bits, "$8, SI")
	inst(e, asmamd64.OperationDecrement64Bits, "CX")
	inst(e, asmamd64.OperationJumpIfNotZero, "scalef64avx_tail_loop")
	e.Blank()
}

// scaleF64AVX2Return emits the done label, VZEROUPPER, and final RET for the AVX2
// in-place float64 scale kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func scaleF64AVX2Return(e *asmgen.Emitter) {
	e.Label("scalef64avx_done")
	e.Instruction(asmamd64.OperationVexZeroUpper)
	e.Instruction(asmamd64.OperationReturn)
}
