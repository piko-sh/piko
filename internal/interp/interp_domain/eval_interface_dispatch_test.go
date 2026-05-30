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
		{expect: int64(10), name: "interface_multi_return_first", code: `
type Parser interface { Parse(s string) (int, bool) }
type IntParser struct{}
func (p IntParser) Parse(s string) (int, bool) {
	if s == "ok" { return 10, true }
	return 0, false
}
var p Parser = IntParser{}
n, _ := p.Parse("ok")
n`},
		{expect: true, name: "interface_multi_return_second", code: `
type Parser interface { Parse(s string) (int, bool) }
type IntParser struct{}
func (p IntParser) Parse(s string) (int, bool) {
	if s == "ok" { return 10, true }
	return 0, false
}
var p Parser = IntParser{}
_, ok := p.Parse("ok")
ok`},
		{expect: false, name: "interface_multi_return_fail", code: `
type Parser interface { Parse(s string) (int, bool) }
type IntParser struct{}
func (p IntParser) Parse(s string) (int, bool) {
	if s == "ok" { return 10, true }
	return 0, false
}
var p Parser = IntParser{}
_, ok := p.Parse("nope")
ok`},
		{expect: "hello", name: "interface_multi_return_string", code: `
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
		{expect: int64(10), name: "interface_multi_return_first", code: `
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
		{expect: int(7), name: "generic_sub", code: `
func sub(a, b any) any {
	return a.(int) - b.(int)
}
sub(10, 3)`},
		{expect: int(5), name: "generic_div", code: `
func div(a, b any) any {
	return a.(int) / b.(int)
}
div(10, 2)`},
		{expect: int(1), name: "generic_rem", code: `
func rem(a, b any) any {
	return a.(int) % b.(int)
}
rem(10, 3)`},
		{expect: float64(2.5), name: "generic_float_sub", code: `
func sub(a, b any) any {
	return a.(float64) - b.(float64)
}
sub(5.5, 3.0)`},
		{expect: float64(2.5), name: "generic_float_div", code: `
func div(a, b any) any {
	return a.(float64) / b.(float64)
}
div(5.0, 2.0)`},
	})
}

func TestTailCallCrossBankArgs(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: 42, name: "tail_call_int_to_any", code: `
func accept(v any) any { return v }
func wrap(n int) any { return accept(n) }
wrap(42)`},
		{expect: "hello", name: "tail_call_string_to_any", code: `
func accept(v any) any { return v }
func wrap(s string) any { return accept(s) }
wrap("hello")`},
		{expect: true, name: "tail_call_bool_to_any", code: `
func accept(v any) any { return v }
func wrap(b bool) any { return accept(b) }
wrap(true)`},
		{expect: float64(3.14), name: "tail_call_float_to_any", code: `
func accept(v any) any { return v }
func wrap(f float64) any { return accept(f) }
wrap(3.14)`},
	})
}

func TestGoDispatchTailCallCrossBankArgs(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{expect: 42, name: "tail_call_int_to_any", code: `
func accept(v any) any { return v }
func wrap(n int) any { return accept(n) }
wrap(42)`},
		{expect: "hello", name: "tail_call_string_to_any", code: `
func accept(v any) any { return v }
func wrap(s string) any { return accept(s) }
wrap("hello")`},
	})
}

