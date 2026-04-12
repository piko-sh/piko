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
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDisassembleOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		compiledFunction func() *CompiledFunction
		contains         []string
	}{
		{name: "int_constant", compiledFunction: func() *CompiledFunction {
			b := newBytecodeBuilder()
			b.addIntConst(42)
			b.intRegisters(1)
			b.emit(opLoadIntConst, 0, 0, 0)
			b.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 1)
			b.returnInt()
			return b.build()
		}, contains: []string{"LOAD_INT_CONST", "ints[0] = 42"}},
		{name: "float_constant", compiledFunction: func() *CompiledFunction {
			b := newBytecodeBuilder()
			b.addFloatConst(3.14)
			b.floatRegisters(1)
			b.emit(opLoadFloatConst, 0, 0, 0)
			b.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 1)
			b.returnFloat()
			return b.build()
		}, contains: []string{"LOAD_FLOAT_CONST", "floats[0] = 3.14"}},
		{name: "string_constant", compiledFunction: func() *CompiledFunction {
			b := newBytecodeBuilder()
			b.addStringConst("hello")
			b.stringRegisters(1)
			b.emit(opLoadStringConst, 0, 0, 0)
			b.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 1)
			b.returnString()
			return b.build()
		}, contains: []string{"LOAD_STRING_CONST", "strings[0] = \"hello\""}},
		{name: "bool_constant", compiledFunction: func() *CompiledFunction {
			b := newBytecodeBuilder()
			b.addBoolConst(true)
			b.boolRegisters(1)
			b.emit(opLoadBoolConst, 0, 0, 0)
			b.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 1)
			b.returnBool()
			return b.build()
		}, contains: []string{"LOAD_BOOL_CONST", "bools[0] = true"}},
		{name: "load_bool_inline", compiledFunction: func() *CompiledFunction {
			b := newBytecodeBuilder()
			b.intRegisters(1)
			b.emit(opDrillTier1, uint8(subOpLoadBool), 0, 1)
			b.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 1)
			b.returnInt()
			return b.build()
		}, contains: []string{"LOAD_BOOL", "ints[0] = true"}},
		{name: "load_bool_false", compiledFunction: func() *CompiledFunction {
			b := newBytecodeBuilder()
			b.intRegisters(1)
			b.emit(opDrillTier1, uint8(subOpLoadBool), 0, 0)
			b.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 1)
			b.returnInt()
			return b.build()
		}, contains: []string{"LOAD_BOOL", "ints[0] = false"}},
		{name: "load_nil", compiledFunction: func() *CompiledFunction {
			b := newBytecodeBuilder()
			b.generalRegisters(1)
			b.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2LoadNil), 0)
			b.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 1)
			b.returnGeneral()
			return b.build()
		}, contains: []string{"TIER2_LOAD_NIL"}},
		{name: "load_int_const_small", compiledFunction: func() *CompiledFunction {
			b := newBytecodeBuilder()
			b.intRegisters(1)
			b.emit(opDrillTier1, uint8(subOpLoadIntConstSmall), 0, 7)
			b.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 1)
			b.returnInt()
			return b.build()
		}, contains: []string{"LOAD_INT_CONST_SMALL", "ints[0] = 7"}},
		{name: "jump_comment", compiledFunction: func() *CompiledFunction {
			b := newBytecodeBuilder()
			b.intRegisters(1)
			b.emitJump(opDrillTier1, uint8(subOpJump), 2)
			b.emit(opNop, 0, 0, 0)
			b.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 1)
			b.returnInt()
			return b.build()
		}, contains: []string{"JUMP", "goto"}},
		{name: "return_comment", compiledFunction: func() *CompiledFunction {
			b := newBytecodeBuilder()
			b.intRegisters(1)
			b.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 2)
			return b.build()
		}, contains: []string{"RETURN", "2 values"}},
		{name: "return_void_comment", compiledFunction: func() *CompiledFunction {
			b := newBytecodeBuilder()
			b.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2DrillTier3), uint8(subOpTier3ReturnVoid))
			return b.build()
		}, contains: []string{"TIER3_RETURN_VOID"}},
		{name: "add_int_const_comment", compiledFunction: func() *CompiledFunction {
			b := newBytecodeBuilder()
			b.addIntConst(5)
			b.intRegisters(2)
			b.emit(opAddIntConst, 0, 1, 0)
			b.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 1)
			b.returnInt()
			return b.build()
		}, contains: []string{"ADD_INT_CONST", "const = 5"}},
		{name: "long_string_truncated", compiledFunction: func() *CompiledFunction {
			b := newBytecodeBuilder()
			b.addStringConst("this is a very long string that should be truncated in disassembly output")
			b.stringRegisters(1)
			b.emit(opLoadStringConst, 0, 0, 0)
			b.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 1)
			b.returnString()
			return b.build()
		}, contains: []string{"...", "LOAD_STRING_CONST"}},
		{name: "empty_body", compiledFunction: func() *CompiledFunction {
			b := newBytecodeBuilder()
			return b.build()
		}, contains: nil},
		{name: "jump_if_true", compiledFunction: func() *CompiledFunction {
			b := newBytecodeBuilder()
			b.intRegisters(1)
			b.emitJump(opJumpIfTrue, 0, 1)
			b.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 1)
			b.returnInt()
			return b.build()
		}, contains: []string{"JUMP_IF_TRUE", "if ints[0] != 0 goto"}},
		{name: "jump_if_false", compiledFunction: func() *CompiledFunction {
			b := newBytecodeBuilder()
			b.intRegisters(1)
			b.emitJump(opJumpIfFalse, 0, 1)
			b.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 1)
			b.returnInt()
			return b.build()
		}, contains: []string{"JUMP_IF_FALSE", "if ints[0] == 0 goto"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			compiledFunction := tt.compiledFunction()
			output := compiledFunction.Disassemble()
			if tt.contains == nil {
				require.Empty(t, output)
				return
			}
			for _, substr := range tt.contains {
				require.True(t, strings.Contains(output, substr),
					"disassembly should contain %q, got:\n%s", substr, output)
			}
		})
	}
}

func TestDisassembleRange(t *testing.T) {
	t.Parallel()

	b := newBytecodeBuilder()
	b.addIntConst(1)
	b.addIntConst(2)
	b.intRegisters(2)
	b.emit(opLoadIntConst, 0, 0, 0)
	b.emit(opLoadIntConst, 1, 0, 1)
	b.emit(opAddInt, 0, 0, 1)
	b.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 1)
	b.returnInt()
	compiledFunction := b.build()

	tests := []struct {
		name  string
		start int
		end   int
		lines int
	}{
		{name: "full_range", start: 0, end: 4, lines: 4},
		{name: "partial", start: 1, end: 3, lines: 2},
		{name: "single", start: 0, end: 1, lines: 1},
		{name: "clamped_start", start: -5, end: 2, lines: 2},
		{name: "clamped_end", start: 0, end: 100, lines: 4},
		{name: "empty_start_ge_end", start: 3, end: 2, lines: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			output := compiledFunction.DisassembleRange(tt.start, tt.end)
			if tt.lines == 0 {
				require.Empty(t, output)
			} else {
				lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
				require.Equal(t, tt.lines, len(lines))
			}
		})
	}
}

func TestServiceCompileErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		code string
	}{
		{name: "undefined_variable", code: `x`},
		{name: "type_mismatch", code: `var x int = "hello"`},
		{name: "syntax_error", code: `func {`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			_, err := service.Eval(context.Background(), tt.code)
			require.Error(t, err)
		})
	}
}
