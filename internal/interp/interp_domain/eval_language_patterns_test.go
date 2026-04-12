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
)

func TestNamedReturnValues(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: int64(42), name: "named_return_int", code: "func f() (x int) { x = 42; return }\nf()"},
		{expect: "hi", name: "named_return_string", code: "func f() (s string) { s = \"hi\"; return }\nf()"},
		{expect: true, name: "named_return_bool", code: "func f() (b bool) { b = true; return }\nf()"},
		{expect: float64(3.14), name: "named_return_float", code: "func f() (v float64) { v = 3.14; return }\nf()"},
		{expect: int64(1), name: "named_return_multi_first", code: "func f() (a int, b string) { a = 1; b = \"x\"; return }\na, _ := f()\na"},
		{expect: "x", name: "named_return_multi_second", code: "func f() (a int, b string) { a = 1; b = \"x\"; return }\n_, b := f()\nb"},
	})
}

func TestGoDispatchNamedReturnValues(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{expect: int64(42), name: "named_return_int", code: "func f() (x int) { x = 42; return }\nf()"},
		{expect: "hi", name: "named_return_string", code: "func f() (s string) { s = \"hi\"; return }\nf()"},
		{expect: int64(1), name: "named_return_multi_first", code: "func f() (a int, b string) { a = 1; b = \"x\"; return }\na, _ := f()\na"},
	})
}

func TestZeroValueDeclarations(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: int64(0), name: "zero_int", code: `var x int; x`},
		{expect: float64(0), name: "zero_float64", code: `var f float64; f`},
		{expect: "", name: "zero_string", code: `var s string; s`},
		{expect: false, name: "zero_bool", code: `var b bool; b`},
		{expect: uint64(0), name: "zero_uint", code: `var u uint; u`},
	})
}

func TestDeepRecursion(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: int64(50), name: "deep_recursion_50", code: `
func f(n int) int {
	if n <= 0 { return 0 }
	return f(n-1) + 1
}
f(50)`},
	})
}

func TestFloatComparisons(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: true, name: "eq_float", code: `a := 1.5; b := 1.5; a == b`},
		{expect: false, name: "eq_float_false", code: `a := 1.5; b := 2.5; a == b`},
		{expect: true, name: "ne_float", code: `a := 1.5; b := 2.5; a != b`},
		{expect: false, name: "ne_float_false", code: `a := 1.5; b := 1.5; a != b`},
		{expect: true, name: "lt_float", code: `a := 1.5; b := 2.5; a < b`},
		{expect: false, name: "lt_float_false", code: `a := 2.5; b := 1.5; a < b`},
		{expect: true, name: "le_float", code: `a := 1.5; b := 2.5; a <= b`},
		{expect: true, name: "le_float_eq", code: `a := 1.5; b := 1.5; a <= b`},
		{expect: true, name: "gt_float", code: `a := 2.5; b := 1.5; a > b`},
		{expect: false, name: "gt_float_false", code: `a := 1.5; b := 2.5; a > b`},
		{expect: true, name: "ge_float", code: `a := 2.5; b := 1.5; a >= b`},
		{expect: true, name: "ge_float_eq", code: `a := 1.5; b := 1.5; a >= b`},
	})
}

func TestGoDispatchFloatComparisons(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{expect: true, name: "eq_float", code: `a := 1.5; b := 1.5; a == b`},
		{expect: true, name: "ne_float", code: `a := 1.5; b := 2.5; a != b`},
		{expect: true, name: "lt_float", code: `a := 1.5; b := 2.5; a < b`},
		{expect: true, name: "le_float", code: `a := 1.5; b := 2.5; a <= b`},
		{expect: true, name: "gt_float", code: `a := 2.5; b := 1.5; a > b`},
		{expect: true, name: "ge_float", code: `a := 2.5; b := 1.5; a >= b`},
	})
}

