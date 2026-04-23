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
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSafeInvokeLinkedSiblingRecoversFromPanic(t *testing.T) {
	t.Parallel()

	panicking := reflect.ValueOf(func(_ int) int {
		panic("boom")
	})

	results, err := safeInvokeLinkedSibling(panicking, []reflect.Value{reflect.ValueOf(1)})

	require.Nil(t, results)
	require.Error(t, err)
	require.ErrorIs(t, err, errLinkedSiblingPanic)
}

func TestSafeInvokeLinkedSiblingPropagatesResults(t *testing.T) {
	t.Parallel()

	identity := reflect.ValueOf(func(value int) int { return value })

	results, err := safeInvokeLinkedSibling(identity, []reflect.Value{reflect.ValueOf(42)})

	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, int64(42), results[0].Int())
}

func TestValidateLinkedCallShapeRejectsMismatchedArity(t *testing.T) {
	t.Parallel()

	sibling := reflect.ValueOf(func(_ reflect.Type, _ int) reflect.Value {
		return reflect.Value{}
	})
	site := &callSite{
		linkedTypeArgs: []reflect.Type{reflect.TypeFor[string]()},
		arguments:      []varLocation{{}, {}},
	}

	err := validateLinkedCallShape(sibling, site)

	require.Error(t, err)
	require.ErrorIs(t, err, errLinkedSiblingShapeMismatch)
}

func TestValidateLinkedCallShapeAcceptsExactArity(t *testing.T) {
	t.Parallel()

	sibling := reflect.ValueOf(func(_ reflect.Type, _ int) reflect.Value {
		return reflect.Value{}
	})
	site := &callSite{
		linkedTypeArgs: []reflect.Type{reflect.TypeFor[string]()},
		arguments:      []varLocation{{}},
	}

	err := validateLinkedCallShape(sibling, site)

	require.NoError(t, err)
}

func TestValidateLinkedCallShapeAcceptsVariadic(t *testing.T) {
	t.Parallel()

	sibling := reflect.ValueOf(func(_ reflect.Type, _ int, _ ...string) reflect.Value {
		return reflect.Value{}
	})
	site := &callSite{
		linkedTypeArgs: []reflect.Type{reflect.TypeFor[string]()},
		arguments:      []varLocation{{}, {}, {}, {}},
	}

	err := validateLinkedCallShape(sibling, site)

	require.NoError(t, err)
}

func TestValidateLinkedCallShapeRejectsNonFunc(t *testing.T) {
	t.Parallel()

	site := &callSite{}

	err := validateLinkedCallShape(reflect.ValueOf(42), site)
	require.Error(t, err)
	require.ErrorIs(t, err, errLinkedSiblingShapeMismatch)
}

func TestUnwrapReflectValueResultPeelsSingleLayer(t *testing.T) {
	t.Parallel()

	inner := reflect.ValueOf("hello")
	wrapper := reflect.ValueOf(inner)

	result := unwrapReflectValueResult(wrapper)
	require.Equal(t, "hello", result.String())
}

func TestUnwrapReflectValueResultRespectsDepthCap(t *testing.T) {
	t.Parallel()

	value := reflect.ValueOf("deep")
	for range maxReflectValueUnwrapDepth + 2 {
		value = reflect.ValueOf(value)
	}

	result := unwrapReflectValueResult(value)
	require.Equal(t, linkedResultReflectValueType, result.Type(),
		"deliberate over-wrap should leave a reflect.Value intact after the depth cap")
}

func TestUnwrapReflectValueResultPassesThroughNonWrapper(t *testing.T) {
	t.Parallel()

	plain := reflect.ValueOf(99)

	result := unwrapReflectValueResult(plain)
	require.Equal(t, int64(99), result.Int())
}

func TestErrLinkedSentinelsAreDistinct(t *testing.T) {
	t.Parallel()

	require.False(t, errors.Is(errLinkedSiblingPanic, errLinkedSiblingShapeMismatch))
	require.False(t, errors.Is(errLinkedSiblingShapeMismatch, errLinkedSiblingPanic))
}
