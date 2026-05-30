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
	"piko.sh/piko/wdk/asmgen"
	"piko.sh/piko/wdk/asmgen/asmamd64"

	"piko.sh/piko/internal/vectormaths/asm/unrolled"
)

// amd64VectormathsOps implements VectormathsOperationsPort for x86-64. Each method emits
// the complete function body (after the TEXT directive) for a SIMD vectormaths kernel in
// the requested variant.
type amd64VectormathsOps struct{}

var (
	_ asmgen.VectormathsOperationsPort = (*amd64VectormathsOps)(nil)
)

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

// EmitEuclideanDistanceSquared emits the squared Euclidean distance function body for the
// given SIMD variant.
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

// EmitNormalise emits the in-place L2 normalisation function body for the given SIMD
// variant.
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

// EmitSum emits the float64 slice sum function body for the given variant.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes variant (string) which must be "SSE" or "AVX2".
func (v *amd64VectormathsOps) EmitSum(e *asmgen.Emitter, variant string) {
	switch variant {
	case "SSE":
		v.emitSumF64SSE(e)
	case "AVX2":
		v.emitSumF64AVX2(e)
	}
}

// EmitAdd emits the float64 pointwise add function body for the given variant.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes variant (string) which must be "SSE" or "AVX2".
func (v *amd64VectormathsOps) EmitAdd(e *asmgen.Emitter, variant string) {
	switch variant {
	case "SSE":
		v.emitAddF64SSE(e)
	case "AVX2":
		v.emitAddF64AVX2(e)
	}
}

// EmitDotF64 emits the float64 dot product function body for the given variant.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes variant (string) which must be "SSE" or "AVX2".
func (v *amd64VectormathsOps) EmitDotF64(e *asmgen.Emitter, variant string) {
	switch variant {
	case "SSE":
		v.emitDotF64SSE(e)
	case "AVX2":
		v.emitDotF64AVX2(e)
	}
}

// EmitScaleF64 emits the in-place float64 vector-by-scalar multiply function body for the
// given variant.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes variant (string) which must be "SSE" or "AVX2".
func (v *amd64VectormathsOps) EmitScaleF64(e *asmgen.Emitter, variant string) {
	switch variant {
	case "SSE":
		v.emitScaleF64SSE(e)
	case "AVX2":
		v.emitScaleF64AVX2(e)
	}
}

// emitDotF64SSE emits the complete SSE2 float64 dot product function body. 4-way unrolled
// across XMM accumulators X0/X4/X5/X6 (2 doubles each) to break the ADDPD dependency
// chain.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly.
func (v *amd64VectormathsOps) emitDotF64SSE(e *asmgen.Emitter) {
	accumulators := []string{"X0", "X4", "X5", "X6"}
	unrolled.EmitUnrolledReduction(e, unrolled.UnrolledReductionSpec{
		LabelPrefix:        "dotf64sse",
		NarrowerTailLabel:  "_tail2",
		Accumulators:       accumulators,
		LanesPerAcc:        2,
		BytesPerElement:    8,
		BlanksBetweenSlots: true,
		EmitPrologue: func(e *asmgen.Emitter, unrollWidth int, narrowerTailFullLabel string) {
			dotF64SSEPrologue(e, accumulators, unrollWidth, narrowerTailFullLabel)
		},
		EmitOneAccumBody: func(e *asmgen.Emitter, acc string, off int, _ int) {
			dotF64SSEAccumBody(e, acc, off)
		},
		EmitLoopFooter:   dotF64SSELoopFooter,
		EmitFold:         dotF64SSEFold,
		EmitNarrowerLoop: dotF64SSENarrowerLoop,
		EmitScalarTail:   dotF64SSEScalarTail,
		EmitReduceAndReturn: func(e *asmgen.Emitter) {
			dotF64SSEReduceAndReturn(e, v)
		},
	})
}

// emitDotF64AVX2 emits the complete AVX2 float64 dot product function body. 8-way
// unrolled across YMM accumulators Y0/Y4-Y10 (4 doubles each) using VFMADD231PD to fuse
// the multiply and accumulate per slot.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly.
func (v *amd64VectormathsOps) emitDotF64AVX2(e *asmgen.Emitter) {
	accumulators := []string{"Y0", "Y4", "Y5", "Y6", "Y7", "Y8", "Y9", "Y10"}
	unrolled.EmitUnrolledReduction(e, unrolled.UnrolledReductionSpec{
		LabelPrefix:        "dotf64avx",
		NarrowerTailLabel:  "_tail4",
		Accumulators:       accumulators,
		LanesPerAcc:        4,
		BytesPerElement:    8,
		BlanksBetweenSlots: true,
		EmitPrologue: func(e *asmgen.Emitter, unrollWidth int, narrowerTailFullLabel string) {
			dotF64AVX2Prologue(e, accumulators, unrollWidth, narrowerTailFullLabel)
		},
		EmitOneAccumBody: func(e *asmgen.Emitter, acc string, off int, _ int) {
			dotF64AVX2AccumBody(e, acc, off)
		},
		EmitLoopFooter:   dotF64AVX2LoopFooter,
		EmitFold:         dotF64AVX2Fold,
		EmitNarrowerLoop: dotF64AVX2NarrowerLoop,
		EmitScalarTail:   dotF64AVX2ScalarTail,
		EmitReduceAndReturn: func(e *asmgen.Emitter) {
			dotF64AVX2ReduceAndReturn(e, v)
		},
	})
}