func TestSwitchMultiCase(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: int64(10), name: "multi_case_match", code: `x := 2; switch x { case 1, 2, 3: x = 10; case 4: x = 20 }; x`},
		{expect: int64(10), name: "multi_case_first", code: `x := 1; switch x { case 1, 2, 3: x = 10; case 4: x = 20 }; x`},
		{expect: int64(10), name: "multi_case_last", code: `x := 3; switch x { case 1, 2, 3: x = 10; case 4: x = 20 }; x`},
		{expect: int64(20), name: "multi_case_other", code: `x := 4; switch x { case 1, 2, 3: x = 10; case 4: x = 20 }; x`},
		{expect: int64(99), name: "multi_case_default", code: `x := 5; switch x { case 1, 2, 3: x = 10; default: x = 99 }; x`},
	})
}

func TestTypeSwitchDefault(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: int64(1), name: "type_switch_int", code: `var v any = 42; result := 0; switch v.(type) { case int: result = 1; case string: result = 2; default: result = 99 }; result`},
		{expect: int64(2), name: "type_switch_string", code: `var v any = "hi"; result := 0; switch v.(type) { case int: result = 1; case string: result = 2; default: result = 99 }; result`},
		{expect: int64(99), name: "type_switch_default", code: `var v any = 3.14; result := 0; switch v.(type) { case int: result = 1; case string: result = 2; default: result = 99 }; result`},
	})
}

func TestMapCommaOk(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: true, name: "comma_ok_found", code: `m := map[string]int{"a": 1}; _, ok := m["a"]; ok`},
		{expect: false, name: "comma_ok_missing", code: `m := map[string]int{"a": 1}; _, ok := m["b"]; ok`},
		{expect: int64(1), name: "comma_ok_value", code: `m := map[string]int{"a": 1}; v, _ := m["a"]; v`},
		{expect: int64(0), name: "comma_ok_zero", code: `m := map[string]int{"a": 1}; v, _ := m["b"]; v`},
	})
}

func TestAppendVariousTypes(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: "c", name: "append_string", code: `s := []string{"a"}; s = append(s, "b", "c"); s[2]`},
		{expect: float64(2.0), name: "append_float", code: `s := []float64{1.0}; s = append(s, 2.0); s[1]`},
		{expect: false, name: "append_bool", code: `s := []bool{true}; s = append(s, false); s[1]`},
		{expect: int64(4), name: "append_int_multi", code: `s := []int{1}; s = append(s, 2, 3, 4); s[3]`},
		{expect: uint64(5), name: "append_uint", code: `s := []uint{1}; s = append(s, 5); s[1]`},
	})
}

func TestGoDispatchAppendVariousTypes(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{expect: "c", name: "append_string", code: `s := []string{"a"}; s = append(s, "b", "c"); s[2]`},
		{expect: float64(2.0), name: "append_float", code: `s := []float64{1.0}; s = append(s, 2.0); s[1]`},
		{expect: false, name: "append_bool", code: `s := []bool{true}; s = append(s, false); s[1]`},
	})
}

func TestAppendGeneric(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: int64(3), name: "append_any_slice", code: `s := []any{1, "two"}; s = append(s, 3.0); len(s)`},
		{expect: int64(2), name: "append_interface_slice", code: `var s []any; s = append(s, 1); s = append(s, 2); len(s)`},
	})
}

func TestRangeSliceValue(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: int64(60), name: "range_int_sum", code: `sum := 0; for _, v := range []int{10, 20, 30} { sum += v }; sum`},
		{expect: "ab", name: "range_string_concat", code: `result := ""; for _, s := range []string{"a", "b"} { result += s }; result`},
		{expect: int64(2), name: "range_index_only", code: `count := 0; for i := range []int{10, 20, 30} { count = i }; count`},
	})
}

func TestGoDispatchRangeSliceValue(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{expect: int64(60), name: "range_int_sum", code: `sum := 0; for _, v := range []int{10, 20, 30} { sum += v }; sum`},
		{expect: "ab", name: "range_string_concat", code: `result := ""; for _, s := range []string{"a", "b"} { result += s }; result`},
	})
}

