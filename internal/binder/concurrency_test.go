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

package binder

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConcurrent_RegisterConverterDuringBind(t *testing.T) {
	binder := NewASTBinder()

	var wg sync.WaitGroup

	for range 10 {
		wg.Go(func() {
			for range 100 {
				var form SimpleForm
				src := map[string][]string{
					"Name": {"Alice"},
					"Age":  {"30"},
				}
				_ = binder.Bind(context.Background(), &form, src)
			}
		})
	}

	for i := range 10 {
		index := i
		wg.Go(func() {
			type CustomType struct {
				Val int
			}

			binder.RegisterConverter(reflect.TypeFor[CustomType](), func(value string) (reflect.Value, error) {
				return reflect.ValueOf(CustomType{Val: index}), nil
			})
		})
	}

	wg.Wait()

	var form SimpleForm
	src := map[string][]string{"Name": {"Final"}}
	err := binder.Bind(context.Background(), &form, src)
	require.NoError(t, err)
	assert.Equal(t, "Final", form.Name)
}

func TestConcurrent_DistinctTypeStampede(t *testing.T) {
	binder := NewASTBinder()

	type Payload[T any] struct {
		Value T
	}

	var wg sync.WaitGroup

	bind := func(dst any) {
		src := map[string][]string{
			"Value": {"42"},
		}
		_ = binder.Bind(context.Background(), dst, src)
	}

	wg.Go(func() { var f Payload[int]; bind(&f); assert.Equal(t, 42, f.Value) })
	wg.Go(func() { var f Payload[int8]; bind(&f); assert.Equal(t, int8(42), f.Value) })
	wg.Go(func() { var f Payload[int16]; bind(&f); assert.Equal(t, int16(42), f.Value) })
	wg.Go(func() { var f Payload[int32]; bind(&f); assert.Equal(t, int32(42), f.Value) })
	wg.Go(func() { var f Payload[int64]; bind(&f); assert.Equal(t, int64(42), f.Value) })
	wg.Go(func() { var f Payload[uint]; bind(&f); assert.Equal(t, uint(42), f.Value) })
	wg.Go(func() { var f Payload[uint8]; bind(&f); assert.Equal(t, uint8(42), f.Value) })
	wg.Go(func() { var f Payload[uint16]; bind(&f); assert.Equal(t, uint16(42), f.Value) })
	wg.Go(func() { var f Payload[uint32]; bind(&f); assert.Equal(t, uint32(42), f.Value) })
	wg.Go(func() { var f Payload[uint64]; bind(&f); assert.Equal(t, uint64(42), f.Value) })
	wg.Go(func() { var f Payload[string]; bind(&f); assert.Equal(t, "42", f.Value) })
	wg.Go(func() { var f Payload[bool]; bind(&f) })
	wg.Go(func() { var f Payload[float32]; bind(&f); assert.Equal(t, float32(42), f.Value) })
	wg.Go(func() { var f Payload[float64]; bind(&f); assert.Equal(t, float64(42), f.Value) })

	wg.Wait()

	var f Payload[int]
	src := map[string][]string{"Value": {"99"}}
	err := binder.Bind(context.Background(), &f, src)
	require.NoError(t, err)
	assert.Equal(t, 99, f.Value)
}

func TestConcurrent_BindWithAllConfigMutation(t *testing.T) {
	binder := NewASTBinder()

	var wg sync.WaitGroup

	for range 10 {
		wg.Go(func() {
			for range 200 {
				var form SimpleForm
				src := map[string][]string{
					"Name": {"Test"},
				}
				_ = binder.Bind(context.Background(), &form, src)
			}
		})
	}

	for i := range 5 {
		value := i
		wg.Go(func() {
			for range 200 {
				binder.SetMaxSliceSize(value*10 + 50)
				binder.SetMaxPathDepth(value*2 + 5)
				binder.SetMaxPathLength(value*100 + 200)
				binder.SetMaxFieldCount(value*50 + 100)
				binder.SetMaxValueLength(value*100 + 500)
				binder.SetIgnoreUnknownKeys(value%2 == 0)
			}
		})
	}

	wg.Wait()

	var form SimpleForm
	binder.SetMaxFieldCount(1000)
	src := map[string][]string{"Name": {"Final"}}
	err := binder.Bind(context.Background(), &form, src)
	require.NoError(t, err)
	assert.Equal(t, "Final", form.Name)
}

