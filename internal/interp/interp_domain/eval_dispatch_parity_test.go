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
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDispatchParitySpill(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source string
		expect int64
	}{
		{name: "spill_basic_sum", source: generateAllAliveProgram(260), expect: triangular(260)},
		{name: "spill_with_inc_dec", source: generateSpillIncDecProgram(260), expect: triangular(260) + 1 - 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			asmService := NewService()
			asmResult, asmErr := asmService.EvalFile(ctx, tt.source, "run")

			goService := NewService(WithForceGoDispatch())
			goResult, goErr := goService.EvalFile(ctx, tt.source, "run")

			if asmErr != nil {
				require.Error(t, goErr, "ASM errored but Go did not: asm=%v", asmErr)
			} else {
				require.NoError(t, goErr, "Go errored but ASM did not: go=%v", goErr)
			}

			require.Equal(t, asmResult, goResult,
				"dispatch parity failed: asm=%v go=%v", asmResult, goResult)
			require.Equal(t, tt.expect, asmResult)
		})
	}
}

func generateSpillIncDecProgram(n int) string {
	var b strings.Builder
	b.WriteString("package main\n\nfunc run() int {\n")
	for i := range n {
		fmt.Fprintf(&b, "\tv%d := %d\n", i, i)
	}
	fmt.Fprintf(&b, "\tv%d++\n", n-1)
	fmt.Fprintf(&b, "\tv%d--\n", n-2)
	b.WriteString("\tresult := 0\n")
	for i := range n {
		fmt.Fprintf(&b, "\tresult += v%d\n", i)
	}
	b.WriteString("\treturn result\n}\n")
	return b.String()
}

func TestDispatchParity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{name: "int_arithmetic", code: "2 + 3 * 4", expect: int64(14)},
		{name: "int_division", code: `x := 7; y := 2; x / y`, expect: int64(3)},
		{name: "int_remainder", code: `x := 7; y := 3; x % y`, expect: int64(1)},
		{name: "float_arithmetic", code: `x := 3.14; y := 2.0; x * y`, expect: float64(6.28)},
		{name: "float_division", code: `x := 10.0; y := 4.0; x / y`, expect: float64(2.5)},
		{name: "negation", code: `x := 42; -x`, expect: int64(-42)},
		{name: "float_negation", code: `x := 3.14; -x`, expect: float64(-3.14)},
		{name: "bitwise_and", code: `x := 0xFF; y := 0x0F; x & y`, expect: int64(0x0F)},
		{name: "bitwise_or", code: `x := 0xF0; y := 0x0F; x | y`, expect: int64(0xFF)},
		{name: "bitwise_xor", code: `x := 0xFF; y := 0x0F; x ^ y`, expect: int64(0xF0)},
		{name: "shift_left", code: `x := 1; y := 4; x << y`, expect: int64(16)},
		{name: "shift_right", code: `x := 16; y := 2; x >> y`, expect: int64(4)},
		{name: "comparison_eq_true", code: `x := 42; x == 42`, expect: true},
		{name: "comparison_eq_false", code: `x := 42; x == 43`, expect: false},
		{name: "comparison_lt", code: `x := 5; x < 10`, expect: true},
		{name: "comparison_gt", code: `x := 10; x > 5`, expect: true},
		{name: "boolean_not", code: `x := true; !x`, expect: false},
		{name: "conditional", code: `x := 5; if x > 3 { x = 100 }; x`, expect: int64(100)},
		{name: "for_loop_sum", code: `sum := 0; for i := 0; i < 10; i++ { sum += i }; sum`, expect: int64(45)},
		{name: "string_concat", code: `x := "hello"; y := " world"; x + y`, expect: "hello world"},
		{name: "nested_calls", code: "func fib(n int) int {\n\tif n <= 1 {\n\t\treturn n\n\t}\n\treturn fib(n-1) + fib(n-2)\n}\nfib(10)", expect: int64(55)},
		{name: "closure_capture", code: `x := 10; f := func() int { return x + 5 }; f()`, expect: int64(15)},
		{name: "int_to_float", code: `x := 42; float64(x)`, expect: float64(42.0)},
		{name: "float_to_int", code: `x := 42.7; int(x)`, expect: int64(42)},
		{name: "multiple_returns", code: `
			func swap(a, b int) (int, int) { return b, a }
			x, y := swap(1, 2)
			x*10 + y`, expect: int64(21)},
		{name: "string_length", code: `x := "hello"; len(x)`, expect: int64(5)},
		{name: "slice_operations", code: `
			s := make([]int, 0)
			s = append(s, 1, 2, 3)
			s[0] + s[1] + s[2]`, expect: int64(6)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			asmService := NewService()
			asmResult, asmErr := asmService.Eval(ctx, tt.code)

			goService := NewService(WithForceGoDispatch())
			goResult, goErr := goService.Eval(ctx, tt.code)

			if asmErr != nil {
				require.Error(t, goErr, "ASM errored but Go did not: asm=%v", asmErr)
			} else {
				require.NoError(t, goErr, "Go errored but ASM did not: go=%v", goErr)
			}

			require.Equal(t, asmResult, goResult,
				"dispatch parity failed: asm=%v go=%v", asmResult, goResult)

			require.Equal(t, tt.expect, asmResult)
		})
	}
}
