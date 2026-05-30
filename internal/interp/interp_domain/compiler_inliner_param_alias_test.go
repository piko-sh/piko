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
	"context"
	"fmt"
	"testing"
)

func TestInlinerPreservesByValueParamSemantics_GenericToGeneric(t *testing.T) {
	t.Parallel()
	const src = `package main

type Number interface {
	~int | ~float64
}

func tripled[T Number](v T) T {
	return v + v + v
}

func sevenTimes[T Number](v T) T {
	return tripled(v) + tripled(v) + v
}

func entrypoint() int {
	return sevenTimes(9)
}
`
	service := NewService()
	cfs, err := service.CompileFileSet(context.Background(), map[string]string{"main.go": src})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.ExecuteEntrypoint(context.Background(), cfs, "entrypoint")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(result) != "63" {
		t.Fatalf("sevenTimes(9) = %v, want 63 (tripled(9)+tripled(9)+9 = 27+27+9); "+
			"189 indicates the inliner is clobbering the caller's parameter via "+
			"the callee's return-prep MOVE", result)
	}
}

func TestInlinerPreservesByValueParamSemantics_StructArgument(t *testing.T) {
	t.Parallel()
	const src = `package main

type Box struct {
	N int
}

func mutate(b Box) {
	b.N = 999
}

func entrypoint() int {
	b := Box{N: 7}
	mutate(b)
	return b.N
}
`
	service := NewService()
	cfs, err := service.CompileFileSet(context.Background(), map[string]string{"main.go": src})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.ExecuteEntrypoint(context.Background(), cfs, "entrypoint")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(result) != "7" {
		t.Fatalf("b.N after mutate(b) = %v, want 7 (Go's by-value parameter "+
			"semantics); 999 indicates the inliner is letting the callee's "+
			"SET_FIELD on its struct param slot leak into the caller's struct", result)
	}
}

func TestInlinerPreservesByValueParamSemantics_ValueReceiver(t *testing.T) {
	t.Parallel()
	const src = `package main

type Box struct {
	N int
}

func (b Box) MutateLocal() int {
	b.N = 99
	return b.N
}

func entrypoint() int {
	p := &Box{N: 7}
	r := p.MutateLocal()
	return p.N*100 + r
}
`
	service := NewService()
	cfs, err := service.CompileFileSet(context.Background(), map[string]string{"main.go": src})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.ExecuteEntrypoint(context.Background(), cfs, "entrypoint")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(result) != "799" {
		t.Fatalf("p.N*100+r = %v, want 799 (p.N stays 7, r=99); 9999 indicates "+
			"the value receiver copy was elided across the inline boundary",
			result)
	}
}

func TestInlinerHandlesRecursiveStructFieldAccess(t *testing.T) {
	t.Parallel()
	const src = `package main

type Node[T any] struct {
	Value T
	Next  *Node[T]
}

func listLen[T any](head *Node[T]) int {
	n := 0
	for cur := head; cur != nil; cur = cur.Next {
		n++
	}
	return n
}

func entrypoint() int {
	c := &Node[string]{Value: "c"}
	b := &Node[string]{Value: "b", Next: c}
	a := &Node[string]{Value: "a", Next: b}
	count := listLen(a)
	return count + 0 // suppress tail call so the inliner gets a shot
}
`
	service := NewService()
	cfs, err := service.CompileFileSet(context.Background(), map[string]string{"main.go": src})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.ExecuteEntrypoint(context.Background(), cfs, "entrypoint")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(result) != "3" {
		t.Fatalf("listLen(a→b→c) = %v, want 3; 1 indicates GET_FIELD's "+
			"destination operand isn't being remapped through the inliner "+
			"and so each `cur.Next` read overwrites the caller's `a` and "+
			"reads the never-written remapped slot for cur", result)
	}
}

