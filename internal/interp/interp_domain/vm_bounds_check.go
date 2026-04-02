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
	"reflect"
	"strings"
)

const (
	// boundsTableIntConstant is the table name for integer constants
	// in bounds-check error messages.
	boundsTableIntConstant = "int constant"

	// boundsTableFloatConstant is the table name for float constants
	// in bounds-check error messages.
	boundsTableFloatConstant = "float constant"

	// boundsTableStringConstant is the table name for string
	// constants in bounds-check error messages.
	boundsTableStringConstant = "string constant"

	// boundsTableGeneralConstant is the table name for general
	// constants in bounds-check error messages.
	boundsTableGeneralConstant = "general constant"

	// boundsTableBoolConstant is the table name for boolean
	// constants in bounds-check error messages.
	boundsTableBoolConstant = "bool constant"

	// boundsTableUintConstant is the table name for unsigned integer
	// constants in bounds-check error messages.
	boundsTableUintConstant = "uint constant"

	// boundsTableComplexConstant is the table name for complex number
	// constants in bounds-check error messages.
	boundsTableComplexConstant = "complex constant"

	// boundsTableFunction is the table name for functions in
	// bounds-check error messages.
	boundsTableFunction = "function"

	// boundsTableTypeTable is the table name for type tables in
	// bounds-check error messages.
	boundsTableTypeTable = "type table"

	// boundsTableCallSite is the table name for call sites in
	// bounds-check error messages.
	boundsTableCallSite = "call site"

	// registerRoleMap is the role name for map registers in diagnostic messages.
	registerRoleMap = "map"

	// registerRoleStruct is the role name for struct registers in diagnostic messages.
	registerRoleStruct = "struct"
)

// vmBoundsErr is a structured VM bounds-check error that keeps the Error()
// message low-cardinality for log aggregation while preserving detailed
// diagnostic information via DiagnosticDetail().
type vmBoundsErr struct {
	// tableName identifies the kind of table that was accessed out of range.
	tableName string

	// funcName holds the name of the function where the error occurred.
	funcName string

	// index holds the out-of-range index that was requested.
	index int

	// tableSize holds the actual size of the table.
	tableSize int

	// pc holds the program counter at the point of the error.
	pc int
}

// Error returns a low-cardinality error message suitable for log aggregation.
//
// Returns string containing the table name and "index out of range".
func (e *vmBoundsErr) Error() string {
	return fmt.Sprintf("%s index out of range", e.tableName)
}

// DiagnosticDetail returns the full diagnostic context for debugging,
// including the index, table size, program counter, and function name.
//
// Returns string containing the formatted diagnostic fields.
func (e *vmBoundsErr) DiagnosticDetail() string {
	return fmt.Sprintf(
		"index=%d tableSize=%d pc=%d funcName=%s",
		e.index, e.tableSize, e.pc, e.funcName,
	)
}

// vmBoundsError sets a diagnostic error on the VM when a
// bytecode-referenced table index is out of range.
//
// Takes vm (*VM) which is the virtual machine to set the error on.
// Takes frame (*callFrame) which provides the current program
// counter and function name.
// Takes tableName (string) which identifies the table that was
// accessed out of range.
// Takes index (int) which is the requested index.
// Takes tableSize (int) which is the actual size of the table.
func vmBoundsError(vm *VM, frame *callFrame, tableName string, index int, tableSize int) {
	vm.evalError = &vmBoundsErr{
		tableName: tableName,
		index:     index,
		tableSize: tableSize,
		pc:        frame.programCounter,
		funcName:  frame.function.name,
	}
}

