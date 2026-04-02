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

func TestUintIncDec(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{uint64(6), "uint_inc", `var a uint = 5; a++; a`},
		{uint64(4), "uint_dec", `var a uint = 5; a--; a`},
	})
}

func TestGoDispatchUintIncDec(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{uint64(6), "uint_inc", `var a uint = 5; a++; a`},
		{uint64(4), "uint_dec", `var a uint = 5; a--; a`},
	})
}

func TestUintBitNot(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{uint64(0xFFFFFFFFFFFFFF00), "uint_bitnot", `var a uint = 0xFF; ^a`},
	})
}

func TestGoDispatchUintBitNot(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{uint64(0xFFFFFFFFFFFFFF00), "uint_bitnot", `var a uint = 0xFF; ^a`},
	})
}

func TestMapCommaOkAssign(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(1), "comma_ok_assign_found", `
m := map[string]int{"a": 1}
v := 0
ok := false
v, ok = m["a"]
_ = ok
v`},
		{true, "comma_ok_assign_ok", `
m := map[string]int{"a": 1}
v := 0
ok := false
v, ok = m["a"]
_ = v
ok`},
		{int64(0), "comma_ok_assign_missing", `
m := map[string]int{"a": 1}
v := 0
ok := false
v, ok = m["b"]
_ = ok
v`},
		{false, "comma_ok_assign_missing_ok", `
m := map[string]int{"a": 1}
v := 0
ok := false
v, ok = m["b"]
_ = v
ok`},
	}

	runEvalTable(t, nil, tests)
}

func TestRuneStringConcat(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{"Hello", "rune_concat_prefix", `r := 'H'; string(r) + "ello"`},
		{"aX", "rune_concat_suffix", `s := "a"; r := 'X'; s + string(r)`},
	}

	runEvalTable(t, nil, tests)
}

func TestGoDispatchRuneStringConcat(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{"Hello", "rune_concat_prefix", `r := 'H'; string(r) + "ello"`},
		{"aX", "rune_concat_suffix", `s := "a"; r := 'X'; s + string(r)`},
	}

	runEvalTable(t, []Option{WithForceGoDispatch()}, tests)
}

func TestSliceUintSetGet(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{uint64(10), "uint_slice_set", `s := []uint{1, 2, 3}; s[0] = 10; s[0]`},
		{uint64(3), "uint_slice_get", `s := []uint{1, 2, 3}; s[2]`},
	}

	runEvalTable(t, nil, tests)
}

func TestGoDispatchSliceUint(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{uint64(10), "uint_slice_set", `s := []uint{1, 2, 3}; s[0] = 10; s[0]`},
		{uint64(3), "uint_slice_get", `s := []uint{1, 2, 3}; s[2]`},
	}

	runEvalTable(t, []Option{WithForceGoDispatch()}, tests)
}

func TestCompoundAssignMapExtended(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(15), "compound_map_int_key", `m := map[int]int{1: 10}; m[1] += 5; m[1]`},
		{"helloworld", "compound_map_string_val", `m := map[string]string{"a": "hello"}; m["a"] += "world"; m["a"]`},
	}

	runEvalTable(t, nil, tests)
}

func TestVariousZeroValues(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(0), "zero_int", `var x int; x`},
		{float64(0), "zero_float", `var x float64; x`},
		{"", "zero_string", `var x string; x`},
		{false, "zero_bool", `var x bool; x`},
		{uint64(0), "zero_uint", `var x uint; x`},
	}

	runEvalTable(t, nil, tests)
}

func TestGoDispatchVariousZeroValues(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(0), "zero_int", `var x int; x`},
		{float64(0), "zero_float", `var x float64; x`},
		{"", "zero_string", `var x string; x`},
		{false, "zero_bool", `var x bool; x`},
		{uint64(0), "zero_uint", `var x uint; x`},
	}

	runEvalTable(t, []Option{WithForceGoDispatch()}, tests)
}