func TestConcurrent_GetSlowDuplicateBuild(t *testing.T) {
	binder := NewASTBinder()

	type UniqueFormForDedup struct {
		Name  string
		Email string
		Age   int
	}

	const goroutines = 100
	results := make([]*structInfo, goroutines)
	var wg sync.WaitGroup

	for i := range goroutines {
		index := i
		wg.Go(func() {
			info := binder.cache.get(reflect.TypeFor[UniqueFormForDedup](), 10)
			results[index] = info
		})
	}

	wg.Wait()

	first := results[0]
	require.NotNil(t, first)
	for i := 1; i < goroutines; i++ {
		assert.Same(t, first, results[i],
			"goroutine %d got a different *structInfo pointer", i)
	}
}

func TestConcurrent_FastMapConsistencyAfterBurst(t *testing.T) {
	binder := NewASTBinder()

	var wg sync.WaitGroup

	type A struct{ V string }
	type B struct{ V string }
	type C struct{ V string }
	type D struct{ V string }
	type E struct{ V string }
	type F struct{ V string }
	type G struct{ V string }
	type H struct{ V string }
	type I struct{ V string }
	type J struct{ V string }

	src := map[string][]string{"V": {"ok"}}

	bindN := func(newDst func() any) {
		for range 5 {
			wg.Go(func() {
				_ = binder.Bind(context.Background(), newDst(), src)
			})
		}
	}

	bindN(func() any { return &A{} })
	bindN(func() any { return &B{} })
	bindN(func() any { return &C{} })
	bindN(func() any { return &D{} })
	bindN(func() any { return &E{} })
	bindN(func() any { return &F{} })
	bindN(func() any { return &G{} })
	bindN(func() any { return &H{} })
	bindN(func() any { return &I{} })
	bindN(func() any { return &J{} })

	wg.Wait()

	verifyTypes := []any{&A{}, &B{}, &C{}, &D{}, &E{}, &F{}, &G{}, &H{}, &I{}, &J{}}
	for i, dst := range verifyTypes {
		err := binder.Bind(context.Background(), dst, src)
		require.NoError(t, err, "type %d should bind successfully after burst", i)
	}
}

func TestConcurrent_PerCallOptionsAreSafe(t *testing.T) {
	binder := NewASTBinder()
	binder.SetMaxFieldCount(1000)

	var wg sync.WaitGroup

	for range 20 {
		wg.Go(func() {
			for range 100 {
				var form SimpleForm
				src := map[string][]string{
					"Name": {"Alice"},
				}

				err := binder.Bind(context.Background(), &form, src,
					WithMaxFieldCount(50),
					WithMaxValueLength(500),
				)
				require.NoError(t, err)
				assert.Equal(t, "Alice", form.Name)
			}
		})
	}

	for range 20 {
		wg.Go(func() {
			for range 100 {
				var form SimpleForm
				src := map[string][]string{
					"Name": {"Bob"},
				}
				err := binder.Bind(context.Background(), &form, src)
				require.NoError(t, err)
				assert.Equal(t, "Bob", form.Name)
			}
		})
	}

	wg.Wait()
}

func TestConcurrent_ASTParseCacheConsistency(t *testing.T) {
	binder := NewASTBinder()

	paths := []string{
		"User.Name",
		"Items[0].Price",
		`Config["key"]`,
		"Profile.Email",
		"Tags[1]",
	}

	var wg sync.WaitGroup

	for range 50 {
		wg.Go(func() {
			for _, path := range paths {
				var form NestedForm
				src := map[string][]string{
					path: {"value"},
				}

				_ = binder.Bind(context.Background(), &form, src)
			}
		})
	}

	wg.Wait()

	var form NestedForm
	src := map[string][]string{"User.Name": {"Final"}}
	err := binder.Bind(context.Background(), &form, src)
	require.NoError(t, err)
	assert.Equal(t, "Final", form.User.Name)
}

func TestConcurrent_MapBindingThreadSafety(t *testing.T) {
	binder := NewASTBinder()

	var wg sync.WaitGroup

	for i := range 20 {
		index := i
		wg.Go(func() {
			type MapForm struct {
				Data map[string]string
			}

			form := MapForm{Data: make(map[string]string)}
			src := map[string][]string{
				fmt.Sprintf(`Data["key_%d"]`, index): {fmt.Sprintf("val_%d", index)},
			}

			err := binder.Bind(context.Background(), &form, src)
			require.NoError(t, err)
			assert.Equal(t, fmt.Sprintf("val_%d", index), form.Data[fmt.Sprintf("key_%d", index)])
		})
	}

	wg.Wait()
}