// vmDiagnosticContext generates rich diagnostic context for VM
// panics, including disassembled bytecode and nearby registers.
//
// Takes frame (*callFrame) which provides the current program
// counter and function bytecode.
// Takes registers (*Registers) which provides the register file
// to inspect.
// Takes focusRegister (int) which is the register index to
// centre the diagnostic output around.
//
// Returns a multi-line string with disassembled bytecode around
// the current program counter and the types of nearby general
// registers.
func vmDiagnosticContext(frame *callFrame, registers *Registers, focusRegister int) string {
	var b strings.Builder

	pc := frame.programCounter
	start := max(pc-6, 0)
	end := pc + 2
	b.WriteString("bytecode around pc:\n")
	b.WriteString(frame.function.DisassembleRange(start, end))
	b.WriteByte('\n')

	regBase := max(focusRegister-3, 0)
	regEnd := min(focusRegister+4, len(registers.general))
	b.WriteString("nearby registers:\n")
	for i := regBase; i < regEnd; i++ {
		v := registers.general[i]
		if v.IsValid() {
			fmt.Fprintf(&b, "  general[%d]: %v (%s)\n", i, v.Type(), v.Kind())
		} else {
			fmt.Fprintf(&b, "  general[%d]: <zero>\n", i)
		}
	}

	return b.String()
}

// vmCallSiteDiagnostic generates diagnostic context specific to a
// native call site, including argument and return register mappings.
//
// Takes frame (*callFrame) which provides the current program
// counter and function bytecode for inspecting preceding
// instructions.
// Takes site (*callSite) which is the call site to diagnose.
//
// Returns a multi-line string with the current site's return and
// argument mappings, plus the preceding CALL_NATIVE site if
// present.
func vmCallSiteDiagnostic(frame *callFrame, site *callSite) string {
	var b strings.Builder

	b.WriteString("current site returns:\n")
	for i, ret := range site.returns {
		fmt.Fprintf(&b, "  returns[%d]: kind=%d register=%d\n", i, ret.kind, ret.register)
	}
	b.WriteString("current site args:\n")
	for i, arg := range site.arguments {
		fmt.Fprintf(&b, "  args[%d]: kind=%d register=%d\n", i, arg.kind, arg.register)
	}

	vmDumpPrecedingCallNative(&b, frame)

	return b.String()
}

// vmDumpPrecedingCallNative appends diagnostic information about
// the preceding instruction when it is a CALL_NATIVE.
//
// Takes b (*strings.Builder) which is the buffer to append
// diagnostic output to.
// Takes frame (*callFrame) which provides the current program
// counter and function bytecode.
func vmDumpPrecedingCallNative(b *strings.Builder, frame *callFrame) {
	pc := frame.programCounter
	if pc < 2 {
		return
	}
	prevInstr := frame.function.body[pc-2]
	if prevInstr.op != opCallNative {
		return
	}
	prevSiteIndex := prevInstr.wideIndex()
	if int(prevSiteIndex) >= len(frame.function.callSites) {
		return
	}
	prevSite := &frame.function.callSites[prevSiteIndex]
	fmt.Fprintf(b,
		"preceding CALL_NATIVE (pc-2) site %d: nativeRegister=%d isMethod=%v methodRecvReg=%d args=%d returns=%d\n",
		prevSiteIndex, prevSite.nativeRegister,
		prevSite.isMethod, prevSite.methodRecvReg,
		len(prevSite.arguments), len(prevSite.returns),
	)
	for i, ret := range prevSite.returns {
		fmt.Fprintf(b, "  prev.returns[%d]: kind=%d register=%d\n", i, ret.kind, ret.register)
	}
	for i, arg := range prevSite.arguments {
		fmt.Fprintf(b, "  prev.args[%d]: kind=%d register=%d\n", i, arg.kind, arg.register)
	}
}

