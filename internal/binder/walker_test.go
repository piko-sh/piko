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

func TestSetByAST(t *testing.T) {
	binder := NewASTBinder()

	testLimits := binderOptions{
		ignoreUnknownKeys: false,
		maxFieldCount:     1000,
		maxPathLength:     4096,
		maxValueLength:    65536,
		maxPathDepth:      32,
		maxSliceSize:      1000,
	}

	t.Run("sets a top-level field", func(t *testing.T) {
		var form SimpleForm
		v := reflect.ValueOf(&form).Elem()
		expression := ast_domain.NewExpressionParser(context.Background(), "Name", "")
		pathAST, _ := expression.ParseExpression(context.Background())

		err := binder.setByAST(v, pathAST, "Charlie", "Name", testLimits)
		require.NoError(t, err)
		assert.Equal(t, "Charlie", form.Name)
	})

	t.Run("sets a nested field", func(t *testing.T) {
		var form NestedForm
		v := reflect.ValueOf(&form).Elem()
		expression := ast_domain.NewExpressionParser(context.Background(), "User.Name", "")
		pathAST, _ := expression.ParseExpression(context.Background())

		err := binder.setByAST(v, pathAST, "David", "User.Name", testLimits)
		require.NoError(t, err)
		assert.Equal(t, "David", form.User.Name)
	})

	t.Run("sets a field in a pointer-to-struct, initialising it", func(t *testing.T) {
		var form NestedForm
		require.Nil(t, form.Profile)
		v := reflect.ValueOf(&form).Elem()
		expression := ast_domain.NewExpressionParser(context.Background(), "Profile.Email", "")
		pathAST, _ := expression.ParseExpression(context.Background())

		err := binder.setByAST(v, pathAST, "david@example.com", "Profile.Email", testLimits)
		require.NoError(t, err)
		require.NotNil(t, form.Profile)
		assert.Equal(t, "david@example.com", form.Profile.Email)
	})

	t.Run("fails on invalid expression type", func(t *testing.T) {
		var form SimpleForm
		v := reflect.ValueOf(&form).Elem()

		expression := ast_domain.NewExpressionParser(context.Background(), "1 + 2", "")
		pathAST, _ := expression.ParseExpression(context.Background())

		err := binder.setByAST(v, pathAST, "any", "1 + 2", testLimits)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid path")
	})
}
