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

// normaliseHandlers returns the handler definitions for the
// SIMD vector normalisation functions.
//
// Returns []asmgen.HandlerDefinition[VectormathsArchitecturePort]
// containing SSE, AVX2, and NEON variants.
func normaliseHandlers() []asmgen.HandlerDefinition[VectormathsArchitecturePort] {
	return []asmgen.HandlerDefinition[VectormathsArchitecturePort]{
		handlerNormaliseF32SSE(),
		handlerNormaliseF32AVX2(),
		handlerNormaliseF32NEON(),
	}
}

// handlerNormaliseF32SSE returns the handler definition for
// the SSE in-place normalisation function.
//
// Returns asmgen.HandlerDefinition[VectormathsArchitecturePort]
// targeting amd64 with a $0-24 NOSPLIT frame.
func handlerNormaliseF32SSE() asmgen.HandlerDefinition[VectormathsArchitecturePort] {
	return asmgen.HandlerDefinition[VectormathsArchitecturePort]{
		Name:          "normaliseF32SSE",
		Comment:       "func normaliseF32SSE(v []float32)",
		Architectures: []asmgen.Architecture{asmgen.ArchitectureAMD64},
		FrameSize:     "$0-24", Flags: "NOSPLIT",
		Emit: func(emitter *asmgen.Emitter, architecture VectormathsArchitecturePort) {
			architecture.EmitNormalise(emitter, "SSE")
		},
	}
}

// handlerNormaliseF32AVX2 returns the handler definition for
// the AVX2 in-place normalisation function.
//
// Returns asmgen.HandlerDefinition[VectormathsArchitecturePort]
// targeting amd64 with a $0-24 NOSPLIT frame.
func handlerNormaliseF32AVX2() asmgen.HandlerDefinition[VectormathsArchitecturePort] {
	return asmgen.HandlerDefinition[VectormathsArchitecturePort]{
		Name:          "normaliseF32AVX2",
		Comment:       "func normaliseF32AVX2(v []float32)",
		Architectures: []asmgen.Architecture{asmgen.ArchitectureAMD64},
		FrameSize:     "$0-24", Flags: "NOSPLIT",
		Emit: func(emitter *asmgen.Emitter, architecture VectormathsArchitecturePort) {
			architecture.EmitNormalise(emitter, "AVX2")
		},
	}
}

// handlerNormaliseF32NEON returns the handler definition for
// the NEON in-place normalisation function.
//
// Returns asmgen.HandlerDefinition[VectormathsArchitecturePort]
// targeting arm64 with a $0-24 NOSPLIT frame.
func handlerNormaliseF32NEON() asmgen.HandlerDefinition[VectormathsArchitecturePort] {
	return asmgen.HandlerDefinition[VectormathsArchitecturePort]{
		Name:          "normaliseF32",
		Comment:       "func normaliseF32(v []float32)",
		Architectures: []asmgen.Architecture{asmgen.ArchitectureARM64},
		FrameSize:     "$0-24", Flags: "NOSPLIT",
		Emit: func(emitter *asmgen.Emitter, architecture VectormathsArchitecturePort) {
			architecture.EmitNormalise(emitter, "NEON")
		},
	}
}