// emitScaleF64SSE emits the complete SSE2 in-place float64 scale function body. 2-way
// unrolled across XMM register pairs (X0/X1 and X2/X3) so the two load-multiply-store
// streams can issue independently.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly.
func (*amd64VectormathsOps) emitScaleF64SSE(e *asmgen.Emitter) {
	unrolled.EmitUnrolledPointwise(e, unrolled.UnrolledPointwiseSpec{
		LabelPrefix:            "scalef64sse",
		FirstNarrowerTailLabel: "_tail2",
		UnrollFactor:           2,
		LanesPerSlot:           2,
		BytesPerElement:        8,
		BlanksBetweenSlots:     true,
		EmitPrologue:           scaleF64SSEPrologue,
		EmitOneSlotBody:        scaleF64SSESlotBody,
		EmitLoopFooter:         scaleF64SSELoopFooter,
		NarrowerTails:          []func(e *asmgen.Emitter){scaleF64SSETail2},
		EmitScalarTail:         scaleF64SSEScalarTail,
		EmitReturn:             scaleF64SSEReturn,
	})
}

// emitScaleF64AVX2 emits the complete AVX2 in-place float64 scale function body. 4-way
// unrolled (each slot is one VMULPD with the broadcast scalar in Y7) plus chained 4-wide
// and 2-wide tails before the scalar drain.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly.
func (*amd64VectormathsOps) emitScaleF64AVX2(e *asmgen.Emitter) {
	unrolled.EmitUnrolledPointwise(e, unrolled.UnrolledPointwiseSpec{
		LabelPrefix:            "scalef64avx",
		FirstNarrowerTailLabel: "_tail4",
		UnrollFactor:           4,
		LanesPerSlot:           4,
		BytesPerElement:        8,
		BlanksBetweenSlots:     true,
		EmitPrologue:           scaleF64AVX2Prologue,
		EmitOneSlotBody:        scaleF64AVX2SlotBody,
		EmitLoopFooter:         scaleF64AVX2LoopFooter,
		NarrowerTails:          []func(e *asmgen.Emitter){scaleF64AVX2Tail4, scaleF64AVX2Tail2},
		EmitScalarTail:         scaleF64AVX2ScalarTail,
		EmitReturn:             scaleF64AVX2Return,
	})
}

// emitAddF64SSE emits the complete SSE2 pointwise add. 2-way unrolled across XMM pairs
// (X0/X1 and X2/X3) so the two slots can issue independent load/add/store streams.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (*amd64VectormathsOps) emitAddF64SSE(e *asmgen.Emitter) {
	unrolled.EmitUnrolledPointwise(e, unrolled.UnrolledPointwiseSpec{
		LabelPrefix:            "addf64sse",
		FirstNarrowerTailLabel: "_tail2",
		UnrollFactor:           2,
		LanesPerSlot:           2,
		BytesPerElement:        8,
		BlanksBetweenSlots:     true,
		EmitPrologue:           addF64SSEPrologue,
		EmitOneSlotBody:        addF64SSESlotBody,
		EmitLoopFooter:         addF64SSELoopFooter,
		NarrowerTails:          []func(e *asmgen.Emitter){addF64SSETail2},
		EmitScalarTail:         addF64SSEScalarTail,
		EmitReturn:             addF64SSEReturn,
	})
}

// emitAddF64AVX2 emits the complete AVX2 pointwise add. 4-way unrolled using VADDPD's
// memory-operand form so each slot is a single load, FMA-style fused add, and store.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (*amd64VectormathsOps) emitAddF64AVX2(e *asmgen.Emitter) {
	unrolled.EmitUnrolledPointwise(e, unrolled.UnrolledPointwiseSpec{
		LabelPrefix:            "addf64avx",
		FirstNarrowerTailLabel: "_tail4",
		UnrollFactor:           4,
		LanesPerSlot:           4,
		BytesPerElement:        8,
		BlanksBetweenSlots:     true,
		EmitPrologue:           addF64AVX2Prologue,
		EmitOneSlotBody:        addF64AVX2SlotBody,
		EmitLoopFooter:         addF64AVX2LoopFooter,
		NarrowerTails:          []func(e *asmgen.Emitter){addF64AVX2Tail4, addF64AVX2Tail2},
		EmitScalarTail:         addF64AVX2ScalarTail,
		EmitReturn:             addF64AVX2Return,
	})
}

