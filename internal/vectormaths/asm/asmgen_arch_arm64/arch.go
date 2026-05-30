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
	"fmt"
	"strings"

	"piko.sh/piko/wdk/asmgen"
	"piko.sh/piko/wdk/asmgen/asmarm64"
	core "piko.sh/piko/wdk/asmgen/asmgen_arch_arm64"

	"piko.sh/piko/internal/vectormaths/asm/unrolled"
)

const (
	// defaultColumnWidth is the standard mnemonic column width for ARM64 Plan 9 assembly.
	defaultColumnWidth = 5
)

// VectormathsARM64Arch extends the core ARM64Arch with SIMD vectormaths operations for
// dot product and euclidean distance.
type VectormathsARM64Arch struct {
	core.ARM64Arch
}

// New creates a new vectormaths-specific ARM64 architecture adapter.
//
// Returns *VectormathsARM64Arch ready for use.
func New() *VectormathsARM64Arch {
	return &VectormathsARM64Arch{}
}

// inst emits a tab-indented instruction with mnemonic padded to the given column width.
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

// EmitSum emits the float64 slice sum function body for the given variant.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes variant (string) which must be "NEON"; other variants emit nothing.
func (v *VectormathsARM64Arch) EmitSum(e *asmgen.Emitter, variant string) {
	if variant != "NEON" {
		return
	}
	v.emitSumF64NEON(e)
}

// EmitAdd emits the float64 pointwise add function body for the given variant.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes variant (string) which must be "NEON"; other variants emit nothing.
func (v *VectormathsARM64Arch) EmitAdd(e *asmgen.Emitter, variant string) {
	if variant != "NEON" {
		return
	}
	v.emitAddF64NEON(e)
}

// EmitDotF64 emits the NEON float64 dot product function body. 8-way unrolled across
// V0/V4-V10 (2 doubles each) using WORD-encoded FMLA.2D to break the dependency chain.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes variant (string) which must be "NEON"; other variants emit nothing.
func (*VectormathsARM64Arch) EmitDotF64(e *asmgen.Emitter, variant string) {
	if variant != "NEON" {
		return
	}
	accumulators := []string{"V0", "V4", "V5", "V6", "V7", "V8", "V9", "V10"}
	unrolled.EmitUnrolledReduction(e, unrolled.UnrolledReductionSpec{
		LabelPrefix:        "dotf64",
		NarrowerTailLabel:  "_tail2",
		Accumulators:       accumulators,
		LanesPerAcc:        2,
		BytesPerElement:    8,
		BlanksBetweenSlots: true,
		EmitPrologue: func(e *asmgen.Emitter, unrollWidth int, narrowerTailFullLabel string) {
			dotF64NEONPrologue(e, accumulators, unrollWidth, narrowerTailFullLabel)
		},
		EmitOneAccumBody: dotF64NEONAccumBody,
		EmitLoopFooter:   dotF64NEONLoopFooter,
		EmitFold:         dotF64NEONFold,
		EmitNarrowerLoop: dotF64NEONNarrowerLoop,
		EmitScalarTail:   dotF64NEONScalarTail,
		EmitReduceAndReturn: func(e *asmgen.Emitter) {
			dotF64NEONReduceAndReturn(e)
		},
	})
}

// EmitScaleF64 emits the NEON in-place float64 vector-by-scalar multiply function body.
// 4-way unrolled (8 doubles per iteration) using WORD-encoded FMUL.2D against a broadcast
// copy of k in V7.2D.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes variant (string) which must be "NEON"; other variants emit nothing.
func (*VectormathsARM64Arch) EmitScaleF64(e *asmgen.Emitter, variant string) {
	if variant != "NEON" {
		return
	}
	unrolled.EmitUnrolledPointwise(e, unrolled.UnrolledPointwiseSpec{
		LabelPrefix:            "scalef64neon",
		FirstNarrowerTailLabel: "_tail2",
		UnrollFactor:           4,
		LanesPerSlot:           2,
		BytesPerElement:        8,
		BlanksBetweenSlots:     true,
		EmitPrologue:           scaleF64NEONPrologue,
		EmitOneSlotBody:        scaleF64NEONSlotBody,
		EmitLoopFooter:         scaleF64NEONLoopFooter,
		NarrowerTails:          []func(e *asmgen.Emitter){scaleF64NEONNarrowerTail2},
		EmitScalarTail:         scaleF64NEONScalarTail,
		EmitReturn:             scaleF64NEONReturn,
	})
}

// EmitDotProduct emits the NEON dot product function body. 8-way unrolled across
// V0/V4-V10 to break the VFMLA dependency chain.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes variant (string) which must be "NEON"; other variants emit nothing.
func (v *VectormathsARM64Arch) EmitDotProduct(e *asmgen.Emitter, variant string) {
	if variant != "NEON" {
		return
	}
	accumulators := []string{"V0", "V4", "V5", "V6", "V7", "V8", "V9", "V10"}
	unrolled.EmitUnrolledReduction(e, unrolled.UnrolledReductionSpec{
		LabelPrefix:        "dot",
		NarrowerTailLabel:  "_tail4",
		Accumulators:       accumulators,
		LanesPerAcc:        4,
		BytesPerElement:    4,
		BlanksBetweenSlots: true,
		EmitPrologue: func(e *asmgen.Emitter, unrollWidth int, narrowerTailFullLabel string) {
			dotF32NEONPrologue(e, accumulators, unrollWidth, narrowerTailFullLabel)
		},
		EmitOneAccumBody: dotF32NEONAccumBody,
		EmitLoopFooter: func(e *asmgen.Emitter, byteStride int) {
			dotF32NEONLoopFooter(e, byteStride, "dot")
		},
		EmitFold:         foldF32NEON,
		EmitNarrowerLoop: dotF32NEONNarrowerLoop,
		EmitScalarTail:   dotF32NEONScalarTail,
		EmitReduceAndReturn: func(e *asmgen.Emitter) {
			v.dotF32NEONReduceAndReturn(e)
		},
	})
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
	accumulators := []string{"V0", "V4", "V5", "V6", "V7", "V8", "V9", "V10"}
	fsubWord := neonFSUB4S(1, 1, 2)
	unrolled.EmitUnrolledReduction(e, unrolled.UnrolledReductionSpec{
		LabelPrefix:        "euclid",
		NarrowerTailLabel:  "_tail4",
		Accumulators:       accumulators,
		LanesPerAcc:        4,
		BytesPerElement:    4,
		BlanksBetweenSlots: true,
		EmitPrologue: func(e *asmgen.Emitter, unrollWidth int, narrowerTailFullLabel string) {
			dotF32NEONPrologue(e, accumulators, unrollWidth, narrowerTailFullLabel)
		},
		EmitOneAccumBody: func(e *asmgen.Emitter, acc string, _ int, slotIndex int) {
			euclidF32NEONAccumBody(e, acc, slotIndex, fsubWord)
		},
		EmitLoopFooter: func(e *asmgen.Emitter, byteStride int) {
			dotF32NEONLoopFooter(e, byteStride, "euclid")
		},
		EmitFold: foldF32NEON,
		EmitNarrowerLoop: func(e *asmgen.Emitter) {
			euclidF32NEONNarrowerLoop(e, fsubWord)
		},
		EmitScalarTail: euclidF32NEONScalarTail,
		EmitReduceAndReturn: func(e *asmgen.Emitter) {
			v.euclidF32NEONReduceAndReturn(e)
		},
	})
}

