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

// euclideanDistanceHandlers returns the handler definitions
// for the squared Euclidean distance SIMD functions across
// all supported architectures.
//
// Returns []asmgen.HandlerDefinition[VectormathsArchitecturePort]
// containing SSE, AVX2, and NEON variants.
func euclideanDistanceHandlers() []asmgen.HandlerDefinition[VectormathsArchitecturePort] {
	return []asmgen.HandlerDefinition[VectormathsArchitecturePort]{
		handlerEuclidSqF32SSE(),
		handlerEuclidSqF32AVX2(),
		handlerEuclidSqF32NEON(),
	}
}

// handlerEuclidSqF32SSE returns the handler definition for
// the SSE squared Euclidean distance function.
//
// Returns asmgen.HandlerDefinition[VectormathsArchitecturePort]
// targeting amd64 with a $0-52 NOSPLIT frame.
func handlerEuclidSqF32SSE() asmgen.HandlerDefinition[VectormathsArchitecturePort] {
	return asmgen.HandlerDefinition[VectormathsArchitecturePort]{
		Name:          "euclidSqF32SSE",
		Comment:       "func euclidSqF32SSE(a, b []float32) float32",
		Architectures: []asmgen.Architecture{asmgen.ArchitectureAMD64},
		FrameSize:     "$0-52", Flags: "NOSPLIT",
		Emit: func(emitter *asmgen.Emitter, architecture VectormathsArchitecturePort) {
			architecture.EmitEuclideanDistanceSquared(emitter, "SSE")
		},
	}
}

// handlerEuclidSqF32AVX2 returns the handler definition for
// the AVX2 squared Euclidean distance function.
//
// Returns asmgen.HandlerDefinition[VectormathsArchitecturePort]
// targeting amd64 with a $0-52 NOSPLIT frame.
func handlerEuclidSqF32AVX2() asmgen.HandlerDefinition[VectormathsArchitecturePort] {
	return asmgen.HandlerDefinition[VectormathsArchitecturePort]{
		Name:          "euclidSqF32AVX2",
		Comment:       "func euclidSqF32AVX2(a, b []float32) float32",
		Architectures: []asmgen.Architecture{asmgen.ArchitectureAMD64},
		FrameSize:     "$0-52", Flags: "NOSPLIT",
		Emit: func(emitter *asmgen.Emitter, architecture VectormathsArchitecturePort) {
			architecture.EmitEuclideanDistanceSquared(emitter, "AVX2")
		},
	}
}

// handlerEuclidSqF32NEON returns the handler definition for
// the NEON squared Euclidean distance function.
//
// Returns asmgen.HandlerDefinition[VectormathsArchitecturePort]
// targeting arm64 with a $0-52 NOSPLIT frame.
func handlerEuclidSqF32NEON() asmgen.HandlerDefinition[VectormathsArchitecturePort] {
	return asmgen.HandlerDefinition[VectormathsArchitecturePort]{
		Name:          "euclidSqF32Kern",
		Comment:       "func euclidSqF32Kern(a, b []float32) float32",
		Architectures: []asmgen.Architecture{asmgen.ArchitectureARM64},
		FrameSize:     "$0-52", Flags: "NOSPLIT",
		Emit: func(emitter *asmgen.Emitter, architecture VectormathsArchitecturePort) {
			architecture.EmitEuclideanDistanceSquared(emitter, "NEON")
		},
	}
}
