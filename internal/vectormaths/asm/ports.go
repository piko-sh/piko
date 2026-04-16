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

	vectormaths_amd64 "piko.sh/piko/internal/vectormaths/asm/asmgen_arch_amd64"
	vectormaths_arm64 "piko.sh/piko/internal/vectormaths/asm/asmgen_arch_arm64"
)

var _ VectormathsArchitecturePort = (*vectormaths_amd64.VectormathsAMD64Arch)(nil)
var _ VectormathsArchitecturePort = (*vectormaths_arm64.VectormathsARM64Arch)(nil)

// VectormathsArchitecturePort extends the core ArchitecturePort with
// SIMD vectormaths operations for dot product and euclidean distance.
type VectormathsArchitecturePort interface {
	asmgen.ArchitecturePort

	// EmitDotProduct emits the dot product assembly function body for the given SIMD variant.
	EmitDotProduct(emitter *asmgen.Emitter, variant string)

	// EmitEuclideanDistanceSquared emits the squared Euclidean
	// distance assembly function body for the given SIMD
	// variant.
	EmitEuclideanDistanceSquared(emitter *asmgen.Emitter, variant string)

	// EmitNormalise emits the in-place normalisation assembly
	// function body for the given SIMD variant.
	EmitNormalise(emitter *asmgen.Emitter, variant string)
}
