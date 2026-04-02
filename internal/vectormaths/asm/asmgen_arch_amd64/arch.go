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

package asmgen_arch_amd64

import (
	"strings"

	"piko.sh/piko/wdk/asmgen"
	core "piko.sh/piko/wdk/asmgen/asmgen_arch_amd64"
)

// mnemonicColumnWidth is the standard column width for amd64 Plan 9
// assembly mnemonics.
const mnemonicColumnWidth = 8

// VectormathsAMD64Arch extends the core AMD64Arch with SIMD
// vectormaths operations for dot product and euclidean distance.
type VectormathsAMD64Arch struct {
	core.AMD64Arch
}

// New creates a new vectormaths-specific AMD64 architecture adapter.
//
// Returns *VectormathsAMD64Arch ready for use.
func New() *VectormathsAMD64Arch {
	return &VectormathsAMD64Arch{}
}

// inst emits a tab-indented instruction line with the mnemonic padded
// to mnemonicColumnWidth columns (the standard alignment for amd64
// Plan 9 assembly).
//
// Takes e (*asmgen.Emitter) which receives the formatted instruction line.
// Takes mnemonic (string) which is the assembly mnemonic to emit.
// Takes operands (string) which is the operand string to append after padding.
func inst(e *asmgen.Emitter, mnemonic, operands string) {
	padding := max(mnemonicColumnWidth-len(mnemonic), 1)
	e.Instruction(mnemonic + strings.Repeat(" ", padding) + operands)
}

// EmitDotProduct implements VectormathsArchitecturePort.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes variant (string) which selects the SIMD variant ("SSE" or "AVX2").
func (*VectormathsAMD64Arch) EmitDotProduct(e *asmgen.Emitter, variant string) {
	(&amd64VectormathsOps{}).EmitDotProduct(e, variant)
}

// EmitEuclideanDistanceSquared implements VectormathsArchitecturePort.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes variant (string) which selects the SIMD variant ("SSE" or "AVX2").
func (*VectormathsAMD64Arch) EmitEuclideanDistanceSquared(e *asmgen.Emitter, variant string) {
	(&amd64VectormathsOps{}).EmitEuclideanDistanceSquared(e, variant)
}

// EmitNormalise implements VectormathsArchitecturePort.
//
// Takes e (*asmgen.Emitter) which receives the generated assembly instructions.
// Takes variant (string) which selects the SIMD variant ("SSE" or "AVX2").
func (*VectormathsAMD64Arch) EmitNormalise(e *asmgen.Emitter, variant string) {
	(&amd64VectormathsOps{}).EmitNormalise(e, variant)
}
