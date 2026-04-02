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

func TestEvalGenericArithmetic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "named integer type addition",
			code: `type MyInt int
var a MyInt = 5
var b MyInt = 3
int(a + b)`,
			expect: int64(8),
		},
		{
			name: "named integer type subtraction",
			code: `type MyInt int
var a MyInt = 10
var b MyInt = 4
int(a - b)`,
			expect: int64(6),
		},
		{
			name: "named integer type multiplication",
			code: `type MyInt int
var a MyInt = 3
var b MyInt = 7
int(a * b)`,
			expect: int64(21),
		},
		{
			name: "named integer type division",
			code: `type MyInt int
var a MyInt = 20
var b MyInt = 4
int(a / b)`,
			expect: int64(5),
		},
		{
			name: "named integer type modulo",
			code: `type MyInt int
var a MyInt = 17
var b MyInt = 5
int(a % b)`,
			expect: int64(2),
		},
		{
			name: "named float type addition",
			code: `type MyFloat float64
var a MyFloat = 2.5
var b MyFloat = 1.5
float64(a + b)`,
			expect: 4.0,
		},
		{
			name: "named string type concatenation",
			code: `type MyString string
var a MyString = "hello"
var b MyString = " world"
string(a + b)`,
			expect: "hello world",
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
