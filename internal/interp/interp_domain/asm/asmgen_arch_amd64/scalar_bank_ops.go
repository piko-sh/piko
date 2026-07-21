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
	"piko.sh/asmgen"
	"piko.sh/asmgen/asmamd64"
)

// LoadFromUintBank implements BytecodeArchPort. Loads CTX_UINTS_BASE into baseScratch,
// then loads the indexed uint64 into the destination.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes indexRegister (string) which holds the register index.
// Takes destinationRegister (string) which receives the loaded value.
// Takes baseScratch (string) which is a scratch register for the base.
func (*BytecodeAMD64Arch) LoadFromUintBank(e *asmgen.Emitter, indexRegister, destinationRegister, baseScratch string) {
	inst(e, asmamd64.OperationMove64Bits, "CTX_UINTS_BASE(R15), "+baseScratch)
	inst(e, asmamd64.OperationMove64Bits, "("+baseScratch+")("+indexRegister+"*8), "+destinationRegister)
}

// StoreToUintBank implements BytecodeArchPort. Loads CTX_UINTS_BASE into baseScratch,
// then stores the source register at the index.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes sourceRegister (string) which holds the value to store.
// Takes indexRegister (string) which holds the destination index.
// Takes baseScratch (string) which is a scratch register for the base.
func (*BytecodeAMD64Arch) StoreToUintBank(e *asmgen.Emitter, sourceRegister, indexRegister, baseScratch string) {
	inst(e, asmamd64.OperationMove64Bits, "CTX_UINTS_BASE(R15), "+baseScratch)
	inst(e, asmamd64.OperationMove64Bits, sourceRegister+", ("+baseScratch+")("+indexRegister+"*8)")
}

// LoadFromBoolBank implements BytecodeArchPort. Bools are 1-byte elements; the load uses
// MOVBQZX for zero-extension to a 64-bit destination so the boolean cleanly becomes 0 or
// 1.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes indexRegister (string) which holds the register index.
// Takes destinationRegister (string) which receives the loaded value.
// Takes baseScratch (string) which is a scratch register for the base.
func (*BytecodeAMD64Arch) LoadFromBoolBank(e *asmgen.Emitter, indexRegister, destinationRegister, baseScratch string) {
	inst(e, asmamd64.OperationMove64Bits, "CTX_BOOLS_BASE(R15), "+baseScratch)
	inst(e, asmamd64.OperationMove8To64BitsZeroExtended, "("+baseScratch+")("+indexRegister+"*1), "+destinationRegister)
}

// StoreToBoolBank implements BytecodeArchPort. Stores the low byte of the source register
// into the bool bank at the index.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes sourceRegister (string) which holds the value to store.
// Takes indexRegister (string) which holds the destination index.
// Takes baseScratch (string) which is a scratch register for the base.
func (*BytecodeAMD64Arch) StoreToBoolBank(e *asmgen.Emitter, sourceRegister, indexRegister, baseScratch string) {
	inst(e, asmamd64.OperationMove64Bits, "CTX_BOOLS_BASE(R15), "+baseScratch)
	inst(e, asmamd64.OperationMove8Bits, lowByteOf(sourceRegister)+", ("+baseScratch+")("+indexRegister+"*1)")
}

// BitwiseNotInPlace implements BytecodeArchPort. amd64's NOTQ flips every bit of a 64-bit
// register in place.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes register (string) which is the register to complement.
func (*BytecodeAMD64Arch) BitwiseNotInPlace(e *asmgen.Emitter, register string) {
	inst(e, asmamd64.OperationBitwiseNot64Bits, register)
}

// LogicalSetNonZero implements BytecodeArchPort.
//
// Writes 1 to the destination register if the source is non-zero, 0 otherwise. Mirrors
// handleIntToBool's Go-tier semantics.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes destinationRegister (string) which receives 0 or 1.
// Takes sourceRegister (string) which is tested for non-zero.
func (*BytecodeAMD64Arch) LogicalSetNonZero(e *asmgen.Emitter, destinationRegister, sourceRegister string) {
	inst(e, asmamd64.OperationTest64Bits, sourceRegister+", "+sourceRegister)
	inst(e, asmamd64.OperationMove64Bits, "$0, "+destinationRegister)
	inst(e, asmamd64.OperationSetIfNotEqual, lowByteOf(destinationRegister))
}

// LogicalNot implements BytecodeArchPort.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes destinationIndex (string) which is the result register index.
// Takes sourceIndex (string) which is the source register index.
func (*BytecodeAMD64Arch) LogicalNot(e *asmgen.Emitter, destinationIndex, sourceIndex string) {
	inst(e, asmamd64.OperationMove64Bits, "(R8)("+sourceIndex+"*8), SI")
	inst(e, asmamd64.OperationTest64Bits, "SI, SI")
	inst(e, asmamd64.OperationMove64Bits, "$0, SI")
	inst(e, asmamd64.OperationSetIfEqual, "SI")
	inst(e, asmamd64.OperationMove64Bits, "SI, (R8)("+destinationIndex+"*8)")
}

// lowByteOf returns the byte-sized name of an amd64 64-bit GP register using Go's
// plan9-flavoured assembler convention.
//
// Differs subtly from Intel syntax: SI's low byte is just "SI" (the assembler infers the
// size from MOVB), and DI's low byte is "DIB". Falls back to low8Map when available,
// returning the input unchanged for any register the map doesn't cover.
//
// Takes reg (string) which is the 64-bit register name.
//
// Returns the 8-bit low-byte register name, or reg unchanged when the register is not in
// low8Map.
func lowByteOf(reg string) string {
	if mapped, ok := low8Map[reg]; ok {
		return mapped
	}
	return reg
}
