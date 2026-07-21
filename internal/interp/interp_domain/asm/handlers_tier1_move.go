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

// contextBaseBank identifies which context-loaded bank a context-based move handler
// should access. The architecture port's LoadFromBoolBank / LoadFromUintBank primitives
// accept the bank implicitly via their named entry points; this enum lets one helper
// dispatch to the right pair.
type contextBaseBank int

const (
	// contextBaseBool selects the boolean context bank.
	contextBaseBool contextBaseBank = iota

	// contextBaseUint selects the unsigned-integer context bank.
	contextBaseUint
)

// tier1MoveHandlers returns the tier-1 register-move ASM handlers.
//
// Operand byte 2 (ExtractB) holds the destination index and byte 3 (ExtractC) the source
// index. Installing these in tier1JumpTable keeps each move inside ASM dispatch instead
// of taking an ASM<->Go round-trip via the tier2Fallback path.
//
// Coverage spans subOpMoveInt (integer bank via the pinned R8 base and generic
// LoadFromBank/StoreToBank primitives), subOpMoveFloat (float bank via the pinned R9 base
// and a float scratch), and subOpMoveBool/subOpMoveUint (bank base at CTX_BOOLS_BASE/
// CTX_UINTS_BASE, loaded lazily into a scratch by LoadFromBoolBank/ LoadFromUintBank).
// subOpMoveString is handled separately via the StringCopy arch primitive which transfers
// the 16-byte Go string header in one handler body. subOpMoveGeneral stays on the Go tier
// because the general bank holds 24-byte reflect.Value triplets that do not fit the
// single-load/single-store template here.
//
// Returns the set of move-family tier-1 sub-op handlers.
func tier1MoveHandlers() []asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return []asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		moveHandler("handlerSubOpMoveInt", asmgen.RegisterBankInteger),
		moveHandlerFloat("handlerSubOpMoveFloat"),
		moveHandlerContextBased("handlerSubOpMoveBool", contextBaseBool),
		moveHandlerContextBased("handlerSubOpMoveUint", contextBaseUint),
		moveHandlerString("handlerSubOpMoveString"),
	}
}

// moveHandlerString builds the string-bank tier-1 register-move handler.
//
// Unlike the integer/float banks (which have pinned base registers) and the bool/uint
// banks (which the arch port loads lazily via dedicated helpers), the string bank is
// accessed via the architecture port's StringCopy primitive which transfers the full
// 16-byte Go string header (data pointer + length) in one handler body.
//
// Takes name (string) which is the generated symbol name.
//
// Returns the assembled string-move handler definition.
func moveHandlerString(name string) asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      name,
		Comment:   name + " copies strings[C] to strings[B] for the tier-1 register-move sub-op.",
		FrameSize: frameSizeZero,
		Flags:     flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractB(emitter, scratches[0])
			architecture.ExtractC(emitter, scratches[1])
			architecture.StringCopy(emitter, scratches[0], scratches[1])
			architecture.DispatchNext(emitter)
		},
	}
}

// moveHandler builds a tier-1 register-to-register move handler.
//
// Reads the destination index from operand B and the source index from operand C, loads
// the value from the source bank slot, and stores it to the destination bank slot.
// Mirrors the LoadIntConstSmall handler in operand layout: operand A is the sub-op
// discriminator and the two register indices sit at B and C.
//
// Takes name (string) which is the asmgen-generated symbol name.
// Takes bank (asmgen.RegisterBank) which selects the source and destination register bank
// for the load/store pair.
//
// Returns the handler definition ready to register in a FileGroup.
func moveHandler(name string, bank asmgen.RegisterBank) asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      name,
		Comment:   name + " copies the value at bank[C] into bank[B] for the tier-1 register-move sub-op.",
		FrameSize: frameSizeZero,
		Flags:     flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractB(emitter, scratches[0])
			architecture.ExtractC(emitter, scratches[1])
			temp := architecture.DataTemporary(dataTempScratch0)
			architecture.LoadFromBank(emitter, bank, scratches[1], temp)
			architecture.StoreToBank(emitter, bank, temp, scratches[0])
			architecture.DispatchNext(emitter)
		},
	}
}

// moveHandlerFloat builds the float-bank tier-1 register-move handler.
//
// The float bank has a pinned base register (R9 on amd64, R24 on arm64), so we can reuse
// LoadFromBank/StoreToBank with RegisterBankFloat; the per-arch port routes that bank to
// the pinned register. The intermediate value lives in a float scratch (X0 on amd64, F0
// on arm64) so the integer scratches can hold the extracted operand indices.
//
// Takes name (string) which is the generated symbol name.
//
// Returns the assembled float-move handler definition.
func moveHandlerFloat(name string) asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      name,
		Comment:   name + " copies floats[C] to floats[B] for the tier-1 register-move sub-op.",
		FrameSize: frameSizeZero,
		Flags:     flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			floatScratches := architecture.FloatScratchRegisters()
			architecture.ExtractB(emitter, scratches[0])
			architecture.ExtractC(emitter, scratches[1])
			architecture.LoadFromBank(emitter, asmgen.RegisterBankFloat, scratches[1], floatScratches[0])
			architecture.StoreToBank(emitter, asmgen.RegisterBankFloat, floatScratches[0], scratches[0])
			architecture.DispatchNext(emitter)
		},
	}
}

// moveHandlerContextBased builds a tier-1 register-move handler for a context-base-loaded
// bank.
//
// Used for banks whose base address is loaded lazily from the DispatchContext
// (CTX_BOOLS_BASE or CTX_UINTS_BASE). The arch port's LoadFromBoolBank / LoadFromUintBank
// primitives handle the context-load via a baseScratch register; this helper composes the
// same pair as the tier-0 handlerMoveBool / handlerMoveUint templates but with
// ExtractB/ExtractC for tier-1 operand layout.
//
// Takes name (string) which is the generated symbol name.
// Takes bank (contextBaseBank) which selects Bool vs Uint bank access primitives.
//
// Returns the assembled context-based move handler definition.
func moveHandlerContextBased(name string, bank contextBaseBank) asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      name,
		Comment:   name + " copies bank[C] to bank[B] for a context-base-loaded register-move sub-op.",
		FrameSize: frameSizeZero,
		Flags:     flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			const (
				scratchDstReg  = 0
				scratchSrcReg  = 1
				scratchTemp    = 2
				scratchBaseReg = 3
			)
			scratches := architecture.ScratchRegisters()

			architecture.ExtractB(emitter, scratches[scratchDstReg])
			architecture.ExtractC(emitter, scratches[scratchSrcReg])
			temp := scratches[scratchTemp]
			baseScratch := scratches[scratchBaseReg]
			switch bank {
			case contextBaseBool:
				architecture.LoadFromBoolBank(emitter, scratches[scratchSrcReg], temp, baseScratch)
				architecture.StoreToBoolBank(emitter, temp, scratches[scratchDstReg], baseScratch)
			case contextBaseUint:
				architecture.LoadFromUintBank(emitter, scratches[scratchSrcReg], temp, baseScratch)
				architecture.StoreToUintBank(emitter, temp, scratches[scratchDstReg], baseScratch)
			}
			architecture.DispatchNext(emitter)
		},
	}
}
