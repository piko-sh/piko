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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTypedSliceOperations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{

		{name: "set_string", code: `s := make([]string, 3); s[0] = "hi"; s[1] = "there"; s[0]`, expect: "hi"},
		{name: "set_bool", code: `s := make([]bool, 2); s[0] = true; s[1] = false; s[0]`, expect: true},
		{name: "set_bool_read_false", code: `s := make([]bool, 2); s[0] = true; s[1] = false; s[1]`, expect: false},

		{name: "append_int", code: `s := []int{1}; s = append(s, 2, 3); s[2]`, expect: int64(3)},
		{name: "append_string", code: `s := []string{"a"}; s = append(s, "b"); s[1]`, expect: "b"},
		{name: "append_float", code: `s := []float64{1.0}; s = append(s, 2.5); s[1]`, expect: float64(2.5)},
		{name: "append_bool", code: `s := []bool{true}; s = append(s, false); s[1]`, expect: false},
		{name: "append_multiple", code: `s := []int{1}; s = append(s, 2, 3, 4, 5); len(s)`, expect: int64(5)},
		{name: "append_empty", code: `s := make([]int, 0); s = append(s, 42); s[0]`, expect: int64(42)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err, "code: %s", tt.code)
			require.Equal(t, tt.expect, result, "code: %s", tt.code)
		})
	}
}

func TestMapIntKeyOperations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{name: "map_int_int_set_get", code: `m := map[int]int{1: 10}; m[2] = 20; m[1] + m[2]`, expect: int64(30)},
		{name: "map_int_int_overwrite", code: `m := map[int]int{1: 10}; m[1] = 99; m[1]`, expect: int64(99)},
		{name: "map_int_int_len", code: `m := map[int]int{}; m[1] = 1; m[2] = 2; m[3] = 3; len(m)`, expect: int64(3)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err, "code: %s", tt.code)
			require.Equal(t, tt.expect, result, "code: %s", tt.code)
		})
	}
}

func TestRangeOverSlice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{name: "range_index_value", code: `sum := 0; for _, v := range []int{10, 20, 30} { sum += v }; sum`, expect: int64(60)},
		{name: "range_index_only", code: `last := 0; for i := range []int{10, 20, 30} { last = i }; last`, expect: int64(2)},
		{name: "range_string_slice", code: `
s := ""
for _, v := range []string{"a", "b", "c"} {
    s += v
}
s`, expect: "abc"},
		{name: "range_float_slice", code: `sum := 0.0; for _, v := range []float64{1.1, 2.2, 3.3} { sum += v }; sum`, expect: float64(6.6)},
		{name: "range_bool_slice", code: `count := 0; for _, v := range []bool{true, false, true} { if v { count++ } }; count`, expect: int64(2)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err, "code: %s", tt.code)
			require.Equal(t, tt.expect, result, "code: %s", tt.code)
		})
	}
}

func TestMethodValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{name: "closure_variable_call", code: `
f := func(x int) int { return x * 2 }
f(21)`, expect: int64(42)},
		{name: "closure_in_slice", code: `
fns := []func(int) int{
    func(x int) int { return x + 1 },
    func(x int) int { return x * 2 },
}
fns[0](10) + fns[1](10)`, expect: int64(31)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err, "code: %s", tt.code)
			require.Equal(t, tt.expect, result, "code: %s", tt.code)
		})
	}
}

func TestSelectSend(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{name: "select_send", code: `
ch := make(chan int, 1)
select {
case ch <- 42:
}
<-ch`, expect: int64(42)},
		{name: "select_default", code: `
ch := make(chan int)
x := 0
select {
case v := <-ch:
    x = v
default:
    x = -1
}
x`, expect: int64(-1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err, "code: %s", tt.code)
			require.Equal(t, tt.expect, result, "code: %s", tt.code)
		})
	}
}

func TestAddressOf(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{name: "addr_deref", code: `x := 42; p := &x; *p`, expect: int64(42)},
		{name: "addr_modify", code: `x := 10; p := &x; *p = 20; x`, expect: int64(20)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err, "code: %s", tt.code)
			require.Equal(t, tt.expect, result, "code: %s", tt.code)
		})
	}
}
