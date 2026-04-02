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

import "sync/atomic"

// Debugger provides the public API for controlling interpreter
// debugging.
//
// It is created before execution and attached to the VM via
// WithDebugger. The debugger synchronises with the VM goroutine
// using channels; the VM blocks on resume when paused, and the
// caller blocks on paused when waiting for a pause.
type Debugger struct {
	// state holds breakpoints and stepping state, accessed by the
	// VM goroutine.
	state *debugState

	// active is shared with the VM and set to 1 when debugging is
	// active. The VM checks this via atomic load in the hot loop.
	active *atomic.Uint32

	// paused is sent on by the VM goroutine when it hits a debug
	// pause point.
	paused chan struct{}

	// resume is received from by the VM goroutine after pausing,
	// to get the next action.
	resume chan DebugAction

	// snapshot holds the execution state at the most recent pause.
	snapshot *DebugSnapshot

	// vm is set when the debugger is attached to a VM.
	vm *VM
}

// NewDebugger creates a new Debugger ready to be attached to a VM
// via WithDebugger.
//
// Returns *Debugger which can be configured with breakpoints before
// execution begins.
func NewDebugger() *Debugger {
	return &Debugger{
		state:  newDebugState(),
		active: &atomic.Uint32{},
		paused: make(chan struct{}, 1),
		resume: make(chan DebugAction, 1),
	}
}

// SetBreakpoint registers a breakpoint at the given file and line.
// The file path should match the path used during compilation.
//
// Takes file (string) which is the source file path.
// Takes line (int) which is the 1-based line number.
func (d *Debugger) SetBreakpoint(file string, line int) {
	d.state.breakpoints[breakpointKey{file: file, line: line}] = true
	d.active.Store(1)
}

// ClearBreakpoint removes a breakpoint at the given file and line.
//
// Takes file (string) which is the source file path.
// Takes line (int) which is the 1-based line number.
func (d *Debugger) ClearBreakpoint(file string, line int) {
	delete(d.state.breakpoints, breakpointKey{file: file, line: line})
	if len(d.state.breakpoints) == 0 && d.state.stepping == stepModeNone {
		d.active.Store(0)
	}
}

// Continue resumes execution until the next breakpoint or end.
func (d *Debugger) Continue() {
	d.resume <- DebugActionContinue
}

// StepIn resumes execution until the next source line, entering
// function calls.
func (d *Debugger) StepIn() {
	d.resume <- DebugActionStepIn
}

// StepOver resumes execution until the next source line at the same
// or shallower call depth.
func (d *Debugger) StepOver() {
	d.resume <- DebugActionStepOver
}

// StepOut resumes execution until the current function returns.
func (d *Debugger) StepOut() {
	d.resume <- DebugActionStepOut
}

// Stop terminates execution immediately with ErrDebuggerStop.
func (d *Debugger) Stop() {
	d.resume <- DebugActionStop
}

// WaitForPause blocks until the VM pauses at a breakpoint or step
// point. Returns the snapshot of the execution state at the pause.
//
// Returns *DebugSnapshot which describes the paused location and
// stack trace.
func (d *Debugger) WaitForPause() *DebugSnapshot {
	<-d.paused
	return d.snapshot
}

// Variables returns the variables visible at the given stack frame
// index (0 = innermost/current frame).
//
// Takes frameIndex (int) which is the stack frame to inspect.
//
// Returns []VariableInfo describing each visible variable.
func (d *Debugger) Variables(frameIndex int) []VariableInfo {
	if d.vm == nil {
		return nil
	}

	targetFP := d.vm.framePointer - frameIndex
	if targetFP < 0 || targetFP >= len(d.vm.callStack) {
		return nil
	}

	frame := &d.vm.callStack[targetFP]
	fn := frame.function
	if fn.debugVarTable == nil {
		return nil
	}

	pc := frame.programCounter

	live := fn.debugVarTable.LiveVariables(pc)
	result := make([]VariableInfo, 0, len(live))
	for _, entry := range live {
		val := readVariable(frame, entry)
		result = append(result, VariableInfo{
			Name:  entry.name,
			Value: val,
			Kind:  entry.location.kind.String(),
		})
	}
	return result
}

