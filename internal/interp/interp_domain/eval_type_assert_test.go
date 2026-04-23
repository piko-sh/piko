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

func TestEvalTypeSwitch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "type switch int",
			code: `var x interface{} = 42
result := 0
switch v := x.(type) {
case int:
	result = v + 1
case string:
	result = -1
}
result`,
			expect: int64(43),
		},
		{
			name: "type switch string",
			code: `var x interface{} = "hello"
result := ""
switch v := x.(type) {
case int:
	result = "wrong"
case string:
	result = v + " world"
}
result`,
			expect: "hello world",
		},
		{
			name: "type switch default",
			code: `var x interface{} = 3.14
result := "unknown"
switch x.(type) {
case int:
	result = "int"
case string:
	result = "string"
default:
	result = "other"
}
result`,
			expect: "other",
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

func TestEvalTypeAssertion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "assert interface to int",
			code: `var x interface{} = 42
v := x.(int)
v`,
			expect: int64(42),
		},
		{
			name: "assert interface to string",
			code: `var x interface{} = "hello"
v := x.(string)
v`,
			expect: "hello",
		},
		{
			name: "assert interface to bool then branch",
			code: `var x interface{} = true
result := "no"
if x.(bool) {
	result = "yes"
}
result`,
			expect: "yes",
		},
		{
			name: "assert interface to bool false then branch",
			code: `var x interface{} = false
result := "no"
if x.(bool) {
	result = "yes"
}
result`,
			expect: "no",
		},
		{
			name: "assert function result to bool as if condition",
			code: `pick := func(v interface{}) interface{} { return v }
result := "no"
if pick(true).(bool) {
	result = "yes"
}
result`,
			expect: "yes",
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
