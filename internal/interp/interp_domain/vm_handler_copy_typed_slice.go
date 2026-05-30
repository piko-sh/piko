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

// handleSubOpCopySliceIntDirect implements `copy(dst, src)` between two typed slicesInt
// headers without crossing through reflect.Copy.
//
// Reads the destination header from slicesInt[instr.b], the source header from
// slicesInt[instr.c], performs Go's builtin copy, and writes the element count to
// ints[extensionWord.a].
//
// Encoding:
//
//	{opDrillTier1, subOpCopySliceIntDirect, dstB, srcC}
//	{opExt, countResultA, 0, 0}
//
// Takes frame (*callFrame) whose programCounter is advanced past the extension word.
// Takes registers (*Registers) which provides the slicesInt banks.
// Takes instr (instruction) which encodes the destination slicesInt register B and the
// source slicesInt register C.
//
// Returns opResult which is opContinue.
func handleSubOpCopySliceIntDirect(_ *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	count := copy(registers.slicesInt[instr.b], registers.slicesInt[instr.c])
	registers.ints[extensionWord.a] = int64(count)
	return opContinue
}

// handleSubOpCopySliceFloatDirect mirrors handleSubOpCopySliceIntDirect for slicesFloat.
//
// Takes vm, frame, registers, instr (see handleSubOpCopySliceIntDirect).
//
// Returns opContinue.
func handleSubOpCopySliceFloatDirect(_ *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	count := copy(registers.slicesFloat[instr.b], registers.slicesFloat[instr.c])
	registers.ints[extensionWord.a] = int64(count)
	return opContinue
}

// handleSubOpCopySliceStringDirect mirrors handleSubOpCopySliceIntDirect for
// slicesString.
//
// Takes vm, frame, registers, instr.
//
// Returns opContinue.
func handleSubOpCopySliceStringDirect(_ *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	count := copy(registers.slicesString[instr.b], registers.slicesString[instr.c])
	registers.ints[extensionWord.a] = int64(count)
	return opContinue
}

// handleSubOpCopySliceBoolDirect mirrors handleSubOpCopySliceIntDirect for slicesBool.
//
// Takes vm, frame, registers, instr.
//
// Returns opContinue.
func handleSubOpCopySliceBoolDirect(_ *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	count := copy(registers.slicesBool[instr.b], registers.slicesBool[instr.c])
	registers.ints[extensionWord.a] = int64(count)
	return opContinue
}

// handleSubOpCopySliceUintDirect mirrors handleSubOpCopySliceIntDirect for slicesUint.
//
// Takes vm, frame, registers, instr.
//
// Returns opContinue.
func handleSubOpCopySliceUintDirect(_ *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	count := copy(registers.slicesUint[instr.b], registers.slicesUint[instr.c])
	registers.ints[extensionWord.a] = int64(count)
	return opContinue
}

// handleSubOpCopySliceByteDirect mirrors handleSubOpCopySliceIntDirect for slicesByte.
//
// Takes vm, frame, registers, instr.
//
// Returns opContinue.
func handleSubOpCopySliceByteDirect(_ *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	count := copy(registers.slicesByte[instr.b], registers.slicesByte[instr.c])
	registers.ints[extensionWord.a] = int64(count)
	return opContinue
}