func TestTailCallRecursion(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: int64(5050), name: "tail_call_sum", code: `
func sum(n, acc int) int {
	if n == 0 { return acc }
	return sum(n-1, acc+n)
}
sum(100, 0)`},
		{expect: int64(0), name: "tail_call_countdown", code: `
func f(n int) int {
	if n <= 0 { return 0 }
	return f(n-1)
}
f(100)`},
	})
}

func TestImplicitReturn(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: true, name: "implicit_return_void", code: "func f() { x := 1; _ = x }\nf()\ntrue"},
		{expect: true, name: "implicit_return_void_with_if", code: "func f(x int) { if x > 0 { _ = x } }\nf(1)\ntrue"},
	})
}

func TestDeferNamedReturns(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: int64(2), name: "defer_modifies_named", code: `
func f() (result int) {
	result = 1
	defer func() { result = 2 }()
	return
}
f()`},
		{expect: int64(42), name: "defer_preserves_named", code: `
func f() (result int) {
	result = 42
	defer func() {}()
	return
}
f()`},
	})
}

func TestSelectChannelOps(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: true, name: "select_default", code: `
ch := make(chan int)
selected := false
select {
case <-ch:
default:
	selected = true
}
selected`},
		{expect: int64(42), name: "select_recv_direct", code: `
ch := make(chan int, 1)
ch <- 42
<-ch`},
	})
}

func TestGoroutineWithClosure(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: true, name: "goroutine_closure", code: `
ch := make(chan bool, 1)
go func() { ch <- true }()
<-ch`},
		{expect: int64(42), name: "goroutine_value", code: `
ch := make(chan int, 1)
go func() { ch <- 42 }()
<-ch`},
	})
}

func TestCrossBankReturn(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: int64(42), name: "int_to_any", code: `
func f() any {
	return 42
}
x := f().(int)
x`},
		{expect: "hello", name: "string_to_any", code: `
func f() any {
	return "hello"
}
s := f().(string)
s`},
	})
}

func TestGoDispatchCrossBankReturn(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{expect: int64(42), name: "int_to_any", code: `
func f() any {
	return 42
}
x := f().(int)
x`},
	})
}

func TestEmbeddedStructFields(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: int64(42), name: "embedded_field", code: `
type Base struct { Value int }
type Derived struct { Base }
d := Derived{Base: Base{Value: 42}}
d.Value`},
		{expect: int64(10), name: "embedded_method", code: `
type Base struct { X int }
func (b Base) GetX() int { return b.X }
type Derived struct { Base }
d := Derived{Base: Base{X: 10}}
d.GetX()`},
	})
}

func TestLabelledBreakContinue(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: int64(1), name: "labelled_break", code: `
count := 0
outer:
for i := 0; i < 3; i++ {
	for j := 0; j < 3; j++ {
		if j == 1 { break outer }
		count++
	}
}
count`},
		{expect: int64(3), name: "labelled_continue", code: `
count := 0
outer:
for i := 0; i < 3; i++ {
	for j := 0; j < 3; j++ {
		if j == 1 { continue outer }
		count++
	}
}
count`},
	})
}

func TestPanicRecover(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: "caught", name: "basic_recover", code: `
func f() string {
	defer func() { recover() }()
	panic("oops")
}
f()
"caught"`},
		{expect: "oops", name: "recover_value", code: `
func f() string {
	defer func() {
		if r := recover(); r != nil {
			_ = r
		}
	}()
	panic("oops")
}
f()
"oops"`},
	})
}

func TestRangeOverMap(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: int64(3), name: "range_map_count", code: `
m := map[string]int{"a": 1, "b": 2, "c": 3}
count := 0
for range m { count++ }
count`},
		{expect: int64(6), name: "range_map_sum", code: `
m := map[string]int{"a": 1, "b": 2, "c": 3}
sum := 0
for _, v := range m { sum += v }
sum`},
	})
}

func TestRangeOverString(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: int64(5), name: "range_string_count", code: `count := 0; for range "hello" { count++ }; count`},
	})
}

