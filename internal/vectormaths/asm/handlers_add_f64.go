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

// addF64Handlers returns the handler definitions for the float64 pointwise add SIMD
// functions across all supported architectures.
//
// Returns []asmgen.HandlerDefinition[VectormathsArchitecturePort] containing SSE, AVX2,
// and NEON variants.
func addF64Handlers() []asmgen.HandlerDefinition[VectormathsArchitecturePort] {
	return []asmgen.HandlerDefinition[VectormathsArchitecturePort]{
		handlerAddF64SSE(),
		handlerAddF64AVX2(),
		handlerAddF64NEON(),
	}
}

// handlerAddF64SSE returns the handler definition for the SSE float64 pointwise add
// function.
//
// Returns asmgen.HandlerDefinition[VectormathsArchitecturePort] targeting amd64 with a
// $0-72 NOSPLIT frame.
func handlerAddF64SSE() asmgen.HandlerDefinition[VectormathsArchitecturePort] {
	return asmgen.HandlerDefinition[VectormathsArchitecturePort]{
		Name:          "addF64SSE",
		Comment:       "addF64SSE computes dst[i] = a[i] + b[i] for f64 vectors using 2-way SSE2 unrolling.",
		Architectures: []asmgen.Architecture{asmgen.ArchitectureAMD64},
		FrameSize:     "$0-72", Flags: "NOSPLIT",
		Emit: func(emitter *asmgen.Emitter, architecture VectormathsArchitecturePort) {
			architecture.EmitAdd(emitter, "SSE")
		},
	}
}

// handlerAddF64AVX2 returns the handler definition for the AVX2 float64 pointwise add
// function.
//
// Returns asmgen.HandlerDefinition[VectormathsArchitecturePort] targeting amd64 with a
// $0-72 NOSPLIT frame.
func handlerAddF64AVX2() asmgen.HandlerDefinition[VectormathsArchitecturePort] {
	return asmgen.HandlerDefinition[VectormathsArchitecturePort]{
		Name:          "addF64AVX2",
		Comment:       "addF64AVX2 computes dst[i] = a[i] + b[i] for f64 vectors using 4-way AVX2 unrolling.",
		Architectures: []asmgen.Architecture{asmgen.ArchitectureAMD64},
		FrameSize:     "$0-72", Flags: "NOSPLIT",
		Emit: func(emitter *asmgen.Emitter, architecture VectormathsArchitecturePort) {
			architecture.EmitAdd(emitter, "AVX2")
		},
	}
}

// handlerAddF64NEON returns the handler definition for the NEON float64 pointwise add
// function.
//
// Returns asmgen.HandlerDefinition[VectormathsArchitecturePort] targeting arm64 with a
// $0-72 NOSPLIT frame.
func handlerAddF64NEON() asmgen.HandlerDefinition[VectormathsArchitecturePort] {
	return asmgen.HandlerDefinition[VectormathsArchitecturePort]{
		Name:          "addF64NEON",
		Comment:       "addF64NEON computes dst[i] = a[i] + b[i] for f64 vectors using 4-way NEON unrolling.",
		Architectures: []asmgen.Architecture{asmgen.ArchitectureARM64},
		FrameSize:     "$0-72", Flags: "NOSPLIT",
		Emit: func(emitter *asmgen.Emitter, architecture VectormathsArchitecturePort) {
			architecture.EmitAdd(emitter, "NEON")
		},
	}
}
