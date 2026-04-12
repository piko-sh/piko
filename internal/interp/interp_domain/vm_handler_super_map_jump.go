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
	"reflect"
)

// handleMapIndexOkJumpIfFalseIntInt fuses opMapIndexOkIntInt and opJumpIfFalse.
//
// The handler reads ints[A] = map[int]int (or int64) in general[B] with key ints[C]; the
// extension word carries the ok-register index in ext.a and the signed jump offset packed
// into ext.b (lo) and ext.c (hi). On !ok programCounter advances by jumpOffset.
//
// The fuser packs offset = (jumpPos - i - 1) + origOffset, so after the handler consumes
// the ext word at PC=i+1 the program counter sits at i+2; on !ok PC += offset, landing
// exactly where the original jump-taken branch landed. On ok the handler returns
// opContinue and dispatch walks through any intermediate MOVE_INT/MOVE_GENERAL ops left
// in place by the fuser plus the trailing opNop that replaced the original jumpIfFalse.
// Body mirrors handleMapIndexOkIntInt with the added jump tail.
//
// Takes vm (*VM) which supplies the per-VM scratch key value.
// Takes frame (*callFrame) which carries the program counter and function body used to
// read the extension word.
// Takes registers (*Registers) which holds the source map and key and receives the value
// and ok bit.
// Takes instruction (instruction) which encodes (a, b, c) operand indices for value, map,
// and key.
//
// Returns opContinue once dispatch consumes the extension word and optionally adjusts the
// program counter.
func handleMapIndexOkJumpIfFalseIntInt(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	m := registers.general[instruction.b]
	if !m.IsValid() {
		vmPanicInvalidRegister("handleMapIndexOkJumpIfFalseIntInt", registerRoleMap, instruction.b, instruction, frame, registers)
	}
	if !mapKindError(vm, m) {
		return opPanicError
	}
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	key := registers.ints[instruction.c]
	ok := false
	if concrete, hit := reflect.TypeAssert[map[int]int](m); hit {
		value, present := concrete[int(key)]
		registers.ints[instruction.a] = int64(value)
		ok = present
	} else if concrete, hit := reflect.TypeAssert[map[int64]int64](m); hit {
		value, present := concrete[key]
		registers.ints[instruction.a] = value
		ok = present
	} else {
		keyReflectValue := intMapKeyScratch(vm, m.Type().Key())
		keyReflectValue.SetInt(key)
		result := m.MapIndex(keyReflectValue)
		if result.IsValid() {
			registers.ints[instruction.a] = result.Int()
			ok = true
		} else {
			registers.ints[instruction.a] = 0
		}
	}
	registers.ints[extensionWord.a] = boolToInt64(ok)
	if !ok {
		offset := joinOffset(extensionWord.b, extensionWord.c)
		frame.programCounter += int(offset)
	}
	return opContinue
}

// handleMapIndexOkJumpIfFalseStringInt fuses opMapIndexOkStringInt and opJumpIfFalse.
// Same shape as handleMapIndexOkJumpIfFalseIntInt for map[string]int with string keys.
//
// Takes frame (*callFrame) which carries the program counter and function body used to
// read the extension word.
// Takes registers (*Registers) which holds the source map and key and receives the value
// and ok bit.
// Takes instruction (instruction) which encodes (a, b, c) operand indices for value, map,
// and key.
//
// Returns opContinue once dispatch consumes the extension word and optionally adjusts the
// program counter.
func handleMapIndexOkJumpIfFalseStringInt(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	m := registers.general[instruction.b]
	if !m.IsValid() {
		vmPanicInvalidRegister("handleMapIndexOkJumpIfFalseStringInt", registerRoleMap, instruction.b, instruction, frame, registers)
	}
	if !mapKindError(vm, m) {
		return opPanicError
	}
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	key := registers.strings[instruction.c]
	ok := false
	if concrete, hit := reflect.TypeAssert[map[string]int](m); hit {
		value, present := concrete[key]
		registers.ints[instruction.a] = int64(value)
		ok = present
	} else if concrete, hit := reflect.TypeAssert[map[string]int64](m); hit {
		value, present := concrete[key]
		registers.ints[instruction.a] = value
		ok = present
	} else {
		keyReflectValue := reflect.ValueOf(key).Convert(m.Type().Key())
		result := m.MapIndex(keyReflectValue)
		if result.IsValid() {
			registers.ints[instruction.a] = result.Int()
			ok = true
		} else {
			registers.ints[instruction.a] = 0
		}
	}
	registers.ints[extensionWord.a] = boolToInt64(ok)
	if !ok {
		offset := joinOffset(extensionWord.b, extensionWord.c)
		frame.programCounter += int(offset)
	}
	return opContinue
}

