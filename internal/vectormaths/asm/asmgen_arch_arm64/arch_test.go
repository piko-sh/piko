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

func TestEmitDotF64NEON(t *testing.T) {
	output := emitToString(func(arch *VectormathsARM64Arch, emitter *asmgen.Emitter) {
		arch.EmitDotF64(emitter, "NEON")
	})

	assert.Contains(t, output, "VLD1")
	assert.Contains(t, output, "WORD $0x4E62CC20", "expected FMLA V0.2D, V1.2D, V2.2D encoding")
	assert.Contains(t, output, "WORD $0x4E62CC24", "expected FMLA V4.2D, V1.2D, V2.2D encoding")
	assert.Contains(t, output, "WORD $0x6E60D400", "expected FADDP.2D word encoding")
	assert.Contains(t, output, "FMOVD")
	assert.Contains(t, output, "RET")
}

func TestEmitScaleF64NEON(t *testing.T) {
	output := emitToString(func(arch *VectormathsARM64Arch, emitter *asmgen.Emitter) {
		arch.EmitScaleF64(emitter, "NEON")
	})

	assert.Contains(t, output, "VLD1")
	assert.Contains(t, output, "VST1")
	assert.Contains(t, output, "WORD $0x4E0804E7", "expected DUP V7.2D, V7.D[0] encoding")
	assert.Contains(t, output, "WORD $0x6E67DC00", "expected FMUL.2D word encoding for V0, V0, V7")
	assert.Contains(t, output, "RET")
}

func TestEmitDotF64_UnknownVariant(t *testing.T) {
	output := emitToString(func(arch *VectormathsARM64Arch, emitter *asmgen.Emitter) {
		arch.EmitDotF64(emitter, "SSE")
	})

	assert.Empty(t, output, "expected empty output for unsupported variant")
}

func TestEmitScaleF64_UnknownVariant(t *testing.T) {
	output := emitToString(func(arch *VectormathsARM64Arch, emitter *asmgen.Emitter) {
		arch.EmitScaleF64(emitter, "SSE")
	})

	assert.Empty(t, output, "expected empty output for unsupported variant")
}

func TestNEONFMLA2DEncoding(t *testing.T) {
	assert.Equal(t, uint32(0x4E60CC00), neonFMLA2D(0, 0, 0), "FMLA V0.2D,V0.2D,V0.2D")
	assert.Equal(t, uint32(0x4E62CC20), neonFMLA2D(0, 1, 2), "FMLA V0.2D,V1.2D,V2.2D")
	assert.Equal(t, uint32(0x4E62CC24), neonFMLA2D(4, 1, 2), "FMLA V4.2D,V1.2D,V2.2D")
}

func TestNEONFMUL2DEncoding(t *testing.T) {
	assert.Equal(t, uint32(0x6E60DC00), neonFMUL2D(0, 0, 0), "FMUL V0.2D,V0.2D,V0.2D")
	assert.Equal(t, uint32(0x6E67DC00), neonFMUL2D(0, 0, 7), "FMUL V0.2D,V0.2D,V7.2D")
}

func TestNEONDUP2DFromDoubleEncoding(t *testing.T) {
	assert.Equal(t, uint32(0x4E080400), neonDUP2DFromDouble(0, 0), "DUP V0.2D, V0.D[0]")
	assert.Equal(t, uint32(0x4E0804E7), neonDUP2DFromDouble(7, 7), "DUP V7.2D, V7.D[0]")
}
