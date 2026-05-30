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
	"reflect"
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
		{name: "vm", got: unsafe.Offsetof(ctx.vm), expect: 0},
		{name: "codeBase", got: unsafe.Offsetof(ctx.codeBase), expect: 8},
		{name: "codeLength", got: unsafe.Offsetof(ctx.codeLength), expect: 16},
		{name: "pc", got: unsafe.Offsetof(ctx.programCounter), expect: 24},
		{name: "intsBase", got: unsafe.Offsetof(ctx.intsBase), expect: 32},
		{name: "intsLength", got: unsafe.Offsetof(ctx.intsLength), expect: 40},
		{name: "floatsBase", got: unsafe.Offsetof(ctx.floatsBase), expect: 48},
		{name: "floatsLength", got: unsafe.Offsetof(ctx.floatsLength), expect: 56},
		{name: "intConstantsBase", got: unsafe.Offsetof(ctx.intConstantsBase), expect: 64},
		{name: "intConstantsLength", got: unsafe.Offsetof(ctx.intConstantsLength), expect: 72},
		{name: "floatConstantsBase", got: unsafe.Offsetof(ctx.floatConstantsBase), expect: 80},
		{name: "floatConstantsLength", got: unsafe.Offsetof(ctx.floatConstantsLength), expect: 88},
		{name: "jumpTable", got: unsafe.Offsetof(ctx.jumpTable), expect: 96},
		{name: "exitReason", got: unsafe.Offsetof(ctx.exitReason), expect: 104},
		{name: "exitProgramCounter", got: unsafe.Offsetof(ctx.exitProgramCounter), expect: 112},
		{name: "asmCallInfoBase", got: unsafe.Offsetof(ctx.asmCallInfoBase), expect: 120},
		{name: "callStackBase", got: unsafe.Offsetof(ctx.callStackBase), expect: 128},
		{name: "callStackLength", got: unsafe.Offsetof(ctx.callStackLength), expect: 136},
		{name: "fp", got: unsafe.Offsetof(ctx.framePointer), expect: 144},
		{name: "baseFramePointer", got: unsafe.Offsetof(ctx.baseFramePointer), expect: 152},
		{name: "callDepthLimit", got: unsafe.Offsetof(ctx.callDepthLimit), expect: 160},
		{name: "arenaIntSlab", got: unsafe.Offsetof(ctx.arenaIntSlab), expect: 168},
		{name: "arenaIntCapacity", got: unsafe.Offsetof(ctx.arenaIntCapacity), expect: 176},
		{name: "arenaIntIndex", got: unsafe.Offsetof(ctx.arenaIntIndex), expect: 184},
		{name: "arenaFloatSlab", got: unsafe.Offsetof(ctx.arenaFloatSlab), expect: 192},
		{name: "arenaFloatCapacity", got: unsafe.Offsetof(ctx.arenaFloatCapacity), expect: 200},
		{name: "arenaFloatIndex", got: unsafe.Offsetof(ctx.arenaFloatIndex), expect: 208},
		{name: "arenaStringIndex", got: unsafe.Offsetof(ctx.arenaStringIndex), expect: 216},
		{name: "arenaGeneralIndex", got: unsafe.Offsetof(ctx.arenaGeneralIndex), expect: 224},
		{name: "arenaBoolIndex", got: unsafe.Offsetof(ctx.arenaBoolIndex), expect: 232},
		{name: "arenaUintIndex", got: unsafe.Offsetof(ctx.arenaUintIndex), expect: 240},
		{name: "arenaComplexIndex", got: unsafe.Offsetof(ctx.arenaComplexIndex), expect: 248},
		{name: "deferStackLength", got: unsafe.Offsetof(ctx.deferStackLength), expect: 256},
		{name: "asmCallInfoBasesPointer", got: unsafe.Offsetof(ctx.asmCallInfoBasesPointer), expect: 264},
		{name: "dispatchSavesPointer", got: unsafe.Offsetof(ctx.dispatchSavesPointer), expect: 272},
		{name: "stringsBase", got: unsafe.Offsetof(ctx.stringsBase), expect: 280},
		{name: "uintsBase", got: unsafe.Offsetof(ctx.uintsBase), expect: 288},
		{name: "boolsBase", got: unsafe.Offsetof(ctx.boolsBase), expect: 296},
		{name: "arenaStringSlab", got: unsafe.Offsetof(ctx.arenaStringSlab), expect: 304},
		{name: "arenaStringCapacity", got: unsafe.Offsetof(ctx.arenaStringCapacity), expect: 312},
		{name: "arenaBoolSlab", got: unsafe.Offsetof(ctx.arenaBoolSlab), expect: 320},
		{name: "arenaBoolCapacity", got: unsafe.Offsetof(ctx.arenaBoolCapacity), expect: 328},
		{name: "arenaUintSlab", got: unsafe.Offsetof(ctx.arenaUintSlab), expect: 336},
		{name: "arenaUintCapacity", got: unsafe.Offsetof(ctx.arenaUintCapacity), expect: 344},
		{name: "slicesIntBase", got: unsafe.Offsetof(ctx.slicesIntBase), expect: 352},
		{name: "slicesFloatBase", got: unsafe.Offsetof(ctx.slicesFloatBase), expect: 360},
		{name: "slicesStringBase", got: unsafe.Offsetof(ctx.slicesStringBase), expect: 368},
		{name: "slicesBoolBase", got: unsafe.Offsetof(ctx.slicesBoolBase), expect: 376},
		{name: "slicesUintBase", got: unsafe.Offsetof(ctx.slicesUintBase), expect: 384},
		{name: "complexBase", got: unsafe.Offsetof(ctx.complexBase), expect: 392},
		{name: "stringConstantsBase", got: unsafe.Offsetof(ctx.stringConstantsBase), expect: 400},
		{name: "stringConstantsLength", got: unsafe.Offsetof(ctx.stringConstantsLength), expect: 408},
		{name: "boolConstantsBase", got: unsafe.Offsetof(ctx.boolConstantsBase), expect: 416},
		{name: "boolConstantsLength", got: unsafe.Offsetof(ctx.boolConstantsLength), expect: 424},
		{name: "savedPC", got: unsafe.Offsetof(ctx.savedPC), expect: 432},

		{name: "tier2Result", got: unsafe.Offsetof(ctx.tier2Result), expect: 440},
		{name: "structLayoutTableBase", got: unsafe.Offsetof(ctx.structLayoutTableBase), expect: 448},
		{name: "structLayoutTableLength", got: unsafe.Offsetof(ctx.structLayoutTableLength), expect: 456},
		{name: "typeTableBase", got: unsafe.Offsetof(ctx.typeTableBase), expect: 464},
		{name: "typeTableLength", got: unsafe.Offsetof(ctx.typeTableLength), expect: 472},
		{name: "slicesByteBase", got: unsafe.Offsetof(ctx.slicesByteBase), expect: 480},
		{name: "arenaSliceByteSlab", got: unsafe.Offsetof(ctx.arenaSliceByteSlab), expect: 488},
		{name: "arenaSliceByteCapacity", got: unsafe.Offsetof(ctx.arenaSliceByteCapacity), expect: 496},
		{name: "arenaSliceByteIndex", got: unsafe.Offsetof(ctx.arenaSliceByteIndex), expect: 504},
		{name: "arenaSliceIntIndex", got: unsafe.Offsetof(ctx.arenaSliceIntIndex), expect: 512},
		{name: "arenaSliceFloatIndex", got: unsafe.Offsetof(ctx.arenaSliceFloatIndex), expect: 520},
		{name: "arenaSliceStringIndex", got: unsafe.Offsetof(ctx.arenaSliceStringIndex), expect: 528},
		{name: "arenaSliceBoolIndex", got: unsafe.Offsetof(ctx.arenaSliceBoolIndex), expect: 536},
		{name: "arenaSliceUintIndex", got: unsafe.Offsetof(ctx.arenaSliceUintIndex), expect: 544},
	}

	for _, tt := range tests {
		if tt.got != tt.expect {
			t.Errorf("DispatchContext.%s offset = %d, want %d", tt.name, tt.got, tt.expect)
		}
	}

	if sz := unsafe.Sizeof(ctx); sz != 552 {
		t.Errorf("DispatchContext size = %d, want 552", sz)
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

		{name: "fn", got: unsafe.Offsetof(f.function), expect: 0},
		{name: "sharedCells", got: unsafe.Offsetof(f.sharedCells), expect: 8},
		{name: "simpleDefer", got: unsafe.Offsetof(f.simpleDefer), expect: 16},
		{name: "registers.ints", got: unsafe.Offsetof(f.registers) + unsafe.Offsetof(f.registers.ints), expect: 24},
		{name: "registers.floats", got: unsafe.Offsetof(f.registers) + unsafe.Offsetof(f.registers.floats), expect: 48},
		{name: "registers.strings", got: unsafe.Offsetof(f.registers) + unsafe.Offsetof(f.registers.strings), expect: 72},
		{name: "registers.general", got: unsafe.Offsetof(f.registers) + unsafe.Offsetof(f.registers.general), expect: 96},
		{name: "registers.bools", got: unsafe.Offsetof(f.registers) + unsafe.Offsetof(f.registers.bools), expect: 120},
		{name: "registers.uints", got: unsafe.Offsetof(f.registers) + unsafe.Offsetof(f.registers.uints), expect: 144},
		{name: "registers.complex", got: unsafe.Offsetof(f.registers) + unsafe.Offsetof(f.registers.complex), expect: 168},
		{name: "registers.slicesInt", got: unsafe.Offsetof(f.registers) + unsafe.Offsetof(f.registers.slicesInt), expect: 192},
		{name: "registers.slicesFloat", got: unsafe.Offsetof(f.registers) + unsafe.Offsetof(f.registers.slicesFloat), expect: 216},
		{name: "registers.slicesString", got: unsafe.Offsetof(f.registers) + unsafe.Offsetof(f.registers.slicesString), expect: 240},
		{name: "registers.slicesBool", got: unsafe.Offsetof(f.registers) + unsafe.Offsetof(f.registers.slicesBool), expect: 264},
		{name: "registers.slicesUint", got: unsafe.Offsetof(f.registers) + unsafe.Offsetof(f.registers.slicesUint), expect: 288},
		{name: "registers.slicesByte", got: unsafe.Offsetof(f.registers) + unsafe.Offsetof(f.registers.slicesByte), expect: 312},
		{name: "upvalues", got: unsafe.Offsetof(f.upvalues), expect: 336},
		{name: "returnDestination", got: unsafe.Offsetof(f.returnDestination), expect: 360},
		{name: "arenaSave", got: unsafe.Offsetof(f.arenaSave), expect: 384},
		{name: "arenaSave.intIndex", got: unsafe.Offsetof(f.arenaSave) + unsafe.Offsetof(f.arenaSave.intIndex), expect: 384},
		{name: "arenaSave.floatIndex", got: unsafe.Offsetof(f.arenaSave) + unsafe.Offsetof(f.arenaSave.floatIndex), expect: 392},
		{name: "arenaSave.stringIndex", got: unsafe.Offsetof(f.arenaSave) + unsafe.Offsetof(f.arenaSave.stringIndex), expect: 400},
		{name: "arenaSave.generalIndex", got: unsafe.Offsetof(f.arenaSave) + unsafe.Offsetof(f.arenaSave.generalIndex), expect: 408},
		{name: "arenaSave.boolIndex", got: unsafe.Offsetof(f.arenaSave) + unsafe.Offsetof(f.arenaSave.boolIndex), expect: 416},
		{name: "arenaSave.uintIndex", got: unsafe.Offsetof(f.arenaSave) + unsafe.Offsetof(f.arenaSave.uintIndex), expect: 424},
		{name: "arenaSave.complexIndex", got: unsafe.Offsetof(f.arenaSave) + unsafe.Offsetof(f.arenaSave.complexIndex), expect: 432},
		{name: "arenaSave.slicesIntIndex", got: unsafe.Offsetof(f.arenaSave) + unsafe.Offsetof(f.arenaSave.slicesIntIndex), expect: 440},
		{name: "arenaSave.slicesFloatIndex", got: unsafe.Offsetof(f.arenaSave) + unsafe.Offsetof(f.arenaSave.slicesFloatIndex), expect: 448},
		{name: "arenaSave.slicesStringIndex", got: unsafe.Offsetof(f.arenaSave) + unsafe.Offsetof(f.arenaSave.slicesStringIndex), expect: 456},
		{name: "arenaSave.slicesBoolIndex", got: unsafe.Offsetof(f.arenaSave) + unsafe.Offsetof(f.arenaSave.slicesBoolIndex), expect: 464},
		{name: "arenaSave.slicesUintIndex", got: unsafe.Offsetof(f.arenaSave) + unsafe.Offsetof(f.arenaSave.slicesUintIndex), expect: 472},
		{name: "arenaSave.slicesByteIndex", got: unsafe.Offsetof(f.arenaSave) + unsafe.Offsetof(f.arenaSave.slicesByteIndex), expect: 480},
		{name: "arenaSave.upvalueCellIndex", got: unsafe.Offsetof(f.arenaSave) + unsafe.Offsetof(f.arenaSave.upvalueCellIndex), expect: 488},
		{name: "arenaSave.upvalueReferenceIndex", got: unsafe.Offsetof(f.arenaSave) + unsafe.Offsetof(f.arenaSave.upvalueReferenceIndex), expect: 496},
		{name: "pc", got: unsafe.Offsetof(f.programCounter), expect: 504},
		{name: "deferBase", got: unsafe.Offsetof(f.deferBase), expect: 512},
		{name: "hasGeneralAlloc", got: unsafe.Offsetof(f.hasGeneralAlloc), expect: 520},
	}

	for _, tt := range tests {
		if tt.got != tt.expect {
			t.Errorf("callFrame.%s offset = %d, want %d", tt.name, tt.got, tt.expect)
		}
	}

	if sz := unsafe.Sizeof(f); sz != 528 {
		t.Errorf("callFrame size = %d, want 528", sz)
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

		{name: "SourceType", got: unsafe.Offsetof(v.sourceType), expect: 0},
		{name: "UpvalueIndex", got: unsafe.Offsetof(v.upvalueIndex), expect: 16},
		{name: "SpillSlot", got: unsafe.Offsetof(v.spillSlot), expect: 24},
		{name: "Register", got: unsafe.Offsetof(v.register), expect: 26},
		{name: "Kind", got: unsafe.Offsetof(v.kind), expect: 27},
		{name: "IsUpvalue", got: unsafe.Offsetof(v.isUpvalue), expect: 28},
		{name: "IsIndirect", got: unsafe.Offsetof(v.isIndirect), expect: 29},
		{name: "OriginalKind", got: unsafe.Offsetof(v.originalKind), expect: 30},
		{name: "IsCaptured", got: unsafe.Offsetof(v.isCaptured), expect: 31},
		{name: "IsSpilled", got: unsafe.Offsetof(v.isSpilled), expect: 32},
	}

	for _, tt := range tests {
		if tt.got != tt.expect {
			t.Errorf("varLocation.%s offset = %d, want %d", tt.name, tt.got, tt.expect)
		}
	}

	if sz := unsafe.Sizeof(v); sz != 40 {
		t.Errorf("varLocation size = %d, want 40", sz)
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
		{name: "returnDestinationRegister", got: unsafe.Offsetof(ci.returnDestinationRegister), expect: 216},
		{name: "returnDestinationPointer", got: unsafe.Offsetof(ci.returnDestinationPointer), expect: 224},
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
		{name: "calleeStringConstants", got: unsafe.Offsetof(ci.calleeStringConstants), expect: 496},
		{name: "calleeBoolConstants", got: unsafe.Offsetof(ci.calleeBoolConstants), expect: 504},
		{name: "calleeGeneralCount", got: unsafe.Offsetof(ci.calleeGeneralCount), expect: 512},
		{name: "generalArgumentCount", got: unsafe.Offsetof(ci.generalArgumentCount), expect: 520},
		{name: "generalArgumentSources", got: unsafe.Offsetof(ci.generalArgumentSources), expect: 528},
		{name: "calleeSliceByteCount", got: unsafe.Offsetof(ci.calleeSliceByteCount), expect: 592},
		{name: "sliceByteArgumentCount", got: unsafe.Offsetof(ci.sliceByteArgumentCount), expect: 600},
		{name: "sliceByteArgumentSources", got: unsafe.Offsetof(ci.sliceByteArgumentSources), expect: 608},
	}

	for _, tt := range tests {
		if tt.got != tt.expect {
			t.Errorf("asmCallInfo.%s offset = %d, want %d", tt.name, tt.got, tt.expect)
		}
	}

	if sz := unsafe.Sizeof(ci); sz != 1024 {
		t.Errorf("asmCallInfo size = %d, want 1024", sz)
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
		{name: "stringConstantsBase", got: unsafe.Offsetof(ds.stringConstantsBase), expect: 32},
		{name: "boolConstantsBase", got: unsafe.Offsetof(ds.boolConstantsBase), expect: 40},
	}

	for _, tt := range tests {
		if tt.got != tt.expect {
			t.Errorf("asmDispatchSave.%s offset = %d, want %d", tt.name, tt.got, tt.expect)
		}
	}

	if sz := unsafe.Sizeof(ds); sz != 64 {
		t.Errorf("asmDispatchSave size = %d, want 64", sz)
	}
}

