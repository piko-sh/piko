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

func TestEvalClosureUpvalueInt(t *testing.T) {
	t.Parallel()

	service := NewService()
	result, err := service.Eval(context.Background(), `x := 0; inc := func() { x++ }; inc(); inc(); inc(); x`)
	require.NoError(t, err)
	require.Equal(t, int64(3), result)
}

func TestEvalClosureUpvalueFloat(t *testing.T) {
	t.Parallel()

	service := NewService()
	result, err := service.Eval(context.Background(), `x := 0.0; add := func() { x += 1.5 }; add(); add(); x`)
	require.NoError(t, err)
	require.Equal(t, 3.0, result)
}

func TestEvalClosureUpvalueString(t *testing.T) {
	t.Parallel()

	service := NewService()
	result, err := service.Eval(context.Background(), `s := ""; app := func(w string) { s += w }; app("hello"); app(" world"); s`)
	require.NoError(t, err)
	require.Equal(t, "hello world", result)
}

func TestEvalClosureSharedUpvalue(t *testing.T) {
	t.Parallel()

	service := NewService()
	result, err := service.Eval(context.Background(), `x := 0
inc := func() { x++ }
get := func() int { return x }
inc()
inc()
get()`)
	require.NoError(t, err)
	require.Equal(t, int64(2), result)
}

func TestEvalClosureCounter(t *testing.T) {
	t.Parallel()

	service := NewService()
	result, err := service.Eval(context.Background(), `func makeCounter() func() int {
    n := 0
    return func() int { n++; return n }
}
c := makeCounter()
c()
c()
c()`)
	require.NoError(t, err)
	require.Equal(t, int64(3), result)
}

func TestEvalClosureNested(t *testing.T) {
	t.Parallel()

	service := NewService()
	result, err := service.Eval(context.Background(), `func outer() func() func() int {
    x := 10
    return func() func() int {
        return func() int { return x }
    }
}
outer()()()`)
	require.NoError(t, err)
	require.Equal(t, int64(10), result)
}
