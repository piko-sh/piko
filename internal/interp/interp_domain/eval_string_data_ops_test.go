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

func execGoDispatch(t *testing.T, compiledFunction *CompiledFunction) (any, error) {
	t.Helper()
	service := NewService(WithForceGoDispatch())
	return service.Execute(context.Background(), compiledFunction)
}

func requireGoDispatchResult(t *testing.T, compiledFunction *CompiledFunction, expect any) {
	t.Helper()
	result, err := execGoDispatch(t, compiledFunction)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expect {
		t.Fatalf("expected %v, got %v", expect, result)
	}
}

func TestStringIndexToInt(t *testing.T) {
	t.Parallel()

	t.Run("first byte", func(t *testing.T) {
		t.Parallel()
		bb := newBytecodeBuilder()
		strIndex := bb.addStringConst("hello")
		bb.intRegisters(2).stringRegisters(1).returnInt()
		bb.emit(opLoadStringConst, 0, strIndex, 0)
		bb.emit(opLoadIntConstSmall, 1, 0, 0)
		bb.emit(opStringIndexToInt, 0, 0, 1)
		bb.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, bb.build(), int64('h'))
	})

	t.Run("last byte", func(t *testing.T) {
		t.Parallel()
		bb := newBytecodeBuilder()
		strIndex := bb.addStringConst("hello")
		bb.intRegisters(2).stringRegisters(1).returnInt()
		bb.emit(opLoadStringConst, 0, strIndex, 0)
		bb.emit(opLoadIntConstSmall, 1, 4, 0)
		bb.emit(opStringIndexToInt, 0, 0, 1)
		bb.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, bb.build(), int64('o'))
	})

	t.Run("out of bounds positive", func(t *testing.T) {
		t.Parallel()
		bb := newBytecodeBuilder()
		strIndex := bb.addStringConst("hello")
		bb.intRegisters(2).stringRegisters(1).returnInt()
		bb.emit(opLoadStringConst, 0, strIndex, 0)
		bb.emit(opLoadIntConstSmall, 1, 5, 0)
		bb.emit(opStringIndexToInt, 0, 0, 1)
		bb.emit(opReturn, 1, 0, 0)

		_, err := execSynthetic(t, bb.build())
		require.Error(t, err)
		require.True(t, errors.Is(err, errIndexOutOfRange), "expected errIndexOutOfRange, got: %v", err)
	})

	t.Run("out of bounds negative", func(t *testing.T) {
		t.Parallel()
		bb := newBytecodeBuilder()
		strIndex := bb.addStringConst("hello")
		negIndex := bb.addIntConst(-1)
		bb.intRegisters(2).stringRegisters(1).returnInt()
		bb.emit(opLoadStringConst, 0, strIndex, 0)
		bb.emit(opLoadIntConst, 1, negIndex, 0)
		bb.emit(opStringIndexToInt, 0, 0, 1)
		bb.emit(opReturn, 1, 0, 0)

		_, err := execSynthetic(t, bb.build())
		require.Error(t, err)
		require.True(t, errors.Is(err, errIndexOutOfRange), "expected errIndexOutOfRange, got: %v", err)
	})

	t.Run("empty string index zero", func(t *testing.T) {
		t.Parallel()
		bb := newBytecodeBuilder()
		strIndex := bb.addStringConst("")
		bb.intRegisters(2).stringRegisters(1).returnInt()
		bb.emit(opLoadStringConst, 0, strIndex, 0)
		bb.emit(opLoadIntConstSmall, 1, 0, 0)
		bb.emit(opStringIndexToInt, 0, 0, 1)
		bb.emit(opReturn, 1, 0, 0)

		_, err := execSynthetic(t, bb.build())
		require.Error(t, err)
		require.True(t, errors.Is(err, errIndexOutOfRange), "expected errIndexOutOfRange, got: %v", err)
	})

	t.Run("multi-byte UTF-8 returns byte not rune", func(t *testing.T) {
		t.Parallel()
		bb := newBytecodeBuilder()

		strIndex := bb.addStringConst("caf\xc3\xa9")
		bb.intRegisters(2).stringRegisters(1).returnInt()
		bb.emit(opLoadStringConst, 0, strIndex, 0)
		bb.emit(opLoadIntConstSmall, 1, 3, 0)
		bb.emit(opStringIndexToInt, 0, 0, 1)
		bb.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, bb.build(), int64(0xC3))
	})
}

