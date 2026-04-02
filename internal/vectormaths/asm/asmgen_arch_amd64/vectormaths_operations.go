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

import "piko.sh/piko/wdk/asmgen"

const (
	// mnemonicMOVQ represents the MOVQ x86-64 assembly mnemonic.
	mnemonicMOVQ = "MOVQ"

	// mnemonicMOVSS represents the MOVSS x86-64 assembly mnemonic.
	mnemonicMOVSS = "MOVSS"

	// mnemonicMOVUPS represents the MOVUPS x86-64 assembly mnemonic.
	mnemonicMOVUPS = "MOVUPS"

	// mnemonicMOVL represents the MOVL x86-64 assembly mnemonic.
	mnemonicMOVL = "MOVL"

	// mnemonicADDQ represents the ADDQ x86-64 assembly mnemonic.
	mnemonicADDQ = "ADDQ"

	// mnemonicSUBQ represents the SUBQ x86-64 assembly mnemonic.
	mnemonicSUBQ = "SUBQ"

	// mnemonicCMPQ represents the CMPQ x86-64 assembly mnemonic.
	mnemonicCMPQ = "CMPQ"

	// mnemonicDECQ represents the DECQ x86-64 assembly mnemonic.
	mnemonicDECQ = "DECQ"

	// mnemonicTESTQ represents the TESTQ x86-64 assembly mnemonic.
	mnemonicTESTQ = "TESTQ"

	// mnemonicADDPS represents the ADDPS x86-64 SSE assembly mnemonic.
	mnemonicADDPS = "ADDPS"

	// mnemonicADDSS represents the ADDSS x86-64 SSE assembly mnemonic.
	mnemonicADDSS = "ADDSS"

	// mnemonicMULPS represents the MULPS x86-64 SSE assembly mnemonic.
	mnemonicMULPS = "MULPS"

	// mnemonicMULSS represents the MULSS x86-64 SSE assembly mnemonic.
	mnemonicMULSS = "MULSS"

	// mnemonicXORPS represents the XORPS x86-64 SSE assembly mnemonic.
	mnemonicXORPS = "XORPS"

	// mnemonicVMOVUPS represents the VMOVUPS x86-64 AVX assembly mnemonic.
	mnemonicVMOVUPS = "VMOVUPS"

	// mnemonicVMOVSS represents the VMOVSS x86-64 AVX assembly mnemonic.
	mnemonicVMOVSS = "VMOVSS"

	// mnemonicVADDPS represents the VADDPS x86-64 AVX assembly mnemonic.
	mnemonicVADDPS = "VADDPS"

	// mnemonicVADDSS represents the VADDSS x86-64 AVX assembly mnemonic.
	mnemonicVADDSS = "VADDSS"

	// mnemonicVMULPS represents the VMULPS x86-64 AVX assembly mnemonic.
	mnemonicVMULPS = "VMULPS"

	// mnemonicVMULSS represents the VMULSS x86-64 AVX assembly mnemonic.
	mnemonicVMULSS = "VMULSS"

	// mnemonicVXORPS represents the VXORPS x86-64 AVX assembly mnemonic.
	mnemonicVXORPS = "VXORPS"

	// mnemonicJGE represents the JGE x86-64 assembly mnemonic.
	mnemonicJGE = "JGE"

	// mnemonicJL represents the JL x86-64 assembly mnemonic.
	mnemonicJL = "JL"

	// mnemonicJNZ represents the JNZ x86-64 assembly mnemonic.
	mnemonicJNZ = "JNZ"

	// mnemonicJZ represents the JZ x86-64 assembly mnemonic.
	mnemonicJZ = "JZ"

	// mnemonicJE represents the JE x86-64 assembly mnemonic.
	mnemonicJE = "JE"

	// mnemonicRET represents the RET x86-64 assembly mnemonic.
	mnemonicRET = "RET"

	// mnemonicVZEROUPPER represents the VZEROUPPER x86-64 AVX assembly mnemonic.
	mnemonicVZEROUPPER = "VZEROUPPER"

	// operandSliceABaseSI loads slice a's base pointer into SI.
	operandSliceABaseSI = "a_base+0(FP), SI"

	// operandSliceALenCX loads slice a's length into CX.
	operandSliceALenCX = "a_len+8(FP), CX"

	// operandSliceBBaseDI loads slice b's base pointer into DI.
	operandSliceBBaseDI = "b_base+24(FP), DI"

	// operandSliceVBaseSI loads slice v's base pointer into SI.
	operandSliceVBaseSI = "v_base+0(FP), SI"

	// operandSliceVLenCX loads slice v's length into CX.
	operandSliceVLenCX = "v_len+8(FP), CX"

	// operandReturnF32 stores the float32 result from X0 to the return slot.
	operandReturnF32 = "X0, ret+48(FP)"

	// operandTestCXSelf tests CX against itself to set flags.
	operandTestCXSelf = "CX, CX"

	// operandClearX0 clears the X0 register via XOR with itself.
	operandClearX0 = "X0, X0"

	// operandCompare4 compares CX against the immediate value 4.
	operandCompare4 = "CX, $4"

	// operandCompare8 compares CX against the immediate value 8.
	operandCompare8 = "CX, $8"

	// operandCompare0 compares CX against the immediate value 0.
	operandCompare0 = "CX, $0"

	// operandAdd16SI advances the SI pointer by 16 bytes.
	operandAdd16SI = "$16, SI"

	// operandAdd4SI advances the SI pointer by 4 bytes.
	operandAdd4SI = "$4, SI"

	// operandAdd16DI advances the DI pointer by 16 bytes.
	operandAdd16DI = "$16, DI"

	// operandAdd4DI advances the DI pointer by 4 bytes.
	operandAdd4DI = "$4, DI"

	// operandSub4CX decrements CX by 4.
	operandSub4CX = "$4, CX"

	// operandSub8CX decrements CX by 8.
	operandSub8CX = "$8, CX"

	// operandAdd32SI advances the SI pointer by 32 bytes.
	operandAdd32SI = "$32, SI"

	// operandAdd32DI advances the DI pointer by 32 bytes.
	operandAdd32DI = "$32, DI"

	// operandIEEE754One loads the IEEE 754 bit pattern for 1.0f into AX.
	operandIEEE754One = "$0x3F800000, AX"

	// operandLoadX1FromSI loads a float32 from the address in SI into X1.
	operandLoadX1FromSI = "(SI), X1"

	// operandLoadX2FromDI loads a float32 from the address in DI into X2.
	operandLoadX2FromDI = "(DI), X2"

	// operandMulX2X1 multiplies X2 into X1.
	operandMulX2X1 = "X2, X1"

	// operandSquareX1 squares X1 by multiplying it with itself.
	operandSquareX1 = "X1, X1"

	// operandAccumX1ToX0 accumulates X1 into X0.
	operandAccumX1ToX0 = "X1, X0"

	// operandDecCX is the operand for decrementing CX.
	operandDecCX = "CX"
)

