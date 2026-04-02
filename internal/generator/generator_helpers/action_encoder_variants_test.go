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

package generator_helpers

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

func TestEncodeStaticActionPayload(t *testing.T) {
	t.Parallel()

	t.Run("zero arguments", func(t *testing.T) {
		t.Parallel()

		payload := templater_dto.ActionPayload{
			Function: "handleClick",
			Args:     []templater_dto.ActionArgument{},
		}

		result := EncodeStaticActionPayload(payload)
		assert.NotEmpty(t, result)

		jsonBytes, err := base64.RawURLEncoding.DecodeString(result)
		require.NoError(t, err)

		var parsed templater_dto.ActionPayload
		require.NoError(t, json.Unmarshal(jsonBytes, &parsed))
		assert.Equal(t, "handleClick", parsed.Function)
		assert.NotNil(t, parsed.Args)
		assert.Empty(t, parsed.Args)
	})

	t.Run("with arguments", func(t *testing.T) {
		t.Parallel()

		payload := templater_dto.ActionPayload{
			Function: "handleSubmit",
			Args: []templater_dto.ActionArgument{
				{Type: "s", Value: "hello"},
			},
		}

		result := EncodeStaticActionPayload(payload)
		assert.NotEmpty(t, result)

		jsonBytes, err := base64.RawURLEncoding.DecodeString(result)
		require.NoError(t, err)

		var parsed templater_dto.ActionPayload
		require.NoError(t, json.Unmarshal(jsonBytes, &parsed))
		assert.Equal(t, "handleSubmit", parsed.Function)
		require.Len(t, parsed.Args, 1)
		assert.Equal(t, "s", parsed.Args[0].Type)
	})

	t.Run("nil arguments", func(t *testing.T) {
		t.Parallel()

		payload := templater_dto.ActionPayload{
			Function: "noArgs",
			Args:     nil,
		}

		result := EncodeStaticActionPayload(payload)
		assert.NotEmpty(t, result)

		jsonBytes, err := base64.RawURLEncoding.DecodeString(result)
		require.NoError(t, err)

		var parsed templater_dto.ActionPayload
		require.NoError(t, json.Unmarshal(jsonBytes, &parsed))
		assert.Equal(t, "noArgs", parsed.Function)
		assert.Nil(t, parsed.Args)
	})
}

func TestEncodeActionPayloadBytesArena(t *testing.T) {
	t.Parallel()

	arena := ast_domain.GetArena()
	defer ast_domain.PutArena(arena)

	payload := templater_dto.ActionPayload{
		Function: "handleClick",
		Args: []templater_dto.ActionArgument{
			{Type: "s", Value: "val"},
		},
	}

	result := EncodeActionPayloadBytesArena(arena, payload)
	require.NotNil(t, result)

	jsonBytes, err := base64.RawURLEncoding.DecodeString(string(*result))
	require.NoError(t, err)

	var parsed templater_dto.ActionPayload
	require.NoError(t, json.Unmarshal(jsonBytes, &parsed))
	assert.Equal(t, "handleClick", parsed.Function)
}

func TestEncodeActionPayloadBytes0Arena(t *testing.T) {
	t.Parallel()

	arena := ast_domain.GetArena()
	defer ast_domain.PutArena(arena)

	result := EncodeActionPayloadBytes0Arena(arena, "handleClick")
	require.NotNil(t, result)

	jsonBytes, err := base64.RawURLEncoding.DecodeString(string(*result))
	require.NoError(t, err)

	var parsed templater_dto.ActionPayload
	require.NoError(t, json.Unmarshal(jsonBytes, &parsed))
	assert.Equal(t, "handleClick", parsed.Function)
	assert.Empty(t, parsed.Args)
}