func TestMultiCaseSwitch(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(10), "multi_case_1", `x := 1; switch x { case 1, 2, 3: x = 10; case 4, 5: x = 20 }; x`},
		{int64(10), "multi_case_2", `x := 2; switch x { case 1, 2, 3: x = 10; case 4, 5: x = 20 }; x`},
		{int64(20), "multi_case_4", `x := 4; switch x { case 1, 2, 3: x = 10; case 4, 5: x = 20 }; x`},
		{int64(99), "multi_case_default", `x := 99; switch x { case 1, 2, 3: x = 10; default: x = 99 }; x`},
	}

	runEvalTable(t, nil, tests)
}

func TestGoDispatchMultiCaseSwitch(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{int64(10), "multi_case_1", `x := 1; switch x { case 1, 2, 3: x = 10; case 4, 5: x = 20 }; x`},
		{int64(10), "multi_case_2", `x := 2; switch x { case 1, 2, 3: x = 10; case 4, 5: x = 20 }; x`},
		{int64(20), "multi_case_4", `x := 4; switch x { case 1, 2, 3: x = 10; case 4, 5: x = 20 }; x`},
	})
}

func TestIncDecSelector(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(6), "inc_selector", `
type S struct { X int }
s := S{X: 5}
s.X++
s.X`},
		{int64(4), "dec_selector", `
type S struct { X int }
s := S{X: 5}
s.X--
s.X`},
	}

	runEvalTable(t, nil, tests)
}

func TestGoDispatchIncDecSelector(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{int64(6), "inc_selector", `
type S struct { X int }
s := S{X: 5}
s.X++
s.X`},
		{int64(4), "dec_selector", `
type S struct { X int }
s := S{X: 5}
s.X--
s.X`},
	})
}

func TestCompoundAssignSliceExtended(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(15), "slice_add_assign", `s := []int{10, 20}; s[0] += 5; s[0]`},
		{int64(5), "slice_sub_assign", `s := []int{10, 20}; s[0] -= 5; s[0]`},
		{int64(50), "slice_mul_assign", `s := []int{10, 20}; s[0] *= 5; s[0]`},
	}

	runEvalTable(t, nil, tests)
}

func TestCompoundAssignSelectorExtended(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(15), "selector_add_assign", `
type S struct { X int }
s := S{X: 10}
s.X += 5
s.X`},
		{int64(5), "selector_sub_assign", `
type S struct { X int }
s := S{X: 10}
s.X -= 5
s.X`},
		{int64(50), "selector_mul_assign", `
type S struct { X int }
s := S{X: 10}
s.X *= 5
s.X`},
	}

	runEvalTable(t, nil, tests)
}

func TestForLoopRangeString(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(5), "range_string_count", `
count := 0
for range "hello" { count++ }
count`},
		{int64(532), "range_string_rune_sum", `
sum := 0
for _, r := range "hello" { sum += int(r) }
sum`},
	}

	runEvalTable(t, nil, tests)
}

func TestGoDispatchForLoopRangeString(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{int64(5), "range_string_count", `
count := 0
for range "hello" { count++ }
count`},
	})
}

func TestAddressOfSelector(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(42), "addr_of_field", `
type S struct { X int }
s := S{X: 42}
p := &s.X
*p`},
		{int64(99), "write_through_addr", `
type S struct { X int }
s := S{X: 42}
p := &s.X
*p = 99
s.X`},
	}

	runEvalTable(t, nil, tests)
}

func TestMultiReturnFunctions(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(3), "multi_return_first", `
func f() (int, string) { return 3, "hello" }
x, _ := f()
x`},
		{"hello", "multi_return_second", `
func f() (int, string) { return 3, "hello" }
_, s := f()
s`},
		{int64(10), "multi_return_sum", `
func f() (int, int) { return 3, 7 }
a, b := f()
a + b`},
	}

	runEvalTable(t, nil, tests)
}