// emitSumF64NEON emits the complete NEON float64 sum function body. 8-way unrolled across
// V0/V4-V10 (2 doubles each).
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (*VectormathsARM64Arch) emitSumF64NEON(e *asmgen.Emitter) {
	accumulators := []string{"V0", "V4", "V5", "V6", "V7", "V8", "V9", "V10"}
	unrolled.EmitUnrolledReduction(e, unrolled.UnrolledReductionSpec{
		LabelPrefix:        "sum",
		NarrowerTailLabel:  "_tail2",
		Accumulators:       accumulators,
		LanesPerAcc:        2,
		BytesPerElement:    8,
		BlanksBetweenSlots: true,
		EmitPrologue: func(e *asmgen.Emitter, unrollWidth int, narrowerTailFullLabel string) {
			sumF64NEONPrologue(e, accumulators, unrollWidth, narrowerTailFullLabel)
		},
		EmitOneAccumBody:    sumF64NEONAccumBody,
		EmitLoopFooter:      sumF64NEONLoopFooter,
		EmitFold:            sumF64NEONFold,
		EmitNarrowerLoop:    sumF64NEONNarrowerLoop,
		EmitScalarTail:      sumF64NEONScalarTail,
		EmitReduceAndReturn: sumF64NEONReduceAndReturn,
	})
}

// sumF64NEONPrologue emits the prologue for the NEON float64 sum kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes accumulators ([]string) which lists the NEON accumulator registers to zero.
// Takes unrollWidth (int) which is the per-iteration lane count guarding loop entry.
// Takes narrowerTailFullLabel (string) which is the branch target for short inputs.
func sumF64NEONPrologue(e *asmgen.Emitter, accumulators []string, unrollWidth int, narrowerTailFullLabel string) {
	inst(e, asmarm64.OperationMove64Bits, "a_base+0(FP), R0", 4)
	inst(e, asmarm64.OperationMove64Bits, "a_len+8(FP), R2", 4)
	e.Blank()
	for _, acc := range accumulators {
		inst(e, asmarm64.OperationVectorExclusiveOr, fmt.Sprintf("%s.B16, %s.B16, %s.B16", acc, acc, acc), 4)
	}
	inst(e, asmarm64.OperationVectorExclusiveOr, "V3.B16, V3.B16, V3.B16", 4)
	e.Blank()
	inst(e, asmarm64.OperationCompare, fmt.Sprintf("$%d, R2", unrollWidth), 4)
	inst(e, asmarm64.OperationBranchIfLessSigned, narrowerTailFullLabel, 4)
}

// sumF64NEONAccumBody emits one slot of the NEON float64 sum main loop body.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes acc (string) which is the NEON accumulator register receiving this slot's add.
// Takes slotIndex (int) which gates an inline pointer-advance for slots after the first.
func sumF64NEONAccumBody(e *asmgen.Emitter, acc string, _ int, slotIndex int) {
	if slotIndex > 0 {
		inst(e, asmarm64.OperationAdd, "$16, R0", 4)
	}
	inst(e, asmarm64.OperationVectorLoadSingle, "(R0), [V1.D2]", 4)
	rd := vRegIndex(acc)
	word := neonFADD2D(rd, rd, 1)
	inst(e, asmarm64.DirectiveWord, fmt.Sprintf("$0x%08X", word), 4)
}

// sumF64NEONLoopFooter emits the main-loop footer for the NEON float64 sum kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes byteStride (int) which is the per-iteration byte stride.
func sumF64NEONLoopFooter(e *asmgen.Emitter, byteStride int) {
	unrollWidth := byteStride / 8
	inst(e, asmarm64.OperationAdd, "$16, R0", 4)
	inst(e, asmarm64.OperationSubtract, fmt.Sprintf("$%d, R2", unrollWidth), 4)
	inst(e, asmarm64.OperationCompare, fmt.Sprintf("$%d, R2", unrollWidth), 4)
	inst(e, asmarm64.OperationBranchIfGreaterOrEqualSigned, fmt.Sprintf("sum_loop%d", unrollWidth), 4)
}

// sumF64NEONFold folds the NEON float64 sum accumulators into accs[0].
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes accs ([]string) which is the accumulator list; accs[1:] are folded into accs[0].
func sumF64NEONFold(e *asmgen.Emitter, accs []string) {
	for _, acc := range accs[1:] {
		word := neonFADD2D(0, 0, vRegIndex(acc))
		inst(e, asmarm64.DirectiveWord, fmt.Sprintf("$0x%08X", word), 4)
	}
}

// sumF64NEONNarrowerLoop emits the 2-wide narrower tail loop for sum.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func sumF64NEONNarrowerLoop(e *asmgen.Emitter) {
	e.Label("sum_tail2")
	inst(e, asmarm64.OperationCompare, "$2, R2", 4)
	inst(e, asmarm64.OperationBranchIfLessSigned, "sum_tail", 4)
	e.Blank()
	e.Label("sum_loop2")
	inst(e, asmarm64.OperationVectorLoadSingle, "(R0), [V1.D2]", 4)
	word := neonFADD2D(0, 0, 1)
	inst(e, asmarm64.DirectiveWord, fmt.Sprintf("$0x%08X", word), 4)
	inst(e, asmarm64.OperationAdd, "$16, R0", 4)
	inst(e, asmarm64.OperationSubtract, "$2, R2", 4)
	inst(e, asmarm64.OperationCompare, "$2, R2", 4)
	inst(e, asmarm64.OperationBranchIfGreaterOrEqualSigned, "sum_loop2", 4)
	e.Blank()
}

// sumF64NEONScalarTail emits the scalar drain loop for sum.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func sumF64NEONScalarTail(e *asmgen.Emitter) {
	e.Label("sum_tail")
	inst(e, asmarm64.OperationCompareAndBranchIfZero, "R2, sum_reduce", 4)
	e.Blank()
	e.Label("sum_tail_loop")
	inst(e, asmarm64.OperationFloatMove64Bits, "(R0), F1", 5)
	inst(e, asmarm64.OperationFloatAddScalarDouble, "F1, F3, F3", 5)
	inst(e, asmarm64.OperationAdd, "$8, R0", 4)
	inst(e, asmarm64.OperationSubtract, "$1, R2", 4)
	inst(e, asmarm64.OperationCompareAndBranchIfNotZero, "R2, sum_tail_loop", 4)
	e.Blank()
}

// sumF64NEONReduceAndReturn emits the horizontal reduce and return for sum.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func sumF64NEONReduceAndReturn(e *asmgen.Emitter) {
	e.Label("sum_reduce")
	faddp := neonFADDP2D(0, 0, 0)
	inst(e, asmarm64.DirectiveWord, fmt.Sprintf("$0x%08X", faddp), 4)
	inst(e, asmarm64.OperationFloatAddScalarDouble, "F3, F0, F0", 5)
	e.Blank()
	inst(e, asmarm64.OperationFloatMove64Bits, "F0, ret+24(FP)", 5)
	e.Instruction(asmarm64.OperationReturn)
}

