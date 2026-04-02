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

package asmgen_arch_arm64

import (
	"strings"

	"piko.sh/piko/wdk/asmgen"
	core "piko.sh/piko/wdk/asmgen/asmgen_arch_arm64"
)

const (
	// mnemonicMOVD represents the MOVD ARM64 assembly mnemonic.
	mnemonicMOVD = "MOVD"

	// mnemonicFMOVS represents the FMOVS ARM64 assembly mnemonic.
	mnemonicFMOVS = "FMOVS"

	// mnemonicVEOR represents the VEOR ARM64 NEON assembly mnemonic.
	mnemonicVEOR = "VEOR"

	// mnemonicADD represents the ADD ARM64 assembly mnemonic.
	mnemonicADD = "ADD"

	// mnemonicSUB represents the SUB ARM64 assembly mnemonic.
	mnemonicSUB = "SUB"

	// mnemonicCMP represents the CMP ARM64 assembly mnemonic.
	mnemonicCMP = "CMP"

	// mnemonicVLD1 represents the VLD1 ARM64 NEON assembly mnemonic.
	mnemonicVLD1 = "VLD1"

	// mnemonicCBZ represents the CBZ ARM64 assembly mnemonic.
	mnemonicCBZ = "CBZ"

	// mnemonicBLT represents the BLT ARM64 assembly mnemonic.
	mnemonicBLT = "BLT"

	// mnemonicBGE represents the BGE ARM64 assembly mnemonic.
	mnemonicBGE = "BGE"

	// operandImm4R2 is the operand string for subtracting 4 from R2.
	operandImm4R2 = "$4, R2"

	// operandImm16R0 is the operand string for advancing R0 by 16 bytes.
	operandImm16R0 = "$16, R0"

	// defaultColumnWidth is the standard mnemonic column width for ARM64 Plan 9 assembly.
	defaultColumnWidth = 5
)

// VectormathsARM64Arch extends the core ARM64Arch with SIMD
// vectormaths operations for dot product and euclidean distance.
type VectormathsARM64Arch struct {
	core.ARM64Arch
}

// New creates a new vectormaths-specific ARM64 architecture adapter.
//
// Returns *VectormathsARM64Arch ready for use.
func New() *VectormathsARM64Arch {
	return &VectormathsARM64Arch{}
}

// inst emits a tab-indented instruction with mnemonic padded to the
// given column width.
//
// Takes e (*asmgen.Emitter) which receives the formatted instruction line.
// Takes mnemonic (string) which is the assembly mnemonic to emit.
// Takes operands (string) which is the operand string to append after padding.
// Takes pad (int) which is the target column width for mnemonic alignment.
func inst(e *asmgen.Emitter, mnemonic, operands string, pad int) {
	padding := max(pad-len(mnemonic), 1)
	e.Instruction(mnemonic + strings.Repeat(" ", padding) + operands)
}

// inst5 emits with 5-column padding (the default for arm64).
//
// Takes e (*asmgen.Emitter) which receives the formatted instruction line.
// Takes mnemonic (string) which is the assembly mnemonic to emit.
// Takes operands (string) which is the operand string to append after padding.
func inst5(e *asmgen.Emitter, mnemonic, operands string) {
	inst(e, mnemonic, operands, defaultColumnWidth)
}

// EmitDotProduct emits the NEON dot product function body.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes variant (string) which must be "NEON"; other variants emit nothing.
func (v *VectormathsARM64Arch) EmitDotProduct(e *asmgen.Emitter, variant string) {
	if variant != "NEON" {
		return
	}

	inst5(e, mnemonicMOVD, "a_base+0(FP), R0")
	inst5(e, mnemonicMOVD, "a_len+8(FP), R2")
	inst5(e, mnemonicMOVD, "b_base+24(FP), R1")
	e.Blank()
	inst5(e, mnemonicVEOR, "V0.B16, V0.B16, V0.B16")
	inst5(e, mnemonicVEOR, "V3.B16, V3.B16, V3.B16")
	e.Blank()
	inst5(e, mnemonicCMP, operandImm4R2)
	inst5(e, mnemonicBLT, "dot_tail")
	e.Blank()
	v.emitDotProductVectorLoop(e)
	v.emitDotProductScalarTail(e)
	v.emitDotProductReduceAndReturn(e)
}

