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
		{int64(42), "named_return_int", "func f() (x int) { x = 42; return }\nf()"},
		{"hi", "named_return_string", "func f() (s string) { s = \"hi\"; return }\nf()"},
		{true, "named_return_bool", "func f() (b bool) { b = true; return }\nf()"},
		{float64(3.14), "named_return_float", "func f() (v float64) { v = 3.14; return }\nf()"},
		{int64(1), "named_return_multi_first", "func f() (a int, b string) { a = 1; b = \"x\"; return }\na, _ := f()\na"},
		{"x", "named_return_multi_second", "func f() (a int, b string) { a = 1; b = \"x\"; return }\n_, b := f()\nb"},
	})
}

func TestGoDispatchNamedReturnValues(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{int64(42), "named_return_int", "func f() (x int) { x = 42; return }\nf()"},
		{"hi", "named_return_string", "func f() (s string) { s = \"hi\"; return }\nf()"},
		{int64(1), "named_return_multi_first", "func f() (a int, b string) { a = 1; b = \"x\"; return }\na, _ := f()\na"},
	})
}

func TestZeroValueDeclarations(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{int64(0), "zero_int", `var x int; x`},
		{float64(0), "zero_float64", `var f float64; f`},
		{"", "zero_string", `var s string; s`},
		{false, "zero_bool", `var b bool; b`},
		{uint64(0), "zero_uint", `var u uint; u`},
	})
}

