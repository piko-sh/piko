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

const (
	// goSymbolMathSin is the Plan-9 ASM symbol of the Go trampoline forwarding to math.Sin.
	goSymbolMathSin = "·asmCallMathSin(SB)"

	// goSymbolMathCos is the Plan-9 ASM symbol of the Go trampoline forwarding to math.Cos.
	goSymbolMathCos = "·asmCallMathCos(SB)"

	// goSymbolMathExp is the Plan-9 ASM symbol of the Go trampoline forwarding to math.Exp.
	goSymbolMathExp = "·asmCallMathExp(SB)"

	// goSymbolMathTan is the Plan-9 ASM symbol of the Go trampoline forwarding to math.Tan.
	goSymbolMathTan = "·asmCallMathTan(SB)"

	// goSymbolMathMod is the Plan-9 ASM symbol of the Go trampoline forwarding to math.Mod
	// (3-operand).
	goSymbolMathMod = "·asmCallMathMod(SB)"
)

// tier1MathHandlers returns the handler definitions for the tier-1 umbrella sub-ops that
// delegate to Go math intrinsics. Each math op produces ONE single-level NOSPLIT|NOFRAME
// handler that uses ADJSP to open a scratch frame around the abi0 CALL.
//
// Returns []asmgen.HandlerDefinition[BytecodeArchitecturePort] of 5 entries.
func tier1MathHandlers() []asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return []asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		handlerSubOpMathSin(),
		handlerSubOpMathCos(),
		handlerSubOpMathExp(),
		handlerSubOpMathTan(),
		handlerSubOpMathMod(),
	}
}

// handlerSubOpMathSin builds the single-level handler for subOpMathSin. Installed in
// tier1JumpTable: ADJSP $32, marshal abi0 args, CALL asmCallMathSin, reload regs,
// DISPATCH_NEXT.
//
// Returns the handler definition for subOpMathSin.
func handlerSubOpMathSin() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return inlineGoTwoOperandShim(
		"handlerSubOpMathSin",
		"handlerSubOpMathSin sets floats[B] = math.Sin(floats[C]) via asmCallMathSin.",
		goSymbolMathSin,
	)
}

// handlerSubOpMathCos builds the single-level handler for subOpMathCos.
//
// Returns the handler definition for subOpMathCos.
func handlerSubOpMathCos() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return inlineGoTwoOperandShim(
		"handlerSubOpMathCos",
		"handlerSubOpMathCos sets floats[B] = math.Cos(floats[C]) via asmCallMathCos.",
		goSymbolMathCos,
	)
}

// handlerSubOpMathExp builds the single-level handler for subOpMathExp.
//
// Returns the handler definition for subOpMathExp.
func handlerSubOpMathExp() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return inlineGoTwoOperandShim(
		"handlerSubOpMathExp",
		"handlerSubOpMathExp sets floats[B] = math.Exp(floats[C]) via asmCallMathExp.",
		goSymbolMathExp,
	)
}

// handlerSubOpMathTan builds the single-level handler for subOpMathTan.
//
// Returns the handler definition for subOpMathTan.
func handlerSubOpMathTan() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return inlineGoTwoOperandShim(
		"handlerSubOpMathTan",
		"handlerSubOpMathTan sets floats[B] = math.Tan(floats[C]) via asmCallMathTan.",
		goSymbolMathTan,
	)
}

// handlerSubOpMathMod builds the single-level handler for subOpMathMod (3-operand: B =
// math.Mod(C, ext.A)). Uses ADJSP $40 to fit the 4-arg abi0 frame.
//
// Returns the handler definition for subOpMathMod.
func handlerSubOpMathMod() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return inlineGoThreeOperandShim(
		"handlerSubOpMathMod",
		"handlerSubOpMathMod sets floats[B] = math.Mod(floats[C], floats[ext.A]) via asmCallMathMod.",
		goSymbolMathMod,
	)
}

// inlineGoTwoOperandShim builds the single-level NOSPLIT|NOFRAME, $0 handler definition
// for a tier-1 sub-op of the shape "B = goFn(C)".
//
// One TEXT owns the trampoline CALL via ADJSP $32 / ADJSP $-32 (the canonical Go-runtime
// scratch-frame pattern), avoiding the extra CALL+RET pair and the Plan-9 auto-injected
// PUSHQ/POPQ BP that a separate wrapper would add.
//
// Takes name (string) which is the handler TEXT label.
// Takes comment (string) which is the leading comment for the emitted TEXT block.
// Takes goSymbol (string) which is the Plan-9 symbol of the Go trampoline (e.g.
// asmCallMathSin(SB)).
//
// Returns the handler definition wired to emit the inline shim.
func inlineGoTwoOperandShim(name, comment, goSymbol string) asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      name,
		Comment:   comment,
		FrameSize: frameSizeZero,
		Flags:     flagsNoSplitNoFrame,
		//nolint:exhaustive // sparse arm64-only override map; missing keys fall back to default Flags
		ArchFlags: map[asmgen.Architecture]string{
			asmgen.ArchitectureARM64: flagNoSplit,
		},
		//nolint:exhaustive // sparse arm64-only override map; missing keys fall back to default FrameSize
		ArchFrameSize: map[asmgen.Architecture]string{
			asmgen.ArchitectureARM64: frameSizeShim3ArgARM64,
		},
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.EmitInlineGoCallTwoOperandShim(emitter, goSymbol)
		},
	}
}

// inlineGoThreeOperandShim builds the single-level NOSPLIT|NOFRAME, $0 handler definition
// for a 3-operand sub-op (B, C, and a third operand read from the next instruction word's
// A field). Uses ADJSP $40 / $-40 to fit the 4-arg abi0 frame.
//
// Takes name (string) which is the handler TEXT label.
// Takes comment (string) which is the leading comment for the emitted TEXT block.
// Takes goSymbol (string) which is the Plan-9 symbol of the Go trampoline.
//
// Returns the handler definition wired to emit the inline shim.
func inlineGoThreeOperandShim(name, comment, goSymbol string) asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      name,
		Comment:   comment,
		FrameSize: frameSizeZero,
		Flags:     flagsNoSplitNoFrame,
		//nolint:exhaustive // sparse arm64-only override map; missing keys fall back to default Flags
		ArchFlags: map[asmgen.Architecture]string{
			asmgen.ArchitectureARM64: flagNoSplit,
		},
		//nolint:exhaustive // sparse arm64-only override map; missing keys fall back to default FrameSize
		ArchFrameSize: map[asmgen.Architecture]string{
			asmgen.ArchitectureARM64: frameSizeShim4ArgARM64,
		},
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.EmitInlineGoCallThreeOperandShim(emitter, goSymbol)
		},
	}
}
