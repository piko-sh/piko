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
)

func makeDoubleCallee() *CompiledFunction {
	return &CompiledFunction{
		body: []instruction{
			makeInstruction(opAddInt, 0, 0, 0),
			makeInstruction(
				opDrillTier1,
				byte(subOpDrillTier2),
				byte(subOpTier2Return),
				1,
			),
		},
		parameterKinds: []registerKind{registerInt},
		resultKinds:    []registerKind{registerInt},
		numRegisters:   [NumRegisterKinds]uint32{registerInt: 1},
	}
}

func makeVoidNoopCallee() *CompiledFunction {
	return &CompiledFunction{
		body: []instruction{
			makeInstruction(
				opDrillTier1,
				byte(subOpDrillTier2),
				byte(subOpTier2DrillTier3),
				byte(subOpTier3ReturnVoid),
			),
		},
		numRegisters: [NumRegisterKinds]uint32{},
	}
}

func makeOpCallSlot(siteIdx uint16) instruction {
	return makeInstruction(opCall, 0, byte(siteIdx&0xFF), byte(siteIdx>>8))
}

func TestSplice_VoidNoop_ReplacesOpCallWithJump(t *testing.T) {
	t.Parallel()
	callee := makeVoidNoopCallee()
	siteIdx := uint16(0)
	caller := &CompiledFunction{
		body: []instruction{
			makeOpCallSlot(siteIdx),
			makeInstruction(opNop, 0, 0, 0),
		},
		callSites: []callSite{
			{
				cachedCallee: callee,
				arguments:    nil,
				returns:      nil,
			},
		},
	}
	result := trySpliceCall(caller, siteIdx)
	if !result.spliced {
		t.Fatalf("splice failed: %v", result.reason)
	}

	if len(caller.body) != 5 {
		t.Fatalf("body length %d want 5", len(caller.body))
	}
	got := caller.body[0]
	if got.op != opDrillTier1 || subOpcode(got.a) != subOpJump || decodeJumpOffset(got) != 2 {
		t.Fatalf("pc 0: want subOpJump offset=2, got op=%v a=%v offset=%d",
			got.op, got.a, decodeJumpOffset(got))
	}
	got = caller.body[2]
	if got.op != opDrillTier1 || subOpcode(got.a) != subOpJump || decodeJumpOffset(got) != 2 {
		t.Fatalf("pc 2 (skip-jump): want subOpJump offset=2, got op=%v a=%v offset=%d",
			got.op, got.a, decodeJumpOffset(got))
	}
	got = caller.body[3]
	if got.op != opDrillTier1 || subOpcode(got.a) != subOpJump || decodeJumpOffset(got) != 0 {
		t.Fatalf("pc 3 (return → retPrep): want subOpJump offset=0, got op=%v a=%v offset=%d",
			got.op, got.a, decodeJumpOffset(got))
	}
	got = caller.body[4]
	if got.op != opDrillTier1 || subOpcode(got.a) != subOpJump || decodeJumpOffset(got) != -4 {
		t.Fatalf("pc 4 (retPrep back-jump): want subOpJump offset=-4, got op=%v a=%v offset=%d",
			got.op, got.a, decodeJumpOffset(got))
	}
}

func TestSplice_DoubleCallee_ValueReturn(t *testing.T) {
	t.Parallel()
	callee := makeDoubleCallee()
	siteIdx := uint16(0)
	caller := &CompiledFunction{
		body: []instruction{
			makeOpCallSlot(siteIdx),
			makeInstruction(opNop, 0, 0, 0),
		},
		callSites: []callSite{
			{
				cachedCallee: callee,
				arguments:    []varLocation{{register: 5, kind: registerInt}},
				returns:      []varLocation{{register: 6, kind: registerInt}},
			},
		},
		numRegisters: [NumRegisterKinds]uint32{registerInt: 7},
	}
	result := trySpliceCall(caller, siteIdx)
	if !result.spliced {
		t.Fatalf("splice failed: %v", result.reason)
	}

	if len(caller.body) != 8 {
		t.Fatalf("body length %d want 8; body=%+v", len(caller.body), caller.body)
	}
	if caller.numRegisters[registerInt] != 8 {
		t.Fatalf("numRegisters[int] %d want 8 (caller's 7 + 1 fresh param slot)",
			caller.numRegisters[registerInt])
	}

	got := caller.body[3]
	if got.op != opDrillTier1 || subOpcode(got.a) != subOpMoveInt {
		t.Fatalf("pc 3 want pre-copy subOpMoveInt, got op=%v a=%v", got.op, got.a)
	}
	if got.b != 7 || got.c != 5 {
		t.Fatalf("pc 3 (pre-copy) want dst=7 src=5, got b=%d c=%d", got.b, got.c)
	}

	got = caller.body[4]
	if got.op != opAddInt || got.a != 7 || got.b != 7 || got.c != 7 {
		t.Fatalf("pc 4 want opAddInt 7,7,7 (fresh slot), got op=%v a=%v b=%v c=%v",
			got.op, got.a, got.b, got.c)
	}

	got = caller.body[6]
	if got.op != opDrillTier1 || subOpcode(got.a) != subOpMoveInt {
		t.Fatalf("pc 6 want subOpMoveInt, got op=%v a=%v", got.op, got.a)
	}
	if got.b != 6 || got.c != 7 {
		t.Fatalf("pc 6 (return move) want dst=6 src=7, got b=%d c=%d", got.b, got.c)
	}
	got = caller.body[7]
	if got.op != opDrillTier1 || subOpcode(got.a) != subOpJump || decodeJumpOffset(got) != -7 {
		t.Fatalf("pc 7 want subOpJump offset=-7, got op=%v a=%v offset=%d",
			got.op, got.a, decodeJumpOffset(got))
	}
}

