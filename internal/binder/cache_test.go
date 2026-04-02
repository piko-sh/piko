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
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/maths"
)

type TestSimple struct {
	Name     string `json:"name"`
	Password string `json:"-"`
	Age      int    `json:"age,omitempty"`
	IsActive bool
}

type TestComplex struct {
	SubPtr          *TestSub
	Embedded2       *TestEmbedded
	Embedded1       TestEmbedded
	ID              string `json:"id"`
	unexportedField string
	Sub             TestSub
}

type TestSub struct {
	Value float64 `json:"value"`
}

type TestEmbedded struct {
	Timestamp time.Time `json:"timestamp"`
	Counter   int       `json:"counter"`
}

func TestBinderCache_Get(t *testing.T) {
	t.Run("get should build, cache, and return the same instance on subsequent calls", func(t *testing.T) {
		c := &binderCache{}
		typ := reflect.TypeFor[TestSimple]()

		info1 := c.get(typ, defaultMaxPathDepth)
		require.NotNil(t, info1)
		assert.NotEmpty(t, info1.Fields)

		info2 := c.get(typ, defaultMaxPathDepth)
		require.NotNil(t, info2)

		assert.Same(t, info1, info2, "Expected the same *structInfo instance to be returned from cache")
	})

	t.Run("get should be thread-safe", func(t *testing.T) {
		c := &binderCache{}
		typ := reflect.TypeFor[TestComplex]()
		var wg sync.WaitGroup
		numGoroutines := 50

		wg.Add(numGoroutines)
		for range numGoroutines {
			go func() {
				defer wg.Done()
				info := c.get(typ, defaultMaxPathDepth)
				require.NotNil(t, info)

				require.Contains(t, info.Fields, "id")
			}()
		}
		wg.Wait()
	})
}

func TestBinderCache_Build(t *testing.T) {
	c := &binderCache{}

	t.Run("build simple struct", func(t *testing.T) {
		typ := reflect.TypeFor[TestSimple]()
		info := c.build(typ, defaultMaxPathDepth)
		require.NotNil(t, info)
		require.Len(t, info.Fields, 3, "Should have 3 exported fields, ignoring the one with tag '-'")

		fName, ok := info.Fields["name"]
		require.True(t, ok, "Field 'name' should be in the cache")
		assert.Equal(t, []int{0}, fName.Index)
		assert.Equal(t, reflect.TypeFor[string](), fName.Type)
		assert.Equal(t, "name", fName.Path)

		fIsActive, ok := info.Fields["IsActive"]
		require.True(t, ok, "Field 'IsActive' should be in the cache")
		assert.Equal(t, []int{3}, fIsActive.Index)
		assert.Equal(t, reflect.TypeFor[bool](), fIsActive.Type)
		assert.Equal(t, "IsActive", fIsActive.Path)

		_, ok = info.Fields["Password"]
		assert.False(t, ok, "Field 'Password' with tag '-' should be ignored")
		_, ok = info.Fields["-"]
		assert.False(t, ok, "Field with tag '-' should not be in the map")
	})

	t.Run("build complex struct with nesting and embedding", func(t *testing.T) {
		typ := reflect.TypeFor[TestComplex]()
		info := c.build(typ, defaultMaxPathDepth)
		require.NotNil(t, info)

		require.Len(t, info.Fields, 11)

		fID, ok := info.Fields["id"]
		require.True(t, ok)
		assert.Equal(t, []int{3}, fID.Index)
		assert.Equal(t, "id", fID.Path)

		fSubValue, ok := info.Fields["Sub.value"]
		require.True(t, ok)
		assert.Equal(t, []int{5, 0}, fSubValue.Index)
		assert.Equal(t, "Sub.value", fSubValue.Path)

		fSubPtrValue, ok := info.Fields["SubPtr.value"]
		require.True(t, ok)
		assert.Equal(t, []int{0, 0}, fSubPtrValue.Index)
		assert.Equal(t, "SubPtr.value", fSubPtrValue.Path)

		fCounter1, ok := info.Fields["Embedded1.counter"]
		require.True(t, ok, "Nested field 'Embedded1.counter' should be in cache")
		assert.Equal(t, []int{2, 1}, fCounter1.Index)
		assert.Equal(t, "Embedded1.counter", fCounter1.Path)

		fTimestamp1, ok := info.Fields["Embedded1.timestamp"]
		require.True(t, ok, "Nested field 'Embedded1.timestamp' should be in cache")
		assert.Equal(t, []int{2, 0}, fTimestamp1.Index)
		assert.Equal(t, "Embedded1.timestamp", fTimestamp1.Path)

		fCounter2, ok := info.Fields["Embedded2.counter"]
		require.True(t, ok, "Nested field 'Embedded2.counter' should be in cache")
		assert.Equal(t, []int{1, 1}, fCounter2.Index)
		assert.Equal(t, "Embedded2.counter", fCounter2.Path)

		fTimestamp2, ok := info.Fields["Embedded2.timestamp"]
		require.True(t, ok, "Nested field 'Embedded2.timestamp' should be in cache")
		assert.Equal(t, []int{1, 0}, fTimestamp2.Index)
		assert.Equal(t, "Embedded2.timestamp", fTimestamp2.Path)

		_, ok = info.Fields["unexportedField"]
		assert.False(t, ok)
	})

	t.Run("build struct with custom types", func(t *testing.T) {
		type MyStruct struct {
			Date  time.Time     `json:"date"`
			Price maths.Decimal `json:"price"`
		}
		typ := reflect.TypeFor[MyStruct]()
		info := c.build(typ, defaultMaxPathDepth)
		require.NotNil(t, info)
		require.Len(t, info.Fields, 2)

		fDate, ok := info.Fields["date"]
		require.True(t, ok)
		assert.Equal(t, []int{0}, fDate.Index)
		assert.Equal(t, reflect.TypeFor[time.Time](), fDate.Type)

		fPrice, ok := info.Fields["price"]
		require.True(t, ok)
		assert.Equal(t, []int{1}, fPrice.Index)
		assert.Equal(t, reflect.TypeFor[maths.Decimal](), fPrice.Type)
	})
}

func TestIsCustomType(t *testing.T) {
	testCases := []struct {
		input    reflect.Type
		name     string
		expected bool
	}{
		{name: "time.Time is a custom type", input: reflect.TypeFor[time.Time](), expected: true},
		{name: "maths.Decimal is a custom type", input: reflect.TypeFor[maths.Decimal](), expected: true},
		{name: "maths.Money is a custom type", input: reflect.TypeFor[maths.Money](), expected: true},
		{name: "string is not a custom type", input: reflect.TypeFor[string](), expected: false},
		{name: "int is not a custom type", input: reflect.TypeFor[int](), expected: false},
		{name: "user-defined struct is not a custom type by default", input: reflect.TypeFor[TestSimple](), expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, isCustomType(tc.input))
		})
	}
}