// amd64VectormathsOps implements VectormathsOperationsPort for x86-64.
// Each method emits the complete function body (after the TEXT
// directive) for a SIMD vectormaths kernel in the requested variant.
type amd64VectormathsOps struct{}

// Ensure amd64VectormathsOps implements VectormathsOperationsPort at compile time.
var _ asmgen.VectormathsOperationsPort = (*amd64VectormathsOps)(nil)

// EmitDotProduct emits the dot product function body for the given SIMD variant.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes variant (string) which must be "SSE" or "AVX2".
func (v *amd64VectormathsOps) EmitDotProduct(e *asmgen.Emitter, variant string) {
	switch variant {
	case "SSE":
		v.emitDotProductSSE(e)
	case "AVX2":
		v.emitDotProductAVX2(e)
	}
}

// EmitEuclideanDistanceSquared emits the squared Euclidean
// distance function body for the given SIMD variant.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes variant (string) which must be "SSE" or "AVX2".
func (v *amd64VectormathsOps) EmitEuclideanDistanceSquared(e *asmgen.Emitter, variant string) {
	switch variant {
	case "SSE":
		v.emitEuclideanDistanceSquaredSSE(e)
	case "AVX2":
		v.emitEuclideanDistanceSquaredAVX2(e)
	}
}

// EmitNormalise emits the in-place L2 normalisation function
// body for the given SIMD variant.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes variant (string) which must be "SSE" or "AVX2".
func (v *amd64VectormathsOps) EmitNormalise(e *asmgen.Emitter, variant string) {
	switch variant {
	case "SSE":
		v.emitNormaliseSSE(e)
	case "AVX2":
		v.emitNormaliseAVX2(e)
	}
}

// emitSSEHorizontalReduce emits the SSE horizontal reduction sequence
// that collapses the four float32 lanes of X0 into a single scalar in
// the low lane of X0.
//
// The algorithm moves the high pair down (MOVHLPS), adds the two
// halves, then shuffles the second element into X1 and adds once
// more, leaving the scalar sum in X0[0].
//
// Expects the packed accumulator in X0. Uses X1 as scratch.
// Produces a scalar sum in the low lane of X0.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (*amd64VectormathsOps) emitSSEHorizontalReduce(e *asmgen.Emitter) {
	inst(e, "MOVHLPS", "X0, X1")
	inst(e, mnemonicADDPS, operandAccumX1ToX0)
	inst(e, "MOVAPS", "X0, X1")
	inst(e, "SHUFPS", "$0x55, X1, X1")
	inst(e, mnemonicADDSS, operandAccumX1ToX0)
}

// emitAVX2HorizontalReduce emits the AVX2 horizontal reduction
// sequence that collapses the eight float32 lanes of Y0 into a single
// scalar in the low lane of X0.
//
// The algorithm first extracts the upper 128-bit half of Y0 into X1
// (VEXTRACTF128), adds it to the lower half, then performs two
// cross-lane shuffles (VSHUFPS) with adds to reduce the four
// remaining lanes down to one scalar.
//
// Expects the packed accumulator in Y0/X0. Uses X1 as scratch.
// Produces a scalar sum in the low lane of X0.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (*amd64VectormathsOps) emitAVX2HorizontalReduce(e *asmgen.Emitter) {
	inst(e, "VEXTRACTF128", "$1, Y0, X1")
	inst(e, mnemonicVADDPS, "X1, X0, X0")
	inst(e, "VSHUFPS", "$0x4E, X0, X0, X1")
	inst(e, mnemonicVADDPS, "X1, X0, X0")
	inst(e, "VSHUFPS", "$0x55, X0, X0, X1")
	inst(e, mnemonicVADDSS, "X1, X0, X0")
}