func TestGoDispatchMultiReturn(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{int64(3), "multi_return_first", `
func f() (int, string) { return 3, "hello" }
x, _ := f()
x`},
		{"hello", "multi_return_second", `
func f() (int, string) { return 3, "hello" }
_, s := f()
s`},
	})
}

func TestShortVarRedeclare(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(5), "redecl_first", `
x := 1
x, y := 5, 10
_ = y
x`},
		{int64(10), "redecl_second", `
x := 1
x, y := 5, 10
_ = x
y`},
	}

	runEvalTable(t, nil, tests)
}

func TestForLoopWithBreakContinue(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(5), "for_break", `
sum := 0
for i := 0; i < 100; i++ {
	if i >= 5 { break }
	sum++
}
sum`},
		{int64(50), "for_continue", `
sum := 0
for i := 0; i < 100; i++ {
	if i%2 != 0 { continue }
	sum++
}
sum`},
		{int64(3), "for_range_break", `
count := 0
for _, v := range []int{1, 2, 3, 4, 5} {
	if v > 3 { break }
	count++
}
count`},
	}

	runEvalTable(t, nil, tests)
}

func TestGoDispatchForLoopBreakContinue(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{int64(5), "for_break", `
sum := 0
for i := 0; i < 100; i++ {
	if i >= 5 { break }
	sum++
}
sum`},
		{int64(50), "for_continue", `
sum := 0
for i := 0; i < 100; i++ {
	if i%2 != 0 { continue }
	sum++
}
sum`},
	})
}

func TestClosureCaptureAndCall(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(15), "closure_capture", `
func make() func() int {
	x := 15
	return func() int { return x }
}
f := make()
f()`},
		{int64(10), "closure_counter", `
func counter() func() int {
	n := 0
	return func() int { n++; return n }
}
c := counter()
sum := 0
for i := 0; i < 4; i++ { sum += c() }
sum`},
		{int64(120), "recursive_closure", `
func factorial(n int) int {
	if n <= 1 { return 1 }
	return n * factorial(n-1)
}
factorial(5)`},
	}

	runEvalTable(t, nil, tests)
}

func TestGoDispatchClosuresExtended(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{int64(15), "closure_capture", `
func make() func() int {
	x := 15
	return func() int { return x }
}
f := make()
f()`},
		{int64(120), "recursive_closure", `
func factorial(n int) int {
	if n <= 1 { return 1 }
	return n * factorial(n-1)
}
factorial(5)`},
	})
}

func TestInterfaceTypeSwitch(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(1), "type_switch_int", `
var x any = 42
result := 0
switch x.(type) {
case int: result = 1
case string: result = 2
}
result`},
		{int64(2), "type_switch_string", `
var x any = "hello"
result := 0
switch x.(type) {
case int: result = 1
case string: result = 2
}
result`},
		{int64(3), "type_switch_default", `
var x any = 3.14
result := 0
switch x.(type) {
case int: result = 1
case string: result = 2
default: result = 3
}
result`},
	}

	runEvalTable(t, nil, tests)
}

func TestGoDispatchTypeSwitch(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{int64(1), "type_switch_int", `
var x any = 42
result := 0
switch x.(type) {
case int: result = 1
case string: result = 2
}
result`},
		{int64(3), "type_switch_default", `
var x any = 3.14
result := 0
switch x.(type) {
case int: result = 1
default: result = 3
}
result`},
	})
}

func TestDeferWithPanic(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(42), "defer_recover", `
func f() int {
	defer func() { recover() }()
	panic("test")
	return 0
}
func g() int {
	defer func() {}()
	return 42
}
g()`},
		{"caught", "recover_message", `
func f() string {
	defer func() {
		if r := recover(); r != nil {
			_ = r
		}
	}()
	panic("caught")
	return ""
}
f()
"caught"`},
	}

	runEvalTable(t, nil, tests)
}

func TestChannelOperations(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(42), "chan_send_recv", `
ch := make(chan int, 1)
ch <- 42
<-ch`},
		{int64(10), "chan_buffered", `
ch := make(chan int, 3)
ch <- 10
ch <- 20
ch <- 30
<-ch`},
	}

	runEvalTable(t, nil, tests)
}

