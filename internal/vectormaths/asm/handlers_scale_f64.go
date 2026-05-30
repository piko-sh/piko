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

// scaleF64Handlers returns the handler definitions for the in-place float64
// vector-by-scalar multiply SIMD functions across all supported architectures.
//
// Returns []asmgen.HandlerDefinition[VectormathsArchitecturePort] containing SSE, AVX2,
// and NEON variants.
func scaleF64Handlers() []asmgen.HandlerDefinition[VectormathsArchitecturePort] {
	return []asmgen.HandlerDefinition[VectormathsArchitecturePort]{
		handlerScaleF64SSE(),
		handlerScaleF64AVX2(),
		handlerScaleF64NEON(),
	}
}

// handlerScaleF64SSE returns the handler definition for the SSE in-place float64 scale
// function.
//
// Returns asmgen.HandlerDefinition[VectormathsArchitecturePort] targeting amd64 with a
// $0-32 NOSPLIT frame.
func handlerScaleF64SSE() asmgen.HandlerDefinition[VectormathsArchitecturePort] {
	return asmgen.HandlerDefinition[VectormathsArchitecturePort]{
		Name:          "scaleF64SSE",
		Comment:       "scaleF64SSE multiplies each f64 element by k in place using 2-way SSE2 unrolling.",
		Architectures: []asmgen.Architecture{asmgen.ArchitectureAMD64},
		FrameSize:     "$0-32", Flags: "NOSPLIT",
		Emit: func(emitter *asmgen.Emitter, architecture VectormathsArchitecturePort) {
			architecture.EmitScaleF64(emitter, "SSE")
		},
	}
}

// handlerScaleF64AVX2 returns the handler definition for the AVX2 in-place float64 scale
// function.
//
// Returns asmgen.HandlerDefinition[VectormathsArchitecturePort] targeting amd64 with a
// $0-32 NOSPLIT frame.
func handlerScaleF64AVX2() asmgen.HandlerDefinition[VectormathsArchitecturePort] {
	return asmgen.HandlerDefinition[VectormathsArchitecturePort]{
		Name:          "scaleF64AVX2",
		Comment:       "scaleF64AVX2 multiplies each f64 element by k in place using 4-way AVX2 unrolling.",
		Architectures: []asmgen.Architecture{asmgen.ArchitectureAMD64},
		FrameSize:     "$0-32", Flags: "NOSPLIT",
		Emit: func(emitter *asmgen.Emitter, architecture VectormathsArchitecturePort) {
			architecture.EmitScaleF64(emitter, "AVX2")
		},
	}
}

// handlerScaleF64NEON returns the handler definition for the NEON in-place float64 scale
// function.
//
// Returns asmgen.HandlerDefinition[VectormathsArchitecturePort] targeting arm64 with a
// $0-32 NOSPLIT frame.
func handlerScaleF64NEON() asmgen.HandlerDefinition[VectormathsArchitecturePort] {
	return asmgen.HandlerDefinition[VectormathsArchitecturePort]{
		Name:          "scaleF64Kern",
		Comment:       "scaleF64Kern multiplies each f64 element by k in place using 4-way NEON unrolling.",
		Architectures: []asmgen.Architecture{asmgen.ArchitectureARM64},
		FrameSize:     "$0-32", Flags: "NOSPLIT",
		Emit: func(emitter *asmgen.Emitter, architecture VectormathsArchitecturePort) {
			architecture.EmitScaleF64(emitter, "NEON")
		},
	}
}
