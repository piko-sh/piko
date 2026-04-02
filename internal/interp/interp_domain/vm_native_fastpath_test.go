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
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func makeRegisters() Registers {
	return Registers{
		ints:    make([]int64, 8),
		floats:  make([]float64, 8),
		strings: make([]string, 8),
		general: make([]reflect.Value, 8),
		bools:   make([]bool, 8),
		uints:   make([]uint64, 8),
		complex: make([]complex128, 8),
	}
}

func TestNativeFastPath_StringToString(t *testing.T) {
	t.Parallel()
	regs := makeRegisters()
	regs.strings[0] = "hello"

	site := &callSite{
		arguments: []varLocation{{register: 0, kind: registerString}},
		returns:   []varLocation{{register: 1, kind: registerString}},
	}
	reflectedFunction := reflect.ValueOf(strings.ToUpper)

	ok, _ := tryNativeFastPath(site, reflectedFunction.Interface(), &regs)
	require.True(t, ok)
	require.Equal(t, "HELLO", regs.strings[1])
}

func TestNativeFastPath_StringStringToBool(t *testing.T) {
	t.Parallel()
	regs := makeRegisters()
	regs.strings[0] = "hello world"
	regs.strings[1] = "world"

	site := &callSite{
		arguments: []varLocation{
			{register: 0, kind: registerString},
			{register: 1, kind: registerString},
		},
		returns: []varLocation{{register: 0, kind: registerBool}},
	}
	reflectedFunction := reflect.ValueOf(strings.Contains)

	ok, _ := tryNativeFastPath(site, reflectedFunction.Interface(), &regs)
	require.True(t, ok)
	require.True(t, regs.bools[0])
}

func TestNativeFastPath_IntToString(t *testing.T) {
	t.Parallel()
	regs := makeRegisters()
	regs.ints[0] = 42

	site := &callSite{
		arguments: []varLocation{{register: 0, kind: registerInt}},
		returns:   []varLocation{{register: 0, kind: registerString}},
	}
	reflectedFunction := reflect.ValueOf(strconv.Itoa)

	ok, _ := tryNativeFastPath(site, reflectedFunction.Interface(), &regs)
	require.True(t, ok)
	require.Equal(t, "42", regs.strings[0])
}

func TestNativeFastPath_FormatInt(t *testing.T) {
	t.Parallel()
	regs := makeRegisters()
	regs.ints[0] = 255
	regs.ints[1] = 16

	site := &callSite{
		arguments: []varLocation{
			{register: 0, kind: registerInt},
			{register: 1, kind: registerInt},
		},
		returns: []varLocation{{register: 0, kind: registerString}},
	}
	reflectedFunction := reflect.ValueOf(strconv.FormatInt)

	ok, _ := tryNativeFastPath(site, reflectedFunction.Interface(), &regs)
	require.True(t, ok)
	require.Equal(t, "ff", regs.strings[0])
}

