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
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func newBoundsCheckFrame(t *testing.T) *callFrame {
	t.Helper()
	cf := &CompiledFunction{
		name: "TestFunc",
		body: []instruction{
			makeInstruction(opNop, 0, 0, 0),
			makeInstruction(opNop, 0, 0, 0),
			makeInstruction(opNop, 0, 0, 0),
			makeInstruction(opNop, 0, 0, 0),
		},
	}
	return &callFrame{
		function:       cf,
		programCounter: 2,
	}
}

func TestVMBoundsErr_Error(t *testing.T) {
	t.Parallel()
	err := &vmBoundsErr{tableName: "intConstants", index: 7, tableSize: 3}
	require.Equal(t, "intConstants index out of range", err.Error())
}

func TestVMBoundsErr_DiagnosticDetail(t *testing.T) {
	t.Parallel()
	err := &vmBoundsErr{
		tableName: "stringConstants",
		index:     12,
		tableSize: 4,
		pc:        99,
		funcName:  "calc",
	}
	got := err.DiagnosticDetail()
	require.Contains(t, got, "index=12")
	require.Contains(t, got, "tableSize=4")
	require.Contains(t, got, "pc=99")
	require.Contains(t, got, "funcName=calc")
}

func TestVMBoundsError_PopulatesVMEvalError(t *testing.T) {
	t.Parallel()
	vm := newTestVM(t)
	frame := newBoundsCheckFrame(t)

	vmBoundsError(vm, frame, "boolConstants", 5, 2)

	require.Error(t, vm.evalError)
	var boundsErr *vmBoundsErr
	ok := errors.As(vm.evalError, &boundsErr)
	require.True(t, ok, "expected *vmBoundsErr, got %T", vm.evalError)
	require.Equal(t, "boolConstants", boundsErr.tableName)
	require.Equal(t, 5, boundsErr.index)
	require.Equal(t, 2, boundsErr.tableSize)
	require.Equal(t, frame.programCounter, boundsErr.pc)
	require.Equal(t, frame.function.name, boundsErr.funcName)
}

func TestVMDiagnosticContext_RendersBytecodeAndRegisters(t *testing.T) {
	t.Parallel()
	frame := newBoundsCheckFrame(t)
	registers := &Registers{
		general: make([]reflect.Value, 8),
	}
	registers.general[3] = reflect.ValueOf("hello")

	got := vmDiagnosticContext(frame, registers, 3)

	require.Contains(t, got, "bytecode around pc:")
	require.Contains(t, got, "nearby registers:")
	require.Contains(t, got, "general[3]: string")
}

func TestVMDiagnosticContext_ZeroValueRegistersAnnotated(t *testing.T) {
	t.Parallel()
	frame := newBoundsCheckFrame(t)
	registers := &Registers{general: make([]reflect.Value, 4)}

	got := vmDiagnosticContext(frame, registers, 2)
	require.Contains(t, got, "general[")
	require.Contains(t, got, "<zero>")
}

func TestVMCallSiteDiagnostic_RendersArgsAndReturns(t *testing.T) {
	t.Parallel()
	frame := newBoundsCheckFrame(t)
	site := &callSite{
		arguments: []varLocation{
			{register: 1, kind: registerInt},
			{register: 2, kind: registerString},
		},
		returns: []varLocation{
			{register: 3, kind: registerGeneral},
		},
	}
	got := vmCallSiteDiagnostic(frame, site)
	require.NotEmpty(t, got)
}

func TestVMCallSiteDiagnostic_NoArgsCompact(t *testing.T) {
	t.Parallel()
	frame := newBoundsCheckFrame(t)
	site := &callSite{}
	got := vmCallSiteDiagnostic(frame, site)
	_ = got
}

func TestVMPanicInvalidRegister_PanicsWithDetail(t *testing.T) {
	t.Parallel()
	frame := newBoundsCheckFrame(t)
	registers := &Registers{general: make([]reflect.Value, 4)}
	inst := makeInstruction(opNop, 0, 0, 0)

	defer func() {
		recovered := recover()
		require.NotNil(t, recovered)
		message, ok := recovered.(string)
		require.True(t, ok, "expected string panic, got %T", recovered)
		require.True(t, strings.Contains(message, "handler") || strings.Contains(message, "register"),
			"panic message should mention handler/register, got %q", message)
	}()

	vmPanicInvalidRegister("testHandler", "receiver", 9, inst, frame, registers)
}

func TestVMPanicNotStruct_PanicsWithDetail(t *testing.T) {
	t.Parallel()
	frame := newBoundsCheckFrame(t)
	registers := &Registers{general: make([]reflect.Value, 4)}
	inst := makeInstruction(opNop, 0, 0, 0)

	defer func() {
		recovered := recover()
		require.NotNil(t, recovered)
	}()

	vmPanicNotStruct("testHandler", 1, reflect.Int, inst, frame, registers)
}

func TestVMPanicFieldIndex_PanicsWithDetail(t *testing.T) {
	t.Parallel()
	frame := newBoundsCheckFrame(t)
	registers := &Registers{general: make([]reflect.Value, 4)}
	inst := makeInstruction(opNop, 0, 0, 0)

	structType := reflect.TypeOf(struct{ A int }{})

	defer func() {
		recovered := recover()
		require.NotNil(t, recovered)
	}()

	vmPanicFieldIndex("testHandler", structType, 99, inst, frame, registers)
}

func TestVMPanicTypeMismatch_PanicsWithDetail(t *testing.T) {
	t.Parallel()
	frame := newBoundsCheckFrame(t)
	registers := &Registers{general: make([]reflect.Value, 4)}
	inst := makeInstruction(opNop, 0, 0, 0)

	defer func() {
		recovered := recover()
		require.NotNil(t, recovered)
	}()

	vmPanicTypeMismatch(
		"testHandler",
		reflect.TypeFor[string](),
		reflect.TypeFor[int](),
		inst,
		frame,
		registers,
	)
}
