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

//go:build !safe && !(js && wasm) && (amd64 || arm64)

package interp_domain

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSubOpInlineGoCallAMD64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		build  func() *CompiledFunction
		name   string
	}{
		{
			name: "MathSin_via_ASM",
			build: func() *CompiledFunction {
				builder := newBytecodeBuilder()
				srcConst := builder.addFloatConst(0.5)
				builder.intRegisters(2).floatRegisters(2).returnFloat()
				builder.emit(opLoadFloatConst, 1, srcConst, 0)
				builder.emit(opDrillTier1, uint8(subOpMathSin), 0, 1)
				builder.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 0)
				return builder.build()
			},
			expect: math.Sin(0.5),
		},
		{
			name: "MathCos_via_ASM",
			build: func() *CompiledFunction {
				builder := newBytecodeBuilder()
				srcConst := builder.addFloatConst(1.0)
				builder.intRegisters(2).floatRegisters(2).returnFloat()
				builder.emit(opLoadFloatConst, 1, srcConst, 0)
				builder.emit(opDrillTier1, uint8(subOpMathCos), 0, 1)
				builder.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 0)
				return builder.build()
			},
			expect: math.Cos(1.0),
		},
		{
			name: "MathMod_via_ASM",
			build: func() *CompiledFunction {
				builder := newBytecodeBuilder()
				src1Const := builder.addFloatConst(7.5)
				src2Const := builder.addFloatConst(2.0)
				builder.intRegisters(2).floatRegisters(3).returnFloat()
				builder.emit(opLoadFloatConst, 1, src1Const, 0)
				builder.emit(opLoadFloatConst, 2, src2Const, 0)
				builder.emit(opDrillTier1, uint8(subOpMathMod), 0, 1)
				builder.emit(opExt, 2, 0, 0)
				builder.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 0)
				return builder.build()
			},
			expect: math.Mod(7.5, 2.0),
		},
		{
			name: "StrconvItoa_via_ASM",
			build: func() *CompiledFunction {
				builder := newBytecodeBuilder()
				srcConst := builder.addIntConst(42)
				builder.intRegisters(2).stringRegisters(1).returnString()
				builder.emit(opLoadIntConst, 1, srcConst, 0)
				builder.emit(opDrillTier1, uint8(subOpStrconvItoa), 0, 1)
				builder.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 0)
				return builder.build()
			},
			expect: "42",
		},
		{
			name: "StrconvFormatBool_via_ASM",
			build: func() *CompiledFunction {
				builder := newBytecodeBuilder()
				oneConst := builder.addIntConst(1)
				builder.intRegisters(2).boolRegisters(2).stringRegisters(1).returnString()
				builder.emit(opLoadIntConst, 1, oneConst, 0)
				builder.emit(opDrillTier1, uint8(subOpIntToBool), 1, 1)
				builder.emit(opDrillTier1, uint8(subOpStrconvFormatBool), 0, 1)
				builder.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 0)
				return builder.build()
			},
			expect: "true",
		},
		{
			name: "StrconvFormatInt_via_ASM",
			build: func() *CompiledFunction {
				builder := newBytecodeBuilder()
				valConst := builder.addIntConst(255)
				baseConst := builder.addIntConst(16)
				builder.intRegisters(3).stringRegisters(1).returnString()
				builder.emit(opLoadIntConst, 1, valConst, 0)
				builder.emit(opLoadIntConst, 2, baseConst, 0)
				builder.emit(opDrillTier1, uint8(subOpStrconvFormatInt), 0, 1)
				builder.emit(opExt, 2, 0, 0)
				builder.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 0)
				return builder.build()
			},
			expect: "ff",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			compiled := testCase.build()
			result, err := service.Execute(context.Background(), compiled)
			require.NoError(t, err)
			if expected, ok := testCase.expect.(float64); ok {
				require.InDelta(t, expected, result, 1e-10)
			} else {
				require.Equal(t, testCase.expect, result)
			}
		})
	}
}
