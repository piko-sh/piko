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
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaxFieldCount(t *testing.T) {
	t.Run("allows forms within the limit", func(t *testing.T) {
		binder := NewASTBinder()
		binder.SetMaxFieldCount(100)

		var form SimpleForm
		src := make(map[string][]string)
		for i := range 50 {
			src[fmt.Sprintf("field%d", i)] = []string{"value"}
		}

		err := binder.Bind(context.Background(), &form, src)

		if err != nil {
			multiErr, ok := errors.AsType[MultiError](err)
			require.True(t, ok, "Should return MultiError for invalid fields, not field count error")
			_ = multiErr
		}
	})

	t.Run("rejects forms exceeding the limit", func(t *testing.T) {
		binder := NewASTBinder()
		binder.SetMaxFieldCount(50)

		var form SimpleForm
		src := make(map[string][]string)
		for i := range 100 {
			src[fmt.Sprintf("field%d", i)] = []string{"value"}
		}

		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "number of form fields (100) exceeds maximum limit of 50")
	})

	t.Run("allows unlimited fields when set to 0", func(t *testing.T) {
		binder := NewASTBinder()
		binder.SetMaxFieldCount(0)

		var form SimpleForm
		src := make(map[string][]string)
		for i := range 2000 {
			src[fmt.Sprintf("field%d", i)] = []string{"value"}
		}

		err := binder.Bind(context.Background(), &form, src)

		if err != nil {
			assert.NotContains(t, err.Error(), "exceeds maximum limit")
		}
	})

	t.Run("handles negative values as unlimited", func(t *testing.T) {
		binder := NewASTBinder()
		binder.SetMaxFieldCount(-10)

		var form SimpleForm
		src := make(map[string][]string)
		for i := range 2000 {
			src[fmt.Sprintf("field%d", i)] = []string{"value"}
		}

		err := binder.Bind(context.Background(), &form, src)

		if err != nil {
			assert.NotContains(t, err.Error(), "exceeds maximum limit")
		}
	})

	t.Run("is thread-safe", func(t *testing.T) {
		binder := NewASTBinder()
		var wg sync.WaitGroup

		for i := range 10 {
			limit := i
			wg.Go(func() {
				binder.SetMaxFieldCount(limit * 10)
			})
		}
		wg.Wait()

		var form SimpleForm
		src := map[string][]string{"Name": {"test"}}
		_ = binder.Bind(context.Background(), &form, src)
	})
}

