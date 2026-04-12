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

	"github.com/stretchr/testify/require"
)

type addressableTestCounter struct {
	Value int
}

func TestEnsureAddressableStructReceiverCopiesNonAddressableStruct(t *testing.T) {
	t.Parallel()

	original := addressableTestCounter{Value: 42}
	receiver := reflect.ValueOf(original)
	require.Equal(t, reflect.Struct, receiver.Kind(),
		"baseline: a struct passed through reflect.ValueOf is Struct kind")
	require.False(t, receiver.CanAddr(),
		"baseline: a struct extracted from a value (not from a pointer) is not addressable")

	result := ensureAddressableStructReceiver(receiver)
	require.True(t, result.CanAddr(),
		"the returned receiver must be addressable so pointer-receiver methods can take Addr()")
	require.Equal(t, int64(42), result.FieldByName("Value").Int(),
		"the addressable copy preserves the field values")
}

func TestEnsureAddressableStructReceiverPassesThroughAddressableStruct(t *testing.T) {
	t.Parallel()

	holder := reflect.New(reflect.TypeFor[addressableTestCounter]()).Elem()
	holder.FieldByName("Value").SetInt(99)
	require.True(t, holder.CanAddr(),
		"baseline: reflect.New(t).Elem() yields an addressable value")

	result := ensureAddressableStructReceiver(holder)
	require.True(t, result.CanAddr())
	require.Equal(t, holder.Addr().Pointer(), result.Addr().Pointer(),
		"already-addressable values must pass through without copying so callers see the original memory")
}

func TestEnsureAddressableStructReceiverPassesThroughPointerReceiver(t *testing.T) {
	t.Parallel()

	original := &addressableTestCounter{Value: 7}
	receiver := reflect.ValueOf(original)
	require.Equal(t, reflect.Pointer, receiver.Kind())

	result := ensureAddressableStructReceiver(receiver)
	require.Equal(t, receiver.UnsafePointer(), result.UnsafePointer(),
		"pointer receivers are not struct kind; they pass through unchanged so the callee can use the pointer directly")
}

func TestEnsureAddressableStructReceiverPassesThroughNonStructKinds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value reflect.Value
	}{
		{name: "int", value: reflect.ValueOf(42)},
		{name: "string", value: reflect.ValueOf("hello")},
		{name: "slice", value: reflect.ValueOf([]int{1, 2, 3})},
		{name: "map", value: reflect.ValueOf(map[string]int{"a": 1})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := ensureAddressableStructReceiver(tt.value)
			require.Equal(t, tt.value.Kind(), result.Kind(),
				"non-struct kinds must pass through without modification")
		})
	}
}