// emitSSEDoubleHorizontalReduce emits the SSE2 horizontal reduction sequence that
// collapses the two float64 lanes of X0 into a single scalar in the low lane of X0 via
// UNPCKHPD + ADDSD. Uses X1 as scratch.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly.
func (*amd64VectormathsOps) emitSSEDoubleHorizontalReduce(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMoveAlignedPackedDoubles, "X0, X1")
	inst(e, asmamd64.OperationUnpackHighPackedDoubles, "X0, X1")
	inst(e, asmamd64.OperationAddScalarDouble, "X1, X0")
}

// emitAVX2DoubleHorizontalReduce emits the AVX2 horizontal reduction sequence that
// collapses the four float64 lanes of Y0 into a single scalar in the low lane of X0 via
// VEXTRACTF128 + VADDPD + VSHUFPD $1. Uses X1 as scratch.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly.
func (*amd64VectormathsOps) emitAVX2DoubleHorizontalReduce(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationVexExtractFloat128, "$1, Y0, X1")
	inst(e, asmamd64.OperationVexAddPackedDoubles, "X1, X0, X0")
	inst(e, asmamd64.OperationVexShufflePackedDoubles, "$1, X0, X0, X1")
	inst(e, asmamd64.OperationVexAddScalarDouble, "X1, X0, X0")
}

// emitSumF64SSE emits the complete SSE2 float64 sum function body. 4-way unrolled across
// XMM accumulators X0/X4/X5/X6 (2 doubles each).
//
// Takes e (*asmgen.Emitter) which receives the generated assembly.
func (v *amd64VectormathsOps) emitSumF64SSE(e *asmgen.Emitter) {
	accumulators := []string{"X0", "X4", "X5", "X6"}
	unrolled.EmitUnrolledReduction(e, unrolled.UnrolledReductionSpec{
		LabelPrefix:        "sumf64sse",
		NarrowerTailLabel:  "_tail2",
		Accumulators:       accumulators,
		LanesPerAcc:        2,
		BytesPerElement:    8,
		BlanksBetweenSlots: false,
		EmitPrologue: func(e *asmgen.Emitter, unrollWidth int, narrowerTailFullLabel string) {
			sumF64SSEPrologue(e, accumulators, unrollWidth, narrowerTailFullLabel)
		},
		EmitOneAccumBody: func(e *asmgen.Emitter, acc string, off int, _ int) {
			sumF64SSEAccumBody(e, acc, off)
		},
		EmitLoopFooter:   sumF64SSELoopFooter,
		EmitFold:         sumF64SSEFold,
		EmitNarrowerLoop: sumF64SSENarrowerLoop,
		EmitScalarTail:   sumF64SSEScalarTail,
		EmitReduceAndReturn: func(e *asmgen.Emitter) {
			sumF64SSEReduceAndReturn(e, v)
		},
	})
}

// emitSumF64AVX2 emits the complete AVX2 float64 sum function body. 8-way unrolled across
// YMM accumulators Y0/Y4-Y10 (4 doubles each).
//
// Takes e (*asmgen.Emitter) which receives the generated assembly.
func (v *amd64VectormathsOps) emitSumF64AVX2(e *asmgen.Emitter) {
	accumulators := []string{"Y0", "Y4", "Y5", "Y6", "Y7", "Y8", "Y9", "Y10"}
	unrolled.EmitUnrolledReduction(e, unrolled.UnrolledReductionSpec{
		LabelPrefix:        "sumf64avx",
		NarrowerTailLabel:  "_tail4",
		Accumulators:       accumulators,
		LanesPerAcc:        4,
		BytesPerElement:    8,
		BlanksBetweenSlots: false,
		EmitPrologue: func(e *asmgen.Emitter, unrollWidth int, narrowerTailFullLabel string) {
			sumF64AVX2Prologue(e, accumulators, unrollWidth, narrowerTailFullLabel)
		},
		EmitOneAccumBody: func(e *asmgen.Emitter, acc string, off int, _ int) {
			sumF64AVX2AccumBody(e, acc, off)
		},
		EmitLoopFooter:   sumF64AVX2LoopFooter,
		EmitFold:         sumF64AVX2Fold,
		EmitNarrowerLoop: sumF64AVX2NarrowerLoop,
		EmitScalarTail:   sumF64AVX2ScalarTail,
		EmitReduceAndReturn: func(e *asmgen.Emitter) {
			sumF64AVX2ReduceAndReturn(e, v)
		},
	})
}

