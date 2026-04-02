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

package asmgen_arch_arm64

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/wdk/asmgen"
)

func emitToString(fn func(arch *VectormathsARM64Arch, emitter *asmgen.Emitter)) string {
	emitter := asmgen.NewEmitter()
	arch := New()
	fn(arch, emitter)
	return emitter.String()
}

func TestEmitDotProductNEON(t *testing.T) {
	output := emitToString(func(arch *VectormathsARM64Arch, emitter *asmgen.Emitter) {
		arch.EmitDotProduct(emitter, "NEON")
	})

	assert.Contains(t, output, "VLD1")
	assert.Contains(t, output, "VFMLA")
	assert.Contains(t, output, "WORD $0x6E20D400", "expected FADDP word encoding")
	assert.Contains(t, output, "FMOVS")
	assert.Contains(t, output, "RET")
}

func TestEmitEuclideanDistanceSquaredNEON(t *testing.T) {
	output := emitToString(func(arch *VectormathsARM64Arch, emitter *asmgen.Emitter) {
		arch.EmitEuclideanDistanceSquared(emitter, "NEON")
	})

	assert.Contains(t, output, "VLD1")
	assert.Contains(t, output, "WORD $0x4EA2D421", "expected FSUB word encoding")
	assert.Contains(t, output, "VFMLA")
	assert.Contains(t, output, "RET")
}

func TestEmitNormaliseNEON(t *testing.T) {
	output := emitToString(func(arch *VectormathsARM64Arch, emitter *asmgen.Emitter) {
		arch.EmitNormalise(emitter, "NEON")
	})

	assert.Contains(t, output, "FSQRTS")
	assert.Contains(t, output, "FMOVS $1.0")
	assert.Contains(t, output, "WORD $0x4E040421", "expected DUP word encoding")
	assert.Contains(t, output, "WORD $0x6E21DC42", "expected FMUL word encoding")
	assert.Contains(t, output, "RET")
}

func TestEmitDotProduct_UnknownVariant(t *testing.T) {
	output := emitToString(func(arch *VectormathsARM64Arch, emitter *asmgen.Emitter) {
		arch.EmitDotProduct(emitter, "SSE")
	})

	assert.Empty(t, output, "expected empty output for unsupported variant")
}
