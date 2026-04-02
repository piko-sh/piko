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

var TestNewVM = newVM

var TestVMExecute = (*VM).execute

var TestNewGlobalStore = newGlobalStore

var TestVMRun = (*VM).run

var TestVMPushFrame = (*VM).pushFrame

var TestMakeInstruction = makeInstruction

var TestNewDebugState = newDebugState

var TestErrDebuggerStop = ErrDebuggerStop

func ExportSourceMapOf(cf *CompiledFunction) *sourceMap { return cf.debugSourceMap }

func ExportVarTableOf(cf *CompiledFunction) *debugVarTable { return cf.debugVarTable }

func ExportDebugStateHasBreakpoint(ds any, fn *CompiledFunction, pc int) bool {
	return ds.(*debugState).hasBreakpoint(fn, pc)
}

func ExportDebugStateShouldStep(ds any, fn *CompiledFunction, pc int, fp int) (bool, DebugEvent) {
	return ds.(*debugState).shouldStep(fn, pc, fp)
}

func ExportDebugStateApplyAction(ds any, action DebugAction, fp int, file string, line int) {
	ds.(*debugState).applyAction(action, fp, file, line)
}

func ExportDebugStateSetBreakpoint(ds any, file string, line int) {
	ds.(*debugState).breakpoints[breakpointKey{file: file, line: line}] = true
}

func ExportReadVariable(frame any, entry any) any {
	return readVariable(frame.(*callFrame), entry.(debugVarEntry))
}

func ExportNewCompiledFunctionWithSourceMap(name string, bodyLen int, positions []sourcePosition, files []string) *CompiledFunction {
	filesCopy := make([]string, len(files))
	copy(filesCopy, files)
	cf := &CompiledFunction{
		name: name,
		body: make([]instruction, bodyLen),
		debugSourceMap: &sourceMap{
			files:     &filesCopy,
			positions: positions,
		},
	}
	return cf
}

type ExportSourcePosition = sourcePosition
