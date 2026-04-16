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
	"errors"
	"fmt"
	"image/color"
	"net"
	"net/mail"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type SimpleForm struct {
	Score    *float64
	Name     string
	Age      int
	IsActive bool
}

type NestedForm struct {
	Profile *Profile
	User    struct {
		Name string
		ID   int
	}
}

type Profile struct {
	Email   string
	IsAdmin bool
}

type SliceForm struct {
	Tags  []string
	Items []Item
}

type Item struct {
	Name  string
	Price float64
}

func TestGetBinder(t *testing.T) {
	t.Run("returns a non-nil binder", func(t *testing.T) {
		b := GetBinder()
		require.NotNil(t, b)
	})

	t.Run("returns the same singleton instance", func(t *testing.T) {
		b1 := GetBinder()
		b2 := GetBinder()
		assert.Same(t, b1, b2, "GetBinder should return a singleton")
	})
}

func TestASTBinder_Bind(t *testing.T) {
	binder := NewASTBinder()

	t.Run("successful bind with simple paths (fast path)", func(t *testing.T) {
		var form SimpleForm
		src := map[string][]string{
			"Name":     {"Alice"},
			"Age":      {"30"},
			"IsActive": {"on"},
			"Score":    {"99.5"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		assert.Equal(t, "Alice", form.Name)
		assert.Equal(t, 30, form.Age)
		assert.True(t, form.IsActive)
		assert.Equal(t, new(99.5), form.Score)
	})

	t.Run("successful bind with nested paths (fast path)", func(t *testing.T) {
		var form NestedForm
		src := map[string][]string{
			"User.Name":     {"Bob"},
			"User.ID":       {"123"},
			"Profile.Email": {"bob@example.com"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		assert.Equal(t, "Bob", form.User.Name)
		assert.Equal(t, 123, form.User.ID)
		require.NotNil(t, form.Profile)
		assert.Equal(t, "bob@example.com", form.Profile.Email)
		assert.False(t, form.Profile.IsAdmin)
	})

	t.Run("successful bind with slice paths (slow path)", func(t *testing.T) {
		var form SliceForm
		src := map[string][]string{
			"Items[0].Name":  {"Apple"},
			"Items[0].Price": {"0.99"},
			"Items[2].Name":  {"Orange"},
			"Items[2].Price": {"1.29"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		require.Len(t, form.Items, 3, "Slice should be auto-grown to length 3")

		assert.Equal(t, "Apple", form.Items[0].Name)
		assert.Equal(t, 0.99, form.Items[0].Price)
		assert.Equal(t, "Orange", form.Items[2].Name)
		assert.Equal(t, 1.29, form.Items[2].Price)

		assert.Equal(t, "", form.Items[1].Name)
		assert.Equal(t, 0.0, form.Items[1].Price)
	})

	t.Run("returns MultiError on failure", func(t *testing.T) {
		var form SimpleForm
		src := map[string][]string{
			"Name":           {"Valid"},
			"Age":            {"thirty"},
			"NonExistent[0]": {"invalid"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err)

		multiErr, ok := errors.AsType[MultiError](err)
		require.True(t, ok, "Error should be a MultiError")
		require.Len(t, multiErr, 2, "Should contain two errors")

		convErr := multiErr["Age"]
		require.NotNil(t, convErr)
		assert.Contains(t, convErr.Error(), "could not set field 'Age'")

		pathErr := multiErr["NonExistent[0]"]
		require.NotNil(t, pathErr)
		assert.Contains(t, pathErr.Error(), "NonExistent[0]")
		assert.Contains(t, pathErr.Error(), "field not found")
	})

	t.Run("returns errInvalidTarget for non-pointer or non-struct dst", func(t *testing.T) {
		var form SimpleForm
		var notAPointer int
		src := map[string][]string{}

		err := binder.Bind(context.Background(), form, src)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "destination must be a pointer to a struct")
		assert.Contains(t, err.Error(), "binder.SimpleForm")

		err = binder.Bind(context.Background(), &notAPointer, src)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "destination must be a pointer to a struct")
		assert.Contains(t, err.Error(), "*int")

		err = binder.Bind(context.Background(), nil, src)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "destination must be a pointer to a struct")
		assert.Contains(t, err.Error(), "nil")
	})
}

func TestGrowSliceToFitIndex(t *testing.T) {
	t.Run("does nothing if index is within length", func(t *testing.T) {
		s := make([]int, 5)
		v := reflect.ValueOf(&s).Elem()
		err := growSliceToFitIndex(v, 4, 1_000)
		require.NoError(t, err)
		assert.Len(t, s, 5)
		assert.Equal(t, 5, cap(s))
	})

	t.Run("expands length if index is within capacity", func(t *testing.T) {
		s := make([]int, 2, 5)
		v := reflect.ValueOf(&s).Elem()
		err := growSliceToFitIndex(v, 4, 1_000)
		require.NoError(t, err)
		assert.Len(t, s, 5)
		assert.Equal(t, 5, cap(s))
	})

	t.Run("expands capacity and length if index is out of capacity", func(t *testing.T) {
		s := make([]int, 2, 3)
		v := reflect.ValueOf(&s).Elem()
		err := growSliceToFitIndex(v, 10, 1_000)
		require.NoError(t, err)
		assert.Len(t, s, 11)
		assert.Equal(t, 11, cap(s))
	})

	t.Run("returns error if not a slice", func(t *testing.T) {
		var i int
		v := reflect.ValueOf(&i).Elem()
		err := growSliceToFitIndex(v, 1, 1_000)
		require.Error(t, err)
		assert.Equal(t, "value is not a slice", err.Error())
	})
}

func TestASTCache(t *testing.T) {
	t.Run("cache stores and retrieves parsed AST for complex paths", func(t *testing.T) {
		binder := NewASTBinder()
		var form SliceForm

		src1 := map[string][]string{
			"Items[0].Name": {"Apple"},
		}
		err := binder.Bind(context.Background(), &form, src1)
		require.NoError(t, err)
		assert.Equal(t, "Apple", form.Items[0].Name)

		cachedAST, ok := binder.astCache.Load("Items[0].Name")
		require.True(t, ok, "AST should be cached after first bind")
		require.NotNil(t, cachedAST)
	})

	t.Run("cache is reused across multiple Bind calls with same path", func(t *testing.T) {
		binder := NewASTBinder()

		var form1 SliceForm
		src1 := map[string][]string{"Items[0].Name": {"First"}}
		err := binder.Bind(context.Background(), &form1, src1)
		require.NoError(t, err)

		cachedAST1, _ := binder.astCache.Load("Items[0].Name")

		var form2 SliceForm
		src2 := map[string][]string{"Items[0].Name": {"Second"}}
		err = binder.Bind(context.Background(), &form2, src2)
		require.NoError(t, err)

		cachedAST2, _ := binder.astCache.Load("Items[0].Name")

		assert.Same(t, cachedAST1, cachedAST2, "Should reuse the same cached AST")
		assert.Equal(t, "Second", form2.Items[0].Name)
	})

	t.Run("cache handles multiple different paths independently", func(t *testing.T) {
		binder := NewASTBinder()
		var form SliceForm

		src := map[string][]string{
			"Items[0].Name":  {"Apple"},
			"Items[1].Price": {"1.99"},
			"Items[2].Name":  {"Orange"},
		}
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		_, ok1 := binder.astCache.Load("Items[0].Name")
		_, ok2 := binder.astCache.Load("Items[1].Price")
		_, ok3 := binder.astCache.Load("Items[2].Name")

		assert.True(t, ok1, "First path should be cached")
		assert.True(t, ok2, "Second path should be cached")
		assert.True(t, ok3, "Third path should be cached")

		assert.Equal(t, "Apple", form.Items[0].Name)
		assert.Equal(t, 1.99, form.Items[1].Price)
		assert.Equal(t, "Orange", form.Items[2].Name)
	})

	t.Run("cache does not store paths with parse errors", func(t *testing.T) {
		binder := NewASTBinder()
		var form SliceForm

		src := map[string][]string{
			"Items[0].Name": {"Valid"},
			"Items[invalid": {"Bad"},
		}
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err, "Unparseable paths should be silently skipped")

		_, ok1 := binder.astCache.Load("Items[0].Name")
		assert.True(t, ok1, "Valid path should be cached")

		_, ok2 := binder.astCache.Load("Items[invalid")
		assert.False(t, ok2, "Path with parse error should not be cached")
	})

	t.Run("cache is thread-safe under concurrent access", func(t *testing.T) {
		binder := NewASTBinder()
		var wg sync.WaitGroup
		numGoroutines := 50

		wg.Add(numGoroutines)
		for i := range numGoroutines {
			go func(index int) {
				defer wg.Done()
				var form SliceForm
				src := map[string][]string{
					"Items[0].Name": {fmt.Sprintf("Item%d", index)},
				}
				err := binder.Bind(context.Background(), &form, src)
				require.NoError(t, err)
			}(i)
		}
		wg.Wait()

		cachedAST, ok := binder.astCache.Load("Items[0].Name")
		require.True(t, ok, "AST should be cached after concurrent access")
		require.NotNil(t, cachedAST)
	})

	t.Run("cache works with indexed member expressions", func(t *testing.T) {
		binder := NewASTBinder()

		var form SliceForm

		src := map[string][]string{
			"Items[2].Name":  {"IndexedItem"},
			"Items[2].Price": {"99.99"},
		}
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		assert.Equal(t, "IndexedItem", form.Items[2].Name)
		assert.Equal(t, 99.99, form.Items[2].Price)

		cachedAST1, ok1 := binder.astCache.Load("Items[2].Name")
		require.True(t, ok1, "Indexed member path should be cached in AST cache")
		require.NotNil(t, cachedAST1)

		cachedAST2, ok2 := binder.astCache.Load("Items[2].Price")
		require.True(t, ok2, "Another indexed member path should be cached in AST cache")
		require.NotNil(t, cachedAST2)
	})

	t.Run("cache persists across singleton binder instance", func(t *testing.T) {

		binder1 := GetBinder()
		binder2 := GetBinder()

		require.Same(t, binder1, binder2)

		var form1 SliceForm
		src1 := map[string][]string{"Items[5].Name": {"Cached"}}
		err := binder1.Bind(context.Background(), &form1, src1)
		require.NoError(t, err)

		cachedAST, ok := binder2.astCache.Load("Items[5].Name")
		require.True(t, ok, "Cache should persist in singleton")
		require.NotNil(t, cachedAST)

		var form2 SliceForm
		src2 := map[string][]string{"Items[5].Name": {"Reused"}}
		err = binder2.Bind(context.Background(), &form2, src2)
		require.NoError(t, err)
		assert.Equal(t, "Reused", form2.Items[5].Name)
	})
}

func TestMaxSliceSize(t *testing.T) {
	t.Run("default behaviour allows unlimited slice growth", func(t *testing.T) {
		binder := NewASTBinder()
		var form SliceForm

		src := map[string][]string{
			"Items[999].Name": {"LargeIndex"},
		}
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		assert.Len(t, form.Items, 1000)
		assert.Equal(t, "LargeIndex", form.Items[999].Name)
	})

	t.Run("setting maxSliceSize enforces the limit", func(t *testing.T) {
		binder := NewASTBinder()
		binder.SetMaxSliceSize(100)
		var form SliceForm

		src := map[string][]string{
			"Items[50].Name": {"WithinLimit"},
		}
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		assert.Len(t, form.Items, 51)
		assert.Equal(t, "WithinLimit", form.Items[50].Name)
	})

	t.Run("exceeding maxSliceSize returns error", func(t *testing.T) {
		binder := NewASTBinder()
		binder.SetMaxSliceSize(100)
		var form SliceForm

		src := map[string][]string{
			"Items[100].Name": {"ExceedsLimit"},
		}
		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err)

		multiErr, ok := errors.AsType[MultiError](err)
		require.True(t, ok, "Error should be a MultiError")
		require.Contains(t, multiErr, "Items[100].Name")

		setFieldErr := multiErr["Items[100].Name"]
		require.NotNil(t, setFieldErr, "Error should exist for Items[100].Name")
		assert.Contains(t, setFieldErr.Error(), "exceeds maximum allowed size of 100")
	})

	t.Run("maxSliceSize exactly at boundary", func(t *testing.T) {
		binder := NewASTBinder()
		binder.SetMaxSliceSize(10)
		var form SliceForm

		src1 := map[string][]string{
			"Items[9].Name": {"AtBoundary"},
		}
		err := binder.Bind(context.Background(), &form, src1)
		require.NoError(t, err)
		assert.Equal(t, "AtBoundary", form.Items[9].Name)

		src2 := map[string][]string{
			"Items[10].Name": {"OverBoundary"},
		}
		err = binder.Bind(context.Background(), &form, src2)
		require.Error(t, err)
	})

	t.Run("SetMaxSliceSize with negative value sets to zero (unlimited)", func(t *testing.T) {
		binder := NewASTBinder()
		binder.SetMaxSliceSize(-5)
		var form SliceForm

		src := map[string][]string{
			"Items[200].Name": {"Unlimited"},
		}
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		assert.Equal(t, "Unlimited", form.Items[200].Name)
	})

	t.Run("SetMaxSliceSize is thread-safe", func(t *testing.T) {
		binder := NewASTBinder()
		var wg sync.WaitGroup

		for i := range 10 {
			size := i
			wg.Go(func() {
				binder.SetMaxSliceSize(size * 10)
			})
		}
		wg.Wait()

		var form SliceForm
		src := map[string][]string{
			"Items[5].Name": {"ThreadSafe"},
		}

		_ = binder.Bind(context.Background(), &form, src)
	})

	t.Run("maxSliceSize protects against memory exhaustion", func(t *testing.T) {
		binder := NewASTBinder()
		binder.SetMaxSliceSize(1000)
		var form SliceForm

		src := map[string][]string{
			"Items[9999999].Name": {"Attack"},
		}
		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err)

		assert.Len(t, form.Items, 0, "Slice should not be grown when limit is exceeded")
	})

	t.Run("maxSliceSize applies to nested slices", func(t *testing.T) {
		type NestedSliceForm struct {
			Outer []struct {
				Inner []string
			}
		}

		binder := NewASTBinder()
		binder.SetMaxSliceSize(50)
		var form NestedSliceForm

		src := map[string][]string{
			"Outer[5].Inner[100]": {"Nested"},
		}
		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err)
	})
}