func TestSwitchPatterns(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: int64(1), name: "switch_no_tag", code: `
x := 5
result := 0
switch {
case x < 3: result = -1
case x < 10: result = 1
default: result = 2
}
result`},
		{expect: "medium", name: "switch_string_tag", code: `
s := "b"
result := ""
switch s {
case "a": result = "first"
case "b", "c": result = "medium"
default: result = "other"
}
result`},
	})
}

func TestMapDelete(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: int64(2), name: "map_delete", code: `
m := map[string]int{"a": 1, "b": 2, "c": 3}
delete(m, "a")
len(m)`},
	})
}

func TestRangeOverIntegers(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: int64(10), name: "range_int_sum", code: `sum := 0; for i := range 5 { sum += i }; sum`},
		{expect: int64(5), name: "range_int_count", code: `count := 0; for range 5 { count++ }; count`},
	})
}

func TestCrossBankConversions(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: float64(42), name: "uint_to_float", code: `var u uint = 42; float64(u)`},
		{expect: uint64(3), name: "float_to_uint", code: `x := 3.7; uint(x)`},
		{expect: int64(1), name: "bool_to_int", code: `
func boolToInt(b bool) int {
	if b { return 1 }
	return 0
}
boolToInt(true)`},
	})
}

func TestGoDispatchCrossBankConversions(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{expect: float64(42), name: "uint_to_float", code: `var u uint = 42; float64(u)`},
		{expect: uint64(3), name: "float_to_uint", code: `x := 3.7; uint(x)`},
	})
}

func TestComplexNumbers(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: complex128(3 + 4i), name: "complex_literal", code: `c := 3 + 4i; c`},
		{expect: complex128(4 + 6i), name: "complex_add", code: `a := 1 + 2i; b := 3 + 4i; a + b`},
		{expect: complex128(-2 - 2i), name: "complex_sub", code: `a := 1 + 2i; b := 3 + 4i; a - b`},
		{expect: complex128(-5 + 10i), name: "complex_mul", code: `a := 1 + 2i; b := 3 + 4i; a * b`},
		{expect: float64(3), name: "complex_real", code: `c := 3 + 4i; real(c)`},
		{expect: float64(4), name: "complex_imag", code: `c := 3 + 4i; imag(c)`},
		{expect: complex128(5 + 0i), name: "complex_from_func", code: `c := complex(5.0, 0.0); c`},
	})
}

func TestGoDispatchComplexNumbers(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{expect: complex128(3 + 4i), name: "complex_literal", code: `c := 3 + 4i; c`},
		{expect: complex128(4 + 6i), name: "complex_add", code: `a := 1 + 2i; b := 3 + 4i; a + b`},
	})
}

func TestTypedIntegers(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: int64(42), name: "int8_var", code: `var x int8 = 42; int(x)`},
		{expect: int64(42), name: "int16_var", code: `var x int16 = 42; int(x)`},
		{expect: int64(42), name: "int32_var", code: `var x int32 = 42; int(x)`},
		{expect: int64(42), name: "int64_var", code: `var x int64 = 42; int(x)`},
		{expect: uint64(42), name: "uint8_var", code: `var x uint8 = 42; uint(x)`},
		{expect: uint64(42), name: "uint16_var", code: `var x uint16 = 42; uint(x)`},
		{expect: uint64(42), name: "uint32_var", code: `var x uint32 = 42; uint(x)`},
	})
}

func TestByteSlices(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: int64(3), name: "byte_slice_len", code: `s := []byte{1, 2, 3}; len(s)`},
		{expect: "hello", name: "byte_to_string", code: `s := []byte("hello"); string(s)`},
		{expect: int64(5), name: "string_to_bytes_len", code: `s := []byte("hello"); len(s)`},
	})
}

func TestMultiWayIfElse(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: "small", name: "if_else_chain", code: `
x := 3
result := ""
if x > 10 {
	result = "big"
} else if x > 5 {
	result = "medium"
} else {
	result = "small"
}
result`},
		{expect: "medium", name: "if_else_chain_middle", code: `
x := 7
result := ""
if x > 10 {
	result = "big"
} else if x > 5 {
	result = "medium"
} else {
	result = "small"
}
result`},
	})
}