func TestMaxPathLength(t *testing.T) {
	t.Run("allows paths within the limit", func(t *testing.T) {
		binder := NewASTBinder()
		binder.SetMaxPathLength(100)

		var form SimpleForm
		src := map[string][]string{
			"Name": {"Alice"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		assert.Equal(t, "Alice", form.Name)
	})

	t.Run("rejects paths exceeding the limit", func(t *testing.T) {
		binder := NewASTBinder()
		binder.SetMaxPathLength(50)

		var form SimpleForm
		longPath := strings.Repeat("a", 100)
		src := map[string][]string{
			longPath: {"value"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err)
		multiErr, ok := errors.AsType[MultiError](err)
		require.True(t, ok)
		assert.Contains(t, multiErr[longPath].Error(), "path length exceeds maximum limit of 50")
	})

	t.Run("allows unlimited path length when set to 0", func(t *testing.T) {
		binder := NewASTBinder()
		binder.SetMaxPathLength(0)

		var form SimpleForm
		longPath := strings.Repeat("Name", 1000)
		src := map[string][]string{
			longPath: {"value"},
		}

		err := binder.Bind(context.Background(), &form, src)

		if err != nil {
			multiErr, ok := errors.AsType[MultiError](err)
			require.True(t, ok)
			assert.NotContains(t, multiErr[longPath].Error(), "path length exceeds")
		}
	})

	t.Run("handles negative values as unlimited", func(t *testing.T) {
		binder := NewASTBinder()
		binder.SetMaxPathLength(-1)

		var form SimpleForm
		longPath := strings.Repeat("Name", 500)
		src := map[string][]string{
			longPath: {"value"},
		}

		err := binder.Bind(context.Background(), &form, src)

		if err != nil {
			assert.NotContains(t, err.Error(), "path length exceeds")
		}
	})

	t.Run("checks path length before parsing", func(t *testing.T) {
		binder := NewASTBinder()
		binder.SetMaxPathLength(20)

		var form SliceForm

		longPath := "Items[0].Name.ExtraField"
		src := map[string][]string{
			longPath: {"value"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err)
		multiErr, ok := errors.AsType[MultiError](err)
		require.True(t, ok)
		assert.Contains(t, multiErr[longPath].Error(), "path length exceeds maximum limit of 20")
	})
}

func TestMaxValueLength(t *testing.T) {
	t.Run("allows values within the limit", func(t *testing.T) {
		binder := NewASTBinder()
		binder.SetMaxValueLength(100)

		var form SimpleForm
		src := map[string][]string{
			"Name": {"Alice"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		assert.Equal(t, "Alice", form.Name)
	})

	t.Run("rejects values exceeding the limit", func(t *testing.T) {
		binder := NewASTBinder()
		binder.SetMaxValueLength(50)

		var form SimpleForm
		longValue := strings.Repeat("a", 100)
		src := map[string][]string{
			"Name": {longValue},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err)
		multiErr, ok := errors.AsType[MultiError](err)
		require.True(t, ok)
		assert.Contains(t, multiErr["Name"].Error(), "value length exceeds maximum limit of 50")
	})

	t.Run("allows unlimited value length when set to 0", func(t *testing.T) {
		binder := NewASTBinder()
		binder.SetMaxValueLength(0)

		var form SimpleForm
		longValue := strings.Repeat("a", 100000)
		src := map[string][]string{
			"Name": {longValue},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		assert.Equal(t, longValue, form.Name)
	})

	t.Run("handles negative values as unlimited", func(t *testing.T) {
		binder := NewASTBinder()
		binder.SetMaxValueLength(-10)

		var form SimpleForm
		longValue := strings.Repeat("a", 100000)
		src := map[string][]string{
			"Name": {longValue},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		assert.Equal(t, longValue, form.Name)
	})

	t.Run("checks value length before conversion", func(t *testing.T) {
		binder := NewASTBinder()
		binder.SetMaxValueLength(10)

		var form SimpleForm

		longValue := "12345678901234567890"
		src := map[string][]string{
			"Age": {longValue},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err)
		multiErr, ok := errors.AsType[MultiError](err)
		require.True(t, ok)
		assert.Contains(t, multiErr["Age"].Error(), "value length exceeds maximum limit of 10")
	})
}

func TestMaxPathDepth(t *testing.T) {
	t.Run("allows paths within depth limit", func(t *testing.T) {
		binder := NewASTBinder()
		binder.SetMaxPathDepth(10)

		var form NestedForm
		src := map[string][]string{
			"User.Name": {"Bob"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		assert.Equal(t, "Bob", form.User.Name)
	})

	t.Run("rejects paths exceeding depth limit in slow path", func(t *testing.T) {
		binder := NewASTBinder()
		binder.SetMaxPathDepth(1)

		type DeepStruct struct {
			Items []struct {
				L1 struct {
					L2 struct {
						L3 struct {
							L4 struct {
								Value string
							}
						}
					}
				}
			}
		}

		var form DeepStruct

		src := map[string][]string{
			"Items[0].L1.L2.L3.L4.Value": {"deep"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err)
		multiErr, ok := errors.AsType[MultiError](err)
		require.True(t, ok)
		assert.Contains(t, multiErr["Items[0].L1.L2.L3.L4.Value"].Error(), "path depth exceeds maximum limit of 1")
	})

	t.Run("allows unlimited depth when set to 0", func(t *testing.T) {
		binder := NewASTBinder()
		binder.SetMaxPathDepth(0)

		type DeepStruct struct {
			L1 struct {
				L2 struct {
					L3 struct {
						L4 struct {
							L5 struct {
								Value string
							}
						}
					}
				}
			}
		}

		var form DeepStruct
		src := map[string][]string{
			"L1.L2.L3.L4.L5.Value": {"verydeep"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		assert.Equal(t, "verydeep", form.L1.L2.L3.L4.L5.Value)
	})

	t.Run("protects cache building from deeply nested structs", func(t *testing.T) {
		binder := NewASTBinder()
		binder.SetMaxPathDepth(3)

		type DeepStruct struct {
			L1 struct {
				L2 struct {
					L3 struct {
						L4 struct {
							L5 string
						}
					}
				}
			}
		}

		var form DeepStruct
		src := map[string][]string{
			"L1.L2.L3.L4.L5": {"value"},
		}

		err := binder.Bind(context.Background(), &form, src)

		if err != nil {
			multiErr, ok := errors.AsType[MultiError](err)
			require.True(t, ok)

			_ = multiErr
		}
	})

	t.Run("handles negative values as unlimited", func(t *testing.T) {
		binder := NewASTBinder()
		binder.SetMaxPathDepth(-5)

		type DeepStruct struct {
			L1 struct {
				L2 struct {
					L3 struct {
						L4 struct {
							L5 struct {
								Value string
							}
						}
					}
				}
			}
		}

		var form DeepStruct
		src := map[string][]string{
			"L1.L2.L3.L4.L5.Value": {"verydeep"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		assert.Equal(t, "verydeep", form.L1.L2.L3.L4.L5.Value)
	})

	t.Run("counts depth correctly with array indexing", func(t *testing.T) {
		binder := NewASTBinder()
		binder.SetMaxPathDepth(5)

		type ArrayStruct struct {
			Items []struct {
				Name    string
				Details struct {
					Value string
				}
			}
		}

		var form ArrayStruct

		src := map[string][]string{
			"Items[0].Details.Value": {"nested"},
		}

		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
		assert.Equal(t, "nested", form.Items[0].Details.Value)
	})
}

func TestDoSProtectionDefaults(t *testing.T) {
	t.Run("default limits are set correctly", func(t *testing.T) {
		binder := NewASTBinder()

		var form SimpleForm

		src := make(map[string][]string)
		for i := range 1001 {
			src[fmt.Sprintf("field%d", i)] = []string{"value"}
		}
		err := binder.Bind(context.Background(), &form, src)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum limit of 1000")
	})

	t.Run("all setters accept zero to disable", func(t *testing.T) {
		binder := NewASTBinder()
		binder.SetMaxFieldCount(0)
		binder.SetMaxPathLength(0)
		binder.SetMaxValueLength(0)
		binder.SetMaxPathDepth(0)

		var form SimpleForm
		src := map[string][]string{"Name": {"test"}}
		err := binder.Bind(context.Background(), &form, src)
		require.NoError(t, err)
	})
}
