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

func TestEvalRangeOverFunc(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "basic 1-value iterator",
			code: `
iter := func(yield func(int) bool) {
	for i := 0; i < 5; i++ {
		if !yield(i) { return }
	}
}
sum := 0
for v := range iter { sum += v }
sum`,
			expect: int64(10),
		},
		{
			name: "2-value iterator",
			code: `
iter := func(yield func(string, int) bool) {
	yield("a", 1)
	yield("b", 2)
}
total := 0
for _, v := range iter { total += v }
total`,
			expect: int64(3),
		},
		{
			name: "break stops iteration",
			code: `
iter := func(yield func(int) bool) {
	for i := 0; i < 10; i++ {
		if !yield(i) { return }
	}
}
sum := 0
for v := range iter {
	if v >= 3 { break }
	sum += v
}
sum`,
			expect: int64(3),
		},
		{
			name: "continue skips iteration",
			code: `
iter := func(yield func(int) bool) {
	for i := 0; i < 5; i++ {
		if !yield(i) { return }
	}
}
sum := 0
for v := range iter {
	if v%2 == 0 { continue }
	sum += v
}
sum`,
			expect: int64(4),
		},
		{
			name: "0-value yield",
			code: `
iter := func(yield func() bool) {
	yield()
	yield()
	yield()
}
count := 0
for range iter { count++ }
count`,
			expect: int64(3),
		},
		{
			name: "nested range-over-func",
			code: `
iter := func(yield func(int) bool) {
	for i := 0; i < 3; i++ {
		if !yield(i) { return }
	}
}
sum := 0
for a := range iter {
	for b := range iter {
		sum += a*10 + b
	}
}
sum`,
			expect: int64(99),
		},
		{
			name: "return from inside",
			code: `
func find(target int) int {
	iter := func(yield func(int) bool) {
		for i := 0; i < 10; i++ {
			if !yield(i) { return }
		}
	}
	for v := range iter {
		if v == target { return v * 10 }
	}
	return -1
}
find(3)`,
			expect: int64(30),
		},
		{
			name: "iterator that ignores yield result",
			code: `
iter := func(yield func(int) bool) {
	yield(1)
	yield(2)
	yield(3)
}
sum := 0
for v := range iter { sum += v }
sum`,
			expect: int64(6),
		},
		{
			name: "range key only with 2-param yield",
			code: `
iter := func(yield func(int, string) bool) {
	yield(0, "a")
	yield(1, "b")
	yield(2, "c")
}
last := 0
for k := range iter { last = k }
last`,
			expect: int64(2),
		},
		{
			name: "labelled break to outer for loop",
			code: `
iter := func(yield func(int) bool) {
	for i := 0; i < 5; i++ {
		if !yield(i) { return }
	}
}
sum := 0
outer:
for i := 0; i < 3; i++ {
	for v := range iter {
		if v == 2 { break outer }
		sum += v
	}
}
sum`,
			expect: int64(1),
		},
		{
			name: "labelled continue to outer for loop",
			code: `
iter := func(yield func(int) bool) {
	for i := 0; i < 5; i++ {
		if !yield(i) { return }
	}
}
sum := 0
outer:
for i := 0; i < 3; i++ {
	for v := range iter {
		if v == 1 { continue outer }
		sum += v
	}
}
sum`,
			expect: int64(0),
		},
		{
			name: "labelled break with inner regular loop",
			code: `
iter := func(yield func(int) bool) {
	for i := 0; i < 3; i++ {
		if !yield(i) { return }
	}
}
result := ""
outer:
for i := 0; i < 3; i++ {
	for j := 0; j < 3; j++ {
		for v := range iter {
			if v == 1 { break outer }
			result += "x"
		}
	}
}
result`,
			expect: "x",
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