// EmitNormalise emits the NEON in-place L2 normalisation function body.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes variant (string) which must be "NEON"; other variants emit nothing.
func (v *VectormathsARM64Arch) EmitNormalise(e *asmgen.Emitter, variant string) {
	if variant != "NEON" {
		return
	}

	v.emitNormaliseSumOfSquares(e)
	v.emitNormaliseZeroCheckAndReciprocalSqrt(e)
	v.emitNormaliseWriteBack(e)
}

// EmitEuclideanDistanceSquared emits the NEON squared Euclidean distance function body.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes variant (string) which must be "NEON"; other variants emit nothing.
func (v *VectormathsARM64Arch) EmitEuclideanDistanceSquared(e *asmgen.Emitter, variant string) {
	if variant != "NEON" {
		return
	}

	inst5(e, mnemonicMOVD, "a_base+0(FP), R0")
	inst5(e, mnemonicMOVD, "a_len+8(FP), R2")
	inst5(e, mnemonicMOVD, "b_base+24(FP), R1")
	e.Blank()
	inst5(e, mnemonicVEOR, "V0.B16, V0.B16, V0.B16")
	inst5(e, mnemonicVEOR, "V3.B16, V3.B16, V3.B16")
	e.Blank()
	inst5(e, mnemonicCMP, operandImm4R2)
	inst5(e, mnemonicBLT, "euclid_tail")
	e.Blank()
	v.emitEuclideanDistanceSquaredVectorLoop(e)
	v.emitEuclideanDistanceSquaredScalarTail(e)
	v.emitEuclideanDistanceSquaredReduceAndReturn(e)
}

// emitNEONHorizontalReduce emits the NEON horizontal reduction
// sequence that collapses the four float32 lanes of V0.S4 into a
// single scalar in F0.
//
// The algorithm performs two pairwise float additions (FADDP)
// encoded as raw WORD instructions ($0x6E20D400), since the Go
// assembler does not support the FADDP mnemonic for 4S vectors.
// The first FADDP reduces four lanes to two, and the second reduces
// two lanes to one scalar in F0.
//
// Expects the packed accumulator in V0.S4.
// Produces a scalar sum in F0 (the low lane of V0).
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (*VectormathsARM64Arch) emitNEONHorizontalReduce(e *asmgen.Emitter) {
	e.Instruction("WORD $0x6E20D400")
	e.Instruction("WORD $0x6E20D400")
}

// emitNEONScalarTail emits a scalar tail loop that handles remaining
// float32 elements (fewer than 4) after the main NEON vector loop for
// two-operand kernels (dot product, Euclidean distance).
//
// Each remaining pair of float32 values is loaded from (R0) and (R1).
// The caller-provided bodyFn is invoked to emit the per-element
// arithmetic (e.g. multiply-accumulate for dot product, or
// subtract-then-square-accumulate for Euclidean distance). Both
// pointers advance by 4 bytes and R2 is decremented by 1 on each
// iteration.
//
// Expects R0 and R1 at the first unprocessed element, R2 holding the
// remaining count, and F3 initialised for accumulation. The bodyFn
// receives the emitter and should use F1, F2, F3.
// Produces the scalar tail sum in F3.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes tailLabel (string) which is the label for the tail entry point.
// Takes reduceLabel (string) which is the label to jump to when count is zero.
// Takes loopLabel (string) which is the label for the scalar loop body.
// Takes bodyFn (func(e *asmgen.Emitter)) which emits per-element arithmetic.
func (*VectormathsARM64Arch) emitNEONScalarTail(
	e *asmgen.Emitter,
	tailLabel string,
	reduceLabel string,
	loopLabel string,
	bodyFn func(e *asmgen.Emitter),
) {
	e.Label(tailLabel)
	inst5(e, mnemonicCBZ, "R2, "+reduceLabel)
	e.Blank()
	e.Label(loopLabel)
	inst5(e, mnemonicFMOVS, "(R0), F1")
	inst5(e, mnemonicFMOVS, "(R1), F2")
	bodyFn(e)
	inst5(e, mnemonicADD, "$4, R0")
	inst5(e, mnemonicADD, "$4, R1")
	inst5(e, mnemonicSUB, "$1, R2")
	inst5(e, "CBNZ", "R2, "+loopLabel)
	e.Blank()
}