// emitDotProductSSE emits the complete SSE dot product function body.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (v *amd64VectormathsOps) emitDotProductSSE(e *asmgen.Emitter) {
	inst(e, mnemonicMOVQ, operandSliceABaseSI)
	inst(e, mnemonicMOVQ, operandSliceALenCX)
	inst(e, mnemonicMOVQ, operandSliceBBaseDI)
	e.Blank()
	inst(e, mnemonicXORPS, operandClearX0)
	inst(e, mnemonicXORPS, "X3, X3")
	e.Blank()
	inst(e, mnemonicCMPQ, operandCompare4)
	inst(e, mnemonicJL, "dotsse_tail")
	e.Blank()
	v.emitDotProductSSEVectorLoop(e)
	v.emitDotProductSSEScalarTail(e)
	v.emitDotProductSSEReduceAndReturn(e)
}

// emitDotProductSSEVectorLoop emits the main four-wide SSE loop body
// for the dot product.
//
// On each iteration, four float32 pairs are loaded from (SI) and
// (DI), multiplied together, and accumulated into X0. The pointers
// advance by 16 bytes and the remaining count in CX is decremented
// by 4. The loop continues while CX >= 4.
//
// Expects SI and DI pointing to the current positions in vectors a
// and b, CX holding the remaining element count, and X0 initialised
// to zero. Uses X1 and X2 as scratch.
// Produces the partial packed sum in X0, with SI/DI/CX advanced past
// all complete 4-element groups.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (*amd64VectormathsOps) emitDotProductSSEVectorLoop(e *asmgen.Emitter) {
	e.Label("dotsse_loop4")
	inst(e, mnemonicMOVUPS, operandLoadX1FromSI)
	inst(e, mnemonicMOVUPS, operandLoadX2FromDI)
	inst(e, mnemonicMULPS, operandMulX2X1)
	inst(e, mnemonicADDPS, operandAccumX1ToX0)
	inst(e, mnemonicADDQ, operandAdd16SI)
	inst(e, mnemonicADDQ, operandAdd16DI)
	inst(e, mnemonicSUBQ, operandSub4CX)
	inst(e, mnemonicCMPQ, operandCompare4)
	inst(e, mnemonicJGE, "dotsse_loop4")
	e.Blank()
}

// emitDotProductSSEScalarTail emits the scalar tail loop that
// handles any remaining elements (fewer than 4) after the main SSE
// vector loop.
//
// Each remaining pair of float32 values is loaded from (SI) and (DI),
// multiplied, and accumulated into the scalar register X3. The
// pointers advance by 4 bytes and CX is decremented by 1 on each
// iteration.
//
// Expects SI and DI at the first unprocessed element, CX holding the
// remaining count (0..3), and X3 initialised to zero.
// Produces the scalar tail sum in X3.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (*amd64VectormathsOps) emitDotProductSSEScalarTail(e *asmgen.Emitter) {
	e.Label("dotsse_tail")
	inst(e, mnemonicTESTQ, operandTestCXSelf)
	inst(e, mnemonicJZ, "dotsse_reduce")
	e.Blank()
	e.Label("dotsse_tail_loop")
	inst(e, mnemonicMOVSS, operandLoadX1FromSI)
	inst(e, mnemonicMOVSS, operandLoadX2FromDI)
	inst(e, mnemonicMULSS, operandMulX2X1)
	inst(e, mnemonicADDSS, "X1, X3")
	inst(e, mnemonicADDQ, operandAdd4SI)
	inst(e, mnemonicADDQ, operandAdd4DI)
	inst(e, mnemonicDECQ, operandDecCX)
	inst(e, mnemonicJNZ, "dotsse_tail_loop")
	e.Blank()
}

// emitDotProductSSEReduceAndReturn emits the horizontal reduction
// of the packed accumulator X0, merges the scalar tail sum from X3,
// stores the final result to the return slot, and returns.
//
// After horizontal reduction collapses X0 to a scalar, the tail
// sum in X3 is added and the result is written to ret+48(FP).
//
// Expects the packed sum in X0 and the scalar tail sum in X3.
// Produces the final dot product scalar in ret+48(FP).
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (v *amd64VectormathsOps) emitDotProductSSEReduceAndReturn(e *asmgen.Emitter) {
	e.Label("dotsse_reduce")
	v.emitSSEHorizontalReduce(e)
	e.Blank()
	inst(e, mnemonicADDSS, "X3, X0")
	e.Blank()
	inst(e, mnemonicMOVSS, operandReturnF32)
	e.Instruction(mnemonicRET)
}

// emitDotProductAVX2 emits the complete AVX2 dot product function body.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (v *amd64VectormathsOps) emitDotProductAVX2(e *asmgen.Emitter) {
	inst(e, mnemonicMOVQ, operandSliceABaseSI)
	inst(e, mnemonicMOVQ, operandSliceALenCX)
	inst(e, mnemonicMOVQ, operandSliceBBaseDI)
	e.Blank()
	inst(e, mnemonicVXORPS, "Y0, Y0, Y0")
	inst(e, mnemonicVXORPS, "X3, X3, X3")
	e.Blank()
	inst(e, mnemonicCMPQ, operandCompare8)
	inst(e, mnemonicJL, "dotavx_tail")
	e.Blank()
	v.emitDotProductAVX2VectorLoop(e)
	v.emitDotProductAVX2ScalarTail(e)
	v.emitDotProductAVX2ReduceAndReturn(e)
}

