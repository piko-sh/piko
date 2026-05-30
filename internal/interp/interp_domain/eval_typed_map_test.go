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

func TestTypedMapFastPathCorrectness(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "map_string_int_set_then_get",
			code: `func run() int {
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	return m["a"] + m["b"] + m["c"]
}
run()`,
			expect: int64(6),
		},
		{
			name: "map_string_int_make_and_populate",
			code: `func run() int {
	m := make(map[string]int, 8)
	m["one"] = 1
	m["two"] = 2
	m["three"] = 3
	return m["one"]*100 + m["two"]*10 + m["three"]
}
run()`,
			expect: int64(123),
		},
		{
			name: "map_string_int_missing_key_returns_zero",
			code: `func run() int {
	m := map[string]int{"present": 42}
	return m["absent"]
}
run()`,
			expect: int64(0),
		},
		{
			name: "map_string_string_set_then_get",
			code: `func run() string {
	m := map[string]string{"hello": "world"}
	m["foo"] = "bar"
	return m["hello"] + " " + m["foo"]
}
run()`,
			expect: "world bar",
		},
		{
			name: "map_int_string_set_then_get",
			code: `func run() string {
	m := map[int]string{1: "one", 2: "two"}
	m[3] = "three"
	return m[1] + "-" + m[2] + "-" + m[3]
}
run()`,
			expect: "one-two-three",
		},
		{
			name: "map_int_int_set_then_get",
			code: `func run() int {
	m := make(map[int]int, 8)
	m[10] = 100
	m[20] = 200
	m[30] = 300
	return m[10] + m[20] + m[30]
}
run()`,
			expect: int64(600),
		},
		{
			name: "map_string_int_loop_lookup",
			code: `func run() int {
	m := map[string]int{"a": 1, "b": 2, "c": 3, "d": 4, "e": 5}
	keys := []string{"a", "b", "c", "d", "e"}
	s := 0
	for i := 0; i < 100; i++ {
		s += m[keys[i%5]]
	}
	return s
}
run()`,
			expect: int64(20 * (1 + 2 + 3 + 4 + 5)),
		},
		{
			name: "map_string_int_overwrite",
			code: `func run() int {
	m := make(map[string]int, 4)
	m["k"] = 1
	m["k"] = 2
	m["k"] = 3
	return m["k"]
}
run()`,
			expect: int64(3),
		},
		{
			name: "named_map_string_int",
			code: `type Config map[string]int
func run() int {
	c := Config{"version": 7, "build": 100}
	return c["version"] + c["build"]
}
run()`,
			expect: int64(107),
		},
		{
			name: "map_string_int_compound_assign",
			code: `func run() int {
	m := map[string]int{"counter": 0}
	for i := 0; i < 10; i++ {
		m["counter"] = m["counter"] + 1
	}
	return m["counter"]
}
run()`,
			expect: int64(10),
		},
		{
			name: "map_int_string_lookup_in_loop",
			code: `func run() string {
	m := map[int]string{0: "zero", 1: "one", 2: "two"}
	result := ""
	for i := 0; i < 3; i++ {
		result += m[i]
	}
	return result
}
run()`,
			expect: "zeroonetwo",
		},
		{
			name: "map_string_string_iteration_via_keys",
			code: `func run() string {
	m := map[string]string{"x": "X", "y": "Y", "z": "Z"}
	keys := []string{"x", "y", "z"}
	result := ""
	for _, k := range keys {
		result += m[k]
	}
	return result
}
run()`,
			expect: "XYZ",
		},
		{
			name: "map_string_int_comma_ok_present",
			code: `func run() int {
	m := map[string]int{"a": 1, "b": 2}
	if v, ok := m["a"]; ok {
		return v * 10
	}
	return -1
}
run()`,
			expect: int64(10),
		},
		{
			name: "map_string_int_comma_ok_absent",
			code: `func run() int {
	m := map[string]int{"a": 1}
	if v, ok := m["missing"]; ok {
		return v
	} else {
		return v - 1
	}
}
run()`,
			expect: int64(-1),
		},
		{
			name: "map_string_string_comma_ok_present",
			code: `func run() string {
	m := map[string]string{"hello": "world"}
	if v, ok := m["hello"]; ok {
		return v + "!"
	}
	return "missing"
}
run()`,
			expect: "world!",
		},
		{
			name: "map_string_string_comma_ok_absent",
			code: `func run() string {
	m := map[string]string{"hi": "there"}
	v, ok := m["bye"]
	if ok {
		return v
	}
	return "fallback"
}
run()`,
			expect: "fallback",
		},
		{
			name: "map_int_int_comma_ok",
			code: `func run() int {
	m := map[int]int{1: 10, 2: 20}
	v1, ok1 := m[1]
	v2, ok2 := m[99]
	if !ok1 || ok2 {
		return -1
	}
	return v1 + v2
}
run()`,
			expect: int64(10),
		},
		{
			name: "map_int_string_comma_ok",
			code: `func run() string {
	m := map[int]string{1: "one", 2: "two"}
	if v, ok := m[2]; ok {
		return "found-" + v
	}
	return "miss"
}
run()`,
			expect: "found-two",
		},
		{
			name: "comma_ok_loop_count_present",
			code: `func run() int {
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	keys := []string{"a", "b", "c", "d", "e"}
	count := 0
	for _, k := range keys {
		if _, ok := m[k]; ok {
			count++
		}
	}
	return count
}
run()`,
			expect: int64(3),
		},
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