// emitSSEHorizontalReduce emits the SSE horizontal reduction sequence that collapses the
// four float32 lanes of X0 into a single scalar in the low lane of X0.
//
// The algorithm moves the high pair down (MOVHLPS), adds the two halves, then shuffles
// the second element into X1 and adds once more, leaving the scalar sum in X0[0].
//
// Expects the packed accumulator in X0. Uses X1 as scratch. Produces a scalar sum in the
// low lane of X0.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (*amd64VectormathsOps) emitSSEHorizontalReduce(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMovePackedSinglesHighToLow, "X0, X1")
	inst(e, asmamd64.OperationAddPackedSingles, "X1, X0")
	inst(e, asmamd64.OperationMoveAlignedPackedSingles, "X0, X1")
	inst(e, asmamd64.OperationShufflePackedSingles, "$0x55, X1, X1")
	inst(e, asmamd64.OperationAddScalarSingle, "X1, X0")
}

// emitAVX2HorizontalReduce emits the AVX2 horizontal reduction sequence that collapses
// the eight float32 lanes of Y0 into a single scalar in the low lane of X0.
//
// The algorithm first extracts the upper 128-bit half of Y0 into X1 (VEXTRACTF128), adds
// it to the lower half, then performs two cross-lane shuffles (VSHUFPS) with adds to
// reduce the four remaining lanes down to one scalar.
//
// Expects the packed accumulator in Y0/X0. Uses X1 as scratch. Produces a scalar sum in
// the low lane of X0.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (*amd64VectormathsOps) emitAVX2HorizontalReduce(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationVexExtractFloat128, "$1, Y0, X1")
	inst(e, asmamd64.OperationVexAddPackedSingles, "X1, X0, X0")
	inst(e, asmamd64.OperationVexShufflePackedSingles, "$0x4E, X0, X0, X1")
	inst(e, asmamd64.OperationVexAddPackedSingles, "X1, X0, X0")
	inst(e, asmamd64.OperationVexShufflePackedSingles, "$0x55, X0, X0, X1")
	inst(e, asmamd64.OperationVexAddScalarSingle, "X1, X0, X0")
}

// emitDotProductSSE emits the complete SSE dot product function body. 4-way unrolled
// across XMM accumulators X0/X4/X5/X6 to break the ADDPS dependency chain.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (v *amd64VectormathsOps) emitDotProductSSE(e *asmgen.Emitter) {
	accumulators := []string{"X0", "X4", "X5", "X6"}
	unrolled.EmitUnrolledReduction(e, unrolled.UnrolledReductionSpec{
		LabelPrefix:        "dotsse",
		NarrowerTailLabel:  "_tail4",
		Accumulators:       accumulators,
		LanesPerAcc:        4,
		BytesPerElement:    4,
		BlanksBetweenSlots: true,
		EmitPrologue: func(e *asmgen.Emitter, unrollWidth int, narrowerTailFullLabel string) {
			dotProductSSEPrologue(e, accumulators, unrollWidth, narrowerTailFullLabel)
		},
		EmitOneAccumBody: func(e *asmgen.Emitter, acc string, off int, _ int) {
			dotProductSSEAccumBody(e, acc, off)
		},
		EmitLoopFooter:   dotProductSSELoopFooter,
		EmitFold:         dotProductSSEFold,
		EmitNarrowerLoop: dotProductSSENarrowerLoop,
		EmitScalarTail:   dotProductSSEScalarTail,
		EmitReduceAndReturn: func(e *asmgen.Emitter) {
			dotProductSSEReduceAndReturn(e, v)
		},
	})
}

// emitDotProductAVX2 emits the complete AVX2 dot product function body. 8-way unrolled
// across YMM accumulators Y0/Y4-Y10 to break the VADDPS dependency chain so each
// iteration retires at VMULPS throughput rather than the latency rate of the add.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (v *amd64VectormathsOps) emitDotProductAVX2(e *asmgen.Emitter) {
	accumulators := []string{"Y0", "Y4", "Y5", "Y6", "Y7", "Y8", "Y9", "Y10"}
	unrolled.EmitUnrolledReduction(e, unrolled.UnrolledReductionSpec{
		LabelPrefix:        "dotavx",
		NarrowerTailLabel:  "_tail8",
		Accumulators:       accumulators,
		LanesPerAcc:        8,
		BytesPerElement:    4,
		BlanksBetweenSlots: true,
		EmitPrologue: func(e *asmgen.Emitter, unrollWidth int, narrowerTailFullLabel string) {
			dotProductAVX2Prologue(e, accumulators, unrollWidth, narrowerTailFullLabel)
		},
		EmitOneAccumBody: func(e *asmgen.Emitter, acc string, off int, _ int) {
			dotProductAVX2AccumBody(e, acc, off)
		},
		EmitLoopFooter:   dotProductAVX2LoopFooter,
		EmitFold:         dotProductAVX2Fold,
		EmitNarrowerLoop: dotProductAVX2NarrowerLoop,
		EmitScalarTail:   dotProductAVX2ScalarTail,
		EmitReduceAndReturn: func(e *asmgen.Emitter) {
			dotProductAVX2ReduceAndReturn(e, v)
		},
	})
}