func TestMapBinding(t *testing.T) {
	t.Run("basic map[int]Item binding with sparse keys", func(t *testing.T) {
		type MapForm struct {
			Items map[int]Item
		}

		binder := NewASTBinder()
		var form MapForm

		src := map[string][]string{
			"Items[101].Name":  {"Apple"},
			"Items[101].Price": {"0.99"},
			"Items[105].Name":  {"Orange"},
			"Items[105].Price": {"1.29"},
			"Items[210].Name":  {"Banana"},
			"Items[210].Price": {"0.79"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		require.NotNil(t, form.Items)
		require.Len(t, form.Items, 3, "Map should have exactly 3 entries")

		assert.Equal(t, "Apple", form.Items[101].Name)
		assert.Equal(t, 0.99, form.Items[101].Price)
		assert.Equal(t, "Orange", form.Items[105].Name)
		assert.Equal(t, 1.29, form.Items[105].Price)
		assert.Equal(t, "Banana", form.Items[210].Name)
		assert.Equal(t, 0.79, form.Items[210].Price)
	})

	t.Run("map[string]Item binding", func(t *testing.T) {
		type StringKeyMapForm struct {
			Products map[string]Item
		}

		binder := NewASTBinder()
		var form StringKeyMapForm

		src := map[string][]string{
			`Products["apple"].Name`:   {"Apple"},
			`Products["apple"].Price`:  {"0.99"},
			`Products["orange"].Name`:  {"Orange"},
			`Products["orange"].Price`: {"1.29"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		require.NotNil(t, form.Products)
		require.Len(t, form.Products, 2)

		assert.Equal(t, "Apple", form.Products["apple"].Name)
		assert.Equal(t, 0.99, form.Products["apple"].Price)
		assert.Equal(t, "Orange", form.Products["orange"].Name)
		assert.Equal(t, 1.29, form.Products["orange"].Price)
	})

	t.Run("nil map initialisation", func(t *testing.T) {
		type MapForm struct {
			Items map[int]Item
		}

		binder := NewASTBinder()
		var form MapForm

		require.Nil(t, form.Items)

		src := map[string][]string{
			"Items[1].Name": {"Test"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		require.NotNil(t, form.Items)
		assert.Equal(t, "Test", form.Items[1].Name)
	})

	t.Run("map with pointer element types", func(t *testing.T) {
		type MapPtrForm struct {
			Items map[int]*Item
		}

		binder := NewASTBinder()
		var form MapPtrForm

		src := map[string][]string{
			"Items[1].Name":  {"First"},
			"Items[1].Price": {"1.00"},
			"Items[2].Name":  {"Second"},
			"Items[2].Price": {"2.00"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		require.NotNil(t, form.Items)
		require.NotNil(t, form.Items[1])
		require.NotNil(t, form.Items[2])

		assert.Equal(t, "First", form.Items[1].Name)
		assert.Equal(t, 1.00, form.Items[1].Price)
		assert.Equal(t, "Second", form.Items[2].Name)
		assert.Equal(t, 2.00, form.Items[2].Price)
	})

	t.Run("mixed slice and map in same struct", func(t *testing.T) {
		type MixedForm struct {
			MapItems   map[int]Item
			SliceItems []Item
		}

		binder := NewASTBinder()
		var form MixedForm

		src := map[string][]string{
			"SliceItems[0].Name": {"SliceItem1"},
			"SliceItems[1].Name": {"SliceItem2"},
			"MapItems[100].Name": {"MapItem1"},
			"MapItems[200].Name": {"MapItem2"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		require.Len(t, form.SliceItems, 2)
		assert.Equal(t, "SliceItem1", form.SliceItems[0].Name)
		assert.Equal(t, "SliceItem2", form.SliceItems[1].Name)

		require.Len(t, form.MapItems, 2)
		assert.Equal(t, "MapItem1", form.MapItems[100].Name)
		assert.Equal(t, "MapItem2", form.MapItems[200].Name)
	})

	t.Run("map with different integer key types", func(t *testing.T) {
		type SimpleValue struct {
			Value string
		}
		type MultiKeyForm struct {
			Int32Map  map[int32]SimpleValue
			Int64Map  map[int64]SimpleValue
			Uint32Map map[uint32]SimpleValue
		}

		binder := NewASTBinder()
		var form MultiKeyForm

		src := map[string][]string{
			"Int32Map[42].Value":  {"int32value"},
			"Int64Map[999].Value": {"int64value"},
			"Uint32Map[10].Value": {"uint32value"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		assert.Equal(t, "int32value", form.Int32Map[42].Value)
		assert.Equal(t, "int64value", form.Int64Map[999].Value)
		assert.Equal(t, "uint32value", form.Uint32Map[10].Value)
	})

	t.Run("error: field is not a slice or map", func(t *testing.T) {
		type InvalidForm struct {
			NotACollection string
		}

		binder := NewASTBinder()
		var form InvalidForm

		src := map[string][]string{
			"NotACollection[0]": {"value"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err)

		multiErr, ok := errors.AsType[MultiError](err)
		require.True(t, ok)

		assert.Contains(t, multiErr["NotACollection[0]"].Error(), "field is not a slice, map, or struct")
	})

	t.Run("map preserves existing entries", func(t *testing.T) {
		type MapForm struct {
			Items map[int]Item
		}

		binder := NewASTBinder()
		var form MapForm

		form.Items = make(map[int]Item)
		form.Items[99] = Item{Name: "Existing", Price: 9.99}

		src := map[string][]string{
			"Items[101].Name": {"New"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		assert.Equal(t, "Existing", form.Items[99].Name)
		assert.Equal(t, 9.99, form.Items[99].Price)

		assert.Equal(t, "New", form.Items[101].Name)
	})

	t.Run("map with nested struct fields", func(t *testing.T) {
		type Address struct {
			Street string
			City   string
		}
		type Person struct {
			Name    string
			Address Address
		}
		type PersonMap struct {
			People map[int]Person
		}

		binder := NewASTBinder()
		var form PersonMap

		src := map[string][]string{
			"People[1].Name":           {"Alice"},
			"People[1].Address.Street": {"123 Main St"},
			"People[1].Address.City":   {"Springfield"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		require.NotNil(t, form.People)
		assert.Equal(t, "Alice", form.People[1].Name)
		assert.Equal(t, "123 Main St", form.People[1].Address.Street)
		assert.Equal(t, "Springfield", form.People[1].Address.City)
	})

	t.Run("error: invalid key conversion for map", func(t *testing.T) {
		type MapForm struct {
			Items map[int]Item
		}

		binder := NewASTBinder()
		var form MapForm

		src := map[string][]string{
			`Items["notanumber"].Name`: {"test"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err)

		multiErr, ok := errors.AsType[MultiError](err)
		require.True(t, ok)
		assert.Contains(t, multiErr[`Items["notanumber"].Name`].Error(), "could not convert map key")
	})

	t.Run("large sparse map keys", func(t *testing.T) {
		type MapForm struct {
			Items map[int]Item
		}

		binder := NewASTBinder()
		var form MapForm

		src := map[string][]string{
			"Items[1000000].Name": {"Million"},
			"Items[2000000].Name": {"TwoMillion"},
			"Items[5000000].Name": {"FiveMillion"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		require.Len(t, form.Items, 3, "Map should only have 3 entries (efficient memory usage)")
		assert.Equal(t, "Million", form.Items[1000000].Name)
		assert.Equal(t, "TwoMillion", form.Items[2000000].Name)
		assert.Equal(t, "FiveMillion", form.Items[5000000].Name)
	})
}

func TestComplexScenarios(t *testing.T) {
	binder := NewASTBinder()

	t.Run("Scenario 1: Slice of Structs with Pointers", func(t *testing.T) {
		type PtrItem struct {
			ID   *int
			Name *string
		}
		type PtrItemForm struct {
			Items []*PtrItem
		}

		src := map[string][]string{
			"Items[0].ID":   {"101"},
			"Items[0].Name": {"Apple"},
			"Items[2].ID":   {"103"},
			"Items[2].Name": {"Orange"},
		}

		var form PtrItemForm
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		require.Len(t, form.Items, 3, "Slice should be auto-grown to length 3")

		require.NotNil(t, form.Items[0], "Items[0] should not be nil")
		require.NotNil(t, form.Items[0].ID, "Items[0].ID should not be nil")
		assert.Equal(t, 101, *form.Items[0].ID)
		require.NotNil(t, form.Items[0].Name, "Items[0].Name should not be nil")
		assert.Equal(t, "Apple", *form.Items[0].Name)

		assert.Nil(t, form.Items[1], "Items[1] should be nil (sparse slice)")

		require.NotNil(t, form.Items[2], "Items[2] should not be nil")
		require.NotNil(t, form.Items[2].ID, "Items[2].ID should not be nil")
		assert.Equal(t, 103, *form.Items[2].ID)
		require.NotNil(t, form.Items[2].Name, "Items[2].Name should not be nil")
		assert.Equal(t, "Orange", *form.Items[2].Name)
	})

	t.Run("Scenario 2: Map of Structs with Pointer Fields", func(t *testing.T) {
		type MapItem struct {
			SKU   *string
			Price *float64
		}
		type MapItemForm struct {
			Products map[string]*MapItem
		}

		src := map[string][]string{
			`Products["abc-123"].SKU`:   {"abc-123"},
			`Products["abc-123"].Price`: {"19.99"},
			`Products["xyz-789"].SKU`:   {"xyz-789"},
			`Products["xyz-789"].Price`: {"25.50"},
		}

		var form MapItemForm
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		require.NotNil(t, form.Products, "Products map should be initialised")
		require.Len(t, form.Products, 2, "Should have 2 products")

		require.NotNil(t, form.Products["abc-123"], "Product abc-123 should exist")
		require.NotNil(t, form.Products["abc-123"].SKU, "SKU should not be nil")
		assert.Equal(t, "abc-123", *form.Products["abc-123"].SKU)
		require.NotNil(t, form.Products["abc-123"].Price, "Price should not be nil")
		assert.Equal(t, 19.99, *form.Products["abc-123"].Price)

		require.NotNil(t, form.Products["xyz-789"], "Product xyz-789 should exist")
		require.NotNil(t, form.Products["xyz-789"].SKU, "SKU should not be nil")
		assert.Equal(t, "xyz-789", *form.Products["xyz-789"].SKU)
		require.NotNil(t, form.Products["xyz-789"].Price, "Price should not be nil")
		assert.Equal(t, 25.50, *form.Products["xyz-789"].Price)
	})

	t.Run("Scenario 3: Deeply Nested Structs and Slices", func(t *testing.T) {
		type Office struct {
			Room     *int
			Building string
		}
		type Employee struct {
			Office Office
			Name   string
		}
		type Department struct {
			Name      string
			Employees []Employee
		}
		type CompanyForm struct {
			Departments []Department
		}

		src := map[string][]string{
			"Departments[0].Name": {"Engineering"},
			"Departments[1].Name": {"Marketing"},
		}

		var form CompanyForm
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		require.Len(t, form.Departments, 2, "Should have 2 departments")
		assert.Equal(t, "Engineering", form.Departments[0].Name)
		assert.Equal(t, "Marketing", form.Departments[1].Name)

		srcChained := map[string][]string{
			"Departments[0].Employees[0].Name": {"Alice"},
			"Departments[0].Employees[1].Name": {"Bob"},
		}
		var form2 CompanyForm
		err = binder.Bind(context.Background(), &form2, srcChained)
		require.NoError(t, err, "Chained indexing should now work")
		require.Len(t, form2.Departments, 1, "Should have 1 department")
		require.Len(t, form2.Departments[0].Employees, 2, "Should have 2 employees")
		assert.Equal(t, "Alice", form2.Departments[0].Employees[0].Name)
		assert.Equal(t, "Bob", form2.Departments[0].Employees[1].Name)
	})

	t.Run("Scenario 4: Map of Slices of Primitives (Limitation Test)", func(t *testing.T) {
		type TagsForm struct {
			TagsByCategory map[string][]string
		}

		src := map[string][]string{
			`TagsByCategory["tech"][0]`: {"golang"},
		}

		var form TagsForm
		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err, "Chained map-to-slice indexing should fail")
		multiErr, ok := errors.AsType[MultiError](err)
		require.True(t, ok)
		assert.Contains(t, multiErr[`TagsByCategory["tech"][0]`].Error(), "cannot index into slice obtained from map value")
	})

	t.Run("Scenario 5: Slice containing a Map", func(t *testing.T) {
		type ConfigOption struct {
			Settings map[string]string
			Name     string
		}
		type ConfigForm struct {
			Configs []ConfigOption
		}

		src := map[string][]string{
			"Configs[0].Name": {"Config1"},
			"Configs[1].Name": {"Config2"},
		}

		var form ConfigForm
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		require.Len(t, form.Configs, 2, "Should have 2 configs")
		assert.Equal(t, "Config1", form.Configs[0].Name)
		assert.Equal(t, "Config2", form.Configs[1].Name)

		src2 := map[string][]string{
			`Configs[0].Settings["theme"]`: {"dark"},
		}
		var form2 ConfigForm
		err = binder.Bind(context.Background(), &form2, src2)
		require.NoError(t, err, "Nested map access after slice index should now work")
		require.Len(t, form2.Configs, 1, "Should have 1 config")
		require.NotNil(t, form2.Configs[0].Settings, "Settings map should be initialised")
		require.Contains(t, form2.Configs[0].Settings, "theme", "Map should contain the key 'theme'")
		assert.Equal(t, "dark", form2.Configs[0].Settings["theme"])
	})

	t.Run("Scenario 6: Map of Maps (Limitation Test)", func(t *testing.T) {
		type PermissionsForm struct {
			Permissions map[string]map[string]bool
		}

		src := map[string][]string{
			`Permissions["admin"]["users.create"]`: {"true"},
		}

		var form PermissionsForm
		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err, "Chained map indexing should fail")
		multiErr, ok := errors.AsType[MultiError](err)
		require.True(t, ok)
		assert.Contains(t, multiErr[`Permissions["admin"]["users.create"]`].Error(), "chained")
	})

	t.Run("Scenario 7: Anonymous/Embedded Structs (Flattened Namespace)", func(t *testing.T) {
		type Address struct {
			City  string
			State string
		}
		type Contact struct {
			Address
			Email string
		}
		type EmbeddedForm struct {
			Contact
		}

		src := map[string][]string{
			"City":  {"Springfield"},
			"State": {"IL"},
			"Email": {"test@example.com"},
		}

		var form EmbeddedForm
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		assert.Equal(t, "Springfield", form.City)
		assert.Equal(t, "IL", form.State)
		assert.Equal(t, "test@example.com", form.Email)
	})

	t.Run("Scenario 8: Pointer to an Anonymous Struct", func(t *testing.T) {
		type PtrAddress struct {
			Street string
		}
		type PtrEmbeddedForm struct {
			*PtrAddress
			ZipCode string
		}

		src := map[string][]string{
			"Street":  {"123 Main St"},
			"ZipCode": {"90210"},
		}

		var form PtrEmbeddedForm
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		require.NotNil(t, form.PtrAddress, "Embedded pointer should be initialised")
		assert.Equal(t, "123 Main St", form.Street)
		assert.Equal(t, "90210", form.ZipCode)
	})

	t.Run("Scenario 9: Mixed Slices and Maps with Non-String Keys (Limitation Test)", func(t *testing.T) {
		type UserRoles struct {
			RolesByOrg map[int][]string
		}

		src := map[string][]string{
			"RolesByOrg[100][0]": {"admin"},
		}

		var form UserRoles
		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err, "Chained map-to-slice indexing should fail")
		multiErr, ok := errors.AsType[MultiError](err)
		require.True(t, ok)
		assert.Contains(t, multiErr["RolesByOrg[100][0]"].Error(), "cannot index into slice obtained from map value")
	})

	t.Run("Scenario 10: Array instead of Slice (Limitation Test)", func(t *testing.T) {
		type ArrayForm struct {
			Coordinates [3]float64
		}

		src := map[string][]string{
			"Coordinates[0]": {"1.1"},
		}

		var form ArrayForm
		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err, "Arrays should not be supported")
		multiErr, ok := errors.AsType[MultiError](err)
		require.True(t, ok)
		assert.Contains(t, multiErr["Coordinates[0]"].Error(), "field is not a slice, map, or struct, but got array")
	})

	t.Run("Scenario 11: Type Aliases for Primitives and Structs", func(t *testing.T) {
		type UserID int
		type ProductSKU string
		type AliasItem struct{ SKU ProductSKU }
		type AliasForm struct {
			Items []AliasItem
			ID    UserID
		}

		src := map[string][]string{
			"ID":           {"12345"},
			"Items[0].SKU": {"abc-definition"},
		}

		var form AliasForm
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		assert.Equal(t, UserID(12345), form.ID)
		require.Len(t, form.Items, 1, "Should have 1 item")
		assert.Equal(t, ProductSKU("abc-definition"), form.Items[0].SKU)
	})

	t.Run("Scenario 12: Structs Implementing encoding.TextUnmarshaler", func(t *testing.T) {
		type CustomDate struct {
			time.Time
		}

		customDateUnmarshalText := func(cd *CustomDate, text []byte) error {
			t, err := time.Parse("2006-01-02", string(text))
			if err == nil {
				cd.Time = t
			}
			return err
		}

		binder.RegisterConverter(reflect.TypeFor[CustomDate](), func(value string) (reflect.Value, error) {
			var cd CustomDate
			err := customDateUnmarshalText(&cd, []byte(value))
			return reflect.ValueOf(cd), err
		})

		type UnmarshalerForm struct {
			EventDate CustomDate
		}

		src := map[string][]string{
			"EventDate": {"2025-10-09"},
		}

		var form UnmarshalerForm
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		expectedDate, _ := time.Parse("2006-01-02", "2025-10-09")
		assert.Equal(t, expectedDate.Year(), form.EventDate.Year())
		assert.Equal(t, expectedDate.Month(), form.EventDate.Month())
		assert.Equal(t, expectedDate.Day(), form.EventDate.Day())
	})

	t.Run("Scenario 13: Slice of a Type Implementing TextUnmarshaler (Limitation Test)", func(t *testing.T) {
		type CustomDate struct {
			Value string
		}

		type UnmarshalerSliceForm struct {
			Holidays []CustomDate
		}

		src := map[string][]string{
			"Holidays[0].Value": {"2025-01-01"},
			"Holidays[1].Value": {"2025-12-25"},
		}

		var form UnmarshalerSliceForm
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		require.Len(t, form.Holidays, 2, "Should have 2 holidays")
		assert.Equal(t, "2025-01-01", form.Holidays[0].Value)
		assert.Equal(t, "2025-12-25", form.Holidays[1].Value)

		src2 := map[string][]string{
			"Holidays[0]": {"2025-01-01"},
		}
		var form2 UnmarshalerSliceForm
		err = binder.Bind(context.Background(), &form2, src2)
		require.Error(t, err, "Bare index expression should fail")
		multiErr, ok := errors.AsType[MultiError](err)
		require.True(t, ok)
		assert.Contains(t, multiErr["Holidays[0]"].Error(), "unsupported type")
	})

	t.Run("Scenario 14: Overlapping Names with Embedded Structs", func(t *testing.T) {
		type Base struct {
			ID string
		}
		type Overlap struct {
			Base
			ID string
		}

		src := map[string][]string{
			"ID": {"shadowed"},
		}

		var form Overlap
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		assert.Equal(t, "shadowed", form.ID, "Should bind to top-level ID")
		assert.Equal(t, "", form.Base.ID, "Base.ID should remain zero value")
	})

	t.Run("Scenario 15: Generic Structs (Go 1.18+)", func(t *testing.T) {
		type Payload[T any] struct {
			Data T
		}
		type GenericForm struct {
			StringPayload Payload[string]
			IntPayload    Payload[int]
		}

		src := map[string][]string{
			"IntPayload.Data":    {"42"},
			"StringPayload.Data": {"hello"},
		}

		var form GenericForm
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		assert.Equal(t, 42, form.IntPayload.Data)
		assert.Equal(t, "hello", form.StringPayload.Data)
	})

	t.Run("Scenario 16: Slice of Generic Structs", func(t *testing.T) {
		type Payload[T any] struct {
			Data T
		}
		type GenericSliceForm struct {
			Events []Payload[bool]
		}

		src := map[string][]string{
			"Events[0].Data": {"true"},
			"Events[1].Data": {"false"},
		}

		var form GenericSliceForm
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		require.Len(t, form.Events, 2, "Should have 2 events")
		assert.True(t, form.Events[0].Data)
		assert.False(t, form.Events[1].Data)
	})

	t.Run("Scenario 17: Interface Fields (Now Supported)", func(t *testing.T) {

		type InterfaceForm struct {
			Data any
		}

		src := map[string][]string{
			"Data": {"some-value"},
		}

		var form InterfaceForm
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err, "Binding to interface should now succeed")
		assert.Equal(t, "some-value", form.Data)
	})

	t.Run("Scenario 18: Map with Struct Keys (Unparseable path is skipped)", func(t *testing.T) {
		type Key struct{ ID int }
		type StructKeyMapForm struct {
			Data map[Key]string
		}

		src := map[string][]string{
			`Data[invalid]`: {"value"},
		}

		var form StructKeyMapForm
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err, "Unparseable paths should be silently skipped")
		assert.Nil(t, form.Data, "Map should remain nil when path is skipped")
	})

	t.Run("Scenario 19: Unexported fields (Should be Ignored)", func(t *testing.T) {
		type UnexportedForm struct {
			Exported   string
			unexported string
		}

		src := map[string][]string{
			"Exported":   {"visible"},
			"unexported": {"invisible"},
		}

		var form UnexportedForm
		err := binder.Bind(context.Background(), &form, src)

		if err != nil {
			multiErr, ok := errors.AsType[MultiError](err)
			require.True(t, ok, "Error should be a MultiError")

			_, hasUnexported := multiErr["unexported"]
			assert.True(t, hasUnexported, "Should have error for unexported field")
			assert.Len(t, multiErr, 1, "Should only have error for unexported field")
		}

		assert.Equal(t, "visible", form.Exported, "Exported field should be set")
		assert.Equal(t, "", form.unexported, "Unexported field should remain zero value")
	})

	t.Run("Scenario 20: Path Collisions with Maps and Structs", func(t *testing.T) {
		type CollisionForm struct {
			user map[string]string

			User struct {
				Name string
			}
		}

		src := map[string][]string{
			"User.Name": {"Alice"},
		}

		var form CollisionForm
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		assert.Equal(t, "Alice", form.User.Name, "Should bind to exported User struct")
		assert.Nil(t, form.user, "Unexported user map should remain nil")
	})

	t.Run("Scenario 21: map[string]any with nested map access", func(t *testing.T) {

		type DynamicForm struct {
			Fields map[string]any `json:"fields"`
		}

		form := DynamicForm{
			Fields: map[string]any{
				"open_viewing": map[string]any{},
				"room_setup":   map[string]any{},
			},
		}

		src := map[string][]string{
			`Fields["open_viewing"]["enabled"]`: {"true"},
			`Fields["open_viewing"]["time"]`:    {"14:00"},
			`Fields["room_setup"]["bedrooms"]`:  {"3"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err, "Should bind to nested maps within map[string]any")

		openViewing, ok := form.Fields["open_viewing"].(map[string]any)
		require.True(t, ok, "open_viewing should be a map[string]any")
		assert.Equal(t, "true", openViewing["enabled"])
		assert.Equal(t, "14:00", openViewing["time"])

		roomSetup, ok := form.Fields["room_setup"].(map[string]any)
		require.True(t, ok, "room_setup should be a map[string]any")
		assert.Equal(t, "3", roomSetup["bedrooms"])
	})

	t.Run("Scenario 22: map[string]any with slice access (Limitation Test)", func(t *testing.T) {

		type DynamicForm struct {
			Data map[string]any `json:"data"`
		}

		form := DynamicForm{
			Data: map[string]any{
				"items": []any{"", "", ""},
			},
		}

		src := map[string][]string{
			`Data["items"][0]`: {"first"},
			`Data["items"][1]`: {"second"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err, "Chained map[key][index] patterns should fail")

		multiErr, ok := errors.AsType[MultiError](err)
		require.True(t, ok)

		assert.Contains(t, multiErr[`Data["items"][0]`].Error(), "not a map")
	})
}

func TestUncoveredIntegerTypes(t *testing.T) {
	binder := NewASTBinder()

	t.Run("all integer variants with valid values", func(t *testing.T) {
		type IntVariantsForm struct {
			Age16 int16
			Age32 int32
			Age64 int64
			Count uint
			Byte  uint8
			Port  uint16
			ID    uint32
			BigID uint64
		}

		src := map[string][]string{
			"Age16": {"32767"},
			"Age32": {"2147483647"},
			"Age64": {"9223372036854775807"},
			"Count": {"123"},
			"Byte":  {"255"},
			"Port":  {"65535"},
			"ID":    {"4294967295"},
			"BigID": {"18446744073709551615"},
		}

		var form IntVariantsForm
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		assert.Equal(t, int16(32767), form.Age16)
		assert.Equal(t, int32(2147483647), form.Age32)
		assert.Equal(t, int64(9223372036854775807), form.Age64)
		assert.Equal(t, uint(123), form.Count)
		assert.Equal(t, uint8(255), form.Byte)
		assert.Equal(t, uint16(65535), form.Port)
		assert.Equal(t, uint32(4294967295), form.ID)
		assert.Equal(t, uint64(18446744073709551615), form.BigID)
	})

	t.Run("negative values for signed types", func(t *testing.T) {
		type SignedForm struct {
			Val16 int16
			Val32 int32
			Val64 int64
		}

		src := map[string][]string{
			"Val16": {"-32768"},
			"Val32": {"-2147483648"},
			"Val64": {"-9223372036854775808"},
		}

		var form SignedForm
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		assert.Equal(t, int16(-32768), form.Val16)
		assert.Equal(t, int32(-2147483648), form.Val32)
		assert.Equal(t, int64(-9223372036854775808), form.Val64)
	})

	t.Run("zero values", func(t *testing.T) {
		type ZeroForm struct {
			Val16  int16
			Val32  int32
			Val64  int64
			UVal   uint
			UVal8  uint8
			UVal16 uint16
			UVal32 uint32
			UVal64 uint64
		}

		src := map[string][]string{
			"Val16":  {"0"},
			"Val32":  {"0"},
			"Val64":  {"0"},
			"UVal":   {"0"},
			"UVal8":  {"0"},
			"UVal16": {"0"},
			"UVal32": {"0"},
			"UVal64": {"0"},
		}

		var form ZeroForm
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		assert.Equal(t, int16(0), form.Val16)
		assert.Equal(t, int32(0), form.Val32)
		assert.Equal(t, int64(0), form.Val64)
		assert.Equal(t, uint(0), form.UVal)
		assert.Equal(t, uint8(0), form.UVal8)
		assert.Equal(t, uint16(0), form.UVal16)
		assert.Equal(t, uint32(0), form.UVal32)
		assert.Equal(t, uint64(0), form.UVal64)
	})

	t.Run("pointer variants", func(t *testing.T) {
		type PtrForm struct {
			Val16  *int16
			Val32  *int32
			Val64  *int64
			UVal   *uint
			UVal8  *uint8
			UVal16 *uint16
			UVal32 *uint32
			UVal64 *uint64
		}

		src := map[string][]string{
			"Val16":  {"100"},
			"Val32":  {"200"},
			"Val64":  {"300"},
			"UVal":   {"400"},
			"UVal8":  {"50"},
			"UVal16": {"600"},
			"UVal32": {"700"},
			"UVal64": {"800"},
		}

		var form PtrForm
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		require.NotNil(t, form.Val16)
		assert.Equal(t, int16(100), *form.Val16)
		require.NotNil(t, form.Val32)
		assert.Equal(t, int32(200), *form.Val32)
		require.NotNil(t, form.Val64)
		assert.Equal(t, int64(300), *form.Val64)
		require.NotNil(t, form.UVal)
		assert.Equal(t, uint(400), *form.UVal)
		require.NotNil(t, form.UVal8)
		assert.Equal(t, uint8(50), *form.UVal8)
		require.NotNil(t, form.UVal16)
		assert.Equal(t, uint16(600), *form.UVal16)
		require.NotNil(t, form.UVal32)
		assert.Equal(t, uint32(700), *form.UVal32)
		require.NotNil(t, form.UVal64)
		assert.Equal(t, uint64(800), *form.UVal64)
	})
}

func TestTimeFieldParsing(t *testing.T) {
	binder := NewASTBinder()

	t.Run("RFC3339 format", func(t *testing.T) {
		type TimeForm struct {
			CreatedAt time.Time
		}

		src := map[string][]string{
			"CreatedAt": {"2025-10-09T15:04:05Z"},
		}

		var form TimeForm
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		expected, _ := time.Parse(time.RFC3339, "2025-10-09T15:04:05Z")
		assert.Equal(t, expected, form.CreatedAt)
	})

	t.Run("RFC3339 with timezone offset", func(t *testing.T) {
		type TimeForm struct {
			UpdatedAt time.Time
		}

		src := map[string][]string{
			"UpdatedAt": {"2025-10-09T15:04:05-07:00"},
		}

		var form TimeForm
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		expected, _ := time.Parse(time.RFC3339, "2025-10-09T15:04:05-07:00")
		assert.Equal(t, expected, form.UpdatedAt)
	})

	t.Run("pointer to time.Time", func(t *testing.T) {
		type TimeForm struct {
			EventTime *time.Time
		}

		src := map[string][]string{
			"EventTime": {"2025-10-09T12:00:00Z"},
		}

		var form TimeForm
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		require.NotNil(t, form.EventTime)
		expected, _ := time.Parse(time.RFC3339, "2025-10-09T12:00:00Z")
		assert.Equal(t, expected, *form.EventTime)
	})

	t.Run("invalid time format returns error", func(t *testing.T) {
		type TimeForm struct {
			BadTime time.Time
		}

		src := map[string][]string{
			"BadTime": {"not-a-valid-time"},
		}

		var form TimeForm
		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err)

		multiErr, ok := errors.AsType[MultiError](err)
		require.True(t, ok)

		assert.Contains(t, multiErr["BadTime"].Error(), "BadTime")
	})

	t.Run("multiple time fields with RFC3339 format", func(t *testing.T) {
		type TimeForm struct {
			DateTime time.Time
			FullTime time.Time
		}

		src := map[string][]string{
			"DateTime": {"2025-10-09T15:04:05Z"},
			"FullTime": {"2025-01-01T00:00:00-08:00"},
		}

		var form TimeForm
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		dt, _ := time.Parse(time.RFC3339, "2025-10-09T15:04:05Z")
		assert.Equal(t, dt, form.DateTime)

		ft, _ := time.Parse(time.RFC3339, "2025-01-01T00:00:00-08:00")
		assert.Equal(t, ft, form.FullTime)
	})
}

func TestEmptyAndZeroValues(t *testing.T) {
	binder := NewASTBinder()

	t.Run("empty strings for all primitive types return zero values", func(t *testing.T) {
		type EmptyForm struct {
			Name     string
			Age      int
			Price    float64
			Active   bool
			Count    int64
			Unsigned uint
		}

		src := map[string][]string{
			"Name":     {""},
			"Age":      {""},
			"Price":    {""},
			"Active":   {""},
			"Count":    {""},
			"Unsigned": {""},
		}

		var form EmptyForm
		err := binder.Bind(context.Background(), &form, src)

		require.NoError(t, err)
		assert.Equal(t, "", form.Name)
		assert.Equal(t, 0, form.Age)
		assert.Equal(t, 0.0, form.Price)
		assert.False(t, form.Active)
		assert.Equal(t, int64(0), form.Count)
		assert.Equal(t, uint(0), form.Unsigned)
	})

	t.Run("empty string to pointer fields return pointers to zero values", func(t *testing.T) {
		type PtrForm struct {
			Name  *string
			Age   *int
			Price *float64
		}

		src := map[string][]string{
			"Name":  {""},
			"Age":   {""},
			"Price": {""},
		}

		var form PtrForm
		err := binder.Bind(context.Background(), &form, src)

		require.NoError(t, err)
		require.NotNil(t, form.Name)
		assert.Equal(t, "", *form.Name)
		require.NotNil(t, form.Age)
		assert.Equal(t, 0, *form.Age)
		require.NotNil(t, form.Price)
		assert.Equal(t, 0.0, *form.Price)
	})

	t.Run("whitespace-only strings", func(t *testing.T) {
		type WhitespaceForm struct {
			Spaces string
			Tabs   string
			Mixed  string
			Age    int
		}

		src := map[string][]string{
			"Spaces": {"   "},
			"Tabs":   {"\t\t"},
			"Mixed":  {" \t\n "},
			"Age":    {"42"},
		}

		var form WhitespaceForm
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		assert.Equal(t, "   ", form.Spaces)
		assert.Equal(t, "\t\t", form.Tabs)
		assert.Equal(t, " \t\n ", form.Mixed)
		assert.Equal(t, 42, form.Age)
	})

	t.Run("zero string values for numeric types", func(t *testing.T) {
		type ZeroForm struct {
			IntVal    int
			FloatVal  float64
			UintVal   uint
			Int8Val   int8
			Int16Val  int16
			Int32Val  int32
			Int64Val  int64
			Uint8Val  uint8
			Uint16Val uint16
			Uint32Val uint32
			Uint64Val uint64
		}

		src := map[string][]string{
			"IntVal":    {"0"},
			"FloatVal":  {"0"},
			"UintVal":   {"0"},
			"Int8Val":   {"0"},
			"Int16Val":  {"0"},
			"Int32Val":  {"0"},
			"Int64Val":  {"0"},
			"Uint8Val":  {"0"},
			"Uint16Val": {"0"},
			"Uint32Val": {"0"},
			"Uint64Val": {"0"},
		}

		var form ZeroForm
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		assert.Equal(t, 0, form.IntVal)
		assert.Equal(t, 0.0, form.FloatVal)
		assert.Equal(t, uint(0), form.UintVal)
		assert.Equal(t, int8(0), form.Int8Val)
		assert.Equal(t, int16(0), form.Int16Val)
		assert.Equal(t, int32(0), form.Int32Val)
		assert.Equal(t, int64(0), form.Int64Val)
		assert.Equal(t, uint8(0), form.Uint8Val)
		assert.Equal(t, uint16(0), form.Uint16Val)
		assert.Equal(t, uint32(0), form.Uint32Val)
		assert.Equal(t, uint64(0), form.Uint64Val)
	})

	t.Run("zero and false string values", func(t *testing.T) {
		type BoolForm struct {
			Active   bool
			Enabled  bool
			Disabled bool
			ZeroStr  bool
			EmptyStr bool
			FalseStr bool
			TrueStr  bool
			OnStr    bool
		}

		src := map[string][]string{
			"Active":   {"0"},
			"Enabled":  {"1"},
			"Disabled": {"false"},
			"ZeroStr":  {"0"},
			"FalseStr": {"false"},
			"TrueStr":  {"true"},
			"OnStr":    {"on"},
		}

		var form BoolForm
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		assert.False(t, form.Active)
		assert.True(t, form.Enabled)
		assert.False(t, form.Disabled)
		assert.False(t, form.ZeroStr)
		assert.False(t, form.FalseStr)
		assert.True(t, form.TrueStr)
		assert.True(t, form.OnStr)
	})

	t.Run("empty string in nested slice", func(t *testing.T) {
		type Item struct {
			Name string
		}
		type SliceForm struct {
			Items []Item
		}

		src := map[string][]string{
			"Items[0].Name": {"first"},
			"Items[1].Name": {""},
			"Items[2].Name": {"third"},
		}

		var form SliceForm
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		require.Len(t, form.Items, 3)
		assert.Equal(t, "first", form.Items[0].Name)
		assert.Equal(t, "", form.Items[1].Name)
		assert.Equal(t, "third", form.Items[2].Name)
	})

	t.Run("empty strings for all numeric types return zero values", func(t *testing.T) {
		type AllNumericForm struct {
			IntVal     int
			Int8Val    int8
			Int16Val   int16
			Int32Val   int32
			Int64Val   int64
			UintVal    uint
			Uint8Val   uint8
			Uint16Val  uint16
			Uint32Val  uint32
			Uint64Val  uint64
			Float32Val float32
			Float64Val float64
		}

		src := map[string][]string{
			"IntVal":     {""},
			"Int8Val":    {""},
			"Int16Val":   {""},
			"Int32Val":   {""},
			"Int64Val":   {""},
			"UintVal":    {""},
			"Uint8Val":   {""},
			"Uint16Val":  {""},
			"Uint32Val":  {""},
			"Uint64Val":  {""},
			"Float32Val": {""},
			"Float64Val": {""},
		}

		var form AllNumericForm
		err := binder.Bind(context.Background(), &form, src)

		require.NoError(t, err, "Empty strings should succeed for all numeric types")
		assert.Equal(t, 0, form.IntVal)
		assert.Equal(t, int8(0), form.Int8Val)
		assert.Equal(t, int16(0), form.Int16Val)
		assert.Equal(t, int32(0), form.Int32Val)
		assert.Equal(t, int64(0), form.Int64Val)
		assert.Equal(t, uint(0), form.UintVal)
		assert.Equal(t, uint8(0), form.Uint8Val)
		assert.Equal(t, uint16(0), form.Uint16Val)
		assert.Equal(t, uint32(0), form.Uint32Val)
		assert.Equal(t, uint64(0), form.Uint64Val)
		assert.Equal(t, float32(0), form.Float32Val)
		assert.Equal(t, float64(0), form.Float64Val)
	})

	t.Run("empty strings in nested struct with numeric fields", func(t *testing.T) {
		type QuantityConfig struct {
			Quantity int64
			Price    float64
		}
		type Quote struct {
			Name            string
			QuantityConfigs []QuantityConfig
		}

		src := map[string][]string{
			"Name":                        {"Test Quote"},
			"QuantityConfigs[0].Quantity": {"100"},
			"QuantityConfigs[0].Price":    {"19.99"},
			"QuantityConfigs[1].Quantity": {"200"},
			"QuantityConfigs[1].Price":    {"17.99"},
			"QuantityConfigs[2].Quantity": {""},
			"QuantityConfigs[2].Price":    {""},
			"QuantityConfigs[3].Quantity": {""},
			"QuantityConfigs[3].Price":    {""},
			"QuantityConfigs[4].Quantity": {"500"},
			"QuantityConfigs[4].Price":    {"12.99"},
		}

		var form Quote
		err := binder.Bind(context.Background(), &form, src)

		require.NoError(t, err, "Empty strings in nested slices should succeed")
		assert.Equal(t, "Test Quote", form.Name)
		require.Len(t, form.QuantityConfigs, 5)

		assert.Equal(t, int64(100), form.QuantityConfigs[0].Quantity)
		assert.Equal(t, 19.99, form.QuantityConfigs[0].Price)
		assert.Equal(t, int64(200), form.QuantityConfigs[1].Quantity)
		assert.Equal(t, 17.99, form.QuantityConfigs[1].Price)

		assert.Equal(t, int64(0), form.QuantityConfigs[2].Quantity)
		assert.Equal(t, 0.0, form.QuantityConfigs[2].Price)
		assert.Equal(t, int64(0), form.QuantityConfigs[3].Quantity)
		assert.Equal(t, 0.0, form.QuantityConfigs[3].Price)

		assert.Equal(t, int64(500), form.QuantityConfigs[4].Quantity)
		assert.Equal(t, 12.99, form.QuantityConfigs[4].Price)
	})

	t.Run("mixed empty and valid values in same form", func(t *testing.T) {
		type MixedForm struct {
			ValidInt   int
			EmptyInt   int
			ValidFloat float64
			EmptyFloat float64
			ValidBool  bool
			EmptyBool  bool
		}

		src := map[string][]string{
			"ValidInt":   {"42"},
			"EmptyInt":   {""},
			"ValidFloat": {"3.14"},
			"EmptyFloat": {""},
			"ValidBool":  {"true"},
			"EmptyBool":  {""},
		}

		var form MixedForm
		err := binder.Bind(context.Background(), &form, src)

		require.NoError(t, err)
		assert.Equal(t, 42, form.ValidInt)
		assert.Equal(t, 0, form.EmptyInt)
		assert.Equal(t, 3.14, form.ValidFloat)
		assert.Equal(t, 0.0, form.EmptyFloat)
		assert.True(t, form.ValidBool)
		assert.False(t, form.EmptyBool)
	})

	t.Run("empty strings with pointer numeric types", func(t *testing.T) {
		type PtrNumericForm struct {
			IntPtr     *int
			Int64Ptr   *int64
			Float64Ptr *float64
			UintPtr    *uint
		}

		src := map[string][]string{
			"IntPtr":     {""},
			"Int64Ptr":   {""},
			"Float64Ptr": {""},
			"UintPtr":    {""},
		}

		var form PtrNumericForm
		err := binder.Bind(context.Background(), &form, src)

		require.NoError(t, err, "Empty strings for pointer numeric types should succeed")
		require.NotNil(t, form.IntPtr)
		assert.Equal(t, 0, *form.IntPtr)
		require.NotNil(t, form.Int64Ptr)
		assert.Equal(t, int64(0), *form.Int64Ptr)
		require.NotNil(t, form.Float64Ptr)
		assert.Equal(t, 0.0, *form.Float64Ptr)
		require.NotNil(t, form.UintPtr)
		assert.Equal(t, uint(0), *form.UintPtr)
	})
}

func TestNumericOverflowUnderflow(t *testing.T) {
	binder := NewASTBinder()

	t.Run("int8 overflow", func(t *testing.T) {
		type Int8Form struct {
			Val int8
		}

		src := map[string][]string{
			"Val": {"128"},
		}

		var form Int8Form
		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err, "Should error on int8 overflow")
	})

	t.Run("int8 underflow", func(t *testing.T) {
		type Int8Form struct {
			Val int8
		}

		src := map[string][]string{
			"Val": {"-129"},
		}

		var form Int8Form
		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err, "Should error on int8 underflow")
	})

	t.Run("int16 overflow", func(t *testing.T) {
		type Int16Form struct {
			Val int16
		}

		src := map[string][]string{
			"Val": {"32768"},
		}

		var form Int16Form
		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err, "Should error on int16 overflow")
	})

	t.Run("int16 underflow", func(t *testing.T) {
		type Int16Form struct {
			Val int16
		}

		src := map[string][]string{
			"Val": {"-32769"},
		}

		var form Int16Form
		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err, "Should error on int16 underflow")
	})

	t.Run("int32 overflow", func(t *testing.T) {
		type Int32Form struct {
			Val int32
		}

		src := map[string][]string{
			"Val": {"2147483648"},
		}

		var form Int32Form
		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err, "Should error on int32 overflow")
	})

	t.Run("uint8 overflow", func(t *testing.T) {
		type Uint8Form struct {
			Val uint8
		}

		src := map[string][]string{
			"Val": {"256"},
		}

		var form Uint8Form
		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err, "Should error on uint8 overflow")
	})

	t.Run("uint8 negative value", func(t *testing.T) {
		type Uint8Form struct {
			Val uint8
		}

		src := map[string][]string{
			"Val": {"-1"},
		}

		var form Uint8Form
		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err, "Should error on uint8 negative value")
	})

	t.Run("uint16 overflow", func(t *testing.T) {
		type Uint16Form struct {
			Val uint16
		}

		src := map[string][]string{
			"Val": {"65536"},
		}

		var form Uint16Form
		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err, "Should error on uint16 overflow")
	})

	t.Run("uint32 overflow", func(t *testing.T) {
		type Uint32Form struct {
			Val uint32
		}

		src := map[string][]string{
			"Val": {"4294967296"},
		}

		var form Uint32Form
		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err, "Should error on uint32 overflow")
	})

	t.Run("very large number string", func(t *testing.T) {
		type IntForm struct {
			Val int
		}

		src := map[string][]string{
			"Val": {"99999999999999999999999999999999999999999999999999"},
		}

		var form IntForm
		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err, "Should error on extremely large number")
	})

	t.Run("float with invalid string values", func(t *testing.T) {
		type FloatForm struct {
			Value1 float64
			Value2 float64
		}

		src := map[string][]string{
			"Value1": {"not-a-number"},
			"Value2": {"abc123"},
		}

		var form FloatForm
		err := binder.Bind(context.Background(), &form, src)

		require.Error(t, err)
		multiErr, ok := errors.AsType[MultiError](err)
		require.True(t, ok)
		assert.Len(t, multiErr, 2)
	})

	t.Run("very long decimal", func(t *testing.T) {
		type FloatForm struct {
			Val float64
		}

		src := map[string][]string{
			"Val": {"3.141592653589793238462643383279502884197169399375105820974944592307816406286208998628034825342117067982148086513282306647093844609550582231725359408128481117450284102701938521105559644622948954930381964428810975665933446128475648233786783165271201909145648566923460348610454326648213393607260249141273724587006606315588174881520920962829254091715364367892590360011330530548820466521384146951941511609433057270365759591953092186117381932611793105118548074462379962749567351885752724891227938183011949129833673362440656643086021394946395224737190702179860943702770539217176293176752384674818467669405132000568127145263560827785771342757789609173637178721468440901224953430146549585371050792279689258923542019956112129021960864034418159813629774771309960518707211349999998372978049951059731732816096318595024459455346908302642522308253344685035261931188171010003137838752886587533208381420617177669147303598253490428755468731159562863882353787593751957781857780532171226806613001927876611195909216420199"},
		}

		var form FloatForm
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err, "Should handle very long decimal")

		assert.InDelta(t, 3.14159265358979, form.Val, 0.0001)
	})

	t.Run("boundary values succeed", func(t *testing.T) {
		type BoundaryForm struct {
			MaxInt8  int8
			MinInt8  int8
			MaxUint8 uint8
		}

		src := map[string][]string{
			"MaxInt8":  {"127"},
			"MinInt8":  {"-128"},
			"MaxUint8": {"255"},
		}

		var form BoundaryForm
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		assert.Equal(t, int8(127), form.MaxInt8)
		assert.Equal(t, int8(-128), form.MinInt8)
		assert.Equal(t, uint8(255), form.MaxUint8)
	})
}

func TestMalformedInput(t *testing.T) {
	binder := NewASTBinder()

	t.Run("malformed paths - incomplete brackets are skipped", func(t *testing.T) {
		type SimpleForm struct {
			Items []string
		}

		testCases := []struct {
			name string
			path string
		}{
			{name: "unclosed bracket", path: "Items["},
			{name: "no opening bracket", path: "Items]"},
			{name: "empty index", path: "Items[]"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				src := map[string][]string{
					tc.path: {"value"},
				}

				var form SimpleForm
				err := binder.Bind(context.Background(), &form, src)
				require.NoError(t, err, "Malformed paths should be silently skipped: "+tc.path)
				assert.Empty(t, form.Items, "Items should remain empty when path is skipped")
			})
		}
	})

	t.Run("malformed paths - special characters are skipped", func(t *testing.T) {
		type SimpleForm struct {
			Name string
		}

		testCases := []struct {
			name string
			path string
		}{
			{name: "double dot", path: "Name..Other"},
			{name: "leading dot", path: ".Name"},
			{name: "trailing dot", path: "Name."},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				src := map[string][]string{
					tc.path: {"value"},
				}

				var form SimpleForm
				err := binder.Bind(context.Background(), &form, src)
				require.NoError(t, err, "Malformed paths should be silently skipped: "+tc.path)
				assert.Equal(t, "", form.Name, "Name should remain empty when path is skipped")
			})
		}
	})

	t.Run("malformed values for int", func(t *testing.T) {
		type IntForm struct {
			Value int
		}

		testCases := []struct {
			name  string
			value string
		}{
			{name: "word", value: "twelve"},
			{name: "decimal", value: "12.5"},
			{name: "hex", value: "0x10"},
			{name: "scientific", value: "1e10"},
			{name: "with currency", value: "$19"},
			{name: "with commas", value: "1,234"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				src := map[string][]string{
					"Value": {tc.value},
				}

				var form IntForm
				err := binder.Bind(context.Background(), &form, src)
				require.Error(t, err, "Should error on malformed int: "+tc.value)
			})
		}
	})

	t.Run("malformed values for float", func(t *testing.T) {
		type FloatForm struct {
			Price float64
		}

		testCases := []struct {
			name  string
			value string
		}{
			{name: "word", value: "expensive"},
			{name: "with currency", value: "$19.99"},
			{name: "with commas", value: "1,234.56"},
			{name: "multiple dots", value: "12.34.56"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				src := map[string][]string{
					"Price": {tc.value},
				}

				var form FloatForm
				err := binder.Bind(context.Background(), &form, src)
				require.Error(t, err, "Should error on malformed float: "+tc.value)
			})
		}
	})

	t.Run("malformed values for bool", func(t *testing.T) {
		type BoolForm struct {
			Active bool
		}

		testCases := []struct {
			name  string
			value string
		}{
			{name: "yes", value: "yes"},
			{name: "no", value: "no"},
			{name: "maybe", value: "maybe"},
			{name: "number", value: "2"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				src := map[string][]string{
					"Active": {tc.value},
				}

				var form BoolForm
				err := binder.Bind(context.Background(), &form, src)
				require.Error(t, err, "Should error on malformed bool: "+tc.value)
			})
		}
	})

	t.Run("control characters in string", func(t *testing.T) {
		type StringForm struct {
			Name string
		}

		src := map[string][]string{
			"Name": {"\x00\x01\x02"},
		}

		var form StringForm
		err := binder.Bind(context.Background(), &form, src)

		require.NoError(t, err)
		assert.Equal(t, "\x00\x01\x02", form.Name)
	})

	t.Run("operators in path are silently skipped", func(t *testing.T) {
		type Form struct {
			Value int
		}

		testCases := []string{
			"Value + 1",
			"Value - 1",
			"Value * 2",
			"Value / 2",
		}

		for _, path := range testCases {
			t.Run(path, func(t *testing.T) {
				src := map[string][]string{
					path: {"42"},
				}

				var form Form
				err := binder.Bind(context.Background(), &form, src)
				require.NoError(t, err, "Unparseable paths should be silently skipped: "+path)
				assert.Equal(t, 0, form.Value, "Value should remain at zero when path is skipped")
			})
		}
	})
}

