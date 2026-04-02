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

func TestEvalStringOperationsExtended(t *testing.T) {
	t.Parallel()

	intTests := []struct {
		name   string
		code   string
		expect int64
	}{
		{name: "len hello", code: `len("hello")`, expect: 5},
		{name: "len empty", code: `len("")`, expect: 0},
		{name: "len var", code: `s := "test"; len(s)`, expect: 4},
		{name: "index first byte", code: `s := "hello"; int(s[0])`, expect: 104},
		{name: "index last byte", code: `s := "hello"; int(s[4])`, expect: 111},
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

	stringTests := []struct {
		name   string
		code   string
		expect string
	}{
		{name: "slice", code: `"hello"[1:3]`, expect: "el"},
		{name: "slice from start", code: `"hello"[0:2]`, expect: "he"},
		{name: "slice to end", code: `s := "hello"; s[3:]`, expect: "lo"},
		{name: "concat assign", code: `s := "hello"; s += " world"; s`, expect: "hello world"},
		{name: "repeated concat", code: `s := ""; for i := 0; i < 3; i++ { s += "a" }; s`, expect: "aaa"},
	}

	for _, tt := range stringTests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}

	boolTests := []struct {
		expect any
		name   string
		code   string
	}{
		{name: "eq true", code: `"abc" == "abc"`, expect: true},
		{name: "eq false", code: `"abc" == "abd"`, expect: false},
		{name: "ne true", code: `"abc" != "abd"`, expect: true},
		{name: "lt", code: `"abc" < "abd"`, expect: true},
		{name: "le equal", code: `"abc" <= "abc"`, expect: true},
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

func TestEvalStringRange(t *testing.T) {
	t.Parallel()

	service := NewService()
	result, err := service.Eval(context.Background(), `s := "hello"
n := 0
for range s {
	n = n + 1
}
n`)
	require.NoError(t, err)
	require.Equal(t, int64(5), result)
}