func TestSliceString(t *testing.T) {
	t.Parallel()

	t.Run("full slice", func(t *testing.T) {
		t.Parallel()
		bb := newBytecodeBuilder()
		strIndex := bb.addStringConst("hello")
		bb.intRegisters(2).stringRegisters(2).returnString()
		bb.emit(opLoadStringConst, 1, strIndex, 0)
		bb.emit(opSliceString, 0, 1, 0)
		bb.emitExt(0)
		bb.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, bb.build(), "hello")
	})

	t.Run("low bound only", func(t *testing.T) {
		t.Parallel()
		bb := newBytecodeBuilder()
		strIndex := bb.addStringConst("hello")
		bb.intRegisters(2).stringRegisters(2).returnString()
		bb.emit(opLoadStringConst, 1, strIndex, 0)
		bb.emit(opLoadIntConstSmall, 0, 2, 0)
		bb.emit(opSliceString, 0, 1, 1)
		bb.emit(opExt, 0, 0, 0)
		bb.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, bb.build(), "llo")
	})

	t.Run("high bound only", func(t *testing.T) {
		t.Parallel()
		bb := newBytecodeBuilder()
		strIndex := bb.addStringConst("hello")
		bb.intRegisters(2).stringRegisters(2).returnString()
		bb.emit(opLoadStringConst, 1, strIndex, 0)
		bb.emit(opLoadIntConstSmall, 0, 3, 0)
		bb.emit(opSliceString, 0, 1, 2)
		bb.emit(opExt, 0, 0, 0)
		bb.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, bb.build(), "hel")
	})

	t.Run("both bounds", func(t *testing.T) {
		t.Parallel()
		bb := newBytecodeBuilder()
		strIndex := bb.addStringConst("hello")
		bb.intRegisters(2).stringRegisters(2).returnString()
		bb.emit(opLoadStringConst, 1, strIndex, 0)
		bb.emit(opLoadIntConstSmall, 0, 1, 0)
		bb.emit(opLoadIntConstSmall, 1, 4, 0)
		bb.emit(opSliceString, 0, 1, 3)
		bb.emit(opExt, 0, 1, 0)
		bb.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, bb.build(), "ell")
	})

	t.Run("empty result", func(t *testing.T) {
		t.Parallel()
		bb := newBytecodeBuilder()
		strIndex := bb.addStringConst("hello")
		bb.intRegisters(2).stringRegisters(2).returnString()
		bb.emit(opLoadStringConst, 1, strIndex, 0)
		bb.emit(opLoadIntConstSmall, 0, 2, 0)
		bb.emit(opLoadIntConstSmall, 1, 2, 0)
		bb.emit(opSliceString, 0, 1, 3)
		bb.emit(opExt, 0, 1, 0)
		bb.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, bb.build(), "")
	})

	t.Run("negative low panics", func(t *testing.T) {
		t.Parallel()
		bb := newBytecodeBuilder()
		strIndex := bb.addStringConst("hello")
		negIndex := bb.addIntConst(-1)
		bb.intRegisters(2).stringRegisters(2).returnString()
		bb.emit(opLoadStringConst, 1, strIndex, 0)
		bb.emit(opLoadIntConst, 0, negIndex, 0)
		bb.emit(opSliceString, 0, 1, 1)
		bb.emit(opExt, 0, 0, 0)
		bb.emit(opReturn, 1, 0, 0)

		_, err := execSynthetic(t, bb.build())
		require.Error(t, err)
		require.True(t, errors.Is(err, errSliceOutOfRange), "expected errSliceOutOfRange, got: %v", err)
	})

	t.Run("high less than low panics", func(t *testing.T) {
		t.Parallel()
		bb := newBytecodeBuilder()
		strIndex := bb.addStringConst("hello")
		bb.intRegisters(2).stringRegisters(2).returnString()
		bb.emit(opLoadStringConst, 1, strIndex, 0)
		bb.emit(opLoadIntConstSmall, 0, 3, 0)
		bb.emit(opLoadIntConstSmall, 1, 1, 0)
		bb.emit(opSliceString, 0, 1, 3)
		bb.emit(opExt, 0, 1, 0)
		bb.emit(opReturn, 1, 0, 0)

		_, err := execSynthetic(t, bb.build())
		require.Error(t, err)
		require.True(t, errors.Is(err, errSliceOutOfRange), "expected errSliceOutOfRange, got: %v", err)
	})

	t.Run("high exceeds length panics", func(t *testing.T) {
		t.Parallel()
		bb := newBytecodeBuilder()
		strIndex := bb.addStringConst("hello")
		bb.intRegisters(2).stringRegisters(2).returnString()
		bb.emit(opLoadStringConst, 1, strIndex, 0)
		bb.emit(opLoadIntConstSmall, 0, 10, 0)
		bb.emit(opSliceString, 0, 1, 2)
		bb.emit(opExt, 0, 0, 0)
		bb.emit(opReturn, 1, 0, 0)

		_, err := execSynthetic(t, bb.build())
		require.Error(t, err)
		require.True(t, errors.Is(err, errSliceOutOfRange), "expected errSliceOutOfRange, got: %v", err)
	})

	t.Run("empty string full slice", func(t *testing.T) {
		t.Parallel()
		bb := newBytecodeBuilder()
		strIndex := bb.addStringConst("")
		bb.intRegisters(2).stringRegisters(2).returnString()
		bb.emit(opLoadStringConst, 1, strIndex, 0)
		bb.emit(opSliceString, 0, 1, 0)
		bb.emitExt(0)
		bb.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, bb.build(), "")
	})

	t.Run("empty string zero bounds", func(t *testing.T) {
		t.Parallel()
		bb := newBytecodeBuilder()
		strIndex := bb.addStringConst("")
		bb.intRegisters(2).stringRegisters(2).returnString()
		bb.emit(opLoadStringConst, 1, strIndex, 0)
		bb.emit(opLoadIntConstSmall, 0, 0, 0)
		bb.emit(opLoadIntConstSmall, 1, 0, 0)
		bb.emit(opSliceString, 0, 1, 3)
		bb.emit(opExt, 0, 1, 0)
		bb.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, bb.build(), "")
	})
}