func TestUnicodeAndSpecialCharacters(t *testing.T) {
	binder := NewASTBinder()

	t.Run("unicode in string values", func(t *testing.T) {
		type UnicodeForm struct {
			Name    string
			City    string
			Message string
			Emoji   string
		}

		src := map[string][]string{
			"Name":    {"José García"},
			"City":    {"Москва"},
			"Message": {"日本語のテスト"},
			"Emoji":   {"🚀💡🎉"},
		}

		var form UnicodeForm
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		assert.Equal(t, "José García", form.Name)
		assert.Equal(t, "Москва", form.City)
		assert.Equal(t, "日本語のテスト", form.Message)
		assert.Equal(t, "🚀💡🎉", form.Emoji)
	})

	t.Run("unicode in nested structures", func(t *testing.T) {
		type Person struct {
			Name    string
			Country string
		}
		type PersonForm struct {
			People []Person
		}

		src := map[string][]string{
			"People[0].Name":    {"José"},
			"People[0].Country": {"España"},
			"People[1].Name":    {"李明"},
			"People[1].Country": {"中国"},
		}

		var form PersonForm
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		require.Len(t, form.People, 2)
		assert.Equal(t, "José", form.People[0].Name)
		assert.Equal(t, "España", form.People[0].Country)
		assert.Equal(t, "李明", form.People[1].Name)
		assert.Equal(t, "中国", form.People[1].Country)
	})

	t.Run("special characters in strings", func(t *testing.T) {
		type SpecialForm struct {
			HTML      string
			JSON      string
			SQL       string
			Quotes    string
			Backslash string
		}

		src := map[string][]string{
			"HTML":      {"<script>alert('xss')</script>"},
			"JSON":      {`{"key": "value"}`},
			"SQL":       {"'; DROP TABLE users; --"},
			"Quotes":    {`"single" and 'double'`},
			"Backslash": {`C:\Users\Test\file.txt`},
		}

		var form SpecialForm
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		assert.Equal(t, "<script>alert('xss')</script>", form.HTML)
		assert.Equal(t, `{"key": "value"}`, form.JSON)
		assert.Equal(t, "'; DROP TABLE users; --", form.SQL)
		assert.Equal(t, `"single" and 'double'`, form.Quotes)
		assert.Equal(t, `C:\Users\Test\file.txt`, form.Backslash)
	})

	t.Run("newlines and tabs in strings", func(t *testing.T) {
		type MultilineForm struct {
			Text string
		}

		src := map[string][]string{
			"Text": {"Line 1\nLine 2\tTabbed\rCarriage Return"},
		}

		var form MultilineForm
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		assert.Equal(t, "Line 1\nLine 2\tTabbed\rCarriage Return", form.Text)
	})

	t.Run("very long unicode string", func(t *testing.T) {
		type LongForm struct {
			Text string
		}

		var longText strings.Builder
		for range 100 {
			longText.WriteString("Unicode测试🚀")
		}

		src := map[string][]string{
			"Text": {longText.String()},
		}

		var form LongForm
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		assert.Equal(t, longText.String(), form.Text)
		assert.Greater(t, len(form.Text), 1000, "Should handle long unicode strings")
	})

	t.Run("zero-width characters", func(t *testing.T) {
		type ZeroWidthForm struct {
			Text string
		}

		src := map[string][]string{
			"Text": {"Hello\u200BWorld"},
		}

		var form ZeroWidthForm
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		assert.Contains(t, form.Text, "\u200B", "Should preserve zero-width characters")
	})

	t.Run("right-to-left text", func(t *testing.T) {
		type RTLForm struct {
			Arabic string
			Hebrew string
		}

		src := map[string][]string{
			"Arabic": {"مرحبا بك"},
			"Hebrew": {"שלום"},
		}

		var form RTLForm
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		assert.Equal(t, "مرحبا بك", form.Arabic)
		assert.Equal(t, "שלום", form.Hebrew)
	})
}

