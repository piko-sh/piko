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
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestFindTargetByAST(t *testing.T) {
	binder := NewASTBinder()

	testLimits := binderOptions{
		ignoreUnknownKeys: false,
		maxFieldCount:     1000,
		maxPathLength:     4096,
		maxValueLength:    65536,
		maxPathDepth:      32,
		maxSliceSize:      1000,
	}
	form := SliceForm{
		Items: []Item{{Name: "Existing"}},
	}
	v := reflect.ValueOf(&form).Elem()

	t.Run("finds simple identifier field", func(t *testing.T) {
		expression := ast_domain.NewExpressionParser(context.Background(), "Items", "")
		pathAST, _ := expression.ParseExpression(context.Background())

		targetVal, err := binder.findTargetByAST(v, pathAST, "Items", 0, testLimits)
		require.NoError(t, err)
		require.True(t, targetVal.IsValid())
		assert.Equal(t, reflect.Slice, targetVal.Kind())
	})

	t.Run("finds slice element within bounds", func(t *testing.T) {
		expression := ast_domain.NewExpressionParser(context.Background(), "Items[0]", "")
		pathAST, _ := expression.ParseExpression(context.Background())

		targetVal, err := binder.findTargetByAST(v, pathAST, "Items[0]", 0, testLimits)
		require.NoError(t, err)
		require.True(t, targetVal.IsValid())
		assert.Equal(t, reflect.Struct, targetVal.Kind())
		assert.Equal(t, "Existing", targetVal.FieldByName("Name").String())
	})

	t.Run("finds and grows slice for out-of-bounds index", func(t *testing.T) {

		form := SliceForm{Items: make([]Item, 1, 5)}
		v := reflect.ValueOf(&form).Elem()

		expression := ast_domain.NewExpressionParser(context.Background(), "Items[3]", "")
		pathAST, _ := expression.ParseExpression(context.Background())

		targetVal, err := binder.findTargetByAST(v, pathAST, "Items[3]", 0, testLimits)
		require.NoError(t, err)
		require.True(t, targetVal.IsValid())

		assert.Equal(t, 4, len(form.Items))
		assert.Equal(t, 5, cap(form.Items))
	})
}

func TestFindTargetByAST_NonStructIdentifier(t *testing.T) {

	t.Run("identifier on map value returns error not panic", func(t *testing.T) {
		type Form struct {
			Data map[string]string `json:"data"`
		}

		binder := NewASTBinder()
		var form Form

		src := map[string]any{
			"data": map[string]any{
				"key": map[string]any{
					"nonexistent": "value",
				},
			},
		}

		err := binder.BindMap(context.Background(), &form, src)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a slice, map, or struct")
	})

	t.Run("dot notation on non-struct via Bind returns error not panic", func(t *testing.T) {
		type Form struct {
			Value string `json:"value"`
		}

		binder := NewASTBinder()
		var form Form

		err := binder.Bind(context.Background(), &form, map[string][]string{
			"value.name": {"test"},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "non-struct type")
	})
}

func TestIsNestedMapAccess_NonStructCurrentVal(t *testing.T) {

	t.Run("non-struct currentVal returns false without panic", func(t *testing.T) {
		type Form struct {
			Data map[string]any `json:"data"`
		}

		binder := NewASTBinder()
		var form Form
		src := map[string]any{
			"data": map[string]any{
				"nested": map[string]any{
					"key": "value",
				},
			},
		}

		err := binder.BindMap(context.Background(), &form, src, IgnoreUnknownKeys(true))
		require.NoError(t, err)
		assert.Equal(t, "value", form.Data["nested"].(map[string]any)["key"])
	})
}