func TestStringIndexToIntUTF8EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("4-byte emoji returns individual bytes", func(t *testing.T) {
		t.Parallel()

		bb := newBytecodeBuilder()
		strIndex := bb.addStringConst("\xF0\x9F\x8E\x89")
		bb.intRegisters(2).stringRegisters(1).returnInt()
		bb.emit(opLoadStringConst, 0, strIndex, 0)
		bb.emit(opLoadIntConstSmall, 1, 0, 0)
		bb.emit(opStringIndexToInt, 0, 0, 1)
		bb.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, bb.build(), int64(0xF0))
	})

	t.Run("4-byte emoji third byte", func(t *testing.T) {
		t.Parallel()
		bb := newBytecodeBuilder()
		strIndex := bb.addStringConst("\xF0\x9F\x8E\x89")
		bb.intRegisters(2).stringRegisters(1).returnInt()
		bb.emit(opLoadStringConst, 0, strIndex, 0)
		bb.emit(opLoadIntConstSmall, 1, 2, 0)
		bb.emit(opStringIndexToInt, 0, 0, 1)
		bb.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, bb.build(), int64(0x8E))
	})

	t.Run("NUL byte in string", func(t *testing.T) {
		t.Parallel()
		bb := newBytecodeBuilder()
		strIndex := bb.addStringConst("a\x00b")
		bb.intRegisters(2).stringRegisters(1).returnInt()
		bb.emit(opLoadStringConst, 0, strIndex, 0)
		bb.emit(opLoadIntConstSmall, 1, 1, 0)
		bb.emit(opStringIndexToInt, 0, 0, 1)
		bb.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, bb.build(), int64(0))
	})

	t.Run("invalid UTF-8 continuation byte", func(t *testing.T) {
		t.Parallel()

		bb := newBytecodeBuilder()
		strIndex := bb.addStringConst("\x80\x81\x82")
		bb.intRegisters(2).stringRegisters(1).returnInt()
		bb.emit(opLoadStringConst, 0, strIndex, 0)
		bb.emit(opLoadIntConstSmall, 1, 0, 0)
		bb.emit(opStringIndexToInt, 0, 0, 1)
		bb.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, bb.build(), int64(0x80))
	})

	t.Run("single byte string index zero", func(t *testing.T) {
		t.Parallel()
		bb := newBytecodeBuilder()
		strIndex := bb.addStringConst("z")
		bb.intRegisters(2).stringRegisters(1).returnInt()
		bb.emit(opLoadStringConst, 0, strIndex, 0)
		bb.emit(opLoadIntConstSmall, 1, 0, 0)
		bb.emit(opStringIndexToInt, 0, 0, 1)
		bb.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, bb.build(), int64('z'))
	})

	t.Run("single byte string index one out of bounds", func(t *testing.T) {
		t.Parallel()
		bb := newBytecodeBuilder()
		strIndex := bb.addStringConst("z")
		bb.intRegisters(2).stringRegisters(1).returnInt()
		bb.emit(opLoadStringConst, 0, strIndex, 0)
		bb.emit(opLoadIntConstSmall, 1, 1, 0)
		bb.emit(opStringIndexToInt, 0, 0, 1)
		bb.emit(opReturn, 1, 0, 0)

		_, err := execSynthetic(t, bb.build())
		require.Error(t, err)
		require.True(t, errors.Is(err, errIndexOutOfRange))
	})
}

