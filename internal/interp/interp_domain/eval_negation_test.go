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

func TestEvalUnaryOperators(t *testing.T) {
	t.Parallel()

	intTests := []struct {
		name   string
		code   string
		expect int64
	}{
		{name: "negate literal", code: `-42`, expect: -42},
		{name: "negate variable", code: `x := 10; -x`, expect: -10},
		{name: "double negate", code: `x := 5; -(-x)`, expect: 5},
		{name: "bitwise not zero", code: `^0`, expect: -1},
		{name: "bitwise not five", code: `^5`, expect: -6},
		{name: "bitwise not variable", code: `x := 3; ^x`, expect: -4},
	}

	for _, tt := range intTests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}

	floatTests := []struct {
		name   string
		code   string
		expect float64
	}{
		{name: "negate float literal", code: `-3.14`, expect: -3.14},
		{name: "negate float variable", code: `x := 2.5; -x`, expect: -2.5},
		{name: "double negate float", code: `x := 1.0; -(-x)`, expect: 1.0},
	}

	for _, tt := range floatTests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.InDelta(t, tt.expect, result, 0.0001)
		})
	}

	boolTests := []struct {
		expect any
		name   string
		code   string
	}{
		{name: "not true", code: `!true`, expect: false},
		{name: "not false", code: `!false`, expect: true},
		{name: "not expression", code: `!(1 > 2)`, expect: true},
		{name: "double not", code: `!!true`, expect: true},
		{name: "not equality", code: `!(3 == 3)`, expect: false},
	}

	for _, tt := range boolTests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}