// vmPanicInvalidRegister panics with a diagnostic message when a
// VM handler encounters a zero reflect.Value in a general register.
//
// Takes handler (string) which is the name of the VM handler
// that detected the error.
// Takes registerRole (string) which describes the role of the
// register (e.g. "map" or "struct").
// Takes registerIndex (uint8) which is the index of the invalid
// register.
// Takes inst (instruction) which is the current instruction.
// Takes frame (*callFrame) which provides the current program
// counter and function name.
// Takes registers (*Registers) which provides the register file
// for diagnostic context.
//
// Panics unconditionally with a formatted diagnostic message.
func vmPanicInvalidRegister(handler string, registerRole string, registerIndex uint8, inst instruction, frame *callFrame, registers *Registers) {
	panic(fmt.Sprintf(
		"interp: %s - general[%d] (%s) is zero reflect.Value; "+
			"pc=%d funcName=%s; registers: a=%d b=%d c=%d\n%s",
		handler, registerIndex, registerRole,
		frame.programCounter, frame.function.name,
		inst.a, inst.b, inst.c,
		vmDiagnosticContext(frame, registers, int(registerIndex)),
	))
}

// vmPanicNotStruct panics with a diagnostic message when a VM
// handler expects a struct but finds a different kind.
//
// Takes handler (string) which is the name of the VM handler
// that detected the error.
// Takes registerIndex (uint8) which is the index of the register
// containing the non-struct value.
// Takes actual (reflect.Kind) which is the kind that was found
// instead of struct.
// Takes inst (instruction) which is the current instruction.
// Takes frame (*callFrame) which provides the current program
// counter and function name.
// Takes registers (*Registers) which provides the register file
// for diagnostic context.
//
// Panics unconditionally with a formatted diagnostic message.
func vmPanicNotStruct(handler string, registerIndex uint8, actual reflect.Kind, inst instruction, frame *callFrame, registers *Registers) {
	panic(fmt.Sprintf(
		"interp: %s - general[%d] is %v, expected struct; "+
			"pc=%d funcName=%s; registers: a=%d b=%d c=%d\n%s",
		handler, registerIndex, actual,
		frame.programCounter, frame.function.name,
		inst.a, inst.b, inst.c,
		vmDiagnosticContext(frame, registers, int(registerIndex)),
	))
}

// vmPanicFieldIndex panics with a diagnostic message when a struct
// field index is out of range.
//
// Takes handler (string) which is the name of the VM handler
// that detected the error.
// Takes structType (reflect.Type) which is the type of the
// struct being accessed.
// Takes fieldIndex (uint8) which is the out-of-range field
// index.
// Takes inst (instruction) which is the current instruction.
// Takes frame (*callFrame) which provides the current program
// counter and function name.
// Takes registers (*Registers) which provides the register file
// for diagnostic context.
//
// Panics unconditionally with a formatted diagnostic message.
func vmPanicFieldIndex(handler string, structType reflect.Type, fieldIndex uint8, inst instruction, frame *callFrame, registers *Registers) {
	panic(fmt.Sprintf(
		"interp: %s - field index %d out of range for struct %v (has %d fields); "+
			"pc=%d funcName=%s; registers: a=%d b=%d c=%d\n%s",
		handler, fieldIndex, structType, structType.NumField(),
		frame.programCounter, frame.function.name,
		inst.a, inst.b, inst.c,
		vmDiagnosticContext(frame, registers, int(fieldIndex)),
	))
}

// vmPanicTypeMismatch panics with a diagnostic message when a Set
// operation would fail due to incompatible types.
//
// Takes handler (string) which is the name of the VM handler
// that detected the error.
// Takes expected (reflect.Type) which is the type that was
// expected.
// Takes actual (reflect.Type) which is the type that was found.
// Takes inst (instruction) which is the current instruction.
// Takes frame (*callFrame) which provides the current program
// counter and function name.
// Takes registers (*Registers) which provides the register file
// for diagnostic context.
//
// Panics unconditionally with a formatted diagnostic message.
func vmPanicTypeMismatch(handler string, expected, actual reflect.Type, inst instruction, frame *callFrame, registers *Registers) {
	panic(fmt.Sprintf(
		"interp: %s type mismatch - expected %v, got %v; "+
			"pc=%d funcName=%s; registers: a=%d b=%d c=%d\n%s",
		handler, expected, actual,
		frame.programCounter, frame.function.name,
		inst.a, inst.b, inst.c,
		vmDiagnosticContext(frame, registers, int(inst.a)),
	))
}
