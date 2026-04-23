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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildClosureErrorReturns(t *testing.T) {
	t.Parallel()

	failure := errors.New("interp blew up")

	t.Run("threads error into trailing error slot", func(t *testing.T) {
		t.Parallel()
		targetType := reflect.TypeOf(func(int) (string, error) { return "", nil })
		returns := buildClosureErrorReturns(context.Background(), targetType, failure)
		require.Len(t, returns, 2)
		assert.Equal(t, "", returns[0].Interface())
		gotErr, ok := returns[1].Interface().(error)
		require.True(t, ok, "trailing return should be error")
		assert.Same(t, failure, gotErr)
	})

	t.Run("fills zero values for non-error outputs", func(t *testing.T) {
		t.Parallel()
		type payload struct {
			Name string
		}
		targetType := reflect.TypeOf(func() (int, *payload, error) { return 0, nil, nil })
		returns := buildClosureErrorReturns(context.Background(), targetType, failure)
		require.Len(t, returns, 3)
		assert.Equal(t, int64(0), returns[0].Int())
		assert.True(t, returns[1].IsNil())
		gotErr, ok := returns[2].Interface().(error)
		require.True(t, ok)
		assert.Same(t, failure, gotErr)
	})

	t.Run("error-only signature places error in the sole slot", func(t *testing.T) {
		t.Parallel()
		targetType := reflect.TypeOf(func() error { return nil })
		returns := buildClosureErrorReturns(context.Background(), targetType, failure)
		require.Len(t, returns, 1)
		gotErr, ok := returns[0].Interface().(error)
		require.True(t, ok)
		assert.Same(t, failure, gotErr)
	})

	t.Run("signature without error slot returns zero values", func(t *testing.T) {
		t.Parallel()
		targetType := reflect.TypeOf(func() (int, string) { return 0, "" })
		returns := buildClosureErrorReturns(context.Background(), targetType, failure)
		require.Len(t, returns, 2)
		assert.Equal(t, int64(0), returns[0].Int())
		assert.Equal(t, "", returns[1].Interface())
	})

	t.Run("no returns yields nil slice", func(t *testing.T) {
		t.Parallel()
		targetType := reflect.TypeOf(func() {})
		returns := buildClosureErrorReturns(context.Background(), targetType, failure)
		assert.Nil(t, returns)
	})
}