// emitDotProductAVX2VectorLoop emits the main eight-wide AVX2 loop
// body for the dot product.
//
// On each iteration, eight float32 pairs are loaded from (SI) and
// (DI), multiplied with three-operand VMULPS, and accumulated into
// Y0. The pointers advance by 32 bytes and the count in CX is
// decremented by 8. The loop continues while CX >= 8.
//
// Expects SI and DI at the current vector positions, CX holding the
// remaining count, and Y0 initialised to zero. Uses Y1 and Y2 as
// scratch.
// Produces the partial packed sum in Y0, with SI/DI/CX advanced past
// all complete 8-element groups.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (*amd64VectormathsOps) emitDotProductAVX2VectorLoop(e *asmgen.Emitter) {
	e.Label("dotavx_loop8")
	inst(e, mnemonicVMOVUPS, "(SI), Y1")
	inst(e, mnemonicVMOVUPS, "(DI), Y2")
	inst(e, mnemonicVMULPS, "Y2, Y1, Y1")
	inst(e, mnemonicVADDPS, "Y1, Y0, Y0")
	inst(e, mnemonicADDQ, operandAdd32SI)
	inst(e, mnemonicADDQ, operandAdd32DI)
	inst(e, mnemonicSUBQ, operandSub8CX)
	inst(e, mnemonicCMPQ, operandCompare8)
	inst(e, mnemonicJGE, "dotavx_loop8")
	e.Blank()
}

// emitDotProductAVX2ScalarTail emits the scalar tail loop that
// handles remaining elements (fewer than 8) after the main AVX2
// vector loop.
//
// Each remaining pair is loaded via VMOVSS, multiplied, and
// accumulated into X3. The pointers advance by 4 bytes and CX is
// decremented by 1.
//
// Expects SI and DI at the first unprocessed element, CX holding the
// remaining count (0..7), and X3 initialised to zero.
// Produces the scalar tail sum in X3.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (*amd64VectormathsOps) emitDotProductAVX2ScalarTail(e *asmgen.Emitter) {
	e.Label("dotavx_tail")
	inst(e, mnemonicTESTQ, operandTestCXSelf)
	inst(e, mnemonicJZ, "dotavx_reduce")
	e.Blank()
	e.Label("dotavx_tail_loop")
	inst(e, mnemonicVMOVSS, operandLoadX1FromSI)
	inst(e, mnemonicVMOVSS, operandLoadX2FromDI)
	inst(e, mnemonicVMULSS, "X2, X1, X1")
	inst(e, mnemonicVADDSS, "X1, X3, X3")
	inst(e, mnemonicADDQ, operandAdd4SI)
	inst(e, mnemonicADDQ, operandAdd4DI)
	inst(e, mnemonicDECQ, operandDecCX)
	inst(e, mnemonicJNZ, "dotavx_tail_loop")
	e.Blank()
}

// emitDotProductAVX2ReduceAndReturn emits the AVX2 horizontal
// reduction of the packed accumulator Y0, merges the scalar tail sum
// from X3, issues VZEROUPPER, stores the final result, and returns.
//
// After the 256-bit to scalar reduction, X3 is added, and the result
// is written to ret+48(FP). VZEROUPPER clears the upper halves of
// all YMM registers to avoid SSE/AVX transition penalties.
//
// Expects the packed sum in Y0/X0 and the scalar tail sum in X3.
// Produces the final dot product scalar in ret+48(FP).
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (v *amd64VectormathsOps) emitDotProductAVX2ReduceAndReturn(e *asmgen.Emitter) {
	e.Label("dotavx_reduce")
	v.emitAVX2HorizontalReduce(e)
	e.Blank()
	inst(e, mnemonicVADDSS, "X3, X0, X0")
	e.Blank()
	e.Instruction(mnemonicVZEROUPPER)
	inst(e, mnemonicMOVSS, operandReturnF32)
	e.Instruction(mnemonicRET)
}

// emitEuclideanDistanceSquaredSSE emits the complete SSE
// squared Euclidean distance function body.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (v *amd64VectormathsOps) emitEuclideanDistanceSquaredSSE(e *asmgen.Emitter) {
	inst(e, mnemonicMOVQ, operandSliceABaseSI)
	inst(e, mnemonicMOVQ, operandSliceALenCX)
	inst(e, mnemonicMOVQ, operandSliceBBaseDI)
	e.Blank()
	inst(e, mnemonicXORPS, operandClearX0)
	inst(e, mnemonicXORPS, "X3, X3")
	e.Blank()
	inst(e, mnemonicCMPQ, operandCompare4)
	inst(e, mnemonicJL, "euclidsse_tail")
	e.Blank()
	v.emitEuclideanDistanceSquaredSSEVectorLoop(e)
	v.emitEuclideanDistanceSquaredSSEScalarTail(e)
	v.emitEuclideanDistanceSquaredSSEReduceAndReturn(e)
}

// emitEuclideanDistanceSquaredSSEVectorLoop emits the main four-wide
// SSE loop body for squared Euclidean distance.
//
// On each iteration, four float32 pairs are loaded from (SI) and
// (DI), subtracted (SUBPS), squared (MULPS self), and accumulated
// into X0. The pointers advance by 16 bytes and CX is decremented
// by 4. The loop continues while CX >= 4.
//
// Expects SI and DI at the current vector positions, CX holding the
// remaining count, and X0 initialised to zero. Uses X1 and X2 as
// scratch.
// Produces the partial packed sum-of-squared-differences in X0.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (*amd64VectormathsOps) emitEuclideanDistanceSquaredSSEVectorLoop(e *asmgen.Emitter) {
	e.Label("euclidsse_loop4")
	inst(e, mnemonicMOVUPS, operandLoadX1FromSI)
	inst(e, mnemonicMOVUPS, operandLoadX2FromDI)
	inst(e, "SUBPS", operandMulX2X1)
	inst(e, mnemonicMULPS, operandSquareX1)
	inst(e, mnemonicADDPS, operandAccumX1ToX0)
	inst(e, mnemonicADDQ, operandAdd16SI)
	inst(e, mnemonicADDQ, operandAdd16DI)
	inst(e, mnemonicSUBQ, operandSub4CX)
	inst(e, mnemonicCMPQ, operandCompare4)
	inst(e, mnemonicJGE, "euclidsse_loop4")
	e.Blank()
}