func TestShortCircuitEval(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: true, name: "and_short_circuit", code: `x := 5; x > 0 && x < 10`},
		{expect: false, name: "and_short_false", code: `x := 5; x > 10 && x < 20`},
		{expect: true, name: "or_short_circuit", code: `x := 5; x > 10 || x < 10`},
		{expect: true, name: "or_short_true", code: `x := 5; x > 0 || x > 10`},
	})
}

func TestNestedStructAccess(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: int64(42), name: "nested_field", code: `
type Inner struct { V int }
type Outer struct { In Inner }
o := Outer{In: Inner{V: 42}}
o.In.V`},
		{expect: int64(99), name: "nested_set", code: `
type Inner struct { V int }
type Outer struct { In Inner }
o := Outer{In: Inner{V: 42}}
o.In.V = 99
o.In.V`},
	})
}

func TestSliceOperations(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: int64(2), name: "slice_subslice_len", code: `s := []int{1, 2, 3, 4, 5}; t := s[1:3]; len(t)`},
		{expect: int64(2), name: "slice_subslice_first", code: `s := []int{1, 2, 3, 4, 5}; t := s[1:3]; t[0]`},
		{expect: int64(3), name: "slice_subslice_second", code: `s := []int{1, 2, 3, 4, 5}; t := s[1:3]; t[1]`},
		{expect: int64(3), name: "slice_from_start", code: `s := []int{1, 2, 3, 4, 5}; t := s[:3]; len(t)`},
		{expect: int64(2), name: "slice_to_end", code: `s := []int{1, 2, 3, 4, 5}; t := s[3:]; len(t)`},
	})
}

func TestStringOperations(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: int64(5), name: "string_len", code: `s := "hello"; len(s)`},
		{expect: "helloworld", name: "string_concat", code: `a := "hello"; b := "world"; a + b`},
		{expect: int64(104), name: "string_index_byte", code: `s := "hello"; int(s[0])`},
	})
}

func TestSwitchFallthrough(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: int64(3), name: "fallthrough_basic", code: `
result := 0
switch 1 {
case 1:
	result++
	fallthrough
case 2:
	result++
	fallthrough
case 3:
	result++
}
result`},
	})
}

func TestInitVarPatterns(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: int64(10), name: "short_var_decl", code: `x := 10; x`},
		{expect: int64(20), name: "var_reassign", code: `x := 10; x = 20; x`},
		{expect: int64(30), name: "multi_var_decl", code: `x, y := 10, 20; x + y`},
		{expect: int64(5), name: "var_in_if_init", code: `
result := 0
if x := 5; x > 0 { result = x }
result`},
	})
}

func TestConstDeclarations(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: int64(42), name: "const_int", code: `const x = 42; x`},
		{expect: "hello", name: "const_string", code: `const s = "hello"; s`},
		{expect: true, name: "const_bool", code: `const b = true; b`},
		{expect: int64(10), name: "const_expr", code: `const x = 5 * 2; x`},
	})
}

func TestIota(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: int64(2), name: "iota_enum", code: `
const (
	A = iota
	B
	C
)
C`},
		{expect: int64(4), name: "iota_expr", code: `
const (
	X = iota * 2
	Y
	Z
)
Z`},
	})
}

func TestMakeBuiltin(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: int64(0), name: "make_slice_empty", code: `s := make([]int, 0); len(s)`},
		{expect: int64(5), name: "make_slice_len", code: `s := make([]int, 5); len(s)`},
		{expect: int64(10), name: "make_slice_cap", code: `s := make([]int, 5, 10); cap(s)`},
		{expect: int64(0), name: "make_map_empty", code: `m := make(map[string]int); len(m)`},
	})
}

func TestForRangeBlank(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{expect: int64(3), name: "range_blank_key_value", code: `
count := 0
for range []int{1, 2, 3} { count++ }
count`},
		{expect: int64(15), name: "range_blank_key", code: `
sum := 0
for _, v := range []int{1, 2, 3, 4, 5} { sum += v }
sum`},
	})
}
