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
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func testRegCounts() [NumRegisterKinds]uint32 {
	var counts [NumRegisterKinds]uint32
	counts[registerInt] = 4
	counts[registerFloat] = 4
	counts[registerString] = 4
	counts[registerGeneral] = 4
	counts[registerBool] = 4
	counts[registerUint] = 4
	counts[registerComplex] = 4
	return counts
}

func TestZeroTypedRegisterDirect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		check func(t *testing.T, regs *Registers)
		name  string
		kind  registerKind
	}{
		{name: "int", kind: registerInt, check: func(t *testing.T, regs *Registers) {
			regs.ints[0] = 42
			zeroTypedRegister(regs, varLocation{register: 0, kind: registerInt})
			require.Equal(t, int64(0), regs.ints[0])
		}},
		{name: "float", kind: registerFloat, check: func(t *testing.T, regs *Registers) {
			regs.floats[0] = 3.14
			zeroTypedRegister(regs, varLocation{register: 0, kind: registerFloat})
			require.Equal(t, float64(0), regs.floats[0])
		}},
		{name: "string", kind: registerString, check: func(t *testing.T, regs *Registers) {
			regs.strings[0] = "hello"
			zeroTypedRegister(regs, varLocation{register: 0, kind: registerString})
			require.Equal(t, "", regs.strings[0])
		}},
		{name: "bool", kind: registerBool, check: func(t *testing.T, regs *Registers) {
			regs.bools[0] = true
			zeroTypedRegister(regs, varLocation{register: 0, kind: registerBool})
			require.Equal(t, false, regs.bools[0])
		}},
		{name: "uint", kind: registerUint, check: func(t *testing.T, regs *Registers) {
			regs.uints[0] = 42
			zeroTypedRegister(regs, varLocation{register: 0, kind: registerUint})
			require.Equal(t, uint64(0), regs.uints[0])
		}},
		{name: "complex", kind: registerComplex, check: func(t *testing.T, regs *Registers) {
			regs.complex[0] = 1 + 2i
			zeroTypedRegister(regs, varLocation{register: 0, kind: registerComplex})
			require.Equal(t, complex128(0), regs.complex[0])
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.check(t, new(newRegisters(testRegCounts())))
		})
	}
}

func TestCopyReturnToGeneral(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		setup  func(regs *Registers)
		name   string
		kind   registerKind
	}{
		{name: "int", kind: registerInt, setup: func(regs *Registers) { regs.ints[0] = 42 }, expect: int64(42)},
		{name: "float", kind: registerFloat, setup: func(regs *Registers) { regs.floats[0] = 3.14 }, expect: float64(3.14)},
		{name: "string", kind: registerString, setup: func(regs *Registers) { regs.strings[0] = "hi" }, expect: "hi"},
		{name: "bool", kind: registerBool, setup: func(regs *Registers) { regs.bools[0] = true }, expect: true},
		{name: "uint", kind: registerUint, setup: func(regs *Registers) { regs.uints[0] = 99 }, expect: uint64(99)},
		{name: "complex", kind: registerComplex, setup: func(regs *Registers) { regs.complex[0] = 1 + 2i }, expect: complex128(1 + 2i)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srcRegs := newRegisters(testRegCounts())
			tt.setup(&srcRegs)

			callerRegisters := newRegisters(testRegCounts())
			callerFrame := &callFrame{registers: callerRegisters}

			copyReturnToGeneral(callerFrame, &srcRegs, tt.kind, 0, 1)
			require.Equal(t, tt.expect, callerFrame.registers.general[1].Interface())
		})
	}
}

