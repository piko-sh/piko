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
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

type mockBytecodeStore struct {
	savedKey string
	savedCFS *CompiledFileSet
	loadCFS  *CompiledFileSet
	loadErr  error
	saveErr  error
}

func (m *mockBytecodeStore) SaveCompiledFileSet(_ context.Context, key string, cfs *CompiledFileSet) error {
	m.savedKey = key
	m.savedCFS = cfs
	return m.saveErr
}

func (m *mockBytecodeStore) LoadCompiledFileSet(_ context.Context, _ string, _ *SymbolRegistry) (*CompiledFileSet, error) {
	if m.loadErr != nil {
		return nil, m.loadErr
	}
	return m.loadCFS, nil
}

func TestServiceBytecode(t *testing.T) {
	t.Parallel()

	t.Run("SaveCompiled", func(t *testing.T) {
		t.Parallel()

		t.Run("returns_error_when_no_store_configured", func(t *testing.T) {
			t.Parallel()

			service := newTestService(t)
			cfs := compileFileSource(t, `package main
func main() {}`)

			err := service.SaveCompiled(context.Background(), "test-key", cfs)
			require.Error(t, err)
			require.ErrorContains(t, err, "no bytecode store configured")
		})

		t.Run("delegates_to_store_on_success", func(t *testing.T) {
			t.Parallel()

			store := &mockBytecodeStore{}
			service := newTestService(t, WithBytecodeStore(store))
			cfs := compileFileSource(t, `package main
func main() {}`)

			err := service.SaveCompiled(context.Background(), "my-key", cfs)
			require.NoError(t, err)
			require.Equal(t, "my-key", store.savedKey)
			require.Same(t, cfs, store.savedCFS)
		})

		t.Run("propagates_store_error", func(t *testing.T) {
			t.Parallel()

			storeErr := errors.New("disk full")
			store := &mockBytecodeStore{saveErr: storeErr}
			service := newTestService(t, WithBytecodeStore(store))
			cfs := compileFileSource(t, `package main
func main() {}`)

			err := service.SaveCompiled(context.Background(), "key", cfs)
			require.ErrorIs(t, err, storeErr)
		})
	})

	t.Run("LoadCompiled", func(t *testing.T) {
		t.Parallel()

		t.Run("returns_error_when_no_store_configured", func(t *testing.T) {
			t.Parallel()

			service := newTestService(t)

			cfs, err := service.LoadCompiled(context.Background(), "test-key")
			require.Error(t, err)
			require.Nil(t, cfs)
			require.ErrorContains(t, err, "no bytecode store configured")
		})

		t.Run("returns_compiled_file_set_on_success", func(t *testing.T) {
			t.Parallel()

			expected := compileFileSource(t, `package main
func main() {}`)
			store := &mockBytecodeStore{loadCFS: expected}
			service := newTestService(t, WithBytecodeStore(store))

			cfs, err := service.LoadCompiled(context.Background(), "my-key")
			require.NoError(t, err)
			require.Same(t, expected, cfs)
		})

		t.Run("wraps_and_returns_store_error", func(t *testing.T) {
			t.Parallel()

			underlying := errors.New("corrupt data")
			store := &mockBytecodeStore{loadErr: underlying}
			service := newTestService(t, WithBytecodeStore(store))

			cfs, err := service.LoadCompiled(context.Background(), "bad-key")
			require.Error(t, err)
			require.Nil(t, cfs)
			require.ErrorIs(t, err, underlying)
			require.ErrorContains(t, err, "loading compiled bytecode")
			require.ErrorContains(t, err, "bad-key")
		})
	})
}

func TestRegisterPackage(t *testing.T) {
	t.Parallel()

	service := NewService()
	service.RegisterPackage("custom/pkg", map[string]reflect.Value{
		"Hello": reflect.ValueOf(func() string { return "world" }),
	})

	source := `package main

import "custom/pkg"

func run() string {
	return pkg.Hello()
}

func main() {}
`
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, "world", result)
}

func TestWithDebugInfoOption(t *testing.T) {
	t.Parallel()

	service := newTestService(t, WithDebugInfo())
	cfs, err := service.CompileFileSet(context.Background(), map[string]string{
		"main.go": `package main
func main() { x := 1; _ = x }`,
	})
	require.NoError(t, err)

	fn, err := cfs.FindFunction("main")
	require.NoError(t, err)
	require.True(t, fn.HasDebugSourceMap())
	require.True(t, fn.HasDebugVarTable())
}
