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

// Package unrolled orchestrates multi-accumulator-unrolled SIMD kernels.
package unrolled

import (
	"strconv"

	"piko.sh/piko/wdk/asmgen"
)

// UnrolledReductionSpec describes a multi-accumulator SIMD reduction kernel such as dot
// product, sum, or squared Euclidean distance.
//
// All mnemonic-emitting work is delegated to closures supplied by the per-arch caller,
// keeping the orchestrator independent of specific instruction vocabularies and
// arch-specific column-padding helpers.
type UnrolledReductionSpec struct {
	// EmitFold reduces Accumulators[1:] into Accumulators[0]. Typically emits
	// len(accumulators)-1 successive add-style instructions with no blank lines between
	// them.
	EmitFold func(e *asmgen.Emitter, accumulators []string)

	// EmitPrologue emits everything before the main loop label: argument MOVQ/MOVD loads,
	// accumulator zeroing (every entry of Accumulators plus the scalar tail accumulator),
	// and the "CMPQ CX, $unrollWidth ; JL {narrowerTailFullLabel}" guard.
	//
	// Receives the computed unrollWidth (len(Accumulators) * LanesPerAcc) and the
	// fully-qualified narrower-tail label (LabelPrefix + NarrowerTailLabel) so the closure
	// does not need to recompute either.
	EmitPrologue func(e *asmgen.Emitter, unrollWidth int, narrowerTailFullLabel string)

	// EmitOneAccumBody emits a single slot of the unrolled main loop body.
	//
	// Called len(Accumulators) times with the rotating accumulator, the per-slot byte offset
	// (0, lanes*bytes, 2*lanes*bytes, ...), and the slot index. amd64 closures use
	// byteOffset for displacement addressing; arm64 closures often ignore byteOffset and use
	// slotIndex to gate inline pointer-advance instructions emitted before all slots except
	// the first.
	EmitOneAccumBody func(e *asmgen.Emitter, accumulator string, byteOffset int, slotIndex int)

	// EmitLoopFooter emits the per-iteration pointer-advance, counter decrement, comparison,
	// and conditional branch back to the loop label. The orchestrator passes the computed
	// per-iteration byte stride so the closure does not need to recompute it.
	EmitLoopFooter func(e *asmgen.Emitter, byteStride int)

	// EmitNarrowerLoop emits the narrower-SIMD tail label and its body.
	//
	// Owns the {LabelPrefix}{NarrowerTailLabel} label. Falls through to the scalar tail when
	// the narrower count drops below the narrower width.
	EmitNarrowerLoop func(e *asmgen.Emitter)

	// EmitScalarTail emits the {LabelPrefix}_tail label and the one-element-at-a-time drain
	// into the scalar tail accumulator. Falls through to the reduce label.
	EmitScalarTail func(e *asmgen.Emitter)

	// EmitReduceAndReturn emits the {LabelPrefix}_reduce label, the horizontal reduction of
	// Accumulators[0] to a scalar, the addition of the scalar tail accumulator, any
	// VZEROUPPER, the return-slot store, and the final RET.
	EmitReduceAndReturn func(e *asmgen.Emitter)

	// LabelPrefix is the per-kernel namespace for jump labels. For example "dotavx" yields
	// labels "dotavx_loop64", "dotavx_tail8", "dotavx_tail", "dotavx_reduce".
	LabelPrefix string

	// NarrowerTailLabel is the suffix appended to LabelPrefix to name the intermediate
	// narrower-SIMD tail loop label.
	//
	// Example values: "_tail8", "_tail4", "_tail2". The orchestrator emits "CMPQ CX,
	// $unrollWidth ; JL LabelPrefix+NarrowerTailLabel" via EmitPrologue.
	NarrowerTailLabel string

	// Accumulators lists the per-slot accumulator register names in the order they receive
	// their slot output.
	//
	// For amd64 AVX2 reductions this is typically
	// ["Y0","Y4","Y5","Y6","Y7","Y8","Y9","Y10"]; for amd64 SSE typically
	// ["X0","X4","X5","X6"]; for arm64 NEON the V-prefixed equivalents. len(Accumulators) is
	// the unroll factor.
	Accumulators []string

	// LanesPerAcc is the number of SIMD lanes each accumulator holds. Combined with
	// BytesPerElement and len(Accumulators) this determines the per-iteration byte stride
	// and the unroll-width threshold.
	LanesPerAcc int

	// BytesPerElement is the element size in bytes: 4 for float32 kernels, 8 for float64
	// kernels.
	BytesPerElement int

	// BlanksBetweenSlots inserts a blank line between consecutive body slots in the main
	// loop. The hand-written goldens use blanks for dotF32/euclidSqF32/sumF64 amd64 variants
	// but suppress them for the arm64 sumF64Kern variant (where the body slot itself
	// contains an inline pointer-advance ADD that is visually closer to the load).
	BlanksBetweenSlots bool
}

