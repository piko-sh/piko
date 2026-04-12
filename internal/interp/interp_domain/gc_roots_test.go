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
)

type rootCounter struct {
	stringsSeen       []string
	reflectValuesSeen int
	slicesIntSeen     int
	slicesFloatSeen   int
	slicesStringSeen  int
	slicesBoolSeen    int
	slicesUintSeen    int
	slicesByteSeen    int
	anysSeen          int
	upvalueCellsSeen  int
	closuresSeen      int
}

func (c *rootCounter) visitor() rootVisitor {
	return rootVisitor{
		visitString:         func(s string) { c.stringsSeen = append(c.stringsSeen, s) },
		visitReflectValue:   func(_ reflect.Value) { c.reflectValuesSeen++ },
		visitSliceInt:       func(_ []int64) { c.slicesIntSeen++ },
		visitSliceFloat:     func(_ []float64) { c.slicesFloatSeen++ },
		visitSliceString:    func(_ []string) { c.slicesStringSeen++ },
		visitSliceBool:      func(_ []bool) { c.slicesBoolSeen++ },
		visitSliceUint:      func(_ []uint64) { c.slicesUintSeen++ },
		visitSliceByte:      func(_ []byte) { c.slicesByteSeen++ },
		visitAny:            func(_ any) { c.anysSeen++ },
		visitUpvalueCell:    func(_ *upvalueCell) { c.upvalueCellsSeen++ },
		visitRuntimeClosure: func(_ *runtimeClosure) { c.closuresSeen++ },
	}
}

func newRootsTestVM(t *testing.T) *VM {
	t.Helper()
	return &VM{
		globals:   &globalStore{},
		arena:     newRegisterArena(),
		callStack: []callFrame{},
	}
}

func TestWalkRoots_EmptyVM(t *testing.T) {
	vm := newRootsTestVM(t)
	counter := &rootCounter{}
	vm.walkRoots(counter.visitor())
	if counter.reflectValuesSeen != 0 || len(counter.stringsSeen) != 0 {
		t.Errorf("empty VM should visit zero roots, saw strings=%d reflectValues=%d",
			len(counter.stringsSeen), counter.reflectValuesSeen)
	}
}

func TestWalkRoots_GlobalStrings(t *testing.T) {
	vm := newRootsTestVM(t)
	vm.globals.strings = []string{"alpha", "beta", "gamma"}
	counter := &rootCounter{}
	vm.walkRoots(counter.visitor())
	if len(counter.stringsSeen) != 3 {
		t.Errorf("expected 3 global strings visited, got %d", len(counter.stringsSeen))
	}
}

func TestWalkRoots_GlobalGeneral(t *testing.T) {
	vm := newRootsTestVM(t)
	vm.globals.general = []reflect.Value{
		reflect.ValueOf(42),
		reflect.ValueOf("hello"),
	}
	counter := &rootCounter{}
	vm.walkRoots(counter.visitor())
	if counter.reflectValuesSeen != 2 {
		t.Errorf("expected 2 global generals visited, got %d", counter.reflectValuesSeen)
	}
}

func TestWalkRoots_FrameRegisters(t *testing.T) {
	vm := newRootsTestVM(t)
	vm.callStack = []callFrame{{
		registers: Registers{
			strings:    []string{"r0", "r1"},
			general:    []reflect.Value{reflect.ValueOf(1), reflect.ValueOf(2)},
			slicesInt:  [][]int64{{1, 2, 3}, {4, 5}},
			slicesByte: [][]byte{[]byte("hi"), nil},
		},
	}}
	vm.framePointer = 0
	counter := &rootCounter{}
	vm.walkRoots(counter.visitor())
	if len(counter.stringsSeen) != 2 {
		t.Errorf("expected 2 frame strings, got %d", len(counter.stringsSeen))
	}
	if counter.reflectValuesSeen != 2 {
		t.Errorf("expected 2 frame generals, got %d", counter.reflectValuesSeen)
	}
	if counter.slicesIntSeen != 2 {
		t.Errorf("expected 2 frame []int64 slices, got %d", counter.slicesIntSeen)
	}
	if counter.slicesByteSeen != 2 {
		t.Errorf("expected 2 frame []byte slices, got %d", counter.slicesByteSeen)
	}
}

func TestWalkRoots_FrameUpvalues(t *testing.T) {
	vm := newRootsTestVM(t)
	cell1 := &upvalueCell{}
	cell2 := &upvalueCell{}
	vm.callStack = []callFrame{{
		upvalues: []upvalue{{value: cell1}, {value: cell2}},
	}}
	vm.framePointer = 0
	counter := &rootCounter{}
	vm.walkRoots(counter.visitor())
	if counter.upvalueCellsSeen != 2 {
		t.Errorf("expected 2 upvalue cells visited, got %d", counter.upvalueCellsSeen)
	}
}

