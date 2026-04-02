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
	"testing"
	"unsafe"
)

func TestDispatchContextOffsets(t *testing.T) {
	t.Parallel()

	var ctx DispatchContext

	tests := []struct {
		name   string
		got    uintptr
		expect uintptr
	}{
		{name: "codeBase", got: unsafe.Offsetof(ctx.codeBase), expect: 0},
		{name: "codeLength", got: unsafe.Offsetof(ctx.codeLength), expect: 8},
		{name: "pc", got: unsafe.Offsetof(ctx.programCounter), expect: 16},
		{name: "intsBase", got: unsafe.Offsetof(ctx.intsBase), expect: 24},
		{name: "intsLength", got: unsafe.Offsetof(ctx.intsLength), expect: 32},
		{name: "floatsBase", got: unsafe.Offsetof(ctx.floatsBase), expect: 40},
		{name: "floatsLength", got: unsafe.Offsetof(ctx.floatsLength), expect: 48},
		{name: "intConstantsBase", got: unsafe.Offsetof(ctx.intConstantsBase), expect: 56},
		{name: "intConstantsLength", got: unsafe.Offsetof(ctx.intConstantsLength), expect: 64},
		{name: "floatConstantsBase", got: unsafe.Offsetof(ctx.floatConstantsBase), expect: 72},
		{name: "floatConstantsLength", got: unsafe.Offsetof(ctx.floatConstantsLength), expect: 80},
		{name: "jumpTable", got: unsafe.Offsetof(ctx.jumpTable), expect: 88},
		{name: "exitReason", got: unsafe.Offsetof(ctx.exitReason), expect: 96},
		{name: "exitProgramCounter", got: unsafe.Offsetof(ctx.exitProgramCounter), expect: 104},
		{name: "asmCallInfoBase", got: unsafe.Offsetof(ctx.asmCallInfoBase), expect: 112},
		{name: "callStackBase", got: unsafe.Offsetof(ctx.callStackBase), expect: 120},
		{name: "callStackLength", got: unsafe.Offsetof(ctx.callStackLength), expect: 128},
		{name: "fp", got: unsafe.Offsetof(ctx.framePointer), expect: 136},
		{name: "baseFramePointer", got: unsafe.Offsetof(ctx.baseFramePointer), expect: 144},
		{name: "callDepthLimit", got: unsafe.Offsetof(ctx.callDepthLimit), expect: 152},
		{name: "arenaIntSlab", got: unsafe.Offsetof(ctx.arenaIntSlab), expect: 160},
		{name: "arenaIntCapacity", got: unsafe.Offsetof(ctx.arenaIntCapacity), expect: 168},
		{name: "arenaIntIndex", got: unsafe.Offsetof(ctx.arenaIntIndex), expect: 176},
		{name: "arenaFloatSlab", got: unsafe.Offsetof(ctx.arenaFloatSlab), expect: 184},
		{name: "arenaFloatCapacity", got: unsafe.Offsetof(ctx.arenaFloatCapacity), expect: 192},
		{name: "arenaFloatIndex", got: unsafe.Offsetof(ctx.arenaFloatIndex), expect: 200},
		{name: "arenaStringIndex", got: unsafe.Offsetof(ctx.arenaStringIndex), expect: 208},
		{name: "arenaGeneralIndex", got: unsafe.Offsetof(ctx.arenaGeneralIndex), expect: 216},
		{name: "arenaBoolIndex", got: unsafe.Offsetof(ctx.arenaBoolIndex), expect: 224},
		{name: "arenaUintIndex", got: unsafe.Offsetof(ctx.arenaUintIndex), expect: 232},
		{name: "arenaComplexIndex", got: unsafe.Offsetof(ctx.arenaComplexIndex), expect: 240},
		{name: "deferStackLength", got: unsafe.Offsetof(ctx.deferStackLength), expect: 248},
		{name: "asmCallInfoBasesPointer", got: unsafe.Offsetof(ctx.asmCallInfoBasesPointer), expect: 256},
		{name: "dispatchSavesPointer", got: unsafe.Offsetof(ctx.dispatchSavesPointer), expect: 264},
		{name: "stringsBase", got: unsafe.Offsetof(ctx.stringsBase), expect: 272},
		{name: "uintsBase", got: unsafe.Offsetof(ctx.uintsBase), expect: 280},
		{name: "boolsBase", got: unsafe.Offsetof(ctx.boolsBase), expect: 288},
		{name: "arenaStringSlab", got: unsafe.Offsetof(ctx.arenaStringSlab), expect: 296},
		{name: "arenaStringCapacity", got: unsafe.Offsetof(ctx.arenaStringCapacity), expect: 304},
		{name: "arenaBoolSlab", got: unsafe.Offsetof(ctx.arenaBoolSlab), expect: 312},
		{name: "arenaBoolCapacity", got: unsafe.Offsetof(ctx.arenaBoolCapacity), expect: 320},
		{name: "arenaUintSlab", got: unsafe.Offsetof(ctx.arenaUintSlab), expect: 328},
		{name: "arenaUintCapacity", got: unsafe.Offsetof(ctx.arenaUintCapacity), expect: 336},
	}

	for _, tt := range tests {
		if tt.got != tt.expect {
			t.Errorf("DispatchContext.%s offset = %d, want %d", tt.name, tt.got, tt.expect)
		}
	}

	if sz := unsafe.Sizeof(ctx); sz != 344 {
		t.Errorf("DispatchContext size = %d, want 344", sz)
	}
}

