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
)

var (
	// errDivisionByZero is returned when an integer division or remainder operation has a
	// zero divisor.
	errDivisionByZero = errors.New("division by zero")

	// errStackOverflow is returned when the call stack exceeds the maximum depth, indicating
	// infinite recursion or excessively deep call chains.
	errStackOverflow = errors.New("stack overflow")

	// errIndexOutOfRange is returned when a slice, array, or string index is outside the
	// valid range [0, len).
	errIndexOutOfRange = errors.New("index out of range")

	// errNilPointerIndex is returned when an index operation is applied to a nil pointer
	// (e.g. indexing a nil *[N]T).
	errNilPointerIndex = errors.New("index of nil pointer")

	// errSliceOutOfRange is returned when slice bounds are outside the valid range or low
	// exceeds high.
	errSliceOutOfRange = errors.New("slice bounds out of range")

	// errInvalidOpcode is returned when the VM encounters an unrecognised opcode during
	// execution.
	errInvalidOpcode = errors.New("invalid opcode")

	// errCompilation is returned when the source code fails to compile. The underlying error
	// provides specific parsing or type-checking details.
	errCompilation = errors.New("compilation failed")

	// errTypeCheck is returned when go/types rejects the source code.
	errTypeCheck = errors.New("type check failed")

	// errCompilePanic is returned when type-checking or compilation panics. A panic here is
	// an internal interpreter defect, never valid user input; it is recovered and surfaced
	// as an ordinary compile error so a single bad package cannot crash the host.
	errCompilePanic = errors.New("internal compiler panic")

	// errParse is returned when go/parser rejects the source code.
	errParse = errors.New("parse failed")

	// errEntrypointNotFound is returned when the requested entrypoint function does not
	// exist in the compiled file set.
	errEntrypointNotFound = errors.New("entrypoint not found")

	// errCyclicImport is returned when the import graph contains a cycle, which is illegal
	// in Go.
	errCyclicImport = errors.New("cyclic import detected")

	// errExecutionCancelled is returned when the execution context is cancelled or its
	// deadline is exceeded during evaluation.
	errExecutionCancelled = errors.New("execution cancelled")

	// errAllocationLimit is returned when a single allocation (make slice, make chan,
	// unsafe.String, unsafe.Slice) exceeds the configured maximum size.
	errAllocationLimit = errors.New("allocation size limit exceeded")

	// errGoroutineLimit is returned when the number of goroutines spawned by interpreted
	// code exceeds the configured limit.
	errGoroutineLimit = errors.New("goroutine limit exceeded")

	// errOutputLimit is returned when print/println output exceeds the configured maximum
	// size.
	errOutputLimit = errors.New("output size limit exceeded")

	// errNoBytecodeStore is returned when bytecode save/load is attempted without a
	// configured BytecodeStorePort.
	errNoBytecodeStore = errors.New("no bytecode store configured")

	// errFeatureNotAllowed is returned when compiled code uses a language feature that has
	// been disabled via WithFeatures.
	errFeatureNotAllowed = errors.New("language feature not allowed")

	// errCostBudgetExceeded is returned when the runtime cost of executing code exceeds the
	// budget set via WithCostBudget.
	errCostBudgetExceeded = errors.New("cost budget exceeded")

	// errSourceSizeLimit is returned when the total source code size exceeds the configured
	// maximum set via WithMaxSourceSize.
	errSourceSizeLimit = errors.New("source size limit exceeded")

	// errStringLimit is returned when a string concatenation would produce a result
	// exceeding the configured maximum set via WithMaxStringSize.
	errStringLimit = errors.New("string size limit exceeded")

	// errLiteralElementLimit is returned when a composite literal (slice, array, map) has
	// more elements than the configured maximum set via WithMaxLiteralElements.
	errLiteralElementLimit = errors.New("literal element count limit exceeded")

	// errPackageNotInRegistry is returned when the go/types Importer cannot locate an
	// interpreted package in the symbol registry. Typically means a transitive dependency is
	// missing from piko-symbols.yaml.
	errPackageNotInRegistry = errors.New("package not registered with interpreter")

	// errLinkedSiblingPanic wraps any panic recovered while invoking a //piko:link-routed
	// sibling. The usual cause is a stale generated symbol file whose descriptors no longer
	// match the sibling's real signature.
	errLinkedSiblingPanic = errors.New("linked sibling invocation panicked")

	// errLinkedSiblingShapeMismatch is returned when argument arity or kind checks on a
	// //piko:link sibling fail before the call is attempted, so the VM reports a structured
	// error instead of delegating to reflect.Call's panic.
	errLinkedSiblingShapeMismatch = errors.New("linked sibling signature does not match call site")

	// errNativeCallPanic wraps any panic recovered while invoking a registered native
	// function via reflect.Call.
	errNativeCallPanic = errors.New("native call panicked")

	// errGoexit signals that interpreted code invoked runtime.Goexit.
	//
	// The interpreter unwinds the current VM's frame stack running interpreted defers, then
	// surfaces this sentinel to the caller. The host Go goroutine running the VM exits via
	// normal return; the real runtime.Goexit is never invoked.
	errGoexit = errors.New("runtime.Goexit")

	// errBlockingOpUnblocked is surfaced when a blocking channel or select operation woke
	// through its context-cancellation or goroutine-panic wake arm but neither cause could
	// be recovered. It should not occur in practice; it guards against silently continuing
	// with an undefined value after an unexplained wake.
	errBlockingOpUnblocked = errors.New("blocking operation unblocked without a cause")

	// errCorruptTypeDescriptor is returned when a type descriptor is invalid.
	//
	// Surfaces when reconstructing a type from a serialised descriptor encounters a
	// structurally invalid payload, such as a container kind (slice, array, map, pointer,
	// channel) whose required element, key, or value sub-descriptor is absent. A tampered or
	// truncated bytecode payload can produce this; returning it keeps the load path from
	// dereferencing a nil sub-descriptor and crashing the host.
	errCorruptTypeDescriptor = errors.New("corrupt type descriptor")

	// errCompileJumpRange is returned when a jump offset overflows.
	//
	// Surfaces when a function body grows so large that a relative jump offset no longer
	// fits the signed 16-bit encoding. A single very large function (a huge switch, or
	// thousands of statements in one body) can trigger it; surfacing a compile error keeps
	// the compiler from panicking and crashing the host.
	errCompileJumpRange = errors.New("compiled function jump offset exceeds int16 range")

	// errCompileDepthLimit is returned when a recursive compile path (expressions,
	// statements, import topological sort) exceeds the configured maximum depth, defending
	// the host against stack exhaustion from pathological user input.
	errCompileDepthLimit = errors.New("compile recursion depth exceeded")

	// ErrConstantPoolExhausted is returned when a per-function constant pool (int, float,
	// string, bool, uint, complex, general, type or call-site table) would exceed the
	// configured maximum capacity, defending the host against pathological input that drives
	// unbounded pool growth.
	ErrConstantPoolExhausted = errors.New("constant pool exhausted")

	// ErrSpecialisationLimitReached is returned when registering a new generic-function
	// specialisation would exceed the configured maximum. Callers should fall back to the
	// generic reflect path.
	ErrSpecialisationLimitReached = errors.New("specialisation limit reached")

	// ErrMethodTableExhausted is returned when registering a new entry in
	// CompiledFunction.methodTable would exceed the configured maximum. Defends against
	// pathological method declarations.
	ErrMethodTableExhausted = errors.New("method table exhausted")

	// ErrVerifierIterationLimitExceeded is returned when the bytecode verifier's dataflow
	// worklist exceeds the configured iteration cap, defending the host against pathological
	// control-flow graphs crafted to force runaway worklist re-enqueues.
	ErrVerifierIterationLimitExceeded = errors.New("bytecode verifier iteration limit exceeded")

	// errArenaBudgetExceeded is returned when a register-arena grow would push the
	// cumulative bytes allocated within a single Execute past the configured per-execution
	// budget.
	errArenaBudgetExceeded = errors.New("register arena byte budget exceeded")

	// errSpillAreaExhausted is returned (via panic recovered by the compile-time recover)
	// when a per-bank spill area would exceed the uint16 slot-index limit; surfacing the
	// failure prevents the uint16 cast from silently wrapping and aliasing two logical slots
	// onto the same runtime register.
	errSpillAreaExhausted = errors.New("spill area exhausted")

	// errDeferStackExhausted is returned when the per-VM defer stack would exceed
	// maxDeferStackSize, defending the host against `for { defer ... }` style runaway
	// accumulation.
	errDeferStackExhausted = errors.New("defer stack exhausted")

	// ErrCompileTypeConversionArgCount is returned when a type conversion is invoked with
	// anything other than exactly one argument.
	ErrCompileTypeConversionArgCount = errors.New("type conversion requires exactly 1 argument")

	// ErrCompileDereferenceRequiresPointer is returned when the unary * operator is applied
	// to an operand whose register kind is not general (i.e. not a boxed pointer).
	ErrCompileDereferenceRequiresPointer = errors.New("dereference requires pointer in general register")

	// ErrCompileDereferenceAssignRequiresPointer is returned when an assignment through a
	// pointer dereference (*p = v) targets an expression whose register kind is not general.
	ErrCompileDereferenceAssignRequiresPointer = errors.New("dereference assignment requires pointer in general register")

	// ErrCompileMapLiteralExpectKeyValue is returned when a map literal element is not in
	// key-value form.
	ErrCompileMapLiteralExpectKeyValue = errors.New("expected key-value in map literal")

	// ErrCompileSliceIndexMustBeInteger is returned when a slice or array index expression
	// has a non-integer register kind after conversion.
	ErrCompileSliceIndexMustBeInteger = errors.New("slice index must be integer")

	// ErrCompileUnaryMinusUnsupported is returned when unary - is applied to a register kind
	// that has no negation handler.
	ErrCompileUnaryMinusUnsupported = errors.New("unary - not supported for this type")

	// ErrCompileUnaryXorRequiresInteger is returned when unary ^ is applied to an operand
	// whose register kind is not int or uint.
	ErrCompileUnaryXorRequiresInteger = errors.New("unary ^ requires integer operand")

	// ErrCompileChannelReceiveRequiresGeneral is returned when the channel receive operator
	// <- is applied to an operand whose register kind is not general.
	ErrCompileChannelReceiveRequiresGeneral = errors.New("channel receive requires general register operand")

	// ErrCompileBreakOutsideLoopOrSwitch is returned when a break statement is compiled
	// outside an enclosing loop or switch.
	ErrCompileBreakOutsideLoopOrSwitch = errors.New("break outside loop or switch")

	// ErrCompileContinueOutsideLoop is returned when a continue statement is compiled
	// outside an enclosing loop.
	ErrCompileContinueOutsideLoop = errors.New("continue outside loop")

	// ErrCompileFallthroughOutsideSwitch is returned when a fallthrough statement is
	// compiled outside an enclosing switch.
	ErrCompileFallthroughOutsideSwitch = errors.New("fallthrough outside switch")

	// ErrCompileTailCallTargetNotIdent is returned when a tail call's callee expression is
	// not a bare identifier.
	ErrCompileTailCallTargetNotIdent = errors.New("tail call target is not an identifier")

	// ErrCompileTypeSwitchAssignNotTypeAssert is returned when the right-hand side of a
	// type-switch assignment is not a type assertion expression.
	ErrCompileTypeSwitchAssignNotTypeAssert = errors.New("type switch assign RHS is not a type assertion")

	// ErrCompileTypeSwitchExprNotTypeAssert is returned when a type-switch expression
	// statement is not a type assertion.
	ErrCompileTypeSwitchExprNotTypeAssert = errors.New("type switch expression is not a type assertion")

	// ErrCompileMethodExprMissingReceiver is returned when a method expression call has no
	// receiver argument.
	ErrCompileMethodExprMissingReceiver = errors.New("method expression call missing receiver argument")

	// ErrCompileIncDecSelectorNumeric is returned when an inc/dec on a struct field selector
	// targets a non-numeric field kind.
	ErrCompileIncDecSelectorNumeric = errors.New("inc/dec on selector requires numeric field")

	// ErrCompileIncDecRequiresNumeric is returned when an inc/dec statement targets a
	// variable whose kind is not int, float, or uint.
	ErrCompileIncDecRequiresNumeric = errors.New("inc/dec requires numeric variable")

	// ErrCompileMapCommaOkValueNotIdent is returned when the value target of a map comma-ok
	// assignment is not a bare identifier.
	ErrCompileMapCommaOkValueNotIdent = errors.New("map comma-ok value target is not an identifier")

	// ErrCompileMapCommaOkOkNotIdent is returned when the ok target of a map comma-ok
	// assignment is not a bare identifier.
	ErrCompileMapCommaOkOkNotIdent = errors.New("map comma-ok ok target is not an identifier")

	// ErrCompileMapCommaOkSourceNotMap is returned when a `:=` map comma-ok is compiled but
	// the source expression is not a map type.
	ErrCompileMapCommaOkSourceNotMap = errors.New("map comma-ok source is not a map type")

	// ErrCompileChanRecvCommaOkSourceNotChan is returned when a channel-receive comma-ok is
	// compiled but the source expression is not a channel type.
	ErrCompileChanRecvCommaOkSourceNotChan = errors.New("channel receive comma-ok source is not a channel type")

	// ErrCompileChanRecvCommaOkValueNotIdent is returned when the value target of a
	// channel-receive comma-ok is not a bare identifier.
	ErrCompileChanRecvCommaOkValueNotIdent = errors.New("channel receive comma-ok value target is not an identifier")

	// ErrCompileChanRecvCommaOkOkNotIdent is returned when the ok target of a
	// channel-receive comma-ok is not a bare identifier.
	ErrCompileChanRecvCommaOkOkNotIdent = errors.New("channel receive comma-ok ok target is not an identifier")

	// ErrCompileTypeAssertCommaOkValueNotIdent is returned when the value target of a
	// type-assert comma-ok is not a bare identifier.
	ErrCompileTypeAssertCommaOkValueNotIdent = errors.New("type assert comma-ok value target is not an identifier")

	// ErrCompileTypeAssertCommaOkOkNotIdent is returned when the ok target of a type-assert
	// comma-ok is not a bare identifier.
	ErrCompileTypeAssertCommaOkOkNotIdent = errors.New("type assert comma-ok ok target is not an identifier")

	// ErrCompileArithFloatUnsupported is returned when an arithmetic opcode has no
	// float-bank counterpart.
	ErrCompileArithFloatUnsupported = errors.New("operation not supported for float")

	// ErrCompileArithStringUnsupported is returned when an arithmetic opcode has no
	// string-bank counterpart.
	ErrCompileArithStringUnsupported = errors.New("operation not supported for string")

	// ErrCompileArithGeneralUnsupported is returned when an arithmetic opcode has no
	// general-bank counterpart.
	ErrCompileArithGeneralUnsupported = errors.New("operation not supported for this type")

	// ErrCompileArithUintUnsupported is returned when an arithmetic or bitwise opcode has no
	// uint-bank counterpart.
	ErrCompileArithUintUnsupported = errors.New("operation not supported for uint")

	// ErrCompileArithComplexUnsupported is returned when an arithmetic opcode has no
	// complex-bank counterpart.
	ErrCompileArithComplexUnsupported = errors.New("operation not supported for complex")

	// ErrCompileCompareFloatUnsupported is returned when a comparison opcode has no
	// float-bank counterpart.
	ErrCompileCompareFloatUnsupported = errors.New("comparison not supported for float")

	// ErrCompileCompareStringUnsupported is returned when a comparison opcode has no
	// string-bank counterpart.
	ErrCompileCompareStringUnsupported = errors.New("comparison not supported for string")

	// ErrCompileCompareGeneralUnsupported is returned when a comparison opcode has no
	// general-bank counterpart.
	ErrCompileCompareGeneralUnsupported = errors.New("comparison not supported for this type")

	// ErrCompileCompareComplexOrdering is returned when an ordering comparison (<, <=, >,
	// >=) is applied to complex operands, which Go only allows for == and !=.
	ErrCompileCompareComplexOrdering = errors.New("comparison not supported for complex (only == and !=)")

	// ErrCompileBitwiseRequiresInteger is returned when a bitwise or shift operation
	// receives a left operand whose register kind is not int or uint.
	ErrCompileBitwiseRequiresInteger = errors.New("operation requires integer operands")

	// ErrCompileBuiltinLenArgCount is returned when len is called with anything other than
	// exactly one argument.
	ErrCompileBuiltinLenArgCount = errors.New("len requires exactly 1 argument")

	// ErrCompileBuiltinAppendArgCount is returned when append is called with fewer than two
	// arguments.
	ErrCompileBuiltinAppendArgCount = errors.New("append requires at least 2 arguments")

	// ErrCompileBuiltinDeleteArgCount is returned when delete is called with anything other
	// than exactly two arguments.
	ErrCompileBuiltinDeleteArgCount = errors.New("delete requires exactly 2 arguments")

	// ErrCompileBuiltinComplexArgCount is returned when complex is called with anything
	// other than exactly two arguments.
	ErrCompileBuiltinComplexArgCount = errors.New("complex requires exactly 2 arguments")

	// ErrCompileBuiltinComplexRequiresFloat is returned when complex is called with operands
	// whose register kind is not float.
	ErrCompileBuiltinComplexRequiresFloat = errors.New("complex requires float arguments")

	// ErrCompileBuiltinCapArgCount is returned when cap is called with anything other than
	// exactly one argument.
	ErrCompileBuiltinCapArgCount = errors.New("cap requires exactly 1 argument")

	// ErrCompileBuiltinCopyArgCount is returned when copy is called with anything other than
	// exactly two arguments.
	ErrCompileBuiltinCopyArgCount = errors.New("copy requires exactly 2 arguments")

	// ErrCompileBuiltinClearArgCount is returned when clear is called with anything other
	// than exactly one argument.
	ErrCompileBuiltinClearArgCount = errors.New("clear requires exactly 1 argument")

	// ErrCompileBuiltinMinMaxArgCount is returned when min or max is called with fewer than
	// two arguments.
	ErrCompileBuiltinMinMaxArgCount = errors.New("min/max requires at least 2 arguments")

	// ErrCompileBuiltinPanicArgCount is returned when panic is called with anything other
	// than exactly one argument.
	ErrCompileBuiltinPanicArgCount = errors.New("panic requires exactly 1 argument")

	// ErrCompileBuiltinCloseArgCount is returned when close is called with anything other
	// than exactly one argument.
	ErrCompileBuiltinCloseArgCount = errors.New("close expects 1 argument")

	// ErrCompileUnsafeStringDataArgCount is returned when unsafe.StringData is called with
	// anything other than exactly one argument.
	ErrCompileUnsafeStringDataArgCount = errors.New("unsafe.StringData requires 1 argument")

	// ErrCompileUnsafeSliceDataArgCount is returned when unsafe.SliceData is called with
	// anything other than exactly one argument.
	ErrCompileUnsafeSliceDataArgCount = errors.New("unsafe.SliceData requires 1 argument")

	// errLinkedCallNoInstance reports that go/types has no instantiation recorded for a
	// //piko:link call site; usually the source is invalid or the expression was not a
	// generic call.
	errLinkedCallNoInstance = errors.New("//piko:link call target has no type instantiation")

	// errLinkedCallArityMismatch reports a type-argument count that disagrees with the
	// LinkedFunction's declared TypeArgCount.
	errLinkedCallArityMismatch = errors.New("//piko:link call target type argument count mismatch")

	// errLinkedCallTypeArgUnresolvable reports a type argument that cannot be converted from
	// go/types to reflect.Type.
	errLinkedCallTypeArgUnresolvable = errors.New("//piko:link call target cannot resolve type argument")

	// errLinkedCallTooManyTypeArgs reports a LinkedFunction sentinel whose declared
	// TypeArgCount exceeds maxLinkedTypeArgCount. Guards against a malformed or hostile
	// registration driving unbounded allocation through resolveLinkedTypeArgs.
	errLinkedCallTooManyTypeArgs = errors.New("//piko:link call target declares too many type arguments")

	// ErrDebuggerStop is returned when the debugger requests execution to halt via
	// DebugActionStop.
	ErrDebuggerStop = errors.New("debugger: execution stopped")

	// ErrMapLinknameSafeStubInvoked signals a safe-build linkname stub call.
	//
	// Raised when a safe-build stub for one of the runtime map / typedmemmove linkname
	// trampolines has been reached. The stubs are unreachable by design: callers must gate
	// their fast-path use behind useMapFastLinkname, which returns false on the safe /
	// js+wasm build. The sentinel surfacing indicates a build-tag misconfiguration rather
	// than user input.
	ErrMapLinknameSafeStubInvoked = errors.New("safe-build linkname stub invoked: caller must gate via useMapFastLinkname")
)