// emitAddF64NEON emits the complete NEON pointwise add. The 8-doubles- per-iteration body
// uses NEON interleaved VLD1/VST1 to load and store four 2D vectors at once, with four
// WORD-encoded FADDs in between.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (*VectormathsARM64Arch) emitAddF64NEON(e *asmgen.Emitter) {
	unrolled.EmitUnrolledPointwise(e, unrolled.UnrolledPointwiseSpec{
		LabelPrefix:            "addf64neon",
		FirstNarrowerTailLabel: "_tail2",
		UnrollFactor:           1,
		LanesPerSlot:           8,
		BytesPerElement:        8,
		BlanksBetweenSlots:     false,
		EmitPrologue:           addF64NEONPrologue,
		EmitOneSlotBody:        addF64NEONSlotBody,
		EmitLoopFooter:         addF64NEONLoopFooter,
		NarrowerTails: []func(e *asmgen.Emitter){
			addF64NEONNarrowerTail2,
		},
		EmitScalarTail: addF64NEONScalarTail,
		EmitReturn:     addF64NEONReturn,
	})
}

// addF64NEONPrologue emits the prologue for the NEON float64 pointwise add kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes unrollWidth (int) which is the per-iteration lane count guarding loop entry.
// Takes firstTail (string) which is the branch target for short inputs.
func addF64NEONPrologue(e *asmgen.Emitter, unrollWidth int, firstTail string) {
	inst5(e, asmarm64.OperationMove64Bits, "dst_base+0(FP), R0")
	inst5(e, asmarm64.OperationMove64Bits, "dst_len+8(FP), R3")
	inst5(e, asmarm64.OperationMove64Bits, "a_base+24(FP), R1")
	inst5(e, asmarm64.OperationMove64Bits, "b_base+48(FP), R2")
	e.Blank()
	inst5(e, asmarm64.OperationCompare, fmt.Sprintf("$%d, R3", unrollWidth))
	inst5(e, asmarm64.OperationBranchIfLessSigned, firstTail)
}

// addF64NEONSlotBody emits one slot of the NEON float64 pointwise add body.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func addF64NEONSlotBody(e *asmgen.Emitter, _ int) {
	inst5(e, asmarm64.OperationVectorLoadSingle, "(R1), [V0.D2, V1.D2, V2.D2, V3.D2]")
	inst5(e, asmarm64.OperationVectorLoadSingle, "(R2), [V4.D2, V5.D2, V6.D2, V7.D2]")
	for i := range uint8(4) {
		word := neonFADD2D(i, i, i+4)
		inst5(e, asmarm64.DirectiveWord, fmt.Sprintf("$0x%08X", word))
	}
	inst5(e, asmarm64.OperationVectorStoreSingle, "[V0.D2, V1.D2, V2.D2, V3.D2], (R0)")
}

// addF64NEONLoopFooter emits the main-loop footer for the NEON float64 pointwise add
// kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes byteStride (int) which is the per-iteration byte stride used to advance pointers.
func addF64NEONLoopFooter(e *asmgen.Emitter, byteStride int) {
	unrollWidth := byteStride / 8
	inst5(e, asmarm64.OperationAdd, fmt.Sprintf("$%d, R1", byteStride))
	inst5(e, asmarm64.OperationAdd, fmt.Sprintf("$%d, R2", byteStride))
	inst5(e, asmarm64.OperationAdd, fmt.Sprintf("$%d, R0", byteStride))
	inst5(e, asmarm64.OperationSubtract, fmt.Sprintf("$%d, R3", unrollWidth))
	inst5(e, asmarm64.OperationCompare, fmt.Sprintf("$%d, R3", unrollWidth))
	inst5(e, asmarm64.OperationBranchIfGreaterOrEqualSigned, fmt.Sprintf("addf64neon_loop%d", unrollWidth))
}

// addF64NEONNarrowerTail2 emits the 2-wide narrower tail loop for the NEON float64
// pointwise add kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func addF64NEONNarrowerTail2(e *asmgen.Emitter) {
	e.Label("addf64neon_tail2")
	inst5(e, asmarm64.OperationCompare, "$2, R3")
	inst5(e, asmarm64.OperationBranchIfLessSigned, "addf64neon_tail")
	e.Blank()
	e.Label("addf64neon_loop2")
	inst5(e, asmarm64.OperationVectorLoadSingle, "(R1), [V0.D2]")
	inst5(e, asmarm64.OperationVectorLoadSingle, "(R2), [V1.D2]")
	word := neonFADD2D(0, 0, 1)
	inst5(e, asmarm64.DirectiveWord, fmt.Sprintf("$0x%08X", word))
	inst5(e, asmarm64.OperationVectorStoreSingle, "[V0.D2], (R0)")
	inst5(e, asmarm64.OperationAdd, "$16, R1")
	inst5(e, asmarm64.OperationAdd, "$16, R2")
	inst5(e, asmarm64.OperationAdd, "$16, R0")
	inst5(e, asmarm64.OperationSubtract, "$2, R3")
	inst5(e, asmarm64.OperationCompare, "$2, R3")
	inst5(e, asmarm64.OperationBranchIfGreaterOrEqualSigned, "addf64neon_loop2")
	e.Blank()
}

// addF64NEONScalarTail emits the scalar drain loop for the NEON float64 pointwise add
// kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func addF64NEONScalarTail(e *asmgen.Emitter) {
	e.Label("addf64neon_tail")
	inst5(e, asmarm64.OperationCompareAndBranchIfZero, "R3, addf64neon_done")
	e.Blank()
	e.Label("addf64neon_tail_loop")
	inst5(e, asmarm64.OperationFloatMove64Bits, "(R1), F0")
	inst5(e, asmarm64.OperationFloatMove64Bits, "(R2), F1")
	inst5(e, asmarm64.OperationFloatAddScalarDouble, "F1, F0, F0")
	inst5(e, asmarm64.OperationFloatMove64Bits, "F0, (R0)")
	inst5(e, asmarm64.OperationAdd, "$8, R1")
	inst5(e, asmarm64.OperationAdd, "$8, R2")
	inst5(e, asmarm64.OperationAdd, "$8, R0")
	inst5(e, asmarm64.OperationSubtract, "$1, R3")
	inst5(e, asmarm64.OperationCompareAndBranchIfNotZero, "R3, addf64neon_tail_loop")
	e.Blank()
}

// addF64NEONReturn emits the return label and RET for the NEON float64 pointwise add
// kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func addF64NEONReturn(e *asmgen.Emitter) {
	e.Label("addf64neon_done")
	e.Instruction(asmarm64.OperationReturn)
}

