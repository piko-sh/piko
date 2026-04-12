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

package asm

import (
	"piko.sh/piko/wdk/asmgen"
)

// sumF64Handlers returns the handler definitions for the float64 slice sum SIMD functions
// across all supported architectures.
//
// Returns []asmgen.HandlerDefinition[VectormathsArchitecturePort] containing SSE, AVX2,
// and NEON variants.
func sumF64Handlers() []asmgen.HandlerDefinition[VectormathsArchitecturePort] {
	return []asmgen.HandlerDefinition[VectormathsArchitecturePort]{
		handlerSumF64SSE(),
		handlerSumF64AVX2(),
		handlerSumF64NEON(),
	}
}

// handlerSumF64SSE returns the handler definition for the SSE float64 sum function.
//
// Returns asmgen.HandlerDefinition[VectormathsArchitecturePort] targeting amd64 with a
// $0-32 NOSPLIT frame.
func handlerSumF64SSE() asmgen.HandlerDefinition[VectormathsArchitecturePort] {
	return asmgen.HandlerDefinition[VectormathsArchitecturePort]{
		Name:          "sumF64SSE",
		Comment:       "sumF64SSE computes sum(a[i]) for f64 vectors using 4-way SSE2 unrolling.",
		Architectures: []asmgen.Architecture{asmgen.ArchitectureAMD64},
		FrameSize:     "$0-32", Flags: "NOSPLIT",
		Emit: func(emitter *asmgen.Emitter, architecture VectormathsArchitecturePort) {
			architecture.EmitSum(emitter, "SSE")
		},
	}
}

// handlerSumF64AVX2 returns the handler definition for the AVX2 float64 sum function.
//
// Returns asmgen.HandlerDefinition[VectormathsArchitecturePort] targeting amd64 with a
// $0-32 NOSPLIT frame.
func handlerSumF64AVX2() asmgen.HandlerDefinition[VectormathsArchitecturePort] {
	return asmgen.HandlerDefinition[VectormathsArchitecturePort]{
		Name:          "sumF64AVX2",
		Comment:       "sumF64AVX2 computes sum(a[i]) for f64 vectors using 8-way AVX2 unrolling.",
		Architectures: []asmgen.Architecture{asmgen.ArchitectureAMD64},
		FrameSize:     "$0-32", Flags: "NOSPLIT",
		Emit: func(emitter *asmgen.Emitter, architecture VectormathsArchitecturePort) {
			architecture.EmitSum(emitter, "AVX2")
		},
	}
}

// handlerSumF64NEON returns the handler definition for the NEON float64 sum function.
//
// Returns asmgen.HandlerDefinition[VectormathsArchitecturePort] targeting arm64 with a
// $0-32 NOSPLIT frame.
func handlerSumF64NEON() asmgen.HandlerDefinition[VectormathsArchitecturePort] {
	return asmgen.HandlerDefinition[VectormathsArchitecturePort]{
		Name:          "sumF64Kern",
		Comment:       "sumF64Kern computes sum(a[i]) for f64 vectors using 8-way NEON unrolling.",
		Architectures: []asmgen.Architecture{asmgen.ArchitectureARM64},
		FrameSize:     "$0-32", Flags: "NOSPLIT",
		Emit: func(emitter *asmgen.Emitter, architecture VectormathsArchitecturePort) {
			architecture.EmitSum(emitter, "NEON")
		},
	}
}
