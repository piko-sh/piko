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
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHotReloadRegisteredClosure(t *testing.T) {
	t.Parallel()

	type handlerStore struct {
		mu       sync.Mutex
		handlers map[string]func() int
	}
	store := &handlerStore{handlers: map[string]func() int{}}

	provider := &mockSymbolProvider{
		exports: SymbolExports{
			"hot": {
				"Register": reflect.ValueOf(func(name string, fn func() int) int {
					store.mu.Lock()
					defer store.mu.Unlock()
					store.handlers[name] = fn
					return 0
				}),
				"Invoke": reflect.ValueOf(func(name string) int {
					store.mu.Lock()
					fn := store.handlers[name]
					store.mu.Unlock()
					if fn == nil {
						return -1
					}
					return fn()
				}),
			},
		},
	}

	newService := func() *Service {
		s := NewService()
		s.UseSymbolProviders(provider)
		return s
	}

	firstSources := map[string]map[string]string{
		"": {
			"main.go": `package main

import "hot"

var sink = hot.Register("x", func() int { return 42 })

func main() {}
`,
		},
	}

	service := newService()
	cfs1, err := service.CompileProgram(t.Context(), "testmod", firstSources)
	require.NoError(t, err)
	require.NoError(t, service.ExecuteInits(t.Context(), cfs1))

	firstResult, err := callInvoke(provider, "x")
	require.NoError(t, err)
	require.Equal(t, 42, firstResult, "first-version closure should return 42")

	secondSources := map[string]map[string]string{
		"": {
			"main.go": `package main

import "hot"

var sink = hot.Register("x", func() int {
	helper := func() int { return 7 }
	return helper() + 93
})

func main() {}
`,
		},
	}

	cfs2, err := service.CompileProgram(t.Context(), "testmod", secondSources)
	require.NoError(t, err)
	require.NoError(t, service.ExecuteInits(t.Context(), cfs2))

	secondResult, err := callInvoke(provider, "x")
	require.NoError(t, err, "post-hot-reload invocation must not crash")
	require.Equal(t, 100, secondResult, "second-version closure should return 100")
}

func callInvoke(provider *mockSymbolProvider, name string) (result int, err error) {
	defer func() {
		if r := recover(); r != nil {
			result = 0
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	fn := provider.exports["hot"]["Invoke"]
	out := fn.Call([]reflect.Value{reflect.ValueOf(name)})
	return int(out[0].Int()), nil
}