func TestCopyRegisterSlot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		setup func(regs *Registers)
		check func(t *testing.T, regs *Registers)
		name  string
		kind  registerKind
	}{
		{name: "int", kind: registerInt, setup: func(regs *Registers) { regs.ints[0] = 42 }, check: func(t *testing.T, regs *Registers) { require.Equal(t, int64(42), regs.ints[1]) }},
		{name: "float", kind: registerFloat, setup: func(regs *Registers) { regs.floats[0] = 2.5 }, check: func(t *testing.T, regs *Registers) { require.Equal(t, float64(2.5), regs.floats[1]) }},
		{name: "string", kind: registerString, setup: func(regs *Registers) { regs.strings[0] = "hi" }, check: func(t *testing.T, regs *Registers) { require.Equal(t, "hi", regs.strings[1]) }},
		{name: "bool", kind: registerBool, setup: func(regs *Registers) { regs.bools[0] = true }, check: func(t *testing.T, regs *Registers) { require.Equal(t, true, regs.bools[1]) }},
		{name: "uint", kind: registerUint, setup: func(regs *Registers) { regs.uints[0] = 99 }, check: func(t *testing.T, regs *Registers) { require.Equal(t, uint64(99), regs.uints[1]) }},
		{name: "complex", kind: registerComplex, setup: func(regs *Registers) { regs.complex[0] = 1 + 2i }, check: func(t *testing.T, regs *Registers) { require.Equal(t, complex128(1+2i), regs.complex[1]) }},
		{name: "general", kind: registerGeneral, setup: func(regs *Registers) { regs.general[0] = reflect.ValueOf("val") }, check: func(t *testing.T, regs *Registers) { require.Equal(t, "val", regs.general[1].Interface()) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			regs := newRegisters(testRegCounts())
			tt.setup(&regs)
			copyRegisterSlot(&regs, tt.kind, 1, 0)
			tt.check(t, &regs)
		})
	}
}

func TestUnboxGeneralToScalar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		check func(t *testing.T, regs *Registers)
		value reflect.Value
		name  string
		kind  registerKind
	}{
		{name: "int", value: reflect.ValueOf(int64(42)), kind: registerInt, check: func(t *testing.T, regs *Registers) { require.Equal(t, int64(42), regs.ints[0]) }},
		{name: "float", value: reflect.ValueOf(float64(3.14)), kind: registerFloat, check: func(t *testing.T, regs *Registers) { require.Equal(t, float64(3.14), regs.floats[0]) }},
		{name: "string", value: reflect.ValueOf("hello"), kind: registerString, check: func(t *testing.T, regs *Registers) { require.Equal(t, "hello", regs.strings[0]) }},
		{name: "bool", value: reflect.ValueOf(true), kind: registerBool, check: func(t *testing.T, regs *Registers) { require.Equal(t, true, regs.bools[0]) }},
		{name: "uint", value: reflect.ValueOf(uint64(99)), kind: registerUint, check: func(t *testing.T, regs *Registers) { require.Equal(t, uint64(99), regs.uints[0]) }},
		{name: "complex", value: reflect.ValueOf(complex128(1 + 2i)), kind: registerComplex, check: func(t *testing.T, regs *Registers) { require.Equal(t, complex128(1+2i), regs.complex[0]) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			regs := newRegisters(testRegCounts())
			unboxGeneralToScalar(&regs, tt.value, tt.kind, 0, nil)
			tt.check(t, &regs)
		})
	}
}

func TestCopyReturnFromGeneralInvalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		check func(t *testing.T, regs *Registers)
		name  string
		kind  registerKind
	}{
		{name: "int_zero", kind: registerInt, check: func(t *testing.T, regs *Registers) { require.Equal(t, int64(0), regs.ints[0]) }},
		{name: "string_zero", kind: registerString, check: func(t *testing.T, regs *Registers) { require.Equal(t, "", regs.strings[0]) }},
		{name: "bool_zero", kind: registerBool, check: func(t *testing.T, regs *Registers) { require.Equal(t, false, regs.bools[0]) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			regs := newRegisters(testRegCounts())
			regs.ints[0] = 99
			regs.strings[0] = "dirty"
			regs.bools[0] = true
			frame := &callFrame{registers: regs}

			copyReturnFromGeneral(frame, reflect.Value{}, varLocation{register: 0, kind: tt.kind})
			tt.check(t, &frame.registers)
		})
	}
}

