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

package unrolled

import (
	"fmt"
	"strings"
	"testing"

	"piko.sh/piko/wdk/asmgen"

	"github.com/stretchr/testify/require"
)

func TestEmitUnrolledReduction_OrchestrationOrder(t *testing.T) {
	t.Parallel()

	e := &asmgen.Emitter{}
	var trace []string

	spec := UnrolledReductionSpec{
		LabelPrefix:        "stub",
		NarrowerTailLabel:  "_tail4",
		Accumulators:       []string{"A0", "A1", "A2"},
		LanesPerAcc:        4,
		BytesPerElement:    4,
		BlanksBetweenSlots: true,
		EmitPrologue: func(e *asmgen.Emitter, unrollWidth int, narrowerTailFullLabel string) {
			trace = append(trace, fmt.Sprintf("prologue:width=%d:tail=%s", unrollWidth, narrowerTailFullLabel))
			e.Raw("\tPROLOGUE\n")
		},
		EmitOneAccumBody: func(e *asmgen.Emitter, accumulator string, byteOffset int, _ int) {
			trace = append(trace, fmt.Sprintf("body:acc=%s:off=%d", accumulator, byteOffset))
			e.Raw(fmt.Sprintf("\tBODY %s @%d\n", accumulator, byteOffset))
		},
		EmitLoopFooter: func(e *asmgen.Emitter, byteStride int) {
			trace = append(trace, fmt.Sprintf("footer:stride=%d", byteStride))
			e.Raw(fmt.Sprintf("\tFOOTER stride=%d\n", byteStride))
		},
		EmitFold: func(e *asmgen.Emitter, accumulators []string) {
			trace = append(trace, fmt.Sprintf("fold:%s", strings.Join(accumulators, ",")))
			e.Raw(fmt.Sprintf("\tFOLD %s\n", strings.Join(accumulators, ",")))
		},
		EmitNarrowerLoop: func(e *asmgen.Emitter) {
			trace = append(trace, "narrower")
			e.Label("stub_tail4")
			e.Raw("\tNARROWER\n")
		},
		EmitScalarTail: func(e *asmgen.Emitter) {
			trace = append(trace, "scalar")
			e.Label("stub_tail")
			e.Raw("\tSCALAR\n")
		},
		EmitReduceAndReturn: func(e *asmgen.Emitter) {
			trace = append(trace, "reduce")
			e.Label("stub_reduce")
			e.Raw("\tREDUCE_RET\n")
		},
	}

	EmitUnrolledReduction(e, spec)

	require.Equal(t, []string{
		"prologue:width=12:tail=stub_tail4",
		"body:acc=A0:off=0",
		"body:acc=A1:off=16",
		"body:acc=A2:off=32",
		"footer:stride=48",
		"fold:A0,A1,A2",
		"narrower",
		"scalar",
		"reduce",
	}, trace, "closure invocation order")

	got := e.String()
	wantLines := []string{
		"\tPROLOGUE",
		"",
		"stub_loop12:",
		"\tBODY A0 @0",
		"",
		"\tBODY A1 @16",
		"",
		"\tBODY A2 @32",
		"",
		"\tFOOTER stride=48",
		"",
		"\tFOLD A0,A1,A2",
		"",
		"stub_tail4:",
		"\tNARROWER",
		"stub_tail:",
		"\tSCALAR",
		"stub_reduce:",
		"\tREDUCE_RET",
		"",
	}
	require.Equal(t, strings.Join(wantLines, "\n"), got, "emitted text")
}

func TestEmitUnrolledReduction_NoBlanksBetweenSlots(t *testing.T) {
	t.Parallel()

	e := &asmgen.Emitter{}
	spec := UnrolledReductionSpec{
		LabelPrefix:         "stub",
		NarrowerTailLabel:   "_tail2",
		Accumulators:        []string{"A0", "A1"},
		LanesPerAcc:         2,
		BytesPerElement:     8,
		BlanksBetweenSlots:  false,
		EmitPrologue:        func(e *asmgen.Emitter, _ int, _ string) { e.Raw("\tP\n") },
		EmitOneAccumBody:    func(e *asmgen.Emitter, acc string, _ int, _ int) { e.Raw("\tB " + acc + "\n") },
		EmitLoopFooter:      func(e *asmgen.Emitter, _ int) { e.Raw("\tF\n") },
		EmitFold:            func(e *asmgen.Emitter, _ []string) { e.Raw("\tFOLD\n") },
		EmitNarrowerLoop:    func(e *asmgen.Emitter) { e.Raw("\tN\n") },
		EmitScalarTail:      func(e *asmgen.Emitter) { e.Raw("\tS\n") },
		EmitReduceAndReturn: func(e *asmgen.Emitter) { e.Raw("\tR\n") },
	}

	EmitUnrolledReduction(e, spec)

	got := e.String()
	require.NotContains(t, got, "\tB A0\n\n\tB A1\n", "must not have blank between slots when BlanksBetweenSlots=false")
	require.Contains(t, got, "\tB A0\n\tB A1\n", "consecutive bodies appear without separator")
}

func TestEmitUnrolledPointwise_OrchestrationOrder(t *testing.T) {
	t.Parallel()

	e := &asmgen.Emitter{}
	var trace []string

	spec := UnrolledPointwiseSpec{
		LabelPrefix:            "stubp",
		FirstNarrowerTailLabel: "_tail4",
		UnrollFactor:           4,
		LanesPerSlot:           4,
		BytesPerElement:        8,
		BlanksBetweenSlots:     true,
		EmitPrologue: func(e *asmgen.Emitter, unrollWidth int, firstTail string) {
			trace = append(trace, fmt.Sprintf("prologue:width=%d:firstTail=%s", unrollWidth, firstTail))
			e.Raw("\tPROLOGUE\n")
		},
		EmitOneSlotBody: func(e *asmgen.Emitter, byteOffset int) {
			trace = append(trace, fmt.Sprintf("slot:%d", byteOffset))
			e.Raw(fmt.Sprintf("\tSLOT %d\n", byteOffset))
		},
		EmitLoopFooter: func(e *asmgen.Emitter, byteStride int) {
			trace = append(trace, fmt.Sprintf("footer:stride=%d", byteStride))
			e.Raw(fmt.Sprintf("\tFOOTER %d\n", byteStride))
		},
		NarrowerTails: []func(e *asmgen.Emitter){
			func(e *asmgen.Emitter) { trace = append(trace, "tail4"); e.Label("stubp_tail4"); e.Raw("\tN4\n") },
			func(e *asmgen.Emitter) { trace = append(trace, "tail2"); e.Label("stubp_tail2"); e.Raw("\tN2\n") },
		},
		EmitScalarTail: func(e *asmgen.Emitter) { trace = append(trace, "scalar"); e.Label("stubp_tail"); e.Raw("\tS\n") },
		EmitReturn:     func(e *asmgen.Emitter) { trace = append(trace, "ret"); e.Raw("\tRET\n") },
	}

	EmitUnrolledPointwise(e, spec)

	require.Equal(t, []string{
		"prologue:width=16:firstTail=stubp_tail4",
		"slot:0", "slot:32", "slot:64", "slot:96",
		"footer:stride=128",
		"tail4", "tail2", "scalar", "ret",
	}, trace)
}
