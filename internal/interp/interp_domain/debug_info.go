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

// sourcePosition records the source file location for a single
// bytecode instruction. Used by the debugger to map program counters
// back to source lines.
type sourcePosition struct {
	// line is the 1-based source line number, where 0 means unknown
	// or synthetic (e.g. NOP inserted by the optimiser).
	line int32

	// column is the 1-based source column number. 0 means unknown.
	column int16

	// fileID is an index into sourceMap.files, identifying which
	// source file this instruction came from.
	fileID uint16
}

// sourceMap maps program counter offsets to source file positions.
// The positions slice is parallel to CompiledFunction.body - each
// entry corresponds to the instruction at the same index.
type sourceMap struct {
	// files points to the shared file table mapping fileID indices
	// to source file paths. Shared via pointer across functions
	// compiled from the same compilation unit so that appends from
	// any sub-compiler are visible to all.
	files *[]string

	// positions is parallel to CompiledFunction.body. positions[pc]
	// gives the source position for body[pc].
	positions []sourcePosition
}

// SourcePosition returns the file path, line, and column for the
// given program counter.
//
// Takes pc (int) which is the program counter to look up.
//
// Returns the file path, line number, and column number. Returns
// empty string and zeros when the source map is nil, pc is out of
// range, or the position is synthetic.
func (sm *sourceMap) SourcePosition(pc int) (file string, line int, column int) {
	if sm == nil || pc < 0 || pc >= len(sm.positions) {
		return "", 0, 0
	}
	pos := sm.positions[pc]
	if pos.line == 0 {
		return "", 0, 0
	}
	if sm.files == nil || int(pos.fileID) >= len(*sm.files) {
		return "", int(pos.line), int(pos.column)
	}
	return (*sm.files)[pos.fileID], int(pos.line), int(pos.column)
}

// HasDebugSourceMap reports whether the function has a source map.
//
// Returns bool which is true when a source map is present.
func (cf *CompiledFunction) HasDebugSourceMap() bool {
	return cf.debugSourceMap != nil
}

// DebugSourcePosition returns the source position for the given
// program counter.
//
// Takes pc (int) which is the program counter to look up.
//
// Returns the file path, line number, and column number.
func (cf *CompiledFunction) DebugSourcePosition(pc int) (file string, line int, column int) {
	if cf.debugSourceMap == nil {
		return "", 0, 0
	}
	return cf.debugSourceMap.SourcePosition(pc)
}

// HasDebugVarTable reports whether the function has a variable table.
//
// Returns bool which is true when a variable table is present.
func (cf *CompiledFunction) HasDebugVarTable() bool {
	return cf.debugVarTable != nil
}

// debugVarEntry records debug information for a single variable
// declaration, mapping a variable name to its runtime location
// and the bytecode range where it is live.
type debugVarEntry struct {
	// name is the source-level variable name.
	name string

	// location is the register or upvalue location at runtime.
	location varLocation

	// startPC is the first instruction (inclusive) where this
	// variable is in scope.
	startPC int

	// endPC is the first instruction (exclusive) past this
	// variable's scope. 0 means end-of-function.
	endPC int
}

// debugVarTable holds variable debug information for a single
// compiled function.
type debugVarTable struct {
	// entries holds all variable declarations with their liveness
	// ranges.
	entries []debugVarEntry
}

// LiveVariables returns the variable entries that are live at the
// given program counter.
//
// Takes pc (int) which is the program counter to check liveness at.
//
// Returns a slice of debugVarEntry for variables whose scope
// contains the given pc.
func (dvt *debugVarTable) LiveVariables(pc int) []debugVarEntry {
	if dvt == nil {
		return nil
	}
	var live []debugVarEntry
	for _, entry := range dvt.entries {
		end := entry.endPC
		if end == 0 {
			end = int(^uint(0) >> 1)
		}
		if pc >= entry.startPC && pc < end {
			live = append(live, entry)
		}
	}
	return live
}