// dotF32NEONPrologue emits the prologue used by both the dot product and the squared
// Euclidean distance NEON kernels.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes accumulators ([]string) which lists the NEON accumulator registers to zero.
// Takes unrollWidth (int) which is the per-iteration lane count guarding loop entry.
// Takes narrowerTailFullLabel (string) which is the branch target for short inputs.
func dotF32NEONPrologue(e *asmgen.Emitter, accumulators []string, unrollWidth int, narrowerTailFullLabel string) {
	inst5(e, asmarm64.OperationMove64Bits, "a_base+0(FP), R0")
	inst5(e, asmarm64.OperationMove64Bits, "a_len+8(FP), R2")
	inst5(e, asmarm64.OperationMove64Bits, "b_base+24(FP), R1")
	e.Blank()
	for _, acc := range accumulators {
		inst5(e, asmarm64.OperationVectorExclusiveOr, fmt.Sprintf("%s.B16, %s.B16, %s.B16", acc, acc, acc))
	}
	inst5(e, asmarm64.OperationVectorExclusiveOr, "V3.B16, V3.B16, V3.B16")
	e.Blank()
	inst5(e, asmarm64.OperationCompare, fmt.Sprintf("$%d, R2", unrollWidth))
	inst5(e, asmarm64.OperationBranchIfLessSigned, narrowerTailFullLabel)
}

// dotF32NEONAccumBody emits one slot of the NEON float32 dot product body.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes acc (string) which is the NEON accumulator register receiving this slot's FMLA.
// Takes slotIndex (int) which gates an inline pointer-advance for slots after the first.
func dotF32NEONAccumBody(e *asmgen.Emitter, acc string, _ int, slotIndex int) {
	if slotIndex > 0 {
		inst5(e, asmarm64.OperationAdd, "$16, R0")
		inst5(e, asmarm64.OperationAdd, "$16, R1")
	}
	inst5(e, asmarm64.OperationVectorLoadSingle, "(R0), [V1.S4]")
	inst5(e, asmarm64.OperationVectorLoadSingle, "(R1), [V2.S4]")
	inst5(e, asmarm64.OperationVectorFusedMultiplyAdd, fmt.Sprintf("V1.S4, V2.S4, %s.S4", acc))
}

// dotF32NEONLoopFooter emits the main-loop footer shared by the dot product and squared
// Euclidean distance NEON kernels. labelPrefix selects which kernel-specific loop label
// to branch to.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes byteStride (int) which is the per-iteration byte stride.
// Takes labelPrefix (string) which selects the kernel loop label to branch back to.
func dotF32NEONLoopFooter(e *asmgen.Emitter, byteStride int, labelPrefix string) {
	unrollWidth := byteStride / 4
	inst5(e, asmarm64.OperationAdd, "$16, R0")
	inst5(e, asmarm64.OperationAdd, "$16, R1")
	inst5(e, asmarm64.OperationSubtract, fmt.Sprintf("$%d, R2", unrollWidth))
	inst5(e, asmarm64.OperationCompare, fmt.Sprintf("$%d, R2", unrollWidth))
	inst5(e, asmarm64.OperationBranchIfGreaterOrEqualSigned, fmt.Sprintf("%s_loop%d", labelPrefix, unrollWidth))
}

// foldF32NEON folds the NEON float32 accumulators into accs[0] via WORD- encoded FADD.4S
// instructions.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes accs ([]string) which is the accumulator list; accs[1:] are folded into accs[0].
func foldF32NEON(e *asmgen.Emitter, accs []string) {
	for _, acc := range accs[1:] {
		word := neonFADD4S(0, 0, vRegIndex(acc))
		inst5(e, asmarm64.DirectiveWord, fmt.Sprintf("$0x%08X", word))
	}
}

// dotF32NEONNarrowerLoop emits the 4-wide narrower tail loop for the NEON float32 dot
// product kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func dotF32NEONNarrowerLoop(e *asmgen.Emitter) {
	e.Label("dot_tail4")
	inst5(e, asmarm64.OperationCompare, "$4, R2")
	inst5(e, asmarm64.OperationBranchIfLessSigned, "dot_tail")
	e.Blank()
	e.Label("dot_loop4")
	inst5(e, asmarm64.OperationVectorLoadSingle, "(R0), [V1.S4]")
	inst5(e, asmarm64.OperationVectorLoadSingle, "(R1), [V2.S4]")
	inst5(e, asmarm64.OperationVectorFusedMultiplyAdd, "V1.S4, V2.S4, V0.S4")
	inst5(e, asmarm64.OperationAdd, "$16, R0")
	inst5(e, asmarm64.OperationAdd, "$16, R1")
	inst5(e, asmarm64.OperationSubtract, "$4, R2")
	inst5(e, asmarm64.OperationCompare, "$4, R2")
	inst5(e, asmarm64.OperationBranchIfGreaterOrEqualSigned, "dot_loop4")
	e.Blank()
}

// dotF32NEONScalarTail emits the scalar drain loop for the NEON float32 dot product
// kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func dotF32NEONScalarTail(e *asmgen.Emitter) {
	e.Label("dot_tail")
	inst5(e, asmarm64.OperationCompareAndBranchIfZero, "R2, dot_reduce")
	e.Blank()
	e.Label("dot_tail_loop")
	inst(e, asmarm64.OperationFloatMove32Bits, "(R0), F1", 6)
	inst(e, asmarm64.OperationFloatMove32Bits, "(R1), F2", 6)
	inst(e, asmarm64.OperationFloatMultiplyAddScalarSingle, "F1, F3, F2, F3", 6)
	inst(e, asmarm64.OperationAdd, "$4, R0", 6)
	inst(e, asmarm64.OperationAdd, "$4, R1", 6)
	inst(e, asmarm64.OperationSubtract, "$1, R2", 6)
	inst(e, asmarm64.OperationCompareAndBranchIfNotZero, "R2, dot_tail_loop", 6)
	e.Blank()
}

// dotF32NEONReduceAndReturn emits the horizontal reduce and return for the NEON float32
// dot product kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (v *VectormathsARM64Arch) dotF32NEONReduceAndReturn(e *asmgen.Emitter) {
	e.Label("dot_reduce")
	v.emitNEONHorizontalReduce(e)
	e.Blank()
	inst5(e, asmarm64.OperationFloatAddScalarSingle, "F3, F0, F0")
	e.Blank()
	inst5(e, asmarm64.OperationFloatMove32Bits, "F0, ret+48(FP)")
	e.Instruction(asmarm64.OperationReturn)
}

// euclidF32NEONAccumBody emits one slot of the NEON float32 squared Euclidean distance
// main loop body. fsubWord is the precomputed FSUB.4S V1,V1,V2 encoding.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes acc (string) which is the NEON accumulator register receiving this slot's FMLA.
// Takes slotIndex (int) which gates an inline pointer-advance for slots after the first.
// Takes fsubWord (uint32) which is the precomputed FSUB.4S V1,V1,V2 instruction encoding.
func euclidF32NEONAccumBody(e *asmgen.Emitter, acc string, slotIndex int, fsubWord uint32) {
	if slotIndex > 0 {
		inst5(e, asmarm64.OperationAdd, "$16, R0")
		inst5(e, asmarm64.OperationAdd, "$16, R1")
	}
	inst5(e, asmarm64.OperationVectorLoadSingle, "(R0), [V1.S4]")
	inst5(e, asmarm64.OperationVectorLoadSingle, "(R1), [V2.S4]")
	inst5(e, asmarm64.DirectiveWord, fmt.Sprintf("$0x%08X", fsubWord))
	inst5(e, asmarm64.OperationVectorFusedMultiplyAdd, fmt.Sprintf("V1.S4, V1.S4, %s.S4", acc))
}

