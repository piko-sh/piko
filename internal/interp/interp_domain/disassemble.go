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

import (
	"fmt"
	"strings"
)

const (
	// maxDisassembleStringLen is the maximum string constant length shown
	// in disassembly comments before truncation.
	maxDisassembleStringLen = 40

	// truncatedDisassembleStringLen is the prefix length kept when truncating.
	truncatedDisassembleStringLen = 37
)

// Disassemble returns a human-readable listing of the function's
// bytecode. Each line shows the program counter, opcode mnemonic,
// operands, and an inline comment for constant-loading instructions.
//
// Returns the complete disassembly as a string, or an empty string
// when the function body is empty.
func (cf *CompiledFunction) Disassemble() string {
	if len(cf.body) == 0 {
		return ""
	}
	return cf.DisassembleRange(0, len(cf.body))
}

// DisassembleRange returns a human-readable listing for the
// instruction range [start, end). Out-of-bounds indices are clamped
// to the body length.
//
// Takes start (int) which is the inclusive start index.
// Takes end (int) which is the exclusive end index.
//
// Returns the disassembly for the range as a string.
func (cf *CompiledFunction) DisassembleRange(start, end int) string {
	if start < 0 {
		start = 0
	}
	if end > len(cf.body) {
		end = len(cf.body)
	}
	if start >= end {
		return ""
	}

	var builder strings.Builder
	for pc := start; pc < end; pc++ {
		instr := cf.body[pc]
		comment := cf.disassembleComment(instr)
		if comment != "" {
			fmt.Fprintf(&builder, "%04d  %-26s %3d %3d %3d    ; %s\n",
				pc, instr.op, instr.a, instr.b, instr.c, comment)
		} else {
			fmt.Fprintf(&builder, "%04d  %-26s %3d %3d %3d\n",
				pc, instr.op, instr.a, instr.b, instr.c)
		}
	}
	return builder.String()
}

// disassembleComment returns an inline comment for instructions that
// reference constant pools or have special semantics.
//
// Takes instr (instruction) which is the instruction to annotate.
//
// Returns the comment string, or an empty string if none applies.
func (cf *CompiledFunction) disassembleComment(instr instruction) string {
	if comment := cf.disassembleLoadComment(instr); comment != "" {
		return comment
	}
	return disassembleControlComment(instr)
}

// disassembleLoadComment handles constant-loading and data instructions.
//
// Takes instr (instruction) which is the instruction to annotate.
//
// Returns a comment string for constant-load opcodes, or empty string
// for other opcodes.
func (cf *CompiledFunction) disassembleLoadComment(instr instruction) string {
	switch instr.op {
	case opLoadIntConst:
		index := instr.wideIndex()
		if int(index) < len(cf.intConstants) {
			return fmt.Sprintf("ints[%d] = %d", instr.a, cf.intConstants[index])
		}
	case opLoadIntConstSmall:
		return fmt.Sprintf("ints[%d] = %d", instr.a, instr.b)
	case opLoadFloatConst:
		index := instr.wideIndex()
		if int(index) < len(cf.floatConstants) {
			return fmt.Sprintf("floats[%d] = %g", instr.a, cf.floatConstants[index])
		}
	case opLoadStringConst:
		return cf.disassembleStringConstComment(instr)
	case opLoadBool:
		if instr.b != 0 {
			return fmt.Sprintf("ints[%d] = true", instr.a)
		}
		return fmt.Sprintf("ints[%d] = false", instr.a)
	case opLoadBoolConst:
		if int(instr.b) < len(cf.boolConstants) {
			return fmt.Sprintf("bools[%d] = %v", instr.a, cf.boolConstants[instr.b])
		}
	case opLoadNil:
		return fmt.Sprintf("general[%d] = nil", instr.a)
	case opAddIntConst, opSubIntConst, opMulIntConst:
		if int(instr.c) < len(cf.intConstants) {
			return fmt.Sprintf("const = %d", cf.intConstants[instr.c])
		}
	}
	return ""
}

// disassembleStringConstComment formats a string constant load comment.
//
// Takes instr (instruction) which is the opLoadStringConst instruction.
//
// Returns a quoted comment showing the destination register and value.
func (cf *CompiledFunction) disassembleStringConstComment(instr instruction) string {
	index := instr.wideIndex()
	if int(index) >= len(cf.stringConstants) {
		return ""
	}
	s := cf.stringConstants[index]
	if len(s) > maxDisassembleStringLen {
		s = s[:truncatedDisassembleStringLen] + "..."
	}
	return fmt.Sprintf("strings[%d] = %q", instr.a, s)
}

// disassembleControlComment handles control flow and return instructions.
//
// Takes instr (instruction) which is the instruction to annotate.
//
// Returns a comment string for jump and return opcodes, or empty
// string for other opcodes.
func disassembleControlComment(instr instruction) string {
	switch instr.op {
	case opJump:
		return fmt.Sprintf("goto %d", int(instr.signedOffset()))
	case opJumpIfTrue:
		return fmt.Sprintf("if ints[%d] != 0 goto %+d", instr.a, int(instr.signedOffset()))
	case opJumpIfFalse:
		return fmt.Sprintf("if ints[%d] == 0 goto %+d", instr.a, int(instr.signedOffset()))
	case opReturn:
		return fmt.Sprintf("%d values", instr.a)
	case opReturnVoid:
		return "void"
	}
	return ""
}