func TestCopyReturnFromGeneralInterface(t *testing.T) {
	t.Parallel()

	var v any = int64(42)
	regs := newRegisters(testRegCounts())
	frame := &callFrame{registers: regs}
	copyReturnFromGeneral(frame, reflect.ValueOf(&v).Elem(), varLocation{register: 0, kind: registerInt})
	require.Equal(t, int64(42), frame.registers.ints[0])
}

func TestUnpackGeneralToTyped(t *testing.T) {
	t.Parallel()

	tests := []struct {
		check func(t *testing.T, regs *Registers)
		value reflect.Value
		name  string
		dest  varLocation
	}{
		{name: "int", value: reflect.ValueOf(int64(42)), dest: varLocation{register: 0, kind: registerInt}, check: func(t *testing.T, regs *Registers) { require.Equal(t, int64(42), regs.ints[0]) }},
		{name: "float", value: reflect.ValueOf(float64(3.14)), dest: varLocation{register: 0, kind: registerFloat}, check: func(t *testing.T, regs *Registers) { require.Equal(t, float64(3.14), regs.floats[0]) }},
		{name: "string", value: reflect.ValueOf("hi"), dest: varLocation{register: 0, kind: registerString}, check: func(t *testing.T, regs *Registers) { require.Equal(t, "hi", regs.strings[0]) }},
		{name: "bool", value: reflect.ValueOf(true), dest: varLocation{register: 0, kind: registerBool}, check: func(t *testing.T, regs *Registers) { require.Equal(t, true, regs.bools[0]) }},
		{name: "uint", value: reflect.ValueOf(uint64(99)), dest: varLocation{register: 0, kind: registerUint}, check: func(t *testing.T, regs *Registers) { require.Equal(t, uint64(99), regs.uints[0]) }},
		{name: "complex", value: reflect.ValueOf(complex128(1 + 2i)), dest: varLocation{register: 0, kind: registerComplex}, check: func(t *testing.T, regs *Registers) { require.Equal(t, complex128(1+2i), regs.complex[0]) }},
		{name: "bool_to_int", value: reflect.ValueOf(true), dest: varLocation{register: 0, kind: registerInt}, check: func(t *testing.T, regs *Registers) { require.Equal(t, int64(1), regs.ints[0]) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			regs := newRegisters(testRegCounts())
			unpackGeneralToTyped(&regs, tt.value, tt.dest)
			tt.check(t, &regs)
		})
	}
}

func TestReflectBinaryOp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		a      reflect.Value
		b      reflect.Value
		name   string
	}{
		{name: "int_add", a: reflect.ValueOf(int64(10)), b: reflect.ValueOf(int64(3)), expect: int64(13)},
		{name: "float_add", a: reflect.ValueOf(float64(1.5)), b: reflect.ValueOf(float64(2.5)), expect: float64(4.0)},
		{name: "string_concat", a: reflect.ValueOf("hello "), b: reflect.ValueOf("world"), expect: "hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := reflectBinaryOp(tt.a, tt.b,
				func(x, y int64) int64 { return x + y },
				func(x, y float64) float64 { return x + y },
				func(x, y string) string { return x + y },
			)
			require.Equal(t, tt.expect, result.Interface())
		})
	}
}

func TestGrowCallStack(t *testing.T) {
	t.Parallel()

	vm := &VM{
		callStack: make([]callFrame, 2),
	}
	originalCap := len(vm.callStack)

	vm.growCallStack()

	require.Equal(t, originalCap*2, len(vm.callStack))
}

