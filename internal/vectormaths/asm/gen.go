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

// vectormathsBuildConstraint is the build constraint applied
// to generated vectormaths assembly files.
const vectormathsBuildConstraint = "!safe && !(js && wasm)"

// vectormathsOutputDir is the output directory for generated vectormaths assembly files.
const vectormathsOutputDir = "internal/vectormaths"

// vectormathsIncludes lists the assembly include files required by generated SIMD functions.
var vectormathsIncludes = []string{"textflag.h"}

// FileGroups returns all FileGroup definitions for the vectormaths
// SIMD functions.
//
// Returns []asmgen.FileGroup[VectormathsArchitecturePort]
// containing the dot product, Euclidean distance, and
// normalise file groups.
func FileGroups() []asmgen.FileGroup[VectormathsArchitecturePort] {
	return []asmgen.FileGroup[VectormathsArchitecturePort]{
		{
			BaseName:        "asm_dot_f32",
			OutputDir:       vectormathsOutputDir,
			BuildConstraint: vectormathsBuildConstraint,
			Includes:        vectormathsIncludes,
			Handlers:        dotProductHandlers(),
		},
		{
			BaseName:        "asm_euclid_sq_f32",
			OutputDir:       vectormathsOutputDir,
			BuildConstraint: vectormathsBuildConstraint,
			Includes:        vectormathsIncludes,
			Handlers:        euclideanDistanceHandlers(),
		},
		{
			BaseName:        "asm_normalise_f32",
			OutputDir:       vectormathsOutputDir,
			BuildConstraint: vectormathsBuildConstraint,
			Includes:        vectormathsIncludes,
			Handlers:        normaliseHandlers(),
		},
	}
}
