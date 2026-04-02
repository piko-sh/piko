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

import "piko.sh/piko/wdk/asmgen"

// dotProductHandlers returns the handler definitions for the
// dot product SIMD functions across all supported
// architectures.
//
// Returns []asmgen.HandlerDefinition[VectormathsArchitecturePort]
// containing SSE, AVX2, and NEON variants.
func dotProductHandlers() []asmgen.HandlerDefinition[VectormathsArchitecturePort] {
	return []asmgen.HandlerDefinition[VectormathsArchitecturePort]{
		handlerDotF32SSE(),
		handlerDotF32AVX2(),
		handlerDotF32NEON(),
	}
}

// handlerDotF32SSE returns the handler definition for the SSE dot product function.
//
// Returns asmgen.HandlerDefinition[VectormathsArchitecturePort]
// targeting amd64 with a $0-52 NOSPLIT frame.
func handlerDotF32SSE() asmgen.HandlerDefinition[VectormathsArchitecturePort] {
	return asmgen.HandlerDefinition[VectormathsArchitecturePort]{
		Name:          "dotF32SSE",
		Comment:       "func dotF32SSE(a, b []float32) float32",
		Architectures: []asmgen.Architecture{asmgen.ArchitectureAMD64},
		FrameSize:     "$0-52", Flags: "NOSPLIT",
		Emit: func(emitter *asmgen.Emitter, architecture VectormathsArchitecturePort) {
			architecture.EmitDotProduct(emitter, "SSE")
		},
	}
}

// handlerDotF32AVX2 returns the handler definition for the AVX2 dot product function.
//
// Returns asmgen.HandlerDefinition[VectormathsArchitecturePort]
// targeting amd64 with a $0-52 NOSPLIT frame.
func handlerDotF32AVX2() asmgen.HandlerDefinition[VectormathsArchitecturePort] {
	return asmgen.HandlerDefinition[VectormathsArchitecturePort]{
		Name:          "dotF32AVX2",
		Comment:       "func dotF32AVX2(a, b []float32) float32",
		Architectures: []asmgen.Architecture{asmgen.ArchitectureAMD64},
		FrameSize:     "$0-52", Flags: "NOSPLIT",
		Emit: func(emitter *asmgen.Emitter, architecture VectormathsArchitecturePort) {
			architecture.EmitDotProduct(emitter, "AVX2")
		},
	}
}

// handlerDotF32NEON returns the handler definition for the NEON dot product function.
//
// Returns asmgen.HandlerDefinition[VectormathsArchitecturePort]
// targeting arm64 with a $0-52 NOSPLIT frame.
func handlerDotF32NEON() asmgen.HandlerDefinition[VectormathsArchitecturePort] {
	return asmgen.HandlerDefinition[VectormathsArchitecturePort]{
		Name:          "dotF32",
		Comment:       "func dotF32(a, b []float32) float32",
		Architectures: []asmgen.Architecture{asmgen.ArchitectureARM64},
		FrameSize:     "$0-52", Flags: "NOSPLIT",
		Emit: func(emitter *asmgen.Emitter, architecture VectormathsArchitecturePort) {
			architecture.EmitDotProduct(emitter, "NEON")
		},
	}
}