func TestGoDispatchChannelOps(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{int64(42), "chan_send_recv", `
ch := make(chan int, 1)
ch <- 42
<-ch`},
	})
}

func TestGoDispatchComplexMoveAndLoad(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{complex128(1 + 2i), "complex_move", `a := 1+2i; b := a; b`},
		{complex128(5 + 0i), "complex_load_const", `a := 5+0i; a`},
	})
}

func TestNativeFunctionCaching(t *testing.T) {
	t.Parallel()

	service := newTestServiceWithSymbols(t, SymbolExports{
		"fp": {
			"StringInt":       reflect.ValueOf(fpStringInt),
			"Float64Float64":  reflect.ValueOf(fpFloat64Float64),
			"IntBool":         reflect.ValueOf(fpIntBool),
			"StringBool":      reflect.ValueOf(fpStringBool),
			"Float642Float64": reflect.ValueOf(fpFloat642Float64),
		},
	})

	tests := []struct {
		name   string
		code   string
		expect any
	}{

		{"string_int_cached", `import "fp"
a := fp.StringInt("hello")
b := fp.StringInt("world")
a + b`, int64(10)},
		{"float_cached", `import "fp"
a := fp.Float64Float64(2.0)
b := fp.Float64Float64(3.0)
a + b`, float64(10.0)},
		{"bool_cached", `import "fp"
a := fp.IntBool(5)
b := fp.IntBool(-1)
a && !b`, true},
		{"string_bool_cached", `import "fp"
a := fp.StringBool("hi")
b := fp.StringBool("")
a && !b`, true},
		{"float2_cached", `import "fp"
a := fp.Float642Float64(1.0, 2.0)
b := fp.Float642Float64(3.0, 4.0)
a + b`, float64(10.0)},

		{"loop_cached", `import "fp"
sum := 0
for i := 0; i < 10; i++ { sum += fp.StringInt("ab") }
sum`, int64(20)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			localService := service.Clone()
			result, err := localService.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestNarrowIntTypes(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(127), "int8_max", `var x int8 = 127; int(x)`},
		{int64(32767), "int16_max", `var x int16 = 32767; int(x)`},
		{int64(42), "int32_val", `var x int32 = 42; int(x)`},
		{int64(100), "int64_val", `var x int64 = 100; int(x)`},
		{uint64(255), "uint8_val", `var x uint8 = 255; uint(x)`},
		{uint64(65535), "uint16_val", `var x uint16 = 65535; uint(x)`},
		{uint64(42), "uint32_val", `var x uint32 = 42; uint(x)`},
		{uint64(100), "uint64_val", `var x uint64 = 100; uint(x)`},
		{int64(3), "byte_slice_len", `s := []byte{1, 2, 3}; len(s)`},
		{int64(65), "byte_to_int", `var b byte = 'A'; int(b)`},
	}

	runEvalTable(t, nil, tests)
}

func TestGoDispatchNarrowInt(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{int64(127), "int8_max", `var x int8 = 127; int(x)`},
		{uint64(255), "uint8_val", `var x uint8 = 255; uint(x)`},
		{int64(42), "int32_val", `var x int32 = 42; int(x)`},
	})
}

func TestMapOperationsExtended(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(3), "map_len", `m := map[string]int{"a": 1, "b": 2, "c": 3}; len(m)`},
		{int64(0), "map_empty_len", `m := map[string]int{}; len(m)`},
		{int64(2), "map_after_delete", `m := map[string]int{"a": 1, "b": 2, "c": 3}; delete(m, "b"); len(m)`},
		{true, "map_check_after_set", `m := map[string]bool{}; m["key"] = true; m["key"]`},
		{int64(6), "map_range_sum", `
m := map[string]int{"a": 1, "b": 2, "c": 3}
sum := 0
for _, v := range m { sum += v }
sum`},
	}

	runEvalTable(t, nil, tests)
}