func TestGrowCallStackWithASMArrays(t *testing.T) {
	t.Parallel()

	vm := &VM{
		callStack:        make([]callFrame, 2),
		asmCallInfoBases: make([]uintptr, 2),
		asmDispatchSaves: make([]asmDispatchSave, 2),
	}

	vm.growCallStack()

	require.Equal(t, 4, len(vm.callStack))
	require.Equal(t, 4, len(vm.asmCallInfoBases))
	require.Equal(t, 4, len(vm.asmDispatchSaves))
}

func TestKindDefaultReflectType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect reflect.Type
		name   string
		kind   registerKind
	}{
		{name: "int", kind: registerInt, expect: reflect.TypeFor[int64]()},
		{name: "float", kind: registerFloat, expect: reflect.TypeFor[float64]()},
		{name: "string", kind: registerString, expect: reflect.TypeFor[string]()},
		{name: "bool", kind: registerBool, expect: reflect.TypeFor[bool]()},
		{name: "uint", kind: registerUint, expect: reflect.TypeFor[uint64]()},
		{name: "complex", kind: registerComplex, expect: reflect.TypeFor[complex128]()},
		{name: "general", kind: registerGeneral, expect: reflect.TypeFor[any]()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := kindDefaultReflectType(tt.kind)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestWriteRegisterValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		check func(t *testing.T, regs *Registers)
		value reflect.Value
		name  string
		kind  registerKind
	}{
		{name: "int", kind: registerInt, value: reflect.ValueOf(int64(42)), check: func(t *testing.T, regs *Registers) { require.Equal(t, int64(42), regs.ints[0]) }},
		{name: "float", kind: registerFloat, value: reflect.ValueOf(float64(3.14)), check: func(t *testing.T, regs *Registers) { require.Equal(t, float64(3.14), regs.floats[0]) }},
		{name: "string", kind: registerString, value: reflect.ValueOf("hello"), check: func(t *testing.T, regs *Registers) { require.Equal(t, "hello", regs.strings[0]) }},
		{name: "bool", kind: registerBool, value: reflect.ValueOf(true), check: func(t *testing.T, regs *Registers) { require.Equal(t, true, regs.bools[0]) }},
		{name: "uint", kind: registerUint, value: reflect.ValueOf(uint64(99)), check: func(t *testing.T, regs *Registers) { require.Equal(t, uint64(99), regs.uints[0]) }},
		{name: "complex", kind: registerComplex, value: reflect.ValueOf(complex128(1 + 2i)), check: func(t *testing.T, regs *Registers) { require.Equal(t, complex128(1+2i), regs.complex[0]) }},
		{name: "general", kind: registerGeneral, value: reflect.ValueOf("general_val"), check: func(t *testing.T, regs *Registers) { require.Equal(t, "general_val", regs.general[0].Interface()) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			regs := newRegisters(testRegCounts())
			writeRegisterValue(&regs, 0, tt.kind, tt.value)
			tt.check(t, &regs)
		})
	}
}

func TestBoxScalarToGeneral(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		setup  func(regs *Registers)
		name   string
		kind   registerKind
	}{
		{name: "int", kind: registerInt, setup: func(regs *Registers) { regs.ints[0] = 42 }, expect: int64(42)},
		{name: "float", kind: registerFloat, setup: func(regs *Registers) { regs.floats[0] = 3.14 }, expect: float64(3.14)},
		{name: "string", kind: registerString, setup: func(regs *Registers) { regs.strings[0] = "hi" }, expect: "hi"},
		{name: "bool", kind: registerBool, setup: func(regs *Registers) { regs.bools[0] = true }, expect: true},
		{name: "uint", kind: registerUint, setup: func(regs *Registers) { regs.uints[0] = 99 }, expect: uint64(99)},
		{name: "complex", kind: registerComplex, setup: func(regs *Registers) { regs.complex[0] = 1 + 2i }, expect: complex128(1 + 2i)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			src := newRegisters(testRegCounts())
			dst := newRegisters(testRegCounts())
			tt.setup(&src)
			boxScalarToGeneral(&dst, &src, tt.kind, 1, 0)
			require.Equal(t, tt.expect, dst.general[1].Interface())
		})
	}
}