// EmitUnrolledReduction orchestrates the canonical layout of a multi- accumulator SIMD
// reduction kernel. The orchestrator owns only labels, blank lines, and the per-slot
// iteration; every mnemonic-bearing instruction is emitted by spec closures, keeping the
// layout arch-agnostic.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly.
// Takes spec (UnrolledReductionSpec) which describes the kernel.
func EmitUnrolledReduction(e *asmgen.Emitter, spec UnrolledReductionSpec) {
	unrollWidth := len(spec.Accumulators) * spec.LanesPerAcc
	byteStride := unrollWidth * spec.BytesPerElement
	laneBytes := spec.LanesPerAcc * spec.BytesPerElement
	narrowerTailFullLabel := spec.LabelPrefix + spec.NarrowerTailLabel

	spec.EmitPrologue(e, unrollWidth, narrowerTailFullLabel)
	e.Blank()
	e.Label(spec.LabelPrefix + "_loop" + strconv.Itoa(unrollWidth))
	for i, acc := range spec.Accumulators {
		spec.EmitOneAccumBody(e, acc, i*laneBytes, i)
		if spec.BlanksBetweenSlots {
			e.Blank()
		}
	}
	spec.EmitLoopFooter(e, byteStride)
	e.Blank()
	spec.EmitFold(e, spec.Accumulators)
	e.Blank()
	spec.EmitNarrowerLoop(e)
	spec.EmitScalarTail(e)
	spec.EmitReduceAndReturn(e)
}

// UnrolledPointwiseSpec describes a pointwise SIMD kernel such as element- wise addition
// or scaling. Pointwise kernels have no accumulator chain to break and therefore no fold
// or horizontal-reduce phase; the unroll exists purely to amortise loop overhead and feed
// the load/store ports.
//
// EmitUnrolledPointwise mirrors EmitUnrolledReduction's shape but drops the fold and
// reduce stages and supports a chain of progressively narrower tail handlers (e.g. 4-wide
// loop tail, then 2-wide straight-line tail, then scalar drain for amd64 addF64AVX2).
type UnrolledPointwiseSpec struct {
	// EmitPrologue emits everything before the main loop label: argument MOVQ/MOVD loads
	// (including any non-pointer arguments such as a broadcast scalar for scaleF64), the
	// "CMPQ CX, $unrollWidth ; JL {firstNarrowerTailFullLabel}" guard.
	EmitPrologue func(e *asmgen.Emitter, unrollWidth int, firstNarrowerTailFullLabel string)

	// EmitOneSlotBody emits a single slot of the unrolled main loop body at the given byte
	// offset. Called UnrollFactor times.
	EmitOneSlotBody func(e *asmgen.Emitter, byteOffset int)

	// EmitLoopFooter emits the per-iteration pointer advances, counter decrement,
	// comparison, and branch back.
	EmitLoopFooter func(e *asmgen.Emitter, byteStride int)

	// EmitScalarTail emits the final {LabelPrefix}_tail label and the one-element-at-a-time
	// drain.
	EmitScalarTail func(e *asmgen.Emitter)

	// EmitReturn emits the {LabelPrefix}_done label (if any), any VZEROUPPER, and the final
	// RET. Pointwise kernels do not have a reduce phase.
	EmitReturn func(e *asmgen.Emitter)

	// LabelPrefix is the per-kernel namespace for jump labels. For example "addf64avx"
	// yields labels "addf64avx_loop16", "addf64avx_tail4", and any further narrower-tail
	// labels supplied via NarrowerTailLabels.
	LabelPrefix string

	// FirstNarrowerTailLabel is the suffix appended to LabelPrefix for the first (widest)
	// narrower-tail label. Example values: "_tail4", "_tail2".
	FirstNarrowerTailLabel string

	// NarrowerTails is the chain of progressively narrower tail handlers.
	//
	// Each one owns its own label (the first uses FirstNarrowerTailLabel; later ones must
	// emit their own labels). The orchestrator calls them in order; the last one should fall
	// through to the scalar tail.
	NarrowerTails []func(e *asmgen.Emitter)

	// UnrollFactor is the number of body slots emitted per main-loop iteration. Typical
	// values: 4 for AVX2 / NEON pointwise, 2 for SSE.
	UnrollFactor int

	// LanesPerSlot is the number of SIMD lanes processed by each body slot. Combined with
	// UnrollFactor and BytesPerElement this determines the per-iteration byte stride.
	LanesPerSlot int

	// BytesPerElement is the element size in bytes: 4 for float32, 8 for float64.
	BytesPerElement int

	// BlanksBetweenSlots inserts a blank line between consecutive body slots in the main
	// loop.
	BlanksBetweenSlots bool
}

// EmitUnrolledPointwise orchestrates a pointwise (load-op-store) SIMD kernel. Mirrors
// EmitUnrolledReduction but omits the fold and horizontal reduce stages and supports a
// chain of progressively narrower tail handlers.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly.
// Takes spec (UnrolledPointwiseSpec) which describes the kernel.
func EmitUnrolledPointwise(e *asmgen.Emitter, spec UnrolledPointwiseSpec) {
	unrollWidth := spec.UnrollFactor * spec.LanesPerSlot
	byteStride := unrollWidth * spec.BytesPerElement
	slotBytes := spec.LanesPerSlot * spec.BytesPerElement
	firstNarrowerTailFullLabel := spec.LabelPrefix + spec.FirstNarrowerTailLabel

	spec.EmitPrologue(e, unrollWidth, firstNarrowerTailFullLabel)
	e.Blank()
	e.Label(spec.LabelPrefix + "_loop" + strconv.Itoa(unrollWidth))
	for i := range spec.UnrollFactor {
		spec.EmitOneSlotBody(e, i*slotBytes)
		if spec.BlanksBetweenSlots {
			e.Blank()
		}
	}
	spec.EmitLoopFooter(e, byteStride)
	e.Blank()
	for _, narrowerTail := range spec.NarrowerTails {
		narrowerTail(e)
	}
	spec.EmitScalarTail(e)
	spec.EmitReturn(e)
}