func TestGoDispatchMapOps(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{int64(3), "map_len", `m := map[string]int{"a": 1, "b": 2, "c": 3}; len(m)`},
		{int64(2), "map_after_delete", `m := map[string]int{"a": 1, "b": 2, "c": 3}; delete(m, "b"); len(m)`},
	})
}

func TestStructMethodCalls(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(42), "value_method", `
type S struct { X int }
func (s S) Get() int { return s.X }
s := S{X: 42}
s.Get()`},
		{int64(99), "pointer_method", `
type S struct { X int }
func (s *S) Set(v int) { s.X = v }
func (s S) Get() int { return s.X }
s := S{X: 0}
s.Set(99)
s.Get()`},
		{int64(15), "method_chain", `
type Acc struct { Total int }
func (a *Acc) Add(n int) *Acc { a.Total += n; return a }
a := &Acc{}
a.Add(5).Add(10)
a.Total`},
	}

	runEvalTable(t, nil, tests)
}

func TestGoDispatchStructMethods(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{int64(42), "value_method", `
type S struct { X int }
func (s S) Get() int { return s.X }
s := S{X: 42}
s.Get()`},
		{int64(99), "pointer_method", `
type S struct { X int }
func (s *S) Set(v int) { s.X = v }
func (s S) Get() int { return s.X }
s := S{X: 0}
s.Set(99)
s.Get()`},
	})
}

func TestNestedStructs(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(42), "nested_field", `
type Inner struct { X int }
type Outer struct { I Inner }
o := Outer{I: Inner{X: 42}}
o.I.X`},
		{int64(99), "nested_set", `
type Inner struct { X int }
type Outer struct { I Inner }
o := Outer{I: Inner{X: 0}}
o.I.X = 99
o.I.X`},
	}

	runEvalTable(t, nil, tests)
}

func TestGoDispatchNestedStructs(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{int64(42), "nested_field", `
type Inner struct { X int }
type Outer struct { I Inner }
o := Outer{I: Inner{X: 42}}
o.I.X`},
	})
}

func TestEmbeddedStructs(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(42), "embedded_field", `
type Base struct { X int }
type Derived struct { Base }
d := Derived{Base: Base{X: 42}}
d.X`},
		{int64(99), "embedded_method", `
type Base struct { X int }
func (b Base) Value() int { return b.X }
type Derived struct { Base }
d := Derived{Base: Base{X: 99}}
d.Value()`},
	}

	runEvalTable(t, nil, tests)
}

func TestGoDispatchEmbeddedStructs(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{int64(42), "embedded_field", `
type Base struct { X int }
type Derived struct { Base }
d := Derived{Base: Base{X: 42}}
d.X`},
		{int64(99), "embedded_method", `
type Base struct { X int }
func (b Base) Value() int { return b.X }
type Derived struct { Base }
d := Derived{Base: Base{X: 99}}
d.Value()`},
	})
}

func TestCompoundAssignSliceFloat(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{float64(15.5), "float_slice_add_assign", `s := []float64{10.5}; s[0] += 5.0; s[0]`},
	})
}

func TestGoDispatchCompoundAssignSliceFloat(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{float64(15.5), "float_slice_add_assign", `s := []float64{10.5}; s[0] += 5.0; s[0]`},
	})
}

func TestCompoundAssignSelectorFloat(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{float64(15.0), "float_selector_add", `
type S struct { X float64 }
s := S{X: 10.0}
s.X += 5.0
s.X`},
	})
}

func TestCompoundAssignSelectorString(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{"hello world", "string_selector_add", `
type S struct { X string }
s := S{X: "hello"}
s.X += " world"
s.X`},
	})
}

func TestZeroValueComposite(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{int64(0), "zero_array", `var a [3]int; a[0]`},
		{int64(0), "zero_struct", `
type S struct { X int; Y string }
var s S
s.X`},
		{"", "zero_struct_string", `
type S struct { X int; Y string }
var s S
s.Y`},
	})
}