// emitDotProductVectorLoop emits the main four-wide NEON loop body
// for the dot product.
//
// On each iteration, four float32 pairs are loaded from (R0) and (R1)
// via VLD1, then fused-multiply-accumulated into V0.S4 via VFMLA.
// The pointers advance by 16 bytes and the remaining count in R2 is
// decremented by 4. The loop continues while R2 >= 4.
//
// Expects R0 and R1 at the current vector positions, R2 holding the
// remaining element count, and V0 initialised to zero. Uses V1 and
// V2 as scratch.
// Produces the partial packed sum in V0.S4, with R0/R1/R2 advanced
// past all complete 4-element groups.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (*VectormathsARM64Arch) emitDotProductVectorLoop(e *asmgen.Emitter) {
	e.Label("dot_loop4")
	inst5(e, mnemonicVLD1, "(R0), [V1.S4]")
	inst5(e, mnemonicVLD1, "(R1), [V2.S4]")
	inst5(e, "VFMLA", "V1.S4, V2.S4, V0.S4")
	inst5(e, mnemonicADD, operandImm16R0)
	inst5(e, mnemonicADD, "$16, R1")
	inst5(e, mnemonicSUB, operandImm4R2)
	inst5(e, mnemonicCMP, operandImm4R2)
	inst5(e, mnemonicBGE, "dot_loop4")
	e.Blank()
}

// emitDotProductScalarTail emits the scalar tail loop that handles
// remaining elements (fewer than 4) after the main NEON vector loop.
//
// Each remaining pair of float32 values is loaded from (R0) and (R1),
// then fused-multiply-accumulated into F3 via FMADDS. The pointers
// advance by 4 bytes and R2 is decremented by 1 on each iteration.
//
// Expects R0 and R1 at the first unprocessed element, R2 holding the
// remaining count (0..3), and F3 initialised to zero.
// Produces the scalar tail sum in F3.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (v *VectormathsARM64Arch) emitDotProductScalarTail(e *asmgen.Emitter) {
	v.emitNEONScalarTail(e, "dot_tail", "dot_reduce", "dot_tail_loop", func(e *asmgen.Emitter) {
		inst5(e, "FMADDS", "F1, F3, F2, F3")
	})
}

// emitDotProductReduceAndReturn emits the horizontal reduction of the
// packed accumulator V0.S4, merges the scalar tail sum from F3,
// stores the final result to the return slot, and returns.
//
// The two FADDP WORD instructions reduce the four lanes of V0 down to
// a scalar in F0. The tail sum in F3 is then added via FADDS, and the
// result is written to ret+48(FP).
//
// Expects the packed sum in V0.S4 and the scalar tail sum in F3.
// Produces the final dot product scalar in ret+48(FP).
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (v *VectormathsARM64Arch) emitDotProductReduceAndReturn(e *asmgen.Emitter) {
	e.Label("dot_reduce")
	v.emitNEONHorizontalReduce(e)
	e.Blank()
	inst5(e, "FADDS", "F3, F0, F0")
	e.Blank()
	inst5(e, mnemonicFMOVS, "F0, ret+48(FP)")
	e.Instruction("RET")
}