// euclidF32NEONNarrowerLoop emits the 4-wide narrower tail loop for the NEON float32
// squared Euclidean distance kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes fsubWord (uint32) which is the precomputed FSUB.4S V1,V1,V2 instruction encoding.
func euclidF32NEONNarrowerLoop(e *asmgen.Emitter, fsubWord uint32) {
	e.Label("euclid_tail4")
	inst(e, asmarm64.OperationCompare, "$4, R2", 4)
	inst(e, asmarm64.OperationBranchIfLessSigned, "euclid_tail", 4)
	e.Blank()
	e.Label("euclid_loop4")
	inst(e, asmarm64.OperationVectorLoadSingle, "(R0), [V1.S4]", 4)
	inst(e, asmarm64.OperationVectorLoadSingle, "(R1), [V2.S4]", 4)
	inst(e, asmarm64.DirectiveWord, fmt.Sprintf("$0x%08X", fsubWord), 4)
	inst(e, asmarm64.OperationVectorFusedMultiplyAdd, "V1.S4, V1.S4, V0.S4", 4)
	inst(e, asmarm64.OperationAdd, "$16, R0", 4)
	inst(e, asmarm64.OperationAdd, "$16, R1", 4)
	inst(e, asmarm64.OperationSubtract, "$4, R2", 4)
	inst(e, asmarm64.OperationCompare, "$4, R2", 4)
	inst(e, asmarm64.OperationBranchIfGreaterOrEqualSigned, "euclid_loop4", 4)
	e.Blank()
}

// euclidF32NEONScalarTail emits the scalar drain loop for the NEON float32 squared
// Euclidean distance kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func euclidF32NEONScalarTail(e *asmgen.Emitter) {
	e.Label("euclid_tail")
	inst(e, asmarm64.OperationCompareAndBranchIfZero, "R2, euclid_reduce", 4)
	e.Blank()
	e.Label("euclid_tail_loop")
	inst(e, asmarm64.OperationFloatMove32Bits, "(R0), F1", 6)
	inst(e, asmarm64.OperationFloatMove32Bits, "(R1), F2", 6)
	inst(e, asmarm64.OperationFloatSubtractScalarSingle, "F2, F1, F1", 6)
	inst(e, asmarm64.OperationFloatMultiplyAddScalarSingle, "F1, F3, F1, F3", 6)
	inst(e, asmarm64.OperationAdd, "$4, R0", 4)
	inst(e, asmarm64.OperationAdd, "$4, R1", 4)
	inst(e, asmarm64.OperationSubtract, "$1, R2", 4)
	inst(e, asmarm64.OperationCompareAndBranchIfNotZero, "R2, euclid_tail_loop", 4)
	e.Blank()
}

// euclidF32NEONReduceAndReturn emits the horizontal reduce and return for the NEON
// float32 squared Euclidean distance kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (v *VectormathsARM64Arch) euclidF32NEONReduceAndReturn(e *asmgen.Emitter) {
	e.Label("euclid_reduce")
	v.emitNEONHorizontalReduce(e)
	e.Blank()
	inst5(e, asmarm64.OperationFloatAddScalarSingle, "F3, F0, F0")
	e.Blank()
	inst5(e, asmarm64.OperationFloatMove32Bits, "F0, ret+48(FP)")
	e.Instruction(asmarm64.OperationReturn)
}

// neonFADD4S returns the 32-bit instruction encoding for "FADD Vd.4S, Vn.4S, Vm.4S" - a
// 4-lane single-precision add. Go's arm64 assembler lacks this mnemonic so the emit-site
// calls e.Instruction("WORD $0x<hex>") with the encoded word.
//
// Takes rd, rn, rm (uint8) which are the destination, first source, and second source
// NEON register indices (V0..V31).
//
// Returns the 32-bit word that encodes the FADD instruction.
func neonFADD4S(rd, rn, rm uint8) uint32 {
	return 0x4E20D400 | (uint32(rm) << 16) | (uint32(rn) << 5) | uint32(rd)
}

// neonFSUB4S returns the 32-bit encoding for "FSUB Vd.4S, Vn.4S, Vm.4S".
//
// Takes rd, rn, rm (uint8) which are the destination, first source, and second source
// NEON register indices.
//
// Returns the 32-bit word that encodes the FSUB instruction.
func neonFSUB4S(rd, rn, rm uint8) uint32 {
	return 0x4EA0D400 | (uint32(rm) << 16) | (uint32(rn) << 5) | uint32(rd)
}

// neonFADD2D returns the 32-bit encoding for "FADD Vd.2D, Vn.2D, Vm.2D" - a 2-lane
// double-precision add.
//
// Takes rd, rn, rm (uint8) which are the destination, first source, and second source
// NEON register indices.
//
// Returns the 32-bit word that encodes the FADD instruction.
func neonFADD2D(rd, rn, rm uint8) uint32 {
	return 0x4E60D400 | (uint32(rm) << 16) | (uint32(rn) << 5) | uint32(rd)
}

// neonFADDP2D returns the 32-bit encoding for "FADDP Vd.2D, Vn.2D, Vm.2D" - pairwise
// double-precision add.
//
// Takes rd, rn, rm (uint8) which are the destination, first source, and second source
// NEON register indices.
//
// Returns the 32-bit word that encodes the FADDP instruction.
func neonFADDP2D(rd, rn, rm uint8) uint32 {
	return 0x6E60D400 | (uint32(rm) << 16) | (uint32(rn) << 5) | uint32(rd)
}

// neonFMLA2D returns the 32-bit encoding for "FMLA Vd.2D, Vn.2D, Vm.2D" - a 2-lane
// double-precision fused multiply-add: Vd.2D += Vn.2D * Vm.2D.
//
// Takes rd, rn, rm (uint8) which are the destination, first source, and second source
// NEON register indices.
//
// Returns the 32-bit word that encodes the FMLA instruction.
func neonFMLA2D(rd, rn, rm uint8) uint32 {
	return 0x4E60CC00 | (uint32(rm) << 16) | (uint32(rn) << 5) | uint32(rd)
}

// neonFMUL2D returns the 32-bit encoding for "FMUL Vd.2D, Vn.2D, Vm.2D" - a 2-lane
// double-precision multiply.
//
// Takes rd, rn, rm (uint8) which are the destination, first source, and second source
// NEON register indices.
//
// Returns the 32-bit word that encodes the FMUL instruction.
func neonFMUL2D(rd, rn, rm uint8) uint32 {
	return 0x6E60DC00 | (uint32(rm) << 16) | (uint32(rn) << 5) | uint32(rd)
}