func TestEncodeActionPayloadBytes1Arena(t *testing.T) {
	t.Parallel()

	arena := ast_domain.GetArena()
	defer ast_domain.PutArena(arena)

	result := EncodeActionPayloadBytes1Arena(arena, "handleDelete",
		templater_dto.ActionArgument{Type: "s", Value: "item_1"},
	)
	require.NotNil(t, result)

	jsonBytes, err := base64.RawURLEncoding.DecodeString(string(*result))
	require.NoError(t, err)

	var parsed templater_dto.ActionPayload
	require.NoError(t, json.Unmarshal(jsonBytes, &parsed))
	assert.Equal(t, "handleDelete", parsed.Function)
	require.Len(t, parsed.Args, 1)
}

func TestEncodeActionPayloadBytes2Arena(t *testing.T) {
	t.Parallel()

	arena := ast_domain.GetArena()
	defer ast_domain.PutArena(arena)

	result := EncodeActionPayloadBytes2Arena(arena, "handleUpdate",
		templater_dto.ActionArgument{Type: "e"},
		templater_dto.ActionArgument{Type: "s", Value: "val"},
	)
	require.NotNil(t, result)

	jsonBytes, err := base64.RawURLEncoding.DecodeString(string(*result))
	require.NoError(t, err)

	var parsed templater_dto.ActionPayload
	require.NoError(t, json.Unmarshal(jsonBytes, &parsed))
	assert.Equal(t, "handleUpdate", parsed.Function)
	require.Len(t, parsed.Args, 2)
}

func TestEncodeActionPayloadBytes3Arena(t *testing.T) {
	t.Parallel()

	arena := ast_domain.GetArena()
	defer ast_domain.PutArena(arena)

	result := EncodeActionPayloadBytes3Arena(arena, "handleCreate",
		templater_dto.ActionArgument{Type: "e"},
		templater_dto.ActionArgument{Type: "s", Value: "a"},
		templater_dto.ActionArgument{Type: "s", Value: "b"},
	)
	require.NotNil(t, result)

	jsonBytes, err := base64.RawURLEncoding.DecodeString(string(*result))
	require.NoError(t, err)

	var parsed templater_dto.ActionPayload
	require.NoError(t, json.Unmarshal(jsonBytes, &parsed))
	assert.Equal(t, "handleCreate", parsed.Function)
	require.Len(t, parsed.Args, 3)
}

func TestEncodeActionPayloadBytes4Arena(t *testing.T) {
	t.Parallel()

	arena := ast_domain.GetArena()
	defer ast_domain.PutArena(arena)

	result := EncodeActionPayloadBytes4Arena(arena, "handleBatch",
		templater_dto.ActionArgument{Type: "e"},
		templater_dto.ActionArgument{Type: "s", Value: "a"},
		templater_dto.ActionArgument{Type: "s", Value: "b"},
		templater_dto.ActionArgument{Type: "s", Value: "c"},
	)
	require.NotNil(t, result)

	jsonBytes, err := base64.RawURLEncoding.DecodeString(string(*result))
	require.NoError(t, err)

	var parsed templater_dto.ActionPayload
	require.NoError(t, json.Unmarshal(jsonBytes, &parsed))
	assert.Equal(t, "handleBatch", parsed.Function)
	require.Len(t, parsed.Args, 4)
}