func TestSyncCellToRegister(t *testing.T) {
	t.Parallel()

	tests := []struct {
		check func(t *testing.T, regs *Registers)
		name  string
		cell  upvalueCell
		kind  registerKind
	}{
		{name: "int", kind: registerInt, cell: upvalueCell{intValue: 42}, check: func(t *testing.T, regs *Registers) { require.Equal(t, int64(42), regs.ints[0]) }},
		{name: "float", kind: registerFloat, cell: upvalueCell{floatValue: 3.14}, check: func(t *testing.T, regs *Registers) { require.Equal(t, float64(3.14), regs.floats[0]) }},
		{name: "string", kind: registerString, cell: upvalueCell{stringValue: "hi"}, check: func(t *testing.T, regs *Registers) { require.Equal(t, "hi", regs.strings[0]) }},
		{name: "bool", kind: registerBool, cell: upvalueCell{boolValue: true}, check: func(t *testing.T, regs *Registers) { require.Equal(t, true, regs.bools[0]) }},
		{name: "uint", kind: registerUint, cell: upvalueCell{uintValue: 99}, check: func(t *testing.T, regs *Registers) { require.Equal(t, uint64(99), regs.uints[0]) }},
		{name: "complex", kind: registerComplex, cell: upvalueCell{complexValue: 1 + 2i}, check: func(t *testing.T, regs *Registers) { require.Equal(t, complex128(1+2i), regs.complex[0]) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			regs := newRegisters(testRegCounts())
			syncCellToRegister(&regs, &tt.cell, varLocation{register: 0, kind: tt.kind})
			tt.check(t, &regs)
		})
	}
}

func TestAssignReflectArg(t *testing.T) {
	t.Parallel()

	tests := []struct {
		check func(t *testing.T, regs *Registers)
		value reflect.Value
		name  string
		kind  registerKind
	}{
		{name: "int", kind: registerInt, value: reflect.ValueOf(int64(42)), check: func(t *testing.T, regs *Registers) { require.Equal(t, int64(42), regs.ints[0]) }},
		{name: "float", kind: registerFloat, value: reflect.ValueOf(float64(3.14)), check: func(t *testing.T, regs *Registers) { require.Equal(t, float64(3.14), regs.floats[0]) }},
		{name: "string", kind: registerString, value: reflect.ValueOf("hi"), check: func(t *testing.T, regs *Registers) { require.Equal(t, "hi", regs.strings[0]) }},
		{name: "bool", kind: registerBool, value: reflect.ValueOf(true), check: func(t *testing.T, regs *Registers) { require.Equal(t, true, regs.bools[0]) }},
		{name: "uint", kind: registerUint, value: reflect.ValueOf(uint64(99)), check: func(t *testing.T, regs *Registers) { require.Equal(t, uint64(99), regs.uints[0]) }},
		{name: "complex", kind: registerComplex, value: reflect.ValueOf(complex128(1 + 2i)), check: func(t *testing.T, regs *Registers) { require.Equal(t, complex128(1+2i), regs.complex[0]) }},
		{name: "general", kind: registerGeneral, value: reflect.ValueOf("gen_val"), check: func(t *testing.T, regs *Registers) { require.Equal(t, "gen_val", regs.general[0].Interface()) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			regs := newRegisters(testRegCounts())
			assignReflectArg(&regs, tt.kind, 0, tt.value)
			tt.check(t, &regs)
		})
	}
}

