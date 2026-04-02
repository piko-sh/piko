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

package registry_domain

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoolHelpers(t *testing.T) {
	t.Run("getOrderingMap returns usable map", func(t *testing.T) {
		m := getOrderingMap()
		require.NotNil(t, m)
		m["key"] = "value"
		assert.Equal(t, "value", m["key"])
		putOrderingMap(m)
	})

	t.Run("putOrderingMap clears the map", func(t *testing.T) {
		m := getOrderingMap()
		m["a"] = 1
		m["b"] = 2
		putOrderingMap(m)

		m2 := getOrderingMap()
		assert.Empty(t, m2)
		putOrderingMap(m2)
	})

	t.Run("getDedupeMap returns usable map", func(t *testing.T) {
		m := getDedupeMap()
		require.NotNil(t, m)
		m["key"] = struct{}{}
		_, exists := m["key"]
		assert.True(t, exists)
		putDedupeMap(m)
	})

	t.Run("putDedupeMap clears the map", func(t *testing.T) {
		m := getDedupeMap()
		m["x"] = struct{}{}
		putDedupeMap(m)

		m2 := getDedupeMap()
		assert.Empty(t, m2)
		putDedupeMap(m2)
	})

	t.Run("getQueryArgs returns usable slice", func(t *testing.T) {
		s := getQueryArgs()
		require.NotNil(t, s)
		*s = append(*s, "arg1", "arg2")
		assert.Len(t, *s, 2)
		putQueryArgs(s)
	})

	t.Run("putQueryArgs resets slice length", func(t *testing.T) {
		s := getQueryArgs()
		*s = append(*s, "a", "b", "c")
		putQueryArgs(s)

		s2 := getQueryArgs()
		assert.Empty(t, *s2)
		putQueryArgs(s2)
	})

	t.Run("getStringBuilder returns usable builder", func(t *testing.T) {
		b := getStringBuilder()
		require.NotNil(t, b)
		b.WriteString("hello")
		assert.Equal(t, "hello", b.String())
		putStringBuilder(b)
	})

	t.Run("putStringBuilder resets builder", func(t *testing.T) {
		b := getStringBuilder()
		b.WriteString("data")
		putStringBuilder(b)

		b2 := getStringBuilder()
		assert.Equal(t, "", b2.String())
		putStringBuilder(b2)
	})

	t.Run("getTagMap returns usable map", func(t *testing.T) {
		m := getTagMap()
		require.NotNil(t, m)
		m["tag"] = "value"
		assert.Equal(t, "value", m["tag"])
		putTagMap(m)
	})

	t.Run("putTagMap clears map", func(t *testing.T) {
		m := getTagMap()
		m["a"] = "1"
		m["b"] = "2"
		putTagMap(m)

		m2 := getTagMap()
		assert.Empty(t, m2)
		putTagMap(m2)
	})

	t.Run("getEventPayload returns usable map", func(t *testing.T) {
		m := getEventPayload()
		require.NotNil(t, m)
		m["key"] = "val"
		assert.Equal(t, "val", m["key"])
		putEventPayload(m)
	})

	t.Run("putEventPayload clears map", func(t *testing.T) {
		m := getEventPayload()
		m["x"] = 42
		putEventPayload(m)

		m2 := getEventPayload()
		assert.Empty(t, m2)
		putEventPayload(m2)
	})

	t.Run("getStringSlice returns usable slice", func(t *testing.T) {
		s := getStringSlice()
		require.NotNil(t, s)
		*s = append(*s, "a", "b")
		assert.Len(t, *s, 2)
		putStringSlice(s)
	})

	t.Run("putStringSlice resets slice length", func(t *testing.T) {
		s := getStringSlice()
		*s = append(*s, "x", "y", "z")
		putStringSlice(s)

		s2 := getStringSlice()
		assert.Empty(t, *s2)
		putStringSlice(s2)
	})
}