func TestArenaActionEncoder_EquivalenceWithPool(t *testing.T) {
	t.Parallel()

	warmupPool()

	testCases := []struct {
		name      string
		function  string
		arguments []templater_dto.ActionArgument
	}{
		{
			name:      "zero arguments",
			function:  "handleClick",
			arguments: []templater_dto.ActionArgument{},
		},
		{
			name:      "one argument",
			function:  "handleDelete",
			arguments: []templater_dto.ActionArgument{{Type: "s", Value: "item_1"}},
		},
		{
			name:     "two arguments",
			function: "handleUpdate",
			arguments: []templater_dto.ActionArgument{
				{Type: "e"},
				{Type: "s", Value: "test"},
			},
		},
		{
			name:     "three arguments",
			function: "handleCreate",
			arguments: []templater_dto.ActionArgument{
				{Type: "e"},
				{Type: "s", Value: "name"},
				{Type: "s", Value: "value"},
			},
		},
		{
			name:     "four arguments",
			function: "handleBatch",
			arguments: []templater_dto.ActionArgument{
				{Type: "e"},
				{Type: "s", Value: "a"},
				{Type: "s", Value: "b"},
				{Type: "s", Value: "c"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			poolBuf := EncodeActionPayloadBytes(templater_dto.ActionPayload{
				Function: tc.function,
				Args:     tc.arguments,
			})
			require.NotNil(t, poolBuf)
			poolResult := string(*poolBuf)
			ast_domain.PutByteBuf(poolBuf)

			arena := ast_domain.GetArena()
			arenaBuf := EncodeActionPayloadBytesArena(arena, templater_dto.ActionPayload{
				Function: tc.function,
				Args:     tc.arguments,
			})
			require.NotNil(t, arenaBuf)
			arenaResult := string(*arenaBuf)
			ast_domain.PutArena(arena)

			assert.Equal(t, poolResult, arenaResult)
		})
	}
}

func TestEncodeValue_AllNumericTypes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		value any
		want  string
	}{
		{name: "int8", value: int8(42), want: "42"},
		{name: "int16", value: int16(1000), want: "1000"},
		{name: "int32", value: int32(100000), want: "100000"},
		{name: "int64", value: int64(9223372036854775807), want: "9223372036854775807"},
		{name: "uint", value: uint(42), want: "42"},
		{name: "uint8", value: uint8(255), want: "255"},
		{name: "uint16", value: uint16(65535), want: "65535"},
		{name: "uint32", value: uint32(4294967295), want: "4294967295"},
		{name: "uint64", value: uint64(18446744073709551615), want: "18446744073709551615"},
		{name: "float32", value: float32(3.14), want: "3.14"},
		{name: "float64", value: float64(2.718281828), want: "2.718281828"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buffer []byte
			encodeValue(&buffer, tc.value)
			assert.Equal(t, tc.want, string(buffer))
		})
	}
}

func TestEncodeValue_DefaultFallback(t *testing.T) {
	t.Parallel()

	t.Run("byte slice", func(t *testing.T) {
		t.Parallel()

		var buffer []byte
		encodeValue(&buffer, []byte("hello"))

		assert.Equal(t, `"hello"`, string(buffer))
	})

	t.Run("struct via Sprintf", func(t *testing.T) {
		t.Parallel()

		type custom struct{ X int }
		var buffer []byte
		encodeValue(&buffer, custom{X: 42})

		assert.Contains(t, string(buffer), "42")
	})
}

func TestConvertValueToString(t *testing.T) {
	t.Parallel()

	t.Run("string passthrough", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "hello", convertValueToString("hello"))
	})

	t.Run("byte slice", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "world", convertValueToString([]byte("world")))
	})

	t.Run("other type fallback", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "42", convertValueToString(42))
	})

	t.Run("bool fallback", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "true", convertValueToString(true))
	})
}

func TestEscapeJSONString_AllEscapeSequences(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "backspace", input: "\b", want: `\b`},
		{name: "form feed", input: "\f", want: `\f`},
		{name: "newline", input: "\n", want: `\n`},
		{name: "carriage return", input: "\r", want: `\r`},
		{name: "tab", input: "\t", want: `\t`},
		{name: "double quote", input: `"`, want: `\"`},
		{name: "backslash", input: `\`, want: `\\`},
		{name: "control char 0x00", input: "\x00", want: `\u0000`},
		{name: "control char 0x01", input: "\x01", want: `\u0001`},
		{name: "control char 0x1f", input: "\x1f", want: `\u001f`},
		{name: "normal ascii", input: "abc", want: "abc"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buffer []byte
			escapeJSONString(&buffer, tc.input)
			assert.Equal(t, tc.want, string(buffer))
		})
	}
}