// emitEuclideanDistanceSquaredSSEScalarTail emits the scalar tail
// loop for squared Euclidean distance, handling remaining elements
// (fewer than 4) after the main SSE vector loop.
//
// Each remaining pair is loaded, subtracted, squared, and accumulated
// into X3. The pointers advance by 4 bytes and CX is decremented
// by 1.
//
// Expects SI and DI at the first unprocessed element, CX holding the
// remaining count (0..3), and X3 initialised to zero.
// Produces the scalar tail sum in X3.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (*amd64VectormathsOps) emitEuclideanDistanceSquaredSSEScalarTail(e *asmgen.Emitter) {
	e.Label("euclidsse_tail")
	inst(e, mnemonicTESTQ, operandTestCXSelf)
	inst(e, mnemonicJZ, "euclidsse_reduce")
	e.Blank()
	e.Label("euclidsse_tail_loop")
	inst(e, mnemonicMOVSS, operandLoadX1FromSI)
	inst(e, mnemonicMOVSS, operandLoadX2FromDI)
	inst(e, "SUBSS", operandMulX2X1)
	inst(e, mnemonicMULSS, operandSquareX1)
	inst(e, mnemonicADDSS, "X1, X3")
	inst(e, mnemonicADDQ, operandAdd4SI)
	inst(e, mnemonicADDQ, operandAdd4DI)
	inst(e, mnemonicDECQ, operandDecCX)
	inst(e, mnemonicJNZ, "euclidsse_tail_loop")
	e.Blank()
}

// emitEuclideanDistanceSquaredSSEReduceAndReturn emits the horizontal
// reduction of the packed accumulator X0, merges the scalar tail sum
// from X3, stores the final result to the return slot, and returns.
//
// After horizontal reduction collapses X0 to a scalar, the tail sum
// in X3 is added and the result is written to ret+48(FP).
//
// Expects the packed sum in X0 and the scalar tail sum in X3.
// Produces the final squared Euclidean distance scalar in ret+48(FP).
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (v *amd64VectormathsOps) emitEuclideanDistanceSquaredSSEReduceAndReturn(e *asmgen.Emitter) {
	e.Label("euclidsse_reduce")
	v.emitSSEHorizontalReduce(e)
	e.Blank()
	inst(e, mnemonicADDSS, "X3, X0")
	e.Blank()
	inst(e, mnemonicMOVSS, operandReturnF32)
	e.Instruction(mnemonicRET)
}

// emitEuclideanDistanceSquaredAVX2 emits the complete AVX2
// squared Euclidean distance function body.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (v *amd64VectormathsOps) emitEuclideanDistanceSquaredAVX2(e *asmgen.Emitter) {
	inst(e, mnemonicMOVQ, operandSliceABaseSI)
	inst(e, mnemonicMOVQ, operandSliceALenCX)
	inst(e, mnemonicMOVQ, operandSliceBBaseDI)
	e.Blank()
	inst(e, mnemonicVXORPS, "Y0, Y0, Y0")
	inst(e, mnemonicVXORPS, "X3, X3, X3")
	e.Blank()
	inst(e, mnemonicCMPQ, operandCompare8)
	inst(e, mnemonicJL, "euclidavx_tail")
	e.Blank()
	v.emitEuclideanDistanceSquaredAVX2VectorLoop(e)
	v.emitEuclideanDistanceSquaredAVX2ScalarTail(e)
	v.emitEuclideanDistanceSquaredAVX2ReduceAndReturn(e)
}

// emitEuclideanDistanceSquaredAVX2VectorLoop emits the main
// eight-wide AVX2 loop body for squared Euclidean distance.
//
// On each iteration, eight float32 pairs are loaded from (SI) and
// (DI), subtracted (VSUBPS), squared (VMULPS self), and accumulated
// into Y0. The pointers advance by 32 bytes and CX is decremented
// by 8. The loop continues while CX >= 8.
//
// Expects SI and DI at the current vector positions, CX holding the
// remaining count, and Y0 initialised to zero. Uses Y1 and Y2 as
// scratch.
// Produces the partial packed sum-of-squared-differences in Y0.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (*amd64VectormathsOps) emitEuclideanDistanceSquaredAVX2VectorLoop(e *asmgen.Emitter) {
	e.Label("euclidavx_loop8")
	inst(e, mnemonicVMOVUPS, "(SI), Y1")
	inst(e, mnemonicVMOVUPS, "(DI), Y2")
	inst(e, "VSUBPS", "Y2, Y1, Y1")
	inst(e, mnemonicVMULPS, "Y1, Y1, Y1")
	inst(e, mnemonicVADDPS, "Y1, Y0, Y0")
	inst(e, mnemonicADDQ, operandAdd32SI)
	inst(e, mnemonicADDQ, operandAdd32DI)
	inst(e, mnemonicSUBQ, operandSub8CX)
	inst(e, mnemonicCMPQ, operandCompare8)
	inst(e, mnemonicJGE, "euclidavx_loop8")
	e.Blank()
}