func TestDeepRecursion(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{int64(50), "deep_recursion_50", `
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
		{true, "eq_float", `a := 1.5; b := 1.5; a == b`},
		{false, "eq_float_false", `a := 1.5; b := 2.5; a == b`},
		{true, "ne_float", `a := 1.5; b := 2.5; a != b`},
		{false, "ne_float_false", `a := 1.5; b := 1.5; a != b`},
		{true, "lt_float", `a := 1.5; b := 2.5; a < b`},
		{false, "lt_float_false", `a := 2.5; b := 1.5; a < b`},
		{true, "le_float", `a := 1.5; b := 2.5; a <= b`},
		{true, "le_float_eq", `a := 1.5; b := 1.5; a <= b`},
		{true, "gt_float", `a := 2.5; b := 1.5; a > b`},
		{false, "gt_float_false", `a := 1.5; b := 2.5; a > b`},
		{true, "ge_float", `a := 2.5; b := 1.5; a >= b`},
		{true, "ge_float_eq", `a := 1.5; b := 1.5; a >= b`},
	})
}

func TestGoDispatchFloatComparisons(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{true, "eq_float", `a := 1.5; b := 1.5; a == b`},
		{true, "ne_float", `a := 1.5; b := 2.5; a != b`},
		{true, "lt_float", `a := 1.5; b := 2.5; a < b`},
		{true, "le_float", `a := 1.5; b := 2.5; a <= b`},
		{true, "gt_float", `a := 2.5; b := 1.5; a > b`},
		{true, "ge_float", `a := 2.5; b := 1.5; a >= b`},
	})
}

func TestSwitchMultiCase(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{int64(10), "multi_case_match", `x := 2; switch x { case 1, 2, 3: x = 10; case 4: x = 20 }; x`},
		{int64(10), "multi_case_first", `x := 1; switch x { case 1, 2, 3: x = 10; case 4: x = 20 }; x`},
		{int64(10), "multi_case_last", `x := 3; switch x { case 1, 2, 3: x = 10; case 4: x = 20 }; x`},
		{int64(20), "multi_case_other", `x := 4; switch x { case 1, 2, 3: x = 10; case 4: x = 20 }; x`},
		{int64(99), "multi_case_default", `x := 5; switch x { case 1, 2, 3: x = 10; default: x = 99 }; x`},
	})
}

func TestTypeSwitchDefault(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{int64(1), "type_switch_int", `var v any = 42; result := 0; switch v.(type) { case int: result = 1; case string: result = 2; default: result = 99 }; result`},
		{int64(2), "type_switch_string", `var v any = "hi"; result := 0; switch v.(type) { case int: result = 1; case string: result = 2; default: result = 99 }; result`},
		{int64(99), "type_switch_default", `var v any = 3.14; result := 0; switch v.(type) { case int: result = 1; case string: result = 2; default: result = 99 }; result`},
	})
}

func TestMapCommaOk(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{true, "comma_ok_found", `m := map[string]int{"a": 1}; _, ok := m["a"]; ok`},
		{false, "comma_ok_missing", `m := map[string]int{"a": 1}; _, ok := m["b"]; ok`},
		{int64(1), "comma_ok_value", `m := map[string]int{"a": 1}; v, _ := m["a"]; v`},
		{int64(0), "comma_ok_zero", `m := map[string]int{"a": 1}; v, _ := m["b"]; v`},
	})
}

func TestAppendVariousTypes(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{"c", "append_string", `s := []string{"a"}; s = append(s, "b", "c"); s[2]`},
		{float64(2.0), "append_float", `s := []float64{1.0}; s = append(s, 2.0); s[1]`},
		{false, "append_bool", `s := []bool{true}; s = append(s, false); s[1]`},
		{int64(4), "append_int_multi", `s := []int{1}; s = append(s, 2, 3, 4); s[3]`},
		{uint64(5), "append_uint", `s := []uint{1}; s = append(s, 5); s[1]`},
	})
}

func TestGoDispatchAppendVariousTypes(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{"c", "append_string", `s := []string{"a"}; s = append(s, "b", "c"); s[2]`},
		{float64(2.0), "append_float", `s := []float64{1.0}; s = append(s, 2.0); s[1]`},
		{false, "append_bool", `s := []bool{true}; s = append(s, false); s[1]`},
	})
}

func TestAppendGeneric(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{int64(3), "append_any_slice", `s := []any{1, "two"}; s = append(s, 3.0); len(s)`},
		{int64(2), "append_interface_slice", `var s []any; s = append(s, 1); s = append(s, 2); len(s)`},
	})
}

func TestRangeSliceValue(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{int64(60), "range_int_sum", `sum := 0; for _, v := range []int{10, 20, 30} { sum += v }; sum`},
		{"ab", "range_string_concat", `result := ""; for _, s := range []string{"a", "b"} { result += s }; result`},
		{int64(2), "range_index_only", `count := 0; for i := range []int{10, 20, 30} { count = i }; count`},
	})
}

func TestGoDispatchRangeSliceValue(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{int64(60), "range_int_sum", `sum := 0; for _, v := range []int{10, 20, 30} { sum += v }; sum`},
		{"ab", "range_string_concat", `result := ""; for _, s := range []string{"a", "b"} { result += s }; result`},
	})
}

func TestTailCallRecursion(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{int64(5050), "tail_call_sum", `
func sum(n, acc int) int {
	if n == 0 { return acc }
	return sum(n-1, acc+n)
}
sum(100, 0)`},
		{int64(0), "tail_call_countdown", `
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
		{true, "implicit_return_void", "func f() { x := 1; _ = x }\nf()\ntrue"},
		{true, "implicit_return_void_with_if", "func f(x int) { if x > 0 { _ = x } }\nf(1)\ntrue"},
	})
}

