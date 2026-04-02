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

	"github.com/stretchr/testify/assert"
)

func mk(op opcode, a, b, c uint8) instruction {
	return makeInstruction(op, a, b, c)
}

func jumpOffset(offset int16) (uint8, uint8) {
	u := uint16(offset)
	return uint8(u), uint8(u >> 8)
}

func TestPeepholeOptimise(t *testing.T) {
	t.Parallel()
	lo, hi := jumpOffset(5)

	tests := []struct {
		name     string
		consts   []int64
		strConst []string
		body     []instruction
		expect   []opcode
	}{
		{
			name:   "LoadIntConst + SubInt -> SubIntConst",
			consts: []int64{42},
			body: []instruction{
				mk(opLoadIntConst, 3, 0, 0),
				mk(opSubInt, 1, 2, 3),
			},
			expect: []opcode{opSubIntConst, opNop},
		},
		{
			name:   "LoadIntConst + AddInt -> AddIntConst",
			consts: []int64{10},
			body: []instruction{
				mk(opLoadIntConst, 3, 0, 0),
				mk(opAddInt, 1, 2, 3),
			},
			expect: []opcode{opAddIntConst, opNop},
		},
		{
			name:   "LoadIntConst + MulInt -> MulIntConst",
			consts: []int64{7},
			body: []instruction{
				mk(opLoadIntConst, 3, 0, 0),
				mk(opMulInt, 1, 2, 3),
			},
			expect: []opcode{opMulIntConst, opNop},
		},
		{
			name:   "LoadIntConst + LeInt + JumpIfFalse -> LeIntConstJumpFalse",
			consts: []int64{100},
			body: []instruction{
				mk(opLoadIntConst, 3, 0, 0),
				mk(opLeInt, 4, 1, 3),
				mk(opJumpIfFalse, 4, lo, hi),
			},
			expect: []opcode{opLeIntConstJumpFalse, opExt, opNop},
		},
		{
			name:   "LoadIntConst + LtInt + JumpIfFalse -> LtIntConstJumpFalse",
			consts: []int64{50},
			body: []instruction{
				mk(opLoadIntConst, 3, 0, 0),
				mk(opLtInt, 4, 1, 3),
				mk(opJumpIfFalse, 4, lo, hi),
			},
			expect: []opcode{opLtIntConstJumpFalse, opExt, opNop},
		},
		{
			name:   "LoadIntConst + EqInt + JumpIfFalse -> EqIntConstJumpFalse",
			consts: []int64{3},
			body: []instruction{
				mk(opLoadIntConst, 3, 0, 0),
				mk(opEqInt, 4, 1, 3),
				mk(opJumpIfFalse, 4, lo, hi),
			},
			expect: []opcode{opEqIntConstJumpFalse, opExt, opNop},
		},
		{
			name:   "LoadIntConst + EqInt + JumpIfTrue -> EqIntConstJumpTrue",
			consts: []int64{3},
			body: []instruction{
				mk(opLoadIntConst, 3, 0, 0),
				mk(opEqInt, 4, 1, 3),
				mk(opJumpIfTrue, 4, lo, hi),
			},
			expect: []opcode{opEqIntConstJumpTrue, opExt, opNop},
		},
		{
			name:   "LoadIntConst + GeInt + JumpIfFalse -> GeIntConstJumpFalse",
			consts: []int64{10},
			body: []instruction{
				mk(opLoadIntConst, 3, 0, 0),
				mk(opGeInt, 4, 1, 3),
				mk(opJumpIfFalse, 4, lo, hi),
			},
			expect: []opcode{opGeIntConstJumpFalse, opExt, opNop},
		},
		{
			name:   "LoadIntConst + GtInt + JumpIfFalse -> GtIntConstJumpFalse",
			consts: []int64{10},
			body: []instruction{
				mk(opLoadIntConst, 3, 0, 0),
				mk(opGtInt, 4, 1, 3),
				mk(opJumpIfFalse, 4, lo, hi),
			},
			expect: []opcode{opGtIntConstJumpFalse, opExt, opNop},
		},
		{
			name:   "AddIntConst + Jump -> AddIntJump",
			consts: []int64{1},
			body: []instruction{
				mk(opAddIntConst, 0, 0, 0),
				mk(opJump, 0, lo, hi),
			},
			expect: []opcode{opAddIntJump, opExt},
		},
		{
			name: "IncInt + LtInt + JumpIfTrue -> IncIntJumpLt",
			body: []instruction{
				mk(opIncInt, 0, 0, 0),
				mk(opLtInt, 2, 0, 1),
				mk(opJumpIfTrue, 2, lo, hi),
			},
			expect: []opcode{opIncIntJumpLt, opExt, opNop},
		},
		{
			name: "RuneToString + ConcatString -> ConcatRuneString",
			body: []instruction{
				mk(opRuneToString, 3, 1, 0),
				mk(opConcatString, 2, 0, 3),
			},
			expect: []opcode{opConcatRuneString, opNop},
		},
		{
			name:   "LoadIntConst small value -> LoadIntConstSmall",
			consts: []int64{42},
			body: []instruction{
				mk(opLoadIntConst, 0, 0, 0),
			},
			expect: []opcode{opLoadIntConstSmall},
		},
		{
			name:   "LoadIntConst large value stays unchanged",
			consts: []int64{1000},
			body: []instruction{
				mk(opLoadIntConst, 0, 0, 0),
			},
			expect: []opcode{opLoadIntConst},
		},
		{
			name:   "LoadIntConst negative value stays unchanged",
			consts: []int64{-1},
			body: []instruction{
				mk(opLoadIntConst, 0, 0, 0),
			},
			expect: []opcode{opLoadIntConst},
		},
		{
			name:     "LoadStringConst + EqString + JumpIfFalse -> EqStringConstJumpFalse",
			strConst: []string{"hello"},
			body: []instruction{
				mk(opLoadStringConst, 3, 0, 0),
				mk(opEqString, 4, 1, 3),
				mk(opJumpIfFalse, 4, lo, hi),
			},
			expect: []opcode{opEqStringConstJumpFalse, opExt, opNop},
		},
		{
			name: "LoadNil + EqGeneral + JumpIfTrue -> TestNilJumpTrue",
			body: []instruction{
				mk(opLoadNil, 3, 0, 0),
				mk(opEqGeneral, 4, 1, 3),
				mk(opJumpIfTrue, 4, lo, hi),
			},
			expect: []opcode{opTestNilJumpTrue, opNop, opNop},
		},
		{
			name: "LoadNil + EqGeneral + JumpIfFalse -> TestNilJumpFalse",
			body: []instruction{
				mk(opLoadNil, 3, 0, 0),
				mk(opEqGeneral, 4, 1, 3),
				mk(opJumpIfFalse, 4, lo, hi),
			},
			expect: []opcode{opTestNilJumpFalse, opNop, opNop},
		},
		{
			name: "LoadNil + NeGeneral + JumpIfTrue -> TestNilJumpFalse",
			body: []instruction{
				mk(opLoadNil, 3, 0, 0),
				mk(opNeGeneral, 4, 1, 3),
				mk(opJumpIfTrue, 4, lo, hi),
			},
			expect: []opcode{opTestNilJumpFalse, opNop, opNop},
		},
		{
			name: "LoadNil + NeGeneral + JumpIfFalse -> TestNilJumpTrue",
			body: []instruction{
				mk(opLoadNil, 3, 0, 0),
				mk(opNeGeneral, 4, 1, 3),
				mk(opJumpIfFalse, 4, lo, hi),
			},
			expect: []opcode{opTestNilJumpTrue, opNop, opNop},
		},
		{
			name: "LoadNil in B position + EqGeneral + JumpIfTrue -> TestNilJumpTrue",
			body: []instruction{
				mk(opLoadNil, 3, 0, 0),
				mk(opEqGeneral, 4, 3, 1),
				mk(opJumpIfTrue, 4, lo, hi),
			},
			expect: []opcode{opTestNilJumpTrue, opNop, opNop},
		},
		{
			name: "fusion still fires when jump targets first of pair",
			body: []instruction{
				mk(opJump, 0, 0, 0),
				mk(opRuneToString, 3, 1, 0),
				mk(opConcatString, 2, 0, 3),
			},
			expect: []opcode{opJump, opConcatRuneString, opNop},
		},
		{
			name: "no fusion when jump targets second of pair",
			body: []instruction{
				mk(opJump, 0, 1, 0),
				mk(opRuneToString, 3, 1, 0),
				mk(opConcatString, 2, 0, 3),
			},
			expect: []opcode{opJump, opRuneToString, opConcatString},
		},
		{
			name:   "no fusion when registers don't match",
			consts: []int64{10},
			body: []instruction{
				mk(opLoadIntConst, 3, 0, 0),
				mk(opAddInt, 1, 2, 5),
			},
			expect: []opcode{opLoadIntConstSmall, opAddInt},
		},
		{
			name: "StringIndex + UintToInt -> StringIndexToInt",
			body: []instruction{
				mk(opStringIndex, 3, 0, 1),
				mk(opUintToInt, 2, 3, 0),
			},
			expect: []opcode{opStringIndexToInt, opNop},
		},
		{
			name: "no StringIndexToInt when registers don't match",
			body: []instruction{
				mk(opStringIndex, 3, 0, 1),
				mk(opUintToInt, 2, 5, 0),
			},
			expect: []opcode{opStringIndex, opUintToInt},
		},
		{
			name: "LenString + LtInt + JumpIfFalse -> LenStringLtJumpFalse",
			body: []instruction{
				mk(opLenString, 3, 0, 0),
				mk(opLtInt, 4, 1, 3),
				mk(opJumpIfFalse, 4, lo, hi),
			},
			expect: []opcode{opLenStringLtJumpFalse, opExt, opNop},
		},
		{
			name: "no LenStringLtJumpFalse when len reg doesn't match",
			body: []instruction{
				mk(opLenString, 3, 0, 0),
				mk(opLtInt, 4, 1, 5),
				mk(opJumpIfFalse, 4, lo, hi),
			},
			expect: []opcode{opLenString, opLtInt, opJumpIfFalse},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cf := &CompiledFunction{
				body:            make([]instruction, len(tt.body)),
				intConstants:    tt.consts,
				stringConstants: tt.strConst,
			}
			copy(cf.body, tt.body)
			cf.optimise()

			got := make([]opcode, len(cf.body))
			for i, instr := range cf.body {
				got[i] = instr.op
			}
			assert.Equal(t, tt.expect, got, "opcode sequence mismatch")
		})
	}
}