// emitNormaliseSumOfSquares emits phase 1 of NEON normalisation:
// computing the sum of squares of every element.
//
// The vector loop loads four float32 values from (R0), squares and
// accumulates them into V0.S4 via VFMLA (self x self). The scalar
// tail loop handles leftover elements, accumulating into F3 via
// FMADDS. After both loops, the packed accumulator V0 is
// horizontally reduced via two FADDP WORD instructions, and the
// scalar tail F3 is added to produce a single scalar sum in F0.
//
// Expects the function arguments v_base+0(FP) and v_len+8(FP) on
// the stack frame. Uses R0, R2, V0, V1, V3/F3.
// Produces the total sum of squares as a scalar in F0.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (v *VectormathsARM64Arch) emitNormaliseSumOfSquares(e *asmgen.Emitter) {
	inst5(e, mnemonicMOVD, "v_base+0(FP), R0")
	inst5(e, mnemonicMOVD, "v_len+8(FP), R2")
	e.Blank()
	inst5(e, mnemonicVEOR, "V0.B16, V0.B16, V0.B16")
	inst5(e, mnemonicVEOR, "V3.B16, V3.B16, V3.B16")
	e.Blank()
	inst5(e, mnemonicCMP, operandImm4R2)
	inst5(e, mnemonicBLT, "norm_tail")
	e.Blank()
	e.Label("norm_loop4")
	inst5(e, mnemonicVLD1, "(R0), [V1.S4]")
	inst5(e, "VFMLA", "V1.S4, V1.S4, V0.S4")
	inst5(e, mnemonicADD, operandImm16R0)
	inst5(e, mnemonicSUB, operandImm4R2)
	inst5(e, mnemonicCMP, operandImm4R2)
	inst5(e, mnemonicBGE, "norm_loop4")
	e.Blank()
	e.Label("norm_tail")
	inst5(e, mnemonicCBZ, "R2, norm_reduce")
	e.Blank()
	e.Label("norm_tail_loop")
	inst5(e, mnemonicFMOVS, "(R0), F1")
	inst5(e, "FMADDS", "F1, F3, F1, F3")
	inst5(e, mnemonicADD, "$4, R0")
	inst5(e, mnemonicSUB, "$1, R2")
	inst5(e, "CBNZ", "R2, norm_tail_loop")
	e.Blank()

	e.Label("norm_reduce")
	v.emitNEONHorizontalReduce(e)
	e.Blank()
	inst5(e, "FADDS", "F3, F0, F0")
	e.Blank()
}

// emitNormaliseZeroCheckAndReciprocalSqrt emits phase 2 of NEON
// normalisation: the zero-vector check and reciprocal square root
// computation.
//
// If the sum of squares in F0 is zero (checked by moving to R3 and
// testing with CBZ), the function jumps to norm_done, leaving the
// zero vector unchanged. Otherwise, FSQRTS computes the square root,
// and the reciprocal (1.0/sqrt) is computed via FDIVS. The reciprocal
// is then broadcast to all four lanes of V1.S4 via a DUP instruction
// encoded as WORD $0x4E040421.
//
// Expects the sum of squares as a scalar in F0.
// Produces the reciprocal norm broadcast in V1.S4 (and F1), or jumps
// to norm_done if the vector is zero.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (*VectormathsARM64Arch) emitNormaliseZeroCheckAndReciprocalSqrt(e *asmgen.Emitter) {
	inst5(e, mnemonicFMOVS, "F0, R3")
	inst5(e, mnemonicCBZ, "R3, norm_done")
	inst5(e, "FSQRTS", "F0, F0")
	inst5(e, mnemonicFMOVS, "$1.0, F1")
	inst5(e, "FDIVS", "F0, F1, F1")
	e.Instruction("WORD $0x4E040421")
	e.Blank()
}

// emitNormaliseWriteBack emits phase 3 of NEON normalisation:
// multiplying every element by the reciprocal norm and writing the
// results back in place.
//
// The vector loop loads four float32 values from (R0) via VLD1,
// multiplies by the broadcast reciprocal in V1.S4 via a FMUL
// instruction encoded as WORD $0x6E21DC42, and stores back via VST1.
// The scalar tail loop handles leftover elements using FMULS with F1.
// Both loops re-read v_base and v_len from the stack frame since R0
// and R2 were consumed during phase 1.
//
// Expects the broadcast reciprocal norm in V1.S4 (and F1) and the
// function arguments v_base+0(FP) and v_len+8(FP) on the stack
// frame. Uses R0, R2, V2/F2 as scratch.
// Produces the normalised vector written in place at v_base.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (*VectormathsARM64Arch) emitNormaliseWriteBack(e *asmgen.Emitter) {
	inst5(e, mnemonicMOVD, "v_base+0(FP), R0")
	inst5(e, mnemonicMOVD, "v_len+8(FP), R2")
	inst5(e, mnemonicCMP, operandImm4R2)
	inst5(e, mnemonicBLT, "norm_write_tail")
	e.Blank()
	e.Label("norm_write_loop4")
	inst5(e, mnemonicVLD1, "(R0), [V2.S4]")
	e.Instruction("WORD $0x6E21DC42")
	inst5(e, "VST1", "[V2.S4], (R0)")
	inst5(e, mnemonicADD, operandImm16R0)
	inst5(e, mnemonicSUB, operandImm4R2)
	inst5(e, mnemonicCMP, operandImm4R2)
	inst5(e, mnemonicBGE, "norm_write_loop4")
	e.Blank()
	e.Label("norm_write_tail")
	inst5(e, mnemonicCBZ, "R2, norm_done")
	e.Blank()
	e.Label("norm_write_tail_loop")
	inst5(e, mnemonicFMOVS, "(R0), F2")
	inst5(e, "FMULS", "F1, F2, F2")
	inst5(e, mnemonicFMOVS, "F2, (R0)")
	inst5(e, mnemonicADD, "$4, R0")
	inst5(e, mnemonicSUB, "$1, R2")
	inst5(e, "CBNZ", "R2, norm_write_tail_loop")
	e.Blank()
	e.Label("norm_done")
	e.Instruction("RET")
}