func TestDeferNamedReturns(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{int64(2), "defer_modifies_named", `
func f() (result int) {
	result = 1
	defer func() { result = 2 }()
	return
}
f()`},
		{int64(42), "defer_preserves_named", `
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
		{true, "select_default", `
ch := make(chan int)
selected := false
select {
case <-ch:
default:
	selected = true
}
selected`},
		{int64(42), "select_recv_direct", `
ch := make(chan int, 1)
ch <- 42
<-ch`},
	})
}

func TestGoroutineWithClosure(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{true, "goroutine_closure", `
ch := make(chan bool, 1)
go func() { ch <- true }()
<-ch`},
		{int64(42), "goroutine_value", `
ch := make(chan int, 1)
go func() { ch <- 42 }()
<-ch`},
	})
}

func TestCrossBankReturn(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{int64(42), "int_to_any", `
func f() any {
	return 42
}
x := f().(int)
x`},
		{"hello", "string_to_any", `
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
		{int64(42), "int_to_any", `
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
		{int64(42), "embedded_field", `
type Base struct { Value int }
type Derived struct { Base }
d := Derived{Base: Base{Value: 42}}
d.Value`},
		{int64(10), "embedded_method", `
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
		{int64(1), "labelled_break", `
count := 0
outer:
for i := 0; i < 3; i++ {
	for j := 0; j < 3; j++ {
		if j == 1 { break outer }
		count++
	}
}
count`},
		{int64(3), "labelled_continue", `
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
		{"caught", "basic_recover", `
func f() string {
	defer func() { recover() }()
	panic("oops")
}
f()
"caught"`},
		{"oops", "recover_value", `
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
		{int64(3), "range_map_count", `
m := map[string]int{"a": 1, "b": 2, "c": 3}
count := 0
for range m { count++ }
count`},
		{int64(6), "range_map_sum", `
m := map[string]int{"a": 1, "b": 2, "c": 3}
sum := 0
for _, v := range m { sum += v }
sum`},
	})
}

func TestRangeOverString(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{int64(5), "range_string_count", `count := 0; for range "hello" { count++ }; count`},
	})
}

func TestSwitchPatterns(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{int64(1), "switch_no_tag", `
x := 5
result := 0
switch {
case x < 3: result = -1
case x < 10: result = 1
default: result = 2
}
result`},
		{"medium", "switch_string_tag", `
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
		{int64(2), "map_delete", `
m := map[string]int{"a": 1, "b": 2, "c": 3}
delete(m, "a")
len(m)`},
	})
}

func TestRangeOverIntegers(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{int64(10), "range_int_sum", `sum := 0; for i := range 5 { sum += i }; sum`},
		{int64(5), "range_int_count", `count := 0; for range 5 { count++ }; count`},
	})
}

func TestCrossBankConversions(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{float64(42), "uint_to_float", `var u uint = 42; float64(u)`},
		{uint64(3), "float_to_uint", `x := 3.7; uint(x)`},
		{int64(1), "bool_to_int", `
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
		{float64(42), "uint_to_float", `var u uint = 42; float64(u)`},
		{uint64(3), "float_to_uint", `x := 3.7; uint(x)`},
	})
}

func TestComplexNumbers(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{complex128(3 + 4i), "complex_literal", `c := 3 + 4i; c`},
		{complex128(4 + 6i), "complex_add", `a := 1 + 2i; b := 3 + 4i; a + b`},
		{complex128(-2 - 2i), "complex_sub", `a := 1 + 2i; b := 3 + 4i; a - b`},
		{complex128(-5 + 10i), "complex_mul", `a := 1 + 2i; b := 3 + 4i; a * b`},
		{float64(3), "complex_real", `c := 3 + 4i; real(c)`},
		{float64(4), "complex_imag", `c := 3 + 4i; imag(c)`},
		{complex128(5 + 0i), "complex_from_func", `c := complex(5.0, 0.0); c`},
	})
}

func TestGoDispatchComplexNumbers(t *testing.T) {
	t.Parallel()
	runEvalTable(t, []Option{WithForceGoDispatch()}, []evalTestCase{
		{complex128(3 + 4i), "complex_literal", `c := 3 + 4i; c`},
		{complex128(4 + 6i), "complex_add", `a := 1 + 2i; b := 3 + 4i; a + b`},
	})
}

