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
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTypedSliceIntDirectOps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		build  func() *CompiledFunction
		name   string
	}{
		{
			name: "make_then_len",
			build: func() *CompiledFunction {
				builder := newBytecodeBuilder()
				lengthConstantIndex := builder.addIntConst(7)
				builder.intRegisters(2).sliceIntRegisters(1).returnInt()
				builder.emit(opLoadIntConst, 1, lengthConstantIndex, 0)
				builder.emit(opDrillTier1, uint8(subOpMakeSliceInt), 0, 1)
				builder.emit(opExt, 1, 0, 0)
				builder.emit(opDrillTier1, uint8(subOpLenSliceIntDirect), 0, 0)
				builder.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 1)
				return builder.build()
			},
			expect: int64(7),
		},
		{
			name: "set_then_get_round_trip",
			build: func() *CompiledFunction {
				builder := newBytecodeBuilder()
				lengthConstantIndex := builder.addIntConst(3)
				indexConstantIndex := builder.addIntConst(1)
				valueConstantIndex := builder.addIntConst(99)
				builder.intRegisters(4).sliceIntRegisters(1).returnInt()
				builder.emit(opLoadIntConst, 1, lengthConstantIndex, 0)
				builder.emit(opDrillTier1, uint8(subOpMakeSliceInt), 0, 1)
				builder.emit(opExt, 1, 0, 0)
				builder.emit(opLoadIntConst, 2, indexConstantIndex, 0)
				builder.emit(opLoadIntConst, 3, valueConstantIndex, 0)
				builder.emit(opSliceSetIntDirect, 0, 2, 3)
				builder.emit(opSliceGetIntDirect, 0, 0, 2)
				builder.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 1)
				return builder.build()
			},
			expect: int64(99),
		},
		{
			name: "sum_three_elements",
			build: func() *CompiledFunction {
				builder := newBytecodeBuilder()
				lengthConstantIndex := builder.addIntConst(3)
				zeroConstantIndex := builder.addIntConst(0)
				oneConstantIndex := builder.addIntConst(1)
				twoConstantIndex := builder.addIntConst(2)
				tenConstantIndex := builder.addIntConst(10)
				twentyConstantIndex := builder.addIntConst(20)
				thirtyConstantIndex := builder.addIntConst(30)
				builder.intRegisters(6).sliceIntRegisters(1).returnInt()
				builder.emit(opLoadIntConst, 1, lengthConstantIndex, 0)
				builder.emit(opDrillTier1, uint8(subOpMakeSliceInt), 0, 1)
				builder.emit(opExt, 1, 0, 0)
				builder.emit(opLoadIntConst, 2, zeroConstantIndex, 0)
				builder.emit(opLoadIntConst, 3, tenConstantIndex, 0)
				builder.emit(opSliceSetIntDirect, 0, 2, 3)
				builder.emit(opLoadIntConst, 2, oneConstantIndex, 0)
				builder.emit(opLoadIntConst, 3, twentyConstantIndex, 0)
				builder.emit(opSliceSetIntDirect, 0, 2, 3)
				builder.emit(opLoadIntConst, 2, twoConstantIndex, 0)
				builder.emit(opLoadIntConst, 3, thirtyConstantIndex, 0)
				builder.emit(opSliceSetIntDirect, 0, 2, 3)
				builder.emit(opLoadIntConst, 0, zeroConstantIndex, 0)
				builder.emit(opLoadIntConst, 4, zeroConstantIndex, 0)
				builder.emit(opSliceGetIntDirect, 5, 0, 4)
				builder.emit(opAddInt, 0, 0, 5)
				builder.emit(opLoadIntConst, 4, oneConstantIndex, 0)
				builder.emit(opSliceGetIntDirect, 5, 0, 4)
				builder.emit(opAddInt, 0, 0, 5)
				builder.emit(opLoadIntConst, 4, twoConstantIndex, 0)
				builder.emit(opSliceGetIntDirect, 5, 0, 4)
				builder.emit(opAddInt, 0, 0, 5)
				builder.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 1)
				return builder.build()
			},
			expect: int64(60),
		},
		{
			name: "negative_index_panics",
			build: func() *CompiledFunction {
				builder := newBytecodeBuilder()
				lengthConstantIndex := builder.addIntConst(2)
				negativeConstantIndex := builder.addIntConst(-1)
				builder.intRegisters(3).sliceIntRegisters(1).returnInt()
				builder.emit(opLoadIntConst, 0, lengthConstantIndex, 0)
				builder.emit(opDrillTier1, uint8(subOpMakeSliceInt), 0, 0)
				builder.emit(opExt, 0, 0, 0)
				builder.emit(opLoadIntConst, 1, negativeConstantIndex, 0)
				builder.emit(opSliceGetIntDirect, 2, 0, 1)
				builder.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 1)
				return builder.build()
			},
			expect: errIndexOutOfRange,
		},
		{
			name: "out_of_range_index_panics",
			build: func() *CompiledFunction {
				builder := newBytecodeBuilder()
				lengthConstantIndex := builder.addIntConst(2)
				outOfRangeConstantIndex := builder.addIntConst(2)
				builder.intRegisters(3).sliceIntRegisters(1).returnInt()
				builder.emit(opLoadIntConst, 0, lengthConstantIndex, 0)
				builder.emit(opDrillTier1, uint8(subOpMakeSliceInt), 0, 0)
				builder.emit(opExt, 0, 0, 0)
				builder.emit(opLoadIntConst, 1, outOfRangeConstantIndex, 0)
				builder.emit(opSliceGetIntDirect, 2, 0, 1)
				builder.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 1)
				return builder.build()
			},
			expect: errIndexOutOfRange,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			compiled := testCase.build()
			result, err := service.Execute(context.Background(), compiled)
			if expectedError, ok := testCase.expect.(error); ok {
				require.Error(t, err)
				require.True(t, errors.Is(err, expectedError),
					"expected error wrapping %v, got %v", expectedError, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, testCase.expect, result)
		})
	}
}