// emitEuclideanDistanceSquaredSSE emits the complete SSE squared Euclidean distance
// function body. 4-way unrolled across XMM accumulators X0/X4/X5/X6.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (v *amd64VectormathsOps) emitEuclideanDistanceSquaredSSE(e *asmgen.Emitter) {
	accumulators := []string{"X0", "X4", "X5", "X6"}
	unrolled.EmitUnrolledReduction(e, unrolled.UnrolledReductionSpec{
		LabelPrefix:        "euclidsse",
		NarrowerTailLabel:  "_tail4",
		Accumulators:       accumulators,
		LanesPerAcc:        4,
		BytesPerElement:    4,
		BlanksBetweenSlots: true,
		EmitPrologue: func(e *asmgen.Emitter, unrollWidth int, narrowerTailFullLabel string) {
			euclidSSEPrologue(e, accumulators, unrollWidth, narrowerTailFullLabel)
		},
		EmitOneAccumBody: func(e *asmgen.Emitter, acc string, off int, _ int) {
			euclidSSEAccumBody(e, acc, off)
		},
		EmitLoopFooter:   euclidSSELoopFooter,
		EmitFold:         euclidSSEFold,
		EmitNarrowerLoop: euclidSSENarrowerLoop,
		EmitScalarTail:   euclidSSEScalarTail,
		EmitReduceAndReturn: func(e *asmgen.Emitter) {
			euclidSSEReduceAndReturn(e, v)
		},
	})
}

// emitEuclideanDistanceSquaredAVX2 emits the complete AVX2 squared Euclidean distance
// function body. 8-way unrolled across YMM accumulators Y0/Y4-Y10, using VFMADD231PS to
// fuse the square and accumulate after VSUBPS.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (v *amd64VectormathsOps) emitEuclideanDistanceSquaredAVX2(e *asmgen.Emitter) {
	accumulators := []string{"Y0", "Y4", "Y5", "Y6", "Y7", "Y8", "Y9", "Y10"}
	unrolled.EmitUnrolledReduction(e, unrolled.UnrolledReductionSpec{
		LabelPrefix:        "euclidavx",
		NarrowerTailLabel:  "_tail8",
		Accumulators:       accumulators,
		LanesPerAcc:        8,
		BytesPerElement:    4,
		BlanksBetweenSlots: true,
		EmitPrologue: func(e *asmgen.Emitter, unrollWidth int, narrowerTailFullLabel string) {
			euclidAVX2Prologue(e, accumulators, unrollWidth, narrowerTailFullLabel)
		},
		EmitOneAccumBody: func(e *asmgen.Emitter, acc string, off int, _ int) {
			euclidAVX2AccumBody(e, acc, off)
		},
		EmitLoopFooter:   euclidAVX2LoopFooter,
		EmitFold:         euclidAVX2Fold,
		EmitNarrowerLoop: euclidAVX2NarrowerLoop,
		EmitScalarTail:   euclidAVX2ScalarTail,
		EmitReduceAndReturn: func(e *asmgen.Emitter) {
			euclidAVX2ReduceAndReturn(e, v)
		},
	})
}

// emitNormaliseSSE emits the complete SSE in-place normalisation function body.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (v *amd64VectormathsOps) emitNormaliseSSE(e *asmgen.Emitter) {
	v.emitNormaliseSSESumOfSquares(e)
	v.emitNormaliseSSEZeroCheckAndReciprocalSqrt(e)
	v.emitNormaliseSSEWriteBack(e)
}

