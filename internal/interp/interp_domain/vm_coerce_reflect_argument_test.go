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
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCoerceReflectArgument_ZeroValueBecomesTypedNil(t *testing.T) {
	t.Parallel()
	vm := newTestVM(t)
	expectedType := reflect.TypeFor[*time.Location]()

	out := coerceReflectArgument(vm, reflect.Value{}, expectedType, argumentTypeContext{})

	require.True(t, out.IsValid(),
		"zero Value with known expectedType should become a typed nil, not stay invalid")
	require.Equal(t, expectedType, out.Type(),
		"coerced value must carry the parameter's expected type")
	require.True(t, out.IsNil(),
		"typed nil pointer should report IsNil() true")
}

func TestCoerceReflectArgument_ZeroValueNilExpectedTypeIsPassthrough(t *testing.T) {
	t.Parallel()
	vm := newTestVM(t)

	out := coerceReflectArgument(vm, reflect.Value{}, nil, argumentTypeContext{})

	require.False(t, out.IsValid(),
		"zero Value with nil expectedType must stay invalid (no panic)")
}

func TestCoerceReflectArgument_MatchingTypeIsPassthrough(t *testing.T) {
	t.Parallel()
	vm := newTestVM(t)
	input := reflect.ValueOf("hello")

	out := coerceReflectArgument(vm, input, input.Type(), argumentTypeContext{})

	require.Equal(t, input.Interface(), out.Interface())
	require.Equal(t, input.Type(), out.Type())
}

func TestCoerceReflectArgument_ConvertibleIntToInt32(t *testing.T) {
	t.Parallel()
	vm := newTestVM(t)
	input := reflect.ValueOf(int64(42))
	want := reflect.TypeFor[int32]()

	out := coerceReflectArgument(vm, input, want, argumentTypeContext{})

	require.Equal(t, want, out.Type())
	require.EqualValues(t, 42, out.Int())
}

func TestCoerceReflectArgument_Int64ToBool(t *testing.T) {
	t.Parallel()
	vm := newTestVM(t)
	want := reflect.TypeFor[bool]()

	out := coerceReflectArgument(vm, reflect.ValueOf(int64(1)), want, argumentTypeContext{})

	require.Equal(t, want, out.Type())
	require.True(t, out.Bool())

	out = coerceReflectArgument(vm, reflect.ValueOf(int64(0)), want, argumentTypeContext{})
	require.False(t, out.Bool())
}