func TestNativeFastPath_Variadic_Sprintf(t *testing.T) {
	t.Parallel()

	t.Run("no varargs", func(t *testing.T) {
		t.Parallel()
		regs := makeRegisters()
		regs.strings[0] = "hello"

		site := &callSite{
			arguments: []varLocation{{register: 0, kind: registerString}},
			returns:   []varLocation{{register: 1, kind: registerString}},
		}
		reflectedFunction := reflect.ValueOf(fmt.Sprintf)

		ok, _ := tryNativeFastPath(site, reflectedFunction.Interface(), &regs)
		require.True(t, ok)
		require.Equal(t, "hello", regs.strings[1])
	})

	t.Run("one string argument", func(t *testing.T) {
		t.Parallel()
		regs := makeRegisters()
		regs.strings[0] = "hello %s"
		regs.strings[1] = "world"

		site := &callSite{
			arguments: []varLocation{
				{register: 0, kind: registerString},
				{register: 1, kind: registerString},
			},
			returns: []varLocation{{register: 2, kind: registerString}},
		}
		reflectedFunction := reflect.ValueOf(fmt.Sprintf)

		ok, _ := tryNativeFastPath(site, reflectedFunction.Interface(), &regs)
		require.True(t, ok)
		require.Equal(t, "hello world", regs.strings[2])
	})

	t.Run("one int argument", func(t *testing.T) {
		t.Parallel()
		regs := makeRegisters()
		regs.strings[0] = "value: %d"
		regs.ints[0] = 42

		site := &callSite{
			arguments: []varLocation{
				{register: 0, kind: registerString},
				{register: 0, kind: registerInt},
			},
			returns: []varLocation{{register: 1, kind: registerString}},
		}
		reflectedFunction := reflect.ValueOf(fmt.Sprintf)

		ok, _ := tryNativeFastPath(site, reflectedFunction.Interface(), &regs)
		require.True(t, ok)
		require.Equal(t, "value: 42", regs.strings[1])
	})

	t.Run("mixed arguments", func(t *testing.T) {
		t.Parallel()
		regs := makeRegisters()
		regs.strings[0] = "%s=%d (%.1f)"
		regs.strings[1] = "x"
		regs.ints[0] = 10
		regs.floats[0] = 3.14

		site := &callSite{
			arguments: []varLocation{
				{register: 0, kind: registerString},
				{register: 1, kind: registerString},
				{register: 0, kind: registerInt},
				{register: 0, kind: registerFloat},
			},
			returns: []varLocation{{register: 2, kind: registerString}},
		}
		reflectedFunction := reflect.ValueOf(fmt.Sprintf)

		ok, _ := tryNativeFastPath(site, reflectedFunction.Interface(), &regs)
		require.True(t, ok)
		require.Equal(t, "x=10 (3.1)", regs.strings[2])
	})

	t.Run("buffer reuse", func(t *testing.T) {
		t.Parallel()
		regs := makeRegisters()
		site := &callSite{
			arguments: []varLocation{
				{register: 0, kind: registerString},
				{register: 0, kind: registerInt},
			},
			returns: []varLocation{{register: 1, kind: registerString}},
		}
		reflectedFunction := reflect.ValueOf(fmt.Sprintf)

		regs.strings[0] = "v=%d"
		regs.ints[0] = 1
		ok, _ := tryNativeFastPath(site, reflectedFunction.Interface(), &regs)
		require.True(t, ok)
		require.Equal(t, "v=1", regs.strings[1])

		regs.strings[0] = "v=%d"
		regs.ints[0] = 2
		ok, _ = tryNativeFastPath(site, reflectedFunction.Interface(), &regs)
		require.True(t, ok)
		require.Equal(t, "v=2", regs.strings[1])
		require.NotNil(t, site.variadicArgumentsBuffer)
	})
}

func TestNativeFastPath_Variadic_Errorf(t *testing.T) {
	t.Parallel()

	regs := makeRegisters()
	regs.strings[0] = "bad: %s"
	regs.strings[1] = "reason"

	site := &callSite{
		arguments: []varLocation{
			{register: 0, kind: registerString},
			{register: 1, kind: registerString},
		},
		returns: []varLocation{{register: 0, kind: registerGeneral}},
	}
	reflectedFunction := reflect.ValueOf(fmt.Errorf)

	ok, _ := tryNativeFastPath(site, reflectedFunction.Interface(), &regs)
	require.True(t, ok)
	require.True(t, regs.general[0].IsValid())
	err, ok := reflect.TypeAssert[error](regs.general[0])
	require.True(t, ok)
	require.Equal(t, "bad: reason", err.Error())
}

func TestNativeFastPath_Variadic_Sprint(t *testing.T) {
	t.Parallel()

	regs := makeRegisters()
	regs.strings[0] = "hello"
	regs.strings[1] = " "
	regs.strings[2] = "world"

	site := &callSite{
		arguments: []varLocation{
			{register: 0, kind: registerString},
			{register: 1, kind: registerString},
			{register: 2, kind: registerString},
		},
		returns: []varLocation{{register: 3, kind: registerString}},
	}
	reflectedFunction := reflect.ValueOf(fmt.Sprint)

	ok, _ := tryNativeFastPath(site, reflectedFunction.Interface(), &regs)
	require.True(t, ok)
	require.Equal(t, "hello world", regs.strings[3])
}