// handleMapIndexOkJumpIfFalseStringString fuses opMapIndexOkStringString and
// opJumpIfFalse for map[string]string lookups.
//
// Takes frame (*callFrame) which carries the program counter and function body used to
// read the extension word.
// Takes registers (*Registers) which holds the source map and key and receives the value
// and ok bit.
// Takes instruction (instruction) which encodes (a, b, c) operand indices for value, map,
// and key.
//
// Returns opContinue once dispatch consumes the extension word and optionally adjusts the
// program counter.
func handleMapIndexOkJumpIfFalseStringString(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	m := registers.general[instruction.b]
	if !m.IsValid() {
		vmPanicInvalidRegister("handleMapIndexOkJumpIfFalseStringString", registerRoleMap, instruction.b, instruction, frame, registers)
	}
	if !mapKindError(vm, m) {
		return opPanicError
	}
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	key := registers.strings[instruction.c]
	ok := false
	if concrete, hit := reflect.TypeAssert[map[string]string](m); hit {
		value, present := concrete[key]
		registers.strings[instruction.a] = value
		ok = present
	} else {
		keyReflectValue := reflect.ValueOf(key).Convert(m.Type().Key())
		result := m.MapIndex(keyReflectValue)
		if result.IsValid() {
			registers.strings[instruction.a] = result.String()
			ok = true
		} else {
			registers.strings[instruction.a] = ""
		}
	}
	registers.ints[extensionWord.a] = boolToInt64(ok)
	if !ok {
		offset := joinOffset(extensionWord.b, extensionWord.c)
		frame.programCounter += int(offset)
	}
	return opContinue
}

// handleMapIndexOkJumpIfFalseIntString fuses opMapIndexOkIntString and opJumpIfFalse for
// map[int]string / map[int64]string lookups.
//
// Takes vm (*VM) which supplies the per-VM scratch key value.
// Takes frame (*callFrame) which carries the program counter and function body used to
// read the extension word.
// Takes registers (*Registers) which holds the source map and key and receives the value
// and ok bit.
// Takes instruction (instruction) which encodes (a, b, c) operand indices for value, map,
// and key.
//
// Returns opContinue once dispatch consumes the extension word and optionally adjusts the
// program counter.
func handleMapIndexOkJumpIfFalseIntString(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	m := registers.general[instruction.b]
	if !m.IsValid() {
		vmPanicInvalidRegister("handleMapIndexOkJumpIfFalseIntString", registerRoleMap, instruction.b, instruction, frame, registers)
	}
	if !mapKindError(vm, m) {
		return opPanicError
	}
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	key := registers.ints[instruction.c]
	ok := false
	if concrete, hit := reflect.TypeAssert[map[int]string](m); hit {
		value, present := concrete[int(key)]
		registers.strings[instruction.a] = value
		ok = present
	} else if concrete, hit := reflect.TypeAssert[map[int64]string](m); hit {
		value, present := concrete[key]
		registers.strings[instruction.a] = value
		ok = present
	} else {
		keyScratch := intMapKeyScratch(vm, m.Type().Key())
		keyScratch.SetInt(key)
		result := m.MapIndex(keyScratch)
		if result.IsValid() {
			registers.strings[instruction.a] = result.String()
			ok = true
		} else {
			registers.strings[instruction.a] = ""
		}
	}
	registers.ints[extensionWord.a] = boolToInt64(ok)
	if !ok {
		offset := joinOffset(extensionWord.b, extensionWord.c)
		frame.programCounter += int(offset)
	}
	return opContinue
}