// emitNormaliseSSESumOfSquares emits the sum-of-squares step of SSE normalisation by
// squaring every element and accumulating the total.
//
// The vector loop loads four float32 values from (SI), squares them (MULPS self), and
// accumulates into X0. The scalar tail loop handles leftover elements, accumulating into
// X3. After both loops, the packed accumulator X0 and scalar tail X3 are horizontally
// reduced into a single scalar in X0.
//
// Expects the function arguments v_base+0(FP) and v_len+8(FP) on the stack frame. Uses
// SI, CX, X0, X1, X3. Produces the total sum of squares as a scalar in X0.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (v *amd64VectormathsOps) emitNormaliseSSESumOfSquares(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove64Bits, "v_base+0(FP), SI")
	inst(e, asmamd64.OperationMove64Bits, "v_len+8(FP), CX")
	inst(e, asmamd64.OperationXorPackedSingles, "X0, X0")
	inst(e, asmamd64.OperationXorPackedSingles, "X3, X3")
	e.Blank()
	inst(e, asmamd64.OperationCompare64Bits, "CX, $4")
	inst(e, asmamd64.OperationJumpIfLessSigned, "normsse_tail")
	e.Blank()
	e.Label("normsse_loop4")
	inst(e, asmamd64.OperationMoveUnalignedPackedSingles, "(SI), X1")
	inst(e, asmamd64.OperationMultiplyPackedSingles, "X1, X1")
	inst(e, asmamd64.OperationAddPackedSingles, "X1, X0")
	inst(e, asmamd64.OperationAdd64Bits, "$16, SI")
	inst(e, asmamd64.OperationSubtract64Bits, "$4, CX")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $4")
	inst(e, asmamd64.OperationJumpIfGreaterOrEqualSigned, "normsse_loop4")
	e.Blank()
	e.Label("normsse_tail")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $0")
	inst(e, asmamd64.OperationJumpIfEqual, "normsse_reduce")
	e.Blank()
	e.Label("normsse_tail_loop")
	inst(e, asmamd64.OperationMoveScalarSingle, "(SI), X1")
	inst(e, asmamd64.OperationMultiplyScalarSingle, "X1, X1")
	inst(e, asmamd64.OperationAddScalarSingle, "X1, X3")
	inst(e, asmamd64.OperationAdd64Bits, "$4, SI")
	inst(e, asmamd64.OperationDecrement64Bits, "CX")
	inst(e, asmamd64.OperationJumpIfNotZero, "normsse_tail_loop")
	e.Blank()

	e.Label("normsse_reduce")
	inst(e, asmamd64.OperationAddPackedSingles, "X3, X0")
	v.emitSSEHorizontalReduce(e)
	e.Blank()
}

// emitNormaliseSSEZeroCheckAndReciprocalSqrt emits the zero-vector check and
// reciprocal-square-root step of SSE normalisation.
//
// If the sum of squares in X0 is exactly zero (checked via UCOMISS), the function jumps
// to normsse_done, leaving the zero vector unchanged. Otherwise, SQRTSS computes the
// square root, and the reciprocal (1.0/sqrt) is computed via DIVSS. The reciprocal is
// then broadcast to all four lanes of X1 via SHUFPS for use by the write-back step.
//
// Expects the sum of squares as a scalar in X0. Produces the reciprocal norm broadcast in
// all lanes of X1, or jumps to normsse_done if the vector is zero.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (*amd64VectormathsOps) emitNormaliseSSEZeroCheckAndReciprocalSqrt(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationXorPackedSingles, "X4, X4")
	inst(e, asmamd64.OperationUnorderedCompareScalarSingle, "X4, X0")
	inst(e, asmamd64.OperationJumpIfEqual, "normsse_done")
	inst(e, asmamd64.OperationSquareRootScalarSingle, "X0, X0")
	inst(e, asmamd64.OperationMove32Bits, "$0x3F800000, AX")
	inst(e, asmamd64.OperationMove64Bits, "AX, X1")
	inst(e, asmamd64.OperationDivideScalarSingle, "X0, X1")
	inst(e, asmamd64.OperationShufflePackedSingles, "$0x00, X1, X1")
	e.Blank()
}