// emitEuclideanDistanceSquaredAVX2ScalarTail emits the scalar tail
// loop for squared Euclidean distance, handling remaining elements
// (fewer than 8) after the main AVX2 vector loop.
//
// Each remaining pair is loaded via VMOVSS, subtracted, squared, and
// accumulated into X3. The pointers advance by 4 bytes and CX is
// decremented by 1.
//
// Expects SI and DI at the first unprocessed element, CX holding the
// remaining count (0..7), and X3 initialised to zero.
// Produces the scalar tail sum in X3.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (*amd64VectormathsOps) emitEuclideanDistanceSquaredAVX2ScalarTail(e *asmgen.Emitter) {
	e.Label("euclidavx_tail")
	inst(e, mnemonicTESTQ, operandTestCXSelf)
	inst(e, mnemonicJZ, "euclidavx_reduce")
	e.Blank()
	e.Label("euclidavx_tail_loop")
	inst(e, mnemonicVMOVSS, operandLoadX1FromSI)
	inst(e, mnemonicVMOVSS, operandLoadX2FromDI)
	inst(e, "VSUBSS", "X2, X1, X1")
	inst(e, mnemonicVMULSS, "X1, X1, X1")
	inst(e, mnemonicVADDSS, "X1, X3, X3")
	inst(e, mnemonicADDQ, operandAdd4SI)
	inst(e, mnemonicADDQ, operandAdd4DI)
	inst(e, mnemonicDECQ, operandDecCX)
	inst(e, mnemonicJNZ, "euclidavx_tail_loop")
	e.Blank()
}

// emitEuclideanDistanceSquaredAVX2ReduceAndReturn emits the AVX2
// horizontal reduction of the packed accumulator Y0, merges the
// scalar tail sum from X3, issues VZEROUPPER, stores the final
// result, and returns.
//
// After the 256-bit to scalar reduction, X3 is added, and the result
// is written to ret+48(FP). VZEROUPPER clears the upper halves of
// all YMM registers to avoid SSE/AVX transition penalties.
//
// Expects the packed sum in Y0/X0 and the scalar tail sum in X3.
// Produces the final squared Euclidean distance scalar in ret+48(FP).
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (v *amd64VectormathsOps) emitEuclideanDistanceSquaredAVX2ReduceAndReturn(e *asmgen.Emitter) {
	e.Label("euclidavx_reduce")
	v.emitAVX2HorizontalReduce(e)
	e.Blank()
	inst(e, mnemonicVADDSS, "X3, X0, X0")
	e.Blank()
	e.Instruction(mnemonicVZEROUPPER)
	inst(e, mnemonicMOVSS, operandReturnF32)
	e.Instruction(mnemonicRET)
}

// emitNormaliseSSE emits the complete SSE in-place normalisation function body.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (v *amd64VectormathsOps) emitNormaliseSSE(e *asmgen.Emitter) {
	v.emitNormaliseSSESumOfSquares(e)
	v.emitNormaliseSSEZeroCheckAndReciprocalSqrt(e)
	v.emitNormaliseSSEWriteBack(e)
}

// emitNormaliseSSESumOfSquares emits phase 1 of SSE normalisation:
// computing the sum of squares of every element.
//
// The vector loop loads four float32 values from (SI), squares them
// (MULPS self), and accumulates into X0. The scalar tail loop handles
// leftover elements, accumulating into X3. After both loops, the
// packed accumulator X0 and scalar tail X3 are horizontally reduced
// into a single scalar in X0.
//
// Expects the function arguments v_base+0(FP) and v_len+8(FP) on
// the stack frame. Uses SI, CX, X0, X1, X3.
// Produces the total sum of squares as a scalar in X0.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (v *amd64VectormathsOps) emitNormaliseSSESumOfSquares(e *asmgen.Emitter) {
	inst(e, mnemonicMOVQ, operandSliceVBaseSI)
	inst(e, mnemonicMOVQ, operandSliceVLenCX)
	inst(e, mnemonicXORPS, operandClearX0)
	inst(e, mnemonicXORPS, "X3, X3")
	e.Blank()
	inst(e, mnemonicCMPQ, operandCompare4)
	inst(e, mnemonicJL, "normsse_tail")
	e.Blank()
	e.Label("normsse_loop4")
	inst(e, mnemonicMOVUPS, operandLoadX1FromSI)
	inst(e, mnemonicMULPS, operandSquareX1)
	inst(e, mnemonicADDPS, operandAccumX1ToX0)
	inst(e, mnemonicADDQ, operandAdd16SI)
	inst(e, mnemonicSUBQ, operandSub4CX)
	inst(e, mnemonicCMPQ, operandCompare4)
	inst(e, mnemonicJGE, "normsse_loop4")
	e.Blank()
	e.Label("normsse_tail")
	inst(e, mnemonicCMPQ, operandCompare0)
	inst(e, mnemonicJE, "normsse_reduce")
	e.Blank()
	e.Label("normsse_tail_loop")
	inst(e, mnemonicMOVSS, operandLoadX1FromSI)
	inst(e, mnemonicMULSS, operandSquareX1)
	inst(e, mnemonicADDSS, "X1, X3")
	inst(e, mnemonicADDQ, operandAdd4SI)
	inst(e, mnemonicDECQ, operandDecCX)
	inst(e, mnemonicJNZ, "normsse_tail_loop")
	e.Blank()

	e.Label("normsse_reduce")
	inst(e, mnemonicADDPS, "X3, X0")
	v.emitSSEHorizontalReduce(e)
	e.Blank()
}