func TestStructFieldLayoutSize(t *testing.T) {
	t.Parallel()
	var layout structFieldLayout
	if sz := unsafe.Sizeof(layout); sz != 16 {
		t.Errorf("structFieldLayout size = %d, want 16 (LAYOUT_SIZE_SHIFT=4)", sz)
	}
}

func TestStructFieldLayoutOffsets(t *testing.T) {
	t.Parallel()
	var layout structFieldLayout
	tests := []struct {
		name   string
		got    uintptr
		expect uintptr
	}{
		{name: "Offset", got: unsafe.Offsetof(layout.Offset), expect: 0},
		{name: "TypeIndex", got: unsafe.Offsetof(layout.TypeIndex), expect: 4},
		{name: "Kind", got: unsafe.Offsetof(layout.Kind), expect: 11},
		{name: "RegisterKind", got: unsafe.Offsetof(layout.RegisterKind), expect: 12},
		{name: "Flags", got: unsafe.Offsetof(layout.Flags), expect: 13},
		{name: "FieldTypeIndex", got: unsafe.Offsetof(layout.FieldTypeIndex), expect: 14},
	}
	for _, tt := range tests {
		if tt.got != tt.expect {
			t.Errorf("structFieldLayout.%s offset = %d, want %d", tt.name, tt.got, tt.expect)
		}
	}
}

