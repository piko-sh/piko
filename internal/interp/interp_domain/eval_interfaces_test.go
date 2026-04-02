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

func TestEvalInterfaceBoxing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name:   "box and unbox int",
			code:   "func wrap(x any) any { return x }\nwrap(42)",
			expect: int64(42),
		},
		{
			name:   "box and unbox float",
			code:   "func wrap(x any) any { return x }\ny := 3.14\nwrap(y)",
			expect: 3.14,
		},
		{
			name:   "box and unbox bool",
			code:   "func wrap(x any) any { return x }\nb := true\nwrap(b)",
			expect: true,
		},
		{
			name:   "box and unbox string",
			code:   "func wrap(x any) any { return x }\ns := \"hello\"\nwrap(s)",
			expect: "hello",
		},
		{
			name:   "box and unbox uint",
			code:   "func wrap(x any) any { return x }\nvar u uint = 42\nwrap(u)",
			expect: uint64(42),
		},
		{
			name: "unbox to typed int",
			code: `func box() any { return 42 }
func use() int { v := box(); return v.(int) }
use()`,
			expect: int64(42),
		},
		{
			name: "unbox to typed string",
			code: `func box() any { return "hello" }
func use() string { v := box(); return v.(string) }
use()`,
			expect: "hello",
		},
		{
			name: "interface function param float add",
			code: `func add(a any, b any) float64 {
	return a.(float64) + b.(float64)
}
x := 1.5
y := 2.5
add(x, y)`,
			expect: 4.0,
		},
		{
			name: "interface function param bool",
			code: `func negate(x any) bool {
	return !x.(bool)
}
b := true
negate(b)`,
			expect: false,
		},
		{
			name: "type assertion comma ok success",
			code: `func check(x any) bool {
	_, ok := x.(int)
	return ok
}
check(42)`,
			expect: true,
		},
		{
			name: "type assertion comma ok fail",
			code: `func check(x any) bool {
	_, ok := x.(string)
	return ok
}
check(42)`,
			expect: false,
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
