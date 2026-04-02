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
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvalPanicRecover(t *testing.T) {
	t.Parallel()

	t.Run("panic with string returns error", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		_, err := service.Eval(context.Background(), `panic("test error")`)
		require.Error(t, err)
		require.Contains(t, err.Error(), "test error")
	})

	t.Run("panic with int returns error", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		_, err := service.Eval(context.Background(), `panic(42)`)
		require.Error(t, err)
	})

	t.Run("panic in nested call returns error", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		_, err := service.Eval(context.Background(), `
func inner() { panic("deep") }
func outer() string { inner(); return "" }
outer()`)
		require.Error(t, err)
		require.Contains(t, err.Error(), "deep")
	})
}

func TestEvalRecover(t *testing.T) {
	t.Parallel()

	t.Run("recover in defer captures panic value", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		result, err := service.Eval(context.Background(), `
func run() string {
	var message string
	func() {
		defer func() {
			r := recover()
			if r != nil {
				message = "caught"
			}
		}()
		panic("boom")
	}()
	return message
}
run()`)
		require.NoError(t, err)
		require.Equal(t, "caught", result)
	})

	t.Run("recover returns nil when no panic", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		result, err := service.Eval(context.Background(), `
func run() any {
	defer func() { recover() }()
	return recover()
}
run()`)
		require.NoError(t, err)
		require.Nil(t, result)
	})

	t.Run("defers run on panic in LIFO order", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		result, err := service.Eval(context.Background(), `
func run() string {
	var s string
	func() {
		defer func() { s = s + ",first" }()
		defer func() { s = s + ",second" }()
		defer func() {
			recover()
			s = s + "third"
		}()
		panic("boom")
	}()
	return s
}
run()`)
		require.NoError(t, err)
		require.Equal(t, "third,second,first", result)
	})

	t.Run("panic propagation through call stack", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		result, err := service.Eval(context.Background(), `
func deep() { panic("from deep") }
func middle() { deep() }
func top() string {
	defer func() { recover() }()
	middle()
	return "unreachable"
}
top()`)
		require.NoError(t, err)
		require.Equal(t, "", result)
	})
}

func TestEvalPanicNil(t *testing.T) {
	t.Parallel()

	t.Run("panic nil yields PanicNilError", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		result, err := service.Eval(context.Background(), `
func run() any {
	defer func() {}()
	var result any
	func() {
		defer func() { result = recover() }()
		panic(nil)
	}()
	return result
}
run()`)
		require.NoError(t, err)
		require.NotNil(t, result, "recover() from panic(nil) must not be nil")
		require.IsType(t, (*runtime.PanicNilError)(nil), result)
	})

	t.Run("unrecovered panic nil returns error", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		_, err := service.Eval(context.Background(), `panic(nil)`)
		require.Error(t, err)
	})

	t.Run("panic with non-nil value is not PanicNilError", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		result, err := service.Eval(context.Background(), `
func run() any {
	var result any
	func() {
		defer func() { result = recover() }()
		panic("not nil")
	}()
	return result
}
run()`)
		require.NoError(t, err)
		require.Equal(t, "not nil", result)
		_, isPNE := result.(*runtime.PanicNilError)
		require.False(t, isPNE, "non-nil panic should not be PanicNilError")
	})
}
