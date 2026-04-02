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
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/wdk/asmgen"
)

func emitToString(fn func(arch *VectormathsAMD64Arch, emitter *asmgen.Emitter)) string {
	emitter := asmgen.NewEmitter()
	arch := New()
	fn(arch, emitter)
	return emitter.String()
}

func TestEmitDotProductSSE(t *testing.T) {
	output := emitToString(func(arch *VectormathsAMD64Arch, emitter *asmgen.Emitter) {
		arch.EmitDotProduct(emitter, "SSE")
	})

	assert.Contains(t, output, "XORPS")
	assert.Contains(t, output, "MOVUPS")
	assert.Contains(t, output, "MULPS")
	assert.Contains(t, output, "ADDPS")
	assert.Contains(t, output, "MOVHLPS")
	assert.Contains(t, output, "SHUFPS")
	assert.Contains(t, output, "RET")
}

func TestEmitDotProductAVX2(t *testing.T) {
	output := emitToString(func(arch *VectormathsAMD64Arch, emitter *asmgen.Emitter) {
		arch.EmitDotProduct(emitter, "AVX2")
	})

	assert.Contains(t, output, "VMOVUPS")
	assert.Contains(t, output, "VMULPS")
	assert.Contains(t, output, "VADDPS")
	assert.Contains(t, output, "VEXTRACTF128")
	assert.Contains(t, output, "VZEROUPPER")
	assert.Contains(t, output, "RET")
}

func TestEmitEuclideanDistanceSquaredSSE(t *testing.T) {
	output := emitToString(func(arch *VectormathsAMD64Arch, emitter *asmgen.Emitter) {
		arch.EmitEuclideanDistanceSquared(emitter, "SSE")
	})

	assert.Contains(t, output, "SUBPS")
	assert.Contains(t, output, "MULPS")
	assert.Contains(t, output, "RET")
}

func TestEmitEuclideanDistanceSquaredAVX2(t *testing.T) {
	output := emitToString(func(arch *VectormathsAMD64Arch, emitter *asmgen.Emitter) {
		arch.EmitEuclideanDistanceSquared(emitter, "AVX2")
	})

	assert.Contains(t, output, "VSUBPS")
	assert.Contains(t, output, "VMULPS")
	assert.Contains(t, output, "VZEROUPPER")
	assert.Contains(t, output, "RET")
}

func TestEmitNormaliseSSE(t *testing.T) {
	output := emitToString(func(arch *VectormathsAMD64Arch, emitter *asmgen.Emitter) {
		arch.EmitNormalise(emitter, "SSE")
	})

	assert.Contains(t, output, "SQRTSS")
	assert.Contains(t, output, "DIVSS")
	assert.Contains(t, output, "SHUFPS")
	assert.Contains(t, output, "MULPS")
	assert.Contains(t, output, "RET")
}

func TestEmitNormaliseAVX2(t *testing.T) {
	output := emitToString(func(arch *VectormathsAMD64Arch, emitter *asmgen.Emitter) {
		arch.EmitNormalise(emitter, "AVX2")
	})

	assert.Contains(t, output, "VSQRTSS")
	assert.Contains(t, output, "VDIVSS")
	assert.Contains(t, output, "VBROADCASTSS")
	assert.Contains(t, output, "VZEROUPPER")
	assert.Contains(t, output, "RET")
}

func TestEmitDotProduct_UnknownVariant(t *testing.T) {
	output := emitToString(func(arch *VectormathsAMD64Arch, emitter *asmgen.Emitter) {
		arch.EmitDotProduct(emitter, "NEON")
	})

	assert.Empty(t, output, "expected empty output for unsupported variant")
}