func TestSplice_RemapsLocalRegisters(t *testing.T) {
	t.Parallel()
	callee := &CompiledFunction{
		body: []instruction{
			makeInstruction(opAddInt, 0, 0, 0),
			makeInstruction(
				opDrillTier1,
				byte(subOpDrillTier2),
				byte(subOpTier2DrillTier3),
				byte(subOpTier3ReturnVoid),
			),
		},
		numRegisters: [NumRegisterKinds]uint32{registerInt: 1},
	}
	caller := &CompiledFunction{
		body: []instruction{
			makeOpCallSlot(0),
			makeInstruction(opNop, 0, 0, 0),
		},
		callSites: []callSite{
			{cachedCallee: callee},
		},
		numRegisters: [NumRegisterKinds]uint32{registerInt: 4},
	}
	result := trySpliceCall(caller, 0)
	if !result.spliced {
		t.Fatalf("splice failed: %v", result.reason)
	}

	if caller.numRegisters[registerInt] != 5 {
		t.Fatalf("caller numRegisters[int] = %d want 5", caller.numRegisters[registerInt])
	}

	got := caller.body[3]
	if got.op != opAddInt || got.a != 4 || got.b != 4 || got.c != 4 {
		t.Fatalf("appended opAddInt = {a:%d b:%d c:%d} want {4,4,4}", got.a, got.b, got.c)
	}
}

func TestSplice_LoadIntConst_MergesPool(t *testing.T) {
	t.Parallel()
	callee := &CompiledFunction{
		body: []instruction{
			makeInstruction(opLoadIntConst, 0, 0, 0),
			makeInstruction(
				opDrillTier1,
				byte(subOpDrillTier2),
				byte(subOpTier2DrillTier3),
				byte(subOpTier3ReturnVoid),
			),
		},
		intConstants: []int64{7},
		numRegisters: [NumRegisterKinds]uint32{registerInt: 1},
	}
	caller := &CompiledFunction{
		body: []instruction{
			makeOpCallSlot(0),
			makeInstruction(opNop, 0, 0, 0),
		},
		callSites: []callSite{
			{cachedCallee: callee},
		},
		intConstants: []int64{99},
		numRegisters: [NumRegisterKinds]uint32{registerInt: 3},
	}
	result := trySpliceCall(caller, 0)
	if !result.spliced {
		t.Fatalf("Phase 3 splice with constant pool failed: %v", result.reason)
	}

	if len(caller.intConstants) != 2 || caller.intConstants[1] != 7 {
		t.Fatalf("intConstants %v want [99 7]", caller.intConstants)
	}

	got := caller.body[3]
	if got.op != opLoadIntConst {
		t.Fatalf("pc 3 op = %v want opLoadIntConst", got.op)
	}
	if got.a != 3 {
		t.Fatalf("pc 3 dst reg %d want 3 (remapped local)", got.a)
	}
	idx := uint16(got.b) | uint16(got.c)<<8
	if idx != 1 {
		t.Fatalf("pc 3 const index %d want 1 (after pool merge)", idx)
	}
}

func TestSplice_InternalJumpPreserved(t *testing.T) {
	t.Parallel()
	lo, hi := splitJumpOffsetBytes(1)
	callee := &CompiledFunction{
		body: []instruction{
			makeInstruction(opJumpIfFalse, 0, lo, hi),
			makeInstruction(opAddInt, 0, 0, 0),
			makeInstruction(
				opDrillTier1,
				byte(subOpDrillTier2),
				byte(subOpTier2DrillTier3),
				byte(subOpTier3ReturnVoid),
			),
		},
		parameterKinds: []registerKind{registerInt},
		numRegisters:   [NumRegisterKinds]uint32{registerInt: 1},
	}
	caller := &CompiledFunction{
		body: []instruction{
			makeOpCallSlot(0),
			makeInstruction(opNop, 0, 0, 0),
		},
		callSites: []callSite{
			{
				cachedCallee: callee,
				arguments:    []varLocation{{register: 3, kind: registerInt}},
			},
		},
		numRegisters: [NumRegisterKinds]uint32{registerInt: 4},
	}
	result := trySpliceCall(caller, 0)
	if !result.spliced {
		t.Fatalf("splice failed: %v", result.reason)
	}

	got := caller.body[4]
	if got.op != opJumpIfFalse {
		t.Fatalf("pc 4 op = %v want opJumpIfFalse", got.op)
	}
	if got.a != 4 {
		t.Fatalf("pc 4 cond reg %d want 4 (remapped to fresh param slot)", got.a)
	}
	if decoded := decodeJumpOffset(got); decoded != 1 {
		t.Fatalf("pc 4 jump offset %d want 1 (preserved)", decoded)
	}

	pre := caller.body[3]
	if pre.op != opDrillTier1 || subOpcode(pre.a) != subOpMoveInt {
		t.Fatalf("pc 3 want pre-copy subOpMoveInt, got op=%v a=%v", pre.op, pre.a)
	}
	if pre.b != 4 || pre.c != 3 {
		t.Fatalf("pc 3 (pre-copy) want dst=4 src=3, got b=%d c=%d", pre.b, pre.c)
	}
}