func TestGoDispatchZeroValueComposite(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{int64(0), "zero_array", `var a [3]int; a[0]`},
		{int64(0), "zero_struct", `
type S struct { X int; Y string }
var s S
s.X`},
	})
}

func TestForLoopVariousPatterns(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{

		{int64(10), "for_ge_limit", `
sum := 0
for i := 10; i >= 1; i-- { sum++ }
sum`},

		{int64(5), "for_step_3", `
count := 0
for i := 0; i < 15; i += 3 { count++ }
count`},

		{int64(10), "infinite_break", `
i := 0
for { if i >= 10 { break }; i++ }
i`},
	})
}

func TestGoDispatchForLoopVariousPatterns(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{int64(10), "for_ge_limit", `
sum := 0
for i := 10; i >= 1; i-- { sum++ }
sum`},
		{int64(5), "for_step_3", `
count := 0
for i := 0; i < 15; i += 3 { count++ }
count`},
		{int64(10), "infinite_break", `
i := 0
for { if i >= 10 { break }; i++ }
i`},
	})
}

func TestInterfaceConversions(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{int(42), "type_assert_int", `var x any = 42; x.(int)`},
		{"hello", "type_assert_string", `var x any = "hello"; x.(string)`},
		{float64(3.14), "type_assert_float", `var x any = 3.14; x.(float64)`},
		{true, "type_assert_bool", `var x any = true; x.(bool)`},
	})
}

func TestGoDispatchInterfaceConversions(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{int(42), "type_assert_int", `var x any = 42; x.(int)`},
		{"hello", "type_assert_string", `var x any = "hello"; x.(string)`},
	})
}

func TestSliceOfSlice(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{int64(42), "nested_slice", `
s := [][]int{{1, 2}, {3, 4}, {42}}
s[2][0]`},
		{int64(6), "slice_slice", `s := []int{1, 2, 3, 4, 5}; t := s[1:3]; len(s) - len(t) + t[1]`},
	}

	runEvalTable(t, nil, tests)
}

func TestNilTestOps(t *testing.T) {
	t.Parallel()

	tests := []evalTestCase{
		{true, "nil_test_nil", `var p *int; p == nil`},
		{false, "nil_test_nonnull", `x := 42; p := &x; p == nil`},
		{true, "nil_slice", `var s []int; s == nil`},
		{false, "nil_map_false", `m := map[string]int{}; m == nil`},
	}

	runEvalTable(t, nil, tests)
}

func TestClosureCaptureTypes(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{float64(3.14), "capture_float", `
func make() func() float64 {
	x := 3.14
	return func() float64 { return x }
}
f := make()
f()`},
		{"hello", "capture_string", `
func make() func() string {
	x := "hello"
	return func() string { return x }
}
f := make()
f()`},
		{true, "capture_bool", `
func make() func() bool {
	x := true
	return func() bool { return x }
}
f := make()
f()`},
		{int64(10), "capture_mutate", `
func make() func() int {
	x := 0
	return func() int { x += 10; return x }
}
f := make()
f()`},
		{float64(5.0), "capture_mutate_float", `
func make() func() float64 {
	x := 0.0
	return func() float64 { x += 5.0; return x }
}
f := make()
f()`},
		{"ab", "capture_mutate_string", `
func make() func() string {
	x := "a"
	return func() string { x += "b"; return x }
}
f := make()
f()`},
	})
}

func TestGoDispatchClosureCaptureTypes(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{float64(3.14), "capture_float", `
func make() func() float64 {
	x := 3.14
	return func() float64 { return x }
}
f := make()
f()`},
		{"hello", "capture_string", `
func make() func() string {
	x := "hello"
	return func() string { return x }
}
f := make()
f()`},
		{true, "capture_bool", `
func make() func() bool {
	x := true
	return func() bool { return x }
}
f := make()
f()`},
	})
}

