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

//go:build fuzz

package interp_domain

import (
	"testing"
)

func FuzzDecodeInstruction(f *testing.F) {
	f.Add(byte(opNop), byte(0), byte(0), byte(0))
	f.Add(byte(opAddInt), byte(1), byte(2), byte(3))
	f.Add(byte(opLoadIntConst), byte(0), byte(0), byte(0))
	f.Add(byte(opDrillTier1), byte(subOpJump), byte(0), byte(0))
	f.Add(byte(opDrillTier1), byte(subOpDrillTier2), byte(subOpTier2Return), byte(1))
	f.Add(byte(opDrillTier1), byte(subOpDrillTier2), byte(subOpTier2DrillTier3), byte(subOpTier3Nop))
	f.Add(byte(opExt), byte(0xFF), byte(0xFF), byte(0xFF))

	f.Fuzz(func(t *testing.T, op, a, b, c byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("decoder panicked: %v", r)
			}
		}()
		instr := instruction{op: opcode(op), a: a, b: b, c: c}
		_ = instructionDisplayName(instr)
		_ = instr.String()
		_ = instr.signedOffset()
		_ = instr.wideIndex()
		shape := operandShapes[instr.op]
		_ = shape.flags
		_ = shape.a
		_ = shape.b
		_ = shape.c

		cf := &CompiledFunction{name: "fuzz", body: []instruction{instr}}
		_ = cf.Disassemble()
	})
}