func TestPeepholePreservesSemantics(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		consts []int64
		body   []instruction
		checkA uint8
		checkB uint8
		checkC uint8
	}{
		{
			name:   "SubIntConst preserves operand registers",
			consts: []int64{42},
			body: []instruction{
				mk(opLoadIntConst, 5, 0, 0),
				mk(opSubInt, 1, 2, 5),
			},
			checkA: 1,
			checkB: 2,
			checkC: 0,
		},
		{
			name:   "AddIntConst preserves operand registers",
			consts: []int64{10},
			body: []instruction{
				mk(opLoadIntConst, 5, 0, 0),
				mk(opAddInt, 3, 4, 5),
			},
			checkA: 3,
			checkB: 4,
			checkC: 0,
		},
		{
			name: "ConcatRuneString preserves operand registers",
			body: []instruction{
				mk(opRuneToString, 7, 3, 0),
				mk(opConcatString, 5, 2, 7),
			},
			checkA: 5,
			checkB: 2,
			checkC: 3,
		},
		{
			name: "StringIndexToInt preserves operand registers",
			body: []instruction{
				mk(opStringIndex, 7, 3, 2),
				mk(opUintToInt, 5, 7, 0),
			},
			checkA: 5,
			checkB: 3,
			checkC: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cf := &CompiledFunction{
				body:         make([]instruction, len(tt.body)),
				intConstants: tt.consts,
			}
			copy(cf.body, tt.body)
			cf.optimise()

			fused := cf.body[0]
			assert.Equal(t, tt.checkA, fused.a, "A register")
			assert.Equal(t, tt.checkB, fused.b, "B register")
			assert.Equal(t, tt.checkC, fused.c, "C register")
		})
	}
}

