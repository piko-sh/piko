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

func TestEvalInit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "init_calls_function",
			code: `func add(a, b int) int { return a + b }
func init() {
	_ = add(20, 22)
}
add(1, 2)`,
			expect: int64(3),
		},
		{
			name: "init_executes_before_eval",
			code: `func helper() int { return 42 }
func init() {
	_ = helper()
}
helper()`,
			expect: int64(42),
		},
		{
			name: "init_modifies_package_var",
			code: `var count int
func init() { count = 42 }
count`,
			expect: int64(42),
		},
		{
			name: "init_multiple_ordered",
			code: `var result string
func init() { result = "a" }
func init() { result += "b" }
func init() { result += "c" }
result`,
			expect: "abc",
		},
		{
			name: "init_registers_to_map",
			code: `var registry = make(map[string]int)
func init() {
	registry["alpha"] = 1
	registry["beta"] = 2
}
registry["alpha"] + registry["beta"]`,
			expect: int64(3),
		},
		{
			name: "init_calls_closure",
			code: `var value int
func init() {
	f := func(x int) int { return x * 3 }
	value = f(7)
}
value`,
			expect: int64(21),
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
