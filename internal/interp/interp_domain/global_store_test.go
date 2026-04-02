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
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGlobalStoreInt(t *testing.T) {
	t.Parallel()
	g := newGlobalStore()
	idx0 := g.allocInt(42)
	idx1 := g.allocInt(99)
	require.Equal(t, 0, idx0)
	require.Equal(t, 1, idx1)
	require.Equal(t, int64(42), g.getInt(0))
	require.Equal(t, int64(99), g.getInt(1))
	g.setInt(0, 100)
	require.Equal(t, int64(100), g.getInt(0))
}

func TestGlobalStoreFloat(t *testing.T) {
	t.Parallel()
	g := newGlobalStore()
	index := g.allocFloat(3.14)
	require.Equal(t, 0, index)
	require.InDelta(t, 3.14, g.getFloat(0), 0.0001)
	g.setFloat(0, 2.72)
	require.InDelta(t, 2.72, g.getFloat(0), 0.0001)
}

func TestGlobalStoreString(t *testing.T) {
	t.Parallel()
	g := newGlobalStore()
	index := g.allocString("hello")
	require.Equal(t, 0, index)
	require.Equal(t, "hello", g.getString(0))
	g.setString(0, "world")
	require.Equal(t, "world", g.getString(0))
}

func TestGlobalStoreGeneral(t *testing.T) {
	t.Parallel()
	g := newGlobalStore()
	v := reflect.ValueOf([]int{1, 2, 3})
	index := g.allocGeneral(v)
	require.Equal(t, 0, index)
	got := g.getGeneral(0)
	require.Equal(t, 3, got.Len())
	g.setGeneral(0, reflect.ValueOf("replaced"))
	require.Equal(t, "replaced", g.getGeneral(0).Interface())
}

func TestGlobalStoreReset(t *testing.T) {
	t.Parallel()
	g := newGlobalStore()
	g.allocInt(1)
	g.allocFloat(1.0)
	g.allocString("x")
	g.allocGeneral(reflect.ValueOf(true))
	g.reset()
	index := g.allocInt(2)
	require.Equal(t, 0, index)
	require.Equal(t, int64(2), g.getInt(0))
}

func TestGlobalStoreMultipleAllocs(t *testing.T) {
	t.Parallel()
	g := newGlobalStore()
	for i := range 10 {
		index := g.allocInt(int64(i))
		require.Equal(t, i, index)
	}
	for i := range 10 {
		require.Equal(t, int64(i), g.getInt(i))
	}
}

func TestGlobalStoreIsolationBetweenServices(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc1 := NewService()
	svc2 := NewService()

	_, err := svc1.Eval(ctx, `var x int = 42; x`)
	require.NoError(t, err)

	_, err = svc2.Eval(ctx, `x`)
	require.Error(t, err, "svc2 should not have access to svc1's global x")
}

func TestGlobalsPersistInStore(t *testing.T) {
	t.Parallel()

	g := newGlobalStore()

	index := g.allocInt(10)
	require.Equal(t, int64(10), g.getInt(index))

	g.setInt(index, 15)
	require.Equal(t, int64(15), g.getInt(index))

	idx2 := g.allocString("hello")
	g.setInt(index, 42)
	require.Equal(t, int64(42), g.getInt(index))
	require.Equal(t, "hello", g.getString(idx2))
}

func TestServiceResetClearsGlobals(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service := NewService()

	_, err := service.Eval(ctx, `var x int = 42; x`)
	require.NoError(t, err)

	service.Reset()

	_, err = service.Eval(ctx, `x`)
	require.Error(t, err, "global x should not exist after Reset()")
}

func TestServiceCloneIndependentGlobals(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service := NewService()

	result, err := service.Eval(ctx, "1 + 1")
	require.NoError(t, err)
	require.Equal(t, int64(2), result)

	cloned := service.Clone()

	r1, err := service.Eval(ctx, "10 + 20")
	require.NoError(t, err)
	require.Equal(t, int64(30), r1)

	r2, err := cloned.Eval(ctx, "30 + 40")
	require.NoError(t, err)
	require.Equal(t, int64(70), r2)
}

func TestGlobalStoreResetAndReuse(t *testing.T) {
	t.Parallel()

	g := newGlobalStore()

	g.allocInt(1)
	g.allocInt(2)
	g.allocFloat(3.14)
	g.allocString("hello")

	g.reset()

	index := g.allocInt(99)
	require.Equal(t, 0, index)
	require.Equal(t, int64(99), g.getInt(0))

	fidx := g.allocFloat(2.72)
	require.Equal(t, 0, fidx)
	require.InDelta(t, 2.72, g.getFloat(0), 0.0001)
}

func TestGlobalStoreBoolAlloc(t *testing.T) {
	t.Parallel()

	g := newGlobalStore()
	idx0 := g.allocBool(true)
	idx1 := g.allocBool(false)
	require.Equal(t, 0, idx0)
	require.Equal(t, 1, idx1)
	require.True(t, g.getBool(0))
	require.False(t, g.getBool(1))
	g.setBool(0, false)
	require.False(t, g.getBool(0))
}

func TestGlobalStoreUintAlloc(t *testing.T) {
	t.Parallel()

	g := newGlobalStore()
	index := g.allocUint(42)
	require.Equal(t, 0, index)
	require.Equal(t, uint64(42), g.getUint(0))
	g.setUint(0, 99)
	require.Equal(t, uint64(99), g.getUint(0))
}

func TestMultipleServicesParallelEval(t *testing.T) {
	t.Parallel()

	const numServices = 10
	ctx := context.Background()

	services := make([]*Service, numServices)
	for i := range numServices {
		services[i] = NewService()
	}

	t.Run("parallel", func(t *testing.T) {
		for i := range numServices {
			t.Run("", func(t *testing.T) {
				t.Parallel()
				service := services[i]
				result, err := service.Eval(ctx, "1 + 2")
				require.NoError(t, err)
				require.Equal(t, int64(3), result)
			})
		}
	})
}