func TestTypedIntegers(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{int64(42), "int8_var", `var x int8 = 42; int(x)`},
		{int64(42), "int16_var", `var x int16 = 42; int(x)`},
		{int64(42), "int32_var", `var x int32 = 42; int(x)`},
		{int64(42), "int64_var", `var x int64 = 42; int(x)`},
		{uint64(42), "uint8_var", `var x uint8 = 42; uint(x)`},
		{uint64(42), "uint16_var", `var x uint16 = 42; uint(x)`},
		{uint64(42), "uint32_var", `var x uint32 = 42; uint(x)`},
	})
}

func TestByteSlices(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{int64(3), "byte_slice_len", `s := []byte{1, 2, 3}; len(s)`},
		{"hello", "byte_to_string", `s := []byte("hello"); string(s)`},
		{int64(5), "string_to_bytes_len", `s := []byte("hello"); len(s)`},
	})
}

func TestMultiWayIfElse(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{"small", "if_else_chain", `
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
		{"medium", "if_else_chain_middle", `
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
		{true, "and_short_circuit", `x := 5; x > 0 && x < 10`},
		{false, "and_short_false", `x := 5; x > 10 && x < 20`},
		{true, "or_short_circuit", `x := 5; x > 10 || x < 10`},
		{true, "or_short_true", `x := 5; x > 0 || x > 10`},
	})
}

func TestNestedStructAccess(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{int64(42), "nested_field", `
type Inner struct { V int }
type Outer struct { In Inner }
o := Outer{In: Inner{V: 42}}
o.In.V`},
		{int64(99), "nested_set", `
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
		{int64(2), "slice_subslice_len", `s := []int{1, 2, 3, 4, 5}; t := s[1:3]; len(t)`},
		{int64(2), "slice_subslice_first", `s := []int{1, 2, 3, 4, 5}; t := s[1:3]; t[0]`},
		{int64(3), "slice_subslice_second", `s := []int{1, 2, 3, 4, 5}; t := s[1:3]; t[1]`},
		{int64(3), "slice_from_start", `s := []int{1, 2, 3, 4, 5}; t := s[:3]; len(t)`},
		{int64(2), "slice_to_end", `s := []int{1, 2, 3, 4, 5}; t := s[3:]; len(t)`},
	})
}

func TestStringOperations(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{int64(5), "string_len", `s := "hello"; len(s)`},
		{"helloworld", "string_concat", `a := "hello"; b := "world"; a + b`},
		{int64(104), "string_index_byte", `s := "hello"; int(s[0])`},
	})
}

func TestSwitchFallthrough(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{int64(3), "fallthrough_basic", `
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
		{int64(10), "short_var_decl", `x := 10; x`},
		{int64(20), "var_reassign", `x := 10; x = 20; x`},
		{int64(30), "multi_var_decl", `x, y := 10, 20; x + y`},
		{int64(5), "var_in_if_init", `
result := 0
if x := 5; x > 0 { result = x }
result`},
	})
}

func TestConstDeclarations(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{int64(42), "const_int", `const x = 42; x`},
		{"hello", "const_string", `const s = "hello"; s`},
		{true, "const_bool", `const b = true; b`},
		{int64(10), "const_expr", `const x = 5 * 2; x`},
	})
}

func TestIota(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{int64(2), "iota_enum", `
const (
	A = iota
	B
	C
)
C`},
		{int64(4), "iota_expr", `
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
		{int64(0), "make_slice_empty", `s := make([]int, 0); len(s)`},
		{int64(5), "make_slice_len", `s := make([]int, 5); len(s)`},
		{int64(10), "make_slice_cap", `s := make([]int, 5, 10); cap(s)`},
		{int64(0), "make_map_empty", `m := make(map[string]int); len(m)`},
	})
}

func TestForRangeBlank(t *testing.T) {
	t.Parallel()
	runEvalTable(t, nil, []evalTestCase{
		{int64(3), "range_blank_key_value", `
count := 0
for range []int{1, 2, 3} { count++ }
count`},
		{int64(15), "range_blank_key", `
sum := 0
for _, v := range []int{1, 2, 3, 4, 5} { sum += v }
sum`},
	})
}
