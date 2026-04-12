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

func TestInlining_VoidNoop_NoOpCall(t *testing.T) {
	withInlineEnabled(t, func() {
		service := NewService()
		code := `func noop() {}
noop()
42`
		result, err := service.Eval(context.Background(), code)
		require.NoError(t, err)
		require.Equal(t, int64(42), result)
	})
}

func TestInlining_SmallLeaf_ProducesCorrectResult(t *testing.T) {
	withInlineEnabled(t, func() {
		service := NewService()
		code := `func double(x int) int { return x + x }
double(21)`
		result, err := service.Eval(context.Background(), code)
		require.NoError(t, err)
		require.Equal(t, int64(42), result)
	})
}

func TestInlining_MultipleCallsToSameFunction(t *testing.T) {
	withInlineEnabled(t, func() {
		service := NewService()
		code := `func add(a, b int) int { return a + b }
x := add(1, 2)
y := add(x, 3)
add(y, 4)`
		result, err := service.Eval(context.Background(), code)
		require.NoError(t, err)
		require.Equal(t, int64(10), result)
	})
}

func TestInlining_ConditionalReturns(t *testing.T) {
	withInlineEnabled(t, func() {
		service := NewService()
		code := `func absVal(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
a := absVal(-5)
b := absVal(7)
a + b`
		result, err := service.Eval(context.Background(), code)
		require.NoError(t, err)
		require.Equal(t, int64(12), result)
	})
}

func TestInlining_RecursiveFunctionRefused(t *testing.T) {
	withInlineEnabled(t, func() {
		service := NewService()
		code := `func fact(n int) int {
	if n <= 1 {
		return 1
	}
	return n * fact(n-1)
}
fact(5)`
		result, err := service.Eval(context.Background(), code)
		require.NoError(t, err)
		require.Equal(t, int64(120), result)
	})
}

func TestInlining_ClosureRefused_StillCorrect(t *testing.T) {
	withInlineEnabled(t, func() {
		service := NewService()
		code := `func makeAdder(n int) func(int) int {
	return func(x int) int { return x + n }
}
add5 := makeAdder(5)
add5(10)`
		result, err := service.Eval(context.Background(), code)
		require.NoError(t, err)
		require.Equal(t, int64(15), result)
	})
}

func TestInlining_MultiReturn_SameBank(t *testing.T) {
	withInlineEnabled(t, func() {
		service := NewService()
		code := `func swap(a, b int) (int, int) { return b, a }
x, y := swap(3, 7)
x * 10 + y`
		result, err := service.Eval(context.Background(), code)
		require.NoError(t, err)
		require.Equal(t, int64(73), result)
	})
}

func TestInlining_MultiReturn_MixedBanks(t *testing.T) {
	withInlineEnabled(t, func() {
		service := NewService()
		code := `func split(x int) (int, string) { return x * 2, "ok" }
n, s := split(21)
n + len(s)`
		result, err := service.Eval(context.Background(), code)
		require.NoError(t, err)
		require.Equal(t, int64(44), result)
	})
}

func TestInlining_ValueReturnFromTempRegister(t *testing.T) {
	withInlineEnabled(t, func() {
		service := NewService()

		code := `func f(a int) int {
	x := a + 1
	y := a * 2
	_ = x
	return y
}
f(5)`
		result, err := service.Eval(context.Background(), code)
		require.NoError(t, err)
		require.Equal(t, int64(10), result)
	})
}

func withInlineEnabled(t *testing.T, fn func()) {
	t.Helper()
	fn()
}
