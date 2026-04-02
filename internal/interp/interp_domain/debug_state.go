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

import "errors"

// stepMode identifies the current stepping state of the debugger.
type stepMode uint8

const (
	// stepModeNone means no stepping - run until breakpoint or end.
	stepModeNone stepMode = iota

	// stepModeIn steps to the next source line, entering calls.
	stepModeIn

	// stepModeOver steps to the next source line at same or
	// shallower depth.
	stepModeOver

	// stepModeOut runs until the current function returns.
	stepModeOut
)

// ErrDebuggerStop is returned when the debugger requests execution to
// halt via DebugActionStop.
var ErrDebuggerStop = errors.New("debugger: execution stopped")

// breakpointKey identifies a breakpoint by file path and line number.
type breakpointKey struct {
	// file is the source file path of the breakpoint.
	file string

	// line is the source line number of the breakpoint.
	line int
}

// debugState holds the mutable debug state used by the VM's debug
// hook to evaluate breakpoints and stepping conditions.
type debugState struct {
	// breakpoints maps file:line keys to active breakpoints.
	breakpoints map[breakpointKey]bool

	// stepFile is the source file when stepping was initiated.
	stepFile string

	// lastBreakpointFile and lastBreakpointLine track the most
	// recently fired breakpoint location. This prevents the same
	// breakpoint from firing repeatedly when multiple bytecode
	// instructions map to the same source line.
	lastBreakpointFile string

	// stepFramePointer is the frame depth at which stepping was
	// initiated. Used by step-over (pause at same or shallower)
	// and step-out (pause when shallower).
	stepFramePointer int

	// stepLine is the source line when stepping was initiated.
	// Step-over skips instructions on the same line.
	stepLine int

	// lastBreakpointLine is the source line of the most recently
	// fired breakpoint.
	lastBreakpointLine int

	// lastBreakpointFrame is the frame depth at which the most
	// recently fired breakpoint occurred.
	lastBreakpointFrame int

	// stepping is the current step mode.
	stepping stepMode
}

// newDebugState creates a debugState with an initialised breakpoint
// map.
//
// Returns *debugState with an empty breakpoint map ready for use.
func newDebugState() *debugState {
	return &debugState{
		breakpoints: make(map[breakpointKey]bool),
	}
}

// hasBreakpoint checks whether the given function has a breakpoint
// at the given program counter.
//
// Takes fn (*CompiledFunction) which provides the source map.
// Takes pc (int) which is the current program counter.
//
// Returns true if a breakpoint is registered at the source position.
func (ds *debugState) hasBreakpoint(fn *CompiledFunction, pc int) bool {
	if fn.debugSourceMap == nil {
		return false
	}
	file, line, _ := fn.debugSourceMap.SourcePosition(pc)
	if line == 0 {
		return false
	}

	if file == ds.lastBreakpointFile && line == ds.lastBreakpointLine {
		return false
	}

	return ds.breakpoints[breakpointKey{file: file, line: line}]
}

// shouldStep determines whether the debugger should pause at the
// current position based on the active step mode.
//
// Takes fn (*CompiledFunction) which provides the source map.
// Takes pc (int) which is the current program counter.
// Takes framePointer (int) which is the current call stack depth.
//
// Returns true if the debugger should pause, and the event type.
func (ds *debugState) shouldStep(fn *CompiledFunction, pc int, framePointer int) (bool, DebugEvent) {
	if ds.stepping == stepModeNone {
		return false, 0
	}

	if ds.stepping == stepModeOut {
		if framePointer < ds.stepFramePointer {
			return true, DebugEventStep
		}
		return false, 0
	}

	if fn.debugSourceMap == nil {
		return false, 0
	}

	file, line, _ := fn.debugSourceMap.SourcePosition(pc)
	if line == 0 {
		return false, 0
	}

	switch ds.stepping {
	case stepModeIn:
		if file != ds.stepFile || line != ds.stepLine {
			return true, DebugEventStep
		}
	case stepModeOver:
		if framePointer <= ds.stepFramePointer &&
			(file != ds.stepFile || line != ds.stepLine) {
			return true, DebugEventStep
		}
	}

	return false, 0
}

// applyAction updates the step mode based on the debug action
// returned by the hook or debugger API.
//
// Takes action (DebugAction) which is the action to apply.
// Takes framePointer (int) which is the current frame depth.
// Takes file (string) which is the current source file.
// Takes line (int) which is the current source line.
func (ds *debugState) applyAction(action DebugAction, framePointer int, file string, line int) {
	switch action {
	case DebugActionContinue:
		ds.stepping = stepModeNone
	case DebugActionStepIn:
		ds.stepping = stepModeIn
		ds.stepFramePointer = framePointer
		ds.stepFile = file
		ds.stepLine = line
	case DebugActionStepOver:
		ds.stepping = stepModeOver
		ds.stepFramePointer = framePointer
		ds.stepFile = file
		ds.stepLine = line
	case DebugActionStepOut:
		ds.stepping = stepModeOut
		ds.stepFramePointer = framePointer
		ds.stepFile = file
		ds.stepLine = line
	}
}