// Snapshot returns the most recent debug snapshot without blocking.
//
// Returns *DebugSnapshot or nil if no pause has occurred.
func (d *Debugger) Snapshot() *DebugSnapshot {
	return d.snapshot
}

// debugHookImpl is the DebugHook implementation used when a
// Debugger is attached to a VM. It checks breakpoints and stepping
// conditions, builds a snapshot, signals the pause, and blocks
// until the caller sends a resume action.
//
// Takes ctx (DebugContext) which provides the current execution
// state including function, program counter, and frame pointer.
//
// Returns DebugAction which tells the VM how to proceed.
func (d *Debugger) debugHookImpl(ctx DebugContext) DebugAction {
	fn := ctx.Function
	pc := ctx.ProgramCounter

	hitBreakpoint := d.state.hasBreakpoint(fn, pc)

	shouldStep, stepEvent := d.state.shouldStep(fn, pc, ctx.FramePointer)

	if !hitBreakpoint && !shouldStep {
		if fn.debugSourceMap != nil && ctx.FramePointer == d.state.lastBreakpointFrame {
			file, line, _ := fn.debugSourceMap.SourcePosition(pc)
			if line > 0 && (file != d.state.lastBreakpointFile || line != d.state.lastBreakpointLine) {
				d.state.lastBreakpointFile = ""
				d.state.lastBreakpointLine = 0
			}
		}
		return DebugActionContinue
	}

	event := ctx.Event
	if hitBreakpoint {
		event = DebugEventBreakpoint
	} else {
		event = stepEvent
	}

	d.snapshot = d.buildSnapshot(fn, pc, ctx.FramePointer, event)

	if hitBreakpoint {
		d.state.lastBreakpointFile = d.snapshot.File
		d.state.lastBreakpointLine = d.snapshot.Line
		d.state.lastBreakpointFrame = ctx.FramePointer
	}

	d.paused <- struct{}{}
	action := <-d.resume

	file, line := d.snapshot.File, d.snapshot.Line
	d.state.applyAction(action, ctx.FramePointer, file, line)

	if action != DebugActionContinue || len(d.state.breakpoints) > 0 {
		d.active.Store(1)
	}

	return action
}

// buildSnapshot constructs a DebugSnapshot from the current VM
// state.
//
// Takes fn (*CompiledFunction) which is the function being
// executed.
// Takes pc (int) which is the current program counter.
// Takes framePointer (int) which is the current call stack frame
// index.
// Takes event (DebugEvent) which describes why execution paused.
//
// Returns *DebugSnapshot which captures the file, line, column,
// function name, event, and stack trace.
func (d *Debugger) buildSnapshot(
	fn *CompiledFunction,
	pc int,
	framePointer int,
	event DebugEvent,
) *DebugSnapshot {
	file, line, col := "", 0, 0
	if fn.debugSourceMap != nil {
		file, line, col = fn.debugSourceMap.SourcePosition(pc)
	}

	snap := &DebugSnapshot{
		File:         file,
		FunctionName: fn.name,
		Line:         line,
		Column:       col,
		Event:        event,
	}

	if d.vm != nil {
		snap.StackTrace = d.buildStackTrace(framePointer)
	}

	return snap
}

// buildStackTrace walks the VM call stack from the current frame
// pointer down to frame 0, producing a slice of StackFrame
// entries.
//
// Takes framePointer (int) which is the topmost frame index to
// start from.
//
// Returns []StackFrame which lists each frame from innermost to
// outermost.
func (d *Debugger) buildStackTrace(framePointer int) []StackFrame {
	var frames []StackFrame
	for fp := framePointer; fp >= 0; fp-- {
		frame := &d.vm.callStack[fp]
		fn := frame.function
		pc := max(frame.programCounter-1, 0)

		sf := StackFrame{
			FunctionName: fn.name,
		}
		if fn.debugSourceMap != nil {
			sf.File, sf.Line, sf.Column = fn.debugSourceMap.SourcePosition(pc)
		}
		frames = append(frames, sf)
	}
	return frames
}