func TestCallFrameOffsets(t *testing.T) {
	t.Parallel()

	var f callFrame

	tests := []struct {
		name   string
		got    uintptr
		expect uintptr
	}{
		{name: "registers.ints", got: unsafe.Offsetof(f.registers) + unsafe.Offsetof(f.registers.ints), expect: 0},
		{name: "registers.floats", got: unsafe.Offsetof(f.registers) + unsafe.Offsetof(f.registers.floats), expect: 24},
		{name: "registers.strings", got: unsafe.Offsetof(f.registers) + unsafe.Offsetof(f.registers.strings), expect: 48},
		{name: "registers.general", got: unsafe.Offsetof(f.registers) + unsafe.Offsetof(f.registers.general), expect: 72},
		{name: "registers.bools", got: unsafe.Offsetof(f.registers) + unsafe.Offsetof(f.registers.bools), expect: 96},
		{name: "registers.uints", got: unsafe.Offsetof(f.registers) + unsafe.Offsetof(f.registers.uints), expect: 120},
		{name: "registers.complex", got: unsafe.Offsetof(f.registers) + unsafe.Offsetof(f.registers.complex), expect: 144},
		{name: "fn", got: unsafe.Offsetof(f.function), expect: 168},
		{name: "sharedCells", got: unsafe.Offsetof(f.sharedCells), expect: 176},
		{name: "upvalues", got: unsafe.Offsetof(f.upvalues), expect: 184},
		{name: "returnDestination", got: unsafe.Offsetof(f.returnDestination), expect: 208},
		{name: "pc", got: unsafe.Offsetof(f.programCounter), expect: 232},
		{name: "deferBase", got: unsafe.Offsetof(f.deferBase), expect: 240},
		{name: "arenaSave", got: unsafe.Offsetof(f.arenaSave), expect: 248},
		{name: "arenaSave.intIndex", got: unsafe.Offsetof(f.arenaSave) + unsafe.Offsetof(f.arenaSave.intIndex), expect: 248},
		{name: "arenaSave.floatIndex", got: unsafe.Offsetof(f.arenaSave) + unsafe.Offsetof(f.arenaSave.floatIndex), expect: 256},
		{name: "arenaSave.stringIndex", got: unsafe.Offsetof(f.arenaSave) + unsafe.Offsetof(f.arenaSave.stringIndex), expect: 264},
		{name: "arenaSave.generalIndex", got: unsafe.Offsetof(f.arenaSave) + unsafe.Offsetof(f.arenaSave.generalIndex), expect: 272},
		{name: "arenaSave.boolIndex", got: unsafe.Offsetof(f.arenaSave) + unsafe.Offsetof(f.arenaSave.boolIndex), expect: 280},
		{name: "arenaSave.uintIndex", got: unsafe.Offsetof(f.arenaSave) + unsafe.Offsetof(f.arenaSave.uintIndex), expect: 288},
		{name: "arenaSave.complexIndex", got: unsafe.Offsetof(f.arenaSave) + unsafe.Offsetof(f.arenaSave.complexIndex), expect: 296},
		{name: "arenaSave.upvalueCellIndex", got: unsafe.Offsetof(f.arenaSave) + unsafe.Offsetof(f.arenaSave.upvalueCellIndex), expect: 304},
		{name: "arenaSave.upvalueReferenceIndex", got: unsafe.Offsetof(f.arenaSave) + unsafe.Offsetof(f.arenaSave.upvalueReferenceIndex), expect: 312},
	}

	for _, tt := range tests {
		if tt.got != tt.expect {
			t.Errorf("callFrame.%s offset = %d, want %d", tt.name, tt.got, tt.expect)
		}
	}

	if sz := unsafe.Sizeof(f); sz != 320 {
		t.Errorf("callFrame size = %d, want 320", sz)
	}
}

func TestVarLocationOffsets(t *testing.T) {
	t.Parallel()

	var v varLocation

	tests := []struct {
		name   string
		got    uintptr
		expect uintptr
	}{
		{name: "UpvalueIndex", got: unsafe.Offsetof(v.upvalueIndex), expect: 0},
		{name: "Register", got: unsafe.Offsetof(v.register), expect: 8},
		{name: "Kind", got: unsafe.Offsetof(v.kind), expect: 9},
		{name: "IsUpvalue", got: unsafe.Offsetof(v.isUpvalue), expect: 10},
		{name: "IsIndirect", got: unsafe.Offsetof(v.isIndirect), expect: 11},
		{name: "OriginalKind", got: unsafe.Offsetof(v.originalKind), expect: 12},
	}

	for _, tt := range tests {
		if tt.got != tt.expect {
			t.Errorf("varLocation.%s offset = %d, want %d", tt.name, tt.got, tt.expect)
		}
	}

	if sz := unsafe.Sizeof(v); sz != 24 {
		t.Errorf("varLocation size = %d, want 24", sz)
	}
}

