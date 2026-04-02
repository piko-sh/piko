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
		{"int_constant", func() *CompiledFunction {
			b := newBytecodeBuilder()
			b.addIntConst(42)
			b.intRegisters(1)
			b.emit(opLoadIntConst, 0, 0, 0)
			b.emit(opReturn, 1, 0, 0)
			b.returnInt()
			return b.build()
		}, []string{"LOAD_INT_CONST", "ints[0] = 42"}},
		{"float_constant", func() *CompiledFunction {
			b := newBytecodeBuilder()
			b.addFloatConst(3.14)
			b.floatRegisters(1)
			b.emit(opLoadFloatConst, 0, 0, 0)
			b.emit(opReturn, 1, 0, 0)
			b.returnFloat()
			return b.build()
		}, []string{"LOAD_FLOAT_CONST", "floats[0] = 3.14"}},
		{"string_constant", func() *CompiledFunction {
			b := newBytecodeBuilder()
			b.addStringConst("hello")
			b.stringRegisters(1)
			b.emit(opLoadStringConst, 0, 0, 0)
			b.emit(opReturn, 1, 0, 0)
			b.returnString()
			return b.build()
		}, []string{"LOAD_STRING_CONST", "strings[0] = \"hello\""}},
		{"bool_constant", func() *CompiledFunction {
			b := newBytecodeBuilder()
			b.addBoolConst(true)
			b.boolRegisters(1)
			b.emit(opLoadBoolConst, 0, 0, 0)
			b.emit(opReturn, 1, 0, 0)
			b.returnBool()
			return b.build()
		}, []string{"LOAD_BOOL_CONST", "bools[0] = true"}},
		{"load_bool_inline", func() *CompiledFunction {
			b := newBytecodeBuilder()
			b.intRegisters(1)
			b.emit(opLoadBool, 0, 1, 0)
			b.emit(opReturn, 1, 0, 0)
			b.returnInt()
			return b.build()
		}, []string{"LOAD_BOOL", "ints[0] = true"}},
		{"load_bool_false", func() *CompiledFunction {
			b := newBytecodeBuilder()
			b.intRegisters(1)
			b.emit(opLoadBool, 0, 0, 0)
			b.emit(opReturn, 1, 0, 0)
			b.returnInt()
			return b.build()
		}, []string{"LOAD_BOOL", "ints[0] = false"}},
		{"load_nil", func() *CompiledFunction {
			b := newBytecodeBuilder()
			b.generalRegisters(1)
			b.emit(opLoadNil, 0, 0, 0)
			b.emit(opReturn, 1, 0, 0)
			b.returnGeneral()
			return b.build()
		}, []string{"LOAD_NIL", "general[0] = nil"}},
		{"load_int_const_small", func() *CompiledFunction {
			b := newBytecodeBuilder()
			b.intRegisters(1)
			b.emit(opLoadIntConstSmall, 0, 7, 0)
			b.emit(opReturn, 1, 0, 0)
			b.returnInt()
			return b.build()
		}, []string{"LOAD_INT_CONST_SMALL", "ints[0] = 7"}},
		{"jump_comment", func() *CompiledFunction {
			b := newBytecodeBuilder()
			b.intRegisters(1)
			b.emitJump(opJump, 0, 2)
			b.emit(opNop, 0, 0, 0)
			b.emit(opReturn, 1, 0, 0)
			b.returnInt()
			return b.build()
		}, []string{"JUMP", "goto"}},
		{"return_comment", func() *CompiledFunction {
			b := newBytecodeBuilder()
			b.intRegisters(1)
			b.emit(opReturn, 2, 0, 0)
			return b.build()
		}, []string{"RETURN", "2 values"}},
		{"return_void_comment", func() *CompiledFunction {
			b := newBytecodeBuilder()
			b.emit(opReturnVoid, 0, 0, 0)
			return b.build()
		}, []string{"RETURN_VOID", "void"}},
		{"add_int_const_comment", func() *CompiledFunction {
			b := newBytecodeBuilder()
			b.addIntConst(5)
			b.intRegisters(2)
			b.emit(opAddIntConst, 0, 1, 0)
			b.emit(opReturn, 1, 0, 0)
			b.returnInt()
			return b.build()
		}, []string{"ADD_INT_CONST", "const = 5"}},
		{"long_string_truncated", func() *CompiledFunction {
			b := newBytecodeBuilder()
			b.addStringConst("this is a very long string that should be truncated in disassembly output")
			b.stringRegisters(1)
			b.emit(opLoadStringConst, 0, 0, 0)
			b.emit(opReturn, 1, 0, 0)
			b.returnString()
			return b.build()
		}, []string{"...", "LOAD_STRING_CONST"}},
		{"empty_body", func() *CompiledFunction {
			b := newBytecodeBuilder()
			return b.build()
		}, nil},
		{"jump_if_true", func() *CompiledFunction {
			b := newBytecodeBuilder()
			b.intRegisters(1)
			b.emitJump(opJumpIfTrue, 0, 1)
			b.emit(opReturn, 1, 0, 0)
			b.returnInt()
			return b.build()
		}, []string{"JUMP_IF_TRUE", "if ints[0] != 0 goto"}},
		{"jump_if_false", func() *CompiledFunction {
			b := newBytecodeBuilder()
			b.intRegisters(1)
			b.emitJump(opJumpIfFalse, 0, 1)
			b.emit(opReturn, 1, 0, 0)
			b.returnInt()
			return b.build()
		}, []string{"JUMP_IF_FALSE", "if ints[0] == 0 goto"}},
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
	b.emit(opReturn, 1, 0, 0)
	b.returnInt()
	compiledFunction := b.build()

	tests := []struct {
		name  string
		start int
		end   int
		lines int
	}{
		{"full_range", 0, 4, 4},
		{"partial", 1, 3, 2},
		{"single", 0, 1, 1},
		{"clamped_start", -5, 2, 2},
		{"clamped_end", 0, 100, 4},
		{"empty_start_ge_end", 3, 2, 0},
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
		{"undefined_variable", `x`},
		{"type_mismatch", `var x int = "hello"`},
		{"syntax_error", `func {`},
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
