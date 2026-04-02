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

func TestInterfaceMethodMultiReturn(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{int64(10), "interface_multi_return_first", `
type Parser interface { Parse(s string) (int, bool) }
type IntParser struct{}
func (p IntParser) Parse(s string) (int, bool) {
	if s == "ok" { return 10, true }
	return 0, false
}
var p Parser = IntParser{}
n, _ := p.Parse("ok")
n`},
		{true, "interface_multi_return_second", `
type Parser interface { Parse(s string) (int, bool) }
type IntParser struct{}
func (p IntParser) Parse(s string) (int, bool) {
	if s == "ok" { return 10, true }
	return 0, false
}
var p Parser = IntParser{}
_, ok := p.Parse("ok")
ok`},
		{false, "interface_multi_return_fail", `
type Parser interface { Parse(s string) (int, bool) }
type IntParser struct{}
func (p IntParser) Parse(s string) (int, bool) {
	if s == "ok" { return 10, true }
	return 0, false
}
var p Parser = IntParser{}
_, ok := p.Parse("nope")
ok`},
		{"hello", "interface_multi_return_string", `
type Converter interface { Convert(n int) (string, bool) }
type Mapper struct{ M map[int]string }
func (m Mapper) Convert(n int) (string, bool) {
	v, ok := m.M[n]
	return v, ok
}
var c Converter = Mapper{M: map[int]string{1: "hello"}}
s, _ := c.Convert(1)
s`},
	})
}

func TestGoDispatchInterfaceMethodMultiReturn(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{int64(10), "interface_multi_return_first", `
type Parser interface { Parse(s string) (int, bool) }
type IntParser struct{}
func (p IntParser) Parse(s string) (int, bool) {
	if s == "ok" { return 10, true }
	return 0, false
}
var p Parser = IntParser{}
n, _ := p.Parse("ok")
n`},
	})
}

func TestGoDispatchGenericArithmetic(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{int(7), "generic_sub", `
func sub(a, b any) any {
	return a.(int) - b.(int)
}
sub(10, 3)`},
		{int(5), "generic_div", `
func div(a, b any) any {
	return a.(int) / b.(int)
}
div(10, 2)`},
		{int(1), "generic_rem", `
func rem(a, b any) any {
	return a.(int) % b.(int)
}
rem(10, 3)`},
		{float64(2.5), "generic_float_sub", `
func sub(a, b any) any {
	return a.(float64) - b.(float64)
}
sub(5.5, 3.0)`},
		{float64(2.5), "generic_float_div", `
func div(a, b any) any {
	return a.(float64) / b.(float64)
}
div(5.0, 2.0)`},
	})
}

func TestTailCallCrossBankArgs(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{int64(42), "tail_call_int_to_any", `
func accept(v any) any { return v }
func wrap(n int) any { return accept(n) }
wrap(42)`},
		{"hello", "tail_call_string_to_any", `
func accept(v any) any { return v }
func wrap(s string) any { return accept(s) }
wrap("hello")`},
		{true, "tail_call_bool_to_any", `
func accept(v any) any { return v }
func wrap(b bool) any { return accept(b) }
wrap(true)`},
		{float64(3.14), "tail_call_float_to_any", `
func accept(v any) any { return v }
func wrap(f float64) any { return accept(f) }
wrap(3.14)`},
	})
}

func TestGoDispatchTailCallCrossBankArgs(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{int64(42), "tail_call_int_to_any", `
func accept(v any) any { return v }
func wrap(n int) any { return accept(n) }
wrap(42)`},
		{"hello", "tail_call_string_to_any", `
func accept(v any) any { return v }
func wrap(s string) any { return accept(s) }
wrap("hello")`},
	})
}

func TestCrossBankReturnTypedToGeneral(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{int64(42), "return_int_via_any", `
func getInt() int { return 42 }
func wrap() any { return getInt() }
x := wrap()
y := x.(int)
y`},
		{"hi", "return_string_via_any", `
func getString() string { return "hi" }
func wrap() any { return getString() }
x := wrap()
y := x.(string)
y`},
		{true, "return_bool_via_any", `
func getBool() bool { return true }
func wrap() any { return getBool() }
x := wrap()
y := x.(bool)
y`},
		{float64(2.5), "return_float_via_any", `
func getFloat() float64 { return 2.5 }
func wrap() any { return getFloat() }
x := wrap()
y := x.(float64)
y`},
	})
}

func TestGoDispatchCrossBankReturnTypedToGeneral(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{int64(42), "return_int_via_any", `
func getInt() int { return 42 }
func wrap() any { return getInt() }
x := wrap()
y := x.(int)
y`},
		{"hi", "return_string_via_any", `
func getString() string { return "hi" }
func wrap() any { return getString() }
x := wrap()
y := x.(string)
y`},
	})
}

func TestPointerOperations(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{int64(42), "pointer_deref", `x := 42; p := &x; *p`},
		{int64(99), "pointer_set", `x := 42; p := &x; *p = 99; x`},
		{int64(10), "pointer_to_struct_field", `
type S struct { V int }
s := S{V: 10}
p := &s.V
*p`},
		{int64(20), "pointer_modify_struct", `
type S struct { V int }
s := S{V: 10}
p := &s
p.V = 20
s.V`},
	})
}