func TestPeepholeJumpOffsetAdjustment(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		consts       []int64
		body         []instruction
		fusedOp      opcode
		expectOffset int16
	}{
		{
			name:   "3-instr fusion adjusts offset by +1",
			consts: []int64{10},
			body: []instruction{
				mk(opLoadIntConst, 3, 0, 0),
				mk(opLtInt, 4, 1, 3),
				mk(opJumpIfFalse, 4, 5, 0),
			},
			fusedOp:      opLtIntConstJumpFalse,
			expectOffset: 6,
		},
		{
			name: "IncIntJumpLt adjusts offset by +1",
			body: []instruction{
				mk(opIncInt, 0, 0, 0),
				mk(opLtInt, 2, 0, 1),
				mk(opJumpIfTrue, 2, 252, 255),
			},
			fusedOp:      opIncIntJumpLt,
			expectOffset: -3,
		},
		{
			name: "LenStringLtJumpFalse adjusts offset by +1",
			body: []instruction{
				mk(opLenString, 3, 0, 0),
				mk(opLtInt, 4, 1, 3),
				mk(opJumpIfFalse, 4, 5, 0),
			},
			fusedOp:      opLenStringLtJumpFalse,
			expectOffset: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cf := &CompiledFunction{
				body:         make([]instruction, len(tt.body)),
				intConstants: tt.consts,
			}
			copy(cf.body, tt.body)
			cf.optimise()

			assert.Equal(t, tt.fusedOp, cf.body[0].op)

			extensionWord := cf.body[1]
			assert.Equal(t, opExt, extensionWord.op)
			gotOffset := int16(uint16(extensionWord.a) | uint16(extensionWord.b)<<8)
			assert.Equal(t, tt.expectOffset, gotOffset, "adjusted jump offset")
		})
	}
}

func TestPeepholeNilJumpOffset(t *testing.T) {
	t.Parallel()
	cf := &CompiledFunction{
		body: []instruction{
			mk(opLoadNil, 3, 0, 0),
			mk(opEqGeneral, 4, 1, 3),
			mk(opJumpIfTrue, 4, 5, 0),
		},
	}
	cf.optimise()

	assert.Equal(t, opTestNilJumpTrue, cf.body[0].op)
	gotOffset := int16(uint16(cf.body[0].b) | uint16(cf.body[0].c)<<8)
	assert.Equal(t, int16(7), gotOffset, "nil jump offset adjusts by +2")
}

func TestPeepholeRecursive(t *testing.T) {
	t.Parallel()
	child := &CompiledFunction{
		intConstants: []int64{1},
		body: []instruction{
			mk(opLoadIntConst, 3, 0, 0),
			mk(opAddInt, 1, 2, 3),
		},
	}
	parent := &CompiledFunction{
		functions: []*CompiledFunction{child},
		body:      []instruction{mk(opNop, 0, 0, 0)},
	}
	parent.optimise()

	assert.Equal(t, opAddIntConst, child.body[0].op)
	assert.Equal(t, opNop, child.body[1].op)
}