func TestNativeFastPath_ZeroArgString(t *testing.T) {
	t.Parallel()
	regs := makeRegisters()

	called := false
	reflectedFunction := reflect.ValueOf(func() string { called = true; return "result" })

	site := &callSite{
		arguments: nil,
		returns:   []varLocation{{register: 0, kind: registerString}},
	}

	ok, _ := tryNativeFastPath(site, reflectedFunction.Interface(), &regs)
	require.True(t, ok)
	require.True(t, called)
	require.Equal(t, "result", regs.strings[0])
}

func TestNativeFastPath_ZeroArgBool(t *testing.T) {
	t.Parallel()
	regs := makeRegisters()

	reflectedFunction := reflect.ValueOf(func() bool { return true })

	site := &callSite{
		arguments: nil,
		returns:   []varLocation{{register: 0, kind: registerBool}},
	}

	ok, _ := tryNativeFastPath(site, reflectedFunction.Interface(), &regs)
	require.True(t, ok)
	require.True(t, regs.bools[0])
}

func TestNativeFastPath_ZeroArgInt(t *testing.T) {
	t.Parallel()
	regs := makeRegisters()

	reflectedFunction := reflect.ValueOf(func() int { return 99 })

	site := &callSite{
		arguments: nil,
		returns:   []varLocation{{register: 0, kind: registerInt}},
	}

	ok, _ := tryNativeFastPath(site, reflectedFunction.Interface(), &regs)
	require.True(t, ok)
	require.Equal(t, int64(99), regs.ints[0])
}

func TestNativeFastPath_VoidString(t *testing.T) {
	t.Parallel()
	regs := makeRegisters()
	regs.strings[0] = "hello"

	var received string
	reflectedFunction := reflect.ValueOf(func(s string) { received = s })

	site := &callSite{
		arguments: []varLocation{{register: 0, kind: registerString}},
		returns:   nil,
	}

	ok, _ := tryNativeFastPath(site, reflectedFunction.Interface(), &regs)
	require.True(t, ok)
	require.Equal(t, "hello", received)
}

func TestNativeFastPath_VoidTwoStrings(t *testing.T) {
	t.Parallel()
	regs := makeRegisters()
	regs.strings[0] = "key"
	regs.strings[1] = "value"

	var a, b string
	reflectedFunction := reflect.ValueOf(func(k, v string) { a = k; b = v })

	site := &callSite{
		arguments: []varLocation{
			{register: 0, kind: registerString},
			{register: 1, kind: registerString},
		},
		returns: nil,
	}

	ok, _ := tryNativeFastPath(site, reflectedFunction.Interface(), &regs)
	require.True(t, ok)
	require.Equal(t, "key", a)
	require.Equal(t, "value", b)
}

func TestNativeFastPath_StringToError(t *testing.T) {
	t.Parallel()
	regs := makeRegisters()
	regs.strings[0] = "test input"

	reflectedFunction := reflect.ValueOf(func(s string) error {
		return fmt.Errorf("err: %s", s)
	})

	site := &callSite{
		arguments: []varLocation{{register: 0, kind: registerString}},
		returns:   []varLocation{{register: 0, kind: registerGeneral}},
	}

	ok, _ := tryNativeFastPath(site, reflectedFunction.Interface(), &regs)
	require.True(t, ok)
	require.True(t, regs.general[0].IsValid())
	err, ok := reflect.TypeAssert[error](regs.general[0])
	require.True(t, ok)
	require.Equal(t, "err: test input", err.Error())
}

func TestNativeFastPath_StringToError_Nil(t *testing.T) {
	t.Parallel()
	regs := makeRegisters()
	regs.strings[0] = "ok"

	reflectedFunction := reflect.ValueOf(func(s string) error { return nil })

	site := &callSite{
		arguments: []varLocation{{register: 0, kind: registerString}},
		returns:   []varLocation{{register: 0, kind: registerGeneral}},
	}

	ok, _ := tryNativeFastPath(site, reflectedFunction.Interface(), &regs)
	require.True(t, ok)
	require.False(t, regs.general[0].IsValid())
}

