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

// DebugEvent identifies why the debugger hook was invoked.
type DebugEvent uint8

const (
	// DebugEventStep fires when the VM reaches a new source line
	// during single-stepping.
	DebugEventStep DebugEvent = iota

	// DebugEventBreakpoint fires when the VM hits a registered
	// breakpoint.
	DebugEventBreakpoint

	// DebugEventEntry fires when the VM enters a new function.
	DebugEventEntry

	// DebugEventExit fires when the VM is about to exit a function.
	DebugEventExit
)

// DebugAction tells the VM what to do after the debug hook returns.
type DebugAction uint8

const (
	// DebugActionContinue resumes execution until the next
	// breakpoint or program termination.
	DebugActionContinue DebugAction = iota

	// DebugActionStepIn advances to the next source line, entering
	// function calls.
	DebugActionStepIn

	// DebugActionStepOver advances to the next source line at the
	// same or shallower call depth, skipping into function calls.
	DebugActionStepOver

	// DebugActionStepOut runs until the current function returns,
	// then pauses in the caller.
	DebugActionStepOut

	// DebugActionStop terminates execution immediately.
	DebugActionStop
)

// DebugContext provides the debug hook with information about the
// current execution state.
type DebugContext struct {
	// Function is the compiled function currently executing.
	Function *CompiledFunction

	// ProgramCounter is the current instruction offset within the
	// function's body.
	ProgramCounter int

	// FramePointer is the current call stack frame index.
	FramePointer int

	// Event describes why the hook was invoked.
	Event DebugEvent
}

// DebugHook is a callback invoked by the VM at debug-relevant
// points during execution. It receives context about the current
// state and returns an action telling the VM how to proceed.
type DebugHook func(ctx DebugContext) DebugAction

// DebugSnapshot captures the execution state at a debug pause point.
type DebugSnapshot struct {
	// File is the source file path at the pause point.
	File string

	// FunctionName is the name of the function at the pause point.
	FunctionName string

	// StackTrace holds the call stack from innermost to outermost.
	StackTrace []StackFrame

	// Line is the 1-based source line number.
	Line int

	// Column is the 1-based source column number.
	Column int

	// Event describes why execution paused.
	Event DebugEvent
}

// StackFrame describes a single frame in the debug call stack.
type StackFrame struct {
	// FunctionName is the function's qualified name.
	FunctionName string

	// File is the source file path.
	File string

	// Line is the 1-based source line at this frame.
	Line int

	// Column is the 1-based source column at this frame.
	Column int
}

// VariableInfo describes a variable visible at a debug pause point.
type VariableInfo struct {
	// Name is the source-level variable name.
	Name string

	// Value is the variable's current runtime value.
	Value any

	// Kind describes the type (e.g. "int", "float64", "string").
	Kind string
}