// emitNormaliseSSEZeroCheckAndReciprocalSqrt emits phase 2 of SSE
// normalisation: the zero-vector check and reciprocal square root
// computation.
//
// If the sum of squares in X0 is exactly zero (checked via UCOMISS),
// the function jumps to normsse_done, leaving the zero vector
// unchanged. Otherwise, SQRTSS computes the square root, and the
// reciprocal (1.0/sqrt) is computed via DIVSS. The reciprocal is
// then broadcast to all four lanes of X1 via SHUFPS for use by the
// write-back phase.
//
// Expects the sum of squares as a scalar in X0.
// Produces the reciprocal norm broadcast in all lanes of X1, or
// jumps to normsse_done if the vector is zero.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (*amd64VectormathsOps) emitNormaliseSSEZeroCheckAndReciprocalSqrt(e *asmgen.Emitter) {
	inst(e, mnemonicXORPS, "X4, X4")
	inst(e, "UCOMISS", "X4, X0")
	inst(e, mnemonicJE, "normsse_done")
	inst(e, "SQRTSS", "X0, X0")
	inst(e, mnemonicMOVL, operandIEEE754One)
	inst(e, mnemonicMOVQ, "AX, X1")
	inst(e, "DIVSS", "X0, X1")
	inst(e, "SHUFPS", "$0x00, X1, X1")
	e.Blank()
}

// emitNormaliseSSEWriteBack emits phase 3 of SSE normalisation:
// multiplying every element by the reciprocal norm and writing the
// results back in place.
//
// The vector loop loads four float32 values from (SI), multiplies by
// the broadcast reciprocal in X1, and stores back. The scalar tail
// loop handles leftover elements. Both loops re-read v_base and
// v_len from the stack frame since SI and CX were consumed during
// phase 1.
//
// Expects the broadcast reciprocal norm in X1 and the function
// arguments v_base+0(FP) and v_len+8(FP) on the stack frame.
// Uses SI, CX, X2 as scratch.
// Produces the normalised vector written in place at v_base.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (*amd64VectormathsOps) emitNormaliseSSEWriteBack(e *asmgen.Emitter) {
	inst(e, mnemonicMOVQ, operandSliceVBaseSI)
	inst(e, mnemonicMOVQ, operandSliceVLenCX)
	inst(e, mnemonicCMPQ, operandCompare4)
	inst(e, mnemonicJL, "normsse_write_tail")
	e.Blank()
	e.Label("normsse_write_loop4")
	inst(e, mnemonicMOVUPS, "(SI), X2")
	inst(e, mnemonicMULPS, "X1, X2")
	inst(e, mnemonicMOVUPS, "X2, (SI)")
	inst(e, mnemonicADDQ, operandAdd16SI)
	inst(e, mnemonicSUBQ, operandSub4CX)
	inst(e, mnemonicCMPQ, operandCompare4)
	inst(e, mnemonicJGE, "normsse_write_loop4")
	e.Blank()
	e.Label("normsse_write_tail")
	inst(e, mnemonicCMPQ, operandCompare0)
	inst(e, mnemonicJE, "normsse_done")
	e.Blank()
	e.Label("normsse_write_tail_loop")
	inst(e, mnemonicMOVSS, "(SI), X2")
	inst(e, mnemonicMULSS, "X1, X2")
	inst(e, mnemonicMOVSS, "X2, (SI)")
	inst(e, mnemonicADDQ, operandAdd4SI)
	inst(e, mnemonicDECQ, operandDecCX)
	inst(e, mnemonicJNZ, "normsse_write_tail_loop")
	e.Blank()
	e.Label("normsse_done")
	e.Instruction(mnemonicRET)
}

// emitNormaliseAVX2 emits the complete AVX2 in-place normalisation function body.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (v *amd64VectormathsOps) emitNormaliseAVX2(e *asmgen.Emitter) {
	v.emitNormaliseAVX2SumOfSquares(e)
	v.emitNormaliseAVX2ZeroCheckAndReciprocalSqrt(e)
	v.emitNormaliseAVX2WriteBack(e)
}

// emitNormaliseAVX2SumOfSquares emits phase 1 of AVX2 normalisation:
// computing the sum of squares of every element.
//
// The vector loop loads eight float32 values from (SI), squares them
// (VMULPS self), and accumulates into Y0. The scalar tail loop
// handles leftover elements, accumulating into X3. After both loops,
// the packed accumulator Y0 and scalar tail X3 are horizontally
// reduced into a single scalar in X0.
//
// Expects the function arguments v_base+0(FP) and v_len+8(FP) on
// the stack frame. Uses SI, CX, Y0, Y1, X3.
// Produces the total sum of squares as a scalar in X0.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (v *amd64VectormathsOps) emitNormaliseAVX2SumOfSquares(e *asmgen.Emitter) {
	inst(e, mnemonicMOVQ, operandSliceVBaseSI)
	inst(e, mnemonicMOVQ, operandSliceVLenCX)
	inst(e, mnemonicVXORPS, "Y0, Y0, Y0")
	inst(e, mnemonicVXORPS, "X3, X3, X3")
	e.Blank()
	inst(e, mnemonicCMPQ, operandCompare8)
	inst(e, mnemonicJL, "normavx_tail")
	e.Blank()
	e.Label("normavx_loop8")
	inst(e, mnemonicVMOVUPS, "(SI), Y1")
	inst(e, mnemonicVMULPS, "Y1, Y1, Y1")
	inst(e, mnemonicVADDPS, "Y1, Y0, Y0")
	inst(e, mnemonicADDQ, operandAdd32SI)
	inst(e, mnemonicSUBQ, operandSub8CX)
	inst(e, mnemonicCMPQ, operandCompare8)
	inst(e, mnemonicJGE, "normavx_loop8")
	e.Blank()
	e.Label("normavx_tail")
	inst(e, mnemonicCMPQ, operandCompare0)
	inst(e, mnemonicJE, "normavx_reduce")
	e.Blank()
	e.Label("normavx_tail_loop")
	inst(e, mnemonicVMOVSS, operandLoadX1FromSI)
	inst(e, mnemonicVMULSS, "X1, X1, X1")
	inst(e, mnemonicVADDSS, "X1, X3, X3")
	inst(e, mnemonicADDQ, operandAdd4SI)
	inst(e, mnemonicDECQ, operandDecCX)
	inst(e, mnemonicJNZ, "normavx_tail_loop")
	e.Blank()

	e.Label("normavx_reduce")
	v.emitAVX2HorizontalReduce(e)
	inst(e, mnemonicVADDSS, "X3, X0, X0")
	e.Blank()
}