// handleMapIndexOkJumpIfFalseIntGeneral fuses opMapIndexOkIntGeneral and opJumpIfFalse
// for map[int]V where V is in the general bank.
//
// Covers pointers, slices, maps, and interfaces, as well as the `node, ok := m[k]; if !ok
// { ... }` idiom.
//
// Takes vm (*VM) which supplies the per-VM scratch key value.
// Takes frame (*callFrame) which carries the program counter and function body used to
// read the extension word.
// Takes registers (*Registers) which holds the source map and key and receives the value
// and ok bit.
// Takes instruction (instruction) which encodes (a, b, c) operand indices for value, map,
// and key.
//
// Returns opContinue once dispatch consumes the extension word and optionally adjusts the
// program counter.
func handleMapIndexOkJumpIfFalseIntGeneral(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	m := registers.general[instruction.b]
	if !m.IsValid() {
		vmPanicInvalidRegister("handleMapIndexOkJumpIfFalseIntGeneral", registerRoleMap, instruction.b, instruction, frame, registers)
	}
	if !mapKindError(vm, m) {
		return opPanicError
	}
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	key := registers.ints[instruction.c]
	keyType := m.Type().Key()
	elemType := m.Type().Elem()
	ok := false
	if useMapFastLinkname() && mapKeyIsFast64Eligible(keyType) {
		resultPtr, present := mapAccessFast64ToGeneral(m, key)
		if present {
			registers.general[instruction.a] = wrapMapElemFast(elemType, resultPtr)
			ok = true
		} else {
			registers.general[instruction.a] = zeroValueForType(elemType)
		}
	} else {
		keyScratch := intMapKeyScratch(vm, keyType)
		keyScratch.SetInt(key)
		result := m.MapIndex(keyScratch)
		if result.IsValid() {
			registers.general[instruction.a] = result
			ok = true
		} else {
			registers.general[instruction.a] = zeroValueForType(elemType)
		}
	}
	registers.ints[extensionWord.a] = boolToInt64(ok)
	if !ok {
		offset := joinOffset(extensionWord.b, extensionWord.c)
		frame.programCounter += int(offset)
	}
	return opContinue
}

// handleMapIndexOkJumpIfFalseStringGeneral fuses opMapIndexOkStringGeneral and
// opJumpIfFalse for map[string]V where V is in the general bank.
//
// Takes frame (*callFrame) which carries the program counter and function body used to
// read the extension word.
// Takes registers (*Registers) which holds the source map and key and receives the value
// and ok bit.
// Takes instruction (instruction) which encodes (a, b, c) operand indices for value, map,
// and key.
//
// Returns opContinue once dispatch consumes the extension word and optionally adjusts the
// program counter.
func handleMapIndexOkJumpIfFalseStringGeneral(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	m := registers.general[instruction.b]
	if !m.IsValid() {
		vmPanicInvalidRegister("handleMapIndexOkJumpIfFalseStringGeneral", registerRoleMap, instruction.b, instruction, frame, registers)
	}
	if !mapKindError(vm, m) {
		return opPanicError
	}
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	key := registers.strings[instruction.c]
	elemType := m.Type().Elem()
	ok := false
	if !useMapFastLinkname() {
		result := m.MapIndex(reflect.ValueOf(key))
		if result.IsValid() {
			registers.general[instruction.a] = result
			ok = true
		} else {
			registers.general[instruction.a] = zeroValueForType(elemType)
		}
	} else {
		resultPtr, present := mapAccessFastStrToGeneral(m, key)
		if present {
			registers.general[instruction.a] = wrapMapElemFast(elemType, resultPtr)
			ok = true
		} else {
			registers.general[instruction.a] = zeroValueForType(elemType)
		}
	}
	registers.ints[extensionWord.a] = boolToInt64(ok)
	if !ok {
		offset := joinOffset(extensionWord.b, extensionWord.c)
		frame.programCounter += int(offset)
	}
	return opContinue
}
