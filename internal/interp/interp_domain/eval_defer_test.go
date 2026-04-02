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

func TestEvalDefer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "defer modifies slice",
			code: `func run(s []int) {
	defer func() {
		s[0] = 42
	}()
	s[0] = 1
}
s := make([]int, 1)
run(s)
s[0]`,
			expect: int64(42),
		},
		{
			name: "multiple defers LIFO",
			code: `func run(s []int) {
	defer func() {
		s[0] = s[0] * 10
	}()
	defer func() {
		s[0] = s[0] + 5
	}()
	s[0] = 1
}
s := make([]int, 1)
run(s)
s[0]`,
			expect: int64(60),
		},
		{
			name: "defer named function",
			code: `func reset(s []int) {
	s[0] = 99
}
func run(s []int) {
	s[0] = 1
	defer reset(s)
	s[0] = 2
}
s := make([]int, 1)
run(s)
s[0]`,
			expect: int64(99),
		},
		{
			name: "defer with arguments evaluated eagerly",
			code: `func run() int {
	s := make([]int, 1)
	s[0] = 0
	x := 10
	defer func(value int) {
		s[0] = value
	}(x)
	x = 20
	return s[0]
}
run()`,
			expect: int64(0),
		},
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

func TestEvalPanic(t *testing.T) {
	t.Parallel()

	t.Run("unrecovered panic returns error", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		_, err := service.Eval(context.Background(), `panic("boom")`)
		require.Error(t, err)
		require.Contains(t, err.Error(), "boom")
	})
}
