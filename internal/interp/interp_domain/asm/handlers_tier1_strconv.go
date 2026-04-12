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
)

const (
	// goSymbolStrconvItoa is the Plan-9 ASM symbol of the Go trampoline forwarding to
	// strconv.Itoa. See asm_call_trampolines.go for the trampoline implementation.
	goSymbolStrconvItoa = "·asmCallStrconvItoa(SB)"

	// goSymbolStrconvFormatInt is the Plan-9 ASM symbol of the Go trampoline forwarding to
	// strconv.FormatInt.
	goSymbolStrconvFormatInt = "·asmCallStrconvFormatInt(SB)"
)

// tier1StrconvHandlers returns the tier-1 strconv handler definitions.
//
// Itoa and FormatInt are single-level NOSPLIT|NOFRAME shims with ADJSP-managed scratch
// frames. FormatBool is inlined as a single pure-ASM handler that picks between two
// pre-allocated string headers (boolStringTrue / boolStringFalse): no Go call, no
// trampoline frame, no spill dance.
//
// Returns []asmgen.HandlerDefinition[BytecodeArchitecturePort] with 3 entries (1 inlined
// FormatBool + 2 single-level shims).
func tier1StrconvHandlers() []asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return []asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		handlerSubOpStrconvFormatBool(),
		handlerSubOpStrconvItoa(),
		handlerSubOpStrconvFormatInt(),
	}
}

// handlerSubOpStrconvFormatBool builds the single inlined handler for
// subOpStrconvFormatBool. The body reads bools[C], CMOV-selects between two static string
// headers (boolStringTrue and boolStringFalse declared in interp_domain), and stamps the
// chosen header into strings[B]: no Go call required since strconv.FormatBool's return
// values are statically interned.
//
// Returns the handler definition for subOpStrconvFormatBool.
func handlerSubOpStrconvFormatBool() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      "handlerSubOpStrconvFormatBool",
		Comment:   "handlerSubOpStrconvFormatBool sets strings[B] = bools[C] ? \"true\" : \"false\" by stamping a pre-allocated header.",
		FrameSize: frameSizeZero,
		Flags:     flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.EmitSubOpStrconvFormatBool(emitter)
		},
	}
}

// handlerSubOpStrconvItoa builds the single-level handler for subOpStrconvItoa:
// strings[B] = strconv.Itoa(int(ints[C])).
//
// Returns the handler definition for subOpStrconvItoa.
func handlerSubOpStrconvItoa() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return inlineGoTwoOperandShim(
		"handlerSubOpStrconvItoa",
		"handlerSubOpStrconvItoa sets strings[B] = strconv.Itoa(int(ints[C])) via asmCallStrconvItoa.",
		goSymbolStrconvItoa,
	)
}

// handlerSubOpStrconvFormatInt builds the single-level handler for subOpStrconvFormatInt:
// strings[B] = strconv.FormatInt(ints[C], int(ints[ext.A])). 3-operand: second source
// from the next instruction word's A field.
//
// Returns the handler definition for subOpStrconvFormatInt.
func handlerSubOpStrconvFormatInt() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return inlineGoThreeOperandShim(
		"handlerSubOpStrconvFormatInt",
		"handlerSubOpStrconvFormatInt sets strings[B] = strconv.FormatInt(ints[C], int(ints[ext.A])) via asmCallStrconvFormatInt.",
		goSymbolStrconvFormatInt,
	)
}
