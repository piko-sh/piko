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
		{
			"spill_basic_sum",
			generateAllAliveProgram(260),
			triangular(260),
		},
		{
			"spill_with_inc_dec",
			generateSpillIncDecProgram(260),
			triangular(260) + 1 - 1,
		},
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
		name   string
		code   string
		expect any
	}{
		{"int_arithmetic", "2 + 3 * 4", int64(14)},
		{"int_division", `x := 7; y := 2; x / y`, int64(3)},
		{"int_remainder", `x := 7; y := 3; x % y`, int64(1)},
		{"float_arithmetic", `x := 3.14; y := 2.0; x * y`, float64(6.28)},
		{"float_division", `x := 10.0; y := 4.0; x / y`, float64(2.5)},
		{"negation", `x := 42; -x`, int64(-42)},
		{"float_negation", `x := 3.14; -x`, float64(-3.14)},
		{"bitwise_and", `x := 0xFF; y := 0x0F; x & y`, int64(0x0F)},
		{"bitwise_or", `x := 0xF0; y := 0x0F; x | y`, int64(0xFF)},
		{"bitwise_xor", `x := 0xFF; y := 0x0F; x ^ y`, int64(0xF0)},
		{"shift_left", `x := 1; y := 4; x << y`, int64(16)},
		{"shift_right", `x := 16; y := 2; x >> y`, int64(4)},
		{"comparison_eq_true", `x := 42; x == 42`, true},
		{"comparison_eq_false", `x := 42; x == 43`, false},
		{"comparison_lt", `x := 5; x < 10`, true},
		{"comparison_gt", `x := 10; x > 5`, true},
		{"boolean_not", `x := true; !x`, false},
		{"conditional", `x := 5; if x > 3 { x = 100 }; x`, int64(100)},
		{"for_loop_sum", `sum := 0; for i := 0; i < 10; i++ { sum += i }; sum`, int64(45)},
		{"string_concat", `x := "hello"; y := " world"; x + y`, "hello world"},
		{"nested_calls", "func fib(n int) int {\n\tif n <= 1 {\n\t\treturn n\n\t}\n\treturn fib(n-1) + fib(n-2)\n}\nfib(10)", int64(55)},
		{"closure_capture", `x := 10; f := func() int { return x + 5 }; f()`, int64(15)},
		{"int_to_float", `x := 42; float64(x)`, float64(42.0)},
		{"float_to_int", `x := 42.7; int(x)`, int64(42)},
		{"multiple_returns", `
			func swap(a, b int) (int, int) { return b, a }
			x, y := swap(1, 2)
			x*10 + y`, int64(21)},
		{"string_length", `x := "hello"; len(x)`, int64(5)},
		{"slice_operations", `
			s := make([]int, 0)
			s = append(s, 1, 2, 3)
			s[0] + s[1] + s[2]`, int64(6)},
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