func TestReflectKindConstants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		kind   reflect.Kind
		expect uintptr
	}{
		{name: "Invalid", kind: reflect.Invalid, expect: 0},
		{name: "Bool", kind: reflect.Bool, expect: 1},
		{name: "Int", kind: reflect.Int, expect: 2},
		{name: "Int8", kind: reflect.Int8, expect: 3},
		{name: "Int16", kind: reflect.Int16, expect: 4},
		{name: "Int32", kind: reflect.Int32, expect: 5},
		{name: "Int64", kind: reflect.Int64, expect: 6},
		{name: "Uint", kind: reflect.Uint, expect: 7},
		{name: "Uint8", kind: reflect.Uint8, expect: 8},
		{name: "Uint16", kind: reflect.Uint16, expect: 9},
		{name: "Uint32", kind: reflect.Uint32, expect: 10},
		{name: "Uint64", kind: reflect.Uint64, expect: 11},
		{name: "Uintptr", kind: reflect.Uintptr, expect: 12},
		{name: "Float32", kind: reflect.Float32, expect: 13},
		{name: "Float64", kind: reflect.Float64, expect: 14},
		{name: "Interface", kind: reflect.Interface, expect: 20},
		{name: "Pointer", kind: reflect.Pointer, expect: 22},
	}
	for _, tt := range tests {
		if uintptr(tt.kind) != tt.expect {
			t.Errorf("reflect.%s = %d, want %d (Phase B ASM REFLECT_%s define needs update)",
				tt.name, tt.kind, tt.expect, tt.name)
		}
	}
}

func TestReflectValueFlagConstants(t *testing.T) {
	t.Parallel()
	if flagKindMask != 0x1F {
		t.Errorf("flagKindMask = %#x, want 0x1F", flagKindMask)
	}
	if flagIndir != 0x80 {
		t.Errorf("flagIndir = %#x, want 0x80", flagIndir)
	}
	if flagAddr != 0x100 {
		t.Errorf("flagAddr = %#x, want 0x100", flagAddr)
	}
}