// neonDUP2DFromDouble returns the 32-bit encoding for "DUP Vd.2D, Vn.D[0]" - broadcast
// the low double of Vn to both lanes of Vd. The imm5 field is 0b01000, so the encoding is
// fixed apart from the rn and rd register indices.
//
// Takes rd, rn (uint8) which are the destination and source NEON register indices.
//
// Returns the 32-bit word that encodes the DUP instruction.
func neonDUP2DFromDouble(rd, rn uint8) uint32 {
	return 0x4E080400 | (uint32(rn) << 5) | uint32(rd)
}

// dotF64NEONPrologue emits the prologue for the NEON float64 dot product kernel: argument
// loads, accumulator zeroing, and the unroll-width guard.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes accumulators ([]string) which lists the NEON accumulator registers to zero.
// Takes unrollWidth (int) which is the per-iteration lane count guarding loop entry.
// Takes narrowerTailFullLabel (string) which is the branch target for short inputs.
func dotF64NEONPrologue(e *asmgen.Emitter, accumulators []string, unrollWidth int, narrowerTailFullLabel string) {
	inst5(e, asmarm64.OperationMove64Bits, "a_base+0(FP), R0")
	inst5(e, asmarm64.OperationMove64Bits, "a_len+8(FP), R2")
	inst5(e, asmarm64.OperationMove64Bits, "b_base+24(FP), R1")
	e.Blank()
	for _, acc := range accumulators {
		inst5(e, asmarm64.OperationVectorExclusiveOr, fmt.Sprintf("%s.B16, %s.B16, %s.B16", acc, acc, acc))
	}
	inst5(e, asmarm64.OperationVectorExclusiveOr, "V3.B16, V3.B16, V3.B16")
	e.Blank()
	inst5(e, asmarm64.OperationCompare, fmt.Sprintf("$%d, R2", unrollWidth))
	inst5(e, asmarm64.OperationBranchIfLessSigned, narrowerTailFullLabel)
}

// dotF64NEONAccumBody emits one slot of the NEON float64 dot product body.
//
// Loads V1.2D from R0 and V2.2D from R1, then issues a WORD-encoded FMLA.2D into the
// slot's accumulator. Slots after the first advance both pointers by 16 bytes inline so
// the loop footer only handles the final stride.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes acc (string) which is the NEON accumulator register receiving this slot's FMLA.
// Takes slotIndex (int) which gates an inline pointer-advance for slots after the first.
func dotF64NEONAccumBody(e *asmgen.Emitter, acc string, _ int, slotIndex int) {
	if slotIndex > 0 {
		inst5(e, asmarm64.OperationAdd, "$16, R0")
		inst5(e, asmarm64.OperationAdd, "$16, R1")
	}
	inst5(e, asmarm64.OperationVectorLoadSingle, "(R0), [V1.D2]")
	inst5(e, asmarm64.OperationVectorLoadSingle, "(R1), [V2.D2]")
	word := neonFMLA2D(vRegIndex(acc), 1, 2)
	inst5(e, asmarm64.DirectiveWord, fmt.Sprintf("$0x%08X", word))
}

// dotF64NEONLoopFooter emits the main-loop footer for the NEON float64 dot product
// kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes byteStride (int) which is the per-iteration byte stride.
func dotF64NEONLoopFooter(e *asmgen.Emitter, byteStride int) {
	unrollWidth := byteStride / 8
	inst5(e, asmarm64.OperationAdd, "$16, R0")
	inst5(e, asmarm64.OperationAdd, "$16, R1")
	inst5(e, asmarm64.OperationSubtract, fmt.Sprintf("$%d, R2", unrollWidth))
	inst5(e, asmarm64.OperationCompare, fmt.Sprintf("$%d, R2", unrollWidth))
	inst5(e, asmarm64.OperationBranchIfGreaterOrEqualSigned, fmt.Sprintf("dotf64_loop%d", unrollWidth))
}

// dotF64NEONFold folds the NEON float64 dot product accumulators into accs[0] via
// WORD-encoded FADD.2D instructions.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes accs ([]string) which is the accumulator list; accs[1:] are folded into accs[0].
func dotF64NEONFold(e *asmgen.Emitter, accs []string) {
	for _, acc := range accs[1:] {
		word := neonFADD2D(0, 0, vRegIndex(acc))
		inst5(e, asmarm64.DirectiveWord, fmt.Sprintf("$0x%08X", word))
	}
}

// dotF64NEONNarrowerLoop emits the 2-wide narrower tail loop for the NEON float64 dot
// product kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func dotF64NEONNarrowerLoop(e *asmgen.Emitter) {
	e.Label("dotf64_tail2")
	inst5(e, asmarm64.OperationCompare, "$2, R2")
	inst5(e, asmarm64.OperationBranchIfLessSigned, "dotf64_tail")
	e.Blank()
	e.Label("dotf64_loop2")
	inst5(e, asmarm64.OperationVectorLoadSingle, "(R0), [V1.D2]")
	inst5(e, asmarm64.OperationVectorLoadSingle, "(R1), [V2.D2]")
	word := neonFMLA2D(0, 1, 2)
	inst5(e, asmarm64.DirectiveWord, fmt.Sprintf("$0x%08X", word))
	inst5(e, asmarm64.OperationAdd, "$16, R0")
	inst5(e, asmarm64.OperationAdd, "$16, R1")
	inst5(e, asmarm64.OperationSubtract, "$2, R2")
	inst5(e, asmarm64.OperationCompare, "$2, R2")
	inst5(e, asmarm64.OperationBranchIfGreaterOrEqualSigned, "dotf64_loop2")
	e.Blank()
}

// dotF64NEONScalarTail emits the scalar drain loop for the NEON float64 dot product
// kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func dotF64NEONScalarTail(e *asmgen.Emitter) {
	e.Label("dotf64_tail")
	inst5(e, asmarm64.OperationCompareAndBranchIfZero, "R2, dotf64_reduce")
	e.Blank()
	e.Label("dotf64_tail_loop")
	inst5(e, asmarm64.OperationFloatMove64Bits, "(R0), F1")
	inst5(e, asmarm64.OperationFloatMove64Bits, "(R1), F2")
	inst5(e, asmarm64.OperationFloatMultiplyScalarDouble, "F1, F2, F1")
	inst5(e, asmarm64.OperationFloatAddScalarDouble, "F1, F3, F3")
	inst5(e, asmarm64.OperationAdd, "$8, R0")
	inst5(e, asmarm64.OperationAdd, "$8, R1")
	inst5(e, asmarm64.OperationSubtract, "$1, R2")
	inst5(e, asmarm64.OperationCompareAndBranchIfNotZero, "R2, dotf64_tail_loop")
	e.Blank()
}

// dotF64NEONReduceAndReturn emits the horizontal reduce, scalar tail merge, return-slot
// store, and final RET for the NEON float64 dot product kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func dotF64NEONReduceAndReturn(e *asmgen.Emitter) {
	e.Label("dotf64_reduce")
	word := neonFADDP2D(0, 0, 0)
	inst5(e, asmarm64.DirectiveWord, fmt.Sprintf("$0x%08X", word))
	inst5(e, asmarm64.OperationFloatAddScalarDouble, "F3, F0, F0")
	e.Blank()
	inst5(e, asmarm64.OperationFloatMove64Bits, "F0, ret+48(FP)")
	e.Instruction(asmarm64.OperationReturn)
}