func TestPointerChainDepth(t *testing.T) {
	binder := NewASTBinder()

	t.Run("double pointer to int - not supported", func(t *testing.T) {
		type DoublePtr struct {
			Value **int
		}

		src := map[string][]string{
			"Value": {"42"},
		}

		var form DoublePtr
		err := binder.Bind(context.Background(), &form, src)

		require.Error(t, err, "Double pointers are not supported")
		multiErr, ok := errors.AsType[MultiError](err)
		require.True(t, ok)
		assert.Contains(t, multiErr["Value"].Error(), "unsupported type")
	})

	t.Run("single pointer works correctly", func(t *testing.T) {
		type SinglePtr struct {
			Value *int
		}

		src := map[string][]string{
			"Value": {"42"},
		}

		var form SinglePtr
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		require.NotNil(t, form.Value)
		assert.Equal(t, 42, *form.Value)
	})
}

func TestTextUnmarshalerErrors(t *testing.T) {
	binder := NewASTBinder()

	t.Run("TextUnmarshaler returns error", func(t *testing.T) {
		type FailingType struct{}

		binder.RegisterConverter(reflect.TypeFor[FailingType](), func(value string) (reflect.Value, error) {
			return reflect.Value{}, fmt.Errorf("intentional failure: %s", value)
		})

		type Form struct {
			Field FailingType
		}

		src := map[string][]string{
			"Field": {"test"},
		}

		var form Form
		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err)

		multiErr, ok := errors.AsType[MultiError](err)
		require.True(t, ok)
		assert.Contains(t, multiErr["Field"].Error(), "intentional failure")
	})

	t.Run("TextUnmarshaler with invalid format", func(t *testing.T) {
		type TimeForm struct {
			Date time.Time
		}

		src := map[string][]string{
			"Date": {"invalid-date-format"},
		}

		var form TimeForm
		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err)

		multiErr, ok := errors.AsType[MultiError](err)
		require.True(t, ok)

		assert.Contains(t, multiErr["Date"].Error(), "Date")
	})
}