func TestMethodValuesExtended(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{int64(42), "method_value", `
type S struct { X int }
func (s S) Get() int { return s.X }
s := S{X: 42}
f := s.Get
f()`},
		{int64(99), "method_value_ptr", `
type S struct { X int }
func (s *S) Get() int { return s.X }
s := &S{X: 99}
f := s.Get
f()`},
	})
}

func TestGoDispatchMethodValues(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{int64(42), "method_value", `
type S struct { X int }
func (s S) Get() int { return s.X }
s := S{X: 42}
f := s.Get
f()`},
	})
}

func TestRangeWithDifferentTypes(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{float64(6.0), "range_float_slice", `
sum := 0.0
for _, v := range []float64{1.0, 2.0, 3.0} { sum += v }
sum`},
		{"abc", "range_string_slice", `
result := ""
for _, s := range []string{"a", "b", "c"} { result += s }
result`},
		{int64(2), "range_bool_count", `
count := 0
for _, b := range []bool{true, false, true} { if b { count++ } }
count`},
	})
}

func TestGoDispatchRangeTypes(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{float64(6.0), "range_float_slice", `
sum := 0.0
for _, v := range []float64{1.0, 2.0, 3.0} { sum += v }
sum`},
		{"abc", "range_string_slice", `
result := ""
for _, s := range []string{"a", "b", "c"} { result += s }
result`},
	})
}

func TestCrossBankAssign(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{

		{float64(5), "int_to_float_assign", `
func f() float64 { x := 5; return float64(x) }
f()`},

		{int64(3), "float_to_int_assign", `
func f() int { x := 3.9; return int(x) }
f()`},

		{int64(1), "bool_to_int", `
func f() int { b := true; if b { return 1 }; return 0 }
f()`},
	})
}

func TestGoDispatchCrossBankAssign(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{float64(5), "int_to_float_assign", `
func f() float64 { x := 5; return float64(x) }
f()`},
		{int64(3), "float_to_int_assign", `
func f() int { x := 3.9; return int(x) }
f()`},
	})
}

func TestMultiReturnWithDifferentTypes(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{float64(3.14), "multi_ret_float_bool", `
func f() (float64, bool) { return 3.14, true }
v, _ := f()
v`},
		{true, "multi_ret_bool", `
func f() (float64, bool) { return 3.14, true }
_, b := f()
b`},
		{"hello", "multi_ret_string_int", `
func f() (string, int) { return "hello", 42 }
s, _ := f()
s`},
		{int64(42), "multi_ret_int_from_string_int", `
func f() (string, int) { return "hello", 42 }
_, n := f()
n`},
	})
}

func TestGoDispatchMultiReturnTypes(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{float64(3.14), "multi_ret_float_bool", `
func f() (float64, bool) { return 3.14, true }
v, _ := f()
v`},
		{"hello", "multi_ret_string_int", `
func f() (string, int) { return "hello", 42 }
s, _ := f()
s`},
	})
}

func TestDeferOrder(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{"cba", "defer_lifo_order", `
func f() (result string) {
	defer func() { result += "a" }()
	defer func() { result += "b" }()
	defer func() { result += "c" }()
	return ""
}
f()`},
	})
}

func TestGoDispatchDeferOrder(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{"cba", "defer_lifo_order", `
func f() (result string) {
	defer func() { result += "a" }()
	defer func() { result += "b" }()
	defer func() { result += "c" }()
	return ""
}
f()`},
	})
}

func TestComplexConstants(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{complex128(0), "complex_zero", `var c complex128; c`},
		{complex128(1i), "complex_imag_only", `c := 1i; c`},
		{complex128(3.14), "complex_real_only", `c := complex128(3.14); c`},
	})
}

func TestGoDispatchComplexConstants(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{complex128(0), "complex_zero", `var c complex128; c`},
		{complex128(1i), "complex_imag_only", `c := 1i; c`},
	})
}

func TestGoDispatchNilTest(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{true, "nil_test_nil", `var p *int; p == nil`},
		{false, "nil_test_nonnull", `x := 42; p := &x; p == nil`},
	})
}
