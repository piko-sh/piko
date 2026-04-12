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

	"github.com/stretchr/testify/require"
)

func TestRangeNextStringDecodesRunes(t *testing.T) {
	t.Parallel()

	registers := newRegisters(testRegCounts())
	iterator := &rangeIterator{
		isString:     true,
		stringSource: "héllo",
	}
	context := rangeNextContext{
		doneDestination: 0,
		hasKey:          true,
		keyInstruction:  instruction{b: 1, c: uint8(registerInt)},
		hasValue:        true,
		valInstruction:  instruction{b: 2, c: uint8(registerInt)},
	}

	expected := []struct {
		index    int64
		runeCode int64
	}{
		{index: 0, runeCode: 'h'},
		{index: 1, runeCode: 'é'},
		{index: 3, runeCode: 'l'},
		{index: 4, runeCode: 'l'},
		{index: 5, runeCode: 'o'},
	}

	for step, want := range expected {
		rangeNextString(&registers, iterator, context)
		require.Equalf(t, int64(1), registers.ints[0],
			"step %d: doneDestination should be 1 while iteration continues", step)
		require.Equalf(t, want.index, registers.ints[1],
			"step %d: byte index of rune", step)
		require.Equalf(t, want.runeCode, registers.ints[2],
			"step %d: decoded rune value", step)
	}

	rangeNextString(&registers, iterator, context)
	require.Equal(t, int64(0), registers.ints[0],
		"after the last rune, doneDestination should be 0 to terminate the loop")
}

func TestRangeNextStringEmptyString(t *testing.T) {
	t.Parallel()

	registers := newRegisters(testRegCounts())
	iterator := &rangeIterator{
		isString:     true,
		stringSource: "",
	}
	context := rangeNextContext{
		doneDestination: 0,
		hasKey:          true,
		keyInstruction:  instruction{b: 1, c: uint8(registerInt)},
		hasValue:        true,
		valInstruction:  instruction{b: 2, c: uint8(registerInt)},
	}

	rangeNextString(&registers, iterator, context)
	require.Equal(t, int64(0), registers.ints[0],
		"empty string should immediately terminate iteration")
}

func TestRangeNextStringValueOnlyNoKey(t *testing.T) {
	t.Parallel()

	registers := newRegisters(testRegCounts())
	iterator := &rangeIterator{
		isString:     true,
		stringSource: "ab",
	}
	context := rangeNextContext{
		doneDestination: 0,
		hasKey:          false,
		hasValue:        true,
		valInstruction:  instruction{b: 1, c: uint8(registerInt)},
	}

	rangeNextString(&registers, iterator, context)
	require.Equal(t, int64(1), registers.ints[0])
	require.Equal(t, int64('a'), registers.ints[1])

	rangeNextString(&registers, iterator, context)
	require.Equal(t, int64(1), registers.ints[0])
	require.Equal(t, int64('b'), registers.ints[1])
}

func TestRangeNextStringInvalidUTF8YieldsRuneError(t *testing.T) {
	t.Parallel()

	registers := newRegisters(testRegCounts())
	iterator := &rangeIterator{
		isString:     true,
		stringSource: "\xff",
	}
	context := rangeNextContext{
		doneDestination: 0,
		hasKey:          true,
		keyInstruction:  instruction{b: 1, c: uint8(registerInt)},
		hasValue:        true,
		valInstruction:  instruction{b: 2, c: uint8(registerInt)},
	}

	rangeNextString(&registers, iterator, context)
	require.Equal(t, int64(1), registers.ints[0])
	require.Equal(t, int64(0xFFFD), registers.ints[2],
		"invalid UTF-8 sequences should decode to utf8.RuneError (U+FFFD) and consume one byte")

	rangeNextString(&registers, iterator, context)
	require.Equal(t, int64(0), registers.ints[0],
		"after consuming the lone invalid byte, the iteration terminates")
}
