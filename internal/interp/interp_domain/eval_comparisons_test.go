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

func TestEvalIntComparisonsExtended(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{name: "ne true", code: `1 != 2`, expect: true},
		{name: "ne false", code: `1 != 1`, expect: false},
		{name: "le true equal", code: `1 <= 1`, expect: true},
		{name: "le true less", code: `0 <= 1`, expect: true},
		{name: "le false", code: `2 <= 1`, expect: false},
		{name: "ne negative", code: `-1 != 1`, expect: true},
		{name: "le with vars", code: `x := 5; y := 10; x <= y`, expect: true},
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

func TestEvalFloatComparisons(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{name: "eq true", code: `1.0 == 1.0`, expect: true},
		{name: "eq false", code: `1.0 == 2.0`, expect: false},
		{name: "lt true", code: `1.0 < 2.0`, expect: true},
		{name: "lt false", code: `2.0 < 1.0`, expect: false},
		{name: "le true equal", code: `2.0 <= 2.0`, expect: true},
		{name: "le true less", code: `1.0 <= 2.0`, expect: true},
		{name: "le false", code: `3.0 <= 2.0`, expect: false},
		{name: "gt float", code: `3.14 > 2.72`, expect: true},
		{name: "ge float", code: `3.14 >= 3.14`, expect: true},
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

func TestEvalStringComparisons(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{name: "lt true", code: `"abc" < "abd"`, expect: true},
		{name: "lt false", code: `"b" < "a"`, expect: false},
		{name: "le true equal", code: `"abc" <= "abc"`, expect: true},
		{name: "le true less", code: `"a" <= "b"`, expect: true},
		{name: "le false", code: `"z" <= "a"`, expect: false},
		{name: "ne string", code: `"hello" != "world"`, expect: true},
		{name: "ne string false", code: `"same" != "same"`, expect: false},
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

func TestEvalLogicalOperators(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{name: "and true", code: `true && true`, expect: true},
		{name: "and false left", code: `false && true`, expect: false},
		{name: "and false right", code: `true && false`, expect: false},
		{name: "or true left", code: `true || false`, expect: true},
		{name: "or true right", code: `false || true`, expect: true},
		{name: "or false", code: `false || false`, expect: false},
		{name: "and with comparison", code: `1 < 2 && 3 < 4`, expect: true},
		{name: "or with comparison", code: `1 > 2 || 3 < 4`, expect: true},
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

func TestEvalGenericComparisons(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     string
		entrypoint string
		expect     any
	}{
		{
			name: "generic_uint_lt",
			source: `package main

func less[T ~uint | ~uint64](a, b T) bool { return a < b }
func run() bool { return less(uint(10), uint(20)) }
func main() {}
`,
			entrypoint: "run",
			expect:     true,
		},
		{
			name: "generic_uint_ge",
			source: `package main

func ge[T ~uint | ~uint64](a, b T) bool { return a >= b }
func run() bool { return ge(uint(20), uint(10)) }
func main() {}
`,
			entrypoint: "run",
			expect:     true,
		},
		{
			name: "generic_float_lt",
			source: `package main

func less[T ~float64](a, b T) bool { return a < b }
func run() bool { return less(3.14, 2.71) }
func main() {}
`,
			entrypoint: "run",
			expect:     false,
		},
		{
			name: "generic_float_ge",
			source: `package main

func ge[T ~float64](a, b T) bool { return a >= b }
func run() bool { return ge(1.0, 1.0) }
func main() {}
`,
			entrypoint: "run",
			expect:     true,
		},
		{
			name: "generic_string_lt",
			source: `package main

func less[T ~string](a, b T) bool { return a < b }
func run() bool { return less("abc", "abd") }
func main() {}
`,
			entrypoint: "run",
			expect:     true,
		},
		{
			name: "generic_string_ge",
			source: `package main

func ge[T ~string](a, b T) bool { return a >= b }
func run() bool { return ge("xyz", "abc") }
func main() {}
`,
			entrypoint: "run",
			expect:     true,
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

func TestEvalBoolComparisonBoxedAsAny(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		source string
	}{
		{
			name: "bool_compare_returned_as_any",
			source: `package main

func produce(a, b int) any { return a < b }
func run() bool {
	v := produce(1, 2)
	return v.(bool)
}
func main() {}
`,
			expect: true,
		},
		{
			name: "bool_compare_stored_as_any",
			source: `package main

func run() bool {
	var v any = 1 < 2
	return v.(bool)
}
func main() {}
`,
			expect: true,
		},
		{
			name: "bool_compare_in_slice_any",
			source: `package main

func run() bool {
	xs := []any{1 < 2, 1 > 2}
	return xs[0].(bool) && !xs[1].(bool)
}
func main() {}
`,
			expect: true,
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