func TestNativeFastPath_AnyToBool(t *testing.T) {
	t.Parallel()
	regs := makeRegisters()
	regs.strings[0] = "hello"

	site := &callSite{
		arguments: []varLocation{{register: 0, kind: registerString}},
		returns:   []varLocation{{register: 0, kind: registerBool}},
	}
	reflectedFunction := reflect.ValueOf(func(v any) bool { return v != nil && v != "" })

	ok, _ := tryNativeFastPath(site, reflectedFunction.Interface(), &regs)
	require.True(t, ok)
	require.True(t, regs.bools[0])
}

func TestNativeFastPath_AnyToString(t *testing.T) {
	t.Parallel()
	regs := makeRegisters()
	regs.ints[0] = 42

	site := &callSite{
		arguments: []varLocation{{register: 0, kind: registerInt}},
		returns:   []varLocation{{register: 0, kind: registerString}},
	}
	reflectedFunction := reflect.ValueOf(func(v any) string { return fmt.Sprint(v) })

	ok, _ := tryNativeFastPath(site, reflectedFunction.Interface(), &regs)
	require.True(t, ok)
	require.Equal(t, "42", regs.strings[0])
}

func TestNativeFastPath_AnyToInt(t *testing.T) {
	t.Parallel()
	regs := makeRegisters()
	regs.strings[0] = "ignored"

	site := &callSite{
		arguments: []varLocation{{register: 0, kind: registerString}},
		returns:   []varLocation{{register: 0, kind: registerInt}},
	}
	reflectedFunction := reflect.ValueOf(func(v any) int { return 99 })

	ok, _ := tryNativeFastPath(site, reflectedFunction.Interface(), &regs)
	require.True(t, ok)
	require.Equal(t, int64(99), regs.ints[0])
}

func TestNativeFastPath_AnyToInt64(t *testing.T) {
	t.Parallel()
	regs := makeRegisters()
	regs.floats[0] = 3.14

	site := &callSite{
		arguments: []varLocation{{register: 0, kind: registerFloat}},
		returns:   []varLocation{{register: 0, kind: registerInt}},
	}
	reflectedFunction := reflect.ValueOf(func(v any) int64 { return 123 })

	ok, _ := tryNativeFastPath(site, reflectedFunction.Interface(), &regs)
	require.True(t, ok)
	require.Equal(t, int64(123), regs.ints[0])
}

func TestNativeFastPath_AnyToFloat64(t *testing.T) {
	t.Parallel()
	regs := makeRegisters()
	regs.ints[0] = 10

	site := &callSite{
		arguments: []varLocation{{register: 0, kind: registerInt}},
		returns:   []varLocation{{register: 0, kind: registerFloat}},
	}
	reflectedFunction := reflect.ValueOf(func(v any) float64 { return 2.5 })

	ok, _ := tryNativeFastPath(site, reflectedFunction.Interface(), &regs)
	require.True(t, ok)
	require.Equal(t, 2.5, regs.floats[0])
}

func TestNativeFastPath_AnyAnyToAny(t *testing.T) {
	t.Parallel()

	t.Run("returns first truthy", func(t *testing.T) {
		t.Parallel()
		regs := makeRegisters()
		regs.strings[0] = ""
		regs.strings[1] = "fallback"

		site := &callSite{
			arguments: []varLocation{
				{register: 0, kind: registerString},
				{register: 1, kind: registerString},
			},
			returns: []varLocation{{register: 0, kind: registerGeneral}},
		}
		reflectedFunction := reflect.ValueOf(func(a, b any) any {
			if a != nil && a != "" {
				return a
			}
			return b
		})

		ok, _ := tryNativeFastPath(site, reflectedFunction.Interface(), &regs)
		require.True(t, ok)
		require.True(t, regs.general[0].IsValid())
		require.Equal(t, "fallback", regs.general[0].Interface())
	})

	t.Run("returns nil", func(t *testing.T) {
		t.Parallel()
		regs := makeRegisters()

		site := &callSite{
			arguments: []varLocation{
				{register: 0, kind: registerString},
				{register: 1, kind: registerString},
			},
			returns: []varLocation{{register: 0, kind: registerGeneral}},
		}
		reflectedFunction := reflect.ValueOf(func(a, b any) any { return nil })

		ok, _ := tryNativeFastPath(site, reflectedFunction.Interface(), &regs)
		require.True(t, ok)
		require.False(t, regs.general[0].IsValid())
	})
}