func TestSliceStringUTF8EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("slice splits multi-byte rune", func(t *testing.T) {
		t.Parallel()

		bb := newBytecodeBuilder()
		strIndex := bb.addStringConst("caf\xc3\xa9")
		bb.intRegisters(2).stringRegisters(2).returnString()
		bb.emit(opLoadStringConst, 1, strIndex, 0)
		bb.emit(opLoadIntConstSmall, 0, 0, 0)
		bb.emit(opLoadIntConstSmall, 1, 4, 0)
		bb.emit(opSliceString, 0, 1, 3)
		bb.emit(opExt, 0, 1, 0)
		bb.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, bb.build(), "caf\xc3")
	})

	t.Run("slice of all multi-byte characters", func(t *testing.T) {
		t.Parallel()

		bb := newBytecodeBuilder()
		strIndex := bb.addStringConst("日本語")
		bb.intRegisters(2).stringRegisters(2).returnString()
		bb.emit(opLoadStringConst, 1, strIndex, 0)
		bb.emit(opLoadIntConstSmall, 0, 3, 0)
		bb.emit(opLoadIntConstSmall, 1, 6, 0)
		bb.emit(opSliceString, 0, 1, 3)
		bb.emit(opExt, 0, 1, 0)
		bb.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, bb.build(), "本")
	})

	t.Run("slice of emoji string", func(t *testing.T) {
		t.Parallel()

		bb := newBytecodeBuilder()
		strIndex := bb.addStringConst("🎉🎊")
		bb.intRegisters(2).stringRegisters(2).returnString()
		bb.emit(opLoadStringConst, 1, strIndex, 0)
		bb.emit(opLoadIntConstSmall, 0, 4, 0)
		bb.emit(opSliceString, 0, 1, 1)
		bb.emit(opExt, 0, 0, 0)
		bb.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, bb.build(), "🎊")
	})

	t.Run("slice with NUL bytes", func(t *testing.T) {
		t.Parallel()
		bb := newBytecodeBuilder()
		strIndex := bb.addStringConst("a\x00b\x00c")
		bb.intRegisters(2).stringRegisters(2).returnString()
		bb.emit(opLoadStringConst, 1, strIndex, 0)
		bb.emit(opLoadIntConstSmall, 0, 1, 0)
		bb.emit(opLoadIntConstSmall, 1, 4, 0)
		bb.emit(opSliceString, 0, 1, 3)
		bb.emit(opExt, 0, 1, 0)
		bb.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, bb.build(), "\x00b\x00")
	})

	t.Run("full slice of long multi-byte string", func(t *testing.T) {
		t.Parallel()
		bb := newBytecodeBuilder()
		strIndex := bb.addStringConst("αβγδεζηθ")
		bb.intRegisters(2).stringRegisters(2).returnString()
		bb.emit(opLoadStringConst, 1, strIndex, 0)
		bb.emit(opSliceString, 0, 1, 0)
		bb.emitExt(0)
		bb.emit(opReturn, 1, 0, 0)
		requireSyntheticResult(t, bb.build(), "αβγδεζηθ")
	})
}

