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

package interp_domain

import "fmt"

// instruction is a compact 4-byte bytecode instruction.
//
// The format is [Op][A][B][C] where each field is a uint8.
// For most instructions:
//   - A is the destination register
//   - B and C are source registers or immediates
//
// For jump instructions, B|(C<<8) forms a signed 16-bit offset.
// For instructions needing wider operands, an opExt instruction
// follows with A|(B<<8)|(C<<16) forming a 24-bit payload.
type instruction struct {
	// op is the opcode identifying the operation to perform.
	op opcode

	// a is the first operand, typically the destination register index.
	a uint8

	// b is the second operand, typically a source register index or
	// the low byte of a wide immediate.
	b uint8

	// c is the third operand, typically a source register index or
	// the high byte of a wide immediate.
	c uint8
}

// String returns a human-readable representation of the instruction
// for debugging and disassembly.
//
// Returns a formatted string showing the opcode and operand values.
func (i instruction) String() string {
	return fmt.Sprintf("%-18s %3d %3d %3d", i.op, i.a, i.b, i.c)
}

// signedOffset extracts a signed 16-bit jump offset from B|(C<<8).
//
// Returns the offset as an int16.
func (i instruction) signedOffset() int16 {
	return joinOffset(i.b, i.c)
}

// wideIndex extracts an unsigned 16-bit index from B|(C<<8).
//
// Returns the index as a uint16.
func (i instruction) wideIndex() uint16 {
	return joinWide(i.b, i.c)
}

// makeInstruction creates an instruction from its components.
//
// Takes op (opcode) which is the operation to encode.
// Takes a (uint8) which is the first operand.
// Takes b (uint8) which is the second operand.
// Takes c (uint8) which is the third operand.
//
// Returns the assembled instruction.
func makeInstruction(op opcode, a, b, c uint8) instruction {
	return instruction{op: op, a: a, b: b, c: c}
}

// splitWide splits a uint16 into low and high bytes for bytecode
// B|C encoding.
//
// Takes value (uint16) which is the wide operand to split.
//
// Returns lo (uint8) which is the low byte (stored in B).
// Returns hi (uint8) which is the high byte (stored in C).
func splitWide(value uint16) (lo, hi uint8) {
	return uint8(value), uint8(value >> 8) //nolint:gosec // intentional byte extraction
}

// joinWide reconstructs a uint16 from low and high bytes.
//
// Takes lo (uint8) which is the low byte (from B).
// Takes hi (uint8) which is the high byte (from C).
//
// Returns the reconstructed uint16.
func joinWide(lo, hi uint8) uint16 {
	return uint16(lo) | uint16(hi)<<wideBitShift
}

// decodeExtension24 reconstructs a 24-bit unsigned integer from the
// three fields of an opExt instruction (a=low, b=mid, c=high byte).
//
// Takes ext (instruction) which is the extension instruction.
//
// Returns the decoded 24-bit value as an int.
func decodeExtension24(ext instruction) int {
	return int(ext.a) | int(ext.b)<<wideBitShift | int(ext.c)<<(2*wideBitShift)
}

// splitOffset splits a signed 16-bit jump offset into low and high
// bytes, preserving two's complement representation.
//
// Takes offset (int16) which is the signed offset to split.
//
// Returns lo (uint8) which is the low byte.
// Returns hi (uint8) which is the high byte.
func splitOffset(offset int16) (lo, hi uint8) {
	return splitWide(uint16(offset)) //nolint:gosec // two's complement preserved
}

// joinOffset reconstructs a signed 16-bit jump offset from low and
// high bytes.
//
// Takes lo (uint8) which is the low byte.
// Takes hi (uint8) which is the high byte.
//
// Returns the reconstructed int16 offset.
func joinOffset(lo, hi uint8) int16 {
	return int16(joinWide(lo, hi)) //nolint:gosec // intentional reinterpretation
}
