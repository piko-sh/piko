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
		name   string
		code   string
		expect any
	}{

		{"set_string", `s := make([]string, 3); s[0] = "hi"; s[1] = "there"; s[0]`, "hi"},
		{"set_bool", `s := make([]bool, 2); s[0] = true; s[1] = false; s[0]`, true},
		{"set_bool_read_false", `s := make([]bool, 2); s[0] = true; s[1] = false; s[1]`, false},

		{"append_int", `s := []int{1}; s = append(s, 2, 3); s[2]`, int64(3)},
		{"append_string", `s := []string{"a"}; s = append(s, "b"); s[1]`, "b"},
		{"append_float", `s := []float64{1.0}; s = append(s, 2.5); s[1]`, float64(2.5)},
		{"append_bool", `s := []bool{true}; s = append(s, false); s[1]`, false},
		{"append_multiple", `s := []int{1}; s = append(s, 2, 3, 4, 5); len(s)`, int64(5)},
		{"append_empty", `s := make([]int, 0); s = append(s, 42); s[0]`, int64(42)},
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
		name   string
		code   string
		expect any
	}{
		{"map_int_int_set_get", `m := map[int]int{1: 10}; m[2] = 20; m[1] + m[2]`, int64(30)},
		{"map_int_int_overwrite", `m := map[int]int{1: 10}; m[1] = 99; m[1]`, int64(99)},
		{"map_int_int_len", `m := map[int]int{}; m[1] = 1; m[2] = 2; m[3] = 3; len(m)`, int64(3)},
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
		name   string
		code   string
		expect any
	}{
		{"range_index_value", `sum := 0; for _, v := range []int{10, 20, 30} { sum += v }; sum`, int64(60)},
		{"range_index_only", `last := 0; for i := range []int{10, 20, 30} { last = i }; last`, int64(2)},
		{"range_string_slice", `
s := ""
for _, v := range []string{"a", "b", "c"} {
    s += v
}
s`, "abc"},
		{"range_float_slice", `sum := 0.0; for _, v := range []float64{1.1, 2.2, 3.3} { sum += v }; sum`, float64(6.6)},
		{"range_bool_slice", `count := 0; for _, v := range []bool{true, false, true} { if v { count++ } }; count`, int64(2)},
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
		name   string
		code   string
		expect any
	}{
		{"closure_variable_call", `
f := func(x int) int { return x * 2 }
f(21)`, int64(42)},
		{"closure_in_slice", `
fns := []func(int) int{
    func(x int) int { return x + 1 },
    func(x int) int { return x * 2 },
}
fns[0](10) + fns[1](10)`, int64(31)},
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
		name   string
		code   string
		expect any
	}{
		{"select_send", `
ch := make(chan int, 1)
select {
case ch <- 42:
}
<-ch`, int64(42)},
		{"select_default", `
ch := make(chan int)
x := 0
select {
case v := <-ch:
    x = v
default:
    x = -1
}
x`, int64(-1)},
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
		name   string
		code   string
		expect any
	}{
		{"addr_deref", `x := 42; p := &x; *p`, int64(42)},
		{"addr_modify", `x := 10; p := &x; *p = 20; x`, int64(20)},
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