// readVariable extracts the runtime value of a variable from a
// call frame using the variable's location information.
//
// Takes frame (*callFrame) which is the call frame to read from.
// Takes entry (debugVarEntry) which describes the variable's
// storage location.
//
// Returns any which is the variable's current value.
func readVariable(frame *callFrame, entry debugVarEntry) any {
	loc := entry.location
	if loc.isUpvalue {
		return readUpvalue(frame, loc)
	}
	return readRegisterValue(&frame.registers, loc)
}

// readUpvalue extracts a captured variable's value from the
// frame's upvalue table.
//
// Takes frame (*callFrame) which is the call frame containing the
// upvalue table.
// Takes loc (varLocation) which identifies the upvalue index.
//
// Returns any which is the captured variable's value, or nil if
// the upvalue is out of bounds or empty.
func readUpvalue(frame *callFrame, loc varLocation) any {
	if frame.upvalues == nil || loc.upvalueIndex >= len(frame.upvalues) {
		return nil
	}
	uv := frame.upvalues[loc.upvalueIndex]
	if uv.value == nil {
		return nil
	}
	return readUpvalueCell(uv.value)
}

// readRegisterValue reads a variable's value from the typed
// register banks using its location.
//
// Takes regs (*Registers) which holds the typed register banks.
// Takes loc (varLocation) which identifies the register kind and
// index.
//
// Returns any which is the register value, or nil if out of
// bounds.
func readRegisterValue(regs *Registers, loc varLocation) any {
	index := int(loc.register)
	switch loc.kind {
	case registerInt:
		if index < len(regs.ints) {
			return regs.ints[loc.register]
		}
	case registerFloat:
		if index < len(regs.floats) {
			return regs.floats[loc.register]
		}
	case registerString:
		if index < len(regs.strings) {
			return regs.strings[loc.register]
		}
	case registerGeneral:
		return readGeneralRegister(regs, index)
	case registerBool:
		if index < len(regs.bools) {
			return regs.bools[loc.register]
		}
	case registerUint:
		if index < len(regs.uints) {
			return regs.uints[loc.register]
		}
	case registerComplex:
		if index < len(regs.complex) {
			return regs.complex[loc.register]
		}
	}
	return nil
}

// readGeneralRegister reads a reflect.Value register, returning
// its concrete value or nil if invalid or out of bounds.
//
// Takes regs (*Registers) which holds the general register bank.
// Takes index (int) which is the register index to read.
//
// Returns any which is the concrete value, or nil if out of bounds
// or invalid.
func readGeneralRegister(regs *Registers, index int) any {
	if index >= len(regs.general) {
		return nil
	}
	v := regs.general[index]
	if v.IsValid() {
		return v.Interface()
	}
	return nil
}

// readUpvalueCell extracts the value from an upvalue cell based
// on its kind.
//
// Takes cell (*upvalueCell) which is the cell to read.
//
// Returns any which is the stored value from the appropriate
// typed field.
func readUpvalueCell(cell *upvalueCell) any {
	switch cell.kind {
	case registerInt:
		return cell.intValue
	case registerFloat:
		return cell.floatValue
	case registerString:
		return cell.stringValue
	case registerGeneral:
		if cell.generalValue.IsValid() {
			return cell.generalValue.Interface()
		}
		return nil
	case registerBool:
		return cell.boolValue
	case registerUint:
		return cell.uintValue
	case registerComplex:
		return cell.complexValue
	default:
		return nil
	}
}

// attachToVM connects the debugger to a VM, setting up the debug
// hook and shared state.
//
// Takes vm (*VM) which is the virtual machine to attach to.
func (d *Debugger) attachToVM(vm *VM) {
	d.vm = vm
	vm.debugHook = d.debugHookImpl
	vm.debugState = d.state
	vm.debugActive = d.active
	d.active.Store(1)
}