func TestInlinerReadOnlyParamElidesPreCopy(t *testing.T) {
	t.Parallel()
	callee := &CompiledFunction{
		body: []instruction{
			makeInstruction(opAddInt, 2, 0, 1),
			makeInstruction(opDrillTier1, byte(subOpMoveInt), 0, 2),
			makeInstruction(
				opDrillTier1,
				byte(subOpDrillTier2),
				byte(subOpTier2Return),
				1,
			),
		},
		parameterKinds: []registerKind{registerInt, registerInt},
		resultKinds:    []registerKind{registerInt},
		numRegisters:   [NumRegisterKinds]uint32{registerInt: 3},
	}
	caller := &CompiledFunction{
		body: []instruction{
			makeOpCallSlot(0),
			makeInstruction(opNop, 0, 0, 0),
		},
		callSites: []callSite{
			{
				cachedCallee: callee,
				arguments: []varLocation{
					{register: 5, kind: registerInt},
					{register: 6, kind: registerInt},
				},
				returns: []varLocation{{register: 7, kind: registerInt}},
			},
		},
		numRegisters: [NumRegisterKinds]uint32{registerInt: 8},
	}
	result := trySpliceCall(caller, 0)
	if !result.spliced {
		t.Fatalf("splice failed: %v", result.reason)
	}

	if caller.numRegisters[registerInt] != 10 {
		t.Fatalf("numRegisters[int] %d want 10 (8 caller + 1 fresh param + 1 local)",
			caller.numRegisters[registerInt])
	}

	var addInt instruction
	for _, instr := range caller.body {
		if instr.op == opAddInt {
			addInt = instr
			break
		}
	}
	if addInt.op != opAddInt {
		t.Fatalf("expected opAddInt in inlined body, body=%+v", caller.body)
	}
	if addInt.a != 9 {
		t.Fatalf("opAddInt dst %d want 9 (fresh local for callee slot 2)", addInt.a)
	}
	if addInt.b != 8 {
		t.Fatalf("opAddInt srcA %d want 8 (fresh param slot for callee slot 0)", addInt.b)
	}
	if addInt.c != 6 {
		t.Fatalf("opAddInt srcB %d want 6 (alias to caller arg.register for callee slot 1, no pre-copy)", addInt.c)
	}

	preCopyCount := 0
	for _, instr := range caller.body {
		if instr.op == opDrillTier1 && subOpcode(instr.a) == subOpMoveInt &&
			instr.b == 8 && instr.c == 5 {
			preCopyCount++
		}
	}
	if preCopyCount != 1 {
		t.Fatalf("pre-copy MOVE_INT dst=8 src=5 count %d want 1 (read-only param "+
			"must NOT emit a pre-copy for slot 1); body=%+v", preCopyCount, caller.body)
	}

	for _, instr := range caller.body {
		if instr.op == opDrillTier1 && subOpcode(instr.a) == subOpMoveInt &&
			instr.c == 6 {
			t.Fatalf("found pre-copy from caller arg.register 6 - read-only " +
				"param 1 should have skipped pre-copy")
		}
	}
}

func TestInlinerWritesThroughParamForceFreshSlot(t *testing.T) {
	t.Parallel()

	callee := &CompiledFunction{
		body: []instruction{
			makeInstruction(opSetField, 0, 0, 1),
			makeInstruction(
				opDrillTier1,
				byte(subOpDrillTier2),
				byte(subOpTier2DrillTier3),
				byte(subOpTier3ReturnVoid),
			),
		},
		parameterKinds: []registerKind{registerGeneral, registerGeneral},
		numRegisters: [NumRegisterKinds]uint32{
			registerGeneral: 2,
		},
	}
	caller := &CompiledFunction{
		body: []instruction{
			makeOpCallSlot(0),
			makeInstruction(opNop, 0, 0, 0),
		},
		callSites: []callSite{
			{
				cachedCallee: callee,
				arguments: []varLocation{
					{register: 3, kind: registerGeneral},
					{register: 2, kind: registerGeneral},
				},
			},
		},
		numRegisters: [NumRegisterKinds]uint32{
			registerGeneral: 4,
		},
	}
	result := trySpliceCall(caller, 0)
	if !result.spliced {
		t.Fatalf("splice failed: %v", result.reason)
	}

	if caller.numRegisters[registerGeneral] != 5 {
		t.Fatalf("numRegisters[general] %d want 5 (caller's 4 + 1 fresh param slot)",
			caller.numRegisters[registerGeneral])
	}

	foundPreCopy := false
	for _, instr := range caller.body {
		if instr.op == opMoveGeneral && instr.a == 4 && instr.b == 3 {
			foundPreCopy = true
			break
		}
	}
	if !foundPreCopy {
		t.Fatalf("expected opMoveGeneral dst=4 src=3 (pre-copy for struct param), "+
			"body=%+v", caller.body)
	}

	var setField instruction
	for _, instr := range caller.body {
		if instr.op == opSetField {
			setField = instr
			break
		}
	}
	if setField.op != opSetField {
		t.Fatalf("expected opSetField in inlined body, body=%+v", caller.body)
	}
	if setField.a != 4 {
		t.Fatalf("inlined SET_FIELD struct operand %d want 4 (fresh slot); "+
			"3 would mean caller's struct mutation", setField.a)
	}

	if setField.c != 2 {
		t.Fatalf("inlined SET_FIELD value operand %d want 2 (aliased to caller arg, no pre-copy)",
			setField.c)
	}
}