func TestPoolHelpers_TypeAssertionFallback(t *testing.T) {
	t.Run("getOrderingMap fallback on wrong type", func(t *testing.T) {
		orderingMapPool.Put("wrong type")
		m := getOrderingMap()
		require.NotNil(t, m)
		assert.Empty(t, m)
		putOrderingMap(m)
	})

	t.Run("getDedupeMap fallback on wrong type", func(t *testing.T) {
		dedupeMapPool.Put(42)
		m := getDedupeMap()
		require.NotNil(t, m)
		assert.Empty(t, m)
		putDedupeMap(m)
	})

	t.Run("getQueryArgs fallback on wrong type", func(t *testing.T) {
		queryArgsPool.Put("wrong type")
		s := getQueryArgs()
		require.NotNil(t, s)
		assert.Empty(t, *s)
		putQueryArgs(s)
	})

	t.Run("getStringBuilder fallback on wrong type", func(t *testing.T) {
		stringBuilderPool.Put(123)
		b := getStringBuilder()
		require.NotNil(t, b)
		assert.IsType(t, &strings.Builder{}, b)
		assert.Equal(t, "", b.String())
		putStringBuilder(b)
	})

	t.Run("getTagMap fallback on wrong type", func(t *testing.T) {
		tagMapPool.Put([]byte("wrong"))
		m := getTagMap()
		require.NotNil(t, m)
		assert.Empty(t, m)
		putTagMap(m)
	})

	t.Run("getEventPayload fallback on wrong type", func(t *testing.T) {
		eventPayloadPool.Put(false)
		m := getEventPayload()
		require.NotNil(t, m)
		assert.Empty(t, m)
		putEventPayload(m)
	})

	t.Run("getStringSlice fallback on wrong type", func(t *testing.T) {
		stringSlicePool.Put("wrong")
		s := getStringSlice()
		require.NotNil(t, s)
		assert.Empty(t, *s)
		putStringSlice(s)
	})
}

func TestConcurrent_PoolGetPutCycles(t *testing.T) {
	const goroutines = 20
	const iterations = 100

	t.Run("orderingMapPool", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := range goroutines {
			index := i
			wg.Go(func() {
				for j := range iterations {
					m := getOrderingMap()
					m[fmt.Sprintf("key-%d-%d", index, j)] = index
					assert.NotNil(t, m)
					putOrderingMap(m)
				}
			})
		}
		wg.Wait()
	})

	t.Run("dedupeMapPool", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := range goroutines {
			index := i
			wg.Go(func() {
				for j := range iterations {
					m := getDedupeMap()
					m[fmt.Sprintf("key-%d-%d", index, j)] = struct{}{}
					assert.NotNil(t, m)
					putDedupeMap(m)
				}
			})
		}
		wg.Wait()
	})

	t.Run("queryArgsPool", func(t *testing.T) {
		var wg sync.WaitGroup
		for range goroutines {
			wg.Go(func() {
				for range iterations {
					s := getQueryArgs()
					*s = append(*s, "arg1", "arg2")
					assert.Len(t, *s, 2)
					putQueryArgs(s)
				}
			})
		}
		wg.Wait()
	})

	t.Run("stringBuilderPool", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := range goroutines {
			index := i
			wg.Go(func() {
				for range iterations {
					b := getStringBuilder()
					_, _ = fmt.Fprintf(b, "goroutine-%d", index)
					assert.NotEmpty(t, b.String())
					putStringBuilder(b)
				}
			})
		}
		wg.Wait()
	})

	t.Run("tagMapPool", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := range goroutines {
			index := i
			wg.Go(func() {
				for j := range iterations {
					m := getTagMap()
					m["key"] = fmt.Sprintf("val-%d-%d", index, j)
					assert.NotNil(t, m)
					putTagMap(m)
				}
			})
		}
		wg.Wait()
	})

	t.Run("eventPayloadPool", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := range goroutines {
			index := i
			wg.Go(func() {
				for j := range iterations {
					m := getEventPayload()
					m["key"] = fmt.Sprintf("val-%d-%d", index, j)
					assert.NotNil(t, m)
					putEventPayload(m)
				}
			})
		}
		wg.Wait()
	})

	t.Run("stringSlicePool", func(t *testing.T) {
		var wg sync.WaitGroup
		for range goroutines {
			wg.Go(func() {
				for range iterations {
					s := getStringSlice()
					*s = append(*s, "a", "b", "c")
					assert.Len(t, *s, 3)
					putStringSlice(s)
				}
			})
		}
		wg.Wait()
	})
}