// emitNormaliseAVX2ZeroCheckAndReciprocalSqrt emits phase 2 of AVX2
// normalisation: the zero-vector check and reciprocal square root
// computation.
//
// If the sum of squares in X0 is exactly zero (checked via VUCOMISS),
// the function jumps to normavx_done_zero, issuing VZEROUPPER before
// returning to avoid SSE/AVX transition penalties. Otherwise, VSQRTSS
// computes the square root, and the reciprocal (1.0/sqrt) is computed
// via VDIVSS. The reciprocal is then broadcast to all eight lanes of
// Y1 via VBROADCASTSS for use by the write-back phase.
//
// Expects the sum of squares as a scalar in X0.
// Produces the reciprocal norm broadcast in all lanes of Y1, or
// jumps to normavx_done_zero if the vector is zero.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (*amd64VectormathsOps) emitNormaliseAVX2ZeroCheckAndReciprocalSqrt(e *asmgen.Emitter) {
	inst(e, mnemonicVXORPS, "X4, X4, X4")
	inst(e, "VUCOMISS", "X4, X0")
	inst(e, mnemonicJE, "normavx_done_zero")
	inst(e, "VSQRTSS", "X0, X0, X0")
	inst(e, mnemonicMOVL, operandIEEE754One)
	inst(e, mnemonicMOVQ, "AX, X1")
	inst(e, "VDIVSS", "X0, X1, X1")
	inst(e, "VBROADCASTSS", "X1, Y1")
	e.Blank()
}

// emitNormaliseAVX2WriteBack emits phase 3 of AVX2 normalisation:
// multiplying every element by the reciprocal norm and writing the
// results back in place.
//
// The vector loop loads eight float32 values from (SI), multiplies by
// the broadcast reciprocal in Y1, and stores back. The scalar tail
// loop handles leftover elements using X1 (the low lane of Y1 still
// holds the scalar reciprocal). Both loops re-read v_base and v_len
// from the stack frame since SI and CX were consumed during phase 1.
//
// After the write-back is complete (or if no elements remain),
// VZEROUPPER is issued before RET. A separate normavx_done_zero
// label handles the zero-vector early exit with its own VZEROUPPER.
//
// Expects the broadcast reciprocal norm in Y1 (and X1) and the
// function arguments v_base+0(FP) and v_len+8(FP) on the stack
// frame. Uses SI, CX, Y2/X2 as scratch.
// Produces the normalised vector written in place at v_base.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (*amd64VectormathsOps) emitNormaliseAVX2WriteBack(e *asmgen.Emitter) {
	inst(e, mnemonicMOVQ, operandSliceVBaseSI)
	inst(e, mnemonicMOVQ, operandSliceVLenCX)
	inst(e, mnemonicCMPQ, operandCompare8)
	inst(e, mnemonicJL, "normavx_write_tail")
	e.Blank()
	e.Label("normavx_write_loop8")
	inst(e, mnemonicVMOVUPS, "(SI), Y2")
	inst(e, mnemonicVMULPS, "Y1, Y2, Y2")
	inst(e, mnemonicVMOVUPS, "Y2, (SI)")
	inst(e, mnemonicADDQ, operandAdd32SI)
	inst(e, mnemonicSUBQ, operandSub8CX)
	inst(e, mnemonicCMPQ, operandCompare8)
	inst(e, mnemonicJGE, "normavx_write_loop8")
	e.Blank()
	e.Label("normavx_write_tail")
	inst(e, mnemonicCMPQ, operandCompare0)
	inst(e, mnemonicJE, "normavx_done")
	e.Blank()
	e.Label("normavx_write_tail_loop")
	inst(e, mnemonicVMOVSS, "(SI), X2")
	inst(e, mnemonicVMULSS, "X1, X2, X2")
	inst(e, mnemonicVMOVSS, "X2, (SI)")
	inst(e, mnemonicADDQ, operandAdd4SI)
	inst(e, mnemonicDECQ, operandDecCX)
	inst(e, mnemonicJNZ, "normavx_write_tail_loop")
	e.Blank()
	e.Label("normavx_done")
	e.Instruction(mnemonicVZEROUPPER)
	e.Instruction(mnemonicRET)
	e.Blank()
	e.Label("normavx_done_zero")
	e.Instruction(mnemonicVZEROUPPER)
	e.Instruction(mnemonicRET)
}
