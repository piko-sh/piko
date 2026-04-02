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

func TestEvalSliceExtended(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "slice of floats index",
			code: `s := []float64{1.1, 2.2, 3.3}
s[1]`,
			expect: 2.2,
		},
		{
			name: "slice append single then len",
			code: `s := []int{1, 2}
s = append(s, 3)
len(s)`,
			expect: int64(3),
		},
		{
			name: "slice copy then index",
			code: `s := []int{1, 2, 3}
d := make([]int, 3)
copy(d, s)
d[2]`,
			expect: int64(3),
		},
		{
			name: "slice of strings index",
			code: `s := []string{"a", "b"}
s[0]`,
			expect: "a",
		},
		{
			name: "slice capacity via make",
			code: `s := make([]int, 2, 10)
cap(s)`,
			expect: int64(10),
		},
		{
			name: "three-index slice cap",
			code: `s := []int{1, 2, 3, 4, 5}
t := s[1:3:4]
cap(t)`,
			expect: int64(3),
		},
		{
			name: "slice set float",
			code: `s := []float64{0.0, 0.0}
s[0] = 1.5
s[0]`,
			expect: 1.5,
		},
		{
			name: "slice length via variable",
			code: `s := []int{1, 2, 3}
n := len(s)
n`,
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

func TestEvalMapExtended(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect any
		skip   string
	}{
		{
			name: "map delete then len",
			code: `m := map[string]int{"a": 1, "b": 2}
delete(m, "a")
len(m)`,
			expect: int64(1),
		},
		{
			name: "map iteration count",
			code: `m := map[string]int{"a": 1, "b": 2, "c": 3}
count := 0
for range m { count++ }
count`,
			expect: int64(3),
		},
		{
			name: "map comma-ok true returns value",
			code: `m := map[string]int{"x": 42}
v, ok := m["x"]
_ = ok
v`,
			expect: int64(42),
		},
		{
			name: "map comma-ok false returns ok",
			code: `m := map[string]int{}
_, ok := m["x"]
ok`,
			expect: false,
		},
		{
			name:   "map with string values",
			code:   "m := map[int]string{1: \"one\"}\nm[1]",
			expect: "one",
		},
		{
			name: "map clear then len",
			code: `m := map[string]int{"a": 1}
clear(m)
len(m)`,
			expect: int64(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.skip != "" {
				t.Skip(tt.skip)
			}
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err, "code: %s", tt.code)
			require.Equal(t, tt.expect, result, "code: %s", tt.code)
		})
	}
}

func TestEvalStructOperations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "struct field sum",
			code: `type Point struct{ X, Y int }
p := Point{X: 3, Y: 4}
p.X + p.Y`,
			expect: int64(7),
		},
		{
			name: "struct field set then read",
			code: `type Point struct{ X, Y int }
p := Point{X: 1, Y: 2}
p.X = 10
p.X`,
			expect: int64(10),
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