func TestASMCallInfoOffsets(t *testing.T) {
	t.Parallel()

	var ci asmCallInfo

	tests := []struct {
		name   string
		got    uintptr
		expect uintptr
	}{
		{name: "calleeFunction", got: unsafe.Offsetof(ci.calleeFunction), expect: 0},
		{name: "calleeBody", got: unsafe.Offsetof(ci.calleeBody), expect: 8},
		{name: "calleeBodyLength", got: unsafe.Offsetof(ci.calleeBodyLength), expect: 16},
		{name: "calleeIntConstants", got: unsafe.Offsetof(ci.calleeIntConstants), expect: 24},
		{name: "calleeFloatConstants", got: unsafe.Offsetof(ci.calleeFloatConstants), expect: 32},
		{name: "calleeIntCount", got: unsafe.Offsetof(ci.calleeIntCount), expect: 40},
		{name: "calleeFloatCount", got: unsafe.Offsetof(ci.calleeFloatCount), expect: 48},
		{name: "intArgumentCount", got: unsafe.Offsetof(ci.intArgumentCount), expect: 56},
		{name: "intArgumentSources", got: unsafe.Offsetof(ci.intArgumentSources), expect: 64},
		{name: "floatArgumentCount", got: unsafe.Offsetof(ci.floatArgumentCount), expect: 128},
		{name: "floatArgumentSources", got: unsafe.Offsetof(ci.floatArgumentSources), expect: 136},
		{name: "returnCount", got: unsafe.Offsetof(ci.returnCount), expect: 200},
		{name: "returnDestinationKind", got: unsafe.Offsetof(ci.returnDestinationKind), expect: 208},
		{name: "returnDestinationReg", got: unsafe.Offsetof(ci.returnDestinationReg), expect: 216},
		{name: "returnDestinationPtr", got: unsafe.Offsetof(ci.returnDestinationPtr), expect: 224},
		{name: "returnDestinationLen", got: unsafe.Offsetof(ci.returnDestinationLen), expect: 232},
		{name: "calleeCallInfo", got: unsafe.Offsetof(ci.calleeCallInfo), expect: 240},
		{name: "isFastPath", got: unsafe.Offsetof(ci.isFastPath), expect: 248},
		{name: "calleeStringCount", got: unsafe.Offsetof(ci.calleeStringCount), expect: 256},
		{name: "calleeBoolCount", got: unsafe.Offsetof(ci.calleeBoolCount), expect: 264},
		{name: "calleeUintCount", got: unsafe.Offsetof(ci.calleeUintCount), expect: 272},
		{name: "stringArgumentCount", got: unsafe.Offsetof(ci.stringArgumentCount), expect: 280},
		{name: "stringArgumentSources", got: unsafe.Offsetof(ci.stringArgumentSources), expect: 288},
		{name: "boolArgumentCount", got: unsafe.Offsetof(ci.boolArgumentCount), expect: 352},
		{name: "boolArgumentSources", got: unsafe.Offsetof(ci.boolArgumentSources), expect: 360},
		{name: "uintArgumentCount", got: unsafe.Offsetof(ci.uintArgumentCount), expect: 424},
		{name: "uintArgumentSources", got: unsafe.Offsetof(ci.uintArgumentSources), expect: 432},
	}

	for _, tt := range tests {
		if tt.got != tt.expect {
			t.Errorf("asmCallInfo.%s offset = %d, want %d", tt.name, tt.got, tt.expect)
		}
	}

	if sz := unsafe.Sizeof(ci); sz != 512 {
		t.Errorf("asmCallInfo size = %d, want 512", sz)
	}
}

func TestAsmDispatchSaveOffsets(t *testing.T) {
	t.Parallel()

	var ds asmDispatchSave

	tests := []struct {
		name   string
		got    uintptr
		expect uintptr
	}{
		{name: "codeBase", got: unsafe.Offsetof(ds.codeBase), expect: 0},
		{name: "codeLength", got: unsafe.Offsetof(ds.codeLength), expect: 8},
		{name: "intConstantsBase", got: unsafe.Offsetof(ds.intConstantsBase), expect: 16},
		{name: "floatConstantsBase", got: unsafe.Offsetof(ds.floatConstantsBase), expect: 24},
	}

	for _, tt := range tests {
		if tt.got != tt.expect {
			t.Errorf("asmDispatchSave.%s offset = %d, want %d", tt.name, tt.got, tt.expect)
		}
	}

	if sz := unsafe.Sizeof(ds); sz != 32 {
		t.Errorf("asmDispatchSave size = %d, want 32", sz)
	}
}