// scaleF64NEONPrologue emits the prologue for the NEON in-place float64 scale kernel:
// argument loads, broadcast of the scalar coefficient k from F7 into V7.2D via DUP, and
// the unroll-width guard.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes unrollWidth (int) which is the per-iteration lane count guarding loop entry.
// Takes firstTail (string) which is the branch target for short inputs.
func scaleF64NEONPrologue(e *asmgen.Emitter, unrollWidth int, firstTail string) {
	inst5(e, asmarm64.OperationMove64Bits, "a_base+0(FP), R0")
	inst5(e, asmarm64.OperationMove64Bits, "a_len+8(FP), R2")
	inst5(e, asmarm64.OperationFloatMove64Bits, "k+24(FP), F7")
	dup := neonDUP2DFromDouble(7, 7)
	inst5(e, asmarm64.DirectiveWord, fmt.Sprintf("$0x%08X", dup))
	e.Blank()
	inst5(e, asmarm64.OperationCompare, fmt.Sprintf("$%d, R2", unrollWidth))
	inst5(e, asmarm64.OperationBranchIfLessSigned, firstTail)
}

// scaleF64NEONSlotBody emits one body slot of the NEON in-place float64 scale main loop.
// Each slot uses its own working register so the four slots can issue independent
// load-multiply-store streams.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes off (int) which is the slot's byte offset and selects the working register.
func scaleF64NEONSlotBody(e *asmgen.Emitter, off int) {
	registers := []string{"V0", "V1", "V2", "V3"}
	reg := registers[off/16]
	inst5(e, asmarm64.OperationVectorLoadSingle, fmt.Sprintf("(R0), [%s.D2]", reg))
	word := neonFMUL2D(vRegIndex(reg), vRegIndex(reg), 7)
	inst5(e, asmarm64.DirectiveWord, fmt.Sprintf("$0x%08X", word))
	inst5(e, asmarm64.OperationVectorStoreSingle, fmt.Sprintf("[%s.D2], (R0)", reg))
	if off+16 < 64 {
		inst5(e, asmarm64.OperationAdd, "$16, R0")
	}
}

// scaleF64NEONLoopFooter emits the main-loop footer for the NEON in-place float64 scale
// kernel. The slot bodies already advance R0 between slots, so the footer only needs to
// advance for the final slot, decrement R2, and branch back.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes byteStride (int) which is the per-iteration byte stride used to decrement R2.
func scaleF64NEONLoopFooter(e *asmgen.Emitter, byteStride int) {
	unrollWidth := byteStride / 8
	inst5(e, asmarm64.OperationAdd, "$16, R0")
	inst5(e, asmarm64.OperationSubtract, fmt.Sprintf("$%d, R2", unrollWidth))
	inst5(e, asmarm64.OperationCompare, fmt.Sprintf("$%d, R2", unrollWidth))
	inst5(e, asmarm64.OperationBranchIfGreaterOrEqualSigned, fmt.Sprintf("scalef64neon_loop%d", unrollWidth))
}

// scaleF64NEONNarrowerTail2 emits the 2-wide narrower tail loop for the NEON in-place
// float64 scale kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func scaleF64NEONNarrowerTail2(e *asmgen.Emitter) {
	e.Label("scalef64neon_tail2")
	inst5(e, asmarm64.OperationCompare, "$2, R2")
	inst5(e, asmarm64.OperationBranchIfLessSigned, "scalef64neon_tail")
	e.Blank()
	e.Label("scalef64neon_loop2")
	inst5(e, asmarm64.OperationVectorLoadSingle, "(R0), [V0.D2]")
	word := neonFMUL2D(0, 0, 7)
	inst5(e, asmarm64.DirectiveWord, fmt.Sprintf("$0x%08X", word))
	inst5(e, asmarm64.OperationVectorStoreSingle, "[V0.D2], (R0)")
	inst5(e, asmarm64.OperationAdd, "$16, R0")
	inst5(e, asmarm64.OperationSubtract, "$2, R2")
	inst5(e, asmarm64.OperationCompare, "$2, R2")
	inst5(e, asmarm64.OperationBranchIfGreaterOrEqualSigned, "scalef64neon_loop2")
	e.Blank()
}

// scaleF64NEONScalarTail emits the scalar drain loop for the NEON in-place float64 scale
// kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func scaleF64NEONScalarTail(e *asmgen.Emitter) {
	e.Label("scalef64neon_tail")
	inst5(e, asmarm64.OperationCompareAndBranchIfZero, "R2, scalef64neon_done")
	e.Blank()
	e.Label("scalef64neon_tail_loop")
	inst5(e, asmarm64.OperationFloatMove64Bits, "(R0), F0")
	inst5(e, asmarm64.OperationFloatMultiplyScalarDouble, "F7, F0, F0")
	inst5(e, asmarm64.OperationFloatMove64Bits, "F0, (R0)")
	inst5(e, asmarm64.OperationAdd, "$8, R0")
	inst5(e, asmarm64.OperationSubtract, "$1, R2")
	inst5(e, asmarm64.OperationCompareAndBranchIfNotZero, "R2, scalef64neon_tail_loop")
	e.Blank()
}

// scaleF64NEONReturn emits the done label and RET for the NEON in-place float64 scale
// kernel.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func scaleF64NEONReturn(e *asmgen.Emitter) {
	e.Label("scalef64neon_done")
	e.Instruction(asmarm64.OperationReturn)
}

// vRegIndex parses a "Vn" register name and returns its integer index. Panics on invalid
// input - this is a generator-internal helper that receives only literal register strings
// the caller controls.
//
// Takes name (string) which is a NEON register name like "V0" or "V10".
//
// Returns the integer register index 0..31.
func vRegIndex(name string) uint8 {
	if len(name) < 2 || name[0] != 'V' {
		panic("vRegIndex: not a V-register: " + name)
	}
	idx := 0
	for _, c := range name[1:] {
		if c < '0' || c > '9' {
			panic("vRegIndex: bad digit in " + name)
		}
		idx = idx*10 + int(c-'0')
	}
	if idx > 31 {
		panic("vRegIndex: out of range: " + name)
	}
	return uint8(idx)
}

// emitNEONHorizontalReduce emits the NEON horizontal reduction sequence that collapses
// the four float32 lanes of V0.S4 into a single scalar in F0.
//
// The algorithm performs two pairwise float additions (FADDP) encoded as raw WORD
// instructions ($0x6E20D400), since the Go assembler does not support the FADDP mnemonic
// for 4S vectors. The first FADDP reduces four lanes to two, and the second reduces two
// lanes to one scalar in F0.
//
// Expects the packed accumulator in V0.S4. Produces a scalar sum in F0 (the low lane of
// V0).
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (*VectormathsARM64Arch) emitNEONHorizontalReduce(e *asmgen.Emitter) {
	inst5(e, asmarm64.DirectiveWord, "$0x6E20D400")
	inst5(e, asmarm64.DirectiveWord, "$0x6E20D400")
}