func TestDivByZero(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{int64(5), "safe_divide", `
func safeDivide(a, b int) int {
	if b == 0 { return -1 }
	return a / b
}
safeDivide(10, 2)`},
		{int64(-1), "safe_divide_zero", `
func safeDivide(a, b int) int {
	if b == 0 { return -1 }
	return a / b
}
safeDivide(10, 0)`},
	})
}

func TestMultiReturnAssignment(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{int64(3), "multi_return_first", `
func divmod(a, b int) (int, int) { return a / b, a % b }
q, _ := divmod(10, 3)
q`},
		{int64(1), "multi_return_second", `
func divmod(a, b int) (int, int) { return a / b, a % b }
_, r := divmod(10, 3)
r`},
		{int64(10), "multi_return_swap", `
func swap(a, b int) (int, int) { return b, a }
x, _ := swap(5, 10)
x`},
	})
}

func TestClosureUpvalues(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{int64(3), "closure_counter", `
func makeCounter() func() int {
	n := 0
	return func() int { n++; return n }
}
c := makeCounter()
c()
c()
c()`},
		{int64(15), "closure_accumulator", `
func makeAccum() func(int) int {
	total := 0
	return func(n int) int { total += n; return total }
}
add := makeAccum()
add(5)
add(10)`},
		{"hello world", "closure_capture_string", `
prefix := "hello"
f := func(s string) string { return prefix + " " + s }
f("world")`},
	})
}

func TestGoDispatchClosureUpvalues(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{int64(3), "closure_counter", `
func makeCounter() func() int {
	n := 0
	return func() int { n++; return n }
}
c := makeCounter()
c()
c()
c()`},
	})
}

func TestScopedBlocks(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{int64(10), "if_block_scope", `
x := 5
if true {
	x = 10
}
x`},
		{int64(5), "if_else_scope", `
x := 0
if false {
	x = 10
} else {
	x = 5
}
x`},
		{int64(6), "for_accumulate", `
sum := 0
for i := 1; i <= 3; i++ {
	sum += i
}
sum`},
	})
}

func TestStructMethodDispatch(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{int64(25), "value_receiver_method", `
type Rect struct { W, H int }
func (r Rect) Area() int { return r.W * r.H }
r := Rect{W: 5, H: 5}
r.Area()`},
		{int64(6), "pointer_receiver_method", `
type Box struct { X int }
func (b *Box) Double() { b.X *= 2 }
b := &Box{X: 3}
b.Double()
b.X`},
		{int64(15), "chained_method_calls", `
type Calc struct { V int }
func (c Calc) Add(n int) Calc { return Calc{V: c.V + n} }
func (c Calc) Result() int { return c.V }
c := Calc{V: 0}
c.Add(5).Add(10).Result()`},
	})
}

func TestGoDispatchStructMethodDispatch(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{int64(25), "value_receiver_method", `
type Rect struct { W, H int }
func (r Rect) Area() int { return r.W * r.H }
r := Rect{W: 5, H: 5}
r.Area()`},
	})
}

type testNativeCounter struct {
	N int
}

func (c *testNativeCounter) Increment() {
	c.N++
}

func (c testNativeCounter) Value() int {
	return c.N
}

func TestNativeMethodValue(t *testing.T) {
	t.Parallel()

	service := newTestServiceWithSymbols(t, SymbolExports{
		"tp": {
			"Counter": reflect.ValueOf((*testNativeCounter)(nil)),
			"New": reflect.ValueOf(func() *testNativeCounter {
				return &testNativeCounter{N: 0}
			}),
		},
	})

	result, err := service.Eval(context.Background(), `
import "tp"
c := tp.New()
fn := c.Value
fn()`)
	require.NoError(t, err)
	require.Equal(t, int64(0), result)
}

func TestNativeCallWithClosure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect any
	}{
		{"apply_double", "import \"cb\"\ncb.Apply(func(x int) int { return x * 2 }, 5)", int64(10)},
		{"apply_add_one", "import \"cb\"\ncb.Apply(func(x int) int { return x + 1 }, 41)", int64(42)},
		{"map_double", "import \"cb\"\ns := cb.Map([]int{1, 2, 3}, func(x int) int { return x * 2 })\ns[2]", int64(6)},
		{"filter_even", "import \"cb\"\ns := cb.Filter([]int{1, 2, 3, 4}, func(x int) bool { return x % 2 == 0 })\nlen(s)", int64(2)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := newTestServiceWithSymbols(t, SymbolExports{
				"cb": {
					"Apply": reflect.ValueOf(func(fn func(int) int, x int) int {
						return fn(x)
					}),
					"Map": reflect.ValueOf(func(s []int, fn func(int) int) []int {
						result := make([]int, len(s))
						for i, v := range s {
							result[i] = fn(v)
						}
						return result
					}),
					"Filter": reflect.ValueOf(func(s []int, fn func(int) bool) []int {
						var result []int
						for _, v := range s {
							if fn(v) {
								result = append(result, v)
							}
						}
						return result
					}),
				},
			})
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestNativeMethodValuePointerReceiver(t *testing.T) {
	t.Parallel()

	service := newTestServiceWithSymbols(t, SymbolExports{
		"tp": {
			"Counter": reflect.ValueOf((*testNativeCounter)(nil)),
			"New": reflect.ValueOf(func() *testNativeCounter {
				return &testNativeCounter{N: 10}
			}),
		},
	})

	result, err := service.Eval(context.Background(), `
import "tp"
c := tp.New()
fn := c.Value
fn()`)
	require.NoError(t, err)
	require.Equal(t, int64(10), result)
}