func TestWalkRoots_FrameSharedCells(t *testing.T) {
	vm := newRootsTestVM(t)
	cell1 := &upvalueCell{}
	cell2 := &upvalueCell{}
	cell3 := &upvalueCell{}
	scm := &sharedCellMap{
		inlineLen: 2,
		overflow:  []sharedCellEntry{{cell: cell3, key: 99}},
	}
	scm.inlineCells[0] = cell1
	scm.inlineCells[1] = cell2
	vm.callStack = []callFrame{{
		sharedCells: scm,
	}}
	vm.framePointer = 0
	counter := &rootCounter{}
	vm.walkRoots(counter.visitor())
	if counter.upvalueCellsSeen != 3 {
		t.Errorf("expected 3 cells (2 inline + 1 overflow), got %d", counter.upvalueCellsSeen)
	}
}

func TestWalkRoots_DeferStack(t *testing.T) {
	vm := newRootsTestVM(t)
	closure := &runtimeClosure{}
	vm.deferStack = []deferredCall{{
		function:  closure,
		arguments: []reflect.Value{reflect.ValueOf("a"), reflect.ValueOf("b")},
	}}
	counter := &rootCounter{}
	vm.walkRoots(counter.visitor())
	if counter.closuresSeen != 1 {
		t.Errorf("expected 1 defer closure, got %d", counter.closuresSeen)
	}
	if counter.reflectValuesSeen != 2 {
		t.Errorf("expected 2 defer args as reflectValues, got %d", counter.reflectValuesSeen)
	}
}

func TestWalkRoots_VMState(t *testing.T) {
	vm := newRootsTestVM(t)
	vm.panicValue = "oops"
	vm.evalResult = 42
	vm.evalAllResults = []any{"x", "y", "z"}
	counter := &rootCounter{}
	vm.walkRoots(counter.visitor())
	if counter.anysSeen != 5 {
		t.Errorf("expected 5 any roots (panic + result + 3 results), got %d", counter.anysSeen)
	}
}

func TestWalkRoots_ClosureCache(t *testing.T) {
	vm := newRootsTestVM(t)
	closure := &runtimeClosure{}
	vm.closureCache = []reflect.Value{
		reflect.ValueOf(closure),
		{},
		reflect.ValueOf(closure),
	}
	counter := &rootCounter{}
	vm.walkRoots(counter.visitor())
	if counter.reflectValuesSeen != 2 {
		t.Errorf("expected 2 valid cached closures, got %d", counter.reflectValuesSeen)
	}
}

func TestWalkRoots_AllSourcesCombined(t *testing.T) {
	vm := newRootsTestVM(t)
	cell := &upvalueCell{}
	closure := &runtimeClosure{}
	vm.callStack = []callFrame{{
		registers: Registers{
			strings: []string{"frame-string"},
			general: []reflect.Value{reflect.ValueOf("g")},
		},
		upvalues: []upvalue{{value: cell}},
	}}
	vm.framePointer = 0
	vm.globals.strings = []string{"global-string"}
	vm.globals.general = []reflect.Value{reflect.ValueOf(1.5)}
	vm.deferStack = []deferredCall{{function: closure}}
	vm.panicValue = "panic"
	vm.closureCache = []reflect.Value{reflect.ValueOf(closure)}
	counter := &rootCounter{}
	vm.walkRoots(counter.visitor())
	if len(counter.stringsSeen) != 2 {
		t.Errorf("expected 2 strings (1 frame + 1 global), got %d", len(counter.stringsSeen))
	}
	if counter.reflectValuesSeen != 3 {
		t.Errorf("expected 3 reflect.Values (1 frame + 1 global + 1 cache), got %d", counter.reflectValuesSeen)
	}
	if counter.upvalueCellsSeen != 1 {
		t.Errorf("expected 1 upvalue cell, got %d", counter.upvalueCellsSeen)
	}
	if counter.closuresSeen != 1 {
		t.Errorf("expected 1 defer closure, got %d", counter.closuresSeen)
	}
	if counter.anysSeen != 1 {
		t.Errorf("expected 1 panic any, got %d", counter.anysSeen)
	}
}

func TestWalkRoots_OnlyActiveFrames(t *testing.T) {
	vm := newRootsTestVM(t)
	vm.callStack = []callFrame{
		{registers: Registers{strings: []string{"active-frame-0"}}},
		{registers: Registers{strings: []string{"active-frame-1"}}},
		{registers: Registers{strings: []string{"DEAD-frame-2"}}},
	}
	vm.framePointer = 1
	counter := &rootCounter{}
	vm.walkRoots(counter.visitor())
	if len(counter.stringsSeen) != 2 {
		t.Errorf("expected 2 active-frame strings (framePointer=1), got %d", len(counter.stringsSeen))
	}
	for _, s := range counter.stringsSeen {
		if s == "DEAD-frame-2" {
			t.Error("walked dead frame above framePointer")
		}
	}
}

func TestWalkRoots_NilCallbacks(t *testing.T) {
	vm := newRootsTestVM(t)
	vm.globals.strings = []string{"a", "b"}
	vm.globals.general = []reflect.Value{reflect.ValueOf(1)}
	vm.walkRoots(rootVisitor{})
}