func TestStringDataOpsGoDispatchCoverage(t *testing.T) {
	t.Parallel()

	t.Run("StringIndexToInt valid index", func(t *testing.T) {
		t.Parallel()
		bb := newBytecodeBuilder()
		strIndex := bb.addStringConst("hello")
		bb.intRegisters(2).stringRegisters(1).returnInt()
		bb.emit(opLoadStringConst, 0, strIndex, 0)
		bb.emit(opLoadIntConstSmall, 1, 0, 0)
		bb.emit(opStringIndexToInt, 0, 0, 1)
		bb.emit(opReturn, 1, 0, 0)
		requireGoDispatchResult(t, bb.build(), int64('h'))
	})

	t.Run("StringIndexToInt out of bounds", func(t *testing.T) {
		t.Parallel()
		bb := newBytecodeBuilder()
		strIndex := bb.addStringConst("hello")
		bb.intRegisters(2).stringRegisters(1).returnInt()
		bb.emit(opLoadStringConst, 0, strIndex, 0)
		bb.emit(opLoadIntConstSmall, 1, 5, 0)
		bb.emit(opStringIndexToInt, 0, 0, 1)
		bb.emit(opReturn, 1, 0, 0)

		_, err := execGoDispatch(t, bb.build())
		require.Error(t, err)
		require.True(t, errors.Is(err, errIndexOutOfRange), "expected errIndexOutOfRange, got: %v", err)
	})

	t.Run("SliceString no bounds", func(t *testing.T) {
		t.Parallel()
		bb := newBytecodeBuilder()
		strIndex := bb.addStringConst("hello")
		bb.intRegisters(2).stringRegisters(2).returnString()
		bb.emit(opLoadStringConst, 1, strIndex, 0)
		bb.emit(opSliceString, 0, 1, 0)
		bb.emitExt(0)
		bb.emit(opReturn, 1, 0, 0)
		requireGoDispatchResult(t, bb.build(), "hello")
	})

	t.Run("SliceString low bound only", func(t *testing.T) {
		t.Parallel()
		bb := newBytecodeBuilder()
		strIndex := bb.addStringConst("hello")
		bb.intRegisters(2).stringRegisters(2).returnString()
		bb.emit(opLoadStringConst, 1, strIndex, 0)
		bb.emit(opLoadIntConstSmall, 0, 2, 0)
		bb.emit(opSliceString, 0, 1, 1)
		bb.emit(opExt, 0, 0, 0)
		bb.emit(opReturn, 1, 0, 0)
		requireGoDispatchResult(t, bb.build(), "llo")
	})

	t.Run("SliceString high bound only", func(t *testing.T) {
		t.Parallel()
		bb := newBytecodeBuilder()
		strIndex := bb.addStringConst("hello")
		bb.intRegisters(2).stringRegisters(2).returnString()
		bb.emit(opLoadStringConst, 1, strIndex, 0)
		bb.emit(opLoadIntConstSmall, 0, 3, 0)
		bb.emit(opSliceString, 0, 1, 2)
		bb.emit(opExt, 0, 0, 0)
		bb.emit(opReturn, 1, 0, 0)
		requireGoDispatchResult(t, bb.build(), "hel")
	})

	t.Run("SliceString both bounds", func(t *testing.T) {
		t.Parallel()
		bb := newBytecodeBuilder()
		strIndex := bb.addStringConst("hello")
		bb.intRegisters(2).stringRegisters(2).returnString()
		bb.emit(opLoadStringConst, 1, strIndex, 0)
		bb.emit(opLoadIntConstSmall, 0, 1, 0)
		bb.emit(opLoadIntConstSmall, 1, 4, 0)
		bb.emit(opSliceString, 0, 1, 3)
		bb.emit(opExt, 0, 1, 0)
		bb.emit(opReturn, 1, 0, 0)
		requireGoDispatchResult(t, bb.build(), "ell")
	})

	t.Run("SliceString out of range", func(t *testing.T) {
		t.Parallel()
		bb := newBytecodeBuilder()
		strIndex := bb.addStringConst("hello")
		negIndex := bb.addIntConst(-1)
		bb.intRegisters(2).stringRegisters(2).returnString()
		bb.emit(opLoadStringConst, 1, strIndex, 0)
		bb.emit(opLoadIntConst, 0, negIndex, 0)
		bb.emit(opSliceString, 0, 1, 1)
		bb.emit(opExt, 0, 0, 0)
		bb.emit(opReturn, 1, 0, 0)

		_, err := execGoDispatch(t, bb.build())
		require.Error(t, err)
		require.True(t, errors.Is(err, errSliceOutOfRange), "expected errSliceOutOfRange, got: %v", err)
	})
}
