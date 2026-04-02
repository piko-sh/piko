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
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNamedReturns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect any
	}{
		{"named_return_int", `
func f() (result int) {
	result = 42
	return
}
f()`, int64(42)},
		{"named_return_string", `
func f() (s string) {
	s = "hello"
	return
}
f()`, "hello"},
		{"named_return_multiple", `
func f() (a int, b int) {
	a = 10
	b = 20
	return
}
x, y := f()
x + y`, int64(30)},
		{"named_return_modified", `
func f() (x int) {
	x = 1
	x += 9
	return
}
f()`, int64(10)},
		{"named_return_with_defer", `
func f() (x int) {
	defer func() { x = 99 }()
	x = 1
	return
}
f()`, int64(99)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestImplicitReturns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect any
	}{
		{"void_func", `
func f() {}
f()
1`, int64(1)},
		{"void_with_work", `
func f() int { return 0 }
f()
1`, int64(1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestTypeAssertions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect any
	}{
		{"assert_int", `var x any = 42; x.(int)`, int(42)},
		{"assert_string", `var x any = "hello"; x.(string)`, "hello"},
		{"assert_float", `var x any = 3.14; x.(float64)`, float64(3.14)},
		{"assert_bool", `var x any = true; x.(bool)`, true},
		{"assert_comma_ok_true", `
var x any = 42
_, ok := x.(int)
ok`, true},
		{"assert_comma_ok_false", `
var x any = "hello"
_, ok := x.(int)
ok`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestMultiReturnCopyPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect any
	}{
		{"two_ints", `
func f() (int, int) { return 10, 20 }
a, b := f()
a + b`, int64(30)},
		{"int_and_string", `
func f() (int, string) { return 42, "hello" }
_, s := f()
s`, "hello"},
		{"three_values", `
func f() (int, int, int) { return 1, 2, 3 }
a, b, c := f()
a + b + c`, int64(6)},
		{"bool_return", `
func f() (int, bool) { return 42, true }
_, ok := f()
ok`, true},
		{"float_return", `
func f() (float64, float64) { return 1.5, 2.5 }
a, b := f()
a + b`, float64(4.0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestUpvalueSync(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect any
	}{
		{"capture_and_modify", `
x := 10
f := func() { x = 20 }
f()
x`, int64(20)},
		{"capture_multiple", `
func f() int {
	a := 1
	b := 2
	g := func() int { return a + b }
	return g()
}
f()`, int64(3)},
		{"counter_closure", `
func makeCounter() func() int {
	n := 0
	return func() int {
		n++
		return n
	}
}
c := makeCounter()
c()
c()
c()`, int64(3)},
		{"nested_capture", `
func f() int {
	x := 42
	g := func() func() int {
		return func() int { return x }
	}
	return g()()
}
f()`, int64(42)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestReflectBinaryOps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect any
	}{
		{"interface_add", `
var x any = 10
var y any = 20
x.(int) + y.(int)`, int(30)},
		{"interface_nil_check", `
var x any
x == nil`, true},
		{"interface_non_nil", `
var x any = "hello"
x != nil`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestTailCallPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect any
	}{
		{"simple_tail", `
func f(n int) int {
	if n <= 0 { return 0 }
	return f(n - 1)
}
f(10)`, int64(0)},
		{"tail_accumulator", `
func sum(n, acc int) int {
	if n <= 0 { return acc }
	return sum(n-1, acc+n)
}
sum(10, 0)`, int64(55)},
		{"tail_with_string", `
func repeat(s string, n int) string {
	if n <= 0 { return s }
	return repeat(s + "a", n-1)
}
len(repeat("", 5))`, int64(5)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestDeferPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect any
	}{
		{"defer_modifies_named_return", `
func f() (x int) {
	defer func() { x += 10 }()
	x = 5
	return
}
f()`, int64(15)},
		{"defer_order_lifo", `
func f() string {
	result := ""
	defer func() { result += "3" }()
	defer func() { result += "2" }()
	defer func() { result += "1" }()
	return result
}
f()`, ""},
		{"recover_from_panic", `
func f() string {
	defer func() {
		recover()
	}()
	panic("boom")
}
f()`, ""},
		{"defer_runs_on_return", `
func f() int {
	x := 0
	defer func() { x = 99 }()
	return x
}
f()`, int64(0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestZeroTypedRegister(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect any
	}{
		{"zero_int_var", `var x int; x`, int64(0)},
		{"zero_float_var", `var f float64; f`, float64(0)},
		{"zero_string_var", `var s string; s`, ""},
		{"zero_bool_var", `var b bool; b`, false},
		{"zero_slice_nil", `var s []int; s == nil`, true},
		{"zero_map_nil", `var m map[string]int; m == nil`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestVMRangeOverTypedSlices(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source string
		expect any
	}{
		{
			name: "range_over_int_slice",
			source: `package main

func run() int {
	sum := 0
	for _, v := range []int{10, 20, 30} {
		sum += v
	}
	return sum
}

func main() {}
`,
			expect: int64(60),
		},
		{
			name: "range_over_string_slice",
			source: `package main

func run() string {
	result := ""
	for _, s := range []string{"a", "b", "c"} {
		result += s
	}
	return result
}

func main() {}
`,
			expect: "abc",
		},
		{
			name: "range_over_float64_slice",
			source: `package main

func run() float64 {
	sum := 0.0
	for _, v := range []float64{1.5, 2.5, 3.0} {
		sum += v
	}
	return sum
}

func main() {}
`,
			expect: float64(7.0),
		},
		{
			name: "range_over_bool_slice",
			source: `package main

func run() int {
	count := 0
	for _, v := range []bool{true, true, false, true} {
		if v { count++ }
	}
	return count
}

func main() {}
`,
			expect: int64(3),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.EvalFile(context.Background(), tt.source, "run")
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestVMRangeOverString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect any
	}{
		{
			name:   "range_over_string_with_index_and_rune",
			code:   `sum := 0; for i, _ := range "abc" { sum += i }; sum`,
			expect: int64(3),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestVMAppendFastPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect any
	}{
		{
			name:   "append_int",
			code:   `s := []int{1, 2}; s = append(s, 3, 4); len(s)`,
			expect: int64(4),
		},
		{
			name:   "append_string",
			code:   `s := []string{"a"}; s = append(s, "b", "c"); s[2]`,
			expect: "c",
		},
		{
			name:   "append_float",
			code:   `s := []float64{1.0}; s = append(s, 2.0); s[1]`,
			expect: float64(2.0),
		},
		{
			name:   "append_bool",
			code:   `s := []bool{true}; s = append(s, false); s[1]`,
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestVMMapOperations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source string
		expect any
	}{
		{
			name: "map_int_int",
			source: `package main

func run() int {
	m := make(map[int]int)
	m[1] = 10
	m[2] = 20
	return m[1] + m[2]
}

func main() {}
`,
			expect: int64(30),
		},
		{
			name: "map_int_string_slow_path",
			source: `package main

func run() string {
	m := make(map[int]string)
	m[1] = "hello"
	m[2] = "world"
	return m[1]
}

func main() {}
`,
			expect: "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.EvalFile(context.Background(), tt.source, "run")
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestVMSelectWithSend(t *testing.T) {
	t.Parallel()

	source := `package main

func run() int {
	ch := make(chan int, 1)
	select {
	case ch <- 42:
	default:
	}
	return <-ch
}

func main() {}
`
	service := NewService()
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, int64(42), result)
}

func TestVMGoroutines(t *testing.T) {
	t.Parallel()

	source := `package main

func run() int {
	ch := make(chan int, 5)
	for i := 0; i < 5; i++ {
		go func(n int) {
			ch <- n
		}(i)
	}
	sum := 0
	for i := 0; i < 5; i++ {
		sum += <-ch
	}
	return sum
}

func main() {}
`
	service := NewService()
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, int64(10), result)
}

func TestVMBuiltinPrint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect any
	}{
		{
			name:   "print_int",
			code:   `print(42); true`,
			expect: true,
		},
		{
			name:   "print_string",
			code:   `print("hello"); true`,
			expect: true,
		},
		{
			name:   "print_float",
			code:   `print(3.14); true`,
			expect: true,
		},
		{
			name:   "print_bool",
			code:   `print(true); true`,
			expect: true,
		},
		{
			name:   "print_multiple",
			code:   `print(1, " ", "hello", " ", true); true`,
			expect: true,
		},
		{
			name:   "println_multiple",
			code:   `println(1, "hello", true); true`,
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestVMClosureCaptures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source string
		expect any
	}{
		{
			name: "capture_int",
			source: `package main

func run() int {
	x := 42
	f := func() int { return x }
	return f()
}

func main() {}
`,
			expect: int64(42),
		},
		{
			name: "capture_string",
			source: `package main

func run() string {
	s := "hello"
	f := func() string { return s }
	return f()
}

func main() {}
`,
			expect: "hello",
		},
		{
			name: "capture_float",
			source: `package main

func run() float64 {
	x := 3.14
	f := func() float64 { return x }
	return f()
}

func main() {}
`,
			expect: float64(3.14),
		},
		{
			name: "capture_bool",
			source: `package main

func run() bool {
	b := true
	f := func() bool { return b }
	return f()
}

func main() {}
`,
			expect: true,
		},
		{
			name: "capture_uint",
			source: `package main

func run() uint {
	x := uint(42)
	f := func() uint { return x }
	return f()
}

func main() {}
`,
			expect: uint64(42),
		},
		{
			name: "capture_complex",
			source: `package main

func run() complex128 {
	c := complex(1.0, 2.0)
	f := func() complex128 { return c }
	return f()
}

func main() {}
`,
			expect: complex128(1 + 2i),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.EvalFile(context.Background(), tt.source, "run")
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestVMNativeFastPathDispatch(t *testing.T) {
	t.Parallel()

	fns := map[string]reflect.Value{

		"ToUpper": reflect.ValueOf(func(s string) string {
			result := make([]byte, len(s))
			for i, b := range []byte(s) {
				if b >= 'a' && b <= 'z' {
					result[i] = b - 32
				} else {
					result[i] = b
				}
			}
			return string(result)
		}),

		"StrLen": reflect.ValueOf(func(s string) int { return len(s) }),

		"IsEmpty": reflect.ValueOf(func(s string) bool { return len(s) == 0 }),

		"StrEq": reflect.ValueOf(func(a, b string) bool { return a == b }),

		"Concat": reflect.ValueOf(func(a, b string) string { return a + b }),

		"CmpLen": reflect.ValueOf(func(a, b string) int { return len(a) - len(b) }),

		"IntStr": reflect.ValueOf(func(n int) string {
			if n == 0 {
				return "0"
			}
			return "nonzero"
		}),

		"Double": reflect.ValueOf(func(n int) int { return n * 2 }),

		"IsPos": reflect.ValueOf(func(n int) bool { return n > 0 }),

		"AddInts": reflect.ValueOf(func(a, b int) int { return a + b }),

		"IntEq": reflect.ValueOf(func(a, b int) bool { return a == b }),

		"IntPair": reflect.ValueOf(func(a, b int) string { return "pair" }),

		"Halve": reflect.ValueOf(func(f float64) float64 { return f / 2 }),

		"AddF": reflect.ValueOf(func(a, b float64) float64 { return a + b }),

		"IsNil": reflect.ValueOf(func(v any) bool { return v == nil }),

		"AnyStr": reflect.ValueOf(func(v any) string { return "val" }),

		"AnyInt": reflect.ValueOf(func(v any) int { return 1 }),

		"Hello": reflect.ValueOf(func() string { return "hello" }),

		"Yes": reflect.ValueOf(func() bool { return true }),

		"One": reflect.ValueOf(func() int { return 1 }),

		"Pi": reflect.ValueOf(func() float64 { return 3.14 }),

		"Noop": reflect.ValueOf(func() {}),

		"Log": reflect.ValueOf(func(s string) {}),

		"ConsumeI": reflect.ValueOf(func(n int) {}),

		"ConsumeB": reflect.ValueOf(func(b bool) {}),

		"Log2": reflect.ValueOf(func(a, b string) {}),

		"NoErr": reflect.ValueOf(func() error { return nil }),

		"ValidateS": reflect.ValueOf(func(s string) error { return nil }),
	}

	service := newTestServiceWithFunctions(t, "fp", fns)

	source := `package main

import "fp"

func run() int {
	sum := 0

	// String -> String (twice)
	_ = fp.ToUpper("abc")
	_ = fp.ToUpper("def")

	// String -> Int
	sum += fp.StrLen("hello")
	sum += fp.StrLen("world")

	// String -> Bool
	if !fp.IsEmpty("x") { sum += 1 }
	if fp.IsEmpty("") { sum += 1 }

	// String, String -> Bool
	if fp.StrEq("a", "a") { sum += 1 }
	if !fp.StrEq("a", "b") { sum += 1 }

	// String, String -> String
	_ = fp.Concat("a", "b")
	_ = fp.Concat("c", "d")

	// String, String -> Int
	sum += fp.CmpLen("abc", "ab")
	_ = fp.CmpLen("x", "y")

	// Int -> String
	_ = fp.IntStr(1)
	_ = fp.IntStr(0)

	// Int -> Int
	sum += fp.Double(3)
	sum += fp.Double(4)

	// Int -> Bool
	if fp.IsPos(1) { sum += 1 }
	if !fp.IsPos(-1) { sum += 1 }

	// Int, Int -> Int
	sum += fp.AddInts(2, 3)
	sum += fp.AddInts(4, 5)

	// Int, Int -> Bool
	if fp.IntEq(1, 1) { sum += 1 }
	_ = fp.IntEq(1, 2)

	// Int, Int -> String
	_ = fp.IntPair(1, 2)
	_ = fp.IntPair(3, 4)

	// Float -> Float
	f := fp.Halve(10.0)
	f += fp.Halve(20.0)
	sum += int(f)

	// Float, Float -> Float
	f2 := fp.AddF(1.0, 2.0)
	f2 += fp.AddF(3.0, 4.0)
	sum += int(f2)

	// Any -> Bool
	if fp.IsNil(nil) { sum += 1 }
	_ = fp.IsNil(42)

	// Any -> String
	_ = fp.AnyStr(1)
	_ = fp.AnyStr("x")

	// Any -> Int
	sum += fp.AnyInt(nil)
	sum += fp.AnyInt(1)

	// () -> String
	_ = fp.Hello()
	_ = fp.Hello()

	// () -> Bool
	if fp.Yes() { sum += 1 }
	_ = fp.Yes()

	// () -> Int
	sum += fp.One()
	sum += fp.One()

	// () -> Float64
	_ = fp.Pi()
	_ = fp.Pi()

	// () -> Error (nil)
	_ = fp.NoErr()
	_ = fp.NoErr()

	// Void
	fp.Noop()
	fp.Noop()

	// Void(string)
	fp.Log("a")
	fp.Log("b")

	// Void(int)
	fp.ConsumeI(1)
	fp.ConsumeI(2)

	// Void(bool)
	fp.ConsumeB(true)
	fp.ConsumeB(false)

	// Void(string, string)
	fp.Log2("a", "b")
	fp.Log2("c", "d")

	// String -> Error
	_ = fp.ValidateS("x")
	_ = fp.ValidateS("y")

	return sum
}

func main() {}
`
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)

	require.IsType(t, int64(0), result)
	require.Greater(t, result.(int64), int64(30))
}

func TestVMNativeFunctionReturns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		fns    map[string]reflect.Value
		source string
		expect any
	}{
		{
			name: "native_returns_uint",
			fns: map[string]reflect.Value{
				"GetUint": reflect.ValueOf(func() uint64 { return 42 }),
			},
			source: `package main

import "testpkg"

func run() uint64 { return testpkg.GetUint() }

func main() {}
`,
			expect: uint64(42),
		},
		{
			name: "native_returns_complex",
			fns: map[string]reflect.Value{
				"GetComplex": reflect.ValueOf(func() complex128 { return 1 + 2i }),
			},
			source: `package main

import "testpkg"

func run() complex128 { return testpkg.GetComplex() }

func main() {}
`,
			expect: complex128(1 + 2i),
		},
		{
			name: "native_returns_bool",
			fns: map[string]reflect.Value{
				"GetBool": reflect.ValueOf(func() bool { return true }),
			},
			source: `package main

import "testpkg"

func run() bool { return testpkg.GetBool() }

func main() {}
`,
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := newTestServiceWithFunctions(t, "testpkg", tt.fns)
			result, err := service.EvalFile(context.Background(), tt.source, "run")
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestVMRepeatedMethodCalls(t *testing.T) {
	t.Parallel()

	source := `package main

type Counter struct {
	Value int
}

func (c *Counter) Inc() {
	c.Value++
}

func run() int {
	c := &Counter{}
	c.Inc()
	c.Inc()
	c.Inc()
	return c.Value
}

func main() {}
`
	service := NewService()
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, int64(3), result)
}

func TestVMRangeOverMap(t *testing.T) {
	t.Parallel()

	source := `package main

func run() int {
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	sum := 0
	for _, v := range m {
		sum += v
	}
	return sum
}

func main() {}
`
	service := NewService()
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, int64(6), result)
}

func TestVMRangeOverChannel(t *testing.T) {
	t.Parallel()

	source := `package main

func run() int {
	ch := make(chan int, 3)
	ch <- 10
	ch <- 20
	ch <- 30
	close(ch)
	sum := 0
	for v := range ch {
		sum += v
	}
	return sum
}

func main() {}
`
	service := NewService()
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, int64(60), result)
}

func TestVMNativeGoroutine(t *testing.T) {
	t.Parallel()

	ch := make(chan int64, 1)
	fns := map[string]reflect.Value{
		"Send": reflect.ValueOf(func(v int64) { ch <- v }),
	}
	service := newTestServiceWithFunctions(t, "testpkg", fns)

	source := `package main

import "testpkg"

func run() int {
	go testpkg.Send(42)
	return 1
}

func main() {}
`
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, int64(1), result)

	v := <-ch
	require.Equal(t, int64(42), v)
}

func TestVMNativeGoroutineCoercion(t *testing.T) {
	t.Parallel()

	ch := make(chan int, 1)
	fns := map[string]reflect.Value{
		"Send": reflect.ValueOf(func(v int) { ch <- v }),
	}
	service := newTestServiceWithFunctions(t, "testpkg", fns)

	source := `package main

import "testpkg"

func run() int {
	go testpkg.Send(42)
	return 1
}

func main() {}
`
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, int64(1), result)

	v := <-ch
	require.Equal(t, 42, v)
}

func TestVMDeferTypedArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source string
		expect any
	}{
		{
			name: "defer_with_float_arg",
			source: `package main

var result float64

func capture(f float64) { result = f }

func run() float64 {
	defer capture(3.14)
	return result
}

func main() {}
`,
			expect: float64(0),
		},
		{
			name: "defer_with_bool_arg",
			source: `package main

var result int

func check(b bool) { if b { result = 1 } }

func run() int {
	defer check(true)
	return 0
}

func main() {}
`,
			expect: int64(0),
		},
		{
			name: "defer_with_uint_arg",
			source: `package main

var result uint

func store(u uint) { result = u }

func run() uint {
	defer store(42)
	return result
}

func main() {}
`,
			expect: uint64(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.EvalFile(context.Background(), tt.source, "run")
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestVMPrintUintComplex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source string
		expect any
	}{
		{
			name: "print_uint",
			source: `package main

func run() uint {
	var x uint = 42
	print(x)
	return x
}

func main() {}
`,
			expect: uint64(42),
		},
		{
			name: "print_complex",
			source: `package main

func run() complex128 {
	c := complex(1.0, 2.0)
	print(c)
	return c
}

func main() {}
`,
			expect: complex128(1 + 2i),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.EvalFile(context.Background(), tt.source, "run")
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestVMTypeAssertUintFloat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source string
		expect any
	}{
		{
			name: "type_assert_to_uint",
			source: `package main

func run() uint {
	var x interface{} = uint(42)
	return x.(uint)
}

func main() {}
`,
			expect: uint64(42),
		},
		{
			name: "type_assert_to_float",
			source: `package main

func run() float64 {
	var x interface{} = 3.14
	return x.(float64)
}

func main() {}
`,
			expect: float64(3.14),
		},
		{
			name: "type_assert_to_bool",
			source: `package main

func run() bool {
	var x interface{} = true
	return x.(bool)
}

func main() {}
`,
			expect: true,
		},
		{
			name: "type_assert_to_complex",
			source: `package main

func run() complex128 {
	var x interface{} = complex(1.0, 2.0)
	return x.(complex128)
}

func main() {}
`,
			expect: complex128(1 + 2i),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.EvalFile(context.Background(), tt.source, "run")
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestVMDivisionByZero(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source string
	}{
		{
			name: "int_division_by_zero",
			source: `package main

func run() int {
	a := 10
	b := 0
	return a / b
}

func main() {}
`,
		},
		{
			name: "int_remainder_by_zero",
			source: `package main

func run() int {
	a := 10
	b := 0
	return a % b
}

func main() {}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			_, err := service.EvalFile(context.Background(), tt.source, "run")
			require.Error(t, err)
		})
	}
}

func TestVMNamedReturnDefer(t *testing.T) {
	t.Parallel()

	source := `package main

func compute() (result int) {
	defer func() {
		result = 42
	}()
	return 0
}

func run() int {
	return compute()
}

func main() {}
`
	service := NewService()
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, int64(42), result)
}

func TestVMFunctionReturnsUintComplex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source string
		expect any
	}{
		{
			name: "returns_uint",
			source: `package main

func getUint() uint { return 42 }
func run() uint { return getUint() }
func main() {}
`,
			expect: uint64(42),
		},
		{
			name: "returns_complex",
			source: `package main

func getComplex() complex128 { return complex(3.0, 4.0) }
func run() complex128 { return getComplex() }
func main() {}
`,
			expect: complex128(3 + 4i),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.EvalFile(context.Background(), tt.source, "run")
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestVMPanicRecover(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source string
		expect any
	}{
		{
			name: "recover_in_called_function",
			source: `package main

func safe() int {
	defer func() { recover() }()
	panic(42)
}

func run() int {
	safe()
	return 1
}

func main() {}
`,
			expect: int64(1),
		},
		{
			name: "recover_in_entry_function",
			source: `package main

func run() int {
	defer func() { recover() }()
	panic(42)
}

func main() {}
`,
			expect: nil,
		},
		{
			name: "recover_returns_panic_value",
			source: `package main

func run() int {
	defer func() {
		if r := recover(); r != nil {
			_ = r
		}
	}()
	panic(42)
}

func main() {}
`,
			expect: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.EvalFile(context.Background(), tt.source, "run")
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestVMMapNamedIntReflectFallback(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     string
		entrypoint string
		expect     any
	}{
		{
			name: "named_int_map_get",
			source: `package main

type ID int
func run() int {
	m := map[ID]ID{ID(1): ID(42)}
	return int(m[ID(1)])
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(42),
		},
		{
			name: "named_int_map_set",
			source: `package main

type ID int
func run() int {
	m := map[ID]ID{}
	m[ID(1)] = ID(42)
	return int(m[ID(1)])
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(42),
		},
		{
			name: "named_int_map_missing_key",
			source: `package main

type ID int
func run() int {
	m := map[ID]ID{ID(1): ID(42)}
	return int(m[ID(99)])
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.EvalFile(context.Background(), tt.source, tt.entrypoint)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestVMTypeAssertionPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     string
		entrypoint string
		expect     any
	}{
		{
			name: "nil_interface_to_int_comma_ok",
			source: `package main

func run() int {
	var i interface{}
	v, ok := i.(int)
	if ok {
		return v
	}
	return -1
}

func main() {}
`,
			entrypoint: "run",
			expect:     int64(-1),
		},
		{
			name: "nil_interface_to_string_comma_ok",
			source: `package main

func run() string {
	var i interface{}
	v, ok := i.(string)
	if ok {
		return v
	}
	return "nil"
}

func main() {}
`,
			entrypoint: "run",
			expect:     "nil",
		},
		{
			name: "nil_interface_to_bool_comma_ok",
			source: `package main

func run() bool {
	var i interface{}
	_, ok := i.(bool)
	return ok
}

func main() {}
`,
			entrypoint: "run",
			expect:     false,
		},
		{
			name: "uint_from_interface",
			source: `package main

func run() uint64 {
	var i interface{} = uint64(42)
	return i.(uint64)
}

func main() {}
`,
			entrypoint: "run",
			expect:     uint64(42),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.EvalFile(context.Background(), tt.source, tt.entrypoint)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestVMRangeUintAndComplexSlice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     string
		entrypoint string
		expect     any
	}{
		{
			name: "range_uint_slice",
			source: `package main

func run() uint {
	var sum uint
	for _, v := range []uint{10, 20, 30} {
		sum += v
	}
	return sum
}

func main() {}
`,
			entrypoint: "run",
			expect:     uint64(60),
		},
		{
			name: "range_complex_slice",
			source: `package main

func run() complex128 {
	var sum complex128
	for _, v := range []complex128{1+2i, 3+4i} {
		sum += v
	}
	return sum
}

func main() {}
`,
			entrypoint: "run",
			expect:     complex128(4 + 6i),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.EvalFile(context.Background(), tt.source, tt.entrypoint)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestVMReturnToGeneralRegister(t *testing.T) {
	t.Parallel()

	source := `package main

func getInt() int { return 42 }

func run() interface{} {
	var result interface{} = getInt()
	return result
}

func main() {}
`
	service := NewService()
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, int64(42), result)
}

func TestVMIndexSetInterfaceSlice(t *testing.T) {
	t.Parallel()

	service := NewService()
	source := `package main

func run() string {
	s := make([]interface{}, 1)
	s[0] = "hello"
	return s[0].(string)
}

func main() {}
`
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, "hello", result)
}