// emitEuclideanDistanceSquaredVectorLoop emits the main four-wide
// NEON loop body for squared Euclidean distance.
//
// On each iteration, four float32 pairs are loaded from (R0) and (R1)
// via VLD1. The difference is computed by an FSUB instruction encoded
// as WORD $0x4EA2D421 (since the Go assembler lacks FSUB for 4S
// vectors). The squared differences are then accumulated into V0.S4
// via VFMLA (V1 x V1). The pointers advance by 16 bytes and R2 is
// decremented by 4. The loop continues while R2 >= 4.
//
// Expects R0 and R1 at the current vector positions, R2 holding the
// remaining count, and V0 initialised to zero. Uses V1 and V2 as
// scratch.
// Produces the partial packed sum-of-squared-differences in V0.S4.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (*VectormathsARM64Arch) emitEuclideanDistanceSquaredVectorLoop(e *asmgen.Emitter) {
	e.Label("euclid_loop4")
	inst5(e, mnemonicVLD1, "(R0), [V1.S4]")
	inst5(e, mnemonicVLD1, "(R1), [V2.S4]")
	e.Instruction("WORD $0x4EA2D421")
	inst5(e, "VFMLA", "V1.S4, V1.S4, V0.S4")
	inst5(e, mnemonicADD, operandImm16R0)
	inst5(e, mnemonicADD, "$16, R1")
	inst5(e, mnemonicSUB, operandImm4R2)
	inst5(e, mnemonicCMP, operandImm4R2)
	inst5(e, mnemonicBGE, "euclid_loop4")
	e.Blank()
}

// emitEuclideanDistanceSquaredScalarTail emits the scalar tail loop
// for squared Euclidean distance, handling remaining elements (fewer
// than 4) after the main NEON vector loop.
//
// Each remaining pair is loaded from (R0) and (R1), subtracted
// (FSUBS), then squared and accumulated into F3 (FMADDS). The
// pointers advance by 4 bytes and R2 is decremented by 1.
//
// Expects R0 and R1 at the first unprocessed element, R2 holding the
// remaining count (0..3), and F3 initialised to zero.
// Produces the scalar tail sum in F3.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (v *VectormathsARM64Arch) emitEuclideanDistanceSquaredScalarTail(e *asmgen.Emitter) {
	v.emitNEONScalarTail(e, "euclid_tail", "euclid_reduce", "euclid_tail_loop", func(e *asmgen.Emitter) {
		inst5(e, "FSUBS", "F2, F1, F1")
		inst5(e, "FMADDS", "F1, F3, F1, F3")
	})
}

// emitEuclideanDistanceSquaredReduceAndReturn emits the horizontal
// reduction of the packed accumulator V0.S4, merges the scalar tail
// sum from F3, stores the final result to the return slot, and
// returns.
//
// The two FADDP WORD instructions reduce the four lanes of V0 down to
// a scalar in F0. The tail sum in F3 is then added via FADDS, and the
// result is written to ret+48(FP).
//
// Expects the packed sum in V0.S4 and the scalar tail sum in F3.
// Produces the final squared Euclidean distance scalar in ret+48(FP).
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (v *VectormathsARM64Arch) emitEuclideanDistanceSquaredReduceAndReturn(e *asmgen.Emitter) {
	e.Label("euclid_reduce")
	v.emitNEONHorizontalReduce(e)
	e.Blank()
	inst5(e, "FADDS", "F3, F0, F0")
	e.Blank()
	inst5(e, mnemonicFMOVS, "F0, ret+48(FP)")
	e.Instruction("RET")
}
