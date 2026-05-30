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
	"piko.sh/piko/wdk/asmgen"
	"piko.sh/piko/wdk/asmgen/asmarm64"
)

// emitARM64RestoreTypedSliceBanks restores typed-slice bank bases.
//
// Restores the slicesInt/Float/String/Bool/Uint and complex bank bases from the caller's
// callFrame. The inline value/void return paths stay in ASM and never reach the Go-side
// repairRegisterBasesFromCallers, so they must restore these themselves - otherwise a
// callee whose mask includes a typed-slice bank (e.g. []string) shifts
// ctx.slicesStringBase to its own frame, and after the return the caller would keep
// reading the popped callee's bank, so these bases must be restored here before the
// return resumes. Mirrors the amd64 path.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func emitARM64RestoreTypedSliceBanks(e *asmgen.Emitter) {
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_SLICESINT_PTR(R22), R1")
	inst5(e, asmarm64.OperationMove64Bits, "R1, CTX_SLICES_INT_BASE(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_SLICESFLOAT_PTR(R22), R1")
	inst5(e, asmarm64.OperationMove64Bits, "R1, CTX_SLICES_FLOAT_BASE(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_SLICESSTRING_PTR(R22), R1")
	inst5(e, asmarm64.OperationMove64Bits, "R1, CTX_SLICES_STRING_BASE(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_SLICESBOOL_PTR(R22), R1")
	inst5(e, asmarm64.OperationMove64Bits, "R1, CTX_SLICES_BOOL_BASE(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_SLICESUINT_PTR(R22), R1")
	inst5(e, asmarm64.OperationMove64Bits, "R1, CTX_SLICES_UINT_BASE(R19)")
	inst5(e, asmarm64.OperationMove64Bits, "CF_REGS_COMPLEX_PTR(R22), R1")
	inst5(e, asmarm64.OperationMove64Bits, "R1, CTX_COMPLEX_BASE(R19)")
}