func TestConverterPrecedence(t *testing.T) {
	binder := NewASTBinder()

	t.Run("custom converter overrides default for int", func(t *testing.T) {
		type CustomInt int

		binder.RegisterConverter(reflect.TypeFor[CustomInt](), func(value string) (reflect.Value, error) {
			i, err := strconv.Atoi(value)
			if err != nil {
				return reflect.Value{}, err
			}
			return reflect.ValueOf(CustomInt(i + 1000)), nil
		})

		type Form struct {
			Value CustomInt
		}

		src := map[string][]string{
			"Value": {"42"},
		}

		var form Form
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		assert.Equal(t, CustomInt(1042), form.Value)
	})

	t.Run("custom converter for time.Time", func(t *testing.T) {

		testBinder := NewASTBinder()

		fixedTime := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
		testBinder.RegisterConverter(reflect.TypeFor[time.Time](), func(value string) (reflect.Value, error) {
			return reflect.ValueOf(fixedTime), nil
		})

		type TimeForm struct {
			Date time.Time
		}

		src := map[string][]string{
			"Date": {"any-value"},
		}

		var form TimeForm
		err := testBinder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		assert.Equal(t, fixedTime, form.Date)
	})

	t.Run("custom converter with error", func(t *testing.T) {
		type CustomType string

		testBinder := NewASTBinder()
		testBinder.RegisterConverter(reflect.TypeFor[CustomType](), func(value string) (reflect.Value, error) {
			if value == "invalid" {
				return reflect.Value{}, errors.New("custom error: invalid value")
			}
			return reflect.ValueOf(CustomType("custom-" + value)), nil
		})

		type Form struct {
			Field CustomType
		}

		src := map[string][]string{
			"Field": {"invalid"},
		}

		var form Form
		err := testBinder.Bind(context.Background(), &form, src)
		require.Error(t, err)

		multiErr, ok := errors.AsType[MultiError](err)
		require.True(t, ok)
		assert.Contains(t, multiErr["Field"].Error(), "custom error")
	})
}

func TestBind_NestedSlicePath(t *testing.T) {

	type TestNestedQuantityConfig struct {
		Price float64 `json:"price"`
	}

	type TestNestedQuote struct {
		QuantityConfigs []TestNestedQuantityConfig `json:"quantity_configs"`
	}

	type TestNestedAction struct {
		Quote TestNestedQuote `json:"quote"`
	}

	b := NewASTBinder()
	var dst TestNestedAction
	src := map[string][]string{
		"quote.quantity_configs[0].price": {"123.45"},
	}

	err := b.Bind(context.Background(), &dst, src)

	assert.NoError(t, err, "Binding to a nested slice path should not produce an error")

	assert.Len(t, dst.Quote.QuantityConfigs, 1, "Slice should have been auto-grown to length 1")
	if len(dst.Quote.QuantityConfigs) > 0 {
		assert.Equal(t, 123.45, dst.Quote.QuantityConfigs[0].Price, "The nested slice field should be correctly populated")
	}
}

func TestBind_MapPathWithStringKey(t *testing.T) {

	type TestMapValue struct {
		Setting string `json:"setting"`
	}

	type TestMapAction struct {
		Config map[string]TestMapValue `json:"config"`
	}

	b := NewASTBinder()
	var dst TestMapAction
	src := map[string][]string{
		`config["theme"].setting`: {"dark"},
	}

	err := b.Bind(context.Background(), &dst, src)

	assert.NoError(t, err)
	require.NotNil(t, dst.Config, "Map should be initialised")
	require.Contains(t, dst.Config, "theme", "Map should contain the key 'theme'")
	assert.Equal(t, "dark", dst.Config["theme"].Setting)
}

func TestBind_MapPathWithIntegerKey(t *testing.T) {

	type TestMapValue struct {
		Name string `json:"name"`
	}

	type TestMapAction struct {
		Users map[int]TestMapValue `json:"users"`
	}

	b := NewASTBinder()
	var dst TestMapAction
	src := map[string][]string{
		`users[101].name`: {"Alice"},
	}

	err := b.Bind(context.Background(), &dst, src)

	assert.NoError(t, err)
	require.NotNil(t, dst.Users, "Map should be initialised")
	require.Contains(t, dst.Users, 101, "Map should contain the key 101")
	assert.Equal(t, "Alice", dst.Users[101].Name)
}

func TestBind_MixedSliceAndMapPath(t *testing.T) {

	type TestMixedItem struct {
		Properties map[string]string `json:"properties"`
	}

	type TestMixedAction struct {
		Items []TestMixedItem `json:"items"`
	}

	b := NewASTBinder()
	var dst TestMixedAction
	src := map[string][]string{
		`items[0].properties["color"]`: {"blue"},
	}

	err := b.Bind(context.Background(), &dst, src)

	assert.NoError(t, err)
	require.Len(t, dst.Items, 1, "Slice should have been grown")
	require.NotNil(t, dst.Items[0].Properties, "Nested map should have been initialised")
	require.Contains(t, dst.Items[0].Properties, "color")
	assert.Equal(t, "blue", dst.Items[0].Properties["color"])
}

func TestBind_MixedSliceAndMapPathSingleQuote(t *testing.T) {

	type TestMixedItem struct {
		Properties map[string]string `json:"properties"`
	}

	type TestMixedAction struct {
		Items []TestMixedItem `json:"items"`
	}

	b := NewASTBinder()
	var dst TestMixedAction
	src := map[string][]string{
		`items[0].properties['color']`: {"blue"},
	}

	err := b.Bind(context.Background(), &dst, src)

	assert.NoError(t, err)
	require.Len(t, dst.Items, 1, "Slice should have been grown")
	require.NotNil(t, dst.Items[0].Properties, "Nested map should have been initialised")
	require.Contains(t, dst.Items[0].Properties, "color")
	assert.Equal(t, "blue", dst.Items[0].Properties["color"])
}

