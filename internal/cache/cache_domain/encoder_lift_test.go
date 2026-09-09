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

package cache_domain

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type plainEncoder struct{}

func (plainEncoder) Marshal(value string) ([]byte, error) {
	return json.Marshal(value)
}

func (plainEncoder) Unmarshal(data []byte, target *string) error {
	return json.Unmarshal(data, target)
}

func TestTypedEncoderAcceptsAnEncoderThatIsNotAnAnyEncoder(t *testing.T) {
	t.Parallel()

	builder := NewCacheBuilder[string, string](NewService("")).
		TypedEncoder(plainEncoder{})

	require.NotNil(t, builder)
	require.Len(t, builder.encoders, 1,
		"An EncoderPort that does not also implement AnyEncoder must still register")

	registered := builder.encoders[0]
	assert.Equal(t, reflect.TypeFor[string](), registered.HandlesType())

	data, err := registered.MarshalAny("hello")
	require.NoError(t, err)

	decoded, err := registered.UnmarshalAny(data)
	require.NoError(t, err)
	assert.Equal(t, "hello", decoded)
}

func TestTypedEncoderRejectsAValueOfTheWrongType(t *testing.T) {
	t.Parallel()

	builder := NewCacheBuilder[string, string](NewService("")).
		TypedEncoder(plainEncoder{})
	require.Len(t, builder.encoders, 1)

	_, err := builder.encoders[0].MarshalAny(42)
	require.Error(t, err, "A lifted encoder must reject a value that is not its type")
	assert.Contains(t, err.Error(), "string")
}

func TestTypedEncoderPreservesIdentityForAnAnyEncoder(t *testing.T) {
	t.Parallel()

	shipped := NewEncoder(
		func(value string) ([]byte, error) { return json.Marshal(value) },
		func(data []byte, target *string) error { return json.Unmarshal(data, target) },
	)

	builder := NewCacheBuilder[string, string](NewService("")).TypedEncoder(shipped)
	require.Len(t, builder.encoders, 1)

	assert.Same(t, shipped, builder.encoders[0],
		"An encoder that already implements AnyEncoder must not be wrapped")
}

func TestTypedEncoderIgnoresNil(t *testing.T) {
	t.Parallel()

	builder := NewCacheBuilder[string, string](NewService("")).
		TypedEncoder(nil)

	assert.Empty(t, builder.encoders)
}