func TestCrossBankReturnTypedToGeneral(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: int64(42), name: "return_int_via_any", code: `
func getInt() int { return 42 }
func wrap() any { return getInt() }
x := wrap()
y := x.(int)
y`},
		{expect: "hi", name: "return_string_via_any", code: `
func getString() string { return "hi" }
func wrap() any { return getString() }
x := wrap()
y := x.(string)
y`},
		{expect: true, name: "return_bool_via_any", code: `
func getBool() bool { return true }
func wrap() any { return getBool() }
x := wrap()
y := x.(bool)
y`},
		{expect: float64(2.5), name: "return_float_via_any", code: `
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
		{expect: int64(42), name: "return_int_via_any", code: `
func getInt() int { return 42 }
func wrap() any { return getInt() }
x := wrap()
y := x.(int)
y`},
		{expect: "hi", name: "return_string_via_any", code: `
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
		{expect: int64(42), name: "pointer_deref", code: `x := 42; p := &x; *p`},
		{expect: int64(99), name: "pointer_set", code: `x := 42; p := &x; *p = 99; x`},
		{expect: int64(10), name: "pointer_to_struct_field", code: `
type S struct { V int }
s := S{V: 10}
p := &s.V
*p`},
		{expect: int64(20), name: "pointer_modify_struct", code: `
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
		{expect: int64(5), name: "safe_divide", code: `
func safeDivide(a, b int) int {
	if b == 0 { return -1 }
	return a / b
}
safeDivide(10, 2)`},
		{expect: int64(-1), name: "safe_divide_zero", code: `
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
		{expect: int64(3), name: "multi_return_first", code: `
func divmod(a, b int) (int, int) { return a / b, a % b }
q, _ := divmod(10, 3)
q`},
		{expect: int64(1), name: "multi_return_second", code: `
func divmod(a, b int) (int, int) { return a / b, a % b }
_, r := divmod(10, 3)
r`},
		{expect: int64(10), name: "multi_return_swap", code: `
func swap(a, b int) (int, int) { return b, a }
x, _ := swap(5, 10)
x`},
	})
}

func TestClosureUpvalues(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: int64(3), name: "closure_counter", code: `
func makeCounter() func() int {
	n := 0
	return func() int { n++; return n }
}
c := makeCounter()
c()
c()
c()`},
		{expect: int64(15), name: "closure_accumulator", code: `
func makeAccum() func(int) int {
	total := 0
	return func(n int) int { total += n; return total }
}
add := makeAccum()
add(5)
add(10)`},
		{expect: "hello world", name: "closure_capture_string", code: `
prefix := "hello"
f := func(s string) string { return prefix + " " + s }
f("world")`},
	})
}

func TestGoDispatchClosureUpvalues(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{expect: int64(3), name: "closure_counter", code: `
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
		{expect: int64(10), name: "if_block_scope", code: `
x := 5
if true {
	x = 10
}
x`},
		{expect: int64(5), name: "if_else_scope", code: `
x := 0
if false {
	x = 10
} else {
	x = 5
}
x`},
		{expect: int64(6), name: "for_accumulate", code: `
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
		{expect: int64(25), name: "value_receiver_method", code: `
type Rect struct { W, H int }
func (r Rect) Area() int { return r.W * r.H }
r := Rect{W: 5, H: 5}
r.Area()`},
		{expect: int64(6), name: "pointer_receiver_method", code: `
type Box struct { X int }
func (b *Box) Double() { b.X *= 2 }
b := &Box{X: 3}
b.Double()
b.X`},
		{expect: int64(15), name: "chained_method_calls", code: `
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
		{expect: int64(25), name: "value_receiver_method", code: `
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
import (
	"tp"
)
c := tp.New()
fn := c.Value
fn()`)
	require.NoError(t, err)
	require.Equal(t, int64(0), result)
}

func TestNativeCallWithClosure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{name: "apply_double", code: "import \"cb\"\ncb.Apply(func(x int) int { return x * 2 }, 5)", expect: int64(10)},
		{name: "apply_add_one", code: "import \"cb\"\ncb.Apply(func(x int) int { return x + 1 }, 41)", expect: int64(42)},
		{name: "map_double", code: "import \"cb\"\ns := cb.Map([]int{1, 2, 3}, func(x int) int { return x * 2 })\ns[2]", expect: int64(6)},
		{name: "filter_even", code: "import \"cb\"\ns := cb.Filter([]int{1, 2, 3, 4}, func(x int) bool { return x % 2 == 0 })\nlen(s)", expect: int64(2)},
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
import (
	"tp"
)
c := tp.New()
fn := c.Value
fn()`)
	require.NoError(t, err)
	require.Equal(t, int64(10), result)
}