func TestBind_TagPrecedence(t *testing.T) {

	type PrecedenceTestStruct struct {
		FieldA string `bind:"field_a_bind" json:"field_a_json"`

		FieldB string `json:"field_b_json"`

		FieldC string

		FieldD string `bind:"-" json:"field_d_json"`

		FieldE string `json:"-"`

		FieldF string `bind:"field_f_bind" json:"-"`
	}

	b := NewASTBinder()
	var dst PrecedenceTestStruct
	src := map[string][]string{
		"field_a_bind": {"Value A via bind"},
		"field_b_json": {"Value B via json"},
		"FieldC":       {"Value C via name"},
		"field_d_json": {"Value D ignored"},
		"FieldE":       {"Value E ignored"},
		"field_f_bind": {"Value F override"},
	}

	err := b.Bind(context.Background(), &dst, src)

	assert.NoError(t, err)

	assert.Equal(t, "Value A via bind", dst.FieldA, "Should have used 'bind' tag over 'json' tag")

	assert.Equal(t, "Value B via json", dst.FieldB, "Should have used 'json' tag as fallback")

	assert.Equal(t, "Value C via name", dst.FieldC, "Should have used Go field name as final fallback")

	assert.Empty(t, dst.FieldD, "Should be empty because bind:\"-\" was present")

	assert.Empty(t, dst.FieldE, "Should be empty because json:\"-\" was present")

	assert.Equal(t, "Value F override", dst.FieldF, "Should have used 'bind' tag even when json:\"-\" was present")
}