// emitNormaliseSSEWriteBack emits the write-back step of SSE normalisation, multiplying
// every element by the reciprocal norm and storing the result back in place.
//
// The vector loop loads four float32 values from (SI), multiplies by the broadcast
// reciprocal in X1, and stores back. The scalar tail loop handles leftover elements. Both
// loops re-read v_base and v_len from the stack frame since SI and CX were consumed
// during the sum-of-squares step.
//
// Expects the broadcast reciprocal norm in X1 and the function arguments v_base+0(FP) and
// v_len+8(FP) on the stack frame. Uses SI, CX, X2 as scratch. Produces the normalised
// vector written in place at v_base.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (*amd64VectormathsOps) emitNormaliseSSEWriteBack(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove64Bits, "v_base+0(FP), SI")
	inst(e, asmamd64.OperationMove64Bits, "v_len+8(FP), CX")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $4")
	inst(e, asmamd64.OperationJumpIfLessSigned, "normsse_write_tail")
	e.Blank()
	e.Label("normsse_write_loop4")
	inst(e, asmamd64.OperationMoveUnalignedPackedSingles, "(SI), X2")
	inst(e, asmamd64.OperationMultiplyPackedSingles, "X1, X2")
	inst(e, asmamd64.OperationMoveUnalignedPackedSingles, "X2, (SI)")
	inst(e, asmamd64.OperationAdd64Bits, "$16, SI")
	inst(e, asmamd64.OperationSubtract64Bits, "$4, CX")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $4")
	inst(e, asmamd64.OperationJumpIfGreaterOrEqualSigned, "normsse_write_loop4")
	e.Blank()
	e.Label("normsse_write_tail")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $0")
	inst(e, asmamd64.OperationJumpIfEqual, "normsse_done")
	e.Blank()
	e.Label("normsse_write_tail_loop")
	inst(e, asmamd64.OperationMoveScalarSingle, "(SI), X2")
	inst(e, asmamd64.OperationMultiplyScalarSingle, "X1, X2")
	inst(e, asmamd64.OperationMoveScalarSingle, "X2, (SI)")
	inst(e, asmamd64.OperationAdd64Bits, "$4, SI")
	inst(e, asmamd64.OperationDecrement64Bits, "CX")
	inst(e, asmamd64.OperationJumpIfNotZero, "normsse_write_tail_loop")
	e.Blank()
	e.Label("normsse_done")
	e.Instruction(asmamd64.OperationReturn)
}

// emitNormaliseAVX2 emits the complete AVX2 in-place normalisation function body.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (v *amd64VectormathsOps) emitNormaliseAVX2(e *asmgen.Emitter) {
	v.emitNormaliseAVX2SumOfSquares(e)
	v.emitNormaliseAVX2ZeroCheckAndReciprocalSqrt(e)
	v.emitNormaliseAVX2WriteBack(e)
}

// emitNormaliseAVX2SumOfSquares emits the sum-of-squares step of AVX2 normalisation by
// squaring every element and accumulating the total.
//
// The vector loop loads eight float32 values from (SI), squares them (VMULPS self), and
// accumulates into Y0. The scalar tail loop handles leftover elements, accumulating into
// X3. After both loops, the packed accumulator Y0 and scalar tail X3 are horizontally
// reduced into a single scalar in X0.
//
// Expects the function arguments v_base+0(FP) and v_len+8(FP) on the stack frame. Uses
// SI, CX, Y0, Y1, X3. Produces the total sum of squares as a scalar in X0.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (v *amd64VectormathsOps) emitNormaliseAVX2SumOfSquares(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove64Bits, "v_base+0(FP), SI")
	inst(e, asmamd64.OperationMove64Bits, "v_len+8(FP), CX")
	inst(e, asmamd64.OperationVexXorPackedSingles, "Y0, Y0, Y0")
	inst(e, asmamd64.OperationVexXorPackedSingles, "X3, X3, X3")
	e.Blank()
	inst(e, asmamd64.OperationCompare64Bits, "CX, $8")
	inst(e, asmamd64.OperationJumpIfLessSigned, "normavx_tail")
	e.Blank()
	e.Label("normavx_loop8")
	inst(e, asmamd64.OperationVexMoveUnalignedPackedSingles, "(SI), Y1")
	inst(e, asmamd64.OperationVexMultiplyPackedSingles, "Y1, Y1, Y1")
	inst(e, asmamd64.OperationVexAddPackedSingles, "Y1, Y0, Y0")
	inst(e, asmamd64.OperationAdd64Bits, "$32, SI")
	inst(e, asmamd64.OperationSubtract64Bits, "$8, CX")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $8")
	inst(e, asmamd64.OperationJumpIfGreaterOrEqualSigned, "normavx_loop8")
	e.Blank()
	e.Label("normavx_tail")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $0")
	inst(e, asmamd64.OperationJumpIfEqual, "normavx_reduce")
	e.Blank()
	e.Label("normavx_tail_loop")
	inst(e, asmamd64.OperationVexMoveScalarSingle, "(SI), X1")
	inst(e, asmamd64.OperationVexMultiplyScalarSingle, "X1, X1, X1")
	inst(e, asmamd64.OperationVexAddScalarSingle, "X1, X3, X3")
	inst(e, asmamd64.OperationAdd64Bits, "$4, SI")
	inst(e, asmamd64.OperationDecrement64Bits, "CX")
	inst(e, asmamd64.OperationJumpIfNotZero, "normavx_tail_loop")
	e.Blank()

	e.Label("normavx_reduce")
	v.emitAVX2HorizontalReduce(e)
	inst(e, asmamd64.OperationVexAddScalarSingle, "X3, X0, X0")
	e.Blank()
}

