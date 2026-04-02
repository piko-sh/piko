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

func TestEvalBreakContinue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "break in for",
			code: `sum := 0
for i := 0; i < 100; i++ {
	if i == 5 {
		break
	}
	sum = sum + i
}
sum`,
			expect: int64(10),
		},
		{
			name: "continue in for",
			code: `sum := 0
for i := 0; i < 10; i++ {
	if i%2 == 0 {
		continue
	}
	sum = sum + i
}
sum`,
			expect: int64(25),
		},
		{
			name: "break in while-style for",
			code: `n := 0
for {
	n = n + 1
	if n >= 5 {
		break
	}
}
n`,
			expect: int64(5),
		},
		{
			name: "continue skips to post",
			code: `sum := 0
for i := 0; i < 10; i++ {
	if i == 3 || i == 7 {
		continue
	}
	sum = sum + i
}
sum`,
			expect: int64(35),
		},
		{
			name: "nested break",
			code: `count := 0
for i := 0; i < 10; i++ {
	for j := 0; j < 10; j++ {
		if j == 3 {
			break
		}
		count = count + 1
	}
}
count`,
			expect: int64(30),
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

func TestEvalSwitch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "switch basic",
			code: `x := 2
result := 0
switch x {
case 1:
	result = 10
case 2:
	result = 20
case 3:
	result = 30
}
result`,
			expect: int64(20),
		},
		{
			name: "switch default",
			code: `x := 99
result := 0
switch x {
case 1:
	result = 10
case 2:
	result = 20
default:
	result = -1
}
result`,
			expect: int64(-1),
		},
		{
			name: "switch no match no default",
			code: `x := 99
result := 42
switch x {
case 1:
	result = 10
case 2:
	result = 20
}
result`,
			expect: int64(42),
		},
		{
			name: "switch string",
			code: `s := "hello"
result := 0
switch s {
case "world":
	result = 1
case "hello":
	result = 2
}
result`,
			expect: int64(2),
		},
		{
			name: "switch true",
			code: `x := 15
result := 0
switch {
case x < 10:
	result = 1
case x < 20:
	result = 2
case x < 30:
	result = 3
}
result`,
			expect: int64(2),
		},
		{
			name: "switch with init",
			code: `result := 0
switch x := 5; x {
case 5:
	result = 50
default:
	result = -1
}
result`,
			expect: int64(50),
		},
		{
			name: "switch multi-value case",
			code: `x := 3
result := 0
switch x {
case 1, 2:
	result = 10
case 3, 4:
	result = 20
default:
	result = 30
}
result`,
			expect: int64(20),
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

func TestEvalIfElseIf(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "else if chain",
			code: `x := 15
result := 0
if x < 10 {
	result = 1
} else if x < 20 {
	result = 2
} else {
	result = 3
}
result`,
			expect: int64(2),
		},
		{
			name: "if with init statement",
			code: `result := 0
if x := 10; x > 5 {
	result = x
}
result`,
			expect: int64(10),
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