// emitNormaliseSumOfSquares emits the first stage of NEON normalisation: computing the
// sum of squares of every element.
//
// The vector loop loads four float32 values from (R0), squares and accumulates them into
// V0.S4 via VFMLA (self x self). The scalar tail loop handles leftover elements,
// accumulating into F3 via FMADDS. After both loops, the packed accumulator V0 is
// horizontally reduced via two FADDP WORD instructions, and the scalar tail F3 is added
// to produce a single scalar sum in F0.
//
// Expects the function arguments v_base+0(FP) and v_len+8(FP) on the stack frame. Uses
// R0, R2, V0, V1, V3/F3. Produces the total sum of squares as a scalar in F0.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (v *VectormathsARM64Arch) emitNormaliseSumOfSquares(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationMove64Bits, "v_base+0(FP), R0")
	inst5(e, asmarm64.OperationMove64Bits, "v_len+8(FP), R2")
	e.Blank()
	inst5(e, asmarm64.OperationVectorExclusiveOr, "V0.B16, V0.B16, V0.B16")
	inst5(e, asmarm64.OperationVectorExclusiveOr, "V3.B16, V3.B16, V3.B16")
	e.Blank()
	inst5(e, asmarm64.OperationCompare, "$4, R2")
	inst5(e, asmarm64.OperationBranchIfLessSigned, "norm_tail")
	e.Blank()
	e.Label("norm_loop4")
	inst5(e, asmarm64.OperationVectorLoadSingle, "(R0), [V1.S4]")
	inst5(e, asmarm64.OperationVectorFusedMultiplyAdd, "V1.S4, V1.S4, V0.S4")
	inst5(e, asmarm64.OperationAdd, "$16, R0")
	inst5(e, asmarm64.OperationSubtract, "$4, R2")
	inst5(e, asmarm64.OperationCompare, "$4, R2")
	inst5(e, asmarm64.OperationBranchIfGreaterOrEqualSigned, "norm_loop4")
	e.Blank()
	e.Label("norm_tail")
	inst5(e, asmarm64.OperationCompareAndBranchIfZero, "R2, norm_reduce")
	e.Blank()
	e.Label("norm_tail_loop")
	inst5(e, asmarm64.OperationFloatMove32Bits, "(R0), F1")
	inst5(e, asmarm64.OperationFloatMultiplyAddScalarSingle, "F1, F3, F1, F3")
	inst5(e, asmarm64.OperationAdd, "$4, R0")
	inst5(e, asmarm64.OperationSubtract, "$1, R2")
	inst5(e, asmarm64.OperationCompareAndBranchIfNotZero, "R2, norm_tail_loop")
	e.Blank()

	e.Label("norm_reduce")
	v.emitNEONHorizontalReduce(e)
	e.Blank()
	inst5(e, asmarm64.OperationFloatAddScalarSingle, "F3, F0, F0")
	e.Blank()
}

// emitNormaliseZeroCheckAndReciprocalSqrt emits the second stage of NEON normalisation:
// the zero-vector check and reciprocal square root computation.
//
// If the sum of squares in F0 is zero (checked by copying it into R3 and testing with
// CBZ), the function jumps to norm_done, leaving the zero vector unchanged. Otherwise,
// FSQRTS computes the square root, and the reciprocal (1.0/sqrt) is computed via FDIVS.
// The reciprocal is then broadcast to all four lanes of V1.S4 via a DUP instruction
// encoded as WORD $0x4E040421.
//
// Expects the sum of squares as a scalar in F0. Produces the reciprocal norm broadcast in
// V1.S4 (and F1), or jumps to norm_done if the vector is zero.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (*VectormathsARM64Arch) emitNormaliseZeroCheckAndReciprocalSqrt(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationFloatMove32Bits, "F0, R3")
	inst5(e, asmarm64.OperationCompareAndBranchIfZero, "R3, norm_done")
	inst5(e, asmarm64.OperationFloatSquareRootScalarSingle, "F0, F0")
	inst5(e, asmarm64.OperationFloatMove32Bits, "$1.0, F1")
	inst5(e, asmarm64.OperationFloatDivideScalarSingle, "F0, F1, F1")
	inst5(e, asmarm64.DirectiveWord, "$0x4E040421")
	e.Blank()
}

// emitNormaliseWriteBack emits the third stage of NEON normalisation: multiplying every
// element by the reciprocal norm and writing the results back in place.
//
// The vector loop loads four float32 values from (R0) via VLD1, multiplies by the
// broadcast reciprocal in V1.S4 via a FMUL instruction encoded as WORD $0x6E21DC42, and
// stores back via VST1. The scalar tail loop handles leftover elements using FMULS with
// F1. Both loops re-read v_base and v_len from the stack frame since R0 and R2 were
// consumed during the sum-of-squares stage.
//
// Expects the broadcast reciprocal norm in V1.S4 (and F1) and the function arguments
// v_base+0(FP) and v_len+8(FP) on the stack frame. Uses R0, R2, V2/F2 as scratch.
// Produces the normalised vector written in place at v_base.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (*VectormathsARM64Arch) emitNormaliseWriteBack(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationMove64Bits, "v_base+0(FP), R0")
	inst5(e, asmarm64.OperationMove64Bits, "v_len+8(FP), R2")
	inst5(e, asmarm64.OperationCompare, "$4, R2")
	inst5(e, asmarm64.OperationBranchIfLessSigned, "norm_write_tail")
	e.Blank()
	e.Label("norm_write_loop4")
	inst5(e, asmarm64.OperationVectorLoadSingle, "(R0), [V2.S4]")
	inst5(e, asmarm64.DirectiveWord, "$0x6E21DC42")
	inst5(e, asmarm64.OperationVectorStoreSingle, "[V2.S4], (R0)")
	inst5(e, asmarm64.OperationAdd, "$16, R0")
	inst5(e, asmarm64.OperationSubtract, "$4, R2")
	inst5(e, asmarm64.OperationCompare, "$4, R2")
	inst5(e, asmarm64.OperationBranchIfGreaterOrEqualSigned, "norm_write_loop4")
	e.Blank()
	e.Label("norm_write_tail")
	inst5(e, asmarm64.OperationCompareAndBranchIfZero, "R2, norm_done")
	e.Blank()
	e.Label("norm_write_tail_loop")
	inst5(e, asmarm64.OperationFloatMove32Bits, "(R0), F2")
	inst5(e, asmarm64.OperationFloatMultiplyScalarSingle, "F1, F2, F2")
	inst5(e, asmarm64.OperationFloatMove32Bits, "F2, (R0)")
	inst5(e, asmarm64.OperationAdd, "$4, R0")
	inst5(e, asmarm64.OperationSubtract, "$1, R2")
	inst5(e, asmarm64.OperationCompareAndBranchIfNotZero, "R2, norm_write_tail_loop")
	e.Blank()
	e.Label("norm_done")
	e.Instruction(asmarm64.OperationReturn)
}