// emitNormaliseAVX2ZeroCheckAndReciprocalSqrt emits the zero-vector check and
// reciprocal-square-root step of AVX2 normalisation.
//
// If the sum of squares in X0 is exactly zero (checked via VUCOMISS), the function jumps
// to normavx_done_zero, issuing VZEROUPPER before returning to avoid SSE/AVX transition
// penalties. Otherwise, VSQRTSS computes the square root, and the reciprocal (1.0/sqrt)
// is computed via VDIVSS. The reciprocal is then broadcast to all eight lanes of Y1 via
// VBROADCASTSS for use by the write-back step.
//
// Expects the sum of squares as a scalar in X0. Produces the reciprocal norm broadcast in
// all lanes of Y1, or jumps to normavx_done_zero if the vector is zero.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (*amd64VectormathsOps) emitNormaliseAVX2ZeroCheckAndReciprocalSqrt(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationVexXorPackedSingles, "X4, X4, X4")
	inst(e, asmamd64.OperationVexUnorderedCompareScalarSingle, "X4, X0")
	inst(e, asmamd64.OperationJumpIfEqual, "normavx_done_zero")
	inst(e, asmamd64.OperationVexSquareRootScalarSingle, "X0, X0, X0")
	inst(e, asmamd64.OperationMove32Bits, "$0x3F800000, AX")
	inst(e, asmamd64.OperationMove64Bits, "AX, X1")
	inst(e, asmamd64.OperationVexDivideScalarSingle, "X0, X1, X1")
	inst(e, asmamd64.OperationVexBroadcastScalarSingle, "X1, Y1")
	e.Blank()
}

// emitNormaliseAVX2WriteBack emits the write-back step of AVX2 normalisation, multiplying
// every element by the reciprocal norm and storing the result back in place.
//
// The vector loop loads eight float32 values from (SI), multiplies by the broadcast
// reciprocal in Y1, and stores back. The scalar tail loop handles leftover elements using
// X1 (the low lane of Y1 still holds the scalar reciprocal). Both loops re-read v_base
// and v_len from the stack frame since SI and CX were consumed during the sum-of-squares
// step.
//
// After the write-back is complete (or if no elements remain), VZEROUPPER is issued
// before RET. A separate normavx_done_zero label handles the zero-vector early exit with
// its own VZEROUPPER.
//
// Expects the broadcast reciprocal norm in Y1 (and X1) and the function arguments
// v_base+0(FP) and v_len+8(FP) on the stack frame. Uses SI, CX, Y2/X2 as scratch.
// Produces the normalised vector written in place at v_base.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
func (*amd64VectormathsOps) emitNormaliseAVX2WriteBack(e *asmgen.Emitter) {
	inst(e, asmamd64.OperationMove64Bits, "v_base+0(FP), SI")
	inst(e, asmamd64.OperationMove64Bits, "v_len+8(FP), CX")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $8")
	inst(e, asmamd64.OperationJumpIfLessSigned, "normavx_write_tail")
	e.Blank()
	e.Label("normavx_write_loop8")
	inst(e, asmamd64.OperationVexMoveUnalignedPackedSingles, "(SI), Y2")
	inst(e, asmamd64.OperationVexMultiplyPackedSingles, "Y1, Y2, Y2")
	inst(e, asmamd64.OperationVexMoveUnalignedPackedSingles, "Y2, (SI)")
	inst(e, asmamd64.OperationAdd64Bits, "$32, SI")
	inst(e, asmamd64.OperationSubtract64Bits, "$8, CX")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $8")
	inst(e, asmamd64.OperationJumpIfGreaterOrEqualSigned, "normavx_write_loop8")
	e.Blank()
	e.Label("normavx_write_tail")
	inst(e, asmamd64.OperationCompare64Bits, "CX, $0")
	inst(e, asmamd64.OperationJumpIfEqual, "normavx_done")
	e.Blank()
	e.Label("normavx_write_tail_loop")
	inst(e, asmamd64.OperationVexMoveScalarSingle, "(SI), X2")
	inst(e, asmamd64.OperationVexMultiplyScalarSingle, "X1, X2, X2")
	inst(e, asmamd64.OperationVexMoveScalarSingle, "X2, (SI)")
	inst(e, asmamd64.OperationAdd64Bits, "$4, SI")
	inst(e, asmamd64.OperationDecrement64Bits, "CX")
	inst(e, asmamd64.OperationJumpIfNotZero, "normavx_write_tail_loop")
	e.Blank()
	e.Label("normavx_done")
	e.Instruction(asmamd64.OperationVexZeroUpper)
	e.Instruction(asmamd64.OperationReturn)
	e.Blank()
	e.Label("normavx_done_zero")
	e.Instruction(asmamd64.OperationVexZeroUpper)
	e.Instruction(asmamd64.OperationReturn)
}