func TestBind_BuiltInConverters(t *testing.T) {
	binder := NewASTBinder()

	t.Run("UUID converter - valid UUID", func(t *testing.T) {
		type UUIDForm struct {
			ID uuid.UUID `json:"id"`
		}

		var form UUIDForm
		src := map[string][]string{
			"id": {"550e8400-e29b-41d4-a716-446655440000"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		expectedUUID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		assert.Equal(t, expectedUUID, form.ID)
	})

	t.Run("UUID converter - empty string", func(t *testing.T) {
		type UUIDForm struct {
			ID uuid.UUID `json:"id"`
		}

		var form UUIDForm
		src := map[string][]string{
			"id": {""},
		}

		err := binder.Bind(context.Background(), &form, src)

		require.Error(t, err)
		multiErr, ok := errors.AsType[MultiError](err)
		require.True(t, ok)
		assert.Contains(t, multiErr["id"].Error(), "failed to unmarshal text")
	})

	t.Run("UUID converter - invalid UUID", func(t *testing.T) {
		type UUIDForm struct {
			ID uuid.UUID `json:"id"`
		}

		var form UUIDForm
		src := map[string][]string{
			"id": {"not-a-valid-uuid"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err)
		multiErr, ok := errors.AsType[MultiError](err)
		require.True(t, ok)

		assert.Contains(t, multiErr["id"].Error(), "failed to unmarshal text")
	})

	t.Run("URL converter - valid URL", func(t *testing.T) {
		type URLForm struct {
			Website *url.URL `json:"website"`
		}

		var form URLForm
		src := map[string][]string{
			"website": {"https://example.com/path?query=value"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		require.NotNil(t, form.Website)
		assert.Equal(t, "https", form.Website.Scheme)
		assert.Equal(t, "example.com", form.Website.Host)
		assert.Equal(t, "/path", form.Website.Path)
		assert.Equal(t, "query=value", form.Website.RawQuery)
	})

	t.Run("URL converter - empty string", func(t *testing.T) {
		type URLForm struct {
			Website *url.URL `json:"website"`
		}

		var form URLForm
		src := map[string][]string{
			"website": {""},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		require.NotNil(t, form.Website)
		assert.Equal(t, "", form.Website.String())
	})

	t.Run("Duration converter - valid duration", func(t *testing.T) {
		type DurationForm struct {
			Timeout time.Duration `json:"timeout"`
		}

		var form DurationForm
		src := map[string][]string{
			"timeout": {"5m30s"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		assert.Equal(t, 5*time.Minute+30*time.Second, form.Timeout)
	})

	t.Run("Duration converter - empty string", func(t *testing.T) {
		type DurationForm struct {
			Timeout time.Duration `json:"timeout"`
		}

		var form DurationForm
		src := map[string][]string{
			"timeout": {""},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		assert.Equal(t, time.Duration(0), form.Timeout)
	})

	t.Run("Duration converter - invalid duration", func(t *testing.T) {
		type DurationForm struct {
			Timeout time.Duration `json:"timeout"`
		}

		var form DurationForm
		src := map[string][]string{
			"timeout": {"invalid"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err)
		multiErr, ok := errors.AsType[MultiError](err)
		require.True(t, ok)
		assert.Contains(t, multiErr["timeout"].Error(), "could not parse")
	})

	t.Run("IP converter - valid IPv4", func(t *testing.T) {
		type IPForm struct {
			Address net.IP `json:"address"`
		}

		var form IPForm
		src := map[string][]string{
			"address": {"192.168.1.1"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		assert.Equal(t, "192.168.1.1", form.Address.String())
	})

	t.Run("IP converter - valid IPv6", func(t *testing.T) {
		type IPForm struct {
			Address net.IP `json:"address"`
		}

		var form IPForm
		src := map[string][]string{
			"address": {"2001:0db8:85a3:0000:0000:8a2e:0370:7334"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		assert.Equal(t, "2001:db8:85a3::8a2e:370:7334", form.Address.String())
	})

	t.Run("IP converter - empty string", func(t *testing.T) {
		type IPForm struct {
			Address net.IP `json:"address"`
		}

		var form IPForm
		src := map[string][]string{
			"address": {""},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		assert.Nil(t, form.Address)
	})

	t.Run("IP converter - invalid IP", func(t *testing.T) {
		type IPForm struct {
			Address net.IP `json:"address"`
		}

		var form IPForm
		src := map[string][]string{
			"address": {"not-an-ip"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err)
		multiErr, ok := errors.AsType[MultiError](err)
		require.True(t, ok)

		assert.Contains(t, multiErr["address"].Error(), "failed to unmarshal text")
	})

	t.Run("MailAddress converter - valid email", func(t *testing.T) {
		type MailForm struct {
			Email *mail.Address `json:"email"`
		}

		var form MailForm
		src := map[string][]string{
			"email": {"John Doe <john@example.com>"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		require.NotNil(t, form.Email)
		assert.Equal(t, "John Doe", form.Email.Name)
		assert.Equal(t, "john@example.com", form.Email.Address)
	})

	t.Run("MailAddress converter - simple email", func(t *testing.T) {
		type MailForm struct {
			Email *mail.Address `json:"email"`
		}

		var form MailForm
		src := map[string][]string{
			"email": {"simple@example.com"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		require.NotNil(t, form.Email)
		assert.Equal(t, "", form.Email.Name)
		assert.Equal(t, "simple@example.com", form.Email.Address)
	})

	t.Run("MailAddress converter - empty string", func(t *testing.T) {
		type MailForm struct {
			Email *mail.Address `json:"email"`
		}

		var form MailForm
		src := map[string][]string{
			"email": {""},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		require.NotNil(t, form.Email)
		assert.Equal(t, "", form.Email.Name)
		assert.Equal(t, "", form.Email.Address)
	})

	t.Run("MailAddress converter - invalid email", func(t *testing.T) {
		type MailForm struct {
			Email *mail.Address `json:"email"`
		}

		var form MailForm
		src := map[string][]string{
			"email": {"not-an-email"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err)
		multiErr, ok := errors.AsType[MultiError](err)
		require.True(t, ok)
		assert.Contains(t, multiErr["email"].Error(), "could not parse")
	})

	t.Run("Color converter - 6-digit hex", func(t *testing.T) {
		type ColorForm struct {
			Background color.Color `json:"background"`
		}

		var form ColorForm
		src := map[string][]string{
			"background": {"#FF5733"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		r, g, b, a := form.Background.RGBA()

		assert.Equal(t, uint32(0xFF), r>>8)
		assert.Equal(t, uint32(0x57), g>>8)
		assert.Equal(t, uint32(0x33), b>>8)
		assert.Equal(t, uint32(0xFF), a>>8)
	})

	t.Run("Color converter - 3-digit hex", func(t *testing.T) {
		type ColorForm struct {
			Background color.Color `json:"background"`
		}

		var form ColorForm
		src := map[string][]string{
			"background": {"#F53"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		r, g, b, a := form.Background.RGBA()
		assert.Equal(t, uint32(0xFF), r>>8)
		assert.Equal(t, uint32(0x55), g>>8)
		assert.Equal(t, uint32(0x33), b>>8)
		assert.Equal(t, uint32(0xFF), a>>8)
	})

	t.Run("Color converter - 8-digit hex with alpha", func(t *testing.T) {
		type ColorForm struct {
			Background color.Color `json:"background"`
		}

		var form ColorForm
		src := map[string][]string{
			"background": {"#FF573380"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		r, g, b, a := form.Background.RGBA()
		assert.Equal(t, uint32(0xFF), r>>8)
		assert.Equal(t, uint32(0x57), g>>8)
		assert.Equal(t, uint32(0x33), b>>8)
		assert.Equal(t, uint32(0x80), a>>8)
	})

	t.Run("Color converter - without hash prefix", func(t *testing.T) {
		type ColorForm struct {
			Background color.Color `json:"background"`
		}

		var form ColorForm
		src := map[string][]string{
			"background": {"FF5733"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		r, g, b, _ := form.Background.RGBA()
		assert.Equal(t, uint32(0xFF), r>>8)
		assert.Equal(t, uint32(0x57), g>>8)
		assert.Equal(t, uint32(0x33), b>>8)
	})

	t.Run("Color converter - empty string", func(t *testing.T) {
		type ColorForm struct {
			Background color.Color `json:"background"`
		}

		var form ColorForm
		src := map[string][]string{
			"background": {""},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		r, g, b, a := form.Background.RGBA()
		assert.Equal(t, uint32(0), r)
		assert.Equal(t, uint32(0), g)
		assert.Equal(t, uint32(0), b)
		assert.Equal(t, uint32(0), a)
	})

	t.Run("Color converter - invalid format", func(t *testing.T) {
		type ColorForm struct {
			Background color.Color `json:"background"`
		}

		var form ColorForm
		src := map[string][]string{
			"background": {"invalid"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err)
		multiErr, ok := errors.AsType[MultiError](err)
		require.True(t, ok)
		assert.Contains(t, multiErr["background"].Error(), "could not parse")
	})

	t.Run("All converters together in one struct", func(t *testing.T) {
		type CompleteForm struct {
			BgColor color.Color   `json:"bg_color"`
			Website *url.URL      `json:"website"`
			Email   *mail.Address `json:"email"`
			IP      net.IP        `json:"ip"`
			ID      uuid.UUID     `json:"id"`
			Timeout time.Duration `json:"timeout"`
		}

		var form CompleteForm
		src := map[string][]string{
			"id":       {"550e8400-e29b-41d4-a716-446655440000"},
			"website":  {"https://example.com"},
			"timeout":  {"10s"},
			"ip":       {"192.168.1.1"},
			"email":    {"test@example.com"},
			"bg_color": {"#FF5733"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), form.ID)
		assert.Equal(t, "https://example.com", form.Website.String())
		assert.Equal(t, 10*time.Second, form.Timeout)
		assert.Equal(t, "192.168.1.1", form.IP.String())
		assert.Equal(t, "test@example.com", form.Email.Address)
		r, g, b, _ := form.BgColor.RGBA()
		assert.Equal(t, uint32(0xFF), r>>8)
		assert.Equal(t, uint32(0x57), g>>8)
		assert.Equal(t, uint32(0x33), b>>8)
	})
}

func TestBind_MapStringAny(t *testing.T) {
	binder := NewASTBinder()

	t.Run("simple map[string]any binding with bracket notation", func(t *testing.T) {
		type DynamicForm struct {
			Fields map[string]any `json:"fields"`
		}

		var form DynamicForm
		src := map[string][]string{
			`fields["title"]`:   {"My Page Title"},
			`fields["content"]`: {"Some content here"},
			`fields["count"]`:   {"42"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		require.NotNil(t, form.Fields)
		assert.Equal(t, "My Page Title", form.Fields["title"])
		assert.Equal(t, "Some content here", form.Fields["content"])
		assert.Equal(t, "42", form.Fields["count"])
	})

	t.Run("dot notation does not work for map keys (requires bracket notation)", func(t *testing.T) {

		type DynamicForm struct {
			Fields map[string]any `json:"fields"`
		}

		var form DynamicForm
		src := map[string][]string{
			"Fields.title": {"My Page Title"},
		}

		err := binder.Bind(context.Background(), &form, src)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "non-struct type")
	})

	t.Run("map[string]any with empty values", func(t *testing.T) {
		type DynamicForm struct {
			Fields map[string]any `json:"fields"`
		}

		var form DynamicForm
		src := map[string][]string{
			`fields["empty"]`:    {""},
			`fields["nonempty"]`: {"value"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		require.NotNil(t, form.Fields)
		assert.Equal(t, "", form.Fields["empty"])
		assert.Equal(t, "value", form.Fields["nonempty"])
	})

	t.Run("map[string]any with special characters in values", func(t *testing.T) {
		type DynamicForm struct {
			Fields map[string]any `json:"fields"`
		}

		var form DynamicForm
		src := map[string][]string{
			`fields["json"]`:    {`{"key": "value", "nested": {"a": 1}}`},
			`fields["unicode"]`: {"Hello, 世界! 🎉"},
			`fields["html"]`:    {"<p>Some HTML</p>"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		require.NotNil(t, form.Fields)
		assert.Equal(t, `{"key": "value", "nested": {"a": 1}}`, form.Fields["json"])
		assert.Equal(t, "Hello, 世界! 🎉", form.Fields["unicode"])
		assert.Equal(t, "<p>Some HTML</p>", form.Fields["html"])
	})

	t.Run("map[string]any alongside typed fields", func(t *testing.T) {
		type PageUpdateAction struct {
			Fields        map[string]any `json:"fields"`
			EnvironmentID string         `json:"environment_id"`
			BlueprintID   string         `json:"blueprint_id"`
			PageID        string         `json:"page_id"`
			Published     bool           `json:"published"`
		}

		var form PageUpdateAction
		src := map[string][]string{
			"environment_id":   {"env-123"},
			"blueprint_id":     {"bp-456"},
			"page_id":          {"page-789"},
			`fields["title"]`:  {"My Page"},
			`fields["author"]`: {"John Doe"},
			"published":        {"true"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		assert.Equal(t, "env-123", form.EnvironmentID)
		assert.Equal(t, "bp-456", form.BlueprintID)
		assert.Equal(t, "page-789", form.PageID)
		assert.True(t, form.Published)
		require.NotNil(t, form.Fields)
		assert.Equal(t, "My Page", form.Fields["title"])
		assert.Equal(t, "John Doe", form.Fields["author"])
	})

	t.Run("map[string]any with many fields", func(t *testing.T) {
		type DynamicForm struct {
			Fields map[string]any `json:"fields"`
		}

		var form DynamicForm
		src := map[string][]string{}
		for i := range 50 {
			key := fmt.Sprintf(`fields["field_%d"]`, i)
			src[key] = []string{fmt.Sprintf("value_%d", i)}
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		require.NotNil(t, form.Fields)
		assert.Len(t, form.Fields, 50)
		for i := range 50 {
			assert.Equal(t, fmt.Sprintf("value_%d", i), form.Fields[fmt.Sprintf("field_%d", i)])
		}
	})
}

func TestBind_PerCallOptions(t *testing.T) {
	t.Run("per-call options override global defaults", func(t *testing.T) {
		binder := NewASTBinder()
		binder.SetMaxFieldCount(1000)

		var form SimpleForm
		src := map[string][]string{
			"Name":     {"Alice"},
			"Age":      {"30"},
			"IsActive": {"true"},
		}

		err := binder.Bind(context.Background(), &form, src,
			WithMaxFieldCount(2),
			WithMaxSliceSize(50),
			WithMaxPathDepth(10),
			WithMaxPathLength(200),
			WithMaxValueLength(500),
			IgnoreUnknownKeys(true),
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum limit of 2")
	})

	t.Run("per-call options do not affect subsequent calls", func(t *testing.T) {
		binder := NewASTBinder()
		binder.SetMaxFieldCount(1000)

		var form1 SimpleForm
		src := map[string][]string{
			"Name": {"Alice"},
			"Age":  {"30"},
		}

		err := binder.Bind(context.Background(), &form1, src, WithMaxFieldCount(1))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum limit of 1")

		var form2 SimpleForm
		err = binder.Bind(context.Background(), &form2, src)
		require.NoError(t, err)
		assert.Equal(t, "Alice", form2.Name)
	})

	t.Run("per-call MaxSliceSize limits slice growth", func(t *testing.T) {
		binder := NewASTBinder()

		var form SliceForm
		src := map[string][]string{
			"Tags[100]": {"overflow"},
		}

		err := binder.Bind(context.Background(), &form, src, WithMaxSliceSize(10))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum allowed size of 10")
	})

	t.Run("per-call MaxPathDepth limits nesting", func(t *testing.T) {
		binder := NewASTBinder()

		type DeepForm struct {
			L1 struct {
				L2 struct {
					L3 struct {
						Value string
					}
				}
			}
		}

		var form DeepForm
		src := map[string][]string{
			"L1.L2.L3.Value": {"deep"},
		}

		err := binder.Bind(context.Background(), &form, src, WithMaxPathDepth(1))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "depth exceeds maximum limit of 1")
	})

	t.Run("per-call MaxPathLength limits path length", func(t *testing.T) {
		binder := NewASTBinder()

		var form SimpleForm
		longPath := strings.Repeat("a", 200)
		src := map[string][]string{
			longPath: {"value"},
		}

		err := binder.Bind(context.Background(), &form, src, WithMaxPathLength(50))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "path length exceeds maximum limit of 50")
	})

	t.Run("per-call MaxValueLength limits value size", func(t *testing.T) {
		binder := NewASTBinder()

		var form SimpleForm
		longValue := strings.Repeat("v", 500)
		src := map[string][]string{
			"Name": {longValue},
		}

		err := binder.Bind(context.Background(), &form, src, WithMaxValueLength(100))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "value length exceeds maximum limit of 100")
	})

	t.Run("per-call IgnoreUnknownKeys suppresses errors for AST paths", func(t *testing.T) {
		binder := NewASTBinder()

		var form NestedForm
		src := map[string][]string{
			"User.Name":        {"Alice"},
			"User.NonExistent": {"ignored"},
		}

		err := binder.Bind(context.Background(), &form, src, IgnoreUnknownKeys(true))
		require.NoError(t, err)
		assert.Equal(t, "Alice", form.User.Name)
	})

	t.Run("partial options leave others as defaults", func(t *testing.T) {
		binder := NewASTBinder()
		binder.SetMaxFieldCount(1000)

		var form SimpleForm
		longValue := strings.Repeat("v", 200)
		src := map[string][]string{
			"Name": {longValue},
		}

		err := binder.Bind(context.Background(), &form, src, WithMaxValueLength(50))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "value length exceeds maximum limit of 50")
	})
}

func TestBind_IgnoreUnknownKeys(t *testing.T) {
	t.Run("SetIgnoreUnknownKeys true silently ignores unknown nested fields", func(t *testing.T) {
		binder := NewASTBinder()
		binder.SetIgnoreUnknownKeys(true)

		var form NestedForm
		src := map[string][]string{
			"User.Name":        {"Alice"},
			"User.NonExistent": {"should be ignored"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		assert.Equal(t, "Alice", form.User.Name)
	})

	t.Run("SetIgnoreUnknownKeys false returns error for unknown nested fields", func(t *testing.T) {
		binder := NewASTBinder()
		binder.SetIgnoreUnknownKeys(false)

		var form NestedForm
		src := map[string][]string{
			"User.NonExistent": {"value"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "field not found")
	})

	t.Run("handleUncachedField resolves field by Go name when JSON tag differs", func(t *testing.T) {
		type TaggedForm struct {
			Title string `json:"page_title"`
		}

		binder := NewASTBinder()

		type Wrapper struct {
			Inner TaggedForm
		}

		var form Wrapper
		src := map[string][]string{
			"Inner.Title": {"My Page"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		assert.Equal(t, "My Page", form.Inner.Title)
	})
}

func TestWalkerResolverEdgeCases(t *testing.T) {
	t.Run("map[string]*string pointer element binding", func(t *testing.T) {
		type PtrStringMapForm struct {
			MyMap map[string]*string
		}

		binder := NewASTBinder()
		form := PtrStringMapForm{MyMap: make(map[string]*string)}
		src := map[string][]string{
			`MyMap["greeting"]`: {"hello"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		require.NotNil(t, form.MyMap["greeting"])
		assert.Equal(t, "hello", *form.MyMap["greeting"])
	})

	t.Run("chained map[string]any with missing intermediate key", func(t *testing.T) {
		type ChainedMapForm struct {
			Fields map[string]any
		}

		binder := NewASTBinder()
		form := ChainedMapForm{Fields: make(map[string]any)}
		src := map[string][]string{
			`Fields["new_key"]["nested"]`: {"value"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		inner, ok := form.Fields["new_key"].(map[string]any)
		require.True(t, ok, "intermediate value should be map[string]any")
		assert.Equal(t, "value", inner["nested"])
	})

	t.Run("chained map[string]any with nil intermediate value", func(t *testing.T) {
		type ChainedMapForm struct {
			Fields map[string]any
		}

		binder := NewASTBinder()
		form := ChainedMapForm{Fields: map[string]any{
			"nilkey": nil,
		}}
		src := map[string][]string{
			`Fields["nilkey"]["sub"]`: {"created"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)

		inner, ok := form.Fields["nilkey"].(map[string]any)
		require.True(t, ok, "nil intermediate should be replaced with map[string]any")
		assert.Equal(t, "created", inner["sub"])
	})

	t.Run("findFieldByJSONTag skips json dash fields", func(t *testing.T) {
		type SkipFieldForm struct {
			Visible string `json:"visible"`
			Hidden  string `json:"-"`
			Active  bool   `json:"active,omitempty"`
		}

		binder := NewASTBinder()

		type Wrapper struct {
			Inner SkipFieldForm
		}

		var form Wrapper
		src := map[string][]string{
			"Inner.visible": {"shown"},
			"Inner.active":  {"true"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		assert.Equal(t, "shown", form.Inner.Visible)
		assert.True(t, form.Inner.Active)
		assert.Empty(t, form.Inner.Hidden)
	})

	t.Run("resolveSliceIndex with non-integer index errors", func(t *testing.T) {
		binder := NewASTBinder()

		var form SliceForm
		src := map[string][]string{
			`Tags["key"]`: {"value"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err)
	})

	t.Run("resolveMapIndex initialises nil map", func(t *testing.T) {
		type MapStructForm struct {
			Data map[string]Item
		}

		binder := NewASTBinder()
		var form MapStructForm

		src := map[string][]string{
			`Data["first"].Name`:  {"Widget"},
			`Data["first"].Price`: {"9.99"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		require.NotNil(t, form.Data)
		assert.Equal(t, "Widget", form.Data["first"].Name)
		assert.Equal(t, 9.99, form.Data["first"].Price)
	})

	t.Run("resolveMapIndex handles nil pointer element", func(t *testing.T) {
		type NilPtrMapForm struct {
			MapField map[string]*Item
		}

		binder := NewASTBinder()
		form := NilPtrMapForm{
			MapField: map[string]*Item{
				"key": nil,
			},
		}

		src := map[string][]string{
			`MapField["key"].Name`: {"Created"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		require.NotNil(t, form.MapField["key"])
		assert.Equal(t, "Created", form.MapField["key"].Name)
	})

	t.Run("RegisterConverter with pointer type dereferences to element", func(t *testing.T) {
		binder := NewASTBinder()

		called := false
		binder.RegisterConverter(reflect.TypeFor[*time.Time](), func(value string) (reflect.Value, error) {
			called = true
			t2 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
			return reflect.ValueOf(t2), nil
		})

		type TimeForm struct {
			Created time.Time
		}

		var form TimeForm
		src := map[string][]string{
			"Created": {"anything"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		assert.True(t, called, "converter registered with pointer type should be used")
		assert.Equal(t, 2025, form.Created.Year())
	})

	t.Run("bindFields skips empty value slices", func(t *testing.T) {
		binder := NewASTBinder()

		var form SimpleForm
		form.Name = "original"
		src := map[string][]string{
			"Name": {},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		assert.Equal(t, "original", form.Name, "field should remain unchanged with empty value slice")
	})
}

func TestMultiError_And_UintMapKey(t *testing.T) {
	t.Run("MultiError.Error with empty map", func(t *testing.T) {
		var me = MultiError{}
		result := me.Error()
		assert.Equal(t, "(0 errors)", result)
	})

	t.Run("map with uint key type", func(t *testing.T) {
		type UintKeyMapForm struct {
			Counts map[uint]string
		}

		binder := NewASTBinder()
		form := UintKeyMapForm{Counts: make(map[uint]string)}

		src := map[string][]string{
			`Counts[42]`: {"forty-two"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		assert.Equal(t, "forty-two", form.Counts[42])
	})

	t.Run("map with uint64 key type", func(t *testing.T) {
		type Uint64KeyMapForm struct {
			Lookup map[uint64]string
		}

		binder := NewASTBinder()
		form := Uint64KeyMapForm{Lookup: make(map[uint64]string)}

		src := map[string][]string{
			`Lookup[999]`: {"found"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		assert.Equal(t, "found", form.Lookup[999])
	})
}

func TestBind_WeirdInputs(t *testing.T) {
	t.Run("multi-value form field uses last value", func(t *testing.T) {
		binder := NewASTBinder()

		var form SimpleForm
		src := map[string][]string{
			"Name": {"first", "second", "third"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		assert.Equal(t, "third", form.Name)
	})

	t.Run("pointer to time.Duration", func(t *testing.T) {
		type DurationPtrForm struct {
			Timeout *time.Duration
		}

		binder := NewASTBinder()
		var form DurationPtrForm

		src := map[string][]string{
			"Timeout": {"5s"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		require.NotNil(t, form.Timeout)
		assert.Equal(t, 5*time.Second, *form.Timeout)
	})

	t.Run("time parsing with non-RFC3339 formats", func(t *testing.T) {
		type TimeForm struct {
			Date time.Time
		}

		testCases := []struct {
			expected time.Time
			name     string
			input    string
		}{
			{
				name:     "date only format",
				input:    "2025-06-15",
				expected: time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
			},
			{
				name:     "datetime with space",
				input:    "2025-06-15 14:30:00",
				expected: time.Date(2025, 6, 15, 14, 30, 0, 0, time.UTC),
			},
			{
				name:     "UK date format",
				input:    "15/06/2025",
				expected: time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				binder := NewASTBinder()
				var form TimeForm
				src := map[string][]string{
					"Date": {tc.input},
				}
				err := binder.Bind(context.Background(), &form, src)
				require.NoError(t, err)
				assert.True(t, tc.expected.Equal(form.Date), "expected %v, got %v", tc.expected, form.Date)
			})
		}
	})

	t.Run("8-digit RRGGBBAA colour", func(t *testing.T) {
		type ColourForm struct {
			Tint color.Color
		}

		binder := NewASTBinder()
		var form ColourForm

		src := map[string][]string{
			"Tint": {"#FF000080"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		r, _, _, a := form.Tint.RGBA()
		assert.NotZero(t, r)
		assert.NotEqual(t, uint32(0xFFFF), a, "alpha should not be fully opaque")
	})

	t.Run("struct with all nil pointer fields initialised", func(t *testing.T) {
		type AllPtrForm struct {
			Name   *string
			Age    *int
			Score  *float64
			Active *bool
		}

		binder := NewASTBinder()
		var form AllPtrForm

		src := map[string][]string{
			"Name":   {"Bob"},
			"Age":    {"25"},
			"Score":  {"99.5"},
			"Active": {"true"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		require.NotNil(t, form.Name)
		require.NotNil(t, form.Age)
		require.NotNil(t, form.Score)
		require.NotNil(t, form.Active)
		assert.Equal(t, "Bob", *form.Name)
		assert.Equal(t, 25, *form.Age)
		assert.Equal(t, 99.5, *form.Score)
		assert.True(t, *form.Active)
	})

	t.Run("unicode map keys", func(t *testing.T) {
		type UnicodeMapForm struct {
			Data map[string]string
		}

		binder := NewASTBinder()
		form := UnicodeMapForm{Data: make(map[string]string)}

		src := map[string][]string{
			`Data["日本語"]`:    {"Japanese"},
			`Data["émojis"]`: {"French"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		assert.Equal(t, "Japanese", form.Data["日本語"])
		assert.Equal(t, "French", form.Data["émojis"])
	})

	t.Run("very large sparse map key", func(t *testing.T) {
		type LargeKeyMapForm struct {
			Items map[int]string
		}

		binder := NewASTBinder()
		form := LargeKeyMapForm{Items: make(map[int]string)}

		src := map[string][]string{
			"Items[999999999]": {"far away"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		assert.Equal(t, "far away", form.Items[999999999])
	})

	t.Run("whitespace-only values for numeric types produce parse errors", func(t *testing.T) {
		binder := NewASTBinder()

		var form SimpleForm
		src := map[string][]string{
			"Age": {"   "},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err)
	})

	t.Run("binding to interface any field", func(t *testing.T) {
		type AnyForm struct {
			Value any
		}

		binder := NewASTBinder()
		var form AnyForm

		src := map[string][]string{
			"Value": {"dynamic"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		assert.Equal(t, "dynamic", form.Value)
	})
}

func TestBindMap_NestedStruct(t *testing.T) {
	type Address struct {
		Street   string `json:"street"`
		City     string `json:"city"`
		Postcode string `json:"postcode"`
	}

	type Order struct {
		ShippingAddress Address `json:"shippingAddress"`
		Product         string  `json:"product"`
		Quantity        int     `json:"quantity"`
	}

	t.Run("simple nested struct", func(t *testing.T) {
		binder := NewASTBinder()
		var order Order
		src := map[string]any{
			"product":  "Widget",
			"quantity": float64(3),
			"shippingAddress": map[string]any{
				"street":   "123 Main St",
				"city":     "London",
				"postcode": "SW1A 1AA",
			},
		}

		err := binder.BindMap(context.Background(), &order, src, IgnoreUnknownKeys(true))
		require.NoError(t, err, "BindMap should handle nested struct input")

		assert.Equal(t, "Widget", order.Product)
		assert.Equal(t, 3, order.Quantity)
		assert.Equal(t, "123 Main St", order.ShippingAddress.Street)
		assert.Equal(t, "London", order.ShippingAddress.City)
		assert.Equal(t, "SW1A 1AA", order.ShippingAddress.Postcode)
	})

	t.Run("deeply nested struct", func(t *testing.T) {
		type Inner struct {
			Value string `json:"value"`
		}
		type Middle struct {
			Inner Inner `json:"inner"`
		}
		type Outer struct {
			Middle Middle `json:"middle"`
		}

		binder := NewASTBinder()
		var form Outer
		src := map[string]any{
			"middle": map[string]any{
				"inner": map[string]any{
					"value": "deep",
				},
			},
		}

		err := binder.BindMap(context.Background(), &form, src, IgnoreUnknownKeys(true))
		require.NoError(t, err, "BindMap should handle deeply nested structs")
		assert.Equal(t, "deep", form.Middle.Inner.Value)
	})

	t.Run("nested struct inside slice", func(t *testing.T) {
		type Item struct {
			Name    string  `json:"name"`
			Address Address `json:"address"`
		}
		type Form struct {
			Items []Item `json:"items"`
		}

		binder := NewASTBinder()
		var form Form
		src := map[string]any{
			"items": []any{
				map[string]any{
					"name": "First",
					"address": map[string]any{
						"street": "1 High St",
						"city":   "Oxford",
					},
				},
			},
		}

		err := binder.BindMap(context.Background(), &form, src, IgnoreUnknownKeys(true))
		require.NoError(t, err, "BindMap should handle nested structs inside slices")
		require.Len(t, form.Items, 1)
		assert.Equal(t, "First", form.Items[0].Name)
		assert.Equal(t, "1 High St", form.Items[0].Address.Street)
		assert.Equal(t, "Oxford", form.Items[0].Address.City)
	})

	t.Run("nested struct with pointer field", func(t *testing.T) {
		type Config struct {
			Label string `json:"label"`
		}
		type Form struct {
			Config *Config `json:"config"`
		}

		binder := NewASTBinder()
		var form Form
		src := map[string]any{
			"config": map[string]any{
				"label": "test",
			},
		}

		err := binder.BindMap(context.Background(), &form, src, IgnoreUnknownKeys(true))
		require.NoError(t, err, "BindMap should handle pointer-to-struct fields")
		require.NotNil(t, form.Config)
		assert.Equal(t, "test", form.Config.Label)
	})

	t.Run("BindJSON with nested struct", func(t *testing.T) {
		binder := NewASTBinder()
		var order Order
		jsonData := []byte(`{"product":"Gadget","quantity":5,"shippingAddress":{"street":"456 Elm Rd","city":"Manchester","postcode":"M1 1AA"}}`)

		err := binder.BindJSON(context.Background(), &order, jsonData, IgnoreUnknownKeys(true))
		require.NoError(t, err, "BindJSON should handle nested struct input")

		assert.Equal(t, "Gadget", order.Product)
		assert.Equal(t, 5, order.Quantity)
		assert.Equal(t, "456 Elm Rd", order.ShippingAddress.Street)
		assert.Equal(t, "Manchester", order.ShippingAddress.City)
		assert.Equal(t, "M1 1AA", order.ShippingAddress.Postcode)
	})
}

func TestBindMap_NestedStruct_EdgeCases(t *testing.T) {
	type Address struct {
		Street   string `json:"street"`
		City     string `json:"city"`
		Postcode string `json:"postcode"`
	}

	t.Run("integer index on struct returns error", func(t *testing.T) {

		type Form struct {
			Address Address `json:"address"`
		}

		binder := NewASTBinder()
		var form Form
		src := map[string][]string{
			"address[0]": {"invalid"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err, "Integer index on a struct should fail")
	})

	t.Run("unknown nested field returns error when strict", func(t *testing.T) {
		type Form struct {
			Address Address `json:"address"`
		}

		binder := NewASTBinder()
		binder.SetIgnoreUnknownKeys(false)
		var form Form
		src := map[string]any{
			"address": map[string]any{
				"nonExistentField": "value",
			},
		}

		err := binder.BindMap(context.Background(), &form, src)
		require.Error(t, err, "Unknown nested field should return error in strict mode")
		assert.Contains(t, err.Error(), "field not found")
	})

	t.Run("unknown nested field ignored when lenient", func(t *testing.T) {
		type Form struct {
			Address Address `json:"address"`
		}

		binder := NewASTBinder()
		var form Form
		src := map[string]any{
			"address": map[string]any{
				"street":           "123 Main St",
				"nonExistentField": "ignored",
			},
		}

		err := binder.BindMap(context.Background(), &form, src, IgnoreUnknownKeys(true))
		require.NoError(t, err, "Unknown nested field should be silently ignored in lenient mode")
		assert.Equal(t, "123 Main St", form.Address.Street)
	})

	t.Run("empty string index on struct returns error", func(t *testing.T) {
		type Form struct {
			Address Address `json:"address"`
		}

		binder := NewASTBinder()
		var form Form
		src := map[string][]string{
			"address['']": {"empty"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err, "Empty string index on a struct should fail")
	})

	t.Run("null nested object is skipped", func(t *testing.T) {
		type Form struct {
			Name    string  `json:"name"`
			Address Address `json:"address"`
		}

		binder := NewASTBinder()
		var form Form
		src := map[string]any{
			"name":    "Alice",
			"address": nil,
		}

		err := binder.BindMap(context.Background(), &form, src, IgnoreUnknownKeys(true))
		require.NoError(t, err, "Null nested object should be skipped gracefully")
		assert.Equal(t, "Alice", form.Name)
		assert.Equal(t, "", form.Address.Street, "Address fields should remain zero values")
	})

	t.Run("nested struct with all numeric types", func(t *testing.T) {
		type Dimensions struct {
			Width  float64 `json:"width"`
			Height float64 `json:"height"`
			Depth  int     `json:"depth"`
		}
		type Product struct {
			Name       string     `json:"name"`
			Dimensions Dimensions `json:"dimensions"`
		}

		binder := NewASTBinder()
		var form Product
		src := map[string]any{
			"name": "Box",
			"dimensions": map[string]any{
				"width":  float64(10.5),
				"height": float64(20.3),
				"depth":  float64(5),
			},
		}

		err := binder.BindMap(context.Background(), &form, src, IgnoreUnknownKeys(true))
		require.NoError(t, err)
		assert.Equal(t, "Box", form.Name)
		assert.Equal(t, 10.5, form.Dimensions.Width)
		assert.Equal(t, 20.3, form.Dimensions.Height)
		assert.Equal(t, 5, form.Dimensions.Depth)
	})

	t.Run("nested struct with boolean fields", func(t *testing.T) {
		type Settings struct {
			Enabled  bool `json:"enabled"`
			Verbose  bool `json:"verbose"`
			ReadOnly bool `json:"readOnly"`
		}
		type Form struct {
			Settings Settings `json:"settings"`
		}

		binder := NewASTBinder()
		var form Form
		src := map[string]any{
			"settings": map[string]any{
				"enabled":  true,
				"verbose":  false,
				"readOnly": true,
			},
		}

		err := binder.BindMap(context.Background(), &form, src, IgnoreUnknownKeys(true))
		require.NoError(t, err)
		assert.True(t, form.Settings.Enabled)
		assert.False(t, form.Settings.Verbose)
		assert.True(t, form.Settings.ReadOnly)
	})

	t.Run("multiple nested structs at same level", func(t *testing.T) {
		type Billing struct {
			CardType string `json:"cardType"`
		}
		type Shipping struct {
			Method string `json:"method"`
		}
		type Checkout struct {
			Billing  Billing  `json:"billing"`
			Shipping Shipping `json:"shipping"`
		}

		binder := NewASTBinder()
		var form Checkout
		src := map[string]any{
			"billing": map[string]any{
				"cardType": "visa",
			},
			"shipping": map[string]any{
				"method": "express",
			},
		}

		err := binder.BindMap(context.Background(), &form, src, IgnoreUnknownKeys(true))
		require.NoError(t, err)
		assert.Equal(t, "visa", form.Billing.CardType)
		assert.Equal(t, "express", form.Shipping.Method)
	})

	t.Run("nested pointer to pointer struct", func(t *testing.T) {
		type Inner struct {
			Value string `json:"value"`
		}
		type Middle struct {
			Inner *Inner `json:"inner"`
		}
		type Form struct {
			Middle *Middle `json:"middle"`
		}

		binder := NewASTBinder()
		var form Form
		src := map[string]any{
			"middle": map[string]any{
				"inner": map[string]any{
					"value": "deep-ptr",
				},
			},
		}

		err := binder.BindMap(context.Background(), &form, src, IgnoreUnknownKeys(true))
		require.NoError(t, err, "Double pointer nesting should work")
		require.NotNil(t, form.Middle)
		require.NotNil(t, form.Middle.Inner)
		assert.Equal(t, "deep-ptr", form.Middle.Inner.Value)
	})

	t.Run("struct with slice field via BindMap", func(t *testing.T) {
		type Form struct {
			Name string   `json:"name"`
			Tags []string `json:"tags"`
		}

		binder := NewASTBinder()
		var form Form
		src := map[string]any{
			"name": "Article",
			"tags": []any{"go", "web"},
		}

		err := binder.BindMap(context.Background(), &form, src, IgnoreUnknownKeys(true))
		require.NoError(t, err)
		assert.Equal(t, "Article", form.Name)
		assert.Equal(t, []string{"go", "web"}, form.Tags)
	})

	t.Run("nested struct with mixed json tags and Go names", func(t *testing.T) {
		type Metadata struct {
			CreatedBy string `json:"created_by"`
			UpdatedAt string `json:"updated_at"`
		}
		type Form struct {
			Metadata Metadata `json:"metadata"`
		}

		binder := NewASTBinder()
		var form Form
		src := map[string]any{
			"metadata": map[string]any{
				"created_by": "alice",
				"updated_at": "2026-01-01",
			},
		}

		err := binder.BindMap(context.Background(), &form, src, IgnoreUnknownKeys(true))
		require.NoError(t, err)
		assert.Equal(t, "alice", form.Metadata.CreatedBy)
		assert.Equal(t, "2026-01-01", form.Metadata.UpdatedAt)
	})
}

func TestBindMap_NestedStruct_DoSProtection(t *testing.T) {
	t.Run("maxPathDepth limits deeply nested bracket notation", func(t *testing.T) {
		type L5 struct {
			Value string `json:"value"`
		}
		type L4 struct {
			L5 L5 `json:"l5"`
		}
		type L3 struct {
			L4 L4 `json:"l4"`
		}
		type L2 struct {
			L3 L3 `json:"l3"`
		}
		type L1 struct {
			L2 L2 `json:"l2"`
		}
		type Form struct {
			L1 L1 `json:"l1"`
		}

		binder := NewASTBinder()
		binder.SetMaxPathDepth(2)
		var form Form

		src := map[string]any{
			"l1": map[string]any{
				"l2": map[string]any{
					"l3": map[string]any{
						"l4": map[string]any{
							"l5": map[string]any{
								"value": "too-deep",
							},
						},
					},
				},
			},
		}

		err := binder.BindMap(context.Background(), &form, src, IgnoreUnknownKeys(true))
		require.Error(t, err, "Deeply nested bracket-notation paths should be limited by maxPathDepth")
		assert.Contains(t, err.Error(), "path depth exceeds maximum limit")
	})

	t.Run("maxFieldCount limits flattened nested struct fields", func(t *testing.T) {

		type BigStruct struct {
			Data map[string]string `json:"data"`
		}

		binder := NewASTBinder()
		binder.SetMaxFieldCount(5)
		var form BigStruct

		data := make(map[string]any, 20)
		for i := range 20 {
			data[fmt.Sprintf("field%d", i)] = fmt.Sprintf("value%d", i)
		}

		src := map[string]any{
			"data": data,
		}

		err := binder.BindMap(context.Background(), &form, src)
		require.Error(t, err, "Many flattened fields should be rejected by maxFieldCount")
		assert.Contains(t, err.Error(), "exceeds maximum limit of 5")
	})

	t.Run("maxPathLength limits long bracket-notation paths", func(t *testing.T) {
		type Form struct {
			Name string `json:"name"`
		}

		binder := NewASTBinder()
		binder.SetMaxPathLength(30)
		var form Form

		longKey := strings.Repeat("x", 25)
		src := map[string]any{
			"name": map[string]any{
				longKey: "value",
			},
		}

		err := binder.BindMap(context.Background(), &form, src, IgnoreUnknownKeys(true))
		require.Error(t, err, "Long bracket-notation paths should be limited")
		assert.Contains(t, err.Error(), "path length exceeds maximum limit")
	})

	t.Run("maxValueLength limits values in nested struct fields", func(t *testing.T) {
		type Address struct {
			Street string `json:"street"`
		}
		type Form struct {
			Address Address `json:"address"`
		}

		binder := NewASTBinder()
		binder.SetMaxValueLength(10)
		var form Form

		longValue := strings.Repeat("a", 50)
		src := map[string]any{
			"address": map[string]any{
				"street": longValue,
			},
		}

		err := binder.BindMap(context.Background(), &form, src, IgnoreUnknownKeys(true))
		require.Error(t, err, "Long values in nested structs should be limited by maxValueLength")
		assert.Contains(t, err.Error(), "value length exceeds maximum limit")
	})

	t.Run("BindMap with many nested objects respects field count limit", func(t *testing.T) {
		type Item struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		}
		type Form struct {
			Items []Item `json:"items"`
		}

		binder := NewASTBinder()
		binder.SetMaxFieldCount(10)
		var form Form

		items := make([]any, 10)
		for i := range 10 {
			items[i] = map[string]any{
				"name":  fmt.Sprintf("item%d", i),
				"value": fmt.Sprintf("val%d", i),
			}
		}

		src := map[string]any{
			"items": items,
		}

		err := binder.BindMap(context.Background(), &form, src)
		require.Error(t, err, "Many items producing many flattened fields should be limited")
		assert.Contains(t, err.Error(), "exceeds maximum limit of 10")
	})

	t.Run("nil pointer initialisation is bounded by depth limit", func(t *testing.T) {
		type D struct {
			Value string `json:"value"`
		}
		type C struct {
			D *D `json:"d"`
		}
		type B struct {
			C *C `json:"c"`
		}
		type A struct {
			B *B `json:"b"`
		}
		type Form struct {
			A *A `json:"a"`
		}

		binder := NewASTBinder()
		binder.SetMaxPathDepth(2)
		var form Form

		src := map[string]any{
			"a": map[string]any{
				"b": map[string]any{
					"c": map[string]any{
						"d": map[string]any{
							"value": "deep",
						},
					},
				},
			},
		}

		err := binder.BindMap(context.Background(), &form, src, IgnoreUnknownKeys(true))
		require.Error(t, err, "Deep nil pointer chain should be limited by depth")
		assert.Contains(t, err.Error(), "path depth exceeds maximum limit")
	})

	t.Run("unexported struct scalar fields cannot be set via bracket notation", func(t *testing.T) {

		type Form struct {
			Public string `json:"public"`
			hidden string
		}

		binder := NewASTBinder()
		var form Form
		src := map[string]any{
			"public": "visible",
			"hidden": "should-not-bind",
		}

		_ = binder.BindMap(context.Background(), &form, src, IgnoreUnknownKeys(true))
		assert.Equal(t, "visible", form.Public)
		assert.Equal(t, "", form.hidden, "Unexported fields should not be settable")
	})
}
