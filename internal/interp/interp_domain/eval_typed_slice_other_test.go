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

func TestTypedSliceFloatDirectOps(t *testing.T) {
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
				lengthConstantIndex := builder.addIntConst(5)
				builder.intRegisters(2).sliceFloatRegisters(1).returnInt()
				builder.emit(opLoadIntConst, 1, lengthConstantIndex, 0)
				builder.emit(opDrillTier1, uint8(subOpMakeSliceFloat), 0, 1)
				builder.emit(opExt, 1, 0, 0)
				builder.emit(opDrillTier1, uint8(subOpLenSliceFloatDirect), 0, 0)
				builder.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 1)
				return builder.build()
			},
			expect: int64(5),
		},
		{
			name: "set_then_get_round_trip",
			build: func() *CompiledFunction {
				builder := newBytecodeBuilder()
				lengthConstantIndex := builder.addIntConst(3)
				indexConstantIndex := builder.addIntConst(2)
				valueConstantIndex := builder.addFloatConst(3.14159)
				builder.intRegisters(3).floatRegisters(2).sliceFloatRegisters(1).returnFloat()
				builder.emit(opLoadIntConst, 1, lengthConstantIndex, 0)
				builder.emit(opDrillTier1, uint8(subOpMakeSliceFloat), 0, 1)
				builder.emit(opExt, 1, 0, 0)
				builder.emit(opLoadIntConst, 2, indexConstantIndex, 0)
				builder.emit(opLoadFloatConst, 1, valueConstantIndex, 0)
				builder.emit(opDrillTier1, uint8(subOpSliceSetFloatDirect), 0, 2)
				builder.emit(opExt, 1, 0, 0)
				builder.emit(opDrillTier1, uint8(subOpSliceGetFloatDirect), 0, 0)
				builder.emit(opExt, 2, 0, 0)
				builder.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 1)
				return builder.build()
			},
			expect: 3.14159,
		},
		{
			name: "out_of_range_panics",
			build: func() *CompiledFunction {
				builder := newBytecodeBuilder()
				lengthConstantIndex := builder.addIntConst(2)
				outOfRangeConstantIndex := builder.addIntConst(2)
				builder.intRegisters(3).floatRegisters(1).sliceFloatRegisters(1).returnFloat()
				builder.emit(opLoadIntConst, 1, lengthConstantIndex, 0)
				builder.emit(opDrillTier1, uint8(subOpMakeSliceFloat), 0, 1)
				builder.emit(opExt, 1, 0, 0)
				builder.emit(opLoadIntConst, 2, outOfRangeConstantIndex, 0)
				builder.emit(opDrillTier1, uint8(subOpSliceGetFloatDirect), 0, 0)
				builder.emit(opExt, 2, 0, 0)
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

func TestTypedSliceStringDirectOps(t *testing.T) {
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
				lengthConstantIndex := builder.addIntConst(4)
				builder.intRegisters(2).sliceStringRegisters(1).returnInt()
				builder.emit(opLoadIntConst, 1, lengthConstantIndex, 0)
				builder.emit(opDrillTier1, uint8(subOpMakeSliceString), 0, 1)
				builder.emit(opExt, 1, 0, 0)
				builder.emit(opDrillTier1, uint8(subOpLenSliceStringDirect), 0, 0)
				builder.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 1)
				return builder.build()
			},
			expect: int64(4),
		},
		{
			name: "set_then_get_round_trip",
			build: func() *CompiledFunction {
				builder := newBytecodeBuilder()
				lengthConstantIndex := builder.addIntConst(3)
				indexConstantIndex := builder.addIntConst(1)
				valueConstantIndex := builder.addStringConst("hello")
				builder.intRegisters(3).stringRegisters(2).sliceStringRegisters(1).returnString()
				builder.emit(opLoadIntConst, 1, lengthConstantIndex, 0)
				builder.emit(opDrillTier1, uint8(subOpMakeSliceString), 0, 1)
				builder.emit(opExt, 1, 0, 0)
				builder.emit(opLoadIntConst, 2, indexConstantIndex, 0)
				builder.emit(opLoadStringConst, 1, valueConstantIndex, 0)
				builder.emit(opDrillTier1, uint8(subOpSliceSetStringDirect), 0, 2)
				builder.emit(opExt, 1, 0, 0)
				builder.emit(opDrillTier1, uint8(subOpSliceGetStringDirect), 0, 0)
				builder.emit(opExt, 2, 0, 0)
				builder.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 1)
				return builder.build()
			},
			expect: "hello",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			compiled := testCase.build()
			result, err := service.Execute(context.Background(), compiled)
			require.NoError(t, err)
			require.Equal(t, testCase.expect, result)
		})
	}
}

func TestTypedSliceBoolDirectOps(t *testing.T) {
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
				lengthConstantIndex := builder.addIntConst(6)
				builder.intRegisters(2).sliceBoolRegisters(1).returnInt()
				builder.emit(opLoadIntConst, 1, lengthConstantIndex, 0)
				builder.emit(opDrillTier1, uint8(subOpMakeSliceBool), 0, 1)
				builder.emit(opExt, 1, 0, 0)
				builder.emit(opDrillTier1, uint8(subOpLenSliceBoolDirect), 0, 0)
				builder.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 1)
				return builder.build()
			},
			expect: int64(6),
		},
		{
			name: "set_then_get_round_trip",
			build: func() *CompiledFunction {
				builder := newBytecodeBuilder()
				lengthConstantIndex := builder.addIntConst(2)
				indexConstantIndex := builder.addIntConst(0)
				trueConstantIndex := builder.addBoolConst(true)
				builder.intRegisters(3).boolRegisters(2).sliceBoolRegisters(1).returnBool()
				builder.emit(opLoadIntConst, 1, lengthConstantIndex, 0)
				builder.emit(opDrillTier1, uint8(subOpMakeSliceBool), 0, 1)
				builder.emit(opExt, 1, 0, 0)
				builder.emit(opLoadIntConst, 2, indexConstantIndex, 0)
				builder.emit(opLoadBoolConst, 1, trueConstantIndex, 0)
				builder.emit(opDrillTier1, uint8(subOpSliceSetBoolDirect), 0, 2)
				builder.emit(opExt, 1, 0, 0)
				builder.emit(opDrillTier1, uint8(subOpSliceGetBoolDirect), 0, 0)
				builder.emit(opExt, 2, 0, 0)
				builder.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 1)
				return builder.build()
			},
			expect: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			compiled := testCase.build()
			result, err := service.Execute(context.Background(), compiled)
			require.NoError(t, err)
			require.Equal(t, testCase.expect, result)
		})
	}
}

func TestTypedSliceUintDirectOps(t *testing.T) {
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
				lengthConstantIndex := builder.addIntConst(8)
				builder.intRegisters(2).sliceUintRegisters(1).returnInt()
				builder.emit(opLoadIntConst, 1, lengthConstantIndex, 0)
				builder.emit(opDrillTier1, uint8(subOpMakeSliceUint), 0, 1)
				builder.emit(opExt, 1, 0, 0)
				builder.emit(opDrillTier1, uint8(subOpLenSliceUintDirect), 0, 0)
				builder.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 1)
				return builder.build()
			},
			expect: int64(8),
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			compiled := testCase.build()
			result, err := service.Execute(context.Background(), compiled)
			require.NoError(t, err)
			require.Equal(t, testCase.expect, result)
		})
	}
}
