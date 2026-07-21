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
	"piko.sh/asmgen"
)

// tier1MathUnaryHandlers returns the tier-1 ASM handlers for pure-FPU math intrinsics.
//
// Each handler maps directly to a single FPU instruction with no Go-side trampoline
// needed. It reads its source float register index from operand byte 3 (ExtractC),
// applies the architecture port's FloatUnaryOperation primitive with the appropriate
// operation tag, and writes the result to operand byte 2 (ExtractB).
//
// Coverage: math.Sqrt, math.Abs, math.Floor, math.Ceil, math.Trunc, math.Round. All six
// map to direct FPU instructions on both arm64
// (FSQRTD/FABSD/FRINTMD/FRINTPD/FRINTZD/FRINTAD) and amd64 (SQRTSD/BTRQ/ROUNDSD
// variants). Round on amd64 uses a 5-instruction `trunc(x + copysign(0.5, x))` recipe
// because math.Round rounds half away from zero, which neither ROUNDSD mode supports
// directly.
//
// Sin/Cos/Exp/Tan/Mod remain in handlers_tier1_math.go because they require ABI
// marshalling to Go's math package; this file covers only the pure-FPU cases.
//
// Returns []asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the set of
// math-unary tier-1 sub-op handlers.
func tier1MathUnaryHandlers() []asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return []asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		tier1FloatUnaryHandler("handlerSubOpMathSqrt", "SQRT",
			"handlerSubOpMathSqrt sets floats[B] = math.Sqrt(floats[C])."),
		tier1FloatUnaryHandler("handlerSubOpMathAbs", "ABS",
			"handlerSubOpMathAbs sets floats[B] = math.Abs(floats[C])."),
		tier1FloatUnaryHandler("handlerSubOpMathFloor", "FLOOR",
			"handlerSubOpMathFloor sets floats[B] = math.Floor(floats[C])."),
		tier1FloatUnaryHandler("handlerSubOpMathCeil", "CEIL",
			"handlerSubOpMathCeil sets floats[B] = math.Ceil(floats[C])."),
		tier1FloatUnaryHandler("handlerSubOpMathTrunc", "TRUNC",
			"handlerSubOpMathTrunc sets floats[B] = math.Trunc(floats[C])."),
		tier1FloatUnaryHandler("handlerSubOpMathRound", "ROUND",
			"handlerSubOpMathRound sets floats[B] = math.Round(floats[C])."),
	}
}
