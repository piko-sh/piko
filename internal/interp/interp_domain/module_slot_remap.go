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
	"fmt"

	"piko.sh/piko/wdk/safeconv"
)

// relativiseGlobalOperands rewrites global operands as bundle-relative.
//
// Walks every CompiledFunction reachable from root (including nested functions and
// variableInitFunction closures) and subtracts the per-kind base from each
// opGetGlobal{,Wide} / opSetGlobal{,Wide} operand. Used at Service.PackageModule time to
// convert this-package slot indices (which are absolute against the originating Service's
// globalStore) into bundle-relative indices that work in any target Service after
// setGlobalBases reinstates a load-time base.
//
// An operand whose original value is less than the bundle's base belongs to a
// previously-compiled package and would not be valid after load; that is a compiler
// invariant violation and a clear error is returned rather than silently corrupting
// bytecode.
//
// Takes root (*CompiledFunction) which is the root of the function graph to walk.
// Takes base (SlotAllocation) which is the per-kind base offset to subtract from each
// operand.
//
// Returns error when an operand precedes its kind's base or a wide instruction is missing
// its extension word.
func relativiseGlobalOperands(root *CompiledFunction, base SlotAllocation) error {
	if root == nil {
		return nil
	}
	visited := make(map[*CompiledFunction]struct{})
	return walkAndRelativise(root, base, visited)
}

// walkAndRelativise rewrites global operands in fn and its descendants.
//
// Takes fn (*CompiledFunction) which is the function to walk.
// Takes base (SlotAllocation) which is the per-kind base offset to subtract.
// Takes visited (map[*CompiledFunction]struct{}) which tracks already processed functions
// to break cycles.
//
// Returns error when any global operand rewrite fails.
func walkAndRelativise(fn *CompiledFunction, base SlotAllocation, visited map[*CompiledFunction]struct{}) error {
	if fn == nil {
		return nil
	}
	if _, seen := visited[fn]; seen {
		return nil
	}
	visited[fn] = struct{}{}

	for i := 0; i < len(fn.body); i++ {
		skip, err := relativiseGlobalInstruction(fn, i, base)
		if err != nil {
			return err
		}
		i += skip
	}

	if err := walkAndRelativise(fn.variableInitFunction, base, visited); err != nil {
		return err
	}
	for _, nested := range fn.functions {
		if err := walkAndRelativise(nested, base, visited); err != nil {
			return err
		}
	}
	return nil
}

// relativiseGlobalInstruction rewrites one global-access instruction.
//
// Rewrites the instruction at index i in fn.body to bundle-relative form and reports how
// many extra words (a wide-form extension) it consumed so the caller can advance its
// index past them.
//
// Takes fn (*CompiledFunction) which owns the instruction stream.
// Takes i (int) which is the instruction index to rewrite.
// Takes base (SlotAllocation) which is the per-kind base offset to subtract.
//
// Returns int which is the number of extra extension words consumed.
// Returns error when the operand precedes its kind's base or the wide form is missing an
// extension word.
func relativiseGlobalInstruction(fn *CompiledFunction, i int, base SlotAllocation) (int, error) {
	ins := fn.body[i]
	switch opcode(ins.op) {
	case opGetGlobal, opSetGlobal:
		kind := registerKind(ins.c)
		if int(kind) >= NumGlobalRegisterKinds {
			return 0, fmt.Errorf("interp_domain: relativise: kind %d out of range in %s", kind, fn.name)
		}
		b := base[kind]
		if uint16(ins.b) < b {
			return 0, fmt.Errorf("interp_domain: relativise: %s references global %d but bundle base is %d (kind %d)", fn.name, ins.b, b, kind)
		}
		fn.body[i].b = safeconv.MustIntToUint8(int(uint16(ins.b) - b))
		return 0, nil
	case opGetGlobalWide, opSetGlobalWide:
		if i+1 >= len(fn.body) {
			return 0, fmt.Errorf("interp_domain: relativise: wide op missing extension word at pc %d in %s", i, fn.name)
		}
		ext := fn.body[i+1]
		kind := registerKind(ins.c)
		if int(kind) >= NumGlobalRegisterKinds {
			return 0, fmt.Errorf("interp_domain: relativise: kind %d out of range in %s", kind, fn.name)
		}
		b := base[kind]
		encoded := uint16(ext.a) | uint16(ext.b)<<wideBitShift
		if encoded < b {
			return 0, fmt.Errorf("interp_domain: relativise: %s references global %d (wide) but bundle base is %d (kind %d)", fn.name, encoded, b, kind)
		}
		rel := encoded - b
		fn.body[i+1].a = uint8(rel & 0xff)
		fn.body[i+1].b = uint8(rel >> wideBitShift)
		return 1, nil
	default:
		return 0, nil
	}
}

// setGlobalBases assigns bases to every reachable CompiledFunction.
//
// Stamps each function so the VM's global-access handlers offset each encoded operand at
// dispatch time. Called by Service.LoadModule after reserving slots in the target
// Service's globalStore.
//
// Functions are visited transitively (nested closures, variableInitFunction). The same
// *SlotAllocation pointer is installed everywhere so the VM hot path is a single
// indirection.
//
// Takes root (*CompiledFunction) which is the root of the function graph to walk.
// Takes bases (*SlotAllocation) which is the per-kind base table to install on each
// function.
func setGlobalBases(root *CompiledFunction, bases *SlotAllocation) {
	if root == nil || bases == nil {
		return
	}
	visited := make(map[*CompiledFunction]struct{})
	walkAndSetBases(root, bases, visited)
}

// walkAndSetBases stamps bases on fn and its descendants.
//
// Takes fn (*CompiledFunction) which is the function to stamp.
// Takes bases (*SlotAllocation) which is the per-kind base table.
// Takes visited (map[*CompiledFunction]struct{}) which tracks already processed functions
// to break cycles.
func walkAndSetBases(fn *CompiledFunction, bases *SlotAllocation, visited map[*CompiledFunction]struct{}) {
	if fn == nil {
		return
	}
	if _, seen := visited[fn]; seen {
		return
	}
	visited[fn] = struct{}{}
	fn.globalBases = bases
	walkAndSetBases(fn.variableInitFunction, bases, visited)
	for _, nested := range fn.functions {
		walkAndSetBases(nested, bases, visited)
	}
}