func TestNativeFastPath_ZeroArgVoid(t *testing.T) {
	t.Parallel()
	called := false
	reflectedFunction := reflect.ValueOf(func() { called = true })

	site := &callSite{
		arguments: nil,
		returns:   nil,
	}

	ok, _ := tryNativeFastPath(site, reflectedFunction.Interface(), new(makeRegisters()))
	require.True(t, ok)
	require.True(t, called)
}

func TestNativeFastPath_Int64Void(t *testing.T) {
	t.Parallel()
	regs := makeRegisters()
	regs.ints[0] = 42

	var received int64
	reflectedFunction := reflect.ValueOf(func(v int64) { received = v })

	site := &callSite{
		arguments: []varLocation{{register: 0, kind: registerInt}},
		returns:   nil,
	}

	ok, _ := tryNativeFastPath(site, reflectedFunction.Interface(), &regs)
	require.True(t, ok)
	require.Equal(t, int64(42), received)
}

func TestNativeFastPath_NoMatch(t *testing.T) {
	t.Parallel()

	reflectedFunction := reflect.ValueOf(func(a, b, c, d, e int) int { return a })

	site := &callSite{
		arguments: []varLocation{{register: 0, kind: registerInt}},
		returns:   []varLocation{{register: 1, kind: registerInt}},
	}

	ok, _ := tryNativeFastPath(site, reflectedFunction.Interface(), new(makeRegisters()))
	require.False(t, ok)
	require.Equal(t, nativeFastPathNone, site.nativeFastPath)
}

func TestNativeFastPath_RepeatedCalls(t *testing.T) {
	t.Parallel()

	regs := makeRegisters()
	regs.strings[0] = "test"

	site := &callSite{
		arguments: []varLocation{{register: 0, kind: registerString}},
		returns:   []varLocation{{register: 1, kind: registerString}},
	}
	reflectedFunction := reflect.ValueOf(strings.ToUpper)

	ok, _ := tryNativeFastPath(site, reflectedFunction.Interface(), &regs)
	require.True(t, ok)
	require.Equal(t, "TEST", regs.strings[1])

	regs.strings[0] = "again"
	ok, _ = tryNativeFastPath(site, reflectedFunction.Interface(), &regs)
	require.True(t, ok)
	require.Equal(t, "AGAIN", regs.strings[1])
}

func TestNativeFastPath_BoundMethodNotStale(t *testing.T) {
	t.Parallel()

	regs := makeRegisters()

	site := &callSite{
		arguments: []varLocation{{register: 0, kind: registerString}},
		returns: []varLocation{
			{register: 0, kind: registerInt},
			{register: 0, kind: registerGeneral},
		},
	}

	b1 := &strings.Builder{}
	b2 := &strings.Builder{}

	fn1 := reflect.ValueOf(b1).MethodByName("WriteString")
	fn2 := reflect.ValueOf(b2).MethodByName("WriteString")

	regs.strings[0] = "hello"
	ok, _ := tryNativeFastPath(site, fn1.Interface(), &regs)
	require.True(t, ok)
	require.Equal(t, "hello", b1.String())

	regs.strings[0] = "world"
	ok, _ = tryNativeFastPath(site, fn2.Interface(), &regs)
	require.True(t, ok)

	require.Equal(t, "hello", b1.String(), "b1 should not have been modified by second call")
	require.Equal(t, "world", b2.String(), "b2 should have received the second write")
}