func TestHandleOpResult(t *testing.T) {
	t.Parallel()

	t.Run("opContinue returns non-terminal", func(t *testing.T) {
		t.Parallel()
		vm := newTestVM(t)
		result, terminal, err := vm.handleOpResult(opContinue)
		require.Nil(t, result)
		require.False(t, terminal)
		require.NoError(t, err)
	})

	t.Run("opDone returns result and clears evalResult", func(t *testing.T) {
		t.Parallel()
		vm := newTestVM(t)
		vm.evalResult = "hello"
		result, terminal, err := vm.handleOpResult(opDone)
		require.Equal(t, "hello", result)
		require.True(t, terminal)
		require.NoError(t, err)
		require.Nil(t, vm.evalResult)
	})

	t.Run("opDivByZero", func(t *testing.T) {
		t.Parallel()
		vm := newTestVM(t)
		result, terminal, err := vm.handleOpResult(opDivByZero)
		require.Nil(t, result)
		require.True(t, terminal)
		require.ErrorIs(t, err, errDivisionByZero)
	})

	t.Run("opStackOverflow", func(t *testing.T) {
		t.Parallel()
		vm := newTestVM(t)
		result, terminal, err := vm.handleOpResult(opStackOverflow)
		require.Nil(t, result)
		require.True(t, terminal)
		require.ErrorIs(t, err, errStackOverflow)
	})

	t.Run("opPanicError returns error and clears evalError", func(t *testing.T) {
		t.Parallel()
		vm := newTestVM(t)
		testErr := errors.New("test panic")
		vm.evalError = testErr
		result, terminal, err := vm.handleOpResult(opPanicError)
		require.Nil(t, result)
		require.True(t, terminal)
		require.ErrorIs(t, err, testErr)
		require.Nil(t, vm.evalError)
	})

	t.Run("unknown opResult returns non-terminal", func(t *testing.T) {
		t.Parallel()
		vm := newTestVM(t)
		result, terminal, err := vm.handleOpResult(opResult(99))
		require.Nil(t, result)
		require.False(t, terminal)
		require.NoError(t, err)
	})
}

func TestCopyReturnValueAt(t *testing.T) {
	t.Parallel()

	t.Run("same kind int to int", func(t *testing.T) {
		t.Parallel()
		vm := newTestVM(t)
		vm.callStack = make([]callFrame, 2)
		vm.framePointer = 1
		vm.callStack[0].registers = newRegisters(testRegCounts())
		vm.callStack[1].registers = newRegisters(testRegCounts())
		vm.callStack[1].registers.ints[0] = 42

		vm.copyReturnValueAt(
			&vm.callStack[1],
			registerInt, 0,
			varLocation{register: 1, kind: registerInt},
		)
		require.Equal(t, int64(42), vm.callStack[0].registers.ints[1])
	})

	t.Run("same kind string to string", func(t *testing.T) {
		t.Parallel()
		vm := newTestVM(t)
		vm.callStack = make([]callFrame, 2)
		vm.framePointer = 1
		vm.callStack[0].registers = newRegisters(testRegCounts())
		vm.callStack[1].registers = newRegisters(testRegCounts())
		vm.callStack[1].registers.strings[0] = "hello"

		vm.copyReturnValueAt(
			&vm.callStack[1],
			registerString, 0,
			varLocation{register: 1, kind: registerString},
		)
		require.Equal(t, "hello", vm.callStack[0].registers.strings[1])
	})

	t.Run("general to scalar unpacks", func(t *testing.T) {
		t.Parallel()
		vm := newTestVM(t)
		vm.callStack = make([]callFrame, 2)
		vm.framePointer = 1
		vm.callStack[0].registers = newRegisters(testRegCounts())
		vm.callStack[1].registers = newRegisters(testRegCounts())
		vm.callStack[1].registers.general[0] = reflect.ValueOf(int64(42))

		vm.copyReturnValueAt(
			&vm.callStack[1],
			registerGeneral, 0,
			varLocation{register: 1, kind: registerInt},
		)
		require.Equal(t, int64(42), vm.callStack[0].registers.ints[1])
	})

	t.Run("scalar to general packs", func(t *testing.T) {
		t.Parallel()
		vm := newTestVM(t)
		vm.callStack = make([]callFrame, 2)
		vm.framePointer = 1
		vm.callStack[0].registers = newRegisters(testRegCounts())
		vm.callStack[1].registers = newRegisters(testRegCounts())
		vm.callStack[1].registers.ints[0] = 42

		vm.copyReturnValueAt(
			&vm.callStack[1],
			registerInt, 0,
			varLocation{register: 1, kind: registerGeneral},
		)
		require.Equal(t, int64(42), vm.callStack[0].registers.general[1].Interface())
	})

	t.Run("mismatch int to string is silent no-op", func(t *testing.T) {
		t.Parallel()
		vm := newTestVM(t)
		vm.callStack = make([]callFrame, 2)
		vm.framePointer = 1
		vm.callStack[0].registers = newRegisters(testRegCounts())
		vm.callStack[1].registers = newRegisters(testRegCounts())
		vm.callStack[0].registers.strings[1] = "original"
		vm.callStack[1].registers.ints[0] = 42

		vm.copyReturnValueAt(
			&vm.callStack[1],
			registerInt, 0,
			varLocation{register: 1, kind: registerString},
		)

		require.Equal(t, "original", vm.callStack[0].registers.strings[1])
	})

	t.Run("mismatch float to bool is silent no-op", func(t *testing.T) {
		t.Parallel()
		vm := newTestVM(t)
		vm.callStack = make([]callFrame, 2)
		vm.framePointer = 1
		vm.callStack[0].registers = newRegisters(testRegCounts())
		vm.callStack[1].registers = newRegisters(testRegCounts())
		vm.callStack[0].registers.bools[1] = true
		vm.callStack[1].registers.floats[0] = 3.14

		vm.copyReturnValueAt(
			&vm.callStack[1],
			registerFloat, 0,
			varLocation{register: 1, kind: registerBool},
		)
		require.Equal(t, true, vm.callStack[0].registers.bools[1])
	})
}

func TestReflectBinaryOpEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("uint addition", func(t *testing.T) {
		t.Parallel()
		a := reflect.ValueOf(uint64(10))
		b := reflect.ValueOf(uint64(32))
		result := reflectBinaryOp(a, b,
			func(x, y int64) int64 { return x + y },
			nil, nil,
		)
		require.True(t, result.IsValid())
		require.Equal(t, uint64(42), result.Interface())
	})

	t.Run("string concatenation with nil int and float ops", func(t *testing.T) {
		t.Parallel()
		a := reflect.ValueOf("hello ")
		b := reflect.ValueOf("world")
		result := reflectBinaryOp(a, b,
			nil, nil,
			func(x, y string) string { return x + y },
		)
		require.True(t, result.IsValid())
		require.Equal(t, "hello world", result.Interface())
	})

	t.Run("int with nil intOp returns invalid", func(t *testing.T) {
		t.Parallel()
		a := reflect.ValueOf(int64(10))
		b := reflect.ValueOf(int64(20))
		result := reflectBinaryOp(a, b, nil, nil, nil)
		require.False(t, result.IsValid())
	})

	t.Run("float with nil floatOp returns invalid", func(t *testing.T) {
		t.Parallel()
		a := reflect.ValueOf(float64(1.5))
		b := reflect.ValueOf(float64(2.5))
		result := reflectBinaryOp(a, b, nil, nil, nil)
		require.False(t, result.IsValid())
	})

	t.Run("unhandled kind returns invalid", func(t *testing.T) {
		t.Parallel()
		a := reflect.ValueOf(true)
		b := reflect.ValueOf(false)
		result := reflectBinaryOp(a, b,
			func(x, y int64) int64 { return x + y },
			func(x, y float64) float64 { return x + y },
			func(x, y string) string { return x + y },
		)
		require.False(t, result.IsValid())
	})
}
