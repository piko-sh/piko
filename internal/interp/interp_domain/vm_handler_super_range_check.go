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

const (
	// rangeCheckUintFusionNopCount is the number of opNops that fill the slots after the two
	// extension words in a fused range-check pattern. The original 8-op sequence
	// (LoadConst+Cmp+Move+JumpIfFalse twice) collapses to 3 words (primary + 2 ext) plus 5
	// trailing opNops.
	rangeCheckUintFusionNopCount = 5

	// eqUintConstJumpFalseNopCount is the trailing-opNop slot count for the equality fusion.
	// The original 3-op sequence collapses to 2 words (primary + 1 ext) plus 1 trailing
	// opNop.
	eqUintConstJumpFalseNopCount = 1
)

// handleSubOpRangeCheckUintJumpFalse implements the fused range-check super-instruction.
// It collapses the canonical byte classifier pattern `value >= lo && value <= hi` into a
// single tier-1 dispatch with two extension words and five trailing opNops.
//
// The primary word encodes the value register in instr.b. The first extension word
// carries loConst in ext1.a and hiConst in ext1.b. The second extension word carries the
// signed jump offset in ext2.a and ext2.b (packed via splitOffset). The five opNops at
// programCounter positions immediately following the two extension words are skipped here
// so dispatch resumes at the post-range-check block, matching the position the second
// JumpIfFalse would have landed on in the original 8-op sequence.
//
// Takes frame (*callFrame) which provides access to the bytecode body and program
// counter.
// Takes registers (*Registers) which provides the uint register bank.
// Takes instr (instruction) which encodes the value register index in its b field.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleSubOpRangeCheckUintJumpFalse(_ *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	ext1 := frame.function.body[frame.programCounter]
	frame.programCounter++
	ext2 := frame.function.body[frame.programCounter]
	frame.programCounter++
	frame.programCounter += rangeCheckUintFusionNopCount
	value := registers.uints[instr.b]
	lo := uint64(ext1.a)
	hi := uint64(ext1.b)
	if value < lo || value > hi {
		offset := joinOffset(ext2.a, ext2.b)
		frame.programCounter += int(offset)
	}
	return opContinue
}

// handleSubOpEqUintConstJumpFalse implements the 3-op equality-and-jump fusion that
// collapses `LoadUintConstSmall + EqUint + JumpIfFalse` into a single tier-1 dispatch
// with 1 extension word and 1 trailing opNop. Hits the canonical `switch byteValue` / `if
// v == const` dispatch pattern that dominates byte-classifier loops (brainfuck dispatch,
// tokenisers, hand-rolled lexers).
//
// Primary word operands: instr.b = value register, instr.c = 8-bit immediate constant.
// The extension word carries the signed jump offset packed via splitOffset. The trailing
// opNop preserves the original 3-slot length so subsequent instructions and any jump
// targets land at unchanged indices.
//
// Takes frame (*callFrame) which provides the bytecode body and program counter.
// Takes registers (*Registers) which provides the uint register bank.
// Takes instr (instruction) which encodes the value reg in b and the immediate const in
// c.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleSubOpEqUintConstJumpFalse(_ *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	ext1 := frame.function.body[frame.programCounter]
	frame.programCounter++
	frame.programCounter += eqUintConstJumpFalseNopCount
	if registers.uints[instr.b] != uint64(instr.c) {
		offset := joinOffset(ext1.a, ext1.b)
		frame.programCounter += int(offset)
	}
	return opContinue
}
